package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

func RegisterHotspotRoutes(mux *http.ServeMux, worker *workerapi.Client, admin *auth.Administrator, audit *audit.Client, db *sql.DB) {
	mux.HandleFunc("GET /api/hotspot/config", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		config, err := store.GetHotspotConfig(r.Context(), db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(config)
	}))

	mux.HandleFunc("PATCH /api/hotspot/config", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var config map[string]string
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		if err := store.SaveHotspotConfig(r.Context(), db, config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "config_changed", username, map[string]any{"section": "hotspot"})
		w.WriteHeader(http.StatusNoContent)
	}))

	// Redes Wi-Fi visiveis, para o seletor da "rede ancora" (ver
	// WIFI_ANCHOR_SSID em store/hotspot_config_store.go). O worker le o
	// cache do NetworkManager e nunca forca varredura - um scan ativo
	// com o AP no ar interrompe o beacon e chega a derrubar clientes.
	mux.HandleFunc("GET /api/hotspot/wifi-scan", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var networks json.RawMessage
		if err := worker.Call(r.Context(), http.MethodGet, "/network/wifi-scan", nil, &networks); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(networks)
	}))

	mux.HandleFunc("GET /api/hotspot/interfaces", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var interfaces json.RawMessage
		if err := worker.Call(r.Context(), http.MethodGet, "/network/interfaces", nil, &interfaces); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(interfaces)
	}))

	mux.HandleFunc("GET /api/hotspot/status", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var status struct {
			Running   bool   `json:"running"`
			Status    string `json:"status"`
			StartedAt string `json:"startedAt"`
		}
		if err := worker.Call(r.Context(), http.MethodGet, "/hotspot/status", nil, &status); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		response := map[string]any{
			"running":   status.Running,
			"status":    status.Status,
			"startedAt": status.StartedAt,
		}
		if status.Running {
			var logs strings.Builder
			if err := worker.CallText(r.Context(), "/containers/hotspot/logs?tail=200", &logs); err == nil {
				channel, band := parseHotspotChannelBand(logs.String())
				if channel != "" {
					response["channel"] = channel
				}
				if band != "" {
					response["band"] = band
				}
				if internetInterface := parseHotspotInternetInterface(logs.String()); internetInterface != "" {
					response["internetInterface"] = internetInterface
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))

	mux.HandleFunc("GET /api/hotspot/clients", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		clients, err := listEnrichedHotspotClients(r, db, worker)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clients)
	}))

	RegisterHotspotLifecycleRoutes(mux, worker, admin, audit, db)
	RegisterHotspotLogsRoutes(mux, worker, admin, audit, db)
	RegisterHotspotUplinkRoute(mux, admin, audit, db)
	RegisterHotspotAutostartRoutes(mux, admin, audit, db)
}

// StartReasonManual/StartReasonAuto viajam junto com a configuracao no
// corpo de /hotspot/start e /hotspot/apply (chave interna
// _START_REASON, nunca gravada no banco) so pra o entrypoint do hotspot
// saber se pode desbloquear o radio via rfkill: religar um radio que o
// usuario acabou de desligar no sistema faria o interruptor de Wi-Fi
// dele parar de funcionar enquanto o hotspot estivesse ligado. Ver
// ensure_wifi_radio_unblocked em
// services/worker/hotspot/regulatory.sh.
const (
	StartReasonManual = "manual"
	StartReasonAuto   = "auto"
)

func applyHotspotRuntimeConfig(ctx context.Context, db *sql.DB, worker *workerapi.Client, reason string) error {
	config, err := hotspotConfigWithStartReason(ctx, db, worker, reason)
	if err != nil {
		return err
	}
	return worker.Call(ctx, http.MethodPost, "/hotspot/apply", config, nil)
}

func startHotspotRuntimeConfig(ctx context.Context, db *sql.DB, worker *workerapi.Client, reason string) error {
	config, err := hotspotConfigWithStartReason(ctx, db, worker, reason)
	if err != nil {
		return err
	}
	return worker.Call(ctx, http.MethodPost, "/hotspot/start", config, nil)
}

func hotspotConfigWithStartReason(ctx context.Context, db *sql.DB, worker *workerapi.Client, reason string) (map[string]string, error) {
	// Antes de ler a configuracao: e na subida que o canal da rede
	// ancora importa, e e o unico momento em que da pra corrigi-lo sem
	// derrubar clientes (ver refreshAnchorChannel em hotspot_anchor.go).
	refreshAnchorChannel(ctx, db, worker)

	config, err := store.HotspotRuntimeConfig(ctx, db)
	if err != nil {
		return nil, err
	}
	config["_START_REASON"] = reason
	return config, nil
}
