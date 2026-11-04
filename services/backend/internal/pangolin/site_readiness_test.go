package pangolin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOfflineConnectorDoesNotPublishOrReportSuccess(t *testing.T) {
	s := testService(t)
	s.worker = testWorker(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	activated, reads := false, 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := `{}`
		switch r.URL.Path {
		case "/v1/org/org/domains":
			data = `{"domains":[{"domainId":"domain","type":"ns"}],"pagination":{"total":1}}`
		case "/v1/org/org/site":
			data = `{"siteId":8,"newtId":"connector-id","secret":"PRIVATE"}`
		case "/v1/org/org/public-resource":
			data = `{"resourceId":12}`
		case "/v1/public-resource/12/target":
			data = `{"targetId":19}`
		case "/v1/public-resource/12":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			activated = activated || body["enabled"] == true
		case "/v1/site/8":
			reads++
			if reads > 1 {
				w.WriteHeader(503)
				return
			}
			data = `{"siteId":8,"online":false}`
		default:
			t.Errorf("rota inesperada: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"success":true,"data":` + data + `}`))
	}))
	defer remote.Close()
	c := newClient(Config{ID: "test", APIURL: remote.URL, OrgID: "org", Endpoint: "https://app.pangolin.net"})
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"App","domainId":"domain","subdomain":"app","localDomain":"app.local"}`))
	err := s.createSite(httptest.NewRecorder(), r, c)
	if err == nil || activated {
		t.Fatalf("container iniciado com site offline não deve ativar o recurso: %v", err)
	}
	if !strings.Contains(err.Error(), "nenhum recurso foi publicado") {
		t.Fatalf("falha deve orientar a recuperação sem recriar o site: %v", err)
	}
	progress := s.progress.accounts["test"]
	if progress.State != "failed" || progress.SiteID != 8 || progress.ResourceID != 0 {
		t.Fatalf("falha deve preservar identificadores recuperáveis: %+v", progress)
	}
	if _, err = s.localSite(r.Context(), "test", 8); err != nil {
		t.Fatal("site precisa continuar recuperável após falha de conexão")
	}
}
