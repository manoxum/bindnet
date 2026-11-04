package pangolin

import (
	"fmt"
	"net/http"
	"strconv"
)

func (s *service) updateDestination(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	ids, err := s.resourceIDs(r.Context(), c.config.ID)
	if err != nil {
		return fmt.Errorf("não foi possível consultar os recursos geridos")
	}
	siteID, ok := ids[id]
	if !ok {
		http.Error(w, "recurso não gerido por esta conta Bindnet", 403)
		return nil
	}
	var input struct {
		LocalDomain string `json:"localDomain"`
		Method      string `json:"method"`
	}
	if err = decode(w, r, &input); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	target, err := (publicationRequest{Name: "destino", DomainID: "existing", LocalDomain: input.LocalDomain, Method: input.Method}).resource(siteID)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	targets, err := c.targets(r, id)
	if err != nil {
		return err
	}
	if len(targets) != 1 || targets[0].SiteID != siteID {
		http.Error(w, "este recurso tem destinos alterados fora do Bindnet; faça a gestão no Pangolin", 409)
		return nil
	}
	path := "/public-resource/" + strconv.Itoa(id)
	var remote Resource
	if err = c.call(r.Context(), "GET", path, nil, &remote); err != nil {
		return err
	}
	if remote.Mode != "http" {
		http.Error(w, "somente recursos HTTP podem usar um domínio nginx", 400)
		return nil
	}
	// Mantém desativado durante a troca para nunca misturar Host/SNI e destino.
	if err = c.call(r.Context(), "POST", path, map[string]bool{"enabled": false}, nil); err != nil {
		return err
	}
	settings := map[string]any{"setHostHeader": target.LocalDomain, "tlsServerName": nil}
	if target.Method == "https" {
		settings["tlsServerName"] = target.LocalDomain
	}
	if err = c.call(r.Context(), "POST", path, settings, nil); err != nil {
		return fmt.Errorf("recurso %d desativado; não foi possível alterar Host/SNI: %w", id, err)
	}
	body := target.targetBody()
	body["enabled"] = targets[0].Enabled
	if err = c.call(r.Context(), "POST", "/target/"+strconv.Itoa(targets[0].TargetID), body, nil); err != nil {
		return fmt.Errorf("recurso %d desativado; não foi possível alterar o destino: %w", id, err)
	}
	if _, err = s.db.ExecContext(r.Context(), `UPDATE pangolin_account_resources SET local_domain=$3 WHERE account_id=$1 AND resource_id=$2`, c.config.ID, id, target.LocalDomain); err != nil {
		return fmt.Errorf("destino remoto alterado; recurso %d permanece desativado, atualize o cadastro antes de ativar", id)
	}
	if remote.Enabled {
		if err = c.call(r.Context(), "POST", path, map[string]bool{"enabled": true}, nil); err != nil {
			return err
		}
	}
	s.record(r, "pangolin_destination_updated", id)
	w.WriteHeader(204)
	return nil
}
