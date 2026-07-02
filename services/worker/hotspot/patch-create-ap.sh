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
  }
' "${script}" > "${tmp}"

cat "${tmp}" > "${script}"
rm "${tmp}"
