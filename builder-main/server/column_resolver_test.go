package main

import (
	"testing"
)

// TestGetTimeColumnWithDefaultsNil tests that sensible defaults are used when TimeColumns is nil
func TestGetTimeColumnWithDefaultsNil(t *testing.T) {
	report := &Report{TimeColumns: TimeColumns{}}

	tests := []struct {
		dimension string
		expected  string
	}{
		{"year", "year"},
		{"month", "month"},
		{"week", "week"},
		{"invalid", ""},
	}

	for _, tc := range tests {
		t.Run(tc.dimension, func(t *testing.T) {
			result := GetTimeColumn(report, tc.dimension)
			if result != tc.expected {
				t.Errorf("GetTimeColumn(%q) = %q, expected %q", tc.dimension, result, tc.expected)
			}
		})
	}
}

// TestGetTimeColumnWithConfigured tests that configured columns are returned
func TestGetTimeColumnWithConfigured(t *testing.T) {
	report := &Report{
		TimeColumns: TimeColumns{
			Year:  "period_date",
			Month: "period_date",
			Week:  "period",
		},
	}

	tests := []struct {
		dimension string
		expected  string
	}{
		{"year", "period_date"},
		{"month", "period_date"},
		{"week", "period"},
	}

	for _, tc := range tests {
		t.Run(tc.dimension, func(t *testing.T) {
			result := GetTimeColumn(report, tc.dimension)
			if result != tc.expected {
				t.Errorf("GetTimeColumn(%q) = %q, expected %q", tc.dimension, result, tc.expected)
			}
		})
	}
}

// TestGetTimeColumnPartiallyConfigured tests fallback to defaults for unconfigured dimensions
func TestGetTimeColumnPartiallyConfigured(t *testing.T) {
	report := &Report{
		TimeColumns: TimeColumns{
			Year: "period_date",
			// Month and Week are not configured
		},
	}

	tests := []struct {
		dimension string
		expected  string
	}{
		{"year", "period_date"},  // Configured, should return configured value
		{"month", "month"},         // Not configured, should use default
		{"week", "week"},           // Not configured, should use default
	}

	for _, tc := range tests {
		t.Run(tc.dimension, func(t *testing.T) {
			result := GetTimeColumn(report, tc.dimension)
			if result != tc.expected {
				t.Errorf("GetTimeColumn(%q) = %q, expected %q", tc.dimension, result, tc.expected)
			}
		})
	}
}

// TestGetTimeColumnNilReport tests behavior with nil report
func TestGetTimeColumnNilReport(t *testing.T) {
	result := GetTimeColumn(nil, "year")
	if result != "year" {
		t.Errorf("GetTimeColumn(nil, \"year\") = %q, expected \"year\"", result)
	}
}

