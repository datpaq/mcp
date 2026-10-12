// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "encoding-convert",
		Short: "Encode, decode, hash, JWT-decode, normalize, and data-URI conversions",
		Endpoints: []APIEndpoint{
			{Name: "catalog", Short: "List supported operations, encodings, and algorithms", Method: "GET", Path: "/encoding-convert"},
			{Name: "convert", Short: "Run an encoding operation", Method: "POST", Path: "/encoding-convert",
				Params: []APIParam{
					{Flag: "operation", In: "body", Usage: "encode, decode, hash, jwt_decode, normalize, data_uri_encode, or data_uri_decode", Example: "encode", Required: true},
					{Flag: "input", In: "body", Usage: "Text, encoded value, JWT, or data URI", Example: "Hello, world!", Required: true},
					{Flag: "encoding", In: "body", Usage: "base64, base64url, base32, hex, url, html-entity, or punycode", Example: "base64url"},
					{Flag: "algorithm", In: "body", Usage: "Hash algorithm", Example: "sha256"},
					{Flag: "output", In: "body", Usage: "Hash output: hex, base64, or base64url", Example: "hex"},
					{Flag: "form", In: "body", Usage: "Unicode form: NFC, NFD, NFKC, or NFKD", Example: "NFC"},
					{Flag: "mime-type", Wire: "mime_type", In: "body", Usage: "Data URI MIME type", Example: "text/plain"},
				}},
		},
	})
}
