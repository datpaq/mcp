// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "structured-extraction",
		Short: "Extract structured fields from a web page or HTML using CSS selectors",
		Endpoints: []APIEndpoint{
			{Name: "extract", Short: "Extract fields from a URL or HTML", Method: "POST", Path: "/structured-extraction/extract",
				Params: []APIParam{
					{Flag: "selectors", In: "body", Type: "object", Usage: `Selector map JSON, e.g. {"title":"h1"}`, Example: `{"title":"h1"}`, Required: true},
					{Flag: "url", In: "body", Usage: "Public HTTP(S) page URL (or pass --html)", Example: "https://example.com"},
					{Flag: "html", In: "body", Usage: "Raw HTML (or pass --url)", Example: "<h1>Hello</h1>"},
				}},
		},
	})
}
