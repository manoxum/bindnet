#!/usr/bin/env bash
set -euo pipefail

# gen-secrets.sh gera segredos fortes para producao do stack bindnet.
#
# Segredos cobertos (os "Obrigatorias" do .env.example): ADMIN_PASSWORD,
# POSTGRES_PASSWORD, MONGO_PASSWORD, MINIO_ROOT_PASSWORD, REDIS_PASSWORD.
# (WIFI_PASSWORD nao entra aqui: e configuracao do painel, nao do .env.)
#
# Alfabeto proposital: apenas [A-Za-z0-9]. Estes valores vao para
# docker-compose ${VAR}, para URLs de ligacao (postgres/mongo/redis) e
# para o shell - caracteres como @ : / # $ % & = " partiriam essas
# strings ou exigiriam escaping. Alfanumerico evita isso sem perder
# entropia (40 chars ~ 238 bits; 24 chars ~ 142 bits).
#
# Uso:
#   scripts/gen-secrets.sh                 # imprime KEY=valor no stdout
#   scripts/gen-secrets.sh --out .env.main # gera o arquivo a partir do
#                                           # .env.example (recusa se ja existir)
#
# NUNCA commite o .env/.env.main gerado (ja estao no .gitignore).

log() {
  printf '[gen-secrets] %s\n' "$*" >&2
}

# Segredos e seus tamanhos (chars). ADMIN e menor porque um humano o
# digita no login pelo menos uma vez (pode trocar depois no painel);
# os de servico/banco sao so copiados em variaveis de ambiente.
SECRET_SPECS=(
  "ADMIN_PASSWORD:24"
  "POSTGRES_PASSWORD:40"
  "MONGO_PASSWORD:40"
  "MINIO_ROOT_PASSWORD:40"
  "REDIS_PASSWORD:40"
)

# gen_secret imprime uma string alfanumerica de N chars vinda de um
# gerador criptografico. Prefere openssl (comprimento deterministico,
# sem risco de SIGPIPE); cai para /dev/urandom se openssl faltar.
gen_secret() {
  local len="$1"
  local out=""
  if command -v openssl >/dev/null 2>&1; then
    while [ "${#out}" -lt "${len}" ]; do
      out+="$(openssl rand -base64 48 | LC_ALL=C tr -dc 'A-Za-z0-9')"
    done
  else
    while [ "${#out}" -lt "${len}" ]; do
      out+="$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c "${len}" || true)"
    done
  fi
  printf '%s\n' "${out:0:len}"
}

declare -A VALUES
generate_all() {
  local spec key len
  for spec in "${SECRET_SPECS[@]}"; do
    key="${spec%%:*}"
    len="${spec##*:}"
    VALUES["${key}"]="$(gen_secret "${len}")"
  done
}

print_values() {
  local spec key
  for spec in "${SECRET_SPECS[@]}"; do
    key="${spec%%:*}"
    printf '%s=%s\n' "${key}" "${VALUES[${key}]}"
  done
}

# write_env_file gera o arquivo de ambiente a partir do .env.example,
# substituindo APENAS as linhas dos segredos (o resto do template fica
# igual). Recusa sobrescrever um arquivo existente - segredos em
# producao nao devem ser trocados por acidente.
write_env_file() {
  local out="$1"
  local template
  template="$(dirname "$0")/../.env.example"

  [ -f "${template}" ] || { log "ERRO: template ${template} nao encontrado."; exit 1; }
  if [ -e "${out}" ]; then
    log "ERRO: ${out} ja existe - nao vou sobrescrever. Faca backup e remova, ou rode sem --out para gerar so os valores."
    exit 1
  fi

  local line key
  while IFS= read -r line || [ -n "${line}" ]; do
    key="${line%%=*}"
    if [ "${key}" != "${line}" ] && [ -n "${VALUES[${key}]:-}" ]; then
      printf '%s=%s\n' "${key}" "${VALUES[${key}]}"
    else
      printf '%s\n' "${line}"
    fi
  done < "${template}" > "${out}"

  chmod 600 "${out}"
  log "Arquivo ${out} gerado (permissao 600) com ${#SECRET_SPECS[@]} segredos novos."
  log "Revise os demais valores do template (portas, TZ, ADMIN_USERNAME) antes de subir."
}

main() {
  generate_all
  case "${1:-}" in
    "")
      print_values
      log "Copie estas linhas para o seu .env.main (ou rode com --out <arquivo>)."
      ;;
    --out)
      [ -n "${2:-}" ] || { log "ERRO: --out precisa do caminho do arquivo."; exit 1; }
      write_env_file "$2"
      ;;
    *)
      log "ERRO: argumento desconhecido '${1}'. Uso: $0 [--out <arquivo>]"
      exit 1
      ;;
  esac
}

main "$@"
