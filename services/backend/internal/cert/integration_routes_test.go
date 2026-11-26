package cert

import (
	"bindnet/backend/internal/auth"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestIntegrationTokenScope(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`CREATE TABLE integration_tokens(id TEXT PRIMARY KEY,token_hash TEXT,expires_at TEXT,revoked_at TEXT,last_used_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	token, _ := newIntegrationToken()
	other, _ := newIntegrationToken()
	for id, secret := range map[string]string{"one": token, "two": other} {
		if _, err = db.Exec(`INSERT INTO integration_tokens(id,token_hash,expires_at) VALUES ($1,$2,datetime('now','+1 day'))`, id, integrationTokenHash(secret)); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	RegisterIntegrationRoutes(mux, &auth.Administrator{Username: "admin", SecretHex: "00"}, db, nil, nil)
	call := func(method, path, header string) int {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", header)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response.Code
	}
	for _, test := range []struct {
		method, path, header string
		status               int
	}{
		{"GET", "/api/integrations/v1/status", "Bearer " + token, 200},
		{"GET", "/api/integrations/v1/status", token, 401},
		{"GET", "/api/integrations/v1/status", "Bearer bnet_adulterado", 401},
		{"POST", "/api/integrations/tokens", "Bearer " + token, 401},
	} {
		if got := call(test.method, test.path, test.header); got != test.status {
			t.Fatalf("%s %s: %d esperado %d", test.method, test.path, got, test.status)
		}
	}
	if _, err = db.Exec(`UPDATE integration_tokens SET revoked_at=CURRENT_TIMESTAMP WHERE id='one'`); err != nil {
		t.Fatal(err)
	}
	if call("GET", "/api/integrations/v1/status", "Bearer "+token) != 401 {
		t.Fatal("token revogado autenticou")
	}
	if call("GET", "/api/integrations/v1/status", "Bearer "+other) != 200 {
		t.Fatal("revogacao afetou outro token")
	}
	if _, err = db.Exec(`UPDATE integration_tokens SET expires_at=datetime('now','-1 day') WHERE id='two'`); err != nil {
		t.Fatal(err)
	}
	if call("GET", "/api/integrations/v1/status", "Bearer "+other) != 401 {
		t.Fatal("token expirado autenticou")
	}
}
func TestIntegrationTokenFormat(t *testing.T) {
	token, err := newIntegrationToken()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^bnet_[A-Za-z0-9_-]{43}$`).MatchString(token) {
		t.Fatal("formato inesperado")
	}
	other, err := newIntegrationToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == other {
		t.Fatal("segredos repetidos")
	}
}
