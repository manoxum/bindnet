package portal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeWorker struct {
	local bool
	err   error
	path  string
}

func (f *fakeWorker) Call(_ context.Context, method, path string, body, dest any) error {
	if method != http.MethodGet || body != nil {
		return errors.New("consulta deve ser somente leitura")
	}
	f.path = path
	if f.err != nil {
		return f.err
	}
	data, _ := json.Marshal(map[string]bool{"local": f.local})
	return json.Unmarshal(data, dest)
}

func TestPortalAccess(t *testing.T) {
	for _, tc := range []struct {
		name      string
		enabled   bool
		forwarded string
		remote    string
		local     bool
		workerErr error
		code      int
		path      string
	}{
		{"host via proxies", true, "10.91.0.1, 10.91.0.5", "10.91.0.6:8080", true, nil, 403, "/network/local-client?ip=10.91.0.1"},
		{"hotspot via proxies", true, "192.168.12.20, 10.91.0.5", "10.91.0.6:8080", false, nil, 204, "/network/local-client?ip=192.168.12.20"},
		{"loopback direto", true, "", "127.0.0.1:8080", true, nil, 403, "/network/local-client?ip=127.0.0.1"},
		{"IPv6 direto", true, "", "[::1]:8080", true, nil, 403, "/network/local-client?ip=%3A%3A1"},
		{"desligado", false, "127.0.0.1", "", true, nil, 204, ""},
		{"IP invalido", true, "invalido", "", true, nil, 204, ""},
		{"worker indisponivel", true, "127.0.0.1", "", true, errors.New("indisponivel"), 204, "/network/local-client?ip=127.0.0.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			worker := &fakeWorker{local: tc.local, err: tc.workerErr}
			req := httptest.NewRequest(http.MethodGet, "/api/portal/access", nil)
			req.RemoteAddr = tc.remote
			req.Header.Set("X-Forwarded-For", tc.forwarded)
			recorder := httptest.NewRecorder()
			accessHandler(worker, tc.enabled)(recorder, req)
			if recorder.Code != tc.code || worker.path != tc.path {
				t.Fatalf("status=%d caminho=%q; esperado %d %q", recorder.Code, worker.path, tc.code, tc.path)
			}
			if recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("decisao por origem nao pode ser cacheada")
			}
		})
	}
}

func TestPortalRedirectConfiguration(t *testing.T) {
	for _, tc := range []struct {
		value string
		code  int
	}{
		{"", http.StatusForbidden}, {"true", http.StatusForbidden}, {"false", http.StatusNoContent},
	} {
		t.Setenv("PORTAL_HOST_REDIRECT_ENABLED", tc.value)
		mux := http.NewServeMux()
		if err := RegisterRoutes(mux, &fakeWorker{local: true}); err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/portal/access", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		mux.ServeHTTP(recorder, req)
		if recorder.Code != tc.code {
			t.Fatalf("configuracao %q retornou %d, esperado %d", tc.value, recorder.Code, tc.code)
		}
	}
	t.Setenv("PORTAL_HOST_REDIRECT_ENABLED", "invalido")
	if err := RegisterRoutes(http.NewServeMux(), &fakeWorker{}); err == nil {
		t.Fatal("configuracao invalida deve falhar explicitamente")
	}
}
