package main

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// MaxComponentsPerReport caps the number of components in a single report
// to prevent excessive goroutine spawning and resource exhaustion.
// Current max across all reports is 32; cap of 60 provides ample headroom.
const MaxComponentsPerReport = 60

// ChartSQLMaxRows is the maximum number of rows returned for chart SQL components.
// Charts shouldn't render thousands of data points — 500 is generous.
const ChartSQLMaxRows = 500

// isChartType returns true for component types that render as charts (bar, line, bar_line, pie, pyramid).
func isChartType(t string) bool {
	switch t {
	case "bar", "line", "bar_line", "pie", "pyramid":
		return true
	}
	return false
}

// synthesizeKpiContent builds a .kpi-card HTML stub from a kpi component's
// structured fields. The {{value}} placeholder (or a custom valueTemplate) is
// filled by the standard text-substitution pipeline downstream.
//
// accentColor sets a per-card --kpi-accent CSS variable so the card border
// colour can be controlled from YAML without any inline HTML. When accentColor
// is set, the kpi-card--accent modifier class is added automatically.
//
// valueTemplate lets authors express multi-alias display patterns such as
// "{{has_reported}} ({{reporting_percentage}}%)" without writing HTML.
// When omitted, the single "{{value}}" placeholder is used (existing behaviour).
func synthesizeKpiContent(c Component) string {
	classes := "kpi-card"
	if c.Tone != "" {
		classes += " kpi-card--" + c.Tone
	}
	if c.AccentColor != "" && c.Tone == "" {
		// Only auto-add accent modifier when no explicit tone is set; tone takes precedence.
		classes += " kpi-card--accent"
	}

	var sb strings.Builder
	sb.WriteString(`<div class="`)
	sb.WriteString(classes)
	sb.WriteString(`"`)
	if c.AccentColor != "" {
		sb.WriteString(` style="--kpi-accent:`)
		sb.WriteString(html.EscapeString(c.AccentColor))
		sb.WriteString(`"`)
	}
	sb.WriteString(">\n  <h5 class=\"kpi-card__title\">")
	sb.WriteString(html.EscapeString(c.Title))
	sb.WriteString("</h5>\n  <p class=\"kpi-card__value\">")
	if c.ValueTemplate != "" {
		sb.WriteString(c.ValueTemplate)
	} else {
		sb.WriteString("{{value}}")
	}
	if c.Unit != "" {
		sb.WriteString(`<span class="kpi-card__unit">`)
		sb.WriteString(html.EscapeString(c.Unit))
		sb.WriteString(`</span>`)
	}
	sb.WriteString("</p>")
	if c.Description != "" {
		sb.WriteString("\n  <p class=\"kpi-card__sub\">")
		sb.WriteString(html.EscapeString(c.Description))
		sb.WriteString("</p>")
	}
	sb.WriteString("\n</div>")
	return sb.String()
}

// ParseWeekPeriod extracts year and week number from ISO week period string
// Expected format: YYYYWNN (e.g., "2024W15" returns year=2024, week=15)
// Returns an error if the format is invalid
func ParseWeekPeriod(period string) (year int, week int, err error) {
	if len(period) != 7 { // "YYYYWNN"
		return 0, 0, fmt.Errorf("invalid period format: expected YYYYWNN, got %s", period)
	}

	year, err = strconv.Atoi(period[0:4])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year in period: %v", err)
	}

	week, err = strconv.Atoi(period[5:7])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid week in period: %v", err)
	}

	return year, week, nil
}

const maxReportIDSlashes = 5

// ValidateReportID validates report IDs to prevent path traversal and enforce naming rules
// Valid format: category[/subcategory...]/report-name
// - Supports nested report folders up to maxReportIDSlashes separators
// - Only alphanumeric, hyphens, underscores, and forward slash allowed
// - No path traversal sequences (.., ./, etc.)
func ValidateReportID(id string) error {
	if id == "" {
		return fmt.Errorf("report ID cannot be empty")
	}

	// Reject absolute paths
	if strings.HasPrefix(id, "/") || strings.HasPrefix(id, "\\") {
		return fmt.Errorf("report ID cannot be an absolute path: %s", id)
	}

	// Reject path traversal sequences
	if strings.Contains(id, "..") {
		return fmt.Errorf("report ID cannot contain '..' sequences: %s", id)
	}

	if strings.Contains(id, "./") || strings.Contains(id, ".\\") {
		return fmt.Errorf("report ID cannot contain './' sequences: %s", id)
	}

	// Character whitelist: [a-zA-Z0-9_-/]
	if !validReportIDPattern.MatchString(id) {
		return fmt.Errorf("report ID contains invalid characters (allowed: a-z A-Z 0-9 _ - /): %s", id)
	}

	// Bound nesting depth so report IDs remain predictable while still supporting
	// program taxonomies such as Programs/Pharmaceutical-Services/.../report.
	slashCount := strings.Count(id, "/")
	if slashCount > maxReportIDSlashes {
		return fmt.Errorf("report ID cannot have more than %d path separators: %s", maxReportIDSlashes, id)
	}

	// Reject trailing/leading slashes
	if strings.HasPrefix(id, "/") || strings.HasSuffix(id, "/") {
		return fmt.Errorf("report ID cannot start or end with slash: %s", id)
	}

	// Reject empty segments (e.g., "category//report")
	if strings.Contains(id, "//") {
		return fmt.Errorf("report ID cannot contain empty segments: %s", id)
	}

	return nil
}

// Reserved filter names that cannot be used for custom filters
var reservedFilterNames = map[string]bool{
	"district": true, "year": true, "month": true,
	"quarter": true, "week": true, "region": true, "facility": true, "facility_select": true,
}

