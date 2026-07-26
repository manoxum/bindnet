package l7filter

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
)

// soOriginalDst e a opcao getsockopt SO_ORIGINAL_DST do Linux (nivel
// IPPROTO_IP) que devolve o destino ANTES do REDIRECT do iptables.
const soOriginalDst = 80

// originalDst devolve o destino original (IP:porta) de uma conexao TCP
// que passou por REDIRECT - o proxy transparente precisa dele para
// encaminhar as conexoes PERMITIDAS ao servidor real (o REDIRECT
// reescreveu o destino para o proprio proxy). Le a sockaddr_in via
// getsockopt SO_ORIGINAL_DST (idioma padrao de proxy transparente em Go).
func originalDst(conn *net.TCPConn) (string, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return "", err
	}
	var addr string
	var opErr error
	ctlErr := raw.Control(func(fd uintptr) {
		mreq, e := syscall.GetsockoptIPv6Mreq(int(fd), syscall.IPPROTO_IP, soOriginalDst)
		if e != nil {
			opErr = e
			return
		}
		// sockaddr_in: family(2) | port(2, big-endian) | addr(4) | ...
		port := binary.BigEndian.Uint16(mreq.Multiaddr[2:4])
		ip := net.IPv4(mreq.Multiaddr[4], mreq.Multiaddr[5], mreq.Multiaddr[6], mreq.Multiaddr[7])
		addr = fmt.Sprintf("%s:%d", ip.String(), port)
	})
	if ctlErr != nil {
		return "", ctlErr
	}
	if opErr != nil {
		return "", opErr
	}
	return addr, nil
}
