// hotspot_content_dnsforce.go controla o reforco de DNS no worker
// (forcar :53 local + bloquear DoH/DoT). So e ligado quando existe pelo
// menos um plano de conteudo vinculado a algum perfil ou dispositivo -
// sem plano em uso, nada muda no caminho de rede dos clientes (o reforco
// afeta todos os clientes do hotspot, entao so vale a pena quando ha
// bloqueio de conteudo de fato). Ver internal/shaping/dns_enforce.go.
package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"log"
	"net/http"
)

type dnsForcePayload struct {
	Interface string `json:"interface"`
	Enabled   bool   `json:"enabled"`
	Gateway   string `json:"gateway"`
}

func contentPlansInUse(db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM hotspot_profiles WHERE content_plan_id IS NOT NULL)
		    OR EXISTS (SELECT 1 FROM hotspot_device_limits WHERE content_plan_id IS NOT NULL)
	`).Scan(&exists)
	return exists, err
}

func applyDNSForceLive(ctx context.Context, db *sql.DB, worker *workerapi.Client) {
	if err := applyDNSForce(ctx, db, worker); err != nil {
		log.Printf("[backend] reforco de DNS (forcar :53 + DoH/DoT) falhou: %v", err)
	}
}

func applyDNSForce(ctx context.Context, db *sql.DB, worker *workerapi.Client) error {
	iface, err := store.HotspotWifiInterface(ctx, db)
	if err != nil {
		return err
	}
	inUse, err := contentPlansInUse(db)
	if err != nil {
		return err
	}
	if !inUse {
		return worker.Call(ctx, http.MethodPost, "/hotspot/dnsforce/apply", dnsForcePayload{Interface: iface, Enabled: false}, nil)
	}
	config, err := store.GetHotspotConfig(ctx, db)
	if err != nil {
		return err
	}
	gateway := config["HOTSPOT_GATEWAY"]
	if gateway == "" {
		gateway = "192.168.12.1"
	}
	return worker.Call(ctx, http.MethodPost, "/hotspot/dnsforce/apply", dnsForcePayload{Interface: iface, Enabled: true, Gateway: gateway}, nil)
}

// teardownDNSForceWorker desliga o reforco no worker - usado no stop do
// hotspot, junto com os teardowns de isolamento/firewall.
func teardownDNSForceWorker(ctx context.Context, worker *workerapi.Client, iface string) error {
	return worker.Call(ctx, http.MethodPost, "/hotspot/dnsforce/apply", dnsForcePayload{Interface: iface, Enabled: false}, nil)
}
