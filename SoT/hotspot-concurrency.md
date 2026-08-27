# Concorrência Wi-Fi do hotspot

- **Ordem STA → AP é a única suportada num rádio único.**
  `warn_ap_sta_channel_lock` (`radio_caps.sh`, chamada na subida) lê o
  `#channels <= N` da combinação que inclui `AP` em `iw phy<N> info` e,
  quando `N == 1`, loga que hotspot e Wi-Fi cliente **têm** que dividir
  o mesmo canal. O hotspot já resolve isso travando o AP no canal da
  estação — mas só nessa direção: com o AP no ar, associar a máquina a
  uma rede Wi-Fi em outro canal não funciona, porque mover o AP depois
  exigiria derrubar todos os clientes. Não existe seguidor de canal
  automático, por decisão: a mesma nota aparece na aba "Interfaces" da
  configuração no painel (`HotspotInterfacesTab.tsx`). O aviso vale para
  qualquer cenário com estação preservada (inclusive Ethernet para
  Wi-Fi), por isso roda fora de `warn_if_concurrent_ap_sta_risky`, que
  só age quando `WIFI_INTERFACE == INTERNET_INTERFACE`.
- **Wi-Fi para Wi-Fi com `WIFI_CHANNEL=auto`** (`WIFI_INTERFACE` ==
  `INTERNET_INTERFACE`, ver `attempt_hotspot_cycle`/`try_create_ap` em
  `entrypoint.sh`): se a placa já está associada como cliente Wi-Fi no
  momento da tentativa, o hotspot **pula a varredura/ranking de canais
  candidatos inteira** (`rank_channels_for_band`, que faz `iw dev ...
  scan`) e trava direto no canal/banda que a associação já está usando
  (`sta_current_band_channel`) — o próprio `create_ap` sobrescreveria
  banda/canal pra bater com a estação de qualquer forma, então escanear
  antes só arrisca derrubar essa mesma conexão cliente (que alimenta o
  hotspot) sem nenhum ganho. Se a placa **não** está associada como
  cliente nesse momento e `WIFI_INTERFACE` é igual a
  `INTERNET_INTERFACE`, o hotspot **falha explicitamente** (retentável,
  mesmo loop de backoff usado para qualquer outra queda) em vez de
  aceitar `--no-virt` como sucesso: sem uma interface virtual `ap0`
  preservando a associação, a mesma placa não pode ser AP puro e
  estação ao mesmo tempo, e um AP sem nenhuma internet por trás
  apareceria como "rodando" no painel enganosamente. Essa checagem não
  se aplica quando `WIFI_INTERFACE` ≠ `INTERNET_INTERFACE` (ex.:
  hotspot pela Wi-Fi com internet vindo de uma interface cabeada) —
  `--no-virt` nesse caso é comportamento correto e esperado.
  **`sta_link_connected`** (`sta_link.sh`), usada por essa checagem em
  `try_create_ap` e por `sta_current_band_channel` (o "trava direto no
  canal" acima também depende dela), lê o **estado vivo do driver**, e
  não a saída de `iw dev ... link`: `iw dev ... station dump` (tabela
  de estações do mac80211 — em modo managed a única entrada possível é
  o AP da associação atual) confirma a associação, e `iw dev ... info`
  (linha `channel`, o chanctx sintonizado) dá banda/canal; `iw dev
  link` fica só como sinal alternativo. Motivo: `iw dev link` responde
  a partir do **cache de resultados de varredura** do cfg80211, e foi
  confirmado ao vivo (2026-07-16, iwlwifi/AX211, cruzando
  `journalctl` do NetworkManager/`wpa_supplicant` com o log do
  container) que ele reporta "Not connected" por **minutos** seguidos
  com a associação real de pé o tempo todo — foi exatamente isso que
  prendia o Wi-Fi-para-Wi-Fi no loop "não foi possível confirmar o
  canal atual da associação" até desistir. Além do sinal confiável,
  `sta_link_connected` ainda tolera reconexões em andamento: até
  `STA_LINK_CHECK_ATTEMPTS` leituras (padrão 20, a cada
  `STA_LINK_CHECK_INTERVAL_SECONDS`, padrão 0.4s — env do container,
  sem equivalente no painel, mesmo padrão de
  `HOTSPOT_BEACON_FAILURE_THRESHOLD` em `watchdog.sh`) antes de
  considerar "não associada" de verdade, porque o custo de esperar
  mais um pouco aqui é bem menor que cair pro loop de retry externo
  (backoff de 3s/6s/9s.../`HOTSPOT_RESTART_BACKOFF_SECONDS` por
  tentativa) só pra repetir a mesma checagem.
  **Essa paciência só se aplica quando `WIFI_INTERFACE` também é a
  fonte de internet** (Wi-Fi para Wi-Fi de verdade) — em qualquer
  outro modo (ex.: Ethernet para Wi-Fi), `sta_link_connected` faz uma
  única leitura rápida e sem espera, porque a placa foi deliberadamente
  desconectada/desgerenciada antes do hotspot subir e nunca vai
  "reconectar" sozinha; aplicar a mesma paciência ali é inútil e
  caro — `rank_channels_for_band`/`start_hotspot_auto` chamam
  `try_create_ap` (que usa essa checagem) uma vez por canal candidato,
  então 8s de espera à toa por candidato somava mais de um minuto de
  atraso na subida do hotspot Ethernet-para-Wi-Fi.
  **Mesmo com essa paciência, se `sta_current_band_channel` ainda
  assim falhar** (associação genuinamente instável por mais tempo que
  isso), Wi-Fi para Wi-Fi **nunca** cai pro caminho antigo de
  varredura ativa de canais (`start_hotspot_auto`/
  `rank_channels_for_band`, que roda `iw dev ... scan`) — confirmado
  ao vivo que esse scan ativo é bem mais perturbador pra uma estação
  associada do que a varredura passiva de fundo do NetworkManager, e
  cair nele só piorava a instabilidade da própria conexão cliente que
  alimenta o hotspot, num círculo vicioso. Em vez disso, retorna falha
  retentável direto e deixa o loop de retry externo (sem nenhum scan
  no meio) esperar a estação estabilizar sozinha — não há perda
  nenhuma, já que o canal do AP é sempre forçado pro canal da estação
  neste modo de qualquer forma, nunca escolhido pelo ranking de
  interferência. Pela mesma razão, `try_create_ap` também nunca sobe o
  AP com banda/canal não confirmados contra a estação: se
  `sta_current_band_channel` falhar bem no meio de `try_create_ap`
  (STA associada segundos atrás, mas o canal não confirma agora), a
  tentativa falha (retentável) em vez de seguir com os valores
  originais (ex.: um candidato de banda/canal vindo de uma tentativa
  anterior) — confirmado ao vivo que isso produz um `hostapd`
  inconsistente (o `create_ap` força o canal certo nos bastidores, mas
  a banda/capacidades declaradas ficam erradas) e clientes reais nunca
  completam a autenticação ("did not acknowledge authentication
  response"), mesmo com o AP aparecendo "ENABLED" no log.
- O painel filtra do seletor de `INTERNET_INTERFACE` (e de
  `WIFI_INTERFACE`) qualquer interface virtual que nunca é uma saída de
  internet real (`docker*`, `br-*` gerada pelo Docker, `veth*`,
  `virbr*`, `tun*`, `tap*`, `wg*`, `bn-*` — uplink dummy Bindnet — e
  `ap0`, a virtual que o próprio `create_ap` cria) — `GET
  /network/interfaces` (worker) já devolve só interfaces
  físicas/relevantes.
- O hotspot exige que o binário `create_ap` baixado suporte
  `--no-dns` e `--dhcp-dns`; sem isso, falha explicitamente em vez de
  criar um AP com comportamento de DNS inesperado.
- DNS entregue via DHCP aos clientes do hotspot é sempre o próprio
  `HOTSPOT_GATEWAY` em primeiro lugar (o `dns-provider` responde por
  trás dele), seguido por `HOTSPOT_DNS_FALLBACKS` (padrão
  `1.1.1.1,8.8.8.8`) para manter navegação externa se o
  `dns-provider` reiniciar ou ainda não estiver escutando. O hotspot
  nunca delega DNS para o `create_ap` (`--no-dns`).
- O DHCP do hotspot também anuncia `domain-search` para os TLDs locais
  (`DNS_SEARCH_DOMAINS`, ou `DNS_LOCAL_TLDS` quando ausente). Isso é
  necessário para clientes como Ubuntu/systemd-resolved rotearem
  `*.local`, `*.test` e `*.example` para o DNS do link Wi-Fi em vez de
  mDNS/outros links. Entradas com vários labels (`local.com`) são
  aceitas e anunciadas como estão; prefixos `*.`/`**.`/`.` são
  descartados, igual à normalização do `dns-provider`.
