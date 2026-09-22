package network

import (
	"strings"
	"testing"
)

func TestDockerDevicesConfigKeepsDockerInterfacesUnmanaged(t *testing.T) {
	config := string(dockerDevicesConfig)
	for _, matcher := range []string{
		"match-device=interface-name:docker0",
		"match-device=interface-name:br-*",
		"match-device=interface-name:veth*",
		"managed=0",
	} {
		if !strings.Contains(config, matcher) {
			t.Fatalf("configuracao sem %q", matcher)
		}
	}
	if strings.Contains(config, "[keyfile]") {
		t.Fatal("configuracao nao pode sobrescrever o drop-in temporario do hotspot")
	}
}
