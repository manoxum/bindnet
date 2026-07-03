FROM golang:1.25-alpine

WORKDIR /src

RUN apk add --no-cache ca-certificates curl \
    && mkdir -p /usr/local/share/bindnet \
    && curl -fsSL https://standards-oui.ieee.org/oui/oui.csv -o /usr/local/share/bindnet/oui.csv \
    && apk del curl

ENV CGO_ENABLED=0
ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

EXPOSE 8090

ENTRYPOINT ["sh", "-c", "go build -o /tmp/backend . && exec /tmp/backend"]
