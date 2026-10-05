package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type processingScheduleRequest struct {
	DatasetID string     `json:"dataset_id"`
	AssetID   string     `json:"asset_id"`
	Frequency string     `json:"frequency"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
}

type processingSchedule struct {
	ID          string     `json:"id"`
	DatasetID   string     `json:"dataset_id"`
	AssetID     string     `json:"asset_id"`
	Frequency   string     `json:"frequency"`
	Status      string     `json:"status"`
	TenantID    string     `json:"tenant_id"`
	WorkspaceID string     `json:"workspace_id"`
	NextRunAt   *time.Time `json:"next_run_at,omitempty"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	LastStatus  string     `json:"last_status,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type processingScheduleRun struct {
	ID           int64      `json:"id"`
	ScheduleID   string     `json:"schedule_id"`
	DatasetID    string     `json:"dataset_id"`
	ScheduledFor time.Time  `json:"scheduled_for"`
	Status       string     `json:"status"`
	HTTPStatus   int        `json:"http_status,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type processingScheduleClaim struct {
	ID           string
	DatasetID    string
	TenantID     string
	WorkspaceID  string
	Frequency    string
	ScheduledFor time.Time
}

func initProcessingScheduleSchema(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database unavailable")
	}
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS processing_schedules (
  id VARCHAR(128) PRIMARY KEY,
  dataset_id VARCHAR(128) NOT NULL,
  asset_id VARCHAR(128) NOT NULL,
  frequency VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  tenant_id VARCHAR(128) NOT NULL,
  workspace_id VARCHAR(128) NOT NULL,
  next_run_at TIMESTAMPTZ,
  last_run_at TIMESTAMPTZ,
  last_status VARCHAR(32),
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (dataset_id, asset_id, tenant_id, workspace_id)
);
ALTER TABLE processing_schedules ADD COLUMN IF NOT EXISTS last_status VARCHAR(32);
ALTER TABLE processing_schedules ADD COLUMN IF NOT EXISTS last_error TEXT;
CREATE INDEX IF NOT EXISTS idx_processing_schedules_scope
  ON processing_schedules(tenant_id, workspace_id, status, next_run_at);
