// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: hosted MCP transport for Datpaq.
//
// Wraps mcp-go's streamable-HTTP server with a Bearer-token auth
// gate. Each request must present a Datpaq API key as
// `Authorization: Bearer <key>`. The key is shaped into a
// per-request *client.Client and pinned into ctx via
// internal/mcp.ContextWithClient — the generated tool handlers in
// internal/mcp/tools.go read it from there.
//
// We do NOT validate the key against the Datpaq API in this layer.
// Doing so would add a round-trip to every MCP call (no realistic
// caching window — keys can be revoked at any time, and the API is
// the source of truth). Instead, an invalid key produces a 401 from
// the API on the first tool call, which the handler surfaces back to
// the MCP client as a tool-result error. Presence and shape are
// enforced here so we don't even pay the cost of constructing a
// doomed mcp-go session.

package mcphttp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/datpaq/mcp/internal/client"
	"github.com/datpaq/mcp/internal/config"
	mcptools "github.com/datpaq/mcp/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server identity, reported in the legacy InitializeResult and in the
// modern server/discover result. Keep serverVersion in step with the
// "version" field in server.json.
const (
	serverName    = "Datpaq Proapi"
	serverVersion = "1.0.2"

	// serverInstructions is the modern discover result's optional
	// natural-language guidance for LLMs.
	serverInstructions = "Datpaq exposes production data APIs as tools: " +
		"aircraft and vehicle lookups, IP geolocation and intelligence, " +
		"email/phone/domain validation, geocoding, weather, currency and " +
		"precious-metals rates, text and image processing, and more. " +
		"Call tools/list to see what is available."
)

// NewHandler builds the public HTTP handler. baseURL targets the
// Datpaq REST API (production: https://datpaq.com/api/v1). It's a
// constructor argument rather than a hard-coded constant so staging
// deploys can point at a non-prod host without rebuilding.
//
// The returned handler mounts:
//
//	GET  /healthz  → 200 {"ok":true}
//	*    /mcp      → MCP streamable-http endpoint (requires Bearer)
//	POST /         → same as /mcp (so the bare subdomain works)
//	GET  /         → 302 → https://datpaq.com/docs/mcp
//
// The /mcp endpoint is dual-era: modern (2026-07-28) requests are
// served by modern.go off the same tool registry, and everything else
// keeps going to mcp-go's handshake-based stack exactly as before.
// See newDualEraHandler for the dispatch rule.
func NewHandler(baseURL string) http.Handler {
	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)
	// RegisterPublicTools (not RegisterTools): the hosted surface
	// strips local-state tools (search/sql/context) and the cobra-
	// tree CLI shell-out tools, both of which would expose the
	// server's filesystem and config to every authenticated tenant.
	mcptools.RegisterPublicTools(mcpServer)

	ctxFunc := buildContextFunc(baseURL)

	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithHTTPContextFunc(ctxFunc),
	)

	// The modern handler reads the same registry (mcpServer.ListTools,
	// passed as a method value so it stays live) and reuses the same
	// ctxFunc, so both eras run identical tools through identical
	// handlers against the caller's own API key.
	modern := &modernHandler{
		tools:        mcpServer.ListTools,
		withClient:   ctxFunc,
		name:         serverName,
		version:      serverVersion,
		instructions: serverInstructions,
	}
	mcpHandler := requireBearer(newDualEraHandler(modern, httpServer))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.Handle("/mcp", mcpHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// ServeMux routes anything not matched by /healthz or /mcp here.
		// Only `/` itself is meaningful; deeper paths are 404.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		// MCP clients POST JSON-RPC. Route through the same auth +
		// handler chain as /mcp so the bare subdomain is a working
		// endpoint — avoids a redirect that some clients fumble on
		// POST (RFC 7231 §6.4.2: 301 may downgrade POST→GET).
		if r.Method == http.MethodPost {
			mcpHandler.ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, "https://datpaq.com/docs/mcp", http.StatusFound)
	})
	return mux
}

func buildContextFunc(baseURL string) func(ctx context.Context, r *http.Request) context.Context {
	return func(ctx context.Context, r *http.Request) context.Context {
		key := bearerToken(r.Header.Get("Authorization"))
		// requireBearer rejects empty keys before we get here, but
		// the contextFunc is also invoked from session bookkeeping
		// paths in mcp-go that don't share the auth middleware's
		// guarantee. Guarding here keeps the invariant local.
		if key == "" {
			return ctx
		}
		cfg := &config.Config{
			BaseURL:            baseURL,
			DatpaqApiKeyHeader: key,
			AuthSource:         "http-bearer",
		}
		c := client.New(cfg, 30*time.Second, 0)
		// Agents calling through MCP need fresh data every call —
		// the parallel reasoning in NewDiskConfigClient applies here
		// too. Multi-tenant cache hits could also leak data across
		// users on a shared instance, so this is also a safety
		// property, not just freshness.
		c.NoCache = true
		return mcptools.ContextWithClient(ctx, c)
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

// requireBearer rejects requests without an Authorization: Bearer
// header at the HTTP layer so we don't spin up an mcp-go session for
// a request that can't make any useful tool call.
func requireBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bearerToken(r.Header.Get("Authorization")) == "" {
			w.Header().Set("WWW-Authenticate",
				`Bearer realm="datpaq", error="invalid_token", error_description="missing or malformed Authorization header"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized","message":"present your Datpaq API key as 'Authorization: Bearer <key>'"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(authHeader string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}
	return strings.TrimSpace(authHeader[len(prefix):])
}
