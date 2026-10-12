// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	bodyOptions := func(extra ...APIParam) []APIParam {
		return append([]APIParam{
			{Flag: "text", In: "body", Usage: "Text to process (up to 200,000 characters)", Example: "We delve into a rich tapestry of synergy.", Required: true},
			{Flag: "format", In: "body", Usage: "text (default) or markdown", Example: "text"},
			{Flag: "profile", In: "body", Usage: "Rule profile name", Example: "default"},
			{Flag: "categories", In: "body", Type: "array", Usage: "Rule categories to apply (JSON array or comma-separated)", Example: "filler"},
			{Flag: "ignore", In: "body", Type: "array", Usage: "Rule IDs to ignore (JSON array or comma-separated)", Example: "em-dash"},
			{Flag: "preserve", In: "body", Type: "array", Usage: "Content to preserve (JSON array or comma-separated)", Example: "code"},
		}, extra...)
	}

	registerDeclaredAPI(APIGroup{
		Name:  "deslop",
		Short: "Detect and rewrite AI writing patterns in text",
		Endpoints: []APIEndpoint{
			{Name: "rules", Short: "List detection rules", Method: "GET", Path: "/deslop/rules",
				Params: []APIParam{
					{Flag: "layer", Type: "int", Usage: "Rule layer (1-5)", Example: "1"},
					{Flag: "category", Usage: "Rule category", Example: "filler"},
				}},
			{Name: "analyze", Short: "Analyze text for AI writing patterns", Method: "POST", Path: "/deslop/analyze",
				Params: bodyOptions()},
			{Name: "analyze-get", Short: "Analyze short text via GET", Method: "GET", Path: "/deslop/analyze",
				Params: []APIParam{
					{Flag: "text", Usage: "Text to analyze (short; use analyze for long text)", Example: "We delve into synergy.", Required: true},
					{Flag: "format", Usage: "text (default) or markdown", Example: "text"},
					{Flag: "profile", Usage: "Rule profile name", Example: "default"},
					{Flag: "categories", Usage: "Comma-separated rule categories", Example: "filler"},
					{Flag: "ignore", Usage: "Comma-separated rule IDs to ignore", Example: "em-dash"},
					{Flag: "preserve", Usage: "Comma-separated content to preserve", Example: "code"},
				}},
			{Name: "rewrite", Short: "Rewrite text to remove AI writing patterns", Method: "POST", Path: "/deslop/rewrite",
				Params: bodyOptions(APIParam{Flag: "mode", In: "body", Usage: "light, balanced (default), or strict", Example: "balanced"})},
		},
	})
}
