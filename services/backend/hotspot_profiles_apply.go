package main

import (
	"context"
	"database/sql"
)

// defaultProfileID e o id fixo do perfil "Padrao" semeado pela
// migration 20260707000000_hotspot_profiles - mesmo idioma do literal
// 'global' ja usado por hotspot_global_limits.id.
const defaultProfileID = "00000000-0000-0000-0000-000000000001"

// deviceProfileID devolve o perfil vinculado ao MAC, ou o Padrao se o
// dispositivo nunca foi visto (sem linha em hotspot_device_info) ou a
// coluna profile_id estiver nula.
func deviceProfileID(db *sql.DB, mac string) (string, error) {
	var profileID sql.NullString
	err := db.QueryRow(`SELECT profile_id FROM hotspot_device_info WHERE mac_address = $1`, mac).Scan(&profileID)
	if err == sql.ErrNoRows {
		return defaultProfileID, nil
	}
	if err != nil {
		return "", err
	}
	if !profileID.Valid {
		return defaultProfileID, nil
	}
	return profileID.String, nil
}

// assignDeviceProfile so toca a coluna profile_id - mesmo idioma de
// recordDeviceSeen (hotspot_device_info_store.go), nunca sobrescreve
// vendor/device_name/os_name/alias/confidence.
func assignDeviceProfile(db *sql.DB, mac, profileID string) error {
	_, err := db.Exec(`
		INSERT INTO hotspot_device_info (mac_address, profile_id)
		VALUES ($1, $2)
		ON CONFLICT (mac_address) DO UPDATE SET profile_id = EXCLUDED.profile_id
	`, mac, profileID)
	return err
}

// effectiveDeviceLimits resolve os limites que devem valer agora para
// um MAC: override explicito do dispositivo (hotspot_device_limits)
// sempre vence; na ausencia dele, usa o perfil vinculado. Nunca cai
// para hotspot_global_limits - o limite global ja e uma camada HTB
// separada e sempre ativa, empilhar aqui duplicaria o enforcement.
func effectiveDeviceLimits(db *sql.DB, mac string) (hotspotLimits, error) {
	limits, found, err := getDeviceLimits(db, mac)
	if err != nil {
		return hotspotLimits{}, err
	}
	if found {
		return limits, nil
	}
	profileID, err := deviceProfileID(db, mac)
	if err != nil {
		return hotspotLimits{}, err
	}
	limits, _, err = getProfileLimits(db, profileID)
	return limits, err
}

// syncDeviceCreditFromProfile mantem a politica de credito
// (enabled/rechargeAmount/rechargePeriod/plafond) do dispositivo em dia
// com o perfil vinculado - so age quando Configured=false (o
// dispositivo nunca teve config manual de credito nem resgatou um
// voucher, ver hotspot_vouchers.go). Nunca mexe em balance_bytes fora
// da regra de "so reseta o relogio de recarga se o periodo mudou"
// (computeNextRechargeAt, hotspot_credit_recharge.go).
func syncDeviceCreditFromProfile(ctx context.Context, db *sql.DB, worker *workerClient, mac string) (hotspotDeviceCredit, error) {
	credit, err := ensureDeviceCreditRow(db, mac)
	if err != nil {
		return hotspotDeviceCredit{}, err
	}
	if credit.Configured {
		return credit, nil
	}
	profileID, err := deviceProfileID(db, mac)
	if err != nil {
		return hotspotDeviceCredit{}, err
	}
	policy, found, err := getProfileCreditConfig(db, profileID)
	if err != nil {
		return hotspotDeviceCredit{}, err
	}
	if !found || creditPolicyMatches(credit, policy) {
		return credit, nil
	}

	updated, err := applyCreditPolicy(db, mac, policy)
	if err != nil {
		return hotspotDeviceCredit{}, err
	}
	if !updated.Enabled {
		if err := unblockCreditIfNeeded(ctx, db, worker, mac, &updated); err != nil {
			return hotspotDeviceCredit{}, err
		}
	}
	return updated, nil
}

func creditPolicyMatches(credit hotspotDeviceCredit, policy hotspotCreditConfigRequest) bool {
	return credit.Enabled == policy.Enabled &&
		equalInt64Ptr(credit.RechargeAmountBytes, policy.RechargeAmountBytes) &&
		equalStringPtr(credit.RechargePeriod, policy.RechargePeriod) &&
		equalInt64Ptr(credit.PlafondBytes, policy.PlafondBytes)
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// applyCreditPolicy grava so as 4 colunas de politica vindas do perfil
// (nunca configured, balance_bytes ou blocked_by_credit) - usada
// exclusivamente por syncDeviceCreditFromProfile.
func applyCreditPolicy(db *sql.DB, mac string, policy hotspotCreditConfigRequest) (hotspotDeviceCredit, error) {
	existingPeriod, _, err := getDeviceCreditPeriod(db, mac)
	if err != nil {
		return hotspotDeviceCredit{}, err
	}
	existingNext, err := getDeviceNextRechargeAt(db, mac)
	if err != nil {
		return hotspotDeviceCredit{}, err
	}
	nextRechargeAt := computeNextRechargeAt(existingPeriod, existingNext, policy.RechargePeriod)

	var credit hotspotDeviceCredit
	err = db.QueryRow(`
		UPDATE hotspot_device_credit
		SET enabled = $2, recharge_amount_bytes = $3, recharge_period = $4, plafond_bytes = $5,
		    next_recharge_at = $6, updated_at = CURRENT_TIMESTAMP
		WHERE mac_address = $1
		RETURNING mac_address, enabled, balance_bytes, recharge_amount_bytes, recharge_period,
		          plafond_bytes, next_recharge_at, blocked_by_credit, configured
	`, mac, policy.Enabled, policy.RechargeAmountBytes, policy.RechargePeriod, policy.PlafondBytes, nextRechargeAt).Scan(
		&credit.MACAddress, &credit.Enabled, &credit.BalanceBytes, &credit.RechargeAmountBytes,
		&credit.RechargePeriod, &credit.PlafondBytes, &credit.NextRechargeAt, &credit.BlockedByCredit, &credit.Configured)
	return credit, err
}

// applyProfileShapingLive reaplica ao vivo o shaping de todo
// dispositivo conectado agora que herda este perfil (sem override
// proprio) - chamado depois de editar um perfil, mesmo espirito de
// applyGlobalShapingLive/applyDeviceShapingLive.
func applyProfileShapingLive(ctx context.Context, db *sql.DB, worker *workerClient, profileID string) {
	iface, err := hotspotWifiInterface(ctx, db)
	if err != nil {
		return
	}
	clients, err := liveHotspotClients(ctx, worker, iface)
	if err != nil {
		return
	}
	for _, client := range clients {
		id, err := deviceProfileID(db, client.MAC)
		if err != nil || id != profileID {
			continue
		}
		if _, found, err := getDeviceLimits(db, client.MAC); err != nil || found {
			continue
		}
		_ = ensureDeviceShaping(ctx, db, worker, iface, client.MAC, client.IP)
		_, _ = syncDeviceCreditFromProfile(ctx, db, worker, client.MAC)
	}
}
