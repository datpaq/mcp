// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hosted-surface coverage for the table-declared APIs
// (internal/cli/declared_api.go): they must be listed by the real
// handler, every active interface must be reachable, and nothing
// inactive or host-local may leak onto the multi-tenant surface.

package mcphttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/datpaq/mcp/internal/cli"
	"github.com/datpaq/mcp/internal/mcp/cobratree"
)

// hostedToolNames lists tools through the hosted handler's modern
// tools/list, the same registry the legacy era serves.
func hostedToolNames(t *testing.T) map[string]bool {
	t.Helper()
	h := NewHandler("http://example.invalid")
	rr := modernPost(t, h, "tools/list", `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(decodeRPC(t, rr).Result, &result); err != nil {
		t.Fatalf("decoding tools/list result: %v", err)
	}
	names := make(map[string]bool, len(result.Tools))
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}
	return names
}

// toolSlug maps a tool name to the longest active slug it is prefixed by.
func toolSlug(name string, active []string) string {
	best := ""
	for _, slug := range active {
		for _, prefix := range []string{slug, strings.ReplaceAll(slug, "-", "_")} {
			if (name == prefix || strings.HasPrefix(name, prefix+"_")) && len(slug) > len(best) {
				best = slug
			}
		}
	}
	return best
}

func TestHostedToolsListServesDeclaredAPIs(t *testing.T) {
	tools := hostedToolNames(t)

	for _, name := range []string{
		"breach-check_password",
		"iban_validate",
		"interest-rate_current",
		"libor_latest",
		"markdown-convert_to-html",
		"market-cap_lookup",
		"mortgage-calculator_payment",
		"ocr_parse",
		"pdf-toolkit_run",
		"ping-human_cancel",
		"translate_text",
		"unemployment_state",
		"vat-validation-and-rates_validate",
		"web-search_search",
	} {
		if !tools[name] {
			t.Errorf("hosted tools/list missing %q", name)
		}
	}

	// Every declared endpoint is served once its group is active.
	for _, g := range cli.DeclaredAPIGroups() {
		for _, ep := range g.Endpoints {
			if name := g.Name + "_" + ep.Name; !tools[name] {
				t.Errorf("hosted tools/list missing declared tool %q", name)
			}
		}
	}

	excluded := []string{
		// local-state tools
		"search", "sql", "context",
		// inactive interfaces
		"company-enrichment_company-enrich",
		"email-validation_email-validate-single",
		"exchange-rates-and-currency_exchange-rate-get",
		"exchange-rates-and-currency_exchange-rate-bulk",
		"phone-validation_validate",
		"precious-metals_prices",
		"secure-relay_relay",
		"schemas_sample-data",
	}
	// cobra-tree shell-out mirror
	excluded = append(excluded, cobratree.ToolNames(cli.RootCmd())...)
	for _, name := range excluded {
		if tools[name] {
			t.Errorf("hosted tools/list must not serve %q", name)
		}
	}

	// Every served tool belongs to an active interface, and every active
	// interface has at least one served tool.
	active := cli.ListActiveInterfaces()
	covered := map[string]bool{}
	for name := range tools {
		slug := toolSlug(name, active)
		if slug == "" {
			t.Errorf("hosted tool %q does not belong to an active interface", name)
			continue
		}
		covered[slug] = true
	}
	var missing []string
	for _, slug := range active {
		if !covered[slug] {
			missing = append(missing, slug)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("active interfaces with no hosted tool: %v", missing)
	}
}

// TestHostedDeclaredToolUsesCallerKey proves declared handlers run with
// the per-request client built from the caller's bearer token, not a
// client loaded from the server's own environment or config.
func TestHostedDeclaredToolUsesCallerKey(t *testing.T) {
	t.Setenv("DATPAQ_API_KEY", "server-key")
	var gotPath, gotQuery, gotKey string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotKey = r.URL.Path, r.URL.RawQuery, r.Header.Get("x-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"monthly_payment":6320.68}`))
	}))
	defer api.Close()

	h := NewHandler(api.URL)
	rr := modernPost(t, h, "tools/call",
		`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"mortgage-calculator_payment","arguments":{"interest_rate":6.5,"loan_amount":1000000},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		map[string]string{headerMcpName: "mortgage-calculator_payment"})

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	if gotPath != "/mortgage-calculator/payment" {
		t.Errorf("upstream path = %q, want /mortgage-calculator/payment", gotPath)
	}
	if !strings.Contains(gotQuery, "loan_amount=1000000") || !strings.Contains(gotQuery, "interest_rate=6.5") {
		t.Errorf("upstream query = %q, want plain-decimal loan_amount and interest_rate", gotQuery)
	}
	if gotKey != "test-key" {
		t.Errorf("upstream x-api-key = %q, want the caller's key %q", gotKey, "test-key")
	}
	if !strings.Contains(rr.Body.String(), "monthly_payment") {
		t.Errorf("response body did not carry the API payload: %s", rr.Body.String())
	}
}

// TestHostedFileParamsAreNotReadFromDisk proves file-typed params are
// forwarded as the caller's base64 string: the hosted server never
// treats them as paths on its own filesystem.
func TestHostedFileParamsAreNotReadFromDisk(t *testing.T) {
	var body map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":""}`))
	}))
	defer api.Close()

	h := NewHandler(api.URL)
	rr := modernPost(t, h, "tools/call",
		`{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"ocr_parse","arguments":{"file":"/etc/passwd","filename":"scan.pdf"},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		map[string]string{headerMcpName: "ocr_parse"})

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	if body["fileBase64"] != "/etc/passwd" {
		t.Errorf("upstream fileBase64 = %v, want the caller's value verbatim", body["fileBase64"])
	}
}
