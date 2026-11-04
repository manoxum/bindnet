package pangolin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"bindnet/backend/internal/audit"
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/workerapi"
)

type eventRecorder interface {
	Record(context.Context, string, string, map[string]any)
}

type service struct {
	db       *sql.DB
	worker   *workerapi.Client
	admin    *auth.Administrator
	audit    eventRecorder
	mutex    sync.Mutex
	progress publicationTracker
}

type endpoint func(http.ResponseWriter, *http.Request, *client) error

func RegisterRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB, worker *workerapi.Client, aud *audit.Client) *service {
	s := &service{db: db, worker: worker, admin: admin, audit: aud}
	s.registerAccountRoutes(mux)
	mux.HandleFunc("GET /api/tunnels/accounts/{accountId}/config", auth.RequireSession(admin, s.getConfig))
	mux.HandleFunc("PUT /api/tunnels/accounts/{accountId}/config", auth.RequireSession(admin, s.putConfig))
	s.register(mux, "POST /api/tunnels/test", s.test)
	s.register(mux, "GET /api/tunnels/sites", s.listSites)
	s.register(mux, "GET /api/tunnels/publication-status", s.publicationStatus)
	s.register(mux, "POST /api/tunnels/sites", s.createSite)
	s.register(mux, "DELETE /api/tunnels/sites/{id}", s.deleteSite)
	s.register(mux, "GET /api/tunnels/sites/{id}/status", s.connectorStatus)
	s.register(mux, "POST /api/tunnels/sites/{id}/start", s.startConnector)
	s.register(mux, "POST /api/tunnels/sites/{id}/stop", s.stopConnector)
	s.register(mux, "GET /api/tunnels/domains", s.listDomains)
	s.register(mux, "GET /api/tunnels/domain-options", s.listDomainOptions)
	s.register(mux, "GET /api/tunnels/resources", s.listResources)
	s.register(mux, "GET /api/tunnels/access-options/{kind}", s.accessOptions)
	s.register(mux, "GET /api/tunnels/resources/{id}/access/{kind}", s.resourceAccess)
	s.register(mux, "POST /api/tunnels/resources/{id}/auth-methods/{method}", s.updateAuthMethod)
	s.register(mux, "GET /api/tunnels/resources/{id}/mtls", s.mtlsCertificates)
	s.register(mux, "POST /api/tunnels/resources/{id}/mtls", s.mtlsCertificates)
	s.register(mux, "DELETE /api/tunnels/resources/{id}/mtls/{certificateId}", s.mtlsCertificates)
	s.register(mux, "GET /api/tunnels/resources/{id}/settings", s.getResourceSettings)
	s.register(mux, "PATCH /api/tunnels/resources/{id}/settings", s.updateResourceSettings)
	s.register(mux, "GET /api/tunnels/resources/{id}/rules", s.resourceRules)
	s.register(mux, "POST /api/tunnels/resources/{id}/rules", s.resourceRules)
	s.register(mux, "PATCH /api/tunnels/resources/{id}/rules/{ruleId}", s.resourceRules)
	s.register(mux, "DELETE /api/tunnels/resources/{id}/rules/{ruleId}", s.resourceRules)
	s.register(mux, "POST /api/tunnels/resources", s.createResource)
	s.register(mux, "PATCH /api/tunnels/resources/{id}", s.updateResource)
	s.register(mux, "GET /api/tunnels/resources/{id}/targets", s.listTargets)
	s.register(mux, "PATCH /api/tunnels/resources/{id}/destination", s.updateDestination)
	s.register(mux, "DELETE /api/tunnels/resources/{id}", s.deleteResource)
	return s
}

func (s *service) register(mux *http.ServeMux, path string, fn endpoint) {
	path = strings.Replace(path, "/api/tunnels/", "/api/tunnels/accounts/{accountId}/", 1)
	mux.HandleFunc(path, auth.RequireSession(s.admin, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !accountPattern.MatchString(r.PathValue("accountId")) {
			http.Error(w, "conta inválida", 400)
			return
		}
		c, err := loadConfig(r.Context(), s.db, r.PathValue("accountId"))
		if err != nil {
			http.Error(w, "não foi possível ler a configuração Pangolin", 500)
			return
		}
		if err = c.validate(); err != nil {
			http.Error(w, "configure a integração Pangolin antes de continuar", 409)
			return
		}
		if err = s.withOperationLock(fn)(w, r, newClient(c)); err != nil {
			http.Error(w, err.Error(), 502)
		}
	}))
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
func decode(w http.ResponseWriter, r *http.Request, value any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(value); err != nil {
		return fmt.Errorf("corpo da requisição inválido")
	}
	return nil
}
func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("identificador inválido")
	}
	return id, nil
}
func (s *service) record(r *http.Request, event string, id int) {
	user, _ := auth.SessionUser(r, s.admin)
	s.audit.Record(r.Context(), event, user, map[string]any{"section": "pangolin", "accountId": r.PathValue("accountId"), "id": id})
}
func (s *service) localSite(ctx context.Context, account string, id int) (Site, error) {
	sites, err := managedSites(ctx, s.db, account)
	if err != nil {
		return Site{}, err
	}
	for _, site := range sites {
		if site.SiteID == id {
			return site, nil
		}
	}
	return Site{}, fmt.Errorf("este site não é gerido pelo Bindnet")
}
