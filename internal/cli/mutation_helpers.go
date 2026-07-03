// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch helpers for hand-authored API commands added between generator runs.

package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func printMutationResponse(cmd *cobra.Command, flags *rootFlags, resource, path string, statusCode int, data json.RawMessage) error {
	if flags.quiet {
		return nil
	}
	if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
		envelope := map[string]any{
			"action":   "post",
			"resource": resource,
			"path":     path,
			"status":   statusCode,
			"success":  statusCode >= 200 && statusCode < 300,
		}
		if flags.dryRun {
			envelope["dry_run"] = true
			envelope["status"] = 0
			envelope["success"] = false
		}
		filtered := data
		if flags.selectFields != "" {
			filtered = filterFields(filtered, flags.selectFields)
		} else if flags.compact {
			filtered = compactFields(filtered)
		}
		if len(filtered) > 0 {
			var parsed any
			if err := json.Unmarshal(filtered, &parsed); err == nil {
				envelope["data"] = parsed
			}
		}
		envelopeJSON, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		return printOutput(cmd.OutOrStdout(), json.RawMessage(envelopeJSON), true)
	}
	return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
}
