package hotspot

import (
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"time"
)

// computeNextTimeRechargeAt decide o proximo agendamento de recarga
// periodica de tempo. Diferente do credito (que ancora em now+periodo),
// aqui um periodo recem-definido ancora em AGORA: o device recebe o
// primeiro saldo de tempo na hora (ex.: "1h por dia" ja comeca com 1h),
// sem esperar um periodo inteiro. Sem periodo novo ⇒ mantem o relogio
// atual (so trocar valor/plafond nao reinicia).
func computeNextTimeRechargeAt(existingPeriod *string, existingNext *time.Time, newPeriod *string) *time.Time {
	switch {
	case newPeriod == nil:
		return nil
	case existingPeriod == nil || *existingPeriod != *newPeriod:
		now := time.Now()
		return &now
	default:
		return existingNext
	}
}

func getDeviceTimePeriod(db *sql.DB, mac string) (*string, error) {
	var period sql.NullString
	err := db.QueryRow(`SELECT recharge_period FROM hotspot_device_time WHERE mac_address = $1`, mac).Scan(&period)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !period.Valid {
		return nil, nil
	}
	return &period.String, nil
}

func getDeviceTimeNextRechargeAt(db *sql.DB, mac string) (*time.Time, error) {
	var next sql.NullTime
	err := db.QueryRow(`SELECT next_recharge_at FROM hotspot_device_time WHERE mac_address = $1`, mac).Scan(&next)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !next.Valid {
		return nil, nil
	}
	return &next.Time, nil
}

// upsertDeviceTimeConfig grava a politica de tempo definida a mao pelo
// admin - marca configured=true (o device para de herdar do perfil, ver
// syncDeviceTimeFromProfile). Mode default 'budget' quando nao informado
// na primeira configuracao.
func upsertDeviceTimeConfig(db *sql.DB, mac string, req hotspotTimeConfigRequest) error {
	if _, err := ensureDeviceTimeRow(db, mac); err != nil {
		return err
	}
	existingPeriod, err := getDeviceTimePeriod(db, mac)
	if err != nil {
		return err
	}
	existingNext, err := getDeviceTimeNextRechargeAt(db, mac)
	if err != nil {
		return err
	}
	nextRechargeAt := computeNextTimeRechargeAt(existingPeriod, existingNext, req.RechargePeriod)

	_, err = db.Exec(`
		UPDATE hotspot_device_time
		SET mode = COALESCE($2, mode),
		    recharge_seconds = $3,
		    recharge_period = $4,
		    plafond_seconds = $5,
		    deadline_at = $6,
		    next_recharge_at = $7,
		    configured = true,
		    updated_at = CURRENT_TIMESTAMP
		WHERE mac_address = $1
	`, mac, req.Mode, req.RechargeSeconds, req.RechargePeriod, req.PlafondSeconds, req.DeadlineAt, nextRechargeAt)
	return err
}

// applyManualTimeRecharge soma segundos ao saldo (respeitando o plafond)
// e desbloqueia ao vivo se o device estava bloqueado por tempo e o saldo
// voltou a ser positivo - espelha applyManualRecharge do credito.
func applyManualTimeRecharge(ctx context.Context, db *sql.DB, worker *workerapi.Client, mac string, amountSeconds int64) (hotspotDeviceTime, error) {
	if _, err := ensureDeviceTimeRow(db, mac); err != nil {
		return hotspotDeviceTime{}, err
	}
	t, err := scanDeviceTime(db.QueryRow(`
		UPDATE hotspot_device_time
		SET balance_seconds = CASE
		        WHEN plafond_seconds IS NOT NULL THEN LEAST(balance_seconds + $2, plafond_seconds)
		        ELSE balance_seconds + $2
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE mac_address = $1
		RETURNING `+deviceTimeColumns, mac, amountSeconds))
	if err != nil {
		return hotspotDeviceTime{}, err
	}
	if t.BalanceSeconds > 0 {
		if err := unblockTimeIfNeeded(ctx, db, worker, mac, &t); err != nil {
			return t, err
		}
	}
	return t, nil
}

// applyAutomaticTimeRecharges avanca a recarga periodica de tempo de
// todo device cujo next_recharge_at ja passou - so mexe no saldo/relogio
// (o bloqueio/desbloqueio ao vivo fica com reconcileDeviceTime, que tem
// worker+contexto). Mesmo espirito de applyAutomaticRecharges.
func applyAutomaticTimeRecharges(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT mac_address FROM hotspot_device_time
		WHERE mode = 'budget' AND recharge_period IS NOT NULL
		  AND next_recharge_at IS NOT NULL AND next_recharge_at <= CURRENT_TIMESTAMP
	`)
	if err != nil {
		return err
	}
	var macs []string
	for rows.Next() {
		var mac string
		if err := rows.Scan(&mac); err != nil {
			rows.Close()
			return err
		}
		macs = append(macs, mac)
	}
	rows.Close()

	for _, mac := range macs {
		if err := advanceDeviceTimeRecharge(db, mac); err != nil {
			return err
		}
	}
	return nil
}

func advanceDeviceTimeRecharge(db *sql.DB, mac string) error {
	_, err := db.Exec(`
		UPDATE hotspot_device_time
		SET balance_seconds = CASE
		        WHEN plafond_seconds IS NOT NULL THEN LEAST(balance_seconds + COALESCE(recharge_seconds, 0), plafond_seconds)
		        ELSE balance_seconds + COALESCE(recharge_seconds, 0)
		    END,
		    next_recharge_at = next_recharge_at + (
		        CASE recharge_period
		            WHEN 'weekly' THEN interval '7 days'
		            WHEN 'monthly' THEN interval '30 days'
		            ELSE interval '1 day'
		        END
		    ),
		    updated_at = CURRENT_TIMESTAMP
		WHERE mac_address = $1 AND next_recharge_at <= CURRENT_TIMESTAMP
	`, mac)
	return err
}
