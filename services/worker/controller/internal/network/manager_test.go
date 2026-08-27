package network

import "testing"

func TestIsDeviceNotFoundOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "nmcli ingles",
			output: "Error: Device 'wlp0s20f3' not found.",
			want:   true,
		},
		{
			name:   "nmcli portugues sem acento",
			output: "Erro: dispositivo wlp0s20f3 nao encontrado.",
			want:   true,
		},
		{
			name:   "outro erro do networkmanager",
			output: "Error: NetworkManager is not running.",
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isDeviceNotFoundOutput([]byte(test.output)); got != test.want {
				t.Fatalf("isDeviceNotFoundOutput(%q) = %v, want %v", test.output, got, test.want)
			}
		})
	}
}
