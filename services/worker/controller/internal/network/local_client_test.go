package network

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestIsLocalAddress(t *testing.T) {
	addresses := []net.Addr{
		&net.IPNet{IP: net.ParseIP("10.91.0.1"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("192.168.1.50"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("192.168.12.1"), Mask: net.CIDRMask(24, 32)},
		&net.IPNet{IP: net.ParseIP("2001:db8::50"), Mask: net.CIDRMask(64, 128)},
	}
	for _, tc := range []struct {
		ip    string
		local bool
	}{
		{"127.0.0.1", true}, {"127.90.0.5", true}, {"::1", true},
		{"10.91.0.1", true}, {"192.168.1.50", true}, {"192.168.12.1", true},
		{"2001:db8::50", true}, {"::ffff:192.168.1.50", true},
		{"192.168.12.20", false}, {"192.168.1.51", false},
		{"10.91.0.10", false}, {"2001:db8::51", false}, {"8.8.8.8", false},
	} {
		t.Run(tc.ip, func(t *testing.T) {
			if local := isLocalAddress(netip.MustParseAddr(tc.ip), addresses); local != tc.local {
				t.Fatalf("origem %s: local=%v, esperado %v", tc.ip, local, tc.local)
			}
		})
	}
}

func TestLocalClientRoute(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux)
	for _, tc := range []struct {
		query string
		code  int
	}{
		{"?ip=127.0.0.1", http.StatusOK},
		{"?ip=::1", http.StatusOK},
		{"?ip=invalido", http.StatusBadRequest},
		{"", http.StatusBadRequest},
	} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/network/local-client"+tc.query, nil))
		if recorder.Code != tc.code {
			t.Fatalf("%s: retornou %d, esperado %d", tc.query, recorder.Code, tc.code)
		}
		if tc.code == http.StatusOK && recorder.Body.String() != "{\"local\":true}\n" {
			t.Fatalf("loopback deveria pertencer ao host: %s", recorder.Body.String())
		}
	}
}
