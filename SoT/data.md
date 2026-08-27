# Banco de dados e armazenamento (`postgres`, `mongo`, `minio`, `redis`, `migration`)

- **Postgres é o banco de dados principal** do stack. Hoje guarda a CA
  local e os certificados emitidos (`services/backend/certificates.go`)
  e o mapeamento hostname → IP de loopback do `dns-provider`
  (`services/worker/dns/db.go`); disponível para uso por futuras
  funcionalidades. Tanto `backend` quanto `dns-provider` acessam via
  `database/sql` + driver `pgx` puro — nunca via Prisma Client (que é
  Node-only). `postgres` e `redis` têm **IP fixo** na rede `proxy`
  (`docker-compose.yml`, bloco `networks`) porque `dns-provider` roda
  em `network_mode: host` e não enxerga a DNS interna do Docker para
  resolvê-los pelo nome do serviço.
- O schema do Postgres é criado/atualizado só pelo serviço
  `migration` (`services/migration/`, Node.js + Prisma), que roda
  `prisma migrate deploy` uma vez no boot e encerra (`restart: "no"`).
  O `backend` só sobe depois dele terminar com sucesso
  (`depends_on: migration: condition: service_completed_successfully`
  no `docker-compose.yml`) — nunca cria/altera tabelas sozinho.
- **Mongo guarda a trilha de auditoria do painel** (coleção
  `audit_log`) — eventos discretos de ações administrativas: `login`,
  `logout`, `config_changed` (hotspot/DNS), `certificate_issued`,
  `certificate_revoked`, `hotspot_started`, `hotspot_stopped`,
  `dns_applied`. **Não** persiste logs de containers — isso continua
  sendo só streaming ao vivo via `worker` (`LogsPanel.tsx`,
  `worker.streamLogs`). Falha ao gravar auditoria nunca derruba a ação
  principal do usuário (só loga o erro). Mongo também guarda o **trace
  bruto de consumo do hotspot** (coleção `hotspot_credit_debits`,
  `services/backend/hotspot_credit_trace.go`) — uma linha por ciclo de
  reconciliação (a cada 15s) com o tráfego do dispositivo naquele
  ciclo, gravada pra **todo dispositivo conectado**, com ou sem crédito
  habilitado (só quando habilitado é que o mesmo valor também desconta
  `hotspot_device_credit.balance_bytes`, ver `reconcileDeviceCredit`)
  — é esse trace que alimenta o detalhe de consumo ao clicar numa
  sessão (`GET .../sessions/{id}/consumption`). É o item de maior
  volume da conta corrente, por isso não mora no Postgres: expira
  sozinho via índice TTL em `createdAt`, com a retenção controlada por
  `HOTSPOT_CREDIT_TRACE_RETENTION_DAYS` (padrão 180 dias/6 meses).
- **Postgres consolida o consumo por sessão de conexão**
  (`hotspot_device_sessions`, `services/backend/hotspot_sessions.go`)
  em vez de guardar um trace por ciclo de reconciliação: uma sessão
  nasce quando o MAC aparece na lista de clientes ao vivo do hotspot e
  fecha (`ended_at`) quando some dela (ou quando o hotspot para). A
  cada ciclo de reconciliação, o tráfego real do dispositivo
  (`deltaDown+deltaUp`, o mesmo delta de `recordDeviceUsage`)
  incrementa `total_bytes` da sessão aberta daquele MAC — **sempre**,
  independente de o dispositivo ter crédito habilitado (débito de
  saldo é uma consequência separada, só para quem tem crédito, ver
  `reconcileDeviceCredit`). Isso mantém o total consolidado disponível
  mesmo depois do TTL apagar o trace bruto de débito de crédito no
  Mongo, servindo de conferência posterior.
- **A conta corrente de crédito é uma única lista, toda em Postgres**
  (`GET /api/hotspot/devices/{mac}/credit/history`,
  `services/backend/hotspot_credit_history.go`; aba "Movimentações" no
  detalhe do dispositivo): mescla recarga manual/automática/resgate de
  voucher (`hotspot_device_credit_history`, sem TTL — eventos raros,
  ligados a ação humana ou dinheiro) com **toda sessão** de conexão
  (`hotspot_device_sessions`, ativa ou encerrada, com ou sem consumo
  ainda) como linha de débito `session_active`/`session_closed`. Essa
  lista nunca consulta o Mongo diretamente — só ao clicar numa linha de
  sessão específica é que `GET .../sessions/{id}/consumption` busca lá
  o trace bruto daquela janela de tempo (`started_at`–`ended_at`), que
  pode já ter expirado pelo TTL. A UI permite filtrar por tipo (crédito/
  débito ou o `entryType` específico).
- **MinIO está provisionado mas sem uso ainda** — nenhum endpoint do
  `backend` usa isso hoje; existe só para necessidade futura de
  armazenamento de arquivos.
- **Redis é cache, nunca fonte de verdade** — hoje só o `dns-provider`
  usa (mapeamento hostname → offset de IP de loopback, hidratado do
  Postgres na inicialização; ver seção do `dns-provider` acima).
  Perder o volume do Redis não perde dado nenhum, só custa uma
  rehidratação no próximo boot do serviço que o usa.
- Nenhum dos serviços de dados publica porta no host por padrão —
  ficam só na rede interna `proxy`, mesma postura de exposição mínima
  do resto do stack.

