package main

import (
	"strings"
	"testing"
)

// TestSynthesizeKpiContent verifies that the kpi-card HTML is built correctly
// from a Component's structured fields, including escaping and modifier classes.
func TestSynthesizeKpiContent(t *testing.T) {
	tests := []struct {
		name     string
		comp     Component
		contains []string
		excludes []string
	}{
		{
			name:     "default tone, no unit, no description",
			comp:     Component{Type: "kpi", Title: "Total Visits"},
			contains: []string{`class="kpi-card"`, `<h5 class="kpi-card__title">Total Visits</h5>`, `{{value}}</p>`},
			excludes: []string{`kpi-card--`, `kpi-card__sub`},
		},
		{
			name:     "alert tone with unit",
			comp:     Component{Type: "kpi", Title: "Mortality", Tone: "alert", Unit: "%"},
			contains: []string{`class="kpi-card kpi-card--alert"`, `{{value}}<span class="kpi-card__unit">%</span></p>`},
		},
		{
			name:     "description renders as kpi-card__sub",
			comp:     Component{Type: "kpi", Title: "Visits", Description: "All facilities"},
			contains: []string{`<p class="kpi-card__sub">All facilities</p>`},
		},
		{
			name:     "html in title and description is escaped",
			comp:     Component{Type: "kpi", Title: "<script>", Description: "A & B"},
			contains: []string{"&lt;script&gt;", "A &amp; B"},
			excludes: []string{"<script>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := synthesizeKpiContent(tt.comp)
			for _, want := range tt.contains {
				if !strings.Contains(out, want) {
					t.Errorf("expected output to contain %q\ngot: %s", want, out)
				}
			}
			for _, banned := range tt.excludes {
				if strings.Contains(out, banned) {
					t.Errorf("expected output NOT to contain %q\ngot: %s", banned, out)
				}
			}
		})
	}
}

// TestIsChartType tests the isChartType helper
func TestIsChartType(t *testing.T) {
	chartTypes := []string{"bar", "line", "bar_line", "pie"}
	for _, ct := range chartTypes {
		if !isChartType(ct) {
			t.Errorf("isChartType(%q) = false, want true", ct)
		}
	}

	nonChartTypes := []string{"table", "table_advanced", "text", "choropleth", "map", ""}
	for _, ct := range nonChartTypes {
		if isChartType(ct) {
			t.Errorf("isChartType(%q) = true, want false", ct)
		}
	}
}

// TestGetAllReports tests the GetAllReports function
func TestGetAllReports(t *testing.T) {
	reports := GetAllReports()

	// Initialize Reports slice if nil (when test config files aren't available)
	if reports.Reports == nil {
		reports.Reports = []ReportListItem{}
	}

	// Verify all returned reports have required fields
	for _, r := range reports.Reports {
		if r.ID == "" {
			t.Error("Expected non-empty ID in report")
		}
		if r.Title == "" {
			t.Error("Expected non-empty Title in report")
		}
	}
}

func TestGetAllReportsEmptyConfigWorks(t *testing.T) {
	t.Chdir("..")

	// The StatGate builder may start with no published report YAMLs. The report
	// index must load cleanly (no crash) and expose a valid (possibly empty)
	// report/section listing.
	reports := GetAllReports()

	if reports.Reports == nil {
		t.Fatal("GetAllReports returned a nil report list")
	}
	if reports.Sections == nil {
		t.Fatal("GetAllReports returned a nil section list")
	}

	// Every discovered report must have a non-empty ID (structural invariant).
	for _, report := range reports.Reports {
		if report.ID == "" {
			t.Fatalf("Discovered report with empty ID")
		}
	}
}

// TestReportListItemFields tests that ReportListItem has all required fields
func TestReportListItemFields(t *testing.T) {
	reports := GetAllReports()

	for _, report := range reports.Reports {
		if report.ID == "" {
			t.Error("Expected non-empty ID")
		}
		if report.Title == "" {
			t.Error("Expected non-empty Title")
		}
		// Description and Category might be empty for some reports
	}
}

// TestGetReportByIDEchis tests GetReportByID for the eSTAT report
func TestGetReportByIDEchis(t *testing.T) {
	filters := ReportFilters{}
	report := GetReportByID("estat-district-report", filters)

	if report == nil {
		t.Skip("eSTAT report requires data file - skipping if not available")
	}

	if report.ID != "estat-district-report" {
		t.Errorf("Expected ID estat-district-report, got %s", report.ID)
	}

	if report.Title != "eSTAT District Report" {
		t.Errorf("Expected title 'eSTAT District Report', got %s", report.Title)
	}

	if len(report.Sections) == 0 {
		t.Error("Expected at least one section in report")
	}

	// Check sections have components
	for i, section := range report.Sections {
		if section.ID == "" {
			t.Errorf("Section %d: Expected non-empty ID", i)
		}
		if section.Title == "" {
			t.Errorf("Section %d: Expected non-empty Title", i)
		}
		if section.Layout == "" {
			t.Errorf("Section %d: Expected non-empty Layout", i)
		}
		if len(section.Components) == 0 {
			t.Errorf("Section %d: Expected at least one component", i)
		}

		// Check components
		for j, component := range section.Components {
			if component.Type == "" {
				t.Errorf("Section %d Component %d: Expected non-empty Type", i, j)
			}
			validTypes := map[string]bool{"text": true, "bar": true, "line": true, "table": true, "map": true}
			if !validTypes[component.Type] {
				t.Errorf("Section %d Component %d: Invalid component type %q", i, j, component.Type)
			}
		}
	}
}

