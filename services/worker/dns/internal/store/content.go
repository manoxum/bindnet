package store

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

// content.go le, do Postgres, tudo o que o dns-provider precisa para o
// bloqueio de conteudo por dominio: planos, regras, dominios das
// categorias e o mapa IP->plano publicado pelo backend
// (hotspot_content_client_bindings). O schema e criado pelo
// services/migration; o backend e o dono da escrita.

type ContentPlanRow struct {
	ID            string
	DefaultAction string
}

type ContentRuleRow struct {
	PlanID string
	Kind   string
	Value  string
	Action string
}

func LoadContentPlans(ctx context.Context, db *sql.DB) ([]ContentPlanRow, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, default_action FROM hotspot_content_plans`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContentPlanRow
	for rows.Next() {
		var p ContentPlanRow
		if err := rows.Scan(&p.ID, &p.DefaultAction); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LoadContentRules devolve so as regras de dominio/categoria (as de ip
// sao resolvidas pelo firewall do worker, nao aqui).
func LoadContentRules(ctx context.Context, db *sql.DB) ([]ContentRuleRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT plan_id, kind, value, action FROM hotspot_content_rules
		WHERE kind IN ('domain','category')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContentRuleRow
	for rows.Next() {
		var r ContentRuleRow
		if err := rows.Scan(&r.PlanID, &r.Kind, &r.Value, &r.Action); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadCategoryDomains devolve, por slug, o conjunto de dominios
// materializados - SO das categorias em "slugs" (as referenciadas por
// algum plano). Carregar so o necessario evita puxar listas gigantes
// (ex.: malware com milhoes de dominios) que nenhum plano usa para a
// memoria do dns-provider. Chamada quando a assinatura de conteudo muda
// (ver blocklist.Store).
func LoadCategoryDomains(ctx context.Context, db *sql.DB, slugs []string) (map[string]map[string]struct{}, error) {
	out := map[string]map[string]struct{}{}
	if len(slugs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(slugs))
	args := make([]any, len(slugs))
	for i, slug := range slugs {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = slug
	}
	rows, err := db.QueryContext(ctx,
		`SELECT category_slug, domain FROM hotspot_content_category_domains WHERE category_slug IN (`+strings.Join(placeholders, ",")+`)`,
		args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var slug, domain string
		if err := rows.Scan(&slug, &domain); err != nil {
			return nil, err
		}
		set := out[slug]
		if set == nil {
			set = map[string]struct{}{}
			out[slug] = set
		}
		set[domain] = struct{}{}
	}
	return out, rows.Err()
}

// LoadContentBindings devolve o mapa IP->plano dos clientes ao vivo.
func LoadContentBindings(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT ip, plan_id FROM hotspot_content_client_bindings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var ip, planID string
		if err := rows.Scan(&ip, &planID); err != nil {
			return nil, err
		}
		out[ip] = planID
	}
	return out, rows.Err()
}

// LoadContentSignature devolve uma assinatura barata que muda sempre que
// planos/regras/categorias mudam - evita recarregar os dominios das
// categorias (caro) a cada poll quando nada mudou.
func LoadContentSignature(ctx context.Context, db *sql.DB) (string, error) {
	var sig string
	err := db.QueryRowContext(ctx, `
		SELECT
			(SELECT count(*) FROM hotspot_content_plans)::text || ':' ||
			(SELECT coalesce(max(updated_at)::text,'') FROM hotspot_content_plans) || ':' ||
			(SELECT count(*) FROM hotspot_content_rules)::text || ':' ||
			(SELECT coalesce(sum(domain_count),0)::text FROM hotspot_content_categories) || ':' ||
			(SELECT coalesce(max(updated_at)::text,'') FROM hotspot_content_categories)
	`).Scan(&sig)
	return sig, err
}
