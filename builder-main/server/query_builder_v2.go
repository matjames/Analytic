// query_builder_v2.go - SQL query builder using Squirrel
package main

import (
	"fmt"
	"regexp"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// validAggFunctions is the whitelist of allowed SQL aggregation functions.
var validAggFunctions = map[string]bool{
	"SUM": true, "COUNT": true, "COUNT_DISTINCT": true, "AVG": true, "MAX": true, "MIN": true,
	"VARIANCE": true, "STDDEV": true,
}

// selectKeywordRe matches SQL SELECT keyword (case-insensitive) to block subqueries in WHERE clauses.
var selectKeywordRe = regexp.MustCompile(`(?i)\bSELECT\b`)

// validateWhereClause checks for dangerous SQL patterns and balanced parentheses
func validateWhereClause(where string) error {
	if where == "" {
		return nil
	}

	// Block statement separators (prevents multiple statements)
	if strings.Contains(where, ";") {
		return fmt.Errorf("where clause cannot contain semicolons")
	}

	// Block SQL comments (could hide malicious code)
	if strings.Contains(where, "--") {
		return fmt.Errorf("where clause cannot contain SQL comments (--)")
	}

	// Block subqueries (use database views instead)
	if selectKeywordRe.MatchString(where) {
		return fmt.Errorf("where clause cannot contain subqueries (SELECT)")
	}

	// Check balanced parentheses
	count := 0
	for _, ch := range where {
		if ch == '(' {
			count++
		} else if ch == ')' {
			count--
		}
		if count < 0 {
			return fmt.Errorf("where clause has unbalanced parentheses")
		}
	}
	if count != 0 {
		return fmt.Errorf("where clause has unbalanced parentheses")
	}

	return nil
}

// BuildSQLWithSquirrel generates SQL from a structured Query object using Squirrel
// This is the main entry point for the new Squirrel-based query builder
// filters: list of filter names to include in WHERE clause (e.g., ["district", "year", "month"])
func BuildSQLWithSquirrel(query Query, componentType string, tableName string, filters []string) (string, error) {
	if query.Table == "" {
		return "", fmt.Errorf("query table is required")
	}

	if len(query.Aggregations) == 0 {
		return "", fmt.Errorf("query must have at least one aggregation")
	}

	// Start building with Squirrel
	builder := sq.Select()

	// Add SELECT columns (before grouping expressions)
	selectCols, err := buildSelectColumns(query.Aggregations, query.GroupBy, query.Calculate, tableName)
	if err != nil {
		return "", err
	}

	// If seriesBy is set, insert the series column after groupBy labels but before aggregations
	if query.SeriesBy != "" {
		labelCount := len(query.GroupBy)
		seriesCol := fmt.Sprintf("%s AS %s", quoteIdentifier(query.SeriesBy), quoteIdentifier("series"))
		// Insert after groupBy labels
		newCols := make([]string, 0, len(selectCols)+1)
		newCols = append(newCols, selectCols[:labelCount]...)
		newCols = append(newCols, seriesCol)
		newCols = append(newCols, selectCols[labelCount:]...)
		selectCols = newCols
	}

	for _, col := range selectCols {
		builder = builder.Column(col)
	}

	// Add FROM clause - quote table name to preserve case
	quotedTable := quoteTableName(query.Table)
	builder = builder.From(quotedTable)

	// Add WHERE clause with filter placeholders
	builder = builder.Where("1=1 {{where_placeholder}}")

	// Add custom WHERE clause if provided
	if query.Where != "" {
		if err := validateWhereClause(query.Where); err != nil {
			return "", fmt.Errorf("invalid where clause: %w", err)
		}
		builder = builder.Where(sq.Expr(query.Where))
	}

	// Add GROUP BY if needed
	if len(query.GroupBy) > 0 {
		groupCols := buildGroupByColumns(query.GroupBy)
		for _, col := range groupCols {
			builder = builder.GroupBy(col)
		}
	}

	// Add seriesBy column to GROUP BY
	if query.SeriesBy != "" {
		builder = builder.GroupBy(quoteIdentifier(query.SeriesBy))
	}

	// Add ORDER BY if needed
	if len(query.GroupBy) > 0 || len(query.OrderBy) > 0 {
		orderCols := buildOrderByColumns(query.GroupBy, query.OrderBy)
		for _, col := range orderCols {
			builder = builder.OrderBy(col)
		}
	}

	// ✅ Add LIMIT if provided
	if query.Limit > 0 {
		builder = builder.Limit(uint64(query.Limit))
	}

	// Generate SQL from Squirrel builder
	// Use PlaceholderFormat for PostgreSQL $1, $2, etc. syntax
	sql, _, err := builder.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return "", fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Apply custom transformations to the generated SQL
	sql = applyDateFormatting(sql, query.GroupBy)
	sql = applyCalculatedFields(sql, query.Aggregations, query.Calculate, tableName)

	// Replace Squirrel placeholders back to our filter placeholders
	sql = strings.ReplaceAll(sql, "$1=$1", "1=1") // Replace Squirrel's "$1=$1" back to "WHERE 1=1"
	sql = strings.ReplaceAll(sql, " {{where_placeholder}}", " {{district_filter}} {{region_filter}} {{facility_filter}} {{year_filter}} {{month_filter}} {{quarter_filter}}")

	// Apply report-level filter specification (also handles empty/nil filters by removing all placeholders)
	sql = applyReportLevelFilters(sql, filters)

	// Log the generated SQL
	LogGeneratedSQL(sql, query.Table)

	return sql, nil
}

// buildSelectColumns returns list of SELECT columns as strings
// Handles regular aggregations and calculated fields
func buildSelectColumns(aggregations []Aggregation, groupBy []GroupBy, calculations []Calculate, tableName string) ([]string, error) {
	var cols []string

	// Add all groupBy columns to SELECT (needed for pivot and multi-grouping)
	for i, gb := range groupBy {
		labelExpr := getGroupByExpressionForSelect(gb)
		// Use custom alias if provided, otherwise default to "label" for first, "label2" etc for others
		alias := gb.Alias
		if alias == "" {
			if i == 0 {
				alias = "label"
			} else {
				alias = fmt.Sprintf("label%d", i+1)
			}
		}
		cols = append(cols, fmt.Sprintf("%s AS %s", labelExpr, quoteIdentifier(alias)))
	}

	// Always add all regular aggregations (needed for ordering and calculations)
	for _, agg := range aggregations {
		var aggExpr string
		funcUpper := strings.ToUpper(agg.Function)

		if !validAggFunctions[funcUpper] {
			return nil, fmt.Errorf("invalid aggregation function '%s'; must be one of: SUM, COUNT, COUNT_DISTINCT, AVG, MAX, MIN, VARIANCE, STDDEV", agg.Function)
		}

		if funcUpper == "COUNT" {
			// COUNT: no numeric cast, skip empty strings
			wrappedCol := wrapColumnForCount(agg.Column)
			aggExpr = fmt.Sprintf("COUNT(%s) AS %s", wrappedCol, quoteIdentifier(agg.Alias))
		} else if funcUpper == "COUNT_DISTINCT" {
			// COUNT_DISTINCT: like COUNT but with DISTINCT keyword
			wrappedCol := wrapColumnForCount(agg.Column)
			aggExpr = fmt.Sprintf("COUNT(DISTINCT %s) AS %s", wrappedCol, quoteIdentifier(agg.Alias))
		} else {
			// SUM/AVG/MAX/MIN: numeric cast required
			wrappedCol := wrapNumericColumnSquirrel(agg.Column, tableName)
			if agg.RoundTo != nil {
				aggExpr = fmt.Sprintf("ROUND(%s(%s)::numeric, %d) AS %s", funcUpper, wrappedCol, *agg.RoundTo, quoteIdentifier(agg.Alias))
			} else {
				aggExpr = fmt.Sprintf("%s(%s) AS %s", funcUpper, wrappedCol, quoteIdentifier(agg.Alias))
			}
		}
		cols = append(cols, aggExpr)
	}

	// Add all calculated fields
	for _, calc := range calculations {
		calcCol := buildCalculatedFieldColumn(calc, aggregations, tableName)
		cols = append(cols, calcCol)
	}

	return cols, nil
}

// buildGroupByColumns returns list of columns for GROUP BY clause
func buildGroupByColumns(groupBy []GroupBy) []string {
	var cols []string

	for _, gb := range groupBy {
		expr := getGroupByExpression(gb)
		cols = append(cols, expr)

		// For time-based grouping, add columns for chronological ORDER BY
		// Detect if field is a date column (contains "date") vs direct column
		isDateField := strings.Contains(strings.ToLower(gb.Field), "date")

		// Quote the field name for use in SQL
		quotedField := quoteIdentifier(gb.Field)

		// Use GetFormat() to infer format from field name when not explicitly set
		format := gb.GetFormat()

		if format == "month" || format == "month_year" {
			if isDateField {
				cols = append(cols, fmt.Sprintf("EXTRACT(YEAR FROM %s::date)", gb.Field))
				cols = append(cols, fmt.Sprintf("EXTRACT(MONTH FROM %s::date)", gb.Field))
			} else {
				cols = append(cols, quoteIdentifier(gb.GetYearField()))
				cols = append(cols, quotedField)
			}
		} else if format == "month_noyear" {
			// Explicit opt-out: group by month only, merging across years
			if isDateField {
				cols = append(cols, fmt.Sprintf("EXTRACT(MONTH FROM %s::date)", gb.Field))
			} else {
				cols = append(cols, quotedField)
			}
		} else if format == "week" || format == "week_year" {
			if isDateField {
				cols = append(cols, fmt.Sprintf("EXTRACT(ISOYEAR FROM %s::date)", gb.Field))
				cols = append(cols, fmt.Sprintf("EXTRACT(WEEK FROM %s::date)", gb.Field))
			} else {
				cols = append(cols, quoteIdentifier(gb.GetYearField()))
				cols = append(cols, quotedField)
			}
		} else if format == "week_noyear" {
			if isDateField {
				cols = append(cols, fmt.Sprintf("EXTRACT(WEEK FROM %s::date)", gb.Field))
			} else {
				cols = append(cols, quotedField)
			}
		} else if format == "quarter" || format == "quarter_year" {
			if isDateField {
				cols = append(cols, fmt.Sprintf("EXTRACT(YEAR FROM %s::date)", gb.Field))
				cols = append(cols, fmt.Sprintf("EXTRACT(QUARTER FROM %s::date)", gb.Field))
			} else {
				cols = append(cols, quoteIdentifier(gb.GetYearField()))
				cols = append(cols, quotedField)
			}
		} else if format == "quarter_noyear" {
			if isDateField {
				cols = append(cols, fmt.Sprintf("EXTRACT(QUARTER FROM %s::date)", gb.Field))
			} else {
				cols = append(cols, quotedField)
			}
		} else if format == "year" {
			if isDateField {
				cols = append(cols, fmt.Sprintf("EXTRACT(YEAR FROM %s::date)", gb.Field))
			} else {
				cols = append(cols, quoteIdentifier(gb.GetYearField()))
			}
		}
	}

	return cols
}

// buildOrderByColumns returns list of columns for ORDER BY clause
func buildOrderByColumns(groupBy []GroupBy, orderBy []OrderBy) []string {
	var cols []string

	// Helper: find groupBy entry matching a format to resolve yearField and actual field name
	findGroupBy := func(format string) *GroupBy {
		for i := range groupBy {
			if strings.EqualFold(groupBy[i].GetFormat(), format) {
				return &groupBy[i]
			}
		}
		// Fallback: return first groupBy if available
		if len(groupBy) > 0 {
			return &groupBy[0]
		}
		return nil
	}

	resolveYearCol := func(format string) string {
		if gb := findGroupBy(format); gb != nil {
			return quoteIdentifier(gb.GetYearField())
		}
		return quoteIdentifier("year")
	}

	resolvePeriodCol := func(format string, fallback string) string {
		if gb := findGroupBy(format); gb != nil {
			return quoteIdentifier(gb.Field)
		}
		return quoteIdentifier(fallback)
	}

	// Use explicit OrderBy if provided
	if len(orderBy) > 0 {
		for _, ob := range orderBy {
			direction := "ASC"
			if strings.ToUpper(ob.Direction) == "DESC" {
				direction = "DESC"
			}

			// Chronological ordering support for time-based fields
			// Ordering must be by year+period numeric key, not the label
			if strings.EqualFold(ob.Field, "month_year") || strings.EqualFold(ob.Field, "month") {
				yearCol := resolveYearCol(ob.Field)
				periodCol := resolvePeriodCol(ob.Field, "month")
				cols = append(cols, fmt.Sprintf("(%s::int * 100 + %s::int) %s", yearCol, periodCol, direction))
				continue
			}
			if strings.EqualFold(ob.Field, "month_noyear") {
				periodCol := resolvePeriodCol(ob.Field, "month")
				cols = append(cols, fmt.Sprintf("%s::int %s", periodCol, direction))
				continue
			}
			if strings.EqualFold(ob.Field, "week") || strings.EqualFold(ob.Field, "week_year") {
				yearCol := resolveYearCol(ob.Field)
				periodCol := resolvePeriodCol(ob.Field, "week")
				cols = append(cols, fmt.Sprintf("(%s::int * 100 + %s::int) %s", yearCol, periodCol, direction))
				continue
			}
			if strings.EqualFold(ob.Field, "week_noyear") {
				periodCol := resolvePeriodCol(ob.Field, "week")
				cols = append(cols, fmt.Sprintf("%s::int %s", periodCol, direction))
				continue
			}
			if strings.EqualFold(ob.Field, "quarter_year") || strings.EqualFold(ob.Field, "quarter") {
				yearCol := resolveYearCol(ob.Field)
				periodCol := resolvePeriodCol(ob.Field, "quarter")
				cols = append(cols, fmt.Sprintf("(%s::int * 10 + %s::int) %s", yearCol, periodCol, direction))
				continue
			}
			if strings.EqualFold(ob.Field, "quarter_noyear") {
				periodCol := resolvePeriodCol(ob.Field, "quarter")
				cols = append(cols, fmt.Sprintf("%s::int %s", periodCol, direction))
				continue
			}

			// Quote the field name to handle spaces and special characters
			quotedField := quoteIdentifier(ob.Field)
			cols = append(cols, fmt.Sprintf("%s %s", quotedField, direction))
		}
	} else if len(groupBy) > 0 {
		// Default: Order by chronological columns for time-series charts
		// Detect if field is a date column vs direct column
		for _, gb := range groupBy {
			isDateField := strings.Contains(strings.ToLower(gb.Field), "date")
			// Quote the field name for use in SQL
			quotedField := quoteIdentifier(gb.Field)

			// month or month_year format: chronological by year+month
			if strings.EqualFold(gb.GetFormat(), "month_year") || strings.EqualFold(gb.GetFormat(), "month") {
				if isDateField {
					cols = append(cols, fmt.Sprintf("(EXTRACT(YEAR FROM %s::date) * 100 + EXTRACT(MONTH FROM %s::date)) ASC", gb.Field, gb.Field))
				} else {
					yearCol := quoteIdentifier(gb.GetYearField())
					cols = append(cols, fmt.Sprintf("(%s::int * 100 + %s::int) ASC", yearCol, quotedField))
				}
				continue
			}

			// month_noyear: just by month number
			if strings.EqualFold(gb.GetFormat(), "month_noyear") {
				if isDateField {
					cols = append(cols, fmt.Sprintf("EXTRACT(MONTH FROM %s::date) ASC", gb.Field))
				} else {
					cols = append(cols, fmt.Sprintf("%s::int ASC", quotedField))
				}
				continue
			}

			// week format: chronological by year+week
			if strings.EqualFold(gb.GetFormat(), "week") || strings.EqualFold(gb.GetFormat(), "week_year") {
				if isDateField {
					cols = append(cols, fmt.Sprintf("(EXTRACT(ISOYEAR FROM %s::date) * 100 + EXTRACT(WEEK FROM %s::date)) ASC", gb.Field, gb.Field))
				} else {
					yearCol := quoteIdentifier(gb.GetYearField())
					cols = append(cols, fmt.Sprintf("(%s::int * 100 + %s::int) ASC", yearCol, quotedField))
				}
				continue
			}

			// week_noyear: just by week number
			if strings.EqualFold(gb.GetFormat(), "week_noyear") {
				if isDateField {
					cols = append(cols, fmt.Sprintf("EXTRACT(WEEK FROM %s::date) ASC", gb.Field))
				} else {
					cols = append(cols, fmt.Sprintf("%s::int ASC", quotedField))
				}
				continue
			}

			// quarter format: chronological by year+quarter
			if strings.EqualFold(gb.GetFormat(), "quarter") || strings.EqualFold(gb.GetFormat(), "quarter_year") {
				if isDateField {
					cols = append(cols, fmt.Sprintf("(EXTRACT(YEAR FROM %s::date) * 10 + EXTRACT(QUARTER FROM %s::date)) ASC", gb.Field, gb.Field))
				} else {
					yearCol := quoteIdentifier(gb.GetYearField())
					cols = append(cols, fmt.Sprintf("(%s::int * 10 + %s::int) ASC", yearCol, quotedField))
				}
				continue
			}

			// quarter_noyear: just by quarter number
			if strings.EqualFold(gb.GetFormat(), "quarter_noyear") {
				if isDateField {
					cols = append(cols, fmt.Sprintf("EXTRACT(QUARTER FROM %s::date) ASC", gb.Field))
				} else {
					cols = append(cols, fmt.Sprintf("%s::int ASC", quotedField))
				}
				continue
			}

			// year format: just by year
			if strings.EqualFold(gb.GetFormat(), "year") {
				if isDateField {
					cols = append(cols, fmt.Sprintf("EXTRACT(YEAR FROM %s::date) ASC", gb.Field))
				} else {
					cols = append(cols, quoteIdentifier(gb.GetYearField())+" ASC")
				}
				continue
			}

			// Default: use the field as-is
			cols = append(cols, quoteIdentifier(gb.Field))
		}
	}

	return cols
}

// getGroupByExpressionForSelect returns GROUP BY expression formatted for SELECT clause
// Uses the existing getGroupByExpression from query_builder.go
func getGroupByExpressionForSelect(gb GroupBy) string {
	return getGroupByExpression(gb)
}

// wrapNumericColumnSquirrel applies universal safe wrapping to all aggregated columns
// Pattern: COALESCE(CAST(NULLIF(TRIM(CAST("column" AS TEXT)), ”) AS NUMERIC), 0)
// - CAST to TEXT: Enables TRIM for both TEXT and numeric columns
// - TRIM: Removes leading/trailing whitespace (handles spaces, tabs, newlines)
// - NULLIF: Converts empty strings to NULL
// - CAST to NUMERIC: Forces NUMERIC type conversion (preserves decimal values)
// - COALESCE: Defaults NULL to 0
// Works universally for TEXT columns, INTEGER columns, BIGINT, NUMERIC, and all other data types.
// Now supports BODMAS (operator precedence) for expressions like "col1 * col2 + col3"
func wrapNumericColumnSquirrel(column string, tableName string) string {
	// Check if expression contains operators (BODMAS support)
	if containsOperators(column) {
		wrapped, err := ParseColumnExpression(column, tableName)
		if err != nil {
			// Fallback to legacy behavior for backward compatibility
			logWarn("Expression parse error for '%s': %v. Using legacy fallback.", column, err)
			return wrapNumericColumnLegacy(column, tableName)
		}
		return wrapped
	}

	// Single column: wrap with COALESCE/CAST/NULLIF/TRIM
	quotedColumn := quoteIdentifier(column)
	return fmt.Sprintf("COALESCE(CAST(NULLIF(TRIM(CAST(%s AS TEXT)), '') AS NUMERIC), 0)", quotedColumn)
}

// wrapColumnForCount wraps a column for COUNT aggregation
// - Skips empty strings (NULLIF treats ” as NULL, which COUNT ignores)
// - No NUMERIC cast (works with text, numeric, any type)
// - Does NOT support expressions (only single columns or *)
func wrapColumnForCount(column string) string {
	col := strings.TrimSpace(column)
	if col == "*" {
		return "*"
	}
	// NULLIF(TRIM(...), '') converts empty/whitespace to NULL
	// COUNT then skips these NULL values
	return fmt.Sprintf("NULLIF(TRIM(CAST(%s AS TEXT)), '')", quoteIdentifier(col))
}

// containsOperators checks if an expression contains mathematical operators or parentheses
func containsOperators(expr string) bool {
	return strings.ContainsAny(expr, "+-*/()")
}

// wrapNumericColumnLegacy is the original simple implementation for addition-only expressions
// Used as fallback when new parser encounters an error
func wrapNumericColumnLegacy(column string, tableName string) string {
	// Handle column expressions like "col1 + col2"
	if strings.Contains(column, "+") {
		parts := strings.Split(column, "+")
		var wrappedParts []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			quotedPart := quoteIdentifier(part)
			wrapped := fmt.Sprintf("COALESCE(CAST(NULLIF(TRIM(CAST(%s AS TEXT)), '') AS NUMERIC), 0)", quotedPart)
			wrappedParts = append(wrappedParts, wrapped)
		}
		return strings.Join(wrappedParts, " + ")
	}

	// Single column: wrap with COALESCE/CAST/NULLIF/TRIM
	quotedColumn := quoteIdentifier(column)
	return fmt.Sprintf("COALESCE(CAST(NULLIF(TRIM(CAST(%s AS TEXT)), '') AS NUMERIC), 0)", quotedColumn)
}

