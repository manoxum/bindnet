// certificates.go emite/lista/revoga certificados assinados pela CA
// local (ca.go), sob demanda via API (nunca automaticamente por SNI, ao
// contrario do antigo services/worker/cert-proxy). Os parametros
// criptograficos e a validacao de dominio sao os mesmos do cert-proxy
// original - so o gatilho (request HTTP explicito em vez de handshake
// TLS) e o armazenamento (Postgres em vez de arquivo) mudaram.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"errors"
	"log"
	"net"
	"strings"
	"time"
)

const defaultDomain = "localhost.local"

var errCertificateNotRevoked = errors.New("certificado ainda nao foi revogado")

type certificateResponse struct {
	ID          string   `json:"id"`
	Domain      string   `json:"domain"`
	CommonName  string   `json:"commonName"`
	DNSNames    []string `json:"dnsNames,omitempty"`
	IPAddresses []string `json:"ipAddresses,omitempty"`
	IssuedAt    string   `json:"issuedAt"`
	ExpiresAt   string   `json:"expiresAt"`
	RevokedAt   *string  `json:"revokedAt,omitempty"`
}

// normalizeDomain e uma copia byte a byte da funcao homonima do antigo
// cert-proxy: minusculo, sem porta, sem "." final; cai para
// defaultDomain se algum rotulo nao passar na validacao [a-z0-9-].
func normalizeDomain(value string) string {
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if domain == "" {
		return defaultDomain
	}
	if h, _, err := net.SplitHostPort(domain); err == nil {
		domain = h
	}
	if ip := net.ParseIP(domain); ip != nil {
		return domain
	}
	for _, label := range strings.Split(domain, ".") {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return defaultDomain
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return defaultDomain
			}
		}
	}
	return domain
}

// issueCertificate sempre cria uma linha nova (sem cache de arquivo por
// dominio, ao contrario do antigo certificadoPara) - emitir e agora uma
// acao explicita do usuario via UI, nao um lookup implicito por SNI.
func issueCertificate(db *sql.DB, ca *localCA, rawDomain string) (*certificateResponse, error) {
	domain := normalizeDomain(rawDomain)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serialNumber, err := randomSerial()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(2, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	var dnsNames, ipAddresses []string
	if ip := net.ParseIP(domain); ip != nil {
		tpl.IPAddresses = []net.IP{ip}
		ipAddresses = []string{domain}
	} else {
		tpl.DNSNames = []string{domain}
		dnsNames = []string{domain}
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, ca.Cert, &key.PublicKey, ca.Key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	var id string
	err = db.QueryRow(
		`INSERT INTO certificates (domain, common_name, dns_names, ip_addresses, serial_number, certificate_pem, private_key_pem, issued_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		domain, domain, dnsNames, ipAddresses, serialNumber.String(),
		string(certPEM), string(keyPEM), tpl.NotBefore, tpl.NotAfter,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	log.Printf("[backend] certificado emitido para %s", domain)

	if err := syncCertificateToNginxUI(domain, string(certPEM), string(keyPEM)); err != nil {
		log.Printf("[backend] falha ao carregar certificado de %s no nginx-ui: %v", domain, err)
	}

	return &certificateResponse{
		ID: id, Domain: domain, CommonName: domain,
		DNSNames: dnsNames, IPAddresses: ipAddresses,
		IssuedAt: tpl.NotBefore.Format(time.RFC3339), ExpiresAt: tpl.NotAfter.Format(time.RFC3339),
	}, nil
}

func listCertificates(db *sql.DB, revoked bool) ([]certificateResponse, error) {
	revokedFilter := `revoked_at IS NULL`
	if revoked {
		revokedFilter = `revoked_at IS NOT NULL`
	}
	rows, err := db.Query(
		`SELECT id, domain, common_name, COALESCE(array_to_string(dns_names, ','), ''), COALESCE(array_to_string(ip_addresses, ','), ''), issued_at, expires_at, revoked_at
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
		if err := rows.Scan(&r.ID, &r.Domain, &r.CommonName, &dnsNames, &ipAddresses, &issuedAt, &expiresAt, &revokedAt); err != nil {
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
	var domain string
	if err := db.QueryRow(`SELECT domain FROM certificates WHERE id = $1`, id).Scan(&domain); err != nil {
		return "", err
	}
	_, err := db.Exec(`UPDATE certificates SET revoked_at = COALESCE(revoked_at, now()) WHERE id = $1`, id)
	return domain, err
}

func revokedCertificateDomainByID(db *sql.DB, id string) (string, error) {
	var domain string
	var revokedAt sql.NullTime
	err := db.QueryRow(`SELECT domain, revoked_at FROM certificates WHERE id = $1`, id).Scan(&domain, &revokedAt)
	if err != nil {
		return "", err
	}
	if !revokedAt.Valid {
		return "", errCertificateNotRevoked
	}
	return domain, nil
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

func certificatePEMByID(db *sql.DB, id string) (domain, certificatePEM string, err error) {
	err = db.QueryRow(`SELECT domain, certificate_pem FROM certificates WHERE id = $1`, id).Scan(&domain, &certificatePEM)
	return domain, certificatePEM, err
}

func syncIssuedCertificatesToNginxUI(db *sql.DB) error {
	rows, err := db.Query(
		`SELECT domain, certificate_pem, private_key_pem
		 FROM certificates
		 WHERE revoked_at IS NULL
		 ORDER BY created_at ASC`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	failures := 0
	for rows.Next() {
		var domain, certificatePEM, privateKeyPEM string
		if err := rows.Scan(&domain, &certificatePEM, &privateKeyPEM); err != nil {
			return err
		}
		count++
		if err := syncCertificateToNginxUI(domain, certificatePEM, privateKeyPEM); err != nil {
			failures++
			log.Printf("[backend] falha ao importar certificado existente de %s no nginx-ui: %v", domain, err)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count > 0 {
		log.Printf("[backend] sync inicial de certificados no nginx-ui concluido: total=%d falhas=%d", count, failures)
	}
	return nil
}

// splitNonEmpty separa valores de array_to_string(coluna, ',') - retorna
// nil (em vez de []string{""}) quando a coluna original era um array
// Postgres vazio ou nulo.
func splitNonEmpty(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}
