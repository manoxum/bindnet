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

DHCP_SEARCH_DOMAINS="$(normalize_search_domains)"
DHCP_DOMAIN="${DHCP_SEARCH_DOMAINS%%,*}"
export DHCP_SEARCH_DOMAINS DHCP_DOMAIN

# Resolve valores "auto" antes de qualquer tentativa de create_ap:
# banda/canal (resolve_wifi_band, chamada mais abaixo) e a interface de
# internet (resolve_internet_interface). warn_if_concurrent_ap_sta_risky
# so loga um aviso quando WIFI_INTERFACE == INTERNET_INTERFACE - nunca
# bloqueia, quem decide se funciona de fato e o create_ap.
resolve_internet_interface
warn_if_concurrent_ap_sta_risky

# try_create_ap tenta subir o hotspot num canal/banda especificos. Devolve
# sucesso (0) se create_ap encerrar com exit code 0 (parada limpa via
# sinal, ver cleanup/trap mais abaixo) ou falha (!=0) se o create_ap
# rejeitar o canal (ex.: "adapter can not transmit").
try_create_ap() {
  local band="$1"
  local channel="$2"

  log "Preparando hotspot '${WIFI_SSID}' em ${WIFI_INTERFACE}, internet via ${INTERNET_INTERFACE}."
  log "Regiao Wi-Fi: ${WIFI_COUNTRY}; banda: ${band}GHz; canal: ${channel}."
  log "Gateway do hotspot: ${HOTSPOT_GATEWAY}; DNS entregue por DHCP: ${HOTSPOT_GATEWAY}."
  log "Dominios de busca entregues por DHCP: ${DHCP_SEARCH_DOMAINS}."

  create_ap \
    --no-dns \
    --dhcp-dns "${HOTSPOT_GATEWAY}" \
    --country "${WIFI_COUNTRY}" \
    --freq-band "${band}" \
    -c "${channel}" \
    -g "${HOTSPOT_GATEWAY}" \
    "${WIFI_INTERFACE}" \
    "${INTERNET_INTERFACE}" \
    "${WIFI_SSID}" \
    "${WIFI_PASSWORD}"
}

# start_hotspot_auto tenta, em ordem de menor interferencia, cada canal
# candidato da banda informada. Sai do script (via try_create_ap) assim
# que um canal funcionar; devolve falha se todos os candidatos da banda
# forem rejeitados pelo adaptador.
start_hotspot_auto() {
  local band="$1"
  local channel

  rank_channels_for_band "${band}"
  if [[ "${#RANKED_CHANNELS[@]}" -eq 0 ]]; then
    log "AVISO: nenhum canal candidato disponivel para ${band}GHz."
    return 1
  fi

  for channel in "${RANKED_CHANNELS[@]}"; do
    if try_create_ap "${band}" "${channel}"; then
      return 0
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
ORIGINAL_WIFI_FREQ_BAND="${WIFI_FREQ_BAND}"
resolve_wifi_band

cleanup() {
  log "Encerrando hotspot em ${WIFI_INTERFACE}."
  create_ap --stop "${WIFI_INTERFACE}" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

if [[ "${WIFI_CHANNEL}" != "auto" ]]; then
  if ! [[ "${WIFI_CHANNEL}" =~ ^[0-9]+$ ]]; then
    log "ERRO: WIFI_CHANNEL deve ser numerico ou auto."
    exit 1
  fi
  try_create_ap "${WIFI_FREQ_BAND}" "${WIFI_CHANNEL}"
  exit $?
fi

if start_hotspot_auto "${WIFI_FREQ_BAND}"; then
  exit 0
fi

if [[ "${ORIGINAL_WIFI_FREQ_BAND}" == "auto" ]]; then
  FALLBACK_BAND="2.4"
  [[ "${WIFI_FREQ_BAND}" == "2.4" ]] && FALLBACK_BAND="5"
  log "AVISO: nenhum canal funcionou em ${WIFI_FREQ_BAND}GHz; tentando banda alternativa ${FALLBACK_BAND}GHz."
  if start_hotspot_auto "${FALLBACK_BAND}"; then
    exit 0
  fi
fi

log "ERRO: nao foi possivel iniciar o hotspot em nenhum canal/banda testado."
exit 1
