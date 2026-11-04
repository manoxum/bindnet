package pangolin

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

type testAudit struct{ events []string }

func (a *testAudit) Record(_ context.Context, event, _ string, _ map[string]any) {
	a.events = append(a.events, event)
}

func testService(t *testing.T) *service {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	for _, query := range []string{
		`CREATE TABLE pangolin_accounts(id TEXT PRIMARY KEY,name TEXT,api_url TEXT,endpoint TEXT,org_id TEXT,api_key TEXT,updated_at TEXT)`,
		`INSERT INTO pangolin_accounts VALUES('test','Conta A','https://api.pangolin.net','https://app.pangolin.net','org-a','SECRET-A',NULL)`,
		`INSERT INTO pangolin_accounts VALUES('other','Conta B','https://api.pangolin.net','https://app.pangolin.net','org-b','SECRET-B',NULL)`,
		`CREATE TABLE pangolin_account_sites(account_id TEXT,site_id INTEGER,name TEXT,newt_id TEXT,secret TEXT,desired_running BOOLEAN NOT NULL DEFAULT true,UNIQUE(account_id,site_id))`,
		`CREATE TABLE pangolin_account_resources(account_id TEXT,resource_id INTEGER,site_id INTEGER,local_domain TEXT NOT NULL DEFAULT '',UNIQUE(account_id,resource_id))`,
		`INSERT INTO pangolin_account_sites(account_id,site_id,name,newt_id,secret) VALUES('test',7,'Local','newt-id','private-secret')`,
	} {
		if _, err = db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	return &service{db: db, audit: &testAudit{}}
}

func TestCreateResourceStagesBeforeTargetAndKeepsSSO(t *testing.T) {
	s := testService(t)
	var steps []string
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		steps = append(steps, r.Method+" "+r.URL.Path)
		var body map[string]any
		if r.Method != "GET" {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		switch r.URL.Path {
		case "/v1/org/org/domains":
			_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[{"domainId":"domain","baseDomain":"example.com","type":"ns"}],"pagination":{"total":1}}}`))
		case "/v1/org/org/public-resource":
			if body["sso"] != nil || body["enabled"] != nil || body["mode"] != "http" {
				t.Error("criação deve preservar SSO e usar schema oficial")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"resourceId":12,"enabled":true,"sso":true}}`))
		case "/v1/public-resource/12":
			if body["enabled"] != false {
				t.Error("recurso deve ser desativado antes do destino")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
		case "/v1/public-resource/12/target":
			if body["siteId"] != float64(7) || body["method"] != "http" {
				t.Error("destino precisa de site e esquema")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"targetId":19}}`))
		default:
			t.Errorf("rota não esperada: %s", r.URL.Path)
		}
	}))
	defer remote.Close()
	c := newClient(Config{ID: "test", APIURL: remote.URL, OrgID: "org", APIKey: "key"})
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Portal","siteId":7,"domainId":"domain","localDomain":"portal.local","method":"http"}`))
	w := httptest.NewRecorder()
	if err := s.createResource(w, req, c); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 {
		t.Fatalf("status: %d, %s", w.Code, w.Body.String())
	}
	if len(steps) != 4 || !strings.Contains(steps[2], "POST /v1/public-resource/12") || !strings.Contains(steps[3], "/target") {
		t.Fatalf("sequência incorreta: %v", steps)
	}
	var resource Resource
	_ = json.Unmarshal(w.Body.Bytes(), &resource)
	if resource.Enabled || !resource.Managed {
		t.Fatal("recurso deve permanecer desativado e gerido")
	}
	ids, err := s.resourceIDs(req.Context(), "test")
	if err != nil || ids[12] != 7 {
		t.Fatal("recurso deve ser persistido com o site")
	}
}

func TestFailedTargetKeepsRecoverableIDWhenCompensationFails(t *testing.T) {
	s := testService(t)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/domains"):
			_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[{"domainId":"domain"}],"pagination":{"total":1}}}`))
		case r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/public-resource"):
			_, _ = w.Write([]byte(`{"success":true,"data":{"resourceId":12}}`))
		case r.Method == "POST":
			_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
		default:
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"message":"PRIVATE"}`))
		}
	}))
	defer remote.Close()
	c := newClient(Config{ID: "test", APIURL: remote.URL, OrgID: "org", APIKey: "key"})
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Portal","siteId":7,"domainId":"domain","localDomain":"portal.local","method":"http"}`))
	err := s.createResource(httptest.NewRecorder(), req, c)
	if err == nil || !strings.Contains(err.Error(), "12") || strings.Contains(err.Error(), "PRIVATE") {
		t.Fatalf("falha parcial deve identificar recurso sem vazar resposta: %v", err)
	}
	ids, _ := s.resourceIDs(req.Context(), "test")
	if ids[12] != 7 {
		t.Fatal("recurso órfão precisa permanecer rastreável")
	}
}

func TestTunnelRoutesRequireSession(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, nil, nil, nil)
	for _, route := range []string{"GET /api/tunnels/config", "PUT /api/tunnels/config", "POST /api/tunnels/sites", "DELETE /api/tunnels/sites/1", "POST /api/tunnels/sites/1/start", "POST /api/tunnels/resources", "PATCH /api/tunnels/resources/1", "PATCH /api/tunnels/resources/1/destination"} {
		parts := strings.SplitN(route, " ", 2)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(parts[0], strings.Replace(parts[1], "/api/tunnels/", "/api/tunnels/accounts/test/", 1), nil))
		if w.Code != 401 {
			t.Errorf("rota não protegida: %s (%d)", route, w.Code)
		}
	}
}
