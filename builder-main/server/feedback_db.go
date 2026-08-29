// Package main: feedback store (SQLite).
//
// Scoped exception to the read-only design: user feedback on reports is the
// only write surface in this service. See CLAUDE.md. Analytical queries
// against PostgreSQL remain read-only; feedback lives in its own SQLite file.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var feedbackDB *sql.DB

const feedbackSchema = `
CREATE TABLE IF NOT EXISTS feedback (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    report_id  TEXT    NOT NULL,
    user_sub   TEXT    NOT NULL,
    username   TEXT    NOT NULL,
    email      TEXT,
    section_id TEXT    NOT NULL DEFAULT '',
    comment    TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_feedback_report ON feedback(report_id, created_at DESC);
`

// FeedbackEntry is a single feedback row returned from the store.
type FeedbackEntry struct {
	ID        int64  `json:"id"`
	ReportID  string `json:"reportId"`
	SectionID string `json:"sectionId,omitempty"`
	Username  string `json:"username"`
	Email     string `json:"email,omitempty"`
	Comment   string `json:"comment"`
	CreatedAt int64  `json:"createdAt"` // unix seconds
}

// InitFeedbackDB opens (and creates if missing) the SQLite feedback database.
// Path is FEEDBACK_DB_PATH env var, defaulting to ./data/feedback.db.
func InitFeedbackDB() error {
	path := os.Getenv("FEEDBACK_DB_PATH")
	if path == "" {
		path = filepath.Join("data", "feedback.db")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("feedback: create dir: %w", err)
	}

	// _pragma query params are applied by modernc.org/sqlite on each connection.
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("feedback: open: %w", err)
	}

	// Writers must be serialised; SQLite allows only one writer at a time.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, feedbackSchema); err != nil {
		db.Close()
		return fmt.Errorf("feedback: schema: %w", err)
	}

	// Migrate existing databases: add section_id column if absent.
	// SQLite returns "duplicate column name" when the column already exists; that is safe to ignore.
	if _, migrErr := db.ExecContext(ctx, `ALTER TABLE feedback ADD COLUMN section_id TEXT NOT NULL DEFAULT ''`); migrErr != nil {
		if !strings.Contains(migrErr.Error(), "duplicate column name") {
			db.Close()
			return fmt.Errorf("feedback: migration: %w", migrErr)
		}
	}

	// Create section index after migration so it exists whether the table is new or migrated.
	if _, idxErr := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_feedback_section ON feedback(report_id, section_id, created_at DESC)`); idxErr != nil {
		db.Close()
		return fmt.Errorf("feedback: section index: %w", idxErr)
	}

	feedbackDB = db
	logInfo("Feedback DB initialized: %s", path)
	return nil
}

// CloseFeedbackDB closes the feedback database on shutdown.
func CloseFeedbackDB() {
	if feedbackDB != nil {
		if err := feedbackDB.Close(); err != nil {
			logWarn("Feedback DB close: %v", err)
		}
	}
}

// InsertFeedback stores a single feedback entry. Caller is responsible for
// validating reportID and trimming/length-checking the comment.
// sectionID is empty for report-level feedback.
func InsertFeedback(ctx context.Context, reportID, userSub, username, email, sectionID, comment string) (int64, error) {
	if feedbackDB == nil {
		return 0, fmt.Errorf("feedback DB not initialised")
	}
	now := time.Now().Unix()
	res, err := feedbackDB.ExecContext(ctx,
		`INSERT INTO feedback (report_id, user_sub, username, email, section_id, comment, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		reportID, userSub, username, nullableString(email), sectionID, comment, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListFeedback returns feedback entries for a report, newest first.
// If sectionID is non-empty, only entries for that section are returned.
// If before > 0, only entries with created_at < before are returned (keyset pagination).
func ListFeedback(ctx context.Context, reportID string, limit int, before int64, sectionID string) ([]FeedbackEntry, error) {
	if feedbackDB == nil {
		return nil, fmt.Errorf("feedback DB not initialised")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var (
		rows *sql.Rows
		err  error
	)
	if sectionID != "" {
		if before > 0 {
			rows, err = feedbackDB.QueryContext(ctx,
				`SELECT id, report_id, section_id, username, email, comment, created_at
				   FROM feedback
				  WHERE report_id = ? AND section_id = ? AND created_at < ?
				  ORDER BY created_at DESC, id DESC
				  LIMIT ?`,
				reportID, sectionID, before, limit,
			)
		} else {
			rows, err = feedbackDB.QueryContext(ctx,
				`SELECT id, report_id, section_id, username, email, comment, created_at
				   FROM feedback
				  WHERE report_id = ? AND section_id = ?
				  ORDER BY created_at DESC, id DESC
				  LIMIT ?`,
				reportID, sectionID, limit,
			)
		}
	} else {
		if before > 0 {
			rows, err = feedbackDB.QueryContext(ctx,
				`SELECT id, report_id, section_id, username, email, comment, created_at
				   FROM feedback
				  WHERE report_id = ? AND created_at < ?
				  ORDER BY created_at DESC, id DESC
				  LIMIT ?`,
				reportID, before, limit,
			)
		} else {
			rows, err = feedbackDB.QueryContext(ctx,
				`SELECT id, report_id, section_id, username, email, comment, created_at
				   FROM feedback
				  WHERE report_id = ?
				  ORDER BY created_at DESC, id DESC
				  LIMIT ?`,
				reportID, limit,
			)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]FeedbackEntry, 0, limit)
	for rows.Next() {
		var e FeedbackEntry
		var email sql.NullString
		if err := rows.Scan(&e.ID, &e.ReportID, &e.SectionID, &e.Username, &email, &e.Comment, &e.CreatedAt); err != nil {
			return nil, err
		}
		if email.Valid {
			e.Email = email.String
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// ListAllFeedback returns feedback entries across all reports, newest first.
// If before > 0, only entries with created_at < before are returned.
func ListAllFeedback(ctx context.Context, limit int, before int64) ([]FeedbackEntry, error) {
	if feedbackDB == nil {
		return nil, fmt.Errorf("feedback DB not initialised")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var (
		rows *sql.Rows
		err  error
	)
	if before > 0 {
		rows, err = feedbackDB.QueryContext(ctx,
			`SELECT id, report_id, section_id, username, email, comment, created_at
			   FROM feedback
			  WHERE created_at < ?
			  ORDER BY created_at DESC, id DESC
			  LIMIT ?`,
			before, limit,
		)
	} else {
		rows, err = feedbackDB.QueryContext(ctx,
			`SELECT id, report_id, section_id, username, email, comment, created_at
			   FROM feedback
			  ORDER BY created_at DESC, id DESC
			  LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]FeedbackEntry, 0, limit)
	for rows.Next() {
		var e FeedbackEntry
		var email sql.NullString
		if err := rows.Scan(&e.ID, &e.ReportID, &e.SectionID, &e.Username, &email, &e.Comment, &e.CreatedAt); err != nil {
			return nil, err
		}
		if email.Valid {
			e.Email = email.String
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
