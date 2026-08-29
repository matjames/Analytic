package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// GetAvailableSchemas returns all schemas that contain tables, views, or materialized views
func GetAvailableSchemas(dbs ...*sql.DB) ([]string, error) {
	db := DB
	query := `
		SELECT table_schema
		FROM (
			SELECT table_schema FROM information_schema.tables
			WHERE table_type IN ('BASE TABLE', 'VIEW')
				AND table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			UNION
			SELECT schemaname as table_schema FROM pg_matviews
			WHERE schemaname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		) schemas
		ORDER BY
			CASE WHEN table_schema = 'report' THEN 0
			     WHEN table_schema = 'public' THEN 1
			     ELSE 2 END,
			table_schema
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}
		schemas = append(schemas, schema)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	return schemas, nil
}

// GetAllTables queries the database for all available tables and materialized views in report and public schemas
func GetAllTables() ([]TableInfo, error) {
	return GetTablesBySchema("")
}

// GetTablesBySchema queries the database for tables, views, and MVs in a specific schema (or defaults to report/public if empty)
func GetTablesBySchema(schema string, dbs ...*sql.DB) ([]TableInfo, error) {
	db := DB
	var query string
	var args []interface{}

	if schema != "" && isValidIdentifier(schema) {
		query = `
			SELECT table_schema, table_name, 1 as row_count
			FROM information_schema.tables
			WHERE table_schema = $1
				AND table_type IN ('BASE TABLE', 'VIEW')
			UNION ALL
			SELECT schemaname as table_schema, matviewname as table_name, 1 as row_count
			FROM pg_matviews
			WHERE schemaname = $1
			ORDER BY table_schema, table_name
		`
		args = []interface{}{schema}
	} else {
		query = `
			SELECT table_schema, table_name, 1 as row_count
			FROM information_schema.tables
			WHERE table_schema IN ('report', 'public')
				AND table_type IN ('BASE TABLE', 'VIEW')
			UNION ALL
			SELECT schemaname as table_schema, matviewname as table_name, 1 as row_count
			FROM pg_matviews
			WHERE schemaname IN ('report', 'public')
			ORDER BY table_schema, table_name
		`
	}

	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = db.Query(query, args...)
	} else {
		rows, err = db.Query(query)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		err := rows.Scan(&table.Schema, &table.Table, &table.RowCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}
		tables = append(tables, table)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	return tables, nil
}

// GetTableColumns retrieves all columns for a specific table with metadata
func GetTableColumns(schema, table string, dbs ...*sql.DB) ([]ColumnInfo, error) {
	db := DB
	// Check if database connection is available
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Validate input to prevent SQL injection
	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		return nil, fmt.Errorf("invalid schema or table name")
	}

	// Use pg_attribute to get columns for both tables and materialized views
	query := `
		SELECT
			a.attname AS column_name,
			pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
			CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END AS is_nullable,
			pg_get_expr(d.adbin, d.adrelid) AS column_default,
			CASE
				WHEN a.atttypmod > 0 AND pg_catalog.format_type(a.atttypid, a.atttypmod) LIKE '%character%'
				THEN a.atttypmod - 4
				ELSE NULL
			END AS character_maximum_length,
			CASE
				WHEN pg_catalog.format_type(a.atttypid, a.atttypmod) LIKE 'numeric%'
				THEN ((a.atttypmod - 4) >> 16) & 65535
				ELSE NULL
			END AS numeric_precision,
			CASE
				WHEN pg_catalog.format_type(a.atttypid, a.atttypmod) LIKE 'numeric%'
				THEN (a.atttypmod - 4) & 65535
				ELSE NULL
			END AS numeric_scale
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_class c ON a.attrelid = c.oid
		JOIN pg_catalog.pg_namespace n ON c.relnamespace = n.oid
		LEFT JOIN pg_catalog.pg_attrdef d ON a.attrelid = d.adrelid AND a.attnum = d.adnum
		WHERE n.nspname = $1
			AND c.relname = $2
			AND a.attnum > 0
			AND NOT a.attisdropped
		ORDER BY a.attnum
	`

	rows, err := db.Query(query, schema, table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	numericPatterns := getNumericPatterns()

	for rows.Next() {
		var (
			name         string
			dataType     string
			isNullable   string
			defaultValue *string
			maxLength    *int
			numPrecision *int
			numScale     *int
		)

		err := rows.Scan(&name, &dataType, &isNullable, &defaultValue, &maxLength, &numPrecision, &numScale)
		if err != nil {
			return nil, fmt.Errorf("failed to scan column row: %w", err)
		}

		col := ColumnInfo{
			Name:       name,
			DataType:   dataType,
			IsNullable: isNullable == "YES",
		}

		if defaultValue != nil {
			col.DefaultValue = *defaultValue
		}

		if maxLength != nil {
			col.MaxLength = *maxLength
		}

		// Determine if column is numeric (including TEXT-stored numbers)
		col.IsNumeric = isNumericColumn(dataType, name, numericPatterns)

		columns = append(columns, col)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating column rows: %w", err)
	}

	return columns, nil
}

// GetTablePreview retrieves sample data from a table
func GetTablePreview(schema, table string, limit int, dbs ...*sql.DB) ([]string, [][]interface{}, error) {
	db := DB
	if !isValidIdentifier(schema) || !isValidIdentifier(table) {
		return nil, nil, fmt.Errorf("invalid schema or table name")
	}

	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// Build safe query - table and schema are validated identifiers
	query := fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT $1`, schema, table)

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query table preview: %w", err)
	}
	defer rows.Close()

	// Get column names
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get column names: %w", err)
	}

	// Scan rows
	var dataRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columnNames))
		valuePtrs := make([]interface{}, len(columnNames))
		for i := range columnNames {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		dataRows = append(dataRows, values)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating preview rows: %w", err)
	}

	return columnNames, dataRows, nil
}