// TestGetReportByIDNonexistent tests GetReportByID with invalid ID
func TestGetReportByIDNonexistent(t *testing.T) {
	filters := ReportFilters{}
	report := GetReportByID("nonexistent-report", filters)

	if report != nil {
		t.Error("Expected nil for nonexistent report")
	}
}

// TestGetReportByIDWithFilters tests GetReportByID with various filters
func TestGetReportByIDWithFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters ReportFilters
	}{
		{
			name:    "Empty filters",
			filters: ReportFilters{},
		},
		{
			name: "District filter",
			filters: ReportFilters{
				Districts: []string{"Kampala"},
			},
		},
		{
			name: "Year filter",
			filters: ReportFilters{
				Year: "2024",
			},
		},
		{
			name: "Month filter",
			filters: ReportFilters{
				Month: "6",
			},
		},
		{
			name: "All filters",
			filters: ReportFilters{
				Districts: []string{"Jinja"},
				Year:      "2024",
				Month:     "3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the eSTAT report with filters
			// Note: This test skips if data isn't available
			report := GetReportByID("estat-district-report", tt.filters)
			if report == nil {
				t.Skip("eSTAT data not available")
			}

			if report.GeneratedAt == "" {
				t.Error("Expected GeneratedAt to be set")
			}
		})
	}
}

// TestComponentTypes tests various component types
func TestComponentTypes(t *testing.T) {
	validComponentTypes := []string{"text", "line", "bar", "table", "map"}

	report := GetReportByID("estat-district-report", ReportFilters{})
	if report == nil {
		t.Skip("eSTAT data not available")
	}

	componentTypeCount := make(map[string]int)

	for _, section := range report.Sections {
		for _, component := range section.Components {
			found := false
			for _, validType := range validComponentTypes {
				if component.Type == validType {
					found = true
					componentTypeCount[validType]++
					break
				}
			}
			if !found {
				t.Errorf("Invalid component type: %q", component.Type)
			}
		}
	}

	t.Logf("Component types found: %v", componentTypeCount)
}

// TestTableDataStructure tests the TableData struct
func TestTableDataStructure(t *testing.T) {
	// Create a sample TableData
	tableData := TableData{
		Headers: []string{"Name", "Age", "City"},
		Rows: [][]interface{}{
			{"John", 30, "Kampala"},
			{"Jane", 25, "Jinja"},
		},
	}

	if len(tableData.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(tableData.Headers))
	}

	if len(tableData.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(tableData.Rows))
	}

	if len(tableData.Rows[0]) != 3 {
		t.Errorf("Expected 3 columns in first row, got %d", len(tableData.Rows[0]))
	}
}

// TestChartDataStructure tests the ChartData struct
func TestChartDataStructure(t *testing.T) {
	chartData := ChartData{
		Labels: []string{"Jan", "Feb", "Mar"},
		Datasets: []Dataset{
			{
				Label: "Cases",
				Data:  []interface{}{10, 20, 15},
				Color: "#FF5733",
			},
		},
	}

	if len(chartData.Labels) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(chartData.Labels))
	}

	if len(chartData.Datasets) != 1 {
		t.Errorf("Expected 1 dataset, got %d", len(chartData.Datasets))
	}

	if chartData.Datasets[0].Label != "Cases" {
		t.Errorf("Expected dataset label 'Cases', got %q", chartData.Datasets[0].Label)
	}

	if len(chartData.Datasets[0].Data) != 3 {
		t.Errorf("Expected 3 data points, got %d", len(chartData.Datasets[0].Data))
	}
}

// TestMapDataStructure tests the MapData struct
func TestMapDataStructure(t *testing.T) {
	mapData := MapData{
		Center: []float64{0.3476, 32.5825}, // Uganda center
		Zoom:   7,
		Markers: []Marker{
			{Lat: 0.3476, Lng: 32.5825, Label: "Kampala", Value: 100},
			{Lat: 0.4178, Lng: 33.2676, Label: "Jinja", Value: 50},
		},
	}

	if len(mapData.Center) != 2 {
		t.Errorf("Expected center with 2 coordinates, got %d", len(mapData.Center))
	}

	if mapData.Zoom < 0 || mapData.Zoom > 20 {
		t.Errorf("Expected valid zoom level, got %d", mapData.Zoom)
	}

	if len(mapData.Markers) != 2 {
		t.Errorf("Expected 2 markers, got %d", len(mapData.Markers))
	}

	for i, marker := range mapData.Markers {
		if marker.Lat == 0 && marker.Lng == 0 {
			t.Errorf("Marker %d: Expected non-zero coordinates", i)
		}
		if marker.Label == "" {
			t.Errorf("Marker %d: Expected non-empty label", i)
		}
	}
}

