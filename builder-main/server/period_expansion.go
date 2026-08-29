package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// latestPeriodTTL controls how long the "latest available period" results are cached.
// These queries (SELECT MAX year/month/week/quarter) run on every default-filter load,
// so caching them avoids repeated DB hits for data that changes at most monthly.
const latestPeriodTTL = 12 * time.Hour

var latestPeriodMu sync.Map // key string -> *latestPeriodCacheEntry

type latestPeriodCacheEntry struct {
	v1, v2    int    // year+month or year+quarter
	s         string // week string
	createdAt time.Time
}

func latestPeriodKey(table, col1, col2 string) string {
	return table + "|" + col1 + "|" + col2
}

// getLatestDataPeriod queries the database to find the most recent period with data
// Returns (year, month) of the latest data, or (0, 0) if not found
// Handles both date-type columns (period_date) and numeric columns (year, month)
func getLatestDataPeriod(ctx context.Context, table string, report *Report) (int, int) {
	if table == "" || DB == nil {
		return 0, 0
	}

	yearCol := GetTimeColumn(report, "year")
	monthCol := GetTimeColumn(report, "month")

	cacheKey := latestPeriodKey(table, yearCol, monthCol)
	if v, ok := latestPeriodMu.Load(cacheKey); ok {
		e := v.(*latestPeriodCacheEntry)
		if time.Since(e.createdAt) < latestPeriodTTL {
			logDebug("getLatestDataPeriod: cache hit for %s (%d/%d)", table, e.v1, e.v2)
			return e.v1, e.v2
		}
	}
	yearColType := DetectColumnType(yearCol)
	monthColType := ResolveMonthColumnType(report)

	quotedTable := quoteTableName(table)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var year, month int

	if monthColType == "text" && yearColType == "numeric" {
		// Text month column (e.g., "January", "February"): map to numeric for ordering
		var monthName string
		query := fmt.Sprintf(`
			SELECT %s, %s
			FROM %s
			WHERE %s IS NOT NULL AND %s IS NOT NULL
			ORDER BY %s DESC,
				CASE %s
					WHEN 'January' THEN 1 WHEN 'February' THEN 2 WHEN 'March' THEN 3
					WHEN 'April' THEN 4 WHEN 'May' THEN 5 WHEN 'June' THEN 6
					WHEN 'July' THEN 7 WHEN 'August' THEN 8 WHEN 'September' THEN 9
					WHEN 'October' THEN 10 WHEN 'November' THEN 11 WHEN 'December' THEN 12
				END DESC
			LIMIT 1
		`, yearCol, monthCol, quotedTable, yearCol, monthCol, yearCol, monthCol)
		err := DB.QueryRowContext(ctx, query).Scan(&year, &monthName)
		if err != nil {
			logDebug("getLatestDataPeriod: text month query failed for %s: %v", table, err)
			return 0, 0
		}
		// Convert month name to number
		for i, name := range []string{"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December"} {
			if strings.EqualFold(strings.TrimSpace(monthName), name) {
				month = i + 1
				break
			}
		}
	} else if yearColType == "numeric" && monthColType == "numeric" {
		query := fmt.Sprintf(`
			SELECT %s, %s
			FROM %s
			WHERE %s IS NOT NULL AND %s IS NOT NULL
			ORDER BY %s DESC, %s DESC
			LIMIT 1
		`, yearCol, monthCol, quotedTable, yearCol, monthCol, yearCol, monthCol)
		err := DB.QueryRowContext(ctx, query).Scan(&year, &month)
		if err != nil {
			logDebug("getLatestDataPeriod: numeric query failed for %s: %v", table, err)
			return 0, 0
		}
	} else {
		periodColumn := monthCol
		if periodColumn == "" {
			periodColumn = yearCol
		}
		if periodColumn == "" {
			periodColumn = "period_date"
		}
		query := fmt.Sprintf(`
			SELECT
				EXTRACT(YEAR FROM %s::date)::int AS year,
				EXTRACT(MONTH FROM %s::date)::int AS month
			FROM %s
			WHERE %s IS NOT NULL
			ORDER BY %s DESC
			LIMIT 1
		`, periodColumn, periodColumn, quotedTable, periodColumn, periodColumn)
		err := DB.QueryRowContext(ctx, query).Scan(&year, &month)
		if err != nil {
			logDebug("getLatestDataPeriod: date query failed for %s: %v", table, err)
			return 0, 0
		}
	}

	if year > 0 && month > 0 {
		latestPeriodMu.Store(cacheKey, &latestPeriodCacheEntry{v1: year, v2: month, createdAt: time.Now()})
	}
	return year, month
}

