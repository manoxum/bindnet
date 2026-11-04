package pangolin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func (s *service) listSites(w http.ResponseWriter, r *http.Request, c *client) error {
	remote, err := c.sites(r.Context())
	if err != nil {
		return err
	}
	local, err := managedSites(r.Context(), s.db, c.config.ID)
	if err != nil {
		return fmt.Errorf("não foi possível ler os sites geridos")
	}
	ids := map[int]bool{}
	for i := range remote {
		ids[remote[i].SiteID] = true
		for _, site := range local {
			if site.SiteID == remote[i].SiteID {
				remote[i].Managed = true
			}
		}
	}
	// Mantém visíveis conectores cujo site foi apagado diretamente no Pangolin.
	for _, site := range local {
		if !ids[site.SiteID] {
			remote = append(remote, site)
		}
	}
	writeJSON(w, remote)
	return nil
}

func (s *service) deleteSite(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if _, err = s.localSite(r.Context(), c.config.ID, id); err != nil {
		return err
	}
	var count int
	if err = s.db.QueryRowContext(r.Context(), `SELECT count(*) FROM pangolin_account_resources WHERE site_id=$1 AND account_id=$2`, id, c.config.ID).Scan(&count); err != nil {
		return fmt.Errorf("não foi possível verificar os recursos do site")
	}
	if count > 0 {
		http.Error(w, "remova primeiro os recursos geridos deste site", 409)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 60*time.Second)
	defer cancel()
	path := connectorPath(c.config.ID, id)
	if err = s.worker.Call(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("não foi possível remover o conector local; o site remoto foi preservado")
	}
	if err = c.remove(ctx, "/site/"+strconv.Itoa(id)); err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `DELETE FROM pangolin_account_sites WHERE site_id=$1 AND account_id=$2`, id, c.config.ID); err != nil {
		return fmt.Errorf("site removido no Pangolin; repita para limpar o cadastro local")
	}
	s.record(r, "pangolin_site_deleted", id)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *service) connectorStatus(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if _, err = s.localSite(r.Context(), c.config.ID, id); err != nil {
		return err
	}
	var status struct {
		Running bool   `json:"running"`
		Status  string `json:"status"`
		Image   string `json:"image"`
		Health  string `json:"health"`
	}
	if err = s.worker.Call(r.Context(), "GET", connectorPath(c.config.ID, id)+"/status", nil, &status); err != nil {
		return fmt.Errorf("não foi possível consultar o conector no worker")
	}
	writeJSON(w, status)
	return nil
}
func (s *service) startConnector(w http.ResponseWriter, r *http.Request, c *client) error {
	return s.connectorAction(w, r, c, true)
}
func (s *service) stopConnector(w http.ResponseWriter, r *http.Request, c *client) error {
	return s.connectorAction(w, r, c, false)
}
func (s *service) connectorAction(w http.ResponseWriter, r *http.Request, c *client, start bool) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	site, err := s.localSite(r.Context(), c.config.ID, id)
	if err != nil {
		return err
	}
	if err = s.setConnectorDesiredState(r.Context(), c.config.ID, id, start); err != nil {
		return err
	}
	action := "stop"
	if start {
		action = "start"
		err = s.startLocalConnector(r.Context(), c.config.ID, id, c.config.Endpoint, site)
	} else {
		err = s.worker.Slow().Call(r.Context(), "POST", connectorPath(c.config.ID, id)+"/stop", nil, nil)
	}
	if err != nil {
		return fmt.Errorf("não foi possível executar %s no conector Newt; consulte o estado antes de repetir", action)
	}
	s.record(r, "pangolin_connector_"+action, id)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
