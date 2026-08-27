package dns

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"

	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
)

type localTLD struct {
	TLD     string `json:"tld"`
	Mode    string `json:"mode"`
	Address string `json:"address,omitempty"`
}

func RegisterLocalTLDRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, audit *audit.Client) {
	mux.HandleFunc("GET /api/dns/local-tlds", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		entries, err := loadLocalTLDs(r.Context(), db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entries)
	}))

	mux.HandleFunc("PUT /api/dns/local-tlds", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var entries []localTLD
		if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		normalized, err := normalizeLocalTLDs(entries)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := replaceLocalTLDs(r.Context(), db, normalized); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "dns_local_tlds_changed", username, map[string]any{"tlds": normalized})
		w.WriteHeader(http.StatusNoContent)
	}))
}

func loadLocalTLDs(ctx context.Context, db *sql.DB) ([]localTLD, error) {
	cfg, err := getDNSConfig(ctx, db)
	if err != nil {
		return nil, err
	}
	entries := map[string]localTLD{}
	for _, raw := range strings.Split(cfg[KeyLocalTLDs], ",") {
		tld := strings.TrimSpace(strings.ToLower(raw))
		if tld != "" {
			entries[tld] = localTLD{TLD: tld, Mode: "automatic"}
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT tld, address::text FROM local_dns_tlds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tld string
		var address sql.NullString
		if err := rows.Scan(&tld, &address); err != nil {
			return nil, err
		}
		entry := localTLD{TLD: tld, Mode: "automatic"}
		if address.Valid {
			entry.Mode, entry.Address = "static", address.String
		}
		entries[tld] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]localTLD, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].TLD < result[j].TLD })
	return result, nil
}

func normalizeLocalTLDs(entries []localTLD) ([]localTLD, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("DNS_LOCAL_TLDS deve conter pelo menos um TLD valido")
	}
	seen := map[string]bool{}
	result := make([]localTLD, 0, len(entries))
	for _, entry := range entries {
		entry.TLD = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(entry.TLD)), ".")
		if entry.TLD == "" || seen[entry.TLD] || validateLocalTLDs(entry.TLD) != nil {
			return nil, fmt.Errorf("TLD local invalido: %q", entry.TLD)
		}
		seen[entry.TLD] = true
		entry.Address = strings.TrimSpace(entry.Address)
		switch entry.Mode {
		case "automatic":
			entry.Address = ""
		case "static":
			ip := parseStaticIPv4(entry.Address)
			if ip == nil {
				return nil, fmt.Errorf("IP IPv4 estatico invalido para %s", entry.TLD)
			}
			entry.Address = ip.String()
		default:
			return nil, fmt.Errorf("modo invalido para %s", entry.TLD)
		}
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].TLD < result[j].TLD })
	return result, nil
}

// parseStaticIPv4 aceita o formato IPv4 normal e o retorno INET do
// PostgreSQL ("10.234.1.69/32"), sempre normalizando para o host IPv4.
func parseStaticIPv4(raw string) net.IP {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "/") {
		ip, network, err := net.ParseCIDR(raw)
		if err != nil {
			return nil
		}
		ones, bits := network.Mask.Size()
		if bits != 32 || ones != 32 {
			return nil
		}
		return ip.To4()
	}
	ip := net.ParseIP(raw)
	if ip == nil || ip.To4() == nil {
		return nil
	}
	return ip.To4()
}

func replaceLocalTLDs(ctx context.Context, db *sql.DB, entries []localTLD) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := invalidateChangedTLDRecords(ctx, tx, entries); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM local_dns_tlds`); err != nil {
		return err
	}
	for _, entry := range entries {
		var address any
		if entry.Mode == "static" {
			address = entry.Address
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO local_dns_tlds (tld, address) VALUES ($1, $2)`, entry.TLD, address); err != nil {
			return err
		}
	}
	tlds := make([]string, 0, len(entries))
	for _, entry := range entries {
		tlds = append(tlds, entry.TLD)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO dns_config (key, value, updated_at) VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
	`, KeyLocalTLDs, strings.Join(tlds, ","))
	if err != nil {
		return err
	}
	return tx.Commit()
}

// invalidateChangedTLDRecords remove somente as respostas dinamicas dos
// TLDs que foram alterados ou removidos. Registros manuais e os TLDs que
// nao mudaram permanecem intactos. Entradas legadas sem origin_tld sao
// invalidadas na primeira alteracao por nao ser possivel atribui-las com
// seguranca a um TLD especifico.
func invalidateChangedTLDRecords(ctx context.Context, tx *sql.Tx, entries []localTLD) error {
	rows, err := tx.QueryContext(ctx, `SELECT tld, COALESCE(host(address), '') FROM local_dns_tlds`)
	if err != nil {
		return err
	}
	defer rows.Close()

	previous := map[string]string{}
	for rows.Next() {
		var tld, address string
		if err := rows.Scan(&tld, &address); err != nil {
			return err
		}
		previous[tld] = address
	}
	if err := rows.Err(); err != nil {
		return err
	}

	next := map[string]string{}
	for _, entry := range entries {
		next[entry.TLD] = entry.Address
	}
	changed := make([]string, 0)
	for tld, previousAddress := range previous {
		if nextAddress, exists := next[tld]; !exists || nextAddress != previousAddress {
			changed = append(changed, tld)
		}
	}
	for _, tld := range changed {
		if _, err := tx.ExecContext(ctx, `DELETE FROM local_dns_records WHERE source = 'dynamic' AND origin_tld = $1`, tld); err != nil {
			return err
		}
	}
	if len(changed) > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM local_dns_records WHERE source = 'dynamic' AND origin_tld IS NULL`); err != nil {
			return err
		}
	}
	return nil
}
