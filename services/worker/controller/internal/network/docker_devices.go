package network

import (
	"bytes"
	"fmt"
	"os"
)

const dockerDevicesDropin = "/etc/NetworkManager/conf.d/90-bindnet-docker-virtual-devices.conf"

var dockerDevicesConfig = []byte(`# Interfaces efemeras do Docker nao sao conexoes do host.
[device-bindnet-docker0]
match-device=interface-name:docker0
managed=0

[device-bindnet-docker-bridges]
match-device=interface-name:br-*
managed=0

[device-bindnet-docker-veth]
match-device=interface-name:veth*
managed=0
`)

// EnsureDockerVirtualDevicesUnmanaged impede que o NetworkManager trate
// bridges e veths do Docker como conexoes do host. As secoes [device-*]
// nao conflitam com o drop-in [keyfile] temporario do hotspot.
func EnsureDockerVirtualDevicesUnmanaged() error {
	current, err := os.ReadFile(dockerDevicesDropin)
	if err == nil && bytes.Equal(current, dockerDevicesConfig) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("nao foi possivel ler %s: %w", dockerDevicesDropin, err)
	}
	if err := os.WriteFile(dockerDevicesDropin, dockerDevicesConfig, 0644); err != nil {
		return fmt.Errorf("nao foi possivel escrever %s: %w", dockerDevicesDropin, err)
	}
	if output, err := reloadNetworkManagerConfig(); err != nil {
		return fmt.Errorf("nmcli nao conseguiu recarregar a configuracao do Docker: %s: %w", string(output), err)
	}
	return nil
}
