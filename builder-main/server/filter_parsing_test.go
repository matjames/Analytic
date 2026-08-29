package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// =============================================================================
// Success criteria for filter parsing tests:
//
// 1. FILTER EXTRACTION: Query params produce the correct ReportFilters struct
//    - Single values: ?year=2024 → Year="2024"
//    - Multi-value params: ?month=1&month=2 → Month="1,2"
//    - Comma-separated: ?district=A,B → Districts=["A","B"]
//    - Mixed: ?district=A&district=B,C → Districts=["A","B","C"]
//    - Empty/missing params → zero values (empty string, nil slice)
//
// 2. FILTER LIMITS: Excessive params return 400 before hitting DB/config
//    - >12 month params → 400
//    - >200 district values (after comma expansion) → 400
//    - >50 region values → 400
//    - >500 facility values → 400
//    - At-limit values → allowed (not 400)
//
// 3. PRESENCE TRACKING: *Provided booleans reflect whether param was in URL
//    - ?month=1 → MonthProvided=true
//    - (no month param) → MonthProvided=false
//    - ?week= (empty value) → WeekProvided=true (param present, value empty)
//
// 4. CUSTOM FILTERS: Values extracted for columns defined in report YAML
//    - Report defines customFilter column="facility_type"
//    - ?facility_type=Hospital → CustomValues["facility_type"]="Hospital"
//
// 5. HANDLER INTEGRATION: The handler returns correct HTTP status codes
//    - Excessive filters → 400 with error JSON
//    - Nonexistent report → 404
//    - Invalid report ID → 400
// =============================================================================

// --- parseReportFilters unit tests ---

func makeRequest(queryString string) *http.Request {
	return httptest.NewRequest("GET", "/api/report/test?"+queryString, nil)
}

