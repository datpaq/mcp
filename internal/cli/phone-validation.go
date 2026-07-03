// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import "github.com/spf13/cobra"

func newPhoneValidationCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "phone-validation",
		Short:  "Phone number validation, batch validation, and formatting",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}

	cmd.AddCommand(newPhoneValidationValidateCmd(flags))
	cmd.AddCommand(newPhoneValidationValidateBatchCmd(flags))
	cmd.AddCommand(newPhoneValidationFormatCmd(flags))
	return cmd
}
