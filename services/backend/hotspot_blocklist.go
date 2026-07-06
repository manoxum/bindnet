package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type hotspotBlockedDevice struct {
	MACAddress string    `json:"macAddress"`
	Note       string    `json:"note,omitempty"`
	BlockedAt  time.Time `json:"blockedAt"`
}

type hotspotBlockRequest struct {
	MAC  string `json:"mac"`
	Note string `json:"note,omitempty"`
}

func registerHotspotBlocklistRoutes(mux *http.ServeMux, admin *administrator, db *sql.DB, worker *workerClient) {
	mux.HandleFunc("GET /api/hotspot/blocklist", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		devices, err := listHotspotBlockedDevices(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(devices)
	}))

	mux.HandleFunc("POST /api/hotspot/blocklist", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req hotspotBlockRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		mac, err := normalizeHotspotMAC(req.MAC)
		if err != nil {
			http.Error(w, "mac invalido", http.StatusBadRequest)
			return
		}
		device, err := upsertHotspotBlockedDevice(db, mac, strings.TrimSpace(req.Note))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyLiveHotspotBlock(r.Context(), db, worker, mac, true)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(device)
	}))

	mux.HandleFunc("DELETE /api/hotspot/blocklist/{mac}", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		mac, err := normalizeHotspotMAC(r.PathValue("mac"))
		if err != nil {
			http.Error(w, "mac invalido", http.StatusBadRequest)
			return
		}
		if err := deleteHotspotBlockedDevice(db, mac); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		applyLiveHotspotBlock(r.Context(), db, worker, mac, false)
		w.WriteHeader(http.StatusNoContent)
	}))
}

func hotspotBlockedSet(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT mac_address FROM hotspot_blocked_devices`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	blocked := map[string]bool{}
	for rows.Next() {
		var mac string
		if err := rows.Scan(&mac); err != nil {
			return nil, err
		}
		blocked[mac] = true
	}
	return blocked, rows.Err()
}

func listHotspotBlockedDevices(db *sql.DB) ([]hotspotBlockedDevice, error) {
	rows, err := db.Query(`
		SELECT mac_address, COALESCE(note, ''), blocked_at
		FROM hotspot_blocked_devices
		ORDER BY blocked_at DESC, mac_address
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	devices := []hotspotBlockedDevice{}
	for rows.Next() {
		var device hotspotBlockedDevice
		if err := rows.Scan(&device.MACAddress, &device.Note, &device.BlockedAt); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, rows.Err()
}

func upsertHotspotBlockedDevice(db *sql.DB, mac, note string) (hotspotBlockedDevice, error) {
	var device hotspotBlockedDevice
	err := db.QueryRow(`
		INSERT INTO hotspot_blocked_devices (mac_address, note, blocked_at)
		VALUES ($1, NULLIF($2, ''), CURRENT_TIMESTAMP)
		ON CONFLICT (mac_address) DO UPDATE
		SET note = EXCLUDED.note,
		    blocked_at = CURRENT_TIMESTAMP
		RETURNING mac_address, COALESCE(note, ''), blocked_at
	`, mac, note).Scan(&device.MACAddress, &device.Note, &device.BlockedAt)
	return device, err
}

func deleteHotspotBlockedDevice(db *sql.DB, mac string) error {
	_, err := db.Exec(`DELETE FROM hotspot_blocked_devices WHERE mac_address = $1`, mac)
	return err
}

func reapplyHotspotBlocklist(ctx context.Context, db *sql.DB, worker *workerClient, iface string) {
	blocked, err := listHotspotBlockedDevices(db)
	if err != nil {
		log.Printf("[backend] falha ao listar blocklist do hotspot: %v", err)
		return
	}
	for _, device := range blocked {
		var lastErr error
		for attempt := 0; attempt < 6; attempt++ {
			lastErr = worker.call(ctx, http.MethodPost, "/hotspot/block", map[string]string{
				"interface": iface,
				"mac":       device.MACAddress,
			}, nil)
			if lastErr == nil {
				break
			}
			time.Sleep(time.Second)
		}
		if lastErr != nil {
			log.Printf("[backend] bloqueio de %s persistido, mas aplicacao ao vivo falhou: %v", device.MACAddress, lastErr)
		}
	}
}

func applyLiveHotspotBlock(ctx context.Context, db *sql.DB, worker *workerClient, mac string, block bool) {
	iface, err := hotspotWifiInterface(ctx, db)
	if err != nil {
		log.Printf("[backend] blocklist persistida, mas nao foi possivel ler WIFI_INTERFACE: %v", err)
		return
	}
	path := "/hotspot/unblock"
	if block {
		path = "/hotspot/block"
	}
	if err := worker.call(ctx, http.MethodPost, path, map[string]string{"interface": iface, "mac": mac}, nil); err != nil {
		log.Printf("[backend] blocklist persistida, mas aplicacao ao vivo de %s falhou: %v", mac, err)
	}
}
