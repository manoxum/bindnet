package pangolin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDestinationUpdateUsesNginxAndPreservesDisabledTarget(t *testing.T) {
	s := testService(t)
	if _, err := s.db.Exec(`INSERT INTO pangolin_account_resources(account_id,resource_id,site_id) VALUES('test',12,7)`); err != nil {
		t.Fatal(err)
	}
	steps := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		steps = append(steps, r.Method+" "+r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/targets") {
			_, _ = w.Write([]byte(`{"success":true,"data":{"targets":[{"targetId":19,"siteId":7,"ip":"localhost","port":443,"method":"https","enabled":false}],"pagination":{"total":1}}}`))
			return
		}
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"success":true,"data":{"resourceId":12,"mode":"http","enabled":true}}`))
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path == "/v1/target/19" && (body["enabled"] != false || body["siteId"] != float64(7) || body["port"] != float64(443) || body["ip"] != "127.0.0.1") {
			t.Error("edição deve manter destino desativado e encaminhar pelo nginx")
		}
		if body["setHostHeader"] != nil && (body["setHostHeader"] != "new.local" || body["tlsServerName"] != "new.local") {
			t.Error("Host/SNI devem apontar ao domínio local")
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer server.Close()
	c := newClient(Config{ID: "test", APIURL: server.URL})
	req := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"localDomain":"new.local","method":"https"}`))
	req.SetPathValue("id", "12")
	w := httptest.NewRecorder()
	if err := s.updateDestination(w, req, c); err != nil {
		t.Fatal(err)
	}
	if w.Code != 204 || len(steps) != 6 || steps[2] != "POST /v1/public-resource/12" || steps[4] != "POST /v1/target/19" {
		t.Fatalf("atualização fora de ordem: %d %v", w.Code, steps)
	}
}

func TestCannotEnableResourceWithoutManagedTarget(t *testing.T) {
	s := testService(t)
	if _, err := s.db.Exec(`INSERT INTO pangolin_account_resources(account_id,resource_id,site_id) VALUES('test',12,7)`); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" {
			t.Error("recurso incompleto não pode ser ativado")
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"targets":[],"pagination":{"total":0}}}`))
	}))
	defer server.Close()
	c := newClient(Config{ID: "test", APIURL: server.URL})
	req := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"enabled":true}`))
	req.SetPathValue("id", "12")
	w := httptest.NewRecorder()
	if err := s.updateResource(w, req, c); err != nil {
		t.Fatal(err)
	}
	if w.Code != 409 || calls != 1 {
		t.Fatal("ativação de provisionamento incompleto deve ser recusada")
	}
}
