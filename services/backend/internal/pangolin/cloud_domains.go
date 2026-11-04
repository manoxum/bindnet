package pangolin

import (
	"net/url"
	"strings"
)

// IDs confirmados por criação/remoção na Integration API em 02/11/2026.
// São alternativas do Cloud; não comprovam disponibilidade de um subdomínio.
var cloudDomainNamespaces = []string{"hostlocal.app", "tunneled.to", "tunnelmy.app", "visitmy.app"}

func (c *client) appendCloudDomainOptions(result *domainAvailability, subdomain string, seen, occupied map[string]bool) bool {
	origin, err := url.Parse(c.config.APIURL)
	if err != nil || origin.Scheme != "https" || !strings.EqualFold(origin.Hostname(), "api.pangolin.net") || (origin.Port() != "" && origin.Port() != "443") {
		return false
	}
	for _, namespace := range cloudDomainNamespaces {
		fullDomain := subdomain + "." + namespace
		if seen[namespace] || occupied[fullDomain] {
			continue
		}
		result.Options = append(result.Options, domainOption{DomainID: namespace, Namespace: namespace, FullDomain: fullDomain})
		seen[namespace] = true
	}
	result.Message = "Escolha um domínio gratuito conhecido do Pangolin Cloud. A API não consulta a disponibilidade antecipadamente; o nome será confirmado pelo Pangolin na criação."
	if len(result.Options) == 0 {
		result.Message = "Este subdomínio já está em uso nos domínios gratuitos conhecidos desta conta. Digite outro subdomínio público."
	}
	return true
}
