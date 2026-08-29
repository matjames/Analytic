package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Retry configuration constants
const (
	// DBMaxRetries is the maximum number of retry attempts for transient errors
	DBMaxRetries = 3

	// DBInitialBackoff is the initial wait time before first retry
	DBInitialBackoff = 100 * time.Millisecond

	// DBMaxBackoff caps the exponential backoff
	DBMaxBackoff = 2 * time.Second
)

// isRetryableError determines if a database error is transient and worth retrying.
// Returns true for connection issues, network problems, and temporary failures.
// Returns false for logic errors (bad SQL, constraint violations, etc.)
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation/timeout - don't retry, user cancelled
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for driver-level bad connection
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	// Check error message for known transient PostgreSQL errors
	errMsg := strings.ToLower(err.Error())

	// Connection-level errors (retryable)
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timed out",
		"no connection",
		"broken pipe",
		"unexpected eof",
		"server closed the connection",
		"bad connection",
		"driver: bad connection",
		"network is unreachable",
		"i/o timeout",
		"tcp",
		"dial",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// PostgreSQL-specific transient errors
	pqRetryable := []string{
		"pq: sorry, too many clients already",
		"pq: the database system is starting up",
		"pq: the database system is shutting down",
		"pq: terminating connection due to administrator command",
	}

	for _, pattern := range pqRetryable {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// NOT retryable: syntax errors, constraint violations, permission errors
	// These will fail the same way every time
	nonRetryable := []string{
		"syntax error",
		"permission denied",
		"does not exist",
		"violates",
		"duplicate key",
		"invalid input",
	}

	for _, pattern := range nonRetryable {
		if strings.Contains(errMsg, pattern) {
			return false
		}
	}

	return false
}

// QueryResult holds the result of a database query operation
type QueryResult struct {
	Rows *sql.Rows
	Err  error
}

// withRetry executes a database operation with exponential backoff retry.
// Only retries on transient errors (connection issues, timeouts).
// Does not retry on logic errors (bad SQL, constraint violations).
func withRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	backoff := DBInitialBackoff

	for attempt := 0; attempt < DBMaxRetries; attempt++ {
		lastErr = operation()

		// Success
		if lastErr == nil {
			if attempt > 0 {
				logInfo("Database operation succeeded after %d retries", attempt)
			}
			return nil
		}

		// Don't retry non-transient errors
		if !isRetryableError(lastErr) {
			return lastErr
		}

		// Don't retry if context is done
		if ctx.Err() != nil {
			return lastErr
		}

		// Last attempt - don't wait, just return the error
		if attempt == DBMaxRetries-1 {
			break
		}

		// Log retry attempt
		logWarn("Database operation failed (attempt %d/%d): %v. Retrying in %v...",
			attempt+1, DBMaxRetries, lastErr, backoff)

		// Wait with backoff, but respect context cancellation
		select {
		case <-ctx.Done():
			return lastErr
		case <-time.After(backoff):
			// Double backoff for next attempt, cap at max
			backoff *= 2
			if backoff > DBMaxBackoff {
				backoff = DBMaxBackoff
			}
		}
	}

	logError("Database operation failed after %d attempts: %v", DBMaxRetries, lastErr)
	return lastErr
}

// queryWithRetry executes a query with automatic retry on transient failures.
// db selects which database connection to use (pass GetDB(report.Datasource)).
func queryWithRetry(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
	var rows *sql.Rows
	var queryErr error

	err := withRetry(ctx, func() error {
		var err error
		rows, err = db.QueryContext(ctx, query, args...)
		queryErr = err
		return err
	})

	if err != nil {
		return nil, err
	}
	return rows, queryErr
}
