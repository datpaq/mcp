// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "tech-stack-detection",
		Short: "Detect the technologies a website uses",
		Endpoints: []APIEndpoint{
			{Name: "detect", Short: "Detect technologies for a domain", Method: "GET", Path: "/tech-stack-detection/detect",
				Params: []APIParam{
					{Flag: "domain", Usage: "Domain to analyze", Example: "example.com", Required: true},
					{Flag: "url", Usage: "Specific page URL to analyze", Example: "https://example.com/pricing"},
				}},
		},
	})
}
