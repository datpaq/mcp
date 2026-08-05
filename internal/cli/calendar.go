// Copyright 2026 datpaq. Licensed under Apache-2.0. See LICENSE.
// Local patch: active ProApi service not present in the generated spec.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCalendarCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "calendar",
		Short:  "Calendar month, day, range, and holiday lookups",
		Hidden: true,
		RunE:   parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newCalendarMonthCmd(flags))
	cmd.AddCommand(newCalendarDayCmd(flags))
	cmd.AddCommand(newCalendarRangeCmd(flags))
	cmd.AddCommand(newCalendarHolidaysCmd(flags))
	return cmd
}

// calendarCommonFlags holds optional flags shared by month/day/range.
// holidays binds only the holidaysSchema subset — do not call applyShared there
// (calendar Joi validation does not strip unknown keys).
type calendarCommonFlags struct {
	weekStart            string
	timezone             string
	locale               string
	fiscalYearStartMonth int
	includeHolidays      bool
	countryCode          string
	holidaySources       string
	customHolidays       string
	outputFormat         string
	includeWeekends      bool
}

func (f *calendarCommonFlags) bindShared(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.weekStart, "week-start", "monday", "Week start day: sunday or monday")
	cmd.Flags().StringVar(&f.timezone, "timezone", "UTC", "IANA timezone")
	cmd.Flags().StringVar(&f.locale, "locale", "en-US", "BCP 47 locale")
	cmd.Flags().IntVar(&f.fiscalYearStartMonth, "fiscal-year-start-month", 1, "Fiscal year start month (1-12)")
	cmd.Flags().BoolVar(&f.includeHolidays, "include-holidays", false, "Include holiday metadata")
	cmd.Flags().StringVar(&f.countryCode, "country-code", "", "ISO-2 country code for holidays")
	cmd.Flags().StringVar(&f.holidaySources, "holiday-sources", "public", "Holiday sources: public, custom, or both")
	cmd.Flags().StringVar(&f.customHolidays, "custom-holidays", "", "Custom holidays payload when using custom sources")
}

func (f *calendarCommonFlags) applyShared(cmd *cobra.Command, params map[string]string) {
	setQueryString(cmd, params, "week-start", "week_start", f.weekStart)
	setQueryString(cmd, params, "timezone", "timezone", f.timezone)
	setQueryString(cmd, params, "locale", "locale", f.locale)
	setQueryInt(cmd, params, "fiscal-year-start-month", "fiscal_year_start_month", f.fiscalYearStartMonth)
	setQueryBool(cmd, params, "include-holidays", "include_holidays", f.includeHolidays)
	setQueryString(cmd, params, "country-code", "country_code", f.countryCode)
	setQueryString(cmd, params, "holiday-sources", "holiday_sources", f.holidaySources)
	setQueryString(cmd, params, "custom-holidays", "custom_holidays", f.customHolidays)
}

func newCalendarMonthCmd(flags *rootFlags) *cobra.Command {
	var year, month int
	var common calendarCommonFlags

	cmd := &cobra.Command{
		Use:         "month",
		Short:       "Get a calendar month grid",
		Example:     "  datpaq calendar month --year 2026 --month 8 --country-code US --include-holidays",
		Annotations: map[string]string{"pp:endpoint": "calendar.month", "pp:method": "GET", "pp:path": "/calendar/month", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlag(cmd, flags, "year"); err != nil {
				return err
			}
			if err := requireFlag(cmd, flags, "month"); err != nil {
				return err
			}
			params := map[string]string{
				"year":  fmt.Sprintf("%d", year),
				"month": fmt.Sprintf("%d", month),
			}
			common.applyShared(cmd, params)
			setQueryString(cmd, params, "output-format", "output_format", common.outputFormat)
			return printReadResponse(cmd, flags, "calendar", "/calendar/month", params)
		},
	}
	cmd.Flags().IntVar(&year, "year", 0, "Calendar year (1900-2100)")
	cmd.Flags().IntVar(&month, "month", 0, "Calendar month (1-12)")
	common.bindShared(cmd)
	cmd.Flags().StringVar(&common.outputFormat, "output-format", "nested", "Output format: flat or nested")
	return cmd
}

