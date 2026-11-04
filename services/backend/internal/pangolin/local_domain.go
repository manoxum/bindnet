package pangolin

import (
	"context"
	"fmt"
	"net"
	"strings"
)

func validLocalDomain(value string) bool {
	if len(value) > 253 || !strings.Contains(value, ".") || net.ParseIP(value) != nil {
		return false
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || !subdomainPattern.MatchString(label) {
			return false
		}
	}
	return true
}

func (s *service) resourceDomains(ctx context.Context, account string) (map[int]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT resource_id,local_domain FROM pangolin_account_resources WHERE account_id=$1`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int]string{}
	for rows.Next() {
		var id int
		var domain string
		if err = rows.Scan(&id, &domain); err != nil {
			return nil, err
		}
		result[id] = domain
	}
	return result, rows.Err()
}

// publicationRequest contém só o domínio local; IP e portas são definidos pelo Bindnet.
type publicationRequest struct {
	Name        string `json:"name"`
	DomainID    string `json:"domainId"`
	Subdomain   string `json:"subdomain"`
	LocalDomain string `json:"localDomain"`
	Method      string `json:"method"`
	SiteID      int    `json:"siteId"`
}

func (input publicationRequest) resource(siteID int) (resourceRequest, error) {
	method := input.Method
	if method == "" {
		method = "https"
	}
	port := 443
	if method == "http" {
		port = 80
	}
	value := resourceRequest{Name: strings.TrimSpace(input.Name), Mode: "http", SiteID: siteID, DomainID: input.DomainID, Subdomain: strings.TrimSpace(input.Subdomain), LocalDomain: strings.ToLower(strings.TrimSuffix(strings.TrimSpace(input.LocalDomain), ".")), Method: method, IP: "127.0.0.1", Port: port}
	if value.LocalDomain == "" {
		return value, fmt.Errorf("informe o domínio DNS local servido pelo nginx")
	}
	if method != "http" && method != "https" {
		return value, fmt.Errorf("selecione HTTP ou HTTPS para o nginx local")
	}
	return value, value.validate()
}
