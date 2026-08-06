// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Tests for the modern (MCP 2026-07-28) front-door and the dual-era
// dispatch. The regression half — proving legacy clients are
// untouched — lives alongside these deliberately: the whole point of
// the shim is that adding an era costs the old one nothing.

package mcphttp

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mcptools "github.com/datpaq/mcp/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const testAPIKey = "Bearer test-key"

// modernPost builds a POST carrying the modern protocol header. Extra
// headers are applied last so a case can override or omit anything.
func modernPost(t *testing.T, h http.Handler, method, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(headerProtocolVersion, modernProtocolVersion)
	if method != "" {
		req.Header.Set(headerMcpMethod, method)
	}
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
			continue
		}
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// rpcResponse is the decoded JSON-RPC envelope, with result left raw
// so each test can pick out only what it asserts on.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"error"`
}

func decodeRPC(t *testing.T, rr *httptest.ResponseRecorder) rpcResponse {
	t.Helper()
	var got rpcResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decoding response body %q: %v", rr.Body.String(), err)
	}
	if got.JSONRPC != "2.0" {
		t.Fatalf("jsonrpc = %q, want \"2.0\"", got.JSONRPC)
	}
	return got
}

func TestModernDiscover(t *testing.T) {
	h := NewHandler("http://example.invalid")

	rr := modernPost(t, h, "server/discover", `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}

	var result struct {
		ResultType        string   `json:"resultType"`
		SupportedVersions []string `json:"supportedVersions"`
		TTLMs             int      `json:"ttlMs"`
		CacheScope        string   `json:"cacheScope"`
		Capabilities      struct {
			Tools *struct{} `json:"tools"`
		} `json:"capabilities"`
		Meta struct {
			ServerInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"io.modelcontextprotocol/serverInfo"`
		} `json:"_meta"`
	}
	envelope := decodeRPC(t, rr)
	// The id must come back verbatim or a client cannot match the
	// response to its in-flight request.
	if id, ok := envelope.ID.(float64); !ok || id != 1 {
		t.Errorf("id = %v, want 1", envelope.ID)
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decoding discover result: %v", err)
	}

	if result.ResultType != "complete" {
		t.Errorf("resultType = %q, want \"complete\"", result.ResultType)
	}
	if len(result.SupportedVersions) == 0 || result.SupportedVersions[0] != modernProtocolVersion {
		t.Errorf("supportedVersions = %v, want %q first", result.SupportedVersions, modernProtocolVersion)
	}
	if result.Capabilities.Tools == nil {
		t.Error("capabilities.tools missing — the server is tools-only and must advertise it")
	}
	if result.TTLMs != discoverTTLMs {
		t.Errorf("ttlMs = %d, want %d", result.TTLMs, discoverTTLMs)
	}
	if result.CacheScope != cacheScopeAll {
		t.Errorf("cacheScope = %q, want %q", result.CacheScope, cacheScopeAll)
	}
	if result.Meta.ServerInfo.Name != serverName || result.Meta.ServerInfo.Version != serverVersion {
		t.Errorf("serverInfo = %+v, want {%s %s}", result.Meta.ServerInfo, serverName, serverVersion)
	}
}