func newCalendarDayCmd(flags *rootFlags) *cobra.Command {
	var date string
	var common calendarCommonFlags

	cmd := &cobra.Command{
		Use:         "day",
		Short:       "Get calendar metadata for a single date",
		Example:     "  datpaq calendar day --date 2026-08-05 --country-code US --include-holidays",
		Annotations: map[string]string{"pp:endpoint": "calendar.day", "pp:method": "GET", "pp:path": "/calendar/day", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlag(cmd, flags, "date"); err != nil {
				return err
			}
			params := map[string]string{"date": date}
			common.applyShared(cmd, params)
			return printReadResponse(cmd, flags, "calendar", "/calendar/day", params)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "ISO date (YYYY-MM-DD)")
	common.bindShared(cmd)
	return cmd
}

func newCalendarRangeCmd(flags *rootFlags) *cobra.Command {
	var startDate, endDate string
	var common calendarCommonFlags

	cmd := &cobra.Command{
		Use:         "range",
		Short:       "Get calendar days across a date range",
		Example:     "  datpaq calendar range --start-date 2026-08-01 --end-date 2026-08-31",
		Annotations: map[string]string{"pp:endpoint": "calendar.range", "pp:method": "GET", "pp:path": "/calendar/range", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlag(cmd, flags, "start-date"); err != nil {
				return err
			}
			if err := requireFlag(cmd, flags, "end-date"); err != nil {
				return err
			}
			params := map[string]string{
				"start_date": startDate,
				"end_date":   endDate,
			}
			common.applyShared(cmd, params)
			setQueryBool(cmd, params, "include-weekends", "include_weekends", common.includeWeekends)
			setQueryString(cmd, params, "output-format", "output_format", common.outputFormat)
			return printReadResponse(cmd, flags, "calendar", "/calendar/range", params)
		},
	}
	cmd.Flags().StringVar(&startDate, "start-date", "", "Range start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&endDate, "end-date", "", "Range end date (YYYY-MM-DD)")
	common.bindShared(cmd)
	cmd.Flags().BoolVar(&common.includeWeekends, "include-weekends", true, "Include weekend days in the range")
	cmd.Flags().StringVar(&common.outputFormat, "output-format", "flat", "Output format: flat or nested")
	return cmd
}

func newCalendarHolidaysCmd(flags *rootFlags) *cobra.Command {
	var year, month int
	var common calendarCommonFlags

	cmd := &cobra.Command{
		Use:         "holidays",
		Short:       "List holidays for a year (optional month filter)",
		Example:     "  datpaq calendar holidays --year 2026 --country-code US",
		Annotations: map[string]string{"pp:endpoint": "calendar.holidays", "pp:method": "GET", "pp:path": "/calendar/holidays", "mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFlag(cmd, flags, "year"); err != nil {
				return err
			}
			params := map[string]string{"year": fmt.Sprintf("%d", year)}
			setQueryInt(cmd, params, "month", "month", month)
			setQueryString(cmd, params, "country-code", "country_code", common.countryCode)
			setQueryString(cmd, params, "holiday-sources", "holiday_sources", common.holidaySources)
			setQueryString(cmd, params, "custom-holidays", "custom_holidays", common.customHolidays)
			setQueryString(cmd, params, "timezone", "timezone", common.timezone)
			setQueryString(cmd, params, "locale", "locale", common.locale)
			return printReadResponse(cmd, flags, "calendar", "/calendar/holidays", params)
		},
	}
	cmd.Flags().IntVar(&year, "year", 0, "Calendar year (1900-2100)")
	cmd.Flags().IntVar(&month, "month", 0, "Optional month filter (1-12)")
	cmd.Flags().StringVar(&common.countryCode, "country-code", "", "ISO-2 country code")
	cmd.Flags().StringVar(&common.holidaySources, "holiday-sources", "public", "Holiday sources: public, custom, or both")
	cmd.Flags().StringVar(&common.customHolidays, "custom-holidays", "", "Custom holidays payload when using custom sources")
	cmd.Flags().StringVar(&common.timezone, "timezone", "UTC", "IANA timezone")
	cmd.Flags().StringVar(&common.locale, "locale", "en-US", "BCP 47 locale")
	return cmd
}
