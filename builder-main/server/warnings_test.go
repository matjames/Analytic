package main

import (
	"strings"
	"testing"
)

func TestCheckGroupByWithoutYear(t *testing.T) {
	tests := []struct {
		name     string
		report   Report
		wantWarn bool
	}{
		{
			name: "month format is fine (includes year by default)",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							GroupBy: []GroupBy{{Field: "period_date", Format: "month"}},
						},
					}},
				}},
			},
			wantWarn: false,
		},
		{
			name: "week format is fine (includes year by default)",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							GroupBy: []GroupBy{{Field: "period_date", Format: "week"}},
						},
					}},
				}},
			},
			wantWarn: false,
		},
		{
			name: "month_noyear warns",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							GroupBy: []GroupBy{{Field: "period_date", Format: "month_noyear"}},
						},
					}},
				}},
			},
			wantWarn: true,
		},
		{
			name: "week_noyear warns",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							GroupBy: []GroupBy{{Field: "period_date", Format: "week_noyear"}},
						},
					}},
				}},
			},
			wantWarn: true,
		},
		{
			name: "quarter_noyear warns",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							GroupBy: []GroupBy{{Field: "period_date", Format: "quarter_noyear"}},
						},
					}},
				}},
			},
			wantWarn: true,
		},
		{
			name: "month with year is fine",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							GroupBy: []GroupBy{
								{Field: "period_date", Format: "year"},
								{Field: "period_date", Format: "month"},
							},
						},
					}},
				}},
			},
			wantWarn: false,
		},
		{
			name: "year only is fine",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							GroupBy: []GroupBy{{Field: "period_date", Format: "year"}},
						},
					}},
				}},
			},
			wantWarn: false,
		},
		{
			name: "no groupBy is fine",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{},
					}},
				}},
			},
			wantWarn: false,
		},
		{
			name: "no query is fine",
			report: Report{
				Sections: []Section{{
					Components: []Component{{Type: "text"}},
				}},
			},
			wantWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := checkGroupByWithoutYear(&tt.report)
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning but got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("expected no warning but got: %v", warnings[0].Message)
			}
		})
	}
}

func TestCheckNoFilters(t *testing.T) {
	tests := []struct {
		name     string
		filters  []string
		wantWarn bool
	}{
		{"no filters warns", nil, true},
		{"empty filters warns", []string{}, true},
		{"year filter is fine", []string{"year"}, false},
		{"district filter is fine", []string{"district"}, false},
		{"only custom filter warns", []string{"custom_col"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &Report{Filters: tt.filters}
			warnings := checkNoFilters(report)
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning but got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("expected no warning but got: %v", warnings[0].Message)
			}
		})
	}
}

func TestCheckTooManyAggregations(t *testing.T) {
	makeAggs := func(n int) []Aggregation {
		aggs := make([]Aggregation, n)
		for i := range aggs {
			aggs[i] = Aggregation{Column: "col", Function: "sum", Alias: "a"}
		}
		return aggs
	}

	tests := []struct {
		name     string
		compType string
		aggCount int
		wantWarn bool
	}{
		{"bar with 9 aggs warns", "bar", 9, true},
		{"line with 10 aggs warns", "line", 10, true},
		{"pie with 9 aggs warns", "pie", 9, true},
		{"bar_line with 12 aggs warns", "bar_line", 12, true},
		{"bar with 8 aggs is fine", "bar", 8, false},
		{"bar with 3 aggs is fine", "bar", 3, false},
		{"table with 10 aggs is fine", "table", 10, false},
		{"text with 10 aggs is fine", "text", 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &Report{
				Sections: []Section{{
					Components: []Component{{
						Type:  tt.compType,
						Query: &Query{Aggregations: makeAggs(tt.aggCount)},
					}},
				}},
			}
			warnings := checkTooManyAggregations(report)
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning but got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("expected no warning but got: %v", warnings[0].Message)
			}
		})
	}
}

