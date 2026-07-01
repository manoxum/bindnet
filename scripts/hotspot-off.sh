#!/usr/bin/env bash
# Desliga o hotspot 'xCosta': para os servicos hotspot + dns-provider e
# devolve a placa Wi-Fi para o controle normal do NetworkManager, para
# que volte a funcionar como cliente Wi-Fi comum.
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NM_DROPIN=/etc/NetworkManager/conf.d/90-central-hotspot-unmanaged.conf

if [[ -f "${REPO_DIR}/.env" ]]; then
  # shellcheck disable=SC1091
  source "${REPO_DIR}/.env"
fi
WIFI_INTERFACE="${WIFI_INTERFACE:-wlp0s20f3}"

echo "[hotspot-off] Parando hotspot + dns-provider..."
(cd "${REPO_DIR}" && docker compose stop hotspot dns-provider)

if [[ -f "${NM_DROPIN}" ]]; then
  echo "[hotspot-off] Removendo override de NetworkManager..."
  sudo rm -f "${NM_DROPIN}"
  sudo nmcli general reload conf || sudo systemctl reload NetworkManager
fi

echo "[hotspot-off] Devolvendo ${WIFI_INTERFACE} para o NetworkManager..."
sudo nmcli device set "${WIFI_INTERFACE}" managed yes || true

echo "[hotspot-off] Pronto. ${WIFI_INTERFACE} deve aparecer como adaptador Wi-Fi normal."
