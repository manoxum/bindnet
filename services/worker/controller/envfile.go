package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// envPath e o .env compartilhado do repositorio, montado em
// /workspace (raiz do repo) dentro do container do worker.
const envPath = "/workspace/.env"

// envSections restringe quais chaves cada "section" pode ler/alterar - o
// backend nunca manda uma chave arbitraria, so uma dessas secoes.
var envSections = map[string][]string{
	"hotspot": {
		"WIFI_INTERFACE", "INTERNET_INTERFACE", "WIFI_SSID", "WIFI_PASSWORD",
		"WIFI_COUNTRY", "WIFI_CHANNEL", "WIFI_FREQ_BAND", "WIFI_CHANNEL_CANDIDATES",
		"HOTSPOT_GATEWAY", "HOTSPOT_CIDR",
	},
	"dns": {"DNS_LOCAL_TLDS"},
}

func registerEnvRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /env", handleEnvGet)
	mux.HandleFunc("PATCH /env", handleEnvPatch)
}

func handleEnvGet(w http.ResponseWriter, r *http.Request) {
	section := r.URL.Query().Get("section")
	keys, ok := envSections[section]
	if !ok {
		http.Error(w, "secao invalida", http.StatusBadRequest)
		return
	}
	values, err := readEnvValues(envPath, keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(values)
}

func handleEnvPatch(w http.ResponseWriter, r *http.Request) {
	section := r.URL.Query().Get("section")
	allowedKeys, ok := envSections[section]
	if !ok {
		http.Error(w, "secao invalida", http.StatusBadRequest)
		return
	}

	var values map[string]string
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		http.Error(w, "corpo invalido", http.StatusBadRequest)
		return
	}

	allowed := map[string]bool{}
	for _, key := range allowedKeys {
		allowed[key] = true
	}
	for key := range values {
		if !allowed[key] {
			http.Error(w, fmt.Sprintf("chave '%s' nao pode ser alterada na secao '%s'", key, section), http.StatusForbidden)
			return
		}
	}

	if err := updateEnvKeys(envPath, values); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// readEnvValues le o .env e devolve so as chaves pedidas.
func readEnvValues(path string, keys []string) (map[string]string, error) {
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[key] = true
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := parseEnvLine(scanner.Text())
		if ok && wanted[key] {
			values[key] = value
		}
	}
	return values, scanner.Err()
}

// updateEnvKeys reescreve o .env preservando comentarios, ordem e
// chaves nao mencionadas - so troca o valor das chaves passadas em
// "values" (as que ainda nao existirem sao acrescentadas no final).
// Nunca regenera o arquivo do zero.
func updateEnvKeys(path string, values map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	remaining := make(map[string]string, len(values))
	for key, value := range values {
		remaining[key] = value
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		key, _, ok := parseEnvLine(line)
		if !ok {
			continue
		}
		if newValue, exists := remaining[key]; exists {
			lines[i] = fmt.Sprintf("%s=%s", key, newValue)
			delete(remaining, key)
		}
	}
	for key, value := range remaining {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// parseEnvLine reconhece linhas "CHAVE=valor", ignorando comentarios e
// linhas em branco.
func parseEnvLine(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), parts[1], true
}