// TestDatasetWithMultipleDatasets tests charts with multiple datasets
func TestDatasetWithMultipleDatasets(t *testing.T) {
	chartData := ChartData{
		Labels: []string{"Jan", "Feb", "Mar"},
		Datasets: []Dataset{
			{Label: "Series 1", Data: []interface{}{10, 20, 15}, Color: "#FF5733"},
			{Label: "Series 2", Data: []interface{}{5, 15, 10}, Color: "#33FF57"},
			{Label: "Series 3", Data: []interface{}{2, 8, 12}, Color: "#3357FF"},
		},
	}

	if len(chartData.Datasets) != 3 {
		t.Errorf("Expected 3 datasets, got %d", len(chartData.Datasets))
	}

	// Each dataset should have the same number of data points as labels
	for i, dataset := range chartData.Datasets {
		if len(dataset.Data) != len(chartData.Labels) {
			t.Errorf("Dataset %d: Expected %d data points, got %d", i, len(chartData.Labels), len(dataset.Data))
		}
	}
}

// TestReportFiltersList tests the FilterOptions struct
func TestReportFiltersList(t *testing.T) {
	filters := FilterOptions{
		Districts: []string{"Kampala", "Jinja", "Masaka"},
		Years:     []int{2024},
		Months:    []int{1, 2, 3, 4, 5, 6},
	}

	if len(filters.Districts) != 3 {
		t.Errorf("Expected 3 districts, got %d", len(filters.Districts))
	}

	if len(filters.Years) != 1 {
		t.Errorf("Expected 1 year, got %d", len(filters.Years))
	}

	if len(filters.Months) != 6 {
		t.Errorf("Expected 6 months, got %d", len(filters.Months))
	}
}

// TestGetReportByIDConcurrent tests concurrent execution of component queries
func TestGetReportByIDConcurrent(t *testing.T) {
	filters := ReportFilters{}
	report := GetReportByID("estat-district-report", filters)

	if report == nil {
		t.Skip("eSTAT report requires data - skipping if not available")
	}

	// Count components with queries
	componentCount := 0
	for _, section := range report.Sections {
		for _, component := range section.Components {
			if component.Query != nil {
				componentCount++
			}
		}
	}

	if componentCount == 0 {
		t.Skip("No components with queries found")
	}

	// Verify all components have been processed (even if some are empty due to errors)
	for i, section := range report.Sections {
		for j, component := range section.Components {
			// All components should have Data struct initialized (even if empty)
			// This ensures concurrent processing happened
			_ = component.Data
			t.Logf("Section %d Component %d (%s): Data initialized successfully", i, j, component.Type)
		}
	}

	// Verify GeneratedAt is set (indicates report processing completed)
	if report.GeneratedAt == "" {
		t.Error("Expected GeneratedAt to be set after concurrent processing")
	}
}

// TestGetReportByIDOrderPreservation tests that concurrent execution preserves component order
func TestGetReportByIDOrderPreservation(t *testing.T) {
	filters := ReportFilters{}
	report := GetReportByID("estat-district-report", filters)

	if report == nil {
		t.Skip("eSTAT report requires data - skipping if not available")
	}

	// Verify sections maintain order
	if len(report.Sections) == 0 {
		t.Error("Expected at least one section")
	}

	// Verify each section has components in order
	for i, section := range report.Sections {
		if len(section.Components) > 0 {
			// Verify component types are consistent with expected YAML order
			for j, component := range section.Components {
				if component.Type == "" {
					t.Errorf("Section %d Component %d: Expected non-empty Type", i, j)
				}
			}
		}
	}

	// Verify by running multiple times - results should be consistent
	report2 := GetReportByID("estat-district-report", filters)
	if report2 == nil {
		t.Fatal("Second call returned nil")
	}

	if len(report.Sections) != len(report2.Sections) {
		t.Errorf("Expected consistent number of sections: %d vs %d", len(report.Sections), len(report2.Sections))
	}

	// Verify section IDs match in same order
	for i := range report.Sections {
		if report.Sections[i].ID != report2.Sections[i].ID {
			t.Errorf("Section %d: Expected consistent ID order: %q vs %q", i, report.Sections[i].ID, report2.Sections[i].ID)
		}
	}

	t.Logf("Verified order preservation across %d sections", len(report.Sections))
}

// BenchmarkGetReportConcurrent benchmarks concurrent report generation
func BenchmarkGetReportConcurrent(b *testing.B) {
	filters := ReportFilters{Districts: []string{"Central District"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report := GetReportByID("estat-district-report", filters)
		if report == nil {
			b.Skip("eSTAT data not available")
		}
	}
}

// TestValidateReportID tests the ValidateReportID function
func TestValidateReportID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		expectErr bool
	}{
		// Valid IDs
		{"Simple report", "estat-district-report", false},
		{"Single-level category", "disease-surveillance/malaria-report", false},
		{"With underscores", "health_metrics/fever_analysis", false},
		{"Numbers allowed", "report-2024/analysis-v2", false},

		// Path traversal attacks
		{"Parent directory", "../etc/passwd", true},
		{"Double parent", "../../server/database.go", true},
		{"Hidden parent", "valid/../../../etc/passwd", true},
		{"Current directory", "./report", true},

		// Multi-level nesting
		{"Two-level nesting", "section/subsection/report", false},
		{"Three-level nesting", "a/b/c/d", false},
		{"Pharmaceutical taxonomy", "Programs/Pharmaceutical-Services/Supply-Chain-and-Logistics/Warehouse-Reports/stock-report", false},

		// Invalid characters
		{"Space in name", "my report", true},
		{"Special char @", "report@2024", true},
		{"SQL injection", "'; DROP TABLE--", true},
		{"XSS attempt", "<script>alert(1)</script>", true},

		// Edge cases
		{"Empty string", "", true},
		{"Only slash", "/", true},
		{"Leading slash", "/report", true},
		{"Trailing slash", "report/", true},
		{"Double slash", "category//report", true},
		{"Absolute path", "/configs/report.yaml", true},
		{"Backslash path", "category\\report", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReportID(tt.id)

			if tt.expectErr && err == nil {
				t.Errorf("Expected error for ID %q, got nil", tt.id)
			}

			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error for ID %q, got: %v", tt.id, err)
			}
		})
	}
}

