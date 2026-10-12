// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	url := APIParam{Flag: "url", In: "body", Usage: "Public HTTP(S) page URL", Example: "https://example.com", Required: true}
	render := APIParam{Flag: "render", In: "body", Usage: "auto (default), static, or browser", Example: "static"}
	maxChars := APIParam{Flag: "max-chars", Wire: "max_chars", In: "body", Type: "int", Usage: "Max output characters (1000-100000, default 20000)", Example: "20000"}

	registerDeclaredAPI(APIGroup{
		Name:  "diffy",
		Short: "Web page snapshots and change diffs against a previous snapshot",
		Endpoints: []APIEndpoint{
			{Name: "snapshot", Short: "Capture a page snapshot and its diffable state", Method: "POST", Path: "/diffy/snapshot",
				Params: []APIParam{url, render, maxChars,
					{Flag: "format", In: "body", Usage: "markdown (default), blocks, or tree", Example: "markdown"},
					{Flag: "include-state", Wire: "include_state", In: "body", Type: "bool", Usage: "Include state for later diffs (default true)", Example: "true"},
				}},
			{Name: "diff", Short: "Diff a page against a previous snapshot state", Method: "POST", Path: "/diffy/diff",
				Params: []APIParam{url, render, maxChars,
					{Flag: "previous-state", Wire: "previous_state", In: "body", Type: "object", Usage: "State JSON returned by snapshot", Example: `{"schema":1,"blocks":[]}`, Required: true},
				}},
		},
	})
}
