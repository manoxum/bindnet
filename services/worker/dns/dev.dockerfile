FROM golang:1.25-alpine

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN apk add --no-cache ca-certificates

ENV CGO_ENABLED=0
ENV GOCACHE=/go/cache
ENV GOMODCACHE=/go/pkg/mod

EXPOSE 53/udp

ENTRYPOINT ["sh", "-c", "go build -o /tmp/dns-provider . && exec /tmp/dns-provider"]