// validReportIDPattern validates report ID characters
var validReportIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-/]+$`)

// validColumnName validates column names to prevent SQL injection
var validColumnName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateCustomFilters checks custom filter definitions for errors
func ValidateCustomFilters(filters []CustomFilterDef) error {
	for _, f := range filters {
		// Check reserved names
		if reservedFilterNames[f.Column] {
			return fmt.Errorf("custom filter column %q conflicts with reserved filter name", f.Column)
		}
		// Validate column name format (alphanumeric + underscore, must start with letter or underscore)
		if !validColumnName.MatchString(f.Column) {
			return fmt.Errorf("invalid custom filter column name: %s", f.Column)
		}
		// Validate table format
		if !validTableFormat.MatchString(f.Table) {
			return fmt.Errorf("invalid custom filter table: %s", f.Table)
		}
		// Validate type
		if f.Type != "select" && f.Type != "multiselect" {
			return fmt.Errorf("custom filter type must be 'select' or 'multiselect': %s", f.Type)
		}
	}
	return nil
}

// injectCustomFilterPlaceholders adds placeholders for custom filters to the SQL query
// The placeholders are later replaced by applyFiltersToSQL with actual filter values
func injectCustomFilterPlaceholders(sql string, customFilters []CustomFilterDef, componentTable string) string {
	if len(customFilters) == 0 {
		return sql
	}

	// Inject a placeholder for every custom filter defined on the report.
	// cf.Table is the source table for fetching filter option values (the dropdown),
	// NOT a restriction on which component tables the filter applies to — so we
	// intentionally ignore it here and inject unconditionally.
	var customPlaceholders string
	for _, cf := range customFilters {
		customPlaceholders += fmt.Sprintf(" {{custom_filter_%s}}", cf.Column)
	}

	// Append to the end of WHERE clause (before GROUP BY, ORDER BY, or LIMIT)
	// Look for common SQL keywords that come after WHERE
	keywords := []string{" GROUP BY ", " ORDER BY ", " LIMIT "}
	for _, keyword := range keywords {
		idx := strings.Index(strings.ToUpper(sql), keyword)
		if idx != -1 {
			return sql[:idx] + customPlaceholders + sql[idx:]
		}
	}

	// If no keywords found, append to the end
	return sql + customPlaceholders
}

// getPrimaryTable determines the primary data table used by a report
// Counts table usage across all components and returns the most-used table
// Uses alphabetical order as tiebreaker for determinism
// Returns empty string if no tables are defined in the report's components
func getPrimaryTable(report *Report) string {
	tableCounts := make(map[string]int)
	for _, section := range report.Sections {
		for _, component := range section.Components {
			if component.Query != nil && component.Query.Table != "" {
				tableCounts[component.Query.Table]++
			} else if component.FilterTable != "" {
				tableCounts[component.FilterTable]++
			} else if component.SQL != "" {
				// Extract table from raw SQL for table_advanced/choropleth components
				schema, table, found := extractTableFromSQL(component.SQL)
				if found {
					tableCounts[schema+"."+table]++
				}
			}
		}
	}

	if len(tableCounts) == 0 {
		return ""
	}

	// Find the table with the highest usage count (alphabetical tiebreaker)
	var bestTable string
	bestCount := 0
	for table, count := range tableCounts {
		if count > bestCount || (count == bestCount && (bestTable == "" || table < bestTable)) {
			bestTable = table
			bestCount = count
		}
	}
	return bestTable
}

// GetFilterDefinitions returns filter metadata definitions for rendering dynamic UI
// Takes the report to determine data sources and customize endpoints accordingly
// This eliminates the need for hardcoded report IDs in the frontend
func GetFilterDefinitions(filters []string, report *Report) FilterDefinitions {
	primaryTable := getPrimaryTable(report)
	locationFilterTable := primaryTable
	if report != nil && report.FilterOptionsTable != "" {
		locationFilterTable = report.FilterOptionsTable
	}

	// Append the report ID to facility endpoints so GetFacilitiesHandler can load
	// the exact report and resolve locationColumns.facility correctly, even when
	// multiple reports share the same filterOptionsTable.
	reportIDParam := ""
	if report != nil && report.ID != "" {
		reportIDParam = "&report=" + url.QueryEscape(report.ID)
	}

	// Define all available filters and their properties
	allFilters := FilterDefinitions{
		"district": {
			Type:         "select",
			Label:        "District",
			APIEndpoint:  "/api/filters/districts?table=" + locationFilterTable,
			DependsOn:    []string{},
			CascadesTo:   []string{"facility", "facility_select"},
			ParamName:    "district",
			ParentParams: []string{"region"},
		},
		"year": {
			Type:         "select",
			Label:        "Year",
			APIEndpoint:  "/api/filters/years?table=" + primaryTable,
			DependsOn:    []string{},
			CascadesTo:   []string{"quarter", "month", "week"},
			ParamName:    "year",
			ParentParams: []string{},
		},
		"month": {
			Type:         "select",
			Label:        "Month",
			APIEndpoint:  "/api/filters/months",
			DependsOn:    []string{"year"},
			CascadesTo:   []string{"week"},
			ParamName:    "month",
			ParentParams: []string{"year", "quarter"},
		},
		"week": {
			Type:         "select",
			Label:        "Week",
			APIEndpoint:  "/api/filters/weeks?table=" + primaryTable,
			DependsOn:    []string{"year"},
			CascadesTo:   []string{},
			ParamName:    "week",
			ParentParams: []string{"year", "month"},
		},
		"quarter": {
			Type:         "select",
			Label:        "Quarter",
			APIEndpoint:  "/api/filters/quarters",
			DependsOn:    []string{"year"},
			CascadesTo:   []string{"month", "week"},
			ParamName:    "quarter",
			ParentParams: []string{"year", "month"},
		},
		"region": {
			Type:         "multiselect",
			Label:        "Region",
			APIEndpoint:  "/api/filters/regions?table=" + locationFilterTable,
			DependsOn:    []string{},
			CascadesTo:   []string{"district"},
			ParamName:    "region",
			ParentParams: []string{},
		},
		"facility": {
			Type:         "multiselect",
			Label:        "Facility",
			APIEndpoint:  "/api/filters/facilities?table=" + locationFilterTable + reportIDParam,
			DependsOn:    []string{},
			CascadesTo:   []string{},
			ParamName:    "facility",
			ParentParams: []string{"district"},
		},
		// facility_select: same data source as facility but rendered as a plain <select>
		// dropdown instead of the typeahead autocomplete. Use in YAML filters: list
		// as "facility_select" — the backend reads it via the same "facility" URL param.
		"facility_select": {
			Type:         "select",
			Label:        "Facility",
			APIEndpoint:  "/api/filters/facilities?table=" + locationFilterTable + reportIDParam,
			DependsOn:    []string{},
			CascadesTo:   []string{},
			ParamName:    "facility",
			ParentParams: []string{"district"},
		},
	}

	// Filter to only include definitions for filters used by this report
	// Also clean up dependencies to only include filters that actually exist in this report
	filterSet := make(map[string]bool)
	for _, f := range filters {
		filterSet[f] = true
	}

	result := make(FilterDefinitions)
	for _, filterName := range filters {
		if def, exists := allFilters[filterName]; exists {
			// Clean up dependencies: only keep those that exist in this report
			cleanedDef := def

			var cleanedDependencies []string
			for _, dep := range def.DependsOn {
				if filterSet[dep] {
					cleanedDependencies = append(cleanedDependencies, dep)
				}
			}
			cleanedDef.DependsOn = cleanedDependencies

			var cleanedCascades []string
			for _, cascade := range def.CascadesTo {
				if filterSet[cascade] {
					cleanedCascades = append(cleanedCascades, cascade)
				}
			}
			cleanedDef.CascadesTo = cleanedCascades

			// Update parentParams to only include parent filters that exist
			var cleanedParentParams []string
			for _, param := range def.ParentParams {
				if filterSet[param] {
					cleanedParentParams = append(cleanedParentParams, param)
				}
			}
			cleanedDef.ParentParams = cleanedParentParams

			result[filterName] = cleanedDef
		}
	}

	// Add custom filter definitions
	for _, cf := range report.CustomFilters {
		result[cf.Column] = FilterDefinition{
			Type:         cf.Type,
			Label:        cf.Label,
			APIEndpoint:  fmt.Sprintf("/api/filters/custom?table=%s&column=%s", cf.Table, cf.Column),
			DependsOn:    []string{},
			CascadesTo:   []string{},
			ParamName:    cf.Column,
			ParentParams: []string{},
			DefaultValue: cf.DefaultValue,
			NoDefault:    cf.NoDefault,
			HideNone:     cf.HideNone,
		}
	}

	return result
}

// GetAllReports returns the list of available reports by scanning configs directory recursively
// Supports nested subdirectories: configs/section[/subsection...]/report.yaml
// Also returns all section paths (including empty folders) for menu building
func GetAllReports() ReportsList {
	// Always return a non-nil slice so the JSON API renders "reports": []
	// instead of "reports": null when no report YAMLs are configured yet.
	reports := make([]ReportListItem, 0)
	sectionsMap := make(map[string]bool) // Track unique section paths

	// Walk the configs directory recursively
	err := filepath.Walk("configs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logError("Error accessing path %s: %v", path, err)
			return nil
		}

		// Calculate relative path from configs directory
		relPath, err := filepath.Rel("configs", path)
		if err != nil {
			logError("Error calculating relative path for %s: %v", path, err)
			return nil
		}

		// Skip the root configs directory itself
		if relPath == "." {
			return nil
		}

		// Handle directories - collect as potential sections
		if info.IsDir() {
			sectionPath := strings.ReplaceAll(relPath, string(filepath.Separator), "/")
			sectionsMap[sectionPath] = true
			return nil
		}

		// Only process .yaml files
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Enforce the same bounded nesting used by ValidateReportID.
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxReportIDSlashes {
			logDebug("Skipping deeply nested file: %s", relPath)
			return nil
		}

		// Read and parse YAML
		data, err := os.ReadFile(path)
		if err != nil {
			logError("Error reading report file %s: %v", path, err)
			return nil
		}

		var report Report
		if err := yaml.Unmarshal(data, &report); err != nil {
			logError("Error unmarshalling report file %s: %v", path, err)
			return nil
		}

		// Extract category from directory structure or YAML
		category := extractCategory(relPath, &report)

		// Construct report ID from relative path (without .yaml extension)
		reportID := strings.TrimSuffix(relPath, ".yaml")
		// Normalize path separators to forward slash
		reportID = strings.ReplaceAll(reportID, string(filepath.Separator), "/")

		// Validate the constructed ID
		if err := ValidateReportID(reportID); err != nil {
			logWarn("Invalid report ID %q: %v", reportID, err)
			return nil
		}

		// Collect section and component titles for search indexing
		var searchParts []string
		for _, section := range report.Sections {
			if section.Title != "" {
				searchParts = append(searchParts, section.Title)
			}
			for _, comp := range section.Components {
				if comp.Title != "" {
					searchParts = append(searchParts, comp.Title)
				}
			}
		}

		reports = append(reports, ReportListItem{
			ID:          reportID,
			Title:       report.Title,
			Description: report.Description,
			Keywords:    report.Keywords,
			Category:    category,
			SearchText:  strings.Join(searchParts, " "),
		})

		return nil
	})

	if err != nil {
		logError("Error walking configs directory: %v", err)
	}

	// Convert sections map to sorted slice
	sections := make([]string, 0)
	for section := range sectionsMap {
		sections = append(sections, section)
	}
	sort.Strings(sections)

	return ReportsList{Reports: reports, Sections: sections}
}

// extractCategory derives category from directory structure or YAML metadata
// Priority: 1) YAML category field (if set), 2) Directory structure, 3) Empty (root level)
// Returns "Section/Subsection/..." format for nested folders
func extractCategory(relPath string, report *Report) string {
	// Priority 1: YAML category field (explicit override)
	if report.Category != "" {
		return report.Category
	}

	// Priority 2: Extract all directory segments before the report filename.
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "/")
	}

	// Priority 3: Root level reports have no category
	return ""
}

// processFacetedComponent handles choropleth components with facet configuration.
// It supports both structured-query choropleths and custom-SQL choropleths.
func processFacetedComponent(
	ctx context.Context,
	component Component,
	filters ReportFilters,
	reportFilters []string,
	report *Report,
	result *ComponentResult,
) {
	facetConfig := component.Facet

	// Determine anchor year and period from filters
	anchorYear := time.Now().Year()
	anchorPeriod := 0

	if filters.Year != "" {
		// Use the most recent year from filters
		years := strings.Split(filters.Year, ",")
		for _, y := range years {
			if yVal, err := strconv.Atoi(strings.TrimSpace(y)); err == nil && yVal > anchorYear {
				anchorYear = yVal
			}
		}
		// If no year greater than current, use last in list
		if len(years) > 0 {
			if yVal, err := strconv.Atoi(strings.TrimSpace(years[len(years)-1])); err == nil {
				anchorYear = yVal
			}
		}
	}

	switch facetConfig.PeriodType {
	case "month":
		if filters.Month != "" {
			months := strings.Split(filters.Month, ",")
			maxMonth := 0
			for _, m := range months {
				if mVal, err := strconv.Atoi(strings.TrimSpace(m)); err == nil && mVal > maxMonth {
					maxMonth = mVal
				}
			}
			if maxMonth > 0 {
				anchorPeriod = maxMonth
			}
		}
		if anchorPeriod == 0 {
			// Default to 2 months ago
			anchorPeriod = int(time.Now().Month()) - 2
			if anchorPeriod < 1 {
				anchorPeriod += 12
				anchorYear--
			}
		}
	case "quarter":
		if filters.Quarter != "" {
			quarters := strings.Split(filters.Quarter, ",")
			maxQuarter := 0
			for _, q := range quarters {
				if qVal, err := strconv.Atoi(strings.TrimSpace(q)); err == nil && qVal > maxQuarter {
					maxQuarter = qVal
				}
			}
			if maxQuarter > 0 {
				anchorPeriod = maxQuarter
			}
		}
		if anchorPeriod == 0 {
			// Default to current quarter
			anchorPeriod = (int(time.Now().Month())-1)/3 + 1
		}
	case "week":
		if filters.Week != "" {
			// Parse the most recent week from filter
			weeks := strings.Split(filters.Week, ",")
			maxWeek := ""
			for _, w := range weeks {
				w = strings.TrimSpace(w)
				if w > maxWeek {
					maxWeek = w
				}
			}
			if maxWeek != "" {
				year, week, err := ParseWeekPeriod(maxWeek)
				if err == nil {
					anchorYear = year
					anchorPeriod = week
				}
			}
		}
		if anchorPeriod == 0 {
			// Default to 2 weeks ago
			year, week := time.Now().ISOWeek()
			week -= 2
			if week < 1 {
				year--
				week += getISOWeeksInYear(year)
			}
			anchorYear = year
			anchorPeriod = week
		}
	case "year":
		anchorPeriod = 0
	}

	// Generate facets
	facets := GeneratePeriodFacets(facetConfig.PeriodType, facetConfig.Count, anchorYear, anchorPeriod)

	logDebug("Faceted component %q: generating %d %s facets from %d/%d",
		component.Title, len(facets), facetConfig.PeriodType, anchorYear, anchorPeriod)

	if component.SQL == "" && component.Query == nil {
		result.Error = fmt.Errorf("faceted choropleth %s requires query or sql", component.Title)
		result.Data = Data{}
		return
	}

	var sqlQuery string
	if component.Query != nil {
		var err error
		sqlQuery, err = BuildSQL(*component.Query, component.Type, component.Query.Table, reportFilters)
		if err != nil {
			logError("Error building SQL for faceted component %s: %v", component.Title, err)
			result.Error = err
			result.Data = Data{}
			return
		}
		// Inject custom filter placeholders into the SQL
		sqlQuery = injectCustomFilterPlaceholders(sqlQuery, report.CustomFilters, component.Query.Table)
	}

	// Execute query for each facet
	facetInstances := make([]FacetInstance, 0, len(facets))

	for _, facet := range facets {
		// Create modified filters for this facet
		facetFilters := filters
		facetFilters.Year = strconv.Itoa(facet.Year)

		switch facetConfig.PeriodType {
		case "month":
			facetFilters.Month = strconv.Itoa(facet.Period)
			facetFilters.Quarter = ""     // Clear quarter filter
			facetFilters.YearMonths = nil // Clear combined filter
			facetFilters.YearQuarters = nil
		case "quarter":
			facetFilters.Quarter = strconv.Itoa(facet.Period)
			facetFilters.Month = "" // Clear month filter
			facetFilters.YearMonths = nil
			facetFilters.YearQuarters = nil
		case "week":
			facetFilters.Week = fmt.Sprintf("%dW%02d", facet.Year, facet.Period)
			facetFilters.Month = ""
			facetFilters.Quarter = ""
			facetFilters.YearMonths = nil
			facetFilters.YearQuarters = nil
		case "year":
			facetFilters.Month = ""
			facetFilters.Quarter = ""
			facetFilters.YearMonths = nil
			facetFilters.YearQuarters = nil
		}

		if component.SQL != "" {
			choroData, err := executeChoroplethSQLData(ctx, component, facetFilters, report)
			if err != nil {
				logError("Error executing SQL facet %q for component %s: %v", facet.Label, component.Title, err)
				continue
			}

			facetInstances = append(facetInstances, FacetInstance{
				Label: facet.Label,
				Value: facet.Value,
				Data:  choroData,
			})
			continue
		}

		// Apply filters to SQL
		filteredSQL, args := applyFiltersToSQL(sqlQuery, facetFilters, report)
		logDebug("Facet %q SQL: %s", facet.Label, filteredSQL)

		// Execute query
		tableData, err := executeQueryWithContext(ctx, DB, filteredSQL, args...)
		if err != nil {
			logError("Error executing facet %q query for component %s: %v", facet.Label, component.Title, err)
			continue // Skip this facet but continue with others
		}

		// Transform data
		transformedData, err := transformData(component.Type, tableData, &component)
		if err != nil {
			logError("Error transforming facet %q data for component %s: %v", facet.Label, component.Title, err)
			continue
		}

		// Extract choropleth data
		if choroData, ok := transformedData.(ChoroplethData); ok {
			facetInstances = append(facetInstances, FacetInstance{
				Label: facet.Label,
				Value: facet.Value,
				Data:  choroData,
			})
		}
	}

	// Store facet results
	result.Data.Facets = facetInstances

	// Also preserve static choropleth fields from component YAML
	result.Data.GeoJSONPath = component.Data.GeoJSONPath
	result.Data.ColorScheme = component.Data.ColorScheme
	result.Data.BinLabels = component.Data.BinLabels
	result.Data.LegendTitle = component.Data.LegendTitle
	result.Data.BinEdges = component.Data.BinEdges
}

// cleanupUnsubstitutedFilters removes filter conditions that weren't substituted
// This handles the case where a user doesn't select all available filters
// e.g., "AND year = {{year}}" is removed if year wasn't selected
var filterConditionPattern = regexp.MustCompile(`\s*AND\s+\w+\s*=\s*\{\{[^}]+\}\}`)
var rawPlaceholderPattern = regexp.MustCompile(`\{\{[^}]+\}\}`)

func cleanupUnsubstitutedFilters(sql string) string {
	// First, remove full filter conditions like "AND year = {{year}}"
	sql = filterConditionPattern.ReplaceAllString(sql, "")
	// Then remove any remaining raw placeholders
	sql = rawPlaceholderPattern.ReplaceAllString(sql, "")
	return sql
}

// buildSQLList creates a comma-separated list for use in SQL IN clauses
// Each value is escaped (single quotes doubled) but NOT wrapped in quotes
// because substituteFilters will add the outer quotes
// Usage in SQL: WHERE district IN ({{district_list}})
func buildSQLList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	escaped := make([]string, len(values))
	for i, v := range values {
		escaped[i] = strings.ReplaceAll(v, "'", "''")
	}
	return strings.Join(escaped, "', '")
}

// executeChoroplethSQLData runs a choropleth custom SQL query and transforms it.
func executeChoroplethSQLData(
	ctx context.Context,
	component Component,
	filters ReportFilters,
	report *Report,
) (ChoroplethData, error) {
	// Validate the SQL first
	if err := ValidateAdvancedSQL(component.SQL); err != nil {
		return ChoroplethData{}, err
	}

	// Build filter map for substitution
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
	// Apply complex filter placeholders FIRST ({{year_filter}}, {{month_filter}}, etc.)
	// Must run before substituteFilters which would destroy these placeholders
	// Note: {{filter.*}} placeholders are handled by interpolateFilterPlaceholders (without SQL quoting)
	sql, args := applyFiltersToSQL(component.SQL, filters, report)

	// Then substitute simple filter placeholders ({{district}}, {{month}}, etc.)
	sql = substituteFilters(sql, filterMap)

	// Clean up unsubstituted filter conditions
	sql = cleanupUnsubstitutedFilters(sql)

	// Execute query with timeout
	tableData, err := executeQueryWithContext(ctx, DB, sql, args...)
	if err != nil {
		return ChoroplethData{}, err
	}

	// Transform using ChoroplethTransformer with column mapping
	transformedData, err := transformData(component.Type, tableData, &component)
	if err != nil {
		return ChoroplethData{}, err
	}

	if choroData, ok := transformedData.(ChoroplethData); ok {
		return choroData, nil
	}

	return ChoroplethData{}, fmt.Errorf("unexpected choropleth SQL transform result type %T", transformedData)
}

func processChoroplethSQLComponent(
	ctx context.Context,
	component Component,
	filters ReportFilters,
	report *Report,
	result *ComponentResult,
) {
	choroData, err := executeChoroplethSQLData(ctx, component, filters, report)
	if err != nil {
		logError("Error processing choropleth SQL for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Store the transformed choropleth data
	result.Data.GeoJSONPath = choroData.GeoJSONPath
	result.Data.DistrictValues = choroData.DistrictValues
	result.Data.ColorScheme = choroData.ColorScheme
	result.Data.BinLabels = choroData.BinLabels
	result.Data.BinEdges = choroData.BinEdges
	result.Data.LegendTitle = choroData.LegendTitle
}

// processAdvancedSQLComponent handles table_advanced components with raw SQL
func processAdvancedSQLComponent(
	ctx context.Context,
	component Component,
	filters ReportFilters,
	report *Report,
	result *ComponentResult,
) {
	// Validate the SQL first
	if err := ValidateAdvancedSQL(component.SQL); err != nil {
		logError("Invalid SQL for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Build filter map for substitution
	filterMap := map[string]string{}
	if len(filters.Districts) > 0 {
		filterMap["district"] = filters.Districts[0] // Single district for backward compatibility
		// Build comma-separated list for IN clause: {{district_list}}
		filterMap["district_list"] = buildSQLList(filters.Districts)
	}
	if filters.Year != "" {
		filterMap["year"] = filters.Year
	}
	if filters.Month != "" {
		monthVal := ConvertMonthFilterToText(filters.Month, report)
		filterMap["month"] = monthVal
		// Build list for IN clause: {{month_list}}
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
		filterMap["region"] = filters.Regions[0] // Single region for backward compatibility
		filterMap["region_list"] = buildSQLList(filters.Regions)
	}
	if len(filters.Facilities) > 0 {
		filterMap["facility"] = filters.Facilities[0] // Single facility for backward compatibility
		filterMap["facility_list"] = buildSQLList(filters.Facilities)
	}
	// Apply complex filter placeholders FIRST ({{district_filter}}, {{year_filter}}, {{month_filter}})
	// Must run before substituteFilters which would destroy these placeholders
	// Note: {{filter.*}} placeholders are handled by interpolateFilterPlaceholders (without SQL quoting)
	sql, args := applyFiltersToSQL(component.SQL, filters, report)

	// Then substitute simple filter placeholders ({{district}}, {{month}}, etc.)
	sql = substituteFilters(sql, filterMap)
	sql = interpolateFilterPlaceholders(sql, filters)

	// Clean up any unsubstituted filter conditions (e.g., when user doesn't select a filter)
	sql = cleanupUnsubstitutedFilters(sql)

	// Wrap in subquery with LIMIT for safety
	safeSql := fmt.Sprintf("SELECT * FROM (%s) AS _advanced_query LIMIT %d", sql, AdvancedSQLMaxRows)

	// Execute with timeout and parameterized args
	tableData, err := executeQueryWithContext(ctx, DB, safeSql, args...)
	if err != nil {
		logError("Error executing advanced SQL for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// If firstRowIsHeader is set, promote the first data row to column headers
	if component.FirstRowIsHeader && len(tableData.Rows) > 0 {
		firstRow := tableData.Rows[0]
		newHeaders := make([]string, len(tableData.Headers))
		for i, val := range firstRow {
			if i >= len(tableData.Headers) {
				break
			}
			if s, ok := val.(string); ok && s != "" {
				newHeaders[i] = s
			} else {
				newHeaders[i] = tableData.Headers[i] // fall back to SQL column name
			}
		}
		tableData.Headers = newHeaders
		tableData.Rows = tableData.Rows[1:]
	}

	// If pivot config is set, apply server-side pivot transformation
	if component.Pivot != nil {
		pivoted, err := applyPivot(tableData, component.Pivot)
		if err != nil {
			logError("Pivot error for component %s: %v", component.Title, err)
			result.Error = err
			result.Data = Data{}
			return
		}
		tableData = pivoted
	}

	// Convert to table format
	result.Data = Data{
		Headers: tableData.Headers,
		Rows:    tableData.Rows,
	}

	// Pass through YAML-provided column formats so the frontend renders cells
	// with the author's requested decimal precision.
	if len(component.Data.ColumnFormats) > 0 {
		result.Data.ColumnFormats = component.Data.ColumnFormats
	}
}

// processChartSQLComponent handles chart components (bar, line, bar_line, pie, pyramid) with raw SQL.
// Follows the same pattern as processAdvancedSQLComponent but routes results through ChartTransformer.
func processChartSQLComponent(
	ctx context.Context,
	component Component,
	filters ReportFilters,
	report *Report,
	result *ComponentResult,
) {
	// Validate the SQL first
	if err := ValidateAdvancedSQL(component.SQL); err != nil {
		logError("Invalid SQL for chart component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Build filter map for substitution (same as processAdvancedSQLComponent)
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
	// Apply complex filter placeholders FIRST ({{district_filter}}, {{year_filter}}, etc.)
	// Note: {{filter.*}} placeholders are handled by interpolateFilterPlaceholders (without SQL quoting)
	sql, args := applyFiltersToSQL(component.SQL, filters, report)

	// Then substitute simple filter placeholders ({{district}}, {{month}}, etc.)
	sql = substituteFilters(sql, filterMap)
	sql = interpolateFilterPlaceholders(sql, filters)

	// Clean up any unsubstituted filter conditions
	sql = cleanupUnsubstitutedFilters(sql)

	// Wrap in subquery with LIMIT for safety (charts shouldn't have too many rows)
	safeSql := fmt.Sprintf("SELECT * FROM (%s) AS _chart_query LIMIT %d", sql, ChartSQLMaxRows)

	// Execute with timeout and parameterized args
	tableData, err := executeQueryWithContext(ctx, DB, safeSql, args...)
	if err != nil {
		logError("Error executing chart SQL for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Validate column shape
	minCols := 2
	if component.Type == "pie" {
		minCols = 1
	} else if component.Type == "map" {
		minCols = 4
	}
	if len(tableData.Headers) < minCols {
		logError("Chart SQL for %s returned %d columns, need at least %d", component.Title, len(tableData.Headers), minCols)
		result.Error = fmt.Errorf("chart SQL must return at least %d columns, got %d", minCols, len(tableData.Headers))
		result.Data = Data{}
		return
	}

	// Transform through ChartTransformer
	transformedData, err := transformData(component.Type, tableData, &component)
	if err != nil {
		logError("Error transforming chart data for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Store chart data
	if chartData, ok := transformedData.(ChartData); ok {
		// Pyramid: negate first dataset so it mirrors left (matches structured-query path)
		if component.Type == "pyramid" && len(chartData.Datasets) != 2 {
			logWarn("Pyramid component %q has %d datasets, expected exactly 2 (left and right)", component.Title, len(chartData.Datasets))
		}
		if component.Type == "pyramid" && len(chartData.Datasets) > 0 {
			negated := make([]interface{}, len(chartData.Datasets[0].Data))
			for i, v := range chartData.Datasets[0].Data {
				if f, ok := toFloat64(v); ok {
					negated[i] = -f
				} else {
					negated[i] = v
				}
			}
			chartData.Datasets[0].Data = negated
		}
		result.Data.Labels = chartData.Labels
		result.Data.Datasets = chartData.Datasets
		// Preserve legend config from YAML component data
		result.Data.Legend = component.Data.Legend
	} else if mapData, ok := transformedData.(MapData); ok {
		result.Data.Center = mapData.Center
		result.Data.Zoom = mapData.Zoom
		result.Data.Markers = mapData.Markers
		result.Data.Unmapped = mapData.Unmapped
		result.Data.ValueLabel = mapData.ValueLabel
	}
}

// processTextSQLComponent handles text components with raw SQL (KPI cards).
// Executes the SQL, then substitutes {{value}} and named placeholders in content.
// buildTextSQLFilterMap builds the simple-placeholder substitution map used by
// raw-SQL text components. Extracted so previous-period comparison queries can
// rebuild it with shifted filter values.
func buildTextSQLFilterMap(filters ReportFilters, report *Report) map[string]string {
	filterMap := map[string]string{}
	if len(filters.Districts) > 0 {
		filterMap["district"] = filters.Districts[0]
		filterMap["filter.district"] = filters.Districts[0]
		filterMap["district_list"] = buildSQLList(filters.Districts)
	}
	if filters.Year != "" {
		filterMap["year"] = filters.Year
		filterMap["filter.year"] = filters.Year
	}
	if filters.Month != "" {
		monthVal := ConvertMonthFilterToText(filters.Month, report)
		filterMap["month"] = monthVal
		filterMap["filter.month"] = filters.Month
		months := strings.Split(monthVal, ",")
		filterMap["month_list"] = buildSQLList(months)
	}
	if filters.Quarter != "" {
		filterMap["quarter"] = filters.Quarter
		filterMap["filter.quarter"] = filters.Quarter
		quarters := strings.Split(filters.Quarter, ",")
		filterMap["quarter_list"] = buildSQLList(quarters)
	}
	if filters.Week != "" {
		filterMap["week"] = filters.Week
		filterMap["filter.week"] = filters.Week
		weeks := strings.Split(filters.Week, ",")
		filterMap["week_list"] = buildSQLList(weeks)
	}
	if len(filters.Regions) > 0 {
		filterMap["region"] = filters.Regions[0]
		filterMap["filter.region"] = filters.Regions[0]
		filterMap["region_list"] = buildSQLList(filters.Regions)
	}
	if len(filters.Facilities) > 0 {
		filterMap["facility"] = filters.Facilities[0]
		filterMap["filter.facility"] = filters.Facilities[0]
		filterMap["facility_list"] = buildSQLList(filters.Facilities)
	}
	return filterMap
}

// evaluateThresholds tests a numeric value against an ordered list of ThresholdRule
// conditions and returns the Color of the first matching rule, or "" if none match.
// Supported operators: >=  <=  >  <  ==  !=
func evaluateThresholds(value float64, rules []ThresholdRule) string {
	for _, rule := range rules {
		cond := strings.TrimSpace(rule.Condition)
		var op, rest string
		for _, candidate := range []string{">=", "<=", "!=", ">", "<", "=="} {
			if strings.HasPrefix(cond, candidate) {
				op = candidate
				rest = strings.TrimSpace(cond[len(candidate):])
				break
			}
		}
		if op == "" {
			logWarn("ThresholdRule has unparseable condition %q — skipping", rule.Condition)
			continue
		}
		target, err := strconv.ParseFloat(rest, 64)
		if err != nil {
			logWarn("ThresholdRule condition %q has non-numeric operand — skipping", rule.Condition)
			continue
		}
		var match bool
		switch op {
		case ">=":
			match = value >= target
		case "<=":
			match = value <= target
		case ">":
			match = value > target
		case "<":
			match = value < target
		case "==":
			match = value == target
		case "!=":
			match = value != target
		}
		if match {
			return rule.Color
		}
	}
	return ""
}

// applyInfoboxThresholds evaluates any configured threshold rules against the
// SQL result row and, if a rule matches, sets result.AccentColor.
// thresholdColumn names the result column to test; if empty the first column is used.
func applyInfoboxThresholds(component Component, tableData TableData, result *ComponentResult) {
	if len(component.Thresholds) == 0 || len(tableData.Rows) == 0 || len(tableData.Rows[0]) == 0 {
		return
	}
	// Determine which column value to test
	rawVal := ""
	if component.ThresholdColumn != "" {
		for i, h := range tableData.Headers {
			if h == component.ThresholdColumn && i < len(tableData.Rows[0]) {
				rawVal = fmt.Sprintf("%v", tableData.Rows[0][i])
				break
			}
		}
	} else {
		rawVal = fmt.Sprintf("%v", tableData.Rows[0][0])
	}
	if rawVal == "" {
		return
	}
	numVal, err := strconv.ParseFloat(strings.TrimSpace(rawVal), 64)
	if err != nil {
		return // non-numeric column — thresholds don't apply
	}
	if color := evaluateThresholds(numVal, component.Thresholds); color != "" {
		result.AccentColor = color
	}
}

// processInfoboxSQLComponent handles infobox components with raw SQL.
// Executes the SQL, then substitutes {{value}} and named column placeholders
// into InfoboxBody. The resolved body is returned via result.Content so the
// result-assembly loop can write it back to component.InfoboxBody.
func processInfoboxSQLComponent(
	ctx context.Context,
	component Component,
	filters ReportFilters,
	report *Report,
	result *ComponentResult,
) {
	if err := ValidateAdvancedSQL(component.SQL); err != nil {
		logError("Invalid SQL for infobox component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	filterMap := buildTextSQLFilterMap(filters, report)
	sql, args := applyFiltersToSQL(component.SQL, filters, report)
	sql = substituteFilters(sql, filterMap)
	sql = interpolateFilterPlaceholders(sql, filters)
	sql = cleanupUnsubstitutedFilters(sql)

	tableData, err := executeQueryWithContext(ctx, DB, sql, args...)
	if err != nil {
		logError("Error executing SQL for infobox component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	content := component.InfoboxBody
	if len(tableData.Rows) > 0 && len(tableData.Rows[0]) > 0 {
		valueMap := make(map[string]string)
		for i, header := range tableData.Headers {
			if i < len(tableData.Rows[0]) {
				valueMap[header] = html.EscapeString(fmt.Sprintf("%v", tableData.Rows[0][i]))
			}
		}
		for name, val := range valueMap {
			placeholder := "{{" + name + "}}"
			content = strings.ReplaceAll(content, placeholder, val)
		}
		if strings.Contains(content, "{{value}}") {
			firstValue := html.EscapeString(fmt.Sprintf("%v", tableData.Rows[0][0]))
			content = strings.ReplaceAll(content, "{{value}}", firstValue)
		}
	}
	// Substitute any {{filter.*}} placeholders remaining in the body text
	content = interpolateFilterPlaceholders(content, filters)

	// Evaluate threshold rules to determine dynamic accent color
	applyInfoboxThresholds(component, tableData, result)

	result.Content = content
	result.Data = Data{}
}

func processTextSQLComponent(
	ctx context.Context,
	component Component,
	filters ReportFilters,
	report *Report,
	result *ComponentResult,
) {
	if err := ValidateAdvancedSQL(component.SQL); err != nil {
		logError("Invalid SQL for text component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	filterMap := buildTextSQLFilterMap(filters, report)

	// Apply complex filter placeholders FIRST
	sql, args := applyFiltersToSQL(component.SQL, filters, report)

	// Then substitute simple filter placeholders
	sql = substituteFilters(sql, filterMap)
	sql = interpolateFilterPlaceholders(sql, filters)

	// Clean up any unsubstituted filter conditions
	sql = cleanupUnsubstitutedFilters(sql)

	// Execute query
	tableData, err := executeQueryWithContext(ctx, DB, sql, args...)
	if err != nil {
		logError("Error executing text SQL for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Substitute placeholders in content
	content := component.Content
	if len(tableData.Rows) > 0 && len(tableData.Rows[0]) > 0 {
		// Build a map of header names to HTML-escaped values from the first row
		valueMap := make(map[string]string)
		for i, header := range tableData.Headers {
			if i < len(tableData.Rows[0]) {
				valueMap[header] = html.EscapeString(fmt.Sprintf("%v", tableData.Rows[0][i]))
			}
		}

		// Replace named placeholders {{alias_name}} with corresponding values first,
		// so that an explicit "value" column (from resultAlias: value) takes precedence
		// over the backward-compat first-column fallback below.
		for name, val := range valueMap {
			placeholder := "{{" + name + "}}"
			content = strings.ReplaceAll(content, placeholder, val)
		}

		// Replace {{value}} with first column as fallback (backward compatibility for
		// components that have no explicit alias named "value" in their query result).
		if strings.Contains(content, "{{value}}") {
			firstValue := html.EscapeString(fmt.Sprintf("%v", tableData.Rows[0][0]))
			content = strings.ReplaceAll(content, "{{value}}", firstValue)
		}

		// Warn about unmatched placeholders before removing them
		unmatched := rawPlaceholderPattern.FindAllString(content, -1)
		if len(unmatched) > 0 {
			logWarn("Component %q has unmatched placeholders: %v", component.Title, unmatched)
		}
		content = rawPlaceholderPattern.ReplaceAllString(content, "")
	}

	result.Content = content
	result.Data = Data{}

	// Handle previous-period comparison for raw-SQL KPIs. The same SQL is
	// re-executed with shifted filter values.
	if component.Comparison != "" && len(tableData.Rows) > 0 && len(tableData.Rows[0]) > 0 {
		prevFilters := calculatePreviousPeriodFilters(filters, component.Comparison)
		if prevFilters == nil {
			logDebug("Could not calculate previous period for %q (year=%s, month=%s)",
				component.Title, filters.Year, filters.Month)
			return
		}
		prevFilterMap := buildTextSQLFilterMap(*prevFilters, report)
		prevSQL, prevArgs := applyFiltersToSQL(component.SQL, *prevFilters, report)
		prevSQL = substituteFilters(prevSQL, prevFilterMap)
		prevSQL = cleanupUnsubstitutedFilters(prevSQL)

		prevTableData, err := executeQueryWithContext(ctx, DB, prevSQL, prevArgs...)
		if err != nil {
			logError("Comparison SQL error for %q: %v", component.Title, err)
			return
		}
		if len(prevTableData.Rows) == 0 || len(prevTableData.Rows[0]) == 0 {
			logDebug("Comparison query returned no rows for %q", component.Title)
			return
		}
		currentVal, currentOK := toFloat64(tableData.Rows[0][0])
		prevVal, prevOK := toFloat64(prevTableData.Rows[0][0])
		if currentOK && prevOK {
			result.Comparison = calculateComparison(currentVal, prevVal)
			logDebug("Comparison for %q: current=%.2f, previous=%.2f, direction=%s, change=%.2f%%",
				component.Title, currentVal, prevVal, result.Comparison.Direction, result.Comparison.PercentageChange)
		}
	}

	// Handle static target comparison for raw-SQL KPIs
	if component.Target != nil && len(tableData.Rows) > 0 && len(tableData.Rows[0]) > 0 {
		currentVal, currentOK := toFloat64(tableData.Rows[0][0])
		if currentOK {
			result.TargetComparison = calculateTargetComparison(currentVal, component.Target.Value, component.Target.Label)
			logDebug("Target comparison for %q: current=%.2f, target=%.2f, direction=%s, change=%.2f%%",
				component.Title, currentVal, component.Target.Value, result.TargetComparison.Direction, result.TargetComparison.PercentageChange)
		}
	}
}

// processComponent executes the query for a single component in a goroutine
func processComponent(
	ctx context.Context,
	sectionIdx int,
	componentIdx int,
	component Component,
	filters ReportFilters,
	reportFilters []string,
	report *Report,
	result *ComponentResult,
) {
	// Panic recovery - prevents one component crash from freezing all processing
	defer func() {
		if r := recover(); r != nil {
			logError("PANIC in component [%d][%d] %q: %v", sectionIdx, componentIdx, component.Title, r)
			result.Error = fmt.Errorf("component panicked: %v", r)
			result.Data = Data{}
		}
	}()

	// Check if context is already cancelled (timeout from another component)
	select {
	case <-ctx.Done():
		logWarn("Component [%d][%d] %q skipped: %v", sectionIdx, componentIdx, component.Title, ctx.Err())
		result.Error = ctx.Err()
		result.Data = Data{}
		return
	default:
		// Continue processing
	}

	// Initialize result indices
	result.SectionIdx = sectionIdx
	result.ComponentIdx = componentIdx

	// KPI components synthesize their HTML from structured fields, then
	// process exactly like text-with-query/SQL components. Mutating the local
	// copy here is safe because component is passed by value.
	if component.Type == "kpi" {
		component.Content = synthesizeKpiContent(component)
		component.Type = "text"
	}

	// Check if this is a faceted choropleth component
	if component.Facet != nil && component.Type == "choropleth" {
		processFacetedComponent(ctx, component, filters, reportFilters, report, result)
		return
	}

	// Handle choropleth type with custom SQL (non-faceted)
	if component.Type == "choropleth" && component.SQL != "" {
		processChoroplethSQLComponent(ctx, component, filters, report, result)
		return
	}

	// Apply periodLimit expansion for raw SQL components (chart types only).
	// Only expand when the component uses the query-builder (component.Query != nil),
	// because query-builder components have an explicit periodLimit and GroupBy that
	// define which time dimension to expand.
	//
	// Raw SQL components (component.Query == nil) manage their own period range
	// directly in the SQL (e.g. via CTEs or explicit filter placeholders). Auto-
	// expanding their filters would silently change a single-period snapshot chart
	// (e.g. "cases by region for Q2 2025") into a multi-period aggregate — wrong for
	// any snapshot report. Raw SQL trend charts implement period logic in the SQL itself.
	sqlFilters := filters
	if component.SQL != "" && component.Query != nil && filters.Year != "" {
		effectivePeriodLimit := component.Query.PeriodLimit
		if effectivePeriodLimit > 0 {
			// Query-builder components have explicit GroupBy, so we can infer period type.
			var groupBy []GroupBy
			switch {
			case strings.Contains(component.SQL, "{{month_filter}}"):
				groupBy = []GroupBy{{Field: "month", Format: "month"}}
			case strings.Contains(component.SQL, "{{quarter_filter}}"):
				groupBy = []GroupBy{{Field: "quarter", Format: "quarter"}}
			case strings.Contains(component.SQL, "{{week_filter}}"):
				groupBy = []GroupBy{{Field: "week", Format: "week"}}
			}
			tempQuery := &Query{
				PeriodLimit: effectivePeriodLimit,
				GroupBy:     groupBy,
				Table:       "", // not needed for expansion
			}
			sqlFilters = expandPeriodFilters(ctx, tempQuery, filters, report)
		}
	}

	// Handle table_advanced type with raw SQL
	if component.Type == "table_advanced" && component.SQL != "" {
		processAdvancedSQLComponent(ctx, component, sqlFilters, report, result)
		return
	}

	// Handle chart types with raw SQL (bar, line, bar_line, pie, pyramid)
	if (isChartType(component.Type) || component.Type == "map") && component.SQL != "" {
		processChartSQLComponent(ctx, component, sqlFilters, report, result)
		return
	}

	// Handle text type with raw SQL (KPI cards with {{value}} placeholders)
	if component.Type == "text" && component.SQL != "" {
		processTextSQLComponent(ctx, component, filters, report, result)
		return
	}

	// Handle infobox type with raw SQL (substitutes {{value}} and named placeholders into InfoboxBody)
	if component.Type == "infobox" && component.SQL != "" {
		processInfoboxSQLComponent(ctx, component, filters, report, result)
		return
	}

	var sqlQuery string
	var err error

	// Generate SQL from structured query
	if component.Query != nil {
		sqlQuery, err = BuildSQL(*component.Query, component.Type, component.Query.Table, reportFilters)
		if err != nil {
			logError("Error building SQL for component %s: %v", component.Title, err)
			result.Error = err
			result.Data = Data{}
			return
		}
		// Inject custom filter placeholders into the SQL
		sqlQuery = injectCustomFilterPlaceholders(sqlQuery, report.CustomFilters, component.Query.Table)
	} else {
		// Static text with content but no query is valid — render as-is
		if component.Type == "text" && component.Content != "" {
			result.Content = component.Content
			result.Data = Data{}
			return
		}
		// Infobox is always query-free
		if component.Type == "infobox" {
			result.Data = Data{}
			return
		}
		// Query is required for all other component types
		logWarn("Component %s has no query defined", component.Title)
		result.Error = fmt.Errorf("component %s requires a query", component.Title)
		result.Data = Data{}
		return
	}

	// Apply periodLimit expansion if configured (must happen before applyFiltersToSQL)
	// Default to 4 periods for bar, line, bar_line, and table components if not explicitly set
	// Skip expansion when Year is empty (user selected "All years") — respect the explicit choice
	componentFilters := filters
	if component.Query != nil && filters.Year != "" {
		effectivePeriodLimit := component.Query.PeriodLimit
		if effectivePeriodLimit == 0 {
			switch component.Type {
			case "bar", "line", "bar_line", "table", "pyramid":
				effectivePeriodLimit = 4
			}
		}
		if effectivePeriodLimit > 0 {
			// Create a temporary query with the effective period limit for expansion
			tempQuery := *component.Query
			tempQuery.PeriodLimit = effectivePeriodLimit
			componentFilters = expandPeriodFilters(ctx, &tempQuery, filters, report)
		}
	}

	// Apply filters to the SQL query (now returns SQL and parameters)
	filteredSQL, args := applyFiltersToSQL(sqlQuery, componentFilters, report)
	LogFilteredSQL(filteredSQL, filters)

	// Debug: Log SQL for bar/line/bar_line/pyramid components
	if component.Type == "bar" || component.Type == "line" || component.Type == "bar_line" || component.Type == "pyramid" {
		logDebug("Component %q (%s) TimeColumns: year=%q, month=%q", component.Title, component.Type, report.TimeColumns.Year, report.TimeColumns.Month)
		logDebug("Component %q (%s) SQL: %s", component.Title, component.Type, filteredSQL)
		logDebug("Component %q args: %v", component.Title, args)
	}

	tableData, err := executeQueryWithContext(ctx, DB, filteredSQL, args...)
	if err != nil {
		logError("Error executing query for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Debug: Log row count for bar/line/bar_line/pyramid components
	if component.Type == "bar" || component.Type == "line" || component.Type == "bar_line" || component.Type == "pyramid" {
		logDebug("Component %q returned %d rows, headers: %v", component.Title, len(tableData.Rows), tableData.Headers)
	}

	// Derive per-column decimal hints from the structured query so the frontend
	// (and text substitution below) renders exactly the author's roundTo.
	columnFormats := buildColumnFormatsFromQuery(component.Query)

	// Special handling for text/infobox components with query - replace placeholders
	// Supports: {{value}} (first column, backward compat) and {{alias_name}} (by column header)
	if (component.Type == "text" || component.Type == "infobox") && len(tableData.Rows) > 0 && len(tableData.Rows[0]) > 0 {
		content := component.Content
		if component.Type == "infobox" {
			content = component.InfoboxBody
		}

		// Build a map of header names to HTML-escaped values from the first row.
		// When a column has an explicit decimals hint, format numerically so
		// PG's full NUMERIC precision (e.g. "83.2000000000000000") doesn't leak
		// into the rendered card.
		valueMap := make(map[string]string)
		for i, header := range tableData.Headers {
			if i < len(tableData.Rows[0]) {
				valueMap[header] = html.EscapeString(formatSubstitutionValue(tableData.Rows[0][i], columnFormats[header]))
			}
		}

		// Replace named placeholders {{alias_name}} with corresponding values first,
		// so that an explicit "value" column (from resultAlias: value) takes precedence
		// over the backward-compat first-column fallback below.
		for name, val := range valueMap {
			placeholder := "{{" + name + "}}"
			content = strings.ReplaceAll(content, placeholder, val)
		}

		// Replace {{value}} with first column as fallback (backward compatibility for
		// components that have no explicit alias named "value" in their query result).
		if strings.Contains(content, "{{value}}") {
			firstHeader := ""
			if len(tableData.Headers) > 0 {
				firstHeader = tableData.Headers[0]
			}
			firstValue := html.EscapeString(formatSubstitutionValue(tableData.Rows[0][0], columnFormats[firstHeader]))
			content = strings.ReplaceAll(content, "{{value}}", firstValue)
		}

		// Warn about unmatched placeholders before removing them
		unmatched := rawPlaceholderPattern.FindAllString(content, -1)
		if len(unmatched) > 0 {
			logWarn("Component %q has unmatched placeholders: %v", component.Title, unmatched)
		}
		content = rawPlaceholderPattern.ReplaceAllString(content, "")

		result.Content = content

		// Evaluate threshold rules (query-builder infobox path)
		if component.Type == "infobox" {
			applyInfoboxThresholds(component, tableData, result)
		}

		// Handle comparison if configured
		if component.Comparison != "" && component.Query != nil {
			logDebug("Comparison requested for %q: type=%s", component.Title, component.Comparison)
			// Calculate previous period filters
			prevFilters := calculatePreviousPeriodFilters(filters, component.Comparison)
			if prevFilters != nil {
				logDebug("Previous filters calculated: year=%s, month=%s", prevFilters.Year, prevFilters.Month)
				// Build SQL for comparison query (same query, different filters)
				compSQL, err := BuildSQL(*component.Query, component.Type, component.Query.Table, reportFilters)
				if err != nil {
					logError("Comparison SQL build error for %q: %v", component.Title, err)
				} else {
					// Inject custom filter placeholders
					compSQL = injectCustomFilterPlaceholders(compSQL, report.CustomFilters, component.Query.Table)

					// Apply previous period filters
					compFilteredSQL, compArgs := applyFiltersToSQL(compSQL, *prevFilters, report)
					logDebug("Comparison query: %s args: %v", compFilteredSQL, compArgs)

					// Execute comparison query
					compTableData, err := executeQueryWithContext(ctx, DB, compFilteredSQL, compArgs...)
					if err != nil {
						logError("Comparison query error for %q: %v", component.Title, err)
					} else if len(compTableData.Rows) > 0 && len(compTableData.Rows[0]) > 0 {
						// Parse current and previous values
						currentVal, currentOK := toFloat64(tableData.Rows[0][0])
						prevVal, prevOK := toFloat64(compTableData.Rows[0][0])
						logDebug("Comparison values: current=%v (ok=%v), previous=%v (ok=%v)",
							tableData.Rows[0][0], currentOK, compTableData.Rows[0][0], prevOK)

						if currentOK && prevOK {
							result.Comparison = calculateComparison(currentVal, prevVal)
							logDebug("Comparison for %q: current=%.2f, previous=%.2f, direction=%s, change=%.2f%%",
								component.Title, currentVal, prevVal, result.Comparison.Direction, result.Comparison.PercentageChange)
						}
					} else {
						logDebug("Comparison query returned no rows for %q", component.Title)
					}
				}
			} else {
				logDebug("Could not calculate previous period for %q (year=%s, month=%s)",
					component.Title, filters.Year, filters.Month)
			}
		}

		// Handle target comparison if configured
		if component.Target != nil {
			currentVal, currentOK := toFloat64(tableData.Rows[0][0])
			if currentOK {
				result.TargetComparison = calculateTargetComparison(currentVal, component.Target.Value, component.Target.Label)
				logDebug("Target comparison for %q: current=%.2f, target=%.2f, direction=%s, change=%.2f%%",
					component.Title, currentVal, component.Target.Value, result.TargetComparison.Direction, result.TargetComparison.PercentageChange)
			}
		}

		return
	}

	transformedData, err := transformData(component.Type, tableData, &component)
	if err != nil {
		logError("Error transforming data for component %s: %v", component.Title, err)
		result.Error = err
		result.Data = Data{}
		return
	}

	// Create a new Data and assign the transformed data to it
	switch data := transformedData.(type) {
	case ChartData:
		// Debug: Log labels count for bar/line/bar_line/pyramid
		if component.Type == "bar" || component.Type == "line" || component.Type == "bar_line" || component.Type == "pyramid" {
			logDebug("Component %q transformed: %d labels, %d datasets", component.Title, len(data.Labels), len(data.Datasets))
		}
		// Pyramid: warn if not exactly 2 datasets, negate the first dataset so it mirrors left
		if component.Type == "pyramid" && len(data.Datasets) != 2 {
			logWarn("Pyramid component %q has %d datasets, expected exactly 2 (left and right)", component.Title, len(data.Datasets))
		}
		if component.Type == "pyramid" && len(data.Datasets) > 0 {
			negated := make([]interface{}, len(data.Datasets[0].Data))
			for i, v := range data.Datasets[0].Data {
				if f, ok := toFloat64(v); ok {
					negated[i] = -f
				} else {
					negated[i] = v
				}
			}
			data.Datasets[0].Data = negated
		}
		result.Data.Labels = data.Labels
		result.Data.Datasets = data.Datasets
		// Preserve legend config from YAML
		result.Data.Legend = component.Data.Legend
	case MapData:
		result.Data.Center = data.Center
		result.Data.Zoom = data.Zoom
		result.Data.Markers = data.Markers
	case TableData:
		result.Data.Headers = data.Headers
		result.Data.Rows = data.Rows
	case ChoroplethData:
		result.Data.GeoJSONPath = data.GeoJSONPath
		result.Data.DistrictValues = data.DistrictValues
		result.Data.ColorScheme = data.ColorScheme
		result.Data.BinLabels = data.BinLabels
		result.Data.BinEdges = data.BinEdges
		result.Data.LegendTitle = data.LegendTitle
	}

	// Attach per-column formatting hints so the frontend renders the author's
	// roundTo precisely (tables, tooltips, axis ticks, bar labels).
	if len(columnFormats) > 0 {
		result.Data.ColumnFormats = columnFormats
	}
}

// buildColumnFormatsFromQuery derives a ColumnFormats map from a structured
// Query's aggregations and calculations. Returns nil when no column has an
// explicit roundTo. COUNT / COUNT_DISTINCT never get an entry because they
// always return integers and the server ignores their roundTo.
func buildColumnFormatsFromQuery(q *Query) map[string]ColumnFormat {
	if q == nil {
		return nil
	}
	formats := make(map[string]ColumnFormat)
	for _, agg := range q.Aggregations {
		if agg.RoundTo == nil {
			continue
		}
		fn := strings.ToUpper(agg.Function)
		if fn == "COUNT" || fn == "COUNT_DISTINCT" {
			continue
		}
		if agg.Alias == "" {
			continue
		}
		d := *agg.RoundTo
		formats[agg.Alias] = ColumnFormat{Decimals: &d}
	}
	for _, calc := range q.Calculate {
		if calc.RoundTo == nil {
			continue
		}
		name := calc.ResultAlias
		if name == "" {
			name = "value"
		}
		d := *calc.RoundTo
		formats[name] = ColumnFormat{Decimals: &d}
	}
	if len(formats) == 0 {
		return nil
	}
	return formats
}

// formatSubstitutionValue renders a DB value for a text-component placeholder.
// When the column has a decimals hint, the value is parsed as a number and
// formatted to exactly that precision (trailing zeros trimmed) so full PG
// NUMERIC precision never reaches the rendered card. Non-numeric or missing
// hint falls back to the default %v formatter.
func formatSubstitutionValue(v interface{}, fmtHint ColumnFormat) string {
	if fmtHint.Decimals == nil {
		return fmt.Sprintf("%v", v)
	}
	f, ok := toFloat64(v)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	decimals := *fmtHint.Decimals
	if decimals < 0 {
		decimals = 0
	}
	s := strconv.FormatFloat(f, 'f', decimals, 64)
	// Trim trailing zeros so roundTo=2 with value 83.2 renders "83.2" not "83.20".
	if decimals > 0 && strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// GetReportByID returns a specific report by ID by reading a YAML file
// ID format: "report-name" or "category/report-name"
// Validates ID to prevent path traversal attacks
func GetReportByID(id string, filters ReportFilters, parentCtx ...context.Context) *Report {
	// SECURITY: Validate report ID before any file operations
	if err := ValidateReportID(id); err != nil {
		logWarn("Invalid report ID %q: %v", id, err)
		return nil
	}

	// Construct file path from validated ID
	sanitizedPath := strings.ReplaceAll(id, "/", string(filepath.Separator))
	filePath := filepath.Join("configs", sanitizedPath+".yaml")

	// DEFENSE IN DEPTH: Verify resolved path is within configs directory
	absConfigPath, err := filepath.Abs("configs")
	if err != nil {
		logError("Error resolving configs directory: %v", err)
		return nil
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		logError("Error resolving file path %s: %v", filePath, err)
		return nil
	}

	// Ensure the resolved file path is within configs directory
	if !strings.HasPrefix(absFilePath, absConfigPath+string(filepath.Separator)) && absFilePath != absConfigPath {
		logWarn("Security violation: Report ID %q resolves outside configs directory", id)
		return nil
	}

	// Read the YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		logError("Error reading report file %s: %v", filePath, err)
		return nil
	}

	// Compute YAML hash for cache invalidation
	yamlHash := ComputeYAMLHash(data)

	var report Report
	if err := yaml.Unmarshal(data, &report); err != nil {
		logError("Error unmarshalling report file %s: %v", filePath, err)
		return nil
	}

	// Ensure section Components is never nil (empty sections are valid)
	for i := range report.Sections {
		if report.Sections[i].Components == nil {
			report.Sections[i].Components = []Component{}
		}
	}

	// Use request context as parent so client disconnects cancel DB work
	baseCtx := context.Background()
	if len(parentCtx) > 0 && parentCtx[0] != nil {
		baseCtx = parentCtx[0]
	}
	ctx, cancel := context.WithTimeout(baseCtx, ComponentQueryTimeout)
	defer cancel()

	// Enforce component count cap to prevent excessive goroutine spawning
	totalComponents := 0
	for _, section := range report.Sections {
		totalComponents += len(section.Components)
	}
	if totalComponents > MaxComponentsPerReport {
		logError("Report %s has %d components, exceeding cap of %d", id, totalComponents, MaxComponentsPerReport)
		return nil
	}

	// Create WaitGroup and pre-allocated result slices for concurrent execution
	var wg sync.WaitGroup
	results := make([][]ComponentResult, len(report.Sections))
	for i := range results {
		results[i] = make([]ComponentResult, len(report.Sections[i].Components))
	}

	// Track cache hits/misses for logging
	cacheHits := 0
	cacheMisses := 0
	var cacheMu sync.Mutex

	// Launch goroutines for each component to execute queries concurrently
	for i, section := range report.Sections {
		for j, component := range section.Components {
			wg.Add(1)
			go func(sectionIdx, componentIdx int, comp Component) {
				defer wg.Done()

				cacheFilters := filters
				cacheEnabled := report.Cache == nil || *report.Cache

				// Check cache first (unless this report opts out via cache: false)
				if cacheEnabled {
					if cached := componentCache.Get(id, sectionIdx, componentIdx, cacheFilters, yamlHash); cached != nil {
						results[sectionIdx][componentIdx] = *cached
						cacheMu.Lock()
						cacheHits++
						cacheMu.Unlock()
						return
					}
				}

				// Cache miss (or disabled) — deduplicate concurrent identical requests
				cacheMu.Lock()
				cacheMisses++
				cacheMu.Unlock()

				cacheKey := componentCache.buildCacheKey(id, sectionIdx, componentIdx, cacheFilters)
				v, _, _ := cacheFlight.Do(cacheKey, func() (interface{}, error) {
					var result ComponentResult
					processComponent(ctx, sectionIdx, componentIdx, comp, filters, report.Filters, &report, &result)
					if cacheEnabled && result.Error == nil {
						componentCache.Set(id, sectionIdx, componentIdx, cacheFilters, result, yamlHash)
					}
					return &result, nil
				})

				results[sectionIdx][componentIdx] = *v.(*ComponentResult)
			}(i, j, component)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Log cache performance
	totalComponents = cacheHits + cacheMisses
	if totalComponents > 0 {
		logDebug("Report %q cache stats: %d hits, %d misses (%.0f%% hit rate)",
			id, cacheHits, cacheMisses, float64(cacheHits)/float64(totalComponents)*100)
	}

	// Collect failed components for partial failure reporting
	var failedComponents []FailedComponent
	for i := range report.Sections {
		for j := range report.Sections[i].Components {
			result := results[i][j]
			if result.Error != nil {
				failedComponents = append(failedComponents, FailedComponent{
					SectionIndex:   i,
					ComponentIndex: j,
					Title:          report.Sections[i].Components[j].Title,
					Error:          result.Error.Error(),
				})
			}
		}
	}
	if len(failedComponents) > 0 {
		report.PartialFailure = true
		report.FailedComponents = failedComponents
		logDebug("Report %q: %d components failed", id, len(failedComponents))
	}

	// Record component execution metrics
	RecordComponentExecution(id, totalComponents, failedComponents)

	// Apply results to report structure
	for i := range report.Sections {
		for j := range report.Sections[i].Components {
			result := results[i][j]
			if result.Error != nil {
				report.Sections[i].Components[j].Data = Data{}
				report.Sections[i].Components[j].Error = result.Error.Error()
				// Clear content for text/infobox components with queries so unsubstituted
				// {{placeholders}} aren't rendered as literal text on the frontend
				if report.Sections[i].Components[j].Type == "text" && report.Sections[i].Components[j].Query != nil {
					report.Sections[i].Components[j].Content = ""
				}
				if report.Sections[i].Components[j].Type == "infobox" && report.Sections[i].Components[j].Query != nil {
					report.Sections[i].Components[j].InfoboxBody = ""
				}
			} else {
				report.Sections[i].Components[j].Data = result.Data
				if result.Content != "" {
					if report.Sections[i].Components[j].Type == "infobox" {
						report.Sections[i].Components[j].InfoboxBody = result.Content
					} else {
						report.Sections[i].Components[j].Content = result.Content
					}
				}
				// Apply dynamic accent color from threshold evaluation
				if result.AccentColor != "" {
					report.Sections[i].Components[j].AccentColor = result.AccentColor
				}
				// Add comparison data for text components
				if result.Comparison != nil {
					report.Sections[i].Components[j].Data.Comparison = result.Comparison
				}
				// Add target comparison data for text components
				if result.TargetComparison != nil {
					report.Sections[i].Components[j].Data.TargetComparison = result.TargetComparison
				}
				// Pass through reference lines for bar/line charts
				if len(report.Sections[i].Components[j].ReferenceLines) > 0 {
					report.Sections[i].Components[j].Data.ReferenceLines = report.Sections[i].Components[j].ReferenceLines
				}
				// Pass through trend line config for line charts
				if report.Sections[i].Components[j].TrendLine {
					report.Sections[i].Components[j].Data.TrendLine = true
					report.Sections[i].Components[j].Data.TrendLineColor = report.Sections[i].Components[j].TrendLineColor
				}
				// Pass through axis labels for charts
				if report.Sections[i].Components[j].AxisLabels != nil {
					report.Sections[i].Components[j].Data.AxisLabels = report.Sections[i].Components[j].AxisLabels
				}
				// Pass through y-axis limit for bar/line/bar_line charts
				if report.Sections[i].Components[j].YLimit != nil {
					report.Sections[i].Components[j].Data.YLimit = report.Sections[i].Components[j].YLimit
				}
			}
		}
	}

	interpolateReportPlaceholders(&report, filters)

	// Detect if all components returned empty data
	componentCount := 0
	emptyComponents := 0
	for _, section := range report.Sections {
		for _, comp := range section.Components {
			componentCount++
			if isComponentEmpty(&comp) {
				emptyComponents++
			}
		}
	}
	if componentCount > 0 && emptyComponents == componentCount {
		report.AllEmpty = true
		logDebug("Report %q: all %d components returned empty data", id, componentCount)
	}

	// Add custom filter column names to the filters list (for frontend filter bar rendering)
	for _, cf := range report.CustomFilters {
		report.Filters = append(report.Filters, cf.Column)
	}

	// Enforce canonical filter order: region → district → facility → year → quarter → month → week → custom
	report.Filters = sortFiltersCanonical(report.Filters)

	// Add filter definitions to the report (for dynamic frontend filter rendering)
	report.FilterDefinitions = GetFilterDefinitions(report.Filters, &report)

	report.GeneratedAt = time.Now().Format(time.RFC3339)
	return &report
}

// canonicalFilterOrder defines the standard order for filters in the UI.
// Filters appear: geographic (broad→narrow), then temporal (broad→narrow), then custom.
var canonicalFilterOrder = map[string]int{
	"region":   0,
	"district": 1,
	"facility": 2,
	"year":     3,
	"quarter":  4,
	"month":    5,
	"week":     6,
}

// sortFiltersCanonical reorders filters into a predictable standard order:
// region → district → facility → year → quarter → month → week → custom filters.
// Custom filters (not in the canonical list) are placed last, preserving their relative order.
func sortFiltersCanonical(filters []string) []string {
	sort.SliceStable(filters, func(i, j int) bool {
		oi, iKnown := canonicalFilterOrder[filters[i]]
		oj, jKnown := canonicalFilterOrder[filters[j]]
		if iKnown && jKnown {
			return oi < oj
		}
		if iKnown {
			return true
		}
		if jKnown {
			return false
		}
		return false // both custom: preserve original order via SliceStable
	})
	return filters
}

// isComponentEmpty checks whether a component has no meaningful data
func isComponentEmpty(comp *Component) bool {
	if comp.Type == "text" {
		// Static text (no query) with content is not empty
		if comp.Query == nil && comp.Content != "" {
			return false
		}
		// Text with query: if content was successfully substituted, no {{ remains
		// Unsubstituted {{placeholders}} means the query returned no rows
		if comp.Content != "" && !strings.Contains(comp.Content, "{{") {
			return false
		}
		return true
	}
	if comp.Type == "kpi" {
		// KPIs render via synthesizeKpiContent + text-with-query substitution.
		// Empty when content is missing or {{value}} placeholder remained unsubstituted.
		return comp.Content == "" || strings.Contains(comp.Content, "{{")
	}
	d := &comp.Data
	// Chart data
	if len(d.Labels) > 0 || len(d.Datasets) > 0 {
		return false
	}
	// Table data
	if len(d.Headers) > 0 && len(d.Rows) > 0 {
		return false
	}
	// Map data
	if len(d.Markers) > 0 || len(d.Center) > 0 {
		return false
	}
	// Choropleth data
	if len(d.DistrictValues) > 0 || d.GeoJSONPath != "" {
		return false
	}
	// Faceted data
	if len(d.Facets) > 0 {
		return false
	}
	return true
}

func interpolateReportPlaceholders(report *Report, filters ReportFilters) {
	report.Title = interpolateFilterPlaceholders(report.Title, filters)
	for i := range report.Sections {
		report.Sections[i].Title = interpolateFilterPlaceholders(report.Sections[i].Title, filters)
		for j := range report.Sections[i].Components {
			interpolateComponentPlaceholders(&report.Sections[i].Components[j], filters)
		}
	}
}

func interpolateComponentPlaceholders(component *Component, filters ReportFilters) {
	component.Title = interpolateFilterPlaceholders(component.Title, filters)
	component.InfoboxBody = interpolateFilterPlaceholders(component.InfoboxBody, filters)
	component.Content = interpolateFilterPlaceholders(component.Content, filters)
	for i := range component.Data.Datasets {
		component.Data.Datasets[i].Label = interpolateFilterPlaceholders(component.Data.Datasets[i].Label, filters)
	}
	for i := range component.SplitColumns {
		component.SplitColumns[i].Parent = interpolateFilterPlaceholders(component.SplitColumns[i].Parent, filters)
		for j := range component.SplitColumns[i].Children {
			component.SplitColumns[i].Children[j] = interpolateFilterPlaceholders(component.SplitColumns[i].Children[j], filters)
		}
	}
	component.HeaderTooltips = interpolateStringMap(component.HeaderTooltips, filters)
	component.HeaderFormulas = interpolateStringMap(component.HeaderFormulas, filters)
	component.AxisTooltips = interpolateStringMapValuesOnly(component.AxisTooltips, filters)
	component.AxisFormulas = interpolateStringMapValuesOnly(component.AxisFormulas, filters)
}

// interpolateStringMap returns a new map with both keys and values run through
// interpolateFilterPlaceholders. Used for maps keyed by column header where
// the headers themselves can contain {{filter.*}} placeholders.
func interpolateStringMap(in map[string]string, filters ReportFilters) map[string]string {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[interpolateFilterPlaceholders(k, filters)] = interpolateFilterPlaceholders(v, filters)
	}
	return out
}

// interpolateStringMapValuesOnly returns a new map with only values run through
// interpolateFilterPlaceholders. Used for maps with fixed keys (e.g., x/y).
func interpolateStringMapValuesOnly(in map[string]string, filters ReportFilters) map[string]string {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = interpolateFilterPlaceholders(v, filters)
	}
	return out
}

// interpolateFilterPlaceholders replaces {{filter.*}} placeholders in a string
// with the corresponding filter values. Returns the string unchanged if no
// placeholders are present.
// Supports arithmetic expressions: {{filter.year - 1}}, {{filter.month + 1}}, etc.
func interpolateFilterPlaceholders(text string, filters ReportFilters) string {
	if !strings.Contains(text, "{{filter.") {
		return text
	}

	replacements := map[string]string{
		"{{filter.year}}":               filters.Year,
		"{{filter.month}}":              filters.Month,
		"{{filter.week}}":               filters.Week,
		"{{filter.quarter}}":            filters.Quarter,
		"{{filter.district}}":           strings.Join(filters.Districts, ", "),
		"{{filter.region}}":             strings.Join(filters.Regions, ", "),
		"{{filter.facility}}":           strings.Join(filters.Facilities, ", "),
		"{{filter.monthNameYear}}":      "",
		"{{filter.monthShortNameYear}}": "",
	}

	// Build monthName from month numbers
	monthName := ""
	if filters.Month != "" {
		parts := strings.Split(filters.Month, ",")
		names := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if m, err := strconv.Atoi(p); err == nil && m >= 1 && m <= 12 {
				names = append(names, monthNames[m-1]) // monthNames from database.go (full names)
			}
		}
		monthName = strings.Join(names, ", ")
	}
	replacements["{{filter.monthName}}"] = monthName

	if filters.Month != "" && filters.Year != "" &&
		!strings.Contains(filters.Month, ",") && !strings.Contains(filters.Year, ",") {
		monthVal, err1 := strconv.Atoi(strings.TrimSpace(filters.Month))
		yearVal, err2 := strconv.Atoi(strings.TrimSpace(filters.Year))
		if err1 == nil && err2 == nil && monthVal >= 1 && monthVal <= 12 {
			replacements["{{filter.monthNameYear}}"] = fmt.Sprintf("%s %d", monthNames[monthVal-1], yearVal)
			replacements["{{filter.monthShortNameYear}}"] = fmt.Sprintf("%s %d", monthAbbreviations[monthVal-1], yearVal)
		}
	}

	// Build weekLabel: "W09 2026" from "2026W09"
	weekLabel := ""
	if filters.Week != "" && !strings.Contains(filters.Week, ",") {
		if wy, wn, err := ParseWeekPeriod(filters.Week); err == nil {
			weekLabel = fmt.Sprintf("W%02d %d", wn, wy)
		}
	}
	replacements["{{filter.weekLabel}}"] = weekLabel

	// Process arithmetic expressions first (before simple replacements)
	// Examples: {{filter.year - 1}}, {{filter.month + 1}}, {{filter.quarter - 1}}
	text = evaluateFilterArithmetic(text, filters)

	// Apply defaults for empty values
	defaults := map[string]string{
		"{{filter.year}}":               "All Years",
		"{{filter.month}}":              "All Months",
		"{{filter.monthName}}":          "All Months",
		"{{filter.monthNameYear}}":      "All Months",
		"{{filter.monthShortNameYear}}": "All Months",
		"{{filter.week}}":               "All Weeks",
		"{{filter.weekLabel}}":          "All Weeks",
		"{{filter.quarter}}":            "All Quarters",
		"{{filter.district}}":           "All Districts",
		"{{filter.region}}":             "All Regions",
		"{{filter.facility}}":           "All Facilities",
	}

	for placeholder, value := range replacements {
		if !strings.Contains(text, placeholder) {
			continue
		}
		if value == "" {
			value = defaults[placeholder]
		}
		text = strings.ReplaceAll(text, placeholder, value)
	}

	return text
}

// evaluateFilterArithmetic processes arithmetic expressions in filter placeholders
// Supports: {{filter.year ± N}}, {{filter.month ± N}}, {{filter.quarter ± N}}, {{filter.monthName ± N}}
// Only works with single-value filters (ignores multi-select comma-separated values)
// filterArithmeticPattern matches: {{filter.fieldName + number}} or {{filter.fieldName - number}}
var filterArithmeticPattern = regexp.MustCompile(`\{\{filter\.(\w+)\s*([\+\-])\s*(\d+)\}\}`)

func evaluateFilterArithmetic(text string, filters ReportFilters) string {
	arithmeticPattern := filterArithmeticPattern

	matches := arithmeticPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return text
	}

	// Defaults for arithmetic expressions when filter is empty
	arithmeticDefaults := map[string]string{
		"year":               "All Years",
		"month":              "All Months",
		"monthName":          "All Months",
		"monthNameYear":      "All Months",
		"monthShortNameYear": "All Months",
		"quarter":            "All Quarters",
		"weekLabel":          "All Weeks",
	}

	// Fallbacks for multi-select (arithmetic can't be applied to multiple values)
	multiSelectFallbacks := map[string]string{
		"year":               "Selected Years",
		"month":              "Selected Months",
		"monthName":          "Selected Months",
		"monthNameYear":      "Selected Months",
		"monthShortNameYear": "Selected Months",
		"quarter":            "Selected Quarters",
		"weekLabel":          "Selected Weeks",
	}

	for _, match := range matches {
		fullMatch := match[0]  // e.g., "{{filter.year - 1}}"
		fieldName := match[1]  // e.g., "year"
		operator := match[2]   // "+" or "-"
		operandStr := match[3] // e.g., "1"

		operand, err := strconv.Atoi(operandStr)
		if err != nil {
			continue // Skip invalid operands
		}

		var result string
		useDefault := false
		useMultiSelectFallback := false

		switch fieldName {
		case "year":
			if filters.Year == "" {
				useDefault = true
			} else if strings.Contains(filters.Year, ",") {
				useMultiSelectFallback = true
			} else {
				if yearVal, err := strconv.Atoi(filters.Year); err == nil {
					newYear := applyArithmetic(yearVal, operator, operand)
					result = strconv.Itoa(newYear)
				}
			}

		case "month":
			if filters.Month == "" {
				useDefault = true
			} else if strings.Contains(filters.Month, ",") {
				useMultiSelectFallback = true
			} else {
				if monthVal, err := strconv.Atoi(filters.Month); err == nil {
					// Wrap month around 1-12
					newMonth := applyArithmetic(monthVal, operator, operand)
					newMonth = wrapValue(newMonth, 1, 12)
					result = strconv.Itoa(newMonth)
				}
			}

		case "quarter":
			if filters.Quarter == "" {
				useDefault = true
			} else if strings.Contains(filters.Quarter, ",") {
				useMultiSelectFallback = true
			} else {
				if quarterVal, err := strconv.Atoi(filters.Quarter); err == nil {
					// Wrap quarter around 1-4
					newQuarter := applyArithmetic(quarterVal, operator, operand)
					newQuarter = wrapValue(newQuarter, 1, 4)
					result = strconv.Itoa(newQuarter)
				}
			}

		case "monthName":
			// Apply arithmetic to the month number, then convert to name
			if filters.Month == "" {
				useDefault = true
			} else if strings.Contains(filters.Month, ",") {
				useMultiSelectFallback = true
			} else {
				if monthVal, err := strconv.Atoi(filters.Month); err == nil {
					newMonth := applyArithmetic(monthVal, operator, operand)
					newMonth = wrapValue(newMonth, 1, 12)
					if newMonth >= 1 && newMonth <= 12 {
						result = monthNames[newMonth-1]
					}
				}
			}

		case "monthNameYear":
			// Returns "MonthName Year" with year-adjusted arithmetic across year boundaries.
			// Example: month=8, year=2025, "{{filter.monthNameYear - 11}}" → "September 2024"
			// Example: month=12, year=2025, "{{filter.monthNameYear - 11}}" → "January 2025"
			if filters.Month == "" || filters.Year == "" {
				useDefault = true
			} else if strings.Contains(filters.Month, ",") || strings.Contains(filters.Year, ",") {
				useMultiSelectFallback = true
			} else {
				monthVal, err1 := strconv.Atoi(filters.Month)
				yearVal, err2 := strconv.Atoi(filters.Year)
				if err1 == nil && err2 == nil {
					newMonthRaw := applyArithmetic(monthVal, operator, operand)
					finalMonth := wrapValue(newMonthRaw, 1, 12)
					// yearDelta is exact (newMonthRaw - finalMonth is always divisible by 12)
					yearDelta := (newMonthRaw - finalMonth) / 12
					finalYear := yearVal + yearDelta
					if finalMonth >= 1 && finalMonth <= 12 {
						result = fmt.Sprintf("%s %d", monthNames[finalMonth-1], finalYear)
					}
				}
			}

		case "monthShortNameYear":
			// Returns "Mon YYYY" with year-adjusted arithmetic across year boundaries.
			if filters.Month == "" || filters.Year == "" {
				useDefault = true
			} else if strings.Contains(filters.Month, ",") || strings.Contains(filters.Year, ",") {
				useMultiSelectFallback = true
			} else {
				monthVal, err1 := strconv.Atoi(filters.Month)
				yearVal, err2 := strconv.Atoi(filters.Year)
				if err1 == nil && err2 == nil {
					newMonthRaw := applyArithmetic(monthVal, operator, operand)
					finalMonth := wrapValue(newMonthRaw, 1, 12)
					yearDelta := (newMonthRaw - finalMonth) / 12
					finalYear := yearVal + yearDelta
					if finalMonth >= 1 && finalMonth <= 12 {
						result = fmt.Sprintf("%s %d", monthAbbreviations[finalMonth-1], finalYear)
					}
				}
			}

		case "weekLabel":
			// Subtract/add weeks from the selected week, producing "W09 2026" format.
			// Handles year boundaries using ISO week calendar.
			if filters.Week == "" {
				useDefault = true
			} else if strings.Contains(filters.Week, ",") {
				useMultiSelectFallback = true
			} else {
				if wy, wn, err := ParseWeekPeriod(filters.Week); err == nil {
					totalWeeks := wn
					yr := wy
					if operator == "-" {
						totalWeeks -= operand
						for totalWeeks < 1 {
							yr--
							totalWeeks += getISOWeeksInYear(yr)
						}
					} else {
						totalWeeks += operand
						for totalWeeks > getISOWeeksInYear(yr) {
							totalWeeks -= getISOWeeksInYear(yr)
							yr++
						}
					}
					result = fmt.Sprintf("W%02d %d", totalWeeks, yr)
				}
			}
		}

		// Replace with computed result, default, or multi-select fallback
		if useDefault {
			if defaultVal, ok := arithmeticDefaults[fieldName]; ok {
				text = strings.ReplaceAll(text, fullMatch, defaultVal)
			}
		} else if useMultiSelectFallback {
			if fallback, ok := multiSelectFallbacks[fieldName]; ok {
				text = strings.ReplaceAll(text, fullMatch, fallback)
			}
		} else if result != "" {
			text = strings.ReplaceAll(text, fullMatch, result)
		}
	}

	return text
}

// applyArithmetic performs simple addition or subtraction
func applyArithmetic(value int, operator string, operand int) int {
	if operator == "+" {
		return value + operand
	}
	return value - operand
}

// wrapValue wraps a value within min-max bounds (for month 1-12, quarter 1-4)
// Examples: 0 wraps to 12, 13 wraps to 1 (for month)
func wrapValue(value, min, max int) int {
	rangeSize := max - min + 1

	// Normalize to 0-based range
	normalized := value - min

	// Apply modulo wrapping
	normalized = normalized % rangeSize
	if normalized < 0 {
		normalized += rangeSize
	}

	// Convert back to min-based range
	return normalized + min
}

// calculatePreviousPeriodFilters creates filters for the previous period based on comparison type
// comparisonType: "previous_period", "previous_month", or "previous_year"
// For "previous_period": adapts based on active filter (week→week, month→month, quarter→quarter)
// Returns modified filters for the previous period, or nil if comparison cannot be calculated
func calculatePreviousPeriodFilters(filters ReportFilters, comparisonType string) *ReportFilters {
	prevFilters := filters // Copy filters

	switch comparisonType {
	case "previous_period":
		// Dynamic: detect which filter was EXPLICITLY provided by user
		// This ensures user intent is respected even if other filters were auto-defaulted
		// Priority: week > quarter > month (based on explicit user selection)
		if filters.WeekProvided && filters.Week != "" {
			logDebug("previous_period: using week comparison (user provided week=%s)", filters.Week)
			return calculatePreviousWeekFilters(filters)
		} else if filters.QuarterProvided && filters.Quarter != "" && filters.Year != "" {
			logDebug("previous_period: using quarter comparison (user provided quarter=%s)", filters.Quarter)
			return calculatePreviousQuarterFilters(filters)
		} else if filters.MonthProvided && filters.Month != "" && filters.Year != "" {
			logDebug("previous_period: using month comparison (user provided month=%s)", filters.Month)
			return calculatePreviousMonthFilters(filters)
		}
		// Fallback: if no explicit time filter, try to use whatever is available
		// Priority for fallback: week > month > quarter
		if filters.Week != "" {
			logDebug("previous_period: fallback to week comparison (week=%s)", filters.Week)
			return calculatePreviousWeekFilters(filters)
		} else if filters.Month != "" && filters.Year != "" {
			logDebug("previous_period: fallback to month comparison (month=%s)", filters.Month)
			return calculatePreviousMonthFilters(filters)
		} else if filters.Quarter != "" && filters.Year != "" {
			logDebug("previous_period: fallback to quarter comparison (quarter=%s)", filters.Quarter)
			return calculatePreviousQuarterFilters(filters)
		}
		logDebug("previous_period: no suitable time filter found for comparison")
		return nil

	case "previous_month":
		// Explicit previous month (kept for backward compatibility)
		return calculatePreviousMonthFilters(filters)

	case "previous_year":
		// Need year to calculate comparison
		if filters.Year == "" {
			return nil
		}
		currentYear, err := strconv.Atoi(filters.Year)
		if err != nil {
			return nil
		}
		// Same month/quarter, previous year
		prevFilters.Year = strconv.Itoa(currentYear - 1)
		// Clear YearMonths to avoid period expansion conflicts
		prevFilters.YearMonths = nil
		return &prevFilters

	default:
		return nil
	}
}

// calculatePreviousMonthFilters calculates filters for the previous month
func calculatePreviousMonthFilters(filters ReportFilters) *ReportFilters {
	if filters.Year == "" || filters.Month == "" {
		return nil
	}

	currentYear, err := strconv.Atoi(filters.Year)
	if err != nil {
		return nil
	}
	currentMonth, err := strconv.Atoi(filters.Month)
	if err != nil {
		return nil
	}

	// Calculate previous month
	prevMonth := currentMonth - 1
	prevYear := currentYear
	if prevMonth < 1 {
		prevMonth = 12
		prevYear--
	}

	prevFilters := filters
	prevFilters.Year = strconv.Itoa(prevYear)
	prevFilters.Month = strconv.Itoa(prevMonth)
	// Clear other time filters to avoid conflicts - month comparison should only use month
	prevFilters.Week = ""
	prevFilters.Quarter = ""
	prevFilters.YearMonths = nil
	prevFilters.YearQuarters = nil

	return &prevFilters
}

// calculatePreviousWeekFilters calculates filters for the previous week
func calculatePreviousWeekFilters(filters ReportFilters) *ReportFilters {
	if filters.Week == "" {
		return nil
	}

	// Parse current week (format: YYYYWNN)
	year, week, err := ParseWeekPeriod(filters.Week)
	if err != nil {
		return nil
	}

	// Calculate previous week
	prevWeek := week - 1
	prevYear := year
	if prevWeek < 1 {
		prevYear--
		prevWeek = getISOWeeksInYear(prevYear)
	}

	prevFilters := filters
	prevFilters.Week = fmt.Sprintf("%dW%02d", prevYear, prevWeek)
	// Clear other time filters to avoid conflicts - week comparison should only use week
	prevFilters.Year = ""
	prevFilters.Month = ""
	prevFilters.Quarter = ""
	prevFilters.YearMonths = nil
	prevFilters.YearQuarters = nil

	return &prevFilters
}

// calculatePreviousQuarterFilters calculates filters for the previous quarter
func calculatePreviousQuarterFilters(filters ReportFilters) *ReportFilters {
	if filters.Year == "" || filters.Quarter == "" {
		return nil
	}

	currentYear, err := strconv.Atoi(filters.Year)
	if err != nil {
		return nil
	}
	currentQuarter, err := strconv.Atoi(filters.Quarter)
	if err != nil {
		return nil
	}

	// Calculate previous quarter
	prevQuarter := currentQuarter - 1
	prevYear := currentYear
	if prevQuarter < 1 {
		prevQuarter = 4
		prevYear--
	}

	prevFilters := filters
	prevFilters.Year = strconv.Itoa(prevYear)
	prevFilters.Quarter = strconv.Itoa(prevQuarter)
	// Clear other time filters to avoid conflicts - quarter comparison should only use quarter
	prevFilters.Month = ""
	prevFilters.Week = ""
	prevFilters.YearMonths = nil
	prevFilters.YearQuarters = nil

	return &prevFilters
}

// calculateComparison computes the comparison data between current and previous values
func calculateComparison(currentValue, previousValue float64) *ComparisonData {
	change := currentValue - previousValue

	var percentageChange float64
	if previousValue != 0 {
		percentageChange = (change / previousValue) * 100
	} else if currentValue > 0 {
		percentageChange = 100 // From 0 to something is 100% increase
	}

	direction := "unchanged"
	if change > 0 {
		direction = "up"
	} else if change < 0 {
		direction = "down"
	}

	return &ComparisonData{
		PreviousValue:    previousValue,
		Change:           change,
		PercentageChange: percentageChange,
		Direction:        direction,
	}
}

// calculateTargetComparison computes comparison data between current value and a static target
func calculateTargetComparison(currentValue, targetValue float64, label string) *TargetComparisonData {
	change := currentValue - targetValue

	var percentageChange float64
	if targetValue != 0 {
		percentageChange = (change / targetValue) * 100
	} else if currentValue > 0 {
		percentageChange = 100
	}

	direction := "unchanged"
	if change > 0 {
		direction = "up" // Above target
	} else if change < 0 {
		direction = "down" // Below target
	}

	// Default label if not provided
	if label == "" {
		label = "Target"
	}

	return &TargetComparisonData{
		TargetValue:      targetValue,
		TargetLabel:      label,
		Change:           change,
		PercentageChange: percentageChange,
		Direction:        direction,
	}
}

// executeQueryWithContext executes a query with context for timeout support
// This is the preferred method - allows cancellation of long-running queries
// Uses automatic retry with exponential backoff for transient connection errors
// db selects the database connection; pass DB at call sites.
func executeQueryWithContext(ctx context.Context, db *sql.DB, sqlQuery string, args ...interface{}) (TableData, error) {
	start := time.Now()
	table := extractTableNameFromSQL(sqlQuery)

	logAndRecord := func(rowCount int, err error) {
		durationMs := time.Since(start).Seconds() * 1000
		LogExecutedSQL(sqlQuery, rowCount, durationMs, err)
		RecordQueryExecution(table, durationMs, rowCount, err)
	}

	rows, err := queryWithRetry(ctx, db, sqlQuery, args...)
	if err != nil {
		logAndRecord(0, err)
		return TableData{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		logAndRecord(0, err)
		return TableData{}, err
	}

	var results [][]interface{}
	for rows.Next() {
		// Check context cancellation between rows for very large result sets
		select {
		case <-ctx.Done():
			logAndRecord(len(results), ctx.Err())
			return TableData{}, ctx.Err()
		default:
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			logAndRecord(len(results), err)
			return TableData{}, err
		}

		rowSlice := make([]interface{}, len(columns))
		for i, val := range values {
			// Handle byte slices (common for strings, TEXT, etc. in SQLite)
			if b, ok := val.([]byte); ok {
				rowSlice[i] = string(b)
			} else {
				rowSlice[i] = val
			}
		}
		results = append(results, rowSlice)
	}

	if err = rows.Err(); err != nil {
		logAndRecord(len(results), err)
		return TableData{}, err
	}

	logAndRecord(len(results), nil)
	return TableData{Headers: columns, Rows: results}, nil
}

// executeQuery executes a query without context (legacy support)
// Prefer executeQueryWithContext for new code
func executeQuery(sqlQuery string, args ...interface{}) (TableData, error) {
	return executeQueryWithContext(context.Background(), DB, sqlQuery, args...)
}

// ExecuteReportPreview executes a report from the builder for preview purposes
// Captures errors per component instead of failing silently
// Returns the executed report and a list of component errors
func ExecuteReportPreview(report *Report, filters ReportFilters) (*Report, []ComponentError) {
	// Enforce component count cap
	totalComponents := 0
	for _, section := range report.Sections {
		totalComponents += len(section.Components)
	}
	if totalComponents > MaxComponentsPerReport {
		return report, []ComponentError{{
			SectionIndex:   0,
			ComponentIndex: 0,
			Error:          fmt.Sprintf("report has %d components, exceeding maximum of %d", totalComponents, MaxComponentsPerReport),
		}}
	}

	// Create context with timeout to prevent runaway queries
	ctx, cancel := context.WithTimeout(context.Background(), ComponentQueryTimeout)
	defer cancel()

	// Create WaitGroup and pre-allocated result slices for concurrent execution
	var wg sync.WaitGroup
	results := make([][]ComponentResult, len(report.Sections))
	for i := range results {
		results[i] = make([]ComponentResult, len(report.Sections[i].Components))
	}

	// Launch goroutines for each component to execute queries concurrently
	for i, section := range report.Sections {
		for j, component := range section.Components {
			wg.Add(1)
			go func(si, ci int, comp Component) {
				defer wg.Done()
				processComponent(ctx, si, ci, comp, filters, report.Filters, report, &results[si][ci])
			}(i, j, component)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Collect errors and apply results to report structure
	var componentErrors []ComponentError

	for i := range report.Sections {
		for j := range report.Sections[i].Components {
			result := results[i][j]
			if result.Error != nil {
				// Capture the error for this component
				componentErrors = append(componentErrors, ComponentError{
					SectionIndex:   i,
					ComponentIndex: j,
					Title:          report.Sections[i].Components[j].Title,
					Error:          result.Error.Error(),
				})
				// Still set empty data to avoid nil issues
				report.Sections[i].Components[j].Data = Data{}
			} else {
				report.Sections[i].Components[j].Data = result.Data
				if result.Content != "" {
					if report.Sections[i].Components[j].Type == "infobox" {
						report.Sections[i].Components[j].InfoboxBody = result.Content
					} else {
						report.Sections[i].Components[j].Content = result.Content
					}
				}
				// Apply dynamic accent color from threshold evaluation
				if result.AccentColor != "" {
					report.Sections[i].Components[j].AccentColor = result.AccentColor
				}
				// Add comparison data for text components
				if result.Comparison != nil {
					report.Sections[i].Components[j].Data.Comparison = result.Comparison
				}
				// Add target comparison data for text components
				if result.TargetComparison != nil {
					report.Sections[i].Components[j].Data.TargetComparison = result.TargetComparison
				}
				// Pass through reference lines for bar/line charts
				if len(report.Sections[i].Components[j].ReferenceLines) > 0 {
					report.Sections[i].Components[j].Data.ReferenceLines = report.Sections[i].Components[j].ReferenceLines
				}
				// Pass through trend line config for line charts
				if report.Sections[i].Components[j].TrendLine {
					report.Sections[i].Components[j].Data.TrendLine = true
					report.Sections[i].Components[j].Data.TrendLineColor = report.Sections[i].Components[j].TrendLineColor
				}
				// Pass through axis labels for charts
				if report.Sections[i].Components[j].AxisLabels != nil {
					report.Sections[i].Components[j].Data.AxisLabels = report.Sections[i].Components[j].AxisLabels
				}
				// Pass through y-axis limit for bar/line/bar_line charts
				if report.Sections[i].Components[j].YLimit != nil {
					report.Sections[i].Components[j].Data.YLimit = report.Sections[i].Components[j].YLimit
				}
			}
		}
	}

	interpolateReportPlaceholders(report, filters)

	// Add custom filter column names to the filters list (for frontend filter bar rendering)
	for _, cf := range report.CustomFilters {
		report.Filters = append(report.Filters, cf.Column)
	}

	// Enforce canonical filter order: region → district → facility → year → quarter → month → week → custom
	report.Filters = sortFiltersCanonical(report.Filters)

	// Add filter definitions to the report (for dynamic frontend filter rendering)
	report.FilterDefinitions = GetFilterDefinitions(report.Filters, report)

	report.GeneratedAt = time.Now().Format(time.RFC3339)
	return report, componentErrors
}

// isValidYear validates a year string using a range check instead of a hardcoded whitelist.
// Accepts years from 2015 to currentYear+1, preventing both SQL injection and silent expiry.
func isValidYear(yearStr string) bool {
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return false
	}
	return year >= 2015 && year <= time.Now().Year()+1
}

// applyFiltersToSQL replaces filter placeholders with parameterized query values
// Uses report.TimeColumns to determine which columns contain date/time data
// Returns both the SQL query and the parameter arguments
func applyFiltersToSQL(sql string, filters ReportFilters, report *Report) (string, []interface{}) {
	var args []interface{}
	argCount := 1 // PostgreSQL uses $1, $2, $3...
	result := sql

	// Build district filter (multi-select) with parameterized query
	// Only add args if the placeholder exists in the SQL
	if strings.Contains(result, "{{district_filter}}") && len(filters.Districts) > 0 {
		districtCol := GetDistrictColumn(report)
		var districtPlaceholders []string
		for _, district := range filters.Districts {
			districtPlaceholders = append(districtPlaceholders, fmt.Sprintf("$%d", argCount))
			args = append(args, district)
			argCount++
		}

		var placeholder string
		if len(districtPlaceholders) == 1 {
			placeholder = fmt.Sprintf(" AND %s = %s", districtCol, districtPlaceholders[0])
		} else {
			placeholder = fmt.Sprintf(" AND %s IN (%s)", districtCol, strings.Join(districtPlaceholders, ","))
		}
		result = strings.ReplaceAll(result, "{{district_filter}}", placeholder)
	} else {
		result = strings.ReplaceAll(result, "{{district_filter}}", "")
	}

	// Check if YearMonths is populated (from periodLimit expansion)
	// This builds a combined filter to avoid cross-product across year boundaries
	// Only add args if the year_filter placeholder exists in the SQL
	if (strings.Contains(result, "{{year_filter}}") || strings.Contains(result, "{{month_filter}}")) && len(filters.YearMonths) > 0 {
		yearCol := GetTimeColumn(report, "year")
		monthCol := GetTimeColumn(report, "month")
		yearColType := DetectColumnType(yearCol)
		monthColType := ResolveMonthColumnType(report)

		var yearMonthConditions []string

		// Build condition for each year and its months
		for year, months := range filters.YearMonths {
			if !isValidYear(year) {
				continue
			}

			// Build year condition
			var yearCond string
			switch yearColType {
			case "numeric":
				yearCond = fmt.Sprintf("%s = $%d", yearCol, argCount)
			case "date":
				yearCond = fmt.Sprintf("SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount)
			default:
				yearCond = fmt.Sprintf("SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount)
			}
			args = append(args, year)
			argCount++

			// Build month conditions for this year
			var monthConds []string
			for _, month := range months {
				if month < 1 || month > 12 {
					continue
				}
				switch monthColType {
				case "text":
					monthConds = append(monthConds, fmt.Sprintf("$%d", argCount))
					args = append(args, FormatMonthName(month))
				case "numeric":
					monthConds = append(monthConds, fmt.Sprintf("$%d", argCount))
					if report != nil && report.TimeColumns.MonthBindString {
						args = append(args, strconv.Itoa(month))
					} else {
						args = append(args, month)
					}
				case "date":
					monthStr := FormatMonthAbbreviated(strconv.Itoa(month))
					monthConds = append(monthConds, fmt.Sprintf("TO_CHAR(%s::date, 'Mon') = $%d", monthCol, argCount))
					args = append(args, monthStr)
				default:
					monthStr := FormatMonthAbbreviated(strconv.Itoa(month))
					monthConds = append(monthConds, fmt.Sprintf("TO_CHAR(%s::date, 'Mon') = $%d", monthCol, argCount))
					args = append(args, monthStr)
				}
				argCount++
			}

			if len(monthConds) > 0 {
				var monthFilter string
				if monthColType == "numeric" || monthColType == "text" {
					effectiveMonthCol := monthCol
					if monthColType == "numeric" && report != nil && report.TimeColumns.MonthBindString {
						effectiveMonthCol = monthCol + "::text"
					}
					monthFilter = fmt.Sprintf("%s IN (%s)", effectiveMonthCol, strings.Join(monthConds, ","))
				} else {
					monthFilter = strings.Join(monthConds, " OR ")
				}
				yearMonthConditions = append(yearMonthConditions, fmt.Sprintf("(%s AND (%s))", yearCond, monthFilter))
			}
		}

		if len(yearMonthConditions) > 0 {
			// Replace both year and month filters with combined condition
			combinedFilter := fmt.Sprintf(" AND (%s)", strings.Join(yearMonthConditions, " OR "))
			result = strings.ReplaceAll(result, "{{year_filter}}", combinedFilter)
			result = strings.ReplaceAll(result, "{{month_filter}}", "") // Already handled
		} else {
			result = strings.ReplaceAll(result, "{{year_filter}}", "")
			result = strings.ReplaceAll(result, "{{month_filter}}", "")
		}
	} else if strings.Contains(result, "{{year_filter}}") && filters.Year != "" {
		// Build year filter using GetTimeColumn with sensible defaults
		// Support multiple years: comma-separated list (e.g., "2023,2024")
		yearValues := strings.Split(filters.Year, ",")
		var validYearsList []string
		yearCol := GetTimeColumn(report, "year")
		colType := DetectColumnType(yearCol)

		// Validate each year value
		for _, y := range yearValues {
			y = strings.TrimSpace(y)
			if isValidYear(y) {
				validYearsList = append(validYearsList, y)
			}
		}

		if len(validYearsList) == 0 {
			result = strings.ReplaceAll(result, "{{year_filter}}", "")
		} else if len(validYearsList) == 1 {
			// Single year: use = operator
			var placeholder string
			switch colType {
			case "numeric":
				placeholder = fmt.Sprintf(" AND %s = $%d", yearCol, argCount)
			case "string_period":
				placeholder = fmt.Sprintf(" AND SUBSTRING(%s, 1, 4) = $%d", yearCol, argCount)
			case "date":
				placeholder = fmt.Sprintf(" AND SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount)
			default:
				placeholder = fmt.Sprintf(" AND SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount)
			}
			result = strings.ReplaceAll(result, "{{year_filter}}", placeholder)
			args = append(args, validYearsList[0])
			argCount++
		} else {
			// Multiple years: use IN or OR clause
			var yearConditions []string
			switch colType {
			case "numeric":
				// Numeric year column: year IN ($1, $2, ...)
				for _, year := range validYearsList {
					yearConditions = append(yearConditions, fmt.Sprintf("$%d", argCount))
					args = append(args, year)
					argCount++
				}
				placeholder := fmt.Sprintf(" AND %s IN (%s)", yearCol, strings.Join(yearConditions, ","))
				result = strings.ReplaceAll(result, "{{year_filter}}", placeholder)
			case "date":
				// DATE column: use OR for each year
				for _, year := range validYearsList {
					yearConditions = append(yearConditions, fmt.Sprintf("SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount))
					args = append(args, year)
					argCount++
				}
				placeholder := fmt.Sprintf(" AND (%s)", strings.Join(yearConditions, " OR "))
				result = strings.ReplaceAll(result, "{{year_filter}}", placeholder)
			default:
				// Fallback: treat as date
				for _, year := range validYearsList {
					yearConditions = append(yearConditions, fmt.Sprintf("SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount))
					args = append(args, year)
					argCount++
				}
				placeholder := fmt.Sprintf(" AND (%s)", strings.Join(yearConditions, " OR "))
				result = strings.ReplaceAll(result, "{{year_filter}}", placeholder)
			}
		}
	} else {
		result = strings.ReplaceAll(result, "{{year_filter}}", "")
	}

	// Build month filter using GetTimeColumn with sensible defaults
	// Skip if YearMonths was used (already handled combined filter above)
	// Only add args if the placeholder exists in the SQL
	if strings.Contains(result, "{{month_filter}}") && len(filters.YearMonths) == 0 && filters.Month != "" {
		// Support multiple months: comma-separated list (e.g., "1,2,3")
		monthValues := strings.Split(filters.Month, ",")
		var validMonths []int
		monthCol := GetTimeColumn(report, "month")
		colType := ResolveMonthColumnType(report)

		// Validate each month value
		for _, m := range monthValues {
			m = strings.TrimSpace(m)
			monthNum, err := strconv.Atoi(m)
			if err == nil && monthNum >= 1 && monthNum <= 12 {
				validMonths = append(validMonths, monthNum)
			}
		}

		if len(validMonths) == 0 {
			result = strings.ReplaceAll(result, "{{month_filter}}", "")
		} else {
			// Build OR clause for multiple months
			var monthConditions []string

			switch colType {
			case "text":
				// Text month column: month stores full names (January, February, ...)
				for _, monthNum := range validMonths {
					monthConditions = append(monthConditions, fmt.Sprintf("$%d", argCount))
					args = append(args, FormatMonthName(monthNum))
					argCount++
				}
				placeholder := fmt.Sprintf(" AND %s IN (%s)", monthCol, strings.Join(monthConditions, ","))
				result = strings.ReplaceAll(result, "{{month_filter}}", placeholder)
			case "numeric":
				// Numeric month column: month IN ($1, $2, $3, ...)
				for _, monthNum := range validMonths {
					monthConditions = append(monthConditions, fmt.Sprintf("$%d", argCount))
					if report != nil && report.TimeColumns.MonthBindString {
						args = append(args, strconv.Itoa(monthNum))
					} else {
						args = append(args, monthNum)
					}
					argCount++
				}
				effectiveMonthCol := monthCol
				if report != nil && report.TimeColumns.MonthBindString {
					effectiveMonthCol = monthCol + "::text"
				}
				placeholder := fmt.Sprintf(" AND %s IN (%s)", effectiveMonthCol, strings.Join(monthConditions, ","))
				result = strings.ReplaceAll(result, "{{month_filter}}", placeholder)
			case "date":
				// DATE column: use TO_CHAR with OR for each month
				for _, monthNum := range validMonths {
					monthStr := FormatMonthAbbreviated(strconv.Itoa(monthNum))
					monthConditions = append(monthConditions, fmt.Sprintf("TO_CHAR(%s::date, 'Mon') = $%d", monthCol, argCount))
					args = append(args, monthStr)
					argCount++
				}
				var placeholder string
				if len(monthConditions) == 1 {
					placeholder = fmt.Sprintf(" AND %s", monthConditions[0])
				} else {
					placeholder = fmt.Sprintf(" AND (%s)", strings.Join(monthConditions, " OR "))
				}
				result = strings.ReplaceAll(result, "{{month_filter}}", placeholder)
			default:
				// Fallback: assume date column
				for _, monthNum := range validMonths {
					monthStr := FormatMonthAbbreviated(strconv.Itoa(monthNum))
					monthConditions = append(monthConditions, fmt.Sprintf("TO_CHAR(%s::date, 'Mon') = $%d", monthCol, argCount))
					args = append(args, monthStr)
					argCount++
				}
				var placeholder string
				if len(monthConditions) == 1 {
					placeholder = fmt.Sprintf(" AND %s", monthConditions[0])
				} else {
					placeholder = fmt.Sprintf(" AND (%s)", strings.Join(monthConditions, " OR "))
				}
				result = strings.ReplaceAll(result, "{{month_filter}}", placeholder)
			}
		}
	} else if len(filters.YearMonths) == 0 {
		// Only clear month filter if YearMonths wasn't used
		result = strings.ReplaceAll(result, "{{month_filter}}", "")
	}

	// Build week filter using GetTimeColumn with sensible defaults
	// Only apply week filter if the placeholder exists in the query
	if strings.Contains(result, "{{week_filter}}") && filters.Week != "" {
		// Support multiple weeks: comma-separated list (e.g., "2025W01,2025W02,2025W03")
		weekValues := strings.Split(filters.Week, ",")
		weekCol := GetTimeColumn(report, "week")
		yearCol := GetTimeColumn(report, "year")
		colType := DetectColumnType(weekCol)

		// Parse week values in YYYYWNN format
		type WeekValue struct {
			year int
			week int
		}
		var validWeeks []WeekValue
		validWeekFormat := regexp.MustCompile(`^(\d{4})W(\d{2})$`)

		for _, w := range weekValues {
			w = strings.TrimSpace(w)
			matches := validWeekFormat.FindStringSubmatch(w)
			if len(matches) == 3 {
				year, _ := strconv.Atoi(matches[1])
				week, _ := strconv.Atoi(matches[2])
				if year >= 2020 && year <= 2030 && week >= 1 && week <= 53 {
					validWeeks = append(validWeeks, WeekValue{year: year, week: week})
				}
			}
		}

		if len(validWeeks) == 0 {
			result = strings.ReplaceAll(result, "{{week_filter}}", "")
		} else {
			// Check if year and week are in separate columns (numeric week format)
			separateYearWeekCols := yearCol != "" && yearCol != weekCol
			var weekConditions []string

			if separateYearWeekCols {
				// For separate year/week columns, only filter by week
				// Year filtering is already handled by the year filter
				// If multiple years are selected (rare), build: week IN (1,2) OR week IN (3,4)
				// More commonly: week IN (1,2,3) for a single year

				var weekPlaceholders []string
				for _, wv := range validWeeks {
					weekPlaceholders = append(weekPlaceholders, fmt.Sprintf("$%d", argCount))
					args = append(args, wv.week)
					argCount++
				}
				placeholder := fmt.Sprintf(" AND %s IN (%s)", weekCol, strings.Join(weekPlaceholders, ","))
				result = strings.ReplaceAll(result, "{{week_filter}}", placeholder)
			} else if colType == "string_period" {
				// Week is stored as YYYYWNN string - direct comparison
				for _, wv := range validWeeks {
					weekStr := fmt.Sprintf("%dW%02d", wv.year, wv.week)
					weekConditions = append(weekConditions, fmt.Sprintf("%s = $%d", weekCol, argCount))
					args = append(args, weekStr)
					argCount++
				}
				var placeholder string
				if len(weekConditions) == 1 {
					placeholder = fmt.Sprintf(" AND %s", weekConditions[0])
				} else {
					placeholder = fmt.Sprintf(" AND (%s)", strings.Join(weekConditions, " OR "))
				}
				result = strings.ReplaceAll(result, "{{week_filter}}", placeholder)
			} else {
				// Fallback: assume string_period format
				for _, wv := range validWeeks {
					weekStr := fmt.Sprintf("%dW%02d", wv.year, wv.week)
					weekConditions = append(weekConditions, fmt.Sprintf("%s = $%d", weekCol, argCount))
					args = append(args, weekStr)
					argCount++
				}
				var placeholder string
				if len(weekConditions) == 1 {
					placeholder = fmt.Sprintf(" AND %s", weekConditions[0])
				} else {
					placeholder = fmt.Sprintf(" AND (%s)", strings.Join(weekConditions, " OR "))
				}
				result = strings.ReplaceAll(result, "{{week_filter}}", placeholder)
			}
		}
	} else {
		result = strings.ReplaceAll(result, "{{week_filter}}", "")
	}

	// Build quarter filter using GetTimeColumn with sensible defaults
	// Check if YearQuarters is populated (from periodLimit expansion)
	// This builds a combined filter to avoid cross-product across year boundaries
	if strings.Contains(result, "{{quarter_filter}}") && len(filters.YearQuarters) > 0 {
		yearCol := GetTimeColumn(report, "year")
		quarterCol := GetTimeColumn(report, "quarter")
		yearColType := DetectColumnType(yearCol)
		quarterColType := DetectColumnType(quarterCol)

		var yearQuarterConditions []string

		// Build condition for each year and its quarters
		for year, quarters := range filters.YearQuarters {
			if !isValidYear(year) {
				continue
			}

			// Build year condition
			var yearCond string
			switch yearColType {
			case "numeric":
				yearCond = fmt.Sprintf("%s = $%d", yearCol, argCount)
			case "date":
				yearCond = fmt.Sprintf("SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount)
			default:
				yearCond = fmt.Sprintf("SUBSTR(%s::text, 1, 4) = $%d", yearCol, argCount)
			}
			args = append(args, year)
			argCount++

			// Build quarter conditions for this year based on column type
			var quarterConditions []string
			for _, q := range quarters {
				if q < 1 || q > 4 {
					continue
				}
				switch quarterColType {
				case "numeric", "string_period":
					// Direct comparison for numeric quarter column or string_period columns
					// (e.g., "Period" column that stores quarter numbers 1-4)
					quarterConditions = append(quarterConditions, fmt.Sprintf("%s = $%d", quarterCol, argCount))
				case "date":
					quarterConditions = append(quarterConditions, fmt.Sprintf("EXTRACT(QUARTER FROM %s::date) = $%d", quarterCol, argCount))
				default:
					// Default: assume date column for period_date style columns
					quarterConditions = append(quarterConditions, fmt.Sprintf("EXTRACT(QUARTER FROM %s::date) = $%d", quarterCol, argCount))
				}
				args = append(args, q)
				argCount++
			}

			if len(quarterConditions) > 0 {
				var quarterFilter string
				if len(quarterConditions) == 1 {
					quarterFilter = quarterConditions[0]
				} else {
					quarterFilter = fmt.Sprintf("(%s)", strings.Join(quarterConditions, " OR "))
				}
				yearQuarterConditions = append(yearQuarterConditions, fmt.Sprintf("(%s AND %s)", yearCond, quarterFilter))
			}
		}

		if len(yearQuarterConditions) > 0 {
			// Replace quarter filter with combined year-quarter condition
			combinedFilter := fmt.Sprintf(" AND (%s)", strings.Join(yearQuarterConditions, " OR "))
			result = strings.ReplaceAll(result, "{{quarter_filter}}", combinedFilter)
		} else {
			result = strings.ReplaceAll(result, "{{quarter_filter}}", "")
		}
	} else if strings.Contains(result, "{{quarter_filter}}") && filters.Quarter != "" {
		// Fallback: regular quarter filter without year combination
		// Support multiple quarters: comma-separated list (e.g., "1,2,3")
		quarterValues := strings.Split(filters.Quarter, ",")
		quarterCol := GetTimeColumn(report, "quarter")
		colType := DetectColumnType(quarterCol)

		// Validate quarters (1-4)
		var validQuarters []int
		for _, q := range quarterValues {
			q = strings.TrimSpace(q)
			if qVal, err := strconv.Atoi(q); err == nil && qVal >= 1 && qVal <= 4 {
				validQuarters = append(validQuarters, qVal)
			}
		}

		if len(validQuarters) == 0 {
			result = strings.ReplaceAll(result, "{{quarter_filter}}", "")
		} else {
			// Build quarter filter condition based on column type
			var quarterConditions []string
			for _, q := range validQuarters {
				switch colType {
				case "numeric", "string_period":
					// Direct comparison for numeric quarter column or string_period columns
					// (e.g., "Period" column that stores quarter numbers 1-4)
					quarterConditions = append(quarterConditions, fmt.Sprintf("%s = $%d", quarterCol, argCount))
					args = append(args, q)
					argCount++
				case "date":
					// Extract quarter from date column: EXTRACT(QUARTER FROM date)
					quarterConditions = append(quarterConditions, fmt.Sprintf("EXTRACT(QUARTER FROM %s::date) = $%d", quarterCol, argCount))
					args = append(args, q)
					argCount++
				default:
					// Default: assume date column for period_date style columns
					quarterConditions = append(quarterConditions, fmt.Sprintf("EXTRACT(QUARTER FROM %s::date) = $%d", quarterCol, argCount))
					args = append(args, q)
					argCount++
				}
			}

			var placeholder string
			if len(quarterConditions) == 1 {
				placeholder = fmt.Sprintf(" AND %s", quarterConditions[0])
			} else {
				placeholder = fmt.Sprintf(" AND (%s)", strings.Join(quarterConditions, " OR "))
			}
			result = strings.ReplaceAll(result, "{{quarter_filter}}", placeholder)
		}
	} else {
		result = strings.ReplaceAll(result, "{{quarter_filter}}", "")
	}

	// Build region filter (multi-select) using GetRegionColumn
	// Only apply if the placeholder exists and values are provided
	if strings.Contains(result, "{{region_filter}}") && len(filters.Regions) > 0 {
		regionCol := GetRegionColumn(report)
		var regionPlaceholders []string
		for _, region := range filters.Regions {
			regionPlaceholders = append(regionPlaceholders, fmt.Sprintf("$%d", argCount))
			args = append(args, region)
			argCount++
		}

		var placeholder string
		if len(regionPlaceholders) == 1 {
			placeholder = fmt.Sprintf(" AND %s = %s", regionCol, regionPlaceholders[0])
		} else {
			placeholder = fmt.Sprintf(" AND %s IN (%s)", regionCol, strings.Join(regionPlaceholders, ","))
		}
		result = strings.ReplaceAll(result, "{{region_filter}}", placeholder)
	} else {
		result = strings.ReplaceAll(result, "{{region_filter}}", "")
	}

	// Build facility filter (multi-select) using GetFacilityColumn
	// Only apply if the placeholder exists and values are provided
	if strings.Contains(result, "{{facility_filter}}") && len(filters.Facilities) > 0 {
		facilityCol := GetFacilityColumn(report)
		var facilityPlaceholders []string
		for _, facility := range filters.Facilities {
			facilityPlaceholders = append(facilityPlaceholders, fmt.Sprintf("$%d", argCount))
			args = append(args, facility)
			argCount++
		}

		var placeholder string
		if len(facilityPlaceholders) == 1 {
			placeholder = fmt.Sprintf(" AND %s = %s", facilityCol, facilityPlaceholders[0])
		} else {
			placeholder = fmt.Sprintf(" AND %s IN (%s)", facilityCol, strings.Join(facilityPlaceholders, ","))
		}
		result = strings.ReplaceAll(result, "{{facility_filter}}", placeholder)
	} else {
		result = strings.ReplaceAll(result, "{{facility_filter}}", "")
	}

	// Apply custom filters (only reports loaded from YAML declare these)
	if report == nil {
		return result, args
	}
	for _, cf := range report.CustomFilters {
		placeholder := fmt.Sprintf("{{custom_filter_%s}}", cf.Column)
		if !strings.Contains(result, placeholder) {
			continue
		}

		value, exists := filters.CustomValues[cf.Column]
		if !exists || value == "" {
			result = strings.ReplaceAll(result, placeholder, "")
			continue
		}

		quotedColumn := quoteIdentifier(cf.Column)

		if cf.Type == "multiselect" {
			// Handle multiple values (comma-separated)
			values := strings.Split(value, ",")
			var placeholders []string
			for _, v := range values {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(v))
				argCount++
			}
			filterSQL := fmt.Sprintf(" AND %s IN (%s)", quotedColumn, strings.Join(placeholders, ","))
			result = strings.ReplaceAll(result, placeholder, filterSQL)
		} else {
			// Single value
			filterSQL := fmt.Sprintf(" AND %s = $%d", quotedColumn, argCount)
			result = strings.ReplaceAll(result, placeholder, filterSQL)
			args = append(args, value)
			argCount++
		}
	}

	return result, args
}
