// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: `datpaq exec <iface>.<method>` — a curated execution
// surface on top of the generated endpoint commands. Resolves the dotted
// target back to its cobra leaf, prompts for missing required flags when
// stdin is a TTY, then re-dispatches through the root so persistent
// pre-run hooks (auth, profile, deliver) run exactly once per call.
//
// Registered as a top-level command in root.go (companion to api.go for
// browsing and sample.go for snippets). The endpoint tree walk is shared
// via collectEndpoints.

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// execBlocklist names interfaces excluded from `exec`. Their responses
// are binary payloads (encoded images, screenshot PNGs) that don't render
// as terminal-friendly JSON. v1 punts on auto-saving binary to a tmp file;
// users go through the dedicated commands which already handle file output.
//
// The reason strings flow into both the runtime error (when a user asks
// `exec image-processing.…`) and the `exec list` footer, so the redirect
// message stays consistent across surfaces.
var execBlocklist = map[string]string{
	"image-processing": "returns binary image data — run 'datpaq image-processing --help'",
	"web-screenshot":   "returns binary screenshot data — run 'datpaq web-screenshot --help'",
}

func isExecBlocked(iface string) (bool, string) {
	reason, ok := execBlocklist[iface]
	return ok, reason
}

func newAPIExecCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <interface>.<method> [args...]",
		Short: "Execute an API endpoint with prompted required params (pretty JSON output)",
		Long: `Execute any active API endpoint by its dotted name.

Missing required path parameters and flags are prompted interactively when
stdin is a terminal. With --no-input or a piped stdin, missing values
produce a usage error listing them by name so the call can be re-run
non-interactively in one shot.

Output is always pretty-printed JSON. To use a different format, call the
underlying command directly (e.g. 'datpaq aircraft lookup-by-icao …').

Two interfaces are excluded: image-processing and web-screenshot. Their
responses are binary and don't render in a terminal; call those through
their dedicated commands. Run 'datpaq exec list' for the full allowlist.`,
		Example: `  datpaq exec list
  datpaq exec ip-geolocation.get --ip 8.8.8.8
  datpaq exec whois.lookup --domain example.com
  datpaq exec aircraft.lookup-by-icao a12345`,
		Annotations: map[string]string{"mcp:read-only": "false"},
		// Disable cobra/pflag parsing entirely at this command. Two reasons:
		//
		//   - We don't know the endpoint until we read args[0], so we can't
		//     register the endpoint's flags upfront. Without
		//     DisableFlagParsing, pflag would error on every endpoint flag.
		//
		//   - The obvious alternative — FParseErrWhitelist.UnknownFlags —
		//     silently *consumes* unknown flags AND their values from args
		//     (verified against pflag v1.x). That works for boolean
		//     unknowns by accident, but `--domain example.com` becomes
		//     two-token deletion, so the leaf never sees the value.
		//
		// Cost of DisableFlagParsing: root persistent flags (--no-input,
		// --json, …) aren't parsed by cobra before we reach RunE. We
		// compensate by walking args ourselves at the top of RunE and
		// applying any known root flags via their value setters — the
		// flag bindings point into the rootFlags struct, so this updates
		// flags.* in place. The same tokens stay in args and get parsed
		// again by the nested cobra.Execute() below; pflag.Set is
		// idempotent so the double-apply is harmless.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if args[0] == "--help" || args[0] == "-h" {
				return cmd.Help()
			}

			// Pre-apply any root persistent flags found in args so
			// pre-dispatch checks below (canPromptForFlags reads
			// flags.noInput, …) see the correct state. Returns the args
			// with the endpoint target at endpointIdx so we can split
			// rootArgs vs endpointArgs cleanly for forwarding.
			endpointIdx, applyErr := applyRootFlagsInPlace(cmd.Root().PersistentFlags(), args)
			if applyErr != nil {
				return usageErr(applyErr)
			}
			if endpointIdx == -1 {
				// All args were root flags; no endpoint name. Show help.
				return cmd.Help()
			}

			target := args[endpointIdx]
			iface, method, ok := strings.Cut(target, ".")
			if !ok || iface == "" || method == "" {
				return usageErr(fmt.Errorf("invalid endpoint %q: use <interface>.<method> form\n  Run 'datpaq exec list' to see available endpoints.", target))
			}
			if blocked, reason := isExecBlocked(iface); blocked {
				return usageErr(fmt.Errorf("interface %q is not supported by 'exec': %s", iface, reason))
			}
			if !IsActiveInterface(iface) {
				return notFoundErr(fmt.Errorf("interface %q is not active for exec\n  Run 'datpaq exec list' to see available endpoints.", iface))
			}

			endpoints := collectEndpoints(cmd.Root())
			var ep *endpointInfo
			for i := range endpoints {
				if endpoints[i].iface == iface && endpoints[i].method == method {
					ep = &endpoints[i]
					break
				}
			}
			if ep == nil {
				return notFoundErr(fmt.Errorf("endpoint %q not found\n  Run 'datpaq exec list' to see available endpoints.", target))
			}

			// Split args around the endpoint name. rootArgs are the tokens
			// before it (already applied to flags.* by applyRootFlagsInPlace
			// above, but we keep them in the forwarded argv so the nested
			// Execute() sees them too). rest is everything after.
			rootArgs := append([]string(nil), args[:endpointIdx]...)
			rest := append([]string(nil), args[endpointIdx+1:]...)

			// Force pretty JSON. Appending --json is what distinguishes
			// exec from a raw call: the curated surface always emits JSON
			// regardless of other format flags in the user's profile/env.
			rest = append(rest, "--json")

			// Detect what's missing up-front so the prompt header can list
			// everything in one block instead of dribbling questions out
			// over multiple dispatch attempts. Two sources feed this:
			//
			//   - parsePositionalNames(Use)         → path positionals
			//   - requiredFlagsMissing(cmd, args)   → required flags from
			//     either MarkFlagRequired or the printing-press Example
			//     convention; both sources are needed so --dry-run can't
			//     hide a manually-checked requirement from the prompt.
			//
			// The reactive retry loop below stays as a backstop in case
			// some future endpoint enforces a requirement via a path
			// neither source catches.
			positionalNames := parsePositionalNames(ep.cmd.Use)
			provided := countLeadingPositionals(rest)
			var missingPositionals []string
			if provided < len(positionalNames) {
				missingPositionals = positionalNames[provided:]
			}
			missingFlagsPre := requiredFlagsMissing(ep.cmd, rest)

			// headerShown tracks whether the exec-header block has been
			// printed. Lazy printing — only when the first prompt fires —
			// keeps fully-supplied calls quiet (header on stderr would
			// otherwise leak into scripted users' log noise).
			headerShown := false
			showHeader := func() {
				if headerShown {
					return
				}
				headerShown = true
				printExecHeader(cmd.ErrOrStderr(), ep)
			}

			if len(missingPositionals) > 0 || len(missingFlagsPre) > 0 {
				if !canPromptForFlags(flags, cmd.InOrStdin()) {
					return missingParamsErr(missingPositionals, missingFlagsPre)
				}
				showHeader()
				if len(missingPositionals) > 0 {
					values, err := promptForPositionals(missingPositionals, cmd.InOrStdin(), cmd.ErrOrStderr())
					if err != nil {
						return err
					}
					// Positionals must precede flags so the leaf's flag
					// parser sees them as positional args. Prepend rather
					// than append.
					rest = append(values, rest...)
				}
				if len(missingFlagsPre) > 0 {
					supplied, err := promptForFlags(missingFlagsPre, cmd.InOrStdin(), cmd.ErrOrStderr())
					if err != nil {
						return err
					}
					rest = append(rest, supplied...)
				}
			}

			// Silence cobra's automatic error printing for the duration of
			// the dispatch loop. Without this, each retry attempt would
			// echo a stale "required flag X not set" to stderr, and the
			// final error would print twice (once by the inner Execute,
			// once by the outer). The defer restores the prior silence on
			// every exit path so cobra's outer Execute() prints the final
			// error exactly once.
			root := cmd.Root()
			prevSilence := root.SilenceErrors
			root.SilenceErrors = true
			defer func() { root.SilenceErrors = prevSilence }()

			for {
				// Dispatch shape: [root flags] iface method [endpoint args].
				// Preserving rootArgs at the front lets cobra parse them
				// at root level (their canonical home) during the nested
				// Execute, including the PersistentPreRunE meta-flags
				// like --agent that aggregate other flags. Without this,
				// flags would already be set from our pre-apply but the
				// aggregator wouldn't fire and dependent flags (json,
				// no-input, yes, no-color when --agent is passed) would
				// stay at their defaults.
				dispatchArgs := make([]string, 0, len(rootArgs)+2+len(rest))
				dispatchArgs = append(dispatchArgs, rootArgs...)
				dispatchArgs = append(dispatchArgs, iface, method)
				dispatchArgs = append(dispatchArgs, rest...)
				root.SetArgs(dispatchArgs)
				err := root.Execute()
				if err == nil {
					return nil
				}
				flagName := parseMissingRequiredFlag(err)
				if flagName == "" {
					return err
				}
				if !canPromptForFlags(flags, cmd.InOrStdin()) {
					return missingFlagsErr([]paramSpec{{name: flagName, required: true}})
				}
				showHeader()
				value, perr := readOnePromptedFlag(flagName, cmd.InOrStdin(), cmd.ErrOrStderr())
				if perr != nil {
					return perr
				}
				rest = append(rest, "--"+flagName, value)
			}
		},
	}
	cmd.AddCommand(newAPIExecListCmd(flags))
	return cmd
}

