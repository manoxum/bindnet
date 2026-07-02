package main

import (
	"sync"
	"time"
)

// discoveredRoute e uma linha da tabela de descoberta: para um dominio
// remoto conhecido, guarda quem e o dono final ("owner"), por onde
// encaminhar a consulta ("nextHop"), a distancia ate o dono e de onde
// esta rota foi aprendida - ver RULE.md, secao de discover mode.
type discoveredRoute struct {
	Domain   string
	Owner    string
	NextHop  string
	Distance int
	Source   string
	State    string // "ok" | "stale"
	LastSeen time.Time
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

// discoveredPeer e um servidor Bindnet vizinho encontrado por
// broadcast/multicast UDP na mesma rede local (ver
// discover_broadcast.go) - diferente de cfg.peers (DISCOVER_PEERS),
// que e configurado manualmente e serve tambem para vizinhos fora da
// rede local (broadcast/multicast nao atravessa roteadores).
type discoveredPeer struct {
	Address  string
	NodeName string
	LastSeen time.Time
}

// peerRegistry e o snapshot em memoria dos vizinhos auto-descobertos -
// mesma logica de routeTable (mutex-guarded, sem I/O no caminho
// quente); quem persiste no Postgres para o painel e quem expira
// entradas velhas e discover_broadcast.go.
type peerRegistry struct {
	mu    sync.RWMutex
	peers map[string]discoveredPeer
}

func newPeerRegistry() *peerRegistry {
	return &peerRegistry{peers: map[string]discoveredPeer{}}
}

func (r *peerRegistry) upsert(address, nodeName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peers[address] = discoveredPeer{Address: address, NodeName: nodeName, LastSeen: time.Now()}
}

// addresses devolve so os vizinhos vistos ha menos de "maxAge" - um no
// que parou de anunciar sai da lista de polling rapido, mesmo que sua
// ultima entrada continue visivel no painel (isso e responsabilidade
// do registro persistido no Postgres, nao deste snapshot).
func (r *peerRegistry) addresses(maxAge time.Duration) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var addresses []string
	for _, peer := range r.peers {
		if time.Since(peer.LastSeen) <= maxAge {
			addresses = append(addresses, peer.Address)
		}
	}
	return addresses
}
