FROM alpine:3.22

# grep: substitui o applet grep do BusyBox pelo GNU grep. O create_ap
# usa "grep -E '{.* managed.* AP.*}'" (regex com "{" solto) pra checar
# se o adaptador suporta AP+estacao simultaneos via "iw phy info" - o
# BusyBox grep trata "{" sem bound valido como erro de sintaxe e falha
# sempre, fazendo o create_ap concluir (errado, em qualquer adaptador)
# que o modo concorrente nao e suportado. GNU grep trata "{" solto como
# literal (extensao GNU), igual o create_ap espera. Mesma correcao do
# Dockerfile de producao - ver comentario la para detalhes.
RUN apk add --no-cache \
    bash \
    curl \
    dnsmasq \
    grep \
    hostapd \
    iproute2 \
    iptables \
    iw \
    kea-dhcp4 \
    postgresql-client \
    procps \
    util-linux \
    wireless-tools

COPY patch-create-ap.sh /tmp/patch-create-ap.sh
RUN curl -fsSL https://raw.githubusercontent.com/oblique/create_ap/master/create_ap \
    -o /usr/local/bin/create_ap \
    && sed -i 's/24\[0-9\]\[0-9\] MHz/24[0-9][0-9]\\(\\.0\\)\\? MHz/' /usr/local/bin/create_ap \
    && sh /tmp/patch-create-ap.sh \
    && rm /tmp/patch-create-ap.sh \
    && chmod +x /usr/local/bin/create_ap

COPY entrypoint.sh channel.sh interfaces.sh regulatory.sh /usr/local/bin/
RUN mv /usr/local/bin/entrypoint.sh /usr/local/bin/hotspot-entrypoint.sh \
    && chmod +x /usr/local/bin/hotspot-entrypoint.sh /usr/local/bin/channel.sh /usr/local/bin/interfaces.sh /usr/local/bin/regulatory.sh

ENTRYPOINT ["/usr/local/bin/hotspot-entrypoint.sh"]
