// Package publication valida e gera configuracoes de sites para integracoes.
package publication

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type Request struct {
	Name       string   `json:"name"`
	Domains    []string `json:"domains"`
	Alias      string   `json:"alias"`
	Port       int      `json:"port"`
	SSL        string   `json:"ssl"`
	ForceHTTPS bool     `json:"forceHttps"`
}

var identifier = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_.-]{0,61}[a-z0-9])?$`)
var dnsLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func validDNS(value string) bool {
	if len(value) == 0 || len(value) > 253 {
		return false
	}
	for _, part := range strings.Split(value, ".") {
		if !dnsLabel.MatchString(part) {
			return false
		}
	}
	return true
}

func (r *Request) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.Alias = strings.ToLower(strings.TrimSpace(r.Alias))
	if !identifier.MatchString(r.Name) || !validDNS(r.Alias) {
		return errors.New("nome de configuração ou alias inválido")
	}
	if r.Port < 1 || r.Port > 65535 {
		return errors.New("porta de upstream inválida")
	}
	if len(r.Domains) == 0 || len(r.Domains) > 100 {
		return errors.New("indique entre 1 e 100 domínios")
	}
	seen := map[string]bool{}
	for i, domain := range r.Domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if !validDNS(domain) || seen[domain] {
			return errors.New("domínio inválido ou repetido")
		}
		seen[domain] = true
		r.Domains[i] = domain
	}
	switch r.SSL {
	case "none", "letsencrypt", "bindnet", "selfsigned":
	default:
		return errors.New("modo SSL inválido")
	}
	if r.SSL == "none" && r.ForceHTTPS {
		return errors.New("redirecionamento HTTPS requer SSL")
	}
	return nil
}

// Render exige caminhos de certificado determinados pelo servidor, nunca pelo cliente.
func Render(r Request, certificatePath, keyPath string) (string, error) {
	if err := r.Validate(); err != nil {
		return "", err
	}
	if r.SSL != "none" && (!safeCertificatePath(certificatePath) || !safeCertificatePath(keyPath)) {
		return "", errors.New("caminhos de certificado inválidos")
	}
	domains := strings.Join(r.Domains, " ")
	httpLocation := proxyLocation()
	if r.ForceHTTPS {
		httpLocation = "    location / { return 308 https://$host$request_uri; }\n"
	}
	challenge := ""
	if r.SSL == "letsencrypt" {
		challenge = "    location ^~ /.well-known/acme-challenge/ {\n        proxy_set_header Host $host;\n        proxy_set_header X-Real-IP $remote_addr;\n        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n        proxy_pass http://127.0.0.1:9180;\n    }\n"
	}
	upstream := fmt.Sprintf("    resolver 127.0.0.11 valid=10s ipv6=off;\n    resolver_timeout 5s;\n    set $upstream_frontend http://%s:%d;\n    set $connection_upgrade close;\n    if ($http_upgrade != \"\") { set $connection_upgrade upgrade; }\n", r.Alias, r.Port)
	httpServer := fmt.Sprintf("server {\n    listen 80;\n    listen [::]:80;\n    server_name %s;\n%s%s%s}\n", domains, upstream, challenge, httpLocation)
	if r.SSL == "none" {
		return httpServer, nil
	}
	httpsServer := fmt.Sprintf("\nserver {\n    listen 443 ssl;\n    listen [::]:443 ssl;\n    server_name %s;\n    ssl_certificate %s;\n    ssl_certificate_key %s;\n    ssl_protocols TLSv1.2 TLSv1.3;\n%s%s}\n", domains, certificatePath, keyPath, upstream, proxyLocation())
	return httpServer + httpsServer, nil
}

func safeCertificatePath(path string) bool {
	return strings.HasPrefix(path, "/etc/nginx/") && !strings.Contains(path, "..") && regexp.MustCompile(`^/etc/nginx/[a-zA-Z0-9_./-]+$`).MatchString(path)
}

func proxyLocation() string {
	return "    location / {\n        proxy_pass $upstream_frontend;\n        proxy_http_version 1.1;\n        proxy_set_header Host $host;\n        proxy_set_header X-Real-IP $remote_addr;\n        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n        proxy_set_header X-Forwarded-Proto $scheme;\n        proxy_set_header Upgrade $http_upgrade;\n        proxy_set_header Connection $connection_upgrade;\n        proxy_read_timeout 60s;\n    }\n"
}
