package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"gopkg.in/yaml.v3"
)

// pdfSemaphore limits concurrent PDF generations to 5
var pdfSemaphore = make(chan struct{}, 5)

// pdfPerIP tracks concurrent PDF generations per IP (max 2 per IP)
var pdfPerIP sync.Map // map[string]*atomic.Int32

// validTableFormat validates table names to prevent SQL injection.
// Compiled once at package init for performance.
var validTableFormat = regexp.MustCompile(`^[a-zA-Z0-9_.]+$`)

// allowedOrigin restricts CORS to a specific origin when set (e.g., "https://dashboard.example.com").
// When empty, wildcard "*" is used for backward compatibility.
var allowedOrigin = os.Getenv("ALLOWED_ORIGIN")

// setCORSHeaders sets standard CORS headers for JSON API responses.
// When ALLOWED_ORIGIN is set, restricts to that origin and enables credentials.
func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	origin := "*"
	if allowedOrigin != "" {
		origin = allowedOrigin
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// splitAndTrimCSV splits a comma-separated string, trims whitespace, and removes empty entries.
func splitAndTrimCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// loadReportWithDefault loads a report for the given table, returning an empty Report if not found.
func loadReportWithDefault(table string) *Report {
	report := LoadReportForTable(table)
	if report == nil {
		logDebug("No report found for table %s, using defaults", table)
		return &Report{}
	}
	return report
}

func filterReportsByPermissions(all ReportsList, perms *UserPermissions) ReportsList {
	if perms == nil || len(perms.Paths) == 0 {
		return all
	}

	filtered := ReportsList{Reports: []ReportListItem{}, Sections: []string{}}
	visibleSections := make(map[string]bool)

	for _, report := range all.Reports {
		if !UserCanAccessReport(report.ID, perms) {
			continue
		}
		filtered.Reports = append(filtered.Reports, report)

		category := strings.TrimSpace(report.Category)
		for category != "" {
			visibleSections[category] = true
			lastSlash := strings.LastIndex(category, "/")
			if lastSlash < 0 {
				break
			}
			category = category[:lastSlash]
		}
	}

	for _, section := range all.Sections {
		if visibleSections[section] {
			filtered.Sections = append(filtered.Sections, section)
		}
	}

	return filtered
}

// ListReportsHandler returns the list of available reports
func ListReportsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	reports := GetAllReports()
	perms := GetEffectivePermissionsFromRequest(r)
	reports = filterReportsByPermissions(reports, perms)
	json.NewEncoder(w).Encode(reports)
}

// loadReportMetadataWithHash loads YAML metadata without executing queries and returns
// the stable YAML hash used for cache invalidation (same rules as ComputeYAMLHash).
func loadReportMetadataWithHash(reportID string) (*Report, string, error) {
	sanitizedPath := strings.ReplaceAll(reportID, "/", string(filepath.Separator))
	filePath := filepath.Join("configs", sanitizedPath+".yaml")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}

	yamlHash := ComputeYAMLHash(data)

	var report Report
	if err := yaml.Unmarshal(data, &report); err != nil {
		return nil, "", err
	}

	for i := range report.Sections {
		if report.Sections[i].Components == nil {
			report.Sections[i].Components = []Component{}
		}
	}

	return &report, yamlHash, nil
}

// getDefaultMonth returns the default month (2 months before current date)
// Returns (year, month) to handle year boundary crossing
func getDefaultMonth() (string, string) {
	now := time.Now()
	// Go back 2 months
	target := now.AddDate(0, -2, 0)
	return fmt.Sprintf("%d", target.Year()), fmt.Sprintf("%d", int(target.Month()))
}

// getDefaultWeek returns the default week (2 weeks before current date)
// Returns week in YYYYWNN format
func getDefaultWeek() string {
	now := time.Now()
	// Go back 2 weeks
	target := now.AddDate(0, 0, -14)
	year, week := target.ISOWeek()
	return fmt.Sprintf("%dW%02d", year, week)
}

// getDefaultQuarter returns the default quarter (2 quarters before current date)
// Returns (year, quarter) to handle year boundary crossing
func getDefaultQuarter() (string, string) {
	now := time.Now()
	// Current quarter: Q1=Jan-Mar, Q2=Apr-Jun, Q3=Jul-Sep, Q4=Oct-Dec
	currentQuarter := (int(now.Month())-1)/3 + 1
	currentYear := now.Year()

	// Go back 2 quarters
	targetQuarter := currentQuarter - 2
	targetYear := currentYear
	if targetQuarter < 1 {
		targetQuarter += 4
		targetYear--
	}

	return fmt.Sprintf("%d", targetYear), fmt.Sprintf("%d", targetQuarter)
}

// filterSliceContains checks if a string slice contains a value
func filterSliceContains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// expandCommaValues expands comma-separated values in a slice
// Handles both multiple params (?district=A&district=B) and comma-separated (?district=A,B)
func expandCommaValues(values []string) []string {
	var result []string
	for _, v := range values {
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			for _, p := range parts {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					result = append(result, trimmed)
				}
			}
		} else if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func normalizeOptionalFilterValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}

	switch strings.ToLower(v) {
	case "none", "all", "null", "undefined":
		return ""
	default:
		return v
	}
}

