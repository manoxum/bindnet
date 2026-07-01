#!/bin/sh
set -eu

log() {
  printf '[dns-provider] %s\n' "$*"
}

ip_existe() {
  ip -4 addr show | grep -Eq "(^|[[:space:]])$1/"
}

preparar_tlds_locais() {
  tlds_brutos="$(printf '%s' "${DNS_LOCAL_TLDS:-local,test,example}" | tr ',;' '  ')"
  DNS_LOCAL_TLD_ZONES=""
  DNS_LOCAL_TLD_REGEX=""

  for tld in $tlds_brutos; do
    tld="$(printf '%s' "${tld#.}" | tr '[:upper:]' '[:lower:]')"
    case "$tld" in
      ""|*[!a-z0-9-]*|-*|*-)
        log "ERRO: DNS_LOCAL_TLDS contem TLD invalido: ${tld}"
        exit 1
        ;;
    esac

    case " $DNS_LOCAL_TLD_ZONES " in
      *" $tld "*) continue ;;
    esac

    DNS_LOCAL_TLD_ZONES="${DNS_LOCAL_TLD_ZONES}${DNS_LOCAL_TLD_ZONES:+ }${tld}"
    DNS_LOCAL_TLD_REGEX="${DNS_LOCAL_TLD_REGEX}${DNS_LOCAL_TLD_REGEX:+|}${tld}"
  done

  if [ -z "$DNS_LOCAL_TLD_ZONES" ]; then
    log "ERRO: DNS_LOCAL_TLDS deve conter pelo menos um TLD local."
    exit 1
  fi

  export DNS_LOCAL_TLD_ZONES DNS_LOCAL_TLD_REGEX
  log "TLDs locais resolvidos pelo CoreDNS: ${DNS_LOCAL_TLD_ZONES}"
}

renderizar_corefile() {
  COREDNS_RENDERED_CONFIG="${COREDNS_RENDERED_CONFIG:-/tmp/Corefile}"

  # CoreDNS precisa escutar em 127.0.0.1 porque e o endereco que o
  # /etc/systemd/resolved.conf.d do host usa para rotear os TLDs locais;
  # sem isso o host nunca alcanca este servidor (bug original: so
  # escutava no HOTSPOT_GATEWAY).
  BIND_IPS="127.0.0.1"
  [ -n "${DOCKER_HOST_GATEWAY:-}" ] && BIND_IPS="${BIND_IPS} ${DOCKER_HOST_GATEWAY}"
  BIND_IPS="${BIND_IPS} ${HOTSPOT_GATEWAY}"

  sed \
    -e "s@__DNS_LOCAL_TLD_ZONES__@${DNS_LOCAL_TLD_ZONES}@g" \
    -e "s@__DNS_LOCAL_TLD_REGEX__@${DNS_LOCAL_TLD_REGEX}@g" \
    -e "s@__BIND_IPS__@${BIND_IPS}@g" \
    -e "s@__ANSWER_IP__@${HOTSPOT_GATEWAY}@g" \
    /etc/coredns/Corefile > "$COREDNS_RENDERED_CONFIG"

  export COREDNS_RENDERED_CONFIG
}

preparar_tlds_locais
renderizar_corefile

IPS_OBRIGATORIOS="127.0.0.1 ${DOCKER_HOST_GATEWAY:-} ${HOTSPOT_GATEWAY:-}"
TEMPO_LIMITE="${COREDNS_WAIT_TIMEOUT:-90}"

log "Aguardando IPs locais necessarios para bind do CoreDNS: ${IPS_OBRIGATORIOS}"

inicio="$(date +%s)"
while :; do
  faltando=""
  for ip in $IPS_OBRIGATORIOS; do
    [ -n "$ip" ] || continue
    if [ "$ip" = "127.0.0.1" ]; then
      continue
    fi
    if ! ip_existe "$ip"; then
      faltando="$faltando $ip"
    fi
  done

  if [ -z "$faltando" ]; then
    break
  fi

  agora="$(date +%s)"
  if [ $((agora - inicio)) -ge "$TEMPO_LIMITE" ]; then
    log "ERRO: IPs ainda ausentes apos ${TEMPO_LIMITE}s:${faltando}"
    log "Verifique se o hotspot subiu e se DOCKER_HOST_GATEWAY/HOTSPOT_GATEWAY estao corretos."
    exit 1
  fi

  sleep 2
done

log "Iniciando CoreDNS com split-horizon."
if [ "$#" -eq 2 ] && [ "$1" = "-conf" ]; then
  exec /usr/local/bin/coredns -conf "$COREDNS_RENDERED_CONFIG"
fi

exec /usr/local/bin/coredns "$@"