// applyDateFormatting enhances SQL with date formatting expressions
// Ensures TO_CHAR generates proper labels and EXTRACT enables proper ordering
func applyDateFormatting(sql string, groupBy []GroupBy) string {
	// The grouping expressions are already generated in buildGroupByColumns
	// This function is a hook for future enhancements to date formatting
	return sql
}

// buildCalculatedFieldColumn generates a calculated field with the given formula
// Handles division-by-zero protection and rounding
func buildCalculatedFieldColumn(calc Calculate, aggregations []Aggregation, tableName string) string {
	formula := calc.Formula

	// Replace aggregation aliases with their full expressions
	// Wrap with CAST to NUMERIC for expressions that contain division to ensure floating-point math
	hasDivision := strings.Contains(formula, "/")
	for _, agg := range aggregations {
		funcUpper := strings.ToUpper(agg.Function)
		var aggExpr string

		if funcUpper == "COUNT" {
			wrappedColumn := wrapColumnForCount(agg.Column)
			aggExpr = fmt.Sprintf("COUNT(%s)", wrappedColumn)
		} else if funcUpper == "COUNT_DISTINCT" {
			wrappedColumn := wrapColumnForCount(agg.Column)
			aggExpr = fmt.Sprintf("COUNT(DISTINCT %s)", wrappedColumn)
		} else {
			wrappedColumn := wrapNumericColumnSquirrel(agg.Column, tableName)
			aggExpr = fmt.Sprintf("%s(%s)", funcUpper, wrappedColumn)
		}

		// For division operations, cast aggregation to NUMERIC to ensure floating-point arithmetic
		if hasDivision {
			aggExpr = fmt.Sprintf("CAST(%s AS NUMERIC)", aggExpr)
		}

		// Replace the alias in the formula with the aggregation expression (word-boundary aware)
		formula = replaceWordBoundary(formula, agg.Alias, aggExpr)
	}

	// Determine the result column alias (use resultAlias if provided, otherwise default to "value")
	resultAlias := calc.ResultAlias
	if resultAlias == "" {
		resultAlias = "value"
	}
	quotedAlias := quoteIdentifier(resultAlias)

	var calculatedField string
	if hasDivision {
		// Extract denominator for the zero-check
		// Find which aggregation alias appears after "/" in the formula
		if len(aggregations) >= 2 {
			denomIdx := 1 // default fallback
			if slashIdx := strings.Index(calc.Formula, "/"); slashIdx >= 0 {
				denomPart := calc.Formula[slashIdx+1:]
				for i, agg := range aggregations {
					if strings.Contains(denomPart, agg.Alias) {
						denomIdx = i
						break
					}
				}
			}
			denomAgg := aggregations[denomIdx]
			wrappedDenom := wrapNumericColumnSquirrel(denomAgg.Column, tableName)
			denomExpr := fmt.Sprintf("%s(%s)", strings.ToUpper(denomAgg.Function), wrappedDenom)

			whenZeroValue := 0
			if calc.WhenZero != nil {
				switch v := calc.WhenZero.(type) {
				case float64:
					whenZeroValue = int(v)
				case int:
					whenZeroValue = v
				}
			}

			var roundClause string
			if calc.RoundTo != nil {
				roundClause = fmt.Sprintf("ROUND(%s, %d)", formula, *calc.RoundTo)
			} else {
				roundClause = formula
			}

			calculatedField = fmt.Sprintf(
				"CASE WHEN %s > 0 THEN %s ELSE %d END AS %s",
				denomExpr,
				roundClause,
				whenZeroValue,
				quotedAlias,
			)
		} else {
			// Single-aggregation division (e.g. "anc4/1"): no denominator to
			// zero-check, but still honor roundTo so PG doesn't return full
			// NUMERIC precision (e.g. 63.6000000000000000).
			if calc.RoundTo != nil {
				calculatedField = fmt.Sprintf("ROUND(%s, %d) AS %s", formula, *calc.RoundTo, quotedAlias)
			} else {
				calculatedField = fmt.Sprintf("%s AS %s", formula, quotedAlias)
			}
		}
	} else {
		// No division, just apply rounding if needed
		if calc.RoundTo != nil {
			calculatedField = fmt.Sprintf("ROUND(%s, %d) AS %s", formula, *calc.RoundTo, quotedAlias)
		} else {
			calculatedField = fmt.Sprintf("%s AS %s", formula, quotedAlias)
		}
	}

	return calculatedField
}