func normalizeFilterValues(values []string) []string {
	if len(values) == 0 {
		return values
	}

	out := make([]string, 0, len(values))
	for _, v := range values {
		normalized := normalizeOptionalFilterValue(v)
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

// parseReportFilters extracts and validates filter parameters from the HTTP request.
// Returns an error if filter limits are exceeded.
func parseReportFilters(r *http.Request) (ReportFilters, error) {
	monthValues := normalizeFilterValues(expandCommaValues(r.URL.Query()["month"]))
	monthFilter := strings.Join(monthValues, ",")

	districtValues := normalizeFilterValues(expandCommaValues(r.URL.Query()["district"]))
	regionValues := normalizeFilterValues(expandCommaValues(r.URL.Query()["region"]))
	facilityValues := normalizeFilterValues(expandCommaValues(r.URL.Query()["facility"]))

	if len(monthValues) > 12 || len(districtValues) > 200 || len(regionValues) > 50 || len(facilityValues) > 500 {
		return ReportFilters{}, fmt.Errorf("too many filter values")
	}

	_, weekProvided := r.URL.Query()["week"]
	_, monthProvided := r.URL.Query()["month"]
	_, quarterProvided := r.URL.Query()["quarter"]
	_, yearParamProvided := r.URL.Query()["year"]
	normalizedYear := normalizeOptionalFilterValue(r.URL.Query().Get("year"))
	yearProvided := yearParamProvided && normalizedYear != ""
	noTimeDefaultsParam := strings.TrimSpace(r.URL.Query().Get("no_time_defaults"))
	suppressTimeDefaults := noTimeDefaultsParam == "1" || strings.EqualFold(noTimeDefaultsParam, "true")

	// Parse yearmonths param for fiscal year support
	// Format: "2024:7,8,9,10,11,12|2025:1,2,3,4,5,6"
	var yearMonths map[string][]int
	if ymParam := r.URL.Query().Get("yearmonths"); ymParam != "" {
		yearMonths = make(map[string][]int)
		for _, yearGroup := range strings.Split(ymParam, "|") {
			parts := strings.SplitN(yearGroup, ":", 2)
			if len(parts) != 2 {
				continue
			}
			year := strings.TrimSpace(parts[0])
			if !isValidYear(year) {
				continue
			}
			var months []int
			for _, m := range strings.Split(parts[1], ",") {
				month, err := strconv.Atoi(strings.TrimSpace(m))
				if err != nil || month < 1 || month > 12 {
					continue
				}
				months = append(months, month)
			}
			if len(months) > 0 {
				yearMonths[year] = months
			}
		}
		if len(yearMonths) == 0 {
			yearMonths = nil
		}
	}

	return ReportFilters{
		Districts:            districtValues,
		Year:                 normalizedYear,
		Month:                monthFilter,
		Week:                 normalizeOptionalFilterValue(r.URL.Query().Get("week")),
		Quarter:              normalizeOptionalFilterValue(r.URL.Query().Get("quarter")),
		YearProvided:         yearProvided,
		WeekProvided:         weekProvided,
		MonthProvided:        monthProvided,
		QuarterProvided:      quarterProvided,
		Regions:              regionValues,
		Facilities:           facilityValues,
		YearMonths:           yearMonths,
		SuppressTimeDefaults: suppressTimeDefaults,
		CustomValues:         make(map[string]string),
	}, nil
}

// GetReportHandler returns a specific report by ID
// Supports: /api/report/report-name OR /api/report/category/report-name
func GetReportHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract report ID from URL path
	// Supports: /api/report/report-name OR /api/report/category/report-name
	// Note: basePath is set in main.go from BASE_PATH env var (empty string if not set)
	ctx := r.Context()
	prefix := basePath + "/api/report/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "invalid path"))
		return
	}

	reportID := strings.TrimPrefix(r.URL.Path, prefix)

	// SECURITY: Validate report ID before processing
	if err := ValidateReportID(reportID); err != nil {
		logWarnCtx(ctx, "Invalid report ID in request: %q - %v", reportID, err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, fmt.Sprintf("invalid report ID: %v", err)))
		return
	}

	perms := GetEffectivePermissionsFromRequest(r)
	if perms != nil && len(perms.Paths) > 0 && !UserCanAccessReport(reportID, perms) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "forbidden"))
		return
	}

	filters, err := parseReportFilters(r)
	if err != nil {
		logWarnCtx(ctx, "Filter validation failed: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, err.Error()))
		return
	}

	tempReport, yamlHashForCache, err := loadReportMetadataWithHash(reportID)
	if err != nil {
		logWarnCtx(ctx, "Report not found: %s: %v", reportID, err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "report not found"))
		return
	}

	// Parse custom filter values from query params
	for _, cf := range tempReport.CustomFilters {
		if value := r.URL.Query().Get(cf.Column); value != "" {
			filters.CustomValues[cf.Column] = value
		}
	}

	// Apply smart defaults for week/month/quarter filters only if parameter was NOT provided in URL
	// Uses data-aware detection: queries MAX from the report's table, falls back to hardcoded if query fails
	yearProvided := filters.YearProvided

	primaryTable := getPrimaryTable(tempReport)

	if !filters.SuppressTimeDefaults &&
		!filters.YearProvided &&
		!filters.WeekProvided && filters.Week == "" &&
		!filters.MonthProvided && filters.Month == "" &&
		!filters.QuarterProvided && filters.Quarter == "" &&
		filterSliceContains(tempReport.Filters, "week") {
		if dataWeek := getLatestWeekFromData(r.Context(), primaryTable, tempReport); dataWeek != "" {
			filters.Week = dataWeek
			logDebug("Report %q: data-aware week default = %s (table: %s)", reportID, dataWeek, primaryTable)
		} else {
			filters.Week = getDefaultWeek()
			logDebug("Report %q: fallback week default = %s (data-aware query failed, table: %s)", reportID, filters.Week, primaryTable)
		}
	}

	if !filters.SuppressTimeDefaults &&
		!filters.YearProvided &&
		!filters.MonthProvided && len(filters.YearMonths) == 0 && filters.Month == "" &&
		!filters.WeekProvided && filters.Week == "" &&
		!filters.QuarterProvided && filters.Quarter == "" &&
		filterSliceContains(tempReport.Filters, "month") {
		dataYear, dataMonth := getLatestDataPeriod(r.Context(), primaryTable, tempReport)
		if dataYear > 0 && dataMonth > 0 {
			filters.Month = fmt.Sprintf("%d", dataMonth)
			if !yearProvided {
				filters.Year = fmt.Sprintf("%d", dataYear)
			}
			logDebug("Report %q: data-aware month default = %d/%d (table: %s)", reportID, dataYear, dataMonth, primaryTable)
		} else {
			defaultYear, defaultMonth := getDefaultMonth()
			filters.Month = defaultMonth
			if !yearProvided {
				filters.Year = defaultYear
			}
			logDebug("Report %q: fallback month default = %s/%s (data-aware query failed, table: %s)", reportID, defaultYear, defaultMonth, primaryTable)
		}
	}

	if !filters.SuppressTimeDefaults &&
		!filters.YearProvided &&
		!filters.QuarterProvided && filters.Quarter == "" &&
		!filters.WeekProvided && filters.Week == "" &&
		!filters.MonthProvided && filters.Month == "" && len(filters.YearMonths) == 0 &&
		filterSliceContains(tempReport.Filters, "quarter") {
		dataYear, dataQuarter := getLatestQuarterFromData(r.Context(), primaryTable, tempReport)
		if dataYear > 0 && dataQuarter > 0 {
			filters.Quarter = fmt.Sprintf("%d", dataQuarter)
			if !yearProvided {
				filters.Year = fmt.Sprintf("%d", dataYear)
			}
			logDebug("Report %q: data-aware quarter default = Q%d/%d (table: %s)", reportID, dataQuarter, dataYear, primaryTable)
		} else {
			defaultYear, defaultQuarter := getDefaultQuarter()
			filters.Quarter = defaultQuarter
			if !yearProvided {
				filters.Year = defaultYear
			}
			logDebug("Report %q: fallback quarter default = Q%s/%s (data-aware query failed, table: %s)", reportID, defaultQuarter, defaultYear, primaryTable)
		}
	}

	// Year-only reports (year filter but no month/week/quarter): leave year empty
	// so all available data is shown. The frontend already defaults these to "All".
	// Reports with month/week/quarter filters get year set in the blocks above.

	bypassReportCache := strings.EqualFold(r.URL.Query().Get("refresh"), "1") ||
		strings.EqualFold(r.URL.Query().Get("refresh"), "true")

	if reportCacheEnabled() && reportCacheAllowed(reportID) && !bypassReportCache {
		rcKey := buildReportCacheKey(reportID, yamlHashForCache, filters)
		if cachedBody := fullReportCache.Get(rcKey, yamlHashForCache); len(cachedBody) > 0 {
			w.Header().Set("X-Report-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(cachedBody)
			return
		}
	}

	report := GetReportByID(reportID, filters, r.Context())

	if report == nil {
		logErrorCtx(ctx, "Report execution failed: %s", reportID)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "report not found"))
		return
	}

	body, err := json.Marshal(report)
	if err != nil {
		logErrorCtx(ctx, "JSON marshal report %s: %v", reportID, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "failed to encode report"))
		return
	}

	if shouldStoreReportCache(report, reportID, len(body)) {
		rcKey := buildReportCacheKey(reportID, yamlHashForCache, filters)
		fullReportCache.Set(rcKey, body, yamlHashForCache)
	}

	if reportCacheEnabled() {
		w.Header().Set("X-Report-Cache", "MISS")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// GetDistrictsHandler returns the list of available districts
// Requires ?table parameter to query specific table (e.g., ?table=report.cht_form_097b)
func GetDistrictsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get table parameter from query string (e.g., ?table=my_table)
	table := r.URL.Query().Get("table")
	if table == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "table parameter is required"))
		return
	}

	// Validate table name to prevent SQL injection
	if !validTableFormat.MatchString(table) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Return empty list if database is not initialized (e.g., during tests)
	if DB == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Load report to get district column configuration
	report := loadReportWithDefault(table)

	// Get district column (with default fallback)
	districtCol := GetDistrictColumn(report)

	// Quote table name to preserve case (e.g., report.data_example)
	quotedTable := quoteTableName(table)

	// Check for optional region filter (cascade from region multiselect)
	regionParam := r.URL.Query().Get("region")

	var query string
	var args []interface{}

	// Optional extra WHERE clause from locationColumns.districtWhere (e.g. to exclude region-level values)
	extraWhere := ""
	if report != nil && report.LocationColumns.DistrictWhere != "" {
		extraWhere = " AND " + report.LocationColumns.DistrictWhere
	}

	if regions := splitAndTrimCSV(regionParam); len(regions) > 0 {
		regionCol := GetRegionColumn(report)
		placeholders := make([]string, len(regions))
		for i, region := range regions {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, region)
		}
		query = fmt.Sprintf(
			"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL AND %s != ''%s AND %s IN (%s) ORDER BY %s",
			districtCol, quotedTable, districtCol, districtCol, extraWhere,
			regionCol, strings.Join(placeholders, ","), districtCol)
	} else {
		query = fmt.Sprintf(
			"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL AND %s != ''%s ORDER BY %s",
			districtCol, quotedTable, districtCol, districtCol, extraWhere, districtCol)
	}

	rows, err := DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		logErrorCtx(r.Context(), "Error querying districts from %s: %v", table, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to query districts"))
		return
	}
	defer rows.Close()

	districts := []string{}
	for rows.Next() {
		var district string
		if err := rows.Scan(&district); err != nil {
			logWarnCtx(r.Context(), "Scan error in districts query: %v", err)
			continue
		}
		districts = append(districts, district)
	}
	if err := rows.Err(); err != nil {
		logErrorCtx(r.Context(), "Row iteration error in districts query: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read districts"))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(districts)
}

