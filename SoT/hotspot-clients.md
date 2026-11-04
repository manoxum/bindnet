# Clientes do hotspot: identificação e bloqueio por MAC (`services/backend/hotspot_devices.go`)

- `GET /api/hotspot/clients` continua vindo de `create_ap --list-clients`
  (via `worker`), mas agora cada cliente é enriquecido com dados
  cacheados na tabela `hotspot_device_info` (fabricante/tipo/SO
  aproximados) e uma flag `blocked` cruzada com `hotspot_blocked_devices`.
  O enriquecimento nunca é automático a cada poll (a tela reconsulta a
  cada 5s) — só acontece sob demanda.
- `POST /api/hotspot/clients/{mac}/identify` dispara a identificação:
  1. Busca o fingerprint DHCP do dispositivo (opções pedidas + vendor
     class) via `worker`: `GET /hotspot/fingerprint` — lê
     `/var/log/bindnet-dnsmasq-dhcp.log`, caminho fixo que
     `services/worker/hotspot/patch-create-ap.sh` força no
     `dnsmasq.conf` gerado pelo `create_ap` (`log-dhcp` +
     `log-facility`), já que o `CONFDIR` original tem sufixo aleatório
     por execução. **`/var/log` e não `/tmp`**: com
     `fs.protected_regular` ligado no kernel do host (=2 no Ubuntu
     atual), abrir com `O_CREAT` um arquivo pertencente a outro usuário
     dentro de um diretório *sticky* e world-writable (`/tmp`, modo
     1777) devolve `EACCES`. O `dnsmasq` abre este log como root e em
     seguida faz `fchown` para o próprio usuário `dnsmasq` — então a
     primeira execução funcionava e **toda execução seguinte** morria
     com `dnsmasq: cannot open log ...: Permission denied`, derrubando
     o `create_ap` inteiro antes de servir DHCP. O caminho tem que
     continuar igual em `entrypoint.sh` (`DNSMASQ_DHCP_LOG`),
     `patch-create-ap.sh` e `fingerprint.go` (`dnsmasqDHCPLog`).
  2. Descobre o fabricante via `api.macvendors.com` (MAC → nome do
     fabricante); se a chamada externa falhar ou não tiver rede, cai
     para uma base OUI local (`BINDNET_OUI_DB_PATH`, opcional) e por
     fim uma tabela mínima embutida no binário — nunca quebra a tela
     por falta de internet.
  3. Estima tipo de dispositivo/SO por heurística local (substring em
     hostname/fabricante/vendor class DHCP - ver
     `inferHotspotDeviceProfile`) — é uma aproximação com nível de
     confiança (`confidence`), não uma identificação garantida.
  4. Resultado fica em cache (`hotspot_device_info`, por MAC) até o
     operador pedir de novo.
- `PATCH /api/hotspot/devices/{mac}/identity`
  (`services/backend/hotspot_device_identity.go`) grava edição manual
  de `alias`/`vendor`/`deviceName`/`osName` na mesma tabela
  `hotspot_device_info` — campo ausente no corpo do PATCH preserva o
  valor atual (permite editar só o alias na aba de visão geral, ou os
  quatro campos juntos no modal "Identificar" da lista de clientes,
  sem um sobrescrever o outro). Editar manualmente marca
  `confidence = 100` (sinaliza "definido a mão", distinto da heurística
  de `POST .../identify`). `alias` é único (`UNIQUE` em
  `hotspot_device_info.alias`, nulo permitido e não contável para a
  constraint) — conflito devolve 409.
- `GET/POST /api/hotspot/blocklist` e `DELETE
  /api/hotspot/blocklist/{mac}` gerenciam a tabela
  `hotspot_blocked_devices`. Bloquear/desbloquear tem efeito imediato
  via `hostapd_cli deny_acl ADD_MAC`/`DEL_MAC` (+ `deauthenticate` no
  bloqueio) no `worker` (`services/worker/controller/hotspot_acl.go`)
  — não precisa reiniciar o hotspot. Essa ACL só existe na memória do
  `hostapd`: some a cada restart do container, por isso `POST
  /api/hotspot/start` e `POST /api/hotspot/apply` sempre reaplicam a
  blocklist inteira depois de subir o hotspot (com retry curto,
  já que o `hostapd` leva alguns segundos para ficar pronto).
