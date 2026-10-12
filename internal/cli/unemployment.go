// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.
// The /metro and /metro/search routes are omitted: ProApi returns 503 for them
// until the BLS LAUS area table is provisioned.

package cli

func init() {
	period := func(values string) APIParam {
		return APIParam{Flag: "period", Usage: "Window: " + values + " (default latest)", Example: "1y"}
	}

	registerDeclaredAPI(APIGroup{
		Name:  "unemployment",
		Short: "US unemployment rates, labor force, claims, demographics, and states",
		Endpoints: []APIEndpoint{
			{Name: "national", Short: "Get the national unemployment rate", Method: "GET", Path: "/unemployment",
				Params: []APIParam{period("latest, 1y, 5y, or 10y"),
					{Flag: "adjusted", Type: "bool", Usage: "Seasonally adjusted series (default true)", Example: "false"}}},
			{Name: "measures", Short: "Get U-1 through U-6 measures", Method: "GET", Path: "/unemployment/measures",
				Params: []APIParam{period("latest, 1y, or 5y"),
					{Flag: "measure", Usage: "Measure: u1-u6 or all (default)", Example: "u6"}}},
			{Name: "labor-force", Short: "Get participation and employment-population ratios", Method: "GET", Path: "/unemployment/labor-force",
				Params: []APIParam{period("latest, 1y, or 5y"),
					{Flag: "metric", Usage: "Metric: participation, employment-ratio, or both (default)", Example: "participation"}}},
			{Name: "claims", Short: "Get initial and continued jobless claims", Method: "GET", Path: "/unemployment/claims",
				Params: []APIParam{period("latest, 1y, or 2y"),
					{Flag: "type", Usage: "Claims type: initial, continued, or both (default)", Example: "initial"}}},
			{Name: "long-term", Short: "Get long-term unemployment", Method: "GET", Path: "/unemployment/long-term",
				Params: []APIParam{period("latest, 1y, or 5y")}},
			{Name: "demographic", Short: "Get unemployment by demographic group", Method: "GET", Path: "/unemployment/demographic",
				Params: []APIParam{period("latest or 1y"),
					{Flag: "group", Usage: "Group: age, gender, race, or education", Example: "race", Required: true}}},
			{Name: "state", Short: "Get unemployment for one state or DC", Method: "GET", Path: "/unemployment/state",
				Params: []APIParam{period("latest or 1y"),
					{Flag: "state", Usage: "Two-letter US state or DC code", Example: "CA", Required: true}}},
			{Name: "states", Short: "Get latest unemployment for all states and DC", Method: "GET", Path: "/unemployment/states"},
			{Name: "series", Short: "List the unemployment series catalog", Method: "GET", Path: "/unemployment/series"},
		},
	})
}
