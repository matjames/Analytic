package main

import (
	"strings"
	"testing"
)

// TestBuildSQLSimpleAggregation tests simple aggregation query generation
func TestBuildSQLSimpleAggregation(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "num_fever_u5_male + num_fever_u5_female",
				Function: "sum",
				Alias:    "value",
			},
		},
	}

	sql, err := BuildSQL(query, "text", query.Table, []string{"district", "year", "month"})
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Verify key SQL components
	if !strings.Contains(sql, "SELECT") {
		t.Error("SQL should contain SELECT")
	}
	if !strings.Contains(sql, "SUM(") {
		t.Error("SQL should contain SUM aggregation")
	}
	if !strings.Contains(sql, "num_fever_u5_male") {
		t.Error("SQL should contain column name")
	}
	if !strings.Contains(sql, "WHERE 1=1") {
		t.Error("SQL should contain WHERE clause")
	}
	if !strings.Contains(sql, "{{district_filter}}") {
		t.Error("SQL should contain filter placeholders")
	}
	// For cht_form_097b, should wrap with COALESCE/CAST/NULLIF
	if !strings.Contains(sql, "COALESCE") {
		t.Error("SQL should wrap cht_form_097b columns with COALESCE")
	}
}

// TestBuildSQLAggregationRoundTo verifies aggregation-level RoundTo wraps
// SUM/AVG/MAX/MIN in ROUND(..., n) and is ignored for COUNT/COUNT_DISTINCT.
func TestBuildSQLAggregationRoundTo(t *testing.T) {
	t.Run("AVG with RoundTo wraps in ROUND", func(t *testing.T) {
		query := Query{
			Table: "report.mv_rmnch_dist_dataelements",
			Aggregations: []Aggregation{
				{Column: "anc4_coverage", Function: "avg", Alias: "anc4", RoundTo: intPtr(1)},
			},
		}
		sql, err := BuildSQL(query, "text", query.Table, nil)
		if err != nil {
			t.Fatalf("BuildSQL failed: %v", err)
		}
		if !strings.Contains(sql, "ROUND(AVG(") {
			t.Errorf("expected ROUND(AVG(...)) in SQL; got: %s", sql)
		}
		if !strings.Contains(sql, "::numeric, 1)") {
			t.Errorf("expected ::numeric cast with precision 1; got: %s", sql)
		}
	})

	t.Run("SUM with RoundTo wraps in ROUND", func(t *testing.T) {
		query := Query{
			Table: "report.cht_form_097b",
			Aggregations: []Aggregation{
				{Column: "num_fever_u5_male", Function: "sum", Alias: "fever", RoundTo: intPtr(2)},
			},
		}
		sql, err := BuildSQL(query, "text", query.Table, nil)
		if err != nil {
			t.Fatalf("BuildSQL failed: %v", err)
		}
		if !strings.Contains(sql, "ROUND(SUM(") {
			t.Errorf("expected ROUND(SUM(...)) in SQL; got: %s", sql)
		}
		if !strings.Contains(sql, ", 2)") {
			t.Errorf("expected precision 2; got: %s", sql)
		}
	})

	t.Run("COUNT ignores RoundTo", func(t *testing.T) {
		query := Query{
			Table: "report.cht_form_097b",
			Aggregations: []Aggregation{
				{Column: "district", Function: "count", Alias: "n", RoundTo: intPtr(1)},
			},
		}
		sql, err := BuildSQL(query, "text", query.Table, nil)
		if err != nil {
			t.Fatalf("BuildSQL failed: %v", err)
		}
		if strings.Contains(sql, "ROUND(") {
			t.Errorf("COUNT should not be wrapped in ROUND; got: %s", sql)
		}
	})

	t.Run("COUNT_DISTINCT ignores RoundTo", func(t *testing.T) {
		query := Query{
			Table: "report.cht_form_097b",
			Aggregations: []Aggregation{
				{Column: "district", Function: "count_distinct", Alias: "n", RoundTo: intPtr(1)},
			},
		}
		sql, err := BuildSQL(query, "text", query.Table, nil)
		if err != nil {
			t.Fatalf("BuildSQL failed: %v", err)
		}
		if strings.Contains(sql, "ROUND(") {
			t.Errorf("COUNT_DISTINCT should not be wrapped in ROUND; got: %s", sql)
		}
	})

	t.Run("no RoundTo preserves bare aggregation", func(t *testing.T) {
		query := Query{
			Table: "report.mv_rmnch_dist_dataelements",
			Aggregations: []Aggregation{
				{Column: "anc4_coverage", Function: "avg", Alias: "anc4"},
			},
		}
		sql, err := BuildSQL(query, "text", query.Table, nil)
		if err != nil {
			t.Fatalf("BuildSQL failed: %v", err)
		}
		if strings.Contains(sql, "ROUND(AVG(") {
			t.Errorf("bare AVG should not be wrapped in ROUND; got: %s", sql)
		}
		if !strings.Contains(sql, "AVG(") {
			t.Errorf("expected plain AVG; got: %s", sql)
		}
	})
}

// TestBuildSQLWithGrouping tests query generation with GROUP BY
func TestBuildSQLWithGrouping(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		GroupBy: []GroupBy{
			{
				Field:  "period_date",
				Format: "month",
			},
		},
		Aggregations: []Aggregation{
			{
				Column:   "num_fever_u5_male + num_fever_u5_female",
				Function: "sum",
				Alias:    "Fever Cases",
			},
		},
	}

	sql, err := BuildSQL(query, "bar", query.Table, nil)
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Verify grouping components — period_date is a date field, uses TO_CHAR directly
	sqlLower := strings.ToLower(sql)
	if !strings.Contains(sqlLower, "to_char") || !strings.Contains(sqlLower, "'mon yyyy'") {
		t.Errorf("SQL should contain TO_CHAR with 'Mon YYYY' for date field month format. Got: %s", sql)
	}
	if !strings.Contains(sqlLower, "group by") {
		t.Error("SQL should contain GROUP BY clause")
	}
}

// TestBuildSQLWithMultipleGroupBy tests query generation with multiple GROUP BY fields (for pivot)
func TestBuildSQLWithMultipleGroupBy(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		GroupBy: []GroupBy{
			{
				Field: "district",
				Alias: "District",
			},
			{
				Field:  "period_date",
				Format: "month",
				Alias:  "Month",
			},
		},
		Aggregations: []Aggregation{
			{
				Column:   "assess_u5_male",
				Function: "sum",
				Alias:    "TotalAssess",
			},
		},
	}

	sql, err := BuildSQL(query, "table", query.Table, nil)
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Verify both groupBy fields are in SELECT
	if !strings.Contains(sql, `"District"`) {
		t.Errorf("SQL should contain District alias in SELECT. Got: %s", sql)
	}
	if !strings.Contains(sql, `"Month"`) {
		t.Errorf("SQL should contain Month alias in SELECT. Got: %s", sql)
	}

	// Verify grouping
	sqlLower := strings.ToLower(sql)
	if !strings.Contains(sqlLower, "group by") {
		t.Error("SQL should contain GROUP BY clause")
	}
	if !strings.Contains(sqlLower, "order by") {
		t.Error("SQL should contain ORDER BY clause")
	}
	if !strings.Contains(sqlLower, "extract(month from period_date::date)") {
		t.Error("SQL should contain month extraction for grouping and ordering")
	}
}

// TestBuildSQLMultiDataset tests query generation for multi-dataset charts
func TestBuildSQLMultiDataset(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		GroupBy: []GroupBy{
			{
				Field:  "period_date",
				Format: "month",
			},
		},
		Aggregations: []Aggregation{
			{
				Column:   "num_u5_with_pneumonia_and_recieved_amoxicilin_male + num_u5_with_pneumonia_and_recieved_amoxicilin_female",
				Function: "sum",
				Alias:    "Cases",
			},
			{
				Column:   "num_u5_with_pneumonia_and_recieved_amoxicilin_male + num_u5_with_pneumonia_and_recieved_amoxicilin_female",
				Function: "sum",
				Alias:    "Treated with Amoxicillin",
			},
		},
	}

	sql, err := BuildSQL(query, "bar", query.Table, nil)
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Verify multiple datasets
	if !strings.Contains(sql, "\"Cases\"") {
		t.Error("SQL should contain first dataset alias")
	}
	if !strings.Contains(sql, "\"Treated with Amoxicillin\"") {
		t.Error("SQL should contain second dataset alias")
	}
	// Should have two aggregations
	countSum := strings.Count(sql, "SUM(")
	if countSum < 2 {
		t.Errorf("SQL should contain at least 2 SUM aggregations, found %d", countSum)
	}
}