- A página de detalhes do cliente oferece as mesmas ações manuais das
  listas, inclusive para dispositivos offline: **Cortar tráfego** mantém
  a associação Wi-Fi e bloqueia o tráfego (`mode=traffic`); **Bloquear**
  desconecta e impede a reconexão (`mode=deauth`); **Desbloquear** remove
  o bloqueio manual cadastrado. As ações ficam no cabeçalho, disponíveis
  em qualquer aba, com botões acessíveis também no celular. Durante a
  operação ficam desabilitadas para evitar comandos concorrentes no
  mesmo dispositivo; ao concluir, atualizam clientes conectados, lista
  de bloqueados e dispositivos conhecidos. Desbloquear não remove limites
  automáticos por crédito/cota; esses continuam geridos pelas suas abas.

## Perfis de dispositivo, vouchers de recarga e portal cativo (`services/backend/hotspot_profiles*.go`, `hotspot_vouchers.go`, `hotspot_portal.go`)

- **Perfil** (`hotspot_profiles`) é um bundle nomeado e reutilizável de
  limites de tráfego (mesmo shape de `hotspot_device_limits`) + política
  de recarga de crédito (subconjunto de `hotspot_device_credit`: apenas
  `rechargeAmountBytes`/`rechargePeriod`/`plafondBytes` — nunca
  saldo/estado). Existe um perfil "Padrão" com id fixo
  (`00000000-0000-0000-0000-000000000001`, protegido contra remoção),
  vinculado por padrão a todo dispositivo (`hotspot_device_info.profile_id`
  tem esse valor como `DEFAULT`, aplicado inclusive a linhas já
  existentes quando a coluna foi criada).
