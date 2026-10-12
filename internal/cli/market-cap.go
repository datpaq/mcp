// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	symbol := APIParam{Flag: "symbol", Usage: "Ticker symbol", Example: "AAPL", Required: true}
	mode := APIParam{Flag: "mode", Usage: "Data mode: snapshot (default) or live", Example: "snapshot"}
	period := APIParam{Flag: "period", Usage: "Period: 1m, 3m, 6m, 1y, 3y, or 5y", Example: "1y"}
	startDate := APIParam{Flag: "start-date", Wire: "startDate", Usage: "Inclusive lower bound (YYYY-MM-DD)", Example: "2025-01-01"}
	endDate := APIParam{Flag: "end-date", Wire: "endDate", Usage: "Inclusive upper bound (YYYY-MM-DD)", Example: "2026-07-01"}

	registerDeclaredAPI(APIGroup{
		Name:  "market-cap",
		Short: "Company market capitalization lookups, history, trends, and rankings",
		Endpoints: []APIEndpoint{
			{Name: "lookup", Short: "Get market cap for a symbol", Method: "GET", Path: "/market-cap",
				Params: []APIParam{symbol, mode}},
			{Name: "history", Short: "Get market cap history for a symbol", Method: "GET", Path: "/market-cap/history",
				Params: []APIParam{symbol, mode, period, startDate, endDate}},
			{Name: "trend", Short: "Get market cap trend statistics for a symbol", Method: "GET", Path: "/market-cap/trend",
				Params: []APIParam{symbol, mode, period, startDate, endDate}},
			{Name: "top", Short: "List the largest companies by market cap", Method: "GET", Path: "/market-cap/top",
				Params: []APIParam{
					{Flag: "limit", Type: "int", Usage: "Number of companies (1-50, default 5)", Example: "5"},
					{Flag: "sector", Usage: "Exact sector filter", Example: "Technology"},
					{Flag: "sort", Usage: "Sort order: desc (default) or asc", Example: "desc"},
					{Flag: "min-market-cap", Wire: "minMarketCap", Type: "float", Usage: "Inclusive lower market cap bound", Example: "100000000000"},
					{Flag: "max-market-cap", Wire: "maxMarketCap", Type: "float", Usage: "Inclusive upper market cap bound", Example: "3000000000000"},
				}},
			{Name: "compare", Short: "Compare market caps across symbols", Method: "GET", Path: "/market-cap/compare",
				Params: []APIParam{{Flag: "symbols", Usage: "Comma-separated symbols (at least two)", Example: "AAPL,MSFT,NVDA", Required: true}}},
			{Name: "catalog", Short: "List supported symbols, sectors, and periods", Method: "GET", Path: "/market-cap/catalog"},
		},
	})
}
