package main

import (
	"testing"
)

// TestIsChartTypeIncludesPyramid ensures pyramid is treated as a chart type
// for routing decisions (raw-SQL dispatch, periodLimit defaults, debug logging).
func TestIsChartTypeIncludesPyramid(t *testing.T) {
	if !isChartType("pyramid") {
		t.Error("isChartType(\"pyramid\") = false, want true")
	}
}

// TestTransformerRegistryHasPyramid verifies pyramid maps to ChartTransformer.
func TestTransformerRegistryHasPyramid(t *testing.T) {
	registry := NewTransformerRegistry()
	transformer, ok := registry["pyramid"]
	if !ok {
		t.Fatal("pyramid not registered in transformer registry")
	}
	if _, isChart := transformer.(*ChartTransformer); !isChart {
		t.Errorf("pyramid transformer should be *ChartTransformer, got %T", transformer)
	}
}

// TestPyramidTransformShapesLikeBar: pyramid goes through ChartTransformer,
// so with groupBy it should produce labels + one dataset per non-label column.
// Negation is applied in processComponent (not here), so we only verify shape.
func TestPyramidTransformShapesLikeBar(t *testing.T) {
	tableData := TableData{
		Headers: []string{"AgeGroup", "Male", "Female"},
		Rows: [][]interface{}{
			{"0-4", 120, 115},
			{"5-9", 140, 135},
		},
	}
	component := &Component{
		Type: "pyramid",
		Query: &Query{
			Table: "report.t",
			Aggregations: []Aggregation{
				{Alias: "Male"},
				{Alias: "Female"},
			},
			GroupBy: []GroupBy{{Field: "age_group"}},
		},
	}

	result, err := transformData("pyramid", tableData, component)
	if err != nil {
		t.Fatalf("transformData error: %v", err)
	}
	chart, ok := result.(ChartData)
	if !ok {
		t.Fatalf("expected ChartData, got %T", result)
	}
	if len(chart.Labels) != 2 {
		t.Errorf("labels: got %d, want 2", len(chart.Labels))
	}
	if len(chart.Datasets) != 2 {
		t.Fatalf("datasets: got %d, want 2", len(chart.Datasets))
	}
	if chart.Datasets[0].Label != "Male" || chart.Datasets[1].Label != "Female" {
		t.Errorf("dataset labels = [%s, %s], want [Male, Female]", chart.Datasets[0].Label, chart.Datasets[1].Label)
	}
	// Pre-negation: first dataset should carry positive values as transformed
	if v, _ := toFloat64(chart.Datasets[0].Data[0]); v != 120 {
		t.Errorf("chart transformer pre-negation: dataset[0].Data[0] = %v, want 120", v)
	}
}

// TestPyramidValidationRequires2Aggregations ensures validateComponentForYAML rejects
// anything other than exactly 2 aggregations.
func TestPyramidValidationRequires2Aggregations(t *testing.T) {
	cases := []struct {
		name    string
		aggs    []Aggregation
		wantErr bool
	}{
		{"two aggregations ok", []Aggregation{{Alias: "Male"}, {Alias: "Female"}}, false},
		{"one aggregation rejected", []Aggregation{{Alias: "Male"}}, true},
		{"three aggregations rejected", []Aggregation{{Alias: "A"}, {Alias: "B"}, {Alias: "C"}}, true},
		{"zero aggregations rejected", []Aggregation{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			comp := Component{
				Type: "pyramid",
				Query: &Query{
					Table:        "report.t",
					Aggregations: tc.aggs,
					GroupBy:      []GroupBy{{Field: "age_group"}},
				},
			}
			msg := validateComponentForYAML(comp, 1, 1)
			if tc.wantErr && msg == "" {
				t.Errorf("expected validation error, got none")
			}
			if !tc.wantErr && msg != "" {
				t.Errorf("unexpected validation error: %s", msg)
			}
		})
	}
}

// TestPyramidValidationRequiresGroupBy: pyramid needs at least one groupBy column
// (age group, district, etc.) to have anything to mirror across.
func TestPyramidValidationRequiresGroupBy(t *testing.T) {
	comp := Component{
		Type: "pyramid",
		Query: &Query{
			Table:        "report.t",
			Aggregations: []Aggregation{{Alias: "Male"}, {Alias: "Female"}},
			GroupBy:      nil,
		},
	}
	msg := validateComponentForYAML(comp, 1, 1)
	if msg == "" {
		t.Error("expected validation error for pyramid with no groupBy, got none")
	}
}

