package pangolin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func (s *service) managedResource(r *http.Request, c *client) (int, error) {
	id, err := pathID(r)
	if err != nil {
		return 0, err
	}
	ids, err := s.resourceIDs(r.Context(), c.config.ID)
	if err != nil {
		return 0, fmt.Errorf("não foi possível consultar os recursos geridos")
	}
	if _, ok := ids[id]; !ok {
		return 0, fmt.Errorf("recurso não gerido por esta conta Bindnet")
	}
	return id, nil
}

func (s *service) getResourceSettings(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := s.managedResource(r, c)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return nil
	}
	var remote map[string]json.RawMessage
	if err = c.call(r.Context(), "GET", "/public-resource/"+strconv.Itoa(id), nil, &remote); err != nil {
		return err
	}
	result := map[string]any{}
	for key := range settingValidators {
		if value, ok := remote[key]; ok {
			result[key] = value
		}
	}
	for _, key := range []string{"resourceId", "fullDomain", "mode", "enabled", "setHostHeader", "tlsServerName", "resourcePolicyId", "passwordEnabled", "pincodeEnabled", "headerAuthEnabled", "mtlsEnabled", "emailWhitelistEnabled"} {
		if value, ok := remote[key]; ok {
			result[key] = value
		}
	}
	// Segredos de PIN, senha e autenticação de cabeçalho nunca voltam na leitura.
	writeJSON(w, result)
	return nil
}
func (s *service) updateResourceSettings(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := s.managedResource(r, c)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return nil
	}
	var values map[string]json.RawMessage
	if err = decode(w, r, &values); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	if len(values) == 0 {
		http.Error(w, "informe ao menos uma configuração", 400)
		return nil
	}
	for key, value := range values {
		validate, ok := settingValidators[key]
		if !ok {
			http.Error(w, "configuração não permitida: "+key, 400)
			return nil
		}
		if err = validate(value); err != nil {
			http.Error(w, key+": "+err.Error(), 400)
			return nil
		}
	}
	if _, domainChanged := values["domainId"]; domainChanged {
		var domain, subdomain string
		_ = json.Unmarshal(values["domainId"], &domain)
		_ = json.Unmarshal(values["subdomain"], &subdomain)
		if err = c.validateDomain(r.Context(), resourceRequest{Mode: "http", DomainID: domain, Subdomain: subdomain}); err != nil {
			http.Error(w, err.Error(), 400)
			return nil
		}
	} else if _, changed := values["subdomain"]; changed {
		http.Error(w, "envie também domainId ao alterar o subdomínio", 400)
		return nil
	}
	if err = c.call(r.Context(), "POST", "/public-resource/"+strconv.Itoa(id), values, nil); err != nil {
		return err
	}
	s.record(r, "pangolin_resource_settings_changed", id)
	w.WriteHeader(204)
	return nil
}
