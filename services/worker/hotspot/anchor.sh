#!/usr/bin/env bash
# anchor.sh - "rede ancora": a rede Wi-Fi em cujo canal o AP deve subir.
# Sourced pelo entrypoint.sh; usa "log" e as variaveis
# WIFI_ANCHOR_SSID/WIFI_ANCHOR_CHANNEL ja carregadas do banco.
#
# POR QUE ISTO EXISTE: num radio unico, o AP e a associacao Wi-Fi
# cliente tem que dividir a MESMA frequencia ("#channels <= 1" em
# "iw phy info", ver warn_ap_sta_channel_lock em radio_caps.sh). A
# escolha automatica de canal ranqueia por MENOR interferencia, ou seja,
# escolhe justamente o canal onde nao ha rede nenhuma - o pior possivel
# para quem quer continuar conectado a uma rede com o hotspot no ar.
# Confirmado ao vivo: com 15 redes visiveis e nenhuma no canal 1, o
# hotspot subiu no canal 1 e a maquina deixou de conseguir se associar a
# qualquer uma delas (o router de casa estava no 10, outro hotspot no
# 11).
#
# Ancorar inverte esse criterio: o operador escolhe no painel a rede a
# que costuma se conectar, e o AP sobe no canal DELA. Custa interferencia
# (o AP passa a dividir canal com outras redes) e da acesso a uma rede
# de cada vez - as demais, em canais diferentes, ficam inalcancaveis
# enquanto o hotspot estiver no ar. E o trade-off deliberado.

# anchor_band_channel imprime "banda canal" (ex.: "2.4 10") da rede
# ancora configurada, ou falha se nao houver ancora utilizavel.
#
# A fonte e o canal MEMORIZADO (WIFI_ANCHOR_CHANNEL), gravado pelo
# painel no momento em que o operador escolhe a rede e mantido
# atualizado pelo backend - e nao uma varredura feita aqui. Motivo: um
# "iw dev ... scan" ativo derruba o beacon do AP e chega a desconectar
# clientes, e seria feito exatamente no caminho em que o hotspot esta
# subindo. O canal memorizado tambem e o que faz o hotspot subir certo
# quando a rede ancora esta fora do ar no momento (router desligado,
# maquina noutro lugar) - o router raramente muda de canal.
anchor_band_channel() {
  local ssid="${WIFI_ANCHOR_SSID:-}"
  local channel="${WIFI_ANCHOR_CHANNEL:-}"

  [[ -n "${ssid}" ]] || return 1
  if ! [[ "${channel}" =~ ^[0-9]+$ ]]; then
    # ">&2" obrigatorio: esta funcao comunica o resultado pelo stdout
    # ("banda canal"), e o chamador a le com "$(anchor_band_channel)".
    # Um "log" em stdout seria capturado junto com o valor - engolido
    # pela substituicao de comando e nunca visto no log do hotspot.
    log "AVISO: rede ancora '${ssid}' configurada, mas sem canal conhecido - escolha-a de novo na aba Radio do painel para gravar o canal. Caindo na selecao automatica por interferencia." >&2
    return 1
  fi

  # Canais 1-14 sao 2.4GHz; o resto e 5GHz. Mesma classificacao usada em
  # sta_current_band_channel (sta_link.sh) e resolve_wifi_band
  # (channel.sh) - nao ha canal ambiguo entre as duas bandas.
  if (( channel >= 1 && channel <= 14 )); then
    printf '2.4 %s\n' "${channel}"
  else
    printf '5 %s\n' "${channel}"
  fi
}
