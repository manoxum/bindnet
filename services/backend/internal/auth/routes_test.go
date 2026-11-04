package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testAudit struct {
	events []string
}

func (a *testAudit) Record(_ context.Context, event, _ string, _ map[string]any) {
	a.events = append(a.events, event)
}

func TestLoginPersistentCookieAndLogout(t *testing.T) {
	admin, err := newAdministrator("admin", "senha-de-teste")
	if err != nil {
		t.Fatal(err)
	}
	aud := &testAudit{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, admin, aud)
	mux.HandleFunc("GET /protected", RequireSession(admin, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	invalid := httptest.NewRecorder()
	mux.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"errada"}`)))
	if invalid.Code != http.StatusUnauthorized || len(invalid.Result().Cookies()) != 0 {
		t.Fatal("senha incorreta nao deve criar sessao persistente")
	}

	login := httptest.NewRecorder()
	mux.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"admin","password":"senha-de-teste"}`)))
	if login.Code != http.StatusNoContent || len(login.Result().Cookies()) != 1 {
		t.Fatalf("login nao criou sessao: status=%d", login.Code)
	}
	cookie := login.Result().Cookies()[0]
	if cookie.MaxAge != 400*24*60*60 || !cookie.HttpOnly || cookie.Expires.IsZero() {
		t.Fatal("login deve emitir cookie persistente protegido")
	}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.AddCookie(cookie)
	authenticated := httptest.NewRecorder()
	mux.ServeHTTP(authenticated, request)
	if authenticated.Code != http.StatusNoContent {
		t.Fatal("cookie persistente deve autenticar rotas protegidas")
	}

	logout := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	request.AddCookie(cookie)
	mux.ServeHTTP(logout, request)
	cookies := logout.Result().Cookies()
	if logout.Code != http.StatusNoContent || len(cookies) != 1 || cookies[0].MaxAge != -1 || cookies[0].Value != "" || cookies[0].Path != cookie.Path {
		t.Fatal("sair deve remover o cookie persistente do mesmo escopo")
	}
	unauthenticated := httptest.NewRecorder()
	mux.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatal("rota protegida deve exigir login depois de remover o cookie")
	}
	if strings.Join(aud.events, ",") != "login,logout" {
		t.Fatalf("auditoria inesperada: %v", aud.events)
	}
}
