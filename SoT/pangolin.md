# Túneis Pangolin Cloud

O painel `/tunnels` publica domínios locais pelo fluxo:

**Pangolin Cloud → container Newt no Bindnet → nginx integrado → aplicação.**

A integração usa a Integration API oficial, sem automatizar login no painel
Pangolin. Não instala o servidor Pangolin nem altera hotspot, Wi-Fi ou DNS.

## Contas e organizações

É possível cadastrar várias contas/organizações, cada uma com nome, chave,
organização, origem da API e endpoint do conector. Os padrões são
`https://api.pangolin.net` e `https://app.pangolin.net`, sem `/v1` ou caminhos.
A conta selecionada fica no parâmetro `account` da URL. Queries, recursos,
credenciais e containers são isolados por conta, mesmo com IDs remotos iguais.
Os limites, funcionalidades e permissões de cada plano são impostos pelo
Pangolin; o Bindnet não os altera nem transfere recursos entre contas.

A criação da chave de organização ocorre no Pangolin. Chaves não voltam ao
navegador depois de gravadas. Pode trocar nome/chave sem perder vínculos;
trocar organização, servidor ou endpoint exige remover antes os sites geridos.
A remoção de conta é bloqueada enquanto houver sites/recursos locais vinculados.

## Publicar um domínio local

1. Selecione a conta e abra **Sites → Novo site**. O formulário abre num modal,
   com rolagem interna e layout responsivo.
2. Informe o nome do site, subdomínio público e domínio base. Escolha um domínio
   da organização ou uma opção de namespace conhecida/disponível no Cloud.
3. Informe o **domínio DNS local no nginx**, por exemplo `app.empresa.local`.
   As sugestões reutilizam `GET /api/dns/discovered-servers`; também pode digitar.
4. Selecione HTTPS (443, padrão) ou HTTP (80) para o nginx local. Não há campo
   de IP ou porta da aplicação: o nginx já conhece o encaminhamento final.
5. **Criar e publicar site** cria site `newt`, guarda as credenciais, inicia
   o container Newt e aguarda o site ficar online no Cloud (até 60s). Depois
   cria o recurso público, prepara o alvo e ativa o recurso. Container iniciado
   ainda não significa túnel conectado. O primeiro arranque pode baixar a imagem.

O modal mostra a etapa atual, tempo decorrido e IDs já criados. A operação tem
prazo total de 180s no backend e espera de 195s no navegador; consultas de
status não aguardam downloads ou mutações. Mutações concorrentes recebem 409
para evitar filas e publicações duplicadas. A atualização das listas ocorre
em segundo plano, sem manter o formulário preso após a resposta da criação.

O alvo remoto é `127.0.0.1:443` ou `:80`, alcançado pelo Newt na rede do host.
O domínio local é enviado por `setHostHeader` e, em HTTPS, `tlsServerName`.
Assim o nginx escolhe o `server_name` e o certificado corretos sem depender de
resolver nomes locais no Pangolin Cloud ou dentro do container.
TLS público e SSO são preservados por padrão. O transporte TLS do Pangolin
consultado aceita certificados locais quando utiliza `tlsServerName`; nenhuma
chave privada da CA Bindnet é enviada ao Cloud.

Na aba **Serviços → Novo recurso**, o mesmo formulário reutiliza um site
existente. Cria recurso e alvo automaticamente, deixando o recurso desativado
até o operador ativá-lo. A ação **Destino** altera somente domínio local e
HTTP/HTTPS; atualiza Host, SNI e alvo juntos, pausando o recurso durante a troca.
Se alguma etapa falhar, o recurso permanece desativado para recuperação.

A aplicação deve aceitar o endereço público nas suas URLs, redirects, cookies,
CORS e callbacks. A integração não reescreve HTML, cookies ou redirects que
uma aplicação gere fixamente com seu domínio local.

## Domínios gratuitos e compatibilidade Cloud

