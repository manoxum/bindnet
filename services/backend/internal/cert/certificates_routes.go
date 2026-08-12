// certificates_routes.go expoe via HTTP a gestao de certificados
// (certificates.go/ca.go). Substitui o antigo registrarRotasCertProxy
// (so leitura). Quase todas as rotas exigem sessao - excecao
// deliberada: GET /api/mesh/ca, que devolve so o certificado publico
// da CA (nunca a chave privada) sem autenticacao, para que outros nos
// da malha Bindnet consigam buscar essa CA e o usuario decidir se
// confia nela, igual ao antigo cert-proxy (que servia /ca.crt
// anonimamente na porta 80) - so que agora escopado a essa unica rota.
package cert

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/workerapi"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
)

// issueCertificateRequest.Domains vira um unico certificado com todos
// os dominios/IPs como SAN - ver comentario de issueCertificate em
// certificates.go. Aceita dominio curinga (ex.: "*.mydomain") em
// qualquer posicao da lista. ValidityQuantity/ValidityUnit sao
// opcionais - vazios/invalidos caem para o padrao de 2 anos.
type issueCertificateRequest struct {
	Name             string   `json:"name"`
	Domains          []string `json:"domains"`
	ValidityQuantity int      `json:"validityQuantity,omitempty"`
	ValidityUnit     string   `json:"validityUnit,omitempty"`
}

type reissueCertificateRequest struct {
	Domains          []string `json:"domains"`
	ValidityQuantity int      `json:"validityQuantity,omitempty"`
	ValidityUnit     string   `json:"validityUnit,omitempty"`
}

func RegisterCertificateRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, ca *localCA, worker *workerapi.Client, audit *audit.Client) {
	mux.HandleFunc("GET /api/certificates", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		certificates, err := listCertificates(db, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(certificates)
	}))

	mux.HandleFunc("GET /api/certificates/revoked", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		certificates, err := listCertificates(db, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(certificates)
	}))

	mux.HandleFunc("POST /api/certificates", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req issueCertificateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "corpo da requisicao invalido", http.StatusBadRequest)
			return
		}
		name, err := normalizeCertificateName(req.Name)
		if err != nil {
			http.Error(w, "campo 'name' obrigatorio e deve ter no maximo 100 caracteres, sem barras", http.StatusBadRequest)
			return
		}
		if !hasNonEmptyDomain(req.Domains) {
			http.Error(w, "campo 'domains' obrigatorio (ao menos um dominio ou IP)", http.StatusBadRequest)
			return
		}
		nameInUse, err := activeCertificateUsesName(db, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if nameInUse {
			http.Error(w, "ja existe um certificado ativo com esse nome", http.StatusConflict)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		cert, err := issueCertificate(db, ca, name, req.Domains, req.ValidityQuantity, req.ValidityUnit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audit.Record(r.Context(), "certificate_issued", username, map[string]any{
			"id": cert.ID, "name": cert.Name, "domain": cert.Domain, "domains": append(cert.DNSNames, cert.IPAddresses...),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(cert)
	}))

	mux.HandleFunc("PUT /api/certificates/{id}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req reissueCertificateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !hasNonEmptyDomain(req.Domains) {
			http.Error(w, "campo 'domains' obrigatorio (ao menos um dominio ou IP)", http.StatusBadRequest)
			return
		}
		username, _ := auth.SessionUser(r, admin)
		cert, err := reissueCertificate(db, ca, r.PathValue("id"), req.Domains, req.ValidityQuantity, req.ValidityUnit)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificado nao encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, errCertificateRevoked) {
			http.Error(w, "certificado revogado nao pode ser reemitido", http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audit.Record(r.Context(), "certificate_reissued", username, map[string]any{
			"replacesId": r.PathValue("id"), "id": cert.ID, "name": cert.Name,
			"domain": cert.Domain, "domains": append(cert.DNSNames, cert.IPAddresses...),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(cert)
	}))

	mux.HandleFunc("DELETE /api/certificates/{id}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		username, _ := auth.SessionUser(r, admin)
		name, err := revokeCertificate(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificado nao encontrado", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := removeCertificateFromNginxUI(name); err != nil {
			http.Error(w, "certificado revogado no Bindnet, mas falhou ao remover do nginx-ui: "+err.Error(), http.StatusInternalServerError)
			return
		}
		audit.Record(r.Context(), "certificate_revoked", username, map[string]any{"id": id, "name": name})
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("DELETE /api/certificates/{id}/permanent", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		username, _ := auth.SessionUser(r, admin)
		name, err := revokedCertificateNameByID(db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificado nao encontrado", http.StatusNotFound)
			return
		}
		if errors.Is(err, errCertificateNotRevoked) {
			http.Error(w, "apenas certificados revogados podem ser eliminados permanentemente", http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nameInUse, err := activeCertificateUsesName(db, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !nameInUse {
			if err := removeCertificateFromNginxUI(name); err != nil {
				http.Error(w, "falhou ao limpar certificado no nginx-ui: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if err := permanentlyDeleteCertificate(db, id); errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificado revogado nao encontrado", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		audit.Record(r.Context(), "certificate_permanently_deleted", username, map[string]any{"id": id, "name": name})
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("GET /api/certificates/{id}/download", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		name, certificatePEM, err := certificatePEMByID(db, r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "certificado nao encontrado", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		serveCertificate(w, certificatePEM, name+".crt")
	}))

	registerCARoutes(mux, admin, ca, worker, audit)
}
