// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWebSearchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "web-search",
		Short:  "Web search for a single query or a batch of queries",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newWebSearchSearchCmd(flags))
	cmd.AddCommand(newWebSearchBatchCmd(flags))
	return cmd
}

func newWebSearchSearchCmd(flags *rootFlags) *cobra.Command {
	var q string
	var limit int

	cmd := &cobra.Command{
		Use:         "search",
		Short:       "Search the web for a single query",
		Example:     "  datpaq web-search search --q 'datpaq api' --limit 5",
		Annotations: map[string]string{"pp:endpoint": "web-search.search", "pp:method": "GET", "pp:path": "/web-search/search", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlag(cmd, flags, "q"); err != nil {
				return err
			}
			params := map[string]string{"q": q}
			setQueryInt(cmd, params, "limit", "limit", limit)
			return printReadResponse(cmd, flags, "web-search", "/web-search/search", params)
		},
	}
	cmd.Flags().StringVar(&q, "q", "", "Search query (1-256 characters)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max related topics and results per section (1-10)")
	return cmd
}

func newWebSearchBatchCmd(flags *rootFlags) *cobra.Command {
	var queriesRaw string
	var limit int
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "batch",
		Short:       "Batch web search for up to 10 queries",
		Example:     `  datpaq web-search batch --queries '["weather nyc","geocoding api"]' --limit 5`,
		Annotations: map[string]string{"pp:endpoint": "web-search.batch", "pp:method": "POST", "pp:path": "/web-search/search"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlagUnlessStdin(cmd, flags, "queries", stdinBody); err != nil {
				return err
			}
			path := "/web-search/search"
			var body map[string]any
			if stdinBody {
				var err error
				body, err = readStdinJSONBody()
				if err != nil {
					return err
				}
			} else {
				queries, err := parseJSONOrCSVList(queriesRaw)
				if err != nil {
					return fmt.Errorf("parsing --queries: %w", err)
				}
				body = map[string]any{"queries": queries}
				setBodyInt(cmd, body, "limit", "limit", limit)
			}
			return postMutation(cmd, flags, "web-search", path, body)
		},
	}
	cmd.Flags().StringVar(&queriesRaw, "queries", "", "JSON array or comma-separated search queries")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max related topics and results per section (1-10)")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}