CREATE TABLE IF NOT EXISTS processing_schedule_runs (
  id BIGSERIAL PRIMARY KEY,
  schedule_id VARCHAR(128) NOT NULL REFERENCES processing_schedules(id) ON DELETE CASCADE,
  dataset_id VARCHAR(128) NOT NULL,
  scheduled_for TIMESTAMPTZ NOT NULL,
  status VARCHAR(32) NOT NULL,
  http_status INTEGER,
  error_message TEXT,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at TIMESTAMPTZ,
  UNIQUE (schedule_id, scheduled_for)
);
CREATE INDEX IF NOT EXISTS idx_processing_schedule_runs_schedule
  ON processing_schedule_runs(schedule_id, started_at DESC);`)
	return err
}

func validProcessingScheduleFrequency(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hourly", "daily", "weekly", "monthly", "on_demand":
		return true
	default:
		return false
	}
}

func processingScheduleInterval(frequency string) (time.Duration, bool) {
	switch frequency {
	case "hourly":
		return time.Hour, true
	case "daily":
		return 24 * time.Hour, true
	case "weekly":
		return 7 * 24 * time.Hour, true
	case "monthly":
		return 0, true
	default:
		return 0, false
	}
}

func nextProcessingScheduleTime(frequency string, scheduledFor, now time.Time) *time.Time {
	if frequency == "on_demand" {
		return nil
	}
	next := scheduledFor
	if next.IsZero() {
		next = now
	}
	for !next.After(now) {
		if frequency == "monthly" {
			next = next.AddDate(0, 1, 0)
			continue
		}
		interval, ok := processingScheduleInterval(frequency)
		if !ok {
			return nil
		}
		next = next.Add(interval)
	}
	return &next
}

func (s *Server) handleProcessingSchedules(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	uCtx := getUserContext(r)
	workspaceID := requestWorkspaceID(r)
	if uCtx.TenantID == "" || workspaceID == "" {
		http.Error(w, `{"error":"tenant and workspace context are required"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		var request processingScheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		request.DatasetID = strings.TrimSpace(request.DatasetID)
		request.AssetID = strings.TrimSpace(request.AssetID)
		request.Frequency = strings.ToLower(strings.TrimSpace(request.Frequency))
		if !isSafeWorkspaceIdentifier(request.DatasetID) || !isSafeWorkspaceIdentifier(request.AssetID) {
			http.Error(w, `{"error":"dataset_id and asset_id must be safe identifiers"}`, http.StatusBadRequest)
			return
		}
		if !validProcessingScheduleFrequency(request.Frequency) {
			http.Error(w, `{"error":"frequency must be hourly, daily, weekly, monthly, or on_demand"}`, http.StatusBadRequest)
			return
		}
		now := time.Now().UTC()
		nextRunAt := request.NextRunAt
		if request.Frequency == "on_demand" {
			nextRunAt = nil
		} else if nextRunAt == nil {
			interval := map[string]time.Duration{"hourly": time.Hour, "daily": 24 * time.Hour, "weekly": 7 * 24 * time.Hour}[request.Frequency]
			if request.Frequency == "monthly" {
				monthly := now.AddDate(0, 1, 0)
				nextRunAt = &monthly
			} else {
				firstRun := now.Add(interval)
				nextRunAt = &firstRun
			}
		}
		schedule := processingSchedule{
			ID: fmt.Sprintf("schedule-%d", now.UnixNano()), DatasetID: request.DatasetID, AssetID: request.AssetID,
			Frequency: request.Frequency, Status: "active", TenantID: uCtx.TenantID, WorkspaceID: workspaceID,
			NextRunAt: nextRunAt, CreatedAt: now, UpdatedAt: now,
		}
		err := s.db.QueryRow(`
INSERT INTO processing_schedules
  (id, dataset_id, asset_id, frequency, status, tenant_id, workspace_id, next_run_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
ON CONFLICT (dataset_id, asset_id, tenant_id, workspace_id) DO UPDATE SET
  frequency=EXCLUDED.frequency, status='active', next_run_at=EXCLUDED.next_run_at,
  last_status=NULL, last_error=NULL, updated_at=EXCLUDED.updated_at
RETURNING id, dataset_id, asset_id, frequency, status, tenant_id, workspace_id, next_run_at, last_run_at,
  COALESCE(last_status,''), COALESCE(last_error,''), created_at, updated_at`,
			schedule.ID, schedule.DatasetID, schedule.AssetID, schedule.Frequency, schedule.Status,
			schedule.TenantID, schedule.WorkspaceID, schedule.NextRunAt, schedule.CreatedAt).Scan(
			&schedule.ID, &schedule.DatasetID, &schedule.AssetID, &schedule.Frequency, &schedule.Status,
			&schedule.TenantID, &schedule.WorkspaceID, &schedule.NextRunAt, &schedule.LastRunAt,
			&schedule.LastStatus, &schedule.LastError, &schedule.CreatedAt, &schedule.UpdatedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(schedule)
	case http.MethodGet:
		rows, err := s.db.Query(`
SELECT id, dataset_id, asset_id, frequency, status, tenant_id, workspace_id, next_run_at, last_run_at,
  COALESCE(last_status,''), COALESCE(last_error,''), created_at, updated_at
FROM processing_schedules WHERE tenant_id=$1 AND workspace_id=$2 ORDER BY updated_at DESC`, uCtx.TenantID, workspaceID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var schedules []processingSchedule
		for rows.Next() {
			var schedule processingSchedule
			if err := rows.Scan(&schedule.ID, &schedule.DatasetID, &schedule.AssetID, &schedule.Frequency,
				&schedule.Status, &schedule.TenantID, &schedule.WorkspaceID, &schedule.NextRunAt,
				&schedule.LastRunAt, &schedule.LastStatus, &schedule.LastError, &schedule.CreatedAt, &schedule.UpdatedAt); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
				return
			}
			schedules = append(schedules, schedule)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"count": len(schedules), "schedules": schedules})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProcessingScheduleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	uCtx := getUserContext(r)
	workspaceID := requestWorkspaceID(r)
	if uCtx.TenantID == "" || workspaceID == "" {
		http.Error(w, `{"error":"tenant and workspace context are required"}`, http.StatusBadRequest)
		return
	}
	limit := 100
	if n, err := fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit); err != nil || n != 1 || limit < 1 || limit > 500 {
		limit = 100
	}
	query := `SELECT r.id, r.schedule_id, r.dataset_id, r.scheduled_for, r.status, COALESCE(r.http_status,0),
  COALESCE(r.error_message,''), r.started_at, r.completed_at
FROM processing_schedule_runs r JOIN processing_schedules s ON s.id=r.schedule_id
WHERE s.tenant_id=$1 AND s.workspace_id=$2`
	args := []interface{}{uCtx.TenantID, workspaceID}
	if scheduleID := strings.TrimSpace(r.URL.Query().Get("schedule_id")); scheduleID != "" {
		query += " AND r.schedule_id=$3"
		args = append(args, scheduleID)
	}
	query += fmt.Sprintf(" ORDER BY r.started_at DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	runs := make([]processingScheduleRun, 0)
	for rows.Next() {
		var run processingScheduleRun
		if err := rows.Scan(&run.ID, &run.ScheduleID, &run.DatasetID, &run.ScheduledFor, &run.Status,
			&run.HTTPStatus, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
			return
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"count": len(runs), "runs": runs})
}

