// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: modern (MCP 2026-07-28) front-door for the hosted
// Datpaq MCP server.
//
// 2026-07-28 replaced the initialize/Mcp-Session-Id handshake with a
// stateless model: every request self-describes its protocol version,
// client identity, and capabilities in `_meta`, mirrored into HTTP
// headers. mcp-go v0.47.0 only speaks the handshake-based revisions
// (2025-11-25 and earlier), so this file adds the modern half and
// newDualEraHandler picks between them per request.
//
// This is not a reimplementation of MCP. Datpaq's hosted surface is
// tools-only — WithToolCapabilities(false), no resources, prompts,
// sampling, roots, or logging — so the whole modern surface is three
// RPCs: server/discover, tools/list, tools/call. All three are served
// off the *same* mcp-go tool registry the legacy path uses
// (MCPServer.ListTools), so the two eras can never drift: there is one
// set of tools and one set of handlers behind both.
//
// Deliberately absent, because the spec deprecated or removed them and
// Datpaq never used them: MRTR (no tool asks for mid-call input), the
// Tasks extension, subscriptions/listen, and SSE responses (tool
// handlers are plain request/response and emit no progress
// notifications, so a single JSON object is always the right answer).
//
// When mark3labs/mcp-go — or a printing-press-compatible
// modelcontextprotocol/go-sdk — ships real dual-era support, this file
// is meant to be deleted wholesale rather than maintained.

package mcphttp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	// modernProtocolVersion is the stateless, per-request-metadata MCP
	// revision this file implements.
	modernProtocolVersion = "2026-07-28"

	// legacyProtocolVersion is the newest handshake-based revision
	// mcp-go negotiates for us on the other path. Advertised alongside
	// the modern one so a client that cannot do 2026-07-28 can see
	// there is something here it can talk to.
	legacyProtocolVersion = "2025-11-25"
)

// Headers the 2026-07-28 streamable-HTTP binding mirrors from the
// JSON-RPC body so intermediaries can route without parsing it.
const (
	headerProtocolVersion = "MCP-Protocol-Version"
	headerMcpMethod       = "Mcp-Method"
	headerMcpName         = "Mcp-Name"
)

// `_meta` keys defined by the spec. The prefix is mandatory.
const (
	metaKeyProtocolVersion = "io.modelcontextprotocol/protocolVersion"
	metaKeyServerInfo      = "io.modelcontextprotocol/serverInfo"
)

// JSON-RPC error codes. -32020 and -32022 come from the sub-range the
// MCP spec reserves for protocol-defined errors; the rest are stock
// JSON-RPC 2.0.
const (
	errCodeHeaderMismatch     = -32020
	errCodeUnsupportedVersion = -32022
	errCodeInvalidRequest     = -32600
	errCodeMethodNotFound     = -32601
	errCodeInternalError      = -32603
	errCodeParseError         = -32700
)

// Caching hints (spec: server/utilities/caching).
//
// cacheScope is "public" rather than "private" because the tool list
// really is identical for every tenant: RegisterPublicTools runs once
// at startup and applies a *global* active-API filter, and no code on
// the handler path filters tools per user. If per-user tool visibility
// is ever introduced, this MUST become "private" — the spec is explicit
// that a "public" result may be shared across authorization contexts.
const (
	discoverTTLMs  = 3_600_000 // 1h — identity and capabilities change on redeploy
	toolsListTTLMs = 300_000   // 5m — catalog changes on redeploy
	cacheScopeAll  = "public"
)

// maxModernBodyBytes caps a single JSON-RPC request body. Datpaq's
// largest tool arguments are base64 image payloads, which the API
// itself caps well below this.
const maxModernBodyBytes = 8 << 20 // 8 MiB

// Base64 sentinel wrapper for header values that cannot be sent as
// plain ASCII (spec: transports/streamable-http#value-encoding).
const (
	b64Prefix = "=?base64?"
	b64Suffix = "?="
)

// modernRequest is the envelope we need off a JSON-RPC message. Params
// stays raw so an unparseable payload for one method cannot fail a
// request for another.
type modernRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// modernParams covers the params fields the three RPCs we serve use.
type modernParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Meta      map[string]any  `json:"_meta,omitempty"`
}

