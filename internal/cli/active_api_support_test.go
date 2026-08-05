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
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path = path + "?" + r.URL.RawQuery
		}
		requests = append(requests, capturedAPIRequest{
			Method: r.Method,
			Path:   path,
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

func TestWeatherCurrentGetsQueryParams(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"temperature":72}`,
		"weather", "current", "--lat", "40.71", "--lon", "-74.01", "--units", "fahrenheit", "--agent", "--data-source", "live", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	if requests[0].Method != http.MethodGet {
		t.Fatalf("method = %s, want GET", requests[0].Method)
	}
	if !strings.HasPrefix(requests[0].Path, "/weather/current?") && requests[0].Path != "/weather/current" {
		t.Fatalf("path = %s, want /weather/current?...", requests[0].Path)
	}
	if !strings.Contains(requests[0].Path, "lat=40.71") || !strings.Contains(requests[0].Path, "lon=-74.01") {
		t.Fatalf("missing lat/lon query in %s", requests[0].Path)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestGeocodingForwardGetsQuery(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"results":[{"lat":40.75,"lon":-73.99}]}`,
		"geocoding", "forward", "--q", "Empire State Building", "--limit", "3", "--agent", "--data-source", "live", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 || requests[0].Method != http.MethodGet {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	if !strings.HasPrefix(requests[0].Path, "/geocoding/forward") {
		t.Fatalf("path = %s, want /geocoding/forward...", requests[0].Path)
	}
	if !strings.Contains(requests[0].Path, "q=Empire+State+Building") && !strings.Contains(requests[0].Path, "q=Empire%20State%20Building") {
		t.Fatalf("missing q query in %s", requests[0].Path)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestQRCodeGeneratePostsBody(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"format":"png"}`,
		"qr-code", "generate", "--text", "https://datpaq.com", "--format", "png", "--size", "128", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 || requests[0].Method != http.MethodPost || requests[0].Path != "/qr-code/generate" {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(requests[0].Body), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["text"] != "https://datpaq.com" || body["format"] != "png" {
		t.Fatalf("unexpected body: %s", requests[0].Body)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestWebSearchSearchGetsQuery(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"results":[]}`,
		"web-search", "search", "--q", "datpaq api", "--limit", "5", "--agent", "--data-source", "live", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 || requests[0].Method != http.MethodGet || !strings.HasPrefix(requests[0].Path, "/web-search/search") {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestCalendarMonthGetsQuery(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"days":[]}`,
		"calendar", "month", "--year", "2026", "--month", "8", "--country-code", "US", "--include-holidays", "--agent", "--data-source", "live", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 || requests[0].Method != http.MethodGet || !strings.HasPrefix(requests[0].Path, "/calendar/month") {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	for _, want := range []string{"year=2026", "month=8", "country_code=US", "include_holidays=true"} {
		if !strings.Contains(requests[0].Path, want) {
			t.Fatalf("missing %s in %s", want, requests[0].Path)
		}
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}

func TestPDFGenerationFromURLPostsBody(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"success":true,"responseType":"base64"}`,
		"pdf-generation", "from-url", "--url", "https://example.com", "--response-type", "base64", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 1 || requests[0].Method != http.MethodPost || requests[0].Path != "/pdf-generation/from-url" {
		t.Fatalf("unexpected requests: %+v", requests)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(requests[0].Body), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["url"] != "https://example.com" || body["responseType"] != "base64" {
		t.Fatalf("unexpected body: %s", requests[0].Body)
	}
	assertJSONOnly(t, stdout)
	assertNoWarning(t, stderr)
}


func TestGeocodingBatchDryRunSucceedsWithoutQueries(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t,
		`{"ignored":true}`,
		"geocoding", "batch", "--dry-run", "--agent",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(requests) != 0 {
		t.Fatalf("dry-run should not dial network, got %+v", requests)
	}
	assertJSONOnly(t, stdout)
}

func TestQRCodeGenerateOmitsUnchangedDefaults(t *testing.T) {
	_, stderr, requests, err := runAPICommand(t,
		`{"success":true}`,
		"qr-code", "generate", "--text", "hello", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(requests[0].Body), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["text"] != "hello" {
		t.Fatalf("text missing: %s", requests[0].Body)
	}
	for _, unexpected := range []string{"format", "size", "margin", "darkColor", "lightColor", "responseType", "errorCorrectionLevel"} {
		if _, ok := body[unexpected]; ok {
			t.Fatalf("unchanged default %s should be omitted from body: %s", unexpected, requests[0].Body)
		}
	}
	assertNoWarning(t, stderr)
}

func TestPDFGenerationDefaultsResponseTypeBase64(t *testing.T) {
	_, stderr, requests, err := runAPICommand(t,
		`{"success":true}`,
		"pdf-generation", "from-url", "--url", "https://example.com", "--agent", "--no-cache",
	)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(requests[0].Body), &body); err != nil {
		t.Fatalf("body: %v", err)
	}
	if body["responseType"] != "base64" {
		t.Fatalf("responseType = %v, want base64", body["responseType"])
	}
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
