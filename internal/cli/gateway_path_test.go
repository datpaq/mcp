// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: assert production gateway paths for catalog-sync route renames.

package cli

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestDefineUsesDictionaryGatewayRoute(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"data":{"word":"example"}}`,
		"define", "--word", "example", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("method = %s, want POST", req.Method)
	}
	if req.Path != "/dictionary" {
		t.Fatalf("path = %s, want /dictionary", req.Path)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["word"] != "example" {
		t.Fatalf("unexpected body: %s", req.Body)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestStatesListUsesUsStatesGatewayRoute(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"data":[{"abbreviation":"CA"}]}`,
		"us-states", "list", "--agent", "--no-cache",
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
	if !strings.HasPrefix(req.Path, "/us-states") {
		t.Fatalf("path = %s, want /us-states...", req.Path)
	}
	if strings.Contains(req.Path, "/states/states") {
		t.Fatalf("path must not use stale /states/states route: %s", req.Path)
	}
	assertJSONOnly(t, stdout)
	_ = stderr // store ID-skip warnings are unrelated to gateway path routing
}

func TestStatesByAbbrUsesUsStatesGatewayRoute(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"data":{"abbreviation":"CA"}}`,
		"us-states", "by-abbr", "CA", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 || requests[0].Method != http.MethodGet {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if requests[0].Path != "/us-states/CA" {
		t.Fatalf("path = %s, want /us-states/CA", requests[0].Path)
	}
	assertJSONOnly(t, stdout)
	_ = stderr // store ID-skip warnings are unrelated to gateway path routing
}

func TestGenerateUsesSampleDataGatewayRoute(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"data":[{"id":"1"}]}`,
		"generate", "--type", "user", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Method != http.MethodPost {
		t.Fatalf("method = %s, want POST", req.Method)
	}
	if req.Path != "/sample-data/generate" {
		t.Fatalf("path = %s, want /sample-data/generate", req.Path)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestGenerateBatchUsesSampleDataGatewayRoute(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"data":[]}`,
		"generate-batch", "--requests", `[{"type":"user","count":1}]`, "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 || requests[0].Method != http.MethodPost {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if requests[0].Path != "/sample-data/generate-batch" {
		t.Fatalf("path = %s, want /sample-data/generate-batch", requests[0].Path)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestSchemasUsesSampleDataGatewayRoute(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"data":["user","company"]}`,
		"schemas", "--agent", "--no-cache",
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
	if req.Path != "/sample-data/schemas" {
		t.Fatalf("path = %s, want /sample-data/schemas", req.Path)
	}
	assertJSONOnly(t, stdout)
	_ = stderr // store ID-skip warnings are unrelated to gateway path routing
}

func TestSyncResourcePathsUseGatewayServicePrefixes(t *testing.T) {
	cases := map[string]string{
		"schemas": "/sample-data/schemas",
		"us-states": "/us-states",
		"states":    "/us-states",
	}
	for resource, want := range cases {
		got, err := syncResourcePath(resource)
		if err != nil {
			t.Fatalf("syncResourcePath(%q): %v", resource, err)
		}
		if got != want {
			t.Fatalf("syncResourcePath(%q) = %q, want %q", resource, got, want)
		}
	}
}
