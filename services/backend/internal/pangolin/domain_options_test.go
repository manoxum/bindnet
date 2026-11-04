package pangolin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFreeDomainSelectionUsesOfficialAvailabilityAndRechecks(t *testing.T) {
	checks := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/org/org/domains" {
			_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[],"pagination":{"total":0}}}`))
			return
		}
		if r.URL.Path != "/v1/domain/check-namespace-availability" || r.URL.Query().Get("subdomain") != "minha-app" {
			t.Errorf("rota de disponibilidade incorreta: %s", r.URL)
		}
		checks++
		if checks == 1 {
			_, _ = w.Write([]byte(`{"success":true,"data":{"available":true,"options":[{"fullDomain":"minha-app.hostlocal.app","domainId":"free-domain","domainNamespaceId":"hostlocal.app"}]}}`))
		} else {
			_, _ = w.Write([]byte(`{"success":true,"data":{"available":false,"options":[]}}`))
		}
	}))
	defer server.Close()
	c := newClient(Config{APIURL: server.URL, OrgID: "org"})
	options, err := c.domainOptions(context.Background(), "minha-app")
	if err != nil || len(options.Options) != 1 || options.Options[0].DomainID != "free-domain" {
		t.Fatalf("opções da API não preservadas: %v %v", options, err)
	}
	err = c.validateDomain(context.Background(), resourceRequest{Mode: "http", DomainID: "free-domain", Subdomain: "minha-app"})
	if err == nil || !strings.Contains(err.Error(), "não está disponível") || checks != 2 {
		t.Fatal("nome deve ser revalidado antes de criar recurso/site")
	}
}

func TestFreeDomainAcceptsAvailableNamespaceWithoutOwnDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/domains") {
			_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[],"pagination":{"total":0}}}`))
		} else {
			_, _ = w.Write([]byte(`{"success":true,"data":{"available":true,"options":[{"fullDomain":"app.tunneled.to","domainId":"free-domain","domainNamespaceId":"tunneled.to"}]}}`))
		}
	}))
	defer server.Close()
	c := newClient(Config{APIURL: server.URL, OrgID: "org"})
	if err := c.validateDomain(context.Background(), resourceRequest{Mode: "http", DomainID: "free-domain", Subdomain: "app"}); err != nil {
		t.Fatal(err)
	}
}
