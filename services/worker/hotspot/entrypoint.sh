#!/usr/bin/env bash
set -euo pipefail

log() {
  printf '[hotspot] %s\n' "$*"
}

required() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    log "ERRO: variavel obrigatoria ausente: ${name}"
    exit 1
  fi
}

required WIFI_INTERFACE
required INTERNET_INTERFACE
required WIFI_SSID
required WIFI_PASSWORD

HOTSPOT_GATEWAY="${HOTSPOT_GATEWAY:-192.168.12.1}"
HOTSPOT_CIDR="${HOTSPOT_CIDR:-${HOTSPOT_GATEWAY%.*}.0/24}"
WIFI_COUNTRY="${WIFI_COUNTRY:-ST}"
if [[ -z "${WIFI_CHANNEL:-}" && -n "${WIFI_CHANNE:-}" ]]; then
  WIFI_CHANNEL="${WIFI_CHANNE}"
  log "AVISO: usando WIFI_CHANNE como fallback; prefira WIFI_CHANNEL."
fi
WIFI_CHANNEL="${WIFI_CHANNEL:-auto}"
WIFI_FREQ_BAND="${WIFI_FREQ_BAND:-auto}"

# channel.sh: selecao de banda/canal Wi-Fi. interfaces.sh: resolucao/
# avisos sobre a interface de internet. Ambos sourced do mesmo
# diretorio deste script (ver Dockerfile - os tres arquivos vao para
# /usr/local/bin/).
source "$(dirname "$0")/channel.sh"
source "$(dirname "$0")/interfaces.sh"

normalize_search_domains() {
  local raw="${DNS_SEARCH_DOMAINS:-${DNS_LOCAL_TLDS:-local,test,example}}"
  local domain
  local -a domains=()

  raw="${raw//;/,}"
  raw="${raw// /,}"
  IFS=',' read -r -a domains <<< "${raw}"

  for i in "${!domains[@]}"; do
    domain="${domains[$i],,}"
    domain="${domain#.}"
    [[ -n "${domain}" ]] || continue
    if ! [[ "${domain}" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]]; then
      log "ERRO: DNS_SEARCH_DOMAINS/DNS_LOCAL_TLDS contem dominio invalido para DHCP: ${domain}"
      exit 1
    fi
    printf '%s\n' "${domain}"
  done | awk '!seen[$0]++' | paste -sd, -
}

normalize_dns_fallbacks() {
  local raw
  local server
  local -a servers=()

  if [[ -v HOTSPOT_DNS_FALLBACKS ]]; then
    raw="${HOTSPOT_DNS_FALLBACKS}"
  else
    raw="1.1.1.1,8.8.8.8"
  fi
  raw="${raw//;/,}"
  raw="${raw// /,}"
  IFS=',' read -r -a servers <<< "${raw}"

  for i in "${!servers[@]}"; do
    server="${servers[$i]}"
    [[ -n "${server}" ]] || continue
    if ! [[ "${server}" =~ ^[0-9]{1,3}(\.[0-9]{1,3}){3}$ ]]; then
      log "ERRO: HOTSPOT_DNS_FALLBACKS contem DNS IPv4 invalido para DHCP: ${server}"
      exit 1
    fi
    printf '%s\n' "${server}"
  done | awk '!seen[$0]++' | paste -sd, -
}

DHCP_SEARCH_DOMAINS="$(normalize_search_domains)"
DHCP_DOMAIN="${DHCP_SEARCH_DOMAINS%%,*}"
DHCP_DNS_FALLBACKS="$(normalize_dns_fallbacks)"
DHCP_DNS_SERVERS="${HOTSPOT_GATEWAY}"
if [[ -n "${DHCP_DNS_FALLBACKS}" ]]; then
  DHCP_DNS_SERVERS="${DHCP_DNS_SERVERS},${DHCP_DNS_FALLBACKS}"
