package hotspot

import (
	"bindnet/backend/internal/auth"
	"bindnet/backend/internal/hotspot/store"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type contentCategoryToggleRequest struct {
	Enabled bool `json:"enabled"`
}

// RegisterHotspotContentCategoryRoutes expoe o catalogo de categorias e
// o gatilho de sync manual. O sync roda em background (as listas
// publicas sao grandes) e responde 202.
func RegisterHotspotContentCategoryRoutes(mux *http.ServeMux, admin *auth.Administrator, db *sql.DB) {
	mux.HandleFunc("GET /api/hotspot/content-categories", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		cats, err := store.ListContentCategories(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, cats)
	}))

	mux.HandleFunc("PATCH /api/hotspot/content-categories/{slug}", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		var req contentCategoryToggleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "corpo invalido", http.StatusBadRequest)
			return
		}
		found, err := store.SetContentCategoryEnabled(db, r.PathValue("slug"), req.Enabled)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !found {
			http.Error(w, "categoria nao encontrada", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	mux.HandleFunc("POST /api/hotspot/content-categories/{slug}/sync", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		cats, err := store.ListContentCategories(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var target *store.ContentCategory
		for i := range cats {
			if cats[i].Slug == slug {
				target = &cats[i]
				break
			}
		}
		if target == nil {
			http.Error(w, "categoria nao encontrada", http.StatusNotFound)
			return
		}
		category := *target
		go func() {
			if err := syncContentCategory(db, category); err != nil {
				log.Printf("[backend] sync manual da categoria %s falhou: %v", category.Slug, err)
			}
		}()
		w.WriteHeader(http.StatusAccepted)
	}))
}
