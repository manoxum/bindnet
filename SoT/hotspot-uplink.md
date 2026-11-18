# Uplink e modo AP do hotspot

- O `bn-uplink` é uma interface virtual estável usada pelo `create_ap`.
  A troca da fonte física atualiza as regras IPv4 e a rota dedicada
  da tabela 91 com o mesmo uplink e o mesmo AP ativos. O `bn-uplink`
  desativa a geração automática de IPv6 link-local, pois o navegador
  no host interpreta a criação ou remoção desse endereço como mudança
  de rede. Uma troca que altere o endereço de saída pode exigir que
  os clientes reconectem as sessões de Internet já abertas.
- **`WIFI_AP_MODE`** (`auto` — padrão — ou `virtual`) decide o que
  acontece quando a interface física **não** está associada como cliente
  Wi-Fi no momento do start:
  - `auto`: o hotspot passa `--no-virt` e usa a própria interface física
    em modo AP (comportamento histórico). Consequência importante: a
    placa inteira vira AP, o `worker` a desgerencia no NetworkManager
    (`unmanageWifiInterfaceIfIdle`) e ela **some do menu de rede do
    sistema** até o hotspot parar — não há como ligar a máquina a uma
    rede Wi-Fi enquanto isso.
  - `virtual`: o AP sobe numa `ap0` mesmo sem associação a preservar, e
    o `worker` **não** desgerencia a placa. O ganho não é preservar
    conexão nenhuma (não há): é a placa física continuar em modo
    `managed` e visível no sistema, permitindo conectar a máquina a uma
    rede Wi-Fi **depois**, com o hotspot já no ar. Continua valendo
    `#channels <= 1`: essa rede tem que estar no mesmo canal do AP.
  - Esse modo só passou a ser viável com o patch de
    `patch-create-ap.sh` (ver item abaixo): o motivo histórico de
    preferir `--no-virt` com a placa ociosa era o `create_ap` morrer com
    `RTNETLINK ... Resource busy` ao trocar a MAC da `ap0` recém-criada.
  - O padrão continua `auto` para não mudar o comportamento de
    instalações existentes sem o operador pedir.
- **A troca de MAC da `ap0` acontece com ela desligada**
  (`patch-create-ap.sh`). O `create_ap` do upstream cria a `ap0`, que
  nasce com a **mesma** MAC da placa física, detecta a duplicata
  (`get_all_macaddrs | grep -c` devolve 2) e gera uma MAC nova — mas
  aplica essa MAC **antes** do `ip link set down` seguinte. O
  `ieee80211_change_mac()` do mac80211 devolve `-EBUSY`
  (`RTNETLINK answers: Resource busy`) para qualquer mudança de MAC numa
  interface que esteja `UP`, e o `create_ap` morre com *"Maybe your WiFi
  adapter does not fully support virtual interfaces"* — derrubando
  **todo** o modo AP+STA (confirmado em `iwlwifi`/AX211, kernel 7.0). A
  `ap0` chega `UP` porque o `create_ap` só a marca como `unmanaged` no
  NetworkManager quando enxerga o NM vivo por D-Bus, e de dentro do
  container isso não responde — o NM do host levanta a interface nova
  sozinho. O patch move a atribuição para depois do `ip link set down`,
  que é onde o próprio `create_ap` já a fazia no caminho `--no-virt`.
  **Não** serve simplesmente pular a troca e deixar a `ap0` com a MAC da
  placa: o mac80211 recusa MAC idêntica entre uma interface `AP` e uma
  `managed` (`identical_mac_addr_allowed()` só libera monitor,
  P2P-device e AP/AP-VLAN), devolvendo `-ENOTUNIQ` no lugar.
- **A placa física é devolvida a `type managed` ao encerrar**
  (`restore_physical_interface_type`, chamada por `cleanup()`, na subida
  do script e antes de cada tentativa em `attempt_hotspot_cycle`). O
  `create_ap` **não** restaura o tipo da interface ao sair de um
  `--no-virt`, e uma placa presa em `type AP` quebra as duas coisas ao
  mesmo tempo:
  - a combinação suportada permite **uma única** interface AP
    (`#{ AP, P2P-client, P2P-GO } <= 1`), então pedir a `ap0` virtual —
    que também é AP — passa a pedir duas, e o firmware recusa com
    `RTNETLINK answers: Resource busy`. Sem restaurar o tipo, o modo
    virtual falha **para sempre**, em toda tentativa;
  - `sta_link_probe` exige `type managed` para reconhecer uma
    associação, então a placa presa em AP **nunca** parece associada e o
    hotspot conclui "sem associação Wi-Fi cliente" mesmo com o usuário
    conectado a uma rede.
  Confirmado ao vivo (`iwlwifi`/AX211). A chamada em `cleanup()` é a
  prevenção; as outras duas são a cura para o estado deixado por um
  container morto por `SIGKILL` antes do `cleanup` rodar — mesma razão
  de existir de `remove_stale_virtual_interfaces`, que faz o
  equivalente para as `apN` órfãs.
