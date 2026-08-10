package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

func HotspotWifiInterface(ctx context.Context, db *sql.DB) (string, error) {
	config, err := GetHotspotConfig(ctx, db)
	if err != nil {
		return "", err
	}
	iface := strings.TrimSpace(config["WIFI_INTERFACE"])
	if iface == "" {
		return "", errors.New("WIFI_INTERFACE nao configurado pelo painel")
	}
	return iface, nil
}

// hotspotDesiredStateKey guarda a ultima intencao do admin
// (ligar/desligar, ver POST /api/hotspot/start e /stop) na mesma
// tabela hotspot_config, mas fora de hotspotConfigKeys - de proposito,
// pra nao aparecer em GET /api/hotspot/config nem poder ser
// sobrescrita via PATCH (SaveHotspotConfig rejeita chaves fora da
// allowlist). Usada por AutoStartHotspotOnBoot em
// hotspot_autostart.go pra decidir se religa o hotspot sozinho quando
// o backend reinicia, e por recoverHotspotIfDesired em
// hotspot_reconcile.go pra decidir se religa sozinho quando o hotspot
// cai sem pedido do admin (ex.: watchdog de falha de beacon).
const hotspotDesiredStateKey = "_DESIRED_STATE"

// hotspotAutostartKey guarda a resposta do operador ao interruptor
// "iniciar automaticamente no arranque" do painel (ver
// RegisterHotspotAutostartRoutes em hotspot_autostart_routes.go). Mesma
// tabela e mesmo motivo de hotspotDesiredStateKey pra ficar fora de
// hotspotConfigKeys: assim mexer nele nao passa pelo
// PATCH /api/hotspot/config + POST /hotspot/apply do painel, que
// reiniciaria o hotspot so pra gravar uma preferencia de arranque.
//
// Nao confundir com hotspotDesiredStateKey: este diz "o hotspot deve
// subir quando o backend arranca?" (decisao do operador, imutavel por
// conta propria), enquanto aquele diz "qual foi o ultimo clique do
// admin?" (muda a cada start/stop) e continua sendo o unico que governa
// a auto-recuperacao de queda em recoverHotspotIfDesired.
const hotspotAutostartKey = "_AUTOSTART"

// setHotspotStateKey grava uma das chaves internas de estado acima -
// compartilhada pelas duas pra nao duplicar o mesmo upsert.
func setHotspotStateKey(ctx context.Context, db *sql.DB, key, value string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO hotspot_config (key, value, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value,
		    updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

// hotspotStateKeyEquals le uma das chaves internas de estado e compara
// com o valor esperado. Chave ausente conta como "diferente" (padrao
// desligado para as duas: hotspot parado e autostart nao).
func hotspotStateKeyEquals(ctx context.Context, db *sql.DB, key, expected string) (bool, error) {
	var value string
	err := db.QueryRowContext(ctx, `SELECT value FROM hotspot_config WHERE key = $1`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == expected, nil
}

func SetHotspotDesiredState(ctx context.Context, db *sql.DB, running bool) error {
	value := "stopped"
	if running {
		value = "running"
	}
	return setHotspotStateKey(ctx, db, hotspotDesiredStateKey, value)
}

func HotspotDesiredStateRunning(ctx context.Context, db *sql.DB) (bool, error) {
	return hotspotStateKeyEquals(ctx, db, hotspotDesiredStateKey, "running")
}

func SetHotspotAutostart(ctx context.Context, db *sql.DB, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	return setHotspotStateKey(ctx, db, hotspotAutostartKey, value)
}

// HotspotAutostartEnabled devolve false quando a chave nunca foi
// gravada - o padrao e "nao" de proposito: nenhum arranque de backend
// (deploy, docker compose up, crash) deve ressuscitar um hotspot que o
// operador parou de proposito sem ele ter pedido isso explicitamente.
func HotspotAutostartEnabled(ctx context.Context, db *sql.DB) (bool, error) {
	return hotspotStateKeyEquals(ctx, db, hotspotAutostartKey, "true")
}
