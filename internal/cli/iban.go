// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "iban",
		Short: "IBAN structure and mod-97 checksum validation with country metadata",
		Endpoints: []APIEndpoint{
			{Name: "lookup", Short: "Validate one IBAN from the URL path", Method: "GET", Path: "/iban/{iban}",
				Params: []APIParam{{Flag: "iban", In: "path", Usage: "IBAN to validate", Example: "DE89370400440532013000"}}},
			{Name: "validate", Short: "Validate one IBAN", Method: "POST", Path: "/iban/validate",
				Params: []APIParam{
					{Flag: "iban", In: "body", Usage: "IBAN to validate (15-34 characters)", Example: "DE89370400440532013000", Required: true},
					{Flag: "include-metadata", Wire: "includeMetadata", In: "body", Type: "bool", Usage: "Include country/BBAN metadata (default true)", Example: "true"},
				}},
			{Name: "validate-batch", Short: "Validate up to 100 IBANs", Method: "POST", Path: "/iban/validate-batch",
				Params: []APIParam{
					{Flag: "ibans", In: "body", Type: "array", Usage: "JSON array or comma-separated IBANs (1-100)", Example: "DE89370400440532013000,GB82WEST12345698765432", Required: true},
					{Flag: "include-metadata", Wire: "includeMetadata", In: "body", Type: "bool", Usage: "Include country/BBAN metadata (default false)", Example: "true"},
				}},
			{Name: "countries", Short: "List supported IBAN countries", Method: "GET", Path: "/iban/countries"},
		},
	})
}
