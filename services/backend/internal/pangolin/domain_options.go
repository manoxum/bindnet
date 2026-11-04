package pangolin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type domainOption struct {
	FullDomain string `json:"fullDomain"`
	DomainID   string `json:"domainId"`
	Namespace  string `json:"domainNamespaceId"`
}
type domainAvailability struct {
	AvailabilityKnown bool           `json:"availabilityKnown"`
	Message           string         `json:"message,omitempty"`
	Available         bool           `json:"available"`
	Options           []domainOption `json:"options"`
}

func (c *client) domainOptions(ctx context.Context, subdomain string) (domainAvailability, error) {
	result := domainAvailability{Options: []domainOption{}}
	if len(subdomain) > 63 || !subdomainPattern.MatchString(subdomain) {
		return result, fmt.Errorf("informe um subdomínio válido para consultar os domínios gratuitos")
	}
	err := c.call(ctx, "GET", "/domain/check-namespace-availability?subdomain="+url.QueryEscape(subdomain), nil, &result)
	if result.Options == nil {
		result.Options = []domainOption{}
	}
	if remote, ok := err.(*apiError); ok && remote.status == 404 {
		return c.knownDomainOptions(ctx, subdomain)
	}
	result.AvailabilityKnown = err == nil
	return result, err
}
func (s *service) listDomainOptions(w http.ResponseWriter, r *http.Request, c *client) error {
	subdomain := r.URL.Query().Get("subdomain")
	if len(subdomain) > 63 || !subdomainPattern.MatchString(subdomain) {
		http.Error(w, "subdomínio inválido", 400)
		return nil
	}
	options, err := c.domainOptions(r.Context(), subdomain)
	if err != nil {
		return err
	}
	writeJSON(w, options)
	return nil
}
