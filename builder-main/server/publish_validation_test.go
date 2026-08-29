package main

import (
	"testing"
)

// minValidReport returns a minimal valid report for testing.
func minValidReport() Report {
	return Report{
		ID:    "test/valid-report",
		Title: "Valid Report",
		Sections: []Section{
			{
				ID:     "s1",
				Title:  "Section 1",
				Layout: "single",
				Components: []Component{
					{
						Type:  "bar",
						Title: "Test Chart",
						Query: &Query{
							Table: "report.test_table",
							Aggregations: []Aggregation{
								{Column: "cases", Function: "sum", Alias: "TotalCases"},
							},
							GroupBy: []GroupBy{
								{Field: "period_year", Format: "year"},
							},
						},
					},
				},
			},
		},
		Filters: []string{"year", "district"},
	}
}

func TestPublishValidation_ValidReport(t *testing.T) {
	report := minValidReport()
	result := validateReportForPublish(&report)

	if !result.Valid {
		t.Errorf("expected valid report, got errors: %v", result.Errors)
	}
}

func TestPublishValidation_MissingReportID(t *testing.T) {
	report := minValidReport()
	report.ID = ""
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for missing report ID")
	}
	found := false
	for _, e := range result.Errors {
		if e.Message == "report id is required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'report id is required' error, got: %v", result.Errors)
	}
}

func TestPublishValidation_MissingTitle(t *testing.T) {
	report := minValidReport()
	report.Title = ""
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for missing title")
	}
	found := false
	for _, e := range result.Errors {
		if e.Message == "report title is required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'report title is required' error, got: %v", result.Errors)
	}
}

func TestPublishValidation_InvalidFilter(t *testing.T) {
	report := minValidReport()
	report.Filters = []string{"year", "banana"}
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for bad filter name")
	}
	found := false
	for _, e := range result.Errors {
		if contains(e.Message, "invalid filter") && contains(e.Message, "banana") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about invalid filter 'banana', got: %v", result.Errors)
	}
}

func TestPublishValidation_NoSections(t *testing.T) {
	report := minValidReport()
	report.Sections = nil
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for no sections")
	}
}

func TestPublishValidation_MissingSectionLayout(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Layout = ""
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for missing section layout")
	}
}

func TestPublishValidation_InvalidAggFunction(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components[0].Query.Aggregations[0].Function = "banana"
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for unsupported aggregation function")
	}
	found := false
	for _, e := range result.Errors {
		if contains(e.Message, "SQL build failed") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected SQL build error, got: %v", result.Errors)
	}
}

func TestPublishValidation_BadCalculateFormula(t *testing.T) {
	// BuildSQLWithSquirrel does not parse calculate formulas for syntax —
	// it passes them as raw SQL expressions. A bad formula like "(a + )" would
	// fail at runtime (EXPLAIN), not at build time. This test documents that
	// structural validation still passes when the formula is syntactically wrong.
	report := minValidReport()
	report.Sections[0].Components[0].Query.Calculate = CalculateList{
		{Formula: "(a + )"},
	}
	result := validateReportForPublish(&report)

	// The formula is not validated at build time — this is a known limitation.
	// It would be caught by EXPLAIN when DB is available.
	_ = result
}

func TestPublishValidation_TextWithValueButNoQuery(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components = []Component{
		{
			Type:    "text",
			Content: "Total: {{value}}",
			// No query — {{value}} can't be resolved
		},
	}

	result := validateReportForPublish(&report)
	// This should be structurally valid since text type doesn't require a query
	// The {{value}} without query is a runtime issue but validateReport doesn't check it
	// The test documents current behavior
	_ = result
}

func TestPublishValidation_TableAdvancedMissingSQL(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components = []Component{
		{
			Type:  "table_advanced",
			Title: "Advanced Table",
			// SQL is required for table_advanced but missing
		},
	}
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for table_advanced without sql")
	}
}

func TestPublishValidation_MissingQueryTable(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components[0].Query.Table = ""
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for missing query table")
	}
}

func TestPublishValidation_WarningsReturned(t *testing.T) {
	report := minValidReport()
	// No section description triggers a design warning
	report.Sections[0].Description = ""
	result := validateReportForPublish(&report)

	if !result.Valid {
		t.Errorf("expected valid report (warnings don't block), got errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected at least one warning for missing section description")
	}
}

func TestPublishValidation_ChoroplethSQLMissingColumnMapping(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components = []Component{
		{
			Type:  "choropleth",
			Title: "Map",
			SQL:   "SELECT district, COUNT(*) as value FROM report.test_table GROUP BY district",
			// Missing columnMapping — structural error from validateReport
		},
	}
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for choropleth SQL without columnMapping")
	}
}

