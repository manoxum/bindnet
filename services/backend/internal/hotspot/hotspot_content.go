package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
)

type contentPlanRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	DefaultAction string `json:"defaultAction"`
}

type contentPlanDetail struct {
	store.ContentPlan
	Rules []store.ContentRule `json:"rules"`
}

type contentRuleRequest struct {
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Action string `json:"action"`
}

func validContentAction(a string) bool { return a == "allow" || a == "block" }
func validContentKind(k string) bool   { return k == "domain" || k == "ip" || k == "category" }

// normalizeContentValue valida e normaliza o valor de uma regra conforme
// o kind. Dominio vira minusculo sem ponto/curinga na frente; ip aceita
// IP ou CIDR; category exige um slug existente.
func normalizeContentValue(db *sql.DB, kind, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("valor obrigatorio")
	}
	switch kind {
	case "domain":
		v := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(value, "*."), "."))
		if v == "" || strings.ContainsAny(v, " /") {
			return "", errors.New("dominio invalido")
		}
		return v, nil
	case "ip":
		if net.ParseIP(value) == nil {
			if _, _, err := net.ParseCIDR(value); err != nil {
				return "", errors.New("ip/cidr invalido")
			}
		}
		return value, nil
	case "category":
		cats, err := store.ListContentCategories(db)
		if err != nil {
			return "", err
		}
		for _, c := range cats {
			if c.Slug == value {
				return value, nil
			}
		}
		return "", errors.New("categoria inexistente")
	}
	return "", errors.New("kind invalido")
}

// RegisterHotspotContentRoutes expoe o CRUD de planos e regras de
// conteudo. applyContentLive republica o mapa IP->plano e reaplica as
// zonas do firewall a cada mutacao, para o efeito ser imediato.
func RegisterHotspotContentRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, worker *workerapi.Client, audit *audit.Client) {
	mux.HandleFunc("GET /api/hotspot/content-plans", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		plans, err := store.ListContentPlans(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, plans)
	}))

	mux.HandleFunc("POST /api/hotspot/content-plans", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req contentPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
			http.Error(w, "campo 'name' obrigatorio", http.StatusBadRequest)
			return
		}
		if req.DefaultAction == "" {
			req.DefaultAction = "allow"
		}
		if !validContentAction(req.DefaultAction) {
			http.Error(w, "defaultAction invalido", http.StatusBadRequest)
			return
		}
		plan, err := store.InsertContentPlan(db, strings.TrimSpace(req.Name), req.Description, req.DefaultAction)
		if errors.Is(err, store.ErrContentPlanNameTaken) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "content_plan_created", username, map[string]any{"id": plan.ID, "name": plan.Name})
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, plan)
	}))

	mux.HandleFunc("GET /api/hotspot/content-plans/{id}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		plan, found, err := store.GetContentPlan(db, r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "plano nao encontrado", http.StatusNotFound)
			return
		}
		rules, err := store.ListContentRules(db, plan.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, contentPlanDetail{ContentPlan: plan, Rules: rules})
	}))

	mux.HandleFunc("PATCH /api/hotspot/content-plans/{id}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req contentPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
			http.Error(w, "campo 'name' obrigatorio", http.StatusBadRequest)
			return
		}
		if req.DefaultAction == "" {
			req.DefaultAction = "allow"
		}
		if !validContentAction(req.DefaultAction) {
			http.Error(w, "defaultAction invalido", http.StatusBadRequest)
			return
		}
		plan, found, err := store.UpdateContentPlan(db, r.PathValue("id"), strings.TrimSpace(req.Name), req.Description, req.DefaultAction)
		if errors.Is(err, store.ErrContentPlanNameTaken) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "plano nao encontrado", http.StatusNotFound)
			return
		}
		applyContentLive(r.Context(), db, worker)
		writeJSON(w, plan)
	}))

	mux.HandleFunc("DELETE /api/hotspot/content-plans/{id}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		if err := store.DeleteContentPlan(db, r.PathValue("id")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyContentLive(r.Context(), db, worker)
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("POST /api/hotspot/content-plans/{id}/rules", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		planID := r.PathValue("id")
		if _, found, err := store.GetContentPlan(db, planID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !found {
			http.Error(w, "plano nao encontrado", http.StatusNotFound)
			return
		}
		var req contentRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		if req.Action == "" {
			req.Action = "block"
		}
		if !validContentKind(req.Kind) || !validContentAction(req.Action) {
			http.Error(w, "kind/action invalido", http.StatusBadRequest)
			return
		}
		value, err := normalizeContentValue(db, req.Kind, req.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rule, err := store.InsertContentRule(db, planID, req.Kind, value, req.Action)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyContentLive(r.Context(), db, worker)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, rule)
	}))

	mux.HandleFunc("DELETE /api/hotspot/content-rules/{ruleId}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		found, err := store.DeleteContentRule(db, r.PathValue("ruleId"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "regra nao encontrada", http.StatusNotFound)
			return
		}
		applyContentLive(r.Context(), db, worker)
		w.WriteHeader(http.StatusNoContent)
	}))
}

// writeJSON e um atalho para respostas JSON (Content-Type + encode).
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