func TestCheckRawSQLIgnoresFilters(t *testing.T) {
	tests := []struct {
		name     string
		filters  []string
		compType string
		sql      string
		wantWarn bool
	}{
		{
			"table_advanced without placeholders warns",
			[]string{"year"},
			"table_advanced",
			"SELECT * FROM table",
			true,
		},
		{
			"table_advanced with placeholder is fine",
			[]string{"year"},
			"table_advanced",
			"SELECT * FROM table WHERE {{year_filter}}",
			false,
		},
		{
			"table_advanced with district_filter is fine",
			[]string{"district"},
			"table_advanced",
			"SELECT * FROM table WHERE {{district_filter}}",
			false,
		},
		{
			"no report filters skips check",
			nil,
			"table_advanced",
			"SELECT * FROM table",
			false,
		},
		{
			"non-table_advanced type skipped",
			[]string{"year"},
			"bar",
			"SELECT * FROM table",
			false,
		},
		{
			"table_advanced with no SQL skipped",
			[]string{"year"},
			"table_advanced",
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &Report{
				Filters: tt.filters,
				Sections: []Section{{
					Components: []Component{{
						Type: tt.compType,
						SQL:  tt.sql,
					}},
				}},
			}
			warnings := checkRawSQLIgnoresFilters(report)
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning but got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("expected no warning but got: %v", warnings[0].Message)
			}
		})
	}
}

func TestCheckUndefinedFormulaAlias(t *testing.T) {
	tests := []struct {
		name     string
		aggs     []Aggregation
		calcs    CalculateList
		wantWarn bool
	}{
		{
			"undefined alias warns",
			[]Aggregation{{Column: "c", Function: "sum", Alias: "tested"}},
			CalculateList{{Formula: "(positive / tested * 100)"}},
			true,
		},
		{
			"all aliases defined is fine",
			[]Aggregation{
				{Column: "c1", Function: "sum", Alias: "positive"},
				{Column: "c2", Function: "sum", Alias: "tested"},
			},
			CalculateList{{Formula: "(positive / tested * 100)"}},
			false,
		},
		{
			"no calculate is fine",
			[]Aggregation{{Column: "c", Function: "sum", Alias: "val"}},
			nil,
			false,
		},
		{
			"empty formula is fine",
			[]Aggregation{{Column: "c", Function: "sum", Alias: "val"}},
			CalculateList{{Formula: ""}},
			false,
		},
		{
			"CASE WHEN formula with all aliases defined is fine",
			[]Aggregation{
				{Column: "c1", Function: "sum", Alias: "Tested"},
				{Column: "c2", Function: "sum", Alias: "Confirmed"},
			},
			CalculateList{{Formula: "CASE WHEN Tested IS NULL OR Tested = 0 THEN 0 ELSE (Confirmed * 100.0 / Tested) END"}},
			false,
		},
		{
			"CASE WHEN formula with missing alias warns",
			[]Aggregation{
				{Column: "c1", Function: "sum", Alias: "Tested"},
			},
			CalculateList{{Formula: "CASE WHEN Tested IS NULL OR Tested = 0 THEN 0 ELSE (Confirmed * 100.0 / Tested) END"}},
			true,
		},
		{
			"COALESCE in formula is ignored",
			[]Aggregation{
				{Column: "c1", Function: "sum", Alias: "val"},
			},
			CalculateList{{Formula: "COALESCE(val, 0)"}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &Report{
				Sections: []Section{{
					Components: []Component{{
						Query: &Query{
							Aggregations: tt.aggs,
							Calculate:    tt.calcs,
						},
					}},
				}},
			}
			warnings := checkUndefinedFormulaAlias(report)
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning but got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("expected no warning but got: %v", warnings[0].Message)
			}
		})
	}

	t.Run("multiple undefined aliases collapsed into one warning", func(t *testing.T) {
		report := &Report{
			Sections: []Section{
				{Components: []Component{{
					Query: &Query{
						Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "a"}},
						Calculate:    CalculateList{{Formula: "(missing / a * 100)"}},
					},
				}}},
				{Components: []Component{{
					Query: &Query{
						Aggregations: []Aggregation{{Column: "c", Function: "sum", Alias: "b"}},
						Calculate:    CalculateList{{Formula: "(gone / b * 100)"}},
					},
				}}},
			},
		}
		warnings := checkUndefinedFormulaAlias(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 collapsed warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "2 component(s)") {
			t.Errorf("expected '2 component(s)' in message, got: %s", warnings[0].Message)
		}
	})
}

