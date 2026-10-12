// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "document-convert",
		Short: "Convert office documents, PDF, and text formats",
		Endpoints: []APIEndpoint{
			{Name: "formats", Short: "List supported conversion pairs", Method: "GET", Path: "/document-convert/formats"},
			{Name: "convert", Short: "Convert one document from a local file or URL", Method: "POST", Path: "/document-convert/convert",
				Params: []APIParam{
					{Flag: "source-format", Wire: "sourceFormat", In: "body", Usage: "Source format", Example: "docx", Required: true},
					{Flag: "target-format", Wire: "targetFormat", In: "body", Usage: "Target format", Example: "pdf", Required: true},
					{Flag: "file", Wire: "fileBase64", In: "body", Type: "file", Usage: "Local file to upload (or pass --file-url)", Example: "document bytes"},
					{Flag: "file-url", Wire: "fileUrl", In: "body", Usage: "Public HTTP(S) file URL (or pass --file)", Example: "https://example.com/report.docx"},
					{Flag: "filename", In: "body", Usage: "Original filename", Example: "report.docx"},
				}},
		},
	})
}
