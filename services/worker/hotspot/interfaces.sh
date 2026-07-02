#!/usr/bin/env bash
# interfaces.sh - resolucao/avisos sobre a interface de internet,
# sourced pelo entrypoint.sh. Usa "log" e "interface_phy" (definida em
# channel.sh) e as variaveis WIFI_INTERFACE/INTERNET_INTERFACE ja
# resolvidas pelo entrypoint.sh.

# resolve_internet_interface so age quando INTERNET_INTERFACE=auto -
# mesma filosofia de WIFI_CHANNEL/WIFI_FREQ_BAND: "auto" e um valor
# explicito, nunca um comportamento implicito de variavel vazia (essa
# continua sendo barrada por "required" antes desta funcao rodar).
# Detecta a interface da rota padrao IPv4; falha alto se nao achar,
# nunca adivinha silenciosamente.
resolve_internet_interface() {
  if [[ "${INTERNET_INTERFACE}" != "auto" ]]; then
    return
  fi

  local detected
  detected="$(ip route show default 2>/dev/null | awk '/^default/ {for (i=1;i<=NF;i++) if ($i=="dev") print $(i+1)}' | head -n1)"
  if [[ -z "${detected}" ]]; then
    log "ERRO: INTERNET_INTERFACE=auto mas nenhuma rota padrao foi encontrada."
    exit 1
  fi
  INTERNET_INTERFACE="${detected}"
  log "Interface de internet automatica escolhida: ${INTERNET_INTERFACE}."
}

# warn_if_concurrent_ap_sta_risky avisa cedo quando WIFI_INTERFACE e
# INTERNET_INTERFACE sao a mesma placa fisica (hotspot + internet no
# mesmo radio, modo AP+STA concorrente). Nunca bloqueia: quem decide se
# funciona de fato e o proprio create_ap, mesma filosofia ja usada para
# canal/banda ("o hotspot nunca trava por falta de varredura").
warn_if_concurrent_ap_sta_risky() {
  if [[ "${WIFI_INTERFACE}" != "${INTERNET_INTERFACE}" ]]; then
    return
  fi

  log "AVISO: WIFI_INTERFACE e INTERNET_INTERFACE sao a mesma placa (${WIFI_INTERFACE}) - modo AP+STA concorrente no mesmo radio, requer suporte do driver/chipset; o create_ap decide se funciona de fato."

  local phy
  phy="$(interface_phy)"
  if [[ -z "${phy}" ]]; then
    log "AVISO: nao foi possivel identificar o phy de ${WIFI_INTERFACE} para checar combinacoes suportadas."
    return
  fi

  if iw "phy${phy}" info 2>/dev/null | grep -A5 'valid interface combinations' | grep -qi 'AP.*managed\|managed.*AP'; then
    log "Placa ${WIFI_INTERFACE} (phy${phy}) reporta suporte a AP+managed simultaneos."
  else
    log "AVISO: 'iw phy${phy} info' nao reporta uma combinacao AP+managed simultanea; o create_ap pode falhar ao tentar hotspot+internet na mesma placa."
  fi
}
