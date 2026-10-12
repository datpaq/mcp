// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: declarative endpoint table for ProApi services added after the
// last generator run. Each API file registers an APIGroup; this file builds the
// cobra commands, and internal/mcp builds typed MCP tools from the same table,
// so CLI flags, MCP params, sample snippets, and `which` stay in sync.

package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// APIParam is one request parameter of a declared endpoint.
type APIParam struct {
	Flag     string // CLI flag name (kebab-case); path params become positionals
	Wire     string // API wire name; defaults to Flag
	In       string // "query", "body", or "path"
	Type     string // "string", "int", "float", "bool", "array", "object", "file", or "files"
	Usage    string
	Example  string // sample value used in --help examples and tests
	Required bool
}

// WireName returns the API-side parameter name.
func (p APIParam) WireName() string {
	if p.Wire != "" {
		return p.Wire
	}
	return p.Flag
}

// APIEndpoint is one HTTP operation. Path is relative to the API base URL and
// may contain {wire} placeholders for path params.
type APIEndpoint struct {
	Name   string
	Short  string
	Method string
	Path   string
	Params []APIParam
}

// APIGroup is one ProApi service exposed as a top-level command.
type APIGroup struct {
	Name      string
	Short     string
	Endpoints []APIEndpoint
}

var declaredAPIGroups []APIGroup

func registerDeclaredAPI(g APIGroup) { declaredAPIGroups = append(declaredAPIGroups, g) }

// DeclaredAPIGroups returns every table-declared API group (read-only use).
func DeclaredAPIGroups() []APIGroup { return declaredAPIGroups }

const annWire = "pp:wire"

func newDeclaredAPICmd(flags *rootFlags, g APIGroup) *cobra.Command {
	cmd := &cobra.Command{
		Use:    g.Name,
		Short:  g.Short,
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}
	for _, ep := range g.Endpoints {
		cmd.AddCommand(newDeclaredEndpointCmd(flags, g.Name, ep))
	}
	return cmd
}

func newDeclaredEndpointCmd(flags *rootFlags, group string, ep APIEndpoint) *cobra.Command {
	var pathParams, flagParams []APIParam
	for _, p := range ep.Params {
		if p.In == "path" {
			pathParams = append(pathParams, p)
		} else {
			flagParams = append(flagParams, p)
		}
	}
	isBodyMethod := ep.Method == "POST" || ep.Method == "PUT" || ep.Method == "PATCH"
	var stdinBody bool

	use := ep.Name
	for _, p := range pathParams {
		use += " <" + p.Flag + ">"
	}
	annotations := map[string]string{
		"pp:endpoint": group + "." + ep.Name,
		"pp:method":   ep.Method,
		"pp:path":     ep.Path,
	}
	if ep.Method == "GET" {
		annotations["mcp:read-only"] = "true"
	}

	cmd := &cobra.Command{
		Use:         use,
		Short:       ep.Short,
		Example:     declaredExample(group, ep, pathParams, flagParams),
		Annotations: annotations,
		Args:        cobra.RangeArgs(0, len(pathParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < len(pathParams) && !flags.dryRun {
				return usageErr(fmt.Errorf("missing required argument <%s>", pathParams[len(args)].Flag))
			}
			path := ep.Path
			for i, p := range pathParams {
				v := p.Example
				if i < len(args) {
					v = args[i]
				}
				path = strings.Replace(path, "{"+p.WireName()+"}", url.PathEscape(v), 1)
			}

			useStdin := isBodyMethod && stdinBody
			query := map[string]string{}
			body := map[string]any{}
			for _, p := range flagParams {
				if useStdin && p.In == "body" {
					continue // the stdin JSON replaces every body flag
				}
				if p.Required {
					if err := requireFlag(cmd, flags, p.Flag); err != nil {
						return err
					}
				}
				if !cmd.Flags().Changed(p.Flag) {
					continue
				}
				raw := cmd.Flags().Lookup(p.Flag).Value.String()
				if p.Type == "float" {
					// pflag renders floats with %g (1e+06); APIs expect plain decimals.
					f, _ := cmd.Flags().GetFloat64(p.Flag)
					raw = strconv.FormatFloat(f, 'f', -1, 64)
				}
				if p.In == "body" {
					v, err := declaredBodyValue(p, raw)
					if err != nil {
						return usageErr(fmt.Errorf("parsing --%s: %w", p.Flag, err))
					}
					body[p.WireName()] = v
				} else {
					query[p.WireName()] = raw
				}
			}

			if useStdin {
				stdin, err := readStdinJSONBody()
				if err != nil {
					return err
				}
				body = stdin
			}

			switch ep.Method {
			case "GET":
				return printReadResponse(cmd, flags, group, path, query)
			case "DELETE":
				return deleteMutation(cmd, flags, group, path, query)
			default:
				return bodyMutation(cmd, flags, ep.Method, group, path, query, body)
			}
		},
	}

	for _, p := range flagParams {
		usage := p.Usage
		if p.Required {
			usage += " (required)"
		}
		switch p.Type {
		case "int":
			cmd.Flags().Int(p.Flag, 0, usage)
		case "float":
			cmd.Flags().Float64(p.Flag, 0, usage)
		case "bool":
			cmd.Flags().Bool(p.Flag, false, usage)
		default:
			cmd.Flags().String(p.Flag, "", usage)
		}
		_ = cmd.Flags().SetAnnotation(p.Flag, annWire, []string{p.WireName()})
	}
	if isBodyMethod {
		addHiddenStdinFlag(cmd, &stdinBody)
	}
	return cmd
}

// declaredBodyValue converts a flag's string value into its JSON body type.
func declaredBodyValue(p APIParam, raw string) (any, error) {
	switch p.Type {
	case "int", "float":
		n := json.Number(raw)
		if _, err := n.Float64(); err != nil {
			return nil, err
		}
		return n, nil
	case "bool":
		return raw == "true", nil
	case "array":
		return parseJSONOrCSVList(raw)
	case "object":
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, err
		}
		return v, nil
	case "file":
		return readFileBase64(raw)
	case "files":
		var out []string
		for _, path := range strings.Split(raw, ",") {
			if path = strings.TrimSpace(path); path == "" {
				continue
			}
			enc, err := readFileBase64(path)
			if err != nil {
				return nil, err
			}
			out = append(out, enc)
		}
		return out, nil
	default:
		return raw, nil
	}
}

