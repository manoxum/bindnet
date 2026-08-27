# Serviço `dns-provider` (servidor DNS split-horizon próprio — `services/worker/dns/`)

Não usa mais CoreDNS/Corefile — é um binário Go próprio (`miekg/dns`),
pelo mesmo motivo que a gestão de certificados saiu do cert-proxy: o
CoreDNS/`template` plugin não tem como implementar split-horizon **por
IP de bind** nem alocação persistente de IP por hostname sem plugins
customizados fora da imagem oficial.

Regras:

- `DNS_LOCAL_TLDS` define os TLDs tratados como locais (padrão
  `local,test,example`). Cada entrada pode ser um TLD simples
  (`local`) ou um **sufixo com vários labels tratado como TLD local**
  (`local.com`, `a.b.local`): qualquer nome que termine nesse sufixo,
  em qualquer profundidade (`app.local.com`, `x.y.z.local.com`),
  resolve como zona local. Prefixos `*.`/`**.`/`.` e ponto final são
  descartados na normalização (`**.a.b.local` declara o mesmo sufixo
  que `a.b.local`). Cada label é validado (`[a-z0-9-]`, sem começar ou
  terminar com `-`); entrada inválida é erro fatal. Duplicatas são
  ignoradas silenciosamente. Um sufixo declarado em `DNS_LOCAL_TLDS`
  tem precedência sobre o fallback NXDOMAIN de nomes desconhecidos
  dentro de uma raiz ampla de `DOMAINS`. O painel
  (`PATCH /api/dns/config`) valida `DNS_LOCAL_TLDS` e `DOMAINS` com
  estas mesmas regras e rejeita valores inválidos com `400` antes de
  gravar na tabela `dns_config` — sem isso, um valor inválido salvo pelo
  painel deixava o `dns-provider` em loop de restart e derrubava toda a
  resolução local.
  No painel, cada TLD pode usar o modo **IP local automático** (a
  estratégia split-horizon descrita abaixo) ou **IP estático**: neste
  último, todos os nomes sob o sufixo respondem o mesmo IPv4 em qualquer
  view. Um IP configurado numa zona DNS específica é mais específico e
  tem precedência sobre o IP estático do TLD. Entre TLDs sobrepostos, o
  sufixo mais específico sempre vence, mesmo quando estiver no modo
  automático; somente se ele não corresponder é que um TLD pai pode ser
  considerado.
- `DOMAINS` define as zonas que participam do **discover mode**. Valores
  com um label (ex.: `bnet`, `dev`, `discover`) são raízes amplas da
  malha: nomes dentro delas só resolvem se forem locais via
  `server_name`/rota própria ou aprendidos de outro Bindnet. Valores
  com mais de um label (ex.: `costa.bnet`, `*.costa.bnet`) são zonas
  concretas deste nó: a zona inteira é anunciada como local e
  subdomínios como `app.costa.bnet` resolvem localmente. `DOMAINS`
  vazio desliga o discover mode.
- `server_name` declarados no nginx-ui são tratados como anúncios de
  serviços locais deste nó. O `dns-provider` descobre esses nomes a
  partir do volume `nginx_config` montado somente leitura. Nomes exatos
  (ex.: `app.costa.dev`) anunciam só aquele host; wildcards (ex.:
  `*.costa.dev`) anunciam uma zona inteira. Entradas `_`, regex e
  variáveis do Nginx são ignoradas.
- Quando a consulta é para um nome de `DOMAINS` que este nó possui
  localmente (por zona concreta em `DOMAINS`, por `server_name` ou por
  outra fonte local equivalente),
  a resposta segue a mesma resolução local split-horizon dos TLDs de
  `DNS_LOCAL_TLDS`: host recebe loopback persistente, containers
  recebem o gateway Docker em que a consulta chegou e clientes do
  hotspot recebem `HOTSPOT_GATEWAY`.
- Quando a consulta é para um nome de `DOMAINS` pertencente a outro
  servidor descoberto, o Bindnet funciona como um **roteador de
  descoberta**: o nó local responde encaminhando (proxy real da
  consulta, não uma resposta fixa) para o próximo salto conhecido, não
  necessariamente para o servidor final. Exemplo: se a topologia é
  `A <-> B <-> C <-> D`, com `DOMAINS=a.dev`, `DOMAINS=b.dev`,
  `DOMAINS=c.dev` e `DOMAINS=d.dev`, então:
  - no servidor A, uma consulta para `b.dev` é encaminhada para B,
    porque B é o dono direto desse domínio;
  - no servidor A, consultas para `c.dev` ou `d.dev` também são
    encaminhadas para B, porque B é o próximo salto conhecido para
    alcançar C ou D;
  - no servidor B, `c.dev` segue para C e `d.dev` também segue para C,
    que então entrega a D.
