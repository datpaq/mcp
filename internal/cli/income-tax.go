// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	filingStatus := APIParam{Flag: "filing-status", Wire: "filingStatus", Usage: "single (default), married_joint, or head_of_household", Example: "single"}
	taxYear := APIParam{Flag: "tax-year", Wire: "taxYear", Type: "int", Usage: "Tax year (default 2024)", Example: "2024"}

	registerDeclaredAPI(APIGroup{
		Name:  "income-tax",
		Short: "US federal income tax estimates and brackets",
		Endpoints: []APIEndpoint{
			{Name: "estimate", Short: "Estimate federal and state income tax", Method: "GET", Path: "/income-tax",
				Params: []APIParam{
					{Flag: "income", Type: "float", Usage: "Gross income", Example: "120000", Required: true},
					filingStatus, taxYear,
					{Flag: "deductions", Type: "float", Usage: "Deductions beyond the standard deduction", Example: "5000"},
					{Flag: "tax-credits", Wire: "taxCredits", Type: "float", Usage: "Federal tax credits", Example: "2000"},
					{Flag: "state-rate", Wire: "stateRate", Type: "float", Usage: "State tax rate percentage (0-20)", Example: "5.5"},
				}},
			{Name: "brackets", Short: "Get federal tax brackets", Method: "GET", Path: "/income-tax/brackets",
				Params: []APIParam{filingStatus, taxYear}},
			{Name: "catalog", Short: "List supported years, filing statuses, and sources", Method: "GET", Path: "/income-tax/catalog"},
		},
	})
}
