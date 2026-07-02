FROM alpine:3.22

RUN apk add --no-cache \
    bash \
    curl \
    dnsmasq \
    hostapd \
    iproute2 \
    iptables \
    iw \
    kea-dhcp4 \
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

COPY entrypoint.sh channel.sh interfaces.sh /usr/local/bin/
RUN mv /usr/local/bin/entrypoint.sh /usr/local/bin/hotspot-entrypoint.sh \
    && chmod +x /usr/local/bin/hotspot-entrypoint.sh /usr/local/bin/channel.sh /usr/local/bin/interfaces.sh

ENTRYPOINT ["/usr/local/bin/hotspot-entrypoint.sh"]