- Todo patch de `patch-create-ap.sh` casa linhas **literais** do
  `create_ap` baixado do upstream. O script confere no fim se cada
  âncora foi aplicada e **falha o build da imagem** se alguma sumiu —
  sem isso, uma mudança no upstream faria o hotspot subir com
  comportamento silenciosamente diferente do documentado aqui.
- Com uma estação ativa, o `create_ap`
  **sobrescreve o canal/banda pedido pelo do rádio já associado**
  (a combinação normalmente vem com `#channels <= 1` — os dois modos
  têm que estar na mesma frequência; isso é decisão do próprio
  `create_ap`, não da seleção de canal do hotspot). A imagem
  `services/worker/hotspot/Dockerfile` instala o pacote `grep` (GNU
  grep) especificamente para essa detecção funcionar: o `grep` do
  BusyBox (padrão do Alpine) não processa a regex que o `create_ap`
  usa contra a saída de `iw phy info` (`{` sem bound válido) e sempre
  falha, fazendo o hotspot concluir erroneamente, em qualquer
  adaptador, que o modo concorrente não é suportado.
- **`WIFI_ANCHOR_SSID` / `WIFI_ANCHOR_CHANNEL` — a "rede âncora".** Num
  rádio único o AP e a associação de estação dividem a mesma frequência
  (`#channels <= 1`), então **o canal do AP decide a quais redes a
  máquina consegue se associar** enquanto o hotspot está no ar. A
  seleção automática ranqueia por *menor* interferência, ou seja, escolhe
  o canal onde não há rede nenhuma — o pior possível para esse fim.
  Confirmado ao vivo: com 15 redes visíveis e nenhuma no canal 1, o
  hotspot subiu no canal 1 e a máquina deixou de conseguir se associar a
  qualquer uma (router de casa no 10, outro hotspot no 11).
  - Com âncora configurada, `attempt_hotspot_cycle` sobe o AP no canal
    dela (`anchor_band_channel`, `anchor.sh`) em vez de varrer por
    interferência. Se esse canal falhar, cai na varredura normal.
  - A ordem de prioridade é: `WIFI_CHANNEL` fixo → canal da estação já
    associada → **âncora** → ranking por interferência. A estação vem
    antes da âncora porque ali não é preferência, é física: o rádio já
    está sintonizado naquele canal.
  - `WIFI_ANCHOR_CHANNEL` é o canal **memorizado**, gravado pelo painel
    no momento em que o operador escolhe a rede. É ele que o hotspot usa
    — e não uma varredura feita na subida, que derrubaria o beacon do
    próprio AP. Também é o que faz o hotspot subir no canal certo quando
    a rede âncora está fora do ar (router desligado, máquina noutro
    lugar); routers raramente mudam de canal. Sem canal memorizado, o
    hotspot avisa e cai na seleção automática.
  - O canal memorizado é **reconferido imediatamente antes de cada
    subida** (`refreshAnchorChannel`, `hotspot_anchor.go`, chamada por
    `hotspotConfigWithStartReason` — cobre start manual, autostart e
    auto-recuperação). Sem isso, um router que trocasse de canal deixaria
    a âncora envelhecer em silêncio e o sintoma seria idêntico ao de não
    ter âncora nenhuma, sem nada no log a explicar. Não existe versão
    periódica disso de propósito: com o AP já no ar, mudar de canal
    exigiria derrubar todos os clientes. É best-effort — sem worker, sem
    NetworkManager ou com a rede fora do ar, o canal memorizado continua
    valendo, que é exatamente o fallback desenhado.
  - O seletor do painel (`HotspotAnchorField.tsx`, aba **Rádio**) é
    alimentado por `GET /api/hotspot/wifi-scan` → `GET
    /network/wifi-scan` no worker, que lê o **cache** do NetworkManager
    (`nmcli ... --rescan no`) e **nunca** força varredura. Redes com o
    mesmo SSID (mesh/repetidor) aparecem uma vez, a de sinal mais forte;
    SSID oculto é descartado.
  - **Limitação aceita**: dá acesso a **uma** rede de cada vez. Redes em
    canais diferentes ficam inalcançáveis com o hotspot no ar (na casa
    do usuário, `WIFI2.4` está no canal 10 e `Jorge` no 11 — ancorar
    numa torna a outra inacessível). Não existe seguidor de canal
    automático, por decisão: mover o AP derrubaria todos os clientes do
    hotspot a cada troca.