- **Tipo de limitação (`limitType`) é único e mutuamente exclusivo**,
  tanto em perfil quanto em override de dispositivo (`hotspot_device_limits.limit_type`):
  `unlimited` (nenhum teto), `credit` (precisa de saldo, política em
  `hotspot_device_credit`), `quota` (até 3 tetos simultâneos e
  independentes — diário/semanal/mensal, cada um com seu acumulador em
  `hotspot_device_quota_periods` e bloqueio rígido ao estourar),
  `time` (limitação por tempo, política em `hotspot_device_time` — ver
  abaixo) ou `custom` (só válido em **perfil** — ver abaixo). Taxa
  (download/upload, Mbps) é sempre independente do tipo, configurável
  em todos os casos concretos. Esse tipo único existe para impedir cota
  e crédito ficarem ativos ao mesmo tempo no mesmo perfil/dispositivo
  (causa histórica de bloqueio por crédito disfarçado de "cota parou de
  contabilizar").
- **Limitação por tempo (`limitType = "time"`)** tem dois modos
  (`hotspot_device_time.mode`), espelhando a mecânica do crédito (saldo
  em `hotspot_device_time`, política herdada do perfil enquanto
  `configured = false`, bloqueio via traffic block + portal cativo, os
  mesmos de crédito/cota):
  - `budget`: saldo de **segundos de conexão** (`balance_seconds`)
    debitado enquanto o dispositivo está **associado** ao Wi-Fi, mesmo
    ocioso (o ciclo de amostragem só roda para clientes ao vivo — ver
    `reconcileDeviceTime`). Recarga periódica opcional (`recharge_seconds`
    /`recharge_period`, com `plafond_seconds` de teto) reabastece o
    saldo; o primeiro período é creditado **na hora** em que a política é
    aplicada (`computeNextTimeRechargeAt` ancora em *agora*, não em
    *agora+período*), então "1h por dia" já começa com 1h. Ao zerar,
    bloqueia; **desbloqueia sozinho** quando o saldo volta (recarga
    automática/manual) — diferente do crédito, não exige recarga
    explícita para reabrir. Debito por ciclo tem teto (`maxTimeDebitSeconds`)
    para não drenar o saldo em bloco após um downtime do backend.
  - `deadline`: acesso liberado até `deadline_at`; passado o instante,
    bloqueia (e desbloqueia se o prazo for estendido). Sem consumo de
    saldo.
  A recarga automática de tempo entra no mesmo ciclo de reconciliação da
  de crédito (`applyAutomaticTimeRecharges`). A UI configura tempo em
  **minutos** (convertidos para segundos na API) e deadline via
  `<input type="datetime-local">`; no formulário de **dispositivo** o
  tipo é escolhido mas a política mora na aba de tempo do dispositivo
  (`PATCH /api/hotspot/devices/{mac}/time`), igual ao crédito.
- **Taxa e cota aceitam valor fracionário** na unidade escolhida (ex.:
  cota diária de `1.5GB`, taxa de download de `17.5KB/s`), tanto em
  perfil quanto em override de dispositivo. O formulário aceita `.` ou
  `,` como separador decimal, já que a UI é em português (ver
  `optionalPositiveDecimal` em `hotspot-number-schema.ts`). Cota sempre
  coube no banco (é gravada em bytes, `BIGINT`); taxa era `INTEGER` e
  passou a `double precision` na migration
  `20260716000000_hotspot_rate_decimal` — a fração chega até o `tc`
  como está (`rate 17.5kbps`, ver `rate()` em `shaping_tc.go`). A
  política de crédito (recarga/plafond, em GB) **continua só inteira**.
- **`limitType = "custom"` delega a decisão para o dispositivo**: um
  perfil "customizado" não aplica limite nenhum por si só — o
  dispositivo vinculado a ele é quem escolhe sua própria estratégia
  (`unlimited`/`credit`/`quota`, nunca `custom` — um dispositivo é
  sempre o último nível, não há para quem delegar de novo) via
  `PATCH /api/hotspot/devices/{mac}/limits`. `PATCH` nessa rota é
  **rejeitado com 409** se o perfil vinculado não for `custom` no
  momento da chamada.
- **Ordem de resolução por dispositivo é sempre perfil, a menos que o
  perfil seja `custom`** (nunca override > perfil como um fallback
  genérico — não existe mais um limite global independente de
  perfil/dispositivo, removido por não fazer mais sentido com limite
  sempre por dispositivo ou perfil selecionado):
  - `effectiveDeviceLimits` (`hotspot_profiles_apply.go`): resolve o
    perfil vinculado ao MAC; se `limitType != "custom"`, os valores do
    perfil valem inteiros (um override antigo em `hotspot_device_limits`
    fica dormente/ignorado enquanto isso). Só quando o perfil é
    `custom` é que a linha do dispositivo em `hotspot_device_limits`
    passa a valer — na ausência dela, o padrão é `unlimited`.
  - `syncDeviceCreditFromProfile`: decide se crédito está ativo sempre
    pelo `LimitType` **efetivo** (`effectiveDeviceLimits`, já
    resolvido), nunca pelo `limitType` cru do perfil — um dispositivo
    com override `credit` sob perfil `custom` também conta como
    crédito ativo. A política de recarga (`rechargeAmountBytes`/
    `rechargePeriod`/`plafondBytes`), porém, só é herdada do perfil
    quando é o próprio perfil (não `custom`) que dá origem ao crédito;
    sob perfil `custom` o dispositivo configura sua própria política
    via `PATCH .../credit`. Em qualquer caso, só age quando
    `hotspot_device_credit.configured = false` — vira `true` assim que
    o admin configura crédito manualmente ou o dispositivo resgata um
    voucher (que também força `limitType = "credit"` no override do
    dispositivo, já que resgate é auto-serviço sem acesso à aba de
    limites) — a partir daí o perfil para de influenciar aquele MAC.
  - Editar um perfil (`PATCH /api/hotspot/profiles/{id}`) reaplica ao
    vivo (`applyProfileShapingLive`) em todo dispositivo conectado
    vinculado a ele, sem pular quem tem override próprio — desde que
    `effectiveDeviceLimits` já decide sozinho se esse override importa
    (só quando o perfil é `custom`).
- **Vouchers** (`hotspot_vouchers`) são códigos de recarga
  (`XXXX-XXXX-XXXX`, `crypto/rand`) com valor fixo em bytes, emitidos em
  lote pelo admin (`POST /api/hotspot/vouchers`, até 100 por vez) e
  resgatáveis uma única vez. O resgate (`redeemVoucher`,
  `hotspot_vouchers.go`) é uma transação que reivindica o código
  (`UPDATE ... WHERE status='active'`, atômico via lock de linha do
  Postgres) e credita o saldo exatamente como uma recarga manual —
  ganha um novo `entry_type` (`voucher_redemption`) no extrato
  (`hotspot_device_credit_history`). Código inexistente e código já
  usado devolvem o mesmo erro genérico de propósito (evita virar um
  oráculo para quem tenta códigos ao acaso).
- **Portal de autoatendimento** (`GET/POST /api/hotspot/portal/*`,
  `hotspot_portal.go`) são as únicas rotas do hotspot sem
  `requireSession` — mesmo precedente de `GET /api/mesh/ca`. O MAC do
  dispositivo chamador **nunca** vem do corpo/query: é sempre resolvido
  no servidor a partir do IP de origem (`X-Forwarded-For` ou
  `RemoteAddr`), cruzado contra `GET /hotspot/clients` (mesma função
  `liveHotspotClients` já usada pelo resto do hotspot). Falha em
  resolver o MAC devolve 409 — nunca aceita um MAC alternativo vindo do
  cliente. A página vive na raiz de `bindnet.local.com`, em uma
  superfície React própria e sem `RequireAuth`; a área administrativa
  vive exclusivamente em `admin.bindnet.local.com`. Os bundles são
  separados por hostname, embora usem o mesmo serviço frontend.
- **Redirecionamento automático (portal cativo)** não altera o DNS: a
  sondagem de conectividade que o próprio SO do dispositivo já dispara
  ao entrar numa rede Wi-Fi (HTTP simples, nunca HTTPS) é interceptada
  por uma regra `iptables -t nat -I PREROUTING ... REDIRECT` por MAC
  (`services/worker/controller/captive_portal.go`), avaliada antes do
  `filter/FORWARD` onde vive o DROP de `traffic_block.go` — o
  dispositivo continua associado ao Wi-Fi, só a porta 80 é desviada
  para um responder HTTP mínimo que sempre devolve um redirect para
  `http://bindnet.local.com/`. Só é ativado quando o dispositivo é bloqueado **por falta
  de crédito** (`blocked_by_credit`) — nunca pelo bloqueio manual do
  admin (blocklist modo "traffic"), que continua sem portal cativo de
  propósito. HTTPS não é interceptado (limitação universal de qualquer
  portal cativo).

## Avisos aos dispositivos (`services/backend/internal/hotspot/hotspot_messages*.go`, `hotspot_portal_messages.go`)

- **O que é**: o operador envia mensagens de texto aos dispositivos
  conectados pelo painel (aba **Avisos** da tela de hotspot, ou botão
  "Enviar aviso" na ficha de um dispositivo). Cada aviso é **broadcast**
  (`target_mac` nulo, vai para todos os conectados) ou **direcionado** a
  um único MAC. Guardado na tabela `hotspot_messages`; a leitura por
  dispositivo é registrada em `hotspot_message_reads` (um MAC vê cada
  aviso até tocar "Ok" no portal). Rotas admin (`GET/POST /api/hotspot/
  messages`, `DELETE /api/hotspot/messages/{id}`) exigem sessão; a
  remoção é *soft delete* (`active=false`), preservando a trilha.
- **Entrega base (pull)**: o dispositivo vê o aviso na página pública
  `bindnet.local.com` (`GET /api/hotspot/portal/messages`), identificado pelo MAC
  de origem exatamente como o resto do portal (`resolvePortalMAC`, nunca
  um MAC vindo do cliente). Não expirado (`expires_at`) e não lido conta;
  `POST /api/hotspot/portal/messages/{id}/read` marca lido.
- **Entrega urgente (push)**: um aviso marcado **urgente** também dispara
  o balão "Entrar na rede" do próprio SO **reusando o portal cativo por
  MAC** já descrito acima (`applyCaptivePortalRedirect`) — **sem nenhuma
  alteração de DNS nem interceptação L7**. Broadcast urgente liga o
  redirect para todos os MACs conectados no momento; direcionado, só para
  aquele MAC. É *best-effort* (só porta 80/HTTP, só loga em falha) — a
  entrega garantida continua sendo `bindnet.local.com`.
- **Coordenação com o bloqueio de crédito/cota**: o redirect de portal
  cativo é "possuído" originalmente pelos reconciles de crédito/cota. O
  push de aviso é **aditivo**: ao desligar (aviso lido/removido/expirado)
  só remove o redirect se o MAC não estiver retido por bloqueio de
  crédito (`blocked_by_credit`) ou cota (`hotspot_device_quota_periods.
  blocked`). Edge conhecido e aceito: um reconcile de crédito/cota logo
  em seguida pode desligar o push de um aviso urgente — mas o aviso
  continua entregue por `bindnet.local.com`.