// requiredFlagsMissing returns every required flag on target that isn't
// already present in args, drawing from two signals so that all four
// failure modes are caught at the proactive prompt (not the reactive
// retry path, which --dry-run silently bypasses):
//
//   1. cobra's MarkFlagRequired annotation — hand-authored endpoints
//      that opt into cobra-level enforcement.
//   2. The printing-press Example convention — every required flag
//      for a generated endpoint appears in cmd.Example as `--name`,
//      and no optional flag does (verified against the 13 active
//      interfaces). This is the signal for endpoints that enforce
//      required-ness via the `if !cmd.Flags().Changed("X") &&
//      !flags.dryRun` pattern inside RunE, where --dry-run otherwise
//      makes the requirement invisible to my dispatch loop.
//
// Sources are merged with name-level dedup so a flag that's required
// by both signals only appears once. Path-param positional args are
// not modeled here — see parsePositionalNames + the v1 note in the
// exec command Long.
//
// If printing-press ever changes its Example format (e.g. starts
// listing optional flags), the example-derived set overcounts and
// prompts for non-required flags. The upstream fix is a `pp:required`
// annotation; until then this contract holds across every active
// endpoint as of printing-press 4.10.0.
func requiredFlagsMissing(target *cobra.Command, args []string) []paramSpec {
	provided := flagsInArgs(args)
	seen := map[string]bool{}
	var out []paramSpec
	add := func(name string) {
		if seen[name] || provided[name] {
			return
		}
		f := target.LocalFlags().Lookup(name)
		if f == nil {
			return
		}
		seen[name] = true
		out = append(out, paramSpec{
			name:        name,
			typeName:    f.Value.Type(),
			description: f.Usage,
			required:    true,
		})
	}
	// Source 1: cobra MarkFlagRequired annotation.
	target.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if vals := f.Annotations[cobra.BashCompOneRequiredFlag]; len(vals) > 0 && vals[0] == "true" {
			add(f.Name)
		}
	})
	// Source 2: printing-press Example convention.
	for _, m := range exampleFlagRE.FindAllStringSubmatch(target.Example, -1) {
		add(m[1])
	}
	return out
}

