FROM coredns/coredns:1.14.4 AS coredns

FROM alpine:3.22

RUN apk add --no-cache ca-certificates iproute2

COPY --from=coredns /coredns /usr/local/bin/coredns
COPY Corefile /etc/coredns/Corefile
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 53/tcp 53/udp

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["-conf", "/etc/coredns/Corefile"]
