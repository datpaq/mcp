// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.

package mcp

import (
	"strings"
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
	if err := normalizeDeclaredArgs(ep, args); err != nil {
		t.Fatalf("normalizeDeclaredArgs: %v", err)
	}
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

func TestNormalizeDeclaredArgsRejectsMissingAndDotSegmentParams(t *testing.T) {
	ep := cli.APIEndpoint{Params: []cli.APIParam{
		{Flag: "id", In: "path"},
		{Flag: "text", In: "body", Required: true},
	}}
	for _, args := range []map[string]any{
		{"text": "hi"},
		{"id": nil, "text": "hi"},
		{"id": "esc_1"},
		{"id": "..", "text": "hi"},
		{"id": ".", "text": "hi"},
		{"id": "", "text": "hi"},
	} {
		if err := normalizeDeclaredArgs(ep, args); err == nil {
			t.Errorf("normalizeDeclaredArgs(%v) = nil, want error", args)
		}
	}
	if err := normalizeDeclaredArgs(ep, map[string]any{"id": "esc_1", "text": "hi"}); err != nil {
		t.Errorf("valid args rejected: %v", err)
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

func TestDeclaredParamDescriptionsAreMCPWorded(t *testing.T) {
	s := server.NewMCPServer("test", "0")
	RegisterTools(s)
	want := map[string]string{
		"ocr_parse.file":                "Base64-encoded PDF or image (or pass url)",
		"ocr_parse.url":                 "Public HTTP(S) file URL (or pass file)",
		"document-convert_convert.file": "Base64-encoded file to upload (or pass file_url)",
		"pdf-toolkit_run.file":          "Base64-encoded PDF for non-merge operations",
		"pdf-toolkit_run.files":         "Array of base64-encoded PDFs for merge (2-20)",
	}
	for _, g := range cli.DeclaredAPIGroups() {
		for _, ep := range g.Endpoints {
			props := s.GetTool(declaredToolName(g.Name, ep)).Tool.InputSchema.Properties
			for _, p := range ep.Params {
				key := declaredToolName(g.Name, ep) + "." + declaredPublicName(p)
				prop, _ := props[declaredPublicName(p)].(map[string]any)
				desc, _ := prop["description"].(string)
				for _, q := range ep.Params {
					if strings.Contains(desc, "--"+q.Flag) {
						t.Errorf("%s description %q names CLI flag --%s", key, desc, q.Flag)
					}
				}
				if p.Type == "file" || p.Type == "files" {
					if strings.Contains(strings.ToLower(desc), "local") || !strings.Contains(strings.ToLower(desc), "base64") {
						t.Errorf("%s description %q must describe base64 content, not local paths", key, desc)
					}
				}
				if p.Type == "files" {
					if items, _ := prop["items"].(map[string]any); items["type"] != "string" {
						t.Errorf("%s items = %v, want string items", key, prop["items"])
					}
				}
				if w, ok := want[key]; ok && desc != w {
					t.Errorf("%s description = %q, want %q", key, desc, w)
				}
				delete(want, key)
			}
		}
	}
	for key := range want {
		t.Errorf("param %s not found", key)
	}
}
