package main

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildSQL delegates to the Squirrel-based query builder (query_builder_v2.go)
// filters: list of filter names to include in WHERE clause (e.g., ["district", "year", "month"])
// If filters is empty/nil, all three filter placeholders are included (backward compatibility)
func BuildSQL(query Query, componentType string, tableName string, filters []string) (string, error) {
	return BuildSQLWithSquirrel(query, componentType, tableName, filters)
}

// BuildSQLLegacy is the original implementation kept for fallback and reference
// This preserves the manual SQL building approach if the Squirrel builder needs to be bypassed
func BuildSQLLegacy(query Query, componentType string, tableName string, filters []string) (string, error) {
	if query.Table == "" {
		return "", fmt.Errorf("query table is required")
	}

	if len(query.Aggregations) == 0 {
		return "", fmt.Errorf("query must have at least one aggregation")
	}

	// Build SELECT clause
	selectClause, err := buildSelectClause(query.Aggregations, query.GroupBy, query.Calculate, tableName)
	if err != nil {
		return "", err
	}

	// Build FROM clause
	fromClause := fmt.Sprintf("FROM \"%s\"", query.Table)

	// Build WHERE clause with only the specified filters
	whereClause := buildWhereClause(filters)

	// Build GROUP BY clause (if grouping)
	groupByClause := ""
	if len(query.GroupBy) > 0 {
		groupByClause = buildGroupByClause(query.GroupBy)
	}

	// Build ORDER BY clause (if grouping or explicit ordering)
	orderByClause := ""
	if len(query.OrderBy) > 0 || len(query.GroupBy) > 0 {
		orderByClause = buildOrderByClause(query.GroupBy, query.OrderBy)
	}

	// Assemble SQL
	sql := fmt.Sprintf("%s\n%s\n%s", selectClause, fromClause, whereClause)
	if groupByClause != "" {
		sql += "\n" + groupByClause
	}
	if orderByClause != "" {
		sql += "\n" + orderByClause
	}

	LogGeneratedSQL(sql, query.Table)
	logDebug("SQL %s", sql)

	return sql, nil
}

// buildSelectClause generates the SELECT portion of the SQL query
func buildSelectClause(aggregations []Aggregation, groupBy []GroupBy, calculations []Calculate, tableName string) (string, error) {
	var selectParts []string

	// Add label/grouping columns first (if grouping)
	if len(groupBy) > 0 {
		for _, gb := range groupBy {
			labelExpr := getGroupByExpression(gb)
			selectParts = append(selectParts, fmt.Sprintf("%s as label", labelExpr))
			break // Only first grouping becomes the label for charts
		}
	}

	// Always add regular aggregations (needed for ordering and calculations)
	for _, agg := range aggregations {
		funcUpper := strings.ToUpper(agg.Function)
		if !validAggFunctions[funcUpper] {
			return "", fmt.Errorf("invalid aggregation function '%s'; must be one of: SUM, COUNT, AVG, MAX, MIN, VARIANCE, STDDEV", agg.Function)
		}
		wrappedColumn := wrapNumericColumn(agg.Column, tableName)
		aggExpr := fmt.Sprintf("%s(%s)", funcUpper, wrappedColumn)
		selectParts = append(selectParts, fmt.Sprintf("%s as %s", aggExpr, quoteIdentifier(agg.Alias)))
	}

	// Add all calculated fields
	for _, calc := range calculations {
		calcField, err := buildCalculatedField(calc, aggregations, tableName)
		if err != nil {
			return "", err
		}
		selectParts = append(selectParts, calcField)
	}

	return fmt.Sprintf("SELECT %s", strings.Join(selectParts, ",\n       ")), nil
}

// buildGroupByClause generates the GROUP BY portion
func buildGroupByClause(groupBy []GroupBy) string {
	var groupParts []string

	for _, gb := range groupBy {
		expr := getGroupByExpression(gb)
		groupParts = append(groupParts, expr)

		isDateField := strings.Contains(strings.ToLower(gb.Field), "date")
		quotedField := quoteIdentifier(gb.Field)

		// For date-based grouping, add extraction for ordering
		// Use GetFormat() to infer format from field name when not explicitly set
		format := gb.GetFormat()
		if format == "month" || format == "month_year" {
			if isDateField {
				groupParts = append(groupParts, fmt.Sprintf("EXTRACT(YEAR FROM %s::date)", gb.Field))
				groupParts = append(groupParts, fmt.Sprintf("EXTRACT(MONTH FROM %s::date)", gb.Field))
			} else {
				groupParts = append(groupParts, quoteIdentifier(gb.GetYearField()))
				groupParts = append(groupParts, quotedField)
			}
		} else if format == "month_noyear" {
			if isDateField {
				groupParts = append(groupParts, fmt.Sprintf("EXTRACT(MONTH FROM %s::date)", gb.Field))
			} else {
				groupParts = append(groupParts, quotedField)
			}
		} else if format == "year" {
			if isDateField {
				groupParts = append(groupParts, fmt.Sprintf("EXTRACT(YEAR FROM %s::date)", gb.Field))
			} else {
				groupParts = append(groupParts, quotedField)
			}
		}
	}

	return fmt.Sprintf("GROUP BY %s", strings.Join(groupParts, ", "))
}