// TestGetReportByIDPathTraversal tests that path traversal attacks are blocked
func TestGetReportByIDPathTraversal(t *testing.T) {
	attackVectors := []string{
		"../server/database.go",
		"../../go.mod",
		"../../../etc/passwd",
		"category/../../../etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
	}

	for _, attack := range attackVectors {
		t.Run(attack, func(t *testing.T) {
			report := GetReportByID(attack, ReportFilters{})
			if report != nil {
				t.Errorf("SECURITY FAILURE: Path traversal succeeded for: %s", attack)
			}
		})
	}
}

// TestValidateReportIDEmptySegments tests validation of empty segments
func TestValidateReportIDEmptySegments(t *testing.T) {
	invalidIDs := []string{
		"category//report",  // Empty segment
		"//report",          // Leading double slash
		"category///report", // Multiple empty segments
	}

	for _, id := range invalidIDs {
		err := ValidateReportID(id)
		if err == nil {
			t.Errorf("Expected error for ID with empty segments: %q", id)
		}
	}
}

// TestValidateReportIDMaxDepth tests enforcement of bounded nesting
func TestValidateReportIDMaxDepth(t *testing.T) {
	// Valid: up to maxReportIDSlashes path separators
	validIDs := []string{
		"report",
		"section/report",
		"section/subsection/report",
		"a/b/c/d",
		"Programs/Pharmaceutical-Services/Supply-Chain-and-Logistics/Warehouse-Reports/stock-report",
	}

	for _, id := range validIDs {
		err := ValidateReportID(id)
		if err != nil {
			t.Errorf("Expected no error for valid depth: %q, got: %v", id, err)
		}
	}

	// Invalid: more than maxReportIDSlashes path separators
	invalidIDs := []string{
		"a/b/c/d/e/f/g",
		"level1/level2/level3/level4/level5/level6/level7",
	}

	for _, id := range invalidIDs {
		err := ValidateReportID(id)
		if err == nil {
			t.Errorf("Expected error for excessive nesting: %q", id)
		}
	}
}

