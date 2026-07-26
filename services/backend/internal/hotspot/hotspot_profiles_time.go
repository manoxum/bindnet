package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"time"
)

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func isValidRechargePeriod(period string) bool {
	return period == "daily" || period == "weekly" || period == "monthly"
}

// validProfileTimePolicy valida os campos time_* de um ProfileRequest
// (modo e periodo de recarga) antes de gravar - o banco tem CHECK, mas
// vale devolver 400 claro em vez de 500. Campos nil/vazios sao aceitos
// (perfil sem politica de tempo).
func validProfileTimePolicy(req store.ProfileRequest) bool {
	if req.TimeMode != nil && !store.IsValidTimeMode(*req.TimeMode) {
		return false
	}
	if req.TimeRechargePeriod != nil && *req.TimeRechargePeriod != "" && !isValidRechargePeriod(*req.TimeRechargePeriod) {
		return false
	}
	return true
}

// profileTimePolicy extrai a politica de tempo de um perfil no formato
// do request de config (mode default 'budget' quando o perfil nao
// definiu).
func profileTimePolicy(profile store.Profile) hotspotTimeConfigRequest {
	mode := store.TimeModeBudget
	if profile.TimeMode != nil {
		mode = *profile.TimeMode
	}
	return hotspotTimeConfigRequest{
		Mode:            &mode,
		RechargeSeconds: profile.TimeRechargeSeconds,
		RechargePeriod:  profile.TimeRechargePeriod,
		PlafondSeconds:  profile.TimePlafondSeconds,
		DeadlineAt:      profile.TimeDeadlineAt,
	}
}

func timePolicyMatches(t hotspotDeviceTime, policy hotspotTimeConfigRequest) bool {
	return (policy.Mode == nil || t.Mode == *policy.Mode) &&
		equalInt64Ptr(t.RechargeSeconds, policy.RechargeSeconds) &&
		equalStringPtr(t.RechargePeriod, policy.RechargePeriod) &&
		equalInt64Ptr(t.PlafondSeconds, policy.PlafondSeconds) &&
		equalTimePtr(t.DeadlineAt, policy.DeadlineAt)
}

// applyTimePolicy grava so as colunas de politica vindas do perfil
// (nunca configured, balance_seconds ou blocked_by_time) - usada por
// syncDeviceTimeFromProfile quando o perfil vinculado e do tipo time.
func applyTimePolicy(db *sql.DB, mac string, policy hotspotTimeConfigRequest) (hotspotDeviceTime, error) {
	existingPeriod, err := getDeviceTimePeriod(db, mac)
	if err != nil {
		return hotspotDeviceTime{}, err
	}
	existingNext, err := getDeviceTimeNextRechargeAt(db, mac)
	if err != nil {
		return hotspotDeviceTime{}, err
	}
	nextRechargeAt := computeNextTimeRechargeAt(existingPeriod, existingNext, policy.RechargePeriod)

	return scanDeviceTime(db.QueryRow(`
		UPDATE hotspot_device_time
		SET mode = COALESCE($2, mode),
		    recharge_seconds = $3, recharge_period = $4, plafond_seconds = $5,
		    deadline_at = $6, next_recharge_at = $7, updated_at = CURRENT_TIMESTAMP
		WHERE mac_address = $1
		RETURNING `+deviceTimeColumns,
		mac, policy.Mode, policy.RechargeSeconds, policy.RechargePeriod, policy.PlafondSeconds,
		policy.DeadlineAt, nextRechargeAt))
}

// syncDeviceTimeFromProfile mantem a politica de tempo do device em dia
// com o perfil vinculado - so age quando Configured=false (o device
// nunca configurou tempo manualmente). Espelha syncDeviceCreditFromProfile.
// A politica so vem do perfil quando o proprio perfil e do tipo time
// (um device com override "time" sob um perfil "custom" configura a sua
// via PATCH .../time, sem nada para herdar aqui).
func syncDeviceTimeFromProfile(ctx context.Context, db *sql.DB, worker *workerapi.Client, mac string) (hotspotDeviceTime, error) {
	t, err := ensureDeviceTimeRow(db, mac)
	if err != nil {
		return hotspotDeviceTime{}, err
	}
	if t.Configured {
		return t, nil
	}
	effective, err := effectiveDeviceLimits(db, mac)
	if err != nil {
		return hotspotDeviceTime{}, err
	}
	if effective.LimitType != store.LimitTypeTime {
		if t.BlockedByTime {
			if err := unblockTimeIfNeeded(ctx, db, worker, mac, &t); err != nil {
				return hotspotDeviceTime{}, err
			}
		}
		return t, nil
	}

	profileID, err := deviceProfileID(db, mac)
	if err != nil {
		return hotspotDeviceTime{}, err
	}
	profile, found, err := store.GetProfile(db, profileID)
	if err != nil {
		return hotspotDeviceTime{}, err
	}
	if !found || profile.LimitType != store.LimitTypeTime {
		return t, nil
	}
	policy := profileTimePolicy(profile)
	if timePolicyMatches(t, policy) {
		return t, nil
	}
	return applyTimePolicy(db, mac, policy)
}
