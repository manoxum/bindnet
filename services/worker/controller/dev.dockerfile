FROM golang:1.25-alpine

WORKDIR /src

# networkmanager-cli/iptables: ver comentarios no Dockerfile de producao.
RUN apk add --no-cache docker-cli docker-cli-compose networkmanager networkmanager-cli iproute2 iptables iw ca-certificates nss-tools

ENV CGO_ENABLED=0
ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

ENTRYPOINT ["sh", "-c", "go build -o /tmp/worker . && exec /tmp/worker"]