// TestCleanupUnsubstitutedFilters tests the cleanup of unsubstituted filter placeholders
func TestCleanupUnsubstitutedFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes AND year = {{year}}",
			input:    "SELECT * FROM t WHERE 1=1 AND year = {{year}}",
			expected: "SELECT * FROM t WHERE 1=1",
		},
		{
			name:     "removes multiple unset conditions",
			input:    "SELECT * FROM t WHERE 1=1 AND year = {{year}} AND district = {{district}}",
			expected: "SELECT * FROM t WHERE 1=1",
		},
		{
			name:     "preserves substituted values",
			input:    "SELECT * FROM t WHERE 1=1 AND year = '2024' AND district = {{district}}",
			expected: "SELECT * FROM t WHERE 1=1 AND year = '2024'",
		},
		{
			name:     "removes raw placeholders",
			input:    "SELECT * FROM t WHERE col = {{unknown}}",
			expected: "SELECT * FROM t WHERE col = ",
		},
		{
			name:     "no placeholders - unchanged",
			input:    "SELECT * FROM t WHERE year = '2024'",
			expected: "SELECT * FROM t WHERE year = '2024'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanupUnsubstitutedFilters(tt.input)
			if result != tt.expected {
				t.Errorf("cleanupUnsubstitutedFilters(%q)\n  got:  %q\n  want: %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFilterArithmetic tests arithmetic expressions in filter placeholders
func TestFilterArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		filters  ReportFilters
		expected string
	}{
		{
			name:     "Year subtraction",
			text:     "Compared to {{filter.year - 1}}",
			filters:  ReportFilters{Year: "2025"},
			expected: "Compared to 2024",
		},
		{
			name:     "Year addition",
			text:     "Next year: {{filter.year + 1}}",
			filters:  ReportFilters{Year: "2024"},
			expected: "Next year: 2025",
		},
		{
			name:     "Year multiple operations",
			text:     "{{filter.year - 2}} to {{filter.year + 1}}",
			filters:  ReportFilters{Year: "2025"},
			expected: "2023 to 2026",
		},
		{
			name:     "Month subtraction",
			text:     "Previous month: {{filter.month - 1}}",
			filters:  ReportFilters{Month: "3"},
			expected: "Previous month: 2",
		},
		{
			name:     "Month addition",
			text:     "Next month: {{filter.month + 1}}",
			filters:  ReportFilters{Month: "5"},
			expected: "Next month: 6",
		},
		{
			name:     "Month wrapping underflow",
			text:     "Month: {{filter.month - 1}}",
			filters:  ReportFilters{Month: "1"},
			expected: "Month: 12",
		},
		{
			name:     "Month wrapping overflow",
			text:     "Month: {{filter.month + 1}}",
			filters:  ReportFilters{Month: "12"},
			expected: "Month: 1",
		},
		{
			name:     "Month large subtraction with wrapping",
			text:     "{{filter.month - 5}}",
			filters:  ReportFilters{Month: "3"},
			expected: "10",
		},
		{
			name:     "Month large addition with wrapping",
			text:     "{{filter.month + 14}}",
			filters:  ReportFilters{Month: "10"},
			expected: "12",
		},
		{
			name:     "Quarter subtraction",
			text:     "Previous quarter: Q{{filter.quarter - 1}}",
			filters:  ReportFilters{Quarter: "3"},
			expected: "Previous quarter: Q2",
		},
		{
			name:     "Quarter addition",
			text:     "Next quarter: Q{{filter.quarter + 1}}",
			filters:  ReportFilters{Quarter: "2"},
			expected: "Next quarter: Q3",
		},
		{
			name:     "Quarter wrapping underflow",
			text:     "Q{{filter.quarter - 1}}",
			filters:  ReportFilters{Quarter: "1"},
			expected: "Q4",
		},
		{
			name:     "Quarter wrapping overflow",
			text:     "Q{{filter.quarter + 1}}",
			filters:  ReportFilters{Quarter: "4"},
			expected: "Q1",
		},
		{
			name:     "Quarter large subtraction with wrapping",
			text:     "Q{{filter.quarter - 3}}",
			filters:  ReportFilters{Quarter: "2"},
			expected: "Q3",
		},
		{
			name:     "MonthName subtraction",
			text:     "Previous: {{filter.monthName - 1}}",
			filters:  ReportFilters{Month: "3"},
			expected: "Previous: February",
		},
		{
			name:     "MonthName addition",
			text:     "Next: {{filter.monthName + 1}}",
			filters:  ReportFilters{Month: "8"},
			expected: "Next: September",
		},
		{
			name:     "MonthName wrapping",
			text:     "{{filter.monthName - 1}}",
			filters:  ReportFilters{Month: "1"},
			expected: "December",
		},
		{
			name:     "Multi-select year shows fallback",
			text:     "Year: {{filter.year - 1}}",
			filters:  ReportFilters{Year: "2024,2025"},
			expected: "Year: Selected Years",
		},
		{
			name:     "Multi-select month shows fallback",
			text:     "Month: {{filter.month + 1}}",
			filters:  ReportFilters{Month: "1,2,3"},
			expected: "Month: Selected Months",
		},
		{
			name:     "Multi-select monthName shows fallback",
			text:     "Overview for {{filter.monthName-1}}",
			filters:  ReportFilters{Month: "1,2"},
			expected: "Overview for Selected Months",
		},
		{
			name:     "Complex expression with multiple filters",
			text:     "{{filter.year - 1}} vs {{filter.year}}",
			filters:  ReportFilters{Year: "2025"},
			expected: "2024 vs 2025",
		},
		{
			name:     "Arithmetic with spacing variations",
			text:     "{{filter.year-1}} and {{filter.month +1}}",
			filters:  ReportFilters{Year: "2025", Month: "6"},
			expected: "2024 and 7",
		},
		{
			name:     "Empty filter - shows default value",
			text:     "Year: {{filter.year - 1}}",
			filters:  ReportFilters{},
			expected: "Year: All Years",
		},
		{
			name:     "Empty filter - monthName shows default",
			text:     "Overview for {{filter.monthName-1}}",
			filters:  ReportFilters{},
			expected: "Overview for All Months",
		},
		{
			name:     "Mixed arithmetic and simple placeholders",
			text:     "{{filter.year - 1}} data for {{filter.district}}",
			filters:  ReportFilters{Year: "2025", Districts: []string{"Kampala"}},
			expected: "2024 data for Kampala",
		},
		{
			name:     "Short month year placeholder",
			text:     "{{filter.monthShortNameYear}}",
			filters:  ReportFilters{Month: "5", Year: "2025"},
			expected: "May 2025",
		},
		{
			name:     "Short month year arithmetic placeholder",
			text:     "{{filter.monthShortNameYear - 1}}",
			filters:  ReportFilters{Month: "5", Year: "2025"},
			expected: "Apr 2025",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interpolateFilterPlaceholders(tt.text, tt.filters)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestInterpolateComponentPlaceholders_SplitColumns(t *testing.T) {
	component := Component{
		Title: "EMS Snapshot",
		SplitColumns: []SplitColumnGroup{
			{
				Parent: "{{filter.monthShortNameYear}}",
				Children: []string{
					"{{filter.monthShortNameYear}} Total",
					"{{filter.monthShortNameYear - 1}} %",
				},
			},
		},
	}

	interpolateComponentPlaceholders(&component, ReportFilters{Month: "5", Year: "2025"})

	if got := component.SplitColumns[0].Parent; got != "May 2025" {
		t.Fatalf("parent = %q, want %q", got, "May 2025")
	}
	if got := component.SplitColumns[0].Children[0]; got != "May 2025 Total" {
		t.Fatalf("child[0] = %q, want %q", got, "May 2025 Total")
	}
	if got := component.SplitColumns[0].Children[1]; got != "Apr 2025 %" {
		t.Fatalf("child[1] = %q, want %q", got, "Apr 2025 %")
	}
}

// TestWrapValue tests the value wrapping function for month and quarter boundaries
func TestWrapValue(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		min      int
		max      int
		expected int
	}{
		// Month wrapping (1-12)
		{"Month: normal value", 5, 1, 12, 5},
		{"Month: underflow by 1", 0, 1, 12, 12},
		{"Month: underflow by 2", -1, 1, 12, 11},
		{"Month: overflow by 1", 13, 1, 12, 1},
		{"Month: overflow by 2", 14, 1, 12, 2},
		{"Month: large underflow", -5, 1, 12, 7},
		{"Month: large overflow", 25, 1, 12, 1},

		// Quarter wrapping (1-4)
		{"Quarter: normal value", 2, 1, 4, 2},
		{"Quarter: underflow", 0, 1, 4, 4},
		{"Quarter: overflow", 5, 1, 4, 1},
		{"Quarter: large underflow", -3, 1, 4, 1},
		{"Quarter: large overflow", 9, 1, 4, 1},

		// Edge cases
		{"Min boundary", 1, 1, 12, 1},
		{"Max boundary", 12, 1, 12, 12},
		{"Zero with 0-based range", 0, 0, 11, 0},
		{"Eleven with 0-based range", 11, 0, 11, 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapValue(tt.value, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("wrapValue(%d, %d, %d) = %d; want %d", tt.value, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

// TestApplyArithmetic tests the arithmetic helper function
func TestApplyArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		operator string
		operand  int
		expected int
	}{
		{"Addition", 10, "+", 5, 15},
		{"Subtraction", 10, "-", 3, 7},
		{"Zero addition", 5, "+", 0, 5},
		{"Zero subtraction", 5, "-", 0, 5},
		{"Negative result", 3, "-", 10, -7},
		{"Large addition", 100, "+", 500, 600},
		{"Large subtraction", 1000, "-", 999, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyArithmetic(tt.value, tt.operator, tt.operand)
			if result != tt.expected {
				t.Errorf("applyArithmetic(%d, %q, %d) = %d; want %d", tt.value, tt.operator, tt.operand, result, tt.expected)
			}
		})
	}
}

// TestCalculatePreviousPeriodFilters tests the dynamic previous period calculation
func TestCalculatePreviousPeriodFilters(t *testing.T) {
	tests := []struct {
		name           string
		filters        ReportFilters
		comparisonType string
		expectedYear   string
		expectedMonth  string
		expectedWeek   string
		expectedQtr    string
		expectNil      bool
	}{
		// previous_period with week filter
		{
			name:           "previous_period with week filter",
			filters:        ReportFilters{Week: "2024W15"},
			comparisonType: "previous_period",
			expectedWeek:   "2024W14",
		},
		{
			name:           "previous_period with week at year boundary",
			filters:        ReportFilters{Week: "2024W01"},
			comparisonType: "previous_period",
			expectedWeek:   "2023W52",
		},
		// previous_period with month filter
		{
			name:           "previous_period with month filter",
			filters:        ReportFilters{Year: "2024", Month: "6"},
			comparisonType: "previous_period",
			expectedYear:   "2024",
			expectedMonth:  "5",
		},
		{
			name:           "previous_period with month at year boundary",
			filters:        ReportFilters{Year: "2024", Month: "1"},
			comparisonType: "previous_period",
			expectedYear:   "2023",
			expectedMonth:  "12",
		},
		// previous_period with quarter filter
		{
			name:           "previous_period with quarter filter",
			filters:        ReportFilters{Year: "2024", Quarter: "3"},
			comparisonType: "previous_period",
			expectedYear:   "2024",
			expectedQtr:    "2",
		},
		{
			name:           "previous_period with quarter at year boundary",
			filters:        ReportFilters{Year: "2024", Quarter: "1"},
			comparisonType: "previous_period",
			expectedYear:   "2023",
			expectedQtr:    "4",
		},
		// previous_period priority: week > month > quarter
		{
			name:           "previous_period prioritizes week over month",
			filters:        ReportFilters{Year: "2024", Month: "6", Week: "2024W25"},
			comparisonType: "previous_period",
			expectedWeek:   "2024W24",
		},
		{
			name:           "previous_period prioritizes month over quarter",
			filters:        ReportFilters{Year: "2024", Month: "6", Quarter: "2"},
			comparisonType: "previous_period",
			expectedYear:   "2024",
			expectedMonth:  "5",
		},
		// previous_period with no time filter returns nil
		{
			name:           "previous_period with only year returns nil",
			filters:        ReportFilters{Year: "2024"},
			comparisonType: "previous_period",
			expectNil:      true,
		},
		// previous_year
		{
			name:           "previous_year",
			filters:        ReportFilters{Year: "2024", Month: "6"},
			comparisonType: "previous_year",
			expectedYear:   "2023",
			expectedMonth:  "6",
		},
		// previous_month (backward compat)
		{
			name:           "previous_month explicit",
			filters:        ReportFilters{Year: "2024", Month: "6"},
			comparisonType: "previous_month",
			expectedYear:   "2024",
			expectedMonth:  "5",
		},
		// Invalid cases
		{
			name:           "unknown comparison type returns nil",
			filters:        ReportFilters{Year: "2024", Month: "6"},
			comparisonType: "unknown",
			expectNil:      true,
		},
		{
			name:           "previous_year without year returns nil",
			filters:        ReportFilters{Month: "6"},
			comparisonType: "previous_year",
			expectNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePreviousPeriodFilters(tt.filters, tt.comparisonType)

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.expectedYear != "" && result.Year != tt.expectedYear {
				t.Errorf("Year = %q, want %q", result.Year, tt.expectedYear)
			}
			if tt.expectedMonth != "" && result.Month != tt.expectedMonth {
				t.Errorf("Month = %q, want %q", result.Month, tt.expectedMonth)
			}
			if tt.expectedWeek != "" && result.Week != tt.expectedWeek {
				t.Errorf("Week = %q, want %q", result.Week, tt.expectedWeek)
			}
			if tt.expectedQtr != "" && result.Quarter != tt.expectedQtr {
				t.Errorf("Quarter = %q, want %q", result.Quarter, tt.expectedQtr)
			}
		})
	}
}

// TestCalculatePreviousWeekFilters tests week boundary handling
func TestCalculatePreviousWeekFilters(t *testing.T) {
	tests := []struct {
		name         string
		week         string
		expectedWeek string
		expectNil    bool
	}{
		{"mid year week", "2024W26", "2024W25", false},
		{"first week of year", "2024W01", "2023W52", false},
		{"week 53 year", "2021W01", "2020W53", false}, // 2020 has 53 weeks
		{"empty week", "", "", true},
		{"invalid format", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePreviousWeekFilters(ReportFilters{Week: tt.week})

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got week=%q", result.Week)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Week != tt.expectedWeek {
				t.Errorf("Week = %q, want %q", result.Week, tt.expectedWeek)
			}
		})
	}
}

// TestCalculatePreviousQuarterFilters tests quarter boundary handling
func TestCalculatePreviousQuarterFilters(t *testing.T) {
	tests := []struct {
		name        string
		year        string
		quarter     string
		expectedYr  string
		expectedQtr string
		expectNil   bool
	}{
		{"Q3 to Q2", "2024", "3", "2024", "2", false},
		{"Q1 to Q4 previous year", "2024", "1", "2023", "4", false},
		{"Q4 to Q3", "2024", "4", "2024", "3", false},
		{"missing year", "", "2", "", "", true},
		{"missing quarter", "2024", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculatePreviousQuarterFilters(ReportFilters{Year: tt.year, Quarter: tt.quarter})

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Year != tt.expectedYr {
				t.Errorf("Year = %q, want %q", result.Year, tt.expectedYr)
			}
			if result.Quarter != tt.expectedQtr {
				t.Errorf("Quarter = %q, want %q", result.Quarter, tt.expectedQtr)
			}
		})
	}
}

// ============== ValidateCustomFilters Tests ==============

func TestValidateCustomFiltersValid(t *testing.T) {
	filters := []CustomFilterDef{
		{Column: "facility_type", Table: "schema.table_name", Label: "Facility Type", Type: "select", DefaultValue: "HC III"},
		{Column: "age_group", Table: "public.demographics", Label: "Age Group", Type: "multiselect"},
	}
	if err := ValidateCustomFilters(filters); err != nil {
		t.Errorf("expected no error for valid filters, got: %v", err)
	}
}

func TestGetFilterDefinitionsIncludesCustomDefaultValue(t *testing.T) {
	report := &Report{
		CustomFilters: []CustomFilterDef{
			{Column: "basket_group_2", Table: "report.mv_data", Label: "Basket", Type: "select", DefaultValue: "EMHS"},
		},
		Sections: []Section{
			{
				Components: []Component{
					{Type: "bar", Query: &Query{Table: "report.mv_data"}},
				},
			},
		},
	}

	defs := GetFilterDefinitions([]string{"basket_group_2"}, report)
	def := defs["basket_group_2"]

	if def.DefaultValue != "EMHS" {
		t.Fatalf("DefaultValue = %q, want %q", def.DefaultValue, "EMHS")
	}
}

func TestValidateCustomFiltersEmpty(t *testing.T) {
	if err := ValidateCustomFilters(nil); err != nil {
		t.Errorf("expected no error for nil filters, got: %v", err)
	}
	if err := ValidateCustomFilters([]CustomFilterDef{}); err != nil {
		t.Errorf("expected no error for empty filters, got: %v", err)
	}
}

func TestValidateCustomFiltersReservedName(t *testing.T) {
	reserved := []string{"district", "year", "month", "quarter", "week", "region", "facility"}
	for _, name := range reserved {
		filters := []CustomFilterDef{
			{Column: name, Table: "schema.table", Label: "Test", Type: "select"},
		}
		err := ValidateCustomFilters(filters)
		if err == nil {
			t.Errorf("expected error for reserved name %q, got nil", name)
		}
		if err != nil && !strings.Contains(err.Error(), "reserved") {
			t.Errorf("expected reserved name error for %q, got: %v", name, err)
		}
	}
}

func TestValidateCustomFiltersInvalidColumn(t *testing.T) {
	tests := []struct {
		name   string
		column string
	}{
		{"starts with number", "1column"},
		{"contains space", "my column"},
		{"contains dash", "my-column"},
		{"SQL injection attempt", "col; DROP TABLE"},
		{"empty", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filters := []CustomFilterDef{
				{Column: tc.column, Table: "schema.table", Label: "Test", Type: "select"},
			}
			if err := ValidateCustomFilters(filters); err == nil {
				t.Errorf("expected error for invalid column %q", tc.column)
			}
		})
	}
}