// applyCalculatedFields enhances SQL with calculated field expressions
// This is a hook for future enhancements to calculated field handling
func applyCalculatedFields(sql string, aggregations []Aggregation, calculations []Calculate, tableName string) string {
	// Calculated fields are already generated in buildCalculatedFieldColumn
	// This function is a hook for future post-processing if needed
	return sql
}

// applyReportLevelFilters customizes filter placeholders based on report configuration
// Only includes specified filters (district, region, facility, year, month, week, quarter) in the WHERE clause
func applyReportLevelFilters(sql string, filters []string) string {
	filterMap := make(map[string]bool)
	for _, f := range filters {
		filterMap[strings.ToLower(f)] = true
	}

	var filterParts []string
	if filterMap["district"] {
		filterParts = append(filterParts, "{{district_filter}}")
	}
	if filterMap["region"] {
		filterParts = append(filterParts, "{{region_filter}}")
	}
	if filterMap["facility"] {
		filterParts = append(filterParts, "{{facility_filter}}")
	}
	if filterMap["year"] {
		filterParts = append(filterParts, "{{year_filter}}")
	}
	if filterMap["month"] {
		filterParts = append(filterParts, "{{month_filter}}")
	}
	if filterMap["week"] {
		filterParts = append(filterParts, "{{week_filter}}")
	}
	if filterMap["quarter"] {
		filterParts = append(filterParts, "{{quarter_filter}}")
	}

	// Pattern to match the full filter placeholder string
	allFiltersPlaceholder := " {{district_filter}} {{region_filter}} {{facility_filter}} {{year_filter}} {{month_filter}} {{quarter_filter}}"

	if len(filterParts) > 0 {
		replacement := " " + strings.Join(filterParts, " ")
		sql = strings.ReplaceAll(sql, allFiltersPlaceholder, replacement)
	} else {
		// No filters specified - remove all filter placeholders
		sql = strings.ReplaceAll(sql, allFiltersPlaceholder, "")
	}

	return sql
}

