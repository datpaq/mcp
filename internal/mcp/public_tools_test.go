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
		{"us-states_list", "us-states", true},
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

func TestInactiveAPIToolNames_RemovesInactiveKnownInterface(t *testing.T) {
	if cli.IsActiveInterface("schemas") {
		t.Skip("schemas is active in manifest; cannot test inactive filtering")
	}
	slug, ok := mcpToolInterfaceSlug("schemas_sample-data")
	if !ok || slug != "schemas" {
		t.Fatalf("mcpToolInterfaceSlug(schemas_sample-data) = (%q, %v)", slug, ok)
	}
	if cli.IsActiveInterface(slug) {
		t.Fatal("schemas should be inactive")
	}

	s := server.NewMCPServer("datpaq", "test", server.WithToolCapabilities(true))
	RegisterPublicTools(s)
	if _, ok := s.ListTools()["schemas_sample-data"]; ok {
		t.Fatal("inactive schemas_sample-data tool should be removed from public MCP surface")
	}
}

func TestRegisterPublicTools_IncludesUsStates(t *testing.T) {
	if !cli.IsActiveInterface("us-states") {
		t.Fatal("us-states should be active in embedded manifest")
	}
	s := server.NewMCPServer("datpaq", "test", server.WithToolCapabilities(true))
	RegisterPublicTools(s)
	tools := s.ListTools()
	for _, name := range []string{"us-states_by-abbr", "us-states_by-fips", "us-states_list"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("RegisterPublicTools missing %q", name)
		}
	}
	if _, ok := tools["states_list"]; ok {
		t.Fatal("legacy states_list tool should not remain on the public MCP surface")
	}
}

func TestRegisterPublicTools_RemovesInactiveWebsiteCatalogAPIs(t *testing.T) {
	s := server.NewMCPServer("datpaq", "test", server.WithToolCapabilities(true))
	RegisterPublicTools(s)
	tools := s.ListTools()
	for _, name := range []string{
		"company-enrichment_company-enrich",
		"email-validation_email-validate-single",
		"phone-validation_validate",
		"precious-metals_prices",
		"secure-relay_relay",
		"web-search_search",
	} {
		if _, ok := tools[name]; ok {
			t.Errorf("inactive tool %q should be removed from public MCP surface", name)
		}
	}
}
