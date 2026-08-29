package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// complexFilterPlaceholders maps filter names to their complex placeholder names
// These generate proper SQL with parameterized queries at runtime
// Complex placeholders handle date extraction, type casting, and multi-select automatically
var complexFilterPlaceholders = map[string]string{
	"district": "district_filter",
	"region":   "region_filter",
	"facility": "facility_filter",
	"year":     "year_filter",
	"month":    "month_filter",
	"week":     "week_filter",
	"quarter":  "quarter_filter",
}

// extractTableFromSQL parses SQL and extracts the first table from a FROM clause
// Returns schema, table, and whether a table was found
// Handles: FROM schema.table, FROM table, FROM schema.table AS alias
func extractTableFromSQL(sql string) (schema string, table string, found bool) {
	// Pattern to match FROM clause with optional schema
	// Matches: FROM schema.table or FROM table
	// Captures the table name (with optional schema prefix)
	fromPattern := regexp.MustCompile(`(?i)\bFROM\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)

	match := fromPattern.FindStringSubmatch(sql)
	if match == nil || len(match) < 2 {
		return "", "", false
	}

	tableName := match[1]

	// Check if it's schema.table or just table
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		return parts[0], parts[1], true
	}

	// No schema specified, assume "report" as default
	return "report", tableName, true
}

// getColumnMapping builds a map of filter name to actual column name
// based on report-level timeColumns and locationColumns configuration
func getColumnMapping(filters []string, timeColumns TimeColumns, locationColumns LocationColumns) map[string]string {
	mapping := make(map[string]string)

	for _, filter := range filters {
		switch filter {
		case "year":
			if timeColumns.Year != "" {
				mapping["year"] = timeColumns.Year
			} else {
				mapping["year"] = "year"
			}
		case "month":
			if timeColumns.Month != "" {
				mapping["month"] = timeColumns.Month
			} else {
				mapping["month"] = "month"
			}
		case "week":
			if timeColumns.Week != "" {
				mapping["week"] = timeColumns.Week
			} else {
				mapping["week"] = "week"
			}
		case "quarter":
			if timeColumns.Quarter != "" {
				mapping["quarter"] = timeColumns.Quarter
			} else {
				mapping["quarter"] = "quarter"
			}
		case "district":
			if locationColumns.District != "" {
				mapping["district"] = locationColumns.District
			} else {
				mapping["district"] = "district"
			}
		case "region":
			if locationColumns.Region != "" {
				mapping["region"] = locationColumns.Region
			} else {
				mapping["region"] = "region"
			}
		case "facility":
			if locationColumns.Facility != "" {
				mapping["facility"] = locationColumns.Facility
			} else {
				mapping["facility"] = "facility"
			}
		}
	}

	return mapping
}

// hardcodedFilterMatch represents a detected hardcoded filter value
type hardcodedFilterMatch struct {
	filterName  string // e.g., "year", "district"
	columnName  string // actual column name in the match
	fullMatch   string // the full matched string, e.g., "year = 2024"
	replacement string // what to replace with, e.g., "year = {{year}}"
}

// detectHardcodedFilters finds hardcoded filter values in SQL
// Returns the matches found and descriptions of what was detected
// Note: We detect hardcoded values but will remove them entirely since
// complex filter placeholders are injected via CTE wrapper
func detectHardcodedFilters(sql string, columnMapping map[string]string) []hardcodedFilterMatch {
	var matches []hardcodedFilterMatch

	for filterName, columnName := range columnMapping {
		// Skip if column name is empty
		if columnName == "" {
			continue
		}

		placeholder := complexFilterPlaceholders[filterName]
		if placeholder == "" {
			continue
		}

		// For complex placeholders, we'll remove hardcoded conditions entirely
		// since filters are applied via the CTE wrapper with proper SQL generation
		replacement := "" // Will be removed

		// Build patterns to detect hardcoded values for this column
		// Pattern 1: column = 'value' or column = "value" (string)
		stringPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(columnName) + `\s*=\s*['"]([^'"]+)['"]`)
		// Pattern 2: column = number (numeric, including year like 2024)
		numericPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(columnName) + `\s*=\s*(\d+)`)
		// Pattern 3: column IN ('value1', 'value2', ...)
		inStringPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(columnName) + `\s+IN\s*\([^)]+\)`)
		// Pattern 4: column BETWEEN x AND y
		betweenPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(columnName) + `\s+BETWEEN\s+\S+\s+AND\s+\S+`)

		// Check string pattern
		if match := stringPattern.FindString(sql); match != "" {
			matches = append(matches, hardcodedFilterMatch{
				filterName:  filterName,
				columnName:  columnName,
				fullMatch:   match,
				replacement: replacement,
			})
		}

		// Check numeric pattern
		if match := numericPattern.FindString(sql); match != "" {
			// Don't match if already matched by string pattern
			alreadyMatched := false
			for _, m := range matches {
				if m.filterName == filterName {
					alreadyMatched = true
					break
				}
			}
			if !alreadyMatched {
				matches = append(matches, hardcodedFilterMatch{
					filterName:  filterName,
					columnName:  columnName,
					fullMatch:   match,
					replacement: replacement,
				})
			}
		}

		// Check IN pattern
		if match := inStringPattern.FindString(sql); match != "" {
			alreadyMatched := false
			for _, m := range matches {
				if m.filterName == filterName {
					alreadyMatched = true
					break
				}
			}
			if !alreadyMatched {
				matches = append(matches, hardcodedFilterMatch{
					filterName:  filterName,
					columnName:  columnName,
					fullMatch:   match,
					replacement: replacement,
				})
			}
		}

		// Check BETWEEN pattern
		if match := betweenPattern.FindString(sql); match != "" {
			alreadyMatched := false
			for _, m := range matches {
				if m.filterName == filterName {
					alreadyMatched = true
					break
				}
			}
			if !alreadyMatched {
				matches = append(matches, hardcodedFilterMatch{
					filterName:  filterName,
					columnName:  columnName,
					fullMatch:   match,
					replacement: replacement,
				})
			}
		}
	}

	return matches
}

// replaceHardcodedFilters replaces detected hardcoded filter values with placeholders
func replaceHardcodedFilters(sql string, matches []hardcodedFilterMatch) string {
	result := sql
	for _, m := range matches {
		// Use case-insensitive replacement
		pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(m.fullMatch))
		result = pattern.ReplaceAllString(result, m.replacement)
	}
	return result
}

// processAdvancedSQL processes a table_advanced component's SQL
// - Extracts the table from FROM clause
// - Detects and removes hardcoded filter values
// - Wraps SQL in CTE with complex filter placeholders
// Complex placeholders ({{year_filter}}, {{district_filter}}, etc.) generate
// proper SQL at runtime with correct date handling and parameterization
func processAdvancedSQL(sql string, filters []string, timeColumns TimeColumns, locationColumns LocationColumns, componentIndex int) (string, SQLProcessingInfo) {
	info := SQLProcessingInfo{
		ComponentIndex: componentIndex,
		FiltersApplied: []string{},
		Warnings:       []string{},
		Replacements:   []string{},
	}

	// Check if SQL already has complex filter placeholders - if so, don't modify
	for _, placeholder := range complexFilterPlaceholders {
		placeholderSyntax := "{{" + placeholder + "}}"
		if strings.Contains(sql, placeholderSyntax) {
			info.Warnings = append(info.Warnings, "SQL already contains filter placeholders, not modified")
			return sql, info
		}
	}

	// Extract table from FROM clause
	schema, table, found := extractTableFromSQL(sql)
	if !found {
		info.Warnings = append(info.Warnings, "Could not detect table from FROM clause")
		return sql, info
	}

	fullTableName := schema + "." + table
	info.DetectedTable = fullTableName

	// Get column mapping based on report config
	columnMapping := getColumnMapping(filters, timeColumns, locationColumns)

	// Detect hardcoded filter values (these will be removed)
	hardcodedMatches := detectHardcodedFilters(sql, columnMapping)

	// Remove hardcoded filter conditions from SQL
	processedSQL := sql
	for _, match := range hardcodedMatches {
		if match.fullMatch != "" {
			info.Replacements = append(info.Replacements,
				fmt.Sprintf("Removed: %s (handled by {{%s}})", match.fullMatch, complexFilterPlaceholders[match.filterName]))
		}
	}
	processedSQL = replaceHardcodedFilters(processedSQL, hardcodedMatches)

	// Clean up any orphaned WHERE/AND after removing conditions
	processedSQL = cleanupWhereClause(processedSQL)

	// Build list of complex filter placeholders to inject
	var filterPlaceholders []string
	for _, filterName := range filters {
		placeholder := complexFilterPlaceholders[filterName]
		if placeholder != "" {
			filterPlaceholders = append(filterPlaceholders, "{{"+placeholder+"}}")
			info.FiltersApplied = append(info.FiltersApplied, filterName)
		}
	}

	// Wrap SQL in CTE with complex filter placeholders
	if len(filterPlaceholders) > 0 {
		processedSQL = injectFilterConditions(processedSQL, filterPlaceholders, fullTableName)
	}

	return processedSQL, info
}

// cleanupWhereClause removes orphaned WHERE/AND keywords after filter removal
func cleanupWhereClause(sql string) string {
	// Remove "WHERE AND" -> "WHERE"
	whereAndPattern := regexp.MustCompile(`(?i)\bWHERE\s+AND\b`)
	sql = whereAndPattern.ReplaceAllString(sql, "WHERE")

	// Remove "AND AND" -> "AND"
	andAndPattern := regexp.MustCompile(`(?i)\bAND\s+AND\b`)
	sql = andAndPattern.ReplaceAllString(sql, "AND")

	// Remove trailing "WHERE" with nothing after (before GROUP BY, ORDER BY, or end)
	trailingWherePattern := regexp.MustCompile(`(?i)\bWHERE\s*(GROUP\s+BY|ORDER\s+BY|LIMIT|$)`)
	sql = trailingWherePattern.ReplaceAllString(sql, "$1")

	// Remove trailing "AND" before GROUP BY, ORDER BY, or end
	trailingAndPattern := regexp.MustCompile(`(?i)\bAND\s*(GROUP\s+BY|ORDER\s+BY|LIMIT|$)`)
	sql = trailingAndPattern.ReplaceAllString(sql, "$1")

	return sql
}

// injectFilterConditions wraps SQL in a CTE with complex filter placeholders
// Complex placeholders like {{year_filter}}, {{district_filter}} generate proper
// SQL at runtime with correct date extraction and parameterized queries
func injectFilterConditions(sql string, filterPlaceholders []string, tableName string) string {
	if len(filterPlaceholders) == 0 {
		return sql
	}

	// Build the WHERE clause with 1=1 pattern for clean placeholder appending
	// Complex placeholders generate " AND ..." clauses at runtime
	placeholderStr := strings.Join(filterPlaceholders, " ")

	// Build a CTE that filters the source table using complex placeholders
	filteredSourceCTE := fmt.Sprintf("_filtered_source AS (\n  SELECT * FROM %s\n  WHERE 1=1 %s\n)", tableName, placeholderStr)

	// Replace the table reference with _filtered_source
	tablePattern := regexp.MustCompile(`(?i)\bFROM\s+` + regexp.QuoteMeta(tableName) + `\b`)
	if !tablePattern.MatchString(sql) {
		// Table not found, can't inject
		return sql
	}

	modifiedSQL := tablePattern.ReplaceAllString(sql, "FROM _filtered_source")

	// Check if SQL already has a WITH clause
	withPattern := regexp.MustCompile(`(?i)^\s*WITH\s+`)
	if withPattern.MatchString(modifiedSQL) {
		// Insert our CTE after WITH and before existing CTEs
		return withPattern.ReplaceAllString(modifiedSQL, "WITH "+filteredSourceCTE+",\n")
	}

	// No existing WITH - add new one
	return fmt.Sprintf("WITH %s\n%s", filteredSourceCTE, modifiedSQL)
}

// validComponentTypes defines allowed component types
var validComponentTypes = map[string]bool{
	"text":           true,
	"kpi":            true,
	"bar":            true,
	"line":           true,
	"pie":            true,
	"table":          true,
	"table_advanced": true,
	"choropleth":     true,
	"map":            true,
	"bar_line":       true,
	"pyramid":        true,
	"infobox":        true,
}

// validateComponentForYAML validates a component and returns error message if invalid
func validateComponentForYAML(comp Component, sectionNum, compNum int) string {
	prefix := fmt.Sprintf("section %d, component %d", sectionNum, compNum)

	// Validate component type
	if comp.Type == "" {
		return fmt.Sprintf("%s: type is required", prefix)
	}
	if !validComponentTypes[comp.Type] {
		return fmt.Sprintf("%s: invalid type '%s'", prefix, comp.Type)
	}

	// Type-specific validation
	switch comp.Type {
	case "table_advanced":
		if comp.SQL == "" {
			return fmt.Sprintf("%s: SQL is required for table_advanced", prefix)
		}
	case "text":
		// Text components are valid with just type
	case "infobox":
		// Infobox components are valid with just type (body and accent colour are optional)
	case "kpi":
		// KPI components need a title (rendered as the card heading) and a
		// query or SQL to produce the value.
		if comp.Title == "" {
			return fmt.Sprintf("%s: title is required for kpi", prefix)
		}
		if comp.Query == nil && comp.SQL == "" {
			return fmt.Sprintf("%s: query or sql is required for kpi", prefix)
		}
	case "choropleth":
		// Choropleth can use either query builder OR custom SQL with columnMapping
		if comp.SQL != "" {
			// Using custom SQL mode - requires columnMapping
			if comp.ColumnMapping == nil {
				return fmt.Sprintf("%s: columnMapping is required when using SQL for choropleth", prefix)
			}
		} else {
			// Using query builder mode - requires query with table
			if comp.Query == nil {
				return fmt.Sprintf("%s: query is required for choropleth (or use custom SQL)", prefix)
			}
			if comp.Query.Table == "" {
				return fmt.Sprintf("%s: query.table is required", prefix)
			}
		}
	case "bar", "line", "bar_line", "pie":
		// Chart types can use either structured query or raw SQL
		if comp.Query == nil && comp.SQL == "" {
			return fmt.Sprintf("%s: query or sql is required for %s", prefix, comp.Type)
		}
	case "pyramid":
		if comp.Query == nil && comp.SQL == "" {
			return fmt.Sprintf("%s: query or sql is required for pyramid", prefix)
		}
		// Skip structured-query checks when raw SQL is provided. The query block
		// may still be present to carry options like periodLimit.
		if comp.Query != nil && comp.SQL == "" {
			if comp.Query.Table == "" {
				return fmt.Sprintf("%s: query.table is required", prefix)
			}
			if len(comp.Query.Aggregations) != 2 {
				return fmt.Sprintf("%s: pyramid requires exactly 2 aggregations (left and right datasets), got %d", prefix, len(comp.Query.Aggregations))
			}
			if len(comp.Query.GroupBy) == 0 {
				return fmt.Sprintf("%s: pyramid requires at least one groupBy (e.g. age group)", prefix)
			}
		}
	default:
		// Tables need a query with table and aggregations
		if comp.Query == nil {
			return fmt.Sprintf("%s: query is required for %s", prefix, comp.Type)
		}
		if comp.Query.Table == "" {
			return fmt.Sprintf("%s: query.table is required", prefix)
		}
		if len(comp.Query.Aggregations) == 0 {
			return fmt.Sprintf("%s: at least one aggregation is required", prefix)
		}
	}

	return ""
}

// GenerateYAMLHandler generates YAML from a report structure
func GenerateYAMLHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "method not allowed"))
		return
	}

	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req YAMLGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid request"))
		return
	}

	// Validate report ID
	if err := ValidateReportID(req.ID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), err.Error()))
		return
	}

	// Process components with raw SQL (table_advanced and chart types)
	var sqlProcessingInfos []SQLProcessingInfo
	componentIndex := 0
	for i := range req.Sections {
		for j := range req.Sections[i].Components {
			comp := &req.Sections[i].Components[j]
			if (comp.Type == "table_advanced" || isChartType(comp.Type)) && comp.SQL != "" {
				processedSQL, info := processAdvancedSQL(
					comp.SQL,
					req.Filters,
					req.TimeColumns,
					req.LocationColumns,
					componentIndex,
				)
				comp.SQL = processedSQL
				// Clear filterTable as it's no longer needed
				comp.FilterTable = ""
				sqlProcessingInfos = append(sqlProcessingInfos, info)
			}
			componentIndex++
		}
	}

	// Validate required fields
	var validationErrors []string
	if req.Title == "" {
		validationErrors = append(validationErrors, "title is required")
	}
	// Description is optional - keywords can be used instead for search
	if len(req.Sections) == 0 {
		validationErrors = append(validationErrors, "at least one section is required")
	}

	// Validate components
	for i, section := range req.Sections {
		if len(section.Components) == 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("section %d has no components", i+1))
		}
		for j, comp := range section.Components {
			if err := validateComponentForYAML(comp, i+1, j+1); err != "" {
				validationErrors = append(validationErrors, err)
			}
		}
	}

	if len(validationErrors) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "validation failed: "+strings.Join(validationErrors, "; ")))
		return
	}

	// Build report structure with timestamp
	report := Report{
		ID:              req.ID,
		Title:           req.Title,
		Description:     req.Description,
		Keywords:        req.Keywords,
		Category:        req.Category,
		Datasource:      req.Datasource,
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05"),
		Filters:         req.Filters,
		CustomFilters:   req.CustomFilters,
		TimeColumns:     req.TimeColumns,
		LocationColumns: req.LocationColumns,
		Sections:        req.Sections,
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(report)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "failed to generate YAML"))
		return
	}

	// Extract filename from report ID
	filename := extractFilename(req.ID)
	savePath := fmt.Sprintf("configs/%s.yaml", filename)

	response := YAMLGenerateResponse{
		YAML:         string(yamlBytes),
		Filename:     filename + ".yaml",
		SavePath:     savePath,
		SQLProcessed: sqlProcessingInfos,
		Warnings:     ValidateReportWarnings(&report),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ParseYAMLHandler parses YAML string and returns report structure
func ParseYAMLHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "method not allowed"))
		return
	}

	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req YAMLParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(NewErrorResponse(r.Context(), "invalid request"))
		return
	}

	var report Report
	err := yaml.Unmarshal([]byte(req.YAML), &report)

	result := YAMLParseResult{
		Valid: err == nil,
	}

	if err != nil {
		result.Errors = []string{fmt.Sprintf("YAML parse error: %v", err)}
	} else {
		result.Report = report
		// Validate parsed report
		if errs := validateReport(&report); len(errs) > 0 {
			result.Valid = false
			result.Errors = errs
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Helper functions

// extractFilename converts report ID to safe filename
func extractFilename(reportID string) string {
	return reportID
}

// validateReport performs validation checks on a parsed report
func validateReport(report *Report) []string {
	var errors []string

	// Validate ID
	if report.ID == "" {
		errors = append(errors, "report id is required")
	} else if err := ValidateReportID(report.ID); err != nil {
		errors = append(errors, fmt.Sprintf("invalid report id: %v", err))
	}

	// Validate title
	if report.Title == "" {
		errors = append(errors, "report title is required")
	}

	// Description is optional - keywords can be used instead for search discoverability

	// Validate filters
	validFilters := map[string]bool{
		"district":        true,
		"year":            true,
		"month":           true,
		"week":            true,
		"quarter":         true,
		"region":          true,
		"facility":        true,
		"facility_select": true,
	}
	for _, filter := range report.Filters {
		if !validFilters[filter] {
			errors = append(errors, fmt.Sprintf("invalid filter '%s'; must be one of: district, year, month, week, quarter, region, facility, facility_select", filter))
		}
	}

	// Validate sections
	if len(report.Sections) == 0 {
		errors = append(errors, "at least one section is required")
	}

	// Validate total component count
	totalComponents := 0
	for _, section := range report.Sections {
		totalComponents += len(section.Components)
	}
	if totalComponents > MaxComponentsPerReport {
		errors = append(errors, fmt.Sprintf("report has %d components, exceeding maximum of %d", totalComponents, MaxComponentsPerReport))
	}

	for i, section := range report.Sections {
		if section.ID == "" {
			errors = append(errors, fmt.Sprintf("section %d: id is required", i))
		}
		if section.Layout == "" {
			errors = append(errors, fmt.Sprintf("section %d: layout is required", i))
		}
		if len(section.Components) == 0 {
			errors = append(errors, fmt.Sprintf("section %d: at least one component is required", i))
		}

		for j, comp := range section.Components {
			if comp.Type == "" {
				errors = append(errors, fmt.Sprintf("section %d, component %d: type is required", i, j))
			}
			if comp.Type == "text" && comp.Content == "" {
				errors = append(errors, fmt.Sprintf("section %d, component %d: content is required for text type", i, j))
			}
			// For most types, query is required. Exceptions:
			// - text: uses content instead
			// - infobox: static content only
			// - table_advanced: uses sql instead
			// - choropleth with sql: uses sql + columnMapping instead
			// - chart types (bar, line, bar_line, pie, pyramid) with sql: uses raw SQL
			choroplethWithSQL := comp.Type == "choropleth" && comp.SQL != ""
			chartWithSQL := isChartType(comp.Type) && comp.SQL != ""
			kpiWithSQL := comp.Type == "kpi" && comp.SQL != ""
			if comp.Query == nil && comp.Type != "text" && comp.Type != "infobox" && comp.Type != "table_advanced" && !choroplethWithSQL && !chartWithSQL && !kpiWithSQL {
				errors = append(errors, fmt.Sprintf("section %d, component %d: query is required for %s type", i, j, comp.Type))
			}
			if comp.Type == "table_advanced" && comp.SQL == "" {
				errors = append(errors, fmt.Sprintf("section %d, component %d: sql is required for table_advanced type", i, j))
			}
			// Choropleth with SQL requires columnMapping
			if comp.Type == "choropleth" && comp.SQL != "" && comp.ColumnMapping == nil {
				errors = append(errors, fmt.Sprintf("section %d, component %d: columnMapping is required when using sql for choropleth type", i, j))
			}
			// Pyramid requires exactly 2 aggregations and at least 1 groupBy
			if comp.Type == "pyramid" && comp.Query != nil {
				if len(comp.Query.Aggregations) != 2 {
					errors = append(errors, fmt.Sprintf("section %d, component %d: pyramid requires exactly 2 aggregations (left and right datasets), got %d", i, j, len(comp.Query.Aggregations)))
				}
				if len(comp.Query.GroupBy) == 0 {
					errors = append(errors, fmt.Sprintf("section %d, component %d: pyramid requires at least one groupBy (e.g. age group)", i, j))
				}
			}
			if comp.Query != nil && comp.SQL == "" {
				if comp.Query.Table == "" {
					errors = append(errors, fmt.Sprintf("section %d, component %d: query table is required", i, j))
				}
				if len(comp.Query.Aggregations) == 0 && comp.Type != "pyramid" {
					errors = append(errors, fmt.Sprintf("section %d, component %d: at least one aggregation is required", i, j))
				}
			}
		}
	}

	return errors
}
