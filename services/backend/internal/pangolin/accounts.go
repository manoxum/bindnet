package pangolin

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strconv"

	"bindnet/backend/internal/auth"
)

var accountPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func connectorPath(account string, id int) string {
	return "/tunnels/" + account + "/" + strconv.Itoa(id)
}

func (s *service) registerAccountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tunnels/accounts", auth.RequireSession(s.admin, s.listAccounts))
	mux.HandleFunc("POST /api/tunnels/accounts", auth.RequireSession(s.admin, s.createAccount))
	mux.HandleFunc("DELETE /api/tunnels/accounts/{accountId}", auth.RequireSession(s.admin, s.deleteAccount))
}
func (s *service) listAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	rows, err := s.db.QueryContext(r.Context(), `SELECT id,name,api_url,endpoint,org_id,api_key<>'' FROM pangolin_accounts ORDER BY name,id`)
	if err != nil {
		http.Error(w, "não foi possível listar as contas Pangolin", 500)
		return
	}
	defer rows.Close()
	result := []Config{}
	for rows.Next() {
		var c Config
		if err = rows.Scan(&c.ID, &c.Name, &c.APIURL, &c.Endpoint, &c.OrgID, &c.Configured); err != nil {
			http.Error(w, "não foi possível ler as contas", 500)
			return
		}
		result = append(result, c)
	}
	if rows.Err() != nil {
		http.Error(w, "não foi possível ler as contas", 500)
		return
	}
	writeJSON(w, result)
}
func (s *service) createAccount(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var input configRequest
	if err := decode(w, r, &input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	c, err := input.config(Config{})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	token := make([]byte, 8)
	if _, err = rand.Read(token); err != nil {
		http.Error(w, "não foi possível gerar o identificador da conta", 500)
		return
	}
	c.ID = hex.EncodeToString(token)
	_, err = s.db.ExecContext(r.Context(), `INSERT INTO pangolin_accounts(id,name,api_url,endpoint,org_id,api_key) VALUES($1,$2,$3,$4,$5,$6)`, c.ID, c.Name, c.APIURL, c.Endpoint, c.OrgID, c.APIKey)
	if err != nil {
		http.Error(w, "não foi possível guardar a conta Pangolin", 500)
		return
	}
	s.record(r, "pangolin_account_created", 0)
	c.Configured = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	writeJSON(w, c)
}
func (s *service) deleteAccount(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	id := r.PathValue("accountId")
	result, err := s.db.ExecContext(r.Context(), `DELETE FROM pangolin_accounts WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM pangolin_account_sites WHERE account_id=$1) AND NOT EXISTS(SELECT 1 FROM pangolin_account_resources WHERE account_id=$1)`, id)
	if err != nil {
		http.Error(w, "não foi possível remover a conta", 500)
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		http.Error(w, "remova os sites e recursos geridos antes de remover a conta", 409)
		return
	}
	s.record(r, "pangolin_account_deleted", 0)
	w.WriteHeader(204)
}