// modernHandler serves the 2026-07-28 RPCs off the shared registry.
type modernHandler struct {
	// tools is MCPServer.ListTools — read live rather than snapshotted
	// so the modern and legacy views can never disagree.
	tools func() map[string]*server.ServerTool
	// withClient is buildContextFunc's result. This path bypasses
	// mcp-go's StreamableHTTPServer, so nothing else attaches the
	// per-request API client the generated handlers read out of ctx.
	withClient   func(ctx context.Context, r *http.Request) context.Context
	name         string
	version      string
	instructions string
}

// newDualEraHandler routes each request to the era it is speaking.
//
// The dispatch is deliberately conservative: a request reaches the
// modern handler only when it is a POST whose MCP-Protocol-Version
// header names a version we serve as modern. Everything else — no
// header, a handshake-era version, GET, DELETE — goes to the legacy
// stack completely untouched, so existing clients cannot regress.
//
// The one exception is version negotiation. An unknown version that
// also carries the per-request `_meta` protocol version can only have
// come from a modern client, so it gets UnsupportedProtocolVersionError
// listing what we do support instead of being dropped into a legacy
// handler that would answer it nonsensically.
func newDualEraHandler(modern, legacy http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			legacy.ServeHTTP(w, r)
			return
		}

		switch version := r.Header.Get(headerProtocolVersion); version {
		case modernProtocolVersion:
			modern.ServeHTTP(w, r)
		case "", legacyProtocolVersion, "2025-06-18", "2025-03-26", "2024-11-05":
			legacy.ServeHTTP(w, r)
		default:
			body, err := readAndRestoreBody(r)
			if err != nil || !hasMetaProtocolVersion(body) {
				legacy.ServeHTTP(w, r)
				return
			}
			var req modernRequest
			_ = json.Unmarshal(body, &req)
			writeJSONRPC(w, http.StatusBadRequest, req.ID, nil, &jsonRPCError{
				Code:    errCodeUnsupportedVersion,
				Message: "Unsupported protocol version",
				Data: map[string]any{
					"supported": []string{modernProtocolVersion, legacyProtocolVersion},
					"requested": version,
				},
			})
		}
	})
}

// errBodyTooLarge distinguishes an oversized request from a malformed
// one. io.LimitReader alone cannot: it truncates silently, and the
// truncated bytes then fail to parse, reporting a JSON error for a
// request whose JSON was fine.
var errBodyTooLarge = errors.New("request body exceeds the size limit")

// readModernBody reads the body, refusing anything over the cap rather
// than quietly cutting it short. It reads one byte past the limit so
// "at the limit" and "over it" are distinguishable.
func readModernBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxModernBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxModernBodyBytes {
		return nil, errBodyTooLarge
	}
	return body, nil
}

// readAndRestoreBody reads enough of r.Body to make the routing
// decision and puts the stream back intact.
//
// The restore is a MultiReader rather than a plain replay of what we
// read: an oversized body must reach the legacy stack byte-for-byte.
// mcp-go reads it with an unbounded io.ReadAll, so it accepts payloads
// this file would reject, and handing it a truncated body would turn a
// working legacy request into a parse error.
func readAndRestoreBody(r *http.Request) ([]byte, error) {
	peek, err := io.ReadAll(io.LimitReader(r.Body, maxModernBodyBytes+1))
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(peek), r.Body))
	return peek, nil
}

// hasMetaProtocolVersion reports whether the body carries the modern
// per-request protocol version. Only a modern client sends it, which
// makes it a reliable era tell when the header value is unrecognized.
func hasMetaProtocolVersion(body []byte) bool {
	var req modernRequest
	if err := json.Unmarshal(body, &req); err != nil || len(req.Params) == 0 {
		return false
	}
	var params modernParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return false
	}
	_, ok := params.Meta[metaKeyProtocolVersion]
	return ok
}