// buildOrderByClause generates the ORDER BY portion
func buildOrderByClause(groupBy []GroupBy, orderBy []OrderBy) string {
	var orderParts []string

	// Use explicit OrderBy if provided
	if len(orderBy) > 0 {
		for _, ob := range orderBy {
			direction := "ASC"
			if strings.ToUpper(ob.Direction) == "DESC" {
				direction = "DESC"
			}
			orderParts = append(orderParts, fmt.Sprintf("%s %s", ob.Field, direction))
		}
	} else {
		// Default: Order by the same extraction used in GROUP BY
		for _, gb := range groupBy {
			isDateField := strings.Contains(strings.ToLower(gb.Field), "date")
			quotedField := quoteIdentifier(gb.Field)

			format := gb.GetFormat()
			if format == "month" || format == "month_year" {
				if isDateField {
					orderParts = append(orderParts, fmt.Sprintf("EXTRACT(YEAR FROM %s::date)", gb.Field))
					orderParts = append(orderParts, fmt.Sprintf("EXTRACT(MONTH FROM %s::date)", gb.Field))
				} else {
					yearCol := quoteIdentifier(gb.GetYearField())
					orderParts = append(orderParts, fmt.Sprintf("%s::int", yearCol))
					orderParts = append(orderParts, fmt.Sprintf("%s::int", quotedField))
				}
			} else if format == "month_noyear" {
				if isDateField {
					orderParts = append(orderParts, fmt.Sprintf("EXTRACT(MONTH FROM %s::date)", gb.Field))
				} else {
					orderParts = append(orderParts, fmt.Sprintf("%s::int", quotedField))
				}
			} else if format == "year" {
				if isDateField {
					orderParts = append(orderParts, fmt.Sprintf("EXTRACT(YEAR FROM %s::date)", gb.Field))
				} else {
					orderParts = append(orderParts, fmt.Sprintf("%s::int", quotedField))
				}
			} else {
				// Default: order by the field itself
				orderParts = append(orderParts, gb.Field)
			}
		}
	}

	if len(orderParts) == 0 {
		return ""
	}

	return fmt.Sprintf("ORDER BY %s", strings.Join(orderParts, ", "))
}

// getGroupByExpression returns the SQL expression for a GROUP BY field
func getGroupByExpression(gb GroupBy) string {
	// Quote the field name to handle special characters
	quotedField := quoteIdentifier(gb.Field)

	// Detect if field is a date column (contains "date") vs a numeric column
	// Same heuristic used in buildGroupByColumns (query_builder_v2.go)
	isDateField := strings.Contains(strings.ToLower(gb.Field), "date")

	// Use GetFormat() to infer format from field name when not explicitly set
	switch gb.GetFormat() {
	case "month", "month_year":
		// Build "Mon YYYY" label
		// "month" and "month_year" are equivalent — both include year
		if isDateField {
			// Date column: extract month+year directly from the date
			return fmt.Sprintf("TO_CHAR(%s::date, 'Mon YYYY')", quotedField)
		}
		// Numeric column: build date from separate year + month columns
		yearCol := quoteIdentifier(gb.GetYearField())
		return fmt.Sprintf("TO_CHAR(MAKE_DATE(%s::int, %s::int, 1), 'Mon YYYY')", yearCol, quotedField)

	case "week", "week_year":
		// Build "YYYYW##" ISO week label
		if isDateField {
			return fmt.Sprintf("TO_CHAR(%s::date, 'IYYY') || 'W' || LPAD(EXTRACT(WEEK FROM %s::date)::text, 2, '0')", quotedField, quotedField)
		}
		yearCol := quoteIdentifier(gb.GetYearField())
		return fmt.Sprintf("%s::text || 'W' || LPAD(%s::text, 2, '0')", yearCol, quotedField)

	case "quarter", "quarter_year":
		// Build "YYYYQ#" label
		if isDateField {
			return fmt.Sprintf("TO_CHAR(%s::date, 'YYYY') || 'Q' || EXTRACT(QUARTER FROM %s::date)::text", quotedField, quotedField)
		}
		yearCol := quoteIdentifier(gb.GetYearField())
		return fmt.Sprintf("%s::text || 'Q' || %s::text", yearCol, quotedField)

	// Explicit no-year variants: merge data across years intentionally
	case "month_noyear":
		if isDateField {
			return fmt.Sprintf("TO_CHAR(%s::date, 'Mon')", quotedField)
		}
		return fmt.Sprintf("TO_CHAR(MAKE_DATE(2000, %s::int, 1), 'Mon')", quotedField)
	case "week_noyear":
		if isDateField {
			return fmt.Sprintf("'W' || LPAD(EXTRACT(WEEK FROM %s::date)::text, 2, '0')", quotedField)
		}
		return fmt.Sprintf("'W' || LPAD(%s::text, 2, '0')", quotedField)
	case "quarter_noyear":
		if isDateField {
			return fmt.Sprintf("'Q' || EXTRACT(QUARTER FROM %s::date)::text", quotedField)
		}
		return fmt.Sprintf("'Q' || %s::text", quotedField)

	case "year":
		if isDateField {
			return fmt.Sprintf("TO_CHAR(%s::date, 'YYYY')", quotedField)
		}
		return fmt.Sprintf("%s::text", quotedField)
	case "date":
		return fmt.Sprintf("TO_CHAR(%s::date, 'YYYY-MM-DD')", quotedField)
	default:
		return quoteIdentifier(gb.Field)
	}
}

