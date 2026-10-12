// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// spec.yaml parity for declared endpoints, plus a maintenance helper
// (skipped by default) that emits OpenAPI tag/path YAML for
// declared endpoints missing from spec.yaml, in the "# TAGS / # PATHS"
// fragment layout. Usage:
//
//	DATPAQ_SPEC_FRAGMENT_OUT=/tmp/frag.yaml go test ./internal/cli -run TestEmitDeclaredSpecFragment

package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

var specTitleAcronyms = map[string]string{
	"api": "API", "gdp": "GDP", "iban": "IBAN", "libor": "LIBOR", "ocr": "OCR",
	"pdf": "PDF", "sec": "SEC", "vat": "VAT",
}

func specTitle(group string) string {
	words := strings.Split(group, "-")
	for i, w := range words {
		if a, ok := specTitleAcronyms[w]; ok {
			words[i] = a
		} else {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func specOperationID(group, name string) string {
	parts := strings.Split(group+"-"+name, "-")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func specSchemaType(p APIParam) string {
	switch p.Type {
	case "int":
		return "integer"
	case "float":
		return "number"
	case "bool":
		return "boolean"
	case "array", "object":
		return p.Type
	case "files":
		return "array"
	default:
		return "string"
	}
}

func specYAMLString(s string) string {
	return fmt.Sprintf("%q", s)
}

func writeSpecOperation(b *strings.Builder, group string, ep APIEndpoint) {
	fmt.Fprintf(b, "    %s:\n      tags:\n        - %s\n", strings.ToLower(ep.Method), specTitle(group))
	fmt.Fprintf(b, "      operationId: %s\n      summary: %s\n", specOperationID(group, ep.Name), specYAMLString(ep.Short))
	var params, body []APIParam
	for _, p := range ep.Params {
		if p.In == "body" {
			body = append(body, p)
		} else {
			params = append(params, p)
		}
	}
	if len(params) > 0 {
		b.WriteString("      parameters:\n")
		for _, p := range params {
			in := p.In
			if in == "" {
				in = "query"
			}
			fmt.Fprintf(b, "        - name: %s\n          in: %s\n          required: %t\n", p.WireName(), in, p.Required || in == "path")
			fmt.Fprintf(b, "          description: %s\n          schema:\n            type: %s\n", specYAMLString(p.Usage), specSchemaType(p))
			if p.Example != "" {
				example := specYAMLString(p.Example)
				if t := specSchemaType(p); t == "number" || t == "integer" || t == "boolean" {
					example = p.Example // scalar examples must match their schema type
				}
				fmt.Fprintf(b, "            example: %s\n", example)
			}
		}
	}
	if len(body) > 0 {
		b.WriteString("      requestBody:\n        required: true\n        content:\n          application/json:\n            schema:\n              type: object\n")
		var req []string
		for _, p := range body {
			if p.Required {
				req = append(req, p.WireName())
			}
		}
		if len(req) > 0 {
			b.WriteString("              required:\n")
			for _, r := range req {
				fmt.Fprintf(b, "                - %s\n", r)
			}
		}
		b.WriteString("              properties:\n")
		for _, p := range body {
			fmt.Fprintf(b, "                %s:\n                  type: %s\n                  description: %s\n", p.WireName(), specSchemaType(p), specYAMLString(p.Usage))
			if p.Type == "array" || p.Type == "files" {
				b.WriteString("                  items: {}\n")
			}
		}
	}
	b.WriteString(`      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/GenericSuccess"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "429":
          $ref: "#/components/responses/RateLimited"
        default:
          $ref: "#/components/responses/ServiceUnavailable"
`)
}

func TestEmitDeclaredSpecFragment(t *testing.T) {
	out := os.Getenv("DATPAQ_SPEC_FRAGMENT_OUT")
	if out == "" {
		t.Skip("set DATPAQ_SPEC_FRAGMENT_OUT to emit a spec fragment")
	}
	raw, err := os.ReadFile("../../spec.yaml")
	if err != nil {
		t.Fatal(err)
	}
	existing := specOperations(string(raw))
	var tags, paths strings.Builder
	for _, g := range DeclaredAPIGroups() {
		var missing []APIEndpoint
		for _, ep := range g.Endpoints {
			if _, ok := existing[strings.ToLower(ep.Method)+" "+ep.Path]; !ok {
				missing = append(missing, ep)
			}
		}
		if len(missing) == 0 {
			continue
		}
		fmt.Fprintf(&tags, "  - name: %s\n    description: %s\n", specTitle(g.Name), specYAMLString(g.Short))
		var order []string
		byPath := map[string][]APIEndpoint{}
		for _, ep := range missing {
			if _, ok := byPath[ep.Path]; !ok {
				order = append(order, ep.Path)
			}
			byPath[ep.Path] = append(byPath[ep.Path], ep)
		}
		for _, p := range order {
			fmt.Fprintf(&paths, "  %s:\n", p)
			for _, ep := range byPath[p] {
				writeSpecOperation(&paths, g.Name, ep)
			}
		}
	}
	frag := "# TAGS\n" + tags.String() + "# PATHS\n" + paths.String()
	if err := os.WriteFile(out, []byte(frag), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestDeclaredEndpointsDocumentedInSpec keeps spec.yaml (published as the
// website OpenAPI) in step with the CLI: every declared operation and
// parameter must appear under its path. Line-based to avoid a YAML dependency.
func TestDeclaredEndpointsDocumentedInSpec(t *testing.T) {
	raw, err := os.ReadFile("../../spec.yaml")
	if err != nil {
		t.Fatalf("read spec.yaml: %v", err)
	}
	ops := specOperations(string(raw))
	for _, g := range DeclaredAPIGroups() {
		for _, ep := range g.Endpoints {
			key := strings.ToLower(ep.Method) + " " + ep.Path
			op, ok := ops[key]
			if !ok {
				t.Errorf("spec.yaml missing %s", key)
				continue
			}
			for _, p := range ep.Params {
				name := regexp.QuoteMeta(p.WireName())
				re := regexp.MustCompile(`(?m)(- name: ` + name + `$|^ {14,}` + name + `:$)`)
				if !re.MatchString(op) {
					t.Errorf("spec.yaml %s missing parameter %q", key, p.WireName())
				}
			}
		}
	}
}

// specOperations maps "method /path" to that operation's YAML text.
func specOperations(spec string) map[string]string {
	out := map[string]string{}
	paths, _, _ := strings.Cut(spec, "\ncomponents:")
	var path, method string
	var buf strings.Builder
	flush := func() {
		if path != "" && method != "" {
			out[method+" "+path] = buf.String()
		}
		buf.Reset()
	}
	for _, line := range strings.Split(paths, "\n") {
		switch {
		case strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":"):
			flush()
			path, method = strings.TrimSuffix(strings.TrimSpace(line), ":"), ""
		case len(line) > 4 && strings.HasPrefix(line, "    ") && line[4] != ' ' && strings.HasSuffix(line, ":"):
			flush()
			method = strings.TrimSuffix(strings.TrimSpace(line), ":")
		default:
			buf.WriteString(line + "\n")
		}
	}
	flush()
	return out
}
