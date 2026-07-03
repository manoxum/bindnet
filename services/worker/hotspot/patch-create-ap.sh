#!/usr/bin/env sh
set -eu

script="/usr/local/bin/create_ap"
tmp="$(mktemp)"

awk '
  {
    print
  }
  $0 == "dhcp-option-force=option:dns-server,${DHCP_DNS}" {
    print "dhcp-option-force=option:domain-search,${DHCP_SEARCH_DOMAINS:-local,test,example}"
    print "dhcp-option-force=option:domain-name,${DHCP_DOMAIN:-local}"
    print "log-dhcp"
    print "log-facility=/tmp/bindnet-dnsmasq-dhcp.log"
  }
  $0 == "    local FREQ=$1" {
    print "    FREQ=\"${FREQ%%.*}\"  # bindnet: iw novo devolve frequencia com casas decimais (ex.: 2427.0), [[ -eq/-lt/... ]] so aceita inteiro"
  }
' "${script}" > "${tmp}"

cat "${tmp}" > "${script}"
rm "${tmp}"
