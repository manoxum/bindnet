package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"database/sql"
	"encoding/json"
	"net/http"
)

// autostartPayload e o corpo/resposta das duas rotas - um unico
// booleano, mesmo formato nos dois sentidos pra o painel poder reusar o
// tipo.
type autostartPayload struct {
	Enabled bool `json:"enabled"`
}

// RegisterHotspotAutostartRoutes expoe o interruptor "iniciar
// automaticamente no arranque" do painel. Fica em rota propria (e nao
// em GET/PATCH /api/hotspot/config) de proposito: a rota de config
// dispara POST /hotspot/apply no painel, que REINICIA o hotspot - e
// gravar uma preferencia de arranque nunca deve derrubar quem esta
// conectado agora. Mesmo motivo pelo qual a chave mora fora de
// hotspotConfigKeys (ver hotspotAutostartKey em hotspot_config_state.go),
// e mesmo molde de RegisterHotspotUplinkRoute em hotspot_network.go.
func RegisterHotspotAutostartRoutes(mux *http.ServeMux, admin *auth.Administrator, audit *audit.Client, db *sql.DB) {
	mux.HandleFunc("GET /api/hotspot/autostart", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		enabled, err := store.HotspotAutostartEnabled(r.Context(), db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(autostartPayload{Enabled: enabled})
	}))

	mux.HandleFunc("PATCH /api/hotspot/autostart", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var payload autostartPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		if err := store.SetHotspotAutostart(r.Context(), db, payload.Enabled); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "config_changed", username, map[string]any{
			"section": "hotspot_autostart",
			"enabled": payload.Enabled,
		})
		w.WriteHeader(http.StatusNoContent)
	}))
}