O OpenAPI público anuncia `/domain/check-namespace-availability` e
`/domains/namepaces`. Em **2 de novembro de 2026**, consultas autenticadas à
conta real confirmaram **404** nessas rotas, apesar da chave válida com todas
as permissões. A fonte dos roteadores de integração também não as registra.
Isso não deve ser apresentado como chave incorreta.

O Bindnet tenta a consulta documentada. Se receber 404, consulta os recursos
existentes da **mesma conta**, obtém `domainId`, `subdomain` e `fullDomain`
pela API oficial e deriva os namespaces comprovados neles. Para a origem
oficial HTTPS `api.pangolin.net`, também oferece o catálogo Cloud confirmado:
`hostlocal.app`, `tunneled.to`, `tunnelmy.app` e `visitmy.app`. Os quatro IDs
foram aceitos pela criação oficial de recursos temporários em 02/11/2026,
removidos ao terminar. Isso permite criar o primeiro recurso numa conta vazia,
sem depender de outro recurso criado previamente no painel Pangolin.

O catálogo é uma alternativa conhecida, não uma consulta global nem uma lista
exaustiva. A disponibilidade do nome, permissões e limites são confirmados pelo
Pangolin ao criar. Nomes já ocupados na conta são excluídos e namespaces não
são duplicados. Em outras origens, o catálogo Cloud não é utilizado: são
necessários domínios próprios ou namespaces comprovados pelos seus recursos.
Quando a consulta oficial existir, suas opções prevalecem sobre o catálogo.
Não utiliza cookies, scraping ou rotas de sessão do painel.

A consulta tem debounce, seleção explícita, “Mostrar mais”, estado vazio e
mensagem de erro. A publicação fica desativada enquanto consulta, quando a
lista estiver vazia ou até selecionar um domínio válido; erros antigos de
seleção são limpos ao trocar a opção. Antes da criação, o backend revalida a opção. Domínios CNAME
usam o domínio completo, sem subdomínio. Contagens de paginação aceitam tanto
números quanto strings numéricas, como `"0"`, observado no Cloud real.

## Configuração de recursos

A ação **Configurações** abre um modal com dados atuais do Pangolin:

- Gerais: nome, identificador, ID de política compartilhada e endereço público.
- HTTP(S): HTTPS público, sessões persistentes, balanceamento, encaminhamento
  de identidade e cabeçalhos de requisição/resposta. Host/SNI são exibidos,
  mas alterados exclusivamente pela ação de domínio local para manter o alvo.
- Autenticação: SSO, bloqueio, lista de e-mails, funções/utilizadores permitidos,
  PIN, código de acesso, autenticação básica por cabeçalho e CAs públicas mTLS.
- Regras: ativar/desativar avaliação, criar, editar e remover regras com ação,
  correspondência, valor, prioridade e estado.
- Manutenção: habilitar, modo automático/forçado, título, mensagem e previsão.

Campos não alterados não são enviados. Valores herdados de política
compartilhada são identificados. Senhas e PINs podem ser substituídos/removidos,
mas não são retornados na leitura. O backend valida uma lista fechada de
campos/métodos; não existe proxy arbitrário da API nem execução arbitrária.
A disponibilidade de cada opção depende da versão, plano e permissões remotas.

Sites/recursos externos são somente leitura; não há importação de credenciais
Newt existentes. Recursos privados, TCP/UDP públicos, alertas de uptime,
instalação de remote nodes e administração da conta Cloud não fazem parte
 deste fluxo de publicação via nginx.

## Container, dados e segurança

Somente o worker acessa o Docker. O conector utiliza `fosrl/newt:1.19.0`,
rede host, sem privileged, capabilities, volumes, dispositivos ou socket Docker.
SSH e clientes remotos ficam desativados. O healthcheck verifica o arquivo de
conectividade; a UI distingue processo em execução e site online no Pangolin.

