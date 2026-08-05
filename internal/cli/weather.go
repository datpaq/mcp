// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWeatherCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "weather",
		Short:  "Current conditions and forecasts by latitude/longitude",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newWeatherCurrentCmd(flags))
	cmd.AddCommand(newWeatherForecastCmd(flags))
	return cmd
}

func newWeatherCurrentCmd(flags *rootFlags) *cobra.Command {
	var lat, lon float64
	var units string

	cmd := &cobra.Command{
		Use:         "current",
		Short:       "Get current weather conditions for a location",
		Example:     "  datpaq weather current --lat 40.71 --lon -74.01 --units fahrenheit",
		Annotations: map[string]string{"pp:endpoint": "weather.current", "pp:method": "GET", "pp:path": "/weather/current", "mcp:read-only": "true"},
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
			setQueryString(cmd, params, "units", "units", units)
			return printReadResponse(cmd, flags, "weather", "/weather/current", params)
		},
	}
	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude (-90 to 90)")
	cmd.Flags().Float64Var(&lon, "lon", 0, "Longitude (-180 to 180)")
	cmd.Flags().StringVar(&units, "units", "auto", "Temperature units: auto, celsius, or fahrenheit")
	return cmd
}

func newWeatherForecastCmd(flags *rootFlags) *cobra.Command {
	var lat, lon float64
	var units string
	var forecastDays int

	cmd := &cobra.Command{
		Use:         "forecast",
		Short:       "Get weather forecast for a location",
		Example:     "  datpaq weather forecast --lat 40.71 --lon -74.01 --forecast-days 5",
		Annotations: map[string]string{"pp:endpoint": "weather.forecast", "pp:method": "GET", "pp:path": "/weather/forecast", "mcp:read-only": "true"},
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
			setQueryString(cmd, params, "units", "units", units)
			setQueryInt(cmd, params, "forecast-days", "forecastDays", forecastDays)
			return printReadResponse(cmd, flags, "weather", "/weather/forecast", params)
		},
	}
	cmd.Flags().Float64Var(&lat, "lat", 0, "Latitude (-90 to 90)")
	cmd.Flags().Float64Var(&lon, "lon", 0, "Longitude (-180 to 180)")
	cmd.Flags().StringVar(&units, "units", "auto", "Temperature units: auto, celsius, or fahrenheit")
	cmd.Flags().IntVar(&forecastDays, "forecast-days", 7, "Forecast period in days (1-14)")
	return cmd
}
