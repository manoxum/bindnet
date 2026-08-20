// hotspot_messages.go expoe o CRUD de avisos que o operador envia aos
// dispositivos conectados (rotas de admin, protegidas por sessao). A
// entrega em si acontece em dois lugares: o dispositivo le o aviso na
// pagina publica bindnet.local.com (hotspot_portal_messages.go) e, para avisos
// urgentes, tambem via push do portal cativo (hotspot_messages_push.go).
package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	messageMaxTitle = 200
	messageMaxBody  = 2000
)

type hotspotMessageResponse struct {
	ID        string  `json:"id"`
	Title     string  `json:"title,omitempty"`
	Body      string  `json:"body"`
	TargetMAC string  `json:"targetMac,omitempty"`
	Urgent    bool    `json:"urgent"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

type hotspotMessageCreateRequest struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	TargetMAC string `json:"targetMac"`
	Urgent    bool   `json:"urgent"`
	ExpiresAt string `json:"expiresAt"`
}

// messageToResponse converte a linha do banco no DTO JSON (reusado pela
// listagem do painel e pelas rotas publicas do portal).
func messageToResponse(m store.MessageRow) hotspotMessageResponse {
	resp := hotspotMessageResponse{
		ID:        m.ID,
		Title:     m.Title,
		Body:      m.Body,
		TargetMAC: m.TargetMAC,
		Urgent:    m.Urgent,
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
	}
	if m.ExpiresAt.Valid {
		s := m.ExpiresAt.Time.UTC().Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	return resp
}

func messagesToResponse(rows []store.MessageRow) []hotspotMessageResponse {
	out := make([]hotspotMessageResponse, 0, len(rows))
	for _, m := range rows {
		out = append(out, messageToResponse(m))
	}
	return out
}

func RegisterHotspotMessageRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, worker *workerapi.Client, audit *audit.Client) {
	mux.HandleFunc("GET /api/hotspot/messages", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		rows, err := store.ListActiveMessages(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, messagesToResponse(rows))
	}))

	mux.HandleFunc("POST /api/hotspot/messages", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		req, expiresAt, err := parseMessageCreate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		m, err := store.CreateMessage(db, req.Title, req.Body, req.TargetMAC, req.Urgent, expiresAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if m.Urgent {
			reconcileMessageCaptivePush(r.Context(), db, worker, m.TargetMAC)
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "hotspot_message_created", username, map[string]any{
			"id": m.ID, "targetMac": m.TargetMAC, "urgent": m.Urgent,
		})
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, messageToResponse(m))
	}))

	mux.HandleFunc("DELETE /api/hotspot/messages/{id}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		m, found, err := store.GetMessage(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "aviso nao encontrado", http.StatusNotFound)
			return
		}
		if _, err := store.SetMessageActive(db, id, false); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if m.Urgent {
			reconcileMessageCaptivePush(r.Context(), db, worker, m.TargetMAC)
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "hotspot_message_removed", username, map[string]any{"id": id})
		w.WriteHeader(http.StatusNoContent)
	}))
}

// parseMessageCreate valida o corpo do POST: body obrigatorio, MAC alvo
// normalizado quando informado (vazio = broadcast) e expiresAt em RFC3339
// quando informado.
func parseMessageCreate(r *http.Request) (hotspotMessageCreateRequest, sql.NullTime, error) {
	var req hotspotMessageCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, sql.NullTime{}, errors.New("corpo invalido")
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		return req, sql.NullTime{}, errors.New("campo 'body' obrigatorio")
	}
	if len(req.Title) > messageMaxTitle || len(req.Body) > messageMaxBody {
		return req, sql.NullTime{}, errors.New("aviso muito longo")
	}
	if strings.TrimSpace(req.TargetMAC) != "" {
		mac, err := normalizeHotspotMAC(req.TargetMAC)
		if err != nil {
			return req, sql.NullTime{}, errors.New("mac alvo invalido")
		}
		req.TargetMAC = mac
	} else {
		req.TargetMAC = ""
	}
	var expiresAt sql.NullTime
	if s := strings.TrimSpace(req.ExpiresAt); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return req, sql.NullTime{}, errors.New("expiresAt deve ser uma data RFC3339")
		}
		expiresAt = sql.NullTime{Time: t, Valid: true}
	}
	return req, expiresAt, nil
}
