package pangolin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Lista somente campos públicos para não devolver credenciais de contas Pangolin.
func (s *service) accessOptions(w http.ResponseWriter, r *http.Request, c *client) error {
	kind := r.PathValue("kind")
	if kind != "roles" && kind != "users" && kind != "resource-policies" {
		http.Error(w, "lista inválida", 400)
		return nil
	}
	key := kind
	if kind == "resource-policies" {
		key = "resourcePolicies"
	}
	result := []map[string]json.RawMessage{}
	for page := 1; page <= 100; page++ {
		var data map[string]json.RawMessage
		if err := c.call(r.Context(), "GET", c.orgPath(fmt.Sprintf("/%s?page=%d&pageSize=100", kind, page)), nil, &data); err != nil {
			return err
		}
		var entries []map[string]json.RawMessage
		if err := json.Unmarshal(data[key], &entries); err != nil {
			return fmt.Errorf("formato de opções de acesso incompatível")
		}
		var pagination struct {
			Total jsonCount `json:"total"`
		}
		_ = json.Unmarshal(data["pagination"], &pagination)
		for _, entry := range entries {
			clean := map[string]json.RawMessage{}
			for _, field := range []string{"id", "userId", "roleId", "resourcePolicyId", "name", "email", "description", "isAdmin"} {
				if value, ok := entry[field]; ok {
					clean[field] = value
				}
			}
			result = append(result, clean)
		}
		if len(result) >= int(pagination.Total) || len(entries) == 0 {
			writeJSON(w, result)
			return nil
		}
	}
	return fmt.Errorf("lista de acesso excede 10000 itens")
}
func (s *service) resourceAccess(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := s.managedResource(r, c)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return nil
	}
	kind := r.PathValue("kind")
	if kind != "roles" && kind != "users" && kind != "whitelist" {
		http.Error(w, "lista inválida", 400)
		return nil
	}
	var data map[string]json.RawMessage
	if err = c.call(r.Context(), "GET", "/public-resource/"+strconv.Itoa(id)+"/"+kind, nil, &data); err != nil {
		return err
	}
	// As respostas oficiais dessas listas não incluem PIN, senhas ou chaves.
	result := map[string]json.RawMessage{}
	for _, key := range []string{"roles", "users", "emails", "whitelist", "pagination"} {
		if value, ok := data[key]; ok {
			result[key] = value
		}
	}
	writeJSON(w, result)
	return nil
}
