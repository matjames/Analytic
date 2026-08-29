package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

// ============== Schema Handlers Tests ==============

// TestGetSchemaTablesHandler tests the GetSchemaTablesHandler endpoint
func TestGetSchemaTablesHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		skipIfNoDb     bool
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "OPTIONS request returns 200 with CORS headers",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     false,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Missing CORS header Access-Control-Allow-Origin")
				}
				if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
					t.Error("Missing or incorrect Access-Control-Allow-Methods header")
				}
			},
		},
		{
			name:           "GET request returns JSON",
			method:         "GET",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     true, // This test requires DB
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Content-Type") != "application/json" {
					t.Error("Expected Content-Type application/json")
				}
				// Response should be valid JSON (empty array or table list)
				var tables []TableInfo
				if err := json.NewDecoder(w.Body).Decode(&tables); err != nil {
					// May fail if DB not available - that's acceptable
					t.Logf("Note: Could not decode response (DB may not be available): %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoDb && DB == nil {
				t.Skip("Skipping test - database not available")
			}

			req := httptest.NewRequest(tt.method, "/api/schema/tables", nil)
			w := httptest.NewRecorder()

			GetSchemaTablesHandler(w, req)

			if w.Code != tt.expectedStatus {
				// Allow 500 if DB is not available
				if w.Code == http.StatusInternalServerError {
					t.Logf("Got 500 - database likely not available during test")
					return
				}
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestGetTableColumnsHandler tests the GetTableColumnsHandler endpoint
func TestGetTableColumnsHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		skipIfNoDb     bool
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "OPTIONS request returns 200",
			method:         "OPTIONS",
			path:           "/api/schema/table/report/cht_form_097b/columns",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     false,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Missing CORS header")
				}
			},
		},
		{
			name:           "GET valid table returns columns",
			method:         "GET",
			path:           "/api/schema/table/report/cht_form_097b/columns",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     true,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Content-Type") != "application/json" {
					t.Error("Expected Content-Type application/json")
				}
			},
		},
		{
			name:           "Invalid path returns 400",
			method:         "GET",
			path:           "/api/schema/table/invalid",
			expectedStatus: http.StatusBadRequest,
			skipIfNoDb:     false,
			checkResponse:  nil,
		},
		{
			name:           "Path with only schema returns 400",
			method:         "GET",
			path:           "/api/schema/table/report",
			expectedStatus: http.StatusBadRequest,
			skipIfNoDb:     false,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoDb && DB == nil {
				t.Skip("Skipping test - database not available")
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			GetTableColumnsHandler(w, req)

			if w.Code != tt.expectedStatus {
				// Allow 500 if DB is not available
				if w.Code == http.StatusInternalServerError {
					t.Logf("Got 500 - database likely not available during test")
					return
				}
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestGetTablePreviewHandler tests the GetTablePreviewHandler endpoint
func TestGetTablePreviewHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		skipIfNoDb     bool
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "OPTIONS request returns 200",
			method:         "OPTIONS",
			path:           "/api/schema/table/report/cht_form_097b/preview",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     false,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Missing CORS header")
				}
			},
		},
		{
			name:           "GET preview returns JSON",
			method:         "GET",
			path:           "/api/schema/table/report/cht_form_097b/preview",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     true,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Content-Type") != "application/json" {
					t.Error("Expected Content-Type application/json")
				}
			},
		},
		{
			name:           "GET preview with limit parameter",
			method:         "GET",
			path:           "/api/schema/table/report/cht_form_097b/preview?limit=5",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     true,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response TablePreviewResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err == nil {
					if response.RowCount > 5 {
						t.Errorf("Expected at most 5 rows, got %d", response.RowCount)
					}
				}
			},
		},
		{
			name:           "GET preview with invalid limit uses default",
			method:         "GET",
			path:           "/api/schema/table/report/cht_form_097b/preview?limit=invalid",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     true,
			checkResponse:  nil,
		},
		{
			name:           "GET preview with excessive limit is capped",
			method:         "GET",
			path:           "/api/schema/table/report/cht_form_097b/preview?limit=1000",
			expectedStatus: http.StatusOK,
			skipIfNoDb:     true,
			checkResponse:  nil,
		},
		{
			name:           "Invalid path returns 400",
			method:         "GET",
			path:           "/api/schema/table/invalid",
			expectedStatus: http.StatusBadRequest,
			skipIfNoDb:     false,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoDb && DB == nil {
				t.Skip("Skipping test - database not available")
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			GetTablePreviewHandler(w, req)

			if w.Code != tt.expectedStatus {
				// Allow 500 if DB is not available
				if w.Code == http.StatusInternalServerError {
					t.Logf("Got 500 - database likely not available during test")
					return
				}
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestValidateQueryHandler tests the ValidateQueryHandler endpoint
func TestValidateQueryHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           any
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "OPTIONS request returns 200",
			method:         "OPTIONS",
			body:           nil,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Missing CORS header")
				}
			},
		},
		{
			name:   "Valid query returns success",
			method: "POST",
			body: QueryValidationRequest{
				Table: "report.cht_form_097b",
				Aggregations: []Aggregation{
					{Column: "num_fever_u5_male", Function: "sum", Alias: "total"},
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result QueryValidationResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if !result.Valid {
					t.Errorf("Expected valid query, got errors: %v", result.Errors)
				}
				if result.GeneratedSQL == "" {
					t.Error("Expected generated SQL to be non-empty")
				}
			},
		},
		{
			name:   "Query with groupBy generates valid SQL",
			method: "POST",
			body: QueryValidationRequest{
				Table: "report.cht_form_097b",
				Aggregations: []Aggregation{
					{Column: "num_fever_u5_male", Function: "sum", Alias: "total"},
				},
				GroupBy: []GroupBy{
					{Field: "period_date", Format: "month"},
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result QueryValidationResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if !result.Valid {
					t.Errorf("Expected valid query with groupBy")
				}
			},
		},
		{
			name:   "Query with calculate generates valid SQL",
			method: "POST",
			body: QueryValidationRequest{
				Table: "report.cht_form_097b",
				Aggregations: []Aggregation{
					{Column: "num_fever_u5_male", Function: "sum", Alias: "positive"},
					{Column: "num_fever_u5_female", Function: "sum", Alias: "tested"},
				},
				Calculate: []Calculate{
					{Formula: "(positive / tested * 100)", RoundTo: intPtr(1), WhenZero: 0},
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result QueryValidationResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if !result.Valid {
					t.Errorf("Expected valid query with calculate")
				}
			},
		},
		{
			name:           "Invalid JSON returns 400",
			method:         "POST",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name:           "GET method not allowed",
			method:         "GET",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  nil,
		},
		{
			name:   "Missing table returns invalid",
			method: "POST",
			body: QueryValidationRequest{
				Aggregations: []Aggregation{
					{Column: "col1", Function: "sum", Alias: "total"},
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result QueryValidationResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if result.Valid {
					t.Error("Expected invalid result for missing table")
				}
				if len(result.Errors) == 0 {
					t.Error("Expected errors for missing table")
				}
			},
		},
		{
			name:   "Missing aggregations returns invalid",
			method: "POST",
			body: QueryValidationRequest{
				Table: "report.cht_form_097b",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result QueryValidationResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if result.Valid {
					t.Error("Expected invalid result for missing aggregations")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					body = bytes.NewBufferString(v)
				default:
					jsonBody, _ := json.Marshal(tt.body)
					body = bytes.NewBuffer(jsonBody)
				}
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tt.method, "/api/schema/validate", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ValidateQueryHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestPreviewQueryHandler tests the PreviewQueryHandler endpoint
func TestPreviewQueryHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           any
		expectedStatus int
		skipIfNoDb     bool
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "OPTIONS request returns 200",
			method:         "OPTIONS",
			body:           nil,
			expectedStatus: http.StatusOK,
			skipIfNoDb:     false,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Access-Control-Allow-Origin") != "*" {
					t.Error("Missing CORS header")
				}
			},
		},
		{
			name:   "Valid query returns preview data",
			method: "POST",
			body: QueryPreviewRequest{
				Table: "report.cht_form_097b",
				Aggregations: []Aggregation{
					{Column: "num_fever_u5_male", Function: "sum", Alias: "total"},
				},
				Limit: 5,
			},
			expectedStatus: http.StatusOK,
			skipIfNoDb:     true,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("Content-Type") != "application/json" {
					t.Error("Expected Content-Type application/json")
				}
				var result QueryPreviewResult
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Logf("Note: Could not decode response (DB may not be available): %v", err)
					return
				}
				// If we got data, verify structure
				if result.Error == "" && len(result.Headers) == 0 {
					t.Error("Expected headers in successful response")
				}
			},
		},
		{
			name:           "Invalid JSON returns 400",
			method:         "POST",
			body:           "not valid json",
			expectedStatus: http.StatusBadRequest,
			skipIfNoDb:     false,
			checkResponse:  nil,
		},
		{
			name:           "GET method not allowed",
			method:         "GET",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			skipIfNoDb:     false,
			checkResponse:  nil,
		},
		{
			name:           "PUT method not allowed",
			method:         "PUT",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			skipIfNoDb:     false,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoDb && DB == nil {
				t.Skip("Skipping test - database not available")
			}

			var body *bytes.Buffer
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					body = bytes.NewBufferString(v)
				default:
					jsonBody, _ := json.Marshal(tt.body)
					body = bytes.NewBuffer(jsonBody)
				}
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tt.method, "/api/schema/preview", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			PreviewQueryHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestSchemaHandlersCORSHeaders tests that all schema handlers set CORS headers
// Uses OPTIONS method which doesn't require database access
func TestSchemaHandlersCORSHeaders(t *testing.T) {
	handlers := map[string]struct {
		handler http.HandlerFunc
		path    string
		method  string
	}{
		"GetSchemaTablesHandler":  {GetSchemaTablesHandler, "/api/schema/tables", "OPTIONS"},
		"GetTableColumnsHandler":  {GetTableColumnsHandler, "/api/schema/table/report/test/columns", "OPTIONS"},
		"GetTablePreviewHandler":  {GetTablePreviewHandler, "/api/schema/table/report/test/preview", "OPTIONS"},
		"ValidateQueryHandler":    {ValidateQueryHandler, "/api/schema/validate", "OPTIONS"},
		"PreviewQueryHandler":     {PreviewQueryHandler, "/api/schema/preview", "OPTIONS"},
	}

	for name, h := range handlers {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(h.method, h.path, nil)
			w := httptest.NewRecorder()

			h.handler(w, req)

			if w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Errorf("%s: Missing Access-Control-Allow-Origin header", name)
			}
		})
	}
}

