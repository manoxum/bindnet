package hotspot

import (
	"bindnet/backend/internal/hotspot/store"
	"bindnet/backend/internal/workerapi"
	"context"
	"database/sql"
	"time"
)

// maxTimeDebitSeconds limita quanto de saldo um unico ciclo debita:
// reconcileDeviceTime roda a ~1s por cliente ao vivo, entao o delta
// normal e ~1s. O teto evita drenar o saldo em bloco se o painel ficou
// fora do ar por muito tempo com o device ainda associado - nao se deve
// punir o usuario pelo downtime do backend (o last_charged_at nao anda
// enquanto o painel esta fora).
const maxTimeDebitSeconds = 30

// reconcileDeviceTime aplica a limitacao por tempo, chamado do ciclo de
// amostragem quando o LimitType efetivo e "time" (irmao de
// reconcileDeviceCredit/reconcileDeviceQuota). Como so roda para
// clientes AO VIVO, no modo budget estar aqui ja significa "associado" -
// debita o tempo decorrido (decisao do plano: conta enquanto associado,
// mesmo ocioso). Bloqueia ao esgotar/vencer e DESBLOQUEIA sozinho quando
// o saldo volta (recarga) ou o deadline e estendido.
func reconcileDeviceTime(ctx context.Context, db *sql.DB, worker *workerapi.Client, mac, ip string) error {
	t, err := syncDeviceTimeFromProfile(ctx, db, worker, mac)
	if err != nil {
		return err
	}
	if t.Mode == store.TimeModeDeadline {
		expired := t.DeadlineAt != nil && !time.Now().Before(*t.DeadlineAt)
		return setDeviceTimeBlock(ctx, db, worker, mac, ip, &t, expired)
	}
	return reconcileTimeBudget(ctx, db, worker, mac, ip, t)
}

func reconcileTimeBudget(ctx context.Context, db *sql.DB, worker *workerapi.Client, mac, ip string, t hotspotDeviceTime) error {
	if t.BlockedByTime {
		// Mantem o marco de cobranca fresco enquanto bloqueado para nao
		// debitar o intervalo parado ao reabrir; se o saldo voltou
		// (recarga), desbloqueia.
		if err := touchDeviceTimeCharged(db, mac); err != nil {
			return err
		}
		return setDeviceTimeBlock(ctx, db, worker, mac, ip, &t, t.BalanceSeconds <= 0)
	}
	// Bootstrap: um device novo (last_charged_at nulo) com recarga
	// periodica recebe o saldo do primeiro periodo AGORA, antes da
	// decisao de bloqueio - senao ficaria bloqueado ate a recarga
	// automatica do ciclo de 15s (janela de "1h/dia comeca bloqueado").
	// computeNextTimeRechargeAt ja ancora next_recharge_at em agora, entao
	// advanceDeviceTimeRecharge concede na hora. So no primeiro tick:
	// debitDeviceTime grava last_charged_at, entao este ramo nao repete.
	if t.BalanceSeconds <= 0 && t.RechargePeriod != nil && t.LastChargedAt == nil {
		if err := advanceDeviceTimeRecharge(db, mac); err != nil {
			return err
		}
		refreshed, err := ensureDeviceTimeRow(db, mac)
		if err != nil {
			return err
		}
		t = refreshed
	}
	newBalance, err := debitDeviceTime(db, mac)
	if err != nil {
		return err
	}
	if newBalance <= 0 {
		return setDeviceTimeBlock(ctx, db, worker, mac, ip, &t, true)
	}
	return nil
}

// setDeviceTimeBlock leva o device ao estado de bloqueio desejado,
// aplicando/removendo o traffic block + portal cativo ao vivo. Quando ja
// esta no estado certo e bloqueado, reforca ao vivo (idempotente - cobre
// regra perdida num restart do hotspot); quando ja esta desbloqueado,
// nao faz chamada nenhuma.
func setDeviceTimeBlock(ctx context.Context, db *sql.DB, worker *workerapi.Client, mac, ip string, t *hotspotDeviceTime, blocked bool) error {
	if blocked == t.BlockedByTime {
		if blocked {
			applyLiveTrafficBlock(ctx, db, worker, mac, ip, true)
			applyCaptivePortalRedirect(ctx, db, worker, mac, true)
		}
		return nil
	}
	if err := setDeviceTimeBlockedFlag(db, mac, blocked); err != nil {
		return err
	}
	t.BlockedByTime = blocked
	applyLiveTrafficBlock(ctx, db, worker, mac, ip, blocked)
	applyCaptivePortalRedirect(ctx, db, worker, mac, blocked)
	return nil
}

// debitDeviceTime desconta do saldo os segundos decorridos desde o
// ultimo debito (relogio do banco, via COALESCE para o primeiro tick sem
// last_charged_at), com teto maxTimeDebitSeconds, e reancorra
// last_charged_at. Devolve o saldo resultante.
func debitDeviceTime(db *sql.DB, mac string) (int64, error) {
	var balance int64
	err := db.QueryRow(`
		UPDATE hotspot_device_time
		SET balance_seconds = balance_seconds - LEAST($2::bigint,
		        GREATEST(0, EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - COALESCE(last_charged_at, CURRENT_TIMESTAMP)))::bigint)),
		    last_charged_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE mac_address = $1
		RETURNING balance_seconds
	`, mac, maxTimeDebitSeconds).Scan(&balance)
	return balance, err
}

func touchDeviceTimeCharged(db *sql.DB, mac string) error {
	_, err := db.Exec(`UPDATE hotspot_device_time SET last_charged_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE mac_address = $1`, mac)
	return err
}

func setDeviceTimeBlockedFlag(db *sql.DB, mac string, blocked bool) error {
	_, err := db.Exec(`UPDATE hotspot_device_time SET blocked_by_time = $2, updated_at = CURRENT_TIMESTAMP WHERE mac_address = $1`, mac, blocked)
	return err
}

// unblockTimeIfNeeded desbloqueia ao vivo um device bloqueado por tempo
// (no-op se nao estava) - compartilhado por applyManualTimeRecharge e
// syncDeviceTimeFromProfile, mesmo idioma de unblockCreditIfNeeded.
func unblockTimeIfNeeded(ctx context.Context, db *sql.DB, worker *workerapi.Client, mac string, t *hotspotDeviceTime) error {
	if !t.BlockedByTime {
		return nil
	}
	if err := setDeviceTimeBlockedFlag(db, mac, false); err != nil {
		return err
	}
	t.BlockedByTime = false
	applyLiveTrafficBlock(ctx, db, worker, mac, "", false)
	applyCaptivePortalRedirect(ctx, db, worker, mac, false)
	return nil
}

// clearStaleDeviceTimeBlock desfaz um bloqueio por tempo deixado de um
// LimitType anterior - chamado quando o device (configurado) nao e mais
// do tipo time, mesmo espirito de clearStaleDeviceQuotaBlock. No-op se
// nao havia bloqueio.
func clearStaleDeviceTimeBlock(ctx context.Context, db *sql.DB, worker *workerapi.Client, mac, ip string) error {
	var blocked bool
	err := db.QueryRow(`SELECT blocked_by_time FROM hotspot_device_time WHERE mac_address = $1`, mac).Scan(&blocked)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if !blocked {
		return nil
	}
	if err := setDeviceTimeBlockedFlag(db, mac, false); err != nil {
		return err
	}
	applyLiveTrafficBlock(ctx, db, worker, mac, ip, false)
	applyCaptivePortalRedirect(ctx, db, worker, mac, false)
	return nil
}
