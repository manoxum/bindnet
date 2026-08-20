// dns_records.go expoe a tabela local_dns_records para o painel: cada
// hostname e tambem uma zona por sufixo. Por exemplo, empresa.local cobre
// empresa.local e todo *.empresa.local; o endereco opcional fixa um A
// record para a zona, enquanto a ausencia dele preserva o loopback da view
// host do split-horizon.
package dns

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

type localDnsRecord struct {
	Hostname          string  `json:"hostname"`
	Address           string  `json:"address"`
	ConfiguredAddress *string `json:"configuredAddress"`
	LoopbackOffset    int64   `json:"loopbackOffset"`
	CreatedAt         string  `json:"createdAt"`
}

// offsetToLoopback replica services/worker/dns/zones.go:offsetToLoopback -
// os 24 bits menos significativos do offset viram os tres ultimos
// octetos de um IP 127.0.0.0/8.
func offsetToLoopback(offset int64) string {
	b := uint32(offset) & 0xFFFFFF
	return net.IPv4(127, byte(b>>16), byte(b>>8), byte(b)).String()
}

func RegisterDNSRecordRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, audit *audit.Client) {
	mux.HandleFunc("GET /api/dns/records", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `
			SELECT hostname, loopback_offset, address::text, created_at FROM local_dns_records ORDER BY hostname
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		records := []localDnsRecord{}
		for rows.Next() {
			var record localDnsRecord
			if err := rows.Scan(&record.Hostname, &record.LoopbackOffset, &record.ConfiguredAddress, &record.CreatedAt); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			record.Address = offsetToLoopback(record.LoopbackOffset)
			if record.ConfiguredAddress != nil {
				record.Address = *record.ConfiguredAddress
			}
			records = append(records, record)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(records)
	}))

	mux.HandleFunc("POST /api/dns/records", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Hostname string `json:"hostname"`
			Address  string `json:"address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		// O operador pode colar um FQDN canonico (empresa.local.). Armazena
		// sem o ponto final, como o dns-provider faz antes de comparar zonas.
		hostname := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(req.Hostname)), ".")
		if hostname == "" || !isValidDNSDomain(hostname) {
			http.Error(w, "hostname invalido", http.StatusBadRequest)
			return
		}
		var address any
		if rawAddress := strings.TrimSpace(req.Address); rawAddress != "" {
			ip := net.ParseIP(rawAddress)
			if ip == nil || ip.To4() == nil {
				http.Error(w, "IP IPv4 invalido", http.StatusBadRequest)
				return
			}
			address = ip.To4().String()
		}

		if _, err := db.ExecContext(r.Context(), `
			INSERT INTO local_dns_records (hostname, loopback_offset, address)
			VALUES ($1, nextval('local_dns_records_offset_seq'), $2)
			ON CONFLICT (hostname) DO UPDATE SET address = EXCLUDED.address
		`, hostname, address); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var record localDnsRecord
		record.Hostname = hostname
		err := db.QueryRowContext(r.Context(), `
			SELECT loopback_offset, address::text, created_at FROM local_dns_records WHERE hostname = $1
		`, hostname).Scan(&record.LoopbackOffset, &record.ConfiguredAddress, &record.CreatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		record.Address = offsetToLoopback(record.LoopbackOffset)
		if record.ConfiguredAddress != nil {
			record.Address = *record.ConfiguredAddress
		}

		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "dns_record_added", username, map[string]any{"hostname": hostname, "address": record.ConfiguredAddress})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(record)
	}))

	mux.HandleFunc("DELETE /api/dns/records", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		if _, err := db.ExecContext(r.Context(), `DELETE FROM local_dns_records`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "dns_records_cleared", username, nil)
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("DELETE /api/dns/records/{hostname}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		hostname := r.PathValue("hostname")
		if hostname == "" {
			http.Error(w, "hostname obrigatorio", http.StatusBadRequest)
			return
		}
		if _, err := db.ExecContext(r.Context(), `DELETE FROM local_dns_records WHERE hostname = $1`, hostname); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "dns_record_removed", username, map[string]any{"hostname": hostname})
		w.WriteHeader(http.StatusNoContent)
	}))
}
