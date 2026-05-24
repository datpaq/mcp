// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored (not generated): name-enumeration helper. Mirrors the
// walk performed by RegisterAll but only collects tool names instead
// of registering handlers. Used by transports that need to scrub
// shell-out CLI tools from their public surface (e.g. the hosted HTTP
// MCP server, which cannot meaningfully execute sibling-CLI commands
// on behalf of a remote tenant).
//
// Lives in this package so it can reuse the unexported walker /
// classifier / name-builder without exporting more of the generated
// API than necessary.

package cobratree

import "github.com/spf13/cobra"

// ToolNames returns the names of every shell-out tool that RegisterAll
// would have registered against root, without actually registering
// them. Order matches RegisterAll's walk order.
func ToolNames(root *cobra.Command) []string {
	if root == nil {
		return nil
	}
	var names []string
	walk(root, nil, func(cmd *cobra.Command, path []string) {
		switch classify(cmd) {
		case commandHidden, commandEndpoint, commandFramework:
			return
		}
		if !cmd.Runnable() {
			return
		}
		if n := toolNameForPath(path); n != "" {
			names = append(names, n)
		}
	})
	return names
}
