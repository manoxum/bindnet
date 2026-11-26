package cert

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/publication"
	"bindnet/backend/internal/settings"
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
)

var integrationCertificateMutex sync.Mutex

// RegisterIntegrationRoutes fornece uma chave limitada a emissao de certificados.
func RegisterIntegrationRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, ca *localCA, aud *audit.Client) {
	registerIntegrationTokenRoutes(mux, admin, db, aud)
	mux.HandleFunc("GET /api/integrations/v1/status", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := requireIntegrationToken(w, r, db); !ok {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"bindnet","version":1}`))
	})
	mux.HandleFunc("POST /api/integrations/v1/certificates", func(w http.ResponseWriter, r *http.Request) {
		tokenID, ok := requireIntegrationToken(w, r, db)
		if !ok {
			return
		}
		var req struct {
			Name    string   `json:"name"`
			Domains []string `json:"domains"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&req); err != nil {
			http.Error(w, "pedido inválido", 400)
			return
		}
		check := publication.Request{Name: req.Name, Domains: req.Domains, Alias: "integration.wnet", Port: 80, SSL: "bindnet"}
		if err := check.Validate(); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		integrationCertificateMutex.Lock()
		defer integrationCertificateMutex.Unlock()
		exists, err := activeCertificateUsesName(db, check.Name)
		if err != nil {
			http.Error(w, "falha ao consultar certificado", 500)
			return
		}
		ownershipKey := "INTEGRATION_CERT:" + tokenID + ":" + check.Name
		if exists {
			owned, err := settings.Get(r.Context(), db, ownershipKey)
			if err != nil || owned == "" {
				http.Error(w, "nome de certificado já usado fora da integração", 409)
				return
			}
		} else {
			issued, err := issueCertificateWithSync(db, ca, check.Name, check.Domains, 1, "years", false)
			if err != nil {
				http.Error(w, "falha ao emitir certificado", 500)
				return
			}
			if _, err = db.ExecContext(r.Context(), `INSERT INTO panel_config (key,value,updated_at) VALUES ($1,$2,CURRENT_TIMESTAMP) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value,updated_at=CURRENT_TIMESTAMP`, ownershipKey, issued.ID); err != nil {
				http.Error(w, "falha ao registar certificado da integração", 500)
				return
			}
		}
		var certPEM, keyPEM string
		err = db.QueryRowContext(r.Context(), `SELECT certificate_pem,private_key_pem FROM certificates WHERE name=$1 AND id::text=(SELECT value FROM panel_config WHERE key=$3) AND revoked_at IS NULL AND expires_at>CURRENT_TIMESTAMP AND dns_names @> $2::text[] ORDER BY created_at DESC LIMIT 1`, check.Name, check.Domains, ownershipKey).Scan(&certPEM, &keyPEM)
		if err != nil {
			http.Error(w, "o certificado existente não cobre estes domínios ou expirou; reemita no Bindnet", 409)
			return
		}
		aud.Record(r.Context(), "integration_certificate_issued", "integration:"+tokenID, map[string]any{"name": check.Name, "domains": check.Domains})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"certificatePem": certPEM, "privateKeyPem": keyPEM})
	})
}
