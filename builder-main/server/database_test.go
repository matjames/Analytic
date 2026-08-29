package main

import (
	"strings"
	"testing"
)

// TestFormatMonthYYYYMM tests the FormatMonthYYYYMM function
func TestFormatMonthYYYYMM(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{"1", "2024-01", "January"},
		{"6", "2024-06", "June"},
		{"12", "2024-12", "December"},
		{"0", "", "Invalid month zero"},
		{"13", "", "Invalid month 13"},
		{"invalid", "", "Non-numeric input"},
		{"-1", "", "Negative month"},
		{"", "", "Empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMonthYYYYMM(tt.input)
			if result != tt.expected {
				t.Errorf("FormatMonthYYYYMM(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatMonthAbbreviated tests the FormatMonthAbbreviated function
func TestFormatMonthAbbreviated(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		name     string
	}{
		{"1", "Jan", "January"},
		{"2", "Feb", "February"},
		{"3", "Mar", "March"},
		{"6", "Jun", "June"},
		{"12", "Dec", "December"},
		{"0", "", "Invalid month zero"},
		{"13", "", "Invalid month 13"},
		{"invalid", "", "Non-numeric input"},
		{"-5", "", "Negative month"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMonthAbbreviated(tt.input)
			if result != tt.expected {
				t.Errorf("FormatMonthAbbreviated(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGetMonthAbbreviation tests the GetMonthAbbreviation function
func TestGetMonthAbbreviation(t *testing.T) {
	tests := []struct {
		input    int
		expected string
		name     string
	}{
		{1, "Jan", "January"},
		{2, "Feb", "February"},
		{6, "Jun", "June"},
		{12, "Dec", "December"},
		{0, "", "Invalid month zero"},
		{13, "", "Invalid month 13"},
		{-1, "", "Negative month"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMonthAbbreviation(tt.input)
			if result != tt.expected {
				t.Errorf("GetMonthAbbreviation(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestApplyFiltersToSQL tests the applyFiltersToSQL function with parameterized queries
func TestApplyFiltersToSQL(t *testing.T) {
	tests := []struct {
		name              string
		sql               string
		filters           ReportFilters
		expectedSQL       string
		expectedArgsCount int
		expectedArgs      []interface{}
	}{
		{
			name:              "No filters applied",
			sql:               "SELECT * FROM patients {{district_filter}} {{year_filter}} {{month_filter}}",
			filters:           ReportFilters{},
			expectedSQL:       "SELECT * FROM patients   ",
			expectedArgsCount: 0,
			expectedArgs:      []interface{}{},
		},
		{
			name:              "District filter applied",
			sql:               "SELECT * FROM patients WHERE 1=1{{district_filter}}",
			filters:           ReportFilters{Districts: []string{"Central District"}},
			expectedSQL:       "SELECT * FROM patients WHERE 1=1 AND district = $1",
			expectedArgsCount: 1,
			expectedArgs:      []interface{}{"Central District"},
		},
		{
			name:              "District filter applied (parameterized, any value safe)",
			sql:               "SELECT * FROM patients WHERE 1=1{{district_filter}}",
			filters:           ReportFilters{Districts: []string{"InvalidDistrict"}},
			expectedSQL:       "SELECT * FROM patients WHERE 1=1 AND district = $1",
			expectedArgsCount: 1,
			expectedArgs:      []interface{}{"InvalidDistrict"},
		},
		{
			name:              "Year filter applied for cht_form_097b",
			sql:               "SELECT * FROM report.cht_form_097b WHERE 1=1{{year_filter}}",
			filters:           ReportFilters{Year: "2024"},
			expectedSQL:       "SELECT * FROM report.cht_form_097b WHERE 1=1 AND SUBSTR(period_date::text, 1, 4) = $1",
			expectedArgsCount: 1,
			expectedArgs:      []interface{}{"2024"},
		},
		{
			name:              "Month filter for cht_form_097b",
			sql:               "SELECT * FROM report.cht_form_097b WHERE 1=1{{month_filter}}",
			filters:           ReportFilters{Month: "6"},
			expectedSQL:       "SELECT * FROM report.cht_form_097b WHERE 1=1 AND TO_CHAR(period_date::date, 'Mon') = $1",
			expectedArgsCount: 1,
			expectedArgs:      []interface{}{"Jun"},
		},
		{
			name:              "Multiple filters applied",
			sql:               "SELECT * FROM report.cht_form_097b WHERE 1=1{{district_filter}}{{year_filter}}{{month_filter}}",
			filters:           ReportFilters{Districts: []string{"Jinja City"}, Year: "2024", Month: "1"},
			expectedSQL:       "SELECT * FROM report.cht_form_097b WHERE 1=1 AND district = $1 AND SUBSTR(period_date::text, 1, 4) = $2 AND TO_CHAR(period_date::date, 'Mon') = $3",
			expectedArgsCount: 3,
			expectedArgs:      []interface{}{"Jinja City", "2024", "Jan"},
		},
		{
			name:              "Invalid month number",
			sql:               "SELECT * FROM report.cht_form_097b WHERE 1=1{{month_filter}}",
			filters:           ReportFilters{Month: "13"},
			expectedSQL:       "SELECT * FROM report.cht_form_097b WHERE 1=1",
			expectedArgsCount: 0,
			expectedArgs:      []interface{}{},
		},
		{
			name:              "Invalid year filter",
			sql:               "SELECT * FROM report.cht_form_097b WHERE 1=1{{year_filter}}",
			filters:           ReportFilters{Year: "2050"},
			expectedSQL:       "SELECT * FROM report.cht_form_097b WHERE 1=1",
			expectedArgsCount: 0,
			expectedArgs:      []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a report with TimeColumns for cht_form_097b tests
			report := &Report{
				TimeColumns: TimeColumns{
					Year:  "period_date",
					Month: "period_date",
				},
			}
			resultSQL, resultArgs := applyFiltersToSQL(tt.sql, tt.filters, report)
			if resultSQL != tt.expectedSQL {
				t.Errorf("applyFiltersToSQL() SQL = %q\nwant %q", resultSQL, tt.expectedSQL)
			}
			if len(resultArgs) != tt.expectedArgsCount {
				t.Errorf("applyFiltersToSQL() args count = %d, want %d", len(resultArgs), tt.expectedArgsCount)
			}
			for i, arg := range resultArgs {
				if arg != tt.expectedArgs[i] {
					t.Errorf("applyFiltersToSQL() arg[%d] = %v, want %v", i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}

// TestLoadDBConfig tests the LoadDBConfig function
func TestLoadDBConfig(t *testing.T) {
	config, err := LoadDBConfig()
	if err != nil {
		t.Skipf("skipping: no .env available (%v)", err)
	}

	// Check that all required fields were populated
	if config.Host == "" {
		t.Error("Expected default host")
	}
	if config.Port == "" {
		t.Error("Expected default port")
	}
	if config.User == "" {
		t.Error("Expected default user")
	}
	if config.Schema == "" {
		t.Error("Expected default schema")
	}
}

// TestDistrictFilterParameterized tests that district filters are parameterized (safe from SQL injection)
// All district values are accepted - security comes from parameterized query binding, not whitelist
func TestDistrictFilterParameterized(t *testing.T) {
	testDistricts := []string{
		"Central District",
		"Jinja City",
		"InvalidDistrict",
		"'; DROP TABLE users; --", // SQL injection attempt - safe via parameterized query
		"<script>alert('xss')</script>",
	}

	sql := "SELECT * FROM patients WHERE 1=1{{district_filter}}"
	report := &Report{TimeColumns: TimeColumns{}}

	for _, district := range testDistricts {
		filters := ReportFilters{Districts: []string{district}}
		resultSQL, resultArgs := applyFiltersToSQL(sql, filters, report)

		// All non-empty districts should have filter applied via parameterized query
		if !strings.Contains(resultSQL, "AND district = $") {
			t.Errorf("Expected filter to be applied for district %q, got %q", district, resultSQL)
		}

		// Args should contain the exact district value (parameterized, safe)
		if len(resultArgs) == 0 || resultArgs[0] != district {
			t.Errorf("Expected district %q in args, got %v", district, resultArgs)
		}
	}

	// Test empty districts slice (should not apply filter)
	filters := ReportFilters{Districts: []string{}}
	resultSQL, resultArgs := applyFiltersToSQL(sql, filters, report)
	expectedNoFilter := "SELECT * FROM patients WHERE 1=1"
	if resultSQL != expectedNoFilter {
		t.Errorf("Expected no filter for empty districts slice, got %q", resultSQL)
	}
	if len(resultArgs) != 0 {
		t.Errorf("Expected no args for empty districts, got %v", resultArgs)
	}
}

// TestFilterProcessingOrder verifies the critical ordering constraint:
// applyFiltersToSQL MUST run before substituteFilters.
//
// applyFiltersToSQL converts complex placeholders like {{district_filter}} into
// parameterized SQL (e.g., "AND district = $1") with bound args.
// substituteFilters replaces simple {{key}} placeholders and strips any remaining
// complex placeholders ({{district_filter}}, {{year_filter}}, etc.) to empty string.
//
// If substituteFilters runs first, it strips the complex placeholders before
// applyFiltersToSQL can convert them — filters silently disappear.
func TestFilterProcessingOrder(t *testing.T) {
	report := &Report{
		TimeColumns: TimeColumns{
			Year: "year",
		},
	}

	sql := "SELECT * FROM report.data WHERE 1=1{{district_filter}}{{year_filter}}"
	filters := ReportFilters{
		Districts: []string{"Central District"},
		Year:      "2024",
	}
	simpleFilters := map[string]string{
		"custom_field": "some_value",
	}

	t.Run("correct_order_applyFilters_then_substitute", func(t *testing.T) {
		// Step 1: applyFiltersToSQL converts complex placeholders to parameterized SQL
		resultSQL, args := applyFiltersToSQL(sql, filters, report)

		// Step 2: substituteFilters handles any remaining simple placeholders
		resultSQL = substituteFilters(resultSQL, simpleFilters)

		// Complex placeholders should have been converted to parameterized queries
		if !strings.Contains(resultSQL, "$1") {
			t.Errorf("correct order: expected $1 parameterization, got %q", resultSQL)
		}
		if !strings.Contains(resultSQL, "AND district = $1") {
			t.Errorf("correct order: expected district filter, got %q", resultSQL)
		}
		if len(args) < 1 {
			t.Errorf("correct order: expected at least 1 arg, got %d", len(args))
		}
		if len(args) >= 1 && args[0] != "Central District" {
			t.Errorf("correct order: expected first arg 'Central District', got %v", args[0])
		}
	})

	t.Run("wrong_order_substitute_then_applyFilters", func(t *testing.T) {
		// Step 1: substituteFilters runs first — strips complex placeholders
		wrongSQL := substituteFilters(sql, simpleFilters)

		// Complex placeholders should already be gone
		if strings.Contains(wrongSQL, "{{district_filter}}") {
			t.Errorf("wrong order: substituteFilters should have removed {{district_filter}}")
		}
		if strings.Contains(wrongSQL, "{{year_filter}}") {
			t.Errorf("wrong order: substituteFilters should have removed {{year_filter}}")
		}

		// Step 2: applyFiltersToSQL has nothing to replace — filters are lost
		resultSQL, args := applyFiltersToSQL(wrongSQL, filters, report)

		// Parameterization should be absent — the placeholders were already stripped
		if strings.Contains(resultSQL, "$1") {
			t.Errorf("wrong order: expected NO parameterization (placeholders already stripped), got %q", resultSQL)
		}
		if len(args) != 0 {
			t.Errorf("wrong order: expected 0 args (filters lost), got %d: %v", len(args), args)
		}
	})
}

// ============== parseEnvLine Tests ==============

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantOk    bool
	}{
		{"standard equals", "DB_HOST=localhost", "DB_HOST", "localhost", true},
		{"equals with spaces", "  DB_HOST = localhost  ", "DB_HOST", "localhost", true},
		{"colon delimiter", "DB_HOST: localhost", "DB_HOST", "localhost", true},
		{"value with equals", "DB_URL=postgres://user:pass@host/db", "DB_URL", "postgres://user:pass@host/db", true},
		{"empty line", "", "", "", false},
		{"whitespace only", "   ", "", "", false},
		{"comment line", "# this is a comment", "", "", false},
		{"comment with space", "  # comment", "", "", false},
		{"no delimiter", "justakeyword", "", "", false},
		{"empty value", "KEY=", "KEY", "", true},
		{"empty key with equals", "=value", "", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, value, ok := parseEnvLine(tc.line)
			if ok != tc.wantOk {
				t.Errorf("parseEnvLine(%q) ok = %v, want %v", tc.line, ok, tc.wantOk)
				return
			}
			if !tc.wantOk {
				return
			}
			if key != tc.wantKey {
				t.Errorf("parseEnvLine(%q) key = %q, want %q", tc.line, key, tc.wantKey)
			}
			if value != tc.wantValue {
				t.Errorf("parseEnvLine(%q) value = %q, want %q", tc.line, value, tc.wantValue)
			}
		})
	}
}
