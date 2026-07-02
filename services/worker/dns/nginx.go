package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var serverNameRegex = regexp.MustCompile(`(?s)\bserver_name\s+([^;]+);`)

type nginxNames struct {
	hosts map[string]bool
	zones map[string]bool
}

func loadNginxNames(root string) nginxNames {
	names := nginxNames{hosts: map[string]bool{}, zones: map[string]bool{}}
	if root == "" {
		return names
	}
	if _, err := os.Stat(root); err != nil {
		log.Printf("[dns-provider] aviso: configuracao do nginx-ui indisponivel em %s: %v", root, err)
		return names
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[dns-provider] aviso: falha ao ler %s: %v", path, err)
			return nil
		}
		for _, match := range serverNameRegex.FindAllStringSubmatch(stripNginxComments(string(data)), -1) {
			for _, token := range strings.Fields(match[1]) {
				addNginxName(names, token)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("[dns-provider] aviso: falha ao varrer configuracao do nginx-ui: %v", err)
	}
	return names
}

func addNginxName(names nginxNames, value string) {
	name := normalizeZone(value)
	if name == "" || strings.ContainsAny(name, "$~") || name == "_" || !isValidDomain(name) {
		return
	}
	if strings.HasPrefix(strings.TrimSpace(value), "*.") {
		names.zones[name] = true
		return
	}
	names.hosts[name] = true
}

func stripNginxComments(raw string) string {
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}
