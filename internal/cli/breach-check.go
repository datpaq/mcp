// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.
// /password also accepts a plaintext password query param; the CLI exposes only
// the k-anonymity hash prefix so passwords never land in URLs or shell history.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "breach-check",
		Short: "Email breach exposure, password k-anonymity checks, and breach catalog",
		Endpoints: []APIEndpoint{
			{Name: "check", Short: "Check one email for known breaches", Method: "GET", Path: "/breach-check/check",
				Params: []APIParam{{Flag: "email", Usage: "Email address", Example: "test@example.com", Required: true}}},
			{Name: "check-batch", Short: "Check up to 25 emails for known breaches", Method: "POST", Path: "/breach-check/check",
				Params: []APIParam{{Flag: "emails", In: "body", Type: "array", Usage: "JSON array or comma-separated emails (1-25)", Example: "a@example.com,b@example.com", Required: true}}},
			{Name: "password", Short: "Check a SHA-1 hash prefix against breached passwords", Method: "GET", Path: "/breach-check/password",
				Params: []APIParam{{Flag: "hash-prefix", Wire: "hash_prefix", Usage: "First 5 hex characters of the password's SHA-1", Example: "5BAA6", Required: true}}},
			{Name: "breaches", Short: "List known breaches", Method: "GET", Path: "/breach-check/breaches",
				Params: []APIParam{{Flag: "domain", Usage: "Filter by breached domain", Example: "adobe.com"}}},
		},
	})
}
