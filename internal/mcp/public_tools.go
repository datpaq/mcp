// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored (not generated): curated tool registration for
// transports that serve untrusted, multi-tenant traffic (i.e. the
// hosted HTTP MCP server in cmd/datpaq-mcp-http).
//
// The generated RegisterTools(s) registers everything: the 80+ API-
// endpoint tools, plus three local-state tools (search/sql/context
// that read a sibling SQLite at ~/.local/share/datpaq/data.db) and a
// cobra-tree mirror that shells out to a sibling datpaq CLI binary on
// the host. All of those make sense on a single-user developer
// machine running the stdio binary. None of them make sense on a
// hosted server where the local filesystem and CLI belong to nobody
// in particular and reading them is at best useless, at worst a
// cross-tenant leak.

package mcp

import (
	"strings"

	"github.com/datpaq/mcp/internal/cli"
	"github.com/datpaq/mcp/internal/mcp/cobratree"
	"github.com/mark3labs/mcp-go/server"
)

// localStateToolNames are tools registered by RegisterTools that read
// local filesystem state (the sibling SQLite DB) or otherwise depend
// on the host being a specific user's machine. Stripped from the
// hosted HTTP surface.
//
// "context" is also stripped: it front-loads useful domain knowledge
// but its description ("Call this first") biases agents toward a tool
// that only works after a local `datpaq sync`, which a remote user
// has never run.
var localStateToolNames = []string{"search", "sql", "context"}

// RegisterPublicTools registers the curated tool set appropriate for a
// hosted, multi-tenant MCP transport. It populates everything that
// RegisterTools would, then removes:
//
//   - the local-state tools (search, sql, context)
//   - every shell-out tool that the cobra-tree mirror adds — those
//     would exec a sibling datpaq CLI binary on the host, against the
//     server's own ~/.config/datpaq/config.toml, on behalf of every
//     authenticated tenant.
//
// What remains are the generated API-endpoint tools, which the per-
// request client (constructed from the inbound Authorization header)
// drives against the Datpaq API on the calling user's behalf.
//
// On top of the curation, applies the active-API filter from
// internal/cli/active-apis.json — non-production interfaces are
// removed from the registered set, not just hidden in listings.
// This is stricter than the CLI's behavior (where inactive
// interfaces are still callable directly); the hosted MCP surface
// has no equivalent escape hatch, so unactivated APIs must not be
// reachable.
func RegisterPublicTools(s *server.MCPServer) {
	RegisterTools(s)
	s.DeleteTools(localStateToolNames...)
	s.DeleteTools(cobratree.ToolNames(cli.RootCmd())...)
	s.DeleteTools(inactiveAPIToolNames(s)...)
}

// inactiveAPIToolNames returns the names of currently-registered
// tools whose interface slug is not in the active-APIs manifest.
// Run AFTER the local-state and cobra-tree deletions so we only
// consider API-mirror tools.
//
// API-mirror tool names follow `{slug}_{action}` where slug is a
// hyphen-cased interface identifier ("convert-time",
// "image-processing", "aircraft") and action is whatever comes
// after. The slug never contains underscores, so splitting on the
// first '_' is sufficient. Tools with no underscore at all (e.g.
// the framework tools we already deleted) are skipped — they're
// not subject to activation gating.
func inactiveAPIToolNames(s *server.MCPServer) []string {
	var out []string
	for name := range s.ListTools() {
		slug := interfaceSlugFromToolName(name)
		if slug == "" {
			continue
		}
		if !cli.IsActiveInterface(slug) {
			out = append(out, name)
		}
	}
	return out
}

func interfaceSlugFromToolName(name string) string {
	i := strings.IndexByte(name, '_')
	if i <= 0 {
		return ""
	}
	return name[:i]
}
