// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: unit coverage for api_exec.go helpers. End-to-end
// dispatch is exercised by the smoke-test scripts in CI; this file
// targets the positional-detection logic that no active endpoint
// currently exercises at runtime.

package cli

import (
	"errors"
	"sort"
	"testing"

	"github.com/spf13/cobra"
)

func TestParsePositionalNames(t *testing.T) {
	cases := []struct {
		name string
		use  string
		want []string
	}{
		{"none", "current-time", nil},
		{"one", "lookup-by-icao <hex>", []string{"hex"}},
		{"two", "transfer <from> <to>", []string{"from", "to"}},
		{"hyphenated", "lookup <user-id>", []string{"user-id"}},
		{"underscored", "lookup <user_id>", []string{"user_id"}},
		{"square-brackets-ignored", "feedback [text]", nil},
		{"mixed", "save <name> [--flag <value>]", []string{"name", "value"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePositionalNames(tc.use)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCountLeadingPositionals(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"empty", nil, 0},
		{"one-positional", []string{"a12345"}, 1},
		{"two-positional", []string{"a", "b"}, 2},
		{"flag-only", []string{"--json"}, 0},
		{"positional-then-flag", []string{"a12345", "--json"}, 1},
		{"flag-first-stops-count", []string{"--json", "a12345"}, 0},
		{"shorthand-flag-stops-count", []string{"-v", "a"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countLeadingPositionals(tc.args)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestParseMissingRequiredFlag(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"cobra-single", errors.New(`required flag(s) "domain" not set`), "domain"},
		{"cobra-multiple", errors.New(`required flag(s) "domain", "email" not set`), "domain"},
		{"manual-check", errors.New(`required flag "domain" not set`), "domain"},
		{"unrelated", errors.New("HTTP 500 internal server error"), ""},
		{"wrapped", errors.New(`could not run: required flag "ip" not set somewhere`), "ip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseMissingRequiredFlag(tc.err)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsExecBlocked(t *testing.T) {
	if blocked, _ := isExecBlocked("image-processing"); !blocked {
		t.Error("image-processing should be blocked")
	}
	if blocked, _ := isExecBlocked("web-screenshot"); !blocked {
		t.Error("web-screenshot should be blocked")
	}
	if blocked, _ := isExecBlocked("ip-geolocation"); blocked {
		t.Error("ip-geolocation should NOT be blocked")
	}
}

// TestEndpointPositionalsAreDiscoverable verifies that for an endpoint
// known to take a positional path param, parsePositionalNames extracts
// it from the live cobra tree. This is the integration point between
// printing-press output and the exec prompting layer — if the printer
// changes Use-string conventions, this test catches it before the
// positional prompting silently no-ops.
//
// We use aircraft.lookup-by-icao because it's a stable generated
// example of '<hex>' that's been in the spec since v1.0.0. The
// interface is not currently in active-apis.json (exec wouldn't
// surface it to end users) — but the cobra command is still registered,
// so this test reaches the leaf without depending on the active set.
// TestRequiredFlagsMissing_FromExampleConvention exercises the printing-
// press Example-string heuristic that lets exec prompt for manually-
// checked required flags (the `Changed("X") && !flags.dryRun` pattern).
// The fake leaf mimics the shape printing-press emits: an Example listing
// the canonical invocation with required flags, real cobra flags
// registered but NO MarkFlagRequired call — exactly the configuration
// where my annotation-only detector would miss the requirement and the
// reactive retry loop is bypassed by --dry-run.
func TestRequiredFlagsMissing_FromExampleConvention(t *testing.T) {
	leaf := &cobra.Command{
		Use:     "demo",
		Example: "  datpaq demo --from miles --to km --value 42",
	}
	leaf.Flags().String("from", "", "source unit")
	leaf.Flags().String("to", "", "target unit")
	leaf.Flags().String("value", "", "numeric value")
	leaf.Flags().String("precision", "", "optional digits") // optional; not in Example

	got := requiredFlagsMissing(leaf, nil)
	names := make([]string, len(got))
	for i, p := range got {
		names[i] = p.name
	}
	sort.Strings(names)
	want := []string{"from", "to", "value"}
	if len(names) != len(want) {
		t.Fatalf("got %v, want %v", names, want)
	}
	for i := range names {
		if names[i] != want[i] {
			t.Errorf("[%d] got %q, want %q", i, names[i], want[i])
		}
	}

	// Already-provided flags must be filtered out (no duplicate prompt).
	got = requiredFlagsMissing(leaf, []string{"--from", "miles", "--value=42"})
	if len(got) != 1 || got[0].name != "to" {
		t.Errorf("expected only [to] missing, got %+v", got)
	}
}

// TestRequiredFlagsMissing_AnnotationStillWorks confirms the
// MarkFlagRequired path still feeds the same detector — the Example
// convention is additive, not a replacement.
func TestRequiredFlagsMissing_AnnotationStillWorks(t *testing.T) {
	leaf := &cobra.Command{Use: "demo"}
	leaf.Flags().String("id", "", "identifier")
	if err := leaf.MarkFlagRequired("id"); err != nil {
		t.Fatalf("MarkFlagRequired: %v", err)
	}
	got := requiredFlagsMissing(leaf, nil)
	if len(got) != 1 || got[0].name != "id" {
		t.Errorf("expected [id], got %+v", got)
	}
}

// TestRequiredFlagsMissing_BothSourcesDedupe verifies a flag that's
// required by BOTH signals only appears once in the missing list.
func TestRequiredFlagsMissing_BothSourcesDedupe(t *testing.T) {
	leaf := &cobra.Command{Use: "demo", Example: "  datpaq demo --id 7"}
	leaf.Flags().String("id", "", "identifier")
	if err := leaf.MarkFlagRequired("id"); err != nil {
		t.Fatalf("MarkFlagRequired: %v", err)
	}
	got := requiredFlagsMissing(leaf, nil)
	if len(got) != 1 || got[0].name != "id" {
		t.Errorf("expected single [id], got %+v", got)
	}
}

func TestEndpointPositionalsAreDiscoverable(t *testing.T) {
	root := RootCmd()
	endpoints := collectEndpoints(root)
	var found *endpointInfo
	for i := range endpoints {
		if endpoints[i].iface == "aircraft" && endpoints[i].method == "lookup-by-icao" {
			found = &endpoints[i]
			break
		}
	}
	if found == nil {
		t.Skip("aircraft.lookup-by-icao not registered in this build")
	}
	names := parsePositionalNames(found.cmd.Use)
	if len(names) != 1 || names[0] != "hex" {
		t.Errorf("expected [hex], got %v (Use: %q)", names, found.cmd.Use)
	}
}
