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
WIFI_FREQ_BAND="${WIFI_FREQ_BAND:-2.4}"

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

interface_phy() {
  iw dev "${WIFI_INTERFACE}" info 2>/dev/null | awk '/wiphy/{print $2}'
}

band_supported() {
  local band="$1"
  local phy
  local info

  phy="$(interface_phy)"
  [[ -n "${phy}" ]] || return 1

  info="$(iw "phy${phy}" info 2>/dev/null || true)"
  [[ -n "${info}" ]] || return 1

  if [[ "${band}" == "5" ]]; then
    grep -Eq '\* 5[0-9]{3}(\.[0-9])? MHz' <<< "${info}"
  else
    grep -Eq '\* 24[0-9]{2}(\.[0-9])? MHz' <<< "${info}"
  fi
}

resolve_wifi_band() {
  if [[ "${WIFI_FREQ_BAND}" != "auto" ]]; then
    if [[ "${WIFI_FREQ_BAND}" != "2.4" && "${WIFI_FREQ_BAND}" != "5" ]]; then
      log "ERRO: WIFI_FREQ_BAND deve ser 2.4, 5 ou auto."
      exit 1
    fi
    return
  fi

  if [[ "${WIFI_CHANNEL}" =~ ^[0-9]+$ ]]; then
    if (( WIFI_CHANNEL >= 1 && WIFI_CHANNEL <= 14 )); then
      WIFI_FREQ_BAND="2.4"
    else
      WIFI_FREQ_BAND="5"
    fi
    log "Banda Wi-Fi inferida a partir do canal ${WIFI_CHANNEL}: ${WIFI_FREQ_BAND}GHz."
    return
  fi

  ip link set "${WIFI_INTERFACE}" up >/dev/null 2>&1 || true

  if band_supported "5"; then
    WIFI_FREQ_BAND="5"
  elif band_supported "2.4"; then
    WIFI_FREQ_BAND="2.4"
  else
    WIFI_FREQ_BAND="2.4"
    log "AVISO: nao foi possivel detectar as bandas suportadas por ${WIFI_INTERFACE}; usando 2.4GHz."
  fi
  log "Banda Wi-Fi automatica escolhida: ${WIFI_FREQ_BAND}GHz."
}

freq_to_channel() {
  local freq="$1"
  # Versoes recentes do "iw" retornam a frequencia com casas decimais
  # (ex.: "2467.0" em vez de "2467") - bash nao faz aritmetica com
  # ponto flutuante em "(( ))", entao trunca antes de comparar.
  freq="${freq%%.*}"

  if (( freq == 2484 )); then
    printf '14\n'
  elif (( freq >= 2412 && freq <= 2472 )); then
    printf '%s\n' "$(((freq - 2407) / 5))"
  elif (( freq >= 5000 && freq <= 5900 )); then
    printf '%s\n' "$(((freq - 5000) / 5))"
  fi
}

channel_abs() {
  local value="$1"
  if (( value < 0 )); then
    printf '%s\n' "$((-value))"
  else
    printf '%s\n' "${value}"
  fi
}

score_channel() {
  local candidate="$1"
  local observed="$2"
  local distance

  if [[ "${WIFI_FREQ_BAND}" == "2.4" ]]; then
    distance="$(channel_abs "$((candidate - observed))")"
    if (( distance == 0 )); then
      printf '8\n'
    elif (( distance == 1 )); then
      printf '6\n'
    elif (( distance == 2 )); then
      printf '4\n'
    elif (( distance <= 4 )); then
      printf '1\n'
    else
      printf '0\n'
    fi
    return
  fi

  if (( candidate == observed )); then
    printf '1\n'
  else
    printf '0\n'
  fi
}

candidate_channels() {
  if [[ -n "${WIFI_CHANNEL_CANDIDATES:-}" ]]; then
    tr ',;' '  ' <<< "${WIFI_CHANNEL_CANDIDATES}"
    return
  fi

  case "${WIFI_FREQ_BAND}" in
    5) printf '36 40 44 48 149 153 157 161\n' ;;
    2.4) printf '1 6 11\n' ;;
    *)
      log "ERRO: WIFI_FREQ_BAND deve ser 2.4 ou 5 para selecao automatica de canal."
      exit 1
      ;;
  esac
}

# rank_channels_for_band preenche RANKED_CHANNELS (array global) com os
# canais candidatos da banda informada, ordenados do menos para o mais
# interferido - usada por start_hotspot_auto para tentar canal por canal
# ate um que o create_ap realmente consiga usar (nem toda banda/canal
# reportado como "suportado" pelo "iw phy info" e transmissivel de fato,
# ver ERRO "adapter can not transmit" do create_ap).
rank_channels_for_band() {
  local band="$1"
  local channels
  local scan
  local freq
  local observed_channel
  local observed_channels=()
  local candidate
  local score
  local -a scored=()

  WIFI_FREQ_BAND="${band}"
  channels="$(candidate_channels)"
  ip link set "${WIFI_INTERFACE}" up >/dev/null 2>&1 || true
  scan="$(iw dev "${WIFI_INTERFACE}" scan 2>/dev/null || iw dev "${WIFI_INTERFACE}" scan ap-force 2>/dev/null || true)"

  while read -r freq; do
    [[ -n "${freq}" ]] || continue
    observed_channel="$(freq_to_channel "${freq}")"
    [[ -n "${observed_channel}" ]] || continue
    observed_channels+=("${observed_channel}")
  done < <(awk '/freq:/ {print $2}' <<< "${scan}")

  for candidate in ${channels}; do
    if ! [[ "${candidate}" =~ ^[0-9]+$ ]]; then
      log "ERRO: WIFI_CHANNEL_CANDIDATES contem canal invalido: ${candidate}"
      exit 1
    fi

    score=0
    for observed_channel in "${observed_channels[@]}"; do
      score=$((score + $(score_channel "${candidate}" "${observed_channel}")))
    done
    scored+=("${score} ${candidate}")
  done

  RANKED_CHANNELS=()
  if [[ "${#scored[@]}" -gt 0 ]]; then
    while read -r pair; do
      RANKED_CHANNELS+=("${pair#* }")
    done < <(printf '%s\n' "${scored[@]}" | sort -n)
  fi

  if [[ -z "${scan}" ]]; then
    log "AVISO: varredura Wi-Fi indisponivel; ordem de canais candidatos (${band}GHz) e arbitraria."
  else
    log "Canais candidatos ordenados por interferencia (${band}GHz): ${RANKED_CHANNELS[*]:-nenhum} (redes avaliadas ${#observed_channels[@]})."
  fi
}

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
