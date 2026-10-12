// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "ocr",
		Short: "Extract text from PDFs and images, with optional OCR",
		Endpoints: []APIEndpoint{
			{Name: "parse", Short: "Extract text from a local file or URL", Method: "POST", Path: "/ocr/parse",
				Params: []APIParam{
					{Flag: "file", Wire: "fileBase64", In: "body", Type: "file", Usage: "Local PDF or image (or pass --url)", Example: "%PDF-1.4"},
					{Flag: "url", In: "body", Usage: "Public HTTP(S) file URL (or pass --file)", Example: "https://example.com/scan.pdf"},
					{Flag: "filename", In: "body", Usage: "Original filename", Example: "scan.pdf"},
					{Flag: "ocr", In: "body", Type: "bool", Usage: "Force OCR on images and scanned pages (default false)", Example: "true"},
					{Flag: "language", In: "body", Usage: "OCR language code (default eng)", Example: "eng"},
				}},
		},
	})
}