func TestParseReportFilters_SingleValues(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		checkFn func(t *testing.T, f ReportFilters)
	}{
		{
			name:  "year extracted",
			query: "year=2024",
			checkFn: func(t *testing.T, f ReportFilters) {
				if f.Year != "2024" {
					t.Errorf("Year = %q, want %q", f.Year, "2024")
				}
			},
		},
		{
			name:  "single month",
			query: "month=6",
			checkFn: func(t *testing.T, f ReportFilters) {
				if f.Month != "6" {
					t.Errorf("Month = %q, want %q", f.Month, "6")
				}
			},
		},
		{
			name:  "week in ISO format",
			query: "week=2024W15",
			checkFn: func(t *testing.T, f ReportFilters) {
				if f.Week != "2024W15" {
					t.Errorf("Week = %q, want %q", f.Week, "2024W15")
				}
			},
		},
		{
			name:  "quarter",
			query: "quarter=3",
			checkFn: func(t *testing.T, f ReportFilters) {
				if f.Quarter != "3" {
					t.Errorf("Quarter = %q, want %q", f.Quarter, "3")
				}
			},
		},
		{
			name:  "no params yields empty filters",
			query: "",
			checkFn: func(t *testing.T, f ReportFilters) {
				if f.Year != "" || f.Month != "" || f.Week != "" || f.Quarter != "" {
					t.Errorf("Expected empty time filters, got year=%q month=%q week=%q quarter=%q",
						f.Year, f.Month, f.Week, f.Quarter)
				}
				if len(f.Districts) != 0 || len(f.Regions) != 0 || len(f.Facilities) != 0 {
					t.Errorf("Expected empty location filters, got districts=%v regions=%v facilities=%v",
						f.Districts, f.Regions, f.Facilities)
				}
			},
		},
		{
			name:  "all time filters together",
			query: "year=2024&month=6&week=2024W15&quarter=2",
			checkFn: func(t *testing.T, f ReportFilters) {
				if f.Year != "2024" {
					t.Errorf("Year = %q, want %q", f.Year, "2024")
				}
				if f.Month != "6" {
					t.Errorf("Month = %q, want %q", f.Month, "6")
				}
				if f.Week != "2024W15" {
					t.Errorf("Week = %q, want %q", f.Week, "2024W15")
				}
				if f.Quarter != "2" {
					t.Errorf("Quarter = %q, want %q", f.Quarter, "2")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeRequest(tt.query)
			filters, err := parseReportFilters(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.checkFn(t, filters)
		})
	}
}

func TestParseReportFilters_IgnoresSentinelValues(t *testing.T) {
	r := makeRequest("year=2025&quarter=1&month=None&region=None&district=All&week=null")
	filters, err := parseReportFilters(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filters.Year != "2025" {
		t.Errorf("Year = %q, want %q", filters.Year, "2025")
	}
	if filters.Quarter != "1" {
		t.Errorf("Quarter = %q, want %q", filters.Quarter, "1")
	}
	if filters.Month != "" {
		t.Errorf("Month = %q, want empty", filters.Month)
	}
	if filters.Week != "" {
		t.Errorf("Week = %q, want empty", filters.Week)
	}
	if len(filters.Regions) != 0 {
		t.Errorf("Regions = %v, want empty", filters.Regions)
	}
	if len(filters.Districts) != 0 {
		t.Errorf("Districts = %v, want empty", filters.Districts)
	}

	if !filters.MonthProvided {
		t.Error("MonthProvided = false, want true when month query param exists")
	}
	if !filters.QuarterProvided {
		t.Error("QuarterProvided = false, want true when quarter query param exists")
	}
}

func TestParseReportFilters_YearSentinelNotExplicitlyProvided(t *testing.T) {
	r := makeRequest("year=None&quarter=1")
	filters, err := parseReportFilters(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filters.Year != "" {
		t.Errorf("Year = %q, want empty", filters.Year)
	}
	if filters.YearProvided {
		t.Error("YearProvided = true, want false when year is sentinel None")
	}
	if filters.Quarter != "1" {
		t.Errorf("Quarter = %q, want %q", filters.Quarter, "1")
	}
	if !filters.QuarterProvided {
		t.Error("QuarterProvided = false, want true when quarter query param exists")
	}
}

func TestParseReportFilters_YearExplicitWithoutNarrowerTimeFilters(t *testing.T) {
	r := makeRequest("year=2025")
	filters, err := parseReportFilters(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filters.Year != "2025" {
		t.Errorf("Year = %q, want %q", filters.Year, "2025")
	}
	if !filters.YearProvided {
		t.Error("YearProvided = false, want true when year query param exists")
	}
	if filters.Month != "" || filters.Quarter != "" || filters.Week != "" {
		t.Errorf("Expected no narrower time filters, got month=%q quarter=%q week=%q", filters.Month, filters.Quarter, filters.Week)
	}
	if filters.MonthProvided || filters.QuarterProvided || filters.WeekProvided {
		t.Errorf("Expected Month/Quarter/WeekProvided false, got month=%v quarter=%v week=%v", filters.MonthProvided, filters.QuarterProvided, filters.WeekProvided)
	}
}

func TestParseReportFilters_NoTimeDefaultsFlag(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "flag true literal", query: "no_time_defaults=true", want: true},
		{name: "flag one", query: "no_time_defaults=1", want: true},
		{name: "flag false", query: "no_time_defaults=false", want: false},
		{name: "flag missing", query: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeRequest(tt.query)
			filters, err := parseReportFilters(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if filters.SuppressTimeDefaults != tt.want {
				t.Errorf("SuppressTimeDefaults = %v, want %v", filters.SuppressTimeDefaults, tt.want)
			}
		})
	}
}

func TestParseReportFilters_MultiValueMonths(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantMonth string
	}{
		{
			name:      "two months via repeated params",
			query:     "month=1&month=2",
			wantMonth: "1,2",
		},
		{
			name:      "three months via repeated params",
			query:     "month=10&month=11&month=12",
			wantMonth: "10,11,12",
		},
		{
			name:      "single month stays as-is",
			query:     "month=6",
			wantMonth: "6",
		},
		{
			name:      "comma-separated months in single param",
			query:     "month=1,6,7",
			wantMonth: "1,6,7",
		},
		{
			name:      "mixed: repeated param + comma-separated",
			query:     "month=1&month=6,7",
			wantMonth: "1,6,7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeRequest(tt.query)
			filters, err := parseReportFilters(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if filters.Month != tt.wantMonth {
				t.Errorf("Month = %q, want %q", filters.Month, tt.wantMonth)
			}
		})
	}
}

func TestParseReportFilters_LocationFilters(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		wantDistricts  []string
		wantRegions    []string
		wantFacilities []string
	}{
		{
			name:          "single district",
			query:         "district=Kampala",
			wantDistricts: []string{"Kampala"},
		},
		{
			name:          "multiple districts via repeated params",
			query:         "district=Kampala&district=Central",
			wantDistricts: []string{"Kampala", "Central"},
		},
		{
			name:          "comma-separated districts",
			query:         "district=Kampala,Central,Jinja",
			wantDistricts: []string{"Kampala", "Central", "Jinja"},
		},
		{
			name:          "mixed: repeated + comma-separated",
			query:         "district=Kampala&district=Central,Jinja",
			wantDistricts: []string{"Kampala", "Central", "Jinja"},
		},
		{
			name:        "regions via comma-separated",
			query:       "region=Central,Western",
			wantRegions: []string{"Central", "Western"},
		},
		{
			name:           "facilities via repeated params",
			query:          "facility=HC+III+Bukoto&facility=HC+IV+Mengo",
			wantFacilities: []string{"HC III Bukoto", "HC IV Mengo"},
		},
		{
			name:           "empty district param ignored",
			query:          "district=",
			wantDistricts:  nil,
			wantRegions:    nil,
			wantFacilities: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeRequest(tt.query)
			filters, err := parseReportFilters(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertStringSlice(t, "Districts", filters.Districts, tt.wantDistricts)
			assertStringSlice(t, "Regions", filters.Regions, tt.wantRegions)
			assertStringSlice(t, "Facilities", filters.Facilities, tt.wantFacilities)
		})
	}
}

