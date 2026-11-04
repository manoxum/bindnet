package pangolin

import (
	"fmt"
	"net/http"
	"strings"
)

type configRequest struct {
	Name     string  `json:"name"`
	APIURL   string  `json:"apiUrl"`
	Endpoint string  `json:"endpoint"`
	OrgID    string  `json:"orgId"`
	APIKey   *string `json:"apiKey"`
}

func (input configRequest) config(old Config) (Config, error) {
	c := Config{ID: old.ID, Name: strings.TrimSpace(input.Name), APIURL: strings.TrimRight(strings.TrimSpace(input.APIURL), "/"), Endpoint: strings.TrimRight(strings.TrimSpace(input.Endpoint), "/"), OrgID: strings.TrimSpace(input.OrgID), APIKey: old.APIKey}
	if input.APIKey != nil {
		c.APIKey = strings.TrimSpace(*input.APIKey)
	}
	if c.Name == "" || len(c.Name) > 100 {
		return c, fmt.Errorf("informe um nome de até 100 caracteres para a conta")
	}
	return c, c.validate()
}
func (s *service) getConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	c, err := loadConfig(r.Context(), s.db, r.PathValue("accountId"))
	if err != nil {
		http.Error(w, "conta Pangolin não encontrada", 404)
		return
	}
	writeJSON(w, c)
}
func (s *service) putConfig(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var input configRequest
	if err := decode(w, r, &input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	old, err := loadConfig(r.Context(), s.db, r.PathValue("accountId"))
	if err != nil {
		http.Error(w, "conta Pangolin não encontrada", 404)
		return
	}
	c, err := input.config(old)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	sites, err := managedSites(r.Context(), s.db, c.ID)
	if err != nil {
		http.Error(w, "não foi possível verificar os conectores", 500)
		return
	}
	if len(sites) > 0 && (c.APIURL != old.APIURL || c.OrgID != old.OrgID || c.Endpoint != old.Endpoint) {
		http.Error(w, "remova os sites geridos antes de mudar o servidor ou a organização; nome e chave podem ser trocados", 409)
		return
	}
	if err = saveConfig(r.Context(), s.db, c); err != nil {
		http.Error(w, "não foi possível guardar a conta", 500)
		return
	}
	s.record(r, "pangolin_config_changed", 0)
	w.WriteHeader(204)
}
func (s *service) test(w http.ResponseWriter, r *http.Request, c *client) error {
	sites, err := c.sites(r.Context())
	if err != nil {
		return err
	}
	domains, err := c.domains(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, map[string]any{"connected": true, "siteCount": len(sites), "domainCount": len(domains), "message": "Autenticação e listagens de sites/domínios confirmadas. A consulta global de nomes gratuitos depende das rotas disponibilizadas pelo Cloud."})
	return nil
}
func (s *service) listDomains(w http.ResponseWriter, r *http.Request, c *client) error {
	domains, err := c.domains(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, domains)
	return nil
}
