package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"regexp"

	"gopkg.in/natefinch/lumberjack.v2"
)

// tableFromSQLRe matches FROM schema.table or FROM "schema"."table" patterns
var tableFromSQLRe = regexp.MustCompile(`(?i)\bFROM\s+("?[\w.]+"?)`)

// extractTableNameFromSQL parses the full table name (schema.table) from a SQL FROM clause
func extractTableNameFromSQL(sql string) string {
	matches := tableFromSQLRe.FindStringSubmatch(sql)
	if len(matches) >= 2 {
		return strings.Trim(matches[1], `"`)
	}
	return ""
}

// SQLLogEntry represents a single SQL log entry
type SQLLogEntry struct {
	Timestamp   string
	QueryType   string // "GENERATED", "RAW_SQL", "FILTERED", "EXECUTED"
	SQL         string
	Filters     map[string]string
	Error       string
	ExecutionMs float64
}

// SQLLogger handles all SQL query logging
type SQLLogger struct {
	writer *lumberjack.Logger
	logger *log.Logger
}

var sqlLogger *SQLLogger

// InitSQLLogger initializes the SQL logger with rotation
func InitSQLLogger() error {
	logFileName := "generated_sql.log"

	// Get absolute path for logging
	absPath, err := filepath.Abs(logFileName)
	if err != nil {
		return fmt.Errorf("could not determine path for log file: %v", err)
	}

	// Configure lumberjack for log rotation
	writer := &lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    20,   // MB - rotate when file reaches 20MB
		MaxAge:     7,    // days - delete old logs after 7 days
		MaxBackups: 3,    // keep at most 3 old log files
		Compress:   true, // compress rotated files
	}

	sqlLogger = &SQLLogger{
		writer: writer,
		logger: log.New(writer, "", 0),
	}

	// Write separator and session header
	entry := formatLogEntry(&SQLLogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		QueryType: "SESSION_START",
		SQL:       "Logger initialized",
	})
	sqlLogger.logger.Println(entry)
	sqlLogger.logger.Println(strings.Repeat("=", 80))

	fmt.Printf("SQL logging initialized: %s (rotation: 20MB/7days/3backups)\n", absPath)
	return nil
}

// LogGeneratedSQL logs SQL generated from structured queries
func LogGeneratedSQL(sql string, tableName string) {
	if sqlLogger == nil {
		return
	}

	entry := formatLogEntry(&SQLLogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		QueryType: "GENERATED",
		SQL:       sql,
	})
	sqlLogger.logger.Println(entry)
}

// LogFilteredSQL logs the SQL after filters are applied
func LogFilteredSQL(sql string, filters ReportFilters) {
	if sqlLogger == nil {
		return
	}

	filterMap := make(map[string]string)
	if len(filters.Districts) > 0 {
		filterMap["district"] = strings.Join(filters.Districts, ",")
	}
	if filters.Year != "" {
		filterMap["year"] = filters.Year
	}
	if filters.Month != "" {
		filterMap["month"] = filters.Month
	}

	entry := formatLogEntry(&SQLLogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		QueryType: "FILTERED",
		SQL:       sql,
		Filters:   filterMap,
	})
	sqlLogger.logger.Println(entry)
}

// LogExecutedSQL logs SQL that was successfully executed
func LogExecutedSQL(sql string, rowsReturned int, executionMs float64, err error) {
	if sqlLogger == nil {
		return
	}

	entry := &SQLLogEntry{
		Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
		QueryType:   "EXECUTED",
		SQL:         sql,
		ExecutionMs: executionMs,
	}

	if err != nil {
		entry.Error = err.Error()
		entry.QueryType = "EXECUTED_ERROR"
	}

	formattedEntry := formatLogEntry(entry)
	if rowsReturned > 0 {
		formattedEntry += fmt.Sprintf("\n  Rows returned: %d", rowsReturned)
	}
	if executionMs > 0 {
		formattedEntry += fmt.Sprintf("\n  Execution time: %.2f ms", executionMs)
	}

	sqlLogger.logger.Println(formattedEntry)
}

// formatLogEntry formats a SQL log entry for writing
func formatLogEntry(entry *SQLLogEntry) string {
	var result string

	result += fmt.Sprintf("[%s] [%s]\n", entry.Timestamp, entry.QueryType)

	if len(entry.Filters) > 0 {
		result += "  Filters: "
		first := true
		for k, v := range entry.Filters {
			if !first {
				result += ", "
			}
			result += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
		result += "\n"
	}

	result += fmt.Sprintf("  Query:\n%s", indentSQL(entry.SQL))

	if entry.Error != "" {
		result += fmt.Sprintf("\n  Error: %s", entry.Error)
	}

	return result
}

// indentSQL indents SQL for readability in logs
func indentSQL(sql string) string {
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

// CloseSQLLogger closes the SQL logger
func CloseSQLLogger() error {
	if sqlLogger != nil && sqlLogger.writer != nil {
		return sqlLogger.writer.Close()
	}
	return nil
}
