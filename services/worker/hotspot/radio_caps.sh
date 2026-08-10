#!/usr/bin/env bash
# radio_caps.sh - o que o radio (phy) desta placa realmente aceita
# fazer ao mesmo tempo: AP junto com estacao Wi-Fi cliente, e em quantas
# frequencias distintas. Extraido de interfaces.sh pra manter cada
# arquivo num unico dominio (ver CLAUDE.md - limite de ~200 linhas):
# interfaces.sh cuida de "que interface usar" (resolucao/ranking de
# candidatas), este cuida de "o que o radio suporta".
#
# Sourced pelo entrypoint.sh depois de channel.sh (usa interface_phy de
# la). Nenhuma funcao aqui bloqueia a subida do hotspot: todas so logam.
# Quem decide se AP+STA funciona de fato e o proprio create_ap, mesma
# filosofia ja usada pra canal/banda ("o hotspot nunca trava por falta
# de varredura").

# warn_if_concurrent_ap_sta_risky avisa cedo quando WIFI_INTERFACE e
# INTERNET_INTERFACE sao a mesma placa fisica (hotspot + internet no
# mesmo radio, modo AP+STA concorrente). Nunca bloqueia: quem decide se
# funciona de fato e o proprio create_ap, mesma filosofia ja usada para
# canal/banda ("o hotspot nunca trava por falta de varredura").
warn_if_concurrent_ap_sta_risky() {
  if [[ "${WIFI_INTERFACE}" != "${REAL_INTERNET_INTERFACE}" ]]; then
    return
  fi

  log "AVISO: WIFI_INTERFACE e INTERNET_INTERFACE sao a mesma placa (${WIFI_INTERFACE}) - modo AP+STA concorrente no mesmo radio, requer suporte do driver/chipset; o create_ap decide se funciona de fato."

  local phy
  # "|| true": interface_phy termina num pipe, e sob o "set -euo
  # pipefail" do entrypoint um pipe reprovado numa substituicao de
  # comando aborta o hotspot inteiro - ver o comentario detalhado em
  # ap_sta_max_channels mais abaixo.
  phy="$(interface_phy || true)"
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

# ap_sta_max_channels imprime quantas frequencias DIFERENTES o phy
# aceita manter no ar ao mesmo tempo na combinacao que inclui AP (o
# "#channels <= N" de "iw phy info"), ou nada se nao der pra ler.
#
# O bloco "valid interface combinations" tem uma entrada por "*", cada
# uma podendo ocupar varias linhas - por isso o sed recorta o bloco (que
# termina na proxima linha com um unico nivel de indentacao), o tr junta
# tudo e reparte por "*", e so entao da pra casar "AP" e "#channels" na
# MESMA entrada. Depende do GNU grep (-o), que o Dockerfile ja instala
# no lugar do applet do BusyBox.
#
# O "|| true" final NAO e decorativo: com o "set -euo pipefail" do
# entrypoint, um grep que nao acha nada (caso normal numa placa cujo
# phy nao reporta combinacoes) reprova o pipeline inteiro, e o
# "channels=$(ap_sta_max_channels ...)" do chamador abortaria o hotspot
# na subida - uma funcao que so LOGA um aviso jamais pode derrubar o
# servico. Mesmo risco ja confirmado ao vivo em
# ensure_wifi_radio_unblocked (regulatory.sh) e sta_link_probe
# (sta_link.sh).
ap_sta_max_channels() {
  local phy="$1"
  iw "phy${phy}" info 2>/dev/null \
    | sed -n '/valid interface combinations/,/^\t[A-Za-z]/p' \
    | tr '\n' ' ' | tr '*' '\n' \
    | grep -E '\bAP\b' \
    | grep -oE '#channels <= [0-9]+' \
    | head -n 1 \
    | grep -oE '[0-9]+' || true
}

# warn_ap_sta_channel_lock explica, uma vez na subida, a consequencia
# pratica de "#channels <= 1": um radio unico so transmite numa
# frequencia por vez, entao o AP e a associacao Wi-Fi cliente TEM que
# dividir o mesmo canal. O hotspot ja resolve isso travando o AP no
# canal da estacao (ver sta_current_band_channel em sta_link.sh), mas
# so na direcao STA -> AP: com o AP ja no ar, conectar a maquina a uma
# rede Wi-Fi que esteja em outro canal nao funciona, porque mover o AP
# depois exigiria derrubar todos os clientes.
#
# Nunca bloqueia - e so o aviso que evita o operador concluir que "o
# hotspot esta quebrado" quando na verdade tentou a ordem inversa.
# Vale para qualquer cenario com estacao preservada, nao so Wi-Fi para
# Wi-Fi: partilhar do Ethernet mantendo o Wi-Fi cliente ligado usa
# exatamente a mesma topologia de radio.
warn_ap_sta_channel_lock() {
  local phy
  # "|| true" pelo mesmo motivo do comentario em ap_sta_max_channels:
  # interface_phy tambem termina num pipe que pode reprovar.
  phy="$(interface_phy || true)"
  [[ -n "${phy}" ]] || return 0

  local channels
  channels="$(ap_sta_max_channels "${phy}")"
  # Rejeita qualquer coisa que nao seja um numero: um formato inesperado
  # de "iw phy info" viraria "[[ ... -gt 1 ]]" com operando invalido, que
  # tambem aborta sob set -e.
  if [[ ! "${channels}" =~ ^[0-9]+$ ]]; then
    return 0
  fi
  if [[ "${channels}" -gt 1 ]]; then
    log "Placa ${WIFI_INTERFACE} (phy${phy}) aceita AP e estacao em ate ${channels} canais distintos - hotspot e Wi-Fi cliente nao precisam dividir o mesmo canal."
    return 0
  fi

  log "AVISO: ${WIFI_INTERFACE} (phy${phy}) so mantem AP e estacao no MESMO canal ('#channels <= 1'). Consequencia: conecte a maquina a rede Wi-Fi ANTES de iniciar o hotspot - o AP e travado no canal da estacao. Com o hotspot ja no ar, so da pra associar a redes que estejam nesse mesmo canal."
}
