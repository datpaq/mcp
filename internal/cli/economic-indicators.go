// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	id := APIParam{Flag: "id", In: "path", Usage: "Catalog indicator ID", Example: "us_gdp_growth"}

	registerDeclaredAPI(APIGroup{
		Name:  "economic-indicators",
		Short: "Macroeconomic indicator catalog, series history, and interest rates",
		Endpoints: []APIEndpoint{
			{Name: "indicators", Short: "List available indicators", Method: "GET", Path: "/economic-indicators/indicators"},
			{Name: "series", Short: "Get history for an indicator", Method: "GET", Path: "/economic-indicators/series/{id}",
				Params: []APIParam{id, {Flag: "limit", Type: "int", Usage: "Observations (1-60, default 12)", Example: "24"}}},
			{Name: "latest", Short: "Get the latest value for an indicator", Method: "GET", Path: "/economic-indicators/series/{id}/latest",
				Params: []APIParam{id}},
			{Name: "interest-rates", Short: "Get key interest rates", Method: "GET", Path: "/economic-indicators/interest-rates"},
		},
	})
}
