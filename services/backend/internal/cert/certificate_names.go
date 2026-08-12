package cert

import (
	"database/sql"
	"errors"
	"strings"
	"unicode"
)

const maxCertificateNameLength = 100

var errInvalidCertificateName = errors.New("nome do certificado invalido")

// normalizeCertificateName valida o nome amigavel usado no Bindnet e no
// nginx-ui. Barras e caracteres de controle sao proibidos porque o nome
// tambem participa do caminho local usado para sincronizar os arquivos PEM.
func normalizeCertificateName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" || len([]rune(name)) > maxCertificateNameLength || name == "." || name == ".." {
		return "", errInvalidCertificateName
	}
	for _, r := range name {
		if unicode.IsControl(r) || r == '/' || r == '\\' || r == '"' {
			return "", errInvalidCertificateName
		}
	}
	return name, nil
}

func activeCertificateUsesName(db *sql.DB, name string) (bool, error) {
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM certificates WHERE name = $1 AND revoked_at IS NULL)`,
		name,
	).Scan(&exists)
	return exists, err
}