// TestModernToolsListMatchesRegistry is the anti-drift test: the modern
// era must expose exactly the curated set RegisterPublicTools builds,
// which is the same set the legacy era serves.
func TestModernToolsListMatchesRegistry(t *testing.T) {
	reference := server.NewMCPServer(serverName, serverVersion, server.WithToolCapabilities(false))
	mcptools.RegisterPublicTools(reference)
	want := reference.ListTools()
	if len(want) == 0 {
		t.Fatal("reference registry is empty — RegisterPublicTools registered nothing")
	}

	h := NewHandler("http://example.invalid")
	rr := modernPost(t, h, "tools/list", `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}

	var result struct {
		ResultType string `json:"resultType"`
		TTLMs      int    `json:"ttlMs"`
		CacheScope string `json:"cacheScope"`
		Tools      []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(decodeRPC(t, rr).Result, &result); err != nil {
		t.Fatalf("decoding tools/list result: %v", err)
	}

	if len(result.Tools) != len(want) {
		t.Errorf("tools count = %d, want %d", len(result.Tools), len(want))
	}
	got := make(map[string]bool, len(result.Tools))
	for _, tool := range result.Tools {
		got[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %q has an empty description", tool.Name)
		}
	}
	for name := range want {
		if !got[name] {
			t.Errorf("tool %q registered but missing from modern tools/list", name)
		}
	}

	// Names must be sorted: the response is cacheable, so a stable
	// ordering keeps client-side and upstream prompt caches from
	// churning on an identical catalog.
	for i := 1; i < len(result.Tools); i++ {
		if result.Tools[i-1].Name > result.Tools[i].Name {
			t.Fatalf("tools not sorted: %q before %q", result.Tools[i-1].Name, result.Tools[i].Name)
		}
	}

	if result.ResultType != "complete" {
		t.Errorf("resultType = %q, want \"complete\"", result.ResultType)
	}
	if result.TTLMs != toolsListTTLMs {
		t.Errorf("ttlMs = %d, want %d", result.TTLMs, toolsListTTLMs)
	}
	if result.CacheScope != cacheScopeAll {
		t.Errorf("cacheScope = %q, want %q", result.CacheScope, cacheScopeAll)
	}
}

// TestModernToolsCallReachesAPI proves the modern path runs the real
// generated handler with a real ctx-attached client — the piece that
// would silently break if buildContextFunc were not reused.
func TestModernToolsCallReachesAPI(t *testing.T) {
	var gotPath, gotQuery, gotAuth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The client forwards the key as x-api-key (see
		// internal/client/client.go), not as Authorization.
		gotPath, gotQuery, gotAuth = r.URL.Path, r.URL.RawQuery, r.Header.Get("x-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tail":"N12345","make":"CESSNA"}`))
	}))
	defer api.Close()

	h := NewHandler(api.URL)
	rr := modernPost(t, h, "tools/call",
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"aircraft_lookup-by-tail","arguments":{"tail":"N12345"},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		map[string]string{headerMcpName: "aircraft_lookup-by-tail"})

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	if gotPath != "/aircraft/lookup" {
		t.Errorf("upstream path = %q, want \"/aircraft/lookup\"", gotPath)
	}
	if !strings.Contains(gotQuery, "tail=N12345") {
		t.Errorf("upstream query = %q, want it to carry tail=N12345", gotQuery)
	}
	// The caller's own key must reach the API — this is the multi-tenant
	// property the hosted server is built on. requireBearer strips the
	// "Bearer " prefix, so the bare key is what goes upstream.
	if gotAuth != "test-key" {
		t.Errorf("upstream x-api-key = %q, want the caller's key %q", gotAuth, "test-key")
	}
	if !strings.Contains(rr.Body.String(), "CESSNA") {
		t.Errorf("response body did not carry the API payload: %s", rr.Body.String())
	}
}

