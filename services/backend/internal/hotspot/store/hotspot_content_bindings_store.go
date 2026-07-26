package store

import "database/sql"

// ContentClientBinding e uma linha do mapa IP -> plano publicado para o
// dns-provider.
type ContentClientBinding struct {
	IP     string
	MAC    string
	PlanID string
}

// ReplaceContentClientBindings troca o mapa inteiro numa transacao (o
// conjunto de clientes ao vivo com plano efetivo e pequeno, entao um
// full-replace por ciclo e barato e mantem o dns-provider sempre com a
// foto atual, sem linhas orfas de clientes que desconectaram).
func ReplaceContentClientBindings(db *sql.DB, bindings []ContentClientBinding) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM hotspot_content_client_bindings`); err != nil {
		return err
	}
	for _, b := range bindings {
		if _, err := tx.Exec(`
			INSERT INTO hotspot_content_client_bindings (ip, plan_id, mac_address)
			VALUES ($1, $2, $3)
			ON CONFLICT (ip) DO UPDATE SET plan_id = EXCLUDED.plan_id, mac_address = EXCLUDED.mac_address, updated_at = CURRENT_TIMESTAMP
		`, b.IP, b.PlanID, b.MAC); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func nullableContentPlanID(row interface{ Scan(...any) error }) (*string, error) {
	var id sql.NullString
	err := row.Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !id.Valid {
		return nil, nil
	}
	return &id.String, nil
}

func ProfileContentPlanID(db *sql.DB, profileID string) (*string, error) {
	return nullableContentPlanID(db.QueryRow(`SELECT content_plan_id FROM hotspot_profiles WHERE id = $1`, profileID))
}

func DeviceContentPlanID(db *sql.DB, mac string) (*string, error) {
	return nullableContentPlanID(db.QueryRow(`SELECT content_plan_id FROM hotspot_device_limits WHERE mac_address = $1`, mac))
}

func SetProfileContentPlanID(db *sql.DB, profileID string, planID *string) (bool, error) {
	res, err := db.Exec(`UPDATE hotspot_profiles SET content_plan_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`, profileID, planID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetDeviceContentPlanID grava o override de plano do dispositivo -
// upsert na linha de hotspot_device_limits (que pode ainda nao existir).
func SetDeviceContentPlanID(db *sql.DB, mac string, planID *string) error {
	_, err := db.Exec(`
		INSERT INTO hotspot_device_limits (mac_address, content_plan_id, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (mac_address) DO UPDATE SET content_plan_id = EXCLUDED.content_plan_id, updated_at = CURRENT_TIMESTAMP
	`, mac, planID)
	return err
}
