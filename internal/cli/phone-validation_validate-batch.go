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

func newPhoneValidationValidateBatchCmd(flags *rootFlags) *cobra.Command {
	var bodyPhoneNumbers string
	var bodyDefaultCountryCode string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "validate-batch",
		Short:       "Validate multiple phone numbers",
		Example:     "  datpaq phone-validation validate-batch --phone-numbers '+14155552671,+442079460958'",
		Annotations: map[string]string{"pp:endpoint": "phone-validation.validate-batch", "pp:method": "POST", "pp:path": "/phone-validation/validate-batch"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !stdinBody && !cmd.Flags().Changed("phone-numbers") && !flags.dryRun {
				return fmt.Errorf("required flag \"%s\" not set", "phone-numbers")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			path := "/phone-validation/validate-batch"
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
				if bodyPhoneNumbers != "" {
					phoneNumbers, err := parseJSONOrCSVList(bodyPhoneNumbers)
					if err != nil {
						return fmt.Errorf("parsing --phone-numbers: %w", err)
					}
					body["phoneNumbers"] = phoneNumbers
				}
				if bodyDefaultCountryCode != "" {
					body["defaultCountryCode"] = bodyDefaultCountryCode
				}
			}

			data, statusCode, err := c.PostWithParams(path, map[string]string{}, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return printMutationResponse(cmd, flags, "phone-validation", path, statusCode, data)
		},
	}
	cmd.Flags().StringVar(&bodyPhoneNumbers, "phone-numbers", "", "Comma-separated phone numbers or JSON array")
	cmd.Flags().StringVar(&bodyDefaultCountryCode, "default-country-code", "", "Default 2-letter ISO country code for national numbers")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}