func TestModernHeaderValidation(t *testing.T) {
	h := NewHandler("http://example.invalid")
	const callBody = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"aircraft_lookup-by-tail","arguments":{},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`

	cases := []struct {
		name       string
		method     string
		body       string
		headers    map[string]string
		wantStatus int
		wantCode   int
	}{
		{
			name:       "Mcp-Name disagreeing with body is rejected",
			method:     "tools/call",
			body:       callBody,
			headers:    map[string]string{headerMcpName: "some_other_tool"},
			wantStatus: http.StatusBadRequest,
			wantCode:   errCodeHeaderMismatch,
		},
		{
			name:       "Mcp-Method disagreeing with body is rejected",
			method:     "tools/list",
			body:       callBody,
			headers:    map[string]string{headerMcpName: "aircraft_lookup-by-tail"},
			wantStatus: http.StatusBadRequest,
			wantCode:   errCodeHeaderMismatch,
		},
		{
			name:   "base64-sentinel Mcp-Name is decoded before comparing",
			method: "tools/call",
			body:   callBody,
			headers: map[string]string{
				headerMcpName: b64Prefix + base64.StdEncoding.EncodeToString([]byte("aircraft_lookup-by-tail")) + b64Suffix,
			},
			wantStatus: http.StatusOK,
		},
		{
			// Deliberate relaxation: absent headers are served, so
			// clients that have not implemented mirroring still work.
			name:       "absent mirrored headers are served",
			method:     "",
			body:       callBody,
			headers:    map[string]string{headerMcpName: ""},
			wantStatus: http.StatusOK,
		},
		{
			name:       "malformed base64 Mcp-Name is rejected",
			method:     "tools/call",
			body:       callBody,
			headers:    map[string]string{headerMcpName: b64Prefix + "!!!not-base64!!!" + b64Suffix},
			wantStatus: http.StatusBadRequest,
			wantCode:   errCodeHeaderMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := modernPost(t, h, tc.method, tc.body, tc.headers)
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rr.Code, tc.wantStatus, rr.Body.String())
			}
			if tc.wantCode == 0 {
				return
			}
			got := decodeRPC(t, rr)
			if got.Error == nil {
				t.Fatalf("want error %d, got result: %s", tc.wantCode, rr.Body.String())
			}
			if got.Error.Code != tc.wantCode {
				t.Errorf("error code = %d, want %d", got.Error.Code, tc.wantCode)
			}
		})
	}
}

func TestModernUnsupportedProtocolVersion(t *testing.T) {
	h := NewHandler("http://example.invalid")

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01"}}}`))
	req.Header.Set("Authorization", testAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerProtocolVersion, "1900-01-01")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rr.Code, rr.Body.String())
	}
	got := decodeRPC(t, rr)
	if got.Error == nil || got.Error.Code != errCodeUnsupportedVersion {
		t.Fatalf("want error %d, got %s", errCodeUnsupportedVersion, rr.Body.String())
	}

	var data struct {
		Supported []string `json:"supported"`
		Requested string   `json:"requested"`
	}
	if err := json.Unmarshal(got.Error.Data, &data); err != nil {
		t.Fatalf("decoding error data: %v", err)
	}
	// The client cannot retry without this list.
	if len(data.Supported) == 0 || data.Supported[0] != modernProtocolVersion {
		t.Errorf("data.supported = %v, want %q first", data.Supported, modernProtocolVersion)
	}
	if data.Requested != "1900-01-01" {
		t.Errorf("data.requested = %q, want \"1900-01-01\"", data.Requested)
	}
}

