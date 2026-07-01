package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	dataDir       = "/data"
	caCertFile    = "/data/ca.crt"
	caKeyFile     = "/data/ca.key"
	certDir       = "/data/certs"
	defaultDomain = "localhost.local"
)

type autoridadeLocal struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
	pem  []byte
	mu   sync.Mutex
}

func main() {
	log.SetFlags(log.LstdFlags)
	log.Println("[cert-proxy] iniciando proxy TLS com CA local")

	ca, err := carregarOuCriarCA()
	if err != nil {
		log.Fatalf("[cert-proxy] erro ao preparar CA local: %v", err)
	}

	httpProxy := novoProxy(getenv("UPSTREAM_HTTP_URL", "http://nginx-ui:80"), false)
	httpsProxy := novoProxy(getenv("UPSTREAM_HTTPS_URL", "https://nginx-ui:443"), true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rotaCA(r) {
			servirCA(w, ca.pem)
			return
		}
		if r.TLS != nil {
			httpsProxy.ServeHTTP(w, r)
			return
		}
		httpProxy.ServeHTTP(w, r)
	})

	go func() {
		log.Println("[cert-proxy] HTTP ativo em :80; CA em /ca.crt")
		if err := http.ListenAndServe(":80", handler); err != nil {
			log.Fatalf("[cert-proxy] erro no HTTP: %v", err)
		}
	}()

	tlsServer := &http.Server{
		Addr:    ":443",
		Handler: handler,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
				dominio := normalizarDominio(hello.ServerName)
				return ca.certificadoPara(dominio)
			},
		},
	}

	log.Println("[cert-proxy] HTTPS ativo em :443 com emissao sob demanda")
	if err := tlsServer.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("[cert-proxy] erro no HTTPS: %v", err)
	}
}

func novoProxy(upstreamRaw string, tlsInseguro bool) *httputil.ReverseProxy {
	upstream, err := url.Parse(upstreamRaw)
	if err != nil {
		log.Fatalf("[cert-proxy] upstream invalido %q: %v", upstreamRaw, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	if tlsInseguro {
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // proxy interno para nginx-ui migrado
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[cert-proxy] erro ao encaminhar %s para %s: %v", r.Host, upstreamRaw, err)
		http.Error(w, "proxy local indisponivel", http.StatusBadGateway)
	}
	return proxy
}

func rotaCA(r *http.Request) bool {
	host := strings.ToLower(strings.Split(r.Host, ":")[0])
	return r.URL.Path == "/ca.crt" ||
		r.URL.Path == "/local-ca.crt" ||
		r.URL.Path == "/.well-known/local-ca.crt" ||
		(host == "ca.local" && (r.URL.Path == "/" || r.URL.Path == ""))
}

func servirCA(w http.ResponseWriter, pem []byte) {
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="central-local-ca.crt"`)
	_, _ = w.Write(pem)
}

func carregarOuCriarCA() (*autoridadeLocal, error) {
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, err
	}

	if existe(caCertFile) && existe(caKeyFile) {
		certPEM, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		keyPEM, err := os.ReadFile(caKeyFile)
		if err != nil {
			return nil, err
		}
		cert, key, err := parseCA(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		log.Println("[cert-proxy] CA local existente carregada")
		return &autoridadeLocal{cert: cert, key: key, pem: certPEM}, nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	serial, err := serial()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   getenv("CA_COMMON_NAME", "Central Local Development CA"),
			Organization: []string{"Central Docker Stack"},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(caCertFile, certPEM, 0644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(caKeyFile, keyPEM, 0600); err != nil {
		return nil, err
	}
	log.Println("[cert-proxy] nova CA local gerada em /data/ca.crt")
	return &autoridadeLocal{cert: tpl, key: key, pem: certPEM}, nil
}

func (a *autoridadeLocal) certificadoPara(dominio string) (*tls.Certificate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	base := nomeArquivoCert(dominio)
	certPath := filepath.Join(certDir, base+".crt")
	keyPath := filepath.Join(certDir, base+".key")
	if existe(certPath) && existe(keyPath) {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		return &cert, err
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serial, err := serial()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: dominio,
		},
		NotBefore:   now.Add(-time.Hour),
		NotAfter:    now.AddDate(2, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(dominio); ip != nil {
		tpl.IPAddresses = []net.IP{ip}
	} else {
		tpl.DNSNames = []string{dominio}
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, a.cert, &key.PublicKey, a.key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, err
	}

	log.Printf("[cert-proxy] certificado emitido para %s", dominio)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	return &cert, err
}

func parseCA(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, errors.New("certificado CA PEM invalido")
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New("chave CA PEM invalida")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func normalizarDominio(valor string) string {
	dominio := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(valor), "."))
	if dominio == "" {
		return defaultDomain
	}
	if h, _, err := net.SplitHostPort(dominio); err == nil {
		dominio = h
	}
	if ip := net.ParseIP(dominio); ip != nil {
		return dominio
	}
	for _, parte := range strings.Split(dominio, ".") {
		if parte == "" || strings.HasPrefix(parte, "-") || strings.HasSuffix(parte, "-") {
			return defaultDomain
		}
		for _, r := range parte {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return defaultDomain
			}
		}
	}
	return dominio
}

func nomeArquivoCert(dominio string) string {
	sum := sha256.Sum256([]byte(dominio))
	seguro := strings.NewReplacer(".", "_", ":", "_").Replace(dominio)
	if len(seguro) > 80 {
		seguro = seguro[:80]
	}
	return fmt.Sprintf("%s-%s", seguro, hex.EncodeToString(sum[:])[:12])
}

func serial() (*big.Int, error) {
	limite := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limite)
}

func existe(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getenv(nome, padrao string) string {
	if valor := os.Getenv(nome); valor != "" {
		return valor
	}
	return padrao
}
