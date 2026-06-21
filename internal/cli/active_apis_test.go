package cli

import (
	"testing"
)

func TestIsActiveInterface_KnownActiveSlug(t *testing.T) {
	if !IsActiveInterface("whois") {
		t.Fatal("whois should be active in embedded manifest")
	}
}

func TestIsActiveInterface_KnownInactiveSlug(t *testing.T) {
	if IsActiveInterface("aircraft") {
		t.Fatal("aircraft should be inactive in embedded manifest")
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


func TestNewLookupAPIsAreActive(t *testing.T) {
	for _, slug := range []string{"dns", "domain-lookup", "mac-address", "mx-lookup"} {
		if !IsActiveInterface(slug) {
			t.Errorf("expected %q in active-apis.json", slug)
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
		if slug == "aircraft" {
			foundInactive = true
		}
	}
	if !foundActive {
		t.Fatal("whois missing from known interfaces")
	}
	if !foundInactive {
		t.Fatal("aircraft missing from known interfaces")
	}
}
