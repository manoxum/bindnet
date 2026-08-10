#!/usr/bin/env sh
set -eu

script="/usr/local/bin/create_ap"
tmp="$(mktemp)"

# As regras de substituicao vem ANTES do "{ print }" geral e usam
# "next": no awk todas as regras que casam rodam, em ordem, entao sem o
# next a linha original sairia junto com a substituta. As regras de
# insercao (que so acrescentam linhas depois da original) continuam
# depois do print geral.
awk '
  # bindnet: a troca de MAC da interface virtual (ap0) tem que acontecer
  # com ela DESLIGADA. O ieee80211_change_mac() do mac80211 devolve
  # -EBUSY ("RTNETLINK answers: Resource busy") para qualquer mudanca de
  # MAC numa interface que esteja UP, e o create_ap original faz a troca
  # ANTES do "ip link set down" logo abaixo. Isso derrubava todo o modo
  # AP+STA (hotspot convivendo com a conexao Wi-Fi cliente) em
  # iwlwifi/AX211: o create_ap morria com "Maybe your WiFi adapter does
  # not fully support virtual interfaces".
  #
  # A ap0 chega UP porque o create_ap so consegue marca-la como
  # unmanaged no NetworkManager quando enxerga o NM vivo por D-Bus - de
  # dentro deste container isso nao responde, entao o NM do host levanta
  # a interface nova sozinho.
  #
  # NAO da pra simplesmente pular a troca e deixar a ap0 com a mesma MAC
  # da placa fisica (o caminho que o create_ap usa para alguns drivers
  # Realtek): o mac80211 recusa MAC identica entre uma interface AP e
  # uma managed (identical_mac_addr_allowed() so libera monitor,
  # P2P-device e AP/AP-VLAN), devolvendo -ENOTUNIQ no lugar.
  $0 == "if [[ $NO_VIRT -eq 0 && -n \"$NEW_MACADDR\" ]]; then" {
    print "if false; then  # bindnet: movido para depois do \"ip link set down\" abaixo"
    next
  }
  # A mesma atribuicao ja existia logo depois do "ip link set down",
  # restrita ao caminho --no-virt. Vale para os dois casos agora - e o
  # unico ponto do script em que a interface esta comprovadamente
  # desligada.
  $0 == "if [[ $NO_VIRT -eq 1 && -n \"$NEW_MACADDR\" ]]; then" {
    print "if [[ -n \"$NEW_MACADDR\" ]]; then  # bindnet: vale tambem com interface virtual"
    next
  }
  {
    print
  }
  $0 == "dhcp-option-force=option:dns-server,${DHCP_DNS}" {
    print "dhcp-option-force=option:domain-search,${DHCP_SEARCH_DOMAINS:-local,test,example}"
    print "dhcp-option-force=option:domain-name,${DHCP_DOMAIN:-local}"
    print "log-dhcp"
    # /var/log e nao /tmp: o kernel recusa (fs.protected_regular) abrir
    # com O_CREAT um arquivo de outro dono dentro de um diretorio sticky
    # world-writable - ver o comentario de DNSMASQ_DHCP_LOG em
    # entrypoint.sh, que precisa apontar para este mesmo caminho.
    print "log-facility=/var/log/bindnet-dnsmasq-dhcp.log"
  }
  $0 == "    local FREQ=$1" {
    print "    FREQ=\"${FREQ%%.*}\"  # bindnet: iw novo devolve frequencia com casas decimais (ex.: 2427.0), [[ -eq/-lt/... ]] so aceita inteiro"
  }
' "${script}" > "${tmp}"

cat "${tmp}" > "${script}"
rm "${tmp}"

# Todo patch acima casa linhas LITERAIS do create_ap baixado do
# upstream. Se qualquer uma delas mudar de forma, o awk passa direto e o
# hotspot subiria com um comportamento silenciosamente diferente do que
# este repo documenta (ver RULE.md) - por isso confere e falha alto aqui,
# no build da imagem, em vez de na placa Wi-Fi do usuario.
assert_patched() {
  if ! grep -q -- "$1" "${script}"; then
    echo "[patch-create-ap] ERRO: patch nao aplicado (ancora perdida): $2" >&2
    echo "[patch-create-ap] o create_ap do upstream provavelmente mudou; revise patch-create-ap.sh" >&2
    exit 1
  fi
}

assert_patched 'log-facility=/var/log/bindnet-dnsmasq-dhcp.log' 'log de DHCP do dnsmasq'
assert_patched 'FREQ="${FREQ%%.*}"' 'frequencia com casas decimais'
assert_patched 'if false; then  # bindnet' 'troca de MAC precoce da interface virtual'
assert_patched 'if \[\[ -n "$NEW_MACADDR" \]\]; then  # bindnet' 'troca de MAC apos desligar a interface'
