// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newWebScrapingCmd(flags *rootFlags) *cobra.Command {
	var bodyURL string
	var bodyFormat string
	var bodyWaitUntil string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "web-scraping",
		Short:       "SSRF-safe public web page content extraction",
		Example:     "  datpaq web-scraping --url https://example.com --format markdown",
		Annotations: map[string]string{"pp:endpoint": "web-scraping.scrape", "pp:method": "POST", "pp:path": "/web-scraping/scrape"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !stdinBody && !cmd.Flags().Changed("url") && !flags.dryRun {
				return fmt.Errorf("required flag \"%s\" not set", "url")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			path := "/web-scraping/scrape"
			body := map[string]any{}
			if stdinBody {
				stdinData, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				if err := json.Unmarshal(stdinData, &body); err != nil {
					return fmt.Errorf("parsing stdin JSON: %w", err)
				}
			} else {
				if bodyURL != "" {
					body["url"] = bodyURL
				}
				if cmd.Flags().Changed("format") {
					body["format"] = bodyFormat
				}
				if cmd.Flags().Changed("wait-until") {
					body["waitUntil"] = bodyWaitUntil
				}
			}

			data, statusCode, err := c.PostWithParams(path, map[string]string{}, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printMutationResponse(cmd, flags, "web-scraping", path, statusCode, data)
		},
	}
	cmd.Flags().StringVar(&bodyURL, "url", "", "Public HTTP(S) URL to scrape")
	cmd.Flags().StringVar(&bodyFormat, "format", "json", "Output format requested from the API (json, markdown, html)")
	cmd.Flags().StringVar(&bodyWaitUntil, "wait-until", "networkidle2", "Browser wait condition (load, domcontentloaded, networkidle0, networkidle2)")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}