func TestParseReportFilters_PresenceTracking(t *testing.T) {
	tests := []struct {
		name              string
		query             string
		wantMonthProvided bool
		wantWeekProvided  bool
		wantQtrProvided   bool
	}{
		{
			name:              "month provided",
			query:             "month=6",
			wantMonthProvided: true,
		},
		{
			name:             "week provided",
			query:            "week=2024W15",
			wantWeekProvided: true,
		},
		{
			name:            "quarter provided",
			query:           "quarter=2",
			wantQtrProvided: true,
		},
		{
			name:              "empty month param still counts as provided",
			query:             "month=",
			wantMonthProvided: true,
		},
		{
			name:              "no time filters — none provided",
			query:             "year=2024&district=Kampala",
			wantMonthProvided: false,
			wantWeekProvided:  false,
			wantQtrProvided:   false,
		},
		{
			name:              "all time filters provided",
			query:             "month=6&week=2024W15&quarter=2",
			wantMonthProvided: true,
			wantWeekProvided:  true,
			wantQtrProvided:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeRequest(tt.query)
			filters, err := parseReportFilters(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if filters.MonthProvided != tt.wantMonthProvided {
				t.Errorf("MonthProvided = %v, want %v", filters.MonthProvided, tt.wantMonthProvided)
			}
			if filters.WeekProvided != tt.wantWeekProvided {
				t.Errorf("WeekProvided = %v, want %v", filters.WeekProvided, tt.wantWeekProvided)
			}
			if filters.QuarterProvided != tt.wantQtrProvided {
				t.Errorf("QuarterProvided = %v, want %v", filters.QuarterProvided, tt.wantQtrProvided)
			}
		})
	}
}

