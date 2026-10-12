// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: staging-catalog ProApi service not present in the generated spec.
// POST /color-convert mirrors the GET route and is not exposed separately.

package cli

func init() {
	registerDeclaredAPI(APIGroup{
		Name:  "color-convert",
		Short: "Color conversion across formats with contrast and gradient helpers",
		Endpoints: []APIEndpoint{
			{Name: "convert", Short: "Convert a color and optionally check contrast", Method: "GET", Path: "/color-convert",
				Params: []APIParam{
					{Flag: "color", Usage: "Color in any supported format (hex, rgb, hsl, name)", Example: "#3B82F6", Required: true},
					{Flag: "contrast-with", Wire: "contrast_with", Usage: "Background color for contrast checks", Example: "#FFFFFF"},
					{Flag: "gradient-angle", Wire: "gradient_angle", Type: "float", Usage: "Gradient angle (-360 to 360, default 90)", Example: "135"},
				}},
		},
	})
}
