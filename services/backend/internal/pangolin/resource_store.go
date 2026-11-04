package pangolin

import "context"

func (s *service) resourceIDs(ctx context.Context, account string) (map[int]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT resource_id,site_id FROM pangolin_account_resources WHERE account_id=$1`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := map[int]int{}
	for rows.Next() {
		var id, site int
		if err := rows.Scan(&id, &site); err != nil {
			return nil, err
		}
		ids[id] = site
	}
	return ids, rows.Err()
}