// ExecuteQueryPreview executes a structured query and returns sample data
func ExecuteQueryPreview(req QueryPreviewRequest) *QueryPreviewResult {
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}

	// Build query from structured request
	queryObj := Query{
		Table:        req.Table,
		Where:        req.Where,
		Aggregations: req.Aggregations,
		GroupBy:      req.GroupBy,
		OrderBy:      req.OrderBy,
		Calculate:    req.Calculate,
	}

	query, err := BuildSQLWithSquirrel(queryObj, "table", req.Table, []string{})
	if err != nil {
		return &QueryPreviewResult{
			Error: fmt.Sprintf("failed to build query: %v", err),
		}
	}

	// Get random filter values from the table to narrow down the query
	sampleFilters := getSampleFiltersFromTable(req.Table)

	// Build applied filters description
	var appliedDesc []string
	if sampleFilters.District != "" {
		appliedDesc = append(appliedDesc, "District: "+sampleFilters.District)
	}
	if sampleFilters.Year != "" {
		appliedDesc = append(appliedDesc, "Year: "+sampleFilters.Year)
	}
	appliedFiltersStr := strings.Join(appliedDesc, ", ")

	// Apply sample filters to the query
	query = applySampleFilters(query, sampleFilters)

	// Add LIMIT clause
	query = fmt.Sprintf("%s LIMIT %d", query, req.Limit)

	start := time.Now()
	rows, err := DB.Query(query)
	if err != nil {
		return &QueryPreviewResult{
			Error: fmt.Sprintf("failed to execute query: %v", err),
		}
	}
	defer rows.Close()

	// Get column names
	columnNames, err := rows.Columns()
	if err != nil {
		return &QueryPreviewResult{
			Error: fmt.Sprintf("failed to get columns: %v", err),
		}
	}

	// Scan rows
	var dataRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columnNames))
		valuePtrs := make([]interface{}, len(columnNames))
		for i := range columnNames {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return &QueryPreviewResult{
				Error: fmt.Sprintf("failed to scan row: %v", err),
			}
		}

		// Convert values for proper JSON serialization
		row := make([]interface{}, len(values))
		for i, val := range values {
			row[i] = convertDBValue(val)
		}
		dataRows = append(dataRows, row)
	}

	if err = rows.Err(); err != nil {
		return &QueryPreviewResult{
			Error: fmt.Sprintf("error iterating rows: %v", err),
		}
	}

	executionTime := time.Since(start)
	return &QueryPreviewResult{
		Headers:        columnNames,
		Rows:           dataRows,
		RowCount:       len(dataRows),
		ExecutionTime:  executionTime.String(),
		AppliedFilters: appliedFiltersStr,
	}
}

