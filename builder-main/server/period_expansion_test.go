package main

import (
	"context"
	"strings"
	"testing"
)

// TestGetPeriodTypeFromGroupBy tests detection of period type from groupBy configuration
func TestGetPeriodTypeFromGroupBy(t *testing.T) {
	tests := []struct {
		name     string
		groupBy  []GroupBy
		expected string
	}{
		{
			name:     "month format",
			groupBy:  []GroupBy{{Field: "period_date", Format: "month"}},
			expected: "month",
		},
		{
			name:     "week format",
			groupBy:  []GroupBy{{Field: "period", Format: "week"}},
			expected: "week",
		},
		{
			name:     "year format",
			groupBy:  []GroupBy{{Field: "period_date", Format: "year"}},
			expected: "year",
		},
		{
			name:     "no format returns empty",
			groupBy:  []GroupBy{{Field: "district"}},
			expected: "",
		},
		{
			name:     "empty groupBy returns empty",
			groupBy:  []GroupBy{},
			expected: "",
		},
		{
			name:     "nil groupBy returns empty",
			groupBy:  nil,
			expected: "",
		},
		{
			name: "multiple groupBy with month",
			groupBy: []GroupBy{
				{Field: "district"},
				{Field: "period_date", Format: "month"},
			},
			expected: "month",
		},
		// Field name inference tests (for tables without format)
		{
			name:     "field month no format",
			groupBy:  []GroupBy{{Field: "month"}},
			expected: "month",
		},
		{
			name:     "field week no format",
			groupBy:  []GroupBy{{Field: "week"}},
			expected: "week",
		},
		{
			name:     "field quarter no format",
			groupBy:  []GroupBy{{Field: "quarter"}},
			expected: "quarter",
		},
		{
			name:     "field Month uppercase",
			groupBy:  []GroupBy{{Field: "Month"}},
			expected: "month",
		},
		{
			name:     "field WEEK uppercase",
			groupBy:  []GroupBy{{Field: "WEEK"}},
			expected: "week",
		},
		{
			name:     "format takes precedence over field",
			groupBy:  []GroupBy{{Field: "month", Format: "week_year"}},
			expected: "week",
		},
		{
			name: "multiple groupBy with field inference",
			groupBy: []GroupBy{
				{Field: "district"},
				{Field: "month"},
			},
			expected: "month",
		},
		{
			name:     "field year no format (inferred)",
			groupBy:  []GroupBy{{Field: "year"}},
			expected: "year",
		},
		{
			name: "field year with month prefers month",
			groupBy: []GroupBy{
				{Field: "year"},
				{Field: "month"},
			},
			expected: "month",
		},
		{
			name: "field year alone with district",
			groupBy: []GroupBy{
				{Field: "district"},
				{Field: "year"},
			},
			expected: "year",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPeriodTypeFromGroupBy(tt.groupBy)
			if result != tt.expected {
				t.Errorf("GetPeriodTypeFromGroupBy() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestExpandMonthRange_SameYear tests month expansion within a single year
func TestExpandMonthRange_SameYear(t *testing.T) {
	tests := []struct {
		name           string
		year           int
		month          int
		count          int
		expectedYears  string
		expectedMonths string
	}{
		{
			name:           "6 months from June",
			year:           2024,
			month:          6,
			count:          6,
			expectedYears:  "2024",
			expectedMonths: "1,2,3,4,5,6",
		},
		{
			name:           "3 months from December",
			year:           2024,
			month:          12,
			count:          3,
			expectedYears:  "2024",
			expectedMonths: "10,11,12",
		},
		{
			name:           "1 month returns single month",
			year:           2024,
			month:          5,
			count:          1,
			expectedYears:  "2024",
			expectedMonths: "5",
		},
		{
			name:           "12 months from December",
			year:           2024,
			month:          12,
			count:          12,
			expectedYears:  "2024",
			expectedMonths: "1,2,3,4,5,6,7,8,9,10,11,12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, months := ExpandMonthRange(tt.year, tt.month, tt.count)
			if years != tt.expectedYears {
				t.Errorf("ExpandMonthRange() years = %q, expected %q", years, tt.expectedYears)
			}
			if months != tt.expectedMonths {
				t.Errorf("ExpandMonthRange() months = %q, expected %q", months, tt.expectedMonths)
			}
		})
	}
}

// TestExpandMonthRange_CrossYear tests month expansion crossing year boundaries
func TestExpandMonthRange_CrossYear(t *testing.T) {
	tests := []struct {
		name           string
		year           int
		month          int
		count          int
		expectedYears  string
		expectedMonths string
	}{
		{
			name:           "6 months from March crosses to previous year",
			year:           2024,
			month:          3,
			count:          6,
			expectedYears:  "2023,2024",
			expectedMonths: "10,11,12,1,2,3",
		},
		{
			name:           "6 months from January crosses to previous year",
			year:           2024,
			month:          1,
			count:          6,
			expectedYears:  "2023,2024",
			expectedMonths: "8,9,10,11,12,1",
		},
		{
			name:           "12 months from January spans full previous year",
			year:           2024,
			month:          1,
			count:          12,
			expectedYears:  "2023,2024",
			expectedMonths: "2,3,4,5,6,7,8,9,10,11,12,1",
		},
		{
			name:           "18 months from March spans 2+ years",
			year:           2024,
			month:          3,
			count:          18,
			expectedYears:  "2022,2023,2024",
			expectedMonths: "10,11,12,1,2,3,4,5,6,7,8,9,10,11,12,1,2,3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, months := ExpandMonthRange(tt.year, tt.month, tt.count)
			if years != tt.expectedYears {
				t.Errorf("ExpandMonthRange() years = %q, expected %q", years, tt.expectedYears)
			}
			if months != tt.expectedMonths {
				t.Errorf("ExpandMonthRange() months = %q, expected %q", months, tt.expectedMonths)
			}
		})
	}
}

// TestExpandMonthRange_EdgeCases tests edge cases for month expansion
func TestExpandMonthRange_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		year           int
		month          int
		count          int
		expectedYears  string
		expectedMonths string
	}{
		{
			name:           "zero count returns single month",
			year:           2024,
			month:          6,
			count:          0,
			expectedYears:  "2024",
			expectedMonths: "6",
		},
		{
			name:           "negative count returns single month",
			year:           2024,
			month:          6,
			count:          -5,
			expectedYears:  "2024",
			expectedMonths: "6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, months := ExpandMonthRange(tt.year, tt.month, tt.count)
			if years != tt.expectedYears {
				t.Errorf("ExpandMonthRange() years = %q, expected %q", years, tt.expectedYears)
			}
			if months != tt.expectedMonths {
				t.Errorf("ExpandMonthRange() months = %q, expected %q", months, tt.expectedMonths)
			}
		})
	}
}

