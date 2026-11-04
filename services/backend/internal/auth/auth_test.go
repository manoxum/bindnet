package auth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPersistentTokenAndSignature(t *testing.T) {
	secret := []byte("segredo-de-teste")
	token, err := issueToken("admin", secret)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.SplitN(token, ".", 2)[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["exp"]; exists {
		t.Fatal("token persistente nao deve ter prazo de expiracao")
	}
	if username, ok := validateToken(token, secret); !ok || username != "admin" {
		t.Fatal("token persistente deveria autenticar o administrador")
	}
	for _, invalid := range []string{"invalido", token + "x"} {
		if _, ok := validateToken(invalid, secret); ok {
			t.Fatal("token adulterado nao pode autenticar")
		}
	}
	if _, ok := validateToken(token, []byte("segredo-novo")); ok {
		t.Fatal("trocar o segredo deve invalidar sessoes persistentes")
	}
}

func TestLegacySessionRenewal(t *testing.T) {
	admin := &Administrator{Username: "admin", SecretHex: "abcdef"}
	mux := http.NewServeMux()
	RegisterRoutes(mux, admin, nil)

	for _, expired := range []bool{false, true} {
		expires := time.Now().Add(time.Hour)
		if expired {
			expires = time.Now().Add(-time.Hour)
		}
		payload, _ := json.Marshal(session{Username: "admin", Exp: expires.Unix()})
		encoded := base64.RawURLEncoding.EncodeToString(payload)
		token := encoded + "." + sign(encoded, admin.secret())
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, req)
		if expired {
			if recorder.Code != http.StatusUnauthorized || len(recorder.Result().Cookies()) != 0 {
				t.Fatal("sessao legada expirada nao deve ser renovada")
			}
			continue
		}
		if recorder.Code != http.StatusOK {
			t.Fatalf("sessao valida retornou %d", recorder.Code)
		}
		cookies := recorder.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("esperava cookie renovado, recebeu %d", len(cookies))
		}
		cookie := cookies[0]
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" {
			t.Fatal("cookie deve preservar protecoes e escopo do dominio administrativo")
		}
		if cookie.MaxAge != 400*24*60*60 || time.Until(cookie.Expires) < 399*24*time.Hour {
			t.Fatal("cookie deve persistir entre reinicios do navegador e renovar por 400 dias")
		}
		payload, _ = base64.RawURLEncoding.DecodeString(strings.SplitN(cookie.Value, ".", 2)[0])
		var renewed session
		if err := json.Unmarshal(payload, &renewed); err != nil || renewed.Exp != 0 {
			t.Fatal("renovacao deve migrar token legado para sessao sem expiracao")
		}
		if recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("resposta da sessao nao deve ser cacheada")
		}
	}
}

func TestPersistentSessionSurvivesBackendReload(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "senha-de-teste")
	path := filepath.Join(t.TempDir(), "admin.json")
	created, err := loadOrCreateAdminAt(path)
	if err != nil {
		t.Fatal(err)
	}
	token, err := issueToken(created.Username, created.secret())
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := loadOrCreateAdminAt(path)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	if username, ok := SessionUser(request, loaded); !ok || username != created.Username {
		t.Fatal("reiniciar backend com as mesmas credenciais deve preservar a sessao")
	}
	t.Setenv("ADMIN_PASSWORD", "nova-senha-de-teste")
	updated, err := loadOrCreateAdminAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := SessionUser(request, updated); ok {
		t.Fatal("trocar a senha deve invalidar a sessao persistente")
	}
}
