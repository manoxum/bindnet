package main

import "testing"

func TestZoneForExactDomainZoneIsLocal(t *testing.T) {
	cfg := &dnsConfig{
		tlds:        map[string]bool{"local": true},
		domainZones: map[string]bool{"costa.bnet": true},
		nginxHosts:  map[string]bool{},
		nginxZones:  map[string]bool{},
		routes:      newRouteTable(),
	}

	zone, kind, nextHop := zoneFor("costa.bnet.", cfg)
	if zone != "costa.bnet." || kind != zoneLocal || nextHop != "" {
		t.Fatalf("zoneFor(costa.bnet.) = (%q, %v, %q), want local costa.bnet.", zone, kind, nextHop)
	}
}

func TestZoneForUnknownNameInsideDomainZoneStaysUnknown(t *testing.T) {
	cfg := &dnsConfig{
		tlds:        map[string]bool{"local": true},
		domainZones: map[string]bool{"costa.bnet": true},
		nginxHosts:  map[string]bool{},
		nginxZones:  map[string]bool{},
		routes:      newRouteTable(),
	}

	zone, kind, nextHop := zoneFor("app.costa.bnet.", cfg)
	if zone != "costa.bnet." || kind != zoneMeshUnknown || nextHop != "" {
		t.Fatalf("zoneFor(app.costa.bnet.) = (%q, %v, %q), want unknown costa.bnet.", zone, kind, nextHop)
	}
}
