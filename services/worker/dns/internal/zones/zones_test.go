package zones

import (
	"net"
	"testing"

	"bindnet/dns-provider/internal/core"
)

func TestConfiguredAddressPrefersExplicitZoneOverStaticTLD(t *testing.T) {
	cfg := &core.Config{TLDAddresses: map[string]net.IP{
		"local": net.ParseIP("192.168.50.10"),
	}}

	ip := configuredAddressFor(cfg, "app.empresa.local", net.ParseIP("10.0.0.25"))
	if got, want := ip.String(), "10.0.0.25"; got != want {
		t.Fatalf("configuredAddressFor() = %s, want explicit zone address %s", got, want)
	}
}

func TestConfiguredAddressUsesStaticTLDForAllSubdomains(t *testing.T) {
	cfg := &core.Config{
		TLDs: map[string]bool{"local": true, "empresa.test": true},
		TLDAddresses: map[string]net.IP{
			"local":        net.ParseIP("192.168.50.10"),
			"empresa.test": net.ParseIP("10.20.30.40"),
		},
	}

	for _, name := range []string{"empresa.test", "api.empresa.test", "deep.api.empresa.test"} {
		ip := configuredAddressFor(cfg, name, nil)
		if got, want := ip.String(), "10.20.30.40"; got != want {
			t.Fatalf("configuredAddressFor(%q) = %s, want static TLD address %s", name, got, want)
		}
	}
}

func TestMoreSpecificAutomaticTLDPreventsParentStaticAddress(t *testing.T) {
	cfg := &core.Config{
		TLDs: map[string]bool{
			"local.com":           true,
			"test.sgdp.local.com": true,
		},
		TLDAddresses: map[string]net.IP{
			"local.com": net.ParseIP("127.0.0.141"),
		},
	}

	if ip := configuredAddressFor(cfg, "a.test.sgdp.local.com", nil); ip != nil {
		t.Fatalf("configuredAddressFor() = %s, want no static address because test.sgdp.local.com is more specific", ip)
	}
}

func TestMoreSpecificStaticTLDWinsOverParent(t *testing.T) {
	cfg := &core.Config{
		TLDs: map[string]bool{
			"local.com":           true,
			"test.sgdp.local.com": true,
		},
		TLDAddresses: map[string]net.IP{
			"local.com":           net.ParseIP("127.0.0.141"),
			"test.sgdp.local.com": net.ParseIP("10.234.1.69"),
		},
	}

	ip := configuredAddressFor(cfg, "a.test.sgdp.local.com", nil)
	if got, want := ip.String(), "10.234.1.69"; got != want {
		t.Fatalf("configuredAddressFor() = %s, want the closest TLD address %s", got, want)
	}
}

func TestStaticTLDIsConfiguredForHotspotWildcard(t *testing.T) {
	cfg := &core.Config{
		TLDs:         map[string]bool{"local": true},
		TLDAddresses: map[string]net.IP{"local": net.ParseIP("192.168.50.10")},
	}

	configured, err := HasConfiguredAddress(cfg, "app.local.")
	if err != nil || !configured {
		t.Fatalf("HasConfiguredAddress(app.local.) = (%t, %v), want (true, nil)", configured, err)
	}
}

func TestZoneForExactDomainZoneIsLocal(t *testing.T) {
	cfg := &core.Config{
		TLDs:        map[string]bool{"local": true},
		DomainZones: map[string]bool{"costa.bnet": true},
		NginxHosts:  map[string]bool{},
		NginxZones:  map[string]bool{},
		Routes:      core.NewTable(),
	}

	zone, kind, nextHop := For("costa.bnet.", cfg)
	if zone != "costa.bnet." || kind != Local || nextHop != "" {
		t.Fatalf("For(costa.bnet.) = (%q, %v, %q), want local costa.bnet.", zone, kind, nextHop)
	}
}

func TestZoneForSubdomainInsideConcreteDomainZoneIsLocal(t *testing.T) {
	cfg := &core.Config{
		TLDs:        map[string]bool{"local": true},
		DomainZones: map[string]bool{"costa.bnet": true},
		NginxHosts:  map[string]bool{},
		NginxZones:  map[string]bool{},
		Routes:      core.NewTable(),
	}

	zone, kind, nextHop := For("app.costa.bnet.", cfg)
	if zone != "costa.bnet." || kind != Local || nextHop != "" {
		t.Fatalf("For(app.costa.bnet.) = (%q, %v, %q), want local costa.bnet.", zone, kind, nextHop)
	}
}

