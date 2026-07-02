package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"strconv"
	"time"
)

// multicastAddr e um endereco multicast de escopo organizacional
// (239.0.0.0/8 - RFC 2365), escolhido arbitrariamente para o anuncio
// de servidores Bindnet na rede local. Multicast (em vez de broadcast)
// evita a necessidade de SO_BROADCAST, que a stdlib do Go nao expoe
// sem depender de syscalls extras - e fica limitado a rede local por
// padrao (TTL baixo), nao atravessa roteadores. Servidores em outra
// rede continuam exigindo configuracao manual via DISCOVER_PEERS.
const multicastAddr = "239.255.42.99"

const broadcastInterval = 10 * time.Second
const peerMaxAge = 3 * broadcastInterval

type peerAnnouncement struct {
	NodeName string `json:"nodeName"`
	Port     int    `json:"port"`
}

// announcePeer publica periodicamente um anuncio deste no no grupo
// multicast local, para outros servidores Bindnet na mesma rede se
// auto-descobrirem sem configuracao manual.
func announcePeer(cfg *dnsConfig, port string) {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		log.Printf("[dns-provider] aviso: DISCOVER_PORT invalido para anuncio multicast: %v", err)
		return
	}
	addr := &net.UDPAddr{IP: net.ParseIP(multicastAddr), Port: portNum}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		log.Printf("[dns-provider] aviso: falha ao abrir anuncio multicast de descoberta: %v", err)
		return
	}
	defer conn.Close()

	payload, err := json.Marshal(peerAnnouncement{NodeName: cfg.nodeName, Port: portNum})
	if err != nil {
		log.Printf("[dns-provider] aviso: falha ao montar anuncio de descoberta: %v", err)
		return
	}

	for {
		if _, err := conn.Write(payload); err != nil {
			log.Printf("[dns-provider] aviso: falha ao publicar anuncio de descoberta: %v", err)
		}
		time.Sleep(broadcastInterval)
	}
}

// listenForPeers escuta o grupo multicast e registra qualquer servidor
// Bindnet anunciado por outro no na mesma rede - tanto no snapshot em
// memoria (usado para decidir quem consultar em pollPeers) quanto no
// Postgres (para o painel mostrar, ver services/backend/dns_peers.go).
func listenForPeers(cfg *dnsConfig, port string) {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		log.Printf("[dns-provider] aviso: DISCOVER_PORT invalido para escuta multicast: %v", err)
		return
	}
	addr := &net.UDPAddr{IP: net.ParseIP(multicastAddr), Port: portNum}
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		log.Printf("[dns-provider] aviso: falha ao escutar multicast de descoberta: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[dns-provider] escutando anuncios de servidores Bindnet em %s:%s (multicast)", multicastAddr, port)

	buf := make([]byte, 512)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		var announcement peerAnnouncement
		if err := json.Unmarshal(buf[:n], &announcement); err != nil {
			continue
		}
		if announcement.NodeName == "" || announcement.NodeName == cfg.nodeName {
			continue // ignora anuncio invalido ou o proprio eco
		}
		address := net.JoinHostPort(src.IP.String(), strconv.Itoa(announcement.Port))
		cfg.discovered.upsert(address, announcement.NodeName)
		persistDiscoveredPeer(cfg.db, address, announcement.NodeName)
	}
}

func persistDiscoveredPeer(db *sql.DB, address, nodeName string) {
	if db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx, `
		INSERT INTO discover_peers (address, node_name, source, last_seen_at)
		VALUES ($1, $2, 'broadcast', now())
		ON CONFLICT (address) DO UPDATE SET node_name = EXCLUDED.node_name, last_seen_at = EXCLUDED.last_seen_at
	`, address, nodeName)
	if err != nil {
		log.Printf("[dns-provider] aviso: falha ao persistir servidor descoberto %s: %v", address, err)
	}
}
