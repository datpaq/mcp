// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch coverage: every table-declared endpoint builds the right request,
// is active in discovery, and is documented in spec.yaml.

package cli

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// declaredEndpointArgs builds a full invocation from each param's Example.
// File-typed params get a temp file whose content is the Example.
func declaredEndpointArgs(t *testing.T, group string, ep APIEndpoint) []string {
	args := []string{group, ep.Name}
	for _, p := range ep.Params {
		if p.In == "path" {
			args = append(args, p.Example)
		}
	}
	tempFile := func(name, content string) string {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	for _, p := range ep.Params {
		switch {
		case p.In == "path":
		case p.Type == "bool":
			args = append(args, "--"+p.Flag+"="+p.Example)
		case p.Type == "file":
			args = append(args, "--"+p.Flag, tempFile("a", p.Example))
		case p.Type == "files":
			args = append(args, "--"+p.Flag, tempFile("a", p.Example)+","+tempFile("b", p.Example))
		default:
			args = append(args, "--"+p.Flag, p.Example)
		}
	}
	return append(args, "--agent", "--data-source", "live", "--no-cache")
}

func TestDeclaredEndpointsBuildRequests(t *testing.T) {
	for _, g := range DeclaredAPIGroups() {
		for _, ep := range g.Endpoints {
			t.Run(g.Name+"."+ep.Name, func(t *testing.T) {
				stdout, stderr, requests, err := runAPICommand(t, `{"success":true}`, declaredEndpointArgs(t, g.Name, ep)...)
				if err != nil {
					t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
				}
				if len(requests) != 1 {
					t.Fatalf("requests = %d, want 1", len(requests))
				}
				req := requests[0]
				if req.Method != ep.Method {
					t.Fatalf("method = %s, want %s", req.Method, ep.Method)
				}
				wantPath := ep.Path
				for _, p := range ep.Params {
					if p.In == "path" {
						wantPath = strings.Replace(wantPath, "{"+p.WireName()+"}", url.PathEscape(p.Example), 1)
					}
				}
				gotPath, rawQuery, _ := strings.Cut(req.Path, "?")
				if gotPath != wantPath {
					t.Fatalf("path = %s, want %s", gotPath, wantPath)
				}
				query, _ := url.ParseQuery(rawQuery)
				var body map[string]any
				if req.Body != "" {
					if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
						t.Fatalf("body is not JSON: %v\n%s", err, req.Body)
					}
				}
				for _, p := range ep.Params {
					switch p.In {
					case "path":
					case "body":
						v, ok := body[p.WireName()]
						if !ok {
							t.Errorf("body missing %q: %s", p.WireName(), req.Body)
						}
						if p.Type == "file" && v != base64.StdEncoding.EncodeToString([]byte(p.Example)) {
							t.Errorf("body %q = %v, want base64 file content", p.WireName(), v)
						}
					default:
						if !query.Has(p.WireName()) {
							t.Errorf("query missing %q: %s", p.WireName(), rawQuery)
						}
					}
				}
				assertJSONOnly(t, stdout)
				assertNoWarning(t, stderr)
			})
		}
	}
}

func TestDeclaredRequiredFlagsAreEnforced(t *testing.T) {
	for _, g := range DeclaredAPIGroups() {
		for _, ep := range g.Endpoints {
			for _, p := range ep.Params {
				if !p.Required || p.In == "path" {
					continue
				}
				args := []string{g.Name, ep.Name}
				for _, pp := range ep.Params {
					if pp.In == "path" {
						args = append(args, pp.Example)
					}
				}
				_, _, requests, err := runAPICommand(t, `{"success":true}`, append(args, "--agent")...)
				if err == nil || !strings.Contains(err.Error(), `required flag "`) {
					t.Errorf("%s %s without required flags: err = %v, want required flag error", g.Name, ep.Name, err)
				}
				if len(requests) != 0 {
					t.Errorf("%s %s sent a request despite missing required flags", g.Name, ep.Name)
				}
				break
			}
		}
	}
}

func TestDeclaredAPIGroupsAreActive(t *testing.T) {
	for _, g := range DeclaredAPIGroups() {
		if !IsActiveInterface(g.Name) {
			t.Errorf("declared API %q missing from active-apis.json", g.Name)
		}
	}
}

func TestDeclaredGroupNamesDoNotCollide(t *testing.T) {
	seen := map[string]int{}
	for _, c := range RootCmd().Commands() {
		seen[c.Name()]++
		for _, a := range c.Aliases {
			seen[a]++
		}
	}
	for _, g := range DeclaredAPIGroups() {
		if seen[g.Name] != 1 {
			t.Errorf("declared API %q is registered %d times among root commands/aliases", g.Name, seen[g.Name])
		}
	}
}

func TestDeclaredFloatFlagsSendPlainDecimals(t *testing.T) {
	_, stderr, requests, err := runAPICommand(t, `{"success":true}`,
		"mortgage-calculator", "payment", "--interest-rate", "6.5", "--loan-amount", "1000000", "--agent", "--data-source", "live", "--no-cache")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	if len(requests) != 1 || !strings.Contains(requests[0].Path, "loan_amount=1000000") {
		t.Fatalf("want loan_amount=1000000 (not 1e+06), got %+v", requests)
	}
}

func TestDeclaredDeleteReportsDeleteAction(t *testing.T) {
	stdout, stderr, requests, err := runAPICommand(t, `{"success":true}`, "ping-human", "cancel", "esc_123", "--json")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s", err, stderr)
	}
	if len(requests) != 1 || requests[0].Method != "DELETE" {
		t.Fatalf("want one DELETE, got %+v", requests)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(stdout), &env); err != nil || env["action"] != "delete" {
		t.Fatalf("envelope action = %v (err %v), want delete\n%s", env["action"], err, stdout)
	}
}
