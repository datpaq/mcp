// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.
// GET /lorem-ipsum is an alias of /lorem-ipsum/generate and is not exposed separately.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "lorem-ipsum",
		Short: "Placeholder text generation as JSON, text, or HTML",
		Endpoints: []APIEndpoint{
			{Name: "generate", Short: "Generate lorem ipsum words, sentences, or paragraphs", Method: "GET", Path: "/lorem-ipsum/generate",
				Params: []APIParam{
					{Flag: "units", Usage: "words, sentences, or paragraphs (default)", Example: "sentences"},
					{Flag: "amount", Type: "int", Usage: "Number of units (default 3)", Example: "5"},
					{Flag: "length", Usage: "short, medium (default), long, or verylong", Example: "short"},
					{Flag: "format", Usage: "json (default), text, or html", Example: "json"},
					{Flag: "start-with-lorem", Wire: "startWithLorem", Type: "bool", Usage: "Start with 'Lorem ipsum' (default true)", Example: "false"},
					{Flag: "seed", Usage: "Seed for deterministic output", Example: "demo"},
					{Flag: "max-length", Wire: "maxLength", Type: "int", Usage: "Maximum output characters", Example: "500"},
				}},
		},
	})
}
