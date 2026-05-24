// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: terminal port of the website's console-banner.tsx. Same
// glyphs, same 6-stop gradient, same 12-segment-per-line interpolation.
// Source of truth: console-banner.tsx in the datpaq website repo
// (src/components/console-banner.tsx). Keep the gradient stops and ASCII
// glyphs in sync with that file if it changes.
//
// 24-bit ANSI is used directly (no lipgloss dependency) to match the
// existing `red`/`green`/`bold` helpers in helpers.go.

package cli

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// DO NOT trim the leading newline — it visually separates the banner from
// the prompt that just rendered above it.
const bannerASCII = `
██████╗   █████╗  ████████╗ ██████╗   █████╗   ██████╗
██╔══██╗ ██╔══██╗ ╚══██╔══╝ ██╔══██╗ ██╔══██╗ ██╔═══██╗
██║  ██║ ███████║    ██║    ██████╔╝ ███████║ ██║   ██║
██║  ██║ ██╔══██║    ██║    ██╔═══╝  ██╔══██║ ██║▄▄ ██║
██████╔╝ ██║  ██║    ██║    ██║      ██║  ██║ ╚██████╔╝
╚═════╝  ╚═╝  ╚═╝    ╚═╝    ╚═╝      ╚═╝  ╚═╝  ╚══▀▀═╝
`

// Gradient stops copied verbatim from console-banner.tsx so the brand color
// ramp in the terminal matches the one in the browser console.
var bannerStops = []string{
	"#b8a3ff",
	"#c89cf5",
	"#d987ea",
	"#c66cd6",
	"#9d4ec0",
	"#6b21a8",
}

const bannerSegments = 12

// bannerColorEnabled is intentionally looser than colorEnabled():
// the banner is decorative chrome shown in interactive sessions, so we
// don't gate it on --human-friendly. NO_COLOR and dumb terminals still
// disable it, and a non-TTY (pipe/redirect) suppresses it entirely.
func bannerColorEnabled(w io.Writer) bool {
	if noColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(w)
}

// RenderBanner writes the DATPAQ banner to w. Falls back to uncolored ASCII
// when w isn't a TTY or color is disabled — never errors, never panics.
func RenderBanner(w io.Writer) {
	if !bannerColorEnabled(w) {
		// Suppress entirely in non-TTY contexts (pipes, file redirects, CI).
		// Printing raw escape-free ASCII art into a logfile or `grep` is just
		// noise — users who want to see it can run `datpaq` interactively.
		return
	}
	fmt.Fprint(w, renderBannerString())
	fmt.Fprintln(w, dim("  Welcome, developer. → https://datpaq.com"))
	fmt.Fprintln(w)
}

// renderBannerString returns the ANSI-colored banner as a string. Split out
// from RenderBanner so tests can assert the rendered escape sequences without
// caring about TTY detection.
//
// IMPORTANT: the banner glyphs (██╗╚═╝) are multi-byte UTF-8. We index by
// rune, not byte — slicing on byte boundaries would split a glyph mid-codepoint
// and render mojibake (literally visible as `█�` between ANSI color codes).
func renderBannerString() string {
	var b strings.Builder
	for _, line := range strings.Split(bannerASCII, "\n") {
		runes := []rune(line)
		if len(runes) == 0 {
			b.WriteByte('\n')
			continue
		}
		// Match the TS algorithm: chunkSize = ceil(len/SEGMENTS), and `t` is
		// the chunk-start index normalized to [0,1]. The TS source uses
		// string.length (a UTF-16 code-unit count); for these BMP glyphs
		// runes give the same logical width.
		chunkSize := int(math.Ceil(float64(len(runes)) / float64(bannerSegments)))
		if chunkSize < 1 {
			chunkSize = 1
		}
		for i := 0; i < len(runes); i += chunkSize {
			end := i + chunkSize
			if end > len(runes) {
				end = len(runes)
			}
			var t float64
			if len(runes) > 1 {
				t = float64(i) / float64(len(runes)-1)
			}
			r, g, bl := sampleGradient(bannerStops, t)
			b.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, bl))
			b.WriteString(string(runes[i:end]))
		}
		b.WriteString("\033[0m\n")
	}
	return b.String()
}