// TestBuildSQLCalculatedField tests query generation with calculated field
func TestBuildSQLCalculatedField(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		GroupBy: []GroupBy{
			{
				Field:  "period_date",
				Format: "month",
			},
		},
		Aggregations: []Aggregation{
			{
				Column:   "num_malaria_u5_male + num_malaria_u5_female",
				Function: "sum",
				Alias:    "positive",
			},
			{
				Column:   "num_u5_with_fever_recieved_mrdt_male + num_u5_with_fever_recieved_mrdt_female",
				Function: "sum",
				Alias:    "tested",
			},
		},
		Calculate: []Calculate{{
			Formula:  "(positive / tested * 100)",
			RoundTo:  intPtr(1),
			WhenZero: 0,
		}},
	}

	sql, err := BuildSQL(query, "line", query.Table, nil)
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Verify calculated field handling
	if !strings.Contains(sql, "CASE WHEN") {
		t.Error("SQL should contain CASE for division by zero check")
	}
	if !strings.Contains(sql, "ROUND(") {
		t.Error("SQL should contain ROUND for decimal places")
	}
	if !strings.Contains(sql, "ELSE 0") {
		t.Error("SQL should contain ELSE clause with whenZero value")
	}
	if !strings.Contains(sql, "positive") && !strings.Contains(sql, "SUM(") {
		t.Error("SQL should replace formula aliases with aggregation expressions")
	}
}

// TestWrapNumericColumnChtForm097b tests type conversion wrapping
func TestWrapNumericColumnChtForm097b(t *testing.T) {
	tests := []struct {
		name      string
		column    string
		tableName string
		expected  string
	}{
		{
			name:      "simple column - cht_form_097b",
			column:    "num_fever_u5_male",
			tableName: "report.cht_form_097b",
			expected:  "COALESCE(CAST(NULLIF(TRIM(CAST(\"num_fever_u5_male\" AS TEXT)), '') AS NUMERIC), 0)",
		},
		{
			name:      "expression with addition",
			column:    "num_fever_u5_male + num_fever_u5_female",
			tableName: "report.cht_form_097b",
			// Note: Parser adds outer parentheses for proper BODMAS precedence handling
		expected:  "(COALESCE(CAST(NULLIF(TRIM(CAST(\"num_fever_u5_male\" AS TEXT)), '') AS NUMERIC), 0) + COALESCE(CAST(NULLIF(TRIM(CAST(\"num_fever_u5_female\" AS TEXT)), '') AS NUMERIC), 0))",
		},
		{
			name:      "simple column - other table (universal wrapping)",
			column:    "some_column",
			tableName: "report.other_table",
			expected:  "COALESCE(CAST(NULLIF(TRIM(CAST(\"some_column\" AS TEXT)), '') AS NUMERIC), 0)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := wrapNumericColumn(test.column, test.tableName)
			if result != test.expected {
				t.Errorf("Expected: %s\nGot: %s", test.expected, result)
			}
		})
	}
}

// TestBuildSQLWithFilterPlaceholders tests that filter placeholders are included
func TestBuildSQLWithFilterPlaceholders(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "num_fever_u5_male",
				Function: "sum",
				Alias:    "value",
			},
		},
	}

	// Pass explicit filters to get filter placeholders in SQL
	sql, err := BuildSQL(query, "text", query.Table, []string{"district", "year", "month"})
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Verify all filter placeholders are present
	placeholders := []string{"{{district_filter}}", "{{year_filter}}", "{{month_filter}}"}
	for _, placeholder := range placeholders {
		if !strings.Contains(sql, placeholder) {
			t.Errorf("SQL should contain %s placeholder", placeholder)
		}
	}
}

// TestGetGroupByExpression tests date formatting for different formats
func TestGetGroupByExpression(t *testing.T) {
	tests := []struct {
		name        string
		groupBy     GroupBy
		expectedStr string
	}{
		// Date field cases (field contains "date")
		{
			name:        "date field: month format",
			groupBy:     GroupBy{Field: "period_date", Format: "month"},
			expectedStr: `TO_CHAR("period_date"::date, 'Mon YYYY')`,
		},
		{
			name:        "date field: month_noyear format",
			groupBy:     GroupBy{Field: "period_date", Format: "month_noyear"},
			expectedStr: `TO_CHAR("period_date"::date, 'Mon')`,
		},
		{
			name:        "date field: week format",
			groupBy:     GroupBy{Field: "period_date", Format: "week"},
			expectedStr: `TO_CHAR("period_date"::date, 'IYYY') || 'W' || LPAD(EXTRACT(WEEK FROM "period_date"::date)::text, 2, '0')`,
		},
		{
			name:        "date field: week_noyear format",
			groupBy:     GroupBy{Field: "period_date", Format: "week_noyear"},
			expectedStr: `'W' || LPAD(EXTRACT(WEEK FROM "period_date"::date)::text, 2, '0')`,
		},
		{
			name:        "date field: quarter format",
			groupBy:     GroupBy{Field: "period_date", Format: "quarter"},
			expectedStr: `TO_CHAR("period_date"::date, 'YYYY') || 'Q' || EXTRACT(QUARTER FROM "period_date"::date)::text`,
		},
		{
			name:        "date field: quarter_noyear format",
			groupBy:     GroupBy{Field: "period_date", Format: "quarter_noyear"},
			expectedStr: `'Q' || EXTRACT(QUARTER FROM "period_date"::date)::text`,
		},
		{
			name:        "date field: year format",
			groupBy:     GroupBy{Field: "period_date", Format: "year"},
			expectedStr: `TO_CHAR("period_date"::date, 'YYYY')`,
		},
		{
			name:        "date field: date format",
			groupBy:     GroupBy{Field: "period_date", Format: "date"},
			expectedStr: `TO_CHAR("period_date"::date, 'YYYY-MM-DD')`,
		},
		// Non-date (numeric) field cases — backward compatibility
		{
			name:        "numeric field: month format uses MAKE_DATE",
			groupBy:     GroupBy{Field: "month", Format: "month"},
			expectedStr: `TO_CHAR(MAKE_DATE("year"::int, "month"::int, 1), 'Mon YYYY')`,
		},
		{
			name:        "numeric field: month_noyear format",
			groupBy:     GroupBy{Field: "month", Format: "month_noyear"},
			expectedStr: `TO_CHAR(MAKE_DATE(2000, "month"::int, 1), 'Mon')`,
		},
		{
			name:        "numeric field: week format",
			groupBy:     GroupBy{Field: "week", Format: "week"},
			expectedStr: `"year"::text || 'W' || LPAD("week"::text, 2, '0')`,
		},
		{
			name:        "numeric field: week_noyear format",
			groupBy:     GroupBy{Field: "week", Format: "week_noyear"},
			expectedStr: `'W' || LPAD("week"::text, 2, '0')`,
		},
		{
			name:        "numeric field: quarter format",
			groupBy:     GroupBy{Field: "quarter", Format: "quarter"},
			expectedStr: `"year"::text || 'Q' || "quarter"::text`,
		},
		{
			name:        "numeric field: quarter_noyear format",
			groupBy:     GroupBy{Field: "quarter", Format: "quarter_noyear"},
			expectedStr: `'Q' || "quarter"::text`,
		},
		{
			name:        "numeric field: quarter_year format with custom field",
			groupBy:     GroupBy{Field: "period", Format: "quarter_year"},
			expectedStr: `"year"::text || 'Q' || "period"::text`,
		},
		// Other cases
		{
			name:        "no format (quoted column)",
			groupBy:     GroupBy{Field: "district", Format: ""},
			expectedStr: `"district"`,
		},
		// report_date also detected as date field
		{
			name:        "report_date field: month format",
			groupBy:     GroupBy{Field: "report_date", Format: "month"},
			expectedStr: `TO_CHAR("report_date"::date, 'Mon YYYY')`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := getGroupByExpression(test.groupBy)
			if result != test.expectedStr {
				t.Errorf("Expected: %s\nGot: %s", test.expectedStr, result)
			}
		})
	}
}

// TestBuildSQLWithDateFieldGroupBy tests full SQL generation with a date field groupBy
func TestBuildSQLWithDateFieldGroupBy(t *testing.T) {
	query := Query{
		Table: "report.estat_community",
		GroupBy: []GroupBy{
			{
				Field:  "period_date",
				Format: "month",
			},
		},
		Aggregations: []Aggregation{
			{
				Column:   "household_visits",
				Function: "sum",
				Alias:    "TotalVisits",
			},
		},
	}

	sql, err := BuildSQL(query, "bar", query.Table, []string{"district", "year", "month"})
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	sqlLower := strings.ToLower(sql)

	// Should use TO_CHAR directly on date field, NOT MAKE_DATE with year column
	if strings.Contains(sqlLower, "make_date") {
		t.Errorf("Date field should NOT use MAKE_DATE. Got: %s", sql)
	}
	if !strings.Contains(sqlLower, `to_char("period_date"::date, 'mon yyyy')`) {
		t.Errorf("Date field should use TO_CHAR on date directly. Got: %s", sql)
	}

	// Should NOT reference a bare "year" column (which doesn't exist in eSTAT tables)
	// But should have year_filter placeholder and EXTRACT(YEAR FROM period_date::date)
	if strings.Contains(sqlLower, "group by") {
		// GROUP BY should use EXTRACT, not bare "year"
		if !strings.Contains(sqlLower, "extract(year from period_date::date)") {
			t.Errorf("GROUP BY should use EXTRACT for date fields. Got: %s", sql)
		}
	}
}