// getLatestWeekFromData queries the database to find the most recent week with data
// Returns week string in ISO format (e.g. "2024W15"), or "" if not found
func getLatestWeekFromData(ctx context.Context, table string, report *Report) string {
	if table == "" || DB == nil {
		return ""
	}

	weekCol := GetTimeColumn(report, "week")

	cacheKey := latestPeriodKey(table, weekCol, "")
	if v, ok := latestPeriodMu.Load(cacheKey); ok {
		e := v.(*latestPeriodCacheEntry)
		if time.Since(e.createdAt) < latestPeriodTTL {
			logDebug("getLatestWeekFromData: cache hit for %s (%s)", table, e.s)
			return e.s
		}
	}

	quotedTable := quoteTableName(table)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var week string
	query := fmt.Sprintf(`
		SELECT %s
		FROM %s
		WHERE %s IS NOT NULL AND TRIM(CAST(%s AS TEXT)) != ''
		ORDER BY %s DESC
		LIMIT 1
	`, weekCol, quotedTable, weekCol, weekCol, weekCol)
	err := DB.QueryRowContext(ctx, query).Scan(&week)
	if err != nil {
		logDebug("getLatestWeekFromData: query failed for %s: %v", table, err)
		return ""
	}
	week = strings.TrimSpace(week)
	if week != "" {
		latestPeriodMu.Store(cacheKey, &latestPeriodCacheEntry{s: week, createdAt: time.Now()})
	}
	return week
}

// getLatestQuarterFromData queries the database to find the most recent quarter with data
// Returns (year, quarter) of the latest data, or (0, 0) if not found
func getLatestQuarterFromData(ctx context.Context, table string, report *Report) (int, int) {
	if table == "" || DB == nil {
		return 0, 0
	}

	yearCol := GetTimeColumn(report, "year")
	quarterCol := GetTimeColumn(report, "quarter")

	cacheKey := latestPeriodKey(table, yearCol, quarterCol)
	if v, ok := latestPeriodMu.Load(cacheKey); ok {
		e := v.(*latestPeriodCacheEntry)
		if time.Since(e.createdAt) < latestPeriodTTL {
			logDebug("getLatestQuarterFromData: cache hit for %s (%d/Q%d)", table, e.v1, e.v2)
			return e.v1, e.v2
		}
	}

	yearColType := DetectColumnType(yearCol)
	quarterColType := DetectColumnType(quarterCol)

	quotedTable := quoteTableName(table)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var year, quarter int

	if yearColType == "numeric" && (quarterColType == "numeric" || quarterColType == "string_period") {
		// Direct numeric query: both year and quarter columns store numbers directly
		// This also handles "string_period" quarter columns like "Period" that store quarter numbers (1-4)
		query := fmt.Sprintf(`
			SELECT %s, %s
			FROM %s
			WHERE %s IS NOT NULL AND %s IS NOT NULL
			ORDER BY %s DESC, %s DESC
			LIMIT 1
		`, yearCol, quarterCol, quotedTable, yearCol, quarterCol, yearCol, quarterCol)
		err := DB.QueryRowContext(ctx, query).Scan(&year, &quarter)
		if err != nil {
			logDebug("getLatestQuarterFromData: numeric query failed for %s: %v", table, err)
			return 0, 0
		}
	} else {
		periodColumn := quarterCol
		if periodColumn == "" {
			periodColumn = "period_date"
		}
		query := fmt.Sprintf(`
			SELECT
				EXTRACT(YEAR FROM %s::date)::int AS year,
				EXTRACT(QUARTER FROM %s::date)::int AS quarter
			FROM %s
			WHERE %s IS NOT NULL
			ORDER BY %s DESC
			LIMIT 1
		`, periodColumn, periodColumn, quotedTable, periodColumn, periodColumn)
		err := DB.QueryRowContext(ctx, query).Scan(&year, &quarter)
		if err != nil {
			logDebug("getLatestQuarterFromData: date query failed for %s: %v", table, err)
			return 0, 0
		}
	}

	if year > 0 && quarter > 0 {
		latestPeriodMu.Store(cacheKey, &latestPeriodCacheEntry{v1: year, v2: quarter, createdAt: time.Now()})
	}
	return year, quarter
}