fi
export DHCP_SEARCH_DOMAINS DHCP_DOMAIN DHCP_DNS_SERVERS

# Resolve valores "auto" antes de qualquer tentativa de create_ap:
# banda/canal (resolve_wifi_band, chamada mais abaixo) e a interface de
# internet (resolve_internet_interface). A internet real fica por tras
# de um uplink virtual estavel para que o hotspot nao precise reiniciar
# quando a fonte muda.
resolve_internet_interface
warn_if_concurrent_ap_sta_risky

# try_create_ap tenta subir o hotspot num canal/banda especificos.
# Devolve 0 em sucesso (create_ap encerrado com exit code 0 - parada
# limpa via sinal, ver cleanup/trap mais abaixo), 2 para uma falha que
# trocar de canal/banda nunca resolve (conflito AP+estacao ou EBUSY ao
# configurar uma interface virtual), ou 1 para qualquer outra falha
# (ex.: "adapter can not transmit" nesse canal especifico).
CREATE_AP_LOG="/tmp/bindnet-hotspot-create_ap.log"
DNSMASQ_DHCP_LOG="/tmp/bindnet-dnsmasq-dhcp.log"

prepare_dnsmasq_dhcp_log() {
  # dnsmasq pode abrir o log depois de baixar privilegios. Se uma
  # execucao anterior deixou o arquivo root:root/0644 em /tmp, ele falha
  # com "Permission denied" antes mesmo de servir DHCP.
  rm -f "${DNSMASQ_DHCP_LOG}" || true
  touch "${DNSMASQ_DHCP_LOG}"
  chmod 0666 "${DNSMASQ_DHCP_LOG}"
}

prepare_dnsmasq_dhcp_log

try_create_ap() {
  local band="$1"
  local channel="$2"
  local status=0
  local -a virtual_interface_args=()

  # Uma interface AP virtual so e util quando ha uma associacao Wi-Fi
  # cliente que precisa ser preservada. Em alguns kernels/drivers (observado
  # com iwlwifi/AX211 no kernel 7.0), o driver aceita criar ap0 mas devolve
  # EBUSY quando o create_ap tenta trocar o MAC duplicado dessa interface.
  # Com a placa desconectada, usar diretamente a interface fisica evita essa
  # operacao sem perder funcionalidade; o uplink continua sendo bn-uplink.
  if iw dev "${WIFI_INTERFACE}" link 2>/dev/null | grep -q '^Connected to '; then
    log "Wi-Fi cliente ativo em ${WIFI_INTERFACE}; preservando-o com uma interface AP virtual."
  else
    virtual_interface_args=(--no-virt)
    log "${WIFI_INTERFACE} sem associacao Wi-Fi cliente; usando a interface fisica diretamente em modo AP (--no-virt)."
  fi

  log "Preparando hotspot '${WIFI_SSID}' em ${WIFI_INTERFACE}, internet via ${CREATE_AP_INTERNET_INTERFACE} (alimentado por ${REAL_INTERNET_INTERFACE})."
  log "Regiao Wi-Fi: ${WIFI_COUNTRY}; banda: ${band}GHz; canal: ${channel}."
  log "Gateway do hotspot: ${HOTSPOT_GATEWAY}; DNS entregues por DHCP: ${DHCP_DNS_SERVERS}."
  log "Dominios de busca entregues por DHCP: ${DHCP_SEARCH_DOMAINS}."

  # Roda em segundo plano e usa "wait $PID" (em vez de um pipe em
  # primeiro plano) de proposito: o bash so garante que um trap (ver
  # "trap cleanup EXIT INT TERM" mais abaixo) interrompe e roda durante
  # um "wait" explicito - bloqueado dentro de um pipe em primeiro
  # plano, o SIGTERM do "docker compose stop"/painel fica pendente ate
  # o create_ap terminar sozinho (nunca termina, ele serve o hotspot
  # indefinidamente), e o Docker acaba forcando SIGKILL apos o prazo,
  # sem o cleanup rodar - deixando a interface virtual orfa (ver
  # remove_stale_virtual_interfaces acima).
  create_ap \
    "${virtual_interface_args[@]}" \
    --no-dns \
    --dhcp-dns "${DHCP_DNS_SERVERS}" \
    --country "${WIFI_COUNTRY}" \
    --freq-band "${band}" \
    -c "${channel}" \
    -g "${HOTSPOT_GATEWAY}" \
    "${WIFI_INTERFACE}" \
    "${CREATE_AP_INTERNET_INTERFACE}" \
    "${WIFI_SSID}" \
    "${WIFI_PASSWORD}" > >(tee "${CREATE_AP_LOG}") 2>&1 &
  CREATE_AP_PID=$!
  wait "${CREATE_AP_PID}" || status=$?
  CREATE_AP_PID=

  if [[ "${status}" -ne 0 ]] && grep -qi 'can not be a station.*and an AP at the same time' "${CREATE_AP_LOG}"; then
    log "ERRO: ${WIFI_INTERFACE} ainda esta associado como estacao (cliente Wi-Fi) - o driver desta placa nao suporta AP+estacao simultaneos, e trocar de canal/banda nao muda isso. Use outra placa para o hotspot ou outra interface Wi-Fi compatível com AP+STA."
    return 2
  fi

  if [[ "${status}" -ne 0 ]] && grep -qi 'RTNETLINK answers: Resource busy' "${CREATE_AP_LOG}"; then
    log "ERRO: o driver deixou criar a interface AP virtual, mas recusou configura-la (RTNETLINK: Resource busy). Desconecte ${WIFI_INTERFACE} da rede Wi-Fi cliente para o Bindnet usar automaticamente --no-virt, ou atualize/reverta o kernel/firmware do adaptador. Trocar de canal nao resolve esta causa."
    return 2
  fi

  return "${status}"
}

