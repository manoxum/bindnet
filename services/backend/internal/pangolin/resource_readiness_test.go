package pangolin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManualActivationRequiresOnlineSite(t *testing.T) {
	for _, online := range []bool{false, true} {
		t.Run(fmt.Sprint(online), func(t *testing.T) {
			s := testService(t)
			if _, err := s.db.Exec(`INSERT INTO pangolin_account_resources(account_id,resource_id,site_id) VALUES('test',12,7)`); err != nil {
				t.Fatal(err)
			}
			activated := false
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data := `{}`
				if strings.HasSuffix(r.URL.Path, "/targets") {
					data = `{"targets":[{"targetId":19,"siteId":7,"ip":"127.0.0.1","port":443,"enabled":true}],"pagination":{"total":1}}`
				} else if r.URL.Path == "/v1/site/7" {
					data = fmt.Sprintf(`{"siteId":7,"online":%t}`, online)
				} else if r.Method == "POST" {
					activated = true
				}
				_, _ = w.Write([]byte(`{"success":true,"data":` + data + `}`))
			}))
			defer remote.Close()
			r := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"enabled":true}`))
			r.SetPathValue("id", "12")
			w := httptest.NewRecorder()
			err := s.updateResource(w, r, newClient(Config{ID: "test", APIURL: remote.URL}))
			want := http.StatusConflict
			if online {
				want = http.StatusNoContent
			}
			if err != nil || w.Code != want || activated != online {
				t.Fatalf("site online=%t: status=%d, ativado=%t, erro=%v", online, w.Code, activated, err)
			}
		})
	}
}
