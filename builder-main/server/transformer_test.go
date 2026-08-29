package main

import (
	"math"
	"testing"
)

// ============== toFloat64 Tests ==============

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"float64", float64(3.14), 3.14, true},
		{"float32", float32(2.5), 2.5, true},
		{"int", int(42), 42.0, true},
		{"int64", int64(100), 100.0, true},
		{"int32", int32(7), 7.0, true},
		{"string number", "42.5", 42.5, true},
		{"string integer", "100", 100.0, true},
		{"string with spaces", "  3.14  ", 3.14, true},
		{"empty string", "", 0, false},
		{"whitespace string", "   ", 0, false},
		{"non-numeric string", "hello", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
		{"negative float", float64(-5.5), -5.5, true},
		{"zero", float64(0), 0, true},
		{"large int64", int64(math.MaxInt32 + 1), float64(math.MaxInt32 + 1), true},
		{"string negative", "-10.5", -10.5, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := toFloat64(tc.input)
			if ok != tc.ok {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tc.input, ok, tc.ok)
			}
			if ok && tc.ok && result != tc.expected {
				t.Errorf("toFloat64(%v) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

// ============== toString Tests ==============

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		ok       bool
	}{
		{"string", "hello", "hello", true},
		{"empty string", "", "", true},
		{"bytes", []byte("world"), "world", true},
		{"nil", nil, "", false},
		{"int", 42, "42", true},
		{"float64", 3.14, "3.14", true},
		{"bool", true, "true", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := toString(tc.input)
			if ok != tc.ok {
				t.Errorf("toString(%v) ok = %v, want %v", tc.input, ok, tc.ok)
			}
			if result != tc.expected {
				t.Errorf("toString(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ============== filterTableDataBySelectColumns Tests ==============

func TestFilterTableDataBySelectColumns(t *testing.T) {
	tableData := TableData{
		Headers: []string{"A", "B", "C", "D"},
		Rows: [][]interface{}{
			{1, 2, 3, 4},
			{5, 6, 7, 8},
		},
	}

	t.Run("empty selectColumns returns original", func(t *testing.T) {
		result := filterTableDataBySelectColumns(tableData, nil)
		if len(result.Headers) != 4 {
			t.Errorf("expected 4 headers, got %d", len(result.Headers))
		}
	})

	t.Run("select specific columns", func(t *testing.T) {
		result := filterTableDataBySelectColumns(tableData, []string{"A", "C"})
		if len(result.Headers) != 2 {
			t.Errorf("expected 2 headers, got %d", len(result.Headers))
		}
		if result.Headers[0] != "A" || result.Headers[1] != "C" {
			t.Errorf("expected [A, C], got %v", result.Headers)
		}
		if result.Rows[0][0] != 1 || result.Rows[0][1] != 3 {
			t.Errorf("expected [1, 3], got %v", result.Rows[0])
		}
	})

	t.Run("no matching columns returns original", func(t *testing.T) {
		result := filterTableDataBySelectColumns(tableData, []string{"X", "Y"})
		if len(result.Headers) != 4 {
			t.Errorf("expected original 4 headers when no match, got %d", len(result.Headers))
		}
	})
}

// ============== filterTableDataWithLabel Tests ==============

func TestFilterTableDataWithLabel(t *testing.T) {
	tableData := TableData{
		Headers: []string{"Label", "A", "B", "C"},
		Rows: [][]interface{}{
			{"Jan", 1, 2, 3},
			{"Feb", 4, 5, 6},
		},
	}

	t.Run("empty selectColumns returns original", func(t *testing.T) {
		result := filterTableDataWithLabel(tableData, nil)
		if len(result.Headers) != 4 {
			t.Errorf("expected 4 headers, got %d", len(result.Headers))
		}
	})

	t.Run("keeps label column plus selected", func(t *testing.T) {
		result := filterTableDataWithLabel(tableData, []string{"A", "C"})
		if len(result.Headers) != 3 {
			t.Errorf("expected 3 headers, got %d", len(result.Headers))
		}
		if result.Headers[0] != "Label" {
			t.Errorf("first header should be Label, got %s", result.Headers[0])
		}
		if result.Headers[1] != "A" || result.Headers[2] != "C" {
			t.Errorf("expected [Label, A, C], got %v", result.Headers)
		}
	})

	t.Run("no matching columns returns original", func(t *testing.T) {
		result := filterTableDataWithLabel(tableData, []string{"X"})
		if len(result.Headers) != 4 {
			t.Errorf("expected original 4 headers when only label matched, got %d", len(result.Headers))
		}
	})
}

// ============== transposeTableData Tests ==============

func TestTransposeTableData(t *testing.T) {
	t.Run("insufficient headers returns original", func(t *testing.T) {
		td := TableData{Headers: []string{"A"}, Rows: [][]interface{}{{"val"}}}
		result, err := transposeTableData(td, "Label")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Headers) != 1 {
			t.Errorf("expected unchanged data, got %d headers", len(result.Headers))
		}
	})

	t.Run("empty rows returns original", func(t *testing.T) {
		td := TableData{Headers: []string{"A", "B"}, Rows: [][]interface{}{}}
		result, err := transposeTableData(td, "Label")
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(result.Rows))
		}
	})

	t.Run("short row handles gracefully", func(t *testing.T) {
		td := TableData{
			Headers: []string{"Period", "A", "B"},
			Rows:    [][]interface{}{{"Jan"}}, // row shorter than headers
		}
		result, err := transposeTableData(td, "Indicator")
		if err != nil {
			t.Fatal(err)
		}
		// Should produce 2 rows (A, B) but B values should be nil
		if len(result.Rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(result.Rows))
		}
		if result.Rows[0][1] != nil {
			t.Errorf("expected nil for short row, got %v", result.Rows[0][1])
		}
	})
}

// ============== ChartTransformer Tests ==============

// TestBarLineChartTransformer tests the bar_line chart transformer
func TestBarLineChartTransformer(t *testing.T) {
	transformer := &ChartTransformer{}

	// Create a component with mixed bar and line datasets
	component := &Component{
		Type: "bar_line",
		Data: Data{
			Datasets: []Dataset{
				{
					Label:     "Volume",
					Data:      []interface{}{},
					Color:     "#0f766e",
					ChartType: "bar",
				},
				{
					Label:     "Rate",
					Data:      []interface{}{},
					Color:     "#e74c3c",
					ChartType: "line",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "Month",
				},
			},
		},
	}

	// Create table data with labels and values
	tableData := TableData{
		Headers: []string{"Month", "Volume", "Rate"},
		Rows: [][]interface{}{
			{"Jan", 100, 25.5},
			{"Feb", 150, 30.2},
			{"Mar", 120, 28.1},
		},
	}

	// Transform the data
	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Verify result is ChartData
	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Verify labels
	expectedLabels := []string{"Jan", "Feb", "Mar"}
	if len(chartData.Labels) != len(expectedLabels) {
		t.Errorf("Expected %d labels, got %d", len(expectedLabels), len(chartData.Labels))
	}
	for i, label := range chartData.Labels {
		if label != expectedLabels[i] {
			t.Errorf("Label %d: expected %s, got %s", i, expectedLabels[i], label)
		}
	}

	// Verify datasets count
	if len(chartData.Datasets) != 2 {
		t.Errorf("Expected 2 datasets, got %d", len(chartData.Datasets))
	}

	// Verify first dataset (bar)
	if chartData.Datasets[0].Label != "Volume" {
		t.Errorf("Expected first dataset label 'Volume', got '%s'", chartData.Datasets[0].Label)
	}
	if chartData.Datasets[0].Color != "#0f766e" {
		t.Errorf("Expected first dataset color '#0f766e', got '%s'", chartData.Datasets[0].Color)
	}
	if chartData.Datasets[0].ChartType != "bar" {
		t.Errorf("Expected first dataset chartType 'bar', got '%s'", chartData.Datasets[0].ChartType)
	}
	if len(chartData.Datasets[0].Data) != 3 {
		t.Errorf("Expected 3 data points in first dataset, got %d", len(chartData.Datasets[0].Data))
	}

	// Verify second dataset (line)
	if chartData.Datasets[1].Label != "Rate" {
		t.Errorf("Expected second dataset label 'Rate', got '%s'", chartData.Datasets[1].Label)
	}
	if chartData.Datasets[1].Color != "#e74c3c" {
		t.Errorf("Expected second dataset color '#e74c3c', got '%s'", chartData.Datasets[1].Color)
	}
	if chartData.Datasets[1].ChartType != "line" {
		t.Errorf("Expected second dataset chartType 'line', got '%s'", chartData.Datasets[1].ChartType)
	}
	if len(chartData.Datasets[1].Data) != 3 {
		t.Errorf("Expected 3 data points in second dataset, got %d", len(chartData.Datasets[1].Data))
	}

	// Verify data values are correct
	expectedData := [][]interface{}{
		{100, 150, 120},           // Volume
		{25.5, 30.2, 28.1},        // Rate
	}
	for dsIdx, dataset := range chartData.Datasets {
		for i, expectedVal := range expectedData[dsIdx] {
			if i >= len(dataset.Data) {
				t.Fatalf("Dataset %d: missing data point %d", dsIdx, i)
			}
			if dataset.Data[i] != expectedVal {
				t.Errorf("Dataset %d, point %d: expected %v, got %v", dsIdx, i, expectedVal, dataset.Data[i])
			}
		}
	}
}

// TestBarLineChartTransformerWithoutChartType tests bar_line chart transformer without explicit chartType
func TestBarLineChartTransformerWithoutChartType(t *testing.T) {
	transformer := &ChartTransformer{}

	// Create a component with datasets that don't specify chartType
	component := &Component{
		Type: "bar_line",
		Data: Data{
			Datasets: []Dataset{
				{
					Label: "Metric1",
					Data:  []interface{}{},
					Color: "#0f766e",
				},
				{
					Label: "Metric2",
					Data:  []interface{}{},
					Color: "#e74c3c",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "Period",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"Period", "Metric1", "Metric2"},
		Rows: [][]interface{}{
			{"P1", 10, 20},
			{"P2", 15, 25},
		},
	}

	// Transform the data
	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Verify that even without chartType, the data is transformed correctly
	if len(chartData.Datasets) != 2 {
		t.Errorf("Expected 2 datasets, got %d", len(chartData.Datasets))
	}

	// Datasets should still have empty chartType (omitempty)
	if chartData.Datasets[0].ChartType != "" {
		t.Errorf("Expected empty chartType for first dataset, got '%s'", chartData.Datasets[0].ChartType)
	}
	if chartData.Datasets[1].ChartType != "" {
		t.Errorf("Expected empty chartType for second dataset, got '%s'", chartData.Datasets[1].ChartType)
	}
}

// TestBarLineChartTransformerSingleDataset tests bar_line with single dataset
func TestBarLineChartTransformerSingleDataset(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "bar_line",
		Data: Data{
			Datasets: []Dataset{
				{
					Label:     "OnlyBar",
					Data:      []interface{}{},
					Color:     "#0f766e",
					ChartType: "bar",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "Day",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"Day", "OnlyBar"},
		Rows: [][]interface{}{
			{"Mon", 100},
			{"Tue", 150},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	if len(chartData.Datasets) != 1 {
		t.Errorf("Expected 1 dataset, got %d", len(chartData.Datasets))
	}

	if chartData.Datasets[0].ChartType != "bar" {
		t.Errorf("Expected chartType 'bar', got '%s'", chartData.Datasets[0].ChartType)
	}
}

// TestLineChartTransformer tests line chart transformer
func TestLineChartTransformer(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "line",
		Data: Data{
			Datasets: []Dataset{
				{
					Label: "Cases",
					Data:  []interface{}{},
					Color: "#0f766e",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "period_date",
					Format: "month",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"period_date", "Cases"},
		Rows: [][]interface{}{
			{"2024-01", 100},
			{"2024-02", 120},
			{"2024-03", 110},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	if len(chartData.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(chartData.Labels))
	}
	if len(chartData.Datasets) != 1 {
		t.Errorf("Expected 1 dataset, got %d", len(chartData.Datasets))
	}
}

// TestBarChartTransformer tests bar chart transformer
func TestBarChartTransformer(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "bar",
		Data: Data{
			Datasets: []Dataset{
				{
					Label: "Cases",
					Data:  []interface{}{},
					Color: "#e74c3c",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "district",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"district", "Cases"},
		Rows: [][]interface{}{
			{"Kampala", 500},
			{"Central", 300},
			{"Jinja", 200},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	if len(chartData.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(chartData.Labels))
	}
	if chartData.Labels[0] != "Kampala" {
		t.Errorf("Expected first label 'Kampala', got '%s'", chartData.Labels[0])
	}
}

// TestPieChartTransformer tests pie chart transformer (no GroupBy)
func TestPieChartTransformer(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "pie",
		Data: Data{
			Datasets: []Dataset{
				{
					Label: "Cases",
					Data:  []interface{}{},
					Color: "#0f766e",
				},
			},
		},
		Query: &Query{
			Aggregations: []Aggregation{
				{
					Alias: "Fever",
					Color: "#e74c3c",
				},
				{
					Alias: "Malaria",
					Color: "#2ecc71",
				},
				{
					Alias: "Diarrhea",
					Color: "#f39c12",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"Fever", "Malaria", "Diarrhea"},
		Rows: [][]interface{}{
			{500, 300, 200},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// For pie charts, labels come from headers
	expectedLabels := []string{"Fever", "Malaria", "Diarrhea"}
	if len(chartData.Labels) != len(expectedLabels) {
		t.Errorf("Expected %d labels, got %d", len(expectedLabels), len(chartData.Labels))
	}

	// For pie, each dataset is a color per value
	if len(chartData.Datasets) != 3 {
		t.Errorf("Expected 3 datasets for pie chart, got %d", len(chartData.Datasets))
	}

	// Verify colors from aggregations
	if chartData.Datasets[0].Color != "#e74c3c" {
		t.Errorf("Expected first dataset color '#e74c3c', got '%s'", chartData.Datasets[0].Color)
	}
	if chartData.Datasets[1].Color != "#2ecc71" {
		t.Errorf("Expected second dataset color '#2ecc71', got '%s'", chartData.Datasets[1].Color)
	}
}

// TestPieChartWithDefaultPalette tests pie chart with default color palette fallback
func TestPieChartWithDefaultPalette(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "pie",
		Data: Data{
			Datasets: []Dataset{},
		},
		Query: &Query{
			Aggregations: []Aggregation{
				{Alias: "Category1"},
				{Alias: "Category2"},
				{Alias: "Category3"},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"Category1", "Category2", "Category3"},
		Rows: [][]interface{}{
			{100, 200, 150},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Should have 3 datasets with default palette colors
	if len(chartData.Datasets) != 3 {
		t.Errorf("Expected 3 datasets, got %d", len(chartData.Datasets))
	}

	// Verify colors are from default palette (not empty)
	for i, ds := range chartData.Datasets {
		if ds.Color == "" {
			t.Errorf("Dataset %d should have a color from default palette", i)
		}
	}
}

// TestChartEmptyData tests chart with empty result set
func TestChartEmptyData(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "bar",
		Data: Data{
			Datasets: []Dataset{
				{
					Label: "Cases",
					Data:  []interface{}{},
					Color: "#0f766e",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "district",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"district", "Cases"},
		Rows:    [][]interface{}{},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Should return empty arrays, not nil
	if chartData.Labels == nil {
		t.Error("Labels should be empty array, not nil")
	}
	if len(chartData.Labels) != 0 {
		t.Errorf("Expected 0 labels, got %d", len(chartData.Labels))
	}
	if len(chartData.Datasets) != 0 {
		t.Errorf("Expected 0 datasets, got %d", len(chartData.Datasets))
	}
}

// TestChartNumericLabelType tests that numeric labels are converted to strings
func TestChartInvalidLabelType(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "bar",
		Data: Data{
			Datasets: []Dataset{
				{
					Label: "Cases",
					Data:  []interface{}{},
					Color: "#0f766e",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "year",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"year", "Cases"},
		Rows: [][]interface{}{
			{2024, 100},
			{2025, 150},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Errorf("Expected no error for numeric labels; got: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	if len(chartData.Labels) != 2 {
		t.Errorf("Expected 2 labels; got %d", len(chartData.Labels))
	}

	// Verify numeric labels were converted to strings
	if chartData.Labels[0] != "2024" {
		t.Errorf("Expected label '2024'; got '%s'", chartData.Labels[0])
	}
	if chartData.Labels[1] != "2025" {
		t.Errorf("Expected label '2025'; got '%s'", chartData.Labels[1])
	}
}

// TestChartMissingDatasetConfig tests chart with insufficient dataset definitions
// Now uses default colors for missing datasets instead of erroring
func TestChartMissingDatasetConfig(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "bar",
		Data: Data{
			Datasets: []Dataset{
				// Only 1 dataset defined but table has 2 data columns
				{
					Label: "Cases",
					Data:  []interface{}{},
					Color: "#0f766e",
				},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{
					Field: "district",
				},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"district", "Cases", "Deaths"},
		Rows: [][]interface{}{
			{"Kampala", 500, 10},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Errorf("Expected no error with default color fallback; got: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Should have 2 datasets even though only 1 was configured
	if len(chartData.Datasets) != 2 {
		t.Errorf("Expected 2 datasets; got %d", len(chartData.Datasets))
	}

	// First dataset should use color from config (matched by label)
	if chartData.Datasets[0].Color != "#0f766e" {
		t.Errorf("Expected first dataset to have configured color; got %s", chartData.Datasets[0].Color)
	}

	// Second dataset should use default palette color
	if chartData.Datasets[1].Color == "" {
		t.Error("Expected second dataset to have default color")
	}
}

// TestChartMultipleDatasets tests chart with multiple datasets
func TestChartMultipleDatasets(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "bar",
		Data: Data{
			Datasets: []Dataset{
				{Label: "Cases", Data: []interface{}{}, Color: "#0f766e"},
				{Label: "Deaths", Data: []interface{}{}, Color: "#e74c3c"},
				{Label: "Recovered", Data: []interface{}{}, Color: "#2ecc71"},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{
				{Field: "district"},
			},
		},
	}

	tableData := TableData{
		Headers: []string{"district", "Cases", "Deaths", "Recovered"},
		Rows: [][]interface{}{
			{"Kampala", 500, 10, 400},
			{"Central", 300, 5, 280},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	if len(chartData.Datasets) != 3 {
		t.Errorf("Expected 3 datasets, got %d", len(chartData.Datasets))
	}

	// Verify data distribution
	if chartData.Datasets[0].Data[0] != 500 {
		t.Errorf("Expected first dataset first value 500, got %v", chartData.Datasets[0].Data[0])
	}
	if chartData.Datasets[1].Data[0] != 10 {
		t.Errorf("Expected second dataset first value 10, got %v", chartData.Datasets[1].Data[0])
	}
	if chartData.Datasets[2].Data[0] != 400 {
		t.Errorf("Expected third dataset first value 400, got %v", chartData.Datasets[2].Data[0])
	}
}

// ============== TableTransformer Tests ==============

// TestTableTransformerBasic tests basic table transformation
func TestTableTransformerBasic(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Data: Data{},
	}

	tableData := TableData{
		Headers: []string{"District", "Cases", "Deaths"},
		Rows: [][]interface{}{
			{"Kampala", 500, 10},
			{"Central", 300, 5},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Table transformer returns data as-is
	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	if len(resultData.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(resultData.Headers))
	}
	if len(resultData.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(resultData.Rows))
	}
	if resultData.Headers[0] != "District" {
		t.Errorf("Expected first header 'District', got '%s'", resultData.Headers[0])
	}
}

// TestTableTransformerEmpty tests table transformer with empty data
func TestTableTransformerEmpty(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
	}

	tableData := TableData{
		Headers: []string{"Col1", "Col2"},
		Rows:    [][]interface{}{},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	if len(resultData.Rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(resultData.Rows))
	}
}

// TestTableTransformerMixedTypes tests table with mixed data types
func TestTableTransformerMixedTypes(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
	}

	tableData := TableData{
		Headers: []string{"Name", "Cases", "Rate", "Active"},
		Rows: [][]interface{}{
			{"Kampala", 500, 25.5, true},
			{"Central", 300, 15.2, false},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// Verify data integrity through transformation
	row := resultData.Rows[0]
	if row[0] != "Kampala" {
		t.Errorf("Expected 'Kampala', got %v", row[0])
	}
	if row[1] != 500 {
		t.Errorf("Expected 500, got %v", row[1])
	}
	if row[2] != 25.5 {
		t.Errorf("Expected 25.5, got %v", row[2])
	}
	if row[3] != true {
		t.Errorf("Expected true, got %v", row[3])
	}
}

// ============== MapTransformer Tests ==============

// TestMapTransformerBasic tests basic map transformation
func TestMapTransformerBasic(t *testing.T) {
	transformer := &MapTransformer{}

	component := &Component{
		Type: "map",
		Data: Data{
			Center: []float64{0.3476, 32.5825},
			Zoom:   6,
		},
	}

	tableData := TableData{
		Headers: []string{"lat", "lng", "label", "value"},
		Rows: [][]interface{}{
			{0.3476, 32.5825, "Kampala", 500},
			{0.4500, 33.5500, "Central", 300},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	mapData, ok := result.(MapData)
	if !ok {
		t.Fatal("Expected MapData result")
	}

	if len(mapData.Markers) != 2 {
		t.Errorf("Expected 2 markers, got %d", len(mapData.Markers))
	}
	if mapData.Markers[0].Label != "Kampala" {
		t.Errorf("Expected marker label 'Kampala', got '%s'", mapData.Markers[0].Label)
	}
	if mapData.Markers[0].Value != 500 {
		t.Errorf("Expected marker value 500, got %v", mapData.Markers[0].Value)
	}
	if mapData.Center[0] != 0.3476 {
		t.Errorf("Expected center lat 0.3476, got %v", mapData.Center[0])
	}
	if mapData.Zoom != 6 {
		t.Errorf("Expected zoom 6, got %d", mapData.Zoom)
	}
}

// TestMapTransformerEmpty tests map transformer with empty markers
func TestMapTransformerEmpty(t *testing.T) {
	transformer := &MapTransformer{}

	component := &Component{
		Type: "map",
		Data: Data{
			Center: []float64{0.3476, 32.5825},
			Zoom:   6,
		},
	}

	tableData := TableData{
		Headers: []string{"lat", "lng", "label", "value"},
		Rows:    [][]interface{}{},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	mapData, ok := result.(MapData)
	if !ok {
		t.Fatal("Expected MapData result")
	}

	// Should return empty array, not nil
	if mapData.Markers == nil {
		t.Error("Markers should be empty array, not nil")
	}
	if len(mapData.Markers) != 0 {
		t.Errorf("Expected 0 markers, got %d", len(mapData.Markers))
	}
}

// TestMapTransformerInsufficientColumns tests map with missing required columns
func TestMapTransformerInsufficientColumns(t *testing.T) {
	transformer := &MapTransformer{}

	component := &Component{
		Type: "map",
		Data: Data{
			Center: []float64{0.3476, 32.5825},
			Zoom:   6,
		},
	}

	tableData := TableData{
		Headers: []string{"lat", "lng"},
		Rows: [][]interface{}{
			{0.3476, 32.5825},
		},
	}

	_, err := transformer.Transform(tableData, component)
	if err == nil {
		t.Error("Expected error for insufficient columns")
	}
}

// TestMapTransformerInvalidLatType tests map with non-numeric latitude (returns error)
func TestMapTransformerInvalidLatType(t *testing.T) {
	transformer := &MapTransformer{}

	component := &Component{
		Type: "map",
		Data: Data{
			Center: []float64{0.3476, 32.5825},
			Zoom:   6,
		},
	}

	tableData := TableData{
		Headers: []string{"lat", "lng", "label", "value"},
		Rows: [][]interface{}{
			{"not-a-number", 32.5825, "Kampala", 500},
		},
	}

	_, err := transformer.Transform(tableData, component)
	if err == nil {
		t.Error("Expected error for non-numeric latitude")
	}
}

// TestMapTransformerInvalidLngType tests map with non-numeric longitude (returns error)
func TestMapTransformerInvalidLngType(t *testing.T) {
	transformer := &MapTransformer{}

	component := &Component{
		Type: "map",
		Data: Data{
			Center: []float64{0.3476, 32.5825},
			Zoom:   6,
		},
	}

	tableData := TableData{
		Headers: []string{"lat", "lng", "label", "value"},
		Rows: [][]interface{}{
			{0.3476, "not-a-number", "Kampala", 500},
		},
	}

	_, err := transformer.Transform(tableData, component)
	if err == nil {
		t.Error("Expected error for non-numeric longitude")
	}
}

// TestMapTransformerNumericLabelConverted tests that numeric labels are converted to strings
func TestMapTransformerNumericLabelConverted(t *testing.T) {
	transformer := &MapTransformer{}

	component := &Component{
		Type: "map",
		Data: Data{
			Center: []float64{0.3476, 32.5825},
			Zoom:   6,
		},
	}

	tableData := TableData{
		Headers: []string{"lat", "lng", "label", "value"},
		Rows: [][]interface{}{
			{0.3476, 32.5825, 123, 500},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mapData := result.(MapData)
	if mapData.Markers[0].Label != "123" {
		t.Errorf("Expected label '123', got '%s'", mapData.Markers[0].Label)
	}
}

// ============== ChoroplethTransformer Tests ==============

// TestChoroplethTransformerBasic tests basic choropleth transformation
func TestChoroplethTransformerBasic(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	component := &Component{
		Type: "choropleth",
		Data: Data{
			GeoJSONPath: "/data/uganda-districts.geojson",
			ColorScheme: []string{"#fff5e6", "#ffe5cc", "#ffd6b2", "#ffc699", "#ffb680", "#ffa366", "#ff9050", "#ff7d33"},
		},
	}

	tableData := TableData{
		Headers: []string{"district", "value"},
		Rows: [][]interface{}{
			{"Kampala", 500},
			{"Central", 300},
			{"Jinja", 200},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	choroplethData, ok := result.(ChoroplethData)
	if !ok {
		t.Fatal("Expected ChoroplethData result")
	}

	if len(choroplethData.DistrictValues) != 3 {
		t.Errorf("Expected 3 districts, got %d", len(choroplethData.DistrictValues))
	}
	if choroplethData.DistrictValues["Kampala"] != 500 {
		t.Errorf("Expected Kampala value 500, got %v", choroplethData.DistrictValues["Kampala"])
	}
	if choroplethData.GeoJSONPath != "/data/uganda-districts.geojson" {
		t.Errorf("Expected GeoJSONPath '/data/uganda-districts.geojson', got '%s'", choroplethData.GeoJSONPath)
	}
	if len(choroplethData.ColorScheme) != 8 {
		t.Errorf("Expected 8 colors in scheme, got %d", len(choroplethData.ColorScheme))
	}
}

// TestChoroplethTransformerEmpty tests choropleth with empty districts
func TestChoroplethTransformerEmpty(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	component := &Component{
		Type: "choropleth",
		Data: Data{
			GeoJSONPath: "/data/uganda-districts.geojson",
		},
	}

	tableData := TableData{
		Headers: []string{"district", "value"},
		Rows:    [][]interface{}{},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	choroplethData, ok := result.(ChoroplethData)
	if !ok {
		t.Fatal("Expected ChoroplethData result")
	}

	// Should return empty map, not nil
	if choroplethData.DistrictValues == nil {
		t.Error("DistrictValues should be empty map, not nil")
	}
	if len(choroplethData.DistrictValues) != 0 {
		t.Errorf("Expected 0 districts, got %d", len(choroplethData.DistrictValues))
	}
}

// TestChoroplethTransformerInsufficientColumns tests choropleth with missing value column
func TestChoroplethTransformerInsufficientColumns(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	component := &Component{
		Type: "choropleth",
	}

	tableData := TableData{
		Headers: []string{"district"},
		Rows: [][]interface{}{
			{"Kampala"},
		},
	}

	_, err := transformer.Transform(tableData, component)
	if err == nil {
		t.Error("Expected error for insufficient columns")
	}
}

// TestChoroplethTransformerNumericDistrictConverted tests that numeric districts are converted to strings
func TestChoroplethTransformerNumericDistrictConverted(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	component := &Component{
		Type: "choropleth",
	}

	tableData := TableData{
		Headers: []string{"district", "value"},
		Rows: [][]interface{}{
			{123, 500},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	choroplethData := result.(ChoroplethData)
	if _, ok := choroplethData.DistrictValues["123"]; !ok {
		t.Error("Expected district '123' in map")
	}
}

// TestChoroplethTransformerMixedValues tests choropleth with various value types
func TestChoroplethTransformerMixedValues(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	component := &Component{
		Type: "choropleth",
		Data: Data{
			GeoJSONPath: "/data/uganda-districts.geojson",
		},
	}

	tableData := TableData{
		Headers: []string{"district", "value"},
		Rows: [][]interface{}{
			{"Kampala", 500},
			{"Central", 300.5},
			{"Jinja", 200},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	choroplethData, ok := result.(ChoroplethData)
	if !ok {
		t.Fatal("Expected ChoroplethData result")
	}

	if choroplethData.DistrictValues["Kampala"] != 500 {
		t.Errorf("Expected Kampala value 500, got %v", choroplethData.DistrictValues["Kampala"])
	}
	if choroplethData.DistrictValues["Central"] != 300.5 {
		t.Errorf("Expected Central value 300.5, got %v", choroplethData.DistrictValues["Central"])
	}
}

// TestChoroplethTransformerWithColumnMapping tests choropleth with explicit column mapping
func TestChoroplethTransformerWithColumnMapping(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	// Component with custom SQL column mapping - columns are NOT at positions 0 and 1
	component := &Component{
		Type: "choropleth",
		Data: Data{
			GeoJSONPath: "/data/uganda-districts.geojson",
		},
		ColumnMapping: &ColumnMapping{
			District: "region_name",
			Value:    "total_cases",
		},
	}

	// Data with columns in different order than default (0, 1)
	tableData := TableData{
		Headers: []string{"year", "region_name", "total_cases", "percentage"},
		Rows: [][]interface{}{
			{2024, "Kampala", 500, 25.5},
			{2024, "Central", 300, 15.0},
			{2024, "Jinja", 200, 10.0},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	choroplethData, ok := result.(ChoroplethData)
	if !ok {
		t.Fatal("Expected ChoroplethData result")
	}

	// Verify that column mapping correctly identified district (col 1) and value (col 2)
	if len(choroplethData.DistrictValues) != 3 {
		t.Errorf("Expected 3 district values, got %d", len(choroplethData.DistrictValues))
	}

	if choroplethData.DistrictValues["Kampala"] != 500 {
		t.Errorf("Expected Kampala value 500, got %v", choroplethData.DistrictValues["Kampala"])
	}
	if choroplethData.DistrictValues["Central"] != 300 {
		t.Errorf("Expected Central value 300, got %v", choroplethData.DistrictValues["Central"])
	}
	if choroplethData.DistrictValues["Jinja"] != 200 {
		t.Errorf("Expected Jinja value 200, got %v", choroplethData.DistrictValues["Jinja"])
	}
}

// TestChoroplethTransformerColumnMappingNotFound tests fallback when mapped columns don't exist
func TestChoroplethTransformerColumnMappingNotFound(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	// Component with column mapping that doesn't match any headers
	component := &Component{
		Type: "choropleth",
		Data: Data{
			GeoJSONPath: "/data/uganda-districts.geojson",
		},
		ColumnMapping: &ColumnMapping{
			District: "nonexistent_district",
			Value:    "nonexistent_value",
		},
	}

	// Standard data - should fall back to columns 0 and 1
	tableData := TableData{
		Headers: []string{"district", "value"},
		Rows: [][]interface{}{
			{"Kampala", 500},
			{"Central", 300},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	choroplethData, ok := result.(ChoroplethData)
	if !ok {
		t.Fatal("Expected ChoroplethData result")
	}

	// Should fall back to default columns 0 and 1
	if choroplethData.DistrictValues["Kampala"] != 500 {
		t.Errorf("Expected Kampala value 500 (fallback), got %v", choroplethData.DistrictValues["Kampala"])
	}
}

// TestChoroplethTransformerWithSelectColumns tests choropleth with selectColumns filter
// This ensures calculated columns are used instead of raw aggregations
func TestChoroplethTransformerWithSelectColumns(t *testing.T) {
	transformer := &ChoroplethTransformer{}

	// Simulates SQL output from: aggregations [TotalCases, TotalTested] + calculate [Rate]
	// User wants Rate (calculated), not TotalCases (first aggregation)
	tableData := TableData{
		Headers: []string{"district", "TotalCases", "TotalTested", "Rate"},
		Rows: [][]interface{}{
			{"Kampala", 100, 200, 50.0},
			{"Central", 150, 300, 50.0},
			{"Jinja", 75, 100, 75.0},
		},
	}

	component := &Component{
		Type: "choropleth",
		Query: &Query{
			SelectColumns: []string{"Rate"},
		},
		Data: Data{
			GeoJSONPath: "/data/uganda-districts.geojson",
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	choroplethData, ok := result.(ChoroplethData)
	if !ok {
		t.Fatal("Expected ChoroplethData result")
	}

	// Should use Rate (selectColumns), not TotalCases (first aggregation column)
	if choroplethData.DistrictValues["Kampala"] != 50.0 {
		t.Errorf("Expected Kampala Rate 50.0, got %v", choroplethData.DistrictValues["Kampala"])
	}
	if choroplethData.DistrictValues["Central"] != 50.0 {
		t.Errorf("Expected Central Rate 50.0, got %v", choroplethData.DistrictValues["Central"])
	}
	if choroplethData.DistrictValues["Jinja"] != 75.0 {
		t.Errorf("Expected Jinja Rate 75.0, got %v", choroplethData.DistrictValues["Jinja"])
	}
}

// ============== TransformerRegistry Tests ==============

// TestTransformerRegistryAllTypes tests that all component types are registered
func TestTransformerRegistryAllTypes(t *testing.T) {
	registry := NewTransformerRegistry()

	expectedTypes := []string{"line", "bar", "bar_line", "pie", "pyramid", "table", "map", "choropleth"}

	for _, typeName := range expectedTypes {
		if _, exists := registry[typeName]; !exists {
			t.Errorf("Transformer for type '%s' not registered", typeName)
		}
	}
}

// TestTransformerRegistryHasBarLine tests that bar_line is registered in the transformer registry
func TestTransformerRegistryHasBarLine(t *testing.T) {
	registry := NewTransformerRegistry()

	if _, exists := registry["bar_line"]; !exists {
		t.Error("bar_line transformer not registered in registry")
	}

	transformer, ok := registry["bar_line"].(*ChartTransformer)
	if !ok {
		t.Fatal("bar_line transformer should be a ChartTransformer")
	}

	if transformer == nil {
		t.Error("bar_line transformer should not be nil")
	}
}

// TestTransformDataDispatch tests the main transformData function dispatches correctly
func TestTransformDataDispatch(t *testing.T) {
	tests := []struct {
		name          string
		componentType string
		headers       []string
		rows          [][]interface{}
		shouldError   bool
	}{
		{
			"line",
			"line",
			[]string{"date", "Cases"},
			[][]interface{}{{"2024-01", 100}},
			false,
		},
		{
			"bar",
			"bar",
			[]string{"district", "Cases"},
			[][]interface{}{{"Kampala", 500}},
			false,
		},
		{
			"bar_line",
			"bar_line",
			[]string{"period", "Cases"},
			[][]interface{}{{"P1", 100}},
			false,
		},
		{
			"pie",
			"pie",
			[]string{"Fever", "Malaria"},
			[][]interface{}{{100.0, 200.0}},
			false,
		},
		{
			"table",
			"table",
			[]string{"Col1", "Col2"},
			[][]interface{}{{"A", 1}},
			false,
		},
		{
			"map",
			"map",
			[]string{"lat", "lng", "label", "value"},
			[][]interface{}{{0.3476, 32.5825, "Kampala", 500}},
			false,
		},
		{
			"choropleth",
			"choropleth",
			[]string{"district", "value"},
			[][]interface{}{{"Kampala", 500}},
			false,
		},
		{
			"unknown",
			"unknown_type",
			[]string{"col"},
			[][]interface{}{{"val"}},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pie charts don't use GroupBy
			var query *Query
			if tt.componentType == "pie" {
				query = &Query{
					Aggregations: []Aggregation{
						{Alias: "Fever", Color: "#e74c3c"},
						{Alias: "Malaria", Color: "#2ecc71"},
					},
				}
			} else {
				query = &Query{
					GroupBy: []GroupBy{{Field: "category"}},
				}
			}

			component := &Component{
				Type: tt.componentType,
				Data: Data{
					Datasets: []Dataset{
						{Label: "Data", Data: []interface{}{}, Color: "#0f766e"},
					},
					Center: []float64{0.3476, 32.5825},
					Zoom:   6,
				},
				Query: query,
			}

			tableData := TableData{
				Headers: tt.headers,
				Rows:    tt.rows,
			}

			_, err := transformData(tt.componentType, tableData, component)
			if tt.shouldError && err == nil {
				t.Error("Expected error for unknown component type")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// ============== SelectColumns Filter Tests ==============

// TestSelectColumnsBarChart tests filtering columns in bar chart
func TestSelectColumnsBarChart(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "bar",
		Data: Data{
			Datasets: []Dataset{
				{Label: "Positive", Data: []interface{}{}, Color: "#0f766e"},
				{Label: "Tested", Data: []interface{}{}, Color: "#e74c3c"},
			},
		},
		Query: &Query{
			GroupBy: []GroupBy{{Field: "month"}},
			Aggregations: []Aggregation{
				{Column: "positive", Function: "sum", Alias: "Positive"},
				{Column: "tested", Function: "sum", Alias: "Tested"},
			},
			Calculate: []Calculate{{
				Formula:     "(Positive / Tested * 100)",
				RoundTo:     intPtr(1),
				ResultAlias: "Rate",
				Color:       "#27ae60",
			}},
			SelectColumns: []string{"Rate"}, // Only show the calculated column
		},
	}

	tableData := TableData{
		Headers: []string{"month", "Positive", "Tested", "Rate"},
		Rows: [][]interface{}{
			{"Jan", 100, 500, 20.0},
			{"Feb", 150, 600, 25.0},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Should only have 1 dataset (Rate) since selectColumns filtered out Positive and Tested
	if len(chartData.Datasets) != 1 {
		t.Errorf("Expected 1 dataset (filtered); got %d", len(chartData.Datasets))
	}

	if chartData.Datasets[0].Label != "Rate" {
		t.Errorf("Expected dataset label 'Rate'; got '%s'", chartData.Datasets[0].Label)
	}

	// Should use the calculated column's color
	if chartData.Datasets[0].Color != "#27ae60" {
		t.Errorf("Expected color '#27ae60'; got '%s'", chartData.Datasets[0].Color)
	}

	// Labels (x-axis) should still be present
	if len(chartData.Labels) != 2 {
		t.Errorf("Expected 2 labels; got %d", len(chartData.Labels))
	}
}

// TestSelectColumnsTable tests filtering columns in table
func TestSelectColumnsTable(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Query: &Query{
			Aggregations: []Aggregation{
				{Column: "a", Function: "sum", Alias: "ColA"},
				{Column: "b", Function: "sum", Alias: "ColB"},
				{Column: "c", Function: "sum", Alias: "ColC"},
			},
			SelectColumns: []string{"ColA", "ColC"}, // Only show A and C
		},
	}

	tableData := TableData{
		Headers: []string{"ColA", "ColB", "ColC"},
		Rows: [][]interface{}{
			{100, 200, 300},
			{150, 250, 350},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	filteredData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// Should only have 2 columns (ColA and ColC)
	if len(filteredData.Headers) != 2 {
		t.Errorf("Expected 2 headers; got %d", len(filteredData.Headers))
	}

	if filteredData.Headers[0] != "ColA" || filteredData.Headers[1] != "ColC" {
		t.Errorf("Expected headers [ColA, ColC]; got %v", filteredData.Headers)
	}

	// Rows should also be filtered
	if len(filteredData.Rows[0]) != 2 {
		t.Errorf("Expected 2 values per row; got %d", len(filteredData.Rows[0]))
	}
}

// TestSelectColumnsTableWithGroupBy tests that the label column is preserved
// when selectColumns is used on a table with groupBy
func TestSelectColumnsTableWithGroupBy(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Query: &Query{
			GroupBy: []GroupBy{
				{Field: "district_name", Alias: "label"},
			},
			Aggregations: []Aggregation{
				{Column: "cases", Function: "sum", Alias: "TotalCases"},
				{Column: "deaths", Function: "sum", Alias: "TotalDeaths"},
			},
			SelectColumns: []string{"TotalCases"}, // Only show TotalCases
		},
	}

	tableData := TableData{
		Headers: []string{"label", "TotalCases", "TotalDeaths"},
		Rows: [][]interface{}{
			{"Gasabo", 100, 5},
			{"Kicukiro", 200, 10},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	filteredData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// Should have 2 columns: label (preserved) + TotalCases (selected)
	if len(filteredData.Headers) != 2 {
		t.Errorf("Expected 2 headers; got %d: %v", len(filteredData.Headers), filteredData.Headers)
	}

	if filteredData.Headers[0] != "label" {
		t.Errorf("Expected first header to be 'label'; got %q", filteredData.Headers[0])
	}

	if filteredData.Headers[1] != "TotalCases" {
		t.Errorf("Expected second header to be 'TotalCases'; got %q", filteredData.Headers[1])
	}

	// Rows should have label + selected value
	if len(filteredData.Rows[0]) != 2 {
		t.Errorf("Expected 2 values per row; got %d", len(filteredData.Rows[0]))
	}

	if filteredData.Rows[0][0] != "Gasabo" {
		t.Errorf("Expected first row label 'Gasabo'; got %v", filteredData.Rows[0][0])
	}
}

// TestSelectColumnsPieChart tests filtering columns in pie chart (no groupBy)
func TestSelectColumnsPieChart(t *testing.T) {
	transformer := &ChartTransformer{}

	component := &Component{
		Type: "pie",
		Data: Data{},
		Query: &Query{
			Aggregations: []Aggregation{
				{Column: "a", Function: "sum", Alias: "SegmentA", Color: "#ff0000"},
				{Column: "b", Function: "sum", Alias: "SegmentB", Color: "#00ff00"},
				{Column: "c", Function: "sum", Alias: "SegmentC", Color: "#0000ff"},
			},
			SelectColumns: []string{"SegmentA", "SegmentC"}, // Only show A and C
		},
	}

	tableData := TableData{
		Headers: []string{"SegmentA", "SegmentB", "SegmentC"},
		Rows: [][]interface{}{
			{100, 200, 300},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Should only have 2 labels (SegmentA and SegmentC)
	if len(chartData.Labels) != 2 {
		t.Errorf("Expected 2 labels; got %d", len(chartData.Labels))
	}

	if chartData.Labels[0] != "SegmentA" || chartData.Labels[1] != "SegmentC" {
		t.Errorf("Expected labels [SegmentA, SegmentC]; got %v", chartData.Labels)
	}
}

// ============== Pivot Transformation Tests ==============

// TestTransposeTableBasic tests the transpose transformation
// Input:  period    | Malaria | Pneumonia | Diarrhoea
//
//	Jan 2024  | 150     | 80        | 200
//	Feb 2024  | 180     | 95        | 220
//
// Output: Indicator | Jan 2024 | Feb 2024
//
//	Malaria   | 150      | 180
//	Pneumonia | 80       | 95
//	Diarrhoea | 200      | 220
func TestTransposeTableBasic(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Query: &Query{
			GroupBy: []GroupBy{{Field: "period", Format: "month"}}, // Has groupBy, so first column is period
			Transpose: &TransposeConfig{
				Enabled: true,
			},
		},
	}

	tableData := TableData{
		Headers: []string{"period", "Malaria", "Pneumonia", "Diarrhoea"},
		Rows: [][]interface{}{
			{"Jan 2024", 150, 80, 200},
			{"Feb 2024", 180, 95, 220},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// Check headers: ["Indicator", "Jan 2024", "Feb 2024"]
	expectedHeaders := []string{"Indicator", "Jan 2024", "Feb 2024"}
	if len(resultData.Headers) != len(expectedHeaders) {
		t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(resultData.Headers))
	}
	for i, h := range expectedHeaders {
		if resultData.Headers[i] != h {
			t.Errorf("Expected header[%d] = %q, got %q", i, h, resultData.Headers[i])
		}
	}

	// Check rows: 3 rows (Malaria, Pneumonia, Diarrhoea)
	if len(resultData.Rows) != 3 {
		t.Fatalf("Expected 3 rows, got %d", len(resultData.Rows))
	}

	// Check Malaria row
	malariaRow := resultData.Rows[0]
	if malariaRow[0] != "Malaria" {
		t.Errorf("Expected first row label to be 'Malaria', got %v", malariaRow[0])
	}
	if malariaRow[1] != 150 {
		t.Errorf("Expected Malaria Jan to be 150, got %v", malariaRow[1])
	}
	if malariaRow[2] != 180 {
		t.Errorf("Expected Malaria Feb to be 180, got %v", malariaRow[2])
	}

	// Check Pneumonia row
	pneumoniaRow := resultData.Rows[1]
	if pneumoniaRow[0] != "Pneumonia" {
		t.Errorf("Expected second row label to be 'Pneumonia', got %v", pneumoniaRow[0])
	}
	if pneumoniaRow[1] != 80 {
		t.Errorf("Expected Pneumonia Jan to be 80, got %v", pneumoniaRow[1])
	}
	if pneumoniaRow[2] != 95 {
		t.Errorf("Expected Pneumonia Feb to be 95, got %v", pneumoniaRow[2])
	}

	// Check Diarrhoea row
	diarrhoeaRow := resultData.Rows[2]
	if diarrhoeaRow[0] != "Diarrhoea" {
		t.Errorf("Expected third row label to be 'Diarrhoea', got %v", diarrhoeaRow[0])
	}
	if diarrhoeaRow[1] != 200 {
		t.Errorf("Expected Diarrhoea Jan to be 200, got %v", diarrhoeaRow[1])
	}
	if diarrhoeaRow[2] != 220 {
		t.Errorf("Expected Diarrhoea Feb to be 220, got %v", diarrhoeaRow[2])
	}
}

// TestTransposeTableEmpty tests transpose with empty data
func TestTransposeTableEmpty(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Query: &Query{
			GroupBy: []GroupBy{{Field: "period"}},
			Transpose: &TransposeConfig{
				Enabled: true,
			},
		},
	}

	tableData := TableData{
		Headers: []string{"period", "Malaria"},
		Rows:    [][]interface{}{},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// Should return original data unchanged
	if len(resultData.Rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(resultData.Rows))
	}
}

// TestTransposeTableSingleColumn tests transpose with only one column (edge case)
func TestTransposeTableSingleColumn(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Query: &Query{
			GroupBy: []GroupBy{{Field: "period"}},
			Transpose: &TransposeConfig{
				Enabled: true,
			},
		},
	}

	tableData := TableData{
		Headers: []string{"period"},
		Rows: [][]interface{}{
			{"Jan 2024"},
			{"Feb 2024"},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// With only one column, transpose returns original data
	if len(resultData.Headers) != 1 {
		t.Errorf("Expected 1 header, got %d", len(resultData.Headers))
	}
}

// TestTransposeTableCustomRowLabel tests transpose with custom row label
func TestTransposeTableCustomRowLabel(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Query: &Query{
			GroupBy: []GroupBy{{Field: "period"}},
			Transpose: &TransposeConfig{
				Enabled:  true,
				RowLabel: "Disease", // Custom label instead of "Indicator"
			},
		},
	}

	tableData := TableData{
		Headers: []string{"period", "Malaria", "Pneumonia"},
		Rows: [][]interface{}{
			{"Jan 2024", 150, 80},
			{"Feb 2024", 180, 95},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// Check first header is custom label "Disease"
	if resultData.Headers[0] != "Disease" {
		t.Errorf("Expected first header to be 'Disease', got %q", resultData.Headers[0])
	}

	// Verify rest of headers are period values
	if resultData.Headers[1] != "Jan 2024" {
		t.Errorf("Expected second header to be 'Jan 2024', got %q", resultData.Headers[1])
	}
	if resultData.Headers[2] != "Feb 2024" {
		t.Errorf("Expected third header to be 'Feb 2024', got %q", resultData.Headers[2])
	}
}

// TestTransposeTableNoGroupBy tests transpose without groupBy (all columns are metrics)
// This is the case where query has aggregations but no groupBy, producing a single row
// Input:  cc     | ff
//
//	28944  | 28818
//
// Output: Indicator | Total
//
//	cc        | 28944
//	ff        | 28818
func TestTransposeTableNoGroupBy(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
		Query: &Query{
			// No GroupBy - all columns are aggregations
			Aggregations: []Aggregation{
				{Column: "col1", Function: "sum", Alias: "cc"},
				{Column: "col2", Function: "sum", Alias: "ff"},
			},
			Transpose: &TransposeConfig{
				Enabled:  true,
				RowLabel: "Indicator",
			},
		},
	}

	// Data without groupBy - single row with just aggregated values
	tableData := TableData{
		Headers: []string{"cc", "ff"},
		Rows: [][]interface{}{
			{28944, 28818},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultData, ok := result.(TableData)
	if !ok {
		t.Fatal("Expected TableData result")
	}

	// Check headers: ["Indicator", "Total"]
	if len(resultData.Headers) != 2 {
		t.Fatalf("Expected 2 headers, got %d: %v", len(resultData.Headers), resultData.Headers)
	}
	if resultData.Headers[0] != "Indicator" {
		t.Errorf("Expected first header to be 'Indicator', got %q", resultData.Headers[0])
	}
	if resultData.Headers[1] != "Total" {
		t.Errorf("Expected second header to be 'Total', got %q", resultData.Headers[1])
	}

	// Check rows: 2 rows (cc, ff)
	if len(resultData.Rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(resultData.Rows))
	}

	// Check cc row
	ccRow := resultData.Rows[0]
	if ccRow[0] != "cc" {
		t.Errorf("Expected first row label to be 'cc', got %v", ccRow[0])
	}
	if ccRow[1] != 28944 {
		t.Errorf("Expected cc value to be 28944, got %v", ccRow[1])
	}

	// Check ff row
	ffRow := resultData.Rows[1]
	if ffRow[0] != "ff" {
		t.Errorf("Expected second row label to be 'ff', got %v", ffRow[0])
	}
	if ffRow[1] != 28818 {
		t.Errorf("Expected ff value to be 28818, got %v", ffRow[1])
	}
}

// ============== ChartTransformer Raw SQL Tests ==============

// TestChartTransformer_RawSQL_BarGroupBy tests bar chart with raw SQL (no Query)
func TestChartTransformer_RawSQL_BarGroupBy(t *testing.T) {
	transformer := &ChartTransformer{}
	component := &Component{
		Type: "bar",
		SQL:  "SELECT month, SUM(cases) as cases, SUM(deaths) as deaths FROM report.data GROUP BY month",
	}

	tableData := TableData{
		Headers: []string{"month", "cases", "deaths"},
		Rows: [][]interface{}{
			{"Jan", 10, 2},
			{"Feb", 20, 3},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Labels from column 0 (hasGroupBy=true for bar)
	if len(chartData.Labels) != 2 {
		t.Fatalf("Expected 2 labels, got %d", len(chartData.Labels))
	}
	if chartData.Labels[0] != "Jan" || chartData.Labels[1] != "Feb" {
		t.Errorf("Expected labels [Jan, Feb], got %v", chartData.Labels)
	}

	// 2 datasets (columns 1 and 2)
	if len(chartData.Datasets) != 2 {
		t.Fatalf("Expected 2 datasets, got %d", len(chartData.Datasets))
	}
	if chartData.Datasets[0].Label != "cases" {
		t.Errorf("Expected first dataset label 'cases', got %q", chartData.Datasets[0].Label)
	}
	if chartData.Datasets[1].Label != "deaths" {
		t.Errorf("Expected second dataset label 'deaths', got %q", chartData.Datasets[1].Label)
	}

	// Verify data values
	if len(chartData.Datasets[0].Data) != 2 {
		t.Fatalf("Expected 2 data points, got %d", len(chartData.Datasets[0].Data))
	}
	if chartData.Datasets[0].Data[0] != 10 || chartData.Datasets[0].Data[1] != 20 {
		t.Errorf("Expected cases data [10, 20], got %v", chartData.Datasets[0].Data)
	}
	if chartData.Datasets[1].Data[0] != 2 || chartData.Datasets[1].Data[1] != 3 {
		t.Errorf("Expected deaths data [2, 3], got %v", chartData.Datasets[1].Data)
	}
}

// TestChartTransformer_RawSQL_PieNoGroupBy tests pie chart with raw SQL (headers as labels)
func TestChartTransformer_RawSQL_PieNoGroupBy(t *testing.T) {
	transformer := &ChartTransformer{}
	component := &Component{
		Type: "pie",
		SQL:  "SELECT SUM(male) as Male, SUM(female) as Female, SUM(other) as Other FROM report.data",
	}

	tableData := TableData{
		Headers: []string{"Male", "Female", "Other"},
		Rows: [][]interface{}{
			{40, 35, 25},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Labels from headers (hasGroupBy=false for pie)
	if len(chartData.Labels) != 3 {
		t.Fatalf("Expected 3 labels, got %d", len(chartData.Labels))
	}
	if chartData.Labels[0] != "Male" || chartData.Labels[1] != "Female" || chartData.Labels[2] != "Other" {
		t.Errorf("Expected labels [Male, Female, Other], got %v", chartData.Labels)
	}

	// 3 datasets (one per label, pie single-row mode)
	if len(chartData.Datasets) != 3 {
		t.Fatalf("Expected 3 datasets, got %d", len(chartData.Datasets))
	}
	if chartData.Datasets[0].Label != "Male" {
		t.Errorf("Expected first dataset label 'Male', got %q", chartData.Datasets[0].Label)
	}

	// Each dataset has one data point
	if len(chartData.Datasets[0].Data) != 1 || chartData.Datasets[0].Data[0] != 40 {
		t.Errorf("Expected Male data [40], got %v", chartData.Datasets[0].Data)
	}
	if len(chartData.Datasets[1].Data) != 1 || chartData.Datasets[1].Data[0] != 35 {
		t.Errorf("Expected Female data [35], got %v", chartData.Datasets[1].Data)
	}
	if len(chartData.Datasets[2].Data) != 1 || chartData.Datasets[2].Data[0] != 25 {
		t.Errorf("Expected Other data [25], got %v", chartData.Datasets[2].Data)
	}
}

// TestChartTransformer_RawSQL_LineGroupBy tests line chart with raw SQL
func TestChartTransformer_RawSQL_LineGroupBy(t *testing.T) {
	transformer := &ChartTransformer{}
	component := &Component{
		Type: "line",
		SQL:  "SELECT week, SUM(value) as total FROM report.data GROUP BY week ORDER BY week",
	}

	tableData := TableData{
		Headers: []string{"week", "total"},
		Rows: [][]interface{}{
			{"W01", 100},
			{"W02", 150},
			{"W03", 120},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Line with SQL → hasGroupBy=true (column 0 as labels)
	if len(chartData.Labels) != 3 {
		t.Fatalf("Expected 3 labels, got %d", len(chartData.Labels))
	}
	if chartData.Labels[0] != "W01" {
		t.Errorf("Expected first label 'W01', got %q", chartData.Labels[0])
	}

	if len(chartData.Datasets) != 1 {
		t.Fatalf("Expected 1 dataset, got %d", len(chartData.Datasets))
	}
	if chartData.Datasets[0].Label != "total" {
		t.Errorf("Expected dataset label 'total', got %q", chartData.Datasets[0].Label)
	}
}

// TestChartTransformer_RawSQL_EmptyRows tests chart with SQL and empty rows
func TestChartTransformer_RawSQL_EmptyRows(t *testing.T) {
	transformer := &ChartTransformer{}
	component := &Component{
		Type: "bar",
		SQL:  "SELECT month, SUM(cases) FROM report.data GROUP BY month",
	}

	tableData := TableData{
		Headers: []string{"month", "cases"},
		Rows:    [][]interface{}{},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatal("Expected ChartData result")
	}

	// Should return empty arrays, not nil
	if chartData.Labels == nil {
		t.Error("Labels should be empty slice, not nil")
	}
	if chartData.Datasets == nil {
		t.Error("Datasets should be empty slice, not nil")
	}
	if len(chartData.Labels) != 0 {
		t.Errorf("Expected 0 labels, got %d", len(chartData.Labels))
	}
	if len(chartData.Datasets) != 0 {
		t.Errorf("Expected 0 datasets, got %d", len(chartData.Datasets))
	}
}

// TestChartTransformer_RawSQL_DefaultColors tests that default palette is applied for SQL charts
func TestChartTransformer_RawSQL_DefaultColors(t *testing.T) {
	transformer := &ChartTransformer{}

	// Bar chart with SQL — no aggregation colors available
	component := &Component{
		Type: "bar",
		SQL:  "SELECT month, SUM(a) as a, SUM(b) as b FROM report.data GROUP BY month",
	}

	tableData := TableData{
		Headers: []string{"month", "a", "b"},
		Rows: [][]interface{}{
			{"Jan", 10, 20},
		},
	}

	result, err := transformer.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData := result.(ChartData)

	defaultPalette := []string{
		"#0f766e", "#e74c3c", "#2ecc71", "#d97706", "#6366f1",
		"#14b8a6", "#334155", "#f59e0b", "#8b5cf6", "#0d9488",
	}

	for i, ds := range chartData.Datasets {
		if ds.Color == "" {
			t.Errorf("Dataset %d should have a color", i)
		}
		if ds.Color != defaultPalette[i%len(defaultPalette)] {
			t.Errorf("Dataset %d color = %q, want %q (from default palette)", i, ds.Color, defaultPalette[i%len(defaultPalette)])
		}
	}

	// Pie chart with SQL — should also get default colors
	pieComponent := &Component{
		Type: "pie",
		SQL:  "SELECT SUM(x) as X, SUM(y) as Y FROM report.data",
	}

	pieTableData := TableData{
		Headers: []string{"X", "Y"},
		Rows:    [][]interface{}{{50, 30}},
	}

	pieResult, err := transformer.Transform(pieTableData, pieComponent)
	if err != nil {
		t.Fatalf("Pie transform failed: %v", err)
	}

	pieChartData := pieResult.(ChartData)
	for i, ds := range pieChartData.Datasets {
		if ds.Color == "" {
			t.Errorf("Pie dataset %d should have a color", i)
		}
		if ds.Color != defaultPalette[i%len(defaultPalette)] {
			t.Errorf("Pie dataset %d color = %q, want %q", i, ds.Color, defaultPalette[i%len(defaultPalette)])
		}
	}
}

// ============== SeriesBy Pivot Tests ==============

func TestChartTransformerSeriesByPivot(t *testing.T) {
	ct := &ChartTransformer{}

	// Flat rows: [label, series, value]
	tableData := TableData{
		Headers: []string{"label", "series", "Quantity"},
		Rows: [][]interface{}{
			{"Jan", "ACTs", 500},
			{"Jan", "RDTs", 300},
			{"Feb", "ACTs", 600},
			{"Feb", "RDTs", 350},
		},
	}

	component := &Component{
		Type: "line",
		Query: &Query{
			SeriesBy: "commodity_name",
			GroupBy:  []GroupBy{{Field: "month", Format: "month"}},
			Aggregations: []Aggregation{
				{Column: "quantity", Function: "sum", Alias: "Quantity"},
			},
		},
	}

	result, err := ct.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform with seriesBy failed: %v", err)
	}

	chartData, ok := result.(ChartData)
	if !ok {
		t.Fatalf("expected ChartData, got %T", result)
	}

	// Check labels
	if len(chartData.Labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(chartData.Labels))
	}
	if chartData.Labels[0] != "Jan" || chartData.Labels[1] != "Feb" {
		t.Errorf("expected labels [Jan, Feb], got %v", chartData.Labels)
	}

	// Check datasets: one per series value
	if len(chartData.Datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(chartData.Datasets))
	}
	if chartData.Datasets[0].Label != "ACTs" {
		t.Errorf("expected first dataset label 'ACTs', got %q", chartData.Datasets[0].Label)
	}
	if chartData.Datasets[1].Label != "RDTs" {
		t.Errorf("expected second dataset label 'RDTs', got %q", chartData.Datasets[1].Label)
	}

	// Check data values
	actsData := chartData.Datasets[0].Data
	if len(actsData) != 2 || actsData[0] != 500 || actsData[1] != 600 {
		t.Errorf("ACTs data: expected [500, 600], got %v", actsData)
	}
	rdtsData := chartData.Datasets[1].Data
	if len(rdtsData) != 2 || rdtsData[0] != 300 || rdtsData[1] != 350 {
		t.Errorf("RDTs data: expected [300, 350], got %v", rdtsData)
	}

	// Check colors assigned
	if chartData.Datasets[0].Color == "" || chartData.Datasets[1].Color == "" {
		t.Error("datasets should have colors assigned")
	}
}

func TestChartTransformerSeriesByMissingCombinations(t *testing.T) {
	ct := &ChartTransformer{}

	// ACTs has Jan+Feb, RDTs only has Jan — Feb should be filled with 0
	tableData := TableData{
		Headers: []string{"label", "series", "Quantity"},
		Rows: [][]interface{}{
			{"Jan", "ACTs", 500},
			{"Jan", "RDTs", 300},
			{"Feb", "ACTs", 600},
		},
	}

	component := &Component{
		Type: "line",
		Query: &Query{
			SeriesBy: "commodity_name",
			GroupBy:  []GroupBy{{Field: "month", Format: "month"}},
			Aggregations: []Aggregation{
				{Column: "quantity", Function: "sum", Alias: "Quantity"},
			},
		},
	}

	result, err := ct.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData := result.(ChartData)

	// RDTs should have 0 for Feb
	rdtsData := chartData.Datasets[1].Data
	if len(rdtsData) != 2 {
		t.Fatalf("expected 2 data points for RDTs, got %d", len(rdtsData))
	}
	if rdtsData[1] != 0 {
		t.Errorf("RDTs Feb should be 0 (missing combo), got %v", rdtsData[1])
	}
}

func TestChartTransformerSeriesByUsesLastValueColumn(t *testing.T) {
	ct := &ChartTransformer{}

	// Multiple value columns: [label, series, agg1, calculated]
	// Should use last column (calculated)
	tableData := TableData{
		Headers: []string{"label", "series", "Total", "Rate"},
		Rows: [][]interface{}{
			{"Jan", "ACTs", 500, 85.5},
			{"Jan", "RDTs", 300, 72.3},
		},
	}

	component := &Component{
		Type: "line",
		Query: &Query{
			SeriesBy: "commodity_name",
			GroupBy:  []GroupBy{{Field: "month", Format: "month"}},
			Aggregations: []Aggregation{
				{Column: "total", Function: "sum", Alias: "Total"},
			},
			Calculate: CalculateList{{Formula: "total / 100", ResultAlias: "Rate"}},
		},
	}

	result, err := ct.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData := result.(ChartData)

	// Should use Rate (last column), not Total
	actsData := chartData.Datasets[0].Data
	if actsData[0] != 85.5 {
		t.Errorf("expected Rate value 85.5, got %v", actsData[0])
	}
}

func TestChartTransformerSeriesByEmptyRows(t *testing.T) {
	ct := &ChartTransformer{}

	tableData := TableData{
		Headers: []string{"label", "series", "Quantity"},
		Rows:    [][]interface{}{},
	}

	component := &Component{
		Type: "line",
		Query: &Query{
			SeriesBy: "commodity_name",
			GroupBy:  []GroupBy{{Field: "month", Format: "month"}},
			Aggregations: []Aggregation{
				{Column: "quantity", Function: "sum", Alias: "Quantity"},
			},
		},
	}

	result, err := ct.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData := result.(ChartData)
	if len(chartData.Labels) != 0 || len(chartData.Datasets) != 0 {
		t.Errorf("expected empty chart data, got %d labels and %d datasets", len(chartData.Labels), len(chartData.Datasets))
	}
}

func TestChartTransformerWithoutSeriesByUnchanged(t *testing.T) {
	ct := &ChartTransformer{}

	// Standard bar chart without seriesBy — should use existing column-based datasets
	tableData := TableData{
		Headers: []string{"label", "Cases", "Deaths"},
		Rows: [][]interface{}{
			{"Jan", 500, 10},
			{"Feb", 600, 12},
		},
	}

	component := &Component{
		Type: "bar",
		Query: &Query{
			GroupBy: []GroupBy{{Field: "month", Format: "month"}},
			Aggregations: []Aggregation{
				{Column: "cases", Function: "sum", Alias: "Cases"},
				{Column: "deaths", Function: "sum", Alias: "Deaths"},
			},
		},
	}

	result, err := ct.Transform(tableData, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	chartData := result.(ChartData)

	// Should have 2 datasets (one per aggregation column), not pivoted
	if len(chartData.Datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(chartData.Datasets))
	}
	if chartData.Datasets[0].Label != "Cases" {
		t.Errorf("expected dataset label 'Cases', got %q", chartData.Datasets[0].Label)
	}
	if chartData.Datasets[1].Label != "Deaths" {
		t.Errorf("expected dataset label 'Deaths', got %q", chartData.Datasets[1].Label)
	}
}
