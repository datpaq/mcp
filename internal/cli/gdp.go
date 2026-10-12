// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	dataset := APIParam{Flag: "dataset", Usage: "Dataset: nominal (default), real, or deflator", Example: "nominal"}
	source := APIParam{Flag: "source", Usage: "Source mode: auto (default), snapshot, or live", Example: "snapshot"}
	period := APIParam{Flag: "period", Usage: "Window: latest, 1y, 5y (default), or 10y", Example: "5y"}
	startDate := APIParam{Flag: "start-date", Wire: "startDate", Usage: "Inclusive lower bound (YYYY-MM-DD)", Example: "2023-01-01"}
	endDate := APIParam{Flag: "end-date", Wire: "endDate", Usage: "Inclusive upper bound (YYYY-MM-DD)", Example: "2026-06-01"}

	registerDeclaredAPI(APIGroup{
		Name:  "gdp",
		Short: "US GDP levels, history, growth, and dataset comparison",
		Endpoints: []APIEndpoint{
			{Name: "latest", Short: "Get the latest GDP observation", Method: "GET", Path: "/gdp",
				Params: []APIParam{dataset, source}},
			{Name: "history", Short: "Get GDP history", Method: "GET", Path: "/gdp/history",
				Params: []APIParam{dataset, source, period, startDate, endDate}},
			{Name: "growth", Short: "Get GDP growth rates", Method: "GET", Path: "/gdp/growth",
				Params: []APIParam{dataset, source, startDate, endDate,
					{Flag: "points", Type: "int", Usage: "Trailing observations (2-80, default 12)", Example: "12"}}},
			{Name: "compare", Short: "Compare GDP datasets", Method: "GET", Path: "/gdp/compare",
				Params: []APIParam{source, period,
					{Flag: "datasets", Usage: "Comma-separated datasets (default all)", Example: "nominal,real"}}},
			{Name: "catalog", Short: "List datasets, sources, and periods", Method: "GET", Path: "/gdp/catalog"},
		},
	})
}
