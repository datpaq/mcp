// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: verifies datpaq.com active APIs appear in the curated
// discovery manifest so `datpaq api` / `datpaq sample` surface them.

package cli

import "testing"

func TestIsActiveInterface_KnownActiveSlug(t *testing.T) {
	if !IsActiveInterface("whois") {
		t.Fatal("whois should be active in embedded manifest")
	}
}

func TestIsActiveInterface_KnownInactiveSlug(t *testing.T) {
	if IsActiveInterface("schemas") {
		t.Fatal("schemas should be inactive in embedded manifest")
	}
}

func TestListActiveInterfacesSorted(t *testing.T) {
	slugs := ListActiveInterfaces()
	if len(slugs) == 0 {
		t.Fatal("expected active slugs")
	}
	for i := 1; i < len(slugs); i++ {
		if slugs[i] < slugs[i-1] {
			t.Fatalf("not sorted: %v", slugs)
		}
	}
}

func TestSupportedWebsiteActiveAPIsAreActive(t *testing.T) {
	// Mirrored from website API_ACTIVE_SLUGS (35), using CLI command/interface
	// names where website slugs differ:
	// dictionary -> define, dns-lookup -> dns. us-states matches the website slug.
	for _, slug := range []string{
		"aircraft",
		"calendar",
		"convert-time",
		"country-codes",
		"current-time",
		"define",
		"dns",
		"domain-lookup",
		"ev-charger",
		"exchange-rates-and-currency",
		"geocoding",
		"helicopter",
		"image-processing",
		"ip-geolocation",
		"ip-intelligence",
		"mac-address",
		"mx-lookup",
		"pdf-generation",
		"profanity",
		"public-holidays",
		"qr-code",
		"sample-data",
		"spell-check",
		"us-states",
		"text-language",
		"thesaurus",
		"unit-conversion",
		"user-avatar",
		"validate-ip",
		"vin-lookup",
		"weather",
		"web-scraping",
		"web-screenshot",
		"whois",
		"working-days",
	} {
		if !IsActiveInterface(slug) {
			t.Errorf("expected %q in active-apis.json", slug)
		}
	}
}

func TestActiveAPICountMatchesSupportedWebsiteActiveAPIs(t *testing.T) {
	got := activeAPICount()
	if got != 35 {
		t.Errorf("activeAPICount() = %d, want 35 supported active APIs", got)
	}
}

func TestInactiveWebsiteCatalogAPIsAreNotActive(t *testing.T) {
	// Previously active in the CLI/MCP manifest but no longer in website API_ACTIVE_SLUGS.
	for _, slug := range []string{
		"company-enrichment",
		"email-validation",
		"phone-validation",
		"precious-metals",
		"secure-relay",
		"web-search",
	} {
		if IsActiveInterface(slug) {
			t.Errorf("did not expect inactive %q in active-apis.json", slug)
		}
	}
}

func TestKnownInterfaceSlugsIncludesInactive(t *testing.T) {
	known := KnownInterfaceSlugs(RootCmd())
	foundActive, foundInactive := false, false
	for _, slug := range known {
		if slug == "whois" {
			foundActive = true
		}
		if slug == "schemas" {
			foundInactive = true
		}
	}
	if !foundActive {
		t.Fatal("whois missing from known interfaces")
	}
	if !foundInactive {
		t.Fatal("schemas missing from known interfaces")
	}
}