// TestPyramidValidationAcceptsRawSQL: pyramid with raw SQL skips structured-query
// validation (2-agg/1-groupBy rule only applies to structured query path).
func TestPyramidValidationAcceptsRawSQL(t *testing.T) {
	comp := Component{
		Type: "pyramid",
		SQL:  "SELECT age_group, male, female FROM report.pop",
	}
	if msg := validateComponentForYAML(comp, 1, 1); msg != "" {
		t.Errorf("pyramid with raw SQL should validate, got error: %s", msg)
	}
}

// TestPyramidNegationOfFirstDataset simulates what processComponent does after
// transformData for pyramid components: negate the first dataset so it mirrors
// left of zero.
func TestPyramidNegationOfFirstDataset(t *testing.T) {
	chart := ChartData{
		Labels: []string{"0-4", "5-9"},
		Datasets: []Dataset{
			{Label: "Male", Data: []interface{}{120, 140}},
			{Label: "Female", Data: []interface{}{115, 135}},
		},
	}

	// Mirror the negation logic from reports.go
	if len(chart.Datasets) > 0 {
		negated := make([]interface{}, len(chart.Datasets[0].Data))
		for i, v := range chart.Datasets[0].Data {
			if f, ok := toFloat64(v); ok {
				negated[i] = -f
			} else {
				negated[i] = v
			}
		}
		chart.Datasets[0].Data = negated
	}

	if v, _ := toFloat64(chart.Datasets[0].Data[0]); v != -120 {
		t.Errorf("first dataset not negated: got %v, want -120", v)
	}
	if v, _ := toFloat64(chart.Datasets[0].Data[1]); v != -140 {
		t.Errorf("first dataset not negated: got %v, want -140", v)
	}
	// Second dataset stays positive (right side of pyramid)
	if v, _ := toFloat64(chart.Datasets[1].Data[0]); v != 115 {
		t.Errorf("second dataset modified: got %v, want 115", v)
	}
}

// TestYAMLValidComponentTypesIncludesPyramid: YAML publish path must accept pyramid.
func TestYAMLValidComponentTypesIncludesPyramid(t *testing.T) {
	if !validComponentTypes["pyramid"] {
		t.Error("validComponentTypes[\"pyramid\"] = false, want true")
	}
}

// TestValidateReportPyramidRules: the parse-time report validator (different path
// from validateComponentForYAML) must enforce the same pyramid invariants.
func TestValidateReportPyramidRules(t *testing.T) {
	baseReport := func(comp Component) *Report {
		return &Report{
			ID:    "test/pyramid",
			Title: "Pyramid Test",
			Sections: []Section{{
				ID:         "s1",
				Layout:     "single",
				Components: []Component{comp},
			}},
		}
	}

	t.Run("valid pyramid passes", func(t *testing.T) {
		errs := validateReport(baseReport(Component{
			Type: "pyramid",
			Query: &Query{
				Table:        "report.pop",
				Aggregations: []Aggregation{{Alias: "Male"}, {Alias: "Female"}},
				GroupBy:      []GroupBy{{Field: "age_group"}},
			},
		}))
		if len(errs) > 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("wrong agg count fails", func(t *testing.T) {
		errs := validateReport(baseReport(Component{
			Type: "pyramid",
			Query: &Query{
				Table:        "report.pop",
				Aggregations: []Aggregation{{Alias: "Only"}},
				GroupBy:      []GroupBy{{Field: "age_group"}},
			},
		}))
		if len(errs) == 0 {
			t.Error("expected error for 1-aggregation pyramid, got none")
		}
	})

	t.Run("missing groupBy fails", func(t *testing.T) {
		errs := validateReport(baseReport(Component{
			Type: "pyramid",
			Query: &Query{
				Table:        "report.pop",
				Aggregations: []Aggregation{{Alias: "Male"}, {Alias: "Female"}},
			},
		}))
		if len(errs) == 0 {
			t.Error("expected error for pyramid with no groupBy, got none")
		}
	})

	t.Run("raw SQL pyramid passes", func(t *testing.T) {
		errs := validateReport(baseReport(Component{
			Type: "pyramid",
			SQL:  "SELECT age_group, male, female FROM report.pop",
		}))
		if len(errs) > 0 {
			t.Errorf("raw SQL pyramid should validate, got %v", errs)
		}
	})
}
