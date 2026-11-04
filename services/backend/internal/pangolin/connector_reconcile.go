package pangolin

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// StartConnectorReconciliationLoop recupera conectores no arranque e após falhas,
// respeitando o estado desejado guardado pelas ações Iniciar/Parar do painel.
func (s *service) StartConnectorReconciliationLoop(ctx context.Context) {
	for ctx.Err() == nil {
		if err := s.reconcileConnectorsOnce(ctx); err != nil && ctx.Err() == nil {
			log.Println("[backend] falha ao recuperar conectores Pangolin; nova tentativa em 15 segundos")
		}
		timer := time.NewTimer(15 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *service) reconcileConnectorsOnce(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT account_id,site_id FROM pangolin_account_sites WHERE desired_running=true ORDER BY account_id,site_id`)
	if err != nil {
		return fmt.Errorf("não foi possível ler os conectores para arranque automático")
	}
	type connectorID struct {
		account string
		site    int
	}
	var ids []connectorID
	for rows.Next() {
		var id connectorID
		if err := rows.Scan(&id.account, &id.site); err != nil {
			rows.Close()
			return fmt.Errorf("não foi possível ler o identificador do conector")
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fmt.Errorf("não foi possível listar os conectores")
	}
	var lastErr error
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Usa o mesmo bloqueio das mutações para não desfazer uma parada/remoção.
		if !s.mutex.TryLock() {
			continue
		}
		err := s.recoverConnector(ctx, id.account, id.site)
		s.mutex.Unlock()
		if err != nil {
			lastErr = err
			log.Printf("[backend] não foi possível recuperar conector Pangolin da conta %s, site %d", id.account, id.site)
		}
	}
	return lastErr
}

func (s *service) recoverConnector(ctx context.Context, account string, id int) error {
	var endpoint string
	var site Site
	err := s.db.QueryRowContext(ctx, `SELECT a.endpoint,s.newt_id,s.secret FROM pangolin_account_sites s JOIN pangolin_accounts a ON a.id=s.account_id WHERE s.account_id=$1 AND s.site_id=$2 AND s.desired_running=true`, account, id).Scan(&endpoint, &site.NewtID, &site.Secret)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("não foi possível ler as credenciais do conector")
	}
	return s.startLocalConnector(ctx, account, id, endpoint, site)
}
