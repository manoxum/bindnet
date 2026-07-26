// hotspot_content_l7.go liga/desliga o filtro de conteudo L7 no worker
// (REDIRECT de 443/80 dos clientes para o proxy transparente do
// dns-provider). Ligado so quando ha plano de conteudo em uso, igual ao
// reforco de DNS - senao nao mexe no caminho de rede dos clientes. As
// portas batem com os defaults L7_SNI_PORT/L7_HTTP_PORT do dns-provider.
package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"log"
	"net/http"
)

const (
	l7SNIPort  = 8443
	l7HTTPPort = 8082
)

type l7FilterPayload struct {
	Interface string `json:"interface"`
	Enabled   bool   `json:"enabled"`
	Gateway   string `json:"gateway"`
	SNIPort   int    `json:"sniPort"`
	HTTPPort  int    `json:"httpPort"`
}

func applyL7FilterLive(ctx context.Context, db *sql.DB, worker *workerapi.Client) {
	if err := applyL7Filter(ctx, db, worker); err != nil {
		log.Printf("[backend] filtro de conteudo L7 (SNI/HTTP) falhou: %v", err)
	}
}

func applyL7Filter(ctx context.Context, db *sql.DB, worker *workerapi.Client) error {
	iface, err := store.HotspotWifiInterface(ctx, db)
	if err != nil {
		return err
	}
	inUse, err := contentPlansInUse(db)
	if err != nil {
		return err
	}
	config, err := store.GetHotspotConfig(ctx, db)
	if err != nil {
		return err
	}
	// Gate de seguranca: o filtro L7 redireciona TODO o 443/80 dos
	// clientes por um proxy transparente novo - so liga quando ha plano
	// em uso E a chave CONTENT_L7_ENABLED=true, para poder validar o
	// proxy antes de afetar o trafego da rede inteira.
	if !inUse || config["CONTENT_L7_ENABLED"] != "true" {
		return worker.Call(ctx, http.MethodPost, "/hotspot/l7filter/apply", l7FilterPayload{Interface: iface, Enabled: false}, nil)
	}
	gateway := config["HOTSPOT_GATEWAY"]
	if gateway == "" {
		gateway = "192.168.12.1"
	}
	return worker.Call(ctx, http.MethodPost, "/hotspot/l7filter/apply", l7FilterPayload{
		Interface: iface, Enabled: true, Gateway: gateway, SNIPort: l7SNIPort, HTTPPort: l7HTTPPort,
	}, nil)
}

func teardownL7FilterWorker(ctx context.Context, worker *workerapi.Client, iface string) error {
	return worker.Call(ctx, http.MethodPost, "/hotspot/l7filter/apply", l7FilterPayload{Interface: iface, Enabled: false}, nil)
}
