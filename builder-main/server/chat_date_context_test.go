package main

import (
	"strings"
	"testing"
	"time"
)

func TestChatDateContext(t *testing.T) {
	cases := []struct {
		name     string
		input    time.Time
		contains []string
	}{
		{
			name:  "mid-year resolves last month within same year",
			input: time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC),
			contains: []string{
				"Today: 2026-04-28",
				`"this month" / "current month" -> filters.year=2026, filters.month=4`,
				`"last month" / "previous month" -> filters.year=2026, filters.month=3`,
				`"this quarter" -> filters.year=2026, filters.quarter=2`,
				`"last quarter" / "previous quarter" -> filters.year=2026, filters.quarter=1`,
				`"this year" / "current year" -> filters.year=2026`,
				`"last year" / "previous year" -> filters.year=2025`,
			},
		},
		{
			name:  "January carries last month into previous year",
			input: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
			contains: []string{
				`"last month" / "previous month" -> filters.year=2025, filters.month=12`,
				`"last quarter" / "previous quarter" -> filters.year=2025, filters.quarter=4`,
			},
		},
		{
			name:  "April lands in Q2 with last quarter Q1",
			input: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
			contains: []string{
				`"this quarter" -> filters.year=2026, filters.quarter=2`,
				`"last quarter" / "previous quarter" -> filters.year=2026, filters.quarter=1`,
			},
		},
		{
			name:  "October lands in Q4 with last quarter Q3",
			input: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
			contains: []string{
				`"this quarter" -> filters.year=2026, filters.quarter=4`,
				`"last quarter" / "previous quarter" -> filters.year=2026, filters.quarter=3`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := chatDateContext(tc.input)
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Errorf("expected output to contain %q\n--- got ---\n%s", want, got)
				}
			}
		})
	}
}

func TestDescribePeriodFilter(t *testing.T) {
	cases := []struct {
		name   string
		filter ChatFilters
		want   string
	}{
		{"year and month", ChatFilters{Year: "2026", Month: "3"}, "March 2026"},
		{"year and month zero-padded", ChatFilters{Year: "2026", Month: "03"}, "March 2026"},
		{"year and quarter", ChatFilters{Year: "2026", Quarter: "1"}, "Q1 2026"},
		{"week", ChatFilters{Week: "2026W17"}, "ISO week 2026W17"},
		{"year only", ChatFilters{Year: "2026"}, "2026 (full year)"},
		{"month without year", ChatFilters{Month: "3"}, "March (year unspecified — across all years in the table)"},
		{"quarter without year", ChatFilters{Quarter: "2"}, "Q2 (year unspecified — across all years in the table)"},
		{"empty filter falls back to default", ChatFilters{}, "all available data (no time filter — results span every period in the table)"},
		{"district-only is still all-time", ChatFilters{District: "Kampala District"}, "all available data (no time filter — results span every period in the table)"},
		{"invalid month string passes through", ChatFilters{Year: "2026", Month: "foo"}, "month=foo 2026"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := describePeriodFilter(tc.filter); got != tc.want {
				t.Errorf("describePeriodFilter(%+v) = %q, want %q", tc.filter, got, tc.want)
			}
		})
	}
}

func TestChatDateContextISOWeekEdge(t *testing.T) {
	// 2027-01-01 is a Friday — ISO week 53 of 2026.
	// Last week (a week earlier, 2026-12-25) is also in 2026, week 52.
	got := chatDateContext(time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC))
	wantThis := `"this week" -> filters.week=2026W53`
	wantLast := `"last week" / "previous week" -> filters.week=2026W52`
	if !strings.Contains(got, wantThis) {
		t.Errorf("expected %q\n--- got ---\n%s", wantThis, got)
	}
	if !strings.Contains(got, wantLast) {
		t.Errorf("expected %q\n--- got ---\n%s", wantLast, got)
	}
}
