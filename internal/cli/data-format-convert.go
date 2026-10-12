// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "data-format-convert",
		Short: "Convert data between JSON, CSV, XML, YAML, and spreadsheet formats",
		Endpoints: []APIEndpoint{
			{Name: "convert", Short: "Convert data from one format to another", Method: "POST", Path: "/data-format-convert/convert",
				Params: []APIParam{
					{Flag: "input-format", Wire: "inputFormat", In: "body", Usage: "Input format", Example: "json", Required: true},
					{Flag: "output-format", Wire: "outputFormat", In: "body", Usage: "Output format", Example: "csv", Required: true},
					{Flag: "data", In: "body", Usage: "Input data as a string (use --stdin to send JSON objects)", Example: `[{"name":"Ada","age":36}]`, Required: true},
					{Flag: "options", In: "body", Type: "object", Usage: "Conversion options JSON (delimiter, pretty, xmlRoot, ...)", Example: `{"pretty":true}`},
				}},
		},
	})
}
