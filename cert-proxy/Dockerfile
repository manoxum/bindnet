FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cert-proxy .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=build /out/cert-proxy /usr/local/bin/cert-proxy

EXPOSE 80 443
ENTRYPOINT ["/usr/local/bin/cert-proxy"]
