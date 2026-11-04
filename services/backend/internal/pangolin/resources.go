package pangolin

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type Target struct {
	TargetID int    `json:"targetId"`
	SiteID   int    `json:"siteId"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Method   string `json:"method"`
	Enabled  bool   `json:"enabled"`
}

type Resource struct {
	DomainID    string   `json:"domainId"`
	Subdomain   string   `json:"subdomain"`
	ResourceID  int      `json:"resourceId"`
	Name        string   `json:"name"`
	LocalDomain string   `json:"localDomain,omitempty"`
	Mode        string   `json:"mode"`
	FullDomain  string   `json:"fullDomain"`
	ProxyPort   *int     `json:"proxyPort"`
	Enabled     bool     `json:"enabled"`
	SSO         bool     `json:"sso"`
	Managed     bool     `json:"managed"`
	Targets     []Target `json:"targets"`
}

type Domain struct {
	DomainID   string `json:"domainId"`
	BaseDomain string `json:"baseDomain"`
	Verified   bool   `json:"verified"`
	Type       string `json:"type"`
}

type resourceRequest struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	DomainID    string `json:"domainId"`
	Subdomain   string `json:"subdomain"`
	ProxyPort   int    `json:"proxyPort"`
	SiteID      int    `json:"siteId"`
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	Method      string `json:"method"`
	LocalDomain string `json:"localDomain"`
}

var hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)
var subdomainPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func (v resourceRequest) validate() error {
	if strings.TrimSpace(v.Name) == "" || len(v.Name) > 100 {
		return fmt.Errorf("informe um nome de até 100 caracteres")
	}
	if v.Mode != "http" && v.Mode != "tcp" && v.Mode != "udp" {
		return fmt.Errorf("protocolo inválido")
	}
	if v.SiteID <= 0 || v.Port < 1 || v.Port > 65535 {
		return fmt.Errorf("site ou porta de destino inválidos")
	}
	if len(v.IP) > 253 || (net.ParseIP(v.IP) == nil && !hostnamePattern.MatchString(v.IP)) {
		return fmt.Errorf("informe somente o IP ou hostname de destino, sem esquema ou caminho")
	}
	if v.LocalDomain != "" {
		if v.Mode != "http" || !validLocalDomain(v.LocalDomain) {
			return fmt.Errorf("domínio local inválido; informe somente o nome DNS do site nginx")
		}
	}
	if v.Mode == "http" {
		if v.DomainID == "" || (v.Subdomain != "" && (len(v.Subdomain) > 63 || !subdomainPattern.MatchString(v.Subdomain))) {
			return fmt.Errorf("domínio ou subdomínio inválido")
		}
		if v.Method != "http" && v.Method != "https" && v.Method != "h2c" {
			return fmt.Errorf("esquema HTTP de destino inválido")
		}
	} else if v.ProxyPort < 1 || v.ProxyPort > 65535 {
		return fmt.Errorf("informe a porta pública entre 1 e 65535")
	}
	return nil
}

func (v resourceRequest) remoteBody() map[string]any {
	body := map[string]any{"name": v.Name, "mode": v.Mode}
	if v.Mode == "http" {
		body["domainId"] = v.DomainID
		if v.Subdomain != "" {
			body["subdomain"] = v.Subdomain
		} else {
			body["subdomain"] = nil
		}
	} else {
		body["proxyPort"] = v.ProxyPort
	}
	return body
}

func (v resourceRequest) targetBody() map[string]any {
	body := map[string]any{"siteId": v.SiteID, "ip": v.IP, "port": v.Port, "enabled": true}
	if v.Mode == "http" {
		body["method"] = v.Method
	}
	return body
}
