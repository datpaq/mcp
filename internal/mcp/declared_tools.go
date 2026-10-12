// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: typed MCP tools for the table-declared APIs in
// internal/cli/declared_api.go, so MCP parameters always match CLI wiring.

package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/datpaq/mcp/internal/cli"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerDeclaredAPITools(s *server.MCPServer) {
	for _, g := range cli.DeclaredAPIGroups() {
		for _, ep := range g.Endpoints {
			s.AddTool(declaredTool(g.Name, ep), declaredHandler(ep))
		}
	}
}

func declaredToolName(group string, ep cli.APIEndpoint) string {
	return group + "_" + ep.Name
}

func declaredPublicName(p cli.APIParam) string {
	return strings.ReplaceAll(p.Flag, "-", "_")
}

func declaredTool(group string, ep cli.APIEndpoint) mcplib.Tool {
	var required, optional []string
	for _, p := range ep.Params {
		if p.Required || p.In == "path" {
			required = append(required, declaredPublicName(p))
		} else {
			optional = append(optional, declaredPublicName(p))
		}
	}
	desc := ep.Short + "."
	if len(required) > 0 {
		desc += " Required: " + strings.Join(required, ", ") + "."
	}
	if len(optional) > 0 {
		desc += " Optional: " + strings.Join(optional, ", ") + "."
	}

	opts := []mcplib.ToolOption{mcplib.WithDescription(desc)}
	for _, p := range ep.Params {
		usage := p.Usage
		if p.Type == "file" || p.Type == "files" {
			usage += " (base64-encoded file content)"
		}
		popts := []mcplib.PropertyOption{mcplib.Description(usage)}
		if p.Required || p.In == "path" {
			popts = append(popts, mcplib.Required())
		}
		name := declaredPublicName(p)
		switch p.Type {
		case "int", "float":
			opts = append(opts, mcplib.WithNumber(name, popts...))
		case "bool":
			opts = append(opts, mcplib.WithBoolean(name, popts...))
		case "array", "files":
			opts = append(opts, mcplib.WithArray(name, popts...))
		case "object":
			opts = append(opts, mcplib.WithObject(name, popts...))
		default:
			opts = append(opts, mcplib.WithString(name, popts...))
		}
	}
	if ep.Method == "GET" {
		opts = append(opts, mcplib.WithReadOnlyHintAnnotation(true))
	}
	opts = append(opts,
		mcplib.WithDestructiveHintAnnotation(ep.Method == "DELETE"),
		mcplib.WithOpenWorldHintAnnotation(true),
	)
	return mcplib.NewTool(declaredToolName(group, ep), opts...)
}

func declaredHandler(ep cli.APIEndpoint) server.ToolHandlerFunc {
	var bindings []mcpParamBinding
	for _, p := range ep.Params {
		loc := p.In
		if loc == "" {
			loc = "query"
		}
		bindings = append(bindings, mcpParamBinding{PublicName: declaredPublicName(p), WireName: p.WireName(), Location: loc})
	}
	inner := makeAPIHandler(ep.Method, ep.Path, false, nil, bindings, nil)
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		normalizeDeclaredArgs(ep, req.GetArguments())
		return inner(ctx, req)
	}
}

// normalizeDeclaredArgs fixes values that makeAPIHandler would format with
// %v: JSON numbers bound to query/path become plain decimals (not 1e+06), and
// path values are URL-escaped. File params are passed through as base64; the
// MCP server never reads local paths on a caller's behalf.
func normalizeDeclaredArgs(ep cli.APIEndpoint, args map[string]any) {
	for _, p := range ep.Params {
		name := declaredPublicName(p)
		v, ok := args[name]
		if !ok || p.In == "body" {
			continue
		}
		s := fmt.Sprintf("%v", v)
		if f, isNum := v.(float64); isNum {
			s = strconv.FormatFloat(f, 'f', -1, 64)
		}
		if p.In == "path" {
			s = url.PathEscape(s)
		}
		args[name] = s
	}
}

// declaredContextResources lists declared groups for the MCP context tool.
func declaredContextResources() []map[string]any {
	var out []map[string]any
	for _, g := range cli.DeclaredAPIGroups() {
		names := make([]string, 0, len(g.Endpoints))
		for _, ep := range g.Endpoints {
			names = append(names, ep.Name)
		}
		out = append(out, map[string]any{
			"name":        g.Name,
			"description": g.Short,
			"endpoints":   names,
			"searchable":  true,
		})
	}
	return out
}
