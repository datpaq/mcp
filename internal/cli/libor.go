// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	benchmark := APIParam{Flag: "benchmark", Usage: "Benchmark: sofr (default) or libor", Example: "sofr"}
	tenor := APIParam{Flag: "tenor", Usage: "Tenor: 1m, 3m, 6m, or 12m", Example: "3m"}
	period := APIParam{Flag: "period", Usage: "Window: latest, 1y, 5y, max, or available", Example: "available"}
	source := APIParam{Flag: "source", Usage: "Data source: auto, fred, or snapshot", Example: "auto"}
	startDate := APIParam{Flag: "start-date", Wire: "startDate", Usage: "Inclusive lower bound (YYYY-MM-DD)", Example: "2026-02-01"}
	endDate := APIParam{Flag: "end-date", Wire: "endDate", Usage: "Inclusive upper bound (YYYY-MM-DD)", Example: "2026-07-01"}
	sort := APIParam{Flag: "sort", Usage: "Sort order: asc (default) or desc", Example: "desc"}

	registerDeclaredAPI(APIGroup{
		Name:  "libor",
		Short: "SOFR-first benchmark rates with archival LIBOR history and tenor spreads",
		Endpoints: []APIEndpoint{
			{Name: "latest", Short: "Get the latest benchmark rate", Method: "GET", Path: "/libor",
				Params: []APIParam{benchmark, tenor, period, source}},
			{Name: "history", Short: "Get benchmark rate history", Method: "GET", Path: "/libor/history",
				Params: []APIParam{benchmark, tenor, period, source, startDate, endDate, sort}},
			{Name: "spread", Short: "Get the spread between two tenors", Method: "GET", Path: "/libor/spread",
				Params: []APIParam{
					benchmark, period, source, startDate, endDate, sort,
					{Flag: "short-tenor", Wire: "shortTenor", Usage: "Short-leg tenor: 1m, 3m, 6m, or 12m", Example: "3m"},
					{Flag: "long-tenor", Wire: "longTenor", Usage: "Long-leg tenor: 1m, 3m, 6m, or 12m", Example: "12m"},
				}},
			{Name: "catalog", Short: "List supported benchmarks, tenors, and coverage", Method: "GET", Path: "/libor/catalog"},
		},
	})
}
