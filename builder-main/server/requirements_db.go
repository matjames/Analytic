// Package main: requirements specifications store (SQLite).
//
// Requirements specs are stored in a local SQLite file alongside the feedback
// store, keeping both write surfaces self-contained and independent of the
// read-only PostgreSQL warehouse connection.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var requirementsDB *sql.DB

// requirementsSchema is the DDL for the requirements_specs table.
// Executed once at startup; IF NOT EXISTS makes it idempotent.
const requirementsSchema = `
CREATE TABLE IF NOT EXISTS requirements_specs (
	id                    INTEGER PRIMARY KEY AUTOINCREMENT,
	report_name           TEXT    NOT NULL,
	report_id             TEXT,
	program_area          TEXT,
	requesting_department TEXT,
	business_owner        TEXT,
	data_analyst          TEXT,
	report_developer      TEXT,
	date_requested        TEXT,
	version               TEXT,
	status                TEXT,
	submitted_by_sub      TEXT    NOT NULL,
	submitted_by_username TEXT    NOT NULL,
	submitted_by_email    TEXT,
	payload_json          TEXT    NOT NULL,
	created_at            INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_req_specs_created
	ON requirements_specs(created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_req_specs_report_id
	ON requirements_specs(report_id, created_at DESC);
`

// RequirementsSpecEntry is a stored digital reporting requirements submission.
type RequirementsSpecEntry struct {
	ID                   int64  `json:"id"`
	ReportName           string `json:"reportName"`
	ReportID             string `json:"reportId,omitempty"`
	ProgramArea          string `json:"programArea,omitempty"`
	RequestingDepartment string `json:"requestingDepartment,omitempty"`
	BusinessOwner        string `json:"businessOwner,omitempty"`
	DataAnalyst          string `json:"dataAnalyst,omitempty"`
	ReportDeveloper      string `json:"reportDeveloper,omitempty"`
	DateRequested        string `json:"dateRequested,omitempty"`
	Version              string `json:"version,omitempty"`
	Status               string `json:"status,omitempty"`
	SubmittedBySub       string `json:"submittedBySub"`
	SubmittedByUsername  string `json:"submittedByUsername"`
	SubmittedByEmail     string `json:"submittedByEmail,omitempty"`
	PayloadJSON          string `json:"payloadJson"`
	CreatedAt            int64  `json:"createdAt"`
}

// InitRequirementsDB opens (and creates if missing) the SQLite requirements
// database. Path is REQUIREMENTS_DB_PATH env var, defaulting to ./data/requirements.db.
func InitRequirementsDB() error {
	path := os.Getenv("REQUIREMENTS_DB_PATH")
	if path == "" {
		path = filepath.Join("data", "requirements.db")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("requirements: create dir: %w", err)
	}

	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("requirements: open: %w", err)
	}

	// Serialise writes; SQLite allows only one writer at a time.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, requirementsSchema); err != nil {
		db.Close()
		return fmt.Errorf("requirements: schema: %w", err)
	}

	requirementsDB = db
	logInfo("Requirements DB initialized: %s", path)
	return nil
}

// CloseRequirementsDB closes the requirements database on shutdown.
func CloseRequirementsDB() {
	if requirementsDB != nil {
		if err := requirementsDB.Close(); err != nil {
			logError("Requirements DB close error: %v", err)
		}
	}
}

