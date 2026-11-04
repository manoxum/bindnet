package pangolin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (s *service) listResources(w http.ResponseWriter, r *http.Request, c *client) error {
	remote, err := c.resources(r.Context())
	if err != nil {
		return err
	}
	ids, err := s.resourceIDs(r.Context(), c.config.ID)
	if err != nil {
		return fmt.Errorf("não foi possível ler os recursos geridos")
	}
	localDomains, err := s.resourceDomains(r.Context(), c.config.ID)
	if err != nil {
		return fmt.Errorf("não foi possível ler os domínios publicados")
	}
	found := map[int]bool{}
	for i := range remote {
		_, remote[i].Managed = ids[remote[i].ResourceID]
		remote[i].LocalDomain = localDomains[remote[i].ResourceID]
		found[remote[i].ResourceID] = true
	}
	for id := range ids {
		if !found[id] {
			remote = append(remote, Resource{ResourceID: id, Name: "Recurso ausente no Pangolin", Managed: true, Targets: []Target{}})
		}
	}
	writeJSON(w, remote)
	return nil
}

func (s *service) createResource(w http.ResponseWriter, r *http.Request, c *client) error {
	var request publicationRequest
	if err := decode(w, r, &request); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	input, err := request.resource(request.SiteID)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	if _, err := s.localSite(r.Context(), c.config.ID, input.SiteID); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	if err := c.validateDomain(r.Context(), input); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 90*time.Second)
	defer cancel()
	created, err := s.provisionResource(ctx, c, input)
	if err != nil {
		return err
	}
	s.record(r, "pangolin_resource_created", created.ResourceID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, created)
	return nil
}

func (s *service) updateResource(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	ids, err := s.resourceIDs(r.Context(), c.config.ID)
	if err != nil {
		return fmt.Errorf("não foi possível consultar os recursos geridos")
	}
	if _, ok := ids[id]; !ok {
		http.Error(w, "este recurso não é gerido pelo Bindnet", 403)
		return nil
	}
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err = decode(w, r, &input); err != nil || input.Enabled == nil {
		http.Error(w, "informe enabled", 400)
		return nil
	}
	if *input.Enabled {
		targets, err := c.targets(r, id)
		if err != nil {
			return err
		}
		ready := false
		for _, target := range targets {
			if target.Enabled && target.SiteID == ids[id] && target.IP != "" && target.Port > 0 {
				ready = true
			}
		}
		if !ready {
			http.Error(w, "recurso sem destino gerido válido; configure o destino antes de ativar", 409)
			return nil
		}
		online, err := c.siteOnline(r.Context(), ids[id])
		if err != nil {
			return err
		}
		if !online {
			http.Error(w, "site offline no Pangolin; inicie o conector na aba Sites e aguarde ficar online antes de ativar o serviço", http.StatusConflict)
			return nil
		}
	}
	if err = c.call(r.Context(), "POST", "/public-resource/"+strconv.Itoa(id), map[string]bool{"enabled": *input.Enabled}, nil); err != nil {
		return err
	}
	s.record(r, "pangolin_resource_updated", id)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *service) deleteResource(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	ids, err := s.resourceIDs(r.Context(), c.config.ID)
	if err != nil {
		return fmt.Errorf("não foi possível consultar os recursos geridos")
	}
	if _, ok := ids[id]; !ok {
		http.Error(w, "este recurso não é gerido pelo Bindnet", 403)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 40*time.Second)
	defer cancel()
	if err = c.remove(ctx, "/public-resource/"+strconv.Itoa(id)); err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM pangolin_account_resources WHERE resource_id=$1 AND account_id=$2`, id, c.config.ID); err != nil {
		return fmt.Errorf("recurso remoto removido; repita para limpar o cadastro local")
	}
	s.record(r, "pangolin_resource_deleted", id)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
