package main

import (
	"fmt"
	"log"
	"strings"
)

// PublishValidationResult is returned by validateReportForPublish.
// Valid=false blocks publishing; warnings are passed through on success.
type PublishValidationResult struct {
	Valid    bool            `json:"valid"`
	Errors   []PublishError  `json:"errors,omitempty"`
	Warnings []ReportWarning `json:"warnings,omitempty"`
}

// PublishError describes a blocking validation error with optional location context.
type PublishError struct {
	Section   string `json:"section,omitempty"`
	Component string `json:"component,omitempty"`
	Field     string `json:"field,omitempty"`
	Message   string `json:"message"`
}

// validateReportForPublish runs three validation layers before allowing publish:
// 1. Structural checks (no DB calls)
// 2. Schema validation (DB column lookups, cached per table)
// 3. SQL validation (BuildSQLWithSquirrel for structured queries, ValidateAdvancedSQL for raw SQL)
func validateReportForPublish(report *Report) PublishValidationResult {
	var errors []PublishError

	// Layer 1: Structural validation (reuse existing validateReport)
	structuralErrors := validateReport(report)
	for _, msg := range structuralErrors {
		errors = append(errors, PublishError{Message: msg})
	}

	// Layer 2: Schema validation (DB column lookups)
	if DB != nil {
		errors = append(errors, validateSchema(report)...)
	}

	// Layer 3: SQL validation (BuildSQLWithSquirrel / ValidateAdvancedSQL)
	errors = append(errors, validateSQL(report)...)

	// Collect warnings (non-blocking)
	warnings := ValidateReportWarnings(report)

	return PublishValidationResult{
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
}

// validateSchema checks that tables and columns referenced in the report actually exist.
func validateSchema(report *Report) []PublishError {
	var errors []PublishError

	// Build column cache: "schema.table" -> set of column names
	columnCache := make(map[string]map[string]bool)
	lookupColumns := func(fullTable string) map[string]bool {
		if cached, ok := columnCache[fullTable]; ok {
			return cached
		}
		schema, table := splitSchemaTable(fullTable)
		if schema == "" || table == "" {
			return nil
		}
		cols, err := GetTableColumns(schema, table)
		if err != nil {
			log.Printf("publish validation: GetTableColumns(%s, %s) error: %v", schema, table, err)
			columnCache[fullTable] = nil
			return nil
		}
		colSet := make(map[string]bool, len(cols))
		for _, c := range cols {
			colSet[c.Name] = true
		}
		columnCache[fullTable] = colSet
		return colSet
	}

	// Validate components
	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			loc := func(field, msg string) PublishError {
				return PublishError{
					Section:   section.Title,
					Component: comp.Title,
					Field:     field,
					Message:   fmt.Sprintf("Section %d, Component %d: %s", si+1, ci+1, msg),
				}
			}

			// Components with structured query
			if comp.Query != nil && comp.Query.Table != "" {
				cols := lookupColumns(comp.Query.Table)
				if cols == nil {
					errors = append(errors, loc("query.table",
						fmt.Sprintf("table %q does not exist or is not accessible", comp.Query.Table)))
					continue // skip column checks for missing table
				}

				// Check aggregation columns (skip expressions)
				for ai, agg := range comp.Query.Aggregations {
					if !containsOperators(agg.Column) && !cols[agg.Column] {
						errors = append(errors, loc(
							fmt.Sprintf("aggregations[%d].column", ai),
							fmt.Sprintf("column %q not found in %s", agg.Column, comp.Query.Table)))
					}
					if agg.Alias == "" {
						errors = append(errors, loc(
							fmt.Sprintf("aggregations[%d].alias", ai),
							"aggregation alias is required"))
					}
				}

				// Check groupBy fields
				for gi, gb := range comp.Query.GroupBy {
					if !cols[gb.Field] {
						errors = append(errors, loc(
							fmt.Sprintf("groupBy[%d].field", gi),
							fmt.Sprintf("column %q not found in %s", gb.Field, comp.Query.Table)))
					}
				}
			}

			// Components with raw SQL — check filterTable exists
			if comp.SQL != "" && comp.FilterTable != "" {
				cols := lookupColumns(comp.FilterTable)
				if cols == nil {
					errors = append(errors, loc("filterTable",
						fmt.Sprintf("filterTable %q does not exist or is not accessible", comp.FilterTable)))
				}
			}

			// For raw SQL components without an explicit filterTable, auto-detect
			// the primary table from the SQL so its columns are included when
			// validating report-level timeColumns / locationColumns references.
			if comp.SQL != "" && comp.FilterTable == "" {
				if schema, tbl, found := extractTableFromSQL(comp.SQL); found {
					lookupColumns(schema + "." + tbl) // populates cache; errors silently ignored
				}
			}
		}
	}

	// Validate report-level column references against any component table
	allTableCols := collectAllTableColumns(columnCache)

	// timeColumns
	checkReportColumn := func(filterName, colName string) {
		if colName == "" {
			return
		}
		if !allTableCols[colName] {
			errors = append(errors, PublishError{
				Field:   "timeColumns." + filterName,
				Message: fmt.Sprintf("timeColumns.%s references column %q which was not found in any component table", filterName, colName),
			})
		}
	}
	checkReportColumn("year", report.TimeColumns.Year)
	checkReportColumn("month", report.TimeColumns.Month)
	checkReportColumn("week", report.TimeColumns.Week)
	checkReportColumn("quarter", report.TimeColumns.Quarter)

	// locationColumns
	checkLocColumn := func(filterName, colName string) {
		if colName == "" {
			return
		}
		if !allTableCols[colName] {
			errors = append(errors, PublishError{
				Field:   "locationColumns." + filterName,
				Message: fmt.Sprintf("locationColumns.%s references column %q which was not found in any component table", filterName, colName),
			})
		}
	}
	checkLocColumn("district", report.LocationColumns.District)
	checkLocColumn("region", report.LocationColumns.Region)
	checkLocColumn("facility", report.LocationColumns.Facility)

	// customFilters
	for i, cf := range report.CustomFilters {
		if cf.Table != "" {
			cols := lookupColumns(cf.Table)
			if cols == nil {
				errors = append(errors, PublishError{
					Field:   fmt.Sprintf("customFilters[%d].table", i),
					Message: fmt.Sprintf("customFilter table %q does not exist", cf.Table),
				})
			} else if !cols[cf.Column] {
				errors = append(errors, PublishError{
					Field:   fmt.Sprintf("customFilters[%d].column", i),
					Message: fmt.Sprintf("customFilter column %q not found in %s", cf.Column, cf.Table),
				})
			}
		}
	}

	if report.FilterOptionsTable != "" {
		if !validTableFormat.MatchString(report.FilterOptionsTable) {
			errors = append(errors, PublishError{
				Field:   "filterOptionsTable",
				Message: fmt.Sprintf("filterOptionsTable %q has invalid format", report.FilterOptionsTable),
			})
		} else {
			cols := lookupColumns(report.FilterOptionsTable)
			if cols == nil {
				errors = append(errors, PublishError{
					Field:   "filterOptionsTable",
					Message: fmt.Sprintf("filterOptionsTable %q does not exist or is not accessible", report.FilterOptionsTable),
				})
			}
		}
	}

	return errors
}

