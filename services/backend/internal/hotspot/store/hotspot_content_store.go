package store

import (
	"database/sql"
	"errors"
)

// ErrContentPlanNameTaken e devolvido quando o nome do plano ja existe
// (violacao do UNIQUE em hotspot_content_plans.name) - o handler traduz
// para 409.
var ErrContentPlanNameTaken = errors.New("ja existe um plano de conteudo com esse nome")

// ContentPlan e um plano de bloqueio/permissao de conteudo. DefaultAction
// 'allow' = lista negra (bloqueia so o que casar block); 'block' = lista
// branca (bloqueia tudo menos allow).
type ContentPlan struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	DefaultAction string `json:"defaultAction"`
}

// ContentRule e uma regra de um plano. Kind domain/category resolvidos
// pelo dns-provider; ip (IP/CIDR) pela zona WAN do firewall.
type ContentRule struct {
	ID     string `json:"id"`
	PlanID string `json:"planId"`
	Kind   string `json:"kind"`
	Value  string `json:"value"`
	Action string `json:"action"`
}

func scanContentPlan(row interface{ Scan(...any) error }) (ContentPlan, error) {
	var p ContentPlan
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.DefaultAction)
	return p, err
}

func ListContentPlans(db *sql.DB) ([]ContentPlan, error) {
	rows, err := db.Query(`SELECT id, name, description, default_action FROM hotspot_content_plans ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := []ContentPlan{}
	for rows.Next() {
		p, err := scanContentPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func GetContentPlan(db *sql.DB, id string) (ContentPlan, bool, error) {
	p, err := scanContentPlan(db.QueryRow(`SELECT id, name, description, default_action FROM hotspot_content_plans WHERE id = $1`, id))
	if err == sql.ErrNoRows {
		return ContentPlan{}, false, nil
	}
	if err != nil {
		return ContentPlan{}, false, err
	}
	return p, true, nil
}

func InsertContentPlan(db *sql.DB, name, description, defaultAction string) (ContentPlan, error) {
	p, err := scanContentPlan(db.QueryRow(`
		INSERT INTO hotspot_content_plans (name, description, default_action)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, default_action
	`, name, description, defaultAction))
	if isUniqueViolation(err) {
		return ContentPlan{}, ErrContentPlanNameTaken
	}
	return p, err
}

func UpdateContentPlan(db *sql.DB, id, name, description, defaultAction string) (ContentPlan, bool, error) {
	p, err := scanContentPlan(db.QueryRow(`
		UPDATE hotspot_content_plans
		SET name = $2, description = $3, default_action = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING id, name, description, default_action
	`, id, name, description, defaultAction))
	if isUniqueViolation(err) {
		return ContentPlan{}, false, ErrContentPlanNameTaken
	}
	if err == sql.ErrNoRows {
		return ContentPlan{}, false, nil
	}
	if err != nil {
		return ContentPlan{}, false, err
	}
	return p, true, nil
}

// DeleteContentPlan apaga o plano, suas regras e desvincula perfis/
// dispositivos que apontavam para ele (content_plan_id -> NULL), tudo na
// mesma transacao - nunca deixa vinculo apontando pra plano inexistente.
func DeleteContentPlan(db *sql.DB, id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE hotspot_profiles SET content_plan_id = NULL WHERE content_plan_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE hotspot_device_limits SET content_plan_id = NULL WHERE content_plan_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM hotspot_content_rules WHERE plan_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM hotspot_content_plans WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func ListContentRules(db *sql.DB, planID string) ([]ContentRule, error) {
	rows, err := db.Query(`SELECT id, plan_id, kind, value, action FROM hotspot_content_rules WHERE plan_id = $1 ORDER BY kind, value`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ContentRule{}
	for rows.Next() {
		var r ContentRule
		if err := rows.Scan(&r.ID, &r.PlanID, &r.Kind, &r.Value, &r.Action); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func InsertContentRule(db *sql.DB, planID, kind, value, action string) (ContentRule, error) {
	var r ContentRule
	err := db.QueryRow(`
		INSERT INTO hotspot_content_rules (plan_id, kind, value, action)
		VALUES ($1, $2, $3, $4)
		RETURNING id, plan_id, kind, value, action
	`, planID, kind, value, action).Scan(&r.ID, &r.PlanID, &r.Kind, &r.Value, &r.Action)
	return r, err
}

func DeleteContentRule(db *sql.DB, id string) (bool, error) {
	res, err := db.Exec(`DELETE FROM hotspot_content_rules WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
