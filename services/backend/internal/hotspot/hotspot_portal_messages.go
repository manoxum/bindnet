// hotspot_portal_messages.go expoe as rotas publicas (sem sessao) de
// avisos - a metade de "pull" da entrega: o dispositivo conectado le, na
// pagina de autoatendimento bindnet.local.com, os avisos direcionados a ele ou em
// broadcast, e confirma a leitura. Mesmo precedente restrito de
// hotspot_portal.go: o MAC do chamador nunca vem do corpo/query, e sempre
// resolvido no servidor a partir do IP de origem (resolvePortalMAC).
package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"database/sql"
	"net/http"
	"strings"
)

func registerHotspotPortalMessageRoutes(mux *http.ServeMux, db *sql.DB, worker *workerapi.Client) {
	mux.HandleFunc("GET /api/hotspot/portal/messages", func(w http.ResponseWriter, r *http.Request) {
		mac, err := resolvePortalMAC(r.Context(), r, db, worker)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		rows, err := store.ActiveMessagesForMAC(db, mac)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, messagesToResponse(rows))
	})

	mux.HandleFunc("POST /api/hotspot/portal/messages/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.PathValue("id"))
		if id == "" {
			http.Error(w, "id do aviso obrigatorio", http.StatusBadRequest)
			return
		}
		mac, err := resolvePortalMAC(r.Context(), r, db, worker)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err := store.MarkMessageRead(db, id, mac); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Se esse era o ultimo aviso urgente pendente do dispositivo, o
		// push do portal cativo pode ser desligado (respeitando bloqueio
		// de credito/cota que ainda o queira ligado).
		reconcileMessageCaptivePush(r.Context(), db, worker, mac)
		w.WriteHeader(http.StatusNoContent)
	})
}
