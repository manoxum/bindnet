package pangolin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientContractPaginationAndSecretFiltering(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/org/bindnet/sites" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("contrato de autenticação/caminho inválido")
		}
		page := r.URL.Query().Get("page")
		if page == "1" {
			_, _ = w.Write([]byte(`{"success":true,"error":false,"data":{"sites":[{"siteId":1,"name":"Primeiro","secret":"remote-secret","newtId":"remote-id"}],"pagination":{"total":2}}}`))
		} else {
			_, _ = w.Write([]byte(`{"success":true,"error":false,"data":{"sites":[{"siteId":2,"name":"Segundo"}],"pagination":{"total":2}}}`))
		}
	}))
	defer server.Close()
	c := newClient(Config{APIURL: server.URL, OrgID: "bindnet", APIKey: "test-key"})
	sites, err := c.sites(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 2 || calls != 2 {
		t.Fatalf("paginação incorreta: %d sites, %d chamadas", len(sites), calls)
	}
	data, _ := json.Marshal(sites)
	if strings.Contains(string(data), "remote-secret") || strings.Contains(string(data), "remote-id") {
		t.Fatal("credenciais não podem ir ao navegador")
	}
	c.config.Configured = true
	data, _ = json.Marshal(c.config)
	if strings.Contains(string(data), "test-key") {
		t.Fatal("a chave da API não pode ir ao navegador")
	}
}

func TestClientRejectsFailureEnvelopesAndRedactsErrors(t *testing.T) {
	for _, status := range []int{200, 401} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"success":false,"error":true,"message":"SECRET","data":null}`))
			}))
			defer server.Close()
			c := newClient(Config{APIURL: server.URL, APIKey: "key"})
			err := c.call(context.Background(), "POST", "/public-resource/1", map[string]bool{"enabled": true}, nil)
			if err == nil || strings.Contains(err.Error(), "SECRET") {
				t.Fatalf("erro remoto não tratado/redigido: %v", err)
			}
		})
	}
}

func TestClientDoesNotFollowRedirectsOrRetryMutations(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; http.Redirect(w, r, "/other", 307) }))
	defer server.Close()
	c := newClient(Config{APIURL: server.URL, APIKey: "key"})
	if err := c.call(context.Background(), "PUT", "/org/x/site", map[string]string{"name": "local"}, nil); err == nil {
		t.Fatal("redirecionamento deve falhar")
	}
	if calls != 1 {
		t.Fatal("mutação não pode ser repetida ou redirecionada")
	}
}

func TestConfigAndTargetValidation(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "https://user:pass@host", "https://host/v1", "https://host?key=secret", "https://host/#fragment"} {
		if validateURL(raw) == nil {
			t.Errorf("URL insegura aceita: %s", raw)
		}
	}
	v := resourceRequest{Name: "Portal", Mode: "http", SiteID: 1, DomainID: "domain", IP: "::1", Port: 8080, Method: "http"}
	if err := v.validate(); err != nil {
		t.Fatal(err)
	}
	if v.remoteBody()["subdomain"] != nil {
		t.Fatal("domínio base deve enviar subdomain nulo")
	}
	v.IP = "http://localhost/path"
	if v.validate() == nil {
		t.Fatal("destino com caminho não pode ser aceito")
	}
}
