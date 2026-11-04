package pangolin

import (
	"context"
	"database/sql"
)

func loadConfig(ctx context.Context, db *sql.DB, id string) (Config, error) {
	var c Config
	err := db.QueryRowContext(ctx, `SELECT id,name,api_url,endpoint,org_id,api_key FROM pangolin_accounts WHERE id=$1`, id).Scan(&c.ID, &c.Name, &c.APIURL, &c.Endpoint, &c.OrgID, &c.APIKey)
	c.Configured = c.APIKey != ""
	return c, err
}

func saveConfig(ctx context.Context, db *sql.DB, c Config) error {
	_, err := db.ExecContext(ctx, `UPDATE pangolin_accounts SET name=$2,api_url=$3,endpoint=$4,org_id=$5,api_key=$6,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, c.ID, c.Name, c.APIURL, c.Endpoint, c.OrgID, c.APIKey)
	return err
}

func managedSites(ctx context.Context, db *sql.DB, account string) ([]Site, error) {
	rows, err := db.QueryContext(ctx, `SELECT site_id,name,newt_id,secret FROM pangolin_account_sites WHERE account_id=$1 ORDER BY site_id`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sites := []Site{}
	for rows.Next() {
		var site Site
		site.Managed = true
		site.Type = "newt"
		if err := rows.Scan(&site.SiteID, &site.Name, &site.NewtID, &site.Secret); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func insertSite(ctx context.Context, db *sql.DB, account string, s Site) error {
	_, err := db.ExecContext(ctx, `INSERT INTO pangolin_account_sites(account_id,site_id,name,newt_id,secret) VALUES($1,$2,$3,$4,$5)`, account, s.SiteID, s.Name, s.NewtID, s.Secret)
	return err
}