// Helper functions

// SampleFilters holds random filter values picked from the table
type SampleFilters struct {
	District   string
	Year       string
	YearColumn string // tracks which column the year was detected from (e.g. "period_date", "year")
}

// getSampleFiltersFromTable picks random valid filter values from the table
func getSampleFiltersFromTable(table string) SampleFilters {
	filters := SampleFilters{}

	// Quote table name for case sensitivity
	quotedTable := quoteTableName(table)

	// Try to get a random district
	districtQuery := fmt.Sprintf(`SELECT district FROM %s WHERE district IS NOT NULL AND district != '' LIMIT 1`, quotedTable)
	row := DB.QueryRow(districtQuery)
	row.Scan(&filters.District)

	// Try to get a random year (from period_date or similar)
	// Try common column names for date/year, tracking which column succeeded
	yearSources := []struct {
		query  string
		column string
	}{
		{fmt.Sprintf(`SELECT DISTINCT EXTRACT(YEAR FROM period_date::date)::text FROM %s WHERE period_date IS NOT NULL ORDER BY 1 DESC LIMIT 1`, quotedTable), "period_date"},
		{fmt.Sprintf(`SELECT DISTINCT EXTRACT(YEAR FROM report_date::date)::text FROM %s WHERE report_date IS NOT NULL ORDER BY 1 DESC LIMIT 1`, quotedTable), "report_date"},
		{fmt.Sprintf(`SELECT DISTINCT year::text FROM %s WHERE year IS NOT NULL ORDER BY 1 DESC LIMIT 1`, quotedTable), "year"},
	}

	for _, ys := range yearSources {
		row := DB.QueryRow(ys.query)
		if err := row.Scan(&filters.Year); err == nil && filters.Year != "" {
			filters.YearColumn = ys.column
			break
		}
	}

	return filters
}

// applySampleFilters replaces filter placeholders with actual filter conditions
func applySampleFilters(query string, filters SampleFilters) string {
	// Build filter conditions
	var conditions []string

	if filters.District != "" {
		conditions = append(conditions, fmt.Sprintf(`district = '%s'`, strings.ReplaceAll(filters.District, "'", "''")))
	}

	if filters.Year != "" {
		// Apply year filter using the column the year was detected from
		if filters.YearColumn == "year" {
			// Numeric year column
			conditions = append(conditions, fmt.Sprintf(`year = %s`, filters.Year))
		} else if filters.YearColumn != "" {
			// Date column (period_date, report_date, etc.)
			conditions = append(conditions, fmt.Sprintf(`EXTRACT(YEAR FROM %s::date) = %s`, filters.YearColumn, filters.Year))
		} else {
			// Fallback: assume period_date
			conditions = append(conditions, fmt.Sprintf(`EXTRACT(YEAR FROM period_date::date) = %s`, filters.Year))
		}
	}

	// Replace placeholders
	filterPlaceholders := sampleFilterPlaceholderPattern

	if len(conditions) > 0 {
		replacement := " AND " + strings.Join(conditions, " AND ")
		// Replace the where_placeholder with our conditions
		query = strings.ReplaceAll(query, "{{where_placeholder}}", replacement)
	}

	// Remove any remaining placeholders
	query = filterPlaceholders.ReplaceAllString(query, "")

	return query
}

// convertDBValue converts database values to JSON-friendly types
func convertDBValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case []byte:
		// Convert bytes to string (handles numeric types returned as bytes)
		return string(v)
	case int64:
		return v
	case float64:
		return v
	case bool:
		return v
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case string:
		return v
	default:
		// Try to convert to string as fallback
		return fmt.Sprintf("%v", v)
	}
}

