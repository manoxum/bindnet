#!/usr/bin/env bash
# Liga o hotspot 'xCosta': marca a placa Wi-Fi como nao-gerenciada pelo
# NetworkManager (para o hostapd assumir o controle) e sobe os servicos
# hotspot + dns-provider (perfil "hotspot" no docker-compose.yml).
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NM_DROPIN=/etc/NetworkManager/conf.d/90-central-hotspot-unmanaged.conf

if [[ -f "${REPO_DIR}/.env" ]]; then
  # shellcheck disable=SC1091
  source "${REPO_DIR}/.env"
fi
WIFI_INTERFACE="${WIFI_INTERFACE:-wlp0s20f3}"

echo "[hotspot-on] Marcando ${WIFI_INTERFACE} como nao-gerenciada pelo NetworkManager..."
sudo tee "${NM_DROPIN}" >/dev/null <<EOF
[keyfile]
unmanaged-devices=interface-name:${WIFI_INTERFACE};interface-name:ap0
EOF
sudo nmcli general reload conf || sudo systemctl reload NetworkManager

echo "[hotspot-on] Subindo hotspot + dns-provider..."
(cd "${REPO_DIR}" && docker compose --profile hotspot up -d hotspot dns-provider)

echo "[hotspot-on] Pronto. Rede 'xCosta' deve estar visivel em poucos segundos."
