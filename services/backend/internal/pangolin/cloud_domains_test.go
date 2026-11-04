package pangolin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testTransport func(*http.Request) (*http.Response, error)

func (fn testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestCloudFreeDomainsDoNotRequireAnExistingResource(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/domain/check-namespace-availability":
			w.WriteHeader(http.StatusNotFound)
		case "/v1/org/org/domains":
			_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[],"pagination":{"total":"0"}}}`))
		case "/v1/org/org/public-resources":
			_, _ = w.Write([]byte(`{"success":true,"data":{"resources":[],"pagination":{"total":0}}}`))
		default:
			t.Errorf("rota inesperada: %s", r.URL.Path)
		}
	}))
	defer remote.Close()
	c := newClient(Config{APIURL: "https://api.pangolin.net", OrgID: "org"})
	c.http.Transport = testTransport(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		u := *r.URL
		u.Scheme, u.Host = "http", strings.TrimPrefix(remote.URL, "http://")
		copy.URL = &u
		return http.DefaultTransport.RoundTrip(copy)
	})
	result, err := c.domainOptions(context.Background(), "webyomi")
	if err != nil || len(result.Options) == 0 {
		t.Fatalf("conta Cloud vazia não pode exigir selecionar um domínio sem oferecer opções: %+v, %v", result, err)
	}
	if result.AvailabilityKnown {
		t.Fatal("sem consulta global, disponibilidade só pode ser confirmada ao criar")
	}
	if len(result.Options) != 4 {
		t.Fatalf("os quatro namespaces Cloud confirmados devem ser oferecidos: %+v", result.Options)
	}
	for _, option := range result.Options {
		if err = c.validateDomain(context.Background(), resourceRequest{Mode: "http", DomainID: option.DomainID, Subdomain: "webyomi"}); err != nil {
			t.Errorf("opção exibida precisa ser aceita para criação: %s: %v", option.FullDomain, err)
		}
	}
	for _, origin := range []string{"http://api.pangolin.net", "https://api.pangolin.net.example.com", "https://pangolin.example.com"} {
		c.config.APIURL = origin
		result, err = c.domainOptions(context.Background(), "webyomi")
		if err != nil || len(result.Options) != 0 {
			t.Errorf("catálogo Cloud não deve ser injetado noutra origem: %s: %+v, %v", origin, result, err)
		}
	}
}

func TestCloudCatalogueSkipsDuplicateAndOccupiedNames(t *testing.T) {
	result := domainAvailability{Options: []domainOption{{DomainID: "hostlocal.app", FullDomain: "webyomi.hostlocal.app"}}}
	c := newClient(Config{APIURL: "https://api.pangolin.net"})
	c.appendCloudDomainOptions(&result, "webyomi", map[string]bool{"hostlocal.app": true}, map[string]bool{"webyomi.tunneled.to": true})
	if len(result.Options) != 3 {
		t.Fatalf("não pode duplicar namespace nem oferecer nome já usado: %+v", result.Options)
	}
	for _, option := range result.Options {
		if option.FullDomain == "webyomi.tunneled.to" {
			t.Fatal("nome ocupado foi oferecido")
		}
	}
}
