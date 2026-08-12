#!/usr/bin/env bash
# regulatory.sh - diagnostico do dominio regulatorio Wi-Fi (iw reg get),
# extraido do entrypoint.sh principal pra manter cada arquivo focado num
# unico dominio (ver CLAUDE.md - limite de ~200 linhas por arquivo).
# Sourced pelo entrypoint.sh; usa "log" e a variavel WIFI_COUNTRY ja
# resolvida por ele. So loga informacao - nunca muda selecao de
# canal/banda nem falha o script.

# self_managed_regulatory_country imprime o pais que o phy
# "self-managed" de WIFI_INTERFACE esta realmente aplicando agora (via
# "iw reg get"), ou nada se a placa nao for self-managed/nao foi
# possivel ler. Firmwares self-managed ignoram --country/iw reg set
# vindo do userspace - esse e o pais que o firmware de fato aceita
# transmitir, independente do que WIFI_COUNTRY diz.
self_managed_regulatory_country() {
  local output
  output="$(iw reg get 2>&1 || true)"
  [[ -n "${output}" ]] || return 1
  awk '
    /\(self-managed\)/ { in_self_managed=1; next }
    in_self_managed && /^country / { print $2; exit }
  ' <<< "${output}" | tr -d ':'
}

# ensure_wifi_radio_unblocked confere se o radio Wi-Fi esta bloqueado
# via rfkill (soft ou hard) e tenta desbloquear via software antes de
# qualquer tentativa de canal - um radio bloqueado rejeita TODOS os
# canais/bandas igualmente com erros genericos do driver ("RTNETLINK
# answers: No error information"), indistinguivel de fora de uma trava
# regulatoria ou de canal especifico. Confirmado ao vivo: um "docker
# compose up --build" que recria o container hotspot com o create_ap
# ativo (interrompendo a limpeza normal no meio - ver stop_grace_period
# em docker-compose.services.yml) deixou o driver iwlwifi reportando
# bloqueio "hard" via rfkill, e um "rfkill unblock" comum (sem
# privilegio extra) resolveu de verdade - confirma que nao era um
# interruptor fisico genuino (esse continuaria bloqueado logo em
# seguida, ja que rfkill unblock nunca desfaz um bloqueio fisico real).
# So loga e tenta desbloquear - nunca falha o script por si so, ja que
# create_ap vai reportar o erro de qualquer forma se o desbloqueio nao
# funcionar (ex.: bloqueio fisico genuino).
# wifi_radio_blocked devolve 0 quando o radio Wi-Fi esta bloqueado por
# rfkill agora. Usada por attempt_hotspot_cycle (entrypoint.sh) pra
# desistir na hora em vez de gastar 5 ciclos x 11 canais candidatos
# contra um radio desligado - nenhum canal funciona com o radio em
# baixo, e cada tentativa ainda sujava o historico de canais com uma
# falsa "rejeicao do adaptador".
#
# "|| true" no grep pelo mesmo motivo detalhado em
# ensure_wifi_radio_unblocked abaixo: sob "set -euo pipefail" um grep
# sem match mata o script inteiro em silencio.
wifi_radio_blocked() {
  command -v rfkill >/dev/null 2>&1 || return 1
  local blocked
  blocked="$(rfkill list wifi 2>/dev/null | grep -Ei 'blocked: yes' || true)"
  [[ -n "${blocked}" ]]
}

ensure_wifi_radio_unblocked() {
  command -v rfkill >/dev/null 2>&1 || return 0

  # "|| true" no grep e necessario: com set -e/pipefail (topo do
  # entrypoint.sh), radio saudavel = grep sem match = exit 1 = pipeline
  # inteira "falha" = set -e mata o script inteiro aqui, silenciosamente,
  # sem nenhuma mensagem de erro (confirmado ao vivo: o hotspot parava
  # de fazer qualquer coisa logo apos "Configuracao operacional...",
  # nunca chegando nem no diagnostico regulatorio nem em erro nenhum).
  wifi_radio_blocked || return 0

  # Desbloquear o radio e desfazer uma acao do usuario: se ele acabou de
  # desligar o Wi-Fi no sistema (que faz rfkill do phy inteiro, matando
  # tambem o AP - e o mesmo radio), religa-lo aqui faria o interruptor
  # de Wi-Fi do sistema simplesmente nao funcionar enquanto o hotspot
  # estivesse ligado. Por isso o desbloqueio automatico vale SO quando o
  # operador pediu o hotspot explicitamente pelo painel
  # (HOTSPOT_START_REASON=manual, propagado pelo worker no "docker exec"
  # - ver ExecHotspotEntrypoint em internal/compose/compose.go); num
  # autostart de boot ou numa auto-recuperacao, respeita o bloqueio e
  # espera o radio voltar.
  #
  # O caso que originou esta funcao continua coberto: um bloqueio
  # espurio deixado por um "docker compose up --build" e destravado no
  # proximo start pelo painel, que e como o operador reage a "o hotspot
  # nao sobe".
  if [[ "${HOTSPOT_START_REASON:-auto}" != "manual" ]]; then
    log "AVISO: radio Wi-Fi bloqueado via rfkill, mas este start nao foi pedido pelo painel (autostart/auto-recuperacao) - respeitando o bloqueio em vez de religar o radio por conta propria. O hotspot sobe sozinho assim que o Wi-Fi for religado."
    return 0
  fi

  log "AVISO: radio Wi-Fi esta bloqueado via rfkill (soft ou hard) - todos os canais/bandas falhariam igualmente com erros genericos do driver. Tentando desbloquear via software."
  rfkill unblock wifi >/dev/null 2>&1 || true
  sleep 1

  if rfkill list wifi 2>/dev/null | grep -qEi 'blocked: yes'; then
    log "ERRO: nao foi possivel desbloquear o radio Wi-Fi via software - se houver uma tecla/interruptor fisico de Wi-Fi/modo aviao nesta maquina, verifique se nao esta acionado (rfkill unblock nao desfaz um bloqueio fisico genuino)."
  else
    log "Radio Wi-Fi desbloqueado com sucesso via rfkill."
  fi
}

# log_wifi_regulatory_info roda "iw reg get" e loga a saida linha a linha,
# destacando quando um phy self-managed esta preso num pais diferente
# de WIFI_COUNTRY - a causa mais comum de "adapter can not transmit" em
# todos os canais de uma banda, sem relacao com a logica de selecao de
# canal deste script.
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
  self_managed_country="$(self_managed_regulatory_country)"

  if [[ -n "${self_managed_country}" && "${self_managed_country}" != "${WIFI_COUNTRY}" ]]; then
    log "AVISO: phy self-managed preso no pais '${self_managed_country}', diferente de WIFI_COUNTRY='${WIFI_COUNTRY}'. Firmwares self-managed ignoram --country/iw reg set vindo do userspace - se todos os canais de uma banda falharem com 'adapter can not transmit', isso e provavelmente uma trava regulatoria do firmware da placa, nao um problema de configuracao do Bindnet."
  fi
}
