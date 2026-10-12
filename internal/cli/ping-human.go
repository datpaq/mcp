// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.
// The /reply/:token routes serve the human responder outside the API gateway
// and are not exposed.

package cli

func init() {
	id := APIParam{Flag: "id", In: "path", Usage: "Escalation ID", Example: "esc_123"}

	registerDeclaredAPI(APIGroup{
		Name:  "ping-human",
		Short: "Escalate a decision to a human via Slack or email and collect the reply",
		Endpoints: []APIEndpoint{
			{Name: "create", Short: "Create an escalation and notify humans", Method: "POST", Path: "/ping-human/escalations",
				Params: []APIParam{
					{Flag: "title", In: "body", Usage: "Short title (1-200 characters)", Example: "Approve refund?", Required: true},
					{Flag: "context", In: "body", Usage: "Decision context (1-8000 characters)", Example: "Customer requests a $120 refund.", Required: true},
					{Flag: "channels", In: "body", Type: "array", Usage: `JSON array of {"type":"email","to":...} or {"type":"slack","webhook_url":...} (1-3)`, Example: `[{"type":"email","to":"ops@example.com"}]`, Required: true},
					{Flag: "options", In: "body", Type: "array", Usage: `JSON array of {"id","label"} choices (2-6)`, Example: `[{"id":"yes","label":"Approve"},{"id":"no","label":"Deny"}]`},
					{Flag: "fields", In: "body", Type: "array", Usage: `JSON array of {"name","type","label","required"} inputs (max 5)`, Example: `[{"name":"note"}]`},
					{Flag: "ttl-seconds", Wire: "ttl_seconds", In: "body", Type: "int", Usage: "Expiry in seconds (60-86400, default 3600)", Example: "3600"},
					{Flag: "callback", In: "body", Type: "object", Usage: `Callback JSON {"url","secret"} for signed reply delivery`, Example: `{"url":"https://example.com/hook","secret":"0123456789abcdef"}`},
				}},
			{Name: "get", Short: "Get an escalation's status and reply", Method: "GET", Path: "/ping-human/escalations/{id}",
				Params: []APIParam{id}},
			{Name: "cancel", Short: "Cancel an open escalation", Method: "DELETE", Path: "/ping-human/escalations/{id}",
				Params: []APIParam{id}},
		},
	})
}
