FROM golang:1.25-alpine

WORKDIR /src

RUN apk add --no-cache ca-certificates

ENV CGO_ENABLED=0
ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

EXPOSE 53/udp

CMD ["go", "run", "."]