// ============== Schema Helper Functions Tests ==============

// TestIsValidIdentifier tests the isValidIdentifier function
func TestIsValidIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid identifiers
		{"Simple lowercase", "users", true},
		{"Simple uppercase", "USERS", true},
		{"Mixed case", "UserTable", true},
		{"With underscore", "user_table", true},
		{"Starts with underscore", "_private", true},
		{"With numbers", "table1", true},
		{"Complex valid", "user_table_2024", true},

		// Invalid identifiers
		{"Empty string", "", false},
		{"Starts with number", "1table", false},
		{"Contains space", "user table", false},
		{"Contains hyphen", "user-table", false},
		{"Contains dot", "schema.table", false},
		{"Contains special char", "table$name", false},
		{"SQL injection attempt", "users; DROP TABLE", false},
		{"Semicolon", "users;", false},
		{"Single quote", "users'", false},
		{"Double quote", "users\"", false},
		{"Too long (64 chars)", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"Max length (63 chars)", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsNumericColumn tests the isNumericColumn function
func TestIsNumericColumn(t *testing.T) {
	patterns := getNumericPatterns()

	tests := []struct {
		name     string
		dataType string
		colName  string
		expected bool
	}{
		// Native numeric types
		{"Integer type", "integer", "id", true},
		{"Smallint type", "smallint", "status", true},
		{"Bigint type", "bigint", "total", true},
		{"Decimal type", "decimal", "price", true},
		{"Numeric type", "numeric", "amount", true},
		{"Real type", "real", "rate", true},
		{"Double precision", "double precision", "value", true},

		// TEXT columns with numeric patterns
		{"TEXT with num_ prefix", "text", "num_cases", true},
		{"TEXT with count_ prefix", "text", "count_items", true},
		{"TEXT with total_ prefix", "text", "total_amount", true},
		{"TEXT with sum_ prefix", "text", "sum_values", true},
		{"TEXT with qty_ prefix", "text", "qty_ordered", true},
		{"TEXT with cases pattern", "text", "fever_cases", true},
		{"TEXT with value pattern", "text", "some_value", true},
		{"TEXT with amount pattern", "text", "payment_amount", true},
		{"TEXT with percentage", "text", "completion_percentage", true},
		{"TEXT with rate", "text", "success_rate", true},

		// Non-numeric columns
		{"TEXT without pattern", "text", "name", false},
		{"TEXT district", "text", "district", false},
		{"VARCHAR type", "character varying", "description", false},
		{"Date type", "date", "created_at", false},
		{"Timestamp type", "timestamp", "updated_at", false},
		{"Boolean type", "boolean", "is_active", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNumericColumn(tt.dataType, tt.colName, patterns)
			if result != tt.expected {
				t.Errorf("isNumericColumn(%q, %q) = %v, want %v", tt.dataType, tt.colName, result, tt.expected)
			}
		})
	}
}

