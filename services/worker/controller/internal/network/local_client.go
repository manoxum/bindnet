package network

import (
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
)

// handleLocalClient so consulta enderecos do host; nunca altera a rede.
func handleLocalClient(w http.ResponseWriter, r *http.Request) {
	ip, err := netip.ParseAddr(r.URL.Query().Get("ip"))
	if err != nil {
		http.Error(w, "endereco IP invalido", http.StatusBadRequest)
		return
	}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		http.Error(w, "falha ao consultar enderecos do host", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Local bool `json:"local"`
	}{Local: isLocalAddress(ip, addresses)})
}

func isLocalAddress(ip netip.Addr, addresses []net.Addr) bool {
	ip = ip.Unmap()
	if ip.IsLoopback() {
		return true
	}
	for _, address := range addresses {
		prefix, err := netip.ParsePrefix(address.String())
		if err == nil && prefix.Addr().Unmap() == ip {
			return true
		}
	}
	return false
}
