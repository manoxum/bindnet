package pangolin

import (
	"context"
	"fmt"
)

func (s *service) startLocalConnector(ctx context.Context, account string, id int, endpoint string, site Site) error {
	body := map[string]string{"endpoint": endpoint, "newtId": site.NewtID, "secret": site.Secret}
	if err := s.worker.Slow().Call(ctx, "POST", connectorPath(account, id)+"/start", body, nil); err != nil {
		return fmt.Errorf("não foi possível iniciar o conector Newt")
	}
	return nil
}

func (s *service) setConnectorDesiredState(ctx context.Context, account string, id int, running bool) error {
	result, err := s.db.ExecContext(ctx, `UPDATE pangolin_account_sites SET desired_running=$3 WHERE account_id=$1 AND site_id=$2`, account, id, running)
	if err != nil {
		return fmt.Errorf("não foi possível guardar o estado desejado do conector")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return fmt.Errorf("este site não é gerido pelo Bindnet")
	}
	return nil
}
