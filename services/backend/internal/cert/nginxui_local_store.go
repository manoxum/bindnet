// nginxui_local_store.go importa/remove certificados diretamente no
// estado compartilhado do nginx-ui (database.db + /etc/nginx/ssl),
// usado quando as credenciais do nginx-ui nao estao
// configuradas no painel (ver syncCertificateToNginxUI em nginxui_sync.go).
package cert

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func syncCertificateToNginxUILocal(name string, sanDomains []string, certificatePEM, privateKeyPEM string) error {
	certPath, keyPath := nginxUICertificatePaths(name)
	hostCertPath := nginxUIContainerPathToBackendPath(certPath)
	hostKeyPath := nginxUIContainerPathToBackendPath(keyPath)

	if err := os.MkdirAll(filepath.Dir(hostCertPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(hostCertPath, []byte(certificatePEM), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(hostKeyPath, []byte(privateKeyPEM), 0600); err != nil {
		return err
	}

	db, err := openNginxUIDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	if len(sanDomains) == 0 {
		sanDomains = []string{defaultDomain}
	}
	domainsJSON, err := json.Marshal(sanDomains)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000")

	var id int64
	err = db.QueryRow(
		`SELECT id FROM certs
		 WHERE deleted_at IS NULL
		   AND (name = ? OR (ssl_certificate_path = ? AND ssl_certificate_key_path = ?))
		 ORDER BY id DESC
		 LIMIT 1`,
		name, certPath, keyPath,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = db.Exec(
			`INSERT INTO certs (
				created_at, updated_at, name, domains, filename,
				ssl_certificate_path, ssl_certificate_key_path,
				auto_cert, challenge_method, key_type, log, sync_node_ids,
				must_staple, lego_disable_cname_support, revoke_old
			) VALUES (?, ?, ?, ?, ?, ?, ?, -1, '', 'RSA2048', '', '[]', 0, 0, 0)`,
			now, now, name, string(domainsJSON), name, certPath, keyPath,
		)
		if err != nil {
			return err
		}
		log.Printf("[backend] certificado %s importado no nginx-ui via database local", name)
		return nil
	}
	if err != nil {
		return err
	}

	_, err = db.Exec(
		`UPDATE certs
		 SET updated_at = ?, domains = ?, filename = ?,
		     ssl_certificate_path = ?, ssl_certificate_key_path = ?,
		     auto_cert = -1, key_type = 'RSA2048'
		 WHERE id = ?`,
		now, string(domainsJSON), name, certPath, keyPath, id,
	)
	if err != nil {
		return err
	}
	log.Printf("[backend] certificado %s atualizado no nginx-ui via database local", name)
	return nil
}

func removeCertificateFromNginxUI(name string) error {
	certPath, keyPath := nginxUICertificatePaths(name)
	db, err := openNginxUIDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	now := time.Now().UTC().Format("2006-01-02 15:04:05.000")
	result, err := db.Exec(
		`UPDATE certs
		 SET updated_at = ?, deleted_at = ?
		 WHERE deleted_at IS NULL
		   AND (name = ? OR (ssl_certificate_path = ? AND ssl_certificate_key_path = ?))`,
		now, now, name, certPath, keyPath,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if err := os.RemoveAll(filepath.Dir(nginxUIContainerPathToBackendPath(certPath))); err != nil {
		return err
	}

	if rowsAffected == 0 {
		log.Printf("[backend] certificado %s nao existia na lista do nginx-ui; arquivos locais limpos se existiam", name)
		return nil
	}
	log.Printf("[backend] certificado %s removido do nginx-ui", name)
	return nil
}

// SyncIssuedCertificatesToNginxUI roda no boot do backend, reimportando
// os certificados ja emitidos (nao revogados) - cobre o caso de o
// database.db do nginx-ui ter sido recriado sem eles.
func SyncIssuedCertificatesToNginxUI(db *sql.DB) error {
	rows, err := db.Query(
		`SELECT name, COALESCE(array_to_string(dns_names, ','), ''), COALESCE(array_to_string(ip_addresses, ','), ''), certificate_pem, private_key_pem
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
		var name, dnsNames, ipAddresses, certificatePEM, privateKeyPEM string
		if err := rows.Scan(&name, &dnsNames, &ipAddresses, &certificatePEM, &privateKeyPEM); err != nil {
			return err
		}
		sanDomains := append(splitNonEmpty(dnsNames), splitNonEmpty(ipAddresses)...)
		count++
		if err := syncCertificateToNginxUI(db, name, sanDomains, certificatePEM, privateKeyPEM); err != nil {
			failures++
			log.Printf("[backend] falha ao importar certificado existente %s no nginx-ui: %v", name, err)
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

func openNginxUIDatabase() (*sql.DB, error) {
	dbPath := filepath.Join(nginxUIBackendDataPath, "database.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database do nginx-ui indisponivel em %s: %w", dbPath, err)
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func nginxUICertificatePaths(name string) (certPath, keyPath string) {
	dir := filepath.ToSlash(filepath.Join(nginxUIContainerNginxConfPath, "ssl", nginxUICertificateDirName(name)))
	return dir + "/server.crt", dir + "/server.key"
}

func nginxUIContainerPathToBackendPath(path string) string {
	return filepath.Join(nginxUIBackendConfigPath, strings.TrimPrefix(path, nginxUIContainerNginxConfPath))
}

// nginxUICertificateDirName sanitiza ":" (porta) e "*" (dominio
// curinga, ex.: "*.mydomain") para uso como nome de diretorio.
func nginxUICertificateDirName(name string) string {
	return strings.NewReplacer(":", "_", "*", "_").Replace(name)
}
