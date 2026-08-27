# Regras de negócio — bindnet

Este documento descreve o comportamento funcional esperado de cada
serviço do stack `bindnet`. É a referência de "o que o sistema deve
fazer"; para "como rodar" veja o [README.md](../README.md).

## Visão geral do domínio

O `bindnet` provê, para uma rede local doméstica/pequeno escritório:

1. Um ponto de acesso Wi-Fi (hotspot) compartilhando a internet de uma
   interface cabeada/outra Wi-Fi.
2. Resolução DNS *split-horizon*, com resposta diferente conforme
   quem pergunta: domínios internos (TLDs locais, ex. `.local`)
   resolvem para um IP de loopback próprio quando a consulta vem do
   próprio host, para o gateway Docker quando vem de um container, ou
   para o gateway do hotspot quando vem de um cliente Wi-Fi; todo o
   resto é encaminhado para DNS público.
3. Uma CA local própria, com emissão explícita de certificados TLS sob
   demanda (via painel de gestão), para permitir HTTPS sem avisos de
   certificado durante desenvolvimento. Não há mais proxy público nas
   portas 80/443 — a emissão é sempre uma ação explícita do usuário.
4. Uma UI (`nginx-ui`) para administrar sites, acessível só pela porta
   administrativa `9080`.
5. Um painel web (`services/frontend` + `services/backend` +
   `services/worker`) para gerenciar hotspot/DNS e emitir/gerenciar
   certificados TLS, sem depender de shell/`.env` manual.
6. Um banco de dados PostgreSQL como armazenamento principal do
   stack, um MongoDB para trilha de auditoria, um Redis como cache do
   `dns-provider`, um MinIO provisionado para uso futuro, e um job
   Node.js/Prisma (`services/migration`) que aplica o schema do
   Postgres antes de `backend`/`dns-provider` subirem.
