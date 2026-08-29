package main

import (
	"reflect"
	"testing"
)

func TestChatFiltersToReportFilters(t *testing.T) {
	tests := []struct {
		name string
		in   ChatFilters
		want ReportFilters
	}{
		{
			name: "all fields with district",
			in:   ChatFilters{District: "Gomba District", Year: "2024", Month: "3", Quarter: "1", Week: "2024W15"},
			want: ReportFilters{Districts: []string{"Gomba District"}, Year: "2024", Month: "3", Quarter: "1", Week: "2024W15"},
		},
		{
			name: "empty district yields nil slice (means All)",
			in:   ChatFilters{Year: "2024"},
			want: ReportFilters{Year: "2024"},
		},
		{
			name: "whitespace is trimmed",
			in:   ChatFilters{District: "  Kampala District ", Year: " 2025 ", Month: " 6 "},
			want: ReportFilters{Districts: []string{"Kampala District"}, Year: "2025", Month: "6"},
		},
		{
			name: "all empty",
			in:   ChatFilters{},
			want: ReportFilters{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatFiltersToReportFilters(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("chatFiltersToReportFilters(%+v) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}

func TestDescribeFilterSupport(t *testing.T) {
	tests := []struct {
		name string
		in   chatFilterSupport
		want string
	}{
		{
			name: "all supported",
			in:   chatFilterSupport{Year: true, Month: true, Quarter: true, Week: true, District: true},
			want: "Filterable by: year, month, quarter, week, district",
		},
		{
			name: "quarter-only table omits month and week",
			in:   chatFilterSupport{Year: true, Quarter: true, District: true},
			want: "Filterable by: year, quarter, district — do NOT set: month, week",
		},
		{
			name: "no filters at all",
			in:   chatFilterSupport{},
			want: "Filterable by: (none — no period or district filter on this table; it returns all rows)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := describeFilterSupport(tt.in); got != tt.want {
				t.Errorf("describeFilterSupport(%+v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeChatFilters(t *testing.T) {
	// Save and restore the package-global support map.
	saved := chatTableFilterSupport
	defer func() { chatTableFilterSupport = saved }()

	chatTableFilterSupport = map[string]chatFilterSupport{
		"schema.month_table":   {Year: true, Month: true, District: true},
		"schema.quarter_table": {Year: true, Quarter: true, District: true},
		"schema.no_filters":    {},
	}

	tests := []struct {
		name        string
		table       string
		in          ChatFilters
		wantFilters ChatFilters
		wantDropped []string
	}{
		{
			name:        "month value dropped on quarter-only table",
			table:       "schema.quarter_table",
			in:          ChatFilters{Year: "2024", Month: "3", District: "Gomba District"},
			wantFilters: ChatFilters{Year: "2024", District: "Gomba District"},
			wantDropped: []string{"month"},
		},
		{
			name:        "supported filters pass through unchanged",
			table:       "schema.month_table",
			in:          ChatFilters{Year: "2024", Month: "3", District: "Gomba District"},
			wantFilters: ChatFilters{Year: "2024", Month: "3", District: "Gomba District"},
			wantDropped: nil,
		},
		{
			name:        "all period/location filters dropped on no-filter table",
			table:       "schema.no_filters",
			in:          ChatFilters{Year: "2024", Quarter: "2", District: "Gomba District"},
			wantFilters: ChatFilters{},
			wantDropped: []string{"year", "quarter", "district"},
		},
		{
			name:        "unknown table left untouched",
			table:       "schema.not_in_catalog",
			in:          ChatFilters{Year: "2024", Month: "3"},
			wantFilters: ChatFilters{Year: "2024", Month: "3"},
			wantDropped: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFilters, gotDropped := sanitizeChatFilters(tt.table, tt.in)
			if !reflect.DeepEqual(gotFilters, tt.wantFilters) {
				t.Errorf("filters = %+v, want %+v", gotFilters, tt.wantFilters)
			}
			if !reflect.DeepEqual(gotDropped, tt.wantDropped) {
				t.Errorf("dropped = %v, want %v", gotDropped, tt.wantDropped)
			}
		})
	}
}
