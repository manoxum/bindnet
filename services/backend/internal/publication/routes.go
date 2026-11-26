package publication

import (
	"bindnet/backend/internal/auth"
	"encoding/json"
	"net/http"
)

// RegisterRoutes expoe a validacao e a previsualizacao autenticadas da integracao.
func RegisterRoutes(mux *http.ServeMux, admin *auth.Administrator) {
	mux.HandleFunc("POST /api/integrations/publications/preview", auth.RequireSession(admin, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request Request
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "requisição inválida", http.StatusBadRequest)
			return
		}
		if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		certificate := "/etc/nginx/ssl/" + request.Name + "/server.crt"
		key := "/etc/nginx/ssl/" + request.Name + "/server.key"
		config, err := Render(request, certificate, key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"config": config, "preview": true})
	}))
}
