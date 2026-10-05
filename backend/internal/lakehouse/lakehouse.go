package lakehouse

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"
)

type EventRecord struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	WorkspaceID string                 `json:"workspace_id,omitempty"`
	Source      string                 `json:"source"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   time.Time              `json:"timestamp"`
	Value       float64                `json:"value"`
}

type AnomalyAlert struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	TenantID    string    `json:"tenant_id"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	MetricName  string    `json:"metric_name"`
	Dataset     string    `json:"dataset"`
	Value       float64   `json:"value"`
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"std_dev"`
	ZScore      float64   `json:"z_score"`
	SigmaScore  float64   `json:"sigma_score"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
	NotebookURL string    `json:"notebook_url"`
}

type StorageEngine struct {
	// PG is the durable store. Nil means the engine cannot persist;
	// construction fails closed, so a nil PGStore must never reach this type.
	db *sql.DB
}

// NewStorageEngine creates a storage engine backed by durable PostgreSQL.
// It provisions the event ledger schema idempotently and returns an error
// instead of a fake in-memory demo store when Postgres is unavailable.
func NewStorageEngine(db *sql.DB) (*StorageEngine, error) {
	if db == nil {
		return nil, fmt.Errorf("lakehouse: postgres handle is required")
	}
	se := &StorageEngine{db: db}
	if err := se.ensureSchema(); err != nil {
		return nil, fmt.Errorf("lakehouse: schema init failed: %w", err)
	}
	return se, nil
}

// ensureSchema self-provisions the event ledger. Idempotent; safe on every
// startup and against a fresh database.

func (se *StorageEngine) ensureSchema() error {
	stmts := []string{
		`CREATE SCHEMA IF NOT EXISTS core`,
		`CREATE TABLE IF NOT EXISTS core.event_ledger (
			id VARCHAR(128) PRIMARY KEY,
			tenant_id VARCHAR(64) NOT NULL,
			workspace_id VARCHAR(128),
			source VARCHAR(128) NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}',
			ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			value DOUBLE PRECISION NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_event_ledger_scope ON core.event_ledger (tenant_id, workspace_id, ts DESC)`,
		`CREATE TABLE IF NOT EXISTS core.anomaly_alerts (
			id VARCHAR(128) PRIMARY KEY,
			event_id VARCHAR(128) NOT NULL,
			tenant_id VARCHAR(64) NOT NULL,
			workspace_id VARCHAR(128),
			metric_name VARCHAR(128) NOT NULL,
			dataset VARCHAR(128) NOT NULL,
			value DOUBLE PRECISION NOT NULL DEFAULT 0,
			mean DOUBLE PRECISION NOT NULL DEFAULT 0,
			std_dev DOUBLE PRECISION NOT NULL DEFAULT 0,
			z_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			sigma_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			severity VARCHAR(16) NOT NULL DEFAULT 'HIGH',
			ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			message TEXT,
			notebook_url TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_anomaly_scope ON core.anomaly_alerts (tenant_id, workspace_id, ts DESC)`,
	}
	for i, s := range stmts {
		if _, err := se.db.Exec(s); err != nil {
			return fmt.Errorf("statement %d: %w", i, err)
		}
	}
	return nil
}

// scopeKey is retained as documentation of the old in-memory scoping model;
// durable queries now filter by (tenant_id, workspace_id) directly in SQL.

