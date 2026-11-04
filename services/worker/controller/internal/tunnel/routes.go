package tunnel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var lifecycle sync.Mutex
var accountPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type credentials struct {
	Endpoint string `json:"endpoint"`
	NewtID   string `json:"newtId"`
	Secret   string `json:"secret"`
}

func (c credentials) validate() error {
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("endpoint Pangolin inválido")
	}
	if c.NewtID == "" || c.Secret == "" || len(c.NewtID) > 256 || len(c.Secret) > 1024 || strings.ContainsAny(c.NewtID+c.Secret, "\r\n\x00") {
		return fmt.Errorf("credenciais Newt inválidas")
	}
	return nil
}

func RegisterRoutes(mux *http.ServeMux) {
	d := newDockerClient()
	mux.HandleFunc("GET /tunnels/{accountId}/{id}/status", handler(d, "status"))
	mux.HandleFunc("POST /tunnels/{accountId}/{id}/start", handler(d, "start"))
	mux.HandleFunc("POST /tunnels/{accountId}/{id}/stop", handler(d, "stop"))
	mux.HandleFunc("DELETE /tunnels/{accountId}/{id}", handler(d, "delete"))
}

func handler(d *dockerClient, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil || id <= 0 {
			http.Error(w, "site inválido", 400)
			return
		}
		account := r.PathValue("accountId")
		if !accountPattern.MatchString(account) {
			http.Error(w, "conta inválida", 400)
			return
		}
		name := containerName(account, id)
		var c credentials
		if action == "start" {
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
			decoder.DisallowUnknownFields()
			if err = decoder.Decode(&c); err != nil {
				http.Error(w, "corpo inválido", 400)
				return
			}
			if err = c.validate(); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		}
		lifecycle.Lock()
		defer lifecycle.Unlock()
		state, err := d.inspect(r.Context(), name)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		switch action {
		case "start":
			if state.ID == "" {
				if err = d.ensureImage(r.Context()); err == nil {
					_, err = d.call(r.Context(), "POST", "/containers/create?name="+name, containerConfig(c), nil)
				}
			}
			if err == nil && !state.State.Running {
				_, err = d.call(r.Context(), "POST", "/containers/"+name+"/start", nil, nil)
			}
		case "stop":
			if state.ID != "" && state.State.Running {
				_, err = d.call(r.Context(), "POST", "/containers/"+name+"/stop?t=10", nil, nil)
			}
		case "delete":
			if state.ID != "" {
				_, err = d.call(r.Context(), "DELETE", "/containers/"+name+"?force=true", nil, nil)
			}
		case "status":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"running": state.State.Running, "status": state.State.Status, "image": image, "health": state.State.Health.Status})
			return
		}
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func containerName(account string, id int) string {
	if account == "legacy" {
		return "bindnet-newt-" + strconv.Itoa(id)
	}
	return "bindnet-newt-" + account + "-" + strconv.Itoa(id)
}
