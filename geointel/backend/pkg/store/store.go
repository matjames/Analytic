package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB

func DB() *sql.DB { return db }

func IsReady() bool {
	if db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return db.PingContext(ctx) == nil
}

func Init(dsn string) error {
	if dsn == "" {
		return errors.New("database DSN must be provided")
	}
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	if err = db.Ping(); err != nil {
		return err
	}
	return ensureSchema(context.Background())
}

func ensureSchema(ctx context.Context) error {
	stmts := []string{
		// ── P44: GIS layers & features ──
		`CREATE TABLE IF NOT EXISTS geo_layers (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			kind TEXT NOT NULL, geometry_type TEXT NOT NULL, source TEXT,
			tile_layer TEXT, metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			status TEXT NOT NULL DEFAULT 'active', created_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_gl_tenant ON geo_layers(tenant_id)`,
		`CREATE TABLE IF NOT EXISTS geo_features (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, layer_id TEXT NOT NULL,
			name TEXT NOT NULL, geometry TEXT NOT NULL, properties JSONB NOT NULL DEFAULT '{}'::jsonb,
			centroid TEXT, created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_gf_layer ON geo_features(layer_id)`,
		`CREATE TABLE IF NOT EXISTS rasters (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			source TEXT, bands INT NOT NULL DEFAULT 1, width INT, height INT,
			bbox TEXT, stats JSONB NOT NULL DEFAULT '{}'::jsonb, file_path TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		// ── Remote Sensing ──
		`CREATE TABLE IF NOT EXISTS scenes (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			platform TEXT NOT NULL, capture_time TIMESTAMPTZ NOT NULL DEFAULT now(),
			bbox TEXT, cloud_cover DOUBLE PRECISION NOT NULL DEFAULT 0,
			resolution DOUBLE PRECISION NOT NULL DEFAULT 10,
			status TEXT NOT NULL DEFAULT 'ingested', thumb_url TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_scenes_tenant ON scenes(tenant_id, platform)`,
		`CREATE TABLE IF NOT EXISTS spectral_indices (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, scene_id TEXT NOT NULL,
			index_type TEXT NOT NULL, values JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		// ── Drone integration ──
		`CREATE TABLE IF NOT EXISTS drones (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
			model TEXT NOT NULL, serial TEXT, status TEXT NOT NULL DEFAULT 'grounded',
			capabilities JSONB NOT NULL DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS flight_plans (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, drone_id TEXT NOT NULL,
			name TEXT NOT NULL, waypoints JSONB NOT NULL DEFAULT '{}'::jsonb,
			altitude DOUBLE PRECISION NOT NULL DEFAULT 120, status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS flights (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, drone_id TEXT NOT NULL,
			plan_id TEXT, status TEXT NOT NULL DEFAULT 'planned',
			summary JSONB NOT NULL DEFAULT '{}'::jsonb,
			started_at TIMESTAMPTZ, ended_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS telemetry (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, drone_id TEXT NOT NULL,
			flight_id TEXT NOT NULL, ts TIMESTAMPTZ NOT NULL DEFAULT now(),
			lat DOUBLE PRECISION NOT NULL, lng DOUBLE PRECISION NOT NULL,
			alt DOUBLE PRECISION NOT NULL DEFAULT 0, battery DOUBLE PRECISION NOT NULL DEFAULT 100,
			speed DOUBLE PRECISION NOT NULL DEFAULT 0)`,
		`CREATE INDEX IF NOT EXISTS idx_telemetry_flight ON telemetry(flight_id)`,
		// ── Tiles & geocoding ──
		`CREATE TABLE IF NOT EXISTS map_tiles (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, layer_id TEXT NOT NULL,
			name TEXT NOT NULL, z INT NOT NULL, x INT NOT NULL, y INT NOT NULL,
			format TEXT NOT NULL DEFAULT 'png', path TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS geocode_cache (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, query TEXT,
			address TEXT, lat DOUBLE PRECISION NOT NULL, lng DOUBLE PRECISION NOT NULL,
			confidence DOUBLE PRECISION NOT NULL DEFAULT 0, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		// ── Object linkage ──
		`CREATE TABLE IF NOT EXISTS object_links (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL,
			source_type TEXT NOT NULL, source_id TEXT NOT NULL,
			target_type TEXT NOT NULL, target_id TEXT NOT NULL,
			relationship TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_links_source ON object_links(source_type, source_id)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

func newID() string   { return uuid.New().String() }
func nowT() time.Time { return time.Now().UTC() }

func jsonB(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func itoa(n int) string { return strconv.Itoa(n) }

func lower(s string) string { return strings.ToLower(s) }

type row interface{ Scan(...interface{}) error }

func jsonUnmarshal(b []byte, v interface{}) {
	if len(b) > 0 {
		_ = json.Unmarshal(b, v)
	}
}