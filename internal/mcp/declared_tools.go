// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: typed MCP tools for the table-declared APIs in
// internal/cli/declared_api.go, so MCP parameters always match CLI wiring.

package mcp

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
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
		popts := []mcplib.PropertyOption{mcplib.Description(declaredParamUsage(ep, p))}
		if p.Required || p.In == "path" {
			popts = append(popts, mcplib.Required())
		}
		name := declaredPublicName(p)
		switch p.Type {
		case "int", "float":
			opts = append(opts, mcplib.WithNumber(name, popts...))
		case "bool":
			opts = append(opts, mcplib.WithBoolean(name, popts...))
		case "array":
			opts = append(opts, mcplib.WithArray(name, popts...))
		case "files":
			opts = append(opts, mcplib.WithArray(name, append(popts, mcplib.WithStringItems())...))
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

var cliFlagRef = regexp.MustCompile(`--([a-z0-9-]+)`)

// declaredParamUsage adapts CLI help for MCP: --flag references become param
// names, and file params describe base64 content instead of local paths.
func declaredParamUsage(ep cli.APIEndpoint, p cli.APIParam) string {
	names := map[string]string{}
	for _, q := range ep.Params {
		names[q.Flag] = declaredPublicName(q)
	}
	usage := cliFlagRef.ReplaceAllStringFunc(p.Usage, func(ref string) string {
		if name, ok := names[ref[2:]]; ok {
			return name
		}
		return ref
	})
	if p.Type != "file" && p.Type != "files" {
		return usage
	}
	usage = strings.TrimPrefix(usage, "Comma-separated ")
	usage = strings.TrimPrefix(strings.TrimPrefix(usage, "Local "), "local ")
	if p.Type == "files" {
		return "Array of base64-encoded " + usage
	}
	return "Base64-encoded " + usage
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
		if err := normalizeDeclaredArgs(ep, req.GetArguments()); err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		return inner(ctx, req)
	}
}

// normalizeDeclaredArgs rejects missing required or path params (mcp-go does
// not enforce the input schema, and a missing path param would leave its
// {placeholder} in the URL) and fixes values that makeAPIHandler would format
// with %v: JSON numbers bound to query/path become plain decimals (not
// 1e+06), and path values are URL-escaped, with dot segments rejected. File
// params are passed through as base64; the MCP server never reads local
// paths on a caller's behalf.
func normalizeDeclaredArgs(ep cli.APIEndpoint, args map[string]any) error {
	for _, p := range ep.Params {
		name := declaredPublicName(p)
		v, ok := args[name]
		if !ok || v == nil {
			if p.Required || p.In == "path" {
				return fmt.Errorf("missing required parameter %q", name)
			}
			continue
		}
		if p.In == "body" {
			continue
		}
		s := fmt.Sprintf("%v", v)
		if f, isNum := v.(float64); isNum {
			s = strconv.FormatFloat(f, 'f', -1, 64)
		}
		if p.In == "path" {
			if s == "" || s == "." || s == ".." {
				return fmt.Errorf("invalid value %q for path parameter %q", s, name)
			}
			s = url.PathEscape(s)
		}
		args[name] = s
	}
	return nil
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
