package main

import (
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestPostgresTrustPersistenceCertification(t *testing.T) {
	dsn := os.Getenv("STATTRUST_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STATTRUST_TEST_DATABASE_URL is not set")
	}

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	db = database
	runMigrations()

	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	runID := time.Now().UTC().Format("150405")
	tenantID := "tenant-cert-" + runID
	workspaceOne := "workspace-one-" + runID
	workspaceTwo := "workspace-two-" + runID
	artifactID := "artifact-" + suffix + "-" + runID

	if _, err := database.Exec(`DELETE FROM artifact_provenance WHERE tenant_id = $1`, tenantID); err != nil {
		t.Fatalf("pre-clean provenance: %v", err)
	}
	if _, err := database.Exec(`DELETE FROM audit_ledger WHERE tenant_id = $1`, tenantID); err != nil {
		t.Fatalf("pre-clean ledger: %v", err)
	}

	t.Cleanup(func() {
		_, _ = database.Exec(`DELETE FROM artifact_provenance WHERE tenant_id = $1 AND artifact_id = $2`, tenantID, artifactID)
		_, _ = database.Exec(`DELETE FROM audit_ledger WHERE tenant_id = $1`, tenantID)
		db = nil
		globalStore = nil
	})

	persistLedgerBlock(AuditLedgerBlock{
		Index:       1,
		TenantID:    tenantID,
		WorkspaceID: workspaceOne,
		PrevHash:    strings.Repeat("0", 64),
		RecordHash:  strings.Repeat("1", 64),
		MerkleRoot:  strings.Repeat("1", 64),
		EventType:   "certification.ledger",
		SourceApp:   "StatTrust",
		ActorID:     "integration-test",
		Payload:     map[string]interface{}{"workspace": workspaceOne},
		Timestamp:   time.Now().UTC(),
	})
	persistLedgerBlock(AuditLedgerBlock{
		Index:       1,
		TenantID:    tenantID,
		WorkspaceID: workspaceTwo,
		PrevHash:    strings.Repeat("0", 64),
		RecordHash:  strings.Repeat("2", 64),
		MerkleRoot:  strings.Repeat("2", 64),
		EventType:   "certification.ledger",
		SourceApp:   "StatTrust",
		ActorID:     "integration-test",
		Payload:     map[string]interface{}{"workspace": workspaceTwo},
		Timestamp:   time.Now().UTC(),
	})

	persistProvenance(ArtifactProvenance{
		ID:             "prov-one-" + runID,
		TenantID:       tenantID,
		WorkspaceID:    workspaceOne,
		ArtifactID:     artifactID,
		ArtifactName:   "Workspace One Artifact",
		ArtifactType:   "DATASET",
		OriginApp:      "integration-test",
		CurrentOwner:   "workspace-one-owner",
		Sha256Checksum: strings.Repeat("a", 64),
		TSATimestamp:   time.Now().UTC(),
		CustodyChain:   []CustodyEvent{{Timestamp: time.Now().UTC(), Actor: "integration-test", Action: "CREATED", App: "StatTrust"}},
		IntegrityState: "VERIFIED",
		LedgerIndex:    1,
	})
	persistProvenance(ArtifactProvenance{
		ID:             "prov-two-" + runID,
		TenantID:       tenantID,
		WorkspaceID:    workspaceTwo,
		ArtifactID:     artifactID,
		ArtifactName:   "Workspace Two Artifact",
		ArtifactType:   "DATASET",
		OriginApp:      "integration-test",
		CurrentOwner:   "workspace-two-owner",
		Sha256Checksum: strings.Repeat("b", 64),
		TSATimestamp:   time.Now().UTC(),
		CustodyChain:   []CustodyEvent{{Timestamp: time.Now().UTC(), Actor: "integration-test", Action: "CREATED", App: "StatTrust"}},
		IntegrityState: "VERIFIED",
		LedgerIndex:    1,
	})

	globalStore = &MemStore{
		ledger:     make([]AuditLedgerBlock, 0),
		provenance: make(map[string]ArtifactProvenance),
	}
	loadTrustPersistence()

	ledgerOne := globalStore.ListLedgerScoped(tenantID, workspaceOne)
	if len(ledgerOne) != 1 || ledgerOne[0].WorkspaceID != workspaceOne {
		t.Fatalf("workspace one should reload only its ledger block, got %#v", ledgerOne)
	}
	ledgerTwo := globalStore.ListLedgerScoped(tenantID, workspaceTwo)
	if len(ledgerTwo) != 1 || ledgerTwo[0].WorkspaceID != workspaceTwo {
		t.Fatalf("workspace two should reload only its ledger block, got %#v", ledgerTwo)
	}

	provOne, ok := globalStore.GetProvenanceScoped(artifactID, tenantID, workspaceOne)
	if !ok || provOne.CurrentOwner != "workspace-one-owner" {
		t.Fatalf("workspace one provenance mismatch: %#v loaded=%#v", provOne, globalStore.provenance)
	}
	provTwo, ok := globalStore.GetProvenanceScoped(artifactID, tenantID, workspaceTwo)
	if !ok || provTwo.CurrentOwner != "workspace-two-owner" {
		t.Fatalf("workspace two provenance mismatch: %#v loaded=%#v", provTwo, globalStore.provenance)
	}
}