# start_hotspot_auto tenta, em ordem de menor interferencia, cada canal
# candidato da banda informada. Devolve 0 assim que um canal funcionar,
# 2 se try_create_ap detectar uma falha independente do canal (aborta a
# varredura imediatamente - continuar testando canais so repetiria o
# mesmo erro) ou 1 se todos os candidatos da banda forem rejeitados por
# outro motivo.
start_hotspot_auto() {
  local band="$1"
  local channel
  local status

  rank_channels_for_band "${band}"
  if [[ "${#RANKED_CHANNELS[@]}" -eq 0 ]]; then
    log "AVISO: nenhum canal candidato disponivel para ${band}GHz."
    return 1
  fi

  for channel in "${RANKED_CHANNELS[@]}"; do
    status=0
    try_create_ap "${band}" "${channel}" || status=$?
    if [[ "${status}" -eq 0 ]]; then
      return 0
    fi
    if [[ "${status}" -eq 2 ]]; then
      return 2
    fi
    log "AVISO: canal ${channel} (${band}GHz) rejeitado pelo adaptador; tentando proximo candidato."
  done
  return 1
}

CREATE_AP_HELP="$(create_ap --help 2>&1 || true)"

if ! grep -q -- '--no-dns' <<< "${CREATE_AP_HELP}"; then
  log "ERRO: a versao baixada do create_ap nao suporta --no-dns."
  exit 1
fi

if ! grep -q -- '--dhcp-dns' <<< "${CREATE_AP_HELP}"; then
  log "ERRO: a versao baixada do create_ap nao suporta --dhcp-dns."
  exit 1
fi

# Guarda a escolha original do usuario antes de resolve_wifi_band
# sobrescrever WIFI_FREQ_BAND - so tenta a banda alternativa (fallback)
# quando o proprio usuario deixou banda E canal em "auto" (modo
# totalmente automatico); um canal ou banda fixados explicitamente sao
# respeitados como estao, sem fallback.
CREATE_AP_PID=
UPLINK_MONITOR_PID=""

