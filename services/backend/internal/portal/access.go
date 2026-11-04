// Package portal decide se a raiz publica deve encaminhar o proprio host ao painel.
package portal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"time"

	"bindnet/backend/internal/platform/config"
	"bindnet/backend/internal/platform/requestip"
)

type workerCaller interface {
	Call(context.Context, string, string, any, any) error
}

func RegisterRoutes(mux *http.ServeMux, worker workerCaller) error {
	enabled, err := strconv.ParseBool(config.Getenv("PORTAL_HOST_REDIRECT_ENABLED", "true"))
	if err != nil {
		return fmt.Errorf("PORTAL_HOST_REDIRECT_ENABLED deve ser true ou false: %w", err)
	}
	mux.HandleFunc("GET /api/portal/access", accessHandler(worker, enabled))
	return nil
}

// O gateway usa auth_request: 204 conserva o portal e 403 aciona o redirect.
// Esta decisao nao concede acesso ao painel; o login continua obrigatorio.
func accessHandler(worker workerCaller, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !enabled {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ip, err := netip.ParseAddr(requestip.ClientIP(r))
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		var result struct {
			Local bool `json:"local"`
		}
		path := "/network/local-client?ip=" + url.QueryEscape(ip.Unmap().String())
		if err := worker.Call(ctx, http.MethodGet, path, nil, &result); err != nil {
			log.Printf("[backend] portal: falha ao identificar origem local: %v", err)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if result.Local {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
