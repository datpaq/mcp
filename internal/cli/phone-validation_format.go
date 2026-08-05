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

func newPhoneValidationFormatCmd(flags *rootFlags) *cobra.Command {
	var bodyPhoneNumber string
	var bodyCountryCode string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "format",
		Short:       "Format a phone number as you type",
		Example:     "  datpaq phone-validation format --phone-number 4155552671 --country-code US",
		Annotations: map[string]string{"pp:endpoint": "phone-validation.format", "pp:method": "POST", "pp:path": "/phone-validation/format"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !stdinBody && !cmd.Flags().Changed("phone-number") && !flags.dryRun {
				return fmt.Errorf("required flag \"%s\" not set", "phone-number")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			path := "/phone-validation/format"
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
				if bodyPhoneNumber != "" {
					body["phoneNumber"] = bodyPhoneNumber
				}
				if bodyCountryCode != "" {
					body["countryCode"] = bodyCountryCode
				}
			}

			data, statusCode, err := c.PostWithParams(path, map[string]string{}, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printMutationResponse(cmd, flags, "phone-validation", path, statusCode, data)
		},
	}
	cmd.Flags().StringVar(&bodyPhoneNumber, "phone-number", "", "Phone number to format")
	cmd.Flags().StringVar(&bodyCountryCode, "country-code", "", "Optional 2-letter ISO country code (e.g. US, GB)")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}
