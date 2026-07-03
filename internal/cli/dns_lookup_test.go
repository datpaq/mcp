// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch coverage for DNS Lookup production gateway route compatibility.

package cli

import (
	"net/http"
	"strings"
	"testing"
)

func TestDnsLookupUsesProductionGatewayRoute(t *testing.T) {
	stdout, stderr, requests, err := runMXCommand(t,
		`{"status":"success","data":[{"domain":"example.com","records":{"A":["93.184.216.34"]}}]}`,
		"dns", "--domain", "example.com", "--type", "A", "--agent", "--data-source", "live", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Method != http.MethodGet {
		t.Fatalf("method = %s, want GET", req.Method)
	}
	if req.Path != "/dns-lookup" {
		t.Fatalf("path = %s, want /dns-lookup", req.Path)
	}
	if strings.Contains(req.Path, "/dns") && req.Path != "/dns-lookup" {
		t.Fatalf("path must not use stale /dns route: %s", req.Path)
	}
	if got := queryValue(t, req.RawQuery, "domain"); got != "example.com" {
		t.Fatalf("domain query = %q, want example.com (raw %s)", got, req.RawQuery)
	}
	if got := queryValue(t, req.RawQuery, "type"); got != "A" {
		t.Fatalf("type query = %q, want A (raw %s)", got, req.RawQuery)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}
