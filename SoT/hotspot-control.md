# Ligar/desligar/recuperar o hotspot pelo painel (`POST /api/hotspot/start` / `/stop` / `/recover-wifi`)

Não existem mais scripts de shell separados (`scripts/hotspot-on.sh` /
`hotspot-off.sh`, removidos) — ligar/desligar o hotspot é feito
inteiramente pelo painel de gestão, via `services/backend/hotspot.go`
delegando ao `services/worker/controller` (único serviço com acesso a
NetworkManager/`docker.sock`). O comportamento é o mesmo que os
scripts tinham, só que dentro do container privilegiado em vez de
`sudo` no host:

- `POST /api/hotspot/start`:
  1. Sobe `hotspot` + `dns-provider` via `docker compose up -d
     --no-build --no-deps` usando diretamente
     `docker-compose.services.yml` (`worker`: `POST /hotspot/apply`) —
     cria os containers se ainda não existirem (1ª subida) e também os
     recria se o `.env` mudou, não apenas `docker start`, sem acionar o
     job `migration` nem o `docker-compose.yml` agregador.
  2. O `worker`, bem em cima do `docker exec ... start`
     (`unmanageWifiInterfaceIfIdle`, `services/worker/controller/compose.go`),
     decide quem fica dono de `WIFI_INTERFACE` no NetworkManager:
     - `WIFI_INTERFACE == INTERNET_INTERFACE` (Wi-Fi para Wi-Fi):
       a placa fica gerenciada **incondicionalmente**, sem checar se
       está associada agora — checar seria uma corrida real (pegar a
       placa momentaneamente sem associação durante uma reconexão
       desgerenciava mesmo assim, travando-a desconectada **para
       sempre**, já que só o NetworkManager a reconectaria; confirmado
       ao vivo). Além de não desgerenciar, o worker **re-gerencia**
       (`manageWifiInterface`, idempotente — remove o drop-in e roda
       `managed yes`): sem isso, trocar de um modo que desgerenciou a
       placa (ex.: Ethernet para Wi-Fi com a placa ociosa) para Wi-Fi
       para Wi-Fi deixava o drop-in órfão e a placa presa "unmanaged".
     - Placas **diferentes** com a placa Wi-Fi **associada como
       cliente** (`interfaceAssociated`): a associação é preservada
       (placa continua gerenciada; mesmo `manageWifiInterface`
       idempotente). O hotspot sobe numa `ap0` virtual travada no
       canal da estação — mesma topologia de rádio do Wi-Fi para
       Wi-Fi, que comprovadamente convive com o NetworkManager. É o
       que garante que **partilhar internet do Ethernet não desconecta
       o Wi-Fi cliente do usuário nem o faz "sumir do sistema"**
       (device unmanaged desaparece do menu de rede do desktop).
     - Placas diferentes **sem associação**: desgerencia/desconecta
       como antes — o AP vai subir em `--no-virt` na placa física
       inteira, e deixá-la gerenciada faria o NetworkManager competir
       pelo rádio com o `hostapd` (escaneando/tentando reassociar),
       derrubando o beacon. Corrida residual (associação cair entre a
       checagem e o `create_ap`): o watchdog de beacon é a rede de
       segurança, e o NetworkManager reassociando devolve o caminho
       preservado na tentativa seguinte.
     O container `hotspot` delega ao `create_ap` a criação da
     interface AP virtual (`ap0`) quando há estação a preservar. A
     fonte de internet entregue ao `create_ap` é sempre o uplink
     virtual `BINDNET_UPLINK_INTERFACE`, alimentado por regras Bindnet
     de NAT/forward a partir da interface real configurada.
- `POST /api/hotspot/stop` desfaz exatamente o inverso, na ordem
  inversa:
  0. **Grava a intenção `_DESIRED_STATE=stopped` antes de qualquer
     chamada ao `worker`** (e falha com `500` sem tocar em nada se essa
     gravação não der certo). A ordem importa: `recoverHotspotIfDesired`
     (`hotspot_reconcile.go`, ciclo de 15s) religa qualquer hotspot que
     encontre parado com a intenção ainda em `running`, então gravá-la
     no fim — depois de um teardown que pode abortar — fazia um "parar"
     malsucedido ser desfeito sozinho segundos depois. Era exatamente o
     sintoma "parei e ele voltou".
  1. Para `hotspot` + `dns-provider` (`docker stop`, via `worker`) —
     internamente, o comando `stop` do `entrypoint.sh` sempre passa
     por `force_stop_create_ap` (mesma função usada por `cleanup()` na
     saída normal do serviço), que aplica timeout + escalona pra
     `SIGKILL` (o `create_ap`/`hostapd` diretos e, por varredura final,
     qualquer processo órfão ainda referenciando o diretório de config
     dessa interface) em vez de confiar só num `create_ap --stop`
     "educado" — um `hostapd` preso no loop de falha de beacon (ver
     `watchdog.sh`) ignora sinal de parada limpa e ficaria órfão
     segurando a placa física indefinidamente, mesmo com o `stop`
     reportando sucesso.
  2. Garante que qualquer drop-in antigo do NetworkManager seja removido
     e devolve `WIFI_INTERFACE` ao controle do NetworkManager
     (`worker`: `POST /network/wifi-manage`, que roda `nmcli device set
     ... managed yes`). O container também remove `bn-uplink` e as
     chains `BINDNET-HOTSPOT` ao sair.
  3. Desmonta isolamento de clientes, zonas `wan`/`local` do firewall e
     reforço de DNS.
  - **Os passos 1–3 são best-effort e nenhum deles aborta os
    seguintes** (`stopHotspotAndTeardown` em
    `services/backend/internal/hotspot/hotspot_network.go`): as falhas
    são acumuladas e a resposta é `502` listando todas, mas o teardown
    corre inteiro. Antes, um erro no passo 1 fazia `return` imediato e a
    placa Wi-Fi nunca era devolvida ao NetworkManager — ela sumia do
    menu de rede do sistema até alguém clicar em "Recuperar Wi-Fi". O
    pior estado aceitável é um hotspot que não morreu; nunca uma placa
    sequestrada.
  - **Contexto próprio, não o da requisição**
    (`detachedHotspotContext`, mesmo arquivo): `start`, `stop`, `apply`
    e `recover-wifi` derrubam a rede de quem as pediu — o painel é
    operado também do celular conectado **pelo próprio hotspot**. Com
    `r.Context()`, o Go cancelava o contexto no instante em que o AP
    caía e tudo que viesse depois (gravar a intenção, `wifi-manage`,
    desmontar chains) falhava com `context canceled`.
  - **Timeout do cliente do `worker`**: essas quatro rotas usam
    `workerapi.Client.Slow()` (90s) em vez do cliente padrão (15s). O
    `stop_service` do `entrypoint.sh` espera até 30s pelo `SIGTERM` do
    runner e depois gasta até mais 15s em `force_stop_create_ap`, ou
    seja, estoura os 15s por desenho — o painel reportava
    `Post "http://worker/hotspot/stop": context deadline exceeded
    (Client.Timeout exceeded while awaiting headers)` e tratava como
    "não parou" um hotspot que tinha parado. Precisa ser um cliente
    separado (e não um contexto com prazo maior): `http.Client.Timeout`
    vale junto com o contexto da requisição, então o contexto só
    consegue **encurtar** o prazo, nunca alargar.