func TestCheckMissingSectionDescription(t *testing.T) {
	t.Run("single missing description", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{Description: ""}},
		}
		warnings := checkMissingSectionDescription(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "(Section 1)") {
			t.Errorf("expected '(Section 1)' in message, got: %s", warnings[0].Message)
		}
	})

	t.Run("multiple missing descriptions collapsed", func(t *testing.T) {
		report := &Report{
			Sections: []Section{
				{Description: "has desc"},
				{Description: ""},
				{Description: ""},
				{Description: "has desc too"},
				{Description: "   "},
			},
		}
		warnings := checkMissingSectionDescription(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 collapsed warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "(Section 2, 3, 5)") {
			t.Errorf("expected '(Section 2, 3, 5)' in message, got: %s", warnings[0].Message)
		}
	})

	t.Run("whitespace-only warns", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{Description: "   "}},
		}
		warnings := checkMissingSectionDescription(report)
		if len(warnings) == 0 {
			t.Error("expected warning but got none")
		}
	})

	t.Run("all have descriptions is fine", func(t *testing.T) {
		report := &Report{
			Sections: []Section{
				{Description: "Overview"},
				{Description: "Details"},
			},
		}
		warnings := checkMissingSectionDescription(report)
		if len(warnings) > 0 {
			t.Errorf("expected no warning but got: %v", warnings[0].Message)
		}
	})
}

func TestCheckLowReferenceUsage(t *testing.T) {
	tests := []struct {
		name     string
		comps    []Component
		wantWarn bool
	}{
		{
			"fewer than 3 eligible components skips check",
			[]Component{
				{Type: "bar", Query: &Query{}},
				{Type: "text"},
			},
			false,
		},
		{
			"0/5 bars have references warns",
			[]Component{
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
			},
			true,
		},
		{
			"0/4 mixed charts and KPIs warns",
			[]Component{
				{Type: "bar", Query: &Query{}},
				{Type: "line", Query: &Query{}},
				{Type: "text"},
				{Type: "text"},
			},
			true,
		},
		{
			"1/5 bars with reference line is fine (20%)",
			[]Component{
				{Type: "bar", ReferenceLines: []ReferenceLineConfig{{Value: 100, Label: "Target"}}},
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
			},
			false,
		},
		{
			"1/5 text with target is fine (20%)",
			[]Component{
				{Type: "text", Target: &TargetConfig{Value: 80}},
				{Type: "text"},
				{Type: "text"},
				{Type: "bar", Query: &Query{}},
				{Type: "line", Query: &Query{}},
			},
			false,
		},
		{
			"tables and pies not counted",
			[]Component{
				{Type: "table", Query: &Query{}},
				{Type: "pie", Query: &Query{}},
				{Type: "table_advanced", SQL: "SELECT 1"},
			},
			false,
		},
		{
			"severity is low",
			[]Component{
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
				{Type: "bar", Query: &Query{}},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &Report{
				Sections: []Section{{Components: tt.comps}},
			}
			warnings := checkLowReferenceUsage(report)
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning but got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("expected no warning but got: %v", warnings[0].Message)
			}
			if tt.name == "severity is low" && len(warnings) > 0 {
				if warnings[0].Severity != "low" {
					t.Errorf("expected severity 'low', got %q", warnings[0].Severity)
				}
			}
		})
	}
}

