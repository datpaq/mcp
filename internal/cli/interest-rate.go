// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "interest-rate",
		Short: "Current benchmark interest rates and the Treasury yield curve",
		Endpoints: []APIEndpoint{
			{Name: "current", Short: "Get current benchmark rates by category", Method: "GET", Path: "/interest-rate/current"},
			{Name: "yield-curve", Short: "Get the ordered Treasury yield curve", Method: "GET", Path: "/interest-rate/yield-curve"},
		},
	})
}
