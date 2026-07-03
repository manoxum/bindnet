FROM golang:1.25-alpine

WORKDIR /src

# networkmanager-cli: ver comentario no Dockerfile de producao - o
# "nmcli" nao vem no pacote "networkmanager" no Alpine 3.22.
RUN apk add --no-cache docker-cli docker-cli-compose networkmanager networkmanager-cli iproute2 iw ca-certificates nss-tools

ENV CGO_ENABLED=0
ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

ENTRYPOINT ["sh", "-c", "go build -o /tmp/worker . && exec /tmp/worker"]
