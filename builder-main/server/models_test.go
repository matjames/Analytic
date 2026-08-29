package main

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func intPtr(v int) *int { return &v }

// ============== CalculateList UnmarshalYAML Tests ==============

func TestCalculateListUnmarshalYAMLArray(t *testing.T) {
	input := `
- formula: "(a / b * 100)"
  roundTo: 1
  resultAlias: "Rate"
- formula: "(a + b)"
  resultAlias: "Total"
`
	var cl CalculateList
	if err := yaml.Unmarshal([]byte(input), &cl); err != nil {
		t.Fatalf("UnmarshalYAML failed: %v", err)
	}
	if len(cl) != 2 {
		t.Fatalf("expected 2 items, got %d", len(cl))
	}
	if cl[0].Formula != "(a / b * 100)" {
		t.Errorf("expected first formula '(a / b * 100)', got %q", cl[0].Formula)
	}
	if cl[1].ResultAlias != "Total" {
		t.Errorf("expected second alias 'Total', got %q", cl[1].ResultAlias)
	}
}

func TestCalculateListUnmarshalYAMLSingle(t *testing.T) {
	input := `
formula: "(a / b * 100)"
roundTo: 1
`
	var cl CalculateList
	if err := yaml.Unmarshal([]byte(input), &cl); err != nil {
		t.Fatalf("UnmarshalYAML failed: %v", err)
	}
	if len(cl) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cl))
	}
	if cl[0].RoundTo == nil || *cl[0].RoundTo != 1 {
		t.Errorf("expected roundTo 1, got %v", cl[0].RoundTo)
	}
}

func TestCalculateListUnmarshalYAMLInvalid(t *testing.T) {
	input := `"just a plain string"`
	var cl CalculateList
	// Should return error for invalid input (fail fast)
	if err := yaml.Unmarshal([]byte(input), &cl); err == nil {
		t.Fatal("UnmarshalYAML should return error for invalid input")
	}
}

// ============== CalculateList UnmarshalJSON Tests ==============

func TestCalculateListUnmarshalJSONArray(t *testing.T) {
	input := `[{"formula":"(a / b * 100)","roundTo":1},{"formula":"(a + b)"}]`
	var cl CalculateList
	if err := json.Unmarshal([]byte(input), &cl); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if len(cl) != 2 {
		t.Fatalf("expected 2 items, got %d", len(cl))
	}
}

func TestCalculateListUnmarshalJSONSingle(t *testing.T) {
	input := `{"formula":"(a / b * 100)","roundTo":2}`
	var cl CalculateList
	if err := json.Unmarshal([]byte(input), &cl); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if len(cl) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cl))
	}
	if cl[0].RoundTo == nil || *cl[0].RoundTo != 2 {
		t.Errorf("expected roundTo 2, got %v", cl[0].RoundTo)
	}
}

func TestCalculateListUnmarshalJSONNull(t *testing.T) {
	input := `null`
	var cl CalculateList
	if err := json.Unmarshal([]byte(input), &cl); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if cl != nil {
		t.Errorf("expected nil for null input, got %v", cl)
	}
}

func TestCalculateListUnmarshalJSONInvalid(t *testing.T) {
	input := `"invalid"`
	var cl CalculateList
	// Should return error for invalid input (fail fast)
	if err := json.Unmarshal([]byte(input), &cl); err == nil {
		t.Fatal("UnmarshalJSON should return error for invalid input")
	}
}

func TestCalculateListUnmarshalJSONWithAllFields(t *testing.T) {
	input := `{"formula":"(a / b * 100)","roundTo":1,"whenZero":0,"resultAlias":"Rate","color":"#ff0000","chartType":"line"}`
	var cl CalculateList
	if err := json.Unmarshal([]byte(input), &cl); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if len(cl) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cl))
	}
	if cl[0].Color != "#ff0000" {
		t.Errorf("expected color '#ff0000', got %q", cl[0].Color)
	}
	if cl[0].ChartType != "line" {
		t.Errorf("expected chartType 'line', got %q", cl[0].ChartType)
	}
}
