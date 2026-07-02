package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// allowedContainers e a lista fechada de containers que o worker aceita
// controlar. Qualquer outro nome (inclusive vindo de um path malicioso) e
// rejeitado - o worker nunca executa "exec arbitrario".
var allowedContainers = map[string]bool{
	"central-hotspot":      true,
	"central-dns-provider": true,
	"central-nginx-ui":     true,
	"central-postgres":     true,
	"central-mongo":        true,
	"central-minio":        true,
}

func registerContainerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /containers/{name}/start", handleContainerAction("start"))
	mux.HandleFunc("POST /containers/{name}/stop", handleContainerAction("stop"))
	mux.HandleFunc("POST /containers/{name}/restart", handleContainerAction("restart"))
	mux.HandleFunc("GET /containers/{name}/status", handleContainerStatus)
	mux.HandleFunc("GET /containers/{name}/logs", handleContainerLogs)
}

func handleContainerAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if !allowedContainers[name] {
			http.Error(w, "container nao permitido", http.StatusForbidden)
			return
		}
		output, err := exec.Command("docker", action, name).CombinedOutput()
		if err != nil {
			log.Printf("[worker] erro ao executar docker %s %s: %v (%s)", action, name, err, output)
			http.Error(w, string(output), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type containerStatus struct {
	Name      string `json:"name"`
	Running   bool   `json:"running"`
	Status    string `json:"status"`
	Image     string `json:"image"`
	StartedAt string `json:"startedAt,omitempty"`
}

func handleContainerStatus(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	w.Header().Set("Content-Type", "application/json")
	if !allowedContainers[name] {
		http.Error(w, "container nao permitido", http.StatusForbidden)
		return
	}

	format := "{{.State.Running}}|{{.State.Status}}|{{.Config.Image}}|{{.State.StartedAt}}"
	output, err := exec.Command("docker", "inspect", "--format", format, name).CombinedOutput()
	if err != nil {
		_ = json.NewEncoder(w).Encode(containerStatus{Name: name, Running: false, Status: "ausente"})
		return
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 4)
	if len(parts) != 4 {
		http.Error(w, "saida inesperada do docker inspect", http.StatusInternalServerError)
		return
	}
	running, _ := strconv.ParseBool(parts[0])
	response := containerStatus{Name: name, Running: running, Status: parts[1], Image: parts[2]}
	if running {
		response.StartedAt = parts[3]
	}
	_ = json.NewEncoder(w).Encode(response)
}

// flushWriter repassa cada escrita imediatamente ao cliente, necessario
// para transmitir "docker logs -f" em tempo real em vez de bufferizar.
type flushWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return n, err
}

func handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !allowedContainers[name] {
		http.Error(w, "container nao permitido", http.StatusForbidden)
		return
	}

	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}
	args := []string{"logs", "--tail", tail}
	if r.URL.Query().Get("follow") == "true" {
		args = append(args, "-f")
	}
	args = append(args, name)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	flusher, _ := w.(http.Flusher)

	cmd := exec.CommandContext(r.Context(), "docker", args...)
	cmd.Stdout = flushWriter{w, flusher}
	cmd.Stderr = flushWriter{w, flusher}
	if err := cmd.Run(); err != nil {
		log.Printf("[worker] docker logs %s encerrou: %v", name, err)
	}
}
