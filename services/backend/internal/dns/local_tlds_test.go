package dns

import "testing"

func TestNormalizeLocalTLDsAcceptsPostgresInetCIDR(t *testing.T) {
	entries, err := normalizeLocalTLDs([]localTLD{{
		TLD:     "test.sgdp.local.com",
		Mode:    "static",
		Address: "10.234.1.69/32",
	}})
	if err != nil {
		t.Fatalf("normalizeLocalTLDs() returned error: %v", err)
	}
	if got, want := entries[0].Address, "10.234.1.69"; got != want {
		t.Fatalf("normalized address = %q, want %q", got, want)
	}
}
