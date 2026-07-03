// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored coverage for copy-pasteable API samples.

package cli

import (
	"bytes"
	"strings"
	"testing"
)

func runSampleCommand(t *testing.T, args ...string) string {
	t.Helper()

	var stdout bytes.Buffer
	cmd := RootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v\nstdout=%s", err, stdout.String())
	}
	return stdout.String()
}

func TestSampleConvertTimeUsesWireQueryNames(t *testing.T) {
	out := runSampleCommand(t, "sample", "convert-time")
	for _, want := range []string{"sourceTime=<string>", "sourceZone=<string>", "targetZone=<string>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("sample missing %q:\n%s", want, out)
		}
	}
	for _, stale := range []string{"source-time", "source-zone", "target-zone"} {
		if strings.Contains(out, stale) {
			t.Fatalf("sample contains stale flag-name query %q:\n%s", stale, out)
		}
	}
}

func TestSampleWebScrapingUsesWireBodyNames(t *testing.T) {
	out := runSampleCommand(t, "sample", "web-scraping")
	for _, want := range []string{`"url"`, `"format"`, `"waitUntil"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("sample missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "wait-until") {
		t.Fatalf("sample contains stale flag-name body key:\n%s", out)
	}
}

func TestSampleMxLookupGetUsesDomainsAndWireOptionNames(t *testing.T) {
	out := runSampleCommand(t, "sample", "mx-lookup", "get")
	for _, want := range []string{"domains=<string>", "resolve_ips=<bool>", "test_smtp=<bool>", "include_spf=<bool>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("sample missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "domain=<") || strings.Contains(out, "resolve-ips") || strings.Contains(out, "test-smtp") || strings.Contains(out, "include-spf") {
		t.Fatalf("sample contains stale MX query names:\n%s", out)
	}
}
