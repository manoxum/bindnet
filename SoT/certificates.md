# Gestão de certificados (`services/backend/certificates.go`)

Substitui o antigo `services/worker/cert-proxy` (removido). A emissão
de certificados agora é sempre uma ação explícita, disparada por
request HTTP autenticado no painel — **não existe mais emissão
automática por SNI, nem nada escutando nas portas 80/443**. Isso é uma
mudança de comportamento intencional, não um efeito colateral.

Regras:

- `loadOrImportCA` roda uma vez, no boot do `backend`, com
  precedência de 3 níveis:
  1. Já existe uma linha na tabela `ca` do Postgres → usa essa, nunca
     regenera.
  2. Senão, se o volume legado `cert_proxy_data` (montado
     somente leitura em `/certproxy-data`) tiver `ca.crt`+`ca.key` →
     importa para o Postgres (`source='imported'`). Isso preserva a
     confiança já estabelecida nos dispositivos que já importaram essa
     CA quando o stack ainda usava `cert-proxy`.
  3. Senão → gera uma CA nova (`source='generated'`): RSA 4096, válida 10
     anos, CN vindo de `panel_config.CA_COMMON_NAME` (definido no painel
     em Configurações; padrão "Bindnet Local Development CA"). Mesmos
     parâmetros criptográficos do antigo `cert-proxy`. Trocar o CN depois
     não renomeia nem reemite a CA já criada — só valeria para uma CA
     gerada do zero.
- `POST /api/certificates` emite um certificado *leaf* (RSA 2048) com
  um `name` amigável obrigatório, separado dos `domains` (lista de
  domínios/IPs) — todos os itens viram
  SAN (Subject Alternative Name) de um único certificado; o primeiro
  item normalizado é o domínio/CN primário, salvo na coluna `domain`;
  `name` identifica a linha nas listagens e na importação para o
  `nginx-ui`. Não pode haver dois certificados ativos com o mesmo
  nome. Um item pode ser um domínio curinga (`*.mydomain`) —
  cobre o caso de emitir um certificado para `*.mydomain` junto com
  `app.mydomain`, `app2.mydomain` etc. em uma única emissão. Sempre
  cria uma linha nova em `certificates` — sem cache/reuso por domínio,
  já que emitir é agora uma ação explícita do usuário, não um lookup
  implícito por SNI. `validityQuantity` (inteiro) e `validityUnit`
  (`days`|`weeks`|`months`|`years`) definem `NotAfter`; ausentes ou
  inválidos caem para o padrão fixo anterior (2 anos), e o resultado
  nunca ultrapassa a validade da própria CA local.
- `PUT /api/certificates/{id}` reemite um certificado ativo: preserva
  seu nome amigável, gera uma chave e um certificado completamente
  novos com os novos `domains`/período de validade e move a versão
  anterior para os revogados. O material novo substitui o anterior no
  `nginx-ui`; eliminar definitivamente a versão antiga não remove esse
  material enquanto a nova versão com o mesmo nome continuar ativa.
- Após persistir no Postgres, o backend importa o certificado para o
  `nginx-ui`, para que ele apareça em
  `/#/certificates/list?search={}`. Se as credenciais do `nginx-ui`
  estiverem definidas no painel (Configurações, tabela `panel_config`:
  `NGINX_UI_USERNAME`/`NGINX_UI_PASSWORD`), usa a API `/api/certs`;
  caso contrário, grava os PEMs em `/etc/nginx/ssl/<nome>/`
  e registra a linha correspondente (com todos os SANs em `domains`) no
  `database.db` do `nginx-ui` via os volumes compartilhados.
- Cada domínio/IP é normalizado antes de emitir: minúsculas, sem
  porta, sem `.` final; o primeiro rótulo pode ser `*` (curinga); se
  algum rótulo restante não passar numa validação básica de caracteres
  (`[a-z0-9-]`), aquela entrada cai para `localhost.local` em vez de
  emitir certificado para um valor não sanitizado — mesma regra do
  antigo `cert-proxy`, agora aplicada item a item da lista.
- `DELETE /api/certificates/{id}` revoga: seta `revoked_at`, **nunca
  deleta a linha** — o certificado revogado continua aparecendo na
  listagem dedicada de revogados do Bindnet, com status "revogado", e
  sai da lista principal de certificados emitidos. A mesma chamada
  remove o certificado da lista do `nginx-ui` e limpa os PEMs importados
  em `/etc/nginx/ssl/<nome>/`.
- `DELETE /api/certificates/{id}/permanent` elimina definitivamente
  uma linha **somente se ela já estiver revogada**; certificados ativos
  precisam passar primeiro pela revogação para sair do `nginx-ui`.
- `GET /api/certificates/ca` e `GET /api/certificates/{id}/download`
  servem os PEMs para download e exigem sessão autenticada, igual às
  demais rotas de `/api/certificates/*`. **Exceção deliberada**:
  `GET /api/mesh/ca` devolve o mesmo PEM da CA (só o certificado
  público, nunca a chave privada) **sem sessão**, com
  `Access-Control-Allow-Origin: *` — usada pela tela "Servidores
  Bindnet" do painel (`services/frontend/src/components/bindnets/`)
  para buscar a CA de outros nós da malha direto do navegador, sem
  autenticação entre backends. É seguro porque uma CA raiz é feita
  para ser distribuída publicamente (mesmo papel do antigo
  `cert-proxy`, que servia `/ca.crt` anonimamente na porta 80 — só que
  agora escopado só a essa rota, em vez de todo o cert-proxy).
- `POST /api/certificates/ca/install-local` instala uma CA no próprio
  host Linux onde o stack está rodando. Por padrão usa a CA deste
  backend; aceita um corpo opcional `{"certificatePem": "..."}` para
  instalar a CA de **outro** servidor Bindnet (buscada via
  `GET /api/mesh/ca` dele) — usado pelo card de CA de nós remotos em
  "Servidores Bindnet". Em qualquer um dos casos o backend só repassa
  o PEM para o `worker`; o `worker` valida que é uma CA, grava
  `/usr/local/share/ca-certificates/bindnet-local-ca.crt` e executa a
  ação fixa `update-ca-certificates` (mais a importação nas stores
  NSS do Chrome/Chromium e Firefox de cada usuário do host, ver
  `services/worker/controller/browser_trust.go`). Essa rota não aceita
  caminho nem comando vindo do frontend/backend, só o conteúdo do
  certificado.
- A chave privada da CA (`private_key_pem` na tabela `ca`) fica no
  `postgres_data` — apagar/recriar esse volume tem o mesmo
  efeito que apagar o antigo `cert_proxy_data`: invalida a CA e todos
  os certificados já confiados nos dispositivos do usuário.

