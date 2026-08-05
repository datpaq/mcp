// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hand-authored: copy-pasteable code samples for any Datpaq endpoint,
// derived from the live Cobra command tree (annotations + flags) so the
// output can't drift from the commands the CLI actually exposes.

package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Each generated endpoint command carries these annotations (see
// `printing-press generate` output). The tree walk groups by interface
// (the part before the dot in pp:endpoint) and surfaces samples per leaf.
const (
	annEndpoint = "pp:endpoint" // "<interface>.<method>"
	annMethod   = "pp:method"   // GET / POST / PUT / PATCH / DELETE
	annPath     = "pp:path"     // "/aircraft/lookup"
)

// Languages the sample command can emit. Order matters: it's the order
// shown in --help and in the "supported: …" error message.
var sampleLangs = []string{"curl", "js", "py", "go"}

func newSampleCmd(flags *rootFlags) *cobra.Command {
	var lang string
	cmd := &cobra.Command{
		Use:   "sample [interface] [method]",
		Short: "Print copy-pasteable code samples for any endpoint",
		Long: "Emits a ready-to-run snippet for a Datpaq API endpoint, " +
			"using $DATPAQ_API_KEY as the credential placeholder.\n\n" +
			"  No args              → list all interfaces (same as 'datpaq api')\n" +
			"  One arg (interface)  → list its methods, or render directly if there's only one\n" +
			"  Two args             → render the snippet\n\n" +
			"Companion to 'datpaq api' (which browses the same endpoints).",
		Example: "  datpaq sample\n" +
			"  datpaq sample aircraft\n" +
			"  datpaq sample aircraft lookup-by-tail\n" +
			"  datpaq sample aircraft lookup-by-tail --lang py\n" +
			"  datpaq sample ip-geolocation get --lang go",
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !contains(sampleLangs, lang) {
				return usageErr(fmt.Errorf("unsupported --lang %q (supported: %s)",
					lang, strings.Join(sampleLangs, ", ")))
			}

			// Walk from the root, not from `cmd` — `cmd` is `sample` itself
			// and only its own children would be visible from there.
			endpoints := collectEndpoints(cmd.Root())

			// No-args path: list every interface so the user can pick one. The
			// splash advertises 'datpaq sample' as a top-level entry point, so
			// running it cold needs to lead somewhere, not just print a cobra
			// usage error.
			if len(args) == 0 {
				listInterfaces(cmd.OutOrStdout(), endpoints)
				return nil
			}

			iface := args[0]
			matches := endpointsForInterface(endpoints, iface)
			if len(matches) == 0 {
				return notFoundErr(fmt.Errorf("unknown interface %q; run 'datpaq api' to list interfaces", iface))
			}

			// Single-arg invocation: if there's exactly one method, render it
			// (one-shot path the user asked for). Otherwise list methods so the
			// next command is obvious.
			if len(args) == 1 {
				if len(matches) == 1 {
					renderSample(cmd.OutOrStdout(), matches[0], lang)
					return nil
				}
				listMethods(cmd.OutOrStdout(), iface, matches)
				return nil
			}

			method := args[1]
			for _, ep := range matches {
				if ep.method == method {
					renderSample(cmd.OutOrStdout(), ep, lang)
					return nil
				}
			}
			return notFoundErr(fmt.Errorf("unknown method %q on interface %q; try one of: %s",
				method, iface, joinMethodNames(matches)))
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "curl",
		fmt.Sprintf("Output language (%s)", strings.Join(sampleLangs, ", ")))
	return cmd
}

// endpointInfo is what we extract from each endpoint command for rendering.
// Kept minimal — anything renderSample needs goes here; everything else stays
// in the cobra.Command we hold a reference to.
type endpointInfo struct {
	iface       string // "aircraft"
	method      string // "lookup-by-tail" (the cobra Use name)
	httpMethod  string // "GET"
	path        string // "/aircraft/lookup"
	short       string // one-line description
	queryParams []paramSpec
	cmd         *cobra.Command
}

type paramSpec struct {
	name        string // flag name as users type it: --tail
	wireName    string // REST query/body key used by generated command code
	typeName    string // "string", "int", "bool", "duration", ...
	description string
	required    bool
}

// collectEndpoints walks the cobra tree and returns every command annotated
// with pp:endpoint. Recurses through hidden parents because printed CLIs mark
// raw resource parents Hidden to keep top-level help curated, but their
// endpoint leaves remain runnable (and runnable = sampleable).
func collectEndpoints(root *cobra.Command) []endpointInfo {
	var out []endpointInfo
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, child := range c.Commands() {
			if ep := child.Annotations[annEndpoint]; ep != "" {
				iface, method, ok := strings.Cut(ep, ".")
				if ok {
					out = append(out, endpointInfo{
						iface:       iface,
						method:      method,
						httpMethod:  child.Annotations[annMethod],
						path:        child.Annotations[annPath],
						short:       child.Short,
						queryParams: extractParams(child, ep),
						cmd:         child,
					})
				}
			}
			walk(child)
		}
	}
	walk(root)
	return out
}