func (h *modernHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r) {
		// Spec requires 403 for an invalid Origin (DNS-rebinding
		// guard). The body has no id: we have not parsed one yet.
		writeJSONRPC(w, http.StatusForbidden, nil, nil, &jsonRPCError{
			Code:    errCodeInvalidRequest,
			Message: "Origin not allowed",
		})
		return
	}

	body, err := readModernBody(r)
	if errors.Is(err, errBodyTooLarge) {
		h.fail(w, http.StatusRequestEntityTooLarge, nil, errCodeInvalidRequest,
			fmt.Sprintf("Request body exceeds the %d byte limit", maxModernBodyBytes))
		return
	}
	if err != nil {
		h.fail(w, http.StatusBadRequest, nil, errCodeParseError, "Failed to read request body")
		return
	}

	var req modernRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.fail(w, http.StatusBadRequest, nil, errCodeParseError, "Failed to parse message")
		return
	}
	if req.JSONRPC != "2.0" {
		h.fail(w, http.StatusBadRequest, req.ID, errCodeInvalidRequest, "Invalid JSON-RPC version")
		return
	}

	// No id means a notification. This revision defines no
	// client-to-server notifications over streamable HTTP, but the
	// transport rules still say to acknowledge one with 202 and no body.
	if len(req.ID) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var params modernParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			h.fail(w, http.StatusBadRequest, req.ID, errCodeInvalidRequest, "Failed to parse params")
			return
		}
	}

	if msg, ok := checkMirroredHeaders(r, req.Method, &params); !ok {
		h.fail(w, http.StatusBadRequest, req.ID, errCodeHeaderMismatch, msg)
		return
	}

	switch req.Method {
	case "server/discover":
		writeJSONRPC(w, http.StatusOK, req.ID, h.discoverResult(), nil)
	case "tools/list":
		writeJSONRPC(w, http.StatusOK, req.ID, h.toolsListResult(), nil)
	case "tools/call":
		h.callTool(w, r, req.ID, &params)
	default:
		// 404, not 200. The status code is load-bearing: it is what
		// lets a dual-era client tell a modern server from a legacy
		// one. mcp-go answers an unknown method with 200 + -32601,
		// which gives such a client no fallback trigger at all.
		h.fail(w, http.StatusNotFound, req.ID, errCodeMethodNotFound, "Method not found: "+req.Method)
	}
}

// discoverResult answers server/discover, which the spec requires every
// modern server to implement and which dual-era clients use as their
// era probe.
func (h *modernHandler) discoverResult() map[string]any {
	return map[string]any{
		"resultType":        "complete",
		"supportedVersions": []string{modernProtocolVersion, legacyProtocolVersion},
		"capabilities":      map[string]any{"tools": map[string]any{}},
		"instructions":      h.instructions,
		"ttlMs":             discoverTTLMs,
		"cacheScope":        cacheScopeAll,
		"_meta": map[string]any{
			metaKeyServerInfo: map[string]any{"name": h.name, "version": h.version},
		},
	}
}

// toolsListResult returns every registered tool in one page. The
// curated public set is on the order of a hundred tools, comfortably
// inside a single response, so no cursor is issued and none is
// accepted.
func (h *modernHandler) toolsListResult() map[string]any {
	registry := h.tools()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)

	tools := make([]mcplib.Tool, 0, len(names))
	for _, name := range names {
		tools = append(tools, registry[name].Tool)
	}
	return map[string]any{
		"resultType": "complete",
		"tools":      tools,
		"ttlMs":      toolsListTTLMs,
		"cacheScope": cacheScopeAll,
	}
}

// callTool invokes the same handler the legacy path would, with the
// same ctx-attached client.
func (h *modernHandler) callTool(w http.ResponseWriter, r *http.Request, id json.RawMessage, params *modernParams) {
	tool, ok := h.tools()[params.Name]
	if !ok {
		h.fail(w, http.StatusNotFound, id, errCodeMethodNotFound, "Unknown tool: "+params.Name)
		return
	}

	var args any
	if len(params.Arguments) > 0 {
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			h.fail(w, http.StatusBadRequest, id, errCodeInvalidRequest, "Failed to parse arguments")
			return
		}
	}

	call := mcplib.CallToolRequest{Header: r.Header.Clone()}
	call.Params.Name = params.Name
	call.Params.Arguments = args

	result, err := tool.Handler(h.withClient(r.Context(), r), call)
	if err != nil {
		// A handler error is an application-level failure, not a
		// transport one, so it rides back on 200 like any other
		// JSON-RPC error response.
		h.fail(w, http.StatusOK, id, errCodeInternalError, err.Error())
		return
	}
	writeJSONRPC(w, http.StatusOK, id, result, nil)
}