func TestZoneForUnknownNameInsideBroadMeshZoneStaysUnknown(t *testing.T) {
	cfg := &core.Config{
		TLDs:        map[string]bool{"local": true},
		DomainZones: map[string]bool{"bnet": true},
		NginxHosts:  map[string]bool{},
		NginxZones:  map[string]bool{},
		Routes:      core.NewTable(),
	}

	zone, kind, nextHop := For("app.unknown.bnet.", cfg)
	if zone != "bnet." || kind != MeshUnknown || nextHop != "" {
		t.Fatalf("For(app.unknown.bnet.) = (%q, %v, %q), want unknown bnet.", zone, kind, nextHop)
	}
}

func TestZoneForRemoteRouteMatchesSubdomains(t *testing.T) {
	routes := core.NewTable()
	routes.Replace(map[string]core.Route{
		"ahmed.bnet": {
			Domain:  "ahmed.bnet",
			Owner:   "Ahmed",
			NextHop: "10.234.2.140",
			Source:  "10.234.2.140:8531",
			State:   core.StateOK,
		},
	})
	cfg := &core.Config{
		TLDs:        map[string]bool{"local": true},
		DomainZones: map[string]bool{"costa.bnet": true},
		NginxHosts:  map[string]bool{},
		NginxZones:  map[string]bool{},
		Routes:      routes,
	}

	zone, kind, nextHop := For("test.ahmed.bnet.", cfg)
	if zone != "ahmed.bnet." || kind != Remote || nextHop != "10.234.2.140" {
		t.Fatalf("For(test.ahmed.bnet.) = (%q, %v, %q), want remote ahmed.bnet via 10.234.2.140", zone, kind, nextHop)
	}
}

func TestZoneForMultiLabelLocalTLDMatchesAnyDepth(t *testing.T) {
	cfg := &core.Config{
		TLDs:        map[string]bool{"local": true, "local.com": true, "a.b.local.org": true},
		DomainZones: map[string]bool{},
		NginxHosts:  map[string]bool{},
		NginxZones:  map[string]bool{},
		Routes:      core.NewTable(),
	}

	cases := []struct {
		name string
		zone string
		kind Kind
	}{
		// Curinga de TLD local casa em qualquer profundidade, mas agora
		// como LocalWildcard (host/container resolvem; hotspot -> NXDOMAIN,
		// ver dnsserver).
		{"foo.local.", "local.", LocalWildcard},
		{"local.com.", "local.com.", LocalWildcard},
		{"app.local.com.", "local.com.", LocalWildcard},
		{"x.y.z.local.com.", "local.com.", LocalWildcard},
		{"svc.a.b.local.org.", "a.b.local.org.", LocalWildcard},
		{"other.com.", "com.", None},
		{"b.local.org.", "org.", None},
	}
	for _, tc := range cases {
		zone, kind, nextHop := For(tc.name, cfg)
		if zone != tc.zone || kind != tc.kind || nextHop != "" {
			t.Errorf("For(%s) = (%q, %v, %q), want (%q, %v, \"\")", tc.name, zone, kind, nextHop, tc.zone, tc.kind)
		}
	}
}

func TestZoneForLocalTLDWinsOverBroadMeshRoot(t *testing.T) {
	cfg := &core.Config{
		TLDs:        map[string]bool{"apps.bnet": true},
		DomainZones: map[string]bool{"bnet": true},
		NginxHosts:  map[string]bool{},
		NginxZones:  map[string]bool{},
		Routes:      core.NewTable(),
	}

	zone, kind, _ := For("web.apps.bnet.", cfg)
	if zone != "apps.bnet." || kind != LocalWildcard {
		t.Fatalf("For(web.apps.bnet.) = (%q, %v), want localwildcard apps.bnet.", zone, kind)
	}
	zone, kind, _ = For("web.other.bnet.", cfg)
	if zone != "bnet." || kind != MeshUnknown {
		t.Fatalf("For(web.other.bnet.) = (%q, %v), want unknown bnet.", zone, kind)
	}
}
