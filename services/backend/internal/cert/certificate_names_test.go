package cert

import "testing"

func TestNormalizeCertificateName(t *testing.T) {
	tests := []struct {
		input string
		want  string
		valid bool
	}{
		{input: "  Portal interno  ", want: "Portal interno", valid: true},
		{input: "Certificado produção", want: "Certificado produção", valid: true},
		{input: "", valid: false},
		{input: "../segredo", valid: false},
		{input: `nome "quebrado`, valid: false},
		{input: "linha\nnova", valid: false},
	}
	for _, test := range tests {
		got, err := normalizeCertificateName(test.input)
		if test.valid && (err != nil || got != test.want) {
			t.Fatalf("normalizeCertificateName(%q) = %q, %v; esperado %q", test.input, got, err, test.want)
		}
		if !test.valid && err == nil {
			t.Fatalf("normalizeCertificateName(%q) deveria falhar", test.input)
		}
	}
}
