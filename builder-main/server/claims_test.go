package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests verify the documented claims in CLAUDE.md about the service's
// security guarantees and behavioral contracts.

// ============== Claim: Read-only by design ==============
// "No INSERT/UPDATE/DELETE" — the query builder must only produce SELECT statements.

func TestQueryBuilderProducesOnlySelectStatements(t *testing.T) {
	tests := []struct {
		name  string
		query Query
		ctype string
	}{
		{
			name: "simple aggregation",
			query: Query{
				Table: "report.test_table",
				Aggregations: []Aggregation{
					{Column: "value_col", Function: "sum", Alias: "total"},
				},
			},
			ctype: "text",
		},
		{
			name: "grouped bar chart",
			query: Query{
				Table: "report.test_table",
				Aggregations: []Aggregation{
					{Column: "cases", Function: "count", Alias: "case_count"},
				},
				GroupBy: []GroupBy{
					{Field: "period_date", Format: "month"},
				},
			},
			ctype: "bar",
		},
		{
			name: "multiple aggregations with calculate",
			query: Query{
				Table: "report.test_table",
				Aggregations: []Aggregation{
					{Column: "positive", Function: "sum", Alias: "positive"},
					{Column: "tested", Function: "sum", Alias: "tested"},
				},
				GroupBy: []GroupBy{
					{Field: "period_date", Format: "year"},
				},
				Calculate: []Calculate{
					{Formula: "(positive / tested * 100)", Alias: "rate", RoundTo: intPtr(1)},
				},
			},
			ctype: "line",
		},
		{
			name: "with where clause",
			query: Query{
				Table: "report.test_table",
				Aggregations: []Aggregation{
					{Column: "value", Function: "avg", Alias: "avg_val"},
				},
				Where: "facility_type = 'hospital'",
			},
			ctype: "table",
		},
		{
			name: "with limit",
			query: Query{
				Table: "report.test_table",
				Aggregations: []Aggregation{
					{Column: "value", Function: "max", Alias: "max_val"},
				},
				Limit: 10,
			},
			ctype: "text",
		},
	}

	writeKeywords := []string{"INSERT ", "UPDATE ", "DELETE ", "DROP ", "ALTER ", "CREATE ", "TRUNCATE "}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := BuildSQLWithSquirrel(tc.query, tc.ctype, tc.query.Table, []string{"district", "year", "month"})
			if err != nil {
				t.Fatalf("BuildSQLWithSquirrel failed: %v", err)
			}

			upper := strings.ToUpper(sql)
			if !strings.HasPrefix(upper, "SELECT ") {
				t.Errorf("generated SQL does not start with SELECT: %s", sql)
			}

			for _, kw := range writeKeywords {
				if strings.Contains(upper, kw) {
					t.Errorf("generated SQL contains write keyword %q: %s", strings.TrimSpace(kw), sql)
				}
			}
		})
	}
}

// ============== Claim: Where clause injection blocked ==============
// The where clause validator must reject dangerous SQL patterns.

func TestWhereClauseBlocksDangerousPatterns(t *testing.T) {
	tests := []struct {
		name  string
		where string
	}{
		{"semicolon injection", "1=1; DROP TABLE users"},
		{"comment injection", "1=1 -- AND secret = 'x'"},
		{"subquery injection", "id IN (SELECT password FROM users)"},
		{"subquery case-insensitive", "id IN (select password from users)"},
		{"unbalanced parens open", "((col = 1)"},
		{"unbalanced parens close", "(col = 1))"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWhereClause(tc.where)
			if err == nil {
				t.Errorf("expected where clause to be rejected: %q", tc.where)
			}
		})
	}
}

func TestWhereClauseAllowsSafePatterns(t *testing.T) {
	safe := []string{
		"facility_type = 'hospital'",
		"(age > 5 AND age < 15)",
		"status IN ('active', 'pending')",
		"",
	}

	for _, w := range safe {
		if err := validateWhereClause(w); err != nil {
			t.Errorf("safe where clause rejected: %q — %v", w, err)
		}
	}
}

// ============== Claim: Rate limiting at per-IP and global layers ==============
// "100 req/sec default" — per-IP limit (20/sec) should reject bursts from a single IP.