func TestValidateReportWarnings_Multiple(t *testing.T) {
	// A report that triggers multiple warnings at once
	report := &Report{
		// No filters → warning 2
		Sections: []Section{
			{
				// No description → warning 6
				Components: []Component{
					{
						Type: "bar",
						Query: &Query{
							// month_noyear → warning 1
							GroupBy: []GroupBy{{Field: "d", Format: "month_noyear"}},
							// 9 aggregations → warning 3
							Aggregations: []Aggregation{
								{Column: "a", Function: "sum", Alias: "a1"},
								{Column: "b", Function: "sum", Alias: "a2"},
								{Column: "c", Function: "sum", Alias: "a3"},
								{Column: "d", Function: "sum", Alias: "a4"},
								{Column: "e", Function: "sum", Alias: "a5"},
								{Column: "f", Function: "sum", Alias: "a6"},
								{Column: "g", Function: "sum", Alias: "a7"},
								{Column: "h", Function: "sum", Alias: "a8"},
								{Column: "i", Function: "sum", Alias: "a9"},
							},
							// references undefined alias → warning 5
							Calculate: CalculateList{{Formula: "(undefined_alias / a1 * 100)"}},
						},
					},
				},
			},
		},
	}

	warnings := ValidateReportWarnings(report)

	// Expect at least 4 warnings: groupBy, no filters, too many aggs, undefined alias, missing description
	if len(warnings) < 4 {
		t.Errorf("expected at least 4 warnings, got %d", len(warnings))
		for _, w := range warnings {
			t.Logf("  - %s", w.Message)
		}
	}
}

func TestCheckCrossTableFilterCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		report   Report
		wantWarn bool
	}{
		{
			name: "multiple tables with filters warns",
			report: Report{
				Filters: []string{"year", "district"},
				Sections: []Section{{
					Components: []Component{
						{Query: &Query{Table: "schema.table_a"}},
						{Query: &Query{Table: "schema.table_b"}},
					},
				}},
			},
			wantWarn: true,
		},
		{
			name: "single table is fine",
			report: Report{
				Filters: []string{"year"},
				Sections: []Section{{
					Components: []Component{
						{Query: &Query{Table: "schema.table_a"}},
						{Query: &Query{Table: "schema.table_a"}},
					},
				}},
			},
			wantWarn: false,
		},
		{
			name: "multiple tables but no filters is fine",
			report: Report{
				Sections: []Section{{
					Components: []Component{
						{Query: &Query{Table: "schema.table_a"}},
						{Query: &Query{Table: "schema.table_b"}},
					},
				}},
			},
			wantWarn: false,
		},
		{
			name: "multiple tables with only custom filters is fine",
			report: Report{
				Filters: []string{"custom_col"},
				Sections: []Section{{
					Components: []Component{
						{Query: &Query{Table: "schema.table_a"}},
						{Query: &Query{Table: "schema.table_b"}},
					},
				}},
			},
			wantWarn: false,
		},
		{
			name: "no query components is fine",
			report: Report{
				Filters: []string{"year"},
				Sections: []Section{{
					Components: []Component{
						{Type: "text", Content: "hello"},
					},
				}},
			},
			wantWarn: false,
		},
		{
			name: "tables across sections warns",
			report: Report{
				Filters: []string{"month"},
				Sections: []Section{
					{Components: []Component{{Query: &Query{Table: "schema.table_a"}}}},
					{Components: []Component{{Query: &Query{Table: "schema.table_b"}}}},
				},
			},
			wantWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := checkCrossTableFilterCompatibility(&tt.report)
			if tt.wantWarn && len(warnings) == 0 {
				t.Error("expected warning but got none")
			}
			if !tt.wantWarn && len(warnings) > 0 {
				t.Errorf("expected no warning but got: %v", warnings[0].Message)
			}
		})
	}
}