func TestParseReportFilters_CustomValuesInitialized(t *testing.T) {
	r := makeRequest("year=2024")
	filters, err := parseReportFilters(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.CustomValues == nil {
		t.Fatal("CustomValues map should be initialized, got nil")
	}
	if len(filters.CustomValues) != 0 {
		t.Errorf("CustomValues should be empty, got %v", filters.CustomValues)
	}
}

// --- Filter limit enforcement ---

func TestParseReportFilters_Limits(t *testing.T) {
	tests := []struct {
		name      string
		buildURL  func() string
		wantError bool
	}{
		{
			name: "13 month params rejected",
			buildURL: func() string {
				params := url.Values{}
				for i := 1; i <= 13; i++ {
					params.Add("month", "1")
				}
				return params.Encode()
			},
			wantError: true,
		},
		{
			name: "13 months in single comma-separated param rejected",
			buildURL: func() string {
				months := make([]string, 13)
				for i := range months {
					months[i] = "1"
				}
				return "month=" + strings.Join(months, ",")
			},
			wantError: true,
		},
		{
			name: "12 month params allowed",
			buildURL: func() string {
				params := url.Values{}
				for i := 1; i <= 12; i++ {
					params.Add("month", "1")
				}
				return params.Encode()
			},
			wantError: false,
		},
		{
			name: "12 months in single comma-separated param allowed",
			buildURL: func() string {
				months := make([]string, 12)
				for i := range months {
					months[i] = "1"
				}
				return "month=" + strings.Join(months, ",")
			},
			wantError: false,
		},
		{
			name: "201 districts rejected (via comma expansion)",
			buildURL: func() string {
				districts := make([]string, 201)
				for i := range districts {
					districts[i] = "District"
				}
				return "district=" + strings.Join(districts, ",")
			},
			wantError: true,
		},
		{
			name: "200 districts allowed",
			buildURL: func() string {
				districts := make([]string, 200)
				for i := range districts {
					districts[i] = "District"
				}
				return "district=" + strings.Join(districts, ",")
			},
			wantError: false,
		},
		{
			name: "51 regions rejected",
			buildURL: func() string {
				regions := make([]string, 51)
				for i := range regions {
					regions[i] = "Region"
				}
				return "region=" + strings.Join(regions, ",")
			},
			wantError: true,
		},
		{
			name: "50 regions allowed",
			buildURL: func() string {
				regions := make([]string, 50)
				for i := range regions {
					regions[i] = "Region"
				}
				return "region=" + strings.Join(regions, ",")
			},
			wantError: false,
		},
		{
			name: "501 facilities rejected",
			buildURL: func() string {
				facilities := make([]string, 501)
				for i := range facilities {
					facilities[i] = "Facility"
				}
				return "facility=" + strings.Join(facilities, ",")
			},
			wantError: true,
		},
		{
			name: "500 facilities allowed",
			buildURL: func() string {
				facilities := make([]string, 500)
				for i := range facilities {
					facilities[i] = "Facility"
				}
				return "facility=" + strings.Join(facilities, ",")
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := makeRequest(tt.buildURL())
			_, err := parseReportFilters(r)
			if tt.wantError && err == nil {
				t.Error("expected error for excessive filters, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// --- Handler-level integration tests (HTTP status codes) ---

func TestGetReportHandler_FilterLimitsReturn400(t *testing.T) {
	// 13 months should be rejected at the filter parsing stage,
	// before any config file or DB access.
	params := url.Values{}
	for i := 1; i <= 13; i++ {
		params.Add("month", "1")
	}

	req := httptest.NewRequest("GET", "/api/report/any-report?"+params.Encode(), nil)
	w := httptest.NewRecorder()
	GetReportHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if errMsg, ok := resp["error"].(string); !ok || !strings.Contains(errMsg, "too many filter values") {
		t.Errorf("error message = %q, want it to contain 'too many filter values'", resp["error"])
	}
}

func TestGetReportHandler_NonexistentReportReturns404(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/report/does-not-exist-xyz?year=2024&month=6", nil)
	w := httptest.NewRecorder()
	GetReportHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetReportHandler_InvalidReportIDReturns400(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"path traversal", "/api/report/../../../etc/passwd"},
		{"special characters", "/api/report/report<script>"},
		{"too many slashes", "/api/report/a/b/c/d/e/f/g"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path+"?year=2024", nil)
			w := httptest.NewRecorder()
			GetReportHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for path %q", w.Code, http.StatusBadRequest, tt.path)
			}
		})
	}
}

// --- helpers ---

func assertStringSlice(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v (len %d), want %v (len %d)", name, got, len(got), want, len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}
