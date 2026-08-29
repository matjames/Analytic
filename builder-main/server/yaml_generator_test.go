package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestValidateReport_Filters tests filter validation in validateReport
func TestValidateReport_Filters(t *testing.T) {
	// Helper to create a minimal valid report with given filters
	createReport := func(filters []string) *Report {
		return &Report{
			ID:          "test/report",
			Title:       "Test Report",
			Description: "Test description",
			Filters:     filters,
			Sections: []Section{
				{
					ID:     "section-1",
					Layout: "single",
					Components: []Component{
						{
							Type: "bar",
							Query: &Query{
								Table: "report.test_table",
								Aggregations: []Aggregation{
									{Column: "value", Function: "sum", Alias: "total"},
								},
							},
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name           string
		filters        []string
		expectValid    bool
		expectedErrors []string
	}{
		{
			name:        "valid filter - district",
			filters:     []string{"district"},
			expectValid: true,
		},
		{
			name:        "valid filter - year",
			filters:     []string{"year"},
			expectValid: true,
		},
		{
			name:        "valid filter - month",
			filters:     []string{"month"},
			expectValid: true,
		},
		{
			name:        "valid filter - week",
			filters:     []string{"week"},
			expectValid: true,
		},
		{
			name:        "valid filter - quarter",
			filters:     []string{"quarter"},
			expectValid: true,
		},
		{
			name:        "valid filters - multiple",
			filters:     []string{"district", "year", "month"},
			expectValid: true,
		},
		{
			name:        "valid filters - all filters",
			filters:     []string{"district", "year", "month", "week", "quarter", "region", "facility"},
			expectValid: true,
		},
		{
			name:        "valid filter - region",
			filters:     []string{"region"},
			expectValid: true,
		},
		{
			name:        "valid filter - facility",
			filters:     []string{"facility"},
			expectValid: true,
		},
		{
			name:        "valid filters - district year quarter",
			filters:     []string{"district", "year", "quarter"},
			expectValid: true,
		},
		{
			name:           "invalid filter - unknown",
			filters:        []string{"unknown"},
			expectValid:    false,
			expectedErrors: []string{"invalid filter 'unknown'"},
		},
		{
			name:           "invalid filter - typo in quarter",
			filters:        []string{"quater"},
			expectValid:    false,
			expectedErrors: []string{"invalid filter 'quater'"},
		},
		{
			name:           "invalid filter - mixed valid and invalid",
			filters:        []string{"district", "invalid", "year"},
			expectValid:    false,
			expectedErrors: []string{"invalid filter 'invalid'"},
		},
		{
			name:           "invalid filter - multiple invalid",
			filters:        []string{"foo", "bar"},
			expectValid:    false,
			expectedErrors: []string{"invalid filter 'foo'", "invalid filter 'bar'"},
		},
		{
			name:        "empty filters - valid",
			filters:     []string{},
			expectValid: true,
		},
		{
			name:        "nil filters - valid",
			filters:     nil,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := createReport(tt.filters)
			errors := validateReport(report)

			// Filter out errors not related to filters
			var filterErrors []string
			for _, err := range errors {
				if strings.Contains(err, "filter") {
					filterErrors = append(filterErrors, err)
				}
			}

			if tt.expectValid && len(filterErrors) > 0 {
				t.Errorf("Expected valid filters, got errors: %v", filterErrors)
			}

			if !tt.expectValid {
				if len(filterErrors) == 0 {
					t.Errorf("Expected filter errors, got none")
				}
				for _, expected := range tt.expectedErrors {
					found := false
					for _, err := range filterErrors {
						if strings.Contains(err, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing %q, got: %v", expected, filterErrors)
					}
				}
			}
		})
	}
}

// TestValidateReport_RequiredFields tests validation of required fields
func TestValidateReport_RequiredFields(t *testing.T) {
	tests := []struct {
		name           string
		report         *Report
		expectedErrors []string
	}{
		{
			name: "missing id",
			report: &Report{
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: "bar", Query: &Query{Table: "t", Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "a"}}}},
						},
					},
				},
			},
			expectedErrors: []string{"report id is required"},
		},
		{
			name: "missing title",
			report: &Report{
				ID:          "test/report",
				Description: "Desc",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: "bar", Query: &Query{Table: "t", Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "a"}}}},
						},
					},
				},
			},
			expectedErrors: []string{"report title is required"},
		},
		// Note: description is now optional - keywords can be used instead for search discoverability
		{
			name: "missing sections",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections:    []Section{},
			},
			expectedErrors: []string{"at least one section is required"},
		},
		{
			name: "missing section id",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						Layout: "single",
						Components: []Component{
							{Type: "bar", Query: &Query{Table: "t", Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "a"}}}},
						},
					},
				},
			},
			expectedErrors: []string{"section 0: id is required"},
		},
		{
			name: "missing section layout",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID: "s1",
						Components: []Component{
							{Type: "bar", Query: &Query{Table: "t", Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "a"}}}},
						},
					},
				},
			},
			expectedErrors: []string{"section 0: layout is required"},
		},
		{
			name: "missing components",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID:         "s1",
						Layout:     "single",
						Components: []Component{},
					},
				},
			},
			expectedErrors: []string{"section 0: at least one component is required"},
		},
		{
			name: "missing component type",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Query: &Query{Table: "t", Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "a"}}}},
						},
					},
				},
			},
			expectedErrors: []string{"section 0, component 0: type is required"},
		},
		{
			name: "text component missing content",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: "text"},
						},
					},
				},
			},
			expectedErrors: []string{"section 0, component 0: content is required for text type"},
		},
		{
			name: "non-text component missing query",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: "bar"},
						},
					},
				},
			},
			expectedErrors: []string{"section 0, component 0: query is required for bar type"},
		},
		{
			name: "query missing table",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: "bar", Query: &Query{Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "a"}}}},
						},
					},
				},
			},
			expectedErrors: []string{"section 0, component 0: query table is required"},
		},
		{
			name: "query missing aggregations",
			report: &Report{
				ID:          "test/report",
				Title:       "Test",
				Description: "Desc",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: "bar", Query: &Query{Table: "t", Aggregations: []Aggregation{}}},
						},
					},
				},
			},
			expectedErrors: []string{"section 0, component 0: at least one aggregation is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateReport(tt.report)

			for _, expected := range tt.expectedErrors {
				found := false
				for _, err := range errors {
					if strings.Contains(err, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing %q, got: %v", expected, errors)
				}
			}
		})
	}
}

// TestValidateReport_ValidReport tests a fully valid report passes validation
func TestValidateReport_ValidReport(t *testing.T) {
	report := &Report{
		ID:          "test/report",
		Title:       "Test Report",
		Description: "A test report description",
		Filters:     []string{"district", "year", "quarter"},
		Sections: []Section{
			{
				ID:     "section-1",
				Title:  "Section 1",
				Layout: "single",
				Components: []Component{
					{
						Type: "bar",
						Query: &Query{
							Table: "report.test_table",
							Aggregations: []Aggregation{
								{Column: "value", Function: "sum", Alias: "total"},
							},
							GroupBy: []GroupBy{
								{Field: "quarter", Format: "quarter_year"},
							},
						},
					},
				},
			},
		},
	}

	errors := validateReport(report)
	if len(errors) > 0 {
		t.Errorf("Expected valid report, got errors: %v", errors)
	}
}

