// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import (
	"github.com/spf13/cobra"
)

func newQRCodeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "qr-code",
		Short:  "Generate and decode QR codes",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newQRCodeGenerateCmd(flags))
	cmd.AddCommand(newQRCodeDecodeCmd(flags))
	return cmd
}

func newQRCodeGenerateCmd(flags *rootFlags) *cobra.Command {
	var text, format, errorCorrectionLevel, darkColor, lightColor, responseType string
	var size, margin int
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "generate",
		Short:       "Generate a QR code image from text",
		Example:     "  datpaq qr-code generate --text 'https://datpaq.com' --format png --size 256",
		Annotations: map[string]string{"pp:endpoint": "qr-code.generate", "pp:method": "POST", "pp:path": "/qr-code/generate"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlagUnlessStdin(cmd, flags, "text", stdinBody); err != nil {
				return err
			}
			path := "/qr-code/generate"
			var body map[string]any
			if stdinBody {
				var err error
				body, err = readStdinJSONBody()
				if err != nil {
					return err
				}
			} else {
				body = map[string]any{"text": text}
				setBodyString(cmd, body, "format", "format", format)
				setBodyInt(cmd, body, "size", "size", size)
				setBodyInt(cmd, body, "margin", "margin", margin)
				setBodyString(cmd, body, "error-correction-level", "errorCorrectionLevel", errorCorrectionLevel)
				setBodyString(cmd, body, "dark-color", "darkColor", darkColor)
				setBodyString(cmd, body, "light-color", "lightColor", lightColor)
				setBodyString(cmd, body, "response-type", "responseType", responseType)
			}
			return postMutation(cmd, flags, "qr-code", path, body)
		},
	}
	cmd.Flags().StringVar(&text, "text", "", "Text or URL to encode")
	cmd.Flags().StringVar(&format, "format", "png", "Image format: png or svg")
	cmd.Flags().IntVar(&size, "size", 256, "Image size in pixels (64-2048)")
	cmd.Flags().IntVar(&margin, "margin", 2, "Quiet-zone margin (0-16)")
	cmd.Flags().StringVar(&errorCorrectionLevel, "error-correction-level", "M", "Error correction: L, M, Q, or H")
	cmd.Flags().StringVar(&darkColor, "dark-color", "#000000", "Dark module color (#RRGGBB)")
	cmd.Flags().StringVar(&lightColor, "light-color", "#ffffff", "Light module color (#RRGGBB)")
	cmd.Flags().StringVar(&responseType, "response-type", "json", "Response type: binary, base64, or json")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}

func newQRCodeDecodeCmd(flags *rootFlags) *cobra.Command {
	var imageBase64 string
	var all bool
	var maxCodes int
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "decode",
		Short:       "Decode QR code(s) from a base64 image",
		Example:     "  datpaq qr-code decode --stdin < image.json",
		Annotations: map[string]string{"pp:endpoint": "qr-code.decode", "pp:method": "POST", "pp:path": "/qr-code/decode"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlagUnlessStdin(cmd, flags, "image-base64", stdinBody); err != nil {
				return err
			}
			path := "/qr-code/decode"
			var body map[string]any
			if stdinBody {
				var err error
				body, err = readStdinJSONBody()
				if err != nil {
					return err
				}
			} else {
				body = map[string]any{"imageBase64": imageBase64}
				setBodyBool(cmd, body, "all", "all", all)
				setBodyInt(cmd, body, "max-codes", "maxCodes", maxCodes)
			}
			return postMutation(cmd, flags, "qr-code", path, body)
		},
	}
	cmd.Flags().StringVar(&imageBase64, "image-base64", "", "Base64-encoded image data (prefer --stdin for large payloads)")
	cmd.Flags().BoolVar(&all, "all", false, "Return all detected codes")
	cmd.Flags().IntVar(&maxCodes, "max-codes", 5, "Max codes to return when --all is set (1-10)")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}
