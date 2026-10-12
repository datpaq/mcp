// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "pdf-toolkit",
		Short: "Merge, split, extract, rotate, watermark, compress, encrypt, and inspect PDFs",
		Endpoints: []APIEndpoint{
			{Name: "catalog", Short: "List supported operations and limits", Method: "GET", Path: "/pdf-toolkit"},
			{Name: "run", Short: "Run a PDF operation on local files", Method: "POST", Path: "/pdf-toolkit",
				Params: []APIParam{
					{Flag: "operation", In: "body", Usage: "merge, split, extract, rotate, watermark, compress, encrypt, or inspect", Example: "inspect", Required: true},
					{Flag: "file", In: "body", Type: "file", Usage: "Local PDF for non-merge operations", Example: "%PDF-1.4"},
					{Flag: "files", In: "body", Type: "files", Usage: "Comma-separated local PDFs for merge (2-20)", Example: "%PDF-1.4"},
					{Flag: "pages", In: "body", Usage: "Pages: all or ascending ranges", Example: "1,2"},
					{Flag: "angle", In: "body", Type: "float", Usage: "Rotate: multiple of 90; watermark: any angle", Example: "90"},
					{Flag: "text", In: "body", Usage: "Watermark text (1-200 characters)", Example: "CONFIDENTIAL"},
					{Flag: "opacity", In: "body", Type: "float", Usage: "Watermark opacity 0-1 (default 0.6)", Example: "0.6"},
					{Flag: "color", In: "body", Usage: "Watermark RGB as three numbers 0-1", Example: "0.6 0.6 0.6"},
					{Flag: "font-size", Wire: "fontSize", In: "body", Type: "float", Usage: "Watermark font size (8-300)", Example: "48"},
					{Flag: "password", In: "body", Usage: "Encrypt: user password (8+ characters)", Example: "example-user-password"},
					{Flag: "owner-password", Wire: "owner_password", In: "body", Usage: "Encrypt: owner password", Example: "example-owner-password"},
				}},
		},
	})
}
