# Site Narto Studio

O domínio local `narto.local.com` pertence à aplicação em `../narto-drama.com`. O site próprio no nginx-ui termina HTTPS e resolve dinamicamente `${STAGE:-main}.frontend.narto.wnet:8080`, acessível pela rede `wnet` já usada pelo nginx-ui. O gateway administrativo e os sites Bindnet permanecem separados. HTTP redireciona para HTTPS.

A referência versionada é `services/gateway/narto-nginx-ui-site.conf`; `scripts/narto/provision.py` mostra o plano por padrão e, com `--apply`, usa login Bindnet e token nginx-ui para emitir/reusar o certificado e instalar/ativar o site pelas APIs existentes. Não escreve diretamente em volumes, não fornece Docker socket à aplicação e não reinicia o stack. O nome do certificado é `narto.local.com`, com SANs `narto.local.com` e `*.narto.local.com`, emitido pela CA já persistida. A importação usa o fluxo normal do backend Bindnet e aparece na lista do nginx-ui.

O provisionador acrescenta a zona concreta `narto.local.com` a `DOMAINS`, preserva as demais zonas e recusa sobrescrever sites divergentes. `--apply-dns` é opcional para atualizar as rotas do resolved quando não existe uma rota abrangente. Não controla hotspot/Wi-Fi. Segredos só ficam em memória, não são gravados em .env, argumentos ou logs. Não utiliza redirects autenticados nem HTTP remoto.

Testes: `python3 -m unittest discover -s scripts/narto -p '*_test.py'`. Depois de aplicar, verificar DNS, `nginx -t`, HTTP 308, HTTPS 200 com CA confiada e certificado visível no nginx-ui. A operação real requer autorização conforme AGENTS.md.