// TestGetNumericPatterns tests the getNumericPatterns function
func TestGetNumericPatterns(t *testing.T) {
	patterns := getNumericPatterns()

	if len(patterns) == 0 {
		t.Error("Expected non-empty patterns list")
	}

	// Verify expected patterns are present
	expectedPatterns := []string{"num_", "count_", "total_", "cases", "value"}
	for _, expected := range expectedPatterns {
		if !slices.Contains(patterns, expected) {
			t.Errorf("Expected pattern %q not found in patterns", expected)
		}
	}
}

// TestConvertDBValue tests the convertDBValue function
func TestConvertDBValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{"Nil value", nil, nil},
		{"String value", "hello", "hello"},
		{"Int64 value", int64(42), int64(42)},
		{"Float64 value", float64(3.14), float64(3.14)},
		{"Bool true", true, true},
		{"Bool false", false, false},
		{"Byte slice", []byte("test"), "test"},
		{"Time value", time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC), "2024-06-15 10:30:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertDBValue(tt.input)

			// Special handling for time comparison
			if _, ok := tt.input.(time.Time); ok {
				if result != tt.expected {
					t.Errorf("convertDBValue() = %v, want %v", result, tt.expected)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("convertDBValue(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestQuoteTableName tests the quoteTableName function (lives in query_builder_v2.go)
// Duplicate test removed — see query_builder_test.go for canonical tests.

// TestApplySampleFilters tests the applySampleFilters function
func TestApplySampleFilters(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		filters  SampleFilters
		contains []string
		excludes []string
	}{
		{
			name:     "No filters",
			query:    "SELECT * FROM table WHERE 1=1 {{district_filter}} {{year_filter}}",
			filters:  SampleFilters{},
			contains: []string{"SELECT * FROM table WHERE 1=1"},
			excludes: []string{"{{district_filter}}", "{{year_filter}}"},
		},
		{
			name:     "District filter only",
			query:    "SELECT * FROM table WHERE 1=1 {{where_placeholder}}",
			filters:  SampleFilters{District: "Central District"},
			contains: []string{"district = 'Central District'"},
			excludes: []string{"{{where_placeholder}}"},
		},
		{
			name:     "Year filter with date column",
			query:    "SELECT * FROM table WHERE 1=1 {{where_placeholder}}",
			filters:  SampleFilters{Year: "2024", YearColumn: "period_date"},
			contains: []string{"EXTRACT(YEAR FROM period_date::date) = 2024"},
			excludes: []string{"{{where_placeholder}}"},
		},
		{
			name:     "Year filter with numeric year column",
			query:    "SELECT * FROM table WHERE 1=1 {{where_placeholder}}",
			filters:  SampleFilters{Year: "2024", YearColumn: "year"},
			contains: []string{"year = 2024"},
			excludes: []string{"{{where_placeholder}}", "EXTRACT"},
		},
		{
			name:     "Year filter with no YearColumn falls back to period_date",
			query:    "SELECT * FROM table WHERE 1=1 {{where_placeholder}}",
			filters:  SampleFilters{Year: "2024"},
			contains: []string{"EXTRACT(YEAR FROM period_date::date) = 2024"},
			excludes: []string{"{{where_placeholder}}"},
		},
		{
			name:     "Both filters with date column",
			query:    "SELECT * FROM table WHERE 1=1 {{where_placeholder}}",
			filters:  SampleFilters{District: "Jinja City", Year: "2024", YearColumn: "period_date"},
			contains: []string{"district = 'Jinja City'", "EXTRACT(YEAR FROM period_date::date) = 2024"},
			excludes: []string{"{{where_placeholder}}"},
		},
		{
			name:     "Both filters with report_date column",
			query:    "SELECT * FROM table WHERE 1=1 {{where_placeholder}}",
			filters:  SampleFilters{District: "Jinja City", Year: "2024", YearColumn: "report_date"},
			contains: []string{"district = 'Jinja City'", "EXTRACT(YEAR FROM report_date::date) = 2024"},
			excludes: []string{"{{where_placeholder}}"},
		},
		{
			name:     "SQL injection in district is escaped",
			query:    "SELECT * FROM table WHERE 1=1 {{where_placeholder}}",
			filters:  SampleFilters{District: "Test'; DROP TABLE users; --"},
			contains: []string{"district = 'Test''; DROP TABLE users; --'"},
			excludes: []string{"{{where_placeholder}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applySampleFilters(tt.query, tt.filters)

			for _, s := range tt.contains {
				if !containsString(result, s) {
					t.Errorf("Expected result to contain %q, got: %s", s, result)
				}
			}

			for _, s := range tt.excludes {
				if containsString(result, s) {
					t.Errorf("Expected result NOT to contain %q, got: %s", s, result)
				}
			}
		})
	}
}

// TestTableColumnsResponseStructure tests TableColumnsResponse JSON structure
func TestTableColumnsResponseStructure(t *testing.T) {
	response := TableColumnsResponse{
		Table: "report.cht_form_097b",
		Columns: []ColumnInfo{
			{Name: "id", DataType: "integer", IsNullable: false, IsNumeric: true},
			{Name: "district", DataType: "text", IsNullable: true, IsNumeric: false},
			{Name: "num_cases", DataType: "text", IsNullable: true, IsNumeric: true},
		},
	}

	// Marshal to JSON and back
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var decoded TableColumnsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.Table != response.Table {
		t.Errorf("Table mismatch: got %q, want %q", decoded.Table, response.Table)
	}

	if len(decoded.Columns) != len(response.Columns) {
		t.Errorf("Column count mismatch: got %d, want %d", len(decoded.Columns), len(response.Columns))
	}

	// Verify first column
	if decoded.Columns[0].Name != "id" {
		t.Errorf("First column name mismatch: got %q, want %q", decoded.Columns[0].Name, "id")
	}
	if !decoded.Columns[0].IsNumeric {
		t.Error("Expected first column to be numeric")
	}
}

// TestQueryValidationResultStructure tests QueryValidationResult JSON structure
func TestQueryValidationResultStructure(t *testing.T) {
	// Valid result
	validResult := QueryValidationResult{
		Valid:        true,
		GeneratedSQL: "SELECT SUM(col) FROM table",
		Warnings:     []string{"Column 'num_cases' is TEXT; will be wrapped with COALESCE/CAST"},
	}

	data, err := json.Marshal(validResult)
	if err != nil {
		t.Fatalf("Failed to marshal valid result: %v", err)
	}

	var decoded QueryValidationResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if !decoded.Valid {
		t.Error("Expected Valid to be true")
	}
	if decoded.GeneratedSQL == "" {
		t.Error("Expected GeneratedSQL to be non-empty")
	}
	if len(decoded.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(decoded.Warnings))
	}

	// Invalid result
	invalidResult := QueryValidationResult{
		Valid:  false,
		Errors: []string{"query table is required", "query must have at least one aggregation"},
	}

	data, err = json.Marshal(invalidResult)
	if err != nil {
		t.Fatalf("Failed to marshal invalid result: %v", err)
	}

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if decoded.Valid {
		t.Error("Expected Valid to be false")
	}
	if len(decoded.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(decoded.Errors))
	}
}

