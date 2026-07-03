package main

import "testing"

func TestDiscoverHostSourceIPsParsesCIDRAndPlainIP(t *testing.T) {
	ips, err := discoverHostSourceIPs("10.234.2.102/32, 10.234.2.103", "10.234.2.103")
	if err != nil {
		t.Fatalf("discoverHostSourceIPs returned error: %v", err)
	}
	if len(ips) != 1 || ips[0] != "10.234.2.102" {
		t.Fatalf("discoverHostSourceIPs returned %v, want [10.234.2.102]", ips)
	}
}

func TestDiscoverHostSourceIPsRejectsLoopback(t *testing.T) {
	if _, err := discoverHostSourceIPs("127.0.0.1/32"); err == nil {
		t.Fatalf("discoverHostSourceIPs accepted loopback HOST_SOURCE_CIDR")
	}
}
