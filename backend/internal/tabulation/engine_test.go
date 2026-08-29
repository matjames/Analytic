package tabulation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestTabulationCrosstabAndChiSquare(t *testing.T) {
	engine := NewEngine()

	// 50 test survey records: Gender x Literacy
	records := make([]map[string]interface{}, 0)
	for i := 0; i < 20; i++ {
		records = append(records, map[string]interface{}{"gender": "Female", "literacy": "Literate", "income": 500})
	}
	for i := 0; i < 10; i++ {
		records = append(records, map[string]interface{}{"gender": "Female", "literacy": "Illiterate", "income": 200})
	}
	for i := 0; i < 15; i++ {
		records = append(records, map[string]interface{}{"gender": "Male", "literacy": "Literate", "income": 600})
	}
	for i := 0; i < 5; i++ {
		records = append(records, map[string]interface{}{"gender": "Male", "literacy": "Illiterate", "income": 250})
	}

	req := TabulationRequest{
		Title:           "Gender by Literacy Crosstabulation",
		RowVariable:     "gender",
		ColVariable:     "literacy",
		MeasureVariable: "income",
		Aggregation:     AggCount,
		DataRecords:     records,
	}

	res, err := engine.Tabulate(context.Background(), req)
	if err != nil {
		t.Fatalf("Tabulate failed: %v", err)
	}

	if res.TotalRecordCount != 50 {
		t.Errorf("Expected 50 total records, got %d", res.TotalRecordCount)
	}
	if len(res.RowCategories) != 2 {
		t.Errorf("Expected 2 row categories (Female, Male), got %d", len(res.RowCategories))
	}
	if len(res.ColCategories) != 2 {
		t.Errorf("Expected 2 col categories (Literate, Illiterate), got %d", len(res.ColCategories))
	}

	// Verify Female count
	if res.RowTotals["Female"] != 30 {
		t.Errorf("Expected 30 females, got %f", res.RowTotals["Female"])
	}
	if res.RowTotals["Male"] != 20 {
		t.Errorf("Expected 20 males, got %f", res.RowTotals["Male"])
	}

	// Verify Chi-Square test generated
	if res.ChiSquareTest == nil {
		t.Errorf("Expected Chi-Square statistical test result on 2x2 table")
	} else {
		if res.ChiSquareTest.DegreesOfFreedom != 1 {
			t.Errorf("Expected 1 degree of freedom on 2x2 table, got %d", res.ChiSquareTest.DegreesOfFreedom)
		}
	}
}
// TestTabulationExportSDMX verifies the SDMX-JSON export produces a parseable
// message with the correct dimensional metadata and a full series set.
func TestTabulationExportSDMX(t *testing.T) {
	e := NewEngine()
	records := []map[string]interface{}{
		{"gender": "Female", "literacy": "Literate"},
		{"gender": "Female", "literacy": "Illiterate"},
		{"gender": "Male", "literacy": "Literate"},
	}
	res, err := e.Tabulate(context.Background(), TabulationRequest{
		Title:       "Gender by Literacy",
		RowVariable: "gender",
		ColVariable: "literacy",
		Aggregation: AggCount,
		DataRecords: records,
	})
	if err != nil {
		t.Fatalf("Tabulate: %v", err)
	}

	b, ctype, err := e.Export(res, ExportFormatSDMX)
	if err != nil {
		t.Fatalf("Export SDMX: %v", err)
	}
	if ctype != "application/json" {
		t.Errorf("expected application/json content type, got %q", ctype)
	}

	var msg sdmxMessage
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatalf("SDMX output is not valid JSON: %v", err)
	}
	if len(msg.Structure.Dimensions.DataSet) != 1 {
		t.Fatalf("expected 1 dataSet dimension, got %d", len(msg.Structure.Dimensions.DataSet))
	}
	if len(msg.Structure.Dimensions.DataSet[0].Values) != 3 { // Female, Male, Total
		t.Errorf("expected 3 dataSet values (incl. Total), got %d", len(msg.Structure.Dimensions.DataSet[0].Values))
	}
	if len(msg.DataSets) != 1 || len(msg.DataSets[0].Series) == 0 {
		t.Fatalf("expected dataset series to be present")
	}
}

// TestTabulationExportCSV verifies the CSV export shape and marginals.
func TestTabulationExportCSV(t *testing.T) {
	e := NewEngine()
	records := []map[string]interface{}{
		{"gender": "Female", "income": 100},
		{"gender": "Female", "income": 200},
		{"gender": "Male", "income": 300},
	}
	res, err := e.Tabulate(context.Background(), TabulationRequest{
		Title:       "Income by Gender",
		RowVariable: "gender",
		Aggregation: AggCount,
		DataRecords: records,
	})
	if err != nil {
		t.Fatalf("Tabulate: %v", err)
	}

	b, ctype, err := e.Export(res, ExportFormatCSV)
	if err != nil {
		t.Fatalf("Export CSV: %v", err)
	}
	if ctype != "text/csv" {
		t.Errorf("expected text/csv content type, got %q", ctype)
	}
	csvOut := string(b)
	if !strings.Contains(csvOut, "Female") || !strings.Contains(csvOut, "Male") {
		t.Errorf("CSV output missing row categories: %s", csvOut)
	}
	if !strings.Contains(csvOut, "Total") {
		t.Errorf("CSV output missing Total marginals: %s", csvOut)
	}
	lines := strings.Split(strings.TrimSpace(csvOut), "\n")
	last := lines[len(lines)-1]
	if !strings.HasSuffix(last, ",3") {
		t.Errorf("CSV grand total row should end in ,3: %q", last)
	}
}

// TestExportUnsupportedFormat ensures unknown formats error clearly.
func TestExportUnsupportedFormat(t *testing.T) {
	e := NewEngine()
	res := &TabulationResult{Title: "x"}
	if _, _, err := e.Export(res, ExportFormatXLSX); err == nil {
		t.Fatal("expected an error for unsupported xlsx format")
	}
}