// flagsInArgs scans dispatch args for `--name` tokens (with or without
// =value) and returns a set of their names. Shorthand `-x` forms aren't
// scanned — printing-press generated endpoints don't register short
// forms on body/query flags, so omitting them avoids false negatives
// from misinterpreting `-h` etc.
func flagsInArgs(args []string) map[string]bool {
	have := map[string]bool{}
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			continue
		}
		name := strings.TrimPrefix(a, "--")
		if i := strings.Index(name, "="); i >= 0 {
			name = name[:i]
		}
		have[name] = true
	}
	return have
}

// exampleFlagRE captures `--name` tokens from a printing-press Example
// string. Names match cobra's allowed flag-name shape: starts with a
// letter, then alphanumerics, hyphens, or underscores. Anchored on `--`
// so positional placeholders ("example-value", "42") never match.
var exampleFlagRE = regexp.MustCompile(`--([a-zA-Z][a-zA-Z0-9_-]*)`)

// applyRootFlagsInPlace walks `args` and, for each `--name` token that
// names a known root persistent flag, calls flag.Value.Set(...) so the
// rootFlags struct sees the value. Returns the index of the first
// non-flag token (the endpoint name), or -1 if every token was a flag.
//
// We need this because exec uses DisableFlagParsing — cobra never gets
// a chance to populate flags.* from these tokens before RunE runs. The
// same args are forwarded to the nested Execute() below, where cobra
// parses them again through its normal pipeline; pflag.Value.Set is
// idempotent so the second pass is a no-op on the values themselves.
// The reason we DO want the second pass is the root PersistentPreRunE
// aggregations (--agent fanning out to --json/--no-input/--yes/...)
// which run there, not here.
//
// Tokens not recognized as root flags are skipped (left in args
// untouched). They're either endpoint flags or — once we cross the
// first non-flag token — endpoint positionals.
func applyRootFlagsInPlace(rootFlagSet *pflag.FlagSet, args []string) (int, error) {
	i := 0
	for i < len(args) {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			// First non-flag token — the endpoint name.
			return i, nil
		}
		// Long form only: printing-press doesn't register shorthands
		// for endpoint body/query flags, and root persistent flags use
		// long names exclusively. A bare `-x` here is treated as
		// "not a root flag" and left in args for the leaf parser to
		// surface its own error.
		if !strings.HasPrefix(a, "--") {
			i++
			continue
		}
		name := strings.TrimPrefix(a, "--")
		var value string
		hasValue := false
		if eq := strings.Index(name, "="); eq >= 0 {
			value = name[eq+1:]
			name = name[:eq]
			hasValue = true
		}
		f := rootFlagSet.Lookup(name)
		if f == nil {
			// Unknown to root — endpoint-level flag. Skip and keep in args.
			i++
			continue
		}
		switch {
		case hasValue:
			if err := f.Value.Set(value); err != nil {
				return -1, fmt.Errorf("invalid value for --%s: %w", name, err)
			}
			i++
		case f.Value.Type() == "bool":
			if err := f.Value.Set("true"); err != nil {
				return -1, fmt.Errorf("setting --%s: %w", name, err)
			}
			i++
		default:
			if i+1 >= len(args) {
				return -1, fmt.Errorf("flag --%s needs a value", name)
			}
			if err := f.Value.Set(args[i+1]); err != nil {
				return -1, fmt.Errorf("invalid value for --%s: %w", name, err)
			}
			i += 2
		}
	}
	return -1, nil
}

