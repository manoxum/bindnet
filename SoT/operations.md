# Operação do Bindnet

O Bindnet controla recursos físicos da máquina. Subir, parar, reiniciar ou
recriar o Compose pode interromper conectividade, o hotspot e a resolução DNS
para utilizadores ativos. Não execute essas ações sem autorização explícita do
operador.

O ponto de entrada operacional para imagens é `bin/promote`; consulte
[bin/SoT/promote.md](../bin/SoT/promote.md) para o contrato de build,
publicação e promoção. A topologia Compose e as configurações de ambiente são
definidas no repositório raiz.

Antes de testar alterações de rede ou hotspot, prefira validação estática,
testes unitários e revisão de configuração. Teste no host real apenas com
consentimento explícito e janela de manutenção.

## Launcher de desenvolvimento

`make install-dev` instala um atalho por utilizador Linux em
`~/.local/share/applications/bindnet-dev.desktop`, um ícone do Bindnet e
`~/.local/bin/bindnet-dev`, ligado ao checkout atual. Não precisa de sudo
e não inicia serviços. Reinstalar atualiza o mesmo atalho, preservando dados
e ambientes. A localização do menu é a do host, mesmo em terminais de IDEs
que exportam caminhos XDG privados.

O atalho e `make start-dev` usam `scripts/desktop/launcher.py`. A tela nativa
Python/Tk aparece antes das verificações e mostra versão, progresso e os
registos desta execução. Valida Linux, ferramentas, acesso ao daemon Docker,
configuração Compose e credenciais de bootstrap. Não inicia o daemon nem
altera permissões do Docker. Ausência de ambiente impede o arranque e indica
como usar `scripts/gen-secrets.sh`; um arquivo existente nunca é sobrescrito.

Antes de qualquer build/arranque, o operador precisa clicar em **Iniciar
Bindnet**, após o aviso de possível interrupção de DNS/Wi-Fi e de reativação
do hotspot pela configuração persistida. Cancelar antes dessa confirmação
não altera serviços. Uma trava impede dois arranques simultâneos pelo launcher.

O launcher usa os manifests Compose existentes, projeto `bindnet` e
`.env.<current_stage>` definido em `promote.yml` (fallback `.env`), inclusive
para `BINDNET_ENV_FILE`, mantendo o mesmo ambiente que o worker utiliza.
Compila todas as imagens com os Dockerfiles normais e o cache do Docker;
uma falha de build impede o `up`. Só então cria as redes externas
`wnet`/`dbnet`, se ausentes, e executa `up --detach --no-build`.
Este fluxo usa diretamente o Compose para separar build e arranque;
Sem alvo, `make` executa `docker compose --env-file .env.main up --build -d`;
`make up` é equivalente. `bin/promote` continua sendo o entrypoint dos restantes
comandos operacionais do Makefile.
Não executa `down`, remove volumes, reescreve configurações do nginx-ui nem
modifica o DNS/CA do host por fora dos serviços existentes. `make dev` e
`make watch` conservam o fluxo de recarregamento de código do `promote.yml`.

Após o arranque, aguarda até 180 segundos pelo HTML Bindnet e a resposta de
autenticação da API em `https://admin.bindnet.local.com`, com validação TLS
normal e sem proxies HTTP do ambiente. Só depois abre essa URL no navegador
padrão por `xdg-open`. O domínio deve resolver localmente, o nginx-ui deve
ter o site/certificado configurado e a CA deve estar confiada. Falhas de
DNS/TLS apresentam instruções para aceder ao painel por IP/porta e concluir
essa configuração; não instalam certificados nem alteram `/etc/hosts`.

Uma falha mantém a tela e os detalhes disponíveis. Registos privados ficam
em `.harness/logs/launcher.log`, substituídos a cada execução; capturas de
`compose config` e `docker info` não são impressas. Fechar a tela após erro
não para serviços que já tenham iniciado. `make check-launcher` verifica
esses contratos com dependências simuladas, sem operar hardware.