// TestGetDistrictColumn tests district column resolution
func TestGetDistrictColumn(t *testing.T) {
	tests := []struct {
		name     string
		report   *Report
		expected string
	}{
		{"nil report", nil, "district"},
		{"empty report", &Report{}, "district"},
		{"report with other data", &Report{ID: "test"}, "district"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetDistrictColumn(tc.report)
			if result != tc.expected {
				t.Errorf("GetDistrictColumn() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

// TestDetectColumnTypeNumeric tests detection of numeric year/month/quarter columns
func TestDetectColumnTypeNumeric(t *testing.T) {
	tests := []string{
		"year",
		"Year",
		"YEAR",
		"month",
		"Month",
		"MONTH",
		"quarter",
		"Quarter",
		"QUARTER",
		// Common abbreviation for quarter
		"qtr",
		"Qtr",
		"QTR",
		// Custom quarter column names (contain "quarter", no "date")
		"fiscal_quarter",
		"quarter_no",
		"my_quarter",
		"reporting_quarter",
		"quarter_number",
		// Custom year column names (contain "year", no "date")
		"audit_year",
		"fiscal_year",
		"report_year",
		"period_year",
		// Custom month column names (contain "month", no "date")
		"audit_month",
		"fiscal_month",
		"report_month",
		"period_month",
	}

	for _, col := range tests {
		t.Run(col, func(t *testing.T) {
			result := DetectColumnType(col)
			if result != "numeric" {
				t.Errorf("DetectColumnType(%q) = %q, expected \"numeric\"", col, result)
			}
		})
	}
}

// TestDetectColumnTypeDate tests detection of date columns
func TestDetectColumnTypeDate(t *testing.T) {
	tests := []string{
		"period_date",
		"period_date",
		"report_date",
		"created_date",
		"timestamp",
		"created_timestamp",
	}

	for _, col := range tests {
		t.Run(col, func(t *testing.T) {
			result := DetectColumnType(col)
			if result != "date" {
				t.Errorf("DetectColumnType(%q) = %q, expected \"date\"", col, result)
			}
		})
	}
}

// TestDetectColumnTypeStringPeriod tests detection of string period columns
func TestDetectColumnTypeStringPeriod(t *testing.T) {
	tests := []string{
		"period",
		"Period",
		"iso_period",
		"reporting_period",
	}

	for _, col := range tests {
		t.Run(col, func(t *testing.T) {
			result := DetectColumnType(col)
			if result != "string_period" {
				t.Errorf("DetectColumnType(%q) = %q, expected \"string_period\"", col, result)
			}
		})
	}
}

// TestDetectColumnTypeDefault tests that unknown column names default to date
func TestDetectColumnTypeDefault(t *testing.T) {
	tests := []string{
		"unknown_column",
		"data_col",
		"value",
	}

	for _, col := range tests {
		t.Run(col, func(t *testing.T) {
			result := DetectColumnType(col)
			if result != "date" {
				t.Errorf("DetectColumnType(%q) = %q, expected \"date\" (default)", col, result)
			}
		})
	}
}

// TestDetectColumnTypePeriodDateEdgeCase tests that period_date is detected as date, not string_period
func TestDetectColumnTypePeriodDateEdgeCase(t *testing.T) {
	// period_date contains "period" but also contains "date", so should be detected as date
	result := DetectColumnType("period_date")
	if result != "date" {
		t.Errorf("DetectColumnType(\"period_date\") = %q, expected \"date\"", result)
	}
}

// TestDetectColumnTypeQuarterDateEdgeCase tests that quarter_date is detected as date, not numeric
func TestDetectColumnTypeQuarterDateEdgeCase(t *testing.T) {
	// quarter_date contains "quarter" but also "date" — should still be detected as date
	result := DetectColumnType("quarter_date")
	if result != "date" {
		t.Errorf("DetectColumnType(\"quarter_date\") = %q, expected \"date\"", result)
	}
}

// TestGetRegionColumn tests region column resolution
func TestGetRegionColumn(t *testing.T) {
	tests := []struct {
		name     string
		report   *Report
		expected string
	}{
		{"nil report", nil, "region"},
		{"empty report", &Report{}, "region"},
		{"no location columns", &Report{ID: "test"}, "region"},
		{"custom region column", &Report{
			LocationColumns: LocationColumns{Region: "region_name"},
		}, "region_name"},
		{"other location columns set", &Report{
			LocationColumns: LocationColumns{District: "dist_name"},
		}, "region"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetRegionColumn(tc.report)
			if result != tc.expected {
				t.Errorf("GetRegionColumn() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

// TestGetFacilityColumn tests facility column resolution
func TestGetFacilityColumn(t *testing.T) {
	tests := []struct {
		name     string
		report   *Report
		expected string
	}{
		{"nil report", nil, "facility"},
		{"empty report", &Report{}, "facility"},
		{"no location columns", &Report{ID: "test"}, "facility"},
		{"custom facility column", &Report{
			LocationColumns: LocationColumns{Facility: "facility_name"},
		}, "facility_name"},
		{"other location columns set", &Report{
			LocationColumns: LocationColumns{District: "dist_name"},
		}, "facility"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetFacilityColumn(tc.report)
			if result != tc.expected {
				t.Errorf("GetFacilityColumn() = %q, expected %q", result, tc.expected)
			}
		})
	}
}

// TestGetDistrictColumnWithCustomColumn tests custom district column configuration
func TestGetDistrictColumnWithCustomColumn(t *testing.T) {
	report := &Report{
		LocationColumns: LocationColumns{District: "district_name"},
	}
	result := GetDistrictColumn(report)
	if result != "district_name" {
		t.Errorf("GetDistrictColumn() = %q, expected %q", result, "district_name")
	}
}

// TestGetTimeColumnQuarterDimension tests quarter dimension resolution
func TestGetTimeColumnQuarterDimension(t *testing.T) {
	tests := []struct {
		name     string
		report   *Report
		expected string
	}{
		{"nil report defaults", nil, "quarter"},
		{"empty config defaults", &Report{TimeColumns: TimeColumns{}}, "quarter"},
		{"configured quarter", &Report{
			TimeColumns: TimeColumns{Quarter: "period_quarter"},
		}, "period_quarter"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := GetTimeColumn(tc.report, "quarter")
			if result != tc.expected {
				t.Errorf("GetTimeColumn(report, \"quarter\") = %q, expected %q", result, tc.expected)
			}
		})
	}
}
