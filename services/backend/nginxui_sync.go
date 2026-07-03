// nginxui_sync.go carrega automaticamente, via API HTTP do nginx-ui, um
// certificado recem-emitido pela CA local (certificates.go) - sem isso o
// usuario precisaria copiar/colar o PEM manualmente na tela de
// certificados do nginx-ui. O login da API do nginx-ui exige um
// handshake proprio (troca de chave publica RSA + payload criptografado
// com RSA-PKCS1v15, ver github.com/0xJacky/Nginx-UI internal/crypto e
// internal/middleware/encrypted_params.go) - replicado aqui porque nao
// ha outra forma documentada de autenticar contra ele por HTTP.
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	nginxUIBackendDataPath        = "/nginx-ui-data"
	nginxUIBackendConfigPath      = "/nginx-config"
	nginxUIContainerNginxConfPath = "/etc/nginx"
)

type nginxUIPublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

type nginxUILoginResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Token   string `json:"token"`
}

// nginxUIConfigured indica se as credenciais da API do nginx-ui foram
// preenchidas em .env - o sync e um extra opcional, nao deve impedir a
// emissao do certificado (que ja e persistida no Postgres) se o usuario
// nao configurou isso.
func nginxUIConfigured() bool {
	return getenv("NGINX_UI_USERNAME", "") != "" && getenv("NGINX_UI_PASSWORD", "") != ""
}

func nginxUIBaseURL() string {
	return getenv("NGINX_UI_URL", "http://nginx-ui:9000")
}

// nginxUILogin replica o fluxo de login criptografado do nginx-ui:
// 1) busca a chave publica RSA temporaria da instancia; 2) criptografa
// usuario/senha com ela; 3) troca isso por um token JWT.
func nginxUILogin() (string, error) {
	publicKey, err := fetchNginxUIPublicKey()
	if err != nil {
		return "", fmt.Errorf("chave publica do nginx-ui: %w", err)
	}

	credentials, err := json.Marshal(map[string]string{
		"name":     getenv("NGINX_UI_USERNAME", ""),
		"password": getenv("NGINX_UI_PASSWORD", ""),
	})
	if err != nil {
		return "", err
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, credentials)
	if err != nil {
		return "", err
	}

	body, err := json.Marshal(map[string]string{
		"encrypted_params": base64.StdEncoding.EncodeToString(encrypted),
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(nginxUIBaseURL()+"/api/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var login nginxUILoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&login); err != nil {
		return "", err
	}
	if login.Token == "" {
		return "", fmt.Errorf("login no nginx-ui falhou: %s", login.Message)
	}
	return login.Token, nil
}

func fetchNginxUIPublicKey() (*rsa.PublicKey, error) {
	body, err := json.Marshal(map[string]any{
		"timestamp":   time.Now().Unix(),
		"fingerprint": "bindnet-backend",
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(nginxUIBaseURL()+"/api/crypto/public_key", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var parsed nginxUIPublicKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	block, _ := pem.Decode([]byte(parsed.PublicKey))
	if block == nil {
		return nil, errors.New("chave publica do nginx-ui invalida")
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

// syncCertificateToNginxUI cadastra o certificado no nginx-ui. Quando as
// credenciais do nginx-ui estiverem configuradas, usa a API oficial. Sem
// credenciais, importa diretamente no estado compartilhado do nginx-ui
// (database.db + /etc/nginx/ssl) para que o certificado apareca na tela
// /#/certificates/list logo apos a emissao no painel Bindnet.
func syncCertificateToNginxUI(domain, certificatePEM, privateKeyPEM string) error {
	if nginxUIConfigured() {
		if err := syncCertificateToNginxUIAPI(domain, certificatePEM, privateKeyPEM); err == nil {
			return nil
		} else {
			log.Printf("[backend] sync via API do nginx-ui falhou para %s, tentando importacao local: %v", domain, err)
		}
	}

	return syncCertificateToNginxUILocal(domain, certificatePEM, privateKeyPEM)
}

func syncCertificateToNginxUIAPI(domain, certificatePEM, privateKeyPEM string) error {
	token, err := nginxUILogin()
	if err != nil {
		return err
	}

	certPath, keyPath := nginxUICertificatePaths(domain)
	payload, err := json.Marshal(map[string]string{
		"name":                     domain,
		"ssl_certificate_path":     certPath,
		"ssl_certificate_key_path": keyPath,
		"ssl_certificate":          certificatePEM,
		"ssl_certificate_key":      privateKeyPEM,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, nginxUIBaseURL()+"/api/certs", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("nginx-ui respondeu %d ao cadastrar certificado", resp.StatusCode)
	}
	log.Printf("[backend] certificado de %s carregado no nginx-ui", domain)
	return nil
}

func syncCertificateToNginxUILocal(domain, certificatePEM, privateKeyPEM string) error {
	certPath, keyPath := nginxUICertificatePaths(domain)
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

	domainsJSON, err := json.Marshal([]string{domain})
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
		domain, certPath, keyPath,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = db.Exec(
			`INSERT INTO certs (
				created_at, updated_at, name, domains, filename,
				ssl_certificate_path, ssl_certificate_key_path,
				auto_cert, challenge_method, key_type, log, sync_node_ids,
				must_staple, lego_disable_cname_support, revoke_old
			) VALUES (?, ?, ?, ?, ?, ?, ?, -1, '', 'RSA2048', '', '[]', 0, 0, 0)`,
			now, now, domain, string(domainsJSON), domain, certPath, keyPath,
		)
		if err != nil {
			return err
		}
		log.Printf("[backend] certificado de %s importado no nginx-ui via database local", domain)
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
		now, string(domainsJSON), domain, certPath, keyPath, id,
	)
	if err != nil {
		return err
	}
	log.Printf("[backend] certificado de %s atualizado no nginx-ui via database local", domain)
	return nil
}

func removeCertificateFromNginxUI(domain string) error {
	certPath, keyPath := nginxUICertificatePaths(domain)
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
		now, now, domain, certPath, keyPath,
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
		log.Printf("[backend] certificado de %s nao existia na lista do nginx-ui; arquivos locais limpos se existiam", domain)
		return nil
	}
	log.Printf("[backend] certificado de %s removido do nginx-ui", domain)
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

func nginxUICertificatePaths(domain string) (certPath, keyPath string) {
	dir := filepath.ToSlash(filepath.Join(nginxUIContainerNginxConfPath, "ssl", nginxUICertificateDirName(domain)))
	return dir + "/server.crt", dir + "/server.key"
}

func nginxUIContainerPathToBackendPath(path string) string {
	return filepath.Join(nginxUIBackendConfigPath, strings.TrimPrefix(path, nginxUIContainerNginxConfPath))
}

func nginxUICertificateDirName(domain string) string {
	return strings.NewReplacer(":", "_").Replace(domain)
}