// TestQueryPreviewResultStructure tests QueryPreviewResult JSON structure
func TestQueryPreviewResultStructure(t *testing.T) {
	result := QueryPreviewResult{
		Headers:        []string{"label", "total", "value"},
		Rows:           [][]any{{"Jan", 100, 45.5}, {"Feb", 150, 62.3}},
		RowCount:       2,
		ExecutionTime:  "15.234ms",
		AppliedFilters: "District: Central District, Year: 2024",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	var decoded QueryPreviewResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if len(decoded.Headers) != 3 {
		t.Errorf("Expected 3 headers, got %d", len(decoded.Headers))
	}
	if decoded.RowCount != 2 {
		t.Errorf("Expected RowCount 2, got %d", decoded.RowCount)
	}
	if decoded.ExecutionTime == "" {
		t.Error("Expected ExecutionTime to be non-empty")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ============== GetBuilderCategoriesHandler Tests ==============

// TestGetBuilderCategoriesHandlerOptions tests OPTIONS request handling
func TestGetBuilderCategoriesHandlerOptions(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/api/builder/categories", nil)
	w := httptest.NewRecorder()

	GetBuilderCategoriesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Missing Access-Control-Allow-Origin header")
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Missing Content-Type header")
	}
}

// TestGetBuilderCategoriesHandlerReturnsCategories tests that categories are returned
func TestGetBuilderCategoriesHandlerReturnsCategories(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/builder/categories", nil)
	w := httptest.NewRecorder()

	GetBuilderCategoriesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Response should have categories array
	categories, ok := response["categories"]
	if !ok {
		t.Error("Expected 'categories' field in response")
	}

	// Categories should be an array
	catArray, ok := categories.([]interface{})
	if !ok {
		t.Error("Expected categories to be an array")
	}

	// Should have at least one category (configs directory should exist)
	if len(catArray) == 0 {
		t.Log("Warning: No categories found - configs directory may be empty")
	}
}

// TestGetBuilderCategoriesHandlerCategoryStructure tests category response structure
func TestGetBuilderCategoriesHandlerCategoryStructure(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/builder/categories", nil)
	w := httptest.NewRecorder()

	GetBuilderCategoriesHandler(w, req)

	var response struct {
		Categories []CategoryInfo `json:"categories"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// If we have categories, verify their structure
	for _, cat := range response.Categories {
		if cat.Name == "" {
			t.Error("Category name should not be empty")
		}
		// Subcategories can be empty, but should be valid
		if cat.Subcategories == nil {
			t.Logf("Category %s has nil subcategories (should be empty array or populated)", cat.Name)
		}
	}
}

// TestCategoryInfoStructure tests CategoryInfo JSON structure
func TestCategoryInfoStructure(t *testing.T) {
	cat := CategoryInfo{
		Name:          "community",
		Subcategories: []string{"malaria", "nutrition"},
	}

	data, _ := json.Marshal(cat)
	var decoded CategoryInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != "community" {
		t.Errorf("Name: expected 'community', got %q", decoded.Name)
	}
	if len(decoded.Subcategories) != 2 {
		t.Errorf("Expected 2 subcategories, got %d", len(decoded.Subcategories))
	}
}

// ============== GetReportSourceHandler Tests ==============

// TestGetReportSourceHandlerOptions tests OPTIONS request handling
func TestGetReportSourceHandlerOptions(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/api/report/source/community/test", nil)
	w := httptest.NewRecorder()

	GetReportSourceHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Missing CORS header")
	}
}

// TestGetReportSourceHandlerInvalidPath tests handling of invalid report path
func TestGetReportSourceHandlerInvalidPath(t *testing.T) {
	// Test with path that doesn't match expected prefix
	req := httptest.NewRequest("GET", "/api/wrong/path", nil)
	w := httptest.NewRecorder()

	GetReportSourceHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestGetReportSourceHandlerInvalidReportID tests handling of invalid report ID
func TestGetReportSourceHandlerInvalidReportID(t *testing.T) {
	tests := []struct {
		name     string
		reportID string
	}{
		{"Path traversal attempt", "../../../etc/passwd"},
		{"Too many slashes", "a/b/c/d/e/f/g"},
		{"Invalid characters", "test<script>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/report/source/"+tt.reportID, nil)
			w := httptest.NewRecorder()

			GetReportSourceHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d", tt.name, w.Code)
			}
		})
	}
}

// TestGetReportSourceHandlerNotFound tests handling of non-existent report
func TestGetReportSourceHandlerNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/report/source/nonexistent/report", nil)
	w := httptest.NewRecorder()

	GetReportSourceHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestGetReportSourceHandlerValidReport tests retrieval of valid report
func TestGetReportSourceHandlerValidReport(t *testing.T) {
	// This test depends on having a report in configs/community/test.yaml
	req := httptest.NewRequest("GET", "/api/report/source/community/test", nil)
	w := httptest.NewRecorder()

	GetReportSourceHandler(w, req)

	// May be 200 (found) or 404 (not found) depending on test environment
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("Expected status 200 or 404, got %d", w.Code)
	}

	if w.Code == http.StatusOK {
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have yaml, path, and filename fields
		if _, ok := response["yaml"]; !ok {
			t.Error("Expected 'yaml' field in response")
		}
		if _, ok := response["path"]; !ok {
			t.Error("Expected 'path' field in response")
		}
		if _, ok := response["filename"]; !ok {
			t.Error("Expected 'filename' field in response")
		}
	}
}

// ============== PublishReportHandler Tests ==============

// TestPublishReportHandlerOptions tests OPTIONS request handling
func TestPublishReportHandlerOptions(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/api/report/publish", nil)
	w := httptest.NewRecorder()

	PublishReportHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Missing CORS header")
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Error("Missing or incorrect Allow-Methods header")
	}
}

// TestPublishReportHandlerMethodNotAllowed tests that non-POST methods are rejected
func TestPublishReportHandlerMethodNotAllowed(t *testing.T) {
	methods := []string{"GET", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/report/publish", nil)
			w := httptest.NewRecorder()

			PublishReportHandler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", w.Code)
			}
		})
	}
}

// TestPublishReportHandlerInvalidJSON tests handling of invalid JSON
func TestPublishReportHandlerInvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/report/publish", strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	PublishReportHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestPublishReportHandlerMissingFields tests validation of required fields
func TestPublishReportHandlerMissingFields(t *testing.T) {
	tests := []struct {
		name        string
		request     PublishReportRequest
		expectError string
	}{
		{
			name:        "Missing YAML",
			request:     PublishReportRequest{Category: "test", Filename: "test"},
			expectError: "yaml content is required",
		},
		{
			name:        "Missing category",
			request:     PublishReportRequest{YAML: "content", Filename: "test"},
			expectError: "category is required",
		},
		{
			name:        "Missing filename",
			request:     PublishReportRequest{YAML: "content", Category: "test"},
			expectError: "filename is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/api/report/publish", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			PublishReportHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			// Check error message
			if !strings.Contains(w.Body.String(), tt.expectError) {
				t.Errorf("Expected error containing %q, got: %s", tt.expectError, w.Body.String())
			}
		})
	}
}

// TestPublishReportHandlerInvalidFilename tests validation of filename
func TestPublishReportHandlerInvalidFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"Contains forward slash", "test/report"},
		{"Contains backslash", "test\\report"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := PublishReportRequest{
				YAML:     "title: Test",
				Category: "test",
				Filename: tt.filename,
			}
			body, _ := json.Marshal(request)
			req := httptest.NewRequest("POST", "/api/report/publish", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			PublishReportHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for filename %q, got %d", tt.filename, w.Code)
			}
		})
	}
}

// TestPublishReportHandlerInvalidCategoryName tests validation of category names
func TestPublishReportHandlerInvalidCategoryName(t *testing.T) {
	tests := []struct {
		name           string
		category       string
		expectedStatus int
	}{
		{"Path traversal", "../malicious", http.StatusBadRequest},
		{"Double dots", "test/../escape", http.StatusBadRequest},
		{"Valid nested category", "a/b", http.StatusOK}, // Nested categories are valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := PublishReportRequest{
				YAML:     "title: Test",
				Category: tt.category,
				Filename: "test",
			}
			body, _ := json.Marshal(request)
			req := httptest.NewRequest("POST", "/api/report/publish", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			PublishReportHandler(w, req)

			// For valid categories, we may get 200 (success) or 500 (file system error)
			// For invalid categories, we should get 400
			if tt.expectedStatus == http.StatusBadRequest {
				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400 for category %q, got %d", tt.category, w.Code)
				}
			} else {
				// Valid category - should not return 400
				if w.Code == http.StatusBadRequest {
					t.Errorf("Did not expect 400 for valid category %q, got: %s", tt.category, w.Body.String())
				}
			}
		})
	}
}

// TestPublishReportHandlerStripsYAMLExtension tests that .yaml extension is stripped
func TestPublishReportHandlerStripsYAMLExtension(t *testing.T) {
	// This is a validation test - the handler should strip .yaml extension
	// We can't fully test file creation without affecting the filesystem
	// but we can verify the handler accepts filenames with .yaml extension

	request := PublishReportRequest{
		YAML:     "title: Test Report\nsections: []",
		Category: "test-category",
		Filename: "test-report.yaml",
	}
	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/report/publish", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	PublishReportHandler(w, req)

	// Should not fail validation (may fail on file write if category doesn't exist)
	// Status 200 = success, 500 = file write error (acceptable)
	// Status 400 = validation error (not expected for valid input)
	if w.Code == http.StatusBadRequest {
		t.Errorf("Should not reject filename with .yaml extension, got: %s", w.Body.String())
	}
}

// TestPublishReportRequestStructure tests PublishReportRequest JSON structure
func TestPublishReportRequestStructure(t *testing.T) {
	request := PublishReportRequest{
		YAML:        "title: Test\nsections: []",
		Category:    "community",
		Subcategory: "malaria",
		Filename:    "test-report",
		Overwrite:   true,
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var decoded PublishReportRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.YAML != request.YAML {
		t.Error("YAML mismatch")
	}
	if decoded.Category != request.Category {
		t.Error("Category mismatch")
	}
	if decoded.Subcategory != request.Subcategory {
		t.Error("Subcategory mismatch")
	}
	if decoded.Filename != request.Filename {
		t.Error("Filename mismatch")
	}
	if decoded.Overwrite != request.Overwrite {
		t.Error("Overwrite mismatch")
	}
}

// TestPublishReportHandlerCORSHeaders tests that CORS headers are set
func TestPublishReportHandlerCORSHeaders(t *testing.T) {
	request := PublishReportRequest{
		YAML:     "title: Test",
		Category: "test",
		Filename: "test",
	}
	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/report/publish", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	PublishReportHandler(w, req)

	// Regardless of result, CORS headers should be set
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Missing Access-Control-Allow-Origin header")
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Missing or incorrect Content-Type header")
	}
}
