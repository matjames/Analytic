package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/lib/pq"

	"statfederation-backend/internal/models"
)

func TestPostgresWorkspaceObjectLinkCertification(t *testing.T) {
	dsn := os.Getenv("STATFEDERATION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STATFEDERATION_TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	applyMigration(t, db, filepath.Join("..", "..", "migrations", "001_init.sql"))
	applyMigration(t, db, filepath.Join("..", "..", "migrations", "002_workspace_scope.sql"))

	store := NewPGStore(db)
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	tenantID := "tenant-cert-" + suffix
	workspaceOne := "workspace-one-" + suffix
	workspaceTwo := "workspace-two-" + suffix
	sourceID := "source-" + suffix
	targetID := "target-" + suffix

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM statfederation.object_links WHERE source_id = $1 AND target_id = $2`, sourceID, targetID)
	})

	link := models.ObjectLink{
		SourceType:   "dataset",
		SourceID:     sourceID,
		TargetType:   "indicator",
		TargetID:     targetID,
		Relationship: "certifies",
		TenantID:     tenantID,
		WorkspaceID:  workspaceOne,
		CreatedBy:    "integration-test",
	}
	if err := store.CreateObjectLink(context.Background(), &link); err != nil {
		t.Fatalf("create workspace one object link: %v", err)
	}
	link.WorkspaceID = workspaceTwo
	if err := store.CreateObjectLink(context.Background(), &link); err != nil {
		t.Fatalf("create workspace two object link with same endpoints: %v", err)
	}

	linksOne, err := store.GetObjectLinks(context.WithValue(context.Background(), "workspace_id", workspaceOne), "dataset", sourceID, tenantID)
	if err != nil {
		t.Fatalf("get workspace one links: %v", err)
	}
	if len(linksOne) != 1 || linksOne[0].WorkspaceID != workspaceOne {
		t.Fatalf("workspace one should see exactly its link, got %#v", linksOne)
	}

	linksTwo, err := store.GetObjectLinks(context.WithValue(context.Background(), "workspace_id", workspaceTwo), "dataset", sourceID, tenantID)
	if err != nil {
		t.Fatalf("get workspace two links: %v", err)
	}
	if len(linksTwo) != 1 || linksTwo[0].WorkspaceID != workspaceTwo {
		t.Fatalf("workspace two should see exactly its link, got %#v", linksTwo)
	}
}

func applyMigration(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	if _, err := db.Exec(string(sqlBytes)); err != nil {
		t.Fatalf("apply migration %s: %v", path, err)
	}
}
