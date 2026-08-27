# Isolamento de clientes (`services/backend/internal/hotspot/hotspot_isolation*.go`, `services/worker/controller/internal/shaping/isolation*.go`)

- **Interruptor geral**: chave `CLIENT_ISOLATION` em `hotspot_config`
  (default `false`), editável pela aba Isolamento do painel
  (`PUT /api/hotspot/isolation`). Ligado, o `entrypoint.sh` sobe o
  create_ap com `--isolate-clients` (`ap_isolate` do hostapd): o AP
  para de retransmitir frames entre as próprias estações em L2.
  **Mudar o interruptor só vale após reiniciar o hotspot** — o
  `ap_isolate` é fixado no `hostapd.conf` no start; o painel avisa e
  nunca reinicia sozinho. Se o create_ap baixado não suportar a flag
  com o isolamento ligado, o entrypoint falha alto em vez de subir um
  AP silenciosamente sem isolamento.
- **Mecânica**: com o L2 direto cortado, o worker liga
  `proxy_arp_pvlan=1` (RFC 3069) e `send_redirects=0` na interface AP —
  o host responde ARP em nome dos outros clientes e o tráfego
  cliente↔cliente passa a ser roteado em hairpin
  (cliente→host→cliente), atravessando `filter/FORWARD` onde o chain
  `BINDNET-ISOLATION` (jump `-i <ap> -o <ap>`) decide:
  `RELATED,ESTABLISHED` no topo, um ACCEPT por par permitido
  (`--mac-source` origem + IP destino, comentários `bn-iso-pair-*`) e
  DROP no fim — **default deny**. Tráfego cliente→internet
  (`-o bn-uplink`) e cliente→gateway (painel/DNS/portal, INPUT) nunca
  passam por esse chain e não são afetados.
- **Política** (motor puro em `hotspot_isolation_policy.go`): a
  comunicação de X para Y é decidida pela regra mais **específica** que
  casar — especificidade = soma das extremidades (dispositivo=2,
  perfil=1, qualquer=0); empate na mesma especificidade → **bloquear
  vence**; nenhuma regra → bloqueado. Regras (`hotspot_comm_rules`) têm
  origem (dispositivo|perfil), destino (dispositivo|perfil|qualquer),
  sentido e ação (permitir|bloquear).
- **Firewall por zonas / camada L4** (base em `hotspot_comm_rules`,
  colunas `zone`/`protocol`/`dst_ports`/`dst_host`, migration
  `20260721010000_hotspot_firewall_l4`): cada regra tem uma **zona** —
  `clients` (cliente↔cliente, chain BINDNET-ISOLATION), `wan`
  (cliente→internet, chain BINDNET-FW-WAN) e `local` (cliente→gateway,
  chain BINDNET-FW-LOCAL). Uma regra pode restringir por **protocolo**
  (`any`/`tcp`/`udp`/`icmp`) e **portas de destino** (`dst_ports`, lista
  `80,443,8000-8100`, só com tcp/udp). Na zona clients, o motor emite,
  por par ordenado (MAC origem→IP destino), as entradas **na ordem em
  que o firewall avalia**: ponta mais específica primeiro, depois L4
  mais específico (protocolo/porta concretos acima de `any`), e
  bloquear antes de permitir em empate. O worker reconstrói o chain só
  quando essa assinatura ordenada muda (idempotente por comparação de
  comentários), instala cada entrada como ACCEPT/DROP com
  `-p <proto> -m multiport --dports <portas>` e mantém o DROP final
  (default deny). Assim "permitir TCP/443 entre A e B" libera só 443 e o
  resto cai no DROP; "bloquear UDP + permitir tudo" bloqueia só UDP.
  Regras retrocompatíveis: sem zona = `clients`, sem protocolo = `any`.
