package main

import (
	"strings"
	"sync"
	"time"
)

// discoveredRoute e uma linha da tabela de descoberta: para um dominio
// remoto conhecido, guarda quem e o dono final ("owner"), por onde
// encaminhar a consulta ("nextHop"), a distancia ate o dono e de onde
// esta rota foi aprendida - ver RULE.md, secao de discover mode.
type discoveredRoute struct {
	Domain           string
	Owner            string
	OwnerFingerprint string
	NextHop          string
	Distance         int
	Source           string
	State            string // "ok" | "stale"
	LastSeen         time.Time
}

const (
	routeStateOK    = "ok"
	routeStateStale = "stale"
)

// routeTable e o snapshot em memoria consultado no caminho quente de
// resolucao DNS (zoneFor) - nunca faz I/O; quem mantem o conteudo
// atualizado e a goroutine de poll em discover_peer.go, que substitui o
// mapa inteiro a cada ciclo via replace().
type routeTable struct {
	mu     sync.RWMutex
	routes map[string]discoveredRoute
}

func newRouteTable() *routeTable {
	return &routeTable{routes: map[string]discoveredRoute{}}
}

func (t *routeTable) lookup(domain string) (discoveredRoute, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	route, ok := t.routes[domain]
	return route, ok
}

func (t *routeTable) lookupSuffix(labels []string) (string, discoveredRoute, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for i := 0; i < len(labels); i++ {
		zone := strings.Join(labels[i:], ".")
		route, ok := t.routes[zone]
		if ok {
			return zone, route, true
		}
	}
	return "", discoveredRoute{}, false
}

func (t *routeTable) snapshot() []discoveredRoute {
	t.mu.RLock()
	defer t.mu.RUnlock()
	routes := make([]discoveredRoute, 0, len(t.routes))
	for _, route := range t.routes {
		routes = append(routes, route)
	}
	return routes
}

func (t *routeTable) replace(routes map[string]discoveredRoute) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.routes = routes
}
