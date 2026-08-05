// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import (
	"github.com/spf13/cobra"
)

func newPDFGenerationCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "pdf-generation",
		Short:  "Generate PDFs from a URL or HTML",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newPDFGenerationFromURLCmd(flags))
	cmd.AddCommand(newPDFGenerationFromHTMLCmd(flags))
	return cmd
}

func newPDFGenerationFromURLCmd(flags *rootFlags) *cobra.Command {
	var url, format, waitUntil, responseType string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "from-url",
		Short:       "Render a public URL to PDF",
		Example:     "  datpaq pdf-generation from-url --url https://example.com --format A4 --response-type base64",
		Annotations: map[string]string{"pp:endpoint": "pdf-generation.from-url", "pp:method": "POST", "pp:path": "/pdf-generation/from-url"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlagUnlessStdin(cmd, flags, "url", stdinBody); err != nil {
				return err
			}
			path := "/pdf-generation/from-url"
			var body map[string]any
			if stdinBody {
				var err error
				body, err = readStdinJSONBody()
				if err != nil {
					return err
				}
			} else {
				body = map[string]any{"url": url}
				setBodyString(cmd, body, "format", "format", format)
				setBodyString(cmd, body, "wait-until", "waitUntil", waitUntil)
				// Always send responseType so agent/CLI get JSON rather than raw PDF bytes by default.
				if cmd.Flags().Changed("response-type") {
					body["responseType"] = responseType
				} else {
					body["responseType"] = "base64"
				}
			}
			return postMutation(cmd, flags, "pdf-generation", path, body)
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "Public HTTP(S) URL to render")
	cmd.Flags().StringVar(&format, "format", "A4", "Page format: A4, Letter, Legal, or Tabloid")
	cmd.Flags().StringVar(&waitUntil, "wait-until", "networkidle2", "Browser wait condition")
	cmd.Flags().StringVar(&responseType, "response-type", "base64", "Response type: binary or base64 (CLI defaults to base64)")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}

func newPDFGenerationFromHTMLCmd(flags *rootFlags) *cobra.Command {
	var html, format, responseType string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "from-html",
		Short:       "Render HTML content to PDF",
		Example:     `  datpaq pdf-generation from-html --html '<h1>Hello</h1>' --response-type base64`,
		Annotations: map[string]string{"pp:endpoint": "pdf-generation.from-html", "pp:method": "POST", "pp:path": "/pdf-generation/from-html"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlagUnlessStdin(cmd, flags, "html", stdinBody); err != nil {
				return err
			}
			path := "/pdf-generation/from-html"
			var body map[string]any
			if stdinBody {
				var err error
				body, err = readStdinJSONBody()
				if err != nil {
					return err
				}
			} else {
				body = map[string]any{"html": html}
				setBodyString(cmd, body, "format", "format", format)
				if cmd.Flags().Changed("response-type") {
					body["responseType"] = responseType
				} else {
					body["responseType"] = "base64"
				}
			}
			return postMutation(cmd, flags, "pdf-generation", path, body)
		},
	}
	cmd.Flags().StringVar(&html, "html", "", "HTML document to render")
	cmd.Flags().StringVar(&format, "format", "A4", "Page format: A4, Letter, Legal, or Tabloid")
	cmd.Flags().StringVar(&responseType, "response-type", "base64", "Response type: binary or base64 (CLI defaults to base64)")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}
