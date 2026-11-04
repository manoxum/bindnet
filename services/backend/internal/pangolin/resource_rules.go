package pangolin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type accessRule struct {
	RuleID   int    `json:"ruleId"`
	Action   string `json:"action"`
	Match    string `json:"match"`
	Value    string `json:"value"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

func (s *service) resourceRules(w http.ResponseWriter, r *http.Request, c *client) error {
	id, err := s.managedResource(r, c)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return nil
	}
	path := "/public-resource/" + strconv.Itoa(id)
	if r.Method == "GET" {
		var data struct {
			Rules []accessRule `json:"rules"`
		}
		if err = c.call(r.Context(), "GET", path+"/rules", nil, &data); err != nil {
			return err
		}
		if data.Rules == nil {
			data.Rules = []accessRule{}
		}
		writeJSON(w, data.Rules)
		return nil
	}
	ruleID := r.PathValue("ruleId")
	if ruleID != "" {
		n, e := strconv.Atoi(ruleID)
		if e != nil || n <= 0 {
			http.Error(w, "regra inválida", 400)
			return nil
		}
	}
	if r.Method == "DELETE" {
		err = c.remove(r.Context(), path+"/rule/"+ruleID)
	} else {
		var rule accessRule
		if err = decode(w, r, &rule); err != nil {
			http.Error(w, err.Error(), 400)
			return nil
		}
		if !strings.Contains("|ACCEPT|DROP|PASS|", "|"+rule.Action+"|") || !strings.Contains("|CIDR|IP|PATH|COUNTRY|COUNTRY_IS_NOT|ASN|REGION|METHOD|", "|"+rule.Match+"|") || rule.Value == "" || len(rule.Value) > 2048 || rule.Priority < 0 {
			http.Error(w, "ação, correspondência, valor ou prioridade inválidos", 400)
			return nil
		}
		body := map[string]any{"action": rule.Action, "match": rule.Match, "value": rule.Value, "priority": rule.Priority, "enabled": rule.Enabled}
		method := "PUT"
		endpoint := path + "/rule"
		if ruleID != "" {
			method = "POST"
			endpoint += "/" + ruleID
		}
		err = c.call(r.Context(), method, endpoint, body, nil)
	}
	if err != nil {
		return fmt.Errorf("não foi possível alterar a regra: %w", err)
	}
	s.record(r, "pangolin_access_rule_changed", id)
	w.WriteHeader(204)
	return nil
}
