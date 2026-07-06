package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const composeProjectName = "bindnet"

func registerComposeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /hotspot/apply", handleHotspotServiceAction("restart", true))
	mux.HandleFunc("POST /hotspot/start", handleHotspotServiceAction("start", true))
	mux.HandleFunc("POST /hotspot/stop", handleHotspotServiceAction("stop", false))
	mux.HandleFunc("GET /hotspot/status", handleHotspotServiceStatus)
	mux.HandleFunc("POST /dns/apply", handleApplyServices([]string{"dns-provider"}, true))
}

func handleHotspotServiceAction(action string, acceptsRuntimeConfig bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var config map[string]string
		if acceptsRuntimeConfig {
			var err error
			config, err = decodeRuntimeConfig(r)
			if err != nil {
				http.Error(w, "corpo invalido", http.StatusBadRequest)
				return
			}
		}

		if action != "stop" {
			if err := ensureHotspotContainer(); err != nil {
				log.Printf("[worker] erro ao garantir container do hotspot: %v", err)
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			if len(config) > 0 {
				if err := applyComposeServicesWithConfig([]string{"dns-provider"}, config); err != nil {
					log.Printf("[worker] erro ao aplicar config do dns-provider: %v", err)
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
			}
		}

		output, err := execHotspotEntrypoint(action)
		if err != nil {
			log.Printf("[worker] erro ao executar hotspot %s: %v (%s)", action, err, output)
			http.Error(w, strings.TrimSpace(string(output)), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleHotspotServiceStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	containerID, running, err := serviceContainerRunning("hotspot")
	if err != nil || containerID == "" || !running {
		_ = json.NewEncoder(w).Encode(containerStatus{Name: "hotspot", Running: false, Status: "stopped"})
		return
	}
	output, err := exec.Command("docker", "exec", containerID, "/usr/local/bin/hotspot-entrypoint.sh", "status").CombinedOutput()
	if err != nil {
		log.Printf("[worker] erro ao ler status do hotspot: %v (%s)", err, output)
		_ = json.NewEncoder(w).Encode(containerStatus{Name: "hotspot", Running: false, Status: "unknown"})
		return
	}
	_, _ = w.Write(output)
}

func decodeRuntimeConfig(r *http.Request) (map[string]string, error) {
	if r.Body == nil || r.ContentLength == 0 {
		return nil, nil
	}
	var config map[string]string
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil && err != io.EOF {
		return nil, err
	}
	return config, nil
}

func ensureHotspotContainer() error {
	output, err := exec.Command("docker", composeArgs("up", "-d", "--no-build", "--no-deps", "hotspot")...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	for i := 0; i < 20; i++ {
		_, running, err := serviceContainerRunning("hotspot")
		if err == nil && running {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("container hotspot nao ficou em execucao")
}

func execHotspotEntrypoint(action string) ([]byte, error) {
	containerID, running, err := serviceContainerRunning("hotspot")
	if err != nil {
		return nil, err
	}
	if containerID == "" || !running {
		if action == "stop" {
			return nil, nil
		}
		return nil, fmt.Errorf("container hotspot nao esta em execucao")
	}
	return exec.Command("docker", "exec", containerID, "/usr/local/bin/hotspot-entrypoint.sh", action).CombinedOutput()
}

func serviceContainerRunning(service string) (string, bool, error) {
	containerID, err := composeServiceContainerID(service)
	if err != nil || containerID == "" {
		return "", false, err
	}
	output, err := exec.Command("docker", "inspect", "--format", "{{.State.Running}}", containerID).CombinedOutput()
	if err != nil {
		return containerID, false, fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return containerID, strings.TrimSpace(string(output)) == "true", nil
}

func applyComposeServicesWithConfig(services []string, config map[string]string) error {
	file, err := writeDNSRuntimeComposeOverride(config)
	if err != nil {
		return err
	}
	defer os.Remove(file)

	args := composeArgsWithFiles([]string{file}, append([]string{"up", "-d", "--no-build", "--no-deps"}, services...)...)
	output, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// handleApplyServices recria os containers informados via "docker
// compose up", unica forma de fazer um container ja existente receber
// o ambiente gerado a partir da configuracao salva pelo painel.
// --no-build evita reconstruir a imagem, so recria o container.
func handleApplyServices(services []string, acceptsRuntimeConfig bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var extraFiles []string
		var cleanup func()
		if acceptsRuntimeConfig && r.Body != nil && r.ContentLength != 0 {
			var config map[string]string
			if err := json.NewDecoder(r.Body).Decode(&config); err != nil && err != io.EOF {
				http.Error(w, "corpo invalido", http.StatusBadRequest)
				return
			}
			if len(config) > 0 {
				file, err := writeDNSRuntimeComposeOverride(config)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				extraFiles = append(extraFiles, file)
				cleanup = func() { _ = os.Remove(file) }
			}
		}
		if cleanup != nil {
			defer cleanup()
		}
		args := composeArgsWithFiles(extraFiles, append([]string{"up", "-d", "--no-build", "--no-deps"}, services...)...)
		output, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			log.Printf("[worker] erro ao aplicar config (%v): %v (%s)", services, err, output)
			http.Error(w, string(output), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func composeArgs(args ...string) []string {
	return composeArgsWithFiles(nil, args...)
}

func composeArgsWithFiles(extraFiles []string, args ...string) []string {
	base := []string{
		"compose",
		"--project-name", composeProjectName,
		"--project-directory", "/workspace",
		"--env-file", envPath(),
		"-f", "/workspace/docker-compose.services.yml",
	}
	for _, file := range extraFiles {
		base = append(base, "-f", file)
	}
	return append(base, args...)
}

func writeDNSRuntimeComposeOverride(config map[string]string) (string, error) {
	file, err := os.CreateTemp("", "bindnet-hotspot-runtime-*.yml")
	if err != nil {
		return "", err
	}
	defer file.Close()

	var b strings.Builder
	b.WriteString("services:\n")
	b.WriteString("  dns-provider:\n    environment:\n")
	for _, key := range dnsProviderRuntimeEnvKeys() {
		value, ok := config[key]
		if !ok {
			continue
		}
		quoted, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		b.WriteString("      ")
		b.WriteString(key)
		b.WriteString(": ")
		b.Write(quoted)
		b.WriteByte('\n')
	}

	if _, err := file.WriteString(b.String()); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func dnsProviderRuntimeEnvKeys() []string {
	return []string{
		"HOTSPOT_GATEWAY",
	}
}

func composeServiceContainerID(service string) (string, error) {
	output, err := exec.Command("docker", composeArgs("ps", "--all", "-q", service)...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", strings.TrimSpace(string(output)), err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			return line, nil
		}
	}
	return "", nil
}
