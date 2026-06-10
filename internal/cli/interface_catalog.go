// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: shared interface discovery from the live Cobra tree.

package cli

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// InterfaceInfo describes one API interface slug exposed by the CLI.
type InterfaceInfo struct {
	Name  string
	Short string
}

// CollectInterfaceCatalog walks root and returns every interface slug the CLI
// knows about, regardless of the active-apis manifest.
func CollectInterfaceCatalog(root *cobra.Command) map[string]InterfaceInfo {
	if root == nil {
		return nil
	}
	meta := map[string]InterfaceInfo{}
	for _, child := range root.Commands() {
		name := child.Name()
		if child.Hidden {
			meta[name] = InterfaceInfo{Name: name, Short: child.Short}
			continue
		}
		if child.Annotations[annEndpoint] != "" {
			iface, _, _ := strings.Cut(child.Annotations[annEndpoint], ".")
			if iface == "" {
				iface = name
			}
			if _, exists := meta[iface]; !exists {
				meta[iface] = InterfaceInfo{Name: iface, Short: child.Short}
			}
		}
	}
	return meta
}

// KnownInterfaceSlugs returns sorted interface slugs from the live Cobra tree.
func KnownInterfaceSlugs(root *cobra.Command) []string {
	meta := CollectInterfaceCatalog(root)
	out := make([]string, 0, len(meta))
	for name := range meta {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ListActiveInterfaces returns sorted slugs from the embedded manifest.
func ListActiveInterfaces() []string {
	if activeAPISet == nil {
		return nil
	}
	out := make([]string, 0, len(activeAPISet))
	for slug := range activeAPISet {
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

// ListActiveInterfaceCatalog returns active interfaces with descriptions.
func ListActiveInterfaceCatalog(root *cobra.Command) []InterfaceInfo {
	meta := CollectInterfaceCatalog(root)
	out := make([]InterfaceInfo, 0, len(meta))
	for name, info := range meta {
		if !IsActiveInterface(name) {
			continue
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// InterfaceEndpoints returns method names for an interface slug.
func InterfaceEndpoints(root *cobra.Command, iface string) []string {
	var methods []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, child := range c.Commands() {
			if ep := child.Annotations[annEndpoint]; ep != "" {
				name, method, ok := strings.Cut(ep, ".")
				if ok && name == iface {
					methods = append(methods, method)
				}
			}
			walk(child)
		}
	}
	walk(root)
	sort.Strings(methods)
	return methods
}