// TestQuoteIdentifier tests identifier quoting
func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple identifier",
			input:    "value",
			expected: "\"value\"",
		},
		{
			name:     "identifier with spaces",
			input:    "Total Cases",
			expected: "\"Total Cases\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := quoteIdentifier(test.input)
			if result != test.expected {
				t.Errorf("Expected: %s\nGot: %s", test.expected, result)
			}
		})
	}
}

// TestBuildSQLErrorHandling tests error cases
func TestBuildSQLErrorHandling(t *testing.T) {
	tests := []struct {
		name  string
		query Query
		err   bool
	}{
		{
			name:  "missing table",
			query: Query{Aggregations: []Aggregation{{Column: "col", Function: "sum", Alias: "val"}}},
			err:   true,
		},
		{
			name:  "missing aggregations",
			query: Query{Table: "my_table"},
			err:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildSQL(test.query, "text", test.query.Table, nil)
			if (err != nil) != test.err {
				t.Errorf("Expected error: %v, got: %v", test.err, err)
			}
		})
	}
}

// TestBuildSQLReportLevelFilters tests report-level filter inheritance
func TestBuildSQLReportLevelFilters(t *testing.T) {
	tests := []struct {
		name           string
		filters        []string
		expectedIn     string
		notExpectedIn  string
	}{
		{
			name:       "All filters",
			filters:    []string{"district", "year", "month"},
			expectedIn: "{{district_filter}} {{year_filter}} {{month_filter}}",
		},
		{
			name:          "Only district filter",
			filters:       []string{"district"},
			expectedIn:    "{{district_filter}}",
			notExpectedIn: "{{year_filter}}",
		},
		{
			name:          "Only year and month filters",
			filters:       []string{"year", "month"},
			expectedIn:    "{{year_filter}} {{month_filter}}",
			notExpectedIn: "{{district_filter}}",
		},
		{
			name:          "Empty filters removes all placeholders",
			filters:       []string{},
			notExpectedIn: "{{district_filter}}",
		},
		{
			name:          "Nil filters removes all placeholders",
			filters:       nil,
			notExpectedIn: "{{district_filter}}",
		},
	}

	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{Column: "num_fever_u5_male", Function: "sum", Alias: "value"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sql, err := BuildSQL(query, "text", query.Table, test.filters)
			if err != nil {
				t.Fatalf("BuildSQL failed: %v", err)
			}

			if test.expectedIn != "" && !strings.Contains(sql, test.expectedIn) {
				t.Errorf("SQL should contain '%s'", test.expectedIn)
			}
			if test.notExpectedIn != "" && strings.Contains(sql, test.notExpectedIn) {
				t.Errorf("SQL should NOT contain '%s'", test.notExpectedIn)
			}
		})
	}
}

// TestBuildSelectClauseFormatting tests proper SELECT clause formatting
func TestBuildSelectClauseFormatting(t *testing.T) {
	aggs := []Aggregation{
		{Column: "col1", Function: "sum", Alias: "alias1"},
		{Column: "col2", Function: "avg", Alias: "alias2"},
	}

	selectClause, err := buildSelectClause(aggs, []GroupBy{}, nil, "report.other_table")
	if err != nil {
		t.Fatalf("buildSelectClause failed: %v", err)
	}

	if !strings.Contains(selectClause, "SELECT") {
		t.Error("SELECT clause should start with SELECT keyword")
	}
	if !strings.Contains(selectClause, "SUM(") {
		t.Error("SELECT clause should contain SUM aggregation")
	}
	if !strings.Contains(selectClause, "AVG(") {
		t.Error("SELECT clause should contain AVG aggregation")
	}
}

// TestGroupByIntegerColumn tests grouping by an integer column (without date format)
func TestGroupByIntegerColumn(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		GroupBy: []GroupBy{
			{
				Field:  "year",
				Format: "", // No format - treating as numeric column
			},
		},
		Aggregations: []Aggregation{
			{
				Column:   "num_fever_u5_male + num_fever_u5_female",
				Function: "sum",
				Alias:    "Fever Cases",
			},
		},
	}

	sql, err := BuildSQL(query, "bar", query.Table, nil)
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Should group by year directly without date formatting
	sqlLower := strings.ToLower(sql)
	if !strings.Contains(sqlLower, "group by") {
		t.Error("SQL should contain GROUP BY clause")
	}
	// Should NOT try to cast to date
	if strings.Contains(sqlLower, "year::date") {
		t.Error("SQL should NOT cast numeric year to date")
	}
	// Should have year in the GROUP BY
	if !strings.Contains(sqlLower, "year") {
		t.Error("SQL should contain year column in GROUP BY")
	}
}

// TestGroupByMonthInteger tests grouping by a numeric month column
func TestGroupByMonthInteger(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		GroupBy: []GroupBy{
			{
				Field:  "month",
				Format: "", // No format - treating as numeric column
			},
		},
		Aggregations: []Aggregation{
			{
				Column:   "num_fever_u5_male + num_fever_u5_female",
				Function: "sum",
				Alias:    "Fever Cases",
			},
		},
	}

	sql, err := BuildSQL(query, "line", query.Table, nil)
	if err != nil {
		t.Fatalf("BuildSQL failed: %v", err)
	}

	// Should group by month directly without date formatting
	sqlLower := strings.ToLower(sql)
	if !strings.Contains(sqlLower, "group by") {
		t.Error("SQL should contain GROUP BY clause")
	}
	// Should NOT try to cast to date
	if strings.Contains(sqlLower, "month::date") {
		t.Error("SQL should NOT cast numeric month to date")
	}
}

// ============== BODMAS Integration Tests ==============

// TestBuildSQLWithBODMASMultiplication tests multiplication in column expressions
func TestBuildSQLWithBODMASMultiplication(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "col1 * col2",
				Function: "sum",
				Alias:    "product",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.cht_form_097b", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(strings.ToUpper(sql), "*") {
		t.Errorf("SQL should contain * operator, got: %s", sql)
	}

	if !strings.Contains(sql, "COALESCE") {
		t.Errorf("SQL should contain COALESCE wrapping, got: %s", sql)
	}
}

// TestBuildSQLWithBODMASDivision tests division in column expressions
func TestBuildSQLWithBODMASDivision(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "col1 / col2",
				Function: "sum",
				Alias:    "ratio",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.cht_form_097b", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(strings.ToUpper(sql), "/") {
		t.Errorf("SQL should contain / operator, got: %s", sql)
	}
}

// TestBuildSQLWithBODMASSubtraction tests subtraction in column expressions
func TestBuildSQLWithBODMASSubtraction(t *testing.T) {
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "col1 - col2",
				Function: "sum",
				Alias:    "difference",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.cht_form_097b", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(strings.ToUpper(sql), "-") {
		t.Errorf("SQL should contain - operator, got: %s", sql)
	}
}

// TestBuildSQLWithBODMASPrecedence tests proper operator precedence
func TestBuildSQLWithBODMASPrecedence(t *testing.T) {
	// a + b * c should compute as a + (b * c), not (a + b) * c
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "col1 + col2 * col3",
				Function: "sum",
				Alias:    "value",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.cht_form_097b", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both operators are present
	if !strings.Contains(sql, "+") {
		t.Errorf("SQL should contain +, got: %s", sql)
	}
	if !strings.Contains(sql, "*") {
		t.Errorf("SQL should contain *, got: %s", sql)
	}

	// Verify COALESCE wrapping for all columns
	coalesceCount := strings.Count(sql, "COALESCE")
	if coalesceCount < 3 {
		t.Errorf("expected at least 3 COALESCE blocks for 3 columns, got %d", coalesceCount)
	}
}

// TestBuildSQLWithBODMASParentheses tests parentheses override precedence
func TestBuildSQLWithBODMASParentheses(t *testing.T) {
	// (a + b) * c should compute grouped addition first
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "(col1 + col2) * col3",
				Function: "sum",
				Alias:    "value",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.cht_form_097b", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify structure
	if !strings.Contains(sql, "+") || !strings.Contains(sql, "*") {
		t.Errorf("SQL should contain both + and *, got: %s", sql)
	}
}