// TestExtractFilename tests the extractFilename function
func TestExtractFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test/report", "test/report"},
		{"disease-surveillance/malaria-report", "disease-surveillance/malaria-report"},
		{"simple-report", "simple-report"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractFilename(tt.input)
			if result != tt.expected {
				t.Errorf("extractFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestYAMLRoundtrip_PeriodLimitForTable tests that periodLimit is preserved in YAML roundtrip for table components
func TestYAMLRoundtrip_PeriodLimitForTable(t *testing.T) {
	report := &Report{
		ID:          "test/table-period-limit",
		Title:       "Test Table with Period Limit",
		Description: "Tests periodLimit in table YAML generation",
		Filters:     []string{"district", "year", "month"},
		Sections: []Section{
			{
				ID:     "section-1",
				Title:  "Monthly Data Table",
				Layout: "single",
				Components: []Component{
					{
						Type:  "table",
						Title: "Monthly Statistics",
						Query: &Query{
							Table: "report.test_table",
							Aggregations: []Aggregation{
								{Column: "value", Function: "sum", Alias: "Total"},
							},
							GroupBy: []GroupBy{
								{Field: "month"}, // No format - relies on field name inference
							},
							PeriodLimit: 6, // Show last 6 months
						},
					},
				},
			},
		},
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal report to YAML: %v", err)
	}

	yamlStr := string(yamlBytes)

	// Verify YAML contains periodLimit
	if !strings.Contains(yamlStr, "periodLimit: 6") {
		t.Errorf("Expected YAML to contain 'periodLimit: 6', got:\n%s", yamlStr)
	}

	// Unmarshal back to report
	var parsedReport Report
	err = yaml.Unmarshal(yamlBytes, &parsedReport)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Verify periodLimit is preserved
	if len(parsedReport.Sections) == 0 || len(parsedReport.Sections[0].Components) == 0 {
		t.Fatal("Parsed report has no sections or components")
	}

	query := parsedReport.Sections[0].Components[0].Query
	if query == nil {
		t.Fatal("Parsed component has no query")
	}

	if query.PeriodLimit != 6 {
		t.Errorf("Expected periodLimit=6, got %d", query.PeriodLimit)
	}

	// Verify groupBy field is preserved (used for period type inference)
	if len(query.GroupBy) == 0 {
		t.Fatal("Parsed query has no groupBy")
	}
	if query.GroupBy[0].Field != "month" {
		t.Errorf("Expected groupBy field='month', got %q", query.GroupBy[0].Field)
	}
}

// TestPeriodTypeInference_TableWithPeriodLimit tests that period type is correctly inferred for tables
func TestPeriodTypeInference_TableWithPeriodLimit(t *testing.T) {
	tests := []struct {
		name         string
		groupBy      []GroupBy
		periodLimit  int
		expectedType string
	}{
		{
			name:         "table with month field",
			groupBy:      []GroupBy{{Field: "month"}},
			periodLimit:  6,
			expectedType: "month",
		},
		{
			name:         "table with week field",
			groupBy:      []GroupBy{{Field: "week"}},
			periodLimit:  4,
			expectedType: "week",
		},
		{
			name:         "table with quarter field",
			groupBy:      []GroupBy{{Field: "quarter"}},
			periodLimit:  4,
			expectedType: "quarter",
		},
		{
			name:         "table with district and month fields",
			groupBy:      []GroupBy{{Field: "district"}, {Field: "month"}},
			periodLimit:  6,
			expectedType: "month",
		},
		{
			name:         "chart style with format",
			groupBy:      []GroupBy{{Field: "period_date", Format: "month"}},
			periodLimit:  6,
			expectedType: "month",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			periodType := GetPeriodTypeFromGroupBy(tt.groupBy)
			if periodType != tt.expectedType {
				t.Errorf("GetPeriodTypeFromGroupBy() = %q, want %q", periodType, tt.expectedType)
			}
		})
	}
}

// TestExtractTableFromSQL tests the table extraction from SQL
func TestExtractTableFromSQL(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		wantSchema string
		wantTable  string
		wantFound  bool
	}{
		{
			name:       "schema.table format",
			sql:        "SELECT * FROM report.mv_data",
			wantSchema: "report",
			wantTable:  "mv_data",
			wantFound:  true,
		},
		{
			name:       "table only - defaults to report schema",
			sql:        "SELECT * FROM mv_data",
			wantSchema: "report",
			wantTable:  "mv_data",
			wantFound:  true,
		},
		{
			name:       "with alias AS",
			sql:        "SELECT * FROM report.mv_data AS d WHERE d.year = 2024",
			wantSchema: "report",
			wantTable:  "mv_data",
			wantFound:  true,
		},
		{
			name:       "with alias no AS",
			sql:        "SELECT * FROM report.mv_data d WHERE d.year = 2024",
			wantSchema: "report",
			wantTable:  "mv_data",
			wantFound:  true,
		},
		{
			name:       "case insensitive",
			sql:        "select * from Report.MV_Data where active = true",
			wantSchema: "Report",
			wantTable:  "MV_Data",
			wantFound:  true,
		},
		{
			name:       "with CTE",
			sql:        "WITH summary AS (SELECT * FROM report.mv_data) SELECT * FROM summary",
			wantSchema: "report",
			wantTable:  "mv_data",
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table, found := extractTableFromSQL(tt.sql)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if schema != tt.wantSchema {
				t.Errorf("schema = %q, want %q", schema, tt.wantSchema)
			}
			if table != tt.wantTable {
				t.Errorf("table = %q, want %q", table, tt.wantTable)
			}
		})
	}
}

