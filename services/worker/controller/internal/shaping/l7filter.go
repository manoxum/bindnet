// l7filter.go redireciona o HTTPS (443) e HTTP (80) dos clientes do
// hotspot para o proxy transparente de conteudo do dns-provider (que le
// SNI/Host e bloqueia/encaminha pela mesma blocklist do DNS). Complementa
// o DNS: pega quem burla o resolver (troca de DNS, acesso por IP direto).
// Puro iptables (nat PREROUTING REDIRECT), mesmo idioma do reforco de DNS
// e do portal cativo. Ligado pelo backend so quando ha plano de conteudo
// em uso. Exclui o proprio gateway como destino (painel/portal).
package shaping

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const l7Comment = "bn-l7"

type l7FilterRequest struct {
	Interface string `json:"interface"`
	Enabled   bool   `json:"enabled"`
	Gateway   string `json:"gateway"`
	SNIPort   int    `json:"sniPort"`
	HTTPPort  int    `json:"httpPort"`
}

func RegisterL7FilterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /hotspot/l7filter/apply", handleL7FilterApply)
}

func handleL7FilterApply(w http.ResponseWriter, r *http.Request) {
	var req l7FilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo invalido", http.StatusBadRequest)
		return
	}
	if !req.Enabled {
		teardownL7Filter()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if strings.TrimSpace(req.Interface) == "" || strings.TrimSpace(req.Gateway) == "" || req.SNIPort == 0 || req.HTTPPort == 0 {
		http.Error(w, "campos 'interface','gateway','sniPort','httpPort' obrigatorios", http.StatusBadRequest)
		return
	}
	apIface, _, err := resolveShapingInterfaces(req.Interface)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if err := ensureL7Redirect(apIface, req.Gateway, "443", req.SNIPort, l7Comment+"-https"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := ensureL7Redirect(apIface, req.Gateway, "80", req.HTTPPort, l7Comment+"-http"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ensureL7Redirect insere (idempotente) o REDIRECT de uma porta para o
// proxy local, excluindo o gateway como destino (painel/portal cativo
// nao devem ser proxeados).
func ensureL7Redirect(apIface, gateway, dport string, toPort int, comment string) error {
	args := []string{
		"-i", apIface, "!", "-d", gateway, "-p", "tcp", "--dport", dport,
		"-m", "comment", "--comment", comment, "-j", "REDIRECT", "--to-ports", strconv.Itoa(toPort),
	}
	if iptablesCheck(append([]string{"-t", "nat", "-C", "PREROUTING"}, args...)...) == nil {
		return nil
	}
	return runIptables(append([]string{"-t", "nat", "-I", "PREROUTING", "1"}, args...)...)
}

func teardownL7Filter() {
	deleteRulesByComment("nat", "PREROUTING", l7Comment+"-https")
	deleteRulesByComment("nat", "PREROUTING", l7Comment+"-http")
}
