package hotspot

import (
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"database/sql"
	"encoding/json"
	"net/http"
)

// effectiveContentPlanID resolve o plano de conteudo que vale para um
// MAC agora: o override do proprio dispositivo (hotspot_device_limits.
// content_plan_id) vence; na ausencia dele, o do perfil vinculado.
// Devolve nil quando nem um nem outro tem plano (sem bloqueio).
func effectiveContentPlanID(db *sql.DB, mac string) (*string, error) {
	if planID, err := store.DeviceContentPlanID(db, mac); err != nil {
		return nil, err
	} else if planID != nil {
		return planID, nil
	}
	profileID, err := deviceProfileID(db, mac)
	if err != nil {
		return nil, err
	}
	return store.ProfileContentPlanID(db, profileID)
}

type contentLinkRequest struct {
	// PlanID nil/"" desvincula (sem plano).
	PlanID *string `json:"planId"`
}

type contentLinkResponse struct {
	PlanID *string `json:"planId"`
}

// RegisterHotspotContentLinkRoutes expoe o vinculo de um plano a um
// perfil ou dispositivo. Toda mudanca reaplica o conteudo ao vivo
// (publica o mapa IP->plano + reaplica firewall) para efeito imediato.
func RegisterHotspotContentLinkRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, worker *workerapi.Client) {
	mux.HandleFunc("GET /api/hotspot/profiles/{id}/content-plan", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		planID, err := store.ProfileContentPlanID(db, r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, contentLinkResponse{PlanID: planID})
	}))

	mux.HandleFunc("PATCH /api/hotspot/profiles/{id}/content-plan", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		planID, ok := decodeContentLink(w, r, db)
		if !ok {
			return
		}
		found, err := store.SetProfileContentPlanID(db, r.PathValue("id"), planID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "perfil nao encontrado", http.StatusNotFound)
			return
		}
		applyContentLive(r.Context(), db, worker)
		writeJSON(w, contentLinkResponse{PlanID: planID})
	}))

	mux.HandleFunc("GET /api/hotspot/devices/{mac}/content-plan", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		mac, err := normalizeHotspotMAC(r.PathValue("mac"))
		if err != nil {
			http.Error(w, "mac invalido", http.StatusBadRequest)
			return
		}
		planID, err := store.DeviceContentPlanID(db, mac)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, contentLinkResponse{PlanID: planID})
	}))

	mux.HandleFunc("PATCH /api/hotspot/devices/{mac}/content-plan", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		mac, err := normalizeHotspotMAC(r.PathValue("mac"))
		if err != nil {
			http.Error(w, "mac invalido", http.StatusBadRequest)
			return
		}
		planID, ok := decodeContentLink(w, r, db)
		if !ok {
			return
		}
		if err := store.SetDeviceContentPlanID(db, mac, planID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyContentLive(r.Context(), db, worker)
		writeJSON(w, contentLinkResponse{PlanID: planID})
	}))
}

// decodeContentLink le o corpo e valida o planId (nil/"" desvincula;
// caso contrario o plano precisa existir). Devolve o ponteiro
// normalizado (nil quando desvincula) e ok=false apos ja ter respondido
// um erro.
func decodeContentLink(w http.ResponseWriter, r *http.Request, db *sql.DB) (*string, bool) {
	var req contentLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo invalido", http.StatusBadRequest)
		return nil, false
	}
	if req.PlanID == nil || *req.PlanID == "" {
		return nil, true
	}
	_, found, err := store.GetContentPlan(db, *req.PlanID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	if !found {
		http.Error(w, "plano nao encontrado", http.StatusBadRequest)
		return nil, false
	}
	return req.PlanID, true
}