func TestPerIPRateLimitRejectsBursts(t *testing.T) {
	// Create a fresh per-IP limiter to avoid interference from other tests
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RateLimitMiddleware(okHandler)

	// Per-IP burst is 40 by default. Sending 50 requests from the same IP
	// should cause some to be rejected.
	allowed := 0
	rejected := 0

	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		// Use a unique IP that won't collide with other tests
		req.Header.Set("X-Real-IP", "10.99.99.99")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			allowed++
		} else if w.Code == http.StatusTooManyRequests {
			rejected++
		}
	}

	if rejected == 0 {
		t.Errorf("expected some requests to be rate-limited, but all %d were allowed", allowed)
	}

	// The rejected response should be JSON with an error message
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.99.99.99")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == http.StatusTooManyRequests {
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("rate limit response is not valid JSON: %v", err)
		}
		if body["error"] == "" {
			t.Error("rate limit response should contain an error message")
		}
	}
}

// ============== Claim: Table output uses empty arrays, not nil ==============
// "Always empty arrays" — table transformer must return non-nil headers and rows.

func TestTableTransformerEmptyDataReturnsNonNilArrays(t *testing.T) {
	transformer := &TableTransformer{}

	component := &Component{
		Type: "table",
	}

	// Empty table data
	result, err := transformer.Transform(TableData{}, component)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Serialize to JSON and check for null vs empty array
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	jsonStr := string(jsonBytes)

	// The JSON should not contain "null" for headers or rows
	// TableData with zero-value slices will serialize headers/rows as null
	// This test documents current behavior — if it fails with null, that's a real bug
	if strings.Contains(jsonStr, `"headers":null`) {
		t.Error("table headers serialized as null — frontend expects an empty array")
	}
	if strings.Contains(jsonStr, `"rows":null`) {
		t.Error("table rows serialized as null — frontend expects an empty array")
	}
}

func TestChartTransformerEmptyDataReturnsNonNilArrays(t *testing.T) {
	transformer := &ChartTransformer{}

	for _, chartType := range []string{"bar", "line", "pie", "bar_line"} {
		t.Run(chartType, func(t *testing.T) {
			component := &Component{Type: chartType}
			result, err := transformer.Transform(TableData{}, component)
			if err != nil {
				t.Fatalf("Transform failed: %v", err)
			}

			jsonBytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("JSON marshal failed: %v", err)
			}

			jsonStr := string(jsonBytes)
			if strings.Contains(jsonStr, `"labels":null`) {
				t.Error("chart labels serialized as null — frontend expects an empty array")
			}
			if strings.Contains(jsonStr, `"datasets":null`) {
				t.Error("chart datasets serialized as null — frontend expects an empty array")
			}
		})
	}
}

// ============== Claim: Aggregation function whitelist ==============
// "Whitelist validation (not blocklist)" — only SUM/COUNT/COUNT_DISTINCT/AVG/MAX/MIN allowed.

func TestAggregationFunctionWhitelistRejectsInvalid(t *testing.T) {
	dangerous := []string{
		"EXEC",
		"DELETE",
		"DROP",
		"INSERT",
		"UPDATE",
		"SLEEP",
		"BENCHMARK",
		"xp_cmdshell",
		"SUM; DROP TABLE",
	}

	for _, fn := range dangerous {
		t.Run(fn, func(t *testing.T) {
			query := Query{
				Table: "report.test_table",
				Aggregations: []Aggregation{
					{Column: "value", Function: fn, Alias: "result"},
				},
			}
			_, err := BuildSQLWithSquirrel(query, "text", query.Table, nil)
			if err == nil {
				t.Errorf("expected aggregation function %q to be rejected", fn)
			}
		})
	}
}

func TestAggregationFunctionWhitelistAcceptsValid(t *testing.T) {
	valid := []string{"sum", "count", "count_distinct", "avg", "max", "min", "variance", "stddev",
		"SUM", "COUNT", "COUNT_DISTINCT", "AVG", "MAX", "MIN", "VARIANCE", "STDDEV"}

	for _, fn := range valid {
		t.Run(fn, func(t *testing.T) {
			query := Query{
				Table: "report.test_table",
				Aggregations: []Aggregation{
					{Column: "value", Function: fn, Alias: "result"},
				},
			}
			_, err := BuildSQLWithSquirrel(query, "text", query.Table, nil)
			if err != nil {
				t.Errorf("valid aggregation function %q was rejected: %v", fn, err)
			}
		})
	}
}
