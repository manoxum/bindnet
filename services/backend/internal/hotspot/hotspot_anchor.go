package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// wifiNetwork espelha o que GET /network/wifi-scan devolve no worker
// (ver WifiNetwork em internal/network/wifi_scan.go).
type wifiNetwork struct {
	SSID    string `json:"ssid"`
	Channel int    `json:"channel"`
}

// refreshAnchorChannel reconfere, imediatamente antes de subir o
// hotspot, em que canal a rede ancora esta agora - e regrava
// WIFI_ANCHOR_CHANNEL se ela tiver mudado.
//
// Sem isto, o canal memorizado no momento em que o operador escolheu a
// rede envelheceria em silencio: se o router trocasse de canal, o
// hotspot subiria no canal antigo e a maquina deixaria de conseguir se
// associar a ele (num radio unico os dois tem que dividir a mesma
// frequencia). O sintoma seria idêntico ao de nao ter ancora nenhuma,
// e sem nada no log explicando.
//
// O momento e deliberado: e na subida que o canal importa, e a leitura
// vem do cache do NetworkManager (o worker nunca forca varredura), logo
// custa praticamente nada. Nao ha versao periodica disso de proposito -
// com o AP ja no ar, mover o canal exigiria derrubar todos os clientes,
// que e justamente o "seguidor de canal" que este projeto decidiu nao
// ter (ver RULE.md).
//
// Best-effort em todos os erros: nunca impede o hotspot de subir. Sem
// worker, sem NetworkManager ou com a rede ancora fora do ar, o canal
// memorizado continua valendo - que e exatamente o fallback desenhado.
func refreshAnchorChannel(ctx context.Context, db *sql.DB, worker *workerapi.Client) {
	config, err := store.GetHotspotConfig(ctx, db)
	if err != nil {
		return
	}
	anchor := strings.TrimSpace(config["WIFI_ANCHOR_SSID"])
	if anchor == "" {
		return
	}

	var networks []wifiNetwork
	if err := worker.Call(ctx, http.MethodGet, "/network/wifi-scan", nil, &networks); err != nil {
		return
	}

	for _, network := range networks {
		if network.SSID != anchor {
			continue
		}
		current := strconv.Itoa(network.Channel)
		if current == strings.TrimSpace(config["WIFI_ANCHOR_CHANNEL"]) {
			return
		}
		if err := store.SaveHotspotConfig(ctx, db, map[string]string{"WIFI_ANCHOR_CHANNEL": current}); err != nil {
			log.Printf("[backend] falha ao atualizar o canal da rede ancora '%s': %v", anchor, err)
			return
		}
		log.Printf("[backend] rede ancora '%s' mudou para o canal %s; hotspot passa a subir nesse canal", anchor, current)
		return
	}
}