// Ingest durably persists one telemetry event, then evaluates it for
// 3-sigma anomaly against the persisted window for the same
// (tenant, workspace, source). It returns an alert when one fires.
func (se *StorageEngine) Ingest(rec EventRecord) AnomalyAlert {
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("ingest-%d", rec.Timestamp.UnixNano())
	}

	payload, err := json.Marshal(rec.Payload)
	if err != nil {
		log.Printf("[lakehouse] ingest: payload marshal failed for %s: %v", rec.ID, err)
		payload = []byte("{}")
	}
	ts := rec.Timestamp.UTC()
	if _, err := se.db.Exec(
		`INSERT INTO core.event_ledger (id, tenant_id, workspace_id, source, payload, ts, value)
		 VALUES ($1, $2, NULLIF($3,''), $4, $5, $6, $7)
		 ON CONFLICT (id) DO NOTHING`,
		rec.ID, rec.TenantID, rec.WorkspaceID, rec.Source, payload, ts, rec.Value,
	); err != nil {
		log.Printf("[lakehouse] ingest: insert failed for %s: %v", rec.ID, err)
		return AnomalyAlert{}
	}

	// Durable statistics window: last 200 persisted values for the scope.
	history, err := se.recentValues(rec.TenantID, rec.WorkspaceID, rec.Source, 200)
	if err != nil {
		log.Printf("[lakehouse] ingest: history read failed: %v", err)
		return AnomalyAlert{}
	}
	if len(history) < 10 {
		return AnomalyAlert{}
	}

	mean, stdDev := meanStdDev(history)
	if stdDev <= 0 {
		return AnomalyAlert{}
	}
	zScore := (rec.Value - mean) / stdDev
	if math.Abs(zScore) < 3.0 {
		return AnomalyAlert{}
	}

	severity := "HIGH"
	if math.Abs(zScore) >= 3.5 {
		severity = "CRITICAL"
	}
	alert := AnomalyAlert{
		ID:          fmt.Sprintf("alert_%d", ts.UnixNano()),
		EventID:     rec.ID,
		TenantID:    rec.TenantID,
		WorkspaceID: rec.WorkspaceID,
		MetricName:  rec.Source,
		Dataset:     "live_telemetry_stream",
		Value:       rec.Value,
		Mean:        mean,
		StdDev:      stdDev,
		ZScore:      zScore,
		SigmaScore:  math.Round(zScore*100) / 100,
		Severity:    severity,
		Timestamp:   ts,
		Message:     fmt.Sprintf("3σ Anomaly: %s = %.2f (σ=%.2f, μ=%.2f)", rec.Source, rec.Value, zScore, mean),
		NotebookURL: "/notebook",
	}
	if _, err := se.db.Exec(
		`INSERT INTO core.anomaly_alerts
		 (id, event_id, tenant_id, workspace_id, metric_name, dataset, value, mean, std_dev, z_score, sigma_score, severity, ts, message, notebook_url)
		 VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 ON CONFLICT (id) DO NOTHING`,
		alert.ID, alert.EventID, alert.TenantID, alert.WorkspaceID,
		alert.MetricName, alert.Dataset, alert.Value, alert.Mean, alert.StdDev,
		alert.ZScore, alert.SigmaScore, alert.Severity, alert.Timestamp,
		alert.Message, alert.NotebookURL,
	); err != nil {
		log.Printf("[lakehouse] ingest: alert persist failed: %v", err)
	}
	return alert
}

// recentValues returns the last n persisted values for a scope, newest last.

// recentValues returns the last n persisted values for a scope, newest last.
func (se *StorageEngine) recentValues(tenantID, workspaceID, source string, n int) ([]float64, error) {
	rows, err := se.db.Query(
		`SELECT value FROM core.event_ledger
		 WHERE tenant_id = $1 AND COALESCE(workspace_id,'') = $2 AND source = $3
		 ORDER BY ts DESC, id DESC LIMIT $4`,
		tenantID, workspaceID, source, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vals := make([]float64, 0, n)
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Reverse so the window reads oldest-first like the old in-memory model.
	for i, j := 0, len(vals)-1; i < j; i, j = i+1, j-1 {
		vals[i], vals[j] = vals[j], vals[i]
	}
	return vals, nil
}

func meanStdDev(xs []float64) (mean, stdDev float64) {
	var sum float64
	for _, v := range xs {
		sum += v
	}
	mean = sum / float64(len(xs))
	var varianceSum float64
	for _, v := range xs {
		varianceSum += math.Pow(v-mean, 2)
	}
	return mean, math.Sqrt(varianceSum / float64(len(xs)))
}

// RecordEvent durably appends a domain event to the lakehouse ledger without
// triggering telemetry anomaly detection. It is used for observability events
// such as institutional condition changes.
func (se *StorageEngine) RecordEvent(rec EventRecord) {
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("evt-%d", rec.Timestamp.UnixNano())
	}

	payload, err := json.Marshal(rec.Payload)
	if err != nil {
		log.Printf("[lakehouse] RecordEvent: payload marshal failed for %s: %v", rec.ID, err)
		payload = []byte("{}")
	}
	if _, err := se.db.Exec(
		`INSERT INTO core.event_ledger (id, tenant_id, workspace_id, source, payload, ts, value)
		 VALUES ($1, $2, NULLIF($3,''), $4, $5, $6, $7)
		 ON CONFLICT (id) DO NOTHING`,
		rec.ID, rec.TenantID, rec.WorkspaceID, rec.Source, payload, rec.Timestamp.UTC(), rec.Value,
	); err != nil {
		log.Printf("[lakehouse] RecordEvent: insert failed for %s: %v", rec.ID, err)
	}
}

