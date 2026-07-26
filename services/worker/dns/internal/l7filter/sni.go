// Package l7filter faz o bloqueio de conteudo na camada 7/4, complementar
// ao DNS: um proxy transparente le o SNI do ClientHello TLS (HTTPS) e o
// cabecalho Host (HTTP) e decide bloquear/permitir pela mesma blocklist
// do dns-provider (reusa blocklist.Store.Blocked). Isso pega quem burla
// o DNS (troca de resolver, acesso por IP direto) - como o DoH ja e
// bloqueado, o ECH nao inicializa e o SNI fica visivel.
package l7filter

import (
	"encoding/binary"
	"errors"
	"strings"
)

var errNoSNI = errors.New("sni ausente ou clienthello incompleto")

// parseSNI extrai o server_name (SNI) de um registro TLS ClientHello.
// Recebe os bytes ja lidos do inicio da conexao (record TLS handshake).
// Puro/sem I/O - devolve o hostname em minusculas ou erro se nao houver
// SNI ou os dados estiverem incompletos.
func parseSNI(buf []byte) (string, error) {
	// TLS record: type(1)=22 handshake, version(2), length(2), payload.
	if len(buf) < 5 || buf[0] != 0x16 {
		return "", errNoSNI
	}
	recLen := int(binary.BigEndian.Uint16(buf[3:5]))
	hs := buf[5:]
	if len(hs) < recLen {
		recLen = len(hs) // ClientHello pode caber; se truncado, tentamos assim mesmo
	}
	hs = hs[:recLen]
	// Handshake: type(1)=1 ClientHello, length(3), body.
	if len(hs) < 4 || hs[0] != 0x01 {
		return "", errNoSNI
	}
	body := hs[4:]
	// client_version(2) + random(32)
	if len(body) < 34 {
		return "", errNoSNI
	}
	p := 34
	// session_id
	if p+1 > len(body) {
		return "", errNoSNI
	}
	sidLen := int(body[p])
	p += 1 + sidLen
	// cipher_suites
	if p+2 > len(body) {
		return "", errNoSNI
	}
	csLen := int(binary.BigEndian.Uint16(body[p : p+2]))
	p += 2 + csLen
	// compression_methods
	if p+1 > len(body) {
		return "", errNoSNI
	}
	cmLen := int(body[p])
	p += 1 + cmLen
	// extensions
	if p+2 > len(body) {
		return "", errNoSNI
	}
	extLen := int(binary.BigEndian.Uint16(body[p : p+2]))
	p += 2
	end := p + extLen
	if end > len(body) {
		end = len(body)
	}
	for p+4 <= end {
		extType := binary.BigEndian.Uint16(body[p : p+2])
		extSize := int(binary.BigEndian.Uint16(body[p+2 : p+4]))
		p += 4
		if p+extSize > end {
			break
		}
		if extType == 0x0000 { // server_name
			return parseServerNameExt(body[p : p+extSize])
		}
		p += extSize
	}
	return "", errNoSNI
}

// parseServerNameExt le a extensao server_name: server_name_list(2) e,
// para cada entrada, type(1)=0 host_name + length(2) + nome.
func parseServerNameExt(ext []byte) (string, error) {
	if len(ext) < 2 {
		return "", errNoSNI
	}
	listLen := int(binary.BigEndian.Uint16(ext[0:2]))
	list := ext[2:]
	if listLen < len(list) {
		list = list[:listLen]
	}
	p := 0
	for p+3 <= len(list) {
		nameType := list[p]
		nameLen := int(binary.BigEndian.Uint16(list[p+1 : p+3]))
		p += 3
		if p+nameLen > len(list) {
			break
		}
		if nameType == 0x00 {
			return strings.ToLower(string(list[p : p+nameLen])), nil
		}
		p += nameLen
	}
	return "", errNoSNI
}