- **Como os nós se conhecem**: não existe mais entrada automática por
  multicast/broadcast. O operador precisa clicar em "Fazer busca" no
  painel para varrer a LAN naquele momento, selecionar o peer
  encontrado e salvar. A busca manual grava apenas candidatos em
  `discover_peers`; a malha efetiva continua sendo exclusivamente os
  peers diretos salvos no Postgres (`discover_configured_peers`).
- Para peers fora da LAN, o operador adiciona manualmente endereços
  `host:porta` do `DISCOVER_PORT`.
- A troca de rotas acontece só com os peers efetivamente presentes em
  `discover_configured_peers`: o `dns-provider` sobe endpoints HTTP
  próprios (`GET /discover/info` e `GET /discover/routes`, porta
  `DISCOVER_PORT`, padrão `8531`) e uma goroutine que consulta cada peer
  a cada 15 segundos, implementando um vetor de distância simples
  (estilo RIP):
  - cada nó anuncia, com distância 0, suas zonas concretas de
    `DOMAINS` e os nomes que ele mesmo possui localmente
    (`server_name` do nginx-ui) que também caem dentro de algum
    `DOMAINS` seu (chamado de "dono": `DISCOVER_NODE_NAME`, padrão o
    hostname do container);
  - ao aprender uma rota de um peer, a distância local é
    `distância_anunciada + 1`; rotas com distância acima de 16 saltos
    são descartadas (limite de segurança contra contagem-ao-infinito);
  - `DISCOVER_REMOTE_ROUTES=auto` permite aprender vizinhos remotos
    anunciados por um peer direto; `DISCOVER_REMOTE_ROUTES=manual`
    aceita apenas rotas próprias do peer direto (`distância anunciada =
    0`), exigindo que vizinhos remotos sejam adicionados explicitamente
    pelo painel;
  - o endpoint de descoberta nunca devolve a um peer uma rota que só
    existe porque foi aprendida dele mesmo (split-horizon, baseado no
    IP de origem da requisição HTTP) e um nó nunca aprende de volta uma
    rota para um domínio que ele mesmo possui;
  - uma rota existente só é substituída se vier do mesmo peer (refresh)
    ou se a nova distância for menor; se um peer parar de responder,
    as rotas aprendidas dele viram `state = "stale"` depois de ~3
    ciclos sem confirmação, mas nunca são apagadas automaticamente —
    ficam visíveis no painel até o peer voltar ou um operador remover
    manualmente.
  - a tabela (domínio anunciado, dono, próximo salto, distância,
    origem, estado, última vez vista) é persistida no Postgres
    (`discover_routes`) só para sobreviver a reinicializações e para o
    painel ler — a resolução de DNS em si nunca consulta o Postgres por
    consulta, só um snapshot em memória atualizado a cada ciclo. O
    painel (`GET /api/dns/routes`) apresenta essa tabela e permite
    remover manualmente uma rota parada (`DELETE
    /api/dns/routes/{domínio}`).
- Nomes dentro de `DOMAINS` que não forem locais nem conhecidos pela
  tabela de descoberta respondem **NXDOMAIN** — nunca são encaminhados
  ao DNS público (evitaria vazar um namespace que é interno da malha)
  nem viram silenciosamente o IP desta máquina.
- **Split-horizon por três "views", uma por IP de bind** — o processo
  abre um socket UDP:53 separado para cada IP abaixo; como cada view é
  um socket próprio, o dns-provider sabe de onde a consulta veio só
  pelo bind que a recebeu, sem precisar inspecionar o IP remoto do
  cliente:
  - `127.0.0.1` (**view host**): consultas do próprio host (o
    `resolved`/resolver do sistema aponta pra cá). Registros `A` para
    os TLDs locais resolvem para um **IP de loopback (127.x.y.z)
    exclusivo daquele hostname**, alocado na primeira consulta e
    persistido no Postgres (tabela `local_dns_records`, coluna
    `loopback_offset` vinda de uma sequence que começa em 2 — o offset
    1/`127.0.0.1` fica reservado) — o mesmo hostname sempre resolve
    para o mesmo IP entre reinícios. Um cache Redis (hash
    `local_dns_records`, hostname → offset) fica na frente do Postgres
  e é **hidratado com todos os registros existentes na
    inicialização do serviço** (não espera cada hostname ser
    consultado de novo para voltar a estar em cache); cache miss vai
    ao Postgres, aloca se necessário, e grava de volta no Redis.
    Cada registro guarda sua origem: **manual**, quando foi cadastrado
    em “Zonas DNS locais”, ou **dinâmica**, quando nasceu de uma
    consulta sob um TLD local (inclusive qual TLD). Ao salvar alterações
    nos TLDs, os dinâmicos são invalidados e o cache é reconstruído;
    os manuais nunca são removidos por essa operação.
  - gateways Docker detectados automaticamente (**view container**):
    consultas vindas de containers Docker (que só alcançam o host por
    esses gateways). Registros `A` para os TLDs locais sempre respondem
    com o próprio gateway Docker em que a consulta chegou.
  - IPs declarados em `HOST_SOURCE_CIDR` (**view peer/LAN**):
    consultas encaminhadas por outros servidores Bindnet chegam pelo IP
    físico/LAN deste nó. Registros `A` para zonas locais respondem com
    esse próprio IP, para que o cliente remoto consiga alcançar o dono
    final do domínio.
  - `HOTSPOT_GATEWAY` (**view hotspot**): consultas vindas de clientes
    conectados ao hotspot Wi-Fi. Registros `A` para os TLDs locais
    sempre respondem com o próprio `HOTSPOT_GATEWAY`.
  - Em qualquer view, registros `AAAA`/`ANY` para os TLDs locais
    sempre respondem `NXDOMAIN` (sem IPv6, sem split-horizon
    multi-tipo) e `SOA` responde um registro sintético mínimo — sem
    isso, resolvers que validam autoridade de zona antes de aceitar a
    resposta (ex.: `systemd-resolved`) tratam a zona inteira como
    quebrada, mesmo a resposta `A` estando correta.
