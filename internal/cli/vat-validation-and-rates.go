// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	checkRegistry := APIParam{Flag: "check-registry", Wire: "checkRegistry", Type: "bool", Usage: "Check live VIES/HMRC registration (default false)", Example: "true"}
	checkRegistryBody := checkRegistry
	checkRegistryBody.In = "body"
	category := APIParam{Flag: "category", Usage: "Rate category: standard (default), reduced, reduced2, superReduced, parking, or zero", Example: "reduced"}

	registerDeclaredAPI(APIGroup{
		Name:  "vat-validation-and-rates",
		Short: "VAT number validation, VAT rates by country, and VAT price calculation",
		Endpoints: []APIEndpoint{
			{Name: "validate", Short: "Validate a VAT number", Method: "GET", Path: "/vat-validation-and-rates/validate",
				Params: []APIParam{
					{Flag: "vat-number", Wire: "vatNumber", Usage: "VAT number with country prefix", Example: "DE123456789", Required: true},
					checkRegistry,
					{Flag: "seller-country-code", Wire: "sellerCountryCode", Usage: "Seller country code; adds reverseChargeApplicable", Example: "FR"},
				}},
			{Name: "validate-number", Short: "Validate a VAT number from the URL path", Method: "GET", Path: "/vat-validation-and-rates/validate/{vatNumber}",
				Params: []APIParam{{Flag: "vat-number", Wire: "vatNumber", In: "path", Usage: "VAT number with country prefix", Example: "DE123456789"}, checkRegistry}},
			{Name: "validate-batch", Short: "Validate up to 50 VAT numbers", Method: "POST", Path: "/vat-validation-and-rates/validate-batch",
				Params: []APIParam{
					{Flag: "vat-numbers", Wire: "vatNumbers", In: "body", Type: "array", Usage: "JSON array or comma-separated VAT numbers (1-50)", Example: "DE123456789,FR40303265045", Required: true},
					checkRegistryBody,
				}},
			{Name: "validate-batch-get", Short: "Validate up to 50 comma-separated VAT numbers via GET", Method: "GET", Path: "/vat-validation-and-rates/validate-batch",
				Params: []APIParam{
					{Flag: "vat-numbers", Wire: "vatNumbers", Usage: "Comma-separated VAT numbers (1-50)", Example: "DE123456789,FR40303265045", Required: true},
					checkRegistry,
				}},
			{Name: "calculate", Short: "Calculate VAT for an amount", Method: "GET", Path: "/vat-validation-and-rates/calculate",
				Params: []APIParam{
					{Flag: "amount", Type: "float", Usage: "Positive amount", Example: "100", Required: true},
					{Flag: "country-code", Wire: "countryCode", Usage: "Two-letter country code", Example: "DE", Required: true},
					category,
				}},
			{Name: "calculate-batch", Short: "Calculate VAT for multiple amounts", Method: "POST", Path: "/vat-validation-and-rates/calculate-batch",
				Params: []APIParam{{Flag: "conversions", In: "body", Type: "array", Usage: `JSON array of {"amount","countryCode","category"} objects`, Example: `[{"amount":100,"countryCode":"DE"}]`, Required: true}}},
			{Name: "calculate-batch-get", Short: "Calculate VAT for multiple amounts via GET", Method: "GET", Path: "/vat-validation-and-rates/calculate-batch",
				Params: []APIParam{{Flag: "conversions", Usage: `JSON-encoded array of {"amount","countryCode","category"} objects`, Example: `[{"amount":100,"countryCode":"DE"}]`, Required: true}}},
			{Name: "rates", Short: "Get VAT rates for a country", Method: "GET", Path: "/vat-validation-and-rates/rates/{countryCode}",
				Params: []APIParam{{Flag: "country-code", Wire: "countryCode", In: "path", Usage: "Two-letter country code", Example: "DE"}}},
			{Name: "rate-countries", Short: "List countries with VAT rate data", Method: "GET", Path: "/vat-validation-and-rates/rates/countries"},
			{Name: "catalog", Short: "List supported countries, categories, and limits", Method: "GET", Path: "/vat-validation-and-rates/catalog"},
		},
	})
}
