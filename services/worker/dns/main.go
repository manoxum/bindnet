// Comando dns-provider e um servidor DNS split-horizon: para os TLDs
// locais (DNS_LOCAL_TLDS) responde de forma diferente conforme o IP em
// que a consulta chegou (host/container/hotspot, ver server.go:view) e
// encaminha qualquer outro dominio para DNS publico. Substitui o antigo
// wrapper baseado em CoreDNS+Corefile por um binario Go proprio, unica
// forma pratica de implementar as tres views + alocacao persistente de
// IP de loopback por hostname (Postgres, com cache Redis).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.Println("[dns-provider] iniciando servidor DNS split-horizon")

	tlds, err := parseTLDs(getenv("DNS_LOCAL_TLDS", "local,test,example"))
	if err != nil {
		log.Fatalf("[dns-provider] %v", err)
	}
	log.Printf("[dns-provider] TLDs locais resolvidos: %v", tldNames(tlds))

	dockerGateway := os.Getenv("DOCKER_HOST_GATEWAY")
	hotspotGateway := getenv("HOTSPOT_GATEWAY", "192.168.12.1")

	timeout := 90 * time.Second
	if raw := os.Getenv("COREDNS_WAIT_TIMEOUT"); raw != "" {
		if seconds, err := time.ParseDuration(raw + "s"); err == nil {
			timeout = seconds
		}
	}
	requiredIPs := []string{"127.0.0.1", dockerGateway}
	log.Printf("[dns-provider] aguardando IPs locais necessarios: %v", requiredIPs)
	if err := waitForIPs(requiredIPs, timeout); err != nil {
		log.Fatalf("[dns-provider] %v", err)
	}

	db, err := openPostgres()
	if err != nil {
		log.Fatalf("[dns-provider] erro ao conectar no Postgres: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cache, err := openRedis(ctx)
	cancel()
	if err != nil {
		log.Fatalf("[dns-provider] erro ao conectar no Redis: %v", err)
	}

	hydrateCtx, hydrateCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := hydrateCache(hydrateCtx, db, cache); err != nil {
		log.Printf("[dns-provider] aviso: falha ao hidratar cache Redis a partir do Postgres: %v", err)
	}
	hydrateCancel()

	cfg := &dnsConfig{
		tlds:           tlds,
		dockerGateway:  dockerGateway,
		hotspotGateway: hotspotGateway,
		db:             db,
		cache:          cache,
	}

	errCh := make(chan error, 2)
	go func() { errCh <- serve("127.0.0.1:53", newHandler(cfg, viewHost)) }()
	if dockerGateway != "" {
		go func() { errCh <- serve(dockerGateway+":53", newHandler(cfg, viewContainer)) }()
	}
	go serveHotspotWhenAvailable(hotspotGateway, timeout, newHandler(cfg, viewHotspot))

	log.Fatalf("[dns-provider] erro no servidor: %v", <-errCh)
}

func serveHotspotWhenAvailable(hotspotGateway string, timeout time.Duration, handler dns.HandlerFunc) {
	for {
		log.Printf("[dns-provider] aguardando IP do hotspot para abrir DNS em %s:53", hotspotGateway)
		if err := waitForIPs([]string{hotspotGateway}, timeout); err != nil {
			log.Printf("[dns-provider] aviso: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if err := serve(hotspotGateway+":53", handler); err != nil {
			log.Printf("[dns-provider] aviso: DNS do hotspot encerrou: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func getenv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// parseTLDs segue as mesmas regras do antigo docker-entrypoint.sh:
// minusculo, sem "." inicial, validado [a-z0-9-] sem hifen nas pontas,
// duplicatas ignoradas, pelo menos um TLD valido exigido.
func parseTLDs(raw string) (map[string]bool, error) {
	tlds := map[string]bool{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		tld := strings.ToLower(strings.TrimPrefix(part, "."))
		if tld == "" {
			continue
		}
		if !isValidTLD(tld) {
			return nil, fmt.Errorf("DNS_LOCAL_TLDS contem TLD invalido: %s", tld)
		}
		tlds[tld] = true
	}
	if len(tlds) == 0 {
		return nil, fmt.Errorf("DNS_LOCAL_TLDS deve conter pelo menos um TLD local")
	}
	return tlds, nil
}

func isValidTLD(tld string) bool {
	if strings.HasPrefix(tld, "-") || strings.HasSuffix(tld, "-") {
		return false
	}
	for _, r := range tld {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func tldNames(tlds map[string]bool) []string {
	names := make([]string, 0, len(tlds))
	for tld := range tlds {
		names = append(names, tld)
	}
	return names
}
