// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newGeocodingCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "geocoding",
		Short:  "Forward, reverse, and batch geocoding",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newGeocodingForwardCmd(flags))
	cmd.AddCommand(newGeocodingReverseCmd(flags))
	cmd.AddCommand(newGeocodingBatchCmd(flags))
	return cmd
}

func newGeocodingForwardCmd(flags *rootFlags) *cobra.Command {
	var q, language, countryCodes, viewbox string
	var limit int
	var bounded bool

	cmd := &cobra.Command{
		Use:         "forward",
		Short:       "Geocode an address or place name to coordinates",
		Example:     "  datpaq geocoding forward --q 'Empire State Building' --limit 3",
		Annotations: map[string]string{"pp:endpoint": "geocoding.forward", "pp:method": "GET", "pp:path": "/geocoding/forward", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlag(cmd, flags, "q"); err != nil {
				return err
			}
			params := map[string]string{"q": q}
			setQueryInt(cmd, params, "limit", "limit", limit)
			setQueryString(cmd, params, "language", "language", language)
			setQueryString(cmd, params, "country-codes", "countryCodes", countryCodes)
			setQueryString(cmd, params, "viewbox", "viewbox", viewbox)
			setQueryBool(cmd, params, "bounded", "bounded", bounded)
			return printReadResponse(cmd, flags, "geocoding", "/geocoding/forward", params)
		},
	}
	cmd.Flags().StringVar(&q, "q", "", "Address or place query")
	cmd.Flags().IntVar(&limit, "limit", 5, "Max results (1-10)")
	cmd.Flags().StringVar(&language, "language", "", "Preferred language code(s)")
	cmd.Flags().StringVar(&countryCodes, "country-codes", "", "Comma-separated ISO-2 country codes")
	cmd.Flags().StringVar(&viewbox, "viewbox", "", "Bounding box: minLon,minLat,maxLon,maxLat")
	cmd.Flags().BoolVar(&bounded, "bounded", false, "Restrict results to viewbox")
	return cmd
}

func newGeocodingReverseCmd(flags *rootFlags) *cobra.Command {
	var lat, lon float64
	var language string
	var zoom int
	var addressDetails bool

	cmd := &cobra.Command{
		Use:         "reverse",
		Short:       "Reverse-geocode coordinates to an address",
		Example:     "  datpaq geocoding reverse --lat 40.7484 --lon -73.9857",
		Annotations: map[string]string{"pp:endpoint": "geocoding.reverse", "pp:method": "GET", "pp:path": "/geocoding/reverse", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlag(cmd, flags, "lat"); err != nil {
				return err
			}
			if err := requireFlag(cmd, flags, "lon"); err != nil {
				return err
			}
			params := map[string]string{
				"lat": fmt.Sprintf("%v", lat),
				"lon": fmt.Sprintf("%v", lon),
			}
			setQueryString(cmd, params, "language", "language", language)
			setQueryInt(cmd, params, "zoom", "zoom", zoom)
			setQueryBool(cmd, params, "addressdetails", "addressdetails", addressDetails)
			return printReadResponse(cmd, flags, "geocoding", "/geocoding/reverse", params)
		},
	}
	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude (-90 to 90)")
	cmd.Flags().Float64Var(&lon, "lon", 0, "Longitude (-180 to 180)")
	cmd.Flags().StringVar(&language, "language", "", "Preferred language code(s)")
	cmd.Flags().IntVar(&zoom, "zoom", 0, "Detail zoom level (0-18)")
	cmd.Flags().BoolVar(&addressDetails, "addressdetails", true, "Include structured address details")
	return cmd
}

func newGeocodingBatchCmd(flags *rootFlags) *cobra.Command {
	var queriesRaw string
	var stdinBody bool

	cmd := &cobra.Command{
		Use:         "batch",
		Short:       "Batch forward and/or reverse geocoding",
		Example:     `  datpaq geocoding batch --queries '[{"q":"Paris"},{"lat":40.71,"lon":-74.01}]'`,
		Annotations: map[string]string{"pp:endpoint": "geocoding.batch", "pp:method": "POST", "pp:path": "/geocoding/batch"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlagUnlessStdin(cmd, flags, "queries", stdinBody); err != nil {
				return err
			}
			path := "/geocoding/batch"
			var body map[string]any
			if stdinBody {
				var err error
				body, err = readStdinJSONBody()
				if err != nil {
					return err
				}
			} else {
				var queries any = []any{}
				if queriesRaw != "" {
					if err := json.Unmarshal([]byte(queriesRaw), &queries); err != nil {
						return fmt.Errorf("parsing --queries JSON: %w", err)
					}
				}
				body = map[string]any{"queries": queries}
			}
			return postMutation(cmd, flags, "geocoding", path, body)
		},
	}
	cmd.Flags().StringVar(&queriesRaw, "queries", "", "JSON array of forward {q} and/or reverse {lat,lon} items")
	addHiddenStdinFlag(cmd, &stdinBody)
	return cmd
}