// sampleGradient returns the RGB triple at position t in [0,1] along the
// piecewise-linear gradient defined by stops. Port of the same-named TS
// function in console-banner.tsx.
func sampleGradient(stops []string, t float64) (int, int, int) {
	if len(stops) == 0 {
		return 255, 255, 255
	}
	if len(stops) == 1 {
		return hexToRGB(stops[0])
	}
	scaled := t * float64(len(stops)-1)
	i := int(math.Floor(scaled))
	if i > len(stops)-2 {
		i = len(stops) - 2
	}
	if i < 0 {
		i = 0
	}
	localT := scaled - float64(i)
	r1, g1, b1 := hexToRGB(stops[i])
	r2, g2, b2 := hexToRGB(stops[i+1])
	return lerp(r1, r2, localT), lerp(g1, g2, localT), lerp(b1, b2, localT)
}

func hexToRGB(hex string) (int, int, int) {
	h := strings.TrimPrefix(hex, "#")
	if len(h) != 6 {
		return 255, 255, 255
	}
	var r, g, b int
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
		return 255, 255, 255
	}
	return r, g, b
}

func lerp(a, b int, t float64) int {
	return int(math.Round(float64(a) + (float64(b)-float64(a))*t))
}

// dim renders text in ANSI dim grey for subtitles. Bypasses colorEnabled() so
// it stays aligned with the banner's own gating (TTY-only) rather than the
// stricter --human-friendly gate.
func dim(s string) string {
	if noColor {
		return s
	}
	return "\033[38;2;100;116;139m" + s + "\033[0m"
}

// RenderSplash writes a compact landing screen for bare `datpaq` invocations:
// banner + welcome paragraphs + a four-line pointer menu. The intent is "tell
// me what to do next in one screenful" — the exhaustive command/flag dump
// stays parked behind `datpaq --help`.
//
// Kept paragraphs are owned by the user (see commit history); changing them
// requires explicit intent, not a passing edit.
func RenderSplash(w io.Writer) {
	RenderBanner(w)
	fmt.Fprintln(w, "Manage datpaq resources via the datpaq API. Rate limits apply.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Add --agent to any command for JSON output + non-interactive mode.")
	fmt.Fprintln(w)
	// Two-column layout: command in default color, description dimmed. Widths
	// are hand-tuned for the entries below; if longer commands get added,
	// switch to text/tabwriter so alignment stays consistent.
	pointers := []struct{ cmd, desc string }{
		{"datpaq --help", "List all commands and flags"},
		{"datpaq api", "Browse all API endpoints"},
		{"datpaq exec", "Run an endpoint (prompts for required params)"},
		{"datpaq sample", "Copy-paste code samples for an endpoint"},
		{"datpaq doctor", "Check auth and connectivity"},
		{"datpaq auth login", "Sign in via browser"},
	}
	for _, p := range pointers {
		fmt.Fprintf(w, "  %-22s%s\n", p.cmd, dim(p.desc))
	}
	fmt.Fprintln(w)
}

// installTrailingNewline adds a blank line to stdout after every successful
// command, so dense output doesn't run flush against the next prompt.
//
// Gating is deliberately conservative:
//   - TTY only — a pipeline like `datpaq … | jq` would otherwise get an
//     extra blank line at EOF and break parsers that expect strict JSON.
//   - Skip machine-output modes (--json, --csv, --quiet, --plain, --agent)
//     even in a TTY: those flags signal "I want raw output, no chrome."
//   - Skip on error: cobra doesn't call PersistentPostRunE when RunE
//     returned an error, so failure messages stay flush with the prompt
//     automatically — no extra check needed here.
//
// Cobra's PersistentPostRunE walks up the parent chain and runs the first
// hook it finds, so a single registration on the root command covers every
// generated endpoint subcommand without touching their generated files.
func installTrailingNewline(rootCmd *cobra.Command, flags *rootFlags) {
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		if !isTerminal(os.Stdout) {
			return nil
		}
		if flags.asJSON || flags.csv || flags.quiet || flags.plain || flags.agent {
			return nil
		}
		fmt.Fprintln(os.Stdout)
		return nil
	}
}

// installBanner wires two behaviors onto the root command:
//   - bare `datpaq` (no subcommand) → RenderSplash
//   - `datpaq --help` → default cobra help, prefixed with the gradient banner
//
// Subcommand help (`datpaq <cmd> --help`) is untouched, so scripted callers
// piping --help output don't get banner noise.
func installBanner(rootCmd *cobra.Command) {
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == rootCmd {
			RenderBanner(cmd.OutOrStderr())
		}
		defaultHelp(cmd, args)
	})
	// Cobra only auto-invokes Help() on a bare parent command when RunE is
	// nil. Assigning RunE here intercepts that path so the splash shows
	// instead of the full help dump. Subcommands have their own RunE and are
	// dispatched before this ever runs.
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		RenderSplash(cmd.OutOrStdout())
		return nil
	}
}
