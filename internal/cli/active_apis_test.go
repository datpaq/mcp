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
	// Mirrored from ProApi/api_list_documentation_upload.sql IsActive=true,
	// using CLI command/interface names where website slugs differ:
	// dictionary -> define, dns-lookup -> dns.
	for _, slug := range []string{
		"aircraft",
		"calendar",
		"company-enrichment",
		"convert-time",
		"country-codes",
		"current-time",
		"define",
		"dns",
		"domain-lookup",
		"email-validation",
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
		"phone-validation",
		"precious-metals",
		"profanity",
		"public-holidays",
		"qr-code",
		"sample-data",
		"secure-relay",
		"spell-check",
		"states",
		"text-language",
		"thesaurus",
		"unit-conversion",
		"user-avatar",
		"validate-ip",
		"vin-lookup",
		"weather",
		"web-scraping",
		"web-screenshot",
		"web-search",
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
	if got != 41 {
		t.Errorf("activeAPICount() = %d, want 41 supported active APIs", got)
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