// TestBuildSQLWithBODMASComplexExpression tests complex multi-operator expression
func TestBuildSQLWithBODMASComplexExpression(t *testing.T) {
	// col1 / col2 - col3 * col4
	query := Query{
		Table: "report.cht_form_097b",
		Aggregations: []Aggregation{
			{
				Column:   "col1 / col2 - col3 * col4",
				Function: "sum",
				Alias:    "value",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.cht_form_097b", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all operators are present
	if !strings.Contains(sql, "/") {
		t.Errorf("SQL should contain /, got: %s", sql)
	}
	if !strings.Contains(sql, "-") {
		t.Errorf("SQL should contain -, got: %s", sql)
	}
	if !strings.Contains(sql, "*") {
		t.Errorf("SQL should contain *, got: %s", sql)
	}

	// Verify COALESCE wrapping
	if !strings.Contains(sql, "COALESCE") {
		t.Errorf("SQL should contain COALESCE wrapping, got: %s", sql)
	}
}

// TestBuildSQLBackwardCompatibilityAdditionOnly tests that old + expressions still work
func TestBuildSQLBackwardCompatibilityAdditionOnly(t *testing.T) {
	tests := []string{
		"col1 + col2",
		"col1 + col2 + col3",
		"num_fever_u5_male + num_fever_u5_female",
	}

	for _, columnExpr := range tests {
		query := Query{
			Table: "report.cht_form_097b",
			Aggregations: []Aggregation{
				{
					Column:   columnExpr,
					Function: "sum",
					Alias:    "value",
				},
			},
		}

		sql, err := BuildSQLWithSquirrel(query, "bar", "report.cht_form_097b", []string{})
		if err != nil {
			t.Errorf("BuildSQLWithSquirrel for '%s' returned error: %v", columnExpr, err)
			continue
		}

		// Verify SQL is generated
		if sql == "" {
			t.Errorf("empty SQL for %s", columnExpr)
		}

		// Verify columns are wrapped
		if !strings.Contains(sql, "COALESCE") {
			t.Errorf("no COALESCE wrapping for %s: %s", columnExpr, sql)
		}

		// Verify addition is preserved
		if !strings.Contains(sql, "+") {
			t.Errorf("+ operator not found for %s: %s", columnExpr, sql)
		}
	}
}

// TestBuildSQLWithWhereClause tests custom WHERE clause functionality
func TestBuildSQLWithWhereClause(t *testing.T) {
	tests := []struct {
		name        string
		where       string
		shouldError bool
		contains    string
	}{
		{
			name:        "simple string condition",
			where:       "age_group = 'under_5'",
			shouldError: false,
			contains:    "age_group = 'under_5'",
		},
		{
			name:        "numeric condition",
			where:       "age > 5",
			shouldError: false,
			contains:    "age > 5",
		},
		{
			name:        "multiple conditions with AND",
			where:       "age_group = 'under_5' AND gender = 'male'",
			shouldError: false,
			contains:    "AND gender = 'male'",
		},
		{
			name:        "IN clause",
			where:       "disease_code IN ('MAL', 'TYP', 'DEN')",
			shouldError: false,
			contains:    "IN ('MAL', 'TYP', 'DEN')",
		},
		{
			name:        "parentheses in condition",
			where:       "(age >= 5 AND age < 18) OR category = 'pediatric'",
			shouldError: false,
			contains:    "(age >= 5 AND age < 18)",
		},
		{
			name:        "semicolon blocked",
			where:       "age > 5; DROP TABLE users",
			shouldError: true,
		},
		{
			name:        "SQL comment blocked",
			where:       "age > 5 -- comment",
			shouldError: true,
		},
		{
			name:        "subquery blocked",
			where:       "id IN (SELECT id FROM other_table)",
			shouldError: true,
		},
		{
			name:        "unbalanced parentheses",
			where:       "(age > 5 AND (gender = 'male')",
			shouldError: true,
		},
		{
			name:        "empty where clause",
			where:       "",
			shouldError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			query := Query{
				Table: "report.test_table",
				Where: tc.where,
				Aggregations: []Aggregation{
					{Column: "cases", Function: "sum", Alias: "TotalCases"},
				},
			}

			sql, err := BuildSQLWithSquirrel(query, "bar", "report.test_table", []string{})

			if tc.shouldError {
				if err == nil {
					t.Errorf("expected error for where clause '%s', got none", tc.where)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for where clause '%s': %v", tc.where, err)
				}
				if tc.contains != "" && !strings.Contains(sql, tc.contains) {
					t.Errorf("expected SQL to contain '%s', got: %s", tc.contains, sql)
				}
			}
		})
	}
}

// TestValidateWhereClause tests the validation function directly
func TestValidateWhereClause(t *testing.T) {
	tests := []struct {
		name        string
		where       string
		shouldError bool
		errorMsg    string
	}{
		{"valid simple", "status = 'active'", false, ""},
		{"valid numeric", "quantity > 100", false, ""},
		{"valid complex", "(a = 1 AND b = 2) OR c = 3", false, ""},
		{"empty string", "", false, ""},
		{"semicolon", "a = 1; b = 2", true, "semicolons"},
		{"comment", "a = 1 -- hack", true, "comments"},
		{"subquery", "a IN (SELECT b FROM c)", true, "subqueries"},
		{"unbalanced open", "(a = 1", true, "unbalanced"},
		{"unbalanced close", "a = 1)", true, "unbalanced"},
		{"nested unbalanced", "((a = 1)", true, "unbalanced"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWhereClause(tc.where)
			if tc.shouldError {
				if err == nil {
					t.Errorf("expected error for '%s'", tc.where)
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("expected error containing '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for '%s': %v", tc.where, err)
				}
			}
		})
	}
}

// TestBuildSQLCountOnTextColumn tests COUNT aggregation on text columns
// COUNT should NOT use numeric casting (unlike SUM/AVG/MAX/MIN)
func TestBuildSQLCountOnTextColumn(t *testing.T) {
	query := Query{
		Table: "report.test_table",
		Aggregations: []Aggregation{
			{
				Column:   "district_name",
				Function: "count",
				Alias:    "DistrictCount",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.test_table", []string{})
	if err != nil {
		t.Fatalf("BuildSQLWithSquirrel failed: %v", err)
	}

	// COUNT should NOT contain NUMERIC casting (that would fail on text)
	if strings.Contains(sql, "AS NUMERIC") {
		t.Errorf("COUNT on text column should NOT use NUMERIC casting, got: %s", sql)
	}

	// COUNT should still filter empty strings via NULLIF
	if !strings.Contains(sql, "NULLIF") {
		t.Errorf("COUNT should use NULLIF to skip empty strings, got: %s", sql)
	}

	// COUNT should use TRIM to handle whitespace
	if !strings.Contains(sql, "TRIM") {
		t.Errorf("COUNT should use TRIM, got: %s", sql)
	}

	// Should contain COUNT function
	if !strings.Contains(sql, "COUNT(") {
		t.Errorf("SQL should contain COUNT, got: %s", sql)
	}
}

// TestBuildSQLCountStar tests COUNT(*) aggregation
func TestBuildSQLCountStar(t *testing.T) {
	query := Query{
		Table: "report.test_table",
		Aggregations: []Aggregation{
			{
				Column:   "*",
				Function: "count",
				Alias:    "TotalRows",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.test_table", []string{})
	if err != nil {
		t.Fatalf("BuildSQLWithSquirrel failed: %v", err)
	}

	// COUNT(*) should use literal *
	if !strings.Contains(sql, "COUNT(*)") {
		t.Errorf("COUNT(*) should produce COUNT(*), got: %s", sql)
	}
}

// TestBuildSQLCountDistinct tests COUNT(DISTINCT ...) aggregation
func TestBuildSQLCountDistinct(t *testing.T) {
	query := Query{
		Table: "report.test_table",
		Aggregations: []Aggregation{
			{
				Column:   "district_name",
				Function: "count_distinct",
				Alias:    "UniqueDistricts",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.test_table", []string{})
	if err != nil {
		t.Fatalf("BuildSQLWithSquirrel failed: %v", err)
	}

	// Should contain COUNT(DISTINCT ...)
	if !strings.Contains(sql, "COUNT(DISTINCT") {
		t.Errorf("count_distinct should produce COUNT(DISTINCT ...), got: %s", sql)
	}

	// Should NOT use numeric casting
	if strings.Contains(sql, "AS NUMERIC") {
		t.Errorf("count_distinct should NOT use NUMERIC casting, got: %s", sql)
	}

	// Should use NULLIF to skip empty strings
	if !strings.Contains(sql, "NULLIF") {
		t.Errorf("count_distinct should use NULLIF to skip empty strings, got: %s", sql)
	}
}

// TestBuildSQLCountDistinctInCalculation tests that count_distinct in a calculated field
// produces COUNT(DISTINCT ...) not COUNT_DISTINCT(...) which PostgreSQL doesn't recognize
func TestBuildSQLCountDistinctInCalculation(t *testing.T) {
	query := Query{
		Table: "report.test_table",
		Aggregations: []Aggregation{
			{
				Column:   "fieldworker_uuid",
				Function: "count_distinct",
				Alias:    "unique_fieldworkers",
			},
			{
				Column:   "expected_fieldworkers",
				Function: "sum",
				Alias:    "total_expected",
			},
		},
		Calculate: []Calculate{
			{
				Formula:     "(unique_fieldworkers / total_expected * 100)",
				RoundTo:     intPtr(1),
				ResultAlias: "rate",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.test_table", []string{})
	if err != nil {
		t.Fatalf("BuildSQLWithSquirrel failed: %v", err)
	}

	// Must NOT contain COUNT_DISTINCT as a function call
	if strings.Contains(sql, "COUNT_DISTINCT(") {
		t.Errorf("calculated field should not produce COUNT_DISTINCT(...), got: %s", sql)
	}

	// Must contain COUNT(DISTINCT ...)
	if !strings.Contains(sql, "COUNT(DISTINCT") {
		t.Errorf("calculated field should produce COUNT(DISTINCT ...), got: %s", sql)
	}

	// count_distinct should use TEXT casting, not NUMERIC
	// The COUNT(DISTINCT ...) part should contain CAST(... AS TEXT)
	if !strings.Contains(sql, "AS TEXT") {
		t.Errorf("count_distinct in calculated field should cast to TEXT, got: %s", sql)
	}
}

// TestBuildSQLMixedAggregations tests mixed COUNT and SUM aggregations
// SUM should use numeric wrapping, COUNT should not
func TestBuildSQLMixedAggregations(t *testing.T) {
	query := Query{
		Table: "report.test_table",
		Aggregations: []Aggregation{
			{
				Column:   "district_name",
				Function: "count",
				Alias:    "DistrictCount",
			},
			{
				Column:   "cases",
				Function: "sum",
				Alias:    "TotalCases",
			},
		},
	}

	sql, err := BuildSQLWithSquirrel(query, "bar", "report.test_table", []string{})
	if err != nil {
		t.Fatalf("BuildSQLWithSquirrel failed: %v", err)
	}

	// Should contain both COUNT and SUM
	if !strings.Contains(sql, "COUNT(") {
		t.Errorf("SQL should contain COUNT, got: %s", sql)
	}
	if !strings.Contains(sql, "SUM(") {
		t.Errorf("SQL should contain SUM, got: %s", sql)
	}

	// SUM should use COALESCE (numeric wrapping)
	if !strings.Contains(sql, "COALESCE") {
		t.Errorf("SUM should use COALESCE wrapping, got: %s", sql)
	}
}

// TestWrapColumnForCount tests the wrapColumnForCount function directly
func TestWrapColumnForCount(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"*", "*"},
		{"district", `NULLIF(TRIM(CAST("district" AS TEXT)), '')`},
		{" district ", `NULLIF(TRIM(CAST("district" AS TEXT)), '')`}, // trimmed input
		{"facility_name", `NULLIF(TRIM(CAST("facility_name" AS TEXT)), '')`},
	}

	for _, tc := range tests {
		result := wrapColumnForCount(tc.input)
		if result != tc.expected {
			t.Errorf("wrapColumnForCount(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// ============== buildWhereClause Tests ==============

func TestBuildWhereClause(t *testing.T) {
	tests := []struct {
		name     string
		filters  []string
		contains []string
		absent   []string
	}{
		{
			name:     "nil filters includes all",
			filters:  nil,
			contains: []string{"WHERE 1=1", "{{district_filter}}", "{{year_filter}}", "{{month_filter}}", "{{week_filter}}", "{{quarter_filter}}"},
		},
		{
			name:     "empty filters includes all",
			filters:  []string{},
			contains: []string{"WHERE 1=1", "{{district_filter}}", "{{year_filter}}", "{{month_filter}}"},
		},
		{
			name:     "district only",
			filters:  []string{"district"},
			contains: []string{"WHERE 1=1", "{{district_filter}}"},
			absent:   []string{"{{year_filter}}", "{{month_filter}}", "{{week_filter}}", "{{quarter_filter}}"},
		},
		{
			name:     "year and month",
			filters:  []string{"year", "month"},
			contains: []string{"{{year_filter}}", "{{month_filter}}"},
			absent:   []string{"{{district_filter}}", "{{week_filter}}"},
		},
		{
			name:     "week only",
			filters:  []string{"week"},
			contains: []string{"{{week_filter}}"},
			absent:   []string{"{{district_filter}}", "{{year_filter}}", "{{month_filter}}", "{{quarter_filter}}"},
		},
		{
			name:     "quarter only",
			filters:  []string{"quarter"},
			contains: []string{"{{quarter_filter}}"},
			absent:   []string{"{{district_filter}}", "{{year_filter}}"},
		},
		{
			name:     "facility only",
			filters:  []string{"facility"},
			contains: []string{"{{facility_filter}}"},
			absent:   []string{"{{district_filter}}", "{{region_filter}}", "{{year_filter}}"},
		},
		{
			name:     "region only",
			filters:  []string{"region"},
			contains: []string{"{{region_filter}}"},
			absent:   []string{"{{district_filter}}", "{{facility_filter}}", "{{year_filter}}"},
		},
		{
			name:     "district and facility",
			filters:  []string{"district", "facility"},
			contains: []string{"{{district_filter}}", "{{facility_filter}}"},
			absent:   []string{"{{year_filter}}", "{{week_filter}}"},
		},
		{
			name:     "case insensitive",
			filters:  []string{"District", "YEAR", "Facility"},
			contains: []string{"{{district_filter}}", "{{year_filter}}", "{{facility_filter}}"},
		},
		{
			name:     "unknown filter ignored",
			filters:  []string{"unknown"},
			contains: []string{"WHERE 1=1"},
			absent:   []string{"{{district_filter}}", "{{year_filter}}"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildWhereClause(tc.filters)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in result: %s", s, result)
				}
			}
			for _, s := range tc.absent {
				if strings.Contains(result, s) {
					t.Errorf("did NOT expect %q in result: %s", s, result)
				}
			}
		})
	}
}

// ============== buildGroupByClause Tests ==============

func TestBuildGroupByClause(t *testing.T) {
	tests := []struct {
		name     string
		groupBy  []GroupBy
		contains []string
	}{
		{
			name:     "raw field (no format)",
			groupBy:  []GroupBy{{Field: "district"}},
			contains: []string{"GROUP BY", "district"},
		},
		{
			name:     "month format adds EXTRACT",
			groupBy:  []GroupBy{{Field: "period_date", Format: "month"}},
			contains: []string{"GROUP BY", "TO_CHAR", "EXTRACT(MONTH FROM"},
		},
		{
			name:     "year format adds EXTRACT",
			groupBy:  []GroupBy{{Field: "period_date", Format: "year"}},
			contains: []string{"GROUP BY", "TO_CHAR", "EXTRACT(YEAR FROM"},
		},
		{
			name:     "no format no EXTRACT",
			groupBy:  []GroupBy{{Field: "year"}},
			contains: []string{"GROUP BY", "year"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildGroupByClause(tc.groupBy)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in: %s", s, result)
				}
			}
		})
	}
}

// ============== buildOrderByClause Tests ==============

func TestBuildOrderByClause(t *testing.T) {
	tests := []struct {
		name     string
		groupBy  []GroupBy
		orderBy  []OrderBy
		contains []string
		absent   []string
	}{
		{
			name:     "explicit orderBy ASC",
			orderBy:  []OrderBy{{Field: "name", Direction: "ASC"}},
			contains: []string{"ORDER BY", "name ASC"},
		},
		{
			name:     "explicit orderBy DESC",
			orderBy:  []OrderBy{{Field: "count", Direction: "DESC"}},
			contains: []string{"ORDER BY", "count DESC"},
		},
		{
			name:     "default direction is ASC",
			orderBy:  []OrderBy{{Field: "id"}},
			contains: []string{"id ASC"},
		},
		{
			name:     "groupBy month defaults to EXTRACT order",
			groupBy:  []GroupBy{{Field: "period_date", Format: "month"}},
			contains: []string{"ORDER BY", "EXTRACT(MONTH FROM"},
		},
		{
			name:     "groupBy year defaults to EXTRACT order",
			groupBy:  []GroupBy{{Field: "period_date", Format: "year"}},
			contains: []string{"ORDER BY", "EXTRACT(YEAR FROM"},
		},
		{
			name:     "groupBy raw field orders by field",
			groupBy:  []GroupBy{{Field: "district"}},
			contains: []string{"ORDER BY", "district"},
		},
		{
			name:    "empty both returns empty",
			groupBy: []GroupBy{},
			orderBy: []OrderBy{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildOrderByClause(tc.groupBy, tc.orderBy)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in: %s", s, result)
				}
			}
			for _, s := range tc.absent {
				if strings.Contains(result, s) {
					t.Errorf("did NOT expect %q in: %s", s, result)
				}
			}
			if len(tc.contains) == 0 && len(tc.groupBy) == 0 && len(tc.orderBy) == 0 && result != "" {
				t.Errorf("expected empty string, got: %s", result)
			}
		})
	}
}

// ============== buildCalculatedField Tests ==============

func TestBuildCalculatedField(t *testing.T) {
	aggs := []Aggregation{
		{Column: "positive", Function: "sum", Alias: "positive"},
		{Column: "tested", Function: "sum", Alias: "tested"},
	}

	tests := []struct {
		name     string
		calc     Calculate
		contains []string
	}{
		{
			name: "division with round",
			calc: Calculate{
				Formula: "(positive / tested * 100)",
				RoundTo: intPtr(1),
			},
			contains: []string{"CASE WHEN", "ROUND(", "ELSE 0", "as value"},
		},
		{
			name: "division with whenZero",
			calc: Calculate{
				Formula:  "(positive / tested * 100)",
				RoundTo:  intPtr(1),
				WhenZero: 0,
			},
			contains: []string{"CASE WHEN", "ELSE 0"},
		},
		{
			name: "no division, no round",
			calc: Calculate{
				Formula: "(positive + tested)",
			},
			contains: []string{"as value"},
		},
		{
			name: "no division, with round",
			calc: Calculate{
				Formula: "(positive + tested)",
				RoundTo: intPtr(2),
			},
			contains: []string{"ROUND(", "as value"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildCalculatedField(tc.calc, aggs, "report.test")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in: %s", s, result)
				}
			}
		})
	}
}

// TestBuildCalculatedFieldDenominatorDetection tests that the CASE WHEN guard references the correct denominator
func TestBuildCalculatedFieldDenominatorDetection(t *testing.T) {
	t.Run("denominator is first aggregation", func(t *testing.T) {
		aggs := []Aggregation{
			{Column: "encounters", Function: "count", Alias: "distinct_encounters"},
			{Column: "antibiotics", Function: "sum", Alias: "total_antibiotics"},
		}
		calc := Calculate{
			Formula: "(total_antibiotics / distinct_encounters)",
			RoundTo: intPtr(1),
		}
		result, err := buildCalculatedField(calc, aggs, "report.test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// CASE WHEN guard should reference distinct_encounters (agg[0]), not total_antibiotics (agg[1])
		if !strings.Contains(result, "COUNT(") {
			t.Errorf("expected CASE WHEN guard to use COUNT (denominator agg), got: %s", result)
		}
	})

	t.Run("denominator is second aggregation", func(t *testing.T) {
		aggs := []Aggregation{
			{Column: "positive", Function: "sum", Alias: "positive"},
			{Column: "tested", Function: "sum", Alias: "tested"},
		}
		calc := Calculate{
			Formula: "(positive / tested * 100)",
			RoundTo: intPtr(1),
		}
		result, err := buildCalculatedField(calc, aggs, "report.test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// CASE WHEN guard should reference tested (agg[1])
		if !strings.Contains(result, "CASE WHEN") {
			t.Errorf("expected CASE WHEN guard, got: %s", result)
		}
	})

	t.Run("formula with multiplication after division", func(t *testing.T) {
		aggs := []Aggregation{
			{Column: "num", Function: "sum", Alias: "numerator"},
			{Column: "den", Function: "sum", Alias: "denominator"},
		}
		calc := Calculate{
			Formula: "(numerator / denominator * 100)",
			RoundTo: intPtr(2),
		}
		result, err := buildCalculatedField(calc, aggs, "report.test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "CASE WHEN") {
			t.Errorf("expected CASE WHEN guard, got: %s", result)
		}
	})
}

// TestBuildCalculatedFieldColumnV2DenominatorDetection tests v2 denominator detection
func TestBuildCalculatedFieldColumnV2DenominatorDetection(t *testing.T) {
	t.Run("denominator is first aggregation", func(t *testing.T) {
		aggs := []Aggregation{
			{Column: "encounters", Function: "count", Alias: "distinct_encounters"},
			{Column: "antibiotics", Function: "sum", Alias: "total_antibiotics"},
		}
		calc := Calculate{
			Formula: "(total_antibiotics / distinct_encounters)",
			RoundTo: intPtr(1),
		}
		result := buildCalculatedFieldColumn(calc, aggs, "report.test")
		// CASE WHEN guard should reference distinct_encounters (agg[0]), not total_antibiotics (agg[1])
		if !strings.Contains(result, "COUNT(") {
			t.Errorf("expected CASE WHEN guard to use COUNT (denominator agg), got: %s", result)
		}
	})

	t.Run("denominator is second aggregation", func(t *testing.T) {
		aggs := []Aggregation{
			{Column: "positive", Function: "sum", Alias: "positive"},
			{Column: "tested", Function: "sum", Alias: "tested"},
		}
		calc := Calculate{
			Formula: "(positive / tested * 100)",
			RoundTo: intPtr(1),
		}
		result := buildCalculatedFieldColumn(calc, aggs, "report.test")
		if !strings.Contains(result, "CASE WHEN") {
			t.Errorf("expected CASE WHEN guard, got: %s", result)
		}
	})
}

// TestBuildCalculatedFieldSingleAggDivision tests division with only one aggregation (fallback)
func TestBuildCalculatedFieldSingleAggDivision(t *testing.T) {
	aggs := []Aggregation{
		{Column: "cases", Function: "sum", Alias: "cases"},
	}
	calc := Calculate{
		Formula: "(cases / 100)",
	}
	result, err := buildCalculatedField(calc, aggs, "report.test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "as value") {
		t.Errorf("expected fallback formula, got: %s", result)
	}
}

// ============== quoteTableName Tests ==============

func TestQuoteTableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"schema.table", `"schema"."table"`},
		{"my_table", `"my_table"`},
		{"public.users", `"public"."users"`},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := quoteTableName(tc.input)
			if result != tc.expected {
				t.Errorf("quoteTableName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ============== containsOperators Tests ==============

func TestContainsOperators(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"col1 + col2", true},
		{"col1 - col2", true},
		{"col1 * col2", true},
		{"col1 / col2", true},
		{"(col1)", true},
		{"simple_column", false},
		{"column_name", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			result := containsOperators(tc.expr)
			if result != tc.expected {
				t.Errorf("containsOperators(%q) = %v, want %v", tc.expr, result, tc.expected)
			}
		})
	}
}

// ============== buildGroupByColumns Tests (v2) ==============

func TestBuildGroupByColumnsV2(t *testing.T) {
	tests := []struct {
		name     string
		groupBy  []GroupBy
		contains []string
		minCols  int
	}{
		{
			name:     "raw field",
			groupBy:  []GroupBy{{Field: "district"}},
			contains: []string{"district"},
			minCols:  1,
		},
		{
			name:     "month format with date field",
			groupBy:  []GroupBy{{Field: "period_date", Format: "month"}},
			contains: []string{"EXTRACT(YEAR FROM period_date::date)", "EXTRACT(MONTH FROM period_date::date)"},
			minCols:  3, // expression + year extraction + month extraction
		},
		{
			name:     "month_year format with non-date field",
			groupBy:  []GroupBy{{Field: "month", Format: "month_year"}},
			contains: []string{"year", `"month"`},
			minCols:  3,
		},
		{
			name:     "week format with date field",
			groupBy:  []GroupBy{{Field: "period_date", Format: "week"}},
			contains: []string{"EXTRACT(ISOYEAR FROM period_date::date)", "EXTRACT(WEEK FROM period_date::date)"},
			minCols:  3,
		},
		{
			name:     "week_year format with non-date field",
			groupBy:  []GroupBy{{Field: "week", Format: "week_year"}},
			contains: []string{"year", `"week"`},
			minCols:  3,
		},
		{
			name:     "quarter_year with date field",
			groupBy:  []GroupBy{{Field: "period_date", Format: "quarter_year"}},
			contains: []string{"EXTRACT(YEAR FROM period_date::date)", "EXTRACT(QUARTER FROM period_date::date)"},
			minCols:  3,
		},
		{
			name:     "quarter_year with non-date field",
			groupBy:  []GroupBy{{Field: "quarter", Format: "quarter_year"}},
			contains: []string{"year", `"quarter"`},
			minCols:  3,
		},
		{
			name:     "year format with date field",
			groupBy:  []GroupBy{{Field: "period_date", Format: "year"}},
			contains: []string{"EXTRACT(YEAR FROM period_date::date)"},
			minCols:  2,
		},
		{
			name:     "year format with non-date field",
			groupBy:  []GroupBy{{Field: "year", Format: "year"}},
			contains: []string{"year"},
			minCols:  2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cols := buildGroupByColumns(tc.groupBy)
			if len(cols) < tc.minCols {
				t.Errorf("expected at least %d columns, got %d: %v", tc.minCols, len(cols), cols)
			}
			joined := strings.Join(cols, " | ")
			for _, s := range tc.contains {
				if !strings.Contains(joined, s) {
					t.Errorf("expected %q in: %v", s, cols)
				}
			}
		})
	}
}

// ============== buildOrderByColumns Tests (v2) ==============

func TestBuildOrderByColumnsV2(t *testing.T) {
	tests := []struct {
		name     string
		groupBy  []GroupBy
		orderBy  []OrderBy
		contains []string
	}{
		{
			name:     "explicit month_year order",
			orderBy:  []OrderBy{{Field: "month_year", Direction: "ASC"}},
			contains: []string{`"year"::int * 100 + "month"::int`},
		},
		{
			name:     "explicit week order",
			orderBy:  []OrderBy{{Field: "week", Direction: "DESC"}},
			contains: []string{`"year"::int * 100 + "week"::int`, "DESC"},
		},
		{
			name:     "explicit quarter order",
			orderBy:  []OrderBy{{Field: "quarter_year", Direction: "ASC"}},
			contains: []string{`"year"::int * 10 + "quarter"::int`},
		},
		{
			name:     "explicit custom field",
			orderBy:  []OrderBy{{Field: "name", Direction: "DESC"}},
			contains: []string{`"name" DESC`},
		},
		{
			name:     "default month_year groupBy",
			groupBy:  []GroupBy{{Field: "month", Format: "month_year"}},
			contains: []string{`"year"::int * 100`},
		},
		{
			name:     "default week groupBy with date",
			groupBy:  []GroupBy{{Field: "period_date", Format: "week"}},
			contains: []string{"EXTRACT(ISOYEAR FROM"},
		},
		{
			name:     "default quarter_year groupBy",
			groupBy:  []GroupBy{{Field: "quarter", Format: "quarter_year"}},
			contains: []string{`"year"::int * 10`},
		},
		{
			name:     "default year groupBy with date",
			groupBy:  []GroupBy{{Field: "period_date", Format: "year"}},
			contains: []string{"EXTRACT(YEAR FROM period_date::date)"},
		},
		{
			name:     "default year groupBy with non-date",
			groupBy:  []GroupBy{{Field: "year", Format: "year"}},
			contains: []string{`"year" ASC`},
		},
		{
			name:     "default raw field groupBy",
			groupBy:  []GroupBy{{Field: "district"}},
			contains: []string{"district"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cols := buildOrderByColumns(tc.groupBy, tc.orderBy)
			joined := strings.Join(cols, " | ")
			for _, s := range tc.contains {
				if !strings.Contains(joined, s) {
					t.Errorf("expected %q in: %v", s, cols)
				}
			}
		})
	}
}

// ============== applyReportLevelFilters Tests ==============

func TestApplyReportLevelFilters(t *testing.T) {
	baseSql := "SELECT * FROM t WHERE 1=1 {{district_filter}} {{region_filter}} {{facility_filter}} {{year_filter}} {{month_filter}} {{quarter_filter}}"

	tests := []struct {
		name     string
		filters  []string
		contains []string
		absent   []string
	}{
		{
			name:     "district only",
			filters:  []string{"district"},
			contains: []string{"{{district_filter}}"},
			absent:   []string{"{{region_filter}}", "{{year_filter}}", "{{month_filter}}"},
		},
		{
			name:     "year and month",
			filters:  []string{"year", "month"},
			contains: []string{"{{year_filter}}", "{{month_filter}}"},
			absent:   []string{"{{district_filter}}", "{{region_filter}}"},
		},
		{
			name:     "region and facility",
			filters:  []string{"region", "facility"},
			contains: []string{"{{region_filter}}", "{{facility_filter}}"},
			absent:   []string{"{{district_filter}}"},
		},
		{
			name:     "week and quarter",
			filters:  []string{"week", "quarter"},
			contains: []string{"{{week_filter}}", "{{quarter_filter}}"},
			absent:   []string{"{{district_filter}}", "{{year_filter}}"},
		},
		{
			name:    "empty filters removes all",
			filters: []string{},
			absent:  []string{"{{district_filter}}", "{{region_filter}}", "{{year_filter}}"},
		},
		{
			name:    "nil filters removes all",
			filters: nil,
			absent:  []string{"{{district_filter}}", "{{year_filter}}"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := applyReportLevelFilters(baseSql, tc.filters)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in: %s", s, result)
				}
			}
			for _, s := range tc.absent {
				if strings.Contains(result, s) {
					t.Errorf("did NOT expect %q in: %s", s, result)
				}
			}
		})
	}
}

// ============== buildCalculatedFieldColumn Tests (v2) ==============

func TestBuildCalculatedFieldColumnV2(t *testing.T) {
	aggs := []Aggregation{
		{Column: "positive", Function: "sum", Alias: "positive"},
		{Column: "tested", Function: "sum", Alias: "tested"},
	}

	tests := []struct {
		name     string
		calc     Calculate
		contains []string
	}{
		{
			name: "division with resultAlias",
			calc: Calculate{
				Formula:     "(positive / tested * 100)",
				RoundTo:     intPtr(1),
				ResultAlias: "Rate",
			},
			contains: []string{"CASE WHEN", "ROUND(", `"Rate"`},
		},
		{
			name: "no division defaults to value alias",
			calc: Calculate{
				Formula: "(positive + tested)",
			},
			contains: []string{`"value"`},
		},
		{
			name: "division with NUMERIC cast",
			calc: Calculate{
				Formula: "(positive / tested * 100)",
				RoundTo: intPtr(2),
			},
			contains: []string{"CAST(", "AS NUMERIC)"},
		},
	}

	// Single-aggregation division must still honor roundTo (regression: the
	// fallback branch previously dropped ROUND, letting PG return
	// full-precision numerics like 63.6000000000000000 into KPI placeholders).
	t.Run("single-aggregation division honors roundTo", func(t *testing.T) {
		oneAgg := []Aggregation{
			{Column: "anc4_coverage", Function: "avg", Alias: "anc4"},
		}
		calc := Calculate{
			Formula:     "anc4/1",
			RoundTo:     intPtr(1),
			ResultAlias: "ANC4 Coverage",
		}
		result := buildCalculatedFieldColumn(calc, oneAgg, "report.test")
		if !strings.Contains(result, "ROUND(") {
			t.Errorf("expected ROUND() wrap in: %s", result)
		}
		if !strings.Contains(result, `"ANC4 Coverage"`) {
			t.Errorf("expected quoted result alias in: %s", result)
		}
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildCalculatedFieldColumn(tc.calc, aggs, "report.test")
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected %q in: %s", s, result)
				}
			}
		})
	}
}

// ============== buildSelectColumns Tests ==============

func TestBuildSelectColumns(t *testing.T) {
	aggs := []Aggregation{
		{Column: "cases", Function: "sum", Alias: "TotalCases"},
		{Column: "district", Function: "count", Alias: "DistrictCount"},
	}

	t.Run("no groupBy", func(t *testing.T) {
		cols, err := buildSelectColumns(aggs, nil, nil, "schema.table")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cols) != 2 {
			t.Errorf("expected 2 columns, got %d", len(cols))
		}
		joined := strings.Join(cols, " ")
		if !strings.Contains(joined, "SUM(") {
			t.Error("expected SUM in columns")
		}
		if !strings.Contains(joined, "COUNT(") {
			t.Error("expected COUNT in columns")
		}
	})

	t.Run("with groupBy adds label", func(t *testing.T) {
		groupBy := []GroupBy{{Field: "district"}}
		cols, err := buildSelectColumns(aggs, groupBy, nil, "schema.table")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cols) != 3 {
			t.Errorf("expected 3 columns (label + 2 aggs), got %d", len(cols))
		}
		if !strings.Contains(cols[0], `"label"`) {
			t.Errorf("first column should be label, got: %s", cols[0])
		}
	})

	t.Run("with groupBy alias", func(t *testing.T) {
		groupBy := []GroupBy{{Field: "district", Alias: "District"}}
		cols, err := buildSelectColumns(aggs, groupBy, nil, "schema.table")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(cols[0], `"District"`) {
			t.Errorf("first column should use custom alias, got: %s", cols[0])
		}
	})

	t.Run("multiple groupBy", func(t *testing.T) {
		groupBy := []GroupBy{
			{Field: "district"},
			{Field: "year"},
		}
		cols, err := buildSelectColumns(aggs[:1], groupBy, nil, "schema.table")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should have 3 columns: label, label2, TotalCases
		if len(cols) != 3 {
			t.Errorf("expected 3 columns, got %d: %v", len(cols), cols)
		}
	})

	t.Run("invalid aggregation function rejected", func(t *testing.T) {
		badAggs := []Aggregation{
			{Column: "cases", Function: "DROP TABLE", Alias: "Bad"},
		}
		_, err := buildSelectColumns(badAggs, nil, nil, "schema.table")
		if err == nil {
			t.Error("expected error for invalid aggregation function")
		}
	})
}

