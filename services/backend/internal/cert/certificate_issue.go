package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"log"
	"net"
	"time"
)

func issueCertificate(db *sql.DB, ca *localCA, name string, domains []string, quantity int, unit string) (*certificateResponse, error) {
	return issueCertificateWithSync(db, ca, name, domains, quantity, unit, true)
}

// issueCertificate sempre cria uma linha nova (sem cache de arquivo por
// dominio, ao contrario do antigo certificadoPara) - emitir e agora uma
// acao explicita do usuario via UI, nao um lookup implicito por SNI.
// rawDomains vira um unico certificado com todos os dominios/IPs como
// SAN (Subject Alternative Name) - a primeira entrada normalizada e o
// dominio/CN primario (coluna "domain"). name e o nome amigavel usado
// nas listagens e na importacao para o nginx-ui. validityQuantity e
// validityUnit (days|weeks|months|years) definem NotAfter - invalidas
// caem para o padrao de 2 anos (ver normalizeValidityPeriod), e o
// resultado nunca ultrapassa a validade da propria CA.
func issueCertificateWithSync(db *sql.DB, ca *localCA, name string, rawDomains []string, validityQuantity int, validityUnit string, syncNginx bool) (*certificateResponse, error) {
	domains := normalizeDomainList(rawDomains)
	primary := domains[0]
	quantity, unit := normalizeValidityPeriod(validityQuantity, validityUnit)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serialNumber, err := randomSerial()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	notAfter := certificateExpiry(now, quantity, unit)
	if notAfter.After(ca.Cert.NotAfter) {
		notAfter = ca.Cert.NotAfter
	}
	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: primary},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	var dnsNames, ipAddresses []string
	for _, domain := range domains {
		if ip := net.ParseIP(domain); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
			ipAddresses = append(ipAddresses, domain)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, domain)
			dnsNames = append(dnsNames, domain)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, ca.Cert, &key.PublicKey, ca.Key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	var id string
	err = db.QueryRow(
		`INSERT INTO certificates (name, domain, common_name, dns_names, ip_addresses, serial_number, certificate_pem, private_key_pem, issued_at, expires_at, validity_quantity, validity_unit)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`,
		name, primary, primary, dnsNames, ipAddresses, serialNumber.String(),
		string(certPEM), string(keyPEM), tpl.NotBefore, tpl.NotAfter,
		quantity, unit,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	log.Printf("[backend] certificado %s emitido para %s (%d dominio(s)/IP(s))", name, primary, len(domains))

	if syncNginx {
		if err := syncCertificateToNginxUI(db, name, domains, string(certPEM), string(keyPEM)); err != nil {
			log.Printf("[backend] falha ao carregar certificado %s no nginx-ui: %v", name, err)
		}
	}

	return &certificateResponse{
		ID: id, Name: name, Domain: primary, CommonName: primary,
		DNSNames: dnsNames, IPAddresses: ipAddresses,
		IssuedAt: tpl.NotBefore.Format(time.RFC3339), ExpiresAt: tpl.NotAfter.Format(time.RFC3339),
		ValidityQuantity: quantity, ValidityUnit: unit,
	}, nil
}
