package lakehouse

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// pgTestDB opens a real PostgreSQL connection for lakehouse tests. It skips
// the test when the database is not reachable so unit runs without Postgres
// stay green; CI with Postgres exercises the durable path.
// testHost resolves the Postgres host for tests. STATGATE_TEST_DSN takes
// precedence; STATGATE_TEST_HOST overrides the host only.
func testHost() string {
	if h := os.Getenv("STATGATE_TEST_HOST"); h != "" {
		return h
	}
	return "localhost"
}

func pgTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("STATGATE_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://pgtest:pgtest_secret@" + testHost() + ":5432/statgate_test?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("postgres unavailable (%v); skipping durable lakehouse test", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("postgres ping failed (%v); skipping durable lakehouse test", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func mustEngine(t *testing.T, db *sql.DB) *StorageEngine {
	t.Helper()
	se, err := NewStorageEngine(db)
	if err != nil {
		t.Fatalf("NewStorageEngine failed: %v", err)
	}
	return se
}

// TestRecordEventAppendsToLedger verifies that domain events written via
// RecordEvent are durably queryable and do not spuriously create anomalies.
func TestRecordEventAppendsToLedger(t *testing.T) {
	se := mustEngine(t, pgTestDB(t))
	tenantID := fmt.Sprintf("tenant-test-%d", time.Now().UnixNano())
	workspaceID := "workspace-test"

	baselineAnomalies := len(se.GetAnomalies(tenantID, ""))

	se.RecordEvent(EventRecord{
		TenantID:   tenantID,
		WorkspaceID: workspaceID,
		Source:     "institutional_condition",
		Payload: map[string]interface{}{
			"event_type": "institutional.condition.changed",
			"level":      "OPTIMAL",
		},
		Timestamp: time.Now().UTC(),
		Value:     95.0,
	})

	records := se.QueryTenantData(tenantID, workspaceID, 5)
	if len(records) == 0 {
		t.Fatal("expected a recorded event in tenant data")
	}
	last := records[0]
	if last.Source != "institutional_condition" {
		t.Errorf("expected institutional_condition source, got %q", last.Source)
	}
	if last.Payload["event_type"] != "institutional.condition.changed" {
		t.Errorf("expected event_type payload, got %v", last.Payload["event_type"])
	}

	after := len(se.GetAnomalies(tenantID, ""))
	if after != baselineAnomalies {
		t.Errorf("RecordEvent must not create anomalies (before=%d after=%d)", baselineAnomalies, after)
	}
}

// TestIngestPersistsAndDetectsRealAnomaly ingests 30 steady values then an
// outlier, and proves both durability (restart-safe readback) and that the
// 3-sigma detector fires on the persisted window only.
func TestIngestPersistsAndDetectsRealAnomaly(t *testing.T) {
	se := mustEngine(t, pgTestDB(t))
	tenantID := fmt.Sprintf("tenant-ingest-%d", time.Now().UnixNano())
	workspaceID := "workspace-ingest"
	source := "test_sensor"

	for i := 0; i < 30; i++ {
		alert := se.Ingest(EventRecord{
			ID:          fmt.Sprintf("steady-%d-%d", time.Now().UnixNano(), i),
			TenantID:    tenantID,
			WorkspaceID: workspaceID,
			Source:      source,
			Payload:     map[string]interface{}{"temperature": 100.0 + float64(i%3)},
			Timestamp:   time.Now().UTC(),
			Value:       100.0 + float64(i%3),
		})
		if alert.ID != "" {
			t.Fatalf("steady value %d unexpectedly raised an alert: %+v", i, alert)
		}
	}

	outlier := se.Ingest(EventRecord{
		ID:          fmt.Sprintf("outlier-%d", time.Now().UnixNano()),
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		Source:      source,
		Payload:     map[string]interface{}{"temperature": 250.0},
		Timestamp:   time.Now().UTC(),
		Value:       250.0,
	})
	if outlier.ID == "" {
		t.Fatal("expected a 3-sigma alert for the outlier, got none")
	}
	if math.Abs(outlier.ZScore) < 3.0 {
		t.Errorf("expected |z| >= 3, got %v", outlier.ZScore)
	}

	// Restart-safe: a second engine over the same DB must see all 31 rows.
	se2, err := NewStorageEngine(se.db)
	if err != nil {
		t.Fatalf("second NewStorageEngine failed: %v", err)
	}
	data := se2.QueryTenantData(tenantID, workspaceID, 100)
	if len(data) != 31 {
		t.Fatalf("expected 31 durable rows after re-open, got %d", len(data))
	}

	stats := se2.GetStats(tenantID, workspaceID)
	if stats["total_records"] != int64(31) {
		t.Errorf("expected stats total_records=31, got %v", stats["total_records"])
	}
	if stats["anomalies_count"] != 1 {
		t.Errorf("expected 1 persisted anomaly, got %v", stats["anomalies_count"])
	}
}

// TestQueryIsolation proves tenant/workspace scoping reads only its own rows.
func TestQueryIsolation(t *testing.T) {
	se := mustEngine(t, pgTestDB(t))
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())

	se.RecordEvent(EventRecord{ID: "iso-a-" + stamp, TenantID: "tenant-a-" + stamp, WorkspaceID: "ws-1", Source: "probe", Payload: map[string]interface{}{"n": 1}, Timestamp: time.Now().UTC(), Value: 1})
	se.RecordEvent(EventRecord{ID: "iso-b-" + stamp, TenantID: "tenant-b-" + stamp, WorkspaceID: "ws-1", Source: "probe", Payload: map[string]interface{}{"n": 2}, Timestamp: time.Now().UTC(), Value: 2})

	if got := se.QueryTenantData("tenant-a-"+stamp, "ws-1", 10); len(got) != 1 {
		t.Errorf("expected 1 row for tenant-a, got %d", len(got))
	}
	if got := se.QueryTenantData("tenant-b-"+stamp, "ws-1", 10); len(got) != 1 {
		t.Errorf("expected 1 row for tenant-b, got %d", len(got))
	}
	if got := se.QueryTenantData("tenant-a-"+stamp, "ws-other", 10); len(got) != 0 {
		t.Errorf("expected 0 rows for a foreign workspace, got %d", len(got))
	}
}