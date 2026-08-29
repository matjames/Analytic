package main

import (
	"testing"
)

func TestBuildColumnFormatsFromQuery_nilQuery(t *testing.T) {
	if got := buildColumnFormatsFromQuery(nil); got != nil {
		t.Errorf("expected nil for nil query, got %v", got)
	}
}

func TestBuildColumnFormatsFromQuery_noRoundTo(t *testing.T) {
	q := &Query{
		Aggregations: []Aggregation{
			{Column: "cases", Function: "sum", Alias: "Cases"},
		},
	}
	if got := buildColumnFormatsFromQuery(q); got != nil {
		t.Errorf("expected nil when no RoundTo set, got %v", got)
	}
}

func TestBuildColumnFormatsFromQuery_aggRoundTo(t *testing.T) {
	q := &Query{
		Aggregations: []Aggregation{
			{Column: "anc4", Function: "avg", Alias: "ANC4 Coverage", RoundTo: intPtr(1)},
			{Column: "doses", Function: "sum", Alias: "Doses", RoundTo: intPtr(0)},
		},
	}
	formats := buildColumnFormatsFromQuery(q)
	if formats == nil {
		t.Fatal("expected non-nil formats")
	}
	if got := formats["ANC4 Coverage"]; got.Decimals == nil || *got.Decimals != 1 {
		t.Errorf("ANC4 Coverage: expected decimals=1, got %v", got.Decimals)
	}
	if got := formats["Doses"]; got.Decimals == nil || *got.Decimals != 0 {
		t.Errorf("Doses: expected decimals=0, got %v", got.Decimals)
	}
}

func TestBuildColumnFormatsFromQuery_countExcluded(t *testing.T) {
	q := &Query{
		Aggregations: []Aggregation{
			{Column: "district", Function: "count", Alias: "N", RoundTo: intPtr(2)},
			{Column: "district", Function: "count_distinct", Alias: "UniqueN", RoundTo: intPtr(3)},
			{Column: "x", Function: "avg", Alias: "Avg", RoundTo: intPtr(1)},
		},
	}
	formats := buildColumnFormatsFromQuery(q)
	if _, ok := formats["N"]; ok {
		t.Error("COUNT should not appear in column formats")
	}
	if _, ok := formats["UniqueN"]; ok {
		t.Error("COUNT_DISTINCT should not appear in column formats")
	}
	if _, ok := formats["Avg"]; !ok {
		t.Error("AVG with roundTo should appear in column formats")
	}
}

func TestBuildColumnFormatsFromQuery_calculateRoundTo(t *testing.T) {
	q := &Query{
		Calculate: []Calculate{
			{Formula: "(a / b * 100)", ResultAlias: "Rate", RoundTo: intPtr(1)},
			{Formula: "(x + y)", RoundTo: intPtr(2)}, // no resultAlias → keyed as "value"
		},
	}
	formats := buildColumnFormatsFromQuery(q)
	if got := formats["Rate"]; got.Decimals == nil || *got.Decimals != 1 {
		t.Errorf("Rate: expected decimals=1, got %v", got.Decimals)
	}
	if got := formats["value"]; got.Decimals == nil || *got.Decimals != 2 {
		t.Errorf("value (default alias): expected decimals=2, got %v", got.Decimals)
	}
}

func TestBuildColumnFormatsFromQuery_emptyAliasSkipped(t *testing.T) {
	q := &Query{
		Aggregations: []Aggregation{
			{Column: "x", Function: "sum", Alias: "", RoundTo: intPtr(1)},
		},
	}
	if got := buildColumnFormatsFromQuery(q); got != nil {
		t.Errorf("expected nil when alias is empty, got %v", got)
	}
}

// ============== formatSubstitutionValue ==============

func TestFormatSubstitutionValue_noHint(t *testing.T) {
	// Without a decimals hint, %v formatting is used verbatim.
	if got := formatSubstitutionValue("83.2000000000000000", ColumnFormat{}); got != "83.2000000000000000" {
		t.Errorf("expected raw string pass-through, got %q", got)
	}
	if got := formatSubstitutionValue(42, ColumnFormat{}); got != "42" {
		t.Errorf("expected 42, got %q", got)
	}
}

func TestFormatSubstitutionValue_withDecimals(t *testing.T) {
	d0, d1, d2 := 0, 1, 2
	cases := []struct {
		name   string
		value  interface{}
		hint   ColumnFormat
		expect string
	}{
		{"string numeric, trims precision", "83.2000000000000000", ColumnFormat{Decimals: &d1}, "83.2"},
		{"float numeric, rounds down", 83.24, ColumnFormat{Decimals: &d1}, "83.2"},
		{"float numeric, rounds up", 83.26, ColumnFormat{Decimals: &d1}, "83.3"},
		{"trims trailing zeros at decimals=2", 83.2, ColumnFormat{Decimals: &d2}, "83.2"},
		{"integer display when decimals=0", 83.7, ColumnFormat{Decimals: &d0}, "84"},
		{"non-numeric falls back to %v", "N/A", ColumnFormat{Decimals: &d1}, "N/A"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatSubstitutionValue(tc.value, tc.hint)
			if got != tc.expect {
				t.Errorf("got %q, want %q", got, tc.expect)
			}
		})
	}
}