// TestModernUnknownMethodReturns404 guards the defect this whole change
// exists to fix. mcp-go answers an unknown method with HTTP 200 and a
// -32601 body; the spec's era-detection fallback keys off a 4xx, so a
// 200 here leaves a dual-era client with no fallback trigger at all.
func TestModernUnknownMethodReturns404(t *testing.T) {
	h := NewHandler("http://example.invalid")

	for _, method := range []string{"resources/list", "prompts/list", "initialize", "tasks/get"} {
		t.Run(method, func(t *testing.T) {
			rr := modernPost(t, h, method,
				`{"jsonrpc":"2.0","id":1,"method":"`+method+`","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, nil)
			if rr.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 — a 200 here breaks dual-era client fallback (body: %s)",
					rr.Code, rr.Body.String())
			}
			got := decodeRPC(t, rr)
			if got.Error == nil || got.Error.Code != errCodeMethodNotFound {
				t.Fatalf("want error %d, got %s", errCodeMethodNotFound, rr.Body.String())
			}
		})
	}
}

func TestModernUnknownToolReturns404(t *testing.T) {
	h := NewHandler("http://example.invalid")
	rr := modernPost(t, h, "tools/call",
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"no_such_tool","arguments":{},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		map[string]string{headerMcpName: "no_such_tool"})

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %s)", rr.Code, rr.Body.String())
	}
	if got := decodeRPC(t, rr); got.Error == nil || got.Error.Code != errCodeMethodNotFound {
		t.Fatalf("want error %d, got %s", errCodeMethodNotFound, rr.Body.String())
	}
}

func TestModernNotificationReturns202(t *testing.T) {
	h := NewHandler("http://example.invalid")
	rr := modernPost(t, h, "notifications/cancelled",
		`{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, nil)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body: %s)", rr.Code, rr.Body.String())
	}
	if body := rr.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestModernOriginCheck(t *testing.T) {
	h := NewHandler("http://example.invalid")
	const body = `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`

	cases := []struct {
		origin     string
		wantStatus int
	}{
		{"", http.StatusOK},
		{"https://datpaq.com", http.StatusOK},
		{"https://mcp.datpaq.com", http.StatusOK},
		{"http://localhost:3000", http.StatusOK},
		{"http://127.0.0.1:8080", http.StatusOK},
		{"https://evil.example", http.StatusForbidden},
		{"https://datpaq.com.evil.example", http.StatusForbidden},
		// Suffix-extended hostnames: all are registerable domains an
		// attacker controls. A prefix match on the origin string lets
		// every one of these through, defeating the rebinding guard.
		{"http://localhost.evil.example", http.StatusForbidden},
		{"http://127.0.0.1.evil.example", http.StatusForbidden},
		{"http://localhost.attacker.co", http.StatusForbidden},
		// Right host, wrong scheme.
		{"http://datpaq.com", http.StatusForbidden},
		{"://not a url", http.StatusForbidden},
	}
	for _, tc := range cases {
		name := tc.origin
		if name == "" {
			name = "absent"
		}
		t.Run(name, func(t *testing.T) {
			rr := modernPost(t, h, "server/discover", body, map[string]string{"Origin": tc.origin})
			if rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestModernOversizedBodyIsRejectedNotTruncated pins the difference
// between refusing a large body and silently cutting it short. A
// truncating limit reports a JSON parse error for a request whose JSON
// was perfectly well-formed, sending the caller after the wrong bug.
func TestModernOversizedBodyIsRejectedNotTruncated(t *testing.T) {
	h := NewHandler("http://example.invalid")

	oversized := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"image-processing_image-compress","arguments":{"image":"` +
		strings.Repeat("A", maxModernBodyBytes) + `"},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`

	rr := modernPost(t, h, "tools/call", oversized,
		map[string]string{headerMcpName: "image-processing_image-compress"})

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rr.Code)
	}
	got := decodeRPC(t, rr)
	if got.Error == nil {
		t.Fatal("want an error response")
	}
	if got.Error.Code == errCodeParseError {
		t.Errorf("oversized body reported as a parse error (%d) — the JSON was valid, only too long",
			errCodeParseError)
	}
	if !strings.Contains(got.Error.Message, "limit") {
		t.Errorf("message = %q, want it to name the size limit", got.Error.Message)
	}
}

// TestDualEraPeekPreservesOversizedBody guards the nastier half of the
// same bug: the legacy stack reads bodies with an unbounded ReadAll, so
// the routing peek must hand on the whole body, not the prefix it
// happened to read.
func TestDualEraPeekPreservesOversizedBody(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"pad":"` +
		strings.Repeat("B", maxModernBodyBytes+1024) + `"}}`

	var seenLen int
	legacy := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		read, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("legacy handler failed reading body: %v", err)
		}
		seenLen = len(read)
	})
	h := newDualEraHandler(http.NotFoundHandler(), legacy)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set(headerProtocolVersion, "2099-01-01") // unknown → triggers the peek
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seenLen != len(body) {
		t.Fatalf("legacy handler saw %d bytes, want the full %d — the peek truncated it",
			seenLen, len(body))
	}
}

func TestModernRequiresBearer(t *testing.T) {
	h := NewHandler("http://example.invalid")
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`))
	req.Header.Set(headerProtocolVersion, modernProtocolVersion)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 — the modern path must sit behind the same auth gate", rr.Code)
	}
}

// TestDualEraDispatchPrefersLegacy pins the safety boundary: anything
// that is not explicitly a modern request must reach the untouched
// mcp-go stack. A modern handler that swallowed these would be an
// outage for every client connected today.
func TestDualEraDispatchPrefersLegacy(t *testing.T) {
	var served string
	legacy := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served = "legacy"
		w.WriteHeader(http.StatusOK)
	})
	modern := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served = "modern"
		w.WriteHeader(http.StatusOK)
	})
	h := newDualEraHandler(modern, legacy)

	cases := []struct {
		name       string
		httpMethod string
		version    string
		body       string
		want       string
	}{
		{"no version header", http.MethodPost, "", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, "legacy"},
		{"2025-11-25", http.MethodPost, "2025-11-25", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "legacy"},
		{"2025-06-18", http.MethodPost, "2025-06-18", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "legacy"},
		{"2025-03-26", http.MethodPost, "2025-03-26", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "legacy"},
		{"2024-11-05", http.MethodPost, "2024-11-05", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "legacy"},
		{"GET stays legacy", http.MethodGet, modernProtocolVersion, "", "legacy"},
		{"DELETE stays legacy", http.MethodDelete, modernProtocolVersion, "", "legacy"},
		// An unknown version with no modern _meta is not provably a
		// modern client, so it must not be hijacked.
		{"unknown version without _meta", http.MethodPost, "2099-01-01", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "legacy"},
		{"modern version", http.MethodPost, modernProtocolVersion, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "modern"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			served = ""
			req := httptest.NewRequest(tc.httpMethod, "/mcp", strings.NewReader(tc.body))
			if tc.version != "" {
				req.Header.Set(headerProtocolVersion, tc.version)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if served != tc.want {
				t.Fatalf("served by %q, want %q", served, tc.want)
			}
		})
	}
}

// TestLegacyInitializeStillNegotiates is the end-to-end no-regression
// proof: a handshake-era client must still get a real InitializeResult
// with the version it asked for, through the same wired-up handler the
// modern era now shares.
func TestLegacyInitializeStillNegotiates(t *testing.T) {
	h := NewHandler("http://example.invalid")

	for _, version := range []string{"2025-11-25", "2025-06-18", "2024-11-05"} {
		t.Run(version, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(
				`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`+version+
					`","capabilities":{},"clientInfo":{"name":"regression","version":"0"}}}`))
			req.Header.Set("Authorization", testAPIKey)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", rr.Code, rr.Body.String())
			}
			// mcp-go may answer as SSE; the JSON payload is on a data:
			// line either way, so assert on the substance.
			body := rr.Body.String()
			if !strings.Contains(body, `"serverInfo"`) || !strings.Contains(body, serverName) {
				t.Fatalf("legacy initialize did not return an InitializeResult: %s", body)
			}
			if !strings.Contains(body, version) {
				t.Errorf("negotiated version missing from result, want %q: %s", version, body)
			}
		})
	}
}

