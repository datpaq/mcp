// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.

package mcp

import (
	"testing"

	"github.com/datpaq/mcp/internal/cli"
	"github.com/mark3labs/mcp-go/server"
)

func TestNormalizeDeclaredArgs(t *testing.T) {
	ep := cli.APIEndpoint{Params: []cli.APIParam{
		{Flag: "iban", In: "path"},
		{Flag: "min-cap", Type: "float"},
		{Flag: "file", In: "body", Type: "file"},
	}}
	args := map[string]any{"iban": "DE89 3704", "min_cap": float64(100000000000), "file": "/etc/passwd"}
	normalizeDeclaredArgs(ep, args)
	if args["iban"] != "DE89%203704" {
		t.Errorf("iban = %v, want URL-escaped path value", args["iban"])
	}
	if args["min_cap"] != "100000000000" {
		t.Errorf("min_cap = %v, want plain decimal", args["min_cap"])
	}
	if args["file"] != "/etc/passwd" {
		t.Errorf("file = %v, want unchanged (MCP must not read local files)", args["file"])
	}
}

func TestDeclaredAPIToolsRegisteredWithParams(t *testing.T) {
	s := server.NewMCPServer("test", "0")
	RegisterTools(s)
	for _, g := range cli.DeclaredAPIGroups() {
		for _, ep := range g.Endpoints {
			name := declaredToolName(g.Name, ep)
			tool := s.GetTool(name)
			if tool == nil {
				t.Errorf("MCP tool %q not registered", name)
				continue
			}
			for _, p := range ep.Params {
				if _, ok := tool.Tool.InputSchema.Properties[declaredPublicName(p)]; !ok {
					t.Errorf("MCP tool %q missing param %q", name, declaredPublicName(p))
				}
			}
		}
	}
}
