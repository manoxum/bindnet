package pangolin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountsIsolateRemoteIDsAndSecrets(t *testing.T) {
	s := testService(t)
	for _, query := range []string{
		`INSERT INTO pangolin_account_sites(account_id,site_id,name,newt_id,secret) VALUES('other',7,'Outro','other-id','OTHER-PRIVATE')`,
		`INSERT INTO pangolin_account_resources(account_id,resource_id,site_id,local_domain) VALUES('test',12,7,'a.local')`,
		`INSERT INTO pangolin_account_resources(account_id,resource_id,site_id,local_domain) VALUES('other',12,7,'b.local')`,
	} {
		if _, err := s.db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	ctx := t.Context()
	a, _ := loadConfig(ctx, s.db, "test")
	b, _ := loadConfig(ctx, s.db, "other")
	if a.APIKey != "SECRET-A" || b.APIKey != "SECRET-B" {
		t.Fatal("chaves devem estar isoladas por conta")
	}
	siteA, _ := s.localSite(ctx, "test", 7)
	siteB, _ := s.localSite(ctx, "other", 7)
	if siteA.Secret != "private-secret" || siteB.Secret != "OTHER-PRIVATE" {
		t.Fatal("siteId igual em contas diferentes não pode misturar credenciais")
	}
	domainsA, _ := s.resourceDomains(ctx, "test")
	domainsB, _ := s.resourceDomains(ctx, "other")
	if domainsA[12] != "a.local" || domainsB[12] != "b.local" {
		t.Fatal("resourceId igual não pode misturar domínios")
	}
	if connectorPath("test", 7) == connectorPath("other", 7) {
		t.Fatal("containers precisam de identidades diferentes")
	}
	w := httptest.NewRecorder()
	s.listAccounts(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatal("listagem de contas não pode expor chaves")
	}
}

func TestOtherAccountCannotMutateResource(t *testing.T) {
	s := testService(t)
	if _, err := s.db.Exec(`INSERT INTO pangolin_account_resources(account_id,resource_id,site_id) VALUES('test',12,7)`); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("DELETE", "/", nil)
	req.SetPathValue("id", "12")
	w := httptest.NewRecorder()
	err := s.deleteResource(w, req, newClient(Config{ID: "other"}))
	if err != nil || w.Code != http.StatusForbidden {
		t.Fatal("recurso de outra conta não pode ser removido")
	}
}

func TestAccountWithConnectorsCannotBeDeleted(t *testing.T) {
	s := testService(t)
	req := httptest.NewRequest("DELETE", "/", nil)
	req.SetPathValue("accountId", "test")
	w := httptest.NewRecorder()
	s.deleteAccount(w, req)
	if w.Code != 409 {
		t.Fatal("conta com sites não pode ser removida")
	}
	if _, err := loadConfig(req.Context(), s.db, "test"); err != nil {
		t.Fatal("configuração deve ser preservada")
	}
}
