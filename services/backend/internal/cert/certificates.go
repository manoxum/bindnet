// certificates.go emite/lista/revoga certificados assinados pela CA
// local (ca.go), sob demanda via API (nunca automaticamente por SNI, ao
// contrario do antigo services/worker/cert-proxy). Os parametros
// criptograficos e a validacao de dominio sao os mesmos do cert-proxy
// original - so o gatilho (request HTTP explicito em vez de handshake
// TLS) e o armazenamento (Postgres em vez de arquivo) mudaram.
package cert

import (
	"database/sql"
	"errors"
	"time"
)

var errCertificateNotRevoked = errors.New("certificado ainda nao foi revogado")

type certificateResponse struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Domain           string   `json:"domain"`
	CommonName       string   `json:"commonName"`
	DNSNames         []string `json:"dnsNames,omitempty"`
	IPAddresses      []string `json:"ipAddresses,omitempty"`
	IssuedAt         string   `json:"issuedAt"`
	ExpiresAt        string   `json:"expiresAt"`
	ValidityQuantity int      `json:"validityQuantity"`
	ValidityUnit     string   `json:"validityUnit"`
	RevokedAt        *string  `json:"revokedAt,omitempty"`
}

func listCertificates(db *sql.DB, revoked bool) ([]certificateResponse, error) {
	revokedFilter := `revoked_at IS NULL`
	if revoked {
		revokedFilter = `revoked_at IS NOT NULL`
	}
	rows, err := db.Query(
		`SELECT id, name, domain, common_name, COALESCE(array_to_string(dns_names, ','), ''), COALESCE(array_to_string(ip_addresses, ','), ''), issued_at, expires_at, validity_quantity, validity_unit, revoked_at
		 FROM certificates
		 WHERE ` + revokedFilter + `
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []certificateResponse{}
	for rows.Next() {
		var r certificateResponse
		var dnsNames, ipAddresses string
		var issuedAt, expiresAt time.Time
		var revokedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Name, &r.Domain, &r.CommonName, &dnsNames, &ipAddresses, &issuedAt, &expiresAt, &r.ValidityQuantity, &r.ValidityUnit, &revokedAt); err != nil {
			return nil, err
		}
		r.DNSNames = splitNonEmpty(dnsNames)
		r.IPAddresses = splitNonEmpty(ipAddresses)
		r.IssuedAt = issuedAt.Format(time.RFC3339)
		r.ExpiresAt = expiresAt.Format(time.RFC3339)
		if revokedAt.Valid {
			formatted := revokedAt.Time.Format(time.RFC3339)
			r.RevokedAt = &formatted
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// revokeCertificate seta revoked_at em vez de deletar a linha -
// mantem o historico visivel na listagem (com status "revogado").
// A operacao e idempotente para permitir nova tentativa de limpeza do
// nginx-ui caso a primeira remocao externa falhe depois da revogacao local.
func revokeCertificate(db *sql.DB, id string) (string, error) {
	var name string
	if err := db.QueryRow(`SELECT name FROM certificates WHERE id = $1`, id).Scan(&name); err != nil {
		return "", err
	}
	_, err := db.Exec(`UPDATE certificates SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1`, id)
	return name, err
}

func revokedCertificateNameByID(db *sql.DB, id string) (string, error) {
	var name string
	var revokedAt sql.NullTime
	err := db.QueryRow(`SELECT name, revoked_at FROM certificates WHERE id = $1`, id).Scan(&name, &revokedAt)
	if err != nil {
		return "", err
	}
	if !revokedAt.Valid {
		return "", errCertificateNotRevoked
	}
	return name, nil
}

func permanentlyDeleteCertificate(db *sql.DB, id string) error {
	result, err := db.Exec(`DELETE FROM certificates WHERE id = $1 AND revoked_at IS NOT NULL`, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func certificatePEMByID(db *sql.DB, id string) (name, certificatePEM string, err error) {
	err = db.QueryRow(`SELECT name, certificate_pem FROM certificates WHERE id = $1`, id).Scan(&name, &certificatePEM)
	return name, certificatePEM, err
}
