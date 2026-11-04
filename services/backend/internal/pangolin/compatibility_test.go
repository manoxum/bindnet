package pangolin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudDomainsAcceptStringPaginationTotal(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[],"pagination":{"total":"0","limit":100,"offset":0}}}`))
	}))
	defer remote.Close()
	result, err := newClient(Config{APIURL: remote.URL}).domains(context.Background())
	if err != nil || len(result) != 0 {
		t.Fatalf("resposta real do Cloud deve ser aceita: %v", err)
	}
}

func TestFreeDomain404UsesNamespacesFromExistingResources(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/domain/check-namespace-availability":
			w.WriteHeader(404)
		case "/v1/org/org/domains":
			_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[],"pagination":{"total":"0"}}}`))
		case "/v1/org/org/public-resources":
			_, _ = w.Write([]byte(`{"success":true,"data":{"resources":[{"resourceId":12,"domainId":"shared","fullDomain":"old.hostlocal.app","mode":"http"}],"pagination":{"total":1}}}`))
		case "/v1/public-resource/12":
			_, _ = w.Write([]byte(`{"success":true,"data":{"resourceId":12,"domainId":"shared","fullDomain":"old.hostlocal.app","subdomain":"old","mode":"http"}}`))
		default:
			t.Errorf("rota inesperada: %s", r.URL.Path)
		}
	}))
	defer remote.Close()
	c := newClient(Config{APIURL: remote.URL, OrgID: "org"})
	result, err := c.domainOptions(context.Background(), "new")
	if err != nil || len(result.Options) != 1 {
		t.Fatalf("404 não deve bloquear o fluxo de namespaces conhecidos: %+v, %v", result, err)
	}
	if result.Options[0].FullDomain != "new.hostlocal.app" || result.AvailabilityKnown {
		t.Fatal("opção inferida não pode prometer disponibilidade global")
	}
}
