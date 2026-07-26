package hotspot

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// hotspotDeviceTime representa o saldo/estado da limitacao por tempo de
// um dispositivo (LimitType "time"). Espelha hotspotDeviceCredit:
// blockedByTime fica separado do bloqueio manual, e Configured=true faz
// o device parar de herdar a politica de tempo do perfil vinculado (ver
// syncDeviceTimeFromProfile). Modo budget usa balanceSeconds/recarga
// (gasto enquanto associado); modo deadline usa deadlineAt.
type hotspotDeviceTime struct {
	MACAddress      string
	Enabled         bool
	Mode            string
	BalanceSeconds  int64
	RechargeSeconds *int64
	RechargePeriod  *string
	PlafondSeconds  *int64
	NextRechargeAt  *time.Time
	DeadlineAt      *time.Time
	LastChargedAt   *time.Time
	BlockedByTime   bool
	Configured      bool
}

const deviceTimeColumns = `mac_address, enabled, mode, balance_seconds, recharge_seconds, recharge_period,
	plafond_seconds, next_recharge_at, deadline_at, last_charged_at, blocked_by_time, configured`

func scanDeviceTime(row interface{ Scan(...any) error }) (hotspotDeviceTime, error) {
	var t hotspotDeviceTime
	err := row.Scan(&t.MACAddress, &t.Enabled, &t.Mode, &t.BalanceSeconds, &t.RechargeSeconds, &t.RechargePeriod,
		&t.PlafondSeconds, &t.NextRechargeAt, &t.DeadlineAt, &t.LastChargedAt, &t.BlockedByTime, &t.Configured)
	return t, err
}

func ensureDeviceTimeRow(db *sql.DB, mac string) (hotspotDeviceTime, error) {
	return scanDeviceTime(db.QueryRow(`
		INSERT INTO hotspot_device_time (mac_address)
		VALUES ($1)
		ON CONFLICT (mac_address) DO UPDATE SET mac_address = EXCLUDED.mac_address
		RETURNING `+deviceTimeColumns, mac))
}

// hotspotTimeConfigRequest configura a politica de tempo (modo/recarga/
// plafond/deadline), nunca o saldo em si - marca configured=true (para
// de herdar do perfil, ver syncDeviceTimeFromProfile).
type hotspotTimeConfigRequest struct {
	Mode            *string    `json:"mode"`
	RechargeSeconds *int64     `json:"rechargeSeconds"`
	RechargePeriod  *string    `json:"rechargePeriod"`
	PlafondSeconds  *int64     `json:"plafondSeconds"`
	DeadlineAt      *time.Time `json:"deadlineAt"`
}

type hotspotTimeResponse struct {
	Enabled         bool    `json:"enabled"`
	Mode            string  `json:"mode"`
	BalanceSeconds  int64   `json:"balanceSeconds"`
	RechargeSeconds *int64  `json:"rechargeSeconds"`
	RechargePeriod  *string `json:"rechargePeriod"`
	PlafondSeconds  *int64  `json:"plafondSeconds"`
	NextRechargeAt  *string `json:"nextRechargeAt"`
	DeadlineAt      *string `json:"deadlineAt"`
	BlockedByTime   bool    `json:"blockedByTime"`
}

type hotspotTimeRechargeRequest struct {
	AmountSeconds int64 `json:"amountSeconds"`
}

func RegisterHotspotTimeRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, worker *workerapi.Client, audit *audit.Client) {
	mux.HandleFunc("GET /api/hotspot/devices/{mac}/time", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		mac, err := normalizeHotspotMAC(r.PathValue("mac"))
		if err != nil {
			http.Error(w, "mac invalido", http.StatusBadRequest)
			return
		}
		t, err := ensureDeviceTimeRow(db, mac)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeDeviceTimeResponse(w, db, mac, t)
	}))

	mux.HandleFunc("PATCH /api/hotspot/devices/{mac}/time", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		mac, err := normalizeHotspotMAC(r.PathValue("mac"))
		if err != nil {
			http.Error(w, "mac invalido", http.StatusBadRequest)
			return
		}
		var req hotspotTimeConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		if req.Mode != nil && !store.IsValidTimeMode(*req.Mode) {
			http.Error(w, "modo de tempo invalido", http.StatusBadRequest)
			return
		}
		if err := upsertDeviceTimeConfig(db, mac, req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		t, err := ensureDeviceTimeRow(db, mac)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeDeviceTimeResponse(w, db, mac, t)
	}))

	mux.HandleFunc("POST /api/hotspot/devices/{mac}/time/recharge", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		mac, err := normalizeHotspotMAC(r.PathValue("mac"))
		if err != nil {
			http.Error(w, "mac invalido", http.StatusBadRequest)
			return
		}
		var req hotspotTimeRechargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountSeconds <= 0 {
			http.Error(w, "campo 'amountSeconds' deve ser positivo", http.StatusBadRequest)
			return
		}
		t, err := applyManualTimeRecharge(r.Context(), db, worker, mac, req.AmountSeconds)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		audit.Record(r.Context(), "device_time_recharged", username, map[string]any{"mac": mac, "amountSeconds": req.AmountSeconds})
		writeDeviceTimeResponse(w, db, mac, t)
	}))
}

func writeDeviceTimeResponse(w http.ResponseWriter, db *sql.DB, mac string, t hotspotDeviceTime) {
	limits, err := effectiveDeviceLimits(db, mac)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(timeResponse(t, limits.LimitType))
}

// timeResponse monta a resposta publica - Enabled derivado do LimitType
// efetivo (nunca da coluna "enabled", vestigial igual em credito).
func timeResponse(t hotspotDeviceTime, effectiveType store.LimitType) hotspotTimeResponse {
	response := hotspotTimeResponse{
		Enabled:         effectiveType == store.LimitTypeTime,
		Mode:            t.Mode,
		BalanceSeconds:  t.BalanceSeconds,
		RechargeSeconds: t.RechargeSeconds,
		RechargePeriod:  t.RechargePeriod,
		PlafondSeconds:  t.PlafondSeconds,
		BlockedByTime:   t.BlockedByTime,
	}
	if t.NextRechargeAt != nil {
		formatted := t.NextRechargeAt.Format(time.RFC3339)
		response.NextRechargeAt = &formatted
	}
	if t.DeadlineAt != nil {
		formatted := t.DeadlineAt.Format(time.RFC3339)
		response.DeadlineAt = &formatted
	}
	return response
}