// GetYearsHandler returns the list of available years from a specified table
// Requires ?table parameter (e.g., ?table=report.cht_form_097b)
func GetYearsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Return empty list if database is not initialized (e.g., during tests)
	if DB == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]int{})
		return
	}

	table := r.URL.Query().Get("table")
	if table == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "table parameter is required"))
		return
	}

	if !validTableFormat.MatchString(table) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]int{})
		return
	}

	// Load report to get timeColumns config
	report := loadReportWithDefault(table)

	// Get year column (with default fallback)
	yearCol := GetTimeColumn(report, "year")
	colType := DetectColumnType(yearCol)

	// Build query based on column type
	var selectExpr string
	switch colType {
	case "numeric":
		selectExpr = fmt.Sprintf("%s as year", yearCol)
	case "string_period":
		selectExpr = fmt.Sprintf("SUBSTRING(%s, 1, 4)::int as year", yearCol)
	case "date":
		selectExpr = fmt.Sprintf("SUBSTR(%s::text, 1, 4)::int as year", yearCol)
	default:
		selectExpr = fmt.Sprintf("SUBSTR(%s::text, 1, 4)::int as year", yearCol)
	}

	// Quote table name to preserve case (e.g., report.data_example)
	quotedTable := quoteTableName(table)
	query := fmt.Sprintf(`
		SELECT DISTINCT %s
		FROM %s
		WHERE %s IS NOT NULL
		ORDER BY year DESC
	`, selectExpr, quotedTable, yearCol)
	rows, err := DB.QueryContext(r.Context(), query)
	if err != nil {
		logErrorCtx(r.Context(), "Error querying years from %s: %v", table, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to query years"))
		return
	}
	defer rows.Close()

	years := []int{}
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			logWarnCtx(r.Context(), "Scan error in years query: %v", err)
			continue
		}
		years = append(years, year)
	}
	if err := rows.Err(); err != nil {
		logErrorCtx(r.Context(), "Row iteration error in years query: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read years"))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(years)
}

