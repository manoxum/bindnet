# Serviço `hotspot` (`services/worker/hotspot/entrypoint.sh`)

Regras:

- Variáveis obrigatórias: `WIFI_INTERFACE`, `INTERNET_INTERFACE`,
  `WIFI_SSID`. Ausência de qualquer uma delas é erro fatal (o container
  não sobe). `WIFI_PASSWORD` é obrigatória só quando `WIFI_OPEN` não é
  `true` (ver item abaixo).
- **`WIFI_OPEN`** (padrão `false`) cria um hotspot livre, sem
  autenticação nenhuma, quando `true`. O `create_ap` baixado
  (`oblique/create_ap`) só escreve as diretivas `wpa_*` no
  `hostapd.conf` quando recebe uma passphrase como último argumento
  posicional; sem ela, o AP sobe aberto. Por isso o hotspot **omite o
  argumento inteiro** (não passa uma string vazia) quando `WIFI_OPEN=true`,
  em vez de reaproveitar `WIFI_PASSWORD` de uma configuração anterior
  com senha. `WIFI_PASSWORD` continua podendo ficar salva no painel
  nesse estado (não é apagada ao ligar o modo livre) - só deixa de ser
  obrigatória e de ser repassada ao `create_ap`, para que voltar
  `WIFI_OPEN` para `false` não exija digitar a senha de novo. A UI
  (`services/frontend/src/components/hotspot/HotspotWifiTab.tsx`)
  desabilita o campo de senha enquanto o modo livre está marcado, e o QR
  de conexão (`HotspotWifiQr.tsx`) usa o formato `WIFI:T:nopass;...`
  (sem campo de senha) em vez de `WIFI:T:WPA;...`.
- `WIFI_CHANNE` (sem "L") é aceito como alias legado de
  `WIFI_CHANNEL`, com aviso de depreciação no log. Não usar em
  configurações novas.
- **`ensure_wifi_radio_unblocked`** (`regulatory.sh`) roda uma vez no
  início, antes de qualquer diagnóstico regulatório ou tentativa de
  canal: confere via `rfkill list wifi` se o rádio Wi-Fi está
  bloqueado (soft ou hard) e, se estiver, tenta `rfkill unblock wifi`
  automaticamente. Um rádio bloqueado rejeita **todos** os
  canais/bandas igualmente com erros genéricos do driver (ex.:
  `RTNETLINK answers: No error information`), indistinguível de fora
  de uma trava regulatória ou de canal específico — sem essa checagem,
  o hotspot esgotaria todos os candidatos de 2.4GHz e 5GHz (e as 5
  tentativas do loop de retry) sem nunca revelar a causa real. Um
  `docker compose up --build` que recria o container `hotspot` com o
  `create_ap` ativo (interrompendo a limpeza normal no meio — ver
  `stop_grace_period` do serviço `hotspot` no
  `docker-compose.services.yml`, aumentado pra 30s de propósito, acima
  do pior caso de `force_stop_create_ap`) já deixou o driver `iwlwifi`
  reportando bloqueio "hard" via `rfkill` numa sessão real, e um
  `rfkill unblock` comum (sem privilégio de root) resolveu de verdade
  — confirma que não era um interruptor físico genuíno (esse
  continuaria bloqueado logo em seguida). Só loga e tenta desbloquear;
  nunca falha o script por si só, já que o `create_ap` reporta o erro
  de qualquer forma se o desbloqueio não funcionar (ex.: bloqueio
  físico genuíno de verdade, que nenhum comando resolve).