- `POST /api/hotspot/recover-wifi` é a ação operacional do botão
  "Recuperar Wi-Fi" na tela "Hotspot Wi-Fi": derruba/para o hotspot em
  execução (mesmo caminho de `POST /api/hotspot/stop`, incluindo o
  `force_stop_create_ap` acima — garante que nenhum `hostapd`/`create_ap`
  fica órfão segurando a placa) e em seguida recupera o controle da
  interface via `wifi-manage`, devolvendo-a ao NetworkManager. É a ação
  segura para destravar a placa quando ela ficou presa como não
  gerenciada (ou com um hotspot zumbi ainda vivo) após queda/restart/
  interrupção fora do fluxo normal.
- `WIFI_INTERFACE` vem da seção `hotspot` do `.env` (via `worker`); se
  não estiver definida, a requisição falha explicitamente (sem
  fallback silencioso para uma interface adivinhada).
- Essas operações mexem em configuração real do host (NetworkManager,
  placa Wi-Fi física, `iptables` e interface dummy `bn-uplink`). O
  fluxo suportado é iniciar/parar pelo painel; "Recuperar Wi-Fi" fica
  como ação segura para limpar estados antigos ou interrupções fora do
  fluxo normal.
- `POST /api/hotspot/start`/`stop`/`recover-wifi` gravam a última
  intenção do admin (chave interna `_DESIRED_STATE` em `hotspot_config`,
  fora da allowlist de `GET`/`PATCH /api/hotspot/config` — ver
  `hotspotDesiredStateKey` em `hotspot_config_state.go`). Ela governa a
  **auto-recuperação de queda**: `recoverHotspotIfDesired`
  (`hotspot_reconcile.go`) só religa um hotspot caído se a última
  intenção foi "ligado". Uma parada deliberada nunca é desfeita.

## Arranque automático do hotspot (`GET`/`PATCH /api/hotspot/autostart`)

- Interruptor **explícito** no painel ("Iniciar automaticamente no
  arranque", aba **Serviço** de "Alterar configuração" —
  `services/frontend/src/components/hotspot/HotspotServiceTab.tsx`),
  gravado na chave interna `_AUTOSTART` de `hotspot_config` — também
  fora da allowlist de `GET`/`PATCH /api/hotspot/config`, e por isso em
  **rota própria**.
- Apesar de ficar dentro do formulário de configuração, o interruptor
  **não** faz parte do "Salvar e aplicar": grava sozinho, na hora, pela
  sua própria rota. Salvar o formulário dispara `POST /hotspot/apply`,
  que reinicia o hotspot e derruba todos os clientes conectados —
  mudar uma preferência de arranque nunca deve custar isso. A aba é
  escondida no assistente de configuração inicial
  (`showServiceTab={false}` em `SetupHotspotStep.tsx`), que só grava
  tudo no último passo.
- **Padrão `false`** (não sobe sozinho). Deliberado: nenhum arranque de
  backend (deploy, `docker compose up`, crash) deve ressuscitar um
  hotspot que o operador parou de propósito sem ele ter pedido isso.
- Ao subir, o backend chama `AutoStartHotspotOnBoot`
  (`hotspot_autostart.go`, goroutine iniciada em `main.go`). Com o
  interruptor **ligado**, ele sobe o hotspot **sempre**, sem consultar
  `_DESIRED_STATE` — "auto-start sim" tem que ser previsível, e um
  hotspot que não volta depois de uma falta de luz só porque estava
  parado naquele instante seria exatamente a surpresa que o interruptor
  existe para eliminar. Grava `_DESIRED_STATE=running` junto, para que a
  auto-recuperação de queda mantenha o hotspot de pé em seguida. Retry
  curto, já que o worker/container do hotspot podem demorar a ficar
  prontos logo após o backend subir.
- O interruptor governa **apenas o arranque**. A auto-recuperação de
  queda (watchdog de beacon → `recoverHotspotIfDesired`) fica sempre
  ligada e continua governada só por `_DESIRED_STATE`: ela existe para
  consertar uma falha de driver, não para desfazer uma decisão do
  operador.

