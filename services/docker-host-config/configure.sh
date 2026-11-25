#!/bin/sh
# Alinha o Docker daemon do host com o DNS e a CA local do Bindnet.
#
# 1. DNS: "dns" em /etc/docker/daemon.json aponta para o gateway da rede
#    "proxy" (onde o dns-provider responde a view container).
# 2. Confianca na CA: instala o wrapper `bindnet-runc` em
#    /usr/local/lib/bindnet (com a CA baixada do backend) e o registra como
#    `default-runtime`. Todo container criado a partir dai recebe a CA
#    (bundle do sistema + NODE_EXTRA_CA_CERTS). Ver bindnet-runc.
# 3. daemon.json so vale depois de reiniciar o Docker. Se o daemon em
#    execucao ainda nao aplica algo (detectado pelo resolv.conf e pelo
#    ambiente deste proprio container), reinicia o Docker no host. Como isso
#    derruba este job e o proprio "docker compose up", o restart e feito por
#    uma unidade transitoria do systemd no host, que espera o Docker voltar e
#    roda "docker compose up -d" de novo para concluir o deploy. Nessa segunda
#    passada DOCKER_DNS_RESTART=false evita laco de restarts.
set -eu

DIR="${DOCKER_CONFIG_DIR:-/host-etc-docker}"
CFG="${DIR}/daemon.json"
GATEWAY="${DOCKER_DNS_GATEWAY:-10.91.0.1}"
FALLBACK="${DOCKER_DNS_FALLBACK:-1.1.1.1}"
RESOLV="${DOCKER_RESOLV_CONF:-/etc/resolv.conf}"
UNIT="bindnet-docker-dns-apply"
# Diretorio do wrapper: montado em BIN_DIR neste container e em HOST_BIN_DIR no host.
BIN_DIR="${BINDNET_BIN_DIR:-/host-bindnet}"
HOST_BIN_DIR="${BINDNET_HOST_BIN_DIR:-/usr/local/lib/bindnet}"
RUNTIME_NAME="bindnet-runc"
CA_URL="${BINDNET_CA_URL:-http://backend:8090/api/mesh/ca}"

log() { printf '[docker-host-config] %s\n' "$*"; }

if [ "${DOCKER_DNS_CONFIGURE:-true}" != "true" ]; then
  log "DOCKER_DNS_CONFIGURE != true: nada a fazer."
  exit 0
fi

if [ ! -d "${DIR}" ]; then
  log "AVISO: ${DIR} nao esta montado; nao e possivel ajustar o Docker do host."
  exit 0
fi

[ -f "${CFG}" ] || printf '{}\n' > "${CFG}"

if ! jq -e . "${CFG}" >/dev/null 2>&1; then
  log "ERRO: ${CFG} nao e um JSON valido; nada foi alterado."
  exit 1
fi

# ---------------------------------------------------------------- CA / wrapper
TRUST_ENABLED=false
if [ "${DOCKER_TRUST_CA:-true}" = "true" ] && [ -d "${BIN_DIR}" ]; then
  tmp_ca="${BIN_DIR}/.ca.crt.$$"
  if curl -fsS --retry 30 --retry-delay 2 --retry-connrefused --max-time 5 \
        -o "${tmp_ca}" "${CA_URL}" 2>/dev/null \
     && openssl x509 -in "${tmp_ca}" -noout >/dev/null 2>&1; then
    chmod 644 "${tmp_ca}"
    if ! cmp -s "${tmp_ca}" "${BIN_DIR}/ca.crt" 2>/dev/null; then
      mv "${tmp_ca}" "${BIN_DIR}/ca.crt"
      log "CA do Bindnet instalada em ${HOST_BIN_DIR}/ca.crt."
    else
      rm -f "${tmp_ca}"
    fi
    cp /opt/bindnet-runc "${BIN_DIR}/.bindnet-runc.$$"
    chmod 755 "${BIN_DIR}/.bindnet-runc.$$"
    mv "${BIN_DIR}/.bindnet-runc.$$" "${BIN_DIR}/${RUNTIME_NAME}"
    # Autoteste NO HOST antes de registrar como runtime padrao: o wrapper
    # precisa executar (python3) e repassar a chamada ao runc.
    if nsenter -t 1 -m -- env BINDNET_RUNC_REAL=/bin/true \
         "${HOST_BIN_DIR}/${RUNTIME_NAME}" selftest >/dev/null 2>&1; then
      TRUST_ENABLED=true
    else
      log "AVISO: o wrapper nao executa no host (python3 ausente?); confianca na CA NAO habilitada."
    fi
  else
    rm -f "${tmp_ca}"
    log "AVISO: nao foi possivel obter a CA em ${CA_URL}; confianca na CA NAO habilitada nesta execucao."
  fi
elif [ "${DOCKER_TRUST_CA:-true}" = "true" ]; then
  log "AVISO: ${BIN_DIR} nao esta montado; confianca na CA NAO habilitada."
fi