// GetMonthsHandler returns the list of available months (optionally filtered by year)
func GetMonthsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	months := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	quarterParam := strings.TrimSpace(r.URL.Query().Get("quarter"))
	if quarterParam != "" {
		allowedMonths := make(map[int]bool)
		for _, q := range strings.Split(quarterParam, ",") {
			q = strings.TrimSpace(q)
			qVal, err := strconv.Atoi(q)
			if err != nil || qVal < 1 || qVal > 4 {
				continue
			}
			startMonth := (qVal-1)*3 + 1
			for m := startMonth; m <= startMonth+2; m++ {
				allowedMonths[m] = true
			}
		}

		if len(allowedMonths) > 0 {
			filtered := make([]int, 0, 12)
			for _, m := range months {
				if allowedMonths[m] {
					filtered = append(filtered, m)
				}
			}
			months = filtered
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(months)
}

// GetQuartersHandler returns the list of available quarters (1-4)
func GetQuartersHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	quarters := []int{1, 2, 3, 4}

	monthParam := strings.TrimSpace(r.URL.Query().Get("month"))
	if monthParam != "" {
		allowedQuarters := make(map[int]bool)
		for _, m := range strings.Split(monthParam, ",") {
			m = strings.TrimSpace(m)
			mVal, err := strconv.Atoi(m)
			if err != nil || mVal < 1 || mVal > 12 {
				continue
			}
			qVal := ((mVal - 1) / 3) + 1
			allowedQuarters[qVal] = true
		}

		if len(allowedQuarters) > 0 {
			filtered := make([]int, 0, 4)
			for _, q := range quarters {
				if allowedQuarters[q] {
					filtered = append(filtered, q)
				}
			}
			quarters = filtered
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(quarters)
}

// GetWeeksHandler returns the list of available weeks from the specified table
// Optionally filtered by year and/or month for cascading filters
// Handles both string-based week formats (e.g., "2024W15") and numeric weeks
func GetWeeksHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Return empty list if database is not initialized (e.g., during tests)
	if DB == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Get table parameter (required)
	table := r.URL.Query().Get("table")
	if table == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "table parameter is required"))
		return
	}

	if !validTableFormat.MatchString(table) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")

	// Load report to get timeColumns config
	report := loadReportWithDefault(table)

	// Get week and year column names
	weekCol := GetTimeColumn(report, "week")
	yearCol := GetTimeColumn(report, "year")
	monthCol := GetTimeColumn(report, "month")
	colType := DetectColumnType(weekCol)

	// Quote table name to preserve case (e.g., report.data_example)
	quotedTable := quoteTableName(table)

	var query string
	var args []any

	// Check if year and week are in separate columns (numeric week format)
	separateYearWeekCols := yearCol != "" && yearCol != weekCol

	if separateYearWeekCols {
		// Year and week are separate columns (e.g., year=2024, week=1-52)
		// Always format as YYYYWNN by selecting both columns
		if year != "" {
			if month != "" {
				monthValues := strings.Split(month, ",")
				var monthPlaceholders []string
				args = []any{year}
				argIdx := 2
				for _, mv := range monthValues {
					mv = strings.TrimSpace(mv)
					mVal, err := strconv.Atoi(mv)
					if err != nil || mVal < 1 || mVal > 12 {
						continue
					}
					monthPlaceholders = append(monthPlaceholders, fmt.Sprintf("$%d", argIdx))
					args = append(args, mVal)
					argIdx++
				}

				if len(monthPlaceholders) > 0 {
					query = fmt.Sprintf(`
						SELECT DISTINCT CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0'))
						FROM %s
						WHERE %s::int = $1
						AND %s::int IN (%s)
						AND %s IS NOT NULL
						ORDER BY CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0')) DESC
					`, yearCol, weekCol, quotedTable, yearCol, monthCol, strings.Join(monthPlaceholders, ","), weekCol, yearCol, weekCol)
				} else {
					query = fmt.Sprintf(`
						SELECT DISTINCT CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0'))
						FROM %s
						WHERE %s::int = $1
						AND %s IS NOT NULL
						ORDER BY CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0')) DESC
					`, yearCol, weekCol, quotedTable, yearCol, weekCol, yearCol, weekCol)
				}
			} else {
				// Filter weeks by year only, format as YYYYWNN
				query = fmt.Sprintf(`
					SELECT DISTINCT CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0'))
					FROM %s
					WHERE %s::int = $1
					AND %s IS NOT NULL
					ORDER BY CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0')) DESC
				`, yearCol, weekCol, quotedTable, yearCol, weekCol, yearCol, weekCol)
				args = []any{year}
			}
		} else {
			// All weeks for table, format as YYYYWNN with year
			query = fmt.Sprintf(`
				SELECT DISTINCT CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0'))
				FROM %s
				WHERE %s IS NOT NULL AND %s IS NOT NULL
				ORDER BY CONCAT(%s::text, 'W', LPAD(%s::text, 2, '0')) DESC
			`, yearCol, weekCol, quotedTable, yearCol, weekCol, yearCol, weekCol)
		}
	} else if colType == "string_period" || colType == "date" {
		// Week is stored as string period (e.g., "2024W15") or date
		if year != "" && month != "" {
			// Filter weeks by year AND month
			query = fmt.Sprintf(`
				SELECT DISTINCT %s
				FROM %s
				WHERE SUBSTRING(%s, 1, 4) = $1
				AND SUBSTRING(%s, 6, 2)::int = $2
				AND %s IS NOT NULL
				ORDER BY %s DESC
			`, weekCol, quotedTable, weekCol, weekCol, weekCol, weekCol)
			args = []any{year, month}
		} else if year != "" {
			// Filter weeks by year only
			query = fmt.Sprintf(`
				SELECT DISTINCT %s
				FROM %s
				WHERE SUBSTRING(%s, 1, 4) = $1
				AND %s IS NOT NULL
				ORDER BY %s DESC
			`, weekCol, quotedTable, weekCol, weekCol, weekCol)
			args = []any{year}
		} else {
			// All weeks for table
			query = fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL ORDER BY %s DESC", weekCol, quotedTable, weekCol, weekCol)
		}
	} else {
		// Fallback for numeric weeks in single column
		if year != "" {
			query = fmt.Sprintf(`
				SELECT DISTINCT %s
				FROM %s
				WHERE %s IS NOT NULL
				ORDER BY %s::int DESC
			`, weekCol, quotedTable, weekCol, weekCol)
			args = []any{year}
		} else {
			query = fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL ORDER BY %s::int DESC", weekCol, quotedTable, weekCol, weekCol)
		}
	}

	rows, err := DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		logErrorCtx(r.Context(), "Error querying weeks from %s: %v", table, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to query weeks"))
		return
	}
	defer rows.Close()

	weeks := []string{} // Initialize as empty slice (not nil) to return [] instead of null
	for rows.Next() {
		var week interface{}
		if err := rows.Scan(&week); err != nil {
			logWarnCtx(r.Context(), "Scan error in weeks query: %v", err)
			continue
		}

		// Format as string - already in YYYYWNN format if separateYearWeekCols
		weekStr := fmt.Sprintf("%v", week)

		// Filter out invalid week 0 (W00 is not ISO 8601 compliant)
		// Valid weeks are W01-W53
		if !strings.HasSuffix(weekStr, "W00") {
			weeks = append(weeks, weekStr)
		}
	}
	if err := rows.Err(); err != nil {
		logErrorCtx(r.Context(), "Row iteration error in weeks query: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read weeks"))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(weeks)
}

// GetRegionsHandler returns the list of available regions from a specified table
// Used for multi-select region filtering
func GetRegionsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Return empty list if database is not initialized (e.g., during tests)
	if DB == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Get table parameter from query string (required)
	table := r.URL.Query().Get("table")
	if table == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "table parameter is required"))
		return
	}

	// Validate table name to prevent SQL injection
	if !validTableFormat.MatchString(table) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Load report to get locationColumns config
	report := loadReportWithDefault(table)

	// Get region column (with default fallback)
	regionCol := GetRegionColumn(report)

	// Quote table name to preserve case
	quotedTable := quoteTableName(table)

	// Query distinct regions from the specified table
	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL AND %s != '' ORDER BY %s", regionCol, quotedTable, regionCol, regionCol, regionCol)
	rows, err := DB.QueryContext(r.Context(), query)
	if err != nil {
		logErrorCtx(r.Context(), "Error querying regions from %s: %v", table, err)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}
	defer rows.Close()

	regions := []string{}
	for rows.Next() {
		var region string
		if err := rows.Scan(&region); err != nil {
			logWarnCtx(r.Context(), "Scan error in regions query: %v", err)
			continue
		}
		regions = append(regions, region)
	}
	if err := rows.Err(); err != nil {
		logErrorCtx(r.Context(), "Row iteration error in regions query: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read regions"))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(regions)
}