// TestDetectHardcodedFilters tests detection of hardcoded filter values
func TestDetectHardcodedFilters(t *testing.T) {
	tests := []struct {
		name          string
		sql           string
		columnMapping map[string]string
		wantDetected  []string // filter names that should be detected
	}{
		{
			name: "numeric year",
			sql:  "SELECT * FROM data WHERE year = 2024",
			columnMapping: map[string]string{
				"year": "year",
			},
			wantDetected: []string{"year"},
		},
		{
			name: "string district",
			sql:  "SELECT * FROM data WHERE district = 'Kampala'",
			columnMapping: map[string]string{
				"district": "district",
			},
			wantDetected: []string{"district"},
		},
		{
			name: "mapped column name",
			sql:  "SELECT * FROM data WHERE d = 'Kampala' AND yr = 2024",
			columnMapping: map[string]string{
				"district": "d",
				"year":     "yr",
			},
			wantDetected: []string{"district", "year"},
		},
		{
			name: "IN clause",
			sql:  "SELECT * FROM data WHERE district IN ('Kampala', 'Central')",
			columnMapping: map[string]string{
				"district": "district",
			},
			wantDetected: []string{"district"},
		},
		{
			name: "BETWEEN clause",
			sql:  "SELECT * FROM data WHERE year BETWEEN 2020 AND 2024",
			columnMapping: map[string]string{
				"year": "year",
			},
			wantDetected: []string{"year"},
		},
		{
			name: "no hardcoded values",
			sql:  "SELECT * FROM data WHERE active = true",
			columnMapping: map[string]string{
				"district": "district",
				"year":     "year",
			},
			wantDetected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := detectHardcodedFilters(tt.sql, tt.columnMapping)

			// Check if all expected filters were detected
			detected := make(map[string]bool)
			for _, m := range matches {
				detected[m.filterName] = true
			}

			for _, want := range tt.wantDetected {
				if !detected[want] {
					t.Errorf("Expected to detect hardcoded filter %q, but didn't. Matches: %+v", want, matches)
				}
			}

			// Check no extra filters were detected
			if len(matches) != len(tt.wantDetected) {
				t.Errorf("Expected %d matches, got %d. Matches: %+v", len(tt.wantDetected), len(matches), matches)
			}
		})
	}
}

// TestProcessAdvancedSQL tests the full SQL processing pipeline
// The new behavior wraps SQL in a CTE with complex filter placeholders
// that generate proper SQL at runtime (handling date extraction, parameterization)
func TestProcessAdvancedSQL(t *testing.T) {
	tests := []struct {
		name             string
		sql              string
		filters          []string
		timeColumns      TimeColumns
		locationColumns  LocationColumns
		wantContain      []string
		wantNotContain   []string
		wantReplacements int // number of replacements (removals) expected
	}{
		{
			name:    "wraps SQL with year filter placeholder",
			sql:     "SELECT * FROM report.mv_data WHERE year = 2024",
			filters: []string{"year"},
			wantContain: []string{
				"_filtered_source",
				"{{year_filter}}",
				"WHERE 1=1",
			},
			wantNotContain: []string{
				"year = 2024",
			},
			wantReplacements: 1, // removed hardcoded year
		},
		{
			name:    "wraps SQL with district filter placeholder",
			sql:     "SELECT * FROM report.mv_data WHERE district = 'Kampala'",
			filters: []string{"district"},
			wantContain: []string{
				"_filtered_source",
				"{{district_filter}}",
				"WHERE 1=1",
			},
			wantNotContain: []string{
				"district = 'Kampala'",
			},
			wantReplacements: 1, // removed hardcoded district
		},
		{
			name:    "uses column mapping for year filter",
			sql:     "SELECT * FROM report.mv_data WHERE period_date = 2024",
			filters: []string{"year"},
			timeColumns: TimeColumns{
				Year: "period_date",
			},
			wantContain: []string{
				"_filtered_source",
				"{{year_filter}}",
			},
			wantNotContain: []string{
				"period_date = 2024",
			},
			wantReplacements: 1,
		},
		{
			name:    "uses column mapping for district filter",
			sql:     "SELECT * FROM report.mv_data WHERE d = 'Kampala'",
			filters: []string{"district"},
			locationColumns: LocationColumns{
				District: "d",
			},
			wantContain: []string{
				"_filtered_source",
				"{{district_filter}}",
			},
			wantNotContain: []string{
				"d = 'Kampala'",
			},
			wantReplacements: 1,
		},
		{
			name:             "SQL already has complex placeholders - no change",
			sql:              "SELECT * FROM report.data WHERE 1=1 {{district_filter}}",
			filters:          []string{"district", "year"},
			wantContain:      []string{"{{district_filter}}"},
			wantNotContain:   []string{"_filtered_source"},
			wantReplacements: 0,
		},
		{
			name:             "empty filters - no change",
			sql:              "SELECT * FROM report.data WHERE year = 2024",
			filters:          []string{},
			wantContain:      []string{"year = 2024"},
			wantNotContain:   []string{"{{"},
			wantReplacements: 0,
		},
		{
			name: "complex SQL with multiple hardcoded values - all removed and wrapped",
			sql: `SELECT district, SUM(cases)
FROM report.mv_data
WHERE year = 2024 AND district = 'Kampala' AND month = 6
GROUP BY district`,
			filters: []string{"year", "district", "month"},
			wantContain: []string{
				"_filtered_source",
				"{{year_filter}}",
				"{{district_filter}}",
				"{{month_filter}}",
				"WHERE 1=1",
			},
			wantNotContain: []string{
				"year = 2024",
				"district = 'Kampala'",
				"month = 6",
			},
			wantReplacements: 3, // removed all three hardcoded values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, info := processAdvancedSQL(tt.sql, tt.filters, tt.timeColumns, tt.locationColumns, 0)

			for _, want := range tt.wantContain {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain %q\nGot:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("Expected result NOT to contain %q\nGot:\n%s", notWant, result)
				}
			}

			if len(info.Replacements) != tt.wantReplacements {
				t.Errorf("Expected %d replacements, got %d: %v", tt.wantReplacements, len(info.Replacements), info.Replacements)
			}
		})
	}
}

// TestProcessAdvancedSQL_DetectedTable tests that table is correctly detected
func TestProcessAdvancedSQL_DetectedTable(t *testing.T) {
	sql := "SELECT * FROM report.mv_097b_data WHERE year = 2024"
	filters := []string{"year"}

	_, info := processAdvancedSQL(sql, filters, TimeColumns{}, LocationColumns{}, 0)

	if info.DetectedTable != "report.mv_097b_data" {
		t.Errorf("Expected detected table 'report.mv_097b_data', got %q", info.DetectedTable)
	}
}

// TestGetColumnMapping tests the column mapping function
func TestGetColumnMapping(t *testing.T) {
	filters := []string{"year", "month", "district", "region"}
	timeColumns := TimeColumns{
		Year:  "period_date",
		Month: "period_date",
	}
	locationColumns := LocationColumns{
		District: "d",
	}

	mapping := getColumnMapping(filters, timeColumns, locationColumns)

	// Check mapped values
	if mapping["year"] != "period_date" {
		t.Errorf("year mapping = %q, want 'period_date'", mapping["year"])
	}
	if mapping["month"] != "period_date" {
		t.Errorf("month mapping = %q, want 'period_date'", mapping["month"])
	}
	if mapping["district"] != "d" {
		t.Errorf("district mapping = %q, want 'd'", mapping["district"])
	}
	// region should default to "region" since not specified
	if mapping["region"] != "region" {
		t.Errorf("region mapping = %q, want 'region'", mapping["region"])
	}
}

