package pangolin

import (
	"context"
	"strconv"
	"strings"
)

// A rota de disponibilidade aparece no OpenAPI, mas não está exposta em todas
// as versões do Cloud. Reutiliza recursos e o catálogo confirmado do Cloud.
func (c *client) knownDomainOptions(ctx context.Context, subdomain string) (domainAvailability, error) {
	result := domainAvailability{Options: []domainOption{}, Message: "A Integration API não disponibiliza consulta global de nomes gratuitos. Estas opções vêm de recursos existentes; a disponibilidade será confirmada pelo Pangolin ao criar."}
	domains, err := c.domains(ctx)
	if err != nil {
		return result, err
	}
	custom := map[string]bool{}
	for _, domain := range domains {
		custom[domain.DomainID] = true
	}
	resources, err := c.resources(ctx)
	if err != nil {
		return result, err
	}
	occupied := map[string]bool{}
	for _, resource := range resources {
		occupied[resource.FullDomain] = true
	}
	seen := map[string]bool{}
	for _, resource := range resources {
		if resource.DomainID == "" || custom[resource.DomainID] || seen[resource.DomainID] || resource.Mode != "http" {
			continue
		}
		var detail Resource
		if err = c.call(ctx, "GET", "/public-resource/"+strconv.Itoa(resource.ResourceID), nil, &detail); err != nil {
			return result, err
		}
		prefix := detail.Subdomain + "."
		if detail.Subdomain == "" || !strings.HasPrefix(detail.FullDomain, prefix) {
			continue
		}
		namespace := strings.TrimPrefix(detail.FullDomain, prefix)
		if !validLocalDomain(namespace) {
			continue
		}
		seen[resource.DomainID] = true
		fullDomain := subdomain + "." + namespace
		if occupied[fullDomain] {
			continue
		}
		result.Options = append(result.Options, domainOption{FullDomain: fullDomain, DomainID: resource.DomainID, Namespace: namespace})
	}
	cloud := c.appendCloudDomainOptions(&result, subdomain, seen, occupied)
	result.Available = len(result.Options) > 0
	if !result.Available && !cloud {
		result.Message = "A Integration API não disponibiliza consulta de nomes gratuitos e não há opções reutilizáveis nesta conta. Crie o primeiro recurso com domínio gratuito no painel Pangolin e atualize o Bindnet, ou selecione um domínio próprio."
	}
	return result, nil
}
