package main

import (
	"context"
	"database/sql"
)

// loadAllRoutes le a tabela discover_routes inteira - usada so para
// hidratar o snapshot em memoria na inicializacao (ver main.go), antes
// do primeiro ciclo de poll aos peers completar.
func loadAllRoutes(ctx context.Context, db *sql.DB) (map[string]discoveredRoute, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT domain, owner, next_hop, distance, source, state, last_seen_at
		FROM discover_routes
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	routes := map[string]discoveredRoute{}
	for rows.Next() {
		var route discoveredRoute
		if err := rows.Scan(&route.Domain, &route.Owner, &route.NextHop, &route.Distance, &route.Source, &route.State, &route.LastSeen); err != nil {
			return nil, err
		}
		routes[route.Domain] = route
	}
	return routes, rows.Err()
}

// persistRoutes grava o snapshot atual da tabela de descoberta no
// Postgres - chamada pela goroutine de poll (discover_peer.go) ao fim
// de cada ciclo, nunca no caminho quente de resolucao DNS.
func persistRoutes(ctx context.Context, db *sql.DB, routes []discoveredRoute) error {
	for _, route := range routes {
		_, err := db.ExecContext(ctx, `
			INSERT INTO discover_routes (domain, owner, next_hop, distance, source, state, last_seen_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (domain) DO UPDATE SET
				owner = EXCLUDED.owner,
				next_hop = EXCLUDED.next_hop,
				distance = EXCLUDED.distance,
				source = EXCLUDED.source,
				state = EXCLUDED.state,
				last_seen_at = EXCLUDED.last_seen_at
		`, route.Domain, route.Owner, route.NextHop, route.Distance, route.Source, route.State, route.LastSeen)
		if err != nil {
			return err
		}
	}
	return nil
}
