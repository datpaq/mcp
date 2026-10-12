// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	docType := APIParam{Flag: "document-type", Wire: "documentType", In: "path", Usage: "Document type slug (see documents)", Example: "invoice"}
	renderBody := []APIParam{
		docType,
		{Flag: "mode", In: "body", Usage: "filled (default) or blank", Example: "filled"},
		{Flag: "data", In: "body", Type: "object", Usage: "Document data JSON (see the type's schema/example)", Example: `{"invoiceNumber":"INV-1001"}`},
		{Flag: "branding", In: "body", Type: "object", Usage: "Branding JSON (organizationName, primaryColor, footerText, ...)", Example: `{"organizationName":"Datpaq"}`},
	}

	registerDeclaredAPI(APIGroup{
		Name:  "document-generation",
		Short: "Generate business documents (invoices, letters, forms) as PDF, DOCX, or HTML",
		Endpoints: []APIEndpoint{
			{Name: "documents", Short: "List available document types", Method: "GET", Path: "/document-generation/documents"},
			{Name: "openapi", Short: "Get the service OpenAPI document", Method: "GET", Path: "/document-generation/openapi.json"},
			{Name: "schema", Short: "Get a document type's field schema", Method: "GET", Path: "/document-generation/{documentType}/schema",
				Params: []APIParam{docType}},
			{Name: "json-schema", Short: "Get a document type's JSON Schema", Method: "GET", Path: "/document-generation/{documentType}/json-schema",
				Params: []APIParam{docType}},
			{Name: "example", Short: "Get example data for a document type", Method: "GET", Path: "/document-generation/{documentType}/example",
				Params: []APIParam{docType}},
			{Name: "render-blank", Short: "Render a blank document as HTML", Method: "GET", Path: "/document-generation/{documentType}/render",
				Params: []APIParam{docType}},
			{Name: "render", Short: "Render a document with data as HTML", Method: "POST", Path: "/document-generation/{documentType}/render",
				Params: renderBody},
			{Name: "generate", Short: "Generate a PDF or DOCX document", Method: "POST", Path: "/document-generation/{documentType}/generate",
				Params: append(append([]APIParam(nil), renderBody...),
					APIParam{Flag: "output", In: "body", Type: "object", Usage: `Output JSON: {"type":"pdf|docx","format":"Letter","responseType":"base64|binary","landscape":false}`, Example: `{"type":"pdf","responseType":"base64"}`})},
			{Name: "batch", Short: "Generate up to 5 documents as base64", Method: "POST", Path: "/document-generation/batch",
				Params: []APIParam{{Flag: "requests", In: "body", Type: "array", Usage: `JSON array of {"documentType","data","output"} requests (1-5)`, Example: `[{"documentType":"invoice","data":{}}]`, Required: true}}},
		},
	})
}