// isWordChar returns true if the byte is a word character (alphanumeric or underscore)
func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// replaceWordBoundary replaces all whole-word occurrences of old with new in s.
// Avoids compiling a regex per call.
func replaceWordBoundary(s, old, newStr string) string {
	if old == "" {
		return s
	}
	var result strings.Builder
	result.Grow(len(s))
	for i := 0; i < len(s); {
		idx := strings.Index(s[i:], old)
		if idx == -1 {
			result.WriteString(s[i:])
			break
		}
		absIdx := i + idx
		// Check word boundaries
		leftOK := absIdx == 0 || !isWordChar(s[absIdx-1])
		rightOK := absIdx+len(old) == len(s) || !isWordChar(s[absIdx+len(old)])
		if leftOK && rightOK {
			result.WriteString(s[i:absIdx])
			result.WriteString(newStr)
		} else {
			result.WriteString(s[i : absIdx+len(old)])
		}
		i = absIdx + len(old)
	}
	return result.String()
}

// quoteTableName properly quotes a schema.table name to preserve case.
// Escapes embedded double quotes per SQL standard.
func quoteTableName(table string) string {
	parts := strings.Split(table, ".")
	if len(parts) == 2 {
		return `"` + strings.ReplaceAll(parts[0], `"`, `""`) + `"."` + strings.ReplaceAll(parts[1], `"`, `""`) + `"`
	}
	return `"` + strings.ReplaceAll(table, `"`, `""`) + `"`
}
