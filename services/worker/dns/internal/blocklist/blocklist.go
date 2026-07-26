// Package blocklist implementa o bloqueio de conteudo por dominio do
// dns-provider: mantem em memoria (foto imutavel trocada atomicamente,
// mesma filosofia de core.Table) o mapa IP->plano publicado pelo backend
// e, por plano, os conjuntos de dominios/categorias a permitir/bloquear.
// A decisao no caminho quente e so consulta de mapa, sem I/O.
package blocklist

import (
	"strings"
	"sync/atomic"
)

type plan struct {
	defaultBlock    bool
	blockDomains    map[string]struct{}
	allowDomains    map[string]struct{}
	blockCategories []string
	allowCategories []string
}

type snapshot struct {
	categories map[string]map[string]struct{}
	plans      map[string]*plan
	planByIP   map[string]string
}

// Store guarda a foto atual, trocada pela goroutine de poll (StartPoll).
type Store struct {
	snap atomic.Pointer[snapshot]
}

func NewStore() *Store {
	s := &Store{}
	s.snap.Store(&snapshot{})
	return s
}

// Blocked decide se a consulta do dominio "name" deve ser bloqueada para
// o cliente de origem clientIP. Sem plano vinculado ao IP (ou sem match
// e default allow), devolve false e a consulta segue o fluxo normal.
// allow vence block; a politica padrao do plano decide o que nenhuma
// regra cobre.
func (s *Store) Blocked(clientIP, name string) bool {
	snap := s.snap.Load()
	if snap == nil {
		return false
	}
	planID := snap.planByIP[clientIP]
	if planID == "" {
		return false
	}
	p := snap.plans[planID]
	if p == nil {
		return false
	}
	suffixes := domainSuffixes(name)
	if len(suffixes) == 0 {
		return false
	}
	if matchAny(p.allowDomains, suffixes) || matchCategories(snap, p.allowCategories, suffixes) {
		return false
	}
	if matchAny(p.blockDomains, suffixes) || matchCategories(snap, p.blockCategories, suffixes) {
		return true
	}
	return p.defaultBlock
}

// domainSuffixes devolve o dominio e todos os seus sufixos, do mais
// especifico ao mais generico ("a.b.example.com" ->
// ["a.b.example.com","b.example.com","example.com","com"]) - bloquear
// "example.com" cobre qualquer subdominio.
func domainSuffixes(name string) []string {
	name = strings.TrimSuffix(strings.ToLower(name), ".")
	if name == "" {
		return nil
	}
	labels := strings.Split(name, ".")
	out := make([]string, 0, len(labels))
	for i := 0; i < len(labels); i++ {
		out = append(out, strings.Join(labels[i:], "."))
	}
	return out
}

func matchAny(set map[string]struct{}, suffixes []string) bool {
	if len(set) == 0 {
		return false
	}
	for _, suffix := range suffixes {
		if _, ok := set[suffix]; ok {
			return true
		}
	}
	return false
}

func matchCategories(snap *snapshot, categories []string, suffixes []string) bool {
	for _, slug := range categories {
		if matchAny(snap.categories[slug], suffixes) {
			return true
		}
	}
	return false
}