// GetFacilitiesHandler returns the list of available facilities from a specified table
// Used for multi-select facility filtering
func GetFacilitiesHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Return empty list if database is not initialized (e.g., during tests)
	if DB == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Get table parameter from query string (required)
	table := r.URL.Query().Get("table")
	if table == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "table parameter is required"))
		return
	}

	// Validate table name to prevent SQL injection
	if !validTableFormat.MatchString(table) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Load report to get locationColumns config.
	// Prefer loading by explicit report ID (passed by GetFilterDefinitions) so we
	// get the correct locationColumns.facility even when multiple reports share the
	// same filterOptionsTable (LoadReportForTable returns the first alphabetical match).
	// Use LoadReportByYAMLID (scans configs matching the id: field) rather than
	// loadReportMetadataWithHash (path-based) because report.ID may not mirror the
	// Directory structure (e.g. id: Sector/Analytics/fac-analytics-001 lives under
	// configs/Sector/Analytics/fac-analytics-001.yaml).
	reportID := r.URL.Query().Get("report")
	var report *Report
	if reportID != "" && validReportIDPattern.MatchString(reportID) {
		report = LoadReportByYAMLID(reportID)
	}
	if report == nil {
		report = loadReportWithDefault(table)
	}

	// Get facility column (with default fallback)
	facilityCol := GetFacilityColumn(report)

	// Quote table name to preserve case
	quotedTable := quoteTableName(table)

	// Check for optional district filter (cascade from district select)
	districtParam := r.URL.Query().Get("district")

	var query string
	var args []interface{}

	if districts := splitAndTrimCSV(districtParam); len(districts) > 0 {
		districtCol := GetDistrictColumn(report)
		placeholders := make([]string, len(districts))
		for i, d := range districts {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, d)
		}
		query = fmt.Sprintf(
			"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL AND %s != '' AND %s IN (%s) ORDER BY %s",
			facilityCol, quotedTable, facilityCol, facilityCol,
			districtCol, strings.Join(placeholders, ","), facilityCol)
	} else {
		query = fmt.Sprintf(
			"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL AND %s != '' ORDER BY %s",
			facilityCol, quotedTable, facilityCol, facilityCol, facilityCol)
	}

	rows, err := DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		logErrorCtx(r.Context(), "Error querying facilities from %s: %v", table, err)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}
	defer rows.Close()

	facilities := []string{}
	for rows.Next() {
		var facility string
		if err := rows.Scan(&facility); err != nil {
			logWarnCtx(r.Context(), "Scan error in facilities query: %v", err)
			continue
		}
		facilities = append(facilities, facility)
	}
	if err := rows.Err(); err != nil {
		logErrorCtx(r.Context(), "Row iteration error in facilities query: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read facilities"))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(facilities)
}

// CustomFilterResponse wraps the filter options response
type CustomFilterResponse struct {
	Values    []string `json:"values"`
	Truncated bool     `json:"truncated"`
	Total     int      `json:"total,omitempty"`
}

// GetCustomFilterHandler returns distinct values for a custom column filter
// GET /api/filters/custom?table=schema.table&column=age_group
func GetCustomFilterHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	table := r.URL.Query().Get("table")
	column := r.URL.Query().Get("column")

	// Validate inputs
	if table == "" || column == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "table and column are required"))
		return
	}

	if !validTableFormat.MatchString(table) || !validColumnName.MatchString(column) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid table or column format"))
		return
	}

	if DB == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CustomFilterResponse{Values: []string{}})
		return
	}

	// Query distinct values (cap at 21 to detect truncation)
	quotedTable := quoteTableName(table)
	quotedColumn := quoteIdentifier(column)
	query := fmt.Sprintf(`
		SELECT DISTINCT %s
		FROM %s
		WHERE %s IS NOT NULL AND TRIM(CAST(%s AS TEXT)) != ''
		ORDER BY %s ASC
		LIMIT 21
	`, quotedColumn, quotedTable, quotedColumn, quotedColumn, quotedColumn)

	rows, err := DB.QueryContext(r.Context(), query)
	if err != nil {
		logErrorCtx(r.Context(), "Error querying custom filter %s.%s: %v", table, column, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to query filter options"))
		return
	}
	defer rows.Close()

	values := make([]string, 0) // Initialize to empty slice (not nil) for proper JSON encoding
	for rows.Next() {
		var value interface{}
		if err := rows.Scan(&value); err != nil {
			logWarnCtx(r.Context(), "Scan error in custom filter query: %v", err)
			continue
		}
		if value != nil {
			values = append(values, fmt.Sprintf("%v", value))
		}
	}
	if err := rows.Err(); err != nil {
		logErrorCtx(r.Context(), "Row iteration error in custom filter query: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read filter options"))
		return
	}

	// Check for truncation
	response := CustomFilterResponse{Values: values, Truncated: false}
	if len(values) > 20 {
		response.Values = values[:20]
		response.Truncated = true
		response.Total = len(values)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// PDFRequest represents the incoming request for PDF generation
type PDFRequest struct {
	HTML     string `json:"html"`
	ReportID string `json:"reportId"`
}

// PreviewReportRequest represents a request to preview a report in the builder
type PreviewReportRequest struct {
	Report  Report                 `json:"report"`
	Filters map[string]interface{} `json:"filters"`
}

// PreviewReportResponse returns the executed report with any component errors
type PreviewReportResponse struct {
	Success  bool             `json:"success"`
	Report   *Report          `json:"report,omitempty"`
	Errors   []ComponentError `json:"errors,omitempty"`
	Error    string           `json:"error,omitempty"`
	Warnings []ReportWarning  `json:"warnings,omitempty"` // Design quality warnings
}

// ComponentError captures an error for a specific component
type ComponentError struct {
	SectionIndex   int    `json:"sectionIndex"`
	ComponentIndex int    `json:"componentIndex"`
	Title          string `json:"title"`
	Error          string `json:"error"`
}

// readCSSFile reads the CSS file from the public directory
func readCSSFile() (string, error) {
	// Get the directory of the current executable
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	// Build path to CSS file
	cssPath := filepath.Join(exeDir, "public", "css", "style.css")

	// Read the CSS file
	cssContent, err := os.ReadFile(cssPath)
	if err != nil {
		return "", fmt.Errorf("could not read CSS file: %v", err)
	}

	return string(cssContent), nil
}

// PreviewReportHandler executes a report from the builder UI for preview
// POST /api/report/preview - accepts report object and filters, returns executed report with errors
func PreviewReportHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(PreviewReportResponse{
			Success: false,
			Error:   "only POST method is allowed",
		})
		return
	}

	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	// Parse request body
	var req PreviewReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PreviewReportResponse{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	// Validate report has at least one section
	if len(req.Report.Sections) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(PreviewReportResponse{
			Success: false,
			Error:   "report has no sections to preview",
		})
		return
	}

	// Convert filters map to ReportFilters
	filters := ReportFilters{
		CustomValues: make(map[string]string),
	}

	// Extract standard filters
	if v, ok := req.Filters["district"]; ok {
		switch val := v.(type) {
		case string:
			if val != "" {
				filters.Districts = []string{val}
			}
		case []interface{}:
			for _, d := range val {
				if s, ok := d.(string); ok && s != "" {
					filters.Districts = append(filters.Districts, s)
				}
			}
		}
	}
	if v, ok := req.Filters["year"].(string); ok {
		filters.Year = v
	}
	if v, ok := req.Filters["month"].(string); ok {
		filters.Month = v
	}
	if v, ok := req.Filters["week"].(string); ok {
		filters.Week = v
	}
	if v, ok := req.Filters["quarter"].(string); ok {
		filters.Quarter = v
	}

	// Extract multi-select location filters
	if v, ok := req.Filters["region"]; ok {
		switch val := v.(type) {
		case string:
			if val != "" {
				filters.Regions = []string{val}
			}
		case []interface{}:
			for _, r := range val {
				if s, ok := r.(string); ok && s != "" {
					filters.Regions = append(filters.Regions, s)
				}
			}
		}
	}
	if v, ok := req.Filters["facility"]; ok {
		switch val := v.(type) {
		case string:
			if val != "" {
				filters.Facilities = []string{val}
			}
		case []interface{}:
			for _, f := range val {
				if s, ok := f.(string); ok && s != "" {
					filters.Facilities = append(filters.Facilities, s)
				}
			}
		}
	}

	// Extract custom filter values
	for _, cf := range req.Report.CustomFilters {
		if v, ok := req.Filters[cf.Column].(string); ok {
			filters.CustomValues[cf.Column] = v
		}
	}

	// Execute the report with error capture
	report, componentErrors := ExecuteReportPreview(&req.Report, filters)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PreviewReportResponse{
		Success:  len(componentErrors) == 0,
		Report:   report,
		Errors:   componentErrors,
		Warnings: ValidateReportWarnings(&req.Report),
	})
}