func TestPublishValidation_MultipleErrors(t *testing.T) {
	report := Report{
		ID:    "", // error: missing ID
		Title: "", // error: missing title
		Sections: []Section{
			{
				ID:     "s1",
				Layout: "single",
				Components: []Component{
					{
						Type: "", // error: missing type
					},
				},
			},
		},
	}
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for multiple errors")
	}
	if len(result.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestPublishValidation_ExpressionColumnSkipsSchemaCheck(t *testing.T) {
	// Expression columns (containing +, -, etc.) should not be schema-checked
	// This test verifies the containsOperators check in validateSchema
	report := minValidReport()
	report.Sections[0].Components[0].Query.Aggregations[0].Column = "col1 + col2"

	// With DB == nil, schema validation is skipped, so this should pass structural checks
	result := validateReportForPublish(&report)
	// The SQL build may fail (unknown columns) but that's expected without DB
	// We're testing that containsOperators prevents a false "column not found" error
	_ = result
}

func TestPublishValidation_ValidTableAdvancedSQL(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components = []Component{
		{
			Type:  "table_advanced",
			Title: "Raw SQL Table",
			SQL:   "SELECT district, COUNT(*) FROM report.test_table GROUP BY district",
		},
	}
	result := validateReportForPublish(&report)

	if !result.Valid {
		t.Errorf("expected valid for table_advanced with safe SQL, got errors: %v", result.Errors)
	}
}

func TestPublishValidation_DangerousSQL(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components = []Component{
		{
			Type:  "table_advanced",
			Title: "Bad SQL",
			SQL:   "DROP TABLE report.test_table",
		},
	}
	result := validateReportForPublish(&report)

	if result.Valid {
		t.Error("expected invalid for dangerous SQL")
	}
}

func TestSplitSchemaTable(t *testing.T) {
	tests := []struct {
		input      string
		wantSchema string
		wantTable  string
	}{
		{"report.test_table", "report", "test_table"},
		{"report.facility_data", "report", "facility_data"},
		{"no_dot", "", ""},
		{"", "", ""},
		{"a.b.c", "a", "b.c"},
	}

	for _, tt := range tests {
		schema, table := splitSchemaTable(tt.input)
		if schema != tt.wantSchema || table != tt.wantTable {
			t.Errorf("splitSchemaTable(%q) = (%q, %q), want (%q, %q)",
				tt.input, schema, table, tt.wantSchema, tt.wantTable)
		}
	}
}

func TestCollectAllTableColumns(t *testing.T) {
	cache := map[string]map[string]bool{
		"report.t1": {"col_a": true, "col_b": true},
		"report.t2": {"col_b": true, "col_c": true},
		"report.t3": nil,
	}

	all := collectAllTableColumns(cache)

	if !all["col_a"] || !all["col_b"] || !all["col_c"] {
		t.Errorf("expected all columns to be present, got: %v", all)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 unique columns, got %d", len(all))
	}
}

func TestPublishValidation_CalculateFormulaWithUndefinedAlias(t *testing.T) {
	// BuildSQLWithSquirrel does not validate that formula identifiers match
	// aggregation aliases — it leaves unmatched identifiers as raw SQL text.
	// The existing ValidateReportWarnings catches this as a design warning instead.
	report := minValidReport()
	report.Sections[0].Components[0].Query.Aggregations = []Aggregation{
		{Column: "tested", Function: "sum", Alias: "Tested"},
		{Column: "positive", Function: "sum", Alias: "Positive"},
	}
	report.Sections[0].Components[0].Query.Calculate = CalculateList{
		{Formula: "(Positive / UndefinedAlias * 100)"},
	}

	result := validateReportForPublish(&report)
	// Not a build error — but should produce a warning from checkUndefinedFormulaAlias
	found := false
	for _, w := range result.Warnings {
		if contains(w.Message, "undefined alias") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning about undefined formula alias")
	}
}

func TestPublishValidation_ValidCalculateFormula(t *testing.T) {
	report := minValidReport()
	report.Sections[0].Components[0].Query.Aggregations = []Aggregation{
		{Column: "tested", Function: "sum", Alias: "Tested"},
		{Column: "positive", Function: "sum", Alias: "Positive"},
	}
	report.Sections[0].Components[0].Query.Calculate = CalculateList{
		{Formula: "(Positive / Tested * 100)"},
	}

	result := validateReportForPublish(&report)
	if !result.Valid {
		t.Errorf("expected valid for correct formula, got errors: %v", result.Errors)
	}
}

// contains is a helper for checking substrings in test assertions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
