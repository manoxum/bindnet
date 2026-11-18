# Encerramento e recuperação do hotspot

- Ao encerrar (`SIGTERM`/`SIGINT`/saída normal), o hotspot **sempre**
  chama `create_ap --stop` antes de sair, para não deixar a placa em
  modo AP "preso".
- O container precisa rodar `privileged: true` e `network_mode: host`
  porque manipula diretamente a interface Wi-Fi física do host.
- **Detecção de falha de beacon e auto-recuperação em duas camadas**
  (`services/worker/hotspot/watchdog.sh` + laço de retry no fim de
  `entrypoint.sh` + `reconcileHotspotOnce` em
  `services/backend/hotspot_reconcile.go`): alguns drivers/kernels
  reproduzem uma regressão conhecida do hostapd 2.11 (802.11be/MLO —
  ver comentário no `Dockerfile`, que por isso fixa `hostapd=2.10-r6`)
  onde, após um cliente Wi-Fi associar/reassociar com o AP já de pé, o
  adaptador passa a recusar `Failed to set beacon parameters`
  indefinidamente sem o `create_ap` sair sozinho — o hotspot fica
  "vivo" mas inutilizável para novos clientes.
  - **Camada 1 (worker, imediata)**: o hotspot acompanha seu próprio
    log em tempo real e, se essa falha se repetir
    (`HOTSPOT_BEACON_FAILURE_THRESHOLD`, padrão 2, dentro de
    `HOTSPOT_BEACON_FAILURE_WINDOW_SECONDS`, padrão 20 — variáveis de
    ambiente do container, sem equivalente no painel), derruba o
    `create_ap` (mesmo sinal de parada limpa usado por
    `stop`/`SIGTERM`). O próprio `entrypoint.sh`, na mesma execução do
    script, detecta que o `create_ap` caiu sem que uma parada de
    verdade tenha sido pedida e tenta de novo no lugar, com backoff
    crescente (`HOTSPOT_RESTART_BACKOFF_SECONDS`, padrão 3s por
    tentativa) até `HOTSPOT_MAX_RESTART_ATTEMPTS` (padrão 5) — a nova
    tentativa começa direto pelo último canal/banda que funcionou
    nesta execução (evita repetir uma varredura completa, incluindo
    bandas travadas por regulamentação de firmware que nunca vão
    funcionar). `cleanup()` usa `timeout` ao chamar `create_ap --stop`
    para nunca travar indefinidamente esperando uma instância presa
    num loop de erro do driver.
  - **Camada 2 (backend, rede de segurança, ciclo de 15s)**: se as
    tentativas locais do worker se esgotarem (ou o processo cair por
    qualquer outro motivo, incluindo o container/host reiniciar),
    `reconcileHotspotOnce` detecta `"running": false"` e, se a última
    intenção do admin foi ligar (`POST /api/hotspot/start`, nunca
    desfeita por um `POST /api/hotspot/stop`), religa sozinho pelo
    mesmo `startHotspotAndReapply` que `AutoStartHotspotOnBoot` usa.
    Uma parada deliberada pelo painel nunca é desfeita por nenhuma das
    duas camadas. Esta camada **não** depende do interruptor de arranque
    automático (ver seção própria abaixo): ele governa só o arranque do
    backend, nunca a recuperação de uma queda.
  - **O estado `starting` também é recuperado.** `status_service`
    reporta três estados: `running` (serviço vivo *e* instância do
    `create_ap` no ar), `starting` (serviço vivo, nenhum `create_ap`) e
    `stopped`. `starting` é legítimo por alguns segundos em todo start,
    mas ficar **preso** nele era um buraco: o runner continuava vivo
    retentando, o worker continuava respondendo `running: true`, e a
    Camada 2 — que só agia com `running: false` — nunca disparava. Do
    lado do operador isso aparecia como "o painel diz ligado mas não
    está, e não recupera sozinho": a única saída era Parar e Iniciar na
    mão. Agora `reconcileHotspotService`
    (`hotspot_reconcile_service.go`) cronometra o estado e, passados
    `hotspotStartingGrace` (60s), força um `restart` — o mesmo caminho
    que o operador usaria à mão, e que mata o runner preso antes de
    subir outro (um `start` sozinho só responderia "Servico do hotspot
    ja esta em execucao"). O painel também deixou de mentir: mostra
    **"a iniciar…"** em vez de "ligado".
  - **Rádio bloqueado (`rfkill`) suspende o arranque automático e a
    recuperação.** O status interno devolve `radioBlocked` e o backend
    mantém a intenção de ligar enquanto aguarda o rádio. A reconciliação
    volta a tentar no ciclo seguinte ao desbloqueio. O arranque confirma
    a disponibilidade antes de iniciar o runner e o runner reconfirma
    antes de configurar o uplink. Uma rodada que começa bloqueada
    termina antes de testar canais ou gravar falhas no histórico.
  - **O desbloqueio automático de `rfkill` vale só no start pedido pelo
    operador** (`HOTSPOT_START_REASON=manual`, propagado pelo backend no
    corpo de `/hotspot/start` e pelo worker no `docker exec -e`). O
    arranque manual comunica o erro se o rádio continuar bloqueado.
    Desbloquear numa auto-recuperação desfaria a ação de quem acabou de
    desligar o Wi-Fi: o interruptor de Wi-Fi do sistema simplesmente
    deixaria de funcionar enquanto o hotspot estivesse ligado. O caso
    que originou o desbloqueio (bloqueio espúrio deixado por um
    `docker compose up --build`) continua coberto — é destravado no
    próximo start pelo painel, que é como o operador reage a "o hotspot
    não sobe".
  - **Falha só conta contra o canal quando é mesmo do canal.**
    `channel_failure_is_attributable` (`history.sh`) filtra, antes de
    gravar em `hotspot_channel_history`, as causas reconhecidamente
    alheias ao canal (`cannot open log`/`Permission denied`,
    `RTNETLINK ... Resource busy`, `can not be a station ... and an AP`,
    `is not an interface`). "AP-ENABLED ausente do log" **não** é prova
    de rejeição do adaptador: qualquer coisa que mate o `create_ap`
    antes do AP subir produz o mesmo sintoma. Isso já aconteceu de
    verdade — o `dnsmasq` morrendo por permissão de log gravou centenas
    de falsas rejeições (o canal 10, que funciona, chegou a 274 falhas
    contra 11 sucessos), envenenando o ranking de bandas a ponto de o
    hotspot passar a preferir 5GHz numa placa onde 2.4GHz é que
    funcionava. O filtro é conservador: causa nova/desconhecida
    continua contando como rejeição.
