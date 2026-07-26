package store

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// ContentCategory e uma categoria do catalogo. SourceURLs (uma URL por
// linha) sao blocklists publicas em formato hosts; categorias sem
// SourceURLs sao embutidas (dominios semeados pela migration).
type ContentCategory struct {
	Slug         string     `json:"slug"`
	Name         string     `json:"name"`
	SourceURLs   string     `json:"sourceUrls"`
	Enabled      bool       `json:"enabled"`
	LastSyncedAt *time.Time `json:"lastSyncedAt"`
	DomainCount  int64      `json:"domainCount"`
}

func scanContentCategory(row interface{ Scan(...any) error }) (ContentCategory, error) {
	var c ContentCategory
	var lastSynced sql.NullTime
	err := row.Scan(&c.Slug, &c.Name, &c.SourceURLs, &c.Enabled, &lastSynced, &c.DomainCount)
	if lastSynced.Valid {
		c.LastSyncedAt = &lastSynced.Time
	}
	return c, err
}

const contentCategoryColumns = `slug, name, source_urls, enabled, last_synced_at, domain_count`

func ListContentCategories(db *sql.DB) ([]ContentCategory, error) {
	rows, err := db.Query(`SELECT ` + contentCategoryColumns + ` FROM hotspot_content_categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ContentCategory{}
	for rows.Next() {
		c, err := scanContentCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func SetContentCategoryEnabled(db *sql.DB, slug string, enabled bool) (bool, error) {
	res, err := db.Exec(`UPDATE hotspot_content_categories SET enabled = $2, updated_at = CURRENT_TIMESTAMP WHERE slug = $1`, slug, enabled)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ContentCategoriesToSync devolve as categorias habilitadas que tem pelo
// menos uma URL de origem (as embutidas nao sao buscadas).
func ContentCategoriesToSync(db *sql.DB) ([]ContentCategory, error) {
	all, err := ListContentCategories(db)
	if err != nil {
		return nil, err
	}
	out := make([]ContentCategory, 0, len(all))
	for _, c := range all {
		if c.Enabled && strings.TrimSpace(c.SourceURLs) != "" {
			out = append(out, c)
		}
	}
	return out, nil
}

// ReplaceCategoryDomains substitui, numa transacao, todos os dominios de
// uma categoria pelos novos (resultado do sync) e atualiza contador +
// last_synced_at. Insercao em lotes para nao montar um INSERT gigante de
// centenas de milhares de linhas de uma vez.
func ReplaceCategoryDomains(db *sql.DB, slug string, domains []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM hotspot_content_category_domains WHERE category_slug = $1`, slug); err != nil {
		return err
	}
	const batchSize = 500
	for start := 0; start < len(domains); start += batchSize {
		end := start + batchSize
		if end > len(domains) {
			end = len(domains)
		}
		if err := insertDomainBatch(tx, slug, domains[start:end]); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
		UPDATE hotspot_content_categories
		SET domain_count = $2, last_synced_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE slug = $1
	`, slug, int64(len(domains))); err != nil {
		return err
	}
	return tx.Commit()
}

func insertDomainBatch(tx *sql.Tx, slug string, domains []string) error {
	if len(domains) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO hotspot_content_category_domains (category_slug, domain) VALUES `)
	args := make([]any, 0, len(domains)*2+1)
	args = append(args, slug)
	for i, domain := range domains {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("($1, $")
		b.WriteString(strconv.Itoa(i + 2))
		b.WriteString(")")
		args = append(args, domain)
	}
	b.WriteString(` ON CONFLICT (category_slug, domain) DO NOTHING`)
	_, err := tx.Exec(b.String(), args...)
	return err
}