// isValidIdentifier checks if a string is a valid PostgreSQL identifier
func isValidIdentifier(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}

	// Must start with letter or underscore
	if !((s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z') || s[0] == '_') {
		return false
	}

	// Subsequent characters must be letters, digits, or underscores
	for _, c := range s[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// getNumericPatterns returns common patterns for numeric column names
func getNumericPatterns() []string {
	return []string{
		"num_", "count_", "total_", "sum_", "qty_",
		"cases", "value", "amount", "percentage", "rate",
	}
}

// isNumericColumn determines if a column contains numeric data
func isNumericColumn(dataType, name string, patterns []string) bool {
	// Native numeric types
	switch dataType {
	case "integer", "smallint", "bigint", "decimal", "numeric", "real", "double precision":
		return true
	}

	// TEXT columns with numeric patterns are likely numeric
	if dataType == "text" {
		nameLower := strings.ToLower(name)
		for _, pattern := range patterns {
			if strings.Contains(nameLower, pattern) {
				return true
			}
		}
	}

	return false
}

// ============== Advanced SQL Functions ==============

// sampleFilterPlaceholderPattern matches filter placeholders used in sample queries
var sampleFilterPlaceholderPattern = regexp.MustCompile(`\{\{(district_filter|year_filter|month_filter|week_filter|quarter_filter|where_placeholder)\}\}`)

// blockedSQLKeywords contains keywords that are not allowed in advanced SQL queries
var blockedSQLKeywords = []string{
	"INSERT", "UPDATE", "DELETE", "DROP", "TRUNCATE",
	"ALTER", "CREATE", "GRANT", "REVOKE", "EXECUTE",
	"COPY", "PG_READ", "PG_WRITE", "LO_IMPORT", "LO_EXPORT",
	"DO", "CALL", "SET", "PREPARE", "DECLARE", "LOAD",
	// Schema enumeration via system catalogs
	"PG_TABLES", "PG_VIEWS", "PG_MATVIEWS", "PG_INDEXES",
	"PG_PROC", "PG_ATTRIBUTE", "PG_CLASS", "PG_NAMESPACE",
	"PG_USER", "PG_ROLES", "PG_DATABASE", "PG_SETTINGS",
	"PG_CATALOG", "INFORMATION_SCHEMA",
	// High-risk functions
	"PG_SLEEP", "PG_TERMINATE_BACKEND", "DBLINK",
}

// Pre-compiled patterns for blocked SQL keyword validation
var blockedSQLPatterns = func() []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, len(blockedSQLKeywords))
	for i, keyword := range blockedSQLKeywords {
		patterns[i] = regexp.MustCompile(`\b` + keyword + `\b`)
	}
	return patterns
}()

// AdvancedSQLTimeout is the hardcoded timeout for advanced SQL queries
const AdvancedSQLTimeout = 30 * time.Second

// AdvancedSQLMaxRows is the maximum number of rows returned
const AdvancedSQLMaxRows = 1000

// stripStringLiterals replaces the content of single-quoted SQL string literals
// with empty strings, so keyword validation doesn't match data values.
// Handles escaped quotes (”) inside literals.
var stringLiteralPattern = regexp.MustCompile(`'(?:[^']|'')*'`)

func stripStringLiterals(sql string) string {
	return stringLiteralPattern.ReplaceAllString(sql, "''")
}

// ValidateAdvancedSQL checks if SQL is safe to execute
// Returns an error if blocked keywords are found or multiple statements detected
func ValidateAdvancedSQL(sql string) error {
	if strings.TrimSpace(sql) == "" {
		return fmt.Errorf("SQL query cannot be empty")
	}

	// Strip string literals before keyword checking so that data values
	// inside quotes (e.g., '%do not go for medical%') don't trigger false positives.
	stripped := stripStringLiterals(sql)

	// Normalize to uppercase for keyword checking
	sqlUpper := strings.ToUpper(stripped)

	// Check for blocked keywords using pre-compiled patterns
	for i, pattern := range blockedSQLPatterns {
		if pattern.MatchString(sqlUpper) {
			return fmt.Errorf("SQL contains blocked keyword: %s", blockedSQLKeywords[i])
		}
	}

	// Check for multiple statements (block semicolons that would allow chained statements)
	// Allow semicolon only at the very end.
	// Use the already-stripped version (no string literal content) and also strip single-line
	// comments so that semicolons inside '...' values or -- comments don't false-positive.
	strippedForSemicolon := regexp.MustCompile(`--[^\n]*`).ReplaceAllString(stripped, "")
	trimmed := strings.TrimSpace(strippedForSemicolon)
	trimmed = strings.TrimSuffix(trimmed, ";")
	if strings.Contains(trimmed, ";") {
		return fmt.Errorf("only single SQL statements are allowed")
	}

	return nil
}

