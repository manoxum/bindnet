package main

import (
	"encoding/json"
	"net/http"
)

var monitoredContainers = []struct {
	Key       string
	Container string
}{
	{"hotspot", "central-hotspot"},
	{"dns", "central-dns-provider"},
	{"nginxUi", "central-nginx-ui"},
	{"postgres", "central-postgres"},
	{"mongo", "central-mongo"},
	{"minio", "central-minio"},
}

type serviceStatus struct {
	Key       string `json:"key"`
	Running   bool   `json:"running"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt,omitempty"`
}

// registerDashboardRoutes agrega o status dos containers monitorados do
// stack numa unica chamada, usada pela tela inicial do painel.
func registerDashboardRoutes(mux *http.ServeMux, worker *workerClient, admin *administrator) {
	mux.HandleFunc("GET /api/dashboard", requireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		result := make([]serviceStatus, 0, len(monitoredContainers))
		for _, item := range monitoredContainers {
			var status struct {
				Running   bool   `json:"rodando"`
				Status    string `json:"status"`
				StartedAt string `json:"iniciadoEm"`
			}
			if err := worker.call(r.Context(), http.MethodGet, "/containers/"+item.Container+"/status", nil, &status); err != nil {
				result = append(result, serviceStatus{Key: item.Key, Status: "indisponivel"})
				continue
			}
			result = append(result, serviceStatus{
				Key:       item.Key,
				Running:   status.Running,
				Status:    status.Status,
				StartedAt: status.StartedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}))
}
