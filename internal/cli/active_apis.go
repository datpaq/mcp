// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: loads the curated active-APIs list from an embedded JSON
// manifest. Source of truth is the admin dashboard's `isActive` field;
// data/active-apis.json mirrors that for the CLI's discovery surface.

package cli

import (
	_ "embed"
	"encoding/json"
)

// File lives next to its loader because //go:embed paths cannot escape the
// package directory (no `..` allowed). Despite being a `internal/cli` file,
// active-apis.json is the curated source of truth for which APIs the CLI
// surfaces in discovery — maintainers edit it directly when the admin
// dashboard's isActive set changes.
//
//go:embed active-apis.json
var activeAPIsRaw []byte

// activeAPIsManifest is the on-disk shape. Leading-underscore fields are
// metadata for humans reading the JSON and are deliberately not parsed.
type activeAPIsManifest struct {
	Active []string `json:"active"`
}

// activeAPISet is the parsed set, indexed for O(1) lookups. Populated once
// at process start by the package-init below. nil means "manifest missing
// or unparseable" — in that case isActive() returns true for any interface
// so the CLI degrades to showing everything rather than hiding everything.
var activeAPISet map[string]bool

func init() {
	if len(activeAPIsRaw) == 0 {
		return
	}
	var m activeAPIsManifest
	if err := json.Unmarshal(activeAPIsRaw, &m); err != nil {
		// Manifest is broken — degrade open. Better to over-show than to
		// silently hide every interface and make `datpaq api` look empty.
		return
	}
	if len(m.Active) == 0 {
		return
	}
	activeAPISet = make(map[string]bool, len(m.Active))
	for _, slug := range m.Active {
		activeAPISet[slug] = true
	}
}

// IsActiveInterface reports whether `slug` appears in the active-APIs
// manifest. When the manifest is missing, empty, or broken, every slug
// is treated as active (open-fail) so the CLI's discovery surface never
// goes blank on a deploy mistake.
//
// For the CLI, "active" affects DISCOVERY only — listings in `datpaq
// api`, `datpaq sample`, and the splash. Direct invocations like
// `datpaq aircraft lookup-by-tail` are unaffected; the underlying
// endpoint command stays registered and runnable regardless.
//
// The hosted MCP server (internal/mcp/public_tools.go) treats the
// same manifest more strictly: inactive interfaces are scrubbed from
// the registered tool set entirely, so an MCP client cannot call
// them at all. Exported (capital I) because the MCP package consumes
// it across the internal/cli boundary.
func IsActiveInterface(slug string) bool {
	if activeAPISet == nil {
		return true
	}
	return activeAPISet[slug]
}

// activeAPICount returns the number of curated active interfaces, or -1
// when no manifest is loaded. Callers use the -1 sentinel to suppress the
// "(N of M)" suffix in human-format listings.
func activeAPICount() int {
	if activeAPISet == nil {
		return -1
	}
	return len(activeAPISet)
}