// substituteFilters replaces {{filter}} placeholders with actual values
// Also removes unfilled complex filter placeholders ({{district_filter}}, {{year_filter}}, etc.)
func substituteFilters(sql string, filters map[string]string) string {
	result := sql
	for key, value := range filters {
		placeholder := "{{" + key + "}}"

		// When the value contains commas (multi-select) and the SQL uses = {{key}},
		// rewrite to IN (v1, v2, ...) so PostgreSQL doesn't get "2,3,1" as one value.
		if strings.Contains(value, ",") && strings.Contains(result, "= "+placeholder) {
			parts := strings.Split(value, ",")
			quoted := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					quoted = append(quoted, "'"+strings.ReplaceAll(p, "'", "''")+"'")
				}
			}
			if len(quoted) > 0 {
				result = strings.ReplaceAll(result, "= "+placeholder, "IN ("+strings.Join(quoted, ", ")+")")
			}
		} else {
			// Escape single quotes in values to prevent SQL injection
			escapedValue := strings.ReplaceAll(value, "'", "''")
			result = strings.ReplaceAll(result, placeholder, "'"+escapedValue+"'")
		}
	}

	// Remove any remaining complex filter placeholders that weren't filled
	// These are used in custom SQL and should be empty when no filter is applied
	complexPlaceholders := []string{
		"{{district_filter}}", "{{region_filter}}", "{{facility_filter}}",
		"{{year_filter}}", "{{month_filter}}", "{{week_filter}}", "{{quarter_filter}}",
	}
	for _, placeholder := range complexPlaceholders {
		result = strings.ReplaceAll(result, placeholder, "")
	}

	return result
}

// ExecuteAdvancedSQLPreview executes a raw SQL query with safety checks
func ExecuteAdvancedSQLPreview(req AdvancedSQLPreviewRequest) *AdvancedSQLPreviewResult {
	start := time.Now()

	// Validate SQL
	if err := ValidateAdvancedSQL(req.SQL); err != nil {
		return &AdvancedSQLPreviewResult{
			Error: err.Error(),
		}
	}

	// Substitute filter placeholders
	sql := substituteFilters(req.SQL, req.Filters)

	// Determine limit
	limit := req.Limit
	if limit <= 0 || limit > AdvancedSQLMaxRows {
		limit = 100 // Default
	}

	// Wrap in subquery with LIMIT for safety
	safeSql := fmt.Sprintf("SELECT * FROM (%s) AS _advanced_query LIMIT %d", sql, limit)

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), AdvancedSQLTimeout)
	defer cancel()

	rows, err := queryWithRetry(ctx, DB, safeSql)
	if err != nil {
		return &AdvancedSQLPreviewResult{
			Error: fmt.Sprintf("query error: %v", err),
		}
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return &AdvancedSQLPreviewResult{
			Error: fmt.Sprintf("failed to get columns: %v", err),
		}
	}

	// Scan rows
	var dataRows [][]interface{}
	for rows.Next() {
		select {
		case <-ctx.Done():
			return &AdvancedSQLPreviewResult{
				Error: "query timeout exceeded",
			}
		default:
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return &AdvancedSQLPreviewResult{
				Error: fmt.Sprintf("failed to scan row: %v", err),
			}
		}

		row := make([]interface{}, len(columns))
		for i, val := range values {
			row[i] = convertDBValue(val)
		}
		dataRows = append(dataRows, row)
	}

	if err = rows.Err(); err != nil {
		return &AdvancedSQLPreviewResult{
			Error: fmt.Sprintf("error iterating rows: %v", err),
		}
	}

	executionTime := time.Since(start)

	tableData := TableData{
		Headers: columns,
		Rows:    dataRows,
	}

	// Apply pivot transformation if requested
	if req.Pivot != nil {
		pivoted, err := applyPivot(tableData, req.Pivot)
		if err != nil {
			return &AdvancedSQLPreviewResult{
				Error: fmt.Sprintf("pivot error: %v", err),
			}
		}
		tableData = pivoted
	}

	return &AdvancedSQLPreviewResult{
		Headers:       tableData.Headers,
		Rows:          tableData.Rows,
		RowCount:      len(tableData.Rows),
		ExecutionTime: executionTime.String(),
	}
}