func TestCheckChoroplethColumnMapping(t *testing.T) {
	tests := []struct {
		name     string
		report   Report
		wantWarn int // expected number of warnings
	}{
		{
			name: "SQL choropleth without columnMapping warns",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Type: "choropleth",
						SQL:  "SELECT district, SUM(cases) FROM t GROUP BY district",
					}},
				}},
			},
			wantWarn: 1,
		},
		{
			name: "SQL choropleth without filterTable and filters warns",
			report: Report{
				Filters: []string{"year"},
				Sections: []Section{{
					Components: []Component{{
						Type:          "choropleth",
						SQL:           "SELECT district, SUM(cases) FROM t GROUP BY district",
						ColumnMapping: &ColumnMapping{District: "district", Value: "value"},
					}},
				}},
			},
			wantWarn: 1,
		},
		{
			name: "SQL choropleth without both warns twice",
			report: Report{
				Filters: []string{"year"},
				Sections: []Section{{
					Components: []Component{{
						Type: "choropleth",
						SQL:  "SELECT district, SUM(cases) FROM t GROUP BY district",
					}},
				}},
			},
			wantWarn: 2,
		},
		{
			name: "SQL choropleth with both is fine",
			report: Report{
				Filters: []string{"year"},
				Sections: []Section{{
					Components: []Component{{
						Type:          "choropleth",
						SQL:           "SELECT district, SUM(cases) FROM t GROUP BY district",
						ColumnMapping: &ColumnMapping{District: "district", Value: "value"},
						FilterTable:   "schema.table",
					}},
				}},
			},
			wantWarn: 0,
		},
		{
			name: "query-based choropleth is fine",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Type:  "choropleth",
						Query: &Query{Table: "schema.table"},
					}},
				}},
			},
			wantWarn: 0,
		},
		{
			name: "non-choropleth with SQL skipped",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Type: "table_advanced",
						SQL:  "SELECT * FROM t",
					}},
				}},
			},
			wantWarn: 0,
		},
		{
			name: "SQL choropleth without filterTable but no report filters is fine",
			report: Report{
				Sections: []Section{{
					Components: []Component{{
						Type:          "choropleth",
						SQL:           "SELECT district, SUM(cases) FROM t GROUP BY district",
						ColumnMapping: &ColumnMapping{District: "district", Value: "value"},
					}},
				}},
			},
			wantWarn: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := checkChoroplethColumnMapping(&tt.report)
			if len(warnings) != tt.wantWarn {
				t.Errorf("expected %d warning(s), got %d", tt.wantWarn, len(warnings))
				for _, w := range warnings {
					t.Logf("  - %s", w.Message)
				}
			}
		})
	}
}

func TestCheckMissingComponentTitle(t *testing.T) {
	t.Run("components with titles is fine", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{
					{Title: "Fever Cases", Type: "bar"},
					{Title: "Summary", Type: "text"},
				},
			}},
		}
		warnings := checkMissingComponentTitle(report)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(warnings))
		}
	})

	t.Run("single missing title warns", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{
					{Title: "", Type: "bar"},
					{Title: "Summary", Type: "line"},
				},
			}},
		}
		warnings := checkMissingComponentTitle(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "1 component(s)") {
			t.Errorf("expected '1 component(s)' in message, got: %s", warnings[0].Message)
		}
	})

	t.Run("text KPI components exempt from title check", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{
					{Title: "", Type: "text"},
					{Title: "", Type: "text"},
					{Title: "Chart", Type: "bar"},
				},
			}},
		}
		warnings := checkMissingComponentTitle(report)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings (only text components missing titles), got %d", len(warnings))
		}
	})

	t.Run("multiple missing titles collapsed", func(t *testing.T) {
		report := &Report{
			Sections: []Section{
				{Components: []Component{{Title: "", Type: "bar"}}},
				{Components: []Component{{Title: "  ", Type: "line"}, {Title: "", Type: "table"}}},
			},
		}
		warnings := checkMissingComponentTitle(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 collapsed warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "3 component(s)") {
			t.Errorf("expected '3 component(s)' in message, got: %s", warnings[0].Message)
		}
	})
}

