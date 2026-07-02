// certificates_routes.go expoe via HTTP a gestao de certificados
// (certificates.go/ca.go). Substitui o antigo registrarRotasCertProxy
// (so leitura). Todas as rotas exigem sessao - diferente do antigo
// cert-proxy, que servia /ca.crt anonimamente na porta 80; esse acesso
// anonimo nao existe mais (nada escuta em 80/443 depois da remocao do
// cert-proxy).
package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
)

type issueCertificateRequest struct {
	Domain string `json:"domain"`
}

func registerCertificateRoutes(mux *http.ServeMux, admin *administrator, db *sql.DB, ca *localCA, audit *auditClient) {
	mux.HandleFunc("GET /api/certificates", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		certificates, err := listCertificates(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(certificates)
	}))

	mux.HandleFunc("POST /api/certificates", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req issueCertificateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Domain == "" {
			http.Error(w, "campo 'domain' obrigatorio", http.StatusBadRequest)
			return
		}
		username, _ := sessionUser(r, admin)
		cert, err := issueCertificate(db, ca, req.Domain)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audit.record(r.Context(), "certificate_issued", username, map[string]any{"id": cert.ID, "domain": cert.Domain})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(cert)
	}))

	mux.HandleFunc("DELETE /api/certificates/{id}", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		username, _ := sessionUser(r, admin)
		domain, err := revokeCertificate(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificado nao encontrado ou ja revogado", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audit.record(r.Context(), "certificate_revoked", username, map[string]any{"id": id, "domain": domain})
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("GET /api/certificates/{id}/download", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		domain, certificatePEM, err := certificatePEMByID(db, r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificado nao encontrado", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		serveCertificate(w, certificatePEM, domain+".crt")
	}))

	mux.HandleFunc("GET /api/certificates/ca", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		serveCertificate(w, ca.CertificatePEM, "bindnet-local-ca.crt")
	}))
}

func serveCertificate(w http.ResponseWriter, pemContent, filename string) {
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte(pemContent))
}
