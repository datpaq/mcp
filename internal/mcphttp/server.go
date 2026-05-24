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

// NewHandler builds the public HTTP handler. baseURL targets the
// Datpaq REST API (production: https://datpaq.com/api/v1). It's a
// constructor argument rather than a hard-coded constant so staging
// deploys can point at a non-prod host without rebuilding.
//
// The returned handler mounts:
//
//	GET  /healthz  → 200 {"ok":true}
//	*    /mcp      → MCP streamable-http endpoint (requires Bearer)
func NewHandler(baseURL string) http.Handler {
	mcpServer := server.NewMCPServer(
		"Datpaq Proapi",
		"1.0.0",
		server.WithToolCapabilities(false),
	)
	// RegisterPublicTools (not RegisterTools): the hosted surface
	// strips local-state tools (search/sql/context) and the cobra-
	// tree CLI shell-out tools, both of which would expose the
	// server's filesystem and config to every authenticated tenant.
	mcptools.RegisterPublicTools(mcpServer)

	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithHTTPContextFunc(buildContextFunc(baseURL)),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.Handle("/mcp", requireBearer(httpServer))
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
