#!/usr/bin/env bash
set -euo pipefail

hotspot_dir="$(cd "$(dirname "$0")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'find "${test_dir}" -depth -delete' EXIT
export HOTSPOT_CONTROL_DIR="${test_dir}/control"
export HOTSPOT_SERVICE_LOG="${test_dir}/service.log"
export COMMAND_LOG="${test_dir}/commands.log"
export RADIO_STATE="${test_dir}/radio"
export PATH="${test_dir}:${PATH}"

cat > "${test_dir}/psql" <<'MOCK'
#!/usr/bin/env bash
case "$*" in
  *to_regclass*) printf 't\n' ;;
  *'FROM hotspot_config'*) printf 'WIFI_INTERFACE\twifi-test\nINTERNET_INTERFACE\teth-test\nWIFI_OPEN\ttrue\nWIFI_SSID\tTeste\n' ;;
esac
MOCK
cat > "${test_dir}/rfkill" <<'MOCK'
#!/usr/bin/env bash
case "$1" in
  list) printf 'Soft blocked: %s\n' "$(cat "${RADIO_STATE}")" ;;
  unblock) printf 'no\n' > "${RADIO_STATE}" ;;
esac
MOCK
cat > "${test_dir}/record" <<'MOCK'
#!/usr/bin/env bash
printf '%s %s\n' "${0##*/}" "$*" >> "${COMMAND_LOG}"
exit 1
MOCK
chmod +x "${test_dir}/psql" "${test_dir}/rfkill" "${test_dir}/record"
for command in create_ap nohup ip iw iptables sysctl conntrack; do
  ln -s record "${test_dir}/${command}"
done

printf 'yes\n' > "${RADIO_STATE}"
if bash "${hotspot_dir}/entrypoint.sh" start || [[ -e "${HOTSPOT_CONTROL_DIR}/runner.pid" ]]; then
  printf 'O arranque automático tentou iniciar um rádio bloqueado.\n' >&2
  exit 1
fi
if bash "${hotspot_dir}/entrypoint.sh" run || [[ -s "${COMMAND_LOG}" ]]; then
  printf 'O runner tocou na rede com o rádio bloqueado.\n' >&2
  exit 1
fi
bash "${hotspot_dir}/entrypoint.sh" status | grep -q '"radioBlocked":true'

HOTSPOT_START_REASON=manual bash "${hotspot_dir}/entrypoint.sh" start
[[ -s "${HOTSPOT_CONTROL_DIR}/runner.pid" && "$(cat "${RADIO_STATE}")" == no ]]
bash "${hotspot_dir}/entrypoint.sh" status | grep -q '"radioBlocked":false'

source "${hotspot_dir}/uplink.sh"
BINDNET_UPLINK_INTERFACE=bn-uplink
REAL_INTERNET_INTERFACE=eth-test
INTERNET_STRATEGY=auto
CREATE_AP_INTERNET_INTERFACE=bn-uplink
HOTSPOT_CIDR=192.0.2.0/24
device_exists=false
ip() {
  printf 'ip %s\n' "$*" >> "${COMMAND_LOG}"
  case "$*" in
    'link show bn-uplink') "${device_exists}" ;;
    'link add '*) device_exists=true ;;
    '-4 route show default dev '*) printf 'default via 192.0.2.1\n' ;;
  esac
}
validate_real_internet_interface() { :; }
is_real_internet_interface() { :; }
ensure_bindnet_uplink_chains() { :; }
ensure_bindnet_uplink_ip_rules() { :; }
log() { printf '%s\n' "$*"; }
iptables() { :; }
sysctl() { :; }
conntrack() { :; }
: > "${COMMAND_LOG}"

setup_bindnet_virtual_uplink
apply_bindnet_uplink_rules wifi-test
setup_bindnet_virtual_uplink

[[ "$(grep -c '^ip link add' "${COMMAND_LOG}")" == 1 ]]
if grep -q '^ip link delete' "${COMMAND_LOG}"; then
  exit 1
fi
[[ "$(sed -n '/^ip link set/{p;q;}' "${COMMAND_LOG}")" == 'ip link set bn-uplink addrgenmode none' ]]
grep -q '^ip route replace default via 192.0.2.1 dev wifi-test table 91$' "${COMMAND_LOG}"
printf 'Arranque com rádio bloqueado, arranque manual e troca de uplink verificados.\n'
