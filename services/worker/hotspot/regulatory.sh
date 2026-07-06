#!/usr/bin/env bash
# regulatory.sh - diagnostico do dominio regulatorio Wi-Fi (iw reg get),
# extraido do entrypoint.sh principal pra manter cada arquivo focado num
# unico dominio (ver CLAUDE.md - limite de ~200 linhas por arquivo).
# Sourced pelo entrypoint.sh; usa "log" e a variavel WIFI_COUNTRY ja
# resolvida por ele. So loga informacao - nunca muda selecao de
# canal/banda nem falha o script.

# log_wifi_regulatory_info roda "iw reg get" e loga a saida linha a linha,
# destacando quando um phy "self-managed" (regulatorio imposto pelo
# firmware da placa, que ignora "iw reg set"/--country do create_ap) esta
# preso num pais diferente de WIFI_COUNTRY - a causa mais comum de
# "adapter can not transmit" em todos os canais de uma banda, sem relacao
# com a logica de selecao de canal deste script.
log_wifi_regulatory_info() {
  local output
  output="$(iw reg get 2>&1 || true)"
  if [[ -z "${output}" ]]; then
    log "AVISO: nao foi possivel obter o dominio regulatorio Wi-Fi (iw reg get)."
    return
  fi

  log "Dominio regulatorio Wi-Fi (iw reg get):"
  while IFS= read -r line; do
    log "  ${line}"
  done <<< "${output}"

  local self_managed_country
  self_managed_country="$(awk '
    /\(self-managed\)/ { in_self_managed=1; next }
    in_self_managed && /^country / { print $2; exit }
  ' <<< "${output}" | tr -d ':')"

  if [[ -n "${self_managed_country}" && "${self_managed_country}" != "${WIFI_COUNTRY}" ]]; then
    log "AVISO: phy self-managed preso no pais '${self_managed_country}', diferente de WIFI_COUNTRY='${WIFI_COUNTRY}'. Firmwares self-managed ignoram --country/iw reg set vindo do userspace - se todos os canais de uma banda falharem com 'adapter can not transmit', isso e provavelmente uma trava regulatoria do firmware da placa, nao um problema de configuracao do Bindnet."
  fi
}