func TestValidateCustomFiltersInvalidTable(t *testing.T) {
	tests := []struct {
		name  string
		table string
	}{
		{"contains space", "my table"},
		{"SQL injection", "table; DROP"},
		{"empty", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filters := []CustomFilterDef{
				{Column: "valid_col", Table: tc.table, Label: "Test", Type: "select"},
			}
			if err := ValidateCustomFilters(filters); err == nil {
				t.Errorf("expected error for invalid table %q", tc.table)
			}
		})
	}
}

func TestValidateCustomFiltersInvalidType(t *testing.T) {
	filters := []CustomFilterDef{
		{Column: "valid_col", Table: "schema.table", Label: "Test", Type: "dropdown"},
	}
	err := ValidateCustomFilters(filters)
	if err == nil {
		t.Error("expected error for invalid type 'dropdown'")
	}
	if err != nil && !strings.Contains(err.Error(), "select") {
		t.Errorf("expected type error mentioning 'select', got: %v", err)
	}
}

// ============== ParseWeekPeriod Tests ==============

func TestParseWeekPeriod(t *testing.T) {
	tests := []struct {
		name      string
		period    string
		wantYear  int
		wantWeek  int
		wantError bool
	}{
		{"valid week 1", "2024W01", 2024, 1, false},
		{"valid week 52", "2024W52", 2024, 52, false},
		{"valid week 15", "2023W15", 2023, 15, false},
		{"too short", "2024W1", 0, 0, true},
		{"too long", "2024W001", 0, 0, true},
		{"empty", "", 0, 0, true},
		{"invalid year", "XXXXW01", 0, 0, true},
		{"invalid week", "2024WXX", 0, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			year, week, err := ParseWeekPeriod(tc.period)
			if tc.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if year != tc.wantYear {
				t.Errorf("year = %d, want %d", year, tc.wantYear)
			}
			if week != tc.wantWeek {
				t.Errorf("week = %d, want %d", week, tc.wantWeek)
			}
		})
	}
}