// GeneratePDFHandler receives HTML content and converts it to PDF
func GeneratePDFHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "only POST method is allowed"))
		return
	}

	start := time.Now()

	// Limit request body to 25MB for PDF payloads
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		RecordPDFDownload("", 0, true)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to read request body"))
		logError("Error reading request body: %v", err)
		return
	}
	defer r.Body.Close()

	var pdfReq PDFRequest
	if err := json.Unmarshal(body, &pdfReq); err != nil {
		RecordPDFDownload("", 0, true)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid JSON"))
		logError("Error unmarshaling JSON: %v", err)
		return
	}

	if pdfReq.HTML == "" {
		RecordPDFDownload(pdfReq.ReportID, 0, true)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "html field is required"))
		return
	}

	// Create PDF-specific CSS (inline styles for chromedp/headless Chrome)
	// Optimized for compact output
	pdfCSS := `
		/* PDF-specific styles - print-dense layout */
		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}

		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Helvetica Neue', sans-serif;
			background: white !important;
			color: #2c3e50;
			line-height: 1.2;
			font-size: 11px;
		}

		/* Report header - compact */
		.report-header {
			text-align: center;
			margin-bottom: 10px;
			padding-bottom: 8px;
			border-bottom: 2px solid #e6dfd5;
		}

		.report-header-brand {
			display: flex;
			flex-direction: column;
			align-items: center;
			gap: 2px;
			margin-bottom: 4px;
		}

		.report-header-logo {
			width: 36px;
			height: auto;
		}

		.report-header-ministry {
			font-size: 1em;
			font-weight: 700;
			color: #2c3e50;
			text-transform: uppercase;
			letter-spacing: 0.5px;
		}

		.report-header-category {
			font-size: 0.95em;
			font-weight: 600;
			color: #5a6c7d;
			text-transform: uppercase;
			margin-bottom: 2px;
		}

		.report-header-title {
			font-size: 1.1em;
			font-weight: 700;
			color: #2c3e50;
			margin-bottom: 4px;
		}

		.report-header-filters {
			font-size: 0.85em;
			color: #5a6c7d;
		}

		/* Sections: allow page breaks freely so Chrome doesn't leave
		   half-page gaps pushing entire sections to the next page */
		.section {
			margin-bottom: 6px;
			background: white;
			border: 1px solid #e6dfd5;
			border-radius: 3px;
			padding: 7px 8px;
		}

		.section-title {
			font-size: 1.05em;
			font-weight: 600;
			color: #2c3e50;
			margin-bottom: 5px;
			padding-bottom: 4px;
			border-bottom: 2px solid #e6dfd5;
		}

		.section-components {
			display: grid;
			gap: 5px;
		}

		.section-components.layout-single {
			grid-template-columns: 1fr;
		}

		.section-components.layout-two-column {
			grid-template-columns: repeat(2, 1fr);
		}

		.section-components.layout-three-column {
			grid-template-columns: repeat(3, 1fr);
		}

		.section-components.layout-four-column {
			grid-template-columns: repeat(4, 1fr);
		}

		/* Faceted choropleth */
		.component-faceted-choropleth {
			grid-column: 1 / -1;
			padding: 4px;
		}

		.facet-grid {
			display: grid;
			grid-template-columns: repeat(4, 1fr);
			gap: 5px;
		}

		.facet-item {
			border: 1px solid #e6dfd5;
			border-radius: 3px;
			overflow: hidden;
			background: white;
		}

		.facet-title {
			padding: 3px 5px;
			font-weight: 600;
			font-size: 0.78em;
			background: #f8f3e9;
			border-bottom: 1px solid #e6dfd5;
			color: #2c3e50;
		}

		.facet-map-container {
			height: 180px;
			position: relative;
		}

		.facet-item .component-choropleth {
			height: 100%;
			border: none;
		}

		/* Components: no page-break-inside:avoid on charts/maps —
		   that was pushing large content to new pages and leaving big gaps.
		   Only small text/KPI cards keep avoid so they don't split mid-card. */
		.component {
			background: white;
			border: 1px solid #e6dfd5;
			border-radius: 3px;
			padding: 6px 8px;
		}

		.component-text {
			page-break-inside: avoid;
			line-height: 1.35;
			color: #2c3e50;
		}

		.component-title {
			font-size: 0.95em;
			font-weight: 600;
			margin-bottom: 5px;
			padding-bottom: 3px;
			border-bottom: 1px solid #e6dfd5;
			color: #2c3e50;
		}

		/* Table styling */
		.component-table {
			overflow-x: visible;
		}

		.component-table table {
			width: 100%;
			border-collapse: collapse;
			font-size: 0.8em;
		}

		.component-table thead {
			background: #f0ebe3;
			border-bottom: 2px solid #d8cfc2;
		}

		.component-table th {
			padding: 4px 6px;
			text-align: left;
			font-weight: 600;
			color: #2c3e50;
			border-bottom: 2px solid #d8cfc2;
		}

		.component-table td {
			padding: 3px 6px;
			border-bottom: 1px solid #e6dfd5;
			color: #2c3e50;
		}

		.component-table tbody tr:nth-child(even) {
			background: #f8f3e9;
		}

		/* Text component prose */
		.component-text h3,
		.component-text h4 {
			margin-top: 5px;
			margin-bottom: 4px;
			font-weight: 600;
			font-size: 0.95em;
		}

		.component-text p {
			margin-bottom: 4px;
		}

		.component-text ul,
		.component-text ol {
			margin-left: 14px;
			margin-bottom: 4px;
		}

		.component-text li {
			margin-bottom: 2px;
		}

		/* Chart component — cap height so tall charts don't dominate a full page */
		.component-chart {
			width: 100%;
		}

		.component-chart img {
			width: 100%;
			max-height: 260px;
			object-fit: contain;
			display: block;
		}

		/* Map/choropleth */
		.component-map,
		.component-choropleth {
			width: 100%;
			border: 1px solid #e6dfd5;
			border-radius: 3px;
			overflow: hidden;
		}

		.component-map img,
		.component-choropleth img {
			width: 100%;
			height: auto;
			display: block;
		}

		/* Hide interactive elements */
		.table-search-container,
		.table-pagination,
		.search-btn,
		.clear-btn,
		button {
			display: none !important;
		}

		/* PDF disclaimer */
		.pdf-disclaimer {
			margin-top: 20px;
			padding: 8px 12px;
			border-top: 1px solid #d5cec4;
			font-size: 8px;
			color: #8b7e74;
			line-height: 1.4;
			text-align: left;
		}
		.pdf-disclaimer strong {
			color: #6b5e54;
		}
	`

	// Wrap the HTML content with CSS in a complete HTML document
	fullHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        %s
    </style>
