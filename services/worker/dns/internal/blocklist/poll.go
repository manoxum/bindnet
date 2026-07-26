package blocklist

import (
	"context"
	"database/sql"
	"log"
	"time"

	"bindnet/dns-provider/internal/store"
)

// dohBypassCategory e a categoria (semeada pela migration) com os
// endpoints de DNS criptografado (DoH/DoT). Bloqueada implicitamente por
// todo plano - ver buildPlans.
const dohBypassCategory = "doh_bypass"

// StartPoll mantem a foto de conteudo atualizada. O mapa IP->plano
// (pequeno, muda a cada cliente que conecta) e recarregado todo ciclo; os
// dominios das categorias (grandes) so sao recarregados quando a
// assinatura de conteudo muda - senao seria caro reler centenas de
// milhares de linhas a cada poll.
func (s *Store) StartPoll(db *sql.DB, interval time.Duration) {
	var lastSig string
	var categories map[string]map[string]struct{}
	var plans map[string]*plan
	for {
		if err := s.pollOnce(db, &lastSig, &categories, &plans); err != nil {
			log.Printf("[dns-provider] poll de conteudo falhou: %v", err)
		}
		time.Sleep(interval)
	}
}

func (s *Store) pollOnce(db *sql.DB, lastSig *string, categories *map[string]map[string]struct{}, plans *map[string]*plan) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sig, err := store.LoadContentSignature(ctx, db)
	if err != nil {
		return err
	}
	if sig != *lastSig || *plans == nil {
		newPlans, neededSlugs, err := buildPlans(ctx, db)
		if err != nil {
			return err
		}
		newCategories, err := store.LoadCategoryDomains(ctx, db, neededSlugs)
		if err != nil {
			return err
		}
		*categories = newCategories
		*plans = newPlans
		*lastSig = sig
		var totalDomains int
		for _, set := range newCategories {
			totalDomains += len(set)
		}
		log.Printf("[dns-provider] conteudo recarregado: %d planos, %d categorias, %d dominios de categoria", len(newPlans), len(newCategories), totalDomains)
	}

	bindings, err := store.LoadContentBindings(ctx, db)
	if err != nil {
		return err
	}
	s.snap.Store(&snapshot{categories: *categories, plans: *plans, planByIP: bindings})
	return nil
}

// buildPlans monta os planos com seus conjuntos de dominios/categorias a
// partir das linhas de plano + regra. Devolve tambem o conjunto de slugs
// de categoria referenciados por alguma regra - so esses precisam ter os
// dominios carregados em memoria (ver LoadCategoryDomains).
func buildPlans(ctx context.Context, db *sql.DB) (map[string]*plan, []string, error) {
	rows, err := store.LoadContentPlans(ctx, db)
	if err != nil {
		return nil, nil, err
	}
	plans := make(map[string]*plan, len(rows))
	for _, row := range rows {
		plans[row.ID] = &plan{
			defaultBlock: row.DefaultAction == "block",
			blockDomains: map[string]struct{}{},
			allowDomains: map[string]struct{}{},
		}
	}

	rules, err := store.LoadContentRules(ctx, db)
	if err != nil {
		return nil, nil, err
	}
	neededSlugs := map[string]struct{}{}
	for _, rule := range rules {
		p := plans[rule.PlanID]
		if p == nil {
			continue
		}
		switch rule.Kind {
		case "domain":
			if rule.Action == "allow" {
				p.allowDomains[rule.Value] = struct{}{}
			} else {
				p.blockDomains[rule.Value] = struct{}{}
			}
		case "category":
			if rule.Action == "allow" {
				p.allowCategories = append(p.allowCategories, rule.Value)
			} else {
				p.blockCategories = append(p.blockCategories, rule.Value)
			}
			neededSlugs[rule.Value] = struct{}{}
		}
	}
	// Todo plano bloqueia IMPLICITAMENTE a categoria doh_bypass (endpoints
	// de DNS criptografado: dns.google, chrome.cloudflare-dns.com, etc.).
	// Sem isso, o navegador resolve o hostname do servidor DoH pelo nosso
	// resolver, faz DoH para um IP que o firewall nao conhece e escapa de
	// TODO o bloqueio (causa real do "xvideos continua abrindo"). O
	// operador pode reverter adicionando uma regra ALLOW da categoria
	// doh_bypass (allow vence block).
	if len(plans) > 0 {
		for _, p := range plans {
			p.blockCategories = append(p.blockCategories, dohBypassCategory)
		}
		neededSlugs[dohBypassCategory] = struct{}{}
	}
	slugs := make([]string, 0, len(neededSlugs))
	for slug := range neededSlugs {
		slugs = append(slugs, slug)
	}
	return plans, slugs, nil
}