// extractParams pulls user-facing flags off a generated endpoint command and
// classifies them as required vs optional. Skips persistent flags (those are
// global config like --json) and the canonical -h/--help.
func extractParams(c *cobra.Command, endpoint string) []paramSpec {
	var out []paramSpec
	// Cobra stores required flag names in an annotation on each flag itself,
	// so we walk the local flag set and read that annotation when present.
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" || f.Hidden {
			return
		}
		required := false
		if vals := f.Annotations[cobra.BashCompOneRequiredFlag]; len(vals) > 0 && vals[0] == "true" {
			required = true
		}
		out = append(out, paramSpec{
			name:        f.Name,
			wireName:    sampleWireParamName(endpoint, f.Name),
			typeName:    f.Value.Type(),
			description: f.Usage,
			required:    required,
		})
	})
	return out
}

func endpointsForInterface(all []endpointInfo, iface string) []endpointInfo {
	var out []endpointInfo
	for _, ep := range all {
		if ep.iface == iface {
			out = append(out, ep)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].method < out[j].method })
	return out
}

// listInterfaces prints every ACTIVE interface that has at least one
// sampleable endpoint, deduped and sorted. Format mirrors `datpaq api`
// so the two commands feel like siblings.
//
// Filtering matches the api command: only active interfaces appear in
// the listing, but `datpaq sample <inactive-iface> <method>` still works
// — activation gates discovery, never capability.
func listInterfaces(w io.Writer, all []endpointInfo) {
	seenAll := map[string]int{}
	for _, ep := range all {
		seenAll[ep.iface]++
	}
	visible := make([]string, 0, len(seenAll))
	for name := range seenAll {
		if IsActiveInterface(name) {
			visible = append(visible, name)
		}
	}
	sort.Strings(visible)

	if activeAPICount() > 0 {
		fmt.Fprintf(w, "Available interfaces (%d active):\n\n", len(visible))
	} else {
		fmt.Fprintf(w, "Available interfaces (%d):\n\n", len(visible))
	}
	maxName := 0
	for _, n := range visible {
		if len(n) > maxName {
			maxName = len(n)
		}
	}
	for _, n := range visible {
		fmt.Fprintf(w, "  %-*s  %d method", maxName, n, seenAll[n])
		if seenAll[n] != 1 {
			fmt.Fprint(w, "s")
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "\nNext:")
	fmt.Fprintln(w, "  datpaq sample <interface>           # list methods on that interface")
	fmt.Fprintln(w, "  datpaq sample <interface> <method>  # render a snippet")
}

func listMethods(w io.Writer, iface string, eps []endpointInfo) {
	fmt.Fprintf(w, "Methods on '%s' (%d):\n\n", iface, len(eps))
	maxName := 0
	for _, ep := range eps {
		if len(ep.method) > maxName {
			maxName = len(ep.method)
		}
	}
	for _, ep := range eps {
		fmt.Fprintf(w, "  %-*s  %s\n", maxName, ep.method, ep.short)
	}
	fmt.Fprintf(w, "\nRender a sample:\n  datpaq sample %s <method>\n", iface)
}

func renderSample(w io.Writer, ep endpointInfo, lang string) {
	url := "https://datpaq.com/api/v1" + ep.path
	switch lang {
	case "curl":
		renderCurl(w, ep, url)
	case "js":
		renderJS(w, ep, url)
	case "py":
		renderPython(w, ep, url)
	case "go":
		renderGo(w, ep, url)
	}
}

// renderCurl emits a single-line-friendly curl. Long URLs get line-continued
// so the snippet stays readable when pasted into a wide terminal. GET/DELETE
// put params in the query string; POST/PUT/PATCH put them in a JSON body.
func renderCurl(w io.Writer, ep endpointInfo, url string) {
	fmt.Fprintln(w, "# "+ep.short)
	if hasBody(ep.httpMethod) {
		fmt.Fprintf(w, "curl -X %s '%s' \\\n", ep.httpMethod, url)
		fmt.Fprintln(w, `  -H "x-api-key: $DATPAQ_API_KEY" \`)
		fmt.Fprintln(w, `  -H "Content-Type: application/json" \`)
		fmt.Fprintln(w, "  -d '"+sampleJSONBody(ep)+"'")
		return
	}
	q := sampleQueryString(ep)
	full := url
	if q != "" {
		full = url + "?" + q
	}
	fmt.Fprintf(w, "curl '%s' \\\n", full)
	fmt.Fprintln(w, `  -H "x-api-key: $DATPAQ_API_KEY"`)
}

func renderJS(w io.Writer, ep endpointInfo, url string) {
	fmt.Fprintln(w, "// "+ep.short)
	if hasBody(ep.httpMethod) {
		fmt.Fprintf(w, "const response = await fetch(%q, {\n", url)
		fmt.Fprintf(w, "  method: %q,\n", ep.httpMethod)
		fmt.Fprintln(w, "  headers: {")
		fmt.Fprintln(w, `    "x-api-key": process.env.DATPAQ_API_KEY,`)
		fmt.Fprintln(w, `    "Content-Type": "application/json",`)
		fmt.Fprintln(w, "  },")
		fmt.Fprintln(w, "  body: JSON.stringify({")
		for _, p := range ep.queryParams {
			fmt.Fprintf(w, "    %s: %s,\n", p.wireName, jsLiteral(p))
		}
		fmt.Fprintln(w, "  }),")
		fmt.Fprintln(w, "});")
		fmt.Fprintln(w, "const data = await response.json();")
		return
	}
	if len(ep.queryParams) == 0 {
		fmt.Fprintf(w, "const response = await fetch(%q, {\n", url)
	} else {
		fmt.Fprintf(w, "const response = await fetch(%q + \"?\" + new URLSearchParams({\n", url)
		for _, p := range ep.queryParams {
			fmt.Fprintf(w, "  %s: %s,\n", p.wireName, jsLiteral(p))
		}
		fmt.Fprintln(w, "}), {")
	}
	fmt.Fprintln(w, "  headers: {")
	fmt.Fprintln(w, `    "x-api-key": process.env.DATPAQ_API_KEY,`)
	fmt.Fprintln(w, "  },")
	fmt.Fprintln(w, "});")
	fmt.Fprintln(w, "const data = await response.json();")
}

func renderPython(w io.Writer, ep endpointInfo, url string) {
	fmt.Fprintln(w, "# "+ep.short)
	fmt.Fprintln(w, "import os, requests")
	fmt.Fprintln(w)
	method := strings.ToLower(ep.httpMethod)
	if hasBody(ep.httpMethod) {
		fmt.Fprintf(w, "response = requests.%s(\n", method)
		fmt.Fprintf(w, "    %q,\n", url)
		fmt.Fprintln(w, `    headers={"x-api-key": os.environ["DATPAQ_API_KEY"]},`)
		fmt.Fprintln(w, "    json={")
		for _, p := range ep.queryParams {
			fmt.Fprintf(w, "        %q: %s,\n", p.wireName, pyLiteral(p))
		}
		fmt.Fprintln(w, "    },")
		fmt.Fprintln(w, ")")
		fmt.Fprintln(w, "data = response.json()")
		return
	}
	fmt.Fprintf(w, "response = requests.%s(\n", method)
	fmt.Fprintf(w, "    %q,\n", url)
	fmt.Fprintln(w, `    headers={"x-api-key": os.environ["DATPAQ_API_KEY"]},`)
	if len(ep.queryParams) > 0 {
		fmt.Fprintln(w, "    params={")
		for _, p := range ep.queryParams {
			fmt.Fprintf(w, "        %q: %s,\n", p.wireName, pyLiteral(p))
		}
		fmt.Fprintln(w, "    },")
	}
	fmt.Fprintln(w, ")")
	fmt.Fprintln(w, "data = response.json()")
}

func renderGo(w io.Writer, ep endpointInfo, url string) {
	fmt.Fprintln(w, "// "+ep.short)
	if hasBody(ep.httpMethod) {
		fmt.Fprintln(w, "body := map[string]any{")
		for _, p := range ep.queryParams {
			fmt.Fprintf(w, "    %q: %s,\n", p.wireName, goLiteral(p))
		}
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w, "raw, _ := json.Marshal(body)")
		fmt.Fprintf(w, "req, _ := http.NewRequest(%q, %q, bytes.NewReader(raw))\n", ep.httpMethod, url)
		fmt.Fprintln(w, `req.Header.Set("x-api-key", os.Getenv("DATPAQ_API_KEY"))`)
		fmt.Fprintln(w, `req.Header.Set("Content-Type", "application/json")`)
		fmt.Fprintln(w, "resp, _ := http.DefaultClient.Do(req)")
		fmt.Fprintln(w, "defer resp.Body.Close()")
		return
	}
	fmt.Fprintf(w, "req, _ := http.NewRequest(%q, %q, nil)\n", ep.httpMethod, url)
	fmt.Fprintln(w, `req.Header.Set("x-api-key", os.Getenv("DATPAQ_API_KEY"))`)
	if len(ep.queryParams) > 0 {
		fmt.Fprintln(w, "q := req.URL.Query()")
		for _, p := range ep.queryParams {
			fmt.Fprintf(w, "q.Set(%q, %s)\n", p.wireName, goLiteral(p))
		}
		fmt.Fprintln(w, "req.URL.RawQuery = q.Encode()")
	}
	fmt.Fprintln(w, "resp, _ := http.DefaultClient.Do(req)")
	fmt.Fprintln(w, "defer resp.Body.Close()")
}

// hasBody is the REST convention: write verbs carry a body, read/delete
// verbs put their inputs in the query string. Tracks what the generated
// endpoint commands actually do (see e.g. aircraft_lookup-by-tail.go).
func hasBody(httpMethod string) bool {
	switch strings.ToUpper(httpMethod) {
	case "POST", "PUT", "PATCH":
		return true
	}
	return false
}

func sampleQueryString(ep endpointInfo) string {
	var parts []string
	for _, p := range ep.queryParams {
		parts = append(parts, p.wireName+"=<"+p.typeName+">")
	}
	return strings.Join(parts, "&")
}

func sampleJSONBody(ep endpointInfo) string {
	var parts []string
	for _, p := range ep.queryParams {
		parts = append(parts, fmt.Sprintf("%q:%s", p.wireName, jsonLiteral(p)))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func sampleWireParamName(endpoint, flagName string) string {
	key := endpoint + "." + flagName
	switch key {
	case "convert-time.convert_time.source-time":
		return "sourceTime"
	case "convert-time.convert_time.source-zone":
		return "sourceZone"
	case "convert-time.convert_time.target-zone":
		return "targetZone"
	case "mx-lookup.get.resolve-ips",
		"mx-lookup.post.resolve-ips":
		return "resolve_ips"
	case "mx-lookup.get.test-smtp",
		"mx-lookup.post.test-smtp":
		return "test_smtp"
	case "mx-lookup.get.include-spf",
		"mx-lookup.post.include-spf":
		return "include_spf"
	case "phone-validation.validate.phone-number",
		"phone-validation.format.phone-number":
		return "phoneNumber"
	case "phone-validation.validate.country-code",
		"phone-validation.format.country-code":
		return "countryCode"
	case "phone-validation.validate-batch.phone-numbers":
		return "phoneNumbers"
	case "phone-validation.validate-batch.default-country-code":
		return "defaultCountryCode"
	case "web-scraping.scrape.wait-until",
		"pdf-generation.from-url.wait-until":
		return "waitUntil"
	case "weather.forecast.forecast-days":
		return "forecastDays"
	case "calendar.month.week-start",
		"calendar.day.week-start",
		"calendar.range.week-start":
		return "week_start"
	case "calendar.month.country-code",
		"calendar.day.country-code",
		"calendar.range.country-code",
		"calendar.holidays.country-code":
		return "country_code"
	case "calendar.month.include-holidays",
		"calendar.day.include-holidays",
		"calendar.range.include-holidays":
		return "include_holidays"
	case "calendar.month.holiday-sources",
		"calendar.day.holiday-sources",
		"calendar.range.holiday-sources",
		"calendar.holidays.holiday-sources":
		return "holiday_sources"
	case "calendar.month.custom-holidays",
		"calendar.day.custom-holidays",
		"calendar.range.custom-holidays",
		"calendar.holidays.custom-holidays":
		return "custom_holidays"
	case "calendar.month.fiscal-year-start-month",
		"calendar.day.fiscal-year-start-month",
		"calendar.range.fiscal-year-start-month":
		return "fiscal_year_start_month"
	case "calendar.month.output-format",
		"calendar.range.output-format":
		return "output_format"
	case "calendar.range.start-date":
		return "start_date"
	case "calendar.range.end-date":
		return "end_date"
	case "calendar.range.include-weekends":
		return "include_weekends"
	case "geocoding.forward.country-codes":
		return "countryCodes"
	case "qr-code.generate.error-correction-level":
		return "errorCorrectionLevel"
	case "qr-code.generate.dark-color":
		return "darkColor"
	case "qr-code.generate.light-color":
		return "lightColor"
	case "qr-code.generate.response-type",
		"pdf-generation.from-url.response-type",
		"pdf-generation.from-html.response-type":
		return "responseType"
	case "qr-code.decode.image-base64":
		return "imageBase64"
	case "qr-code.decode.max-codes":
		return "maxCodes"
	default:
		return flagName
	}
}

// Per-language literal helpers. They all answer the same question — "what
// placeholder value goes here?" — but each language has its own conventions
// for quoting strings vs. unquoting numbers vs. booleans.

func jsLiteral(p paramSpec) string {
	switch p.typeName {
	case "string", "duration":
		return fmt.Sprintf(`"<%s>"`, p.typeName)
	case "bool":
		return "true"
	case "int", "int32", "int64", "float64":
		return "0"
	}
	return fmt.Sprintf(`"<%s>"`, p.typeName)
}

func pyLiteral(p paramSpec) string {
	switch p.typeName {
	case "string", "duration":
		return fmt.Sprintf(`"<%s>"`, p.typeName)
	case "bool":
		return "True"
	case "int", "int32", "int64", "float64":
		return "0"
	}
	return fmt.Sprintf(`"<%s>"`, p.typeName)
}

func goLiteral(p paramSpec) string {
	switch p.typeName {
	case "string", "duration":
		return fmt.Sprintf(`"<%s>"`, p.typeName)
	case "bool":
		return "true"
	case "int", "int32", "int64", "float64":
		return "0"
	}
	return fmt.Sprintf(`"<%s>"`, p.typeName)
}

func jsonLiteral(p paramSpec) string {
	switch p.typeName {
	case "string", "duration":
		return fmt.Sprintf(`"<%s>"`, p.typeName)
	case "bool":
		return "true"
	case "int", "int32", "int64", "float64":
		return "0"
	}
	return fmt.Sprintf(`"<%s>"`, p.typeName)
}

func joinMethodNames(eps []endpointInfo) string {
	names := make([]string, 0, len(eps))
	for _, ep := range eps {
		names = append(names, ep.method)
	}
	return strings.Join(names, ", ")
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