// ============== YearField Tests ==============

func TestGroupByGetYearField(t *testing.T) {
	tests := []struct {
		name     string
		gb       GroupBy
		expected string
	}{
		{
			name:     "default when empty",
			gb:       GroupBy{Field: "month", Format: "month"},
			expected: "year",
		},
		{
			name:     "custom yearField",
			gb:       GroupBy{Field: "month", Format: "month", YearField: "period_year"},
			expected: "period_year",
		},
		{
			name:     "default when zero value",
			gb:       GroupBy{Field: "week", Format: "week", YearField: ""},
			expected: "year",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.gb.GetYearField()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestGetGroupByExpressionWithYearField(t *testing.T) {
	tests := []struct {
		name     string
		gb       GroupBy
		expected string
	}{
		{
			name:     "month format with custom yearField",
			gb:       GroupBy{Field: "month", Format: "month", YearField: "period_year"},
			expected: `TO_CHAR(MAKE_DATE("period_year"::int, "month"::int, 1), 'Mon YYYY')`,
		},
		{
			name:     "week format with custom yearField",
			gb:       GroupBy{Field: "week", Format: "week", YearField: "period_year"},
			expected: `"period_year"::text || 'W' || LPAD("week"::text, 2, '0')`,
		},
		{
			name:     "quarter format with custom yearField",
			gb:       GroupBy{Field: "quarter", Format: "quarter", YearField: "period_year"},
			expected: `"period_year"::text || 'Q' || "quarter"::text`,
		},
		{
			name:     "year format with non-date field",
			gb:       GroupBy{Field: "period_year", Format: "year"},
			expected: `"period_year"::text`,
		},
		{
			name:     "year format with date field still uses TO_CHAR",
			gb:       GroupBy{Field: "period_date", Format: "year"},
			expected: `TO_CHAR("period_date"::date, 'YYYY')`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getGroupByExpression(tc.gb)
			if result != tc.expected {
				t.Errorf("Expected: %s\nGot: %s", tc.expected, result)
			}
		})
	}
}

func TestBuildGroupByColumnsWithYearField(t *testing.T) {
	tests := []struct {
		name     string
		groupBy  []GroupBy
		contains []string
		absent   []string
	}{
		{
			name:     "month with custom yearField",
			groupBy:  []GroupBy{{Field: "period_month", Format: "month", YearField: "period_year"}},
			contains: []string{`"period_year"`, `"period_month"`},
			absent:   []string{`"year"`},
		},
		{
			name:     "week with custom yearField",
			groupBy:  []GroupBy{{Field: "period_week", Format: "week", YearField: "report_year"}},
			contains: []string{`"report_year"`, `"period_week"`},
			absent:   []string{`"year"`},
		},
		{
			name:     "quarter with custom yearField",
			groupBy:  []GroupBy{{Field: "qtr", Format: "quarter", YearField: "fiscal_year"}},
			contains: []string{`"fiscal_year"`, `"qtr"`},
			absent:   []string{`"year"`},
		},
		{
			name:     "year format with custom yearField",
			groupBy:  []GroupBy{{Field: "period_year", Format: "year", YearField: "period_year"}},
			contains: []string{`"period_year"`},
			absent:   []string{`| "year" |`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cols := buildGroupByColumns(tc.groupBy)
			joined := strings.Join(cols, " | ")
			for _, s := range tc.contains {
				if !strings.Contains(joined, s) {
					t.Errorf("expected %q in: %v", s, cols)
				}
			}
			for _, s := range tc.absent {
				if strings.Contains(joined, s) {
					t.Errorf("unexpected %q in: %v", s, cols)
				}
			}
		})
	}
}

func TestBuildOrderByColumnsWithYearField(t *testing.T) {
	tests := []struct {
		name     string
		groupBy  []GroupBy
		orderBy  []OrderBy
		contains []string
		absent   []string
	}{
		{
			name:     "default month order with custom yearField",
			groupBy:  []GroupBy{{Field: "period_month", Format: "month", YearField: "period_year"}},
			contains: []string{`"period_year"::int`, `"period_month"::int`},
			absent:   []string{`"year"::int`},
		},
		{
			name:     "default week order with custom yearField",
			groupBy:  []GroupBy{{Field: "period_week", Format: "week", YearField: "report_year"}},
			contains: []string{`"report_year"::int`, `"period_week"::int`},
			absent:   []string{`"year"::int`},
		},
		{
			name:     "default quarter order with custom yearField",
			groupBy:  []GroupBy{{Field: "qtr", Format: "quarter", YearField: "fiscal_year"}},
			contains: []string{`"fiscal_year"::int`, `"qtr"::int`},
			absent:   []string{`"year"::int`},
		},
		{
			name:     "default year order with custom yearField",
			groupBy:  []GroupBy{{Field: "period_year", Format: "year", YearField: "period_year"}},
			contains: []string{`"period_year" ASC`},
		},
		{
			name:     "explicit month_year order resolves from groupBy",
			groupBy:  []GroupBy{{Field: "period_month", Format: "month", YearField: "period_year"}},
			orderBy:  []OrderBy{{Field: "month", Direction: "ASC"}},
			contains: []string{`"period_year"::int * 100 + "period_month"::int`},
		},
		{
			name:     "explicit week order resolves from groupBy",
			groupBy:  []GroupBy{{Field: "period_week", Format: "week", YearField: "report_year"}},
			orderBy:  []OrderBy{{Field: "week", Direction: "DESC"}},
			contains: []string{`"report_year"::int * 100 + "period_week"::int`, "DESC"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cols := buildOrderByColumns(tc.groupBy, tc.orderBy)
			joined := strings.Join(cols, " | ")
			for _, s := range tc.contains {
				if !strings.Contains(joined, s) {
					t.Errorf("expected %q in: %v", s, cols)
				}
			}
			for _, s := range tc.absent {
				if strings.Contains(joined, s) {
					t.Errorf("unexpected %q in: %v", s, cols)
				}
			}
		})
	}
}

func TestBuildSQLWithCustomYearField(t *testing.T) {
	query := Query{
		Table: "schema.table",
		Aggregations: []Aggregation{
			{Column: "cases", Function: "sum", Alias: "TotalCases"},
		},
		GroupBy: []GroupBy{
			{Field: "period_month", Format: "month", YearField: "period_year"},
		},
	}

	sql, err := BuildSQL(query, "bar", "schema.table", []string{"district", "year", "month"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should reference period_year, not bare year
	if !strings.Contains(sql, `"period_year"`) {
		t.Errorf("expected custom yearField 'period_year' in SQL, got: %s", sql)
	}
	// Should reference period_month
	if !strings.Contains(sql, `"period_month"`) {
		t.Errorf("expected field 'period_month' in SQL, got: %s", sql)
	}
}

// ============== SeriesBy Tests ==============

// TestBuildSQLWithSeriesBy tests that seriesBy adds series column to SELECT and GROUP BY
func TestBuildSQLWithSeriesBy(t *testing.T) {
	query := Query{
		Table: "report.commodities",
		Aggregations: []Aggregation{
			{Column: "quantity", Function: "sum", Alias: "Quantity"},
		},
		GroupBy: []GroupBy{
			{Field: "month", Format: "month"},
		},
		SeriesBy: "commodity_name",
	}

	sql, err := BuildSQL(query, "line", query.Table, []string{"year", "month"})
	if err != nil {
		t.Fatalf("BuildSQL with seriesBy failed: %v", err)
	}

	sqlLower := strings.ToLower(sql)

	// Should contain series column in SELECT
	if !strings.Contains(sql, `"commodity_name" AS "series"`) {
		t.Errorf("SQL should contain series column, got: %s", sql)
	}

	// Should contain series column in GROUP BY
	if !strings.Contains(sqlLower, `group by`) {
		t.Fatal("SQL should contain GROUP BY")
	}
	// The commodity_name should appear in GROUP BY
	if !strings.Contains(sql, `"commodity_name"`) {
		t.Errorf("SQL should contain commodity_name in GROUP BY, got: %s", sql)
	}

	// Should still contain the aggregation
	if !strings.Contains(sqlLower, "sum(") {
		t.Errorf("SQL should contain SUM aggregation, got: %s", sql)
	}
}

// TestBuildSQLWithoutSeriesBy verifies no series column when seriesBy is empty
func TestBuildSQLWithoutSeriesBy(t *testing.T) {
	query := Query{
		Table: "report.commodities",
		Aggregations: []Aggregation{
			{Column: "quantity", Function: "sum", Alias: "Quantity"},
		},
		GroupBy: []GroupBy{
			{Field: "month", Format: "month"},
		},
	}

	sql, err := BuildSQL(query, "line", query.Table, nil)
	if err != nil {
		t.Fatalf("BuildSQL without seriesBy failed: %v", err)
	}

	if strings.Contains(sql, `AS "series"`) {
		t.Errorf("SQL should NOT contain series column when seriesBy is empty, got: %s", sql)
	}
}
