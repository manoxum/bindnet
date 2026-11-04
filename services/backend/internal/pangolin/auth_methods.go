package pangolin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
)

func validateAuthMethod(method string, body map[string]json.RawMessage) error {
	text := func(key string) (string, bool) {
		raw, ok := body[key]
		if !ok {
			return "", false
		}
		if string(raw) == "null" {
			return "", true
		}
		var value string
		if json.Unmarshal(raw, &value) != nil {
			return "", false
		}
		return value, true
	}
	switch method {
	case "password", "pincode":
		if len(body) != 1 {
			return fmt.Errorf("informe somente o valor do método")
		}
		value, ok := text(method)
		if !ok {
			return fmt.Errorf("valor obrigatório")
		}
		if value == "" && string(body[method]) == "null" {
			return nil
		}
		if method == "pincode" {
			if len(value) != 6 || strings.Trim(value, "0123456789") != "" {
				return fmt.Errorf("o PIN deve ter seis dígitos")
			}
		} else if len(value) < 4 || len(value) > 100 {
			return fmt.Errorf("a senha deve ter entre 4 e 100 caracteres")
		}
	case "header-auth":
		if len(body) != 3 {
			return fmt.Errorf("informe utilizador, senha e compatibilidade")
		}
		for _, key := range []string{"user", "password"} {
			value, ok := text(key)
			if !ok || (string(body[key]) != "null" && (len(value) < 4 || len(value) > 100)) {
				return fmt.Errorf("credenciais de cabeçalho inválidas")
			}
		}
		if string(body["extendedCompatibility"]) != "null" {
			if err := boolSetting(body["extendedCompatibility"]); err != nil {
				return err
			}
		}
	case "whitelist":
		var emails []string
		if len(body) != 1 || json.Unmarshal(body["emails"], &emails) != nil || len(emails) > 200 {
			return fmt.Errorf("lista de e-mails inválida")
		}
		for _, email := range emails {
			value := email
			if strings.HasPrefix(value, "*@") {
				value = "wildcard@" + strings.TrimPrefix(value, "*@")
			}
			if _, err := mail.ParseAddress(value); err != nil {
				return fmt.Errorf("e-mail inválido")
			}
		}
	case "roles":
		var ids []int
		if len(body) != 1 || json.Unmarshal(body["roleIds"], &ids) != nil || len(ids) > 200 {
			return fmt.Errorf("lista de funções inválida")
		}
		for _, id := range ids {
			if id <= 0 {
				return fmt.Errorf("função inválida")
			}
		}
	case "users":
		var ids []string
		if len(body) != 1 || json.Unmarshal(body["userIds"], &ids) != nil || len(ids) > 200 {
			return fmt.Errorf("lista de utilizadores inválida")
		}
		for _, id := range ids {
			if id == "" || len(id) > 255 {
				return fmt.Errorf("utilizador inválido")
			}
		}
	default:
		return fmt.Errorf("método de autenticação não permitido")
	}
	return nil
}
func (s *service) updateAuthMethod(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := s.managedResource(r, c)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return nil
	}
	method := r.PathValue("method")
	var body map[string]json.RawMessage
	if err = decode(w, r, &body); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	if err = validateAuthMethod(method, body); err != nil {
		http.Error(w, err.Error(), 400)
		return nil
	}
	if err = c.call(r.Context(), "POST", "/public-resource/"+strconv.Itoa(id)+"/"+method, body, nil); err != nil {
		return err
	}
	s.record(r, "pangolin_resource_auth_changed", id)
	w.WriteHeader(204)
	return nil
}
