// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Hosted MCP server for Datpaq. Listens on $PORT (or :8080) and
// serves the streamable-HTTP MCP endpoint at /mcp. Each request must
// authenticate with the user's own Datpaq API key as
// `Authorization: Bearer <key>`. Transport-only — the tool catalog
// lives in internal/mcp/tools.go; the auth + ctx wiring lives in
// internal/mcphttp.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/datpaq/mcp/internal/mcphttp"
)

func main() {
	addr := flag.String("addr", defaultAddr(), "listen address (default :$PORT or :8080)")
	baseURL := flag.String("base-url", defaultBaseURL(), "Datpaq API base URL (env: DATPAQ_BASE_URL)")
	flag.Parse()

	log.Printf("datpaq-mcp-http: serving MCP at /mcp on %s (api base: %s)", *addr, *baseURL)
	if err := http.ListenAndServe(*addr, mcphttp.NewHandler(*baseURL)); err != nil {
		fmt.Fprintf(os.Stderr, "datpaq-mcp-http: %v\n", err)
		os.Exit(1)
	}
}

func defaultAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":8080"
}

func defaultBaseURL() string {
	if v := os.Getenv("DATPAQ_BASE_URL"); v != "" {
		return v
	}
	return "https://datpaq.com/api/v1"
}
