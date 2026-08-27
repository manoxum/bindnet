// Package store concentra o acesso ao Postgres do dns-provider: alocacao
// persistente de offset de loopback por hostname, tabela de descoberta,
// identidade do no e leitura de config operacional do hotspot. O schema e
// criado pelo services/migration antes deste servico subir.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"bindnet/dns-provider/internal/config"
)

// ExplicitAddress devolve o A record da zona mais especifica que cobre o
// nome. Assim, uma entrada "empresa.local" vale tanto para a propria zona
// quanto para "app.empresa.local" e quaisquer outros subdominios. O DNS
// local e IPv4-only; entradas antigas ou invalidas sao ignoradas.
func ExplicitAddress(ctx context.Context, db *sql.DB, hostname string) (net.IP, bool, error) {
	var address sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT address::text
		FROM local_dns_records
		WHERE hostname = $1 OR $1 LIKE '%.' || hostname
		ORDER BY length(hostname) DESC
		LIMIT 1
	`, hostname).Scan(&address)
	if err == sql.ErrNoRows || !address.Valid || strings.TrimSpace(address.String) == "" {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	ip := parseIPv4(address.String)
	if ip == nil {
		return nil, false, nil
	}
	return ip.To4(), true, nil
}

// LoadLocalTLDAddresses retorna apenas os TLDs com IP estatico valido.
// O mapa e carregado no boot, junto com DNS_LOCAL_TLDS, para nao consultar
// Postgres no caminho quente de cada consulta DNS.
func LoadLocalTLDAddresses(ctx context.Context, db *sql.DB) (map[string]net.IP, error) {
	rows, err := db.QueryContext(ctx, `SELECT tld, address::text FROM local_dns_tlds WHERE address IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	addresses := map[string]net.IP{}
	for rows.Next() {
		var tld, raw string
		if err := rows.Scan(&tld, &raw); err != nil {
			return nil, err
		}
		if ip := parseIPv4(raw); ip != nil {
			addresses[strings.ToLower(strings.TrimSpace(tld))] = ip
		}
	}
	return addresses, rows.Err()
}

// parseIPv4 aceita tanto o formato normal quanto o formato textual INET
// do PostgreSQL, que representa um host como "10.234.1.69/32".
func parseIPv4(raw string) net.IP {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "/") {
		ip, network, err := net.ParseCIDR(raw)
		if err != nil {
			return nil
		}
		ones, bits := network.Mask.Size()
		if bits != 32 || ones != 32 {
			return nil
		}
		return ip.To4()
	}
	ip := net.ParseIP(raw)
	if ip == nil || ip.To4() == nil {
		return nil
	}
	return ip.To4()
}

// OpenPostgres segue o mesmo padrao de services/backend/db.go.
func OpenPostgres() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.Getenv("POSTGRES_USER", "bindnet"),
		config.Getenv("POSTGRES_PASSWORD", ""),
		config.Getenv("POSTGRES_HOST", "postgres"),
		config.Getenv("POSTGRES_PORT", "5432"),
		config.Getenv("POSTGRES_DB", "bindnet"),
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func LoadHotspotGateway(ctx context.Context, db *sql.DB, fallback string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, `SELECT value FROM hotspot_config WHERE key = 'HOTSPOT_GATEWAY'`).Scan(&value)
	if err == sql.ErrNoRows {
		return fallback, nil
	}
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	return value, nil
}

// LoadAllRecords le todos os registros existentes - usado para hidratar o
// cache Redis na inicializacao (ver pacote cache).
func LoadAllRecords(ctx context.Context, db *sql.DB) (map[string]int64, error) {
	rows, err := db.QueryContext(ctx, `SELECT hostname, loopback_offset FROM local_dns_records`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := map[string]int64{}
	for rows.Next() {
		var hostname string
		var offset int64
		if err := rows.Scan(&hostname, &offset); err != nil {
			return nil, err
		}
		records[hostname] = offset
	}
	return records, rows.Err()
}

// GetOrAllocateOffset devolve o offset de loopback ja atribuido a
// "hostname", ou aloca um novo (via local_dns_records_offset_seq) na
// primeira vez que esse hostname e consultado. Concorrencia-segura via
// INSERT ... ON CONFLICT DO NOTHING seguido de SELECT.
func GetOrAllocateOffset(ctx context.Context, db *sql.DB, hostname, originTLD string) (int64, error) {
	var offset int64
	err := db.QueryRowContext(ctx, `SELECT loopback_offset FROM local_dns_records WHERE hostname = $1`, hostname).Scan(&offset)
	if err == nil {
		return offset, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO local_dns_records (hostname, loopback_offset, source, origin_tld)
		 VALUES ($1, nextval('local_dns_records_offset_seq'), 'dynamic', NULLIF($2, ''))
		 ON CONFLICT (hostname) DO NOTHING`,
		hostname, originTLD,
	)
	if err != nil {
		return 0, err
	}

	err = db.QueryRowContext(ctx, `SELECT loopback_offset FROM local_dns_records WHERE hostname = $1`, hostname).Scan(&offset)
	return offset, err
}
