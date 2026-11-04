package pangolin

import (
	"context"
	"fmt"
	"strconv"
)

func (c *client) validateDomain(ctx context.Context, input resourceRequest) error {
	if input.Mode != "http" {
		return nil
	}
	domains, err := c.domains(ctx)
	if err != nil {
		return err
	}
	for _, domain := range domains {
		if domain.DomainID != input.DomainID {
			continue
		}
		if domain.Type == "cname" && input.Subdomain != "" {
			return fmt.Errorf("domínios CNAME não aceitam subdomínio")
		}
		return nil
	}
	if input.Subdomain != "" {
		options, err := c.domainOptions(ctx, input.Subdomain)
		if err != nil {
			return err
		}
		for _, option := range options.Options {
			if option.DomainID == input.DomainID {
				return nil
			}
		}
		return fmt.Errorf("o domínio gratuito selecionado não está disponível; consulte as opções novamente")
	}
	return fmt.Errorf("domínio não encontrado nesta organização; selecione um domínio próprio ou uma opção gratuita disponível")
}

// provisionResource prepara o recurso antes de expor qualquer destino local.
func (s *service) provisionResource(ctx context.Context, c *client, input resourceRequest) (Resource, error) {
	var created Resource
	if err := c.call(ctx, "PUT", c.orgPath("/public-resource"), input.remoteBody(), &created); err != nil {
		return created, err
	}
	if created.ResourceID <= 0 {
		return created, fmt.Errorf("Pangolin retornou um recurso sem identificador; verifique-o antes de repetir")
	}
	id := created.ResourceID
	path := "/public-resource/" + strconv.Itoa(id)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO pangolin_account_resources(resource_id,site_id,local_domain,account_id) VALUES($1,$2,$3,$4)`, id, input.SiteID, input.LocalDomain, c.config.ID); err != nil {
		if cleanup := c.remove(ctx, path); cleanup != nil {
			return created, fmt.Errorf("falha ao guardar o recurso remoto %d; remova-o no Pangolin antes de repetir", id)
		}
		return created, fmt.Errorf("não foi possível guardar o recurso; o recurso remoto foi removido")
	}
	settings := map[string]any{"enabled": false}
	if input.LocalDomain != "" {
		// Host escolhe o virtual host nginx; SNI escolhe o certificado no HTTPS.
		settings["setHostHeader"] = input.LocalDomain
		if input.Method == "https" {
			settings["tlsServerName"] = input.LocalDomain
		}
	}
	reportProgress(ctx, "target", "Configurando Host, SNI e alvo nginx", input.SiteID, id)
	err := c.call(ctx, "POST", path, settings, nil)
	if err == nil {
		err = c.call(ctx, "PUT", path+"/target", input.targetBody(), nil)
	}
	if err != nil {
		if cleanup := c.remove(ctx, path); cleanup == nil {
			_, _ = s.db.ExecContext(ctx, `DELETE FROM pangolin_account_resources WHERE resource_id=$1 AND account_id=$2`, id, c.config.ID)
		}
		return created, fmt.Errorf("falha ao preparar o recurso %d; atualize a lista e remova eventual recurso pendente: %w", id, err)
	}
	created.Enabled = false
	created.Managed = true
	created.LocalDomain = input.LocalDomain
	return created, nil
}
