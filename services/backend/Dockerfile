FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/backend .

FROM alpine:3.22
COPY --from=build /out/backend /usr/local/bin/backend
ENTRYPOINT ["/usr/local/bin/backend"]
