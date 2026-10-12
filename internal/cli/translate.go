// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "translate",
		Short: "Text translation between supported languages",
		Endpoints: []APIEndpoint{
			{Name: "text", Short: "Translate one text or a batch of texts", Method: "POST", Path: "/translate",
				Params: []APIParam{
					{Flag: "target-lang", Wire: "targetLang", In: "body", Usage: "Target language code", Example: "es", Required: true},
					{Flag: "text", In: "body", Usage: "Text to translate (or pass --texts)", Example: "Hello, world"},
					{Flag: "texts", In: "body", Type: "array", Usage: "JSON array or comma-separated texts to translate", Example: `["Hello","Goodbye"]`},
					{Flag: "source-lang", Wire: "sourceLang", In: "body", Usage: "Source language code (default auto)", Example: "en"},
				}},
			{Name: "languages", Short: "List supported languages", Method: "GET", Path: "/translate/languages"},
		},
	})
}
