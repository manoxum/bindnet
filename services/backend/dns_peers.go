// dns_peers.go expoe os servidores Bindnet encontrados por
// broadcast/multicast na rede local (tabela discover_peers, escrita
// pelo dns-provider - ver services/worker/dns/discover_broadcast.go).
// So leitura: quem decide participar da malha e o DISCOVER_PEERS
// manual (services/backend/dns.go, secao "dns" do .env), esta lista
// e so informativa para o operador ver quem mais existe na rede.
package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type discoveredPeer struct {
	Address    string `json:"address"`
	NodeName   string `json:"nodeName"`
	Source     string `json:"source"`
	LastSeenAt string `json:"lastSeenAt"`
}

func registerDNSPeerRoutes(mux *http.ServeMux, admin *administrator, db *sql.DB) {
	mux.HandleFunc("GET /api/dns/peers", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `
			SELECT address, node_name, source, last_seen_at
			FROM discover_peers ORDER BY node_name
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		peers := []discoveredPeer{}
		for rows.Next() {
			var peer discoveredPeer
			if err := rows.Scan(&peer.Address, &peer.NodeName, &peer.Source, &peer.LastSeenAt); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			peers = append(peers, peer)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(peers)
	}))
}
