package mcp

import (
	"testing"

	"github.com/datpaq/mcp/internal/cli"
	"github.com/mark3labs/mcp-go/server"
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

func TestRegisterPublicTools_IncludesLookupAPIs(t *testing.T) {
	for _, slug := range []string{"dns", "domain-lookup", "mac-address", "mx-lookup"} {
		if !cli.IsActiveInterface(slug) {
			t.Fatalf("%q should be active in embedded manifest", slug)
		}
	}
	s := server.NewMCPServer("datpaq", "test", server.WithToolCapabilities(true))
	RegisterPublicTools(s)
	tools := s.ListTools()
	want := []string{
		"dns_lookup",
		"domain-lookup_get",
		"domain-lookup_post",
		"mac-address_lookup",
		"mx-lookup_batch",
		"mx-lookup_get",
		"mx-lookup_post",
	}
	for _, name := range want {
		if _, ok := tools[name]; !ok {
			t.Errorf("RegisterPublicTools missing %q", name)
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
