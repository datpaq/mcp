// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch coverage for MX lookup route and domains parameter compatibility.

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type capturedMXRequest struct {
	Method   string
	Path     string
	RawQuery string
	Body     string
}

func runMXCommand(t *testing.T, responseBody string, args ...string) (string, string, []capturedMXRequest, error) {
	t.Helper()

	var mu sync.Mutex
	var requests []capturedMXRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		mu.Lock()
		requests = append(requests, capturedMXRequest{
			Method:   r.Method,
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
			Body:     string(body),
		})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	defer srv.Close()

	t.Setenv("DATPAQ_BASE_URL", srv.URL)
	t.Setenv("DATPAQ_API_KEY", "test-key")
	t.Setenv("DATPAQ_CONFIG", filepath.Join(t.TempDir(), "config.toml"))
	t.Setenv("PRINTING_PRESS_VERIFY", "")
	t.Setenv("PRINTING_PRESS_VERIFY_LIVE_HTTP", "")

	oldStderr := os.Stderr
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = stderrWriter
	defer func() {
		os.Stderr = oldStderr
		_ = stderrReader.Close()
	}()

	var stdout bytes.Buffer
	var cobraStderr bytes.Buffer
	cmd := RootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&cobraStderr)
	cmd.SetArgs(args)

	execErr := cmd.Execute()
	_ = stderrWriter.Close()
	os.Stderr = oldStderr

	stderrBytes, readErr := io.ReadAll(stderrReader)
	if readErr != nil {
		t.Fatalf("read captured stderr: %v", readErr)
	}
	stderr := string(stderrBytes) + cobraStderr.String()

	mu.Lock()
	defer mu.Unlock()
	return stdout.String(), stderr, append([]capturedMXRequest(nil), requests...), execErr
}

func TestMxLookupGetSendsDomainsQuery(t *testing.T) {
	stdout, stderr, requests, err := runMXCommand(t,
		`{"status":"success","data":[{"id":"mx1","domain":"example.com"}]}`,
		"mx-lookup", "get", "--domains", "example.com", "--resolve-ips", "--agent", "--data-source", "live", "--no-cache",
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
	if req.Path != "/mx-lookup" {
		t.Fatalf("path = %s, want /mx-lookup", req.Path)
	}
	if strings.Contains(req.RawQuery, "domain=") {
		t.Fatalf("query must not contain singular domain: %s", req.RawQuery)
	}
	if got := queryValue(t, req.RawQuery, "domains"); got != "example.com" {
		t.Fatalf("domains query = %q, want example.com (raw %s)", got, req.RawQuery)
	}
	if got := queryValue(t, req.RawQuery, "resolve_ips"); got != "true" {
		t.Fatalf("resolve_ips query = %q, want true (raw %s)", got, req.RawQuery)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestMxLookupGetLegacyDomainFlagMapsToDomainsQuery(t *testing.T) {
	_, stderr, requests, err := runMXCommand(t,
		`{"status":"success","data":[{"id":"mx1","domain":"legacy.example"}]}`,
		"mx-lookup", "get", "--domain", "legacy.example", "--agent", "--data-source", "live", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	if got := queryValue(t, requests[0].RawQuery, "domains"); got != "legacy.example" {
		t.Fatalf("domains query = %q, want legacy.example (raw %s)", got, requests[0].RawQuery)
	}
	if strings.Contains(requests[0].RawQuery, "domain=") {
		t.Fatalf("query must not contain singular domain: %s", requests[0].RawQuery)
	}
	assertNoWarning(t, stderr)
}

func TestMxLookupBatchPostsDomainsBodyToMxLookup(t *testing.T) {
	stdout, stderr, requests, err := runMXCommand(t,
		`{"status":"success","data":[{"id":"mx1","domain":"example.com"},{"id":"mx2","domain":"gmail.com"}]}`,
		"mx-lookup", "batch", "--domains", `["example.com","gmail.com"]`, "--agent", "--no-cache",
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
	if req.Path != "/mx-lookup" {
		t.Fatalf("path = %s, want /mx-lookup", req.Path)
	}
	if req.RawQuery != "" {
		t.Fatalf("query = %q, want empty", req.RawQuery)
	}
	var body struct {
		Domains []string `json:"domains"`
		Domain  string   `json:"domain"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("request body is invalid JSON: %v\n%s", err, req.Body)
	}
	if body.Domain != "" {
		t.Fatalf("body must not contain singular domain: %s", req.Body)
	}
	if got, want := strings.Join(body.Domains, ","), "example.com,gmail.com"; got != want {
		t.Fatalf("body domains = %q, want %q (body %s)", got, want, req.Body)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestMxLookupAgentOutputSuppressesPartialFailureWarning(t *testing.T) {
	stdout, stderr, _, err := runMXCommand(t,
		`{"partialFailureError":{"code":3,"message":"one domain failed"},"results":[{"resourceName":"domains/example.com"}]}`,
		"mx-lookup", "batch", "--domains", "example.com,gmail.com", "--agent", "--allow-partial-failure", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
	if !strings.Contains(stdout, "partial_failure") {
		t.Fatalf("stdout should carry structured partial_failure in JSON envelope: %s", stdout)
	}
}

func queryValue(t *testing.T, rawQuery, key string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	return req.URL.Query().Get(key)
}

func assertJSONOnly(t *testing.T, stdout string) {
	t.Helper()
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		t.Fatal("stdout is empty")
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		t.Fatalf("stdout is not valid JSON only: %v\n%s", err, stdout)
	}
}

func assertNoWarning(t *testing.T, stderr string) {
	t.Helper()
	if strings.Contains(strings.ToLower(stderr), "warning") || strings.Contains(strings.ToLower(stderr), "warn:") {
		t.Fatalf("unexpected warning on stderr: %q", stderr)
	}
}
