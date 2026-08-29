package main

import (
	"strings"
	"testing"
)

func TestValidateChatFilters(t *testing.T) {
	cases := []struct {
		name      string
		filters   ChatFilters
		wantError bool
		// substring of error message; empty means don't check
		wantSubstr string
	}{
		{name: "empty filters are valid", filters: ChatFilters{}, wantError: false},
		{name: "single year ok", filters: ChatFilters{Year: "2024"}, wantError: false},
		{name: "year with whitespace ok", filters: ChatFilters{Year: " 2024 "}, wantError: false},
		{name: "comma-separated year rejected", filters: ChatFilters{Year: "2025,2026"}, wantError: true, wantSubstr: "year"},
		{name: "non-numeric year rejected", filters: ChatFilters{Year: "twenty"}, wantError: true, wantSubstr: "year"},
		{name: "year out of range rejected", filters: ChatFilters{Year: "1999"}, wantError: true, wantSubstr: "year"},
		{name: "month 1-12 ok", filters: ChatFilters{Month: "12"}, wantError: false},
		{name: "month 0 rejected", filters: ChatFilters{Month: "0"}, wantError: true, wantSubstr: "month"},
		{name: "month 13 rejected", filters: ChatFilters{Month: "13"}, wantError: true, wantSubstr: "month"},
		{name: "comma month rejected", filters: ChatFilters{Month: "3,4"}, wantError: true, wantSubstr: "month"},
		{name: "quarter 1-4 ok", filters: ChatFilters{Quarter: "4"}, wantError: false},
		{name: "quarter 5 rejected", filters: ChatFilters{Quarter: "5"}, wantError: true, wantSubstr: "quarter"},
		{name: "ISO week ok", filters: ChatFilters{Week: "2024W15"}, wantError: false},
		{name: "ISO week with whitespace ok", filters: ChatFilters{Week: " 2024W08 "}, wantError: false},
		{name: "week without W rejected", filters: ChatFilters{Week: "202415"}, wantError: true, wantSubstr: "week"},
		{name: "comma week rejected", filters: ChatFilters{Week: "2024W15,2024W16"}, wantError: true, wantSubstr: "week"},
		{name: "single district ok", filters: ChatFilters{District: "Kamuli District"}, wantError: false},
		{name: "comma district rejected", filters: ChatFilters{District: "Kampala District,Central District"}, wantError: true, wantSubstr: "district"},
		{name: "valid combination ok", filters: ChatFilters{District: "Kampala District", Year: "2024", Month: "3"}, wantError: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateChatFilters(tc.filters)
			if tc.wantError && got == "" {
				t.Fatalf("expected error, got none")
			}
			if !tc.wantError && got != "" {
				t.Fatalf("expected no error, got %q", got)
			}
			if tc.wantSubstr != "" && !strings.Contains(got, tc.wantSubstr) {
				t.Fatalf("expected error to mention %q; got %q", tc.wantSubstr, got)
			}
		})
	}
}
