package pangolin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type settingValidator func(json.RawMessage) error

func boolSetting(value json.RawMessage) error {
	var v bool
	if string(value) == "null" || json.Unmarshal(value, &v) != nil {
		return fmt.Errorf("valor booleano obrigatório")
	}
	return nil
}
func textSetting(max int, nullable bool) settingValidator {
	return func(value json.RawMessage) error {
		if nullable && string(value) == "null" {
			return nil
		}
		var v string
		if json.Unmarshal(value, &v) != nil || string(value) == "null" || len(v) > max || strings.ContainsAny(v, "\r\x00") {
			return fmt.Errorf("texto inválido ou demasiado longo")
		}
		return nil
	}
}
func enumSetting(values ...string) settingValidator {
	return func(raw json.RawMessage) error {
		var value string
		if json.Unmarshal(raw, &value) == nil {
			for _, option := range values {
				if value == option {
					return nil
				}
			}
		}
		return fmt.Errorf("opção inválida")
	}
}
func headersSetting(raw json.RawMessage) error {
	var headers []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &headers) != nil || len(headers) > 50 {
		return fmt.Errorf("lista de até 50 cabeçalhos esperada")
	}
	for _, header := range headers {
		if !headerName.MatchString(header.Name) || len(header.Value) > 4096 || strings.ContainsAny(header.Value, "\r\n\x00") {
			return fmt.Errorf("cabeçalho inválido")
		}
		if strings.EqualFold(header.Name, "Host") {
			return fmt.Errorf("o Host é controlado pelo domínio local do nginx")
		}
	}
	return nil
}

var headerName = regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$")

func policySetting(raw json.RawMessage) error {
	if string(raw) == "null" {
		return nil
	}
	var id int
	if json.Unmarshal(raw, &id) != nil || id <= 0 {
		return fmt.Errorf("política inválida")
	}
	return nil
}

var settingValidators = map[string]settingValidator{
	"name": func(raw json.RawMessage) error {
		if err := textSetting(255, false)(raw); err != nil {
			return err
		}
		var name string
		_ = json.Unmarshal(raw, &name)
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("nome obrigatório")
		}
		return nil
	},
	"niceId": func(raw json.RawMessage) error {
		var id string
		if json.Unmarshal(raw, &id) != nil || len(id) > 255 || !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(id) {
			return fmt.Errorf("identificador inválido")
		}
		return nil
	},
	"domainId": textSetting(255, false), "subdomain": textSetting(63, true),
	"ssl": boolSetting, "sso": boolSetting, "blockAccess": boolSetting,
	"stickySession": boolSetting, "forwardIdentityHeaders": boolSetting,
	"applyRules": boolSetting, "emailWhitelistEnabled": boolSetting, "mtlsEnabled": boolSetting,
	"loadBalancingStrategy": enumSetting("wrr", "p2c", "hrw", "leasttime"),
	"requestHeaders":        headersSetting, "responseHeaders": headersSetting,
	"maintenanceModeEnabled": boolSetting, "maintenanceModeType": enumSetting("forced", "automatic"),
	"maintenanceTitle": textSetting(255, true), "maintenanceMessage": textSetting(2000, true), "maintenanceEstimatedTime": textSetting(100, true),
	"resourcePolicyId": policySetting,
}
