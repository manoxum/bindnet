package pangolin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResourceSettingsRedactSecretsAndRejectArbitraryFields(t *testing.T) {
	s := testService(t)
	if _, err := s.db.Exec(`INSERT INTO pangolin_account_resources(account_id,resource_id,site_id) VALUES('test',12,7)`); err != nil {
		t.Fatal(err)
	}
	calls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"success":true,"data":{"name":"Portal","sso":true,"password":"SECRET","pincode":"123456","headerAuthPassword":"SECRET","maintenanceTitle":"Breve","setHostHeader":"app.local"}}`))
	}))
	defer remote.Close()
	c := newClient(Config{ID: "test", APIURL: remote.URL})
	request := func(method, body string) *http.Request {
		r := httptest.NewRequest(method, "/", strings.NewReader(body))
		r.SetPathValue("id", "12")
		return r
	}
	w := httptest.NewRecorder()
	if err := s.getResourceSettings(w, request("GET", ""), c); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(w.Body.String(), "SECRET") || strings.Contains(w.Body.String(), "123456") || !strings.Contains(w.Body.String(), "app.local") {
		t.Fatalf("leitura de configurações insegura: %s", w.Body.String())
	}
	for _, body := range []string{`{"setHostHeader":"evil.local"}`, `{"cmd":"arbitrary"}`, `{"sso":"false"}`, `{"requestHeaders":[{"name":"Host","value":"evil.local"}]}`, `{"responseHeaders":[{"name":"X-Test","value":"line\r\nInjected:true"}]}`, `{"maintenanceModeType":"invalid"}`} {
		w = httptest.NewRecorder()
		if err := s.updateResourceSettings(w, request("PATCH", body), c); err != nil {
			t.Fatal(err)
		}
		if w.Code != 400 {
			t.Errorf("configuração inválida aceita: %s (%d)", body, w.Code)
		}
	}
	if calls != 1 {
		t.Fatal("corpos inválidos não devem chegar à API Pangolin")
	}
}

func TestAuthMethodValidationAndSecretNullRemoval(t *testing.T) {
	for _, test := range []struct {
		method, body string
		valid        bool
	}{
		{"pincode", `{"pincode":"123456"}`, true}, {"pincode", `{"pincode":null}`, true}, {"pincode", `{"pincode":"123"}`, false},
		{"password", `{"password":"secret-value"}`, true}, {"password", `{"password":null}`, true}, {"password", `{"password":""}`, false},
		{"header-auth", `{"user":null,"password":null,"extendedCompatibility":false}`, true},
		{"whitelist", `{"emails":["a@example.com","*@example.org"]}`, true}, {"whitelist", `{"emails":["invalid"]}`, false},
		{"roles", `{"roleIds":[-1]}`, false}, {"users", `{"userIds":["user-id"]}`, true}, {"arbitrary", `{}`, false},
	} {
		var body map[string]json.RawMessage
		_ = json.Unmarshal([]byte(test.body), &body)
		if (validateAuthMethod(test.method, body) == nil) != test.valid {
			t.Errorf("validação incorreta: %s %s", test.method, test.body)
		}
	}
}

func TestUnknownAccountRoutesRemainAuthenticated(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil, nil, nil, nil)
	for _, path := range []string{"/api/tunnels/accounts", "/api/tunnels/accounts/test/resources/12/settings", "/api/tunnels/accounts/test/resources/12/rules", "/api/tunnels/accounts/test/domain-options?subdomain=app", "/api/tunnels/accounts/test/access-options/users"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 401 {
			t.Errorf("rota não protegida: %s (%d)", path, w.Code)
		}
	}
}
