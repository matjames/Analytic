package main

import (
	"testing"
)

// TestDebugTableBODMASSQL prints the generated SQL for table components with BODMAS operations
func TestDebugTableBODMASSQL(t *testing.T) {
	testCases := []struct {
		name       string
		column     string
		function   string
		alias      string
		shouldContain string
	}{
		{
			name:       "Addition",
			column:     "num_fever_u5_male + num_fever_u5_female",
			function:   "sum",
			alias:      "Addition",
			shouldContain: "+",
		},
		{
			name:       "Subtraction",
			column:     "num_fever_u5_male - num_fever_u5_female",
			function:   "sum",
			alias:      "Subtraction",
			shouldContain: "-",
		},
		{
			name:       "Multiplication",
			column:     "num_fever_u5_male * num_fever_u5_female",
			function:   "sum",
			alias:      "Multiplication",
			shouldContain: "*",
		},
		{
			name:       "Division",
			column:     "num_fever_u5_male / num_fever_u5_female",
			function:   "sum",
			alias:      "Division",
			shouldContain: "/",
		},
		{
			name:       "ComplexPrecedence",
			column:     "(num_fever_u5_male + num_fever_u5_female) * num_malaria_u5_male",
			function:   "sum",
			alias:      "Complex",
			shouldContain: "*",
		},
	}

	for _, tc := range testCases {
		query := Query{
			Table: "report.cht_form_097b",
			Aggregations: []Aggregation{
				{
					Column:   tc.column,
					Function: tc.function,
					Alias:    tc.alias,
				},
			},
			GroupBy: []GroupBy{
				{
					Field: "district",
				},
			},
		}

		sql, err := BuildSQLWithSquirrel(query, "table", "report.cht_form_097b", []string{})
		if err != nil {
			t.Logf("ERROR in %s: %v", tc.name, err)
			continue
		}

		t.Logf("=== Test Case: %s ===", tc.name)
		t.Logf("Column Expression: %s", tc.column)
		t.Logf("Expected to contain: %s", tc.shouldContain)
		t.Logf("Generated SQL:\n%s\n", sql)

		// Check if the SQL contains the expected operator
		if tc.shouldContain != "" {
			found := false
			for _, c := range sql {
				if string(c) == tc.shouldContain {
					found = true
					break
				}
			}
			if !found {
				t.Logf("WARNING: SQL does not contain expected operator '%s'", tc.shouldContain)
			}
		}

		// Check COALESCE wrapping
		coalesceCount := 0
		for i := 0; i < len(sql)-7; i++ {
			if sql[i:i+8] == "COALESCE" {
				coalesceCount++
			}
		}
		t.Logf("COALESCE wrapping count: %d (expected >= 2 for 2 columns)\n", coalesceCount)
	}
}
