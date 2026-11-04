package pangolin

import (
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	APIURL     string `json:"apiUrl"`
	Endpoint   string `json:"endpoint"`
	OrgID      string `json:"orgId"`
	APIKey     string `json:"-"`
	Configured bool   `json:"configured"`
}

type Site struct {
	LocalDomain  string `json:"localDomain,omitempty"`
	PublicDomain string `json:"publicDomain,omitempty"`
	ResourceID   int    `json:"resourceId,omitempty"`
	SiteID       int    `json:"siteId"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Online       bool   `json:"online"`
	Managed      bool   `json:"managed"`
	NewtID       string `json:"-"`
	Secret       string `json:"-"`
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("informe a origem HTTP(S), sem caminho, credenciais, parâmetros ou fragmento")
	}
	return nil
}

func (c Config) validate() error {
	if err := validateURL(c.APIURL); err != nil {
		return fmt.Errorf("URL da API: %w", err)
	}
	if err := validateURL(c.Endpoint); err != nil {
		return fmt.Errorf("endpoint do conector: %w", err)
	}
	if c.OrgID == "" || len(c.OrgID) > 100 || strings.ContainsAny(c.OrgID, "/\\?#%") || c.OrgID == "." || c.OrgID == ".." {
		return fmt.Errorf("identificador da organização inválido")
	}
	if c.APIKey == "" || strings.ContainsAny(c.APIKey, "\r\n") {
		return fmt.Errorf("informe uma chave da Integration API")
	}
	return nil
}
