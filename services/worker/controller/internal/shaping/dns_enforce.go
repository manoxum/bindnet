// dns_enforce.go reforca o uso do resolver local pelos clientes do
// hotspot, complementando o bloqueio de conteudo por dominio feito no
// dns-provider: (1) redireciona todo DNS em claro (:53) dos clientes
// para o gateway, para quem tenta trocar de DNS a mao nao escapar; e
// (2) bloqueia DNS criptografado conhecido - DoT (853) e uma lista
// curada de IPs de provedores DoH em 443 - para nao burlar por DoH/DoT.
// Puro iptables idempotente, aplicado/removido pelo backend conforme
// existir plano de conteudo vinculado (ver hotspot_content_dnsforce.go).
package shaping

import (
	"encoding/json"
	"net/http"
	"strings"
)

const dnsForceComment = "bn-dnsforce"

// knownDoHIPs sao IPs de provedores DoH publicos bem conhecidos,
// bloqueados na porta 443 (TCP e UDP/HTTP3) quando o reforco esta ligado.
// Nao pretende ser exaustivo (impossivel), mas cobre os provedores que
// celulares/browsers usam por padrao. Bloquear so TCP deixava passar DoH
// sobre HTTP/3 (QUIC, UDP 443) - por isso applyDNSForce cobre os dois.
var knownDoHIPs = []string{
	// Cloudflare (1.1.1.1, familias de filtragem 1.1.1.2/1.1.1.3)
	"1.1.1.1", "1.0.0.1", "1.1.1.2", "1.0.0.2", "1.1.1.3", "1.0.0.3",
	// Google
	"8.8.8.8", "8.8.4.4",
	// Quad9
	"9.9.9.9", "149.112.112.112", "9.9.9.11", "149.112.112.11",
	// AdGuard
	"94.140.14.14", "94.140.15.15", "94.140.14.15", "94.140.15.16",
	// OpenDNS / Cisco
	"208.67.222.222", "208.67.220.220",
	// NextDNS (anycast)
	"45.90.28.0/24", "45.90.30.0/24",
	// CleanBrowsing
	"185.228.168.9", "185.228.169.9",
	// Mullvad
	"194.242.2.2", "194.242.2.3",
	// ControlD
	"76.76.2.0/24", "76.76.10.0/24",
}

type dnsForceRequest struct {
	Interface string `json:"interface"`
	Enabled   bool   `json:"enabled"`
	Gateway   string `json:"gateway"`
}

func RegisterDNSEnforceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /hotspot/dnsforce/apply", handleDNSForceApply)
}

func handleDNSForceApply(w http.ResponseWriter, r *http.Request) {
	var req dnsForceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo invalido", http.StatusBadRequest)
		return
	}
	if !req.Enabled {
		teardownDNSForce()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if strings.TrimSpace(req.Interface) == "" || strings.TrimSpace(req.Gateway) == "" {
		http.Error(w, "campos 'interface' e 'gateway' obrigatorios", http.StatusBadRequest)
		return
	}
	apIface, _, err := resolveShapingInterfaces(req.Interface)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if err := applyDNSForce(apIface, req.Gateway); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// applyDNSForce instala (idempotente) o redirect de :53 e os DROPs de
// DoT/DoH. O redirect exclui o proprio gateway como destino para nao
// mexer no trafego que ja vai ao resolver local.
func applyDNSForce(apIface, gateway string) error {
	for _, proto := range []string{"udp", "tcp"} {
		if err := ensureNatRedirect(apIface, gateway, proto); err != nil {
			return err
		}
	}
	// DoT: porta 853 (tcp) sempre bloqueada.
	if err := ensureForwardDrop(apIface, "tcp", "853", "", dnsForceComment+"-dot"); err != nil {
		return err
	}
	// DoH: 443 (TCP e UDP/HTTP3) para IPs de provedores conhecidos.
	for _, ip := range knownDoHIPs {
		for _, proto := range []string{"tcp", "udp"} {
			if err := ensureForwardDrop(apIface, proto, "443", ip, dnsForceComment+"-doh"); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureNatRedirect(apIface, gateway, proto string) error {
	comment := dnsForceComment + "-redirect-" + proto
	args := []string{
		"-i", apIface, "!", "-d", gateway, "-p", proto, "--dport", "53",
		"-m", "comment", "--comment", comment, "-j", "REDIRECT", "--to-ports", "53",
	}
	if iptablesCheck(append([]string{"-t", "nat", "-C", "PREROUTING"}, args...)...) == nil {
		return nil
	}
	return runIptables(append([]string{"-t", "nat", "-I", "PREROUTING", "1"}, args...)...)
}

func ensureForwardDrop(apIface, proto, dport, dstHost, comment string) error {
	args := []string{"-i", apIface, "-p", proto}
	if dstHost != "" {
		args = append(args, "-d", dstHost)
	}
	args = append(args, "--dport", dport, "-m", "comment", "--comment", comment, "-j", "DROP")
	if iptablesCheck(append([]string{"-C", "FORWARD"}, args...)...) == nil {
		return nil
	}
	return runIptables(append([]string{"-I", "FORWARD", "1"}, args...)...)
}

func teardownDNSForce() {
	deleteRulesByComment("nat", "PREROUTING", dnsForceComment+"-redirect-udp")
	deleteRulesByComment("nat", "PREROUTING", dnsForceComment+"-redirect-tcp")
	deleteRulesByComment("", "FORWARD", dnsForceComment+"-dot")
	deleteRulesByComment("", "FORWARD", dnsForceComment+"-doh")
}