# ----------------------------------------------------------------- daemon.json
RUNTIME_PATH="${HOST_BIN_DIR}/${RUNTIME_NAME}"
dns_ok=false
jq -e --arg gw "${GATEWAY}" '(.dns // [])[0] == $gw' "${CFG}" >/dev/null && dns_ok=true
trust_cfg_ok=true
if [ "${TRUST_ENABLED}" = "true" ]; then
  trust_cfg_ok=false
  jq -e --arg n "${RUNTIME_NAME}" --arg p "${RUNTIME_PATH}" \
    '(.["default-runtime"] == $n) and (.runtimes[$n].path == $p)' "${CFG}" >/dev/null && trust_cfg_ok=true
fi

if [ "${dns_ok}" = "true" ] && [ "${trust_cfg_ok}" = "true" ]; then
  log "daemon.json ja esta alinhado."
else
  backup="${CFG}.bak.$(date +%Y%m%d%H%M%S)"
  cp -p "${CFG}" "${backup}"
  tmp="${DIR}/.daemon.json.bindnet.$$"
  trap 'rm -f "${tmp}"' EXIT
  jq --arg gw "${GATEWAY}" --arg fb "${FALLBACK}" --arg n "${RUNTIME_NAME}" \
     --arg p "${RUNTIME_PATH}" --argjson trust "${TRUST_ENABLED}" '
    ((.dns // []) - [$gw]) as $others
    | .dns = [$gw] + (if ($others | length) > 0 then $others
                      else ($fb | split(",") | map(select(length > 0))) end)
    | if $trust then
        .runtimes = ((.runtimes // {}) + {($n): {path: $p}})
        | .["default-runtime"] = $n
      else . end
  ' "${CFG}" > "${tmp}"
  jq -e . "${tmp}" >/dev/null
  chmod --reference="${CFG}" "${tmp}" 2>/dev/null || chmod 644 "${tmp}"
  mv "${tmp}" "${CFG}"
  trap - EXIT
  log "daemon.json atualizado (backup em ${backup})."
fi

# ------------------------------------------------- o daemon em execucao aplica?
dns_applied=false
grep -q -F "${GATEWAY}" "${RESOLV}" 2>/dev/null && dns_applied=true
trust_applied=true
if [ "${TRUST_ENABLED}" = "true" ] && [ -z "${NODE_EXTRA_CA_CERTS:-}" ]; then
  trust_applied=false
fi

if [ "${dns_applied}" = "true" ] && [ "${trust_applied}" = "true" ]; then
  log "Docker em execucao ja aplica DNS (${GATEWAY})$( [ "${TRUST_ENABLED}" = "true" ] && echo ' e confianca na CA'). Nada a fazer."
  exit 0
fi

if [ "${DOCKER_DNS_RESTART:-true}" != "true" ]; then
  log "AVISO: o Docker em execucao pode nao aplicar ainda DNS/CA (este job pode ter sido"
  log "iniciado antes do restart). Se containers novos nao resolverem/confiarem, rode:"
  log "  sudo systemctl restart docker   (derruba todos os containers do host)"
  exit 0
fi

PROJECT_DIR="${BINDNET_PROJECT_DIR:-}"
ENV_FILE="${BINDNET_ENV_FILE:-.env.main}"
if [ -z "${PROJECT_DIR}" ]; then
  log "AVISO: BINDNET_PROJECT_DIR vazio; nao da para retomar o deploy apos o restart."
  log "Reinicie o Docker manualmente: sudo systemctl restart docker"
  exit 0
fi

# Roda NO HOST (namespaces do PID 1), fora do Docker.
FINISH='
  sleep 3
  systemctl restart docker
  i=0
  until docker info >/dev/null 2>&1; do
    i=$((i + 1))
    if [ "$i" -gt 45 ]; then
      echo "Docker nao voltou em 90s: restaurando o daemon.json anterior" >&2
      b=$(ls -t /etc/docker/daemon.json.bak.* 2>/dev/null | head -1)
      [ -n "$b" ] && cp "$b" /etc/docker/daemon.json
      systemctl restart docker
      exit 1
    fi
    sleep 2
  done
  cd "$1" || exit 1
  DOCKER_DNS_RESTART=false BINDNET_ENV_FILE="$2" \
    docker compose --env-file "$2" up -d
'

if [ "${DOCKER_DNS_DRY_RUN:-false}" = "true" ]; then
  log "DRY RUN: systemd-run --unit=${UNIT} sh -c <finish> sh '${PROJECT_DIR}' '${ENV_FILE}'"
  exit 0
fi

log "Reiniciando o Docker do host para aplicar DNS/CA (os containers vao cair e voltar)."
log "O deploy sera retomado automaticamente: journalctl -u ${UNIT}"
exec nsenter -t 1 -m -u -i -n -p -- \
  systemd-run --unit="${UNIT}" --collect --no-block \
  /bin/sh -c "${FINISH}" sh "${PROJECT_DIR}" "${ENV_FILE}"