- **`WIFI_CHANNEL`** aceita:
  - um número de canal fixo (validado como inteiro) — usado como está,
    sem retry nem fallback de banda: se `create_ap` rejeitar esse canal
    específico, o hotspot falha (respeita a escolha explícita do
    usuário); ou
  - `auto` (padrão): o hotspot escaneia o ambiente Wi-Fi
    (`iw dev ... scan`) e ordena os canais candidatos da banda
    resolvida do **menos para o mais interferido**.
    - Em 2.4GHz a pontuação penaliza canais sobrepostos (distância 0 a
      4 do canal observado); em 5GHz os canais são considerados
      ortogonais (pontuação binária, mesmo canal = interferência).
    - Se a varredura falhar ou não retornar redes, a ordem dos
      candidatos é arbitrária (com aviso no log) — o hotspot nunca
      trava por falta de varredura.
    - **Retry por canal**: o hotspot tenta `create_ap` em cada canal
      candidato, na ordem de menor interferência, até um que o
      adaptador realmente aceite — a pontuação de interferência é só
      um critério de desempate/ordenação, não uma garantia de que o
      canal é utilizável (o adaptador pode reportar suporte a uma
      banda via `iw phy info` e ainda assim rejeitar canais específicos
      dela, ex.: `create_ap` retornando "adapter can not transmit to
      channel X"). **Exceção**: se o `create_ap` recusar com "adapter
      can not be a station ... and an AP at the same time" (
      `WIFI_INTERFACE` associado como cliente Wi-Fi num adaptador que
      não suporta AP+estação simultâneos — ver item abaixo), o hotspot
      **não** varre os demais canais nem tenta a banda alternativa:
      trocar canal/banda nunca resolve essa causa especificamente, só
      adiaria o mesmo erro por mais tentativas inúteis.
    - **Fallback de banda**: se `WIFI_CHANNEL` **e** `WIFI_FREQ_BAND`
      estiverem ambos em `auto` e nenhum canal da banda escolhida
      funcionar, o hotspot tenta automaticamente a outra banda (todos
      os seus canais candidatos, também em ordem de interferência)
      antes de desistir. Se `WIFI_FREQ_BAND` foi fixado explicitamente
      (`2.4` ou `5`), não há fallback de banda — só retry de canal
      dentro da banda escolhida.
  - Candidatos podem ser sobrescritos via `WIFI_CHANNEL_CANDIDATES`
    (lista separada por vírgula/espaço); caso contrário os candidatos
    padrão são `1 6 11` (2.4GHz) ou `36 40 44 48 149 153 157 161`
    (5GHz).
  - **Histórico persistente** (`hotspot_channel_history` no Postgres,
    `services/worker/hotspot/history.sh`) influencia a ordem dos
    candidatos muito mais que a pontuação de interferência ao vivo:
    cada tentativa de `create_ap` num banda/canal grava sucesso (o log
    mostrou `AP-ENABLED`, mesmo que o AP caia depois por outro motivo)
    ou falha, por combinação (`WIFI_INTERFACE`, modo, banda, canal).
    `rank_channels_for_band` soma `(falhas − sucessos) × 1000` à
    pontuação de interferência de cada candidato antes de ordenar —
    um canal que já falhou consistentemente (ex.: "adapter can not
    transmit") vai sempre pro fim da fila nas próximas subidas, mesmo
    que a varredura ao vivo o mostre como o menos congestionado agora;
    interferência só desempata candidatos com o mesmo histórico
    (incluindo nenhum ainda). "Modo" distingue `same-interface`
    (Wi-Fi para Wi-Fi — histórico é só diagnóstico ali, já que o canal
    é sempre travado pelo canal da estação, sem escolha real) de
    `different-interface` (Ethernet para Wi-Fi/auto — histórico
    realmente influencia a ordem). `resolve_wifi_band` usa a mesma
    lógica num nível acima: se uma banda inteira tem histórico de
    sucesso melhor que a banda preferida por capacidade de hardware
    (5GHz), a automática escolhe a banda historicamente melhor direto,
    evitando reaprender do zero (testando todos os candidatos de 5GHz
    de novo) uma trava regulatória de firmware já conhecida. Histórico
    nunca é uma dependência funcional — qualquer falha ao ler/gravar
    no Postgres é ignorada em silêncio (só perde a otimização de
    velocidade, não trava o hotspot).
- **`WIFI_FREQ_BAND`** aceita `2.4`, `5` ou `auto` (padrão). Resolvida
  **antes** da seleção de canal, porque a lista de candidatos e a
  pontuação de interferência dependem da banda:
  - Se `WIFI_CHANNEL` já for um número fixo, a banda é **inferida do
    canal** (1–14 → 2.4GHz; caso contrário → 5GHz) — não faz sentido
    escanear banda quando o canal já decide isso.
  - Caso contrário, o hotspot inspeciona as capacidades da placa via
    `iw phy<N> info` e **prefere 5GHz quando suportado** (menos
    congestionamento, mais throughput); cai para 2.4GHz se a placa não
    suportar 5GHz ou se a detecção falhar (com aviso no log). Essa é só
    a banda **preferida** para a primeira tentativa — ver fallback de
    banda acima para o que acontece se nenhum canal dela funcionar.
- **`INTERNET_INTERFACE`** aceita o nome de uma interface fixa ou
  `auto`: nesse caso o hotspot avalia as rotas padrão IPv4 do host
  (`ip -o route show default`), ignora interfaces virtuais/loopback, e
  ranqueia as candidatas por maior velocidade reportada em
  `/sys/class/net/<iface>/speed` (métrica da rota desempata; Wi-Fi não
  reporta velocidade e fica atrás de qualquer Ethernet de propósito);
  sem rota padrão, entram interfaces com link físico ativo (carrier)
  como último recurso; se ainda assim não houver candidata, falha
  explicitamente — mesma filosofia de nunca adivinhar silenciosamente
  já usada para canal/banda. A **própria placa Wi-Fi do hotspot entra
  como candidata do `auto`** somente quando está associada como
  estação agora (`sta_link_probe`, que rejeita a placa em modo AP) —
  é o único fallback possível numa máquina com uma única porta
  Ethernet; em `--no-virt` ela nunca é candidata. A interface real
  escolhida **não é passada diretamente ao `create_ap`**: o hotspot
  cria uma interface dummy estável (`BINDNET_UPLINK_INTERFACE`, padrão
  `bn-uplink`), instala regras próprias em chains `BINDNET-HOTSPOT` de
  `FORWARD`/`POSTROUTING` **e roteamento por política** (tabela
  dedicada `91` + `ip rule` para o `HOTSPOT_CIDR`, ver `uplink.sh`):
  o `MASQUERADE -o <iface>` sozinho não muda por onde o tráfego sai —
  com duas rotas default simultâneas (Ethernet + Wi-Fi) o kernel
  continuaria usando a de menor métrica, ignorando a fonte escolhida;
  a tabela dedicada garante que o tráfego dos clientes do hotspot sai
  sempre pela fonte selecionada, independente da rota preferida do
  host. Na troca, as entradas de conntrack dos clientes são derrubadas
  (`conntrack -D`, best-effort) para os fluxos reconectarem já pela
  fonte nova. Um monitor de uplink (`services/worker/hotspot/
  uplink_monitor.sh`) roda **sempre**: lê `INTERNET_INTERFACE` direto
  do banco e, quando o painel troca a fonte (quick-switch do card de
  resumo → `POST /api/hotspot/uplink`, que só grava a chave), alterna
  NAT/rota **ao vivo, sem derrubar/recriar o AP** — os clientes
  conectados não caem. No modo `auto` ele também vigia a **saúde** da
  fonte atual a cada `UPLINK_HEALTH_CHECK_INTERVAL` segundos (padrão
  3s; env do container, sem equivalente no painel): link físico via
  `carrier` **e internet de verdade** via `ping -I` pela própria
  interface (alvos = `HOTSPOT_DNS_FALLBACKS`, timeout 2s — 1s dava
  falso-negativo na STA Wi-Fi, cujo rádio é compartilhado com o AP).
  Link físico morto = failover **imediato e orientado a evento**: o
  monitor dorme escutando `ip monitor link` (netlink) e acorda no
  instante em que a interface atual muda de estado — desligar a placa
  à mão é percebido em ~1s, sem esperar o tick de polling (que
  continua existindo como retaguarda); internet perdida com link de
  pé = failover após `UPLINK_HEALTH_FAILURES_THRESHOLD` checagens
  ruins seguidas (padrão 2, ~6-9s — 1 ping perdido é normal e trocar
  na primeira falha causava ping-pong entre fontes). O destino é
  sempre a próxima candidata **com internet comprovada**; sem nenhuma
  comprovada, mantém a fonte atual (nunca troca "no escuro") — exceto
  com o link físico morto, quando qualquer candidata com carrier
  serve;
  a reavaliação "apareceu candidata melhor?" e a leitura do painel
  continuam na cadência de `UPLINK_MONITOR_INTERVAL` (padrão 10s). O
  painel acompanha: cada troca loga "Fonte real de internet do
  hotspot: ...", linha da qual `GET /api/hotspot/status` extrai a
  interface em uso (`parseHotspotInternetInterface`) — o card de
  resumo mostra "auto (interface)" atualizado no poll de 5s. A única
  troca que exige reiniciar o hotspot é passar a fonte para a própria
  placa Wi-Fi (`INTERNET_INTERFACE == WIFI_INTERFACE`) enquanto o AP
  está em `--no-virt` (sem associação de estação): sem estação não há
  uplink Wi-Fi possível, o monitor loga um aviso e mantém a fonte
  atual até um restart reassociar a placa. `attempt_hotspot_cycle`
  também relê essa chave do banco antes de cada rodada
  (`refresh_internet_strategy_from_db`), para um retry pós-queda já
  raciocinar com a fonte mais recente. `INTERNET_INTERFACE` pode ser a
  **mesma** interface de `WIFI_INTERFACE` (hotspot e saída de internet
  pela mesma placa física, modo AP+STA concorrente no mesmo rádio) — o
  hotspot detecta esse caso e loga um aviso citando se `iw phy<N>
  info` reporta suporte a combinações `AP`+`managed` simultâneas, mas
  **nunca bloqueia**: quem decide se funciona de fato é o próprio
  `create_ap`, exatamente como no retry de canal. O mesmo vale mesmo
  quando `WIFI_INTERFACE` já está conectado como estação a **outra**
  rede (não à internet compartilhada por `INTERNET_INTERFACE`) no
  momento em que o hotspot sobe: se `iw phy<N> info` reportar a
  combinação `AP`+`managed`, o `create_ap` cria uma interface virtual
  (`ap0`) e mantém a conexão de estação existente.