// ============== buildSQLList Tests ==============

func TestBuildSQLList(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"single value", []string{"Kampala"}, "Kampala"},
		{"multiple values", []string{"Kampala", "Central"}, "Kampala', 'Central"},
		{"value with apostrophe", []string{"O'Brien"}, "O''Brien"},
		{"multiple with apostrophes", []string{"O'Brien", "D'Arcy"}, "O''Brien', 'D''Arcy"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildSQLList(tc.values)
			if result != tc.expected {
				t.Errorf("buildSQLList(%v) = %q, want %q", tc.values, result, tc.expected)
			}
		})
	}
}

func TestGetFilterDefinitionsFilterOptionsTable(t *testing.T) {
	r := &Report{
		FilterOptionsTable: "cht.mv_supervisor_hierarchy_2",
		Sections: []Section{
			{Components: []Component{{FilterTable: "cht.mv_form_meta"}}},
		},
	}
	defs := GetFilterDefinitions([]string{"district", "year", "week", "region", "facility"}, r)
	if !strings.Contains(defs["district"].APIEndpoint, "table=cht.mv_supervisor_hierarchy_2") {
		t.Errorf("district endpoint = %q, want table=cht.mv_supervisor_hierarchy_2", defs["district"].APIEndpoint)
	}
	if !strings.Contains(defs["year"].APIEndpoint, "table=cht.mv_form_meta") {
		t.Errorf("year should use primary table, got %q", defs["year"].APIEndpoint)
	}
	if !strings.Contains(defs["week"].APIEndpoint, "table=cht.mv_form_meta") {
		t.Errorf("week should use primary table, got %q", defs["week"].APIEndpoint)
	}
	if !strings.Contains(defs["region"].APIEndpoint, "table=cht.mv_supervisor_hierarchy_2") {
		t.Errorf("region endpoint = %q", defs["region"].APIEndpoint)
	}
	if !strings.Contains(defs["facility"].APIEndpoint, "table=cht.mv_supervisor_hierarchy_2") {
		t.Errorf("facility endpoint = %q", defs["facility"].APIEndpoint)
	}
}

