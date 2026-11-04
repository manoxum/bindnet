package pangolin

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"bindnet/backend/internal/workerapi"
)

func testWorker(t *testing.T, handler http.HandlerFunc) *workerapi.Client {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "worker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return workerapi.New(socket)
}

func TestCreateSitePublishesLocalNginxAndStartsContainer(t *testing.T) {
	s := testService(t)
	var mutex sync.Mutex
	steps := []string{}
	appendStep := func(value string) { mutex.Lock(); defer mutex.Unlock(); steps = append(steps, value) }
	s.worker = testWorker(t, func(w http.ResponseWriter, r *http.Request) {
		appendStep("worker start")
		if r.URL.Path != "/tunnels/test/8/start" {
			t.Errorf("rota do worker incorreta: %s", r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["newtId"] != "connector-id" || body["secret"] != "PRIVATE" || body["endpoint"] != "https://app.pangolin.net" {
			t.Error("conector precisa das credenciais geradas")
		}
		w.WriteHeader(204)
	})
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appendStep(r.Method + " " + r.URL.Path)
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		switch r.URL.Path {
		case "/v1/org/org/domains":
			_, _ = w.Write([]byte(`{"success":true,"data":{"domains":[{"domainId":"domain","type":"ns"}],"pagination":{"total":1}}}`))
		case "/v1/org/org/site":
			if body["name"] != "Minha aplicação" || body["type"] != "newt" {
				t.Error("nome/tipo incorretos")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"siteId":8,"newtId":"connector-id","secret":"PRIVATE"}}`))
		case "/v1/site/8":
			_, _ = w.Write([]byte(`{"success":true,"data":{"siteId":8,"online":true}}`))
		case "/v1/org/org/public-resource":
			if body["domainId"] != "domain" || body["subdomain"] != "public-app" {
				t.Error("nome público incorreto")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"resourceId":12,"fullDomain":"public-app.example.com","sso":true}}`))
		case "/v1/public-resource/12":
			if body["sso"] != nil {
				t.Error("não deve remover autenticação")
			}
			if body["enabled"] == false && (body["setHostHeader"] != "app.empresa.local" || body["tlsServerName"] != "app.empresa.local") {
				t.Error("Host e SNI precisam selecionar o virtual host local")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{}}`))
		case "/v1/public-resource/12/target":
			if body["siteId"] != float64(8) || body["ip"] != "127.0.0.1" || body["port"] != float64(443) || body["method"] != "https" {
				t.Error("destino precisa passar pelo nginx do host")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"targetId":19}}`))
		default:
			t.Errorf("rota inesperada: %s", r.URL.Path)
		}
	}))
	defer remote.Close()
	c := newClient(Config{ID: "test", APIURL: remote.URL, OrgID: "org", Endpoint: "https://app.pangolin.net", APIKey: "key"})
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Minha aplicação","domainId":"domain","subdomain":"public-app","localDomain":"APP.empresa.local."}`))
	w := httptest.NewRecorder()
	if err := s.createSite(w, req, c); err != nil {
		t.Fatal(err)
	}
	if w.Code != 201 || strings.Contains(w.Body.String(), "PRIVATE") {
		t.Fatalf("resposta inválida: %d %s", w.Code, w.Body.String())
	}
	expected := "GET /v1/org/org/domains,PUT /v1/org/org/site,worker start,GET /v1/site/8,PUT /v1/org/org/public-resource,POST /v1/public-resource/12,PUT /v1/public-resource/12/target,POST /v1/public-resource/12"
	if strings.Join(steps, ",") != expected {
		t.Fatalf("provisionamento fora de ordem: %v", steps)
	}
	domains, err := s.resourceDomains(req.Context(), "test")
	if err != nil || domains[12] != "app.empresa.local" {
		t.Fatal("domínio local precisa persistir")
	}
	status := httptest.NewRecorder()
	if err = s.publicationStatus(status, req, c); err != nil {
		t.Fatal(err)
	}
	var progress publicationProgress
	if err = json.Unmarshal(status.Body.Bytes(), &progress); err != nil || progress.State != "complete" || progress.SiteID != 8 || progress.ResourceID != 12 {
		t.Fatalf("progresso final precisa confirmar os objetos criados: %s, %v", status.Body, err)
	}
	other := httptest.NewRecorder()
	_ = s.publicationStatus(other, req, newClient(Config{ID: "other"}))
	if strings.Contains(other.Body.String(), "siteId") || !strings.Contains(other.Body.String(), `"state":"idle"`) {
		t.Fatal("progresso não pode vazar entre contas")
	}
}

func TestCreateSiteRequiresLocalDomainBeforeRemoteMutation(t *testing.T) {
	s := testService(t)
	for _, local := range []string{"", "https://app.local/path", "app.local\r\nInjected:true", "*.app.local", "127.0.0.1", "app..local"} {
		body, _ := json.Marshal(map[string]string{"name": "App", "domainId": "d", "localDomain": local})
		req := httptest.NewRequest("POST", "/", strings.NewReader(string(body)))
		w := httptest.NewRecorder()
		err := s.createSite(w, req, newClient(Config{}))
		if err != nil || w.Code != 400 {
			t.Errorf("DNS local inválido aceito: %q, status=%d, err=%v", local, w.Code, err)
		}
	}
}