// TestValidateReport_ChartWithSQL tests that chart types with SQL pass validation
func TestValidateReport_ChartWithSQL(t *testing.T) {
	chartTypes := []string{"bar", "line", "bar_line", "pie"}

	for _, ct := range chartTypes {
		// Chart with SQL and no query → valid
		t.Run(ct+" with sql", func(t *testing.T) {
			report := &Report{
				ID:    "test/chart-sql",
				Title: "Test Chart SQL",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: ct, SQL: "SELECT month, SUM(cases) FROM report.data GROUP BY month"},
						},
					},
				},
			}
			errors := validateReport(report)
			for _, err := range errors {
				if strings.Contains(err, "query is required") {
					t.Errorf("chart type %q with SQL should not require query, got error: %s", ct, err)
				}
			}
		})

		// Chart with query and no sql → valid (existing behavior)
		t.Run(ct+" with query", func(t *testing.T) {
			query := &Query{
				Table:        "report.test_table",
				Aggregations: []Aggregation{{Column: "value", Function: "sum", Alias: "total"}},
			}
			report := &Report{
				ID:    "test/chart-query",
				Title: "Test Chart Query",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: ct, Query: query},
						},
					},
				},
			}
			errors := validateReport(report)
			for _, err := range errors {
				if strings.Contains(err, "query is required") || strings.Contains(err, "sql is required") {
					t.Errorf("chart type %q with query should be valid, got error: %s", ct, err)
				}
			}
		})

		// Chart with neither sql nor query → error
		t.Run(ct+" with neither", func(t *testing.T) {
			report := &Report{
				ID:    "test/chart-neither",
				Title: "Test Chart Neither",
				Sections: []Section{
					{
						ID:     "s1",
						Layout: "single",
						Components: []Component{
							{Type: ct},
						},
					},
				},
			}
			errors := validateReport(report)
			found := false
			for _, err := range errors {
				if strings.Contains(err, "query is required") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("chart type %q with neither sql nor query should error, got: %v", ct, errors)
			}
		})
	}
}

// TestValidateComponentForYAML_ChartSQL tests component-level validation for chart SQL
func TestValidateComponentForYAML_ChartSQL(t *testing.T) {
	chartTypes := []string{"bar", "line", "bar_line", "pie"}

	for _, ct := range chartTypes {
		// Chart with SQL → no error
		t.Run(ct+" with sql", func(t *testing.T) {
			comp := Component{Type: ct, SQL: "SELECT * FROM report.data"}
			errMsg := validateComponentForYAML(comp, 0, 0)
			if errMsg != "" {
				t.Errorf("expected no error for %s with SQL, got: %s", ct, errMsg)
			}
		})

		// Chart with neither → error
		t.Run(ct+" with neither", func(t *testing.T) {
			comp := Component{Type: ct}
			errMsg := validateComponentForYAML(comp, 0, 0)
			if errMsg == "" {
				t.Errorf("expected error for %s with neither sql nor query", ct)
			}
			if !strings.Contains(errMsg, "query or sql is required") {
				t.Errorf("expected 'query or sql is required' for %s, got: %s", ct, errMsg)
			}
		})
	}
}

// TestYAMLRoundtrip_ChartWithSQL tests YAML marshal/unmarshal preserves chart SQL
func TestYAMLRoundtrip_ChartWithSQL(t *testing.T) {
	report := &Report{
		ID:      "test/bar-sql",
		Title:   "Bar SQL Test",
		Filters: []string{"district", "year"},
		Sections: []Section{
			{
				ID:     "s1",
				Title:  "Section 1",
				Layout: "single",
				Components: []Component{
					{
						Type:  "bar",
						Title: "Cases by Month",
						SQL:   "SELECT month, SUM(cases) as cases FROM report.data GROUP BY month ORDER BY month",
					},
				},
			},
		},
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	yamlStr := string(yamlBytes)

	// Verify YAML contains expected fields
	if !strings.Contains(yamlStr, "sql:") {
		t.Errorf("YAML should contain 'sql:', got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "type: bar") {
		t.Errorf("YAML should contain 'type: bar', got:\n%s", yamlStr)
	}

	// Unmarshal back
	var parsed Report
	if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// SQL preserved, Query is nil
	comp := parsed.Sections[0].Components[0]
	if comp.SQL == "" {
		t.Error("SQL field should be preserved after roundtrip")
	}
	if comp.SQL != report.Sections[0].Components[0].SQL {
		t.Errorf("SQL mismatch: got %q, want %q", comp.SQL, report.Sections[0].Components[0].SQL)
	}
	if comp.Query != nil {
		t.Error("Query should be nil for SQL-only component")
	}

	// Validate parsed report → no errors about missing query
	errors := validateReport(&parsed)
	for _, e := range errors {
		if strings.Contains(e, "query is required") {
			t.Errorf("Parsed report should pass validation, got: %s", e)
		}
	}
}

// TestYAMLRoundtrip_PieWithSQL tests pie chart SQL roundtrip
func TestYAMLRoundtrip_PieWithSQL(t *testing.T) {
	report := &Report{
		ID:      "test/pie-sql",
		Title:   "Pie SQL Test",
		Filters: []string{"year"},
		Sections: []Section{
			{
				ID:     "s1",
				Title:  "Distribution",
				Layout: "single",
				Components: []Component{
					{
						Type:  "pie",
						Title: "Gender Distribution",
						SQL:   "SELECT SUM(male) as Male, SUM(female) as Female FROM report.data",
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(report)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed Report
	if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	comp := parsed.Sections[0].Components[0]
	if comp.SQL != report.Sections[0].Components[0].SQL {
		t.Errorf("SQL mismatch after roundtrip: got %q", comp.SQL)
	}
	if comp.Type != "pie" {
		t.Errorf("Type mismatch: got %q, want pie", comp.Type)
	}
}

// TestProcessAdvancedSQL_ChartTypes tests that processAdvancedSQL works for chart SQL
func TestProcessAdvancedSQL_ChartTypes(t *testing.T) {
	// Bar SQL with hardcoded year → wrapped in CTE, year removed
	t.Run("bar sql with hardcoded year", func(t *testing.T) {
		sql := "SELECT month, SUM(cases) FROM report.mv_data WHERE year = 2024 GROUP BY month"
		result, info := processAdvancedSQL(sql, []string{"year"}, TimeColumns{}, LocationColumns{}, 0)

		if !strings.Contains(result, "_filtered_source") {
			t.Errorf("Expected CTE wrapper, got:\n%s", result)
		}
		if !strings.Contains(result, "{{year_filter}}") {
			t.Errorf("Expected year_filter placeholder, got:\n%s", result)
		}
		if strings.Contains(result, "year = 2024") {
			t.Errorf("Hardcoded year should be removed, got:\n%s", result)
		}
		if len(info.Replacements) != 1 {
			t.Errorf("Expected 1 replacement, got %d", len(info.Replacements))
		}
	})

	// Pie SQL with hardcoded district → wrapped in CTE, district removed
	t.Run("pie sql with hardcoded district", func(t *testing.T) {
		sql := "SELECT SUM(male) as Male, SUM(female) as Female FROM report.mv_data WHERE district = 'Kampala'"
		result, info := processAdvancedSQL(sql, []string{"district"}, TimeColumns{}, LocationColumns{}, 0)

		if !strings.Contains(result, "_filtered_source") {
			t.Errorf("Expected CTE wrapper, got:\n%s", result)
		}
		if !strings.Contains(result, "{{district_filter}}") {
			t.Errorf("Expected district_filter placeholder, got:\n%s", result)
		}
		if strings.Contains(result, "district = 'Kampala'") {
			t.Errorf("Hardcoded district should be removed, got:\n%s", result)
		}
		if len(info.Replacements) != 1 {
			t.Errorf("Expected 1 replacement, got %d", len(info.Replacements))
		}
	})
}

// TestValidateReport_ChartWithSQLNotRequireQuery tests that a full report with chart SQL passes validateReport
func TestValidateReport_ChartWithSQLNotRequireQuery(t *testing.T) {
	report := &Report{
		ID:      "test/multi-chart-sql",
		Title:   "Multi Chart SQL Test",
		Filters: []string{"district", "year"},
		Sections: []Section{
			{
				ID:     "s1",
				Title:  "Charts",
				Layout: "two-column",
				Components: []Component{
					{Type: "bar", Title: "Bar Chart", SQL: "SELECT month, SUM(x) FROM report.data GROUP BY month"},
					{Type: "line", Title: "Line Chart", SQL: "SELECT month, SUM(y) FROM report.data GROUP BY month"},
				},
			},
			{
				ID:     "s2",
				Title:  "More Charts",
				Layout: "two-column",
				Components: []Component{
					{Type: "pie", Title: "Pie Chart", SQL: "SELECT SUM(a) as A, SUM(b) as B FROM report.data"},
				},
			},
		},
	}

	errors := validateReport(report)
	for _, err := range errors {
		if strings.Contains(err, "query is required") {
			t.Errorf("Report with chart SQL should not require query, got: %s", err)
		}
	}
}

// ============================================================
// Handler-level tests for GenerateYAMLHandler
// ============================================================

// TestGenerateYAMLHandler_MethodNotAllowed tests that non-POST methods are rejected
func TestGenerateYAMLHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/report/generate", nil)
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

// TestGenerateYAMLHandler_InvalidJSON tests that malformed JSON returns 400
func TestGenerateYAMLHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/report/generate", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// TestGenerateYAMLHandler_MissingTitle tests validation for missing title
func TestGenerateYAMLHandler_MissingTitle(t *testing.T) {
	body := YAMLGenerateRequest{
		ID: "test/missing-title",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{Type: "bar", Query: &Query{Table: "report.data", Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}}}},
				},
			},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/report/generate", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing title, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "title is required") {
		t.Errorf("Expected 'title is required' error, got: %s", w.Body.String())
	}
}

// TestGenerateYAMLHandler_InvalidReportID tests validation for bad report IDs
func TestGenerateYAMLHandler_InvalidReportID(t *testing.T) {
	body := YAMLGenerateRequest{
		ID:    "../../../etc/passwd",
		Title: "Exploit Attempt",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{Type: "bar", Query: &Query{Table: "report.data", Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}}}},
				},
			},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/report/generate", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid report ID, got %d", w.Code)
	}
}

