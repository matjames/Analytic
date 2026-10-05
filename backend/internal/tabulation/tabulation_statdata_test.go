package tabulation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// pgStatDataDB opens the StatData service database (same Postgres server the
// service uses in Compose) so core tests can bind against real pipeline
// output. Skips when unreachable.
func pgStatDataDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("STATDATA_TEST_DSN")
	if dsn == "" {
		host := os.Getenv("STATGATE_TEST_HOST")
		if host == "" {
			host = "localhost"
		}
		dsn = "postgres://pgtest:pgtest_secret@" + host + ":5432/statdata_test?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("statdata postgres unavailable (%v); skipping dataset-backed tabulation test", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("statdata postgres ping failed (%v); skipping dataset-backed tabulation test", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestTabulateFromStatDataDataset binds tabulation to rows persisted by the
// desimulated StatData pipeline engine: seed a source dataset, run an
// extract/filter/load flow equivalent through the records table, then
// tabulate straight from storage. No request-embedded data_records.
func TestTabulateFromStatDataDataset(t *testing.T) {
	db := pgStatDataDB(t)
	ctx := context.Background()

	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS statdata`); err != nil {
		t.Fatalf("schema create failed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS statdata.dataset_records (
		id BIGSERIAL PRIMARY KEY, dataset_id VARCHAR(64) NOT NULL,
		record JSONB NOT NULL, tenant_id VARCHAR(64) NOT NULL DEFAULT 'default',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		t.Fatalf("records table create failed: %v", err)
	}

	datasetID := fmt.Sprintf("ds-core-tab-%d", time.Now().UnixNano())
	seed := []map[string]interface{}{
		{"region": "North", "sex": "F", "count": 30},
		{"region": "North", "sex": "M", "count": 20},
		{"region": "South", "sex": "F", "count": 10},
		{"region": "South", "sex": "M", "count": 40},
	}
	for _, row := range seed {
		payload, _ := json.Marshal(row)
		if _, err := db.ExecContext(ctx,
			`INSERT INTO statdata.dataset_records (dataset_id, record, tenant_id) VALUES ($1,$2,'default')`,
			datasetID, payload); err != nil {
			t.Fatalf("seed insert failed: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM statdata.dataset_records WHERE dataset_id = $1`, datasetID)
	})

	stored, err := LoadRecordsFromStatData(ctx, db, datasetID, "statdata", 0)
	if err != nil {
		t.Fatalf("LoadRecordsFromStatData failed: %v", err)
	}
	if len(stored) != 4 {
		t.Fatalf("expected 4 persisted rows, got %d", len(stored))
	}

	req := TabulationRequest{
		Title:       "Regional distribution from persisted dataset",
		RowVariable: "region",
		ColVariable: "sex",
		Aggregation: AggCount,
		DataRecords: stored,
	}
	result, err := NewEngine().Tabulate(ctx, req)
	if err != nil {
		t.Fatalf("Tabulate failed: %v", err)
	}
	if result.TotalRecordCount != 4 {
		t.Errorf("expected 4 total records, got %d", result.TotalRecordCount)
	}
	if result.RowTotals["North"] != 2 || result.RowTotals["South"] != 2 {
		t.Errorf("wrong regional totals: %v", result.RowTotals)
	}
	northF := result.Matrix["North"]["F"]
	if northF.Count != 1 {
		t.Errorf("expected North/F count 1, got %+v", northF)
	}
}