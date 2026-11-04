package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fakeDocker(fn func(*http.Request) (int, string)) *dockerClient {
	return &dockerClient{http: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, body := fn(r)
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
}

func TestStartCreatesOnlyRestrictedPinnedConnector(t *testing.T) {
	calls := []string{}
	d := fakeDocker(func(r *http.Request) (int, string) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/containers/"):
			return 404, `{}`
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/images/"):
			return 200, `{}`
		case r.URL.Path == "/containers/create":
			if r.URL.Query().Get("name") != "bindnet-newt-7" {
				t.Error("nome deve ser gerado pelo worker")
			}
			var cfg map[string]any
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				t.Fatal(err)
			}
			host := cfg["HostConfig"].(map[string]any)
			if cfg["Image"] != "fosrl/newt:1.19.0" || host["NetworkMode"] != "host" || host["Privileged"] != nil || host["Binds"] != nil || cfg["Cmd"] != nil {
				t.Fatal("configuração privilegiada/arbitrária no conector")
			}
			encoded, _ := json.Marshal(cfg)
			for _, flag := range []string{"DISABLE_CLIENTS=true", "DISABLE_SSH=true", "no-new-privileges:true", "CapDrop"} {
				if !strings.Contains(string(encoded), flag) {
					t.Errorf("proteção ausente: %s", flag)
				}
			}
			return 201, `{"Id":"connector"}`
		case strings.HasSuffix(r.URL.Path, "/start"):
			return 204, ``
		default:
			t.Errorf("rota Docker inesperada: %s", r.URL.Path)
			return 500, `{}`
		}
	})
	req := httptest.NewRequest("POST", "/tunnels/7/start", strings.NewReader(`{"endpoint":"https://app.pangolin.net","newtId":"newt","secret":"PRIVATE"}`))
	req.SetPathValue("id", "7")
	req.SetPathValue("accountId", "legacy")
	w := httptest.NewRecorder()
	handler(d, "start")(w, req)
	if w.Code != 204 || len(calls) != 4 {
		t.Fatalf("start falhou: %d %s (%v)", w.Code, w.Body.String(), calls)
	}
	if strings.Contains(w.Body.String(), "PRIVATE") {
		t.Fatal("credencial vazou na resposta")
	}
}

func TestRejectsForeignContainersAndInvalidIDs(t *testing.T) {
	calls := 0
	d := fakeDocker(func(r *http.Request) (int, string) {
		calls++
		return 200, `{"Id":"foreign","Config":{"Labels":{}},"State":{"Running":true}}`
	})
	for _, id := range []string{"0", "-1", "../hotspot", "--privileged", "7"} {
		req := httptest.NewRequest("POST", "/tunnels/test/stop", nil)
		req.SetPathValue("id", id)
		req.SetPathValue("accountId", "legacy")
		w := httptest.NewRecorder()
		handler(d, "stop")(w, req)
		if w.Code == 204 {
			t.Fatalf("operação indevida aceita: %s", id)
		}
	}
	if calls != 1 {
		t.Fatal("IDs inválidos não devem chegar ao Docker")
	}
}

func TestStartAndStopAreIdempotent(t *testing.T) {
	for _, running := range []bool{true, false} {
		calls := 0
		d := fakeDocker(func(r *http.Request) (int, string) {
			calls++
			return 200, fmt.Sprintf(`{"Id":"own","Config":{"Labels":{"%s":"true"}},"State":{"Running":%t}}`, ownerLabel, running)
		})
		action := "stop"
		body := ""
		if running {
			action = "start"
			body = `{"endpoint":"https://app.pangolin.net","newtId":"newt","secret":"private"}`
		}
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		req.SetPathValue("id", "7")
		req.SetPathValue("accountId", "legacy")
		w := httptest.NewRecorder()
		handler(d, action)(w, req)
		if w.Code != 204 || calls != 1 {
			t.Fatalf("ação %s deve ser idempotente: %d (%d chamadas)", action, w.Code, calls)
		}
	}
}

func TestPullRejectsErrorInDockerStream(t *testing.T) {
	d := fakeDocker(func(r *http.Request) (int, string) {
		if r.Method == "GET" {
			return 404, `{}`
		}
		return 200, "{\"status\":\"Pulling\"}\n{\"error\":\"PRIVATE REGISTRY DATA\"}\n"
	})
	err := d.ensureImage(context.Background())
	if err == nil || strings.Contains(err.Error(), "PRIVATE") {
		t.Fatalf("erro de pull deve falhar sem vazar detalhes: %v", err)
	}
}

func TestAccountContainerNamesNeverCollide(t *testing.T) {
	if containerName("first", 7) == containerName("second", 7) {
		t.Fatal("sites de contas diferentes não podem controlar o mesmo container")
	}
	if containerName("legacy", 7) != "bindnet-newt-7" {
		t.Fatal("integração legada deve manter o container existente")
	}
	calls := 0
	d := fakeDocker(func(r *http.Request) (int, string) { calls++; return 404, `{}` })
	for _, account := range []string{"", "../docker", "A", "--privileged"} {
		req := httptest.NewRequest("GET", "/", nil)
		req.SetPathValue("id", "7")
		req.SetPathValue("accountId", account)
		w := httptest.NewRecorder()
		handler(d, "status")(w, req)
		if w.Code != 400 {
			t.Errorf("conta inválida aceita: %q", account)
		}
	}
	if calls != 0 {
		t.Fatal("conta inválida não deve contactar o Docker")
	}
}
