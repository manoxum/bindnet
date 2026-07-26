// hotspot_content_sync.go baixa as blocklists publicas das categorias
// (formato hosts) e materializa os dominios em
// hotspot_content_category_domains. Roda como loop de fundo (cadencia
// longa) e sob gatilho manual (POST .../categories/{slug}/sync). O
// backend tem egress de internet e Postgres; nao adiciona dependencia
// nova (net/http + bufio da stdlib).
package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bufio"
	"database/sql"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// contentSyncMinInterval evita re-baixar listas gigantes a cada restart:
// uma categoria so e re-sincronizada pelo loop se nunca foi ou se o
// ultimo sync passou desse intervalo.
const contentSyncMinInterval = 20 * time.Hour

// StartContentBlocklistSyncLoop sincroniza as categorias com fonte
// publica de tempos em tempos - primeira passada logo apos o boot (com
// pequeno atraso para nao competir com a subida do resto), depois na
// cadencia informada.
func StartContentBlocklistSyncLoop(db *sql.DB, interval time.Duration) {
	time.Sleep(30 * time.Second)
	syncStaleContentCategories(db)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		syncStaleContentCategories(db)
	}
}

func syncStaleContentCategories(db *sql.DB) {
	cats, err := store.ContentCategoriesToSync(db)
	if err != nil {
		log.Printf("[backend] sync de conteudo: falha ao listar categorias: %v", err)
		return
	}
	for _, cat := range cats {
		if cat.LastSyncedAt != nil && time.Since(*cat.LastSyncedAt) < contentSyncMinInterval {
			continue
		}
		if err := syncContentCategory(db, cat); err != nil {
			log.Printf("[backend] sync da categoria %s falhou: %v", cat.Slug, err)
		}
	}
}

// syncContentCategory baixa todas as URLs de origem da categoria, junta
// e deduplica os dominios e substitui a materializacao no Postgres.
func syncContentCategory(db *sql.DB, cat store.ContentCategory) error {
	seen := map[string]struct{}{}
	var domains []string
	for _, url := range splitSourceURLs(cat.SourceURLs) {
		fetched, err := fetchHostsDomains(url)
		if err != nil {
			return err
		}
		for _, d := range fetched {
			if _, ok := seen[d]; ok {
				continue
			}
			seen[d] = struct{}{}
			domains = append(domains, d)
		}
	}
	if err := store.ReplaceCategoryDomains(db, cat.Slug, domains); err != nil {
		return err
	}
	log.Printf("[backend] categoria de conteudo %s sincronizada: %d dominio(s)", cat.Slug, len(domains))
	return nil
}

func splitSourceURLs(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' || r == ' ' })
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// fetchHostsDomains baixa e faz o parse de uma lista em formato hosts
// ("0.0.0.0 dominio" / "127.0.0.1 dominio") ou lista simples de
// dominios, um por linha. Ignora comentarios (#), IPs, localhost e
// entradas invalidas.
func fetchHostsDomains(url string) ([]string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &fetchError{url: url, status: resp.StatusCode}
	}

	var domains []string
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if domain := parseHostsLine(scanner.Text()); domain != "" {
			domains = append(domains, domain)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return domains, nil
}

func parseHostsLine(line string) string {
	line = strings.TrimSpace(line)
	if idx := strings.IndexByte(line, '#'); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	if line == "" {
		return ""
	}
	fields := strings.Fields(line)
	// "0.0.0.0 dominio" / "127.0.0.1 dominio": pega o segundo campo.
	// Lista simples: um dominio no primeiro campo.
	candidate := fields[0]
	if len(fields) >= 2 && net.ParseIP(fields[0]) != nil {
		candidate = fields[1]
	}
	candidate = strings.ToLower(strings.TrimSuffix(candidate, "."))
	if candidate == "" || candidate == "localhost" || net.ParseIP(candidate) != nil {
		return ""
	}
	if strings.ContainsAny(candidate, " /") || !strings.Contains(candidate, ".") {
		return ""
	}
	return candidate
}

type fetchError struct {
	url    string
	status int
}

func (e *fetchError) Error() string {
	return "falha ao baixar " + e.url + ": HTTP " + http.StatusText(e.status)
}