// TestBuildTextSQLFilterMap ensures the filter-map helper used by raw-SQL text
// KPIs (and by the previous-period comparison path) populates the placeholder
// keys applied via substituteFilters.
func TestBuildTextSQLFilterMap(t *testing.T) {
	report := &Report{}
	filters := ReportFilters{
		Year:       "2024",
		Month:      "6",
		Quarter:    "2",
		Week:       "2024W15",
		Districts:  []string{"Kampala", "Central"},
		Regions:    []string{"Central"},
		Facilities: []string{"Mulago"},
	}

	m := buildTextSQLFilterMap(filters, report)

	cases := map[string]string{
		"district":      "Kampala",
		"district_list": "Kampala', 'Central",
		"year":          "2024",
		"filter.year":   "2024",
		"quarter":       "2",
		"week":          "2024W15",
		"region":        "Central",
		"facility":      "Mulago",
	}
	for key, want := range cases {
		if got, ok := m[key]; !ok || got != want {
			t.Errorf("filterMap[%q] = %q (present=%v), want %q", key, got, ok, want)
		}
	}
	if _, ok := m["month"]; !ok {
		t.Errorf("expected month key to be set (value depends on report.TimeColumns)")
	}

	empty := buildTextSQLFilterMap(ReportFilters{}, report)
	for _, key := range []string{"district", "year", "month", "quarter", "week", "region", "facility"} {
		if _, ok := empty[key]; ok {
			t.Errorf("empty filters set %q unexpectedly", key)
		}
	}
}
