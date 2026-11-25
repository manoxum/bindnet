# Job one-shot do deploy: deixa o Docker daemon do host alinhado com o
# DNS e os certificados do Bindnet, para que containers de QUALQUER
# projeto (a) resolvam os dominios gerenciados pelo Bindnet pela view
# container e (b) confiem na CA local do Bindnet. Ver configure.sh.
FROM alpine:3.20
RUN apk add --no-cache jq curl openssl
COPY configure.sh /usr/local/bin/configure.sh
COPY bindnet-runc /opt/bindnet-runc
RUN chmod +x /usr/local/bin/configure.sh
ENTRYPOINT ["/usr/local/bin/configure.sh"]
