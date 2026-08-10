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

func applyHotspotRuntimeConfig(ctx context.Context, db *sql.DB, worker *workerapi.Client) error {
	config, err := store.HotspotRuntimeConfig(ctx, db)
	if err != nil {
		return err
	}
	return worker.Call(ctx, http.MethodPost, "/hotspot/apply", config, nil)
}

func startHotspotRuntimeConfig(ctx context.Context, db *sql.DB, worker *workerapi.Client) error {
	config, err := store.HotspotRuntimeConfig(ctx, db)
	if err != nil {
		return err
	}
	return worker.Call(ctx, http.MethodPost, "/hotspot/start", config, nil)
}
