package store

import "testing"

func TestParseIPv4AcceptsPostgresInetCIDR(t *testing.T) {
	ip := parseIPv4("10.234.1.69/32")
	if ip == nil || ip.String() != "10.234.1.69" {
		t.Fatalf("parseIPv4(10.234.1.69/32) = %v, want 10.234.1.69", ip)
	}
}
