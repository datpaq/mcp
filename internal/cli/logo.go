// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "logo",
		Short: "Discover a company's logos, icons, and brand palette by domain",
		Endpoints: []APIEndpoint{
			{Name: "discover", Short: "Discover logos and brand colors for a domain", Method: "GET", Path: "/logo/discover",
				Params: []APIParam{
					{Flag: "domain", Usage: "Company domain", Example: "stripe.com", Required: true},
					{Flag: "include-palette", Wire: "includePalette", Type: "bool", Usage: "Include brand color palette (default true)", Example: "false"},
				}},
		},
	})
}
