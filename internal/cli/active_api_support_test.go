// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch coverage for active ProApi services added between generator runs.

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

type capturedAPIRequest struct {
	Method string
	Path   string
	Body   string
}

func runAPICommand(t *testing.T, responseBody string, args ...string) (string, string, []capturedAPIRequest, error) {
	t.Helper()

	var mu sync.Mutex
	var requests []capturedAPIRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		mu.Lock()
		requests = append(requests, capturedAPIRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(body),
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
	return stdout.String(), stderr, append([]capturedAPIRequest(nil), requests...), execErr
}

func TestPhoneValidationValidatePostsBody(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"valid":true}`,
		"phone-validation", "validate", "--phone-number", "+14155552671", "--country-code", "US", "--agent", "--no-cache",
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
	if req.Path != "/phone-validation/validate" {
		t.Fatalf("path = %s, want /phone-validation/validate", req.Path)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("request body is invalid JSON: %v\n%s", err, req.Body)
	}
	if body["phoneNumber"] != "+14155552671" || body["countryCode"] != "US" {
		t.Fatalf("unexpected request body: %s", req.Body)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestPhoneValidationBatchParsesCSVPhoneNumbers(t *testing.T) {
	_, stderr, requests, err := runAPICommand(t,
		`{"success":true,"totalProcessed":2}`,
		"phone-validation", "validate-batch", "--phone-numbers", "+14155552671,+442079460958", "--default-country-code", "US", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	var body struct {
		PhoneNumbers       []string `json:"phoneNumbers"`
		DefaultCountryCode string   `json:"defaultCountryCode"`
	}
	if err := json.Unmarshal([]byte(requests[0].Body), &body); err != nil {
		t.Fatalf("request body is invalid JSON: %v\n%s", err, requests[0].Body)
	}
	if got := strings.Join(body.PhoneNumbers, ","); got != "+14155552671,+442079460958" {
		t.Fatalf("phoneNumbers = %q, want both numbers", got)
	}
	if body.DefaultCountryCode != "US" {
		t.Fatalf("defaultCountryCode = %q, want US", body.DefaultCountryCode)
	}
	assertNoWarning(t, stderr)
}

func TestWebScrapingPostsScrapeBody(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"title":"Example","text":"Hello"}`,
		"web-scraping", "--url", "https://example.com", "--format", "markdown", "--wait-until", "load", "--agent", "--no-cache",
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
	if req.Path != "/web-scraping/scrape" {
		t.Fatalf("path = %s, want /web-scraping/scrape", req.Path)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("request body is invalid JSON: %v\n%s", err, req.Body)
	}
	if body["url"] != "https://example.com" || body["format"] != "markdown" || body["waitUntil"] != "load" {
		t.Fatalf("unexpected request body: %s", req.Body)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestAgentOutputSuppressesGeneratedMutationPartialFailureWarning(t *testing.T) {
	stdout, stderr, _, err := runAPICommand(t,
		`{"partialFailureError":{"code":3,"message":"one lookup failed"},"results":[{"resourceName":"aircraft/N12345"}]}`,
		"aircraft", "batch-lookup", "--tails", `["N12345","BAD"]`, "--agent", "--allow-partial-failure", "--no-cache",
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