func TestCheckSingleComponentTwoColumn(t *testing.T) {
	t.Run("two-column with 2 components is fine", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Layout:     "two-column",
				Components: []Component{{Title: "A"}, {Title: "B"}},
			}},
		}
		warnings := checkSingleComponentTwoColumn(report)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(warnings))
		}
	})

	t.Run("single layout with 1 component is fine", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Layout:     "single",
				Components: []Component{{Title: "A"}},
			}},
		}
		warnings := checkSingleComponentTwoColumn(report)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(warnings))
		}
	})

	t.Run("two-column with 1 component warns", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Layout:     "two-column",
				Components: []Component{{Title: "A"}},
			}},
		}
		warnings := checkSingleComponentTwoColumn(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "Two-column layout") {
			t.Errorf("unexpected message: %s", warnings[0].Message)
		}
	})

	t.Run("multiple sections collapsed", func(t *testing.T) {
		report := &Report{
			Sections: []Section{
				{Layout: "two-column", Components: []Component{{Title: "A"}}},
				{Layout: "two-column", Components: []Component{{Title: "B"}, {Title: "C"}}},
				{Layout: "two-column", Components: []Component{{Title: "D"}}},
			},
		}
		warnings := checkSingleComponentTwoColumn(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 collapsed warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "Section 1, 3") {
			t.Errorf("expected 'Section 1, 3' in message, got: %s", warnings[0].Message)
		}
	})
}

func TestCheckTooManyComponents(t *testing.T) {
	t.Run("6 components is fine", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{{}, {}, {}, {}, {}, {}},
			}},
		}
		warnings := checkTooManyComponents(report)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(warnings))
		}
	})

	t.Run("7 components warns", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{{}, {}, {}, {}, {}, {}, {}},
			}},
		}
		warnings := checkTooManyComponents(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		if warnings[0].Severity != "low" {
			t.Errorf("expected severity 'low', got %q", warnings[0].Severity)
		}
	})

	t.Run("multiple large sections collapsed", func(t *testing.T) {
		report := &Report{
			Sections: []Section{
				{Components: []Component{{}, {}, {}, {}, {}, {}, {}, {}}},
				{Components: []Component{{}, {}}},
				{Components: []Component{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}}},
			},
		}
		warnings := checkTooManyComponents(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 collapsed warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "Section 1 (8), 3 (10)") {
			t.Errorf("expected 'Section 1 (8), 3 (10)' in message, got: %s", warnings[0].Message)
		}
	})
}

func TestCheckDeprecatedRawSQLChoropleth(t *testing.T) {
	t.Run("SQL choropleth warns deprecated", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{{
					Type: "choropleth",
					SQL:  "SELECT district, SUM(cases) FROM t GROUP BY district",
				}},
			}},
		}
		warnings := checkDeprecatedRawSQLChoropleth(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "deprecated") {
			t.Errorf("expected 'deprecated' in message, got: %s", warnings[0].Message)
		}
		if !strings.Contains(warnings[0].Message, "Section 1, Component 1") {
			t.Errorf("expected location in message, got: %s", warnings[0].Message)
		}
	})

	t.Run("query-based choropleth no warning", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{{
					Type:  "choropleth",
					Query: &Query{Table: "schema.table"},
				}},
			}},
		}
		warnings := checkDeprecatedRawSQLChoropleth(report)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(warnings))
		}
	})

	t.Run("non-choropleth with SQL no warning", func(t *testing.T) {
		report := &Report{
			Sections: []Section{{
				Components: []Component{{
					Type: "table_advanced",
					SQL:  "SELECT * FROM t",
				}},
			}},
		}
		warnings := checkDeprecatedRawSQLChoropleth(report)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d", len(warnings))
		}
	})

	t.Run("multiple SQL choropleths collapsed into one warning", func(t *testing.T) {
		report := &Report{
			Sections: []Section{
				{Components: []Component{{
					Type: "choropleth",
					SQL:  "SELECT district, SUM(a) FROM t GROUP BY district",
				}}},
				{Components: []Component{{
					Type: "choropleth",
					SQL:  "SELECT district, SUM(b) FROM t GROUP BY district",
				}}},
			},
		}
		warnings := checkDeprecatedRawSQLChoropleth(report)
		if len(warnings) != 1 {
			t.Fatalf("expected 1 collapsed warning, got %d", len(warnings))
		}
		if !strings.Contains(warnings[0].Message, "Section 1, Component 1") || !strings.Contains(warnings[0].Message, "Section 2, Component 1") {
			t.Errorf("expected both locations in message, got: %s", warnings[0].Message)
		}
	})
}
