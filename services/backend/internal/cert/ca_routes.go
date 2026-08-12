package cert

import (
	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/workerapi"
	"encoding/json"
	"net/http"
)

type installLocalCARequest struct {
	CertificatePEM string `json:"certificatePem,omitempty"`
}

type installLocalCAWorkerRequest struct {
	CertificatePEM string `json:"certificatePem"`
}

type browserTrustResult struct {
	Store     string `json:"store"`
	Path      string `json:"path"`
	Installed bool   `json:"installed"`
	Error     string `json:"error,omitempty"`
}

type installLocalCAResponse struct {
	Path          string               `json:"path"`
	Output        string               `json:"output,omitempty"`
	BrowserStores []browserTrustResult `json:"browserStores,omitempty"`
}

func registerCARoutes(mux *http.ServeMux, admin *auth.Administrator, ca *localCA, worker *workerapi.Client, auditClient *audit.Client) {
	mux.HandleFunc("GET /api/certificates/ca", auth.RequireSession(admin, func(w http.ResponseWriter, _ *http.Request) {
		serveCertificate(w, ca.CertificatePEM, "bindnet-local-ca.crt")
	}))
	mux.HandleFunc("GET /api/mesh/ca", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		serveCertificate(w, ca.CertificatePEM, "bindnet-local-ca.crt")
	})
	mux.HandleFunc("POST /api/certificates/ca/install-local", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		username, _ := auth.SessionUser(r, admin)
		var req installLocalCARequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		certificatePEM := ca.CertificatePEM
		if req.CertificatePEM != "" {
			certificatePEM = req.CertificatePEM
		}
		var response installLocalCAResponse
		if err := worker.Call(r.Context(), http.MethodPost, "/ca/install-local", installLocalCAWorkerRequest{CertificatePEM: certificatePEM}, &response); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		auditClient.Record(r.Context(), "ca_installed_local", username, map[string]any{"path": response.Path})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func serveCertificate(w http.ResponseWriter, pemContent, filename string) {
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte(pemContent))
}
