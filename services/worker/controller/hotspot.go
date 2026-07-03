package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strings"
)

func registerHotspotClientRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /hotspot/clients", handleHotspotClients)
	mux.HandleFunc("POST /hotspot/block", handleHotspotBlock)
	mux.HandleFunc("POST /hotspot/unblock", handleHotspotUnblock)
}

type hotspotClient struct {
	MAC      string `json:"mac"`
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}

type hotspotMACActionRequest struct {
	Interface string `json:"interface"`
	MAC       string `json:"mac"`
}

// handleHotspotClients usa o comando nativo "create_ap --list-clients"
// (ver print_client/list_clients em /usr/local/bin/create_ap) em vez de
// ler o dnsmasq.leases diretamente, pois o caminho desse arquivo tem um
// sufixo temporario aleatorio (mktemp) que muda a cada execucao.
func handleHotspotClients(w http.ResponseWriter, r *http.Request) {
	iface := r.URL.Query().Get("interface")
	if iface == "" {
		http.Error(w, "parametro 'interface' obrigatorio", http.StatusBadRequest)
		return
	}

	containerID, err := composeServiceContainerID("hotspot")
	if err != nil || containerID == "" {
		log.Printf("[worker] erro ao localizar container do hotspot: %v", err)
		_ = json.NewEncoder(w).Encode([]hotspotClient{})
		return
	}

	output, err := exec.Command("docker", "exec", containerID, "create_ap", "--list-clients", resolveRunningIface(containerID, iface)).CombinedOutput()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("[worker] erro ao listar clientes do hotspot: %v (%s)", err, output)
		_ = json.NewEncoder(w).Encode([]hotspotClient{})
		return
	}

	var clients []hotspotClient
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 || !strings.Contains(fields[0], ":") {
			// pula cabecalho ("MAC IP Hostname") e a linha
			// "No clients connected".
			continue
		}
		clients = append(clients, hotspotClient{MAC: fields[0], IP: fields[1], Hostname: fields[2]})
	}
	_ = json.NewEncoder(w).Encode(clients)
}

// resolveRunningIface traduz WIFI_INTERFACE (ex.: "wlp0s20f3") para a
// interface virtual que o create_ap realmente usa quando sobe em modo
// AP+estacao concorrente (ex.: "ap0") - list_clients() no create_ap so
// aceita a interface que ele mesmo esta rastreando; passar a fisica
// direto falha com "not used from create_ap instance" mesmo com
// clientes conectados de verdade. "create_ap --list-running" imprime
// "<pid> <iface-original> (<iface-real>)" quando os dois nomes
// divergem, ou so "<pid> <iface>" quando sao iguais (modo --no-virt) -
// nesse segundo caso ou se nada for encontrado, devolve a interface
// original sem alteracao.
func resolveRunningIface(containerID, iface string) string {
	output, err := exec.Command("docker", "exec", containerID, "create_ap", "--list-running").CombinedOutput()
	if err != nil {
		return iface
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 || fields[1] != iface {
			continue
		}
		if len(fields) >= 3 {
			return strings.Trim(fields[2], "()")
		}
		return iface
	}
	return iface
}

func handleHotspotBlock(w http.ResponseWriter, r *http.Request) {
	handleHotspotMACAction(w, r, true)
}

func handleHotspotUnblock(w http.ResponseWriter, r *http.Request) {
	handleHotspotMACAction(w, r, false)
}

func handleHotspotMACAction(w http.ResponseWriter, r *http.Request, block bool) {
	var req hotspotMACActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo invalido", http.StatusBadRequest)
		return
	}
	req.Interface = strings.TrimSpace(req.Interface)
	mac, err := normalizeMAC(req.MAC)
	if err != nil {
		http.Error(w, "mac invalido", http.StatusBadRequest)
		return
	}
	if req.Interface == "" {
		http.Error(w, "campo 'interface' obrigatorio", http.StatusBadRequest)
		return
	}

	if err := applyHostapdMACAction(req.Interface, mac, block); err != nil {
		log.Printf("[worker] erro ao aplicar ACL do hotspot para %s: %v", mac, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func applyHostapdMACAction(iface, mac string, block bool) error {
	containerID, err := composeServiceContainerID("hotspot")
	if err != nil || containerID == "" {
		if err == nil {
			err = errors.New("container do hotspot ausente")
		}
		return err
	}

	realIface := resolveRunningIface(containerID, iface)
	ctrlDir, err := hotspotControlDir(containerID, iface, realIface)
	if err != nil {
		return err
	}

	action := []string{"hostapd_cli", "-p", ctrlDir, "-i", realIface, "deny_acl"}
	if block {
		action = append(action, "ADD_MAC", mac)
	} else {
		action = append(action, "DEL_MAC", mac)
	}
	output, err := exec.Command("docker", append([]string{"exec", containerID}, action...)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("hostapd_cli deny_acl falhou: %s: %w", strings.TrimSpace(string(output)), err)
	}

	if block {
		output, err = exec.Command("docker", "exec", containerID, "hostapd_cli", "-p", ctrlDir, "-i", realIface, "deauthenticate", mac).CombinedOutput()
		if err != nil {
			return fmt.Errorf("hostapd_cli deauthenticate falhou: %s: %w", strings.TrimSpace(string(output)), err)
		}
	}
	return nil
}

func hotspotControlDir(containerID, iface, realIface string) (string, error) {
	output, err := exec.Command("docker", "exec", containerID, "sh", "-c", `
set -eu
for path in "/tmp/create_ap.$1.conf."*/hostapd_ctrl/"$2" /tmp/create_ap.*.conf.*/hostapd_ctrl/"$2"; do
  if [ -e "$path" ]; then
    dirname "$path"
    exit 0
  fi
done
exit 1
`, "sh", iface, realIface).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("diretorio de controle do hostapd nao encontrado para %s: %s", realIface, strings.TrimSpace(string(output)))
	}
	ctrlDir := strings.TrimSpace(string(output))
	if ctrlDir == "" {
		return "", fmt.Errorf("diretorio de controle do hostapd vazio para %s", realIface)
	}
	return ctrlDir, nil
}

func normalizeMAC(raw string) (string, error) {
	hw, err := net.ParseMAC(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return strings.ToLower(hw.String()), nil
}
