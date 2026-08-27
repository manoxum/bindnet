# Regras transversais

- Nenhum segredo (senha de Wi-Fi, etc.) deve ir para o Git: apenas
  `.env.example` (com placeholders) é template público. O fluxo
  operacional versiona/usa `.env.main` como arquivo principal do
  `promote.yml/current_stage`; `.env` real continua reservado para
  fallback local/legado e é ignorado (`.gitignore`).
- O entrypoint operacional é `bin/promote` (submódulo `docker-cli`) via
  `Makefile`. A topologia Compose deve ficar dividida em
  `docker-compose.infra.yml`, `docker-compose.services.yml`,
  `docker-compose.assets.build.yml`, `docker-compose.deploy.yml` e o
  agregador `docker-compose.yml`. Portas publicadas devem ficar nos
  compose de infra/services, não em overlays `ports`.
- Todo `entrypoint.sh`/script novo deve seguir `set -euo pipefail` (ou
  equivalente em `sh`), falhar alto (erro explícito + `exit 1`) em vez
  de continuar em estado inconsistente, e logar em português com
  prefixo `[nome-do-servico]`, no padrão já usado pelos scripts
  existentes.
- Serviços que dependem de estado de rede do host (`hotspot`,
  `dns-provider`, `worker`) usam `network_mode: host`; os demais
  (`nginx-ui`, `cert-proxy`, `backend`, `frontend`) ficam isolados na
  rede Docker `proxy`.