// GetPeriodTypeFromGroupBy determines the period type from groupBy configuration
// Returns "month", "week", "quarter", or "" (empty) based on the groupBy format
// Also infers period type from field name when format is not specified
func GetPeriodTypeFromGroupBy(groupBy []GroupBy) string {
	for _, gb := range groupBy {
		// First, check explicit format (existing logic)
		switch gb.Format {
		case "month", "month_year", "month_noyear":
			return "month"
		case "week", "week_year", "week_noyear":
			return "week"
		case "quarter", "quarter_year", "quarter_noyear":
			return "quarter"
		case "year":
			return "year"
		}

		// Infer from field name if format not specified
		if gb.Format == "" {
			fieldLower := strings.ToLower(gb.Field)
			if fieldLower == "month" {
				return "month"
			}
			if fieldLower == "week" {
				return "week"
			}
			if fieldLower == "quarter" {
				return "quarter"
			}
		}
	}

	// Second pass: check for year field (lowest priority, since year often appears alongside other periods)
	for _, gb := range groupBy {
		if gb.Format == "" && strings.ToLower(gb.Field) == "year" {
			return "year"
		}
	}

	return ""
}

// ExpandMonthRange generates a range of months working backwards from an anchor point
// Returns comma-separated years and months strings for filter expansion
// Example: year=2024, month=3, count=6 -> years="2023,2024", months="10,11,12,1,2,3"
func ExpandMonthRange(year, month, count int) (years string, months string) {
	if count <= 0 {
		return strconv.Itoa(year), strconv.Itoa(month)
	}

	yearSet := make(map[int]bool)
	var monthList []int

	currentYear := year
	currentMonth := month

	for i := 0; i < count; i++ {
		yearSet[currentYear] = true
		monthList = append(monthList, currentMonth)

		// Move to previous month
		currentMonth--
		if currentMonth < 1 {
			currentMonth = 12
			currentYear--
		}
	}

	// Sort years ascending
	var yearList []int
	for y := range yearSet {
		yearList = append(yearList, y)
	}
	sort.Ints(yearList)

	// Reverse month list to get chronological order (oldest first)
	for i, j := 0, len(monthList)-1; i < j; i, j = i+1, j-1 {
		monthList[i], monthList[j] = monthList[j], monthList[i]
	}

	// Build comma-separated strings
	var yearStrs []string
	for _, y := range yearList {
		yearStrs = append(yearStrs, strconv.Itoa(y))
	}

	var monthStrs []string
	for _, m := range monthList {
		monthStrs = append(monthStrs, strconv.Itoa(m))
	}

	return strings.Join(yearStrs, ","), strings.Join(monthStrs, ",")
}

// ExpandMonthRangeWithYearMap generates a range of months and returns a map of year -> months
// This avoids the cross-product issue when filtering across year boundaries
// Example: year=2024, month=2, count=5 -> {"2023": [10,11,12], "2024": [1,2]}
func ExpandMonthRangeWithYearMap(year, month, count int) map[string][]int {
	if count <= 0 {
		return map[string][]int{strconv.Itoa(year): {month}}
	}

	yearMonths := make(map[string][]int)

	currentYear := year
	currentMonth := month

	for i := 0; i < count; i++ {
		yearStr := strconv.Itoa(currentYear)
		yearMonths[yearStr] = append(yearMonths[yearStr], currentMonth)

		// Move to previous month
		currentMonth--
		if currentMonth < 1 {
			currentMonth = 12
			currentYear--
		}
	}

	return yearMonths
}