// Missing mirrored headers are tolerated, so they can arrive on every
// single request. Logging each one would emit a line per request
// forever for one non-compliant client, so each kind is reported once
// per process — enough to learn that such clients exist, which is all
// this signal is for.
var (
	missingMethodHeaderOnce sync.Once
	missingNameHeaderOnce   sync.Once
)

func warnOnce(once *sync.Once, header string) {
	once.Do(func() {
		log.Printf("datpaq-mcp: serving modern requests that omit the %s header; "+
			"this is tolerated but non-compliant (logged once per process)", header)
	})
}

// checkMirroredHeaders enforces the spec's header/body agreement rule
// with one deliberate relaxation: a header that *disagrees* with the
// body is rejected, but a *missing* header is served anyway and logged.
//
// The mismatch case is the real security property — it stops a gateway
// routing, metering, or rate-limiting on one value while this server
// executes another. A missing header carries no such risk, and
// rejecting it would 400 every client that has not yet implemented the
// mirroring. The log lines are the signal for tightening this to full
// strictness once clients have caught up.
func checkMirroredHeaders(r *http.Request, method string, params *modernParams) (string, bool) {
	switch got := r.Header.Get(headerMcpMethod); {
	case got == "":
		warnOnce(&missingMethodHeaderOnce, headerMcpMethod)
	case got != method:
		return fmt.Sprintf("Header mismatch: %s header value %q does not match body value %q",
			headerMcpMethod, got, method), false
	}

	// Mcp-Name is required only for tools/call, resources/read, and
	// prompts/get. We serve just the first.
	if method != "tools/call" {
		return "", true
	}

	raw := r.Header.Get(headerMcpName)
	if raw == "" {
		warnOnce(&missingNameHeaderOnce, headerMcpName)
		return "", true
	}
	got, err := decodeHeaderValue(raw)
	if err != nil {
		return fmt.Sprintf("Header mismatch: %s header value is not valid base64", headerMcpName), false
	}
	if got != params.Name {
		return fmt.Sprintf("Header mismatch: %s header value %q does not match body value %q",
			headerMcpName, got, params.Name), false
	}
	return "", true
}

// decodeHeaderValue unwraps the `=?base64?...?=` sentinel clients use
// for header values that are not safe plain ASCII. A value without the
// sentinel is returned as-is.
func decodeHeaderValue(v string) (string, error) {
	if len(v) < len(b64Prefix)+len(b64Suffix) ||
		!strings.HasPrefix(v, b64Prefix) || !strings.HasSuffix(v, b64Suffix) {
		return v, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(v[len(b64Prefix) : len(v)-len(b64Suffix)])
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// originAllowed implements the spec's DNS-rebinding guard. MCP clients
// are not browsers and send no Origin at all, so the common case is
// "absent, allow". A request that does carry one came from a page, and
// the only pages with business here are Datpaq's own and a
// self-hoster's localhost.
//
// The comparison is on the parsed hostname, never a string prefix:
// prefix-matching "http://localhost" also accepts
// http://localhost.evil.example, which is a registerable domain an
// attacker can point anywhere — exactly the rebinding case this check
// exists to stop.
func originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch parsed.Hostname() {
	case "datpaq.com", "www.datpaq.com", "mcp.datpaq.com":
		return parsed.Scheme == "https"
	case "localhost", "127.0.0.1", "::1":
		// Self-hosters driving a local instance from a browser.
		return true
	}
	return false
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (h *modernHandler) fail(w http.ResponseWriter, status int, id json.RawMessage, code int, message string) {
	writeJSONRPC(w, status, id, nil, &jsonRPCError{Code: code, Message: message})
}

// writeJSONRPC emits a single JSON-RPC response. The modern transport
// permits either a JSON object or an SSE stream; Datpaq's handlers
// produce no interim notifications, so a plain object is always
// correct and every client is required to support it.
func writeJSONRPC(w http.ResponseWriter, status int, id json.RawMessage, result any, rpcErr *jsonRPCError) {
	body := map[string]any{"jsonrpc": "2.0"}
	// A null id is correct for errors raised before an id could be
	// parsed; the spec allows it and clients expect the key present.
	if len(id) > 0 {
		body["id"] = id
	} else {
		body["id"] = nil
	}
	if rpcErr != nil {
		body["error"] = rpcErr
	} else {
		body["result"] = result
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		http.Error(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"Failed to encode response"}}`,
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}