func (se *StorageEngine) QueryTenantData(tenantID, workspaceID string, limit int) []EventRecord {
	if limit <= 0 {
		limit = 50
	}
	rows, err := se.db.Query(
		`SELECT id, tenant_id, COALESCE(workspace_id,''), source, payload, ts, value
		 FROM core.event_ledger
		 WHERE tenant_id = $1 AND COALESCE(workspace_id,'') = $2
		 ORDER BY ts DESC, id DESC LIMIT $3`,
		tenantID, workspaceID, limit,
	)
	if err != nil {
		log.Printf("[lakehouse] QueryTenantData failed: %v", err)
		return []EventRecord{}
	}
	defer rows.Close()

	records := make([]EventRecord, 0, limit)
	for rows.Next() {
		var rec EventRecord
		var payload []byte
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.WorkspaceID, &rec.Source, &payload, &rec.Timestamp, &rec.Value); err != nil {
			log.Printf("[lakehouse] QueryTenantData: scan failed: %v", err)
			return records
		}
		rec.Payload = map[string]interface{}{}
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &rec.Payload)
		}
		records = append(records, rec)
	}
	return records
}

func (se *StorageEngine) GetAnomalies(tenantID, workspaceID string) []AnomalyAlert {
	rows, err := se.db.Query(
		`SELECT id, event_id, tenant_id, COALESCE(workspace_id,''), metric_name, dataset,
		        value, mean, std_dev, z_score, sigma_score, severity, ts, COALESCE(message,''), COALESCE(notebook_url,'')
		 FROM core.anomaly_alerts
		 WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR COALESCE(workspace_id,'') = $2)
		 ORDER BY ts DESC LIMIT 100`,
		tenantID, workspaceID,
	)
	if err != nil {
		log.Printf("[lakehouse] GetAnomalies failed: %v", err)
		return []AnomalyAlert{}
	}
	defer rows.Close()

	result := make([]AnomalyAlert, 0)
	for rows.Next() {
		var a AnomalyAlert
		if err := rows.Scan(
			&a.ID, &a.EventID, &a.TenantID, &a.WorkspaceID, &a.MetricName, &a.Dataset,
			&a.Value, &a.Mean, &a.StdDev, &a.ZScore, &a.SigmaScore, &a.Severity,
			&a.Timestamp, &a.Message, &a.NotebookURL,
		); err != nil {
			log.Printf("[lakehouse] GetAnomalies: scan failed: %v", err)
			return result
		}
		result = append(result, a)
	}
	return result
}

func (se *StorageEngine) GetStats(tenantID, workspaceID string) map[string]interface{} {
	var count int64
	var avg sql.NullFloat64
	var last sql.NullTime
	err := se.db.QueryRow(
		`SELECT COUNT(*), AVG(value), MAX(ts) FROM core.event_ledger
		 WHERE tenant_id = $1 AND COALESCE(workspace_id,'') = $2`,
		tenantID, workspaceID,
	).Scan(&count, &avg, &last)
	if err != nil {
		log.Printf("[lakehouse] GetStats failed: %v", err)
		return map[string]interface{}{"tenant_id": tenantID, "workspace_id": workspaceID, "total_records": 0}
	}
	lastAt := time.Now().Format(time.RFC3339)
	if last.Valid {
		lastAt = last.Time.Format(time.RFC3339)
	}
	avgVal := 0.0
	if avg.Valid {
		avgVal = avg.Float64
	}
	return map[string]interface{}{
		"tenant_id":        tenantID,
		"workspace_id":     workspaceID,
		"total_records":    count,
		"avg_value":        avgVal,
		"anomalies_count":  len(se.GetAnomalies(tenantID, workspaceID)),
		"last_ingested_at": lastAt,
	}
}

func scopeKey(tenantID, workspaceID string) string {
	return tenantID + "\x00" + workspaceID
}

// RecordsForScope is kept as a deprecated in-memory helper for tests that
// construct engines without durable storage. Production paths use SQL.
func (se *StorageEngine) recordsForScope(tenantID, workspaceID string) []EventRecord {
	return se.QueryTenantData(tenantID, workspaceID, 0)
}