// validateSQL tries to build SQL for structured queries and validates raw SQL safety.
func validateSQL(report *Report) []PublishError {
	var errors []PublishError

	for si, section := range report.Sections {
		for ci, comp := range section.Components {
			// Structured query validation
			if comp.Query != nil && comp.Query.Table != "" && len(comp.Query.Aggregations) > 0 {
				_, err := BuildSQLWithSquirrel(*comp.Query, comp.Type, comp.Query.Table, report.Filters)
				if err != nil {
					errors = append(errors, PublishError{
						Section:   section.Title,
						Component: comp.Title,
						Message:   fmt.Sprintf("Section %d, Component %d: SQL build failed: %s", si+1, ci+1, err.Error()),
					})
				}
			}

			// Raw SQL safety check (no DB needed)
			if comp.SQL != "" {
				if err := ValidateAdvancedSQL(comp.SQL); err != nil {
					errors = append(errors, PublishError{
						Section:   section.Title,
						Component: comp.Title,
						Field:     "sql",
						Message:   fmt.Sprintf("Section %d, Component %d: %s", si+1, ci+1, err.Error()),
					})
				}
			}
		}
	}

	return errors
}

// splitSchemaTable splits "schema.table" into ("schema", "table").
// Returns ("", "") if the format is invalid.
func splitSchemaTable(fullTable string) (string, string) {
	parts := strings.SplitN(fullTable, ".", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// collectAllTableColumns unions all column names from the cache into a single set.
func collectAllTableColumns(cache map[string]map[string]bool) map[string]bool {
	all := make(map[string]bool)
	for _, cols := range cache {
		for col := range cols {
			all[col] = true
		}
	}
	return all
}
