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
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
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

// syncCertificateToNginxUI cadastra o certificado no nginx-ui gravando o
// PEM no mesmo caminho/convencao ja usado pelos sites existentes
// (ssl/<dominio>/server.crt|key, dentro do proprio /etc/nginx do
// container nginx-ui) - e o nginx-ui quem grava o arquivo, o backend so
// chama a API dele (nunca escreve direto no volume nginx_config).
func syncCertificateToNginxUI(domain, certificatePEM, privateKeyPEM string) error {
	if !nginxUIConfigured() {
		return nil
	}

	token, err := nginxUILogin()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]string{
		"name":                     domain,
		"ssl_certificate_path":     "/etc/nginx/ssl/" + domain + "/server.crt",
		"ssl_certificate_key_path": "/etc/nginx/ssl/" + domain + "/server.key",
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