O nome é `bindnet-newt-<accountId>-<siteId>`. Uma label de propriedade é
validada antes de qualquer ação. A conta migrada `legacy` conserva os nomes
antigos `bindnet-newt-<siteId>`. As rotas internas exigem conta válida e ID
positivo; imagem, comandos e configuração Docker não são aceitos do chamador.
O backend recupera automaticamente os conectores ao arrancar e a cada 15s,
usando endpoint e credenciais guardados no Postgres. O worker reutiliza o
container em execução, inicia o parado e cria o ausente, sem recriar sites ou
recursos no Cloud. Falhas temporárias do worker são repetidas nas próximas
passagens; a recuperação não depende de abrir a página no navegador.
`pangolin_account_sites.desired_running` começa como `true` para sites novos e
existentes. **Parar** guarda `false` antes de contactar o worker e suspende a
recuperação, inclusive após reiniciar o backend; **Iniciar conector** volta a
guardar `true`. Mutações do painel e recuperação compartilham o mesmo bloqueio
para evitar religar um conector enquanto ele é parado ou removido.
`unless-stopped` também mantém conectores iniciados após reiniciar o Docker.
Eles são geridos pelo worker, fora do Compose: `docker compose down` não os remove.
Remova recursos e sites pelo painel antes de descontinuar a integração.

As migrations `20261030000000`, `20261102000000` e `20261102010000` criam o
estado inicial, o domínio local e as tabelas atuais `pangolin_accounts`,
`pangolin_account_sites` e `pangolin_account_resources`. A integração antiga é
copiada para `legacy`, preservando segredos e vínculos. Tabelas legadas não
são apagadas. A migration `20261104000000` acrescenta o estado desejado dos
conectores para o arranque automático e a parada persistente.
Não existem variáveis novas no `.env`.

API keys e credenciais Newt ficam no PostgreSQL; as do Newt também ficam no
ambiente do container, legível por administradores Docker. Backups contêm
esses segredos. Não são enviados ao navegador, argumentos ou auditoria Mongo.
Somente certificados públicos são aceitos no cadastro mTLS. Todas as rotas
`/api/tunnels/*` exigem sessão administrativa e as operações são limitadas
à conta/recurso correspondente. A auditoria registra ações sem os valores.

## Falhas, remoção e validação

Mutações remotas não são repetidas automaticamente. Quando uma resposta de
criação se perde, consulte o estado no Pangolin antes de repetir. IDs recebidos
são persistidos antes das próximas etapas; compensações tentam remover recursos
incompletos. Quando a compensação falha, o vínculo permanece recuperável.
Um site cujo recurso falhou continua visível para criar outro recurso ou remover.
Se o conector não ficar online na criação de site, nenhum recurso é publicado,
e a mensagem informa o ID para recuperar na aba Sites. Pode então criar um
recurso pela aba Serviços. Falhas posteriores preservam IDs recuperáveis. Ativar
somente o recurso remoto não inicia um conector ausente e pode produzir 404.
O botão de ativação do Bindnet recusa sites offline com 409 e orienta iniciar
o conector; a desativação continua disponível independentemente da conexão.

Na remoção, primeiro remova os recursos geridos do site. O worker remove o
conector antes do site remoto; erros preservam o cadastro para repetir.
404 na remoção de um recurso já ausente permite limpar o vínculo local.
Recursos criados externamente no mesmo site podem perder destinos quando o
site for removido; a UI confirma essa operação e avisa do efeito.

O prazo de proxy versionado é de 240s para comportar o provisionamento inicial.
Testes simulam API, banco SQLite e Docker, incluindo isolamento de contas,
Host/SNI, falhas parciais, segredos e as incompatibilidades reais do Cloud.
Publicação e alterações reais exigem ação ou autorização explícita do operador;
conectores cadastrados com estado desejado ligado são recuperados automaticamente.
Em 2 de novembro de 2026, com essa autorização, o teste
real criou site, recurso, alvo e Newt, validou HTTPS público autenticado com
conteúdo idêntico ao nginx local (HTTP 200) e removeu recurso, token, site e
container temporários. O conector do site existente foi recuperado e o mesmo
encaminhamento foi confirmado, preservando SSO e o cadastro do operador.

Fontes e contratos externos: [pesquisa da API](pangolin-api-research.md).
