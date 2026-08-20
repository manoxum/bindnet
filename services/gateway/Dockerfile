# Gateway de roteamento interno do Bindnet - nginx puro, sem build de
# aplicacao. Mesmo padrao do services/frontend: a config vai como
# template em /etc/nginx/templates e a imagem oficial do nginx roda
# envsubst nela no boot, entao FRONTEND_PORT/BACKEND_PORT vem do
# ambiente (docker-compose.services.yml) sem precisar reconstruir a
# imagem quando mudam.
FROM nginx:alpine

COPY nginx.conf.template /etc/nginx/templates/default.conf.template
# Nao vai em /etc/nginx/templates: os arquivos de la sao processados
# por envsubst e copiados para conf.d, onde o nginx os carregaria como
# se fossem uma config completa - este e um fragmento para "include".
COPY bindnet-proxy.conf /etc/nginx/bindnet-proxy.conf

ENV FRONTEND_PORT=9090
ENV BACKEND_PORT=8090