// wrapNumericColumn applies universal safe wrapping to all aggregated columns
// Pattern: COALESCE(CAST(NULLIF(TRIM(CAST("column" AS TEXT)), ”) AS NUMERIC), 0)
// - CAST to TEXT: Enables TRIM for both TEXT and numeric columns
// - TRIM: Removes leading/trailing whitespace (handles spaces, tabs, newlines)
// - NULLIF: Converts empty strings to NULL
// - CAST to NUMERIC: Forces NUMERIC type conversion (preserves decimal values)
// - COALESCE: Defaults NULL to 0
// Works universally for TEXT columns, INTEGER columns, BIGINT, NUMERIC, and all other data types.
func wrapNumericColumn(column string, tableName string) string {
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

// buildCalculatedField generates a calculated field with the given formula
func buildCalculatedField(calc Calculate, aggregations []Aggregation, tableName string) (string, error) {
	formula := calc.Formula

	// Replace aggregation aliases with their full expressions
	for _, agg := range aggregations {
		funcUpper := strings.ToUpper(agg.Function)
		if !validAggFunctions[funcUpper] {
			return "", fmt.Errorf("invalid aggregation function '%s'; must be one of: SUM, COUNT, AVG, MAX, MIN, VARIANCE, STDDEV", agg.Function)
		}
		wrappedColumn := wrapNumericColumn(agg.Column, tableName)
		aggExpr := fmt.Sprintf("%s(%s)", funcUpper, wrappedColumn)

		// Replace the alias in the formula with the aggregation expression
		// Use word boundaries to avoid partial matches
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(agg.Alias))
		re := regexp.MustCompile(pattern)
		formula = re.ReplaceAllString(formula, aggExpr)
	}

	// Detect if there's a division (for zero-check CASE statement)
	hasDivision := strings.Contains(formula, "/")

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
			wrappedDenom := wrapNumericColumn(denomAgg.Column, tableName)
			denomExpr := fmt.Sprintf("%s(%s)", strings.ToUpper(denomAgg.Function), wrappedDenom)

			whenZeroValue := 0
			if calc.WhenZero != nil {
				whenZeroValue = calc.WhenZero.(int)
			}

			var roundClause string
			if calc.RoundTo != nil {
				roundClause = fmt.Sprintf("ROUND(%s, %d)", formula, *calc.RoundTo)
			} else {
				roundClause = formula
			}

			calculatedField = fmt.Sprintf(
				"CASE WHEN %s > 0 THEN %s ELSE %d END as value",
				denomExpr,
				roundClause,
				whenZeroValue,
			)
		} else {
			// Fallback if we don't have enough aggregations
			calculatedField = fmt.Sprintf("%s as value", formula)
		}
	} else {
		// No division, just apply rounding if needed
		if calc.RoundTo != nil {
			calculatedField = fmt.Sprintf("ROUND(%s, %d) as value", formula, *calc.RoundTo)
		} else {
			calculatedField = fmt.Sprintf("%s as value", formula)
		}
	}

	return calculatedField, nil
}

// quoteIdentifier wraps an identifier in double quotes for PostgreSQL.
// Escapes embedded double quotes per SQL standard (double them).
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// buildWhereClause constructs the WHERE clause with filter placeholders
// If filters is empty/nil, includes all filter placeholders (backward compatibility)
func buildWhereClause(filters []string) string {
	whereClause := "WHERE 1=1"

	// If no filters specified, include all filter placeholders (backward compatibility)
	if len(filters) == 0 {
		return whereClause + " {{district_filter}} {{year_filter}} {{month_filter}} {{week_filter}} {{quarter_filter}}"
	}

	// Include only specified filters
	filterMap := make(map[string]bool)
	for _, f := range filters {
		filterMap[strings.ToLower(f)] = true
	}

	var filterParts []string
	if filterMap["region"] {
		filterParts = append(filterParts, "{{region_filter}}")
	}
	if filterMap["district"] {
		filterParts = append(filterParts, "{{district_filter}}")
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

	if len(filterParts) > 0 {
		return whereClause + " " + strings.Join(filterParts, " ")
	}

	return whereClause
}