// TestGenerateYAMLHandler_ValidBarChart tests successful YAML generation for a bar chart
func TestGenerateYAMLHandler_ValidBarChart(t *testing.T) {
	body := YAMLGenerateRequest{
		ID:      "test/bar-chart",
		Title:   "Bar Chart Report",
		Filters: []string{"district", "year"},
		Sections: []Section{
			{
				ID:     "section-1",
				Title:  "Section One",
				Layout: "single",
				Components: []Component{
					{
						Type:  "bar",
						Title: "Cases by Month",
						Query: &Query{
							Table:        "report.mv_data",
							Aggregations: []Aggregation{{Column: "cases", Function: "sum", Alias: "Total Cases"}},
							GroupBy:      []GroupBy{{Field: "period_date", Format: "month"}},
						},
					},
				},
			},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/report/generate", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp YAMLGenerateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify YAML output contains expected fields
	if resp.YAML == "" {
		t.Fatal("Expected non-empty YAML")
	}
	if !strings.Contains(resp.YAML, "title: Bar Chart Report") {
		t.Errorf("YAML missing title, got:\n%s", resp.YAML)
	}
	if !strings.Contains(resp.YAML, "type: bar") {
		t.Errorf("YAML missing component type, got:\n%s", resp.YAML)
	}
	if !strings.Contains(resp.YAML, "table: report.mv_data") {
		t.Errorf("YAML missing query table, got:\n%s", resp.YAML)
	}
	if resp.Filename != "test/bar-chart.yaml" {
		t.Errorf("Expected filename 'test/bar-chart.yaml', got %q", resp.Filename)
	}
	if resp.SavePath != "configs/test/bar-chart.yaml" {
		t.Errorf("Expected savePath 'configs/test/bar-chart.yaml', got %q", resp.SavePath)
	}
}

// TestGenerateYAMLHandler_TableAdvancedSQLProcessing tests that table_advanced SQL gets processed
func TestGenerateYAMLHandler_TableAdvancedSQLProcessing(t *testing.T) {
	body := YAMLGenerateRequest{
		ID:      "test/advanced-table",
		Title:   "Advanced Table",
		Filters: []string{"year", "district"},
		Sections: []Section{
			{
				ID:     "s1",
				Title:  "Data",
				Layout: "single",
				Components: []Component{
					{
						Type: "table_advanced",
						SQL:  "SELECT district, SUM(cases) FROM report.mv_data WHERE year = 2024 AND district = 'Kampala' GROUP BY district",
					},
				},
			},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/report/generate", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp YAMLGenerateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	// SQL should have been processed: hardcoded values removed, CTE wrapper added
	if !strings.Contains(resp.YAML, "_filtered_source") {
		t.Errorf("Expected CTE wrapper in generated YAML, got:\n%s", resp.YAML)
	}
	if strings.Contains(resp.YAML, "year = 2024") {
		t.Errorf("Hardcoded year should be removed from generated YAML")
	}
	if strings.Contains(resp.YAML, "'Kampala'") {
		t.Errorf("Hardcoded district should be removed from generated YAML")
	}

	// Check SQL processing info
	if len(resp.SQLProcessed) != 1 {
		t.Fatalf("Expected 1 SQL processing info, got %d", len(resp.SQLProcessed))
	}
	if resp.SQLProcessed[0].DetectedTable != "report.mv_data" {
		t.Errorf("Expected detected table 'report.mv_data', got %q", resp.SQLProcessed[0].DetectedTable)
	}
	if len(resp.SQLProcessed[0].Replacements) != 2 {
		t.Errorf("Expected 2 replacements, got %d", len(resp.SQLProcessed[0].Replacements))
	}
}

func TestGenerateYAMLHandler_TablePageSizePersists(t *testing.T) {
	body := YAMLGenerateRequest{
		ID:    "test/advanced-table-page-size",
		Title: "Advanced Table Page Size",
		Sections: []Section{
			{
				ID:     "s1",
				Title:  "Data",
				Layout: "single",
				Components: []Component{
					{
						Type:     "table_advanced",
						SQL:      "SELECT district, SUM(cases) FROM report.mv_data GROUP BY district",
						PageSize: 5,
					},
				},
			},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/report/generate", bytes.NewReader(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp YAMLGenerateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !strings.Contains(resp.YAML, "pageSize: 5") {
		t.Fatalf("Expected generated YAML to include pageSize: 5, got:\n%s", resp.YAML)
	}

	var parsed Report
	if err := yaml.Unmarshal([]byte(resp.YAML), &parsed); err != nil {
		t.Fatalf("Failed to parse generated YAML: %v", err)
	}
	got := parsed.Sections[0].Components[0].PageSize
	if got != 5 {
		t.Fatalf("Expected parsed page size 5, got %d", got)
	}
}

// TestGenerateYAMLHandler_OptionsReturns200 tests CORS preflight
func TestGenerateYAMLHandler_OptionsReturns200(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/api/report/generate", nil)
	w := httptest.NewRecorder()
	GenerateYAMLHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for OPTIONS, got %d", w.Code)
	}
}

// ============================================================
// Full roundtrip tests: builder JSON → YAML → parse → verify
// ============================================================

// TestRoundtrip_BarChartWithQuery tests bar chart roundtrip preserves all fields
func TestRoundtrip_BarChartWithQuery(t *testing.T) {
	original := Report{
		ID:              "test/roundtrip-bar",
		Title:           "Roundtrip Bar",
		Description:     "Test description",
		Keywords:        "malaria,cases",
		Filters:         []string{"district", "year", "month"},
		TimeColumns:     TimeColumns{Year: "period_year", Month: "period_month"},
		LocationColumns: LocationColumns{District: "district_name"},
		Sections: []Section{
			{
				ID:     "s1",
				Title:  "Section 1",
				Layout: "two-column",
				Components: []Component{
					{
						Type:  "bar",
						Title: "Monthly Cases",
						Query: &Query{
							Table:        "report.mv_data",
							Aggregations: []Aggregation{{Column: "cases", Function: "sum", Alias: "Total"}},
							GroupBy:      []GroupBy{{Field: "period_date", Format: "month"}},
						},
						ReferenceLines: []ReferenceLineConfig{
							{Value: 15000, Label: "Target", Color: "#22c55e", Style: "dashed"},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Core fields
	if parsed.ID != original.ID {
		t.Errorf("ID: got %q, want %q", parsed.ID, original.ID)
	}
	if parsed.Title != original.Title {
		t.Errorf("Title: got %q, want %q", parsed.Title, original.Title)
	}
	if parsed.Description != original.Description {
		t.Errorf("Description: got %q, want %q", parsed.Description, original.Description)
	}
	if parsed.Keywords != original.Keywords {
		t.Errorf("Keywords: got %q, want %q", parsed.Keywords, original.Keywords)
	}

	// Filters
	if len(parsed.Filters) != 3 {
		t.Fatalf("Filters: got %d, want 3", len(parsed.Filters))
	}
	for i, f := range original.Filters {
		if parsed.Filters[i] != f {
			t.Errorf("Filter[%d]: got %q, want %q", i, parsed.Filters[i], f)
		}
	}

	// TimeColumns / LocationColumns
	if parsed.TimeColumns.Year != "period_year" {
		t.Errorf("TimeColumns.Year: got %q, want 'period_year'", parsed.TimeColumns.Year)
	}
	if parsed.TimeColumns.Month != "period_month" {
		t.Errorf("TimeColumns.Month: got %q, want 'period_month'", parsed.TimeColumns.Month)
	}
	if parsed.LocationColumns.District != "district_name" {
		t.Errorf("LocationColumns.District: got %q, want 'district_name'", parsed.LocationColumns.District)
	}

	// Component
	comp := parsed.Sections[0].Components[0]
	if comp.Type != "bar" {
		t.Errorf("Type: got %q, want 'bar'", comp.Type)
	}
	if comp.Query == nil {
		t.Fatal("Query is nil after roundtrip")
	}
	if comp.Query.Table != "report.mv_data" {
		t.Errorf("Query.Table: got %q", comp.Query.Table)
	}
	if len(comp.Query.Aggregations) != 1 || comp.Query.Aggregations[0].Alias != "Total" {
		t.Errorf("Aggregation alias mismatch")
	}
	if len(comp.Query.GroupBy) != 1 || comp.Query.GroupBy[0].Format != "month" {
		t.Errorf("GroupBy format mismatch")
	}

	// ReferenceLines
	if len(comp.ReferenceLines) != 1 {
		t.Fatalf("ReferenceLines: got %d, want 1", len(comp.ReferenceLines))
	}
	if comp.ReferenceLines[0].Value != 15000 {
		t.Errorf("ReferenceLine value: got %f, want 15000", comp.ReferenceLines[0].Value)
	}
	if comp.ReferenceLines[0].Label != "Target" {
		t.Errorf("ReferenceLine label: got %q", comp.ReferenceLines[0].Label)
	}

	// Validate
	errs := validateReport(&parsed)
	if len(errs) > 0 {
		t.Errorf("Parsed report should be valid, got errors: %v", errs)
	}
}

// TestRoundtrip_TextComponentWithQuery tests text KPI component roundtrip
func TestRoundtrip_TextComponentWithQuery(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-text",
		Title: "KPI Report",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type:    "text",
						Title:   "Total Cases",
						Content: "<h2>{{value}}</h2>",
						Query: &Query{
							Table:        "report.mv_data",
							Aggregations: []Aggregation{{Column: "cases", Function: "sum", Alias: "value"}},
						},
						Target: &TargetConfig{Value: 80, Label: "Goal"},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	comp := parsed.Sections[0].Components[0]
	if comp.Content != "<h2>{{value}}</h2>" {
		t.Errorf("Content: got %q", comp.Content)
	}
	if comp.Target == nil {
		t.Fatal("Target is nil after roundtrip")
	}
	if comp.Target.Value != 80 {
		t.Errorf("Target.Value: got %f, want 80", comp.Target.Value)
	}
	if comp.Target.Label != "Goal" {
		t.Errorf("Target.Label: got %q, want 'Goal'", comp.Target.Label)
	}
}

// TestRoundtrip_PlainTextComponent tests text component without query
func TestRoundtrip_PlainTextComponent(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-plain-text",
		Title: "Plain Text Report",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type:    "text",
						Title:   "Instructions",
						Content: "<p>This report shows malaria data.</p>",
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	comp := parsed.Sections[0].Components[0]
	if comp.Type != "text" {
		t.Errorf("Type: got %q", comp.Type)
	}
	if comp.Query != nil {
		t.Error("Plain text should have nil query")
	}
	if comp.Content != "<p>This report shows malaria data.</p>" {
		t.Errorf("Content mismatch: got %q", comp.Content)
	}

	errs := validateReport(&parsed)
	if len(errs) > 0 {
		t.Errorf("Valid plain text should pass validation: %v", errs)
	}
}

// TestRoundtrip_ChoroplethWithColumnMapping tests choropleth SQL component roundtrip
func TestRoundtrip_ChoroplethWithColumnMapping(t *testing.T) {
	original := Report{
		ID:      "test/roundtrip-choropleth",
		Title:   "Map Report",
		Filters: []string{"year"},
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type:  "choropleth",
						Title: "Cases by District",
						SQL:   "SELECT district, SUM(cases) as value FROM report.mv_data GROUP BY district",
						ColumnMapping: &ColumnMapping{
							District: "district",
							Value:    "value",
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	comp := parsed.Sections[0].Components[0]
	if comp.Type != "choropleth" {
		t.Errorf("Type: got %q", comp.Type)
	}
	if comp.SQL == "" {
		t.Error("SQL should be preserved")
	}
	if comp.ColumnMapping == nil {
		t.Fatal("ColumnMapping is nil after roundtrip")
	}
	if comp.ColumnMapping.District != "district" {
		t.Errorf("ColumnMapping.District: got %q", comp.ColumnMapping.District)
	}
	if comp.ColumnMapping.Value != "value" {
		t.Errorf("ColumnMapping.Value: got %q", comp.ColumnMapping.Value)
	}

	errs := validateReport(&parsed)
	if len(errs) > 0 {
		t.Errorf("Valid choropleth should pass validation: %v", errs)
	}
}

// TestRoundtrip_CalculateFormula tests that calculate field survives roundtrip
func TestRoundtrip_CalculateFormula(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-calculate",
		Title: "Calculate Report",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type: "bar",
						Query: &Query{
							Table: "report.mv_data",
							Aggregations: []Aggregation{
								{Column: "positive", Function: "sum", Alias: "positive"},
								{Column: "tested", Function: "sum", Alias: "tested"},
							},
							GroupBy: []GroupBy{{Field: "period_date", Format: "month"}},
							Calculate: CalculateList{
								{Formula: "(positive / tested * 100)", RoundTo: intPtr(1)},
							},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if !strings.Contains(string(yamlBytes), "positive / tested * 100") {
		t.Errorf("YAML should contain calculate formula, got:\n%s", string(yamlBytes))
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	query := parsed.Sections[0].Components[0].Query
	if query == nil {
		t.Fatal("Query is nil")
	}
	if len(query.Calculate) != 1 {
		t.Fatalf("Calculate: got %d, want 1", len(query.Calculate))
	}
	if !strings.Contains(query.Calculate[0].Formula, "positive / tested * 100") {
		t.Errorf("Calculate formula: got %q", query.Calculate[0].Formula)
	}
	if query.Calculate[0].RoundTo == nil || *query.Calculate[0].RoundTo != 1 {
		t.Errorf("Calculate roundTo: want 1")
	}
}

// TestRoundtrip_CustomFilters tests custom filters survive roundtrip
func TestRoundtrip_CustomFilters(t *testing.T) {
	original := Report{
		ID:      "test/roundtrip-custom-filters",
		Title:   "Custom Filters Report",
		Filters: []string{"district", "year"},
		CustomFilters: []CustomFilterDef{
			{Column: "facility_type", Table: "report.mv_data", Label: "Facility Type", Type: "select", DefaultValue: "HC III"},
			{Column: "age_group", Table: "report.mv_data", Label: "Age Group", Type: "multiselect"},
		},
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{Type: "bar", Query: &Query{Table: "report.mv_data", Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}}}},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	if len(parsed.CustomFilters) != 2 {
		t.Fatalf("CustomFilters: got %d, want 2", len(parsed.CustomFilters))
	}
	if parsed.CustomFilters[0].Column != "facility_type" {
		t.Errorf("CustomFilters[0].Column: got %q", parsed.CustomFilters[0].Column)
	}
	if parsed.CustomFilters[0].Label != "Facility Type" {
		t.Errorf("CustomFilters[0].Label: got %q", parsed.CustomFilters[0].Label)
	}
	if parsed.CustomFilters[0].DefaultValue != "HC III" {
		t.Errorf("CustomFilters[0].DefaultValue: got %q", parsed.CustomFilters[0].DefaultValue)
	}
	if parsed.CustomFilters[1].Type != "multiselect" {
		t.Errorf("CustomFilters[1].Type: got %q", parsed.CustomFilters[1].Type)
	}
}

// TestRoundtrip_MultiSectionMixedTypes tests a multi-section report with different component types
func TestRoundtrip_MultiSectionMixedTypes(t *testing.T) {
	original := Report{
		ID:      "test/roundtrip-multi",
		Title:   "Multi Section Report",
		Filters: []string{"district", "year", "month"},
		Sections: []Section{
			{
				ID:     "kpis",
				Title:  "KPIs",
				Layout: "four-column",
				Components: []Component{
					{Type: "text", Title: "KPI 1", Content: "<h2>{{value}}</h2>", Query: &Query{Table: "report.mv_data", Aggregations: []Aggregation{{Column: "c1", Function: "sum", Alias: "value"}}}},
					{Type: "text", Title: "KPI 2", Content: "<h2>{{value}}</h2>", Query: &Query{Table: "report.mv_data", Aggregations: []Aggregation{{Column: "c2", Function: "sum", Alias: "value"}}}},
				},
			},
			{
				ID:     "charts",
				Title:  "Charts",
				Layout: "two-column",
				Components: []Component{
					{Type: "bar", Title: "Bar", Query: &Query{Table: "report.mv_data", Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}}, GroupBy: []GroupBy{{Field: "period_date", Format: "month"}}}},
					{Type: "line", Title: "Line", Query: &Query{Table: "report.mv_data", Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}}, GroupBy: []GroupBy{{Field: "period_date", Format: "month"}}}},
				},
			},
			{
				ID:     "table",
				Title:  "Data Table",
				Layout: "single",
				Components: []Component{
					{Type: "table", Title: "Summary", Query: &Query{Table: "report.mv_data", Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "Total"}}, GroupBy: []GroupBy{{Field: "district"}}}},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	if len(parsed.Sections) != 3 {
		t.Fatalf("Sections: got %d, want 3", len(parsed.Sections))
	}
	if len(parsed.Sections[0].Components) != 2 {
		t.Errorf("Section 0 components: got %d, want 2", len(parsed.Sections[0].Components))
	}
	if parsed.Sections[0].Layout != "four-column" {
		t.Errorf("Section 0 layout: got %q, want 'four-column'", parsed.Sections[0].Layout)
	}
	if parsed.Sections[1].Components[0].Type != "bar" {
		t.Errorf("Section 1 comp 0 type: got %q", parsed.Sections[1].Components[0].Type)
	}
	if parsed.Sections[1].Components[1].Type != "line" {
		t.Errorf("Section 1 comp 1 type: got %q", parsed.Sections[1].Components[1].Type)
	}
	if parsed.Sections[2].Components[0].Type != "table" {
		t.Errorf("Section 2 comp 0 type: got %q", parsed.Sections[2].Components[0].Type)
	}

	errs := validateReport(&parsed)
	if len(errs) > 0 {
		t.Errorf("Valid multi-section report should pass validation: %v", errs)
	}
}

// TestRoundtrip_BarLineCombo tests bar_line combo chart roundtrip
func TestRoundtrip_BarLineCombo(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-bar-line",
		Title: "Combo Chart",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type: "bar_line",
						Query: &Query{
							Table: "report.mv_data",
							Aggregations: []Aggregation{
								{Column: "cases", Function: "sum", Alias: "Cases", ChartType: "bar"},
								{Column: "rate", Function: "avg", Alias: "Rate", ChartType: "line", Color: "#ff0000"},
							},
							GroupBy: []GroupBy{{Field: "period_date", Format: "month"}},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	comp := parsed.Sections[0].Components[0]
	if comp.Type != "bar_line" {
		t.Errorf("Type: got %q", comp.Type)
	}
	if len(comp.Query.Aggregations) != 2 {
		t.Fatalf("Aggregations: got %d, want 2", len(comp.Query.Aggregations))
	}
	if comp.Query.Aggregations[0].ChartType != "bar" {
		t.Errorf("Agg[0].ChartType: got %q, want 'bar'", comp.Query.Aggregations[0].ChartType)
	}
	if comp.Query.Aggregations[1].ChartType != "line" {
		t.Errorf("Agg[1].ChartType: got %q, want 'line'", comp.Query.Aggregations[1].ChartType)
	}
	if comp.Query.Aggregations[1].Color != "#ff0000" {
		t.Errorf("Agg[1].Color: got %q, want '#ff0000'", comp.Query.Aggregations[1].Color)
	}
}

// TestRoundtrip_TableWithHeatmap tests table heatmap config roundtrip
func TestRoundtrip_TableWithHeatmap(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-heatmap",
		Title: "Heatmap Table",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type: "table",
						Query: &Query{
							Table:        "report.mv_data",
							Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "Total"}},
							GroupBy:      []GroupBy{{Field: "district"}},
						},
						Heatmap: &HeatmapConfig{
							Rules: []HeatmapRule{
								{Column: "Total", Thresholds: []float64{50, 100}, Colors: "red-yellow-green"},
							},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	comp := parsed.Sections[0].Components[0]
	if comp.Heatmap == nil {
		t.Fatal("Heatmap is nil after roundtrip")
	}
	if len(comp.Heatmap.Rules) != 1 {
		t.Fatalf("Heatmap rules: got %d, want 1", len(comp.Heatmap.Rules))
	}
	if comp.Heatmap.Rules[0].Column != "Total" {
		t.Errorf("Rule column: got %q", comp.Heatmap.Rules[0].Column)
	}
	if len(comp.Heatmap.Rules[0].Thresholds) != 2 {
		t.Errorf("Rule thresholds: got %d, want 2", len(comp.Heatmap.Rules[0].Thresholds))
	}
	if comp.Heatmap.Rules[0].Colors != "red-yellow-green" {
		t.Errorf("Rule colors: got %q", comp.Heatmap.Rules[0].Colors)
	}
}

// TestRoundtrip_StackedHorizontalBar tests bar chart options roundtrip
func TestRoundtrip_StackedHorizontalBar(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-stacked",
		Title: "Stacked Bar",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type:       "bar",
						Stacked:    true,
						Horizontal: true,
						ShowValues: true,
						Query: &Query{
							Table:        "report.mv_data",
							Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}},
							GroupBy:      []GroupBy{{Field: "district"}},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	comp := parsed.Sections[0].Components[0]
	if !comp.Stacked {
		t.Error("Stacked should be true after roundtrip")
	}
	if !comp.Horizontal {
		t.Error("Horizontal should be true after roundtrip")
	}
	if !comp.ShowValues {
		t.Error("ShowValues should be true after roundtrip")
	}
}

// TestRoundtrip_LineWithTrendLine tests line chart trendLine roundtrip
func TestRoundtrip_LineWithTrendLine(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-trend",
		Title: "Trend Line Chart",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type:           "line",
						TrendLine:      true,
						TrendLineColor: "#3b82f6",
						Query: &Query{
							Table:        "report.mv_data",
							Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}},
							GroupBy:      []GroupBy{{Field: "period_date", Format: "month"}},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	comp := parsed.Sections[0].Components[0]
	if !comp.TrendLine {
		t.Error("TrendLine should be true after roundtrip")
	}
	if comp.TrendLineColor != "#3b82f6" {
		t.Errorf("TrendLineColor: got %q, want '#3b82f6'", comp.TrendLineColor)
	}
}

// TestRoundtrip_QueryWhereClause tests that query where clause survives roundtrip
func TestRoundtrip_QueryWhereClause(t *testing.T) {
	original := Report{
		ID:    "test/roundtrip-where",
		Title: "Where Clause Report",
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type: "bar",
						Query: &Query{
							Table:        "report.mv_data",
							Where:        "age_group = 'under_5'",
							Aggregations: []Aggregation{{Column: "v", Function: "sum", Alias: "a"}},
							GroupBy:      []GroupBy{{Field: "period_date", Format: "month"}},
						},
					},
				},
			},
		},
	}

	yamlBytes, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Report
	yaml.Unmarshal(yamlBytes, &parsed)

	if parsed.Sections[0].Components[0].Query.Where != "age_group = 'under_5'" {
		t.Errorf("Where clause: got %q", parsed.Sections[0].Components[0].Query.Where)
	}
}

// TestCleanupWhereClause tests SQL cleanup after filter removal
func TestCleanupWhereClause(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "WHERE AND → WHERE",
			input:    "SELECT * FROM t WHERE AND x = 1",
			expected: "SELECT * FROM t WHERE x = 1",
		},
		{
			name:     "AND AND → AND",
			input:    "SELECT * FROM t WHERE x = 1 AND AND y = 2",
			expected: "SELECT * FROM t WHERE x = 1 AND y = 2",
		},
		{
			name:     "trailing WHERE before GROUP BY",
			input:    "SELECT * FROM t WHERE GROUP BY x",
			expected: "SELECT * FROM t GROUP BY x",
		},
		{
			name:     "trailing AND before ORDER BY",
			input:    "SELECT * FROM t WHERE x = 1 AND ORDER BY x",
			expected: "SELECT * FROM t WHERE x = 1 ORDER BY x",
		},
		{
			name:     "no cleanup needed",
			input:    "SELECT * FROM t WHERE x = 1 AND y = 2 GROUP BY x",
			expected: "SELECT * FROM t WHERE x = 1 AND y = 2 GROUP BY x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanupWhereClause(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