- **Zonas wan e local** (`hotspot_firewall_policy.go`,
  `services/worker/controller/internal/shaping/firewall_{zones,wan,local}.go`):
  ao contrário da zona clients, **não** dependem de `ap_isolate` nem de
  reiniciar o hotspot — são puro iptables aplicado ao vivo enquanto o
  hotspot roda (reconciliação, reaplicação pós-start e teardown no stop,
  ver `hotspot_firewall_apply.go`). Cada regra casa pela **origem**
  (dispositivo|perfil|todos os clientes) + L4; a zona `wan` aceita ainda
  um `dst_host` (IP/CIDR externo). Cada zona tem uma **política padrão**
  (`FW_WAN_POLICY`/`FW_LOCAL_POLICY` em `hotspot_config`, default
  `allow`) para o tráfego que nenhuma regra cobre — default `allow` de
  propósito: adicionar o firewall não muda nada até haver regras, e nunca
  corta internet/painel sozinho.
  - **wan** (`BINDNET-FW-WAN`, jump em FORWARD para `-i <ap> -o
    <uplink>`): usa **RETURN** para "permitir" (o pacote segue para
    `BINDNET-HOTSPOT`, onde ainda passam o bloqueio por crédito e o
    MASQUERADE — permitir na wan **não** fura o crédito) e **DROP** para
    "bloquear". O jump é mantido **acima** do de `BINDNET-HOTSPOT` no
    FORWARD (senão o ACCEPT genérico do hotspot liberaria antes das
    regras de bloqueio); a correção de ordem acontece no ciclo de
    reconciliação (janela ≤15s após uma troca de uplink).
  - **local** (`BINDNET-FW-LOCAL`, jump em INPUT para `-i <ap>`): usa
    **ACCEPT** para "permitir" e **DROP** para "bloquear". No topo há
    permissões **fixas e inegociáveis** (established, DHCP, DNS, painel
    HTTP/HTTPS, ICMP) para o operador nunca se trancar fora do
    painel/rede, mesmo com política `deny`.
- **Modalidades no painel** (aba Isolamento): uma regra tem dois
  escopos na UI. **Dentro de um perfil** = comunicação entre os
  clientes do mesmo perfil; é gravada como uma regra normal com origem
  **e** destino iguais ao mesmo perfil (perfil↔próprio-perfil,
  especificidade 2) — não há mais um flag separado por perfil, é uma
  regra como as outras. **Entre origem e destino** = origem e destino
  distintos (dispositivo|perfil, destino também "todos os clientes").
  Assim: dispositivo↔qualquer bloquear empata com a regra interna do
  perfil e vence (cliente totalmente isolado); dispositivo↔dispositivo
  permitir (especificidade 4) vence tudo (exceção pontual). A coluna
  `hotspot_profiles.allow_internal_communication` continua no banco e o
  motor ainda a respeita como allow implícito perfil↔próprio-perfil,
  mas o painel não a define mais (fica sempre `false`) — a comunicação
  interna é expressa por regra mesmo-perfil.
- **Sentido** (`direction`): `to` = a origem pode **iniciar** tráfego
  para o destino; as respostas voltam pelo conntrack
  (`RELATED,ESTABLISHED`), senão TCP não funcionaria. `both` = os dois
  podem iniciar. O caso "só destino→origem" da UI é gravado com as
  extremidades trocadas — o banco só conhece `to`/`both`.
- **Aplicação**: o backend compila o estado desejado completo (pares
  MAC origem→IP destino dos clientes conectados agora) e o worker
  materializa idempotente (`POST /hotspot/isolation/apply`), sem
  estado local — mesmo modelo do shaping. Reaplicado a cada ciclo de
  reconciliação (~15s: cobre cliente novo, renovação de DHCP e regra
  perdida por reinício de container), em toda mutação de
  regra/perfil/vínculo, e com retry pós-start
  (`reapplyHotspotIsolation`); o stop do hotspot desmonta
  chain/sysctls. Janela de até um ciclo para pares novos valerem é
  aceite por desenho.
- **Limitações aceites**: com o isolamento ligado, broadcast/multicast
  entre clientes (mDNS/Chromecast/descoberta de impressora) não
  funciona nem entre pares permitidos — só unicast roteado; IPv6
  link-local entre clientes fica bloqueado por desenho (`ap_isolate`
  corta o L2 e não há proxy NDP). Regras de perfil apagado somem junto
  com o perfil (mesma transação de `DeleteProfile`); os dispositivos
  dele voltam ao perfil Padrão.