cleanup() {
  log "Encerrando hotspot em ${WIFI_INTERFACE}."
  create_ap --stop "${WIFI_INTERFACE}" >/dev/null 2>&1 || true
  # Reforco: "create_ap --stop" ja resolve e sinaliza o PID certo
  # sozinho, mas se por algum motivo ele nao encontrar a instancia,
  # sinaliza direto o PID que guardamos - create_ap trata SIGINT como
  # pedido de parada limpa (clean_exit), igual --stop.
  [[ -n "${CREATE_AP_PID}" ]] && kill -INT "${CREATE_AP_PID}" >/dev/null 2>&1 || true
  cleanup_bindnet_uplink
}
trap cleanup EXIT INT TERM

ORIGINAL_WIFI_FREQ_BAND="${WIFI_FREQ_BAND}"
resolve_wifi_band

# remove_stale_virtual_interfaces apaga interfaces "apN" orfas de uma
# execucao anterior deste mesmo script. O create_ap sempre nomeia as
# interfaces virtuais que cria como "ap" + numero incremental
# (alloc_new_iface no create_ap) - nunca outro nome - entao o filtro
# "^ap[0-9]+$" so pega interfaces que o proprio create_ap criou, nunca
# WIFI_INTERFACE nem nada gerenciado pelo usuario/NetworkManager. Isso
# e necessario porque "network_mode: host" faz a interface virtual
# existir no host de verdade, nao só dentro do container - se o
# container anterior morrer por SIGKILL (ex.: timeout do "docker stop"),
# o trap acima nunca roda e a interface fica presa, fazendo a proxima
# tentativa falhar com "RTNETLINK answers: Resource busy" (a combinacao
# suportada pela placa so permite uma interface AP por vez).
remove_stale_virtual_interfaces() {
  local iface
  for iface in $(ip -o link show 2>/dev/null | awk -F': ' '{print $2}' | grep -E '^ap[0-9]+$' || true); do
    log "AVISO: interface virtual '${iface}' orfa de uma execucao anterior encontrada; removendo antes de tentar de novo."
    iw dev "${iface}" del >/dev/null 2>&1 || true
  done
}
remove_stale_virtual_interfaces

setup_bindnet_virtual_uplink
start_uplink_monitor

if [[ "${WIFI_CHANNEL}" != "auto" ]]; then
  if ! [[ "${WIFI_CHANNEL}" =~ ^[0-9]+$ ]]; then
    log "ERRO: WIFI_CHANNEL deve ser numerico ou auto."
    exit 1
  fi
  STATUS=0
  try_create_ap "${WIFI_FREQ_BAND}" "${WIFI_CHANNEL}" || STATUS=$?
  [[ "${STATUS}" -eq 2 ]] && exit 1
  exit "${STATUS}"
fi

STATUS=0
start_hotspot_auto "${WIFI_FREQ_BAND}" || STATUS=$?
[[ "${STATUS}" -eq 0 ]] && exit 0
[[ "${STATUS}" -eq 2 ]] && exit 1

if [[ "${ORIGINAL_WIFI_FREQ_BAND}" == "auto" ]]; then
  FALLBACK_BAND="2.4"
  [[ "${WIFI_FREQ_BAND}" == "2.4" ]] && FALLBACK_BAND="5"
  log "AVISO: nenhum canal funcionou em ${WIFI_FREQ_BAND}GHz; tentando banda alternativa ${FALLBACK_BAND}GHz."
  STATUS=0
  start_hotspot_auto "${FALLBACK_BAND}" || STATUS=$?
  [[ "${STATUS}" -eq 0 ]] && exit 0
  [[ "${STATUS}" -eq 2 ]] && exit 1
fi

log "ERRO: nao foi possivel iniciar o hotspot em nenhum canal/banda testado."
exit 1
