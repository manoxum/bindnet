package hotspot

import "testing"

func TestHotspotPortalURLUsesDedicatedDomain(t *testing.T) {
	t.Parallel()

	const want = "http://bindnet.local.com/"
	if got := hotspotPortalURL(); got != want {
		t.Fatalf("hotspotPortalURL() = %q, want %q", got, want)
	}
}
