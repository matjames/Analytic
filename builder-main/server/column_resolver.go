package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// GetTimeColumn returns the actual column name for a time dimension
// Falls back to sensible defaults if not specified in timeColumns config
func GetTimeColumn(report *Report, dimension string) string {
	if report == nil || report.TimeColumns == (TimeColumns{}) {
		// Use sensible defaults when no timeColumns config is provided
		switch dimension {
		case "year":
			return "year"
		case "month":
			return "month"
		case "week":
			return "week"
		case "quarter":
			return "quarter"
		default:
			return ""
		}
	}

	// Check if explicitly configured in timeColumns
	switch dimension {
	case "year":
		if report.TimeColumns.Year != "" {
			return report.TimeColumns.Year
		}
		return "year" // Default
	case "month":
		if report.TimeColumns.Month != "" {
			return report.TimeColumns.Month
		}
		return "month" // Default
	case "week":
		if report.TimeColumns.Week != "" {
			return report.TimeColumns.Week
		}
		return "week" // Default
	case "quarter":
		if report.TimeColumns.Quarter != "" {
			return report.TimeColumns.Quarter
		}
		return "quarter" // Default
	default:
		return ""
	}
}

// GetDistrictColumn returns the district column name
// Uses locationColumns config if specified, defaults to "district"
func GetDistrictColumn(report *Report) string {
	if report != nil && report.LocationColumns.District != "" {
		return report.LocationColumns.District
	}
	return "district"
}

// GetRegionColumn returns the region column name
// Uses locationColumns config if specified, defaults to "region"
func GetRegionColumn(report *Report) string {
	if report != nil && report.LocationColumns.Region != "" {
		return report.LocationColumns.Region
	}
	return "region"
}

// GetFacilityColumn returns the facility column name
// Uses locationColumns config if specified, defaults to "facility"
func GetFacilityColumn(report *Report) string {
	if report != nil && report.LocationColumns.Facility != "" {
		return report.LocationColumns.Facility
	}
	return "facility"
}

// DetectColumnType determines if a column is numeric, date, or string period
// This helps determine the correct SQL extraction method for time filtering
func DetectColumnType(columnName string) string {
	lower := strings.ToLower(columnName)

	// Numeric year/month/quarter columns (standalone year, month, or quarter column)
	if lower == "year" || lower == "month" || lower == "quarter" || lower == "qtr" {
		return "numeric"
	}

	// Date/timestamp columns (contains "date" or "timestamp")
	// Check before "quarter" so that "quarter_date" is correctly detected as date
	if strings.Contains(lower, "date") || strings.Contains(lower, "timestamp") {
		return "date"
	}

	// Custom quarter columns (e.g., "fiscal_quarter", "quarter_no", "my_quarter")
	// These store integer quarter numbers (1-4), not dates, so use direct equality
	if strings.Contains(lower, "quarter") {
		return "numeric"
	}

	// Custom year columns (e.g., "audit_year", "fiscal_year", "report_year")
	// These store integer years, not dates, so use direct equality
	if strings.Contains(lower, "year") {
		return "numeric"
	}

	// Custom month columns (e.g., "audit_month", "fiscal_month", "report_month")
	// These store integer months (1-12), not dates, so use direct equality
	if strings.Contains(lower, "month") {
		return "numeric"
	}

	// String period columns (like "2024W15" or "202401")
	// Contains "period" but not "date"
	if strings.Contains(lower, "period") && !strings.Contains(lower, "date") {
		return "string_period"
	}

	// Default: assume it's a date column for safe extraction
	return "date"
}

// ResolveMonthColumnType returns the effective month column type for a report.
// Checks timeColumns.monthFormat override ("text") before falling back to DetectColumnType.
func ResolveMonthColumnType(report *Report) string {
	if report != nil && strings.EqualFold(report.TimeColumns.MonthFormat, "text") {
		return "text"
	}
	monthCol := GetTimeColumn(report, "month")
	return DetectColumnType(monthCol)
}

// FormatMonthName converts a month number (1-12) to full name (January, February, ...)
func FormatMonthName(monthNum int) string {
	if monthNum < 1 || monthNum > 12 {
		return ""
	}
	names := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	return names[monthNum-1]
}

// ConvertMonthFilterToText converts a numeric month filter value (e.g. "3" or "3,4")
// to text month names (e.g. "March" or "March,April") when the report uses text months.
// Returns the original value unchanged if the report doesn't use text months.
func ConvertMonthFilterToText(monthValue string, report *Report) string {
	if ResolveMonthColumnType(report) != "text" {
		return monthValue
	}
	parts := strings.Split(monthValue, ",")
	var converted []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		num, err := strconv.Atoi(p)
		if err != nil || num < 1 || num > 12 {
			converted = append(converted, p) // keep as-is if not a valid number
			continue
		}
		converted = append(converted, FormatMonthName(num))
	}
	return strings.Join(converted, ",")
}

// LoadReportForTable finds the first report YAML that uses the given table
// This is used in handlers to get the report config (including timeColumns)
// when processing filter requests
func LoadReportForTable(tableName string) *Report {
	var match *Report
	filepath.Walk("configs", func(path string, info os.FileInfo, err error) error {
		if err != nil || match != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var report Report
		if err := yaml.Unmarshal(data, &report); err != nil {
			return nil
		}

		// Check if any component in this report uses the requested table
		for _, section := range report.Sections {
			for _, component := range section.Components {
				if component.Query != nil && component.Query.Table == tableName {
					match = &report
					return nil
				}
				if component.FilterTable == tableName {
					match = &report
					return nil
				}
				// Also check raw SQL components (table_advanced, choropleth)
				if component.SQL != "" {
					schema, table, found := extractTableFromSQL(component.SQL)
					if found && schema+"."+table == tableName {
						match = &report
						return nil
					}
				}
			}
		}
		return nil
	})

	return match
}

// LoadReportByYAMLID finds a report whose YAML id: field matches the given id.
// This is more reliable than path-based loading when the id: field does not
// mirror the directory structure (e.g. a report nested under a category folder
// whose name is not part of the id).
func LoadReportByYAMLID(id string) *Report {
	if id == "" {
		return nil
	}
	var match *Report
	filepath.Walk("configs", func(path string, info os.FileInfo, err error) error {
		if err != nil || match != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var report Report
		if err := yaml.Unmarshal(data, &report); err != nil {
			return nil
		}

		if report.ID == id {
			match = &report
		}
		return nil
	})
	return match
}