// TestExpandWeekRange_SameYear tests week expansion within a single year
func TestExpandWeekRange_SameYear(t *testing.T) {
	tests := []struct {
		name       string
		anchorWeek string
		count      int
		expected   string
	}{
		{
			name:       "6 weeks from week 20",
			anchorWeek: "2024W20",
			count:      6,
			expected:   "2024W15,2024W16,2024W17,2024W18,2024W19,2024W20",
		},
		{
			name:       "3 weeks from week 10",
			anchorWeek: "2024W10",
			count:      3,
			expected:   "2024W08,2024W09,2024W10",
		},
		{
			name:       "1 week returns single week",
			anchorWeek: "2024W15",
			count:      1,
			expected:   "2024W15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandWeekRange(tt.anchorWeek, tt.count)
			if result != tt.expected {
				t.Errorf("ExpandWeekRange() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestExpandWeekRange_CrossYear tests week expansion crossing year boundaries
func TestExpandWeekRange_CrossYear(t *testing.T) {
	tests := []struct {
		name       string
		anchorWeek string
		count      int
		expected   string
	}{
		{
			name:       "6 weeks from week 2 crosses to previous year",
			anchorWeek: "2024W02",
			count:      6,
			expected:   "2023W49,2023W50,2023W51,2023W52,2024W01,2024W02",
		},
		{
			name:       "3 weeks from week 1 crosses to previous year",
			anchorWeek: "2024W01",
			count:      3,
			expected:   "2023W51,2023W52,2024W01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandWeekRange(tt.anchorWeek, tt.count)
			if result != tt.expected {
				t.Errorf("ExpandWeekRange() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestExpandWeekRange_EdgeCases tests edge cases for week expansion
func TestExpandWeekRange_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		anchorWeek string
		count      int
		expected   string
	}{
		{
			name:       "zero count returns anchor",
			anchorWeek: "2024W20",
			count:      0,
			expected:   "2024W20",
		},
		{
			name:       "negative count returns anchor",
			anchorWeek: "2024W20",
			count:      -5,
			expected:   "2024W20",
		},
		{
			name:       "empty anchor returns empty",
			anchorWeek: "",
			count:      6,
			expected:   "",
		},
		{
			name:       "invalid anchor format returns anchor",
			anchorWeek: "invalid",
			count:      6,
			expected:   "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandWeekRange(tt.anchorWeek, tt.count)
			if result != tt.expected {
				t.Errorf("ExpandWeekRange() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestGetISOWeeksInYear tests the ISO week calculation
func TestGetISOWeeksInYear(t *testing.T) {
	// Known years with 53 weeks: 2004, 2009, 2015, 2020, 2026
	// Most years have 52 weeks
	tests := []struct {
		year     int
		expected int
	}{
		{2023, 52}, // Normal year
		{2024, 52}, // Leap year but 52 weeks
		{2020, 53}, // Has 53 weeks
		{2015, 53}, // Has 53 weeks
		{2022, 52}, // Normal year
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.year)), func(t *testing.T) {
			result := getISOWeeksInYear(tt.year)
			if result != tt.expected {
				t.Errorf("getISOWeeksInYear(%d) = %d, expected %d", tt.year, result, tt.expected)
			}
		})
	}
}

// TestExpandMonthRange_ChronologicalOrder verifies months are in chronological order
func TestExpandMonthRange_ChronologicalOrder(t *testing.T) {
	// Test that output is chronological (oldest to newest)
	_, months := ExpandMonthRange(2024, 3, 6)

	monthList := strings.Split(months, ",")
	if len(monthList) != 6 {
		t.Errorf("Expected 6 months, got %d", len(monthList))
	}

	// First month should be October (10), last should be March (3)
	if monthList[0] != "10" {
		t.Errorf("First month should be 10 (October), got %s", monthList[0])
	}
	if monthList[5] != "3" {
		t.Errorf("Last month should be 3 (March), got %s", monthList[5])
	}
}

// TestExpandWeekRange_ChronologicalOrder verifies weeks are in chronological order
func TestExpandWeekRange_ChronologicalOrder(t *testing.T) {
	// Test that output is chronological (oldest to newest)
	weeks := ExpandWeekRange("2024W02", 6)

	weekList := strings.Split(weeks, ",")
	if len(weekList) != 6 {
		t.Errorf("Expected 6 weeks, got %d", len(weekList))
	}

	// First week should be 2023W49, last should be 2024W02
	if weekList[0] != "2023W49" {
		t.Errorf("First week should be 2023W49, got %s", weekList[0])
	}
	if weekList[5] != "2024W02" {
		t.Errorf("Last week should be 2024W02, got %s", weekList[5])
	}
}

// TestExpandMonthRangeWithYearMap tests month expansion with year map
func TestExpandMonthRangeWithYearMap(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		month    int
		count    int
		expected map[string][]int
	}{
		{
			name:  "single year",
			year:  2024,
			month: 6,
			count: 3,
			expected: map[string][]int{
				"2024": {6, 5, 4},
			},
		},
		{
			name:  "cross year boundary",
			year:  2024,
			month: 2,
			count: 5,
			expected: map[string][]int{
				"2023": {12, 11, 10},
				"2024": {2, 1},
			},
		},
		{
			name:  "zero count returns single month",
			year:  2024,
			month: 6,
			count: 0,
			expected: map[string][]int{
				"2024": {6},
			},
		},
		{
			name:  "negative count returns single month",
			year:  2024,
			month: 6,
			count: -5,
			expected: map[string][]int{
				"2024": {6},
			},
		},
		{
			name:  "full year",
			year:  2024,
			month: 12,
			count: 12,
			expected: map[string][]int{
				"2024": {12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandMonthRangeWithYearMap(tt.year, tt.month, tt.count)

			// Check same number of years
			if len(result) != len(tt.expected) {
				t.Errorf("ExpandMonthRangeWithYearMap() got %d years, expected %d", len(result), len(tt.expected))
				return
			}

			// Check each year has correct months
			for year, expectedMonths := range tt.expected {
				gotMonths, ok := result[year]
				if !ok {
					t.Errorf("ExpandMonthRangeWithYearMap() missing year %s", year)
					continue
				}
				if len(gotMonths) != len(expectedMonths) {
					t.Errorf("ExpandMonthRangeWithYearMap() year %s: got %d months, expected %d", year, len(gotMonths), len(expectedMonths))
					continue
				}
				for i, m := range expectedMonths {
					if gotMonths[i] != m {
						t.Errorf("ExpandMonthRangeWithYearMap() year %s month %d: got %d, expected %d", year, i, gotMonths[i], m)
					}
				}
			}
		})
	}
}

// TestExpandQuarterRange_SameYear tests quarter expansion within a single year
func TestExpandQuarterRange_SameYear(t *testing.T) {
	tests := []struct {
		name             string
		year             int
		quarter          int
		count            int
		expectedYears    string
		expectedQuarters string
	}{
		{
			name:             "4 quarters from Q4",
			year:             2024,
			quarter:          4,
			count:            4,
			expectedYears:    "2024",
			expectedQuarters: "1,2,3,4",
		},
		{
			name:             "2 quarters from Q3",
			year:             2024,
			quarter:          3,
			count:            2,
			expectedYears:    "2024",
			expectedQuarters: "2,3",
		},
		{
			name:             "1 quarter returns single quarter",
			year:             2024,
			quarter:          2,
			count:            1,
			expectedYears:    "2024",
			expectedQuarters: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, quarters := ExpandQuarterRange(tt.year, tt.quarter, tt.count)
			if years != tt.expectedYears {
				t.Errorf("ExpandQuarterRange() years = %q, expected %q", years, tt.expectedYears)
			}
			if quarters != tt.expectedQuarters {
				t.Errorf("ExpandQuarterRange() quarters = %q, expected %q", quarters, tt.expectedQuarters)
			}
		})
	}
}

// TestExpandQuarterRange_CrossYear tests quarter expansion crossing year boundaries
func TestExpandQuarterRange_CrossYear(t *testing.T) {
	tests := []struct {
		name             string
		year             int
		quarter          int
		count            int
		expectedYears    string
		expectedQuarters string
	}{
		{
			name:             "6 quarters from Q2 crosses to previous year",
			year:             2024,
			quarter:          2,
			count:            6,
			expectedYears:    "2023,2024",
			expectedQuarters: "1,2,3,4,1,2",
		},
		{
			name:             "4 quarters from Q1 crosses to previous year",
			year:             2024,
			quarter:          1,
			count:            4,
			expectedYears:    "2023,2024",
			expectedQuarters: "2,3,4,1",
		},
		{
			name:             "8 quarters spans 2+ years",
			year:             2024,
			quarter:          2,
			count:            8,
			expectedYears:    "2022,2023,2024",
			expectedQuarters: "3,4,1,2,3,4,1,2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, quarters := ExpandQuarterRange(tt.year, tt.quarter, tt.count)
			if years != tt.expectedYears {
				t.Errorf("ExpandQuarterRange() years = %q, expected %q", years, tt.expectedYears)
			}
			if quarters != tt.expectedQuarters {
				t.Errorf("ExpandQuarterRange() quarters = %q, expected %q", quarters, tt.expectedQuarters)
			}
		})
	}
}

// TestExpandQuarterRange_EdgeCases tests edge cases for quarter expansion
func TestExpandQuarterRange_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		year             int
		quarter          int
		count            int
		expectedYears    string
		expectedQuarters string
	}{
		{
			name:             "zero count returns single quarter",
			year:             2024,
			quarter:          3,
			count:            0,
			expectedYears:    "2024",
			expectedQuarters: "3",
		},
		{
			name:             "negative count returns single quarter",
			year:             2024,
			quarter:          3,
			count:            -5,
			expectedYears:    "2024",
			expectedQuarters: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years, quarters := ExpandQuarterRange(tt.year, tt.quarter, tt.count)
			if years != tt.expectedYears {
				t.Errorf("ExpandQuarterRange() years = %q, expected %q", years, tt.expectedYears)
			}
			if quarters != tt.expectedQuarters {
				t.Errorf("ExpandQuarterRange() quarters = %q, expected %q", quarters, tt.expectedQuarters)
			}
		})
	}
}

// TestExpandQuarterRangeWithYearMap tests quarter expansion with year map
func TestExpandQuarterRangeWithYearMap(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		quarter  int
		count    int
		expected map[string][]int
	}{
		{
			name:    "single year",
			year:    2024,
			quarter: 4,
			count:   3,
			expected: map[string][]int{
				"2024": {4, 3, 2},
			},
		},
		{
			name:    "cross year boundary",
			year:    2024,
			quarter: 2,
			count:   6,
			expected: map[string][]int{
				"2023": {4, 3, 2, 1},
				"2024": {2, 1},
			},
		},
		{
			name:    "zero count returns single quarter",
			year:    2024,
			quarter: 3,
			count:   0,
			expected: map[string][]int{
				"2024": {3},
			},
		},
		{
			name:    "negative count returns single quarter",
			year:    2024,
			quarter: 3,
			count:   -5,
			expected: map[string][]int{
				"2024": {3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandQuarterRangeWithYearMap(tt.year, tt.quarter, tt.count)

			if len(result) != len(tt.expected) {
				t.Errorf("ExpandQuarterRangeWithYearMap() got %d years, expected %d", len(result), len(tt.expected))
				return
			}

			for year, expectedQuarters := range tt.expected {
				gotQuarters, ok := result[year]
				if !ok {
					t.Errorf("ExpandQuarterRangeWithYearMap() missing year %s", year)
					continue
				}
				if len(gotQuarters) != len(expectedQuarters) {
					t.Errorf("ExpandQuarterRangeWithYearMap() year %s: got %d quarters, expected %d", year, len(gotQuarters), len(expectedQuarters))
					continue
				}
				for i, q := range expectedQuarters {
					if gotQuarters[i] != q {
						t.Errorf("ExpandQuarterRangeWithYearMap() year %s quarter %d: got %d, expected %d", year, i, gotQuarters[i], q)
					}
				}
			}
		})
	}
}

// TestGeneratePeriodFacets_Month tests month facet generation
func TestGeneratePeriodFacets_Month(t *testing.T) {
	facets := GeneratePeriodFacets("month", 4, 2024, 3)

	if len(facets) != 4 {
		t.Fatalf("Expected 4 facets, got %d", len(facets))
	}

	// Check chronological order (oldest first)
	expected := []struct {
		label string
		year  int
		month int
		value string
	}{
		{"Dec 2023", 2023, 12, "2023-12"},
		{"Jan 2024", 2024, 1, "2024-01"},
		{"Feb 2024", 2024, 2, "2024-02"},
		{"Mar 2024", 2024, 3, "2024-03"},
	}

	for i, exp := range expected {
		if facets[i].Label != exp.label {
			t.Errorf("Facet %d: label = %q, expected %q", i, facets[i].Label, exp.label)
		}
		if facets[i].Year != exp.year {
			t.Errorf("Facet %d: year = %d, expected %d", i, facets[i].Year, exp.year)
		}
		if facets[i].Period != exp.month {
			t.Errorf("Facet %d: period = %d, expected %d", i, facets[i].Period, exp.month)
		}
		if facets[i].Value != exp.value {
			t.Errorf("Facet %d: value = %q, expected %q", i, facets[i].Value, exp.value)
		}
	}
}

// TestGeneratePeriodFacets_Quarter tests quarter facet generation
func TestGeneratePeriodFacets_Quarter(t *testing.T) {
	facets := GeneratePeriodFacets("quarter", 4, 2024, 2)

	if len(facets) != 4 {
		t.Fatalf("Expected 4 facets, got %d", len(facets))
	}

	expected := []struct {
		label   string
		year    int
		quarter int
		value   string
	}{
		{"Q3 2023", 2023, 3, "2023Q3"},
		{"Q4 2023", 2023, 4, "2023Q4"},
		{"Q1 2024", 2024, 1, "2024Q1"},
		{"Q2 2024", 2024, 2, "2024Q2"},
	}

	for i, exp := range expected {
		if facets[i].Label != exp.label {
			t.Errorf("Facet %d: label = %q, expected %q", i, facets[i].Label, exp.label)
		}
		if facets[i].Year != exp.year {
			t.Errorf("Facet %d: year = %d, expected %d", i, facets[i].Year, exp.year)
		}
		if facets[i].Period != exp.quarter {
			t.Errorf("Facet %d: period = %d, expected %d", i, facets[i].Period, exp.quarter)
		}
		if facets[i].Value != exp.value {
			t.Errorf("Facet %d: value = %q, expected %q", i, facets[i].Value, exp.value)
		}
	}
}

// TestGeneratePeriodFacets_Year tests year facet generation
func TestGeneratePeriodFacets_Year(t *testing.T) {
	facets := GeneratePeriodFacets("year", 3, 2024, 0)

	if len(facets) != 3 {
		t.Fatalf("Expected 3 facets, got %d", len(facets))
	}

	expected := []struct {
		label string
		year  int
		value string
	}{
		{"2022", 2022, "2022"},
		{"2023", 2023, "2023"},
		{"2024", 2024, "2024"},
	}

	for i, exp := range expected {
		if facets[i].Label != exp.label {
			t.Errorf("Facet %d: label = %q, expected %q", i, facets[i].Label, exp.label)
		}
		if facets[i].Year != exp.year {
			t.Errorf("Facet %d: year = %d, expected %d", i, facets[i].Year, exp.year)
		}
		if facets[i].Period != 0 {
			t.Errorf("Facet %d: period = %d, expected 0", i, facets[i].Period)
		}
		if facets[i].Value != exp.value {
			t.Errorf("Facet %d: value = %q, expected %q", i, facets[i].Value, exp.value)
		}
	}
}

// TestGeneratePeriodFacets_Week tests week facet generation
func TestGeneratePeriodFacets_Week(t *testing.T) {
	facets := GeneratePeriodFacets("week", 4, 2024, 10)

	if len(facets) != 4 {
		t.Fatalf("Expected 4 facets, got %d", len(facets))
	}

	// Check chronological order (oldest first)
	expected := []struct {
		label string
		year  int
		week  int
		value string
	}{
		{"W07 2024", 2024, 7, "2024W07"},
		{"W08 2024", 2024, 8, "2024W08"},
		{"W09 2024", 2024, 9, "2024W09"},
		{"W10 2024", 2024, 10, "2024W10"},
	}

	for i, exp := range expected {
		if facets[i].Label != exp.label {
			t.Errorf("Facet %d: label = %q, expected %q", i, facets[i].Label, exp.label)
		}
		if facets[i].Year != exp.year {
			t.Errorf("Facet %d: year = %d, expected %d", i, facets[i].Year, exp.year)
		}
		if facets[i].Period != exp.week {
			t.Errorf("Facet %d: period = %d, expected %d", i, facets[i].Period, exp.week)
		}
		if facets[i].Value != exp.value {
			t.Errorf("Facet %d: value = %q, expected %q", i, facets[i].Value, exp.value)
		}
	}
}

// TestGeneratePeriodFacets_Week_CrossYear tests week facets crossing year boundary
func TestGeneratePeriodFacets_Week_CrossYear(t *testing.T) {
	// Anchor at W02 2024 with 4 facets should cross back into 2023
	facets := GeneratePeriodFacets("week", 4, 2024, 2)

	if len(facets) != 4 {
		t.Fatalf("Expected 4 facets, got %d", len(facets))
	}

	// 2023 has 52 weeks, so going back from W02: W02, W01, W52, W51
	expected := []struct {
		label string
		year  int
		week  int
		value string
	}{
		{"W51 2023", 2023, 51, "2023W51"},
		{"W52 2023", 2023, 52, "2023W52"},
		{"W01 2024", 2024, 1, "2024W01"},
		{"W02 2024", 2024, 2, "2024W02"},
	}

	for i, exp := range expected {
		if facets[i].Label != exp.label {
			t.Errorf("Facet %d: label = %q, expected %q", i, facets[i].Label, exp.label)
		}
		if facets[i].Year != exp.year {
			t.Errorf("Facet %d: year = %d, expected %d", i, facets[i].Year, exp.year)
		}
		if facets[i].Period != exp.week {
			t.Errorf("Facet %d: period = %d, expected %d", i, facets[i].Period, exp.week)
		}
		if facets[i].Value != exp.value {
			t.Errorf("Facet %d: value = %q, expected %q", i, facets[i].Value, exp.value)
		}
	}
}

// TestGeneratePeriodFacets_Week_53WeekYear tests facets crossing into a year with 53 ISO weeks
func TestGeneratePeriodFacets_Week_53WeekYear(t *testing.T) {
	// 2020 has 53 ISO weeks. Anchor at W02 2021 should cross back into W53 of 2020.
	facets := GeneratePeriodFacets("week", 4, 2021, 2)

	if len(facets) != 4 {
		t.Fatalf("Expected 4 facets, got %d", len(facets))
	}

	expected := []struct {
		label string
		year  int
		week  int
		value string
	}{
		{"W52 2020", 2020, 52, "2020W52"},
		{"W53 2020", 2020, 53, "2020W53"},
		{"W01 2021", 2021, 1, "2021W01"},
		{"W02 2021", 2021, 2, "2021W02"},
	}

	for i, exp := range expected {
		if facets[i].Label != exp.label {
			t.Errorf("Facet %d: label = %q, expected %q", i, facets[i].Label, exp.label)
		}
		if facets[i].Year != exp.year {
			t.Errorf("Facet %d: year = %d, expected %d", i, facets[i].Year, exp.year)
		}
		if facets[i].Period != exp.week {
			t.Errorf("Facet %d: period = %d, expected %d", i, facets[i].Period, exp.week)
		}
		if facets[i].Value != exp.value {
			t.Errorf("Facet %d: value = %q, expected %q", i, facets[i].Value, exp.value)
		}
	}
}

// TestGeneratePeriodFacets_Week_SingleFacet tests single week facet
func TestGeneratePeriodFacets_Week_SingleFacet(t *testing.T) {
	facets := GeneratePeriodFacets("week", 1, 2024, 15)

	if len(facets) != 1 {
		t.Fatalf("Expected 1 facet, got %d", len(facets))
	}

	if facets[0].Label != "W15 2024" {
		t.Errorf("Label = %q, expected %q", facets[0].Label, "W15 2024")
	}
	if facets[0].Value != "2024W15" {
		t.Errorf("Value = %q, expected %q", facets[0].Value, "2024W15")
	}
	if facets[0].Year != 2024 {
		t.Errorf("Year = %d, expected 2024", facets[0].Year)
	}
	if facets[0].Period != 15 {
		t.Errorf("Period = %d, expected 15", facets[0].Period)
	}
}

// TestGeneratePeriodFacets_CountLimits tests count limiting
func TestGeneratePeriodFacets_CountLimits(t *testing.T) {
	// Test count > 4 is capped
	facets := GeneratePeriodFacets("month", 10, 2024, 6)
	if len(facets) != 4 {
		t.Errorf("Expected count capped at 4, got %d", len(facets))
	}

	// Test count < 1 becomes 1
	facets = GeneratePeriodFacets("month", 0, 2024, 6)
	if len(facets) != 1 {
		t.Errorf("Expected count min of 1, got %d", len(facets))
	}

	facets = GeneratePeriodFacets("month", -5, 2024, 6)
	if len(facets) != 1 {
		t.Errorf("Expected count min of 1 for negative, got %d", len(facets))
	}
}

// TestExpandPeriodFilters_NilQuery tests expandPeriodFilters with nil query
func TestExpandPeriodFilters_NilQuery(t *testing.T) {
	filters := ReportFilters{Year: "2024", Month: "6"}
	result := expandPeriodFilters(context.Background(), nil, filters, nil)

	if result.Year != "2024" || result.Month != "6" {
		t.Errorf("Expected unchanged filters for nil query")
	}
}

// TestExpandPeriodFilters_ZeroPeriodLimit tests expandPeriodFilters with zero periodLimit
func TestExpandPeriodFilters_ZeroPeriodLimit(t *testing.T) {
	query := &Query{PeriodLimit: 0}
	filters := ReportFilters{Year: "2024", Month: "6"}
	result := expandPeriodFilters(context.Background(), query, filters, nil)

	if result.Year != "2024" || result.Month != "6" {
		t.Errorf("Expected unchanged filters for zero periodLimit")
	}
}

// TestExpandPeriodFilters_NoPeriodType tests expandPeriodFilters with no period groupBy
func TestExpandPeriodFilters_NoPeriodType(t *testing.T) {
	query := &Query{
		PeriodLimit: 6,
		GroupBy:     []GroupBy{{Field: "district"}},
	}
	filters := ReportFilters{Year: "2024", Month: "6"}
	result := expandPeriodFilters(context.Background(), query, filters, nil)

	if result.Year != "2024" || result.Month != "6" {
		t.Errorf("Expected unchanged filters when no period groupBy")
	}
}

// TestExpandPeriodFilters_MonthType tests month period expansion
func TestExpandPeriodFilters_MonthType(t *testing.T) {
	query := &Query{
		PeriodLimit: 6,
		GroupBy:     []GroupBy{{Field: "period_date", Format: "month"}},
	}
	filters := ReportFilters{Year: "2024", Month: "6"}
	result := expandPeriodFilters(context.Background(), query, filters, nil)

	// Should expand to 6 months ending in June 2024
	if result.Year != "2024" {
		t.Errorf("Expected year 2024, got %s", result.Year)
	}
	if result.Month != "1,2,3,4,5,6" {
		t.Errorf("Expected months 1,2,3,4,5,6, got %s", result.Month)
	}
	if result.YearMonths == nil {
		t.Errorf("Expected YearMonths to be populated")
	}
}

// TestExpandPeriodFilters_QuarterType tests quarter period expansion
func TestExpandPeriodFilters_QuarterType(t *testing.T) {
	query := &Query{
		PeriodLimit: 4,
		GroupBy:     []GroupBy{{Field: "quarter", Format: "quarter_year"}},
	}
	filters := ReportFilters{Year: "2024", Quarter: "2"}
	result := expandPeriodFilters(context.Background(), query, filters, nil)

	// Should expand to 4 quarters ending in Q2 2024
	if !strings.Contains(result.Year, "2023") || !strings.Contains(result.Year, "2024") {
		t.Errorf("Expected years to span 2023-2024, got %s", result.Year)
	}
	if result.YearQuarters == nil {
		t.Errorf("Expected YearQuarters to be populated")
	}
}

// TestExpandPeriodFilters_WeekType tests week period expansion
func TestExpandPeriodFilters_WeekType(t *testing.T) {
	query := &Query{
		PeriodLimit: 4,
		GroupBy:     []GroupBy{{Field: "period", Format: "week"}},
	}
	filters := ReportFilters{Week: "2024W10"}
	result := expandPeriodFilters(context.Background(), query, filters, nil)

	// Should expand to 4 weeks ending in week 10
	if result.Week != "2024W07,2024W08,2024W09,2024W10" {
		t.Errorf("Expected weeks 2024W07-2024W10, got %s", result.Week)
	}
}

// TestExpandMonthFilters_MultipleYears tests anchor selection from multiple years
func TestExpandMonthFilters_MultipleYears(t *testing.T) {
	filters := ReportFilters{Year: "2023,2024", Month: "3,6"}
	result := expandMonthFilters(context.Background(), 6, filters, nil, "")

	// Should use 2024 (max year) and 6 (max month) as anchor
	if !strings.Contains(result.Year, "2024") {
		t.Errorf("Expected year to contain 2024, got %s", result.Year)
	}
}

// TestExpandWeekFilters_MultipleWeeks tests anchor selection from multiple weeks
func TestExpandWeekFilters_MultipleWeeks(t *testing.T) {
	filters := ReportFilters{Week: "2024W05,2024W10,2024W08"}
	result := expandWeekFilters(context.Background(), 4, filters, nil)

	// Should use 2024W10 (max week) as anchor
	weeks := strings.Split(result.Week, ",")
	lastWeek := weeks[len(weeks)-1]
	if lastWeek != "2024W10" {
		t.Errorf("Expected last week to be 2024W10, got %s", lastWeek)
	}
}

// TestExpandQuarterFilters_MultipleQuarters tests anchor selection from multiple quarters
func TestExpandQuarterFilters_MultipleQuarters(t *testing.T) {
	filters := ReportFilters{Year: "2024", Quarter: "1,3,2"}
	result := expandQuarterFilters(context.Background(), 4, filters, nil)

	// Should use Q3 (max quarter) as anchor
	quarters := strings.Split(result.Quarter, ",")
	lastQuarter := quarters[len(quarters)-1]
	if lastQuarter != "3" {
		t.Errorf("Expected last quarter to be 3, got %s", lastQuarter)
	}
}

// TestExpandYearRange tests year expansion
func TestExpandYearRange(t *testing.T) {
	tests := []struct {
		name     string
		year     int
		count    int
		expected string
	}{
		{
			name:     "5 years from 2024",
			year:     2024,
			count:    5,
			expected: "2020,2021,2022,2023,2024",
		},
		{
			name:     "3 years from 2024",
			year:     2024,
			count:    3,
			expected: "2022,2023,2024",
		},
		{
			name:     "1 year returns single year",
			year:     2024,
			count:    1,
			expected: "2024",
		},
		{
			name:     "zero count returns single year",
			year:     2024,
			count:    0,
			expected: "2024",
		},
		{
			name:     "negative count returns single year",
			year:     2024,
			count:    -5,
			expected: "2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandYearRange(tt.year, tt.count)
			if result != tt.expected {
				t.Errorf("ExpandYearRange(%d, %d) = %q, expected %q", tt.year, tt.count, result, tt.expected)
			}
		})
	}
}

// TestExpandPeriodFilters_YearType tests year period expansion
func TestExpandPeriodFilters_YearType(t *testing.T) {
	query := &Query{
		PeriodLimit: 5,
		GroupBy:     []GroupBy{{Field: "period_date", Format: "year"}},
	}
	filters := ReportFilters{Year: "2024"}
	result := expandPeriodFilters(context.Background(), query, filters, nil)

	if result.Year != "2020,2021,2022,2023,2024" {
		t.Errorf("Expected years 2020,2021,2022,2023,2024, got %s", result.Year)
	}
}

// TestExpandYearFilters_MultipleYears tests anchor selection from multiple years
func TestExpandYearFilters_MultipleYears(t *testing.T) {
	filters := ReportFilters{Year: "2022,2024,2023"}
	result := expandYearFilters(context.Background(), 3, filters, nil, "")

	// Should use 2024 (max year) as anchor
	years := strings.Split(result.Year, ",")
	lastYear := years[len(years)-1]
	if lastYear != "2024" {
		t.Errorf("Expected last year to be 2024, got %s", lastYear)
	}
	if result.Year != "2022,2023,2024" {
		t.Errorf("Expected years 2022,2023,2024, got %s", result.Year)
	}
}

// TestExpandQuarterRange_ChronologicalOrder verifies quarters are in chronological order
func TestExpandQuarterRange_ChronologicalOrder(t *testing.T) {
	_, quarters := ExpandQuarterRange(2024, 2, 6)

	quarterList := strings.Split(quarters, ",")
	if len(quarterList) != 6 {
		t.Errorf("Expected 6 quarters, got %d", len(quarterList))
	}

	// First should be Q1 2023, last should be Q2 2024
	if quarterList[0] != "1" {
		t.Errorf("First quarter should be 1, got %s", quarterList[0])
	}
	if quarterList[5] != "2" {
		t.Errorf("Last quarter should be 2, got %s", quarterList[5])
	}
}
