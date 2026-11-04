# Pesquisa da API oficial do Pangolin

Verificação: **30 de outubro de 2026**. Este documento registra o contrato
externo consultado para a integração; as regras do Bindnet ficam em `SoT/`.
Pangolin avisa que a API pode mudar entre versões. Em instalações próprias,
compare o Swagger da instalação com a especificação pública antes de atualizar.
[Fonte: Integration API](https://docs.pangolin.net/manage/integration-api).

## Conexão e permissões

- A URL da Integration API é distinta do endpoint do conector: por exemplo,
  `https://api.example.com/v1` versus `https://pangolin.example.com`.
- Self-hosted: habilitar `flags.enable_integration_api: true`; a porta interna
  padrão é `3003`, com roteamento HTTPS externo. Swagger: `/v1/docs`.
  [Fonte: habilitação](https://docs.pangolin.net/self-host/advanced/integration-api).
- Autenticar com `Authorization: Bearer <api-key>`. Uma chave de organização
  basta para sites, recursos e destinos daquela organização; chaves root são
  exclusivas de self-hosted e desnecessárias para essa integração. A chave é
  criada no painel Pangolin e exibida uma única vez. A documentação consultada
  não restringe essa integração à Enterprise.
  [Fonte: autenticação](https://docs.pangolin.net/manage/integration-api).
- As permissões verificadas nas rotas são `createSite`, `listSites`, `getSite`,
  `deleteSite`, `createResource`, `listResources`, `getResource`,
  `updateResource`, `deleteResource`, `createTarget`, `listTargets`,
  `updateTarget`, `deleteTarget`, `listOrgDomains`, conforme as operações usadas.
  Não é necessário conceder todas quando a implementação usa só parte delas.
  [Fonte: rotas oficiais](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/integration.ts).

## Sites e credenciais

Todos os caminhos abaixo são relativos a `/v1`.

| Operação | Método e caminho | Corpo mínimo ou consulta |
| --- | --- | --- |
| Criar site | `PUT /org/{orgId}/site` | `{"name":"Bindnet","type":"newt"}` |
| Listar sites | `GET /org/{orgId}/sites` | `page=1&pageSize=100` |
| Consultar site | `GET /site/{siteId}` | — |
| Renomear site | `POST /site/{siteId}` | `{"name":"Novo nome"}` |
| Apagar site | `DELETE /site/{siteId}` | — |

A criação gera `newtId`, `secret` e endereço automaticamente. Não precisa de
`pick-site-defaults` nem de API de geração de credenciais separada. Somente a
resposta de criação inclui o segredo; persistir antes de iniciar o conector.
`siteId` é o identificador numérico usado pela API. `newtId` é a credencial do
conector e não deve ser substituído pelo `siteId`.
[Fonte: fluxo oficial](https://docs.pangolin.net/manage/common-api-routes).

A resposta de criação tem `data.siteId`, `data.newtId`, `data.secret` e os dados
do site. A listagem tem `data.sites` e
`data.pagination:{total,pageSize,page}`. Cada site inclui `siteId`, `niceId`,
`name`, `type`, `online`, `address`, `status`, `newtVersion`, `agent` e
`agentVersion`; `online` é omitido nos sites `local`. `pageSize` aceita inteiro
positivo, sem máximo declarado; percorrer as páginas até `total`.
[Fontes: criação](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/site/createSite.ts),
[listagem](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/site/listSites.ts).

## Recursos públicos e destinos

| Operação | Método e caminho | Corpo mínimo ou consulta |
| --- | --- | --- |
| Listar domínios | `GET /org/{orgId}/domains` | `limit=1000&offset=0` |
| Criar HTTP | `PUT /org/{orgId}/public-resource` | `{"name":"Aplicação","mode":"http","domainId":"...","subdomain":"app"}` |
| Criar TCP/UDP | `PUT /org/{orgId}/public-resource` | `{"name":"Serviço","mode":"tcp","proxyPort":12345}`; ou `mode:"udp"` |
| Listar recursos | `GET /org/{orgId}/public-resources` | `page=1&pageSize=100`; filtro opcional `siteId` |
| Consultar recurso | `GET /public-resource/{resourceId}` | — |
| Atualizar recurso | `POST /public-resource/{resourceId}` | Ex.: `{"enabled":false}` |
| Apagar recurso | `DELETE /public-resource/{resourceId}` | — |
| Criar destino | `PUT /public-resource/{resourceId}/target` | `{"siteId":123,"ip":"127.0.0.1","port":8080,"method":"http"}` |
| Listar destinos | `GET /public-resource/{resourceId}/targets` | `limit=1000&offset=0` |
| Atualizar destino | `POST /target/{targetId}` | `siteId` e `ip` obrigatórios; `port`, `method`, `enabled` opcionais |
| Apagar destino | `DELETE /target/{targetId}` | — |

Portas são inteiros de `1` a `65535`; IDs numéricos devem ser positivos.
Use `mode`: os campos antigos `http`/`protocol` estão obsoletos. HTTP precisa
de `method:"http"`, `"https"` ou `"h2c"` no destino; sem esquema o Pangolin
ignora esse destino. `ip` aceita IP ou hostname acessível pelo conector.
[Fontes: especificação OpenAPI](https://api.pangolin.net/v1/openapi.json),
[guia de destinos](https://docs.pangolin.net/manage/common-api-routes).

O schema de criação é estrito e **não aceita `enabled`, `ssl` ou `sso`**.
HTTP nasce com TLS e SSO habilitados; recursos e destinos nascem habilitados.
Para preparar sem encaminhar tráfego, criar recurso sem destinos, desativar por
`POST`, adicionar destino e só então ativar. Em falha, a integração deve
compensar ou preservar o ID criado para permitir recuperação. Atualizar recurso
não exige `siteId`; isso é exigido na atualização de destino.
[Fontes: criação e políticas padrão](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/resource/createResource.ts),
[schema](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/db/sqlite/schema/schema.ts),
[atualização de destino](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/target/updateTarget.ts).

Domínios retornam `data.domains:[{domainId,baseDomain,verified,type,...}]` e
paginação `{total,limit,offset}`. Domínio `cname` usa `subdomain:null`; `ns` e
`wildcard` aceitam subdomínio ou `null` para o domínio base. Wildcard é exclusivo
de self-hosted. O fluxo automatizado pode selecionar domínios existentes;
alterações no DNS externo continuam dependentes do provedor DNS.
[Fonte: domínios](https://docs.pangolin.net/manage/common-api-routes).

Recursos retornam `data.resources` e paginação `{total,pageSize,page}`. Campos
atuais: `resourceId,name,mode,ssl,fullDomain,proxyPort,enabled,sso,domainId`,
`targets:[{targetId,ip,port,enabled,healthStatus,siteName}]` e
`sites:[{siteId,siteName,siteNiceId,online}]`. A lista atual usa `mode`, sem
campos `http`/`protocol`. Destinos retornam `data.targets` e paginação
`{total,limit,offset}`.
[Fontes: recursos](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/resource/listResources.ts),
[destinos](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/target/listTargets.ts).

TCP/UDP públicos **não têm SSO nem regras de autenticação Pangolin**. Precisam
de `flags.allow_raw_resources:true` no servidor, portas abertas no firewall,
mapeamentos no Gerbil e entrypoints Traefik chamados `tcp-PORT`/`udp-PORT`.
A API de recursos não provisiona essa infraestrutura. Self-hosted CE e EE
suportam esses recursos; Cloud exige remote node. Não prometer exposição
TCP/UDP funcional só porque a criação do recurso teve sucesso.
[Fonte: TCP/UDP](https://docs.pangolin.net/manage/resources/public/raw-resources).

## Conector Docker

Pangolin recomenda a CLI para novos sites (`fosrl/pangolin-cli`, comando
`up site`, variáveis `SITE_ID`, `SITE_SECRET`, `PANGOLIN_ENDPOINT`). Newt
continua distribuído e suportado como opção menor. Ambos usam sites do tipo
`newt` na API.
[Fontes: instalação atual](https://docs.pangolin.net/manage/sites/install-site),
[Newt](https://docs.pangolin.net/manage/sites/install-newt).

Para `fosrl/newt:1.19.0`, as variáveis confirmadas são `NEWT_ID`, `NEWT_SECRET`,
`PANGOLIN_ENDPOINT`. O conector faz proxy no espaço do usuário; a descoberta
Docker é opcional via `DOCKER_SOCKET`, que não deve ser montado no Bindnet.
`DISABLE_CLIENTS=true` e `DISABLE_SSH=true` desativam funcionalidades fora do
escopo de proxy público. Não habilitar interfaces nativas WireGuard.
[Fonte: configuração Newt 1.19.0](https://github.com/fosrl/newt/blob/1.19.0/newtconfig/newtconfig.go).

`HEALTH_FILE=/tmp/healthy` cria/remove um arquivo conforme a conectividade do
túnel; verificar com `CMD-SHELL`, `[ -f /tmp/healthy ]`. Esse indicador local
complementa `site.online` da API. A imagem oficial desativa auto-update do
binário; atualizar pela troca da tag da imagem.
[Fontes: healthcheck](https://docs.pangolin.net/manage/sites/configure-site),
[Dockerfile](https://github.com/fosrl/newt/blob/1.19.0/Dockerfile).

Versões consultadas: [Newt 1.19.0](https://github.com/fosrl/newt/releases/tag/1.19.0)
e [Pangolin CLI 0.19.1](https://github.com/fosrl/cli/releases/tag/0.19.1).

## Tratamento do protocolo

As respostas de operações usam `{data,success,error,message,status}`;
`success` e `error` são booleanos. Verificar o status HTTP e o envelope:
`success:false`/`error:true` devem falhar, mesmo se o transporte aceitar a
resposta. A raiz de saúde da API retorna apenas `{"message":"Healthy"}` e
não comprova autorização da chave para a organização. Limites de plano e
permissões continuam validados pelo servidor.
[Fontes: OpenAPI](https://api.pangolin.net/v1/openapi.json),
[rotas e saúde](https://github.com/fosrl/pangolin/blob/3f352ef2d3907ae87f21a5f46a753335be93512c/server/routers/integration.ts).

## Compatibilidade confirmada em 2 de novembro de 2026

A conta real autenticada retornou HTTP 200 nas listagens de sites, recursos e
domínios, mas HTTP 404 nas duas rotas de namespace descritas no OpenAPI.
O [roteador Integration base](https://github.com/fosrl/pangolin/blob/main/server/routers/integration.ts)
e o [roteador privado](https://github.com/fosrl/pangolin/blob/main/server/private/routers/integration.ts)
não registram essas rotas, embora os handlers registrem seus schemas no OpenAPI.
Não inferir disponibilidade de uma rota somente pela sua presença no Swagger.

A paginação de domínios devolveu `total:"0"` (string), enquanto sites e recursos
usaram números. A implementação precisa aceitar ambos os formatos. O fallback
para nomes gratuitos consulta os recursos existentes e seus detalhes para obter
um namespace comprovado pelo par `subdomain`/`fullDomain`. Um nome pode ser
ocupado em outra conta até a criação.

Uma conta vazia expôs a dependência indevida desse fallback em recursos
anteriores. O teste real pela Integration API criou e removeu, um por vez,
recursos com os IDs `hostlocal.app`, `tunneled.to`, `tunnelmy.app` e `visitmy.app`.
Em todos os casos, `GET /public-resource/{id}` confirmou o endereço esperado
e SSO ativo. Os recursos foram mantidos desativados e sem alvos locais; nenhum
site ou conector foi criado. Todos os recursos temporários foram removidos.
Esses IDs formam o catálogo alternativo somente para a API HTTPS oficial do
Cloud. Não são anunciados como disponibilidade global nem como lista exaustiva.

Host e SNI locais usam os campos oficiais `setHostHeader` e `tlsServerName` no
POST de atualização do recurso. O [middleware de cabeçalhos](https://github.com/fosrl/pangolin/blob/main/server/lib/traefik/headersMiddleware.ts)
usa `setHostHeader` para o Host; o [gerador de transporte](https://github.com/fosrl/pangolin/blob/main/server/lib/traefik/getTraefikConfig.ts)
define o nome TLS e aceita certificados locais nesse transporte. O alvo Newt
pode, assim, apontar para o loopback do host mantendo a seleção virtual do nginx.

Configurações adicionais usam POST do recurso e rotas específicas `/password`,
`/pincode`, `/header-auth`, `/whitelist`, `/roles`, `/users`, `/rule` e
`/mtls/certificate`. Os schemas atuais são consultáveis no
[OpenAPI oficial](https://api.pangolin.net/v1/openapi.json). PIN/senha são escritas
com valor novo ou `null` para remover, sem retorná-las ao consumidor.

## Publicação real e ordem de provisionamento

O teste autorizado confirmou que iniciar o container não basta: o websocket
pode conectar antes do registro do túnel e da atribuição do exit node. Criar
recurso/alvo antes dessa conexão reproduziu HTTP 404 no Cloud, mesmo com Host
e SNI corretos e aplicação acessível no loopback local.

A sequência validada foi: criar site, iniciar Newt, aguardar `GET /site/{id}`
confirmar `online:true`, criar recurso, configurar Host/SNI e alvo, ativar.
O teste completo dessa sequência passou em 52,90s, retornando HTTPS 200 com
hash do conteúdo idêntico ao nginx local. Site, recurso, alvo, conector e token
temporários foram removidos. O cadastro do operador foi preservado, seu Newt
recuperado e seu endereço público também validado com HTTP 200.

Para testar SSO sem desativá-lo, foi emitido token temporário pela Integration
API e enviado nos cabeçalhos `P-Access-Token-Id` e `P-Access-Token`. O token foi
removido ao terminar; nenhum segredo foi gravado nos resultados ou enviado ao
navegador. [Contrato dos cabeçalhos](https://docs.pangolin.net/self-host/advanced/config-file).