</head>
<body>
    <div id="sections-container">
        %s
    </div>
    <div class="pdf-disclaimer">
        <strong>Disclaimer:</strong> This report contains preliminary data from routine health information systems. Figures are subject to change as data quality improves. Not for public distribution.
    </div>
</body>
</html>`, pdfCSS, pdfReq.HTML)

	// Per-IP PDF concurrency check (max 2 per IP)
	ip := getClientIP(r)
	counterVal, _ := pdfPerIP.LoadOrStore(ip, &atomic.Int32{})
	ipCounter, ok := counterVal.(*atomic.Int32)
	if !ok {
		RecordPDFDownload(pdfReq.ReportID, 0, true)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "internal error"))
		logError("PDF per-IP counter type assertion failed for %s", ip)
		return
	}
	if ipCounter.Add(1) > 2 {
		ipCounter.Add(-1)
		RecordPDFDownload(pdfReq.ReportID, 0, true)
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "you already have 2 PDFs generating, please wait"))
		logWarn("PDF per-IP limit reached for %s", ip)
		return
	}

	// Acquire global semaphore (limit to 5 concurrent PDF generations)
	select {
	case pdfSemaphore <- struct{}{}:
		defer func() {
			<-pdfSemaphore
			ipCounter.Add(-1)
		}()
	case <-time.After(30 * time.Second):
		ipCounter.Add(-1)
		RecordPDFDownload(pdfReq.ReportID, 0, true)
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "PDF generation queue is full, please try again later"))
		logError("PDF generation timeout: semaphore full")
		return
	}

	// Create context with timeout for PDF generation
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a new browser context
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	// Write HTML to a temp file (data URLs can exceed size limits with embedded images)
	tmpFile, err := os.CreateTemp("", "pdf-*.html")
	if err != nil {
		RecordPDFDownload(pdfReq.ReportID, 0, true)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to create temp file"))
		logError("Error creating temp file: %v", err)
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(fullHTML); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			logError("Error closing temp file after write failure: %v", closeErr)
		}
		RecordPDFDownload(pdfReq.ReportID, 0, true)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to write temp file"))
		logError("Error writing temp file: %v", err)
		return
	}
	if err := tmpFile.Close(); err != nil {
		logError("Error closing temp file: %v", err)
	}

	fileURL := "file://" + tmpFile.Name()

	var pdfBuffer []byte

	// Navigate and generate PDF
	// Set viewport to desktop width (1400px) to prevent responsive breakpoints from triggering
	err = chromedp.Run(browserCtx,
		chromedp.EmulateViewport(1400, 900),
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuffer, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).         // A4 portrait width in inches
				WithPaperHeight(11.69).       // A4 portrait height in inches
				WithMarginTop(0.2).           // ~5mm - reduced for compact layout
				WithMarginBottom(0.5).        // ~12mm for footer
				WithMarginLeft(0.2).          // ~5mm - reduced for compact layout
				WithMarginRight(0.2).         // ~5mm - reduced for compact layout
				WithPreferCSSPageSize(false). // Use our paper dimensions
				WithScale(0.9).               // Slightly larger scale for better readability
				WithDisplayHeaderFooter(true).
				WithHeaderTemplate("<span></span>").
				WithFooterTemplate(`<div style="font-size:8px;text-align:center;width:100%;margin:0 auto;">Page <span class="pageNumber"></span> of <span class="totalPages"></span></div>`).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		RecordPDFDownload(pdfReq.ReportID, time.Since(start).Milliseconds(), true)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to generate PDF"))
		logError("Error generating PDF with chromedp: %v", err)
		return
	}

	// Send PDF as response
	RecordPDFDownload(pdfReq.ReportID, time.Since(start).Milliseconds(), false)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=report.pdf")
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBuffer)
}

// GetCacheStatsHandler returns cache statistics for monitoring
// GET /api/cache/stats - returns cache size and TTL info
// GET /api/cache/stats?entries=true - includes individual entry details
func GetCacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	includeEntries := r.URL.Query().Get("entries") == "true"
	stats := componentCache.GetStats(includeEntries)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		CacheStats
		ReportCache ReportCacheStats `json:"reportCache"`
	}{
		CacheStats:  stats,
		ReportCache: fullReportCache.stats(),
	})
}

// ClearCacheHandler clears all cached component results
// POST /api/cache/clear - clears the entire cache
func ClearCacheHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "only POST method is allowed"))
		return
	}

	componentCache.Clear()
	fullReportCache.Clear()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "cache cleared"})
}

// DownloadTableCSVHandler streams a table_advanced component as a CSV file without the
// AdvancedSQLMaxRows limit so "All regions" downloads include every row.
//
// GET /api/download/csv?report=<id>&sectionId=<id>&componentIndex=<n>&<filter params>
func DownloadTableCSVHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		setCORSHeaders(w, r)
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()

	// --- 1. Parse route params ---
	reportID := r.URL.Query().Get("report")
	sectionID := r.URL.Query().Get("sectionId")
	componentIndexStr := r.URL.Query().Get("componentIndex")

	if reportID == "" || sectionID == "" || componentIndexStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "report, sectionId and componentIndex are required"))
		return
	}

	if err := ValidateReportID(reportID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, fmt.Sprintf("invalid report ID: %v", err)))
		return
	}

	componentIndex, err := strconv.Atoi(componentIndexStr)
	if err != nil || componentIndex < 0 {
		w.WriteHeader(http.StatusBadRequest)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "componentIndex must be a non-negative integer"))
		return
	}

	// --- 2. Load report metadata ---
	report, _, err := loadReportMetadataWithHash(reportID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "report not found"))
		return
	}

	// --- 3. Parse filters (same as GetReportHandler) ---
	filters, err := parseReportFilters(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, err.Error()))
		return
	}

	// Parse custom filter values
	for _, cf := range report.CustomFilters {
		if value := r.URL.Query().Get(cf.Column); value != "" {
			filters.CustomValues[cf.Column] = value
		}
	}

	// Apply smart time defaults (mirrors GetReportHandler)
	yearProvided := filters.YearProvided
	primaryTable := getPrimaryTable(report)

	if !filters.SuppressTimeDefaults &&
		!filters.YearProvided &&
		!filters.WeekProvided && filters.Week == "" &&
		!filters.MonthProvided && filters.Month == "" &&
		!filters.QuarterProvided && filters.Quarter == "" &&
		filterSliceContains(report.Filters, "week") {
		if dataWeek := getLatestWeekFromData(ctx, primaryTable, report); dataWeek != "" {
			filters.Week = dataWeek
		} else {
			filters.Week = getDefaultWeek()
		}
	}

	if !filters.SuppressTimeDefaults &&
		!filters.YearProvided &&
		!filters.MonthProvided && len(filters.YearMonths) == 0 && filters.Month == "" &&
		!filters.WeekProvided && filters.Week == "" &&
		!filters.QuarterProvided && filters.Quarter == "" &&
		filterSliceContains(report.Filters, "month") {
		dataYear, dataMonth := getLatestDataPeriod(ctx, primaryTable, report)
		if dataYear > 0 && dataMonth > 0 {
			filters.Month = fmt.Sprintf("%d", dataMonth)
			if !yearProvided {
				filters.Year = fmt.Sprintf("%d", dataYear)
			}
		} else {
			defaultYear, defaultMonth := getDefaultMonth()
			filters.Month = defaultMonth
			if !yearProvided {
				filters.Year = defaultYear
			}
		}
	}

	if !filters.SuppressTimeDefaults &&
		!filters.YearProvided &&
		!filters.QuarterProvided && filters.Quarter == "" &&
		!filters.WeekProvided && filters.Week == "" &&
		!filters.MonthProvided && filters.Month == "" && len(filters.YearMonths) == 0 &&
		filterSliceContains(report.Filters, "quarter") {
		dataYear, dataQuarter := getLatestQuarterFromData(ctx, primaryTable, report)
		if dataYear > 0 && dataQuarter > 0 {
			filters.Quarter = fmt.Sprintf("%d", dataQuarter)
			if !yearProvided {
				filters.Year = fmt.Sprintf("%d", dataYear)
			}
		} else {
			defaultYear, defaultQuarter := getDefaultQuarter()
			filters.Quarter = defaultQuarter
			if !yearProvided {
				filters.Year = defaultYear
			}
		}
	}

	// --- 4. Find component ---
	var component *Component
	for i := range report.Sections {
		if report.Sections[i].ID != sectionID {
			continue
		}
		if componentIndex >= len(report.Sections[i].Components) {
			break
		}
		c := report.Sections[i].Components[componentIndex]
		component = &c
		break
	}

	if component == nil {
		w.WriteHeader(http.StatusNotFound)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "component not found"))
		return
	}

	if component.Type != "table_advanced" || component.SQL == "" {
		w.WriteHeader(http.StatusBadRequest)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "component is not a table_advanced with SQL"))
		return
	}

	// --- 5. Validate SQL ---
	if err := ValidateAdvancedSQL(component.SQL); err != nil {
		logErrorCtx(ctx, "Invalid SQL for download component %s: %v", component.Title, err)
		w.WriteHeader(http.StatusBadRequest)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "invalid component SQL"))
		return
	}

	// --- 6. Build SQL (same pipeline as processAdvancedSQLComponent, without LIMIT) ---
	filterMap := map[string]string{}
	if len(filters.Districts) > 0 {
		filterMap["district"] = filters.Districts[0]
		filterMap["district_list"] = buildSQLList(filters.Districts)
	}
	if filters.Year != "" {
		filterMap["year"] = filters.Year
	}
	if filters.Month != "" {
		monthVal := ConvertMonthFilterToText(filters.Month, report)
		filterMap["month"] = monthVal
		months := strings.Split(monthVal, ",")
		filterMap["month_list"] = buildSQLList(months)
	}
	if filters.Quarter != "" {
		filterMap["quarter"] = filters.Quarter
		quarters := strings.Split(filters.Quarter, ",")
		filterMap["quarter_list"] = buildSQLList(quarters)
	}
	if filters.Week != "" {
		filterMap["week"] = filters.Week
		weeks := strings.Split(filters.Week, ",")
		filterMap["week_list"] = buildSQLList(weeks)
	}
	if len(filters.Regions) > 0 {
		filterMap["region"] = filters.Regions[0]
		filterMap["region_list"] = buildSQLList(filters.Regions)
	}
	if len(filters.Facilities) > 0 {
		filterMap["facility"] = filters.Facilities[0]
		filterMap["facility_list"] = buildSQLList(filters.Facilities)
	}

	sql, args := applyFiltersToSQL(component.SQL, filters, report)
	sql = substituteFilters(sql, filterMap)
	sql = interpolateFilterPlaceholders(sql, filters)
	sql = cleanupUnsubstitutedFilters(sql)
	// No LIMIT wrapper — this is the whole point of this endpoint.

	// --- 7. Execute ---
	tableData, err := executeQueryWithContext(ctx, DB, sql, args...)
	if err != nil {
		logErrorCtx(ctx, "Download CSV query error for %s/%s[%d]: %v", reportID, sectionID, componentIndex, err)
		w.WriteHeader(http.StatusInternalServerError)
		setCORSHeaders(w, r)
		json.NewEncoder(w).Encode(NewErrorResponse(ctx, "query failed"))
		return
	}

	// --- 8. Build filename ---
	tableTitle := component.Title
	if tableTitle == "" {
		tableTitle = "table"
	}
	safeTitle := strings.ToLower(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(tableTitle, "-"))
	safeTitle = strings.Trim(safeTitle, "-")
	filename := safeTitle + ".csv"

	// --- 9. Stream CSV ---
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	cw := csv.NewWriter(w)
	_ = cw.Write(tableData.Headers)
	for _, row := range tableData.Rows {
		record := make([]string, len(row))
		for i, cell := range row {
			if cell == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprintf("%v", cell)
			}
		}
		_ = cw.Write(record)
	}
	cw.Flush()
}
