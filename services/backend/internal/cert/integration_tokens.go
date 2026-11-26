package cert

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type integrationTokenResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func newIntegrationToken() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return "bnet_" + base64.RawURLEncoding.EncodeToString(key), nil
}
func integrationTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// requireIntegrationToken consulta o estado atual para respeitar revogacao e validade.
func requireIntegrationToken(w http.ResponseWriter, r *http.Request, db *sql.DB) (string, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer bnet_") {
		http.Error(w, "token de integração inválido", 401)
		return "", false
	}
	token := strings.TrimPrefix(header, "Bearer ")
	var id string
	err := db.QueryRowContext(r.Context(), `SELECT id FROM integration_tokens WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>CURRENT_TIMESTAMP`, integrationTokenHash(token)).Scan(&id)
	if err != nil {
		http.Error(w, "token de integração inválido ou expirado", 401)
		return "", false
	}
	if _, err = db.ExecContext(r.Context(), `UPDATE integration_tokens SET last_used_at=CURRENT_TIMESTAMP WHERE id=$1`, id); err != nil {
		http.Error(w, "falha ao registar uso do token", 500)
		return "", false
	}
	return id, true
}
func registerIntegrationTokenRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, aud *audit.Client) {
	mux.HandleFunc("GET /api/integrations/tokens", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `SELECT id,name,prefix,expires_at,revoked_at,last_used_at,created_at FROM integration_tokens ORDER BY created_at DESC`)
		if err != nil {
			http.Error(w, "falha ao consultar tokens", 500)
			return
		}
		defer rows.Close()
		result := []integrationTokenResponse{}
		for rows.Next() {
			var row integrationTokenResponse
			if err := rows.Scan(&row.ID, &row.Name, &row.Prefix, &row.ExpiresAt, &row.RevokedAt, &row.LastUsedAt, &row.CreatedAt); err != nil {
				http.Error(w, "falha ao ler tokens", 500)
				return
			}
			result = append(result, row)
		}
		if rows.Err() != nil {
			http.Error(w, "falha ao ler tokens", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}))
	mux.HandleFunc("POST /api/integrations/tokens", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name         string `json:"name"`
			ValidityDays int    `json:"validityDays"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
			http.Error(w, "pedido inválido", 400)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.ValidityDays == 0 {
			req.ValidityDays = 90
		}
		if req.Name == "" || len(req.Name) > 100 || req.ValidityDays < 1 || req.ValidityDays > 365 {
			http.Error(w, "indique nome e validade de 1 a 365 dias", 400)
			return
		}
		token, err := newIntegrationToken()
		if err != nil {
			http.Error(w, "falha ao gerar token", 500)
			return
		}
		expiry := time.Now().Add(time.Duration(req.ValidityDays) * 24 * time.Hour)
		var id string
		err = db.QueryRowContext(r.Context(), `INSERT INTO integration_tokens (name,prefix,token_hash,expires_at) VALUES ($1,$2,$3,$4) RETURNING id`, req.Name, token[:13], integrationTokenHash(token), expiry).Scan(&id)
		if err != nil {
			http.Error(w, "falha ao guardar token", 500)
			return
		}
		user, _ := auth.SessionUser(r, admin)
		aud.Record(r.Context(), "integration_token_issued", user, map[string]any{"id": id, "name": req.Name})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "token": token, "expiresAt": expiry})
	}))
	mux.HandleFunc("DELETE /api/integrations/tokens/{id}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if len(id) != 36 {
			http.Error(w, "token inválido", 400)
			return
		}
		result, err := db.ExecContext(r.Context(), `UPDATE integration_tokens SET revoked_at=CURRENT_TIMESTAMP WHERE id=$1`, id)
		if err != nil {
			http.Error(w, "falha ao revogar token", 500)
			return
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			http.Error(w, "token não encontrado", 404)
			return
		}
		user, _ := auth.SessionUser(r, admin)
		aud.Record(r.Context(), "integration_token_revoked", user, map[string]any{"id": id})
		w.WriteHeader(http.StatusNoContent)
	}))
}
