package mcp

import (
	"testing"

	"github.com/datpaq/mcp/internal/cli"
)

func TestMcpToolInterfaceSlug(t *testing.T) {
	cases := []struct {
		tool string
		want string
		ok   bool
	}{
		{"whois_lookup", "whois", true},
		{"aircraft_lookup-by-tail", "aircraft", true},
		{"exchange-rates-and-currency_exchange-rate-get", "exchange-rates-and-currency", true},
		{"generate_batch_sample_data_post", "generate-batch", true},
		{"sync_ip_geolocation", "", false},
		{"search", "", false},
	}
	for _, c := range cases {
		got, ok := mcpToolInterfaceSlug(c.tool)
		if ok != c.ok || got != c.want {
			t.Errorf("mcpToolInterfaceSlug(%q) = (%q, %v), want (%q, %v)", c.tool, got, ok, c.want, c.ok)
		}
	}
}

func TestInactiveAPIToolNames_Aircraft(t *testing.T) {
	if cli.IsActiveInterface("aircraft") {
		t.Skip("aircraft is active in manifest; cannot test inactive filtering")
	}
	slug, ok := mcpToolInterfaceSlug("aircraft_lookup-by-tail")
	if !ok || slug != "aircraft" {
		t.Fatalf("mcpToolInterfaceSlug(aircraft_lookup-by-tail) = (%q, %v)", slug, ok)
	}
	if cli.IsActiveInterface(slug) {
		t.Fatal("aircraft should be inactive")
	}
}
