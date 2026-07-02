package main

import (
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
	domainZones    map[string]bool
	nginxHosts     map[string]bool
	nginxZones     map[string]bool
	dockerGateway  string
	hotspotGateway string
	db             *sql.DB
	cache          *redis.Client
	routes         *routeTable
	nodeName       string
	peers          []string
	discovered     *peerRegistry
}

// newHandler devolve o handler dns.HandlerFunc para uma view especifica -
// cada listener (ver main.go) usa sua propria instancia, entao o handler
// ja sabe, por closure, qual resposta fixa dar para containers/hotspot e
// so precisa de logica extra (Postgres+Redis) para a view do host.
func newHandler(cfg *dnsConfig, v view) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		if len(r.Question) != 1 {
			forwardVia(w, r, upstreamServers)
			return
		}
		question := r.Question[0]
		name := strings.ToLower(question.Name)

		zone, kind, nextHop := zoneFor(name, cfg)
		switch kind {
		case zoneNone:
			forwardVia(w, r, upstreamServers)
			return
		case zoneMeshUnknown:
			msg := new(dns.Msg)
			msg.SetReply(r)
			msg.Rcode = dns.RcodeNameError
			_ = w.WriteMsg(msg)
			return
		case zoneRemote:
			forwardVia(w, r, []string{dnsUpstreamForNextHop(nextHop)})
			return
		}

		msg := new(dns.Msg)
		msg.SetReply(r)
		msg.Authoritative = true

		switch question.Qtype {
		case dns.TypeA:
			ip, err := answerIPFor(cfg, v, kind, name)
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

func dnsUpstreamForNextHop(nextHop string) string {
	if host, _, err := net.SplitHostPort(nextHop); err == nil {
		return net.JoinHostPort(host, "53")
	}
	return net.JoinHostPort(nextHop, "53")
}

// forwardVia encaminha a consulta para a lista de upstreams informada,
// usando o primeiro que responder - mesmo comportamento do antigo
// "forward . 8.8.8.8 1.1.1.1" do Corefile quando upstreams e
// upstreamServers (nomes fora de qualquer zona conhecida), ou um proxy
// de um unico salto quando upstreams e o proximo salto da malha
// (zoneRemote, ver zones.go).
func forwardVia(w dns.ResponseWriter, r *dns.Msg, upstreams []string) {
	client := &dns.Client{Timeout: 3 * time.Second}
	var lastErr error
	for _, upstream := range upstreams {
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
		log.Printf("[dns-provider] erro ao encaminhar consulta para %v: %v", upstreams, lastErr)
	}
}

func serve(addr string, handler dns.HandlerFunc) error {
	mux := dns.NewServeMux()
	mux.HandleFunc(".", handler)
	server := &dns.Server{Addr: addr, Net: "udp", Handler: mux}
	log.Printf("[dns-provider] escutando em %s", addr)
	return server.ListenAndServe()
}