// canPromptForFlags reports whether we can interactively ask the user for
// missing values: stdin must be a TTY and --no-input must not be set.
func canPromptForFlags(flags *rootFlags, in io.Reader) bool {
	if flags != nil && flags.noInput {
		return false
	}
	inFile, _ := in.(*os.File)
	return inFile != nil && isTerminal(inFile)
}

// promptForFlags reads one line of input per missing required flag and
// returns a slice of "--name value" pairs ready to append to args.
// Caller is responsible for checking canPromptForFlags first and printing
// the exec header.
func promptForFlags(missing []paramSpec, in io.Reader, errOut io.Writer) ([]string, error) {
	reader := bufio.NewReader(in)
	out := make([]string, 0, len(missing)*2)
	for _, p := range missing {
		value, err := readPromptedValue(reader, errOut, p.name, p.description)
		if err != nil {
			return nil, err
		}
		out = append(out, "--"+p.name, value)
	}
	return out, nil
}

// promptForPositionals reads one line of input per missing path param and
// returns a slice of values to PREPEND to args (positionals must appear
// before any flag so the leaf's parser binds them to the right slot).
// Positional names come from the leaf's Use string and don't carry a
// separate description, so the prompt is just `? <name>: `.
func promptForPositionals(names []string, in io.Reader, errOut io.Writer) ([]string, error) {
	reader := bufio.NewReader(in)
	out := make([]string, 0, len(names))
	for _, n := range names {
		value, err := readPromptedValue(reader, errOut, n, "")
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

// readOnePromptedFlag prompts for a single flag's value. Used by the
// reactive retry loop where we only learn one missing flag per attempt
// from the leaf command's error message — so no upfront description is
// available.
func readOnePromptedFlag(name string, in io.Reader, errOut io.Writer) (string, error) {
	return readPromptedValue(bufio.NewReader(in), errOut, name, "")
}

// readPromptedValue renders one prompt and returns the trimmed input.
// Format: `? <name> (<description>): ` with ANSI accents when color is
// enabled (`?` green/bold, `<name>` bold). Empty input re-prompts with
// a dim hint; the value is never echoed back, by design — agents and
// log scrapers shouldn't see the raw input mirrored to stderr.
func readPromptedValue(reader *bufio.Reader, errOut io.Writer, name, description string) (string, error) {
	for {
		prompt := green(bold("?")) + " " + bold(name)
		if description != "" {
			prompt += " (" + truncate(description, 80) + ")"
		}
		prompt += ": "
		fmt.Fprint(errOut, prompt)

		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("reading input for %s: %w", name, err)
		}
		value := strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(errOut, "  "+yellow("required — enter a value or Ctrl-C to abort"))
	}
}

// printExecHeader prints a one-block banner for an interactive exec call:
// the dotted endpoint name (bold) and its short description on stderr.
// Only invoked when at least one prompt will fire, so non-interactive
// calls stay silent.
func printExecHeader(w io.Writer, ep *endpointInfo) {
	if ep == nil {
		return
	}
	target := ep.iface + "." + ep.method
	short := truncate(ep.short, 100)
	if short == "" {
		fmt.Fprintln(w, bold(target))
	} else {
		fmt.Fprintf(w, "%s %s %s\n", bold(target), "›", short)
	}
	fmt.Fprintln(w)
}

// positionalRE captures angle-bracketed placeholders in a cobra Use
// string. printing-press emits names like '<hex>', '<code>', '<id>',
// '<iso>', '<serial>' for path parameters. Square-bracketed [optional]
// placeholders are not endpoint positionals (they appear only on
// hand-authored top-level utility commands), so this pattern is
// intentionally narrower than `[^>]+` to skip those.
var positionalRE = regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9_-]*)>`)

func parsePositionalNames(use string) []string {
	matches := positionalRE.FindAllStringSubmatch(use, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// countLeadingPositionals counts non-flag tokens at the start of args.
// This is a heuristic: a user who interleaves positionals after a flag
// (e.g. `--json a12345`) will have the positional undercounted and be
// prompted to provide it again. For v1 that's acceptable — every active
// endpoint takes zero positionals, so the heuristic only matters when
// the active set expands. The fix, if needed later, is to run the
// leaf's flag set against args and use its returned positional slice
// instead of guessing here.
func countLeadingPositionals(args []string) int {
	n := 0
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			break
		}
		n++
	}
	return n
}

// missingParamsErr formats the non-interactive failure message for both
// positionals and flags in a single error so the user can re-run with
// every value at once.
func missingParamsErr(positionals []string, flagsMissing []paramSpec) error {
	var parts []string
	var hint string
	for _, p := range positionals {
		parts = append(parts, "<"+p+">")
	}
	for _, f := range flagsMissing {
		parts = append(parts, "--"+f.name)
	}
	if len(positionals) > 0 {
		hint = "<" + positionals[0] + ">"
	} else if len(flagsMissing) > 0 {
		hint = "--" + flagsMissing[0].name + " <value>"
	}
	return usageErr(fmt.Errorf("missing required parameter(s): %s\n  pass them on the command line (e.g. %s) or run interactively without --no-input", strings.Join(parts, ", "), hint))
}

// missingFlagsErr is the reactive-path equivalent of missingParamsErr,
// invoked when the leaf command surfaces a single missing-flag error
// during dispatch (a manually-checked required flag, typically).
func missingFlagsErr(missing []paramSpec) error {
	return missingParamsErr(nil, missing)
}

// missingFlagRE matches both shapes that surface from the leaf:
//   - cobra MarkFlagRequired:  `required flag(s) "name" not set`
//   - promoted manual check:   `required flag "name" not set`
//
// Captures the first flag name only; if multiple flags are missing,
// successive retry passes catch the rest one at a time.
var missingFlagRE = regexp.MustCompile(`required flag(?:\(s\))? "([^"]+)"`)

