FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /out/worker .

FROM alpine:3.22
RUN apk add --no-cache docker-cli docker-cli-compose networkmanager iproute2 iw
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
