package main

import (
	"context"
	"log"
	"time"
)

const (
	pollInterval = 15 * time.Second
	maxHops      = 16
	staleAfter   = 3 * pollInterval
)

// pollPeers roda para sempre, consultando cada DISCOVER_PEERS a cada
// pollInterval e substituindo o snapshot em memoria da tabela de
// descoberta - nunca bloqueia o caminho de resolucao DNS, que so le o
// snapshot mais recente via routeTable.lookup.
func pollPeers(cfg *dnsConfig) {
	for {
		pollOnce(cfg)
		time.Sleep(pollInterval)
	}
}

// pollOnce executa um ciclo de troca de vetor de distancia: consulta
// cada peer, mescla o que veio de volta com o que ja era conhecido
// (regra: so substitui uma rota existente se ela vier do mesmo peer -
// refresh - ou se a nova distancia for menor), e marca como "stale"
// (sem apagar) qualquer rota que nao foi reconfirmada por tempo demais.
func pollOnce(cfg *dnsConfig) {
	updated := ownRoutes(cfg)
	previous := map[string]discoveredRoute{}
	for _, route := range cfg.routes.snapshot() {
		previous[route.Domain] = route
	}

	for _, peer := range effectivePeers(cfg) {
		advertised, err := fetchRoutes(peer)
		if err != nil {
			log.Printf("[dns-provider] aviso: falha ao consultar peer de descoberta %s: %v", peer, err)
			continue
		}
		mergeAdvertised(cfg, updated, peer, advertised)
	}

	markStaleOrCarryOver(previous, updated)
	cfg.routes.replace(updated)
	persistSnapshot(cfg, updated)
}

// effectivePeers combina os peers configurados manualmente
// (DISCOVER_PEERS, alcancam qualquer rede) com os auto-descobertos por
// multicast na rede local (discover_broadcast.go, expiram se pararem
// de anunciar), sem duplicar enderecos.
func effectivePeers(cfg *dnsConfig) []string {
	seen := map[string]bool{}
	var peers []string
	for _, peer := range cfg.peers {
		if seen[peer] {
			continue
		}
		seen[peer] = true
		peers = append(peers, peer)
	}
	if cfg.discovered != nil {
		for _, peer := range cfg.discovered.addresses(peerMaxAge) {
			if seen[peer] {
				continue
			}
			seen[peer] = true
			peers = append(peers, peer)
		}
	}
	return peers
}

// mergeAdvertised aplica as rotas anunciadas por um peer no mapa
// "updated" (regra de vetor de distancia: distancia+1, nunca aprender
// uma rota de volta pro proprio dono, limite de saltos, so substitui
// uma rota existente vinda de outro peer se a nova distancia for
// menor).
func mergeAdvertised(cfg *dnsConfig, updated map[string]discoveredRoute, peer string, advertised []advertisedRoute) {
	for _, adv := range advertised {
		if adv.Owner == cfg.nodeName {
			continue // nunca aprender uma rota de volta para si mesmo
		}
		newDistance := adv.Distance + 1
		if newDistance > maxHops {
			continue
		}
		if _, isOwn := updated[adv.Domain]; isOwn {
			continue // dominio anunciado localmente sempre vence
		}
		existing, exists := updated[adv.Domain]
		if exists && existing.Source != peer && newDistance >= existing.Distance {
			continue
		}
		updated[adv.Domain] = discoveredRoute{
			Domain:   adv.Domain,
			Owner:    adv.Owner,
			NextHop:  hostOf(peer),
			Distance: newDistance,
			Source:   peer,
			State:    routeStateOK,
			LastSeen: time.Now(),
		}
	}
}

// markStaleOrCarryOver preserva, no mapa "updated", qualquer rota
// aprendida antes que nao foi reconfirmada neste ciclo - marcando como
// "stale" (nunca apagando) se ja passou tempo demais sem confirmacao.
func markStaleOrCarryOver(previous, updated map[string]discoveredRoute) {
	for domain, old := range previous {
		if _, present := updated[domain]; present {
			continue
		}
		if old.Source == "self" {
			continue
		}
		if time.Since(old.LastSeen) > staleAfter {
			old.State = routeStateStale
		}
		updated[domain] = old
	}
}

func persistSnapshot(cfg *dnsConfig, updated map[string]discoveredRoute) {
	if cfg.db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	routes := make([]discoveredRoute, 0, len(updated))
	for _, route := range updated {
		routes = append(routes, route)
	}
	if err := persistRoutes(ctx, cfg.db, routes); err != nil {
		log.Printf("[dns-provider] aviso: falha ao persistir tabela de descoberta: %v", err)
	}
}
