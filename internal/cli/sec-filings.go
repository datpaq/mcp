// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	cik := APIParam{Flag: "cik", Usage: "SEC CIK identifier (or pass --ticker)", Example: "320193"}
	ticker := APIParam{Flag: "ticker", Usage: "Stock ticker (or pass --cik)", Example: "AAPL"}
	filingFilters := []APIParam{
		cik, ticker,
		{Flag: "sort", Usage: "Filing date order: desc (default) or asc", Example: "desc"},
		{Flag: "limit", Type: "int", Usage: "Max filings (1-100, default 20)", Example: "20"},
		{Flag: "offset", Type: "int", Usage: "Pagination offset (0-2000)", Example: "0"},
		{Flag: "from-date", Wire: "fromDate", Usage: "Inclusive lower filing date (YYYY-MM-DD)", Example: "2024-01-01"},
		{Flag: "to-date", Wire: "toDate", Usage: "Inclusive upper filing date (YYYY-MM-DD)", Example: "2024-12-31"},
	}

	registerDeclaredAPI(APIGroup{
		Name:  "sec-filings",
		Short: "SEC EDGAR company profiles, filings, and XBRL facts",
		Endpoints: []APIEndpoint{
			{Name: "catalog", Short: "List supported forms, taxonomies, and limits", Method: "GET", Path: "/sec-filings/catalog"},
			{Name: "company", Short: "Get a company profile by CIK or ticker", Method: "GET", Path: "/sec-filings/company",
				Params: []APIParam{cik, ticker}},
			{Name: "filings", Short: "List recent filings for a company", Method: "GET", Path: "/sec-filings/filings",
				Params: append([]APIParam{
					{Flag: "form", Usage: "Single form filter", Example: "10-K"},
					{Flag: "forms", Usage: "Comma-separated form filters", Example: "10-K,10-Q"},
				}, filingFilters...)},
			{Name: "forms", Short: "List a company's filings of one form type", Method: "GET", Path: "/sec-filings/forms/{formType}",
				Params: append([]APIParam{{Flag: "form-type", Wire: "formType", In: "path", Usage: "Form type", Example: "10-K"}}, filingFilters...)},
			{Name: "facts", Short: "Get XBRL concept facts for a company", Method: "GET", Path: "/sec-filings/facts",
				Params: []APIParam{
					cik, ticker,
					{Flag: "tag", Usage: "XBRL concept tag", Example: "Assets", Required: true},
					{Flag: "taxonomy", Usage: "XBRL taxonomy (default us-gaap)", Example: "us-gaap"},
					{Flag: "units", Usage: "Preferred unit key", Example: "USD"},
					{Flag: "limit", Type: "int", Usage: "Max facts (1-40, default 12)", Example: "12"},
				}},
		},
	})
}