// readFileBase64 loads a local file for APIs that take base64 file content.
func readFileBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// declaredExample lists path args and required flags only, matching the
// printing-press convention that `exec` relies on to detect required flags.
func declaredExample(group string, ep APIEndpoint, pathParams, flagParams []APIParam) string {
	parts := []string{"  datpaq", group, ep.Name}
	for _, p := range pathParams {
		parts = append(parts, shellQuoteExample(p.Example))
	}
	for _, p := range flagParams {
		if p.Required {
			parts = append(parts, "--"+p.Flag, shellQuoteExample(p.Example))
		}
	}
	return strings.Join(parts, " ")
}

func shellQuoteExample(v string) string {
	if v == "" || strings.ContainsAny(v, " \"'[]{}$&|;<>*?,") {
		return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
	}
	return v
}

func bodyMutation(cmd *cobra.Command, flags *rootFlags, method, resource, path string, query map[string]string, body map[string]any) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	send := c.PostWithParams
	switch method {
	case "PUT":
		send = c.PutWithParams
	case "PATCH":
		send = c.PatchWithParams
	}
	data, statusCode, err := send(path, query, body)
	if err != nil {
		return classifyAPIError(err, flags)
	}
	return printMutationResponseAs(cmd, flags, strings.ToLower(method), resource, path, statusCode, data)
}

func deleteMutation(cmd *cobra.Command, flags *rootFlags, resource, path string, query map[string]string) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	data, statusCode, err := c.DeleteWithParams(path, query)
	if err != nil {
		return classifyAPIError(err, flags)
	}
	return printMutationResponseAs(cmd, flags, "delete", resource, path, statusCode, data)
}

// declaredWhichEntries feeds every declared endpoint into `datpaq which`.
func declaredWhichEntries() []whichEntry {
	var out []whichEntry
	for _, g := range declaredAPIGroups {
		for _, ep := range g.Endpoints {
			out = append(out, whichEntry{Command: g.Name + " " + ep.Name, Description: ep.Short, Group: g.Name})
		}
	}
	return out
}
