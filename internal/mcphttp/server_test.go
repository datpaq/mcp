// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.

package mcphttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHandlerRoutes(t *testing.T) {
	h := NewHandler("http://example.invalid")

	cases := []struct {
		name           string
		method         string
		path           string
		authz          string
		body           string
		wantStatus     int
		wantLocation   string
		statusNotEqual int // when >0, status must not equal this (use for "anything but X")
	}{
		{
			name:         "GET / redirects to docs",
			method:       http.MethodGet,
			path:         "/",
			wantStatus:   http.StatusFound,
			wantLocation: "https://datpaq.com/docs/mcp",
		},
		{
			name:       "POST / without auth is 401",
			method:     http.MethodPost,
			path:       "/",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:           "POST / with bearer reaches MCP handler",
			method:         http.MethodPost,
			path:           "/",
			authz:          "Bearer test",
			body:           `{"jsonrpc":"2.0","method":"initialize","id":1}`,
			statusNotEqual: http.StatusUnauthorized,
		},
		{
			name:       "GET /healthz still 200",
			method:     http.MethodGet,
			path:       "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET /unknown is 404",
			method:     http.MethodGet,
			path:       "/unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "POST /mcp without auth is 401 (regression)",
			method:     http.MethodPost,
			path:       "/mcp",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *strings.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			} else {
				body = strings.NewReader("")
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			if tc.authz != "" {
				req.Header.Set("Authorization", tc.authz)
			}
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if tc.wantStatus != 0 && rr.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.statusNotEqual != 0 && rr.Code == tc.statusNotEqual {
				t.Fatalf("status = %d, must not equal %d", rr.Code, tc.statusNotEqual)
			}
			if tc.wantLocation != "" {
				if got := rr.Header().Get("Location"); got != tc.wantLocation {
					t.Fatalf("Location = %q, want %q", got, tc.wantLocation)
				}
			}
		})
	}
}
