package store

import (
	"database/sql"
	"time"
)

// MessageRow e um aviso que o operador envia aos dispositivos conectados
// (ver services/backend/internal/hotspot/hotspot_messages.go). TargetMAC
// vazio = broadcast (todos os conectados); preenchido = aviso a um unico
// dispositivo. Urgent, alem de aparecer no /portal, forca o balao "Entrar
// na rede" via portal cativo (best-effort).
type MessageRow struct {
	ID        string
	Title     string
	Body      string
	TargetMAC string // "" = broadcast
	Urgent    bool
	ExpiresAt sql.NullTime
	CreatedAt time.Time
}

const messageColumns = `id, title, body, COALESCE(target_mac, ''), urgent, expires_at, created_at`

func scanMessage(scanner interface{ Scan(...any) error }) (MessageRow, error) {
	var m MessageRow
	err := scanner.Scan(&m.ID, &m.Title, &m.Body, &m.TargetMAC, &m.Urgent, &m.ExpiresAt, &m.CreatedAt)
	return m, err
}

// ListActiveMessages devolve os avisos ainda ativos (nao removidos pelo
// admin), mais recentes primeiro e com os urgentes no topo - usado pela
// listagem do painel.
func ListActiveMessages(db *sql.DB) ([]MessageRow, error) {
	rows, err := db.Query(`SELECT ` + messageColumns + `
		FROM hotspot_messages WHERE active
		ORDER BY urgent DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

// ActiveMessagesForMAC devolve os avisos que um dispositivo ainda deve
// ver: ativos, nao expirados, direcionados a ele ou broadcast, e que
// esse MAC ainda nao marcou como lidos.
func ActiveMessagesForMAC(db *sql.DB, mac string) ([]MessageRow, error) {
	rows, err := db.Query(`SELECT `+messageColumns+`
		FROM hotspot_messages m
		WHERE m.active
		  AND (m.expires_at IS NULL OR m.expires_at > CURRENT_TIMESTAMP)
		  AND (m.target_mac IS NULL OR m.target_mac = $1)
		  AND NOT EXISTS (
		      SELECT 1 FROM hotspot_message_reads r
		      WHERE r.message_id = m.id AND r.mac = $1
		  )
		ORDER BY m.urgent DESC, m.created_at DESC`, mac)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func collectMessages(rows *sql.Rows) ([]MessageRow, error) {
	var out []MessageRow
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateMessage grava um novo aviso. targetMAC vazio vira broadcast
// (target_mac NULL); expiresAt invalido (Valid=false) vira sem expiracao.
func CreateMessage(db *sql.DB, title, body, targetMAC string, urgent bool, expiresAt sql.NullTime) (MessageRow, error) {
	return scanMessage(db.QueryRow(`
		INSERT INTO hotspot_messages (title, body, target_mac, urgent, expires_at)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5)
		RETURNING `+messageColumns,
		title, body, targetMAC, urgent, expiresAt))
}

// GetMessage devolve um aviso pelo id (found=false se nao existir) -
// usado pelo reconcile de push para saber o alvo/urgencia ao remover.
func GetMessage(db *sql.DB, id string) (MessageRow, bool, error) {
	m, err := scanMessage(db.QueryRow(`SELECT `+messageColumns+`
		FROM hotspot_messages WHERE id = $1`, id))
	if err == sql.ErrNoRows {
		return MessageRow{}, false, nil
	}
	if err != nil {
		return MessageRow{}, false, err
	}
	return m, true, nil
}

// SetMessageActive faz o soft delete (active=false) ou reativa um aviso;
// found=false se o id nao existir.
func SetMessageActive(db *sql.DB, id string, active bool) (found bool, err error) {
	res, err := db.Exec(`UPDATE hotspot_messages SET active = $2 WHERE id = $1`, id, active)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// MarkMessageRead registra que um MAC ja viu um aviso (idempotente).
func MarkMessageRead(db *sql.DB, messageID, mac string) error {
	_, err := db.Exec(`
		INSERT INTO hotspot_message_reads (message_id, mac)
		VALUES ($1, $2)
		ON CONFLICT (message_id, mac) DO NOTHING`, messageID, mac)
	return err
}

// HasActiveUrgentForMAC diz se ainda ha algum aviso urgente pendente
// (ativo, nao expirado, broadcast ou direcionado ao MAC, ainda nao lido)
// para o dispositivo - usado para decidir se o push do portal cativo deve
// continuar ligado depois de uma leitura.
func HasActiveUrgentForMAC(db *sql.DB, mac string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
		    SELECT 1 FROM hotspot_messages m
		    WHERE m.active AND m.urgent
		      AND (m.expires_at IS NULL OR m.expires_at > CURRENT_TIMESTAMP)
		      AND (m.target_mac IS NULL OR m.target_mac = $1)
		      AND NOT EXISTS (
		          SELECT 1 FROM hotspot_message_reads r
		          WHERE r.message_id = m.id AND r.mac = $1
		      )
		)`, mac).Scan(&exists)
	return exists, err
}