## Planos de bloqueio/permissão de conteúdo (`services/backend/internal/hotspot/hotspot_content*.go`, `services/worker/dns/internal/blocklist`, `services/worker/controller/internal/shaping/dns_enforce.go`)

Controle de conteúdo (parental control / firewall de serviços) na aba
**Conteúdo** do painel. Um **plano** (`hotspot_content_plans`) é um
conjunto nomeado de **regras** (`hotspot_content_rules`) vinculável a um
**perfil** ou a um **dispositivo** (colunas `content_plan_id`,
precedência device→perfil, igual aos limites — ver
`effectiveContentPlanID`).

- **Regras** têm `kind` `domain` (ex.: `exemplo.com`, cobre subdomínios
  por sufixo), `ip` (IP/CIDR) ou `category` (um slug de
  `hotspot_content_categories`), e `action` `allow`/`block`. A política
  padrão do plano (`default_action`) decide o não coberto: `allow` =
  lista negra (bloqueia só o `block`), `block` = lista branca. **`allow`
  vence `block`** no casamento.
- **Categorias** (`hotspot_content_categories` + `_category_domains`) são
  populadas de **blocklists públicas** (The Block List Project +
  StevenBlack, formato hosts) por um job de sync no backend
  (`StartContentBlocklistSyncLoop`, 1x/dia + gatilho manual), ou são
  **embutidas** (sem `source_urls`, domínios semeados pela migration —
  ex.: `app_stores`, `doh_bypass`). Categorias semeadas: `ads`, `adult`,
  `gambling`, `social`, `app_stores`, `doh_bypass`, `torrent`, `piracy`,
  `cryptomining`, `drugs`, `scam`, `malware`, `fakenews`. Evita cadastrar
  site por site (ex.: bloquear `+18`/ads/lojas de apps/torrent). Algumas
  listas são enormes (ex.: `malware` ~2,6M domínios).
- **Bloqueio de domínio é feito no `dns-provider`** (pacote `blocklist`):
  o backend publica o mapa IP→plano dos clientes ao vivo em
  `hotspot_content_client_bindings` (canal Postgres, pois o backend não
  fala Redis); o `dns-provider` carrega planos/regras + os domínios
  **apenas das categorias referenciadas por algum plano** (não puxa
  listas gigantes que ninguém usa para a memória) + esse mapa, em memória
  por poll (só recarrega quando a assinatura de conteúdo muda) e, antes
  de encaminhar um domínio público, resolve o plano pela origem e
  responde `NXDOMAIN` (sinkhole) se bloqueado (casamento por sufixo:
  bloquear `x.com` cobre `*.x.com`). Sem plano vinculado ao IP, nada
  muda.
- **Bloqueio de IP/CIDR** (`kind=ip`) entra na **zona WAN** do firewall
  (reusa `compileZoneRules`/`BINDNET-FW-WAN`), casado por MAC do
  cliente, na frente das regras de comunicação.
- **Reforço de DNS** (decisão de projeto: bloqueio efetivo, difícil de
  burlar) no worker (`dns_enforce.go`), ligado **só quando existe algum
  plano vinculado** (senão o caminho de rede dos clientes não muda):
  redireciona todo `:53` dos clientes para o gateway (quem troca de DNS
  a mão não escapa) e bloqueia DNS criptografado conhecido — DoT (853) e
  uma lista curada de IPs de provedores DoH em 443, **tanto TCP quanto
  UDP/HTTP3** (só TCP deixava passar DoH sobre QUIC). É idempotente,
  reaplicado no ciclo de reconciliação e desmontado no stop do hotspot.
  Limitação: DoH para um IP fora da lista curada ainda escapa (inerente
  a bloqueio DNS); o caso comum (celular/browser padrão) é coberto.
- Toda mutação de plano/regra/vínculo reaplica ao vivo
  (`applyContentLive`: republica o mapa + reaplica firewall + reforço de
  DNS). O bloqueio de domínio depende do cliente usar o resolver local
  (garantido pelo reforço de DNS acima).

