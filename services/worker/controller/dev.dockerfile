FROM golang:1.25-alpine

WORKDIR /src

RUN apk add --no-cache docker-cli docker-cli-compose networkmanager iproute2 iw ca-certificates

ENV CGO_ENABLED=0
ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

CMD ["go", "run", "."]
