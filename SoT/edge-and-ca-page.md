# Serviço `nginx-ui`

- Interface administrativa para configurar sites, exposta só em `9080`
  (porta administrativa) — não há mais proxy público nas portas 80/443
  na frente dela.
- Não monta `/var/run/docker.sock`: essa capacidade fica exclusivamente
  no `worker`. O container define `NGINX_UI_IGNORE_DOCKER_SOCKET=true`
  para a instalação do Nginx UI aceitar esse desenho de segurança.
- Estado (configuração de sites, dados da UI, arquivos servidos) é
  persistido nos volumes externos `nginx_config`, `nginx_ui_data` e
  `www_data`. O backend também monta `nginx_config` e `nginx_ui_data`
  com escrita para importar os certificados emitidos no painel Bindnet
  para a lista do `nginx-ui`. Esses volumes **não** são gerenciados pelo
  `docker-compose.yml` (são `external: true`) e precisam existir antes
  do primeiro `docker compose up` (ver README).

## Serviço `gateway` (`services/gateway/`)

Nginx puro (sem aplicação) que decide **qual serviço interno atende cada
domínio** do Bindnet. Não é exposto ao host nem à rede do hotspot: a
única porta pública continua sendo a do `nginx-ui`.

- **Cadeia**: `nginx-ui` termina o TLS e repassa `bindnet.local.com` e
  `admin.bindnet.local.com` para o `gateway` pela rede `wnet`, usando o
  alias `${STAGE:-main}.gateway.bindnet.wnet` (mesmo padrão de
  `${STAGE:-main}.nginx-ui.bindnet.wnet`). O `gateway` roteia por `Host`.
- **Roteamento**:
  - `admin.bindnet.local.com` → painel de administração (SPA na raiz);
  - `bindnet.local.com` → portal cativo diretamente na raiz;
  - `/api/` em ambos → direto ao `backend`;
  - qualquer outro `Host` → `444` (fecha a conexão). Sem esse
    `default_server`, um Host desconhecido cairia no primeiro `server`
    e receberia o painel por engano.
- **Separação no frontend**: o hostname escolhe a superfície antes do
  roteamento React. `PortalApp` e `AdminApp` são imports dinâmicos
  distintos; o domínio público não carrega as páginas administrativas.
  O pathname `/portal` não seleciona mais a aplicação e só permanece no
  gateway como redirect de compatibilidade para a raiz.
- **`X-Forwarded-For` é obrigatório em todo salto**
  (`services/gateway/bindnet-proxy.conf`, incluído em cada `location`):
  o backend descobre o MAC do dispositivo do hotspot a partir do IP de
  origem (`resolvePortalMAC`). `$proxy_add_x_forwarded_for` **acrescenta**
  o IP do salto anterior à cadeia em vez de substituir — perder isso
  faria o portal cativo deixar de reconhecer quem está conectado.
- **Redes**: fica em `wnet` (por onde o `nginx-ui` o alcança por nome) e
  em `proxy` (por onde alcança o host em `10.91.0.1`, já que `frontend`
  e `backend` rodam com `network_mode: host` e não têm nome de serviço
  Docker).
- O site `@` do `nginx-ui` vive no volume `bindnet_nginx_config`, fora
  deste repositório; há uma cópia versionada em
  `services/gateway/nginx-ui-site.conf` como referência. A UI do
  `nginx-ui` pode sobrescrevê-la.
- **Cache do `index.html` é `no-cache`; os assets são `immutable`**
  (`services/frontend/nginx.conf.template`). O `index.html` é o único
  arquivo com nome fixo e é ele que aponta para o bundle atual —
  servido sem `Cache-Control` (como estava), o browser aplica cache
  heurístico a partir do `Last-Modified` e pode **nem revalidar**,
  continuando a carregar uma versão antiga da aplicação inteira. Foi
  exatamente isso que fez `bindnet.local.com` continuar a abrir a área
  administrativa **depois** de o frontend já separar as superfícies por
  hostname: o servidor estava correto, o browser é que servia o HTML
  velho. Os arquivos em `/assets/` levam hash no nome, então podem ser
  `immutable` sem risco — qualquer mudança gera um nome novo.

## Página pública da CA no portal cativo (`bindnet.local.com/ca`)

- Rota **sem sessão**, como a raiz do portal: quem acabou de entrar na rede
  precisa instalar a CA local **antes** de navegar nos endereços
  internos sem aviso de certificado, e nessa altura não tem conta
  nenhuma no painel.
- O download reusa `GET /api/mesh/ca`, a única rota de certificado que
  já era pública por desenho — devolve só o certificado **público** da
  CA (nunca a chave privada) e já vem com
  `Content-Disposition: attachment`, por isso é um `<a href>` direto,
  sem `fetch`/blob.
- Traz instruções de instalação por sistema (Android, iOS, Windows,
  Linux) em `PortalCaInstructions.tsx`. O passo de "Definições de
  confiança em certificados" no iOS é destacado de propósito: sem ele o
  certificado fica instalado mas não é confiado, e o sintoma é
  idêntico ao de não ter instalado nada.
- O link aparece na raiz de `bindnet.local.com` **fora** do bloco que depende da
  identificação do dispositivo: instalar o certificado é útil mesmo
  quando o portal ainda não conseguiu identificar o MAC.

