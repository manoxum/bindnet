package cert

import (
	"database/sql"
	"errors"
)

var errCertificateRevoked = errors.New("certificado ja foi revogado")

// reissueCertificate gera uma chave e um certificado novos com o mesmo nome
// amigavel e, depois da emissao, move a versao anterior para os revogados.
// O sync do nginx-ui feito pela emissao substitui o material anterior pelo novo.
func reissueCertificate(
	db *sql.DB,
	ca *localCA,
	id string,
	rawDomains []string,
	validityQuantity int,
	validityUnit string,
) (*certificateResponse, error) {
	var name string
	var revokedAt sql.NullTime
	if err := db.QueryRow(`SELECT name, revoked_at FROM certificates WHERE id = $1`, id).Scan(&name, &revokedAt); err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		return nil, errCertificateRevoked
	}

	issued, err := issueCertificate(db, ca, name, rawDomains, validityQuantity, validityUnit)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`UPDATE certificates SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id); err != nil {
		return nil, err
	}
	return issued, nil
}
