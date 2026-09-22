# Painel de gestão (`services/frontend` + `services/backend` + `services/worker/controller`)

Regras:

- **`worker`** é o único serviço com acesso privilegiado ao host
  (`/var/run/docker.sock`, `/run/dbus`, `/etc/NetworkManager/conf.d`,
  `/usr/local/share/ca-certificates`, `/etc/ssl/certs`,
  `network_mode: host`). Sua API interna (servida só via socket Unix
  em `worker_ipc`, sem porta TCP) opera com uma lista fechada de
  serviços permitidos (`hotspot`, `dns-provider`, `nginx-ui`,
  `postgres`, `mongo`, `minio`) e ações específicas via
  `docker compose --project-name bindnet` — nunca executa comando
  arbitrário vindo do backend. O `worker` não depende de `hotspot` nem
  `dns-provider` no Compose, para que o painel suba sem acionar a rede
  física; esses serviços só sobem quando o usuário aplica/inicia o
  hotspot pelo painel. `migration` (job one-shot) propositalmente não
  está nessa lista.
- No bootstrap, o `worker` grava de forma idempotente um drop-in que
  marca `docker0`, bridges `br-*` e interfaces `veth*` como não
  gerenciadas pelo NetworkManager. Isso preserva todas as redes Docker
  e evita que ciclos de containers pareçam alterações de rede do host;
  a regra temporária do hotspot continua separada.
- O painel **não edita mais o `.env`**. Toda configuração operacional
  vive no Postgres, em tabelas chave/valor: `hotspot_config`
  (`WIFI_*`, `HOTSPOT_GATEWAY`, `HOTSPOT_CIDR`, …), `dns_config`
  (`DNS_LOCAL_TLDS`, `DOMAINS`, `DISCOVER_NODE_NAME`,
  `DISCOVER_REMOTE_ROUTES`) e `panel_config` (`CA_COMMON_NAME`,
  `NGINX_UI_USERNAME`, `NGINX_UI_PASSWORD`). Cada serviço lê a sua
  tabela direto do banco quando inicia — ninguém reescreve arquivo de
  ambiente. `DISCOVER_PORT` continua vindo do `.env`/compose por ser
  porta de infraestrutura, não configuração de negócio.
- Na primeira subida após essa migração, `backend` e `dns-provider`
  fazem uma **importação única** do que ainda estiver no ambiente para
  as tabelas (`ON CONFLICT DO NOTHING`), para uma instalação existente
  não cair nos defaults. Depois disso o painel é a fonte de verdade e as
  variáveis podem sair do `.env`.
- "Salvar" configuração (grava no banco) e "aplicar" (recria o
  container via `docker compose up -d --no-build`) são ações
  separadas — o painel nunca reinicia o hotspot/DNS sozinho ao salvar,
  só quando o usuário confirma explicitamente "aplicar". Peers diretos
  da malha Bindnet seguem a mesma ideia, no Postgres
  (`discover_configured_peers`).
- **`backend`** nunca monta `docker.sock` nem roda em
  `network_mode: host`; toda operação privilegiada (controlar
  containers, NetworkManager, listar interfaces/clientes, testar DNS)
  é delegada ao `worker`. Exceção: `backend` pode montar volumes de
  dados somente leitura (ex.: `cert_proxy_data:/certproxy-data:ro`)
  para ler arquivos diretamente, já que isso não é controle de
  host/Docker.
- Login é obrigatório: `backend` guarda um único usuário administrador
  em `/data/admin.json` (senha derivada por HMAC-SHA256 iterado +
  segredo de sessão). No primeiro boot, `ADMIN_USERNAME` e
  `ADMIN_PASSWORD` criam esse arquivo; em boots seguintes, se ambas
  estiverem definidas, o backend sincroniza o usuário persistido com o
  `.env` e troca o segredo de sessão quando as credenciais mudarem.
  Se ambas ficarem vazias e o arquivo já existir, o administrador
  persistido continua valendo.
- Go em `services/worker/controller` segue a convenção original: sem
  dependências externas no `go.mod` (inclusive autenticação usa
  HMAC-SHA256 da biblioteca padrão em vez de bcrypt/JWT de terceiros).
  `services/backend` é a exceção deliberada: tem `github.com/jackc/pgx/v5`
  (Postgres) e `go.mongodb.org/mongo-driver/v2` (Mongo) como
  dependências, porque não há alternativa razoável de biblioteca
  padrão para os protocolos de rede desses bancos — diferente do caso
  HMAC-SHA256, que a stdlib já resolve bem.
