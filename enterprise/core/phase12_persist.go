package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — PERSISTENCE (OBJECTS & EDGES)
//
// Write-through to PostgreSQL when dbPool is configured; no-op otherwise
// (in-memory store remains authoritative in development/test).
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

func persistPhase12Object(o *InstitutionalObject) {
	if dbPool == nil {
		return
	}
	meta, _ := json.Marshal(o.Metadata)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO institutional_objects
		(id, tenant_id, canonical_id, object_type, source_system, source_object_id,
		 display_name, projection_status, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11)
		ON CONFLICT (tenant_id, canonical_id) DO UPDATE SET
			object_type = EXCLUDED.object_type,
			source_system = EXCLUDED.source_system,
			source_object_id = EXCLUDED.source_object_id,
			display_name = EXCLUDED.display_name,
			projection_status = EXCLUDED.projection_status,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at`,
		o.ID, o.TenantID, o.CanonicalID, o.ObjectType, o.SourceSystem, o.SourceObjectID,
		o.DisplayName, o.ProjectionStatus, string(meta), o.CreatedAt, o.UpdatedAt)
	if err != nil {
		log.Printf("phase12: persist object failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}

func persistPhase12Edge(e *GraphEdge) {
	if dbPool == nil {
		return
	}
	meta, _ := json.Marshal(e.Metadata)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO institutional_graph_edges
		(id, tenant_id, subject_canonical_id, relationship_type, object_canonical_id,
		 metadata, provenance_type, confidence, source_event_id, valid_from, valid_until, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			valid_until = EXCLUDED.valid_until`,
		e.ID, e.TenantID, e.SubjectCanonicalID, e.RelationshipType, e.ObjectCanonicalID,
		string(meta), string(e.ProvenanceType), e.Confidence, e.SourceEventID,
		e.ValidFrom, e.ValidUntil, e.CreatedBy, e.CreatedAt)
	if err != nil {
		log.Printf("phase12: persist edge failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}

func retirePhase12Edge(e *GraphEdge) {
	if dbPool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx,
		`UPDATE institutional_graph_edges SET valid_until = $1 WHERE id = $2`,
		e.ValidUntil, e.ID)
	if err != nil {
		log.Printf("phase12: retire edge failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}