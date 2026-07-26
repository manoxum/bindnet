package l7filter

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

// Blocker e o que o proxy precisa da blocklist: decidir se um host esta
// bloqueado para um cliente (satisfeito por blocklist.Store.Blocked).
type Blocker interface {
	Blocked(clientIP, host string) bool
}

const (
	dialTimeout = 10 * time.Second
	readTimeout = 5 * time.Second
)

// StartSNIProxy sobe o proxy transparente de HTTPS: o worker redireciona
// o 443 dos clientes com plano para "addr"; aqui lemos o SNI do
// ClientHello e, se bloqueado, resetamos (fechamos); senao encaminhamos
// (splice TCP puro, sem terminar o TLS - nao ha problema de certificado).
func StartSNIProxy(addr string, blocks Blocker) {
	go serve(addr, "sni", func(c *net.TCPConn) { handleSNI(c, blocks) })
}

// StartHTTPProxy sobe o proxy transparente de HTTP: le o Host; se
// bloqueado, serve a pagina de bloqueio; senao encaminha ao destino real.
func StartHTTPProxy(addr string, blocks Blocker) {
	go serve(addr, "http", func(c *net.TCPConn) { handleHTTP(c, blocks) })
}

func serve(addr, name string, handle func(*net.TCPConn)) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[dns-provider] proxy L7 %s nao subiu em %s: %v", name, addr, err)
		return
	}
	log.Printf("[dns-provider] filtro de conteudo L7 (%s) escutando em %s", name, addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		tcp, ok := conn.(*net.TCPConn)
		if !ok {
			conn.Close()
			continue
		}
		go handle(tcp)
	}
}

func handleSNI(client *net.TCPConn, blocks Blocker) {
	defer client.Close()
	client.SetReadDeadline(time.Now().Add(readTimeout))
	hello, err := readTLSRecord(client)
	client.SetReadDeadline(time.Time{})
	if err != nil && len(hello) == 0 {
		return
	}
	ip := remoteIP(client)
	if sni, e := parseSNI(hello); e == nil && sni != "" && blocks.Blocked(ip, sni) {
		log.Printf("[dns-provider] L7 BLOCK sni %s %s", ip, sni)
		return // reset: fecha sem encaminhar
	}
	forward(client, hello)
}

func handleHTTP(client *net.TCPConn, blocks Blocker) {
	defer client.Close()
	client.SetReadDeadline(time.Now().Add(readTimeout))
	buf := make([]byte, 8192)
	n, _ := client.Read(buf)
	client.SetReadDeadline(time.Time{})
	if n == 0 {
		return
	}
	head := buf[:n]
	ip := remoteIP(client)
	if host := parseHTTPHost(head); host != "" && blocks.Blocked(ip, host) {
		log.Printf("[dns-provider] L7 BLOCK http %s %s", ip, host)
		_, _ = client.Write([]byte(blockPageResponse))
		return
	}
	forward(client, head)
}

// forward encaminha a conexao para o destino ORIGINAL (pre-REDIRECT),
// reenviando os bytes ja lidos e fazendo splice bidirecional.
func forward(client *net.TCPConn, head []byte) {
	dst, err := originalDst(client)
	if err != nil {
		return
	}
	upstream, err := net.DialTimeout("tcp", dst, dialTimeout)
	if err != nil {
		return
	}
	defer upstream.Close()
	if len(head) > 0 {
		if _, err := upstream.Write(head); err != nil {
			return
		}
	}
	splice(client, upstream)
}

func splice(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
}

// readTLSRecord le um registro TLS completo (header de 5 bytes + payload)
// para garantir que o ClientHello inteiro esta no buffer antes do parse
// de SNI. Se nao for handshake TLS, devolve o que leu para ser
// reencaminhado (tratado como permitido).
func readTLSRecord(conn net.Conn) ([]byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return header[:0], err
	}
	if header[0] != 0x16 {
		return header, errNoSNI
	}
	recLen := int(binary.BigEndian.Uint16(header[3:5]))
	if recLen <= 0 || recLen > 16384 {
		return header, errNoSNI
	}
	body := make([]byte, recLen)
	if _, err := io.ReadFull(conn, body); err != nil {
		return append(header, body...), err
	}
	return append(header, body...), nil
}

func parseHTTPHost(buf []byte) string {
	for _, line := range strings.Split(string(buf), "\r\n") {
		if len(line) >= 5 && strings.EqualFold(line[:5], "host:") {
			h := strings.TrimSpace(line[5:])
			if i := strings.IndexByte(h, ':'); i >= 0 {
				h = h[:i]
			}
			return strings.ToLower(h)
		}
	}
	return ""
}

func remoteIP(conn net.Conn) string {
	if host, _, err := net.SplitHostPort(conn.RemoteAddr().String()); err == nil {
		return host
	}
	return ""
}
