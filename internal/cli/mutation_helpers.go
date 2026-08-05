// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch helpers for hand-authored API commands added between generator runs.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

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

// printReadResponse runs a GET through resolveRead and prints with the same
// provenance / table / JSON gates as generated read commands.
func printReadResponse(cmd *cobra.Command, flags *rootFlags, resource, path string, params map[string]string) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	data, prov, err := resolveRead(cmd.Context(), c, flags, resource, false, path, params, nil)
	if err != nil {
		return classifyAPIError(err, flags)
	}
	data = extractResponseData(data)

	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		var countItems []json.RawMessage
		if json.Unmarshal(data, &countItems) != nil {
			countItems = []json.RawMessage{data}
		}
		printProvenance(cmd, len(countItems), prov)
	}
	if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
		filtered := data
		if flags.selectFields != "" {
			filtered = filterFields(filtered, flags.selectFields)
		} else if flags.compact {
			filtered = compactFields(filtered)
		}
		wrapped, wrapErr := wrapWithProvenance(filtered, prov)
		if wrapErr != nil {
			return wrapErr
		}
		return printOutput(cmd.OutOrStdout(), wrapped, true)
	}
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		var items []map[string]any
		if json.Unmarshal(data, &items) == nil && len(items) > 0 {
			if err := printAutoTable(cmd.OutOrStdout(), items); err != nil {
				return err
			}
			if len(items) >= 25 {
				fmt.Fprintf(os.Stderr, "\nShowing %d results. To narrow: add --limit, --json --select, or filter flags.\n", len(items))
			}
			return nil
		}
	}
	return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
}

func requireFlag(cmd *cobra.Command, flags *rootFlags, name string) error {
	if !cmd.Flags().Changed(name) && !flags.dryRun {
		return fmt.Errorf("required flag \"%s\" not set", name)
	}
	return nil
}

func setQueryString(cmd *cobra.Command, params map[string]string, flagName, wireName, value string) {
	if cmd.Flags().Changed(flagName) {
		params[wireName] = value
	}
}

func setQueryBool(cmd *cobra.Command, params map[string]string, flagName, wireName string, value bool) {
	if cmd.Flags().Changed(flagName) {
		params[wireName] = fmt.Sprintf("%t", value)
	}
}

func setQueryInt(cmd *cobra.Command, params map[string]string, flagName, wireName string, value int) {
	if cmd.Flags().Changed(flagName) {
		params[wireName] = fmt.Sprintf("%d", value)
	}
}

func setQueryFloat(cmd *cobra.Command, params map[string]string, flagName, wireName string, value float64) {
	if cmd.Flags().Changed(flagName) {
		params[wireName] = fmt.Sprintf("%v", value)
	}
}

func requireFlagUnlessStdin(cmd *cobra.Command, flags *rootFlags, name string, stdinBody bool) error {
	if !stdinBody && !cmd.Flags().Changed(name) && !flags.dryRun {
		return fmt.Errorf("required flag \"%s\" not set", name)
	}
	return nil
}

func readStdinJSONBody() (map[string]any, error) {
	stdinData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	body := map[string]any{}
	if err := json.Unmarshal(stdinData, &body); err != nil {
		return nil, fmt.Errorf("parsing stdin JSON: %w", err)
	}
	return body, nil
}

func setBodyString(cmd *cobra.Command, body map[string]any, flagName, wireName, value string) {
	if cmd.Flags().Changed(flagName) {
		body[wireName] = value
	}
}

func setBodyBool(cmd *cobra.Command, body map[string]any, flagName, wireName string, value bool) {
	if cmd.Flags().Changed(flagName) {
		body[wireName] = value
	}
}

func setBodyInt(cmd *cobra.Command, body map[string]any, flagName, wireName string, value int) {
	if cmd.Flags().Changed(flagName) {
		body[wireName] = value
	}
}

func setBodyAny(cmd *cobra.Command, body map[string]any, flagName, wireName string, value any) {
	if cmd.Flags().Changed(flagName) {
		body[wireName] = value
	}
}

func postMutation(cmd *cobra.Command, flags *rootFlags, resource, path string, body map[string]any) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	data, statusCode, err := c.PostWithParams(path, map[string]string{}, body)
	if err != nil {
		return classifyAPIError(err, flags)
	}
	return printMutationResponse(cmd, flags, resource, path, statusCode, data)
}

func addHiddenStdinFlag(cmd *cobra.Command, stdinBody *bool) {
	cmd.Flags().BoolVar(stdinBody, "stdin", false, "Read request body as JSON from stdin")
	_ = cmd.Flags().MarkHidden("stdin")
}
