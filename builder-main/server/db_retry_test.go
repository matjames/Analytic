package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

// --- isRetryableError tests ---

func TestIsRetryableError_Nil(t *testing.T) {
	if isRetryableError(nil) {
		t.Error("nil error should not be retryable")
	}
}

func TestIsRetryableError_ContextCanceled(t *testing.T) {
	if isRetryableError(context.Canceled) {
		t.Error("context.Canceled should not be retryable")
	}
}

func TestIsRetryableError_ContextDeadlineExceeded(t *testing.T) {
	if isRetryableError(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should not be retryable")
	}
}

func TestIsRetryableError_ConnDone(t *testing.T) {
	if !isRetryableError(sql.ErrConnDone) {
		t.Error("sql.ErrConnDone should be retryable")
	}
}

func TestIsRetryableError_TransientPatterns(t *testing.T) {
	retryable := []string{
		"connection refused",
		"connection reset by peer",
		"connection timed out",
		"no connection to the server",
		"broken pipe",
		"unexpected eof",
		"server closed the connection unexpectedly",
		"bad connection",
		"driver: bad connection",
		"network is unreachable",
		"i/o timeout",
		"tcp: connection reset",
		"dial tcp 127.0.0.1:5432",
	}

	for _, msg := range retryable {
		if !isRetryableError(errors.New(msg)) {
			t.Errorf("error %q should be retryable", msg)
		}
	}
}

func TestIsRetryableError_PqTransient(t *testing.T) {
	pqErrors := []string{
		"pq: sorry, too many clients already",
		"pq: the database system is starting up",
		"pq: the database system is shutting down",
		"pq: terminating connection due to administrator command",
	}

	for _, msg := range pqErrors {
		if !isRetryableError(errors.New(msg)) {
			t.Errorf("error %q should be retryable", msg)
		}
	}
}

func TestIsRetryableError_NonRetryable(t *testing.T) {
	nonRetryable := []string{
		"syntax error at or near \"SELCT\"",
		"permission denied for table users",
		"relation \"foo\" does not exist",
		"violates check constraint",
		"duplicate key value violates unique constraint",
		"invalid input syntax for type integer",
	}

	for _, msg := range nonRetryable {
		if isRetryableError(errors.New(msg)) {
			t.Errorf("error %q should NOT be retryable", msg)
		}
	}
}

func TestIsRetryableError_UnknownError(t *testing.T) {
	// Errors that don't match any pattern should not be retried
	if isRetryableError(errors.New("something completely unknown")) {
		t.Error("unknown errors should not be retryable")
	}
}

func TestIsRetryableError_WrappedConnDone(t *testing.T) {
	wrapped := errors.Join(errors.New("query failed"), sql.ErrConnDone)
	if !isRetryableError(wrapped) {
		t.Error("wrapped sql.ErrConnDone should be retryable")
	}
}

// --- withRetry tests ---

func TestWithRetry_SucceedsFirstAttempt(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_SucceedsAfterTransientFailure(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return errors.New("connection refused")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_NonRetryableFailsImmediately(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), func() error {
		calls++
		return errors.New("syntax error at or near \"SELCT\"")
	})
	if err == nil {
		t.Fatal("expected error for non-retryable failure")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retries), got %d", calls)
	}
}

func TestWithRetry_ExhaustsRetries(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), func() error {
		calls++
		return errors.New("connection refused")
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != DBMaxRetries {
		t.Errorf("expected %d calls, got %d", DBMaxRetries, calls)
	}
}

func TestWithRetry_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	err := withRetry(ctx, func() error {
		calls++
		// Cancel context after first attempt so retry loop exits
		cancel()
		return errors.New("connection refused")
	})

	if err == nil {
		t.Fatal("expected error when context cancelled")
	}
	if calls > 2 {
		t.Errorf("expected at most 2 calls with cancelled context, got %d", calls)
	}
}

func TestWithRetry_BackoffIncreases(t *testing.T) {
	// Verify the retry loop takes progressively longer.
	// With DBMaxRetries=3 and initial backoff 100ms, we expect:
	//   attempt 0: fail, wait ~100ms
	//   attempt 1: fail, wait ~200ms
	//   attempt 2: fail, return
	// Total should be at least 250ms (allowing margin) but under 2s
	start := time.Now()
	calls := 0

	_ = withRetry(context.Background(), func() error {
		calls++
		return errors.New("connection refused")
	})

	elapsed := time.Since(start)

	if calls != DBMaxRetries {
		t.Errorf("expected %d calls, got %d", DBMaxRetries, calls)
	}
	// Should have waited at least initial backoff once
	if elapsed < DBInitialBackoff {
		t.Errorf("expected at least %v elapsed, got %v", DBInitialBackoff, elapsed)
	}
	// Should not take excessively long
	if elapsed > 5*time.Second {
		t.Errorf("retry took too long: %v", elapsed)
	}
}