// ExpandWeekRange generates a range of ISO weeks working backwards from an anchor week
// Returns comma-separated week strings in YYYYWNN format
// Example: anchorWeek="2024W02", count=6 -> "2023W49,2023W50,2023W51,2023W52,2024W01,2024W02"
func ExpandWeekRange(anchorWeek string, count int) string {
	if count <= 0 || anchorWeek == "" {
		return anchorWeek
	}

	// Parse anchor week (format: YYYYWNN)
	year, week, err := ParseWeekPeriod(anchorWeek)
	if err != nil {
		return anchorWeek
	}

	var weeks []string

	for i := 0; i < count; i++ {
		weeks = append(weeks, fmt.Sprintf("%dW%02d", year, week))

		// Move to previous week
		week--
		if week < 1 {
			year--
			week = getISOWeeksInYear(year)
		}
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(weeks)-1; i < j; i, j = i+1, j-1 {
		weeks[i], weeks[j] = weeks[j], weeks[i]
	}

	return strings.Join(weeks, ",")
}

// getISOWeeksInYear returns the number of ISO weeks in a given year (52 or 53)
// A year has 53 weeks if January 1st is a Thursday, or if it's a leap year
// and January 1st is a Wednesday
func getISOWeeksInYear(year int) int {
	// January 4th is always in week 1 of its year (ISO 8601)
	// Find the last day of the last week by checking December 28-31
	dec28 := time.Date(year, 12, 28, 0, 0, 0, 0, time.UTC)
	_, week := dec28.ISOWeek()
	return week
}

// expandPeriodFilters expands the period range based on periodLimit configuration
// Takes a component's query and returns modified filters with expanded period range
func expandPeriodFilters(ctx context.Context, query *Query, filters ReportFilters, report *Report) ReportFilters {
	if query == nil || query.PeriodLimit <= 0 {
		return filters
	}

	periodType := GetPeriodTypeFromGroupBy(query.GroupBy)
	if periodType == "" {
		return filters
	}

	// Create a copy of filters to modify
	newFilters := filters

	switch periodType {
	case "month":
		newFilters = expandMonthFilters(ctx, query.PeriodLimit, filters, report, query.Table)
	case "week":
		newFilters = expandWeekFilters(ctx, query.PeriodLimit, filters, report)
	case "quarter":
		newFilters = expandQuarterFilters(ctx, query.PeriodLimit, filters, report)
	case "year":
		newFilters = expandYearFilters(ctx, query.PeriodLimit, filters, report, query.Table)
	}

	return newFilters
}

// expandMonthFilters expands month filters based on periodLimit
func expandMonthFilters(ctx context.Context, periodLimit int, filters ReportFilters, report *Report, table string) ReportFilters {
	newFilters := filters

	// Determine anchor year and month
	var anchorYear, anchorMonth int

	// Get year: from filter or use default
	if filters.Year != "" {
		// If multiple years, use the largest
		years := strings.Split(filters.Year, ",")
		maxYear := 0
		for _, y := range years {
			if yVal, err := strconv.Atoi(strings.TrimSpace(y)); err == nil && yVal > maxYear {
				maxYear = yVal
			}
		}
		if maxYear > 0 {
			anchorYear = maxYear
		}
	}

	// Get month: from filter or use default
	if filters.Month != "" {
		// If multiple months, use the largest
		months := strings.Split(filters.Month, ",")
		maxMonth := 0
		for _, m := range months {
			if mVal, err := strconv.Atoi(strings.TrimSpace(m)); err == nil && mVal > maxMonth {
				maxMonth = mVal
			}
		}
		if maxMonth > 0 {
			anchorMonth = maxMonth
		}
	}

	// If either year or month is not set from filters, try to detect from data
	if anchorYear == 0 || anchorMonth == 0 {
		dataYear, dataMonth := getLatestDataPeriod(ctx, table, report)
		if dataYear > 0 && dataMonth > 0 {
			if anchorYear == 0 {
				anchorYear = dataYear
			}
			if anchorMonth == 0 {
				anchorMonth = dataMonth
			}
		} else {
			// Fallback to 2 months ago if detection fails
			defaultYear, defaultMonth := getDefaultMonth()
			if anchorYear == 0 {
				if yVal, err := strconv.Atoi(defaultYear); err == nil {
					anchorYear = yVal
				} else {
					anchorYear = time.Now().Year()
				}
			}
			if anchorMonth == 0 {
				if mVal, err := strconv.Atoi(defaultMonth); err == nil {
					anchorMonth = mVal
				} else {
					anchorMonth = int(time.Now().Month())
				}
			}
		}
	}

	// Expand the range
	years, months := ExpandMonthRange(anchorYear, anchorMonth, periodLimit)
	newFilters.Year = years
	newFilters.Month = months

	// Also populate YearMonths map for precise filtering (avoids cross-product)
	newFilters.YearMonths = ExpandMonthRangeWithYearMap(anchorYear, anchorMonth, periodLimit)

	logDebug("PeriodLimit expansion: anchor=%d-%02d, limit=%d -> years=%s, months=%s, yearMonths=%v",
		anchorYear, anchorMonth, periodLimit, years, months, newFilters.YearMonths)

	return newFilters
}

// expandWeekFilters expands week filters based on periodLimit
func expandWeekFilters(ctx context.Context, periodLimit int, filters ReportFilters, report *Report) ReportFilters {
	newFilters := filters

	// Determine anchor week
	var anchorWeek string

	if filters.Week != "" {
		// If multiple weeks, use the largest (lexicographic sort works for YYYYWNN)
		weeks := strings.Split(filters.Week, ",")
		maxWeek := ""
		for _, w := range weeks {
			w = strings.TrimSpace(w)
			if w > maxWeek {
				maxWeek = w
			}
		}
		if maxWeek != "" {
			anchorWeek = maxWeek
		}
	}
	if anchorWeek == "" {
		// Use default week (2 weeks before current date)
		anchorWeek = getDefaultWeek()
	}
	if anchorWeek == "" {
		// Fallback to current week
		year, week := time.Now().ISOWeek()
		anchorWeek = fmt.Sprintf("%dW%02d", year, week)
	}

	// Expand the range
	newFilters.Week = ExpandWeekRange(anchorWeek, periodLimit)

	// Extract years from week range for year filter
	weeks := strings.Split(newFilters.Week, ",")
	yearSet := make(map[string]bool)
	for _, w := range weeks {
		if len(w) >= 4 {
			yearSet[w[:4]] = true
		}
	}
	var yearList []string
	for y := range yearSet {
		yearList = append(yearList, y)
	}
	sort.Strings(yearList)
	newFilters.Year = strings.Join(yearList, ",")

	logDebug("PeriodLimit expansion: anchor=%s, limit=%d -> weeks=%s, years=%s",
		anchorWeek, periodLimit, newFilters.Week, newFilters.Year)

	return newFilters
}

// ExpandQuarterRange generates a range of quarters working backwards from an anchor quarter
// Returns comma-separated years and quarters strings for filter expansion
// Example: year=2024, quarter=2, count=6 -> years="2023,2024", quarters="1,2,3,4,1,2"
func ExpandQuarterRange(year, quarter, count int) (years string, quarters string) {
	if count <= 0 {
		return strconv.Itoa(year), strconv.Itoa(quarter)
	}

	yearSet := make(map[int]bool)
	var quarterList []int

	currentYear := year
	currentQuarter := quarter

	for i := 0; i < count; i++ {
		yearSet[currentYear] = true
		quarterList = append(quarterList, currentQuarter)

		// Move to previous quarter
		currentQuarter--
		if currentQuarter < 1 {
			currentQuarter = 4
			currentYear--
		}
	}

	// Sort years ascending
	var yearList []int
	for y := range yearSet {
		yearList = append(yearList, y)
	}
	sort.Ints(yearList)

	// Reverse quarter list to get chronological order (oldest first)
	for i, j := 0, len(quarterList)-1; i < j; i, j = i+1, j-1 {
		quarterList[i], quarterList[j] = quarterList[j], quarterList[i]
	}

	// Build comma-separated strings
	var yearStrs []string
	for _, y := range yearList {
		yearStrs = append(yearStrs, strconv.Itoa(y))
	}

	var quarterStrs []string
	for _, q := range quarterList {
		quarterStrs = append(quarterStrs, strconv.Itoa(q))
	}

	return strings.Join(yearStrs, ","), strings.Join(quarterStrs, ",")
}

// ExpandQuarterRangeWithYearMap generates a range of quarters and returns a map of year -> quarters
// This avoids the cross-product issue when filtering across year boundaries
// Example: year=2024, quarter=2, count=6 -> {"2023": [1,2,3,4], "2024": [1,2]}
func ExpandQuarterRangeWithYearMap(year, quarter, count int) map[string][]int {
	if count <= 0 {
		return map[string][]int{strconv.Itoa(year): {quarter}}
	}

	yearQuarters := make(map[string][]int)

	currentYear := year
	currentQuarter := quarter

	for i := 0; i < count; i++ {
		yearStr := strconv.Itoa(currentYear)
		yearQuarters[yearStr] = append(yearQuarters[yearStr], currentQuarter)

		// Move to previous quarter
		currentQuarter--
		if currentQuarter < 1 {
			currentQuarter = 4
			currentYear--
		}
	}

	return yearQuarters
}

// PeriodFacet represents a single period facet with its label and filter values
type PeriodFacet struct {
	Label  string // Human-readable label (e.g., "Q4 2024", "Dec 2024", "2024")
	Year   int    // Year value for filtering
	Period int    // Month (1-12), Quarter (1-4), or 0 for year-only
	Value  string // Raw value for query params (e.g., "2024Q4", "2024-12", "2024")
}

// GeneratePeriodFacets generates a slice of period facets working backwards from an anchor date
// periodType: "month", "quarter", "year"
// count: number of facets (max 4)
// anchorYear: starting year
// anchorPeriod: starting month (1-12), quarter (1-4), or 0 for year
func GeneratePeriodFacets(periodType string, count int, anchorYear int, anchorPeriod int) []PeriodFacet {
	// Cap count at 4
	if count > 4 {
		count = 4
	}
	if count < 1 {
		count = 1
	}

	facets := make([]PeriodFacet, 0, count)
	currentYear := anchorYear
	currentPeriod := anchorPeriod

	monthNames := []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

	for i := 0; i < count; i++ {
		var facet PeriodFacet
		facet.Year = currentYear

		switch periodType {
		case "month":
			facet.Period = currentPeriod
			facet.Label = fmt.Sprintf("%s %d", monthNames[currentPeriod], currentYear)
			facet.Value = fmt.Sprintf("%d-%02d", currentYear, currentPeriod)

			// Move to previous month
			currentPeriod--
			if currentPeriod < 1 {
				currentPeriod = 12
				currentYear--
			}

		case "quarter":
			facet.Period = currentPeriod
			facet.Label = fmt.Sprintf("Q%d %d", currentPeriod, currentYear)
			facet.Value = fmt.Sprintf("%dQ%d", currentYear, currentPeriod)

			// Move to previous quarter
			currentPeriod--
			if currentPeriod < 1 {
				currentPeriod = 4
				currentYear--
			}

		case "week":
			facet.Period = currentPeriod
			facet.Label = fmt.Sprintf("W%02d %d", currentPeriod, currentYear)
			facet.Value = fmt.Sprintf("%dW%02d", currentYear, currentPeriod)

			// Move to previous week
			currentPeriod--
			if currentPeriod < 1 {
				currentYear--
				currentPeriod = getISOWeeksInYear(currentYear)
			}

		case "year":
			facet.Period = 0
			facet.Label = fmt.Sprintf("%d", currentYear)
			facet.Value = fmt.Sprintf("%d", currentYear)

			// Move to previous year
			currentYear--
		}

		facets = append(facets, facet)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(facets)-1; i < j; i, j = i+1, j-1 {
		facets[i], facets[j] = facets[j], facets[i]
	}

	return facets
}

// expandQuarterFilters expands quarter filters based on periodLimit
func expandQuarterFilters(ctx context.Context, periodLimit int, filters ReportFilters, report *Report) ReportFilters {
	newFilters := filters

	// Determine anchor year and quarter
	var anchorYear, anchorQuarter int

	// Get year: from filter or use current year
	if filters.Year != "" {
		// If multiple years, use the largest
		years := strings.Split(filters.Year, ",")
		maxYear := 0
		for _, y := range years {
			if yVal, err := strconv.Atoi(strings.TrimSpace(y)); err == nil && yVal > maxYear {
				maxYear = yVal
			}
		}
		if maxYear > 0 {
			anchorYear = maxYear
		}
	}
	if anchorYear == 0 {
		anchorYear = time.Now().Year()
	}

	// Get quarter: from filter or calculate from current month
	if filters.Quarter != "" {
		// If multiple quarters, use the largest
		quarters := strings.Split(filters.Quarter, ",")
		maxQuarter := 0
		for _, q := range quarters {
			if qVal, err := strconv.Atoi(strings.TrimSpace(q)); err == nil && qVal > maxQuarter {
				maxQuarter = qVal
			}
		}
		if maxQuarter > 0 {
			anchorQuarter = maxQuarter
		}
	}
	if anchorQuarter == 0 {
		// Default to current quarter based on current month
		currentMonth := int(time.Now().Month())
		anchorQuarter = (currentMonth-1)/3 + 1 // Q1=Jan-Mar, Q2=Apr-Jun, Q3=Jul-Sep, Q4=Oct-Dec
	}

	// Expand the range
	years, quarters := ExpandQuarterRange(anchorYear, anchorQuarter, periodLimit)
	newFilters.Year = years
	newFilters.Quarter = quarters

	// Also populate YearQuarters map for precise filtering (avoids cross-product)
	newFilters.YearQuarters = ExpandQuarterRangeWithYearMap(anchorYear, anchorQuarter, periodLimit)

	logDebug("PeriodLimit expansion (quarter): anchor=%d-Q%d, limit=%d -> years=%s, quarters=%s, yearQuarters=%v",
		anchorYear, anchorQuarter, periodLimit, years, quarters, newFilters.YearQuarters)

	return newFilters
}

// ExpandYearRange generates a range of years working backwards from an anchor year
// Returns comma-separated year string for filter expansion
// Example: anchorYear=2024, count=5 -> "2020,2021,2022,2023,2024"
func ExpandYearRange(anchorYear, count int) string {
	if count <= 0 {
		return strconv.Itoa(anchorYear)
	}

	years := make([]int, count)
	currentYear := anchorYear

	for i := count - 1; i >= 0; i-- {
		years[i] = currentYear
		currentYear--
	}

	var yearStrs []string
	for _, y := range years {
		yearStrs = append(yearStrs, strconv.Itoa(y))
	}

	return strings.Join(yearStrs, ",")
}

// expandYearFilters expands year filters based on periodLimit
func expandYearFilters(ctx context.Context, periodLimit int, filters ReportFilters, report *Report, table string) ReportFilters {
	newFilters := filters

	var anchorYear int

	if filters.Year != "" {
		years := strings.Split(filters.Year, ",")
		maxYear := 0
		for _, y := range years {
			if yVal, err := strconv.Atoi(strings.TrimSpace(y)); err == nil && yVal > maxYear {
				maxYear = yVal
			}
		}
		if maxYear > 0 {
			anchorYear = maxYear
		}
	}

	if anchorYear == 0 {
		dataYear, _ := getLatestDataPeriod(ctx, table, report)
		if dataYear > 0 {
			anchorYear = dataYear
		} else {
			anchorYear = time.Now().Year()
		}
	}

	newFilters.Year = ExpandYearRange(anchorYear, periodLimit)

	logDebug("PeriodLimit expansion (year): anchor=%d, limit=%d -> years=%s",
		anchorYear, periodLimit, newFilters.Year)

	return newFilters
}
