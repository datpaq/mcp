// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "mortgage-calculator",
		Short: "Mortgage payment, refinance break-even, and current mortgage rates",
		Endpoints: []APIEndpoint{
			{Name: "payment", Short: "Calculate monthly payment, total interest, and optional amortization", Method: "GET", Path: "/mortgage-calculator/payment",
				Params: []APIParam{
					{Flag: "interest-rate", Wire: "interest_rate", Type: "float", Usage: "Annual percentage rate", Example: "6.5", Required: true},
					{Flag: "loan-amount", Wire: "loan_amount", Type: "float", Usage: "Loan principal (or pass --home-value and --downpayment)", Example: "350000"},
					{Flag: "home-value", Wire: "home_value", Type: "float", Usage: "Home value, used with --downpayment", Example: "400000"},
					{Flag: "downpayment", Type: "float", Usage: "Down payment, less than --home-value", Example: "50000"},
					{Flag: "duration-years", Wire: "duration_years", Type: "float", Usage: "Loan term in years (default 30)", Example: "30"},
					{Flag: "monthly-extra-payment", Wire: "monthly_extra_payment", Type: "float", Usage: "Additional monthly principal payment", Example: "200"},
					{Flag: "monthly-hoa", Wire: "monthly_hoa", Type: "float", Usage: "Monthly HOA amount", Example: "150"},
					{Flag: "annual-property-tax", Wire: "annual_property_tax", Type: "float", Usage: "Annual property tax", Example: "4800"},
					{Flag: "annual-home-insurance", Wire: "annual_home_insurance", Type: "float", Usage: "Annual home insurance", Example: "1200"},
					{Flag: "include-schedule", Wire: "includeSchedule", Type: "bool", Usage: "Include an amortization schedule", Example: "true"},
					{Flag: "start-date", Wire: "start_date", Usage: "Schedule start date (YYYY-MM-DD)", Example: "2026-01-01"},
					{Flag: "annual-appreciation-rate", Wire: "annual_appreciation_rate", Type: "float", Usage: "Annual home appreciation percentage (-100 to 100)", Example: "3"},
				}},
			{Name: "refinance", Short: "Simulate a refinance and its break-even point", Method: "GET", Path: "/mortgage-calculator/refinance",
				Params: []APIParam{
					{Flag: "current-loan-amount", Wire: "current_loan_amount", Type: "float", Usage: "Current loan balance", Example: "280000", Required: true},
					{Flag: "current-interest-rate", Wire: "current_interest_rate", Type: "float", Usage: "Current annual rate", Example: "7", Required: true},
					{Flag: "current-remaining-years", Wire: "current_remaining_years", Type: "float", Usage: "Current remaining term in years", Example: "25", Required: true},
					{Flag: "new-interest-rate", Wire: "new_interest_rate", Type: "float", Usage: "Proposed annual rate", Example: "5.75", Required: true},
					{Flag: "new-duration-years", Wire: "new_duration_years", Type: "float", Usage: "Proposed term in years", Example: "25", Required: true},
					{Flag: "refinance-cost", Wire: "refinance_cost", Type: "float", Usage: "Refinance closing cost", Example: "5000"},
				}},
			{Name: "current-rates", Short: "Get weekly average mortgage rates from FRED", Method: "GET", Path: "/mortgage-calculator/current-rates"},
		},
	})
}