// InsertRequirementsSpec stores a single requirements specification submission.
func InsertRequirementsSpec(ctx context.Context, entry RequirementsSpecEntry) (int64, error) {
	if requirementsDB == nil {
		return 0, fmt.Errorf("requirements: DB not initialised")
	}
	now := time.Now().Unix()
	res, err := requirementsDB.ExecContext(ctx,
		`INSERT INTO requirements_specs (
			report_name, report_id, program_area, requesting_department, business_owner,
			data_analyst, report_developer, date_requested, version, status,
			submitted_by_sub, submitted_by_username, submitted_by_email, payload_json, created_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ReportName,
		nullStr(entry.ReportID),
		nullStr(entry.ProgramArea),
		nullStr(entry.RequestingDepartment),
		nullStr(entry.BusinessOwner),
		nullStr(entry.DataAnalyst),
		nullStr(entry.ReportDeveloper),
		nullStr(entry.DateRequested),
		nullStr(entry.Version),
		nullStr(entry.Status),
		entry.SubmittedBySub,
		entry.SubmittedByUsername,
		nullStr(entry.SubmittedByEmail),
		entry.PayloadJSON,
		now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListRequirementsSpecs returns requirements specs newest-first.
// If before > 0, only entries with created_at < before are returned (keyset pagination).
func ListRequirementsSpecs(ctx context.Context, limit int, before int64) ([]RequirementsSpecEntry, error) {
	if requirementsDB == nil {
		return nil, fmt.Errorf("requirements: DB not initialised")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var (
		rows *sql.Rows
		err  error
	)
	if before > 0 {
		rows, err = requirementsDB.QueryContext(ctx,
			`SELECT id, report_name, report_id, program_area, requesting_department, business_owner,
			        data_analyst, report_developer, date_requested, version, status,
			        submitted_by_sub, submitted_by_username, submitted_by_email, payload_json, created_at
			   FROM requirements_specs
			  WHERE created_at < ?
			  ORDER BY created_at DESC, id DESC
			  LIMIT ?`,
			before, limit,
		)
	} else {
		rows, err = requirementsDB.QueryContext(ctx,
			`SELECT id, report_name, report_id, program_area, requesting_department, business_owner,
			        data_analyst, report_developer, date_requested, version, status,
			        submitted_by_sub, submitted_by_username, submitted_by_email, payload_json, created_at
			   FROM requirements_specs
			  ORDER BY created_at DESC, id DESC
			  LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RequirementsSpecEntry, 0, limit)
	for rows.Next() {
		var (
			e                          RequirementsSpecEntry
			reportID, programArea      sql.NullString
			requestingDepartment       sql.NullString
			businessOwner, dataAnalyst sql.NullString
			reportDeveloper            sql.NullString
			dateRequested, version     sql.NullString
			status, email              sql.NullString
		)
		if err := rows.Scan(
			&e.ID,
			&e.ReportName,
			&reportID,
			&programArea,
			&requestingDepartment,
			&businessOwner,
			&dataAnalyst,
			&reportDeveloper,
			&dateRequested,
			&version,
			&status,
			&e.SubmittedBySub,
			&e.SubmittedByUsername,
			&email,
			&e.PayloadJSON,
			&e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if reportID.Valid {
			e.ReportID = reportID.String
		}
		if programArea.Valid {
			e.ProgramArea = programArea.String
		}
		if requestingDepartment.Valid {
			e.RequestingDepartment = requestingDepartment.String
		}
		if businessOwner.Valid {
			e.BusinessOwner = businessOwner.String
		}
		if dataAnalyst.Valid {
			e.DataAnalyst = dataAnalyst.String
		}
		if reportDeveloper.Valid {
			e.ReportDeveloper = reportDeveloper.String
		}
		if dateRequested.Valid {
			e.DateRequested = dateRequested.String
		}
		if version.Valid {
			e.Version = version.String
		}
		if status.Valid {
			e.Status = status.String
		}
		if email.Valid {
			e.SubmittedByEmail = email.String
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// nullStr returns nil for empty strings so SQLite stores NULL rather than "".
func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetRequirementsSpecByID fetches a single requirements spec by primary key.
// Returns (nil, nil) when the row does not exist.
func GetRequirementsSpecByID(ctx context.Context, id int64) (*RequirementsSpecEntry, error) {
	if requirementsDB == nil {
		return nil, fmt.Errorf("requirements: DB not initialised")
	}
	var (
		e                          RequirementsSpecEntry
		reportID, programArea      sql.NullString
		requestingDepartment       sql.NullString
		businessOwner, dataAnalyst sql.NullString
		reportDeveloper            sql.NullString
		dateRequested, version     sql.NullString
		status, email              sql.NullString
	)
	err := requirementsDB.QueryRowContext(ctx,
		`SELECT id, report_name, report_id, program_area, requesting_department, business_owner,
		        data_analyst, report_developer, date_requested, version, status,
		        submitted_by_sub, submitted_by_username, submitted_by_email, payload_json, created_at
		   FROM requirements_specs
		  WHERE id = ?`,
		id,
	).Scan(
		&e.ID,
		&e.ReportName,
		&reportID,
		&programArea,
		&requestingDepartment,
		&businessOwner,
		&dataAnalyst,
		&reportDeveloper,
		&dateRequested,
		&version,
		&status,
		&e.SubmittedBySub,
		&e.SubmittedByUsername,
		&email,
		&e.PayloadJSON,
		&e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if reportID.Valid {
		e.ReportID = reportID.String
	}
	if programArea.Valid {
		e.ProgramArea = programArea.String
	}
	if requestingDepartment.Valid {
		e.RequestingDepartment = requestingDepartment.String
	}
	if businessOwner.Valid {
		e.BusinessOwner = businessOwner.String
	}
	if dataAnalyst.Valid {
		e.DataAnalyst = dataAnalyst.String
	}
	if reportDeveloper.Valid {
		e.ReportDeveloper = reportDeveloper.String
	}
	if dateRequested.Valid {
		e.DateRequested = dateRequested.String
	}
	if version.Valid {
		e.Version = version.String
	}
	if status.Valid {
		e.Status = status.String
	}
	if email.Valid {
		e.SubmittedByEmail = email.String
	}
	return &e, nil
}

// UpdateRequirementsSpec replaces the mutable fields of an existing requirements spec.
// Returns (false, nil) if the row does not exist, (true, nil) on success.
func UpdateRequirementsSpec(ctx context.Context, id int64, entry RequirementsSpecEntry) (bool, error) {
	if requirementsDB == nil {
		return false, fmt.Errorf("requirements: DB not initialised")
	}
	res, err := requirementsDB.ExecContext(ctx,
		`UPDATE requirements_specs
		    SET report_name           = ?,
		        report_id             = ?,
		        program_area          = ?,
		        requesting_department = ?,
		        business_owner        = ?,
		        data_analyst          = ?,
		        report_developer      = ?,
		        date_requested        = ?,
		        version               = ?,
		        status                = ?,
		        payload_json          = ?
		  WHERE id = ?`,
		entry.ReportName,
		nullStr(entry.ReportID),
		nullStr(entry.ProgramArea),
		nullStr(entry.RequestingDepartment),
		nullStr(entry.BusinessOwner),
		nullStr(entry.DataAnalyst),
		nullStr(entry.ReportDeveloper),
		nullStr(entry.DateRequested),
		nullStr(entry.Version),
		nullStr(entry.Status),
		entry.PayloadJSON,
		id,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
