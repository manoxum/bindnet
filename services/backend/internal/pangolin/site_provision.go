package pangolin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (s *service) createSite(w http.ResponseWriter, r *http.Request, c *client) (resultErr error) {
	var input publicationRequest
	if err := decode(w, r, &input); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	resource, err := input.resource(1)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	input.Name = resource.Name
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 180*time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, progressKey{}, progressCallback(func(stage, message string, siteID, resourceID int) {
		s.progress.stage(c.config.ID, stage, message, siteID, resourceID)
	}))
	r = r.WithContext(ctx)
	reportProgress(ctx, "validating", "Validando domínio público no Pangolin", 0, 0)
	defer func() { s.progress.finish(c.config.ID, resultErr) }()
	if err := c.validateDomain(ctx, resource); err != nil {
		return err
	}
	reportProgress(ctx, "site", "Criando site e guardando credenciais", 0, 0)

	site, err := s.provisionSite(ctx, c, input.Name)
	if err != nil {
		return err
	}
	s.record(r, "pangolin_site_created", site.SiteID)
	reportProgress(ctx, "connector", "Baixando imagem e iniciando o conector Newt (até 90 segundos)", site.SiteID, 0)
	if err = s.startLocalConnector(ctx, c.config.ID, site.SiteID, c.config.Endpoint, site); err != nil {
		return fmt.Errorf("site %d criado; conector não confirmado. Inicie-o na aba Sites e crie o recurso após conferir o estado", site.SiteID)
	}
	s.record(r, "pangolin_connector_start", site.SiteID)
	reportProgress(ctx, "connecting", "Aguardando o túnel ficar online no Pangolin (até 60 segundos)", site.SiteID, 0)
	if err = c.waitSiteOnline(ctx, site.SiteID); err != nil {
		return fmt.Errorf("site %d e conector criados, mas a conexão Cloud não foi confirmada; nenhum recurso foi publicado. Confira o conector na aba Sites e crie o recurso após ficar online: %w", site.SiteID, err)
	}
	site.Online = true
	reportProgress(ctx, "resource", "Criando recurso público e alvo nginx", site.SiteID, 0)
	resource.SiteID = site.SiteID
	created, err := s.provisionResource(ctx, c, resource)
	if err != nil {
		return fmt.Errorf("site %d guardado; o recurso não ficou pronto. Consulte a lista de serviços antes de repetir: %w", site.SiteID, err)
	}
	s.record(r, "pangolin_resource_created", created.ResourceID)
	reportProgress(ctx, "activating", "Ativando o recurso no Pangolin", site.SiteID, created.ResourceID)
	if err = c.call(ctx, "POST", "/public-resource/"+strconv.Itoa(created.ResourceID), map[string]bool{"enabled": true}, nil); err != nil {
		return fmt.Errorf("site %d e conector criados; não foi possível confirmar a ativação do recurso %d. Consulte a lista antes de repetir", site.SiteID, created.ResourceID)
	}
	s.record(r, "pangolin_resource_updated", created.ResourceID)
	site.LocalDomain = resource.LocalDomain
	site.ResourceID = created.ResourceID
	site.PublicDomain = created.FullDomain
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, site)
	return nil
}

func (s *service) provisionSite(ctx context.Context, c *client, name string) (Site, error) {
	var created struct {
		SiteID int    `json:"siteId"`
		NewtID string `json:"newtId"`
		Secret string `json:"secret"`
	}
	if err := c.call(ctx, "PUT", c.orgPath("/site"), map[string]any{"name": name, "type": "newt"}, &created); err != nil {
		return Site{}, err
	}
	site := Site{SiteID: created.SiteID, Name: name, NewtID: created.NewtID, Secret: created.Secret, Type: "newt", Managed: true}
	if site.SiteID <= 0 {
		return site, fmt.Errorf("Pangolin retornou um site sem identificador; verifique-o antes de repetir")
	}
	if site.NewtID == "" || site.Secret == "" {
		if err := c.remove(ctx, "/site/"+strconv.Itoa(site.SiteID)); err != nil {
			return site, fmt.Errorf("site remoto %d sem credenciais válidas; remova-o no Pangolin antes de repetir", site.SiteID)
		}
		return site, fmt.Errorf("credenciais inválidas; o site remoto foi removido")
	}
	if err := insertSite(ctx, s.db, c.config.ID, site); err != nil {
		if cleanup := c.remove(ctx, "/site/"+strconv.Itoa(site.SiteID)); cleanup != nil {
			return site, fmt.Errorf("não foi possível guardar nem remover o site remoto %d; remova-o no Pangolin", site.SiteID)
		}
		return site, fmt.Errorf("não foi possível guardar o site; o site remoto foi removido")
	}
	return site, nil
}