func parseMissingRequiredFlag(err error) string {
	if err == nil {
		return ""
	}
	m := missingFlagRE.FindStringSubmatch(err.Error())
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// --- `exec list` ---

func newAPIExecListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "list",
		Short:       "List endpoints available to 'exec'",
		Annotations: map[string]string{"mcp:read-only": "true"},
		Long: `List the curated set of endpoints callable via 'exec'.

The exec allowlist is the active-APIs set minus interfaces whose responses
are binary (image-processing, web-screenshot). Those still work through
their dedicated commands — see the footer for the redirect path.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			endpoints := collectEndpoints(cmd.Root())

			type entry struct {
				Interface string `json:"interface"`
				Method    string `json:"method"`
				Short     string `json:"short"`
				Target    string `json:"target"`
			}
			var supported []entry
			for _, ep := range endpoints {
				if !IsActiveInterface(ep.iface) {
					continue
				}
				if blocked, _ := isExecBlocked(ep.iface); blocked {
					continue
				}
				supported = append(supported, entry{
					Interface: ep.iface,
					Method:    ep.method,
					Short:     ep.short,
					Target:    ep.iface + "." + ep.method,
				})
			}
			sort.Slice(supported, func(i, j int) bool {
				if supported[i].Interface != supported[j].Interface {
					return supported[i].Interface < supported[j].Interface
				}
				return supported[i].Method < supported[j].Method
			})

			type excluded struct {
				Interface string `json:"interface"`
				Reason    string `json:"reason"`
			}
			var excludedList []excluded
			for iface, reason := range execBlocklist {
				if !IsActiveInterface(iface) {
					continue
				}
				excludedList = append(excludedList, excluded{Interface: iface, Reason: reason})
			}
			sort.Slice(excludedList, func(i, j int) bool {
				return excludedList[i].Interface < excludedList[j].Interface
			})

			if flags.asJSON {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"endpoints": supported,
					"excluded":  excludedList,
				}, flags)
			}

			w := cmd.OutOrStdout()
			ifaceSet := map[string]bool{}
			for _, e := range supported {
				ifaceSet[e.Interface] = true
			}
			fmt.Fprintf(w, "Endpoints available to 'exec' (%d endpoints across %d interfaces):\n\n",
				len(supported), len(ifaceSet))
			maxTarget := 0
			for _, e := range supported {
				if len(e.Target) > maxTarget {
					maxTarget = len(e.Target)
				}
			}
			for _, e := range supported {
				fmt.Fprintf(w, "  %-*s  %s\n", maxTarget, e.Target, e.Short)
			}
			if len(excludedList) > 0 {
				fmt.Fprintln(w, "\nExcluded from exec (binary responses):")
				for _, e := range excludedList {
					fmt.Fprintf(w, "  %s — %s\n", e.Interface, e.Reason)
				}
			}
			fmt.Fprintln(w, "\nRun:\n  datpaq exec <interface>.<method> [--flag value]")
			return nil
		},
	}
}
