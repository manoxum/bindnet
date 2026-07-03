#!/usr/bin/env bash
set -euo pipefail

log() {
  printf '[bindnet-docker-dns] %s\n' "$*" >&2
}

daemon_config="${DOCKER_DAEMON_CONFIG:-/etc/docker/daemon.json}"
backup="${daemon_config}.bak.$(date +%Y%m%d%H%M%S)"
tmp="$(mktemp)"

gateway_from_network() {
  local network="$1"
  local ipam_config
  ipam_config="$(docker network inspect "$network" --format '{{json .IPAM.Config}}' 2>/dev/null)" || return 1
  python3 - "${ipam_config}" <<'PY'
import ipaddress
import json
import sys

try:
    configs = json.loads(sys.argv[1])
except Exception:
    sys.exit(1)

for config in configs:
    gateway = config.get("Gateway")
    if gateway:
        print(gateway)
        sys.exit(0)

for config in configs:
    subnet = config.get("Subnet")
    if not subnet:
        continue
    network = ipaddress.ip_network(subnet, strict=False)
    for host in network.hosts():
        print(host)
        sys.exit(0)

sys.exit(1)
PY
}

detect_docker_dns() {
  local project="${COMPOSE_PROJECT_NAME:-${PROJECT:-bindnet}}"
  local network
  local candidates=("${project}_proxy" "bindnet_proxy" "bridge")

  for network in "${candidates[@]}"; do
    local gateway
    if gateway="$(gateway_from_network "$network")" && [[ -n "${gateway}" ]]; then
      log "Gateway Docker detectado na rede ${network}: ${gateway}."
      printf '%s\n' "${gateway}"
      return 0
    fi
  done

  log "ERRO: nao foi possivel detectar um gateway Docker. Suba/crie a rede do Bindnet antes de executar este script."
  return 1
}

docker_dns="$(detect_docker_dns)"

if [[ "${EUID}" -ne 0 && ( ! -f "${daemon_config}" || ! -w "${daemon_config}" ) ]]; then
  log "ERRO: execute com sudo para alterar ${daemon_config}."
  exit 1
fi

cleanup() {
  rm -f "${tmp}"
}
trap cleanup EXIT

mkdir -p "$(dirname "${daemon_config}")"
if [[ -f "${daemon_config}" ]]; then
  cp "${daemon_config}" "${backup}"
  log "Backup criado em ${backup}."
else
  printf '{}\n' > "${daemon_config}"
  log "Criado ${daemon_config}."
fi

python3 - "${daemon_config}" "${tmp}" "${docker_dns}" <<'PY'
import json
import sys

source, target, docker_dns = sys.argv[1:]
with open(source, "r", encoding="utf-8") as handle:
    raw = handle.read().strip() or "{}"

config = json.loads(raw)
config["dns"] = [docker_dns]

with open(target, "w", encoding="utf-8") as handle:
    json.dump(config, handle, indent=2, ensure_ascii=True)
    handle.write("\n")
PY

python3 -m json.tool "${tmp}" >/dev/null
cp "${tmp}" "${daemon_config}"

log "Docker daemon configurado para usar DNS ${docker_dns}."
log "Reinicie o Docker para aplicar: sudo systemctl restart docker"