func (s *Server) handleRunProcessingSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	var request struct {
		ScheduleID string `json:"schedule_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || strings.TrimSpace(request.ScheduleID) == "" {
		http.Error(w, `{"error":"schedule_id is required"}`, http.StatusBadRequest)
		return
	}
	uCtx := getUserContext(r)
	workspaceID := requestWorkspaceID(r)
	if uCtx.TenantID == "" || workspaceID == "" {
		http.Error(w, `{"error":"tenant and workspace context are required"}`, http.StatusBadRequest)
		return
	}
	var claim processingScheduleClaim
	err := s.db.QueryRow(`SELECT id, dataset_id, tenant_id, workspace_id, frequency
FROM processing_schedules WHERE id=$1 AND tenant_id=$2 AND workspace_id=$3 AND status='active'`,
		strings.TrimSpace(request.ScheduleID), uCtx.TenantID, workspaceID).Scan(
		&claim.ID, &claim.DatasetID, &claim.TenantID, &claim.WorkspaceID, &claim.Frequency)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"schedule not found in this workspace"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	claim.ScheduledFor = time.Now().UTC()
	if _, err := s.db.Exec(`INSERT INTO processing_schedule_runs
(schedule_id, dataset_id, scheduled_for, status) VALUES ($1,$2,$3,'running')`, claim.ID, claim.DatasetID, claim.ScheduledFor); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}
	go s.executeProcessingSchedule(context.Background(), claim)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "started", "schedule_id": claim.ID})
}

func (s *Server) runProcessingScheduleWorker(ctx context.Context) {
	if s.db == nil || strings.TrimSpace(os.Getenv("STATCOLLECT_INTERNAL_URL")) == "" || s.internalAPIKey == "" {
		log.Print("[StatGate Core] processing schedule worker disabled: StatCollect URL or internal key missing")
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		s.recoverStaleProcessingScheduleRuns(ctx)
		s.dispatchDueProcessingSchedules(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) recoverStaleProcessingScheduleRuns(ctx context.Context) {
	_, err := s.db.ExecContext(ctx, `
WITH stale AS (
  UPDATE processing_schedule_runs SET status='failed', error_message='worker_interrupted_timeout', completed_at=now()
  WHERE status='running' AND started_at < now() - interval '2 minutes'
  RETURNING schedule_id
)
UPDATE processing_schedules s SET last_status='failed', last_error='worker_interrupted_timeout', updated_at=now()
FROM stale WHERE s.id=stale.schedule_id`)
	if err != nil {
		log.Printf("[StatGate Core] stale schedule recovery failed: %v", err)
	}
}

func (s *Server) dispatchDueProcessingSchedules(ctx context.Context) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[StatGate Core] schedule claim transaction failed: %v", err)
		return
	}
	rows, err := tx.QueryContext(ctx, `
SELECT id, dataset_id, tenant_id, workspace_id, frequency, next_run_at
FROM processing_schedules
WHERE status='active' AND frequency <> 'on_demand' AND next_run_at <= now()
ORDER BY next_run_at LIMIT 20 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		_ = tx.Rollback()
		log.Printf("[StatGate Core] due schedule query failed: %v", err)
		return
	}
	var claims []processingScheduleClaim
	for rows.Next() {
		var claim processingScheduleClaim
		if err := rows.Scan(&claim.ID, &claim.DatasetID, &claim.TenantID, &claim.WorkspaceID, &claim.Frequency, &claim.ScheduledFor); err != nil {
			_ = rows.Close()
			_ = tx.Rollback()
			log.Printf("[StatGate Core] due schedule scan failed: %v", err)
			return
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		_ = tx.Rollback()
		log.Printf("[StatGate Core] due schedule rows failed: %v", err)
		return
	}
	_ = rows.Close()
	for _, claim := range claims {
		now := time.Now().UTC()
		nextRunAt := nextProcessingScheduleTime(claim.Frequency, claim.ScheduledFor, now)
		if _, err := tx.ExecContext(ctx, `INSERT INTO processing_schedule_runs
(schedule_id, dataset_id, scheduled_for, status) VALUES ($1,$2,$3,'running')
ON CONFLICT (schedule_id, scheduled_for) DO NOTHING`, claim.ID, claim.DatasetID, claim.ScheduledFor); err != nil {
			_ = tx.Rollback()
			log.Printf("[StatGate Core] schedule run creation failed (%s): %v", claim.ID, err)
			return
		}
		if _, err := tx.ExecContext(ctx, `UPDATE processing_schedules SET next_run_at=$2, updated_at=now() WHERE id=$1`, claim.ID, nextRunAt); err != nil {
			_ = tx.Rollback()
			log.Printf("[StatGate Core] schedule advancement failed (%s): %v", claim.ID, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("[StatGate Core] schedule claim commit failed: %v", err)
		return
	}
	for _, claim := range claims {
		s.executeProcessingSchedule(ctx, claim)
	}
}

func (s *Server) executeProcessingSchedule(parent context.Context, claim processingScheduleClaim) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]string{"dataset_id": claim.DatasetID})
	url := strings.TrimRight(strings.TrimSpace(os.Getenv("STATCOLLECT_INTERNAL_URL")), "/") + "/internal/statistics/datasets/refresh"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	status, httpStatus, errorMessage := "failed", 0, ""
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-StatGate-Internal-Key", s.internalAPIKey)
		req.Header.Set("X-Tenant-ID", claim.TenantID)
		req.Header.Set("X-Workspace-ID", claim.WorkspaceID)
		req.Header.Set("X-User-ID", "statgate-core-scheduler")
		req.Header.Set("X-User-Role", "operator")
		client := &http.Client{Timeout: 45 * time.Second}
		var response *http.Response
		response, err = client.Do(req)
		if err == nil {
			httpStatus = response.StatusCode
			var result struct {
				Status string `json:"status"`
			}
			decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result)
			response.Body.Close()
			if httpStatus >= 200 && httpStatus < 300 {
				if decodeErr == nil && result.Status == "no_change" {
					status = "no_change"
				} else if decodeErr == nil && result.Status == "refreshed" {
					status = "succeeded"
				} else {
					status = "failed"
					errorMessage = "StatCollect returned an invalid refresh response"
				}
			} else {
				errorMessage = fmt.Sprintf("StatCollect refresh returned HTTP %d", httpStatus)
			}
		}
	}
	if err != nil {
		errorMessage = err.Error()
	}
	if len(errorMessage) > 1000 {
		errorMessage = errorMessage[:1000]
	}
	completedAt := time.Now().UTC()
	retrySoon := err != nil || httpStatus >= http.StatusInternalServerError
	_, persistErr := s.db.ExecContext(context.Background(), `
UPDATE processing_schedule_runs SET status=$3, http_status=NULLIF($4,0), error_message=NULLIF($5,''), completed_at=$6
WHERE schedule_id=$1 AND scheduled_for=$2`, claim.ID, claim.ScheduledFor, status, httpStatus, errorMessage, completedAt)
	if persistErr == nil {
		_, persistErr = s.db.ExecContext(context.Background(), `
		UPDATE processing_schedules SET last_run_at=$2, last_status=$3, last_error=NULLIF($4,''),
  next_run_at=CASE WHEN $5 THEN LEAST(COALESCE(next_run_at,$2),$2 + interval '1 minute') ELSE next_run_at END,
  updated_at=$2 WHERE id=$1`, claim.ID, completedAt, status, errorMessage, retrySoon)
	}
	if persistErr != nil {
		log.Printf("[StatGate Core] schedule result persistence failed (%s): %v", claim.ID, persistErr)
	}
}
