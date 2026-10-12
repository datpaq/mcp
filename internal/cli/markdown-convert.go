// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.
// Common render options are flags; pass --stdin for the full option set.

package cli

func init() {
	md := APIParam{Flag: "markdown", In: "body", Usage: "Markdown input (or pass --html)", Example: "# Title"}
	html := APIParam{Flag: "html", In: "body", Usage: "HTML input (or pass --markdown)", Example: "<h1>Title</h1>"}

	registerDeclaredAPI(APIGroup{
		Name:  "markdown-convert",
		Short: "Markdown to HTML, HTML to Markdown, plaintext, and document outlines",
		Endpoints: []APIEndpoint{
			{Name: "capabilities", Short: "List supported targets, limits, and Markdown flavor", Method: "GET", Path: "/markdown-convert/capabilities"},
			{Name: "to-html", Short: "Render Markdown to sanitized HTML", Method: "POST", Path: "/markdown-convert/to-html",
				Params: []APIParam{
					{Flag: "markdown", In: "body", Usage: "Markdown input", Example: "# Title", Required: true},
					{Flag: "allow-raw-html", Wire: "allowRawHtml", In: "body", Type: "bool", Usage: "Allow raw HTML (default false)", Example: "false"},
					{Flag: "breaks", In: "body", Type: "bool", Usage: "Convert newlines to <br> (default false)", Example: "true"},
					{Flag: "linkify", In: "body", Type: "bool", Usage: "Autolink URLs (default true)", Example: "true"},
				}},
			{Name: "to-markdown", Short: "Convert HTML to Markdown", Method: "POST", Path: "/markdown-convert/to-markdown",
				Params: []APIParam{
					{Flag: "html", In: "body", Usage: "HTML input", Example: "<h1>Title</h1>", Required: true},
					{Flag: "heading-style", Wire: "headingStyle", In: "body", Usage: "atx (default) or setext", Example: "atx"},
					{Flag: "bullet-marker", Wire: "bulletMarker", In: "body", Usage: "Bullet marker: -, *, or +", Example: "-"},
					{Flag: "code-block-style", Wire: "codeBlockStyle", In: "body", Usage: "fenced (default) or indented", Example: "fenced"},
				}},
			{Name: "to-text", Short: "Render Markdown or HTML to plaintext", Method: "POST", Path: "/markdown-convert/to-text",
				Params: []APIParam{md, html,
					{Flag: "wrap", In: "body", Type: "int", Usage: "Wrap column (0-200, default 0 = no wrap)", Example: "80"},
					{Flag: "include-link-urls", Wire: "includeLinkUrls", In: "body", Type: "bool", Usage: "Include link URLs (default false)", Example: "true"},
				}},
			{Name: "outline", Short: "Get document structure and statistics", Method: "POST", Path: "/markdown-convert/outline",
				Params: []APIParam{md, html,
					{Flag: "toc-min-level", Wire: "tocMinLevel", In: "body", Type: "int", Usage: "Minimum heading level (default 1)", Example: "1"},
					{Flag: "toc-max-level", Wire: "tocMaxLevel", In: "body", Type: "int", Usage: "Maximum heading level (default 6)", Example: "3"},
				}},
			{Name: "batch", Short: "Process up to 25 documents in one request", Method: "POST", Path: "/markdown-convert/batch",
				Params: []APIParam{{Flag: "documents", In: "body", Type: "array", Usage: `JSON array of {"target","markdown"|"html"} documents`, Example: `[{"target":"html","markdown":"# Hi"}]`, Required: true}}},
		},
	})
}