// TestDualEraRestoresBodyAfterPeek covers the subtle failure mode in
// the routing peek: deciding "legacy" after reading the body must not
// leave the legacy handler with an empty one.
func TestDualEraRestoresBodyAfterPeek(t *testing.T) {
	const body = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	var seen string
	legacy := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		buf := make([]byte, len(body))
		n, _ := r.Body.Read(buf)
		seen = string(buf[:n])
	})
	h := newDualEraHandler(http.NotFoundHandler(), legacy)

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set(headerProtocolVersion, "2099-01-01") // unknown → triggers the peek
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen != body {
		t.Fatalf("legacy handler read %q, want the full body %q", seen, body)
	}
}

func TestDecodeHeaderValue(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"us-west1", "us-west1", false},
		{"", "", false},
		{b64Prefix + base64.StdEncoding.EncodeToString([]byte("Hello, 世界")) + b64Suffix, "Hello, 世界", false},
		{b64Prefix + base64.StdEncoding.EncodeToString([]byte(" padded ")) + b64Suffix, " padded ", false},
		// Too short to hold both markers — must not panic on the slice.
		{"=?base64?=", "=?base64?=", false},
		{b64Prefix + "%%%" + b64Suffix, "", true},
	}
	for _, tc := range cases {
		got, err := decodeHeaderValue(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("decodeHeaderValue(%q) = %q, want error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("decodeHeaderValue(%q): unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("decodeHeaderValue(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
