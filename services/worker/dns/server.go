package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
)

// view identifica por qual IP de bind a consulta chegou - como cada view
// e um socket UDP separado (ver main.go:serve), o bind em si ja diz de
// onde a consulta veio, sem precisar inspecionar o IP remoto do cliente:
// containers so alcancam o host via DOCKER_HOST_GATEWAY, clientes do
// hotspot so alcancam via HOTSPOT_GATEWAY, e o proprio host consulta via
// 127.0.0.1 (resolv.conf/systemd-resolved apontam pra ai).
type view int

const (
	viewHost view = iota
	viewContainer
	viewHotspot
)

var upstreamServers = []string{"8.8.8.8:53", "1.1.1.1:53"}

type dnsConfig struct {
	tlds           map[string]bool
	dockerGateway  string
	hotspotGateway string
	db             *sql.DB
	cache          *redis.Client
}

// newHandler devolve o handler dns.HandlerFunc para uma view especifica -
// cada listener (ver main.go) usa sua propria instancia, entao o handler
// ja sabe, por closure, qual resposta fixa dar para containers/hotspot e
// so precisa de logica extra (Postgres+Redis) para a view do host.
func newHandler(cfg *dnsConfig, v view) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		if len(r.Question) != 1 {
			forward(w, r)
			return
		}
		question := r.Question[0]
		name := strings.ToLower(question.Name)

		if !matchesLocalTLD(name, cfg.tlds) {
			forward(w, r)
			return
		}

		msg := new(dns.Msg)
		msg.SetReply(r)
		msg.Authoritative = true

		switch question.Qtype {
		case dns.TypeA:
			ip, err := answerIPFor(cfg, v, name)
			if err != nil {
				log.Printf("[dns-provider] erro ao resolver %s: %v", name, err)
				msg.Rcode = dns.RcodeServerFailure
				_ = w.WriteMsg(msg)
				return
			}
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: question.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600},
				A:   ip,
			})
		case dns.TypeSOA:
			zone := zoneFor(name, cfg.tlds)
			msg.Answer = append(msg.Answer, &dns.SOA{
				Hdr:     dns.RR_Header{Name: dns.Fqdn(zone), Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 3600},
				Ns:      "ns." + dns.Fqdn(zone),
				Mbox:    "hostmaster." + dns.Fqdn(zone),
				Serial:  1,
				Refresh: 7200,
				Retry:   3600,
				Expire:  86400,
				Minttl:  3600,
			})
		default:
			// AAAA/ANY/etc no zona local: NXDOMAIN, ver RULE.md - o stack
			// nao tem IPv6 nem faz split-horizon multi-tipo para esses
			// dominios.
			msg.Rcode = dns.RcodeNameError
		}

		_ = w.WriteMsg(msg)
	}
}

// answerIPFor resolve o IP de resposta conforme a view: container/hotspot
// sao sempre o mesmo IP fixo (o proprio gateway); host aloca/busca um IP
// de loopback persistente e unico por hostname (Redis primeiro, Postgres
// no cache miss).
func answerIPFor(cfg *dnsConfig, v view, name string) (net.IP, error) {
	switch v {
	case viewContainer:
		return net.ParseIP(cfg.dockerGateway), nil
	case viewHotspot:
		return net.ParseIP(cfg.hotspotGateway), nil
	default:
		return loopbackIPFor(cfg, name)
	}
}

func loopbackIPFor(cfg *dnsConfig, name string) (net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if offset, ok := cachedOffset(ctx, cfg.cache, name); ok {
		return offsetToLoopback(offset), nil
	}

	offset, err := getOrAllocateOffset(ctx, cfg.db, name)
	if err != nil {
		return nil, err
	}
	storeOffset(ctx, cfg.cache, name, offset)
	return offsetToLoopback(offset), nil
}

// offsetToLoopback converte um offset (sequence do Postgres, comeca em 2)
// num IP dentro de 127.0.0.0/8 - os 24 bits menos significativos do
// offset viram os tres ultimos octetos do IP.
func offsetToLoopback(offset int64) net.IP {
	b := uint32(offset) & 0xFFFFFF
	return net.IPv4(127, byte(b>>16), byte(b>>8), byte(b))
}

func matchesLocalTLD(fqdn string, tlds map[string]bool) bool {
	labels := dns.SplitDomainName(fqdn)
	if len(labels) == 0 {
		return false
	}
	return tlds[labels[len(labels)-1]]
}

func zoneFor(fqdn string, tlds map[string]bool) string {
	labels := dns.SplitDomainName(fqdn)
	if len(labels) == 0 {
		return fqdn
	}
	return labels[len(labels)-1] + "."
}

// forward encaminha qualquer consulta fora dos TLDs locais para os
// resolvedores publicos - mesmo comportamento do antigo "forward . 8.8.8.8
// 1.1.1.1" do Corefile.
func forward(w dns.ResponseWriter, r *dns.Msg) {
	client := &dns.Client{Timeout: 3 * time.Second}
	var lastErr error
	for _, upstream := range upstreamServers {
		resp, _, err := client.Exchange(r, upstream)
		if err != nil {
			lastErr = err
			continue
		}
		_ = w.WriteMsg(resp)
		return
	}
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Rcode = dns.RcodeServerFailure
	_ = w.WriteMsg(msg)
	if lastErr != nil {
		log.Printf("[dns-provider] erro ao encaminhar consulta upstream: %v", lastErr)
	}
}

func serve(addr string, handler dns.HandlerFunc) error {
	mux := dns.NewServeMux()
	mux.HandleFunc(".", handler)
	server := &dns.Server{Addr: addr, Net: "udp", Handler: mux}
	log.Printf("[dns-provider] escutando em %s", addr)
	return server.ListenAndServe()
}