- Para qualquer outro domínio (em qualquer view), a consulta é
  encaminhada para DNS público (`8.8.8.8`, `1.1.1.1`).
- Ao aplicar a configuração pelo painel, o worker mantém uma conexão
  dummy `bindnet-dns` (`bn-dns`) no NetworkManager com `127.0.0.1` como
  servidor e cada entrada de `DNS_LOCAL_TLDS`/`DOMAINS` como domínio
  route-only (`~sufixo`). Isso inclui sufixos com vários labels, como
  `~local.com`, para que o resolver do host envie a consulta ao socket
  da view host em vez do DNS público da ligação default.
- O processo **detecta** os gateways das bridges Docker existentes no
  host e lê `HOST_SOURCE_CIDR` (ou auto-detecta os IPs de LAN quando
  vazio), mas isso nunca bloqueia nem derruba o processo: o servidor
  HTTP de descoberta (`GET /discover/info`/`GET /discover/routes`,
  `DISCOVER_PORT`) e o poll de peers sobem incondicionalmente assim que
  Postgres/Redis/fingerprint estiverem prontos, sem depender de nenhuma
  interface de rede do host. Cada socket DNS UDP:53 que depende de um IP
  específico (gateway Docker, `HOST_SOURCE_CIDR`, `HOTSPOT_GATEWAY`) sobe
  em loop próprio e não-fatal: se o IP ainda não existir (até
  `COREDNS_WAIT_TIMEOUT`, padrão 90s por tentativa), só loga um aviso e
  tenta de novo a cada poucos segundos — nunca derruba o processo nem os
  demais sockets. Só o socket de `127.0.0.1:53` é síncrono/fatal (loopback
  sempre deveria existir; se falhar, é um problema diferente e
  genuinamente fatal). Isso evita que um `HOST_SOURCE_CIDR` desatualizado
  ou um gateway Docker ausente torne o nó inteiro "invisível" para busca
  de peers — antes, essa mesma espera derrubava o processo inteiro
  (`log.Fatalf`) antes do servidor de descoberta sequer subir.
- `dns-provider` não tem (nem deveria ter) `depends_on: hotspot` no
  `docker-compose.services.yml`: ele já sobe/funciona de forma
  independente do container `hotspot` existir ou estar rodando (ver
  espera assíncrona/não-fatal descrita acima) — a única leitura
  relacionada a hotspot é `HOTSPOT_GATEWAY` no Postgres (config
  estática, com fallback), nunca uma chamada ao container `hotspot`
  em si. `depends_on: hotspot: condition: service_started` (sem
  healthcheck) era só ordenação de criação de container no Compose,
  sem efeito funcional real, e foi removido para não sugerir um
  acoplamento que não existe no código.
- `dns-provider` roda com `network_mode: host` (precisa bindar IPs
  reais do host) e por isso **não enxerga a DNS interna do Docker**
  para resolver `postgres`/`redis` pelo nome do serviço — fala com
  eles pelos IPs fixos atribuídos na rede `proxy`
  (`POSTGRES_HOST`/`REDIS_HOST` apontam para esses IPs fixos no
  `docker-compose.services.yml`, não para os nomes dos serviços).
- Para que **qualquer container de qualquer projeto** resolva os TLDs
  locais pela view container, o Docker daemon do host deve usar o
  gateway Docker do Bindnet como DNS upstream. O caminho versionado é
  `sudo scripts/configure-docker-dns.sh`, que detecta esse gateway via
  `docker network inspect` e grava `"dns": ["<gateway>"]` em
  `/etc/docker/daemon.json`; aplicar de fato exige restart explícito do
  Docker pelo operador.

