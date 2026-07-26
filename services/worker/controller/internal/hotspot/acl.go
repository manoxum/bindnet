package hotspot

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"bindnet/worker/internal/compose"
	"bindnet/worker/internal/network"
)

type hotspotMACActionRequest struct {
	Interface string `json:"interface"`
	MAC       string `json:"mac"`
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
	mac, err := network.NormalizeMAC(req.MAC)
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
	containerID, err := compose.ServiceContainerID("hotspot")
	if err != nil || containerID == "" {
		if err == nil {
			err = errors.New("container do hotspot ausente")
		}
		return err
	}

	realIface := compose.ResolveRunningIface(containerID, iface)
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

// hotspotControlDir resolve o diretorio de controle do hostapd VIVO. Um
// container que ja reiniciou o create_ap varias vezes acumula dezenas de
// diretorios "/tmp/create_ap.<iface>.conf.*" orfaos (o create_ap so
// remove o seu numa saida limpa; morto por SIGKILL/watchdog, o
// diretorio fica). Pegar o primeiro match da glob quase sempre resolvia
// um socket MORTO - o deny_acl/deauthenticate ia parar num hostapd que
// nao existe mais e o bloqueio nunca chegava ao AP real. Por isso cada
// candidato so e aceito depois de responder "PONG" a um "hostapd_cli
// ping": so o socket da instancia viva passa.
func hotspotControlDir(containerID, iface, realIface string) (string, error) {
	output, err := exec.Command("docker", "exec", containerID, "sh", "-c", `
set -eu
for path in "/tmp/create_ap.$1.conf."*/hostapd_ctrl/"$2" /tmp/create_ap.*.conf.*/hostapd_ctrl/"$2"; do
  [ -e "$path" ] || continue
  dir="$(dirname "$path")"
  if hostapd_cli -p "$dir" -i "$2" ping 2>/dev/null | grep -q PONG; then
    printf '%s\n' "$dir"
    exit 0
  fi
done
exit 1
`, "sh", iface, realIface).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("socket de controle do hostapd vivo nao encontrado para %s (nenhum candidato respondeu ping): %s", realIface, strings.TrimSpace(string(output)))
	}
	ctrlDir := strings.TrimSpace(string(output))
	if ctrlDir == "" {
		return "", fmt.Errorf("diretorio de controle do hostapd vazio para %s", realIface)
	}
	return ctrlDir, nil
}
