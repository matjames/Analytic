package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — OBJECTIVES & KPI FRAMEWORK (SPRINT 4)
//
// StatGovernance remains the source of truth for governance-owned objectives
// and workflows. Phase XII consumes governance information (Event Bus / API /
// UOI / Data Fabric) and maintains the strategic performance projection.
//
// Every KPI measurement distinguishes ACTUAL | ESTIMATED | STALE | MISSING |
// INVALID. Stale data is NEVER presented as current data (directive §13).
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ─── Objectives ────────────────────────────────────────────────────────────────

func (s *Phase12Store) UpsertObjective(o *InstitutionalObjective) error {
	if o == nil {
		return fmt.Errorf("objective is nil")
	}
	if o.TenantID == "" || o.Name == "" {
		return fmt.Errorf("objective requires tenant_id and name")
	}
	if o.Status == "" {
		o.Status = "ACTIVE"
	}
	now := utcNow()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}
	o.UpdatedAt = now
	if o.ID == "" {
		o.ID = fmt.Sprintf("obj_%d", time.Now().UnixNano())
	}
	if o.CanonicalID == "" {
		o.CanonicalID = "uoi_gov_objective_" + o.ID
	}

	s.mu.Lock()
	s.objectives[objectKey(o.TenantID, o.ID)] = o
	s.mu.Unlock()
	persistPhase12Objective(o)
	return nil
}

func (s *Phase12Store) ListObjectives(tenantID string) []*InstitutionalObjective {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*InstitutionalObjective
	for _, o := range s.objectives {
		if o.TenantID == tenantID {
			out = append(out, o)
		}
	}
	return out
}

func persistPhase12Objective(o *InstitutionalObjective) {
	if dbPool == nil {
		return
	}
	meta, _ := json.Marshal(o.Metadata)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO institutional_objectives
		(id, tenant_id, canonical_id, name, description, owner, status, governance_ref,
		 start_date, end_date, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13)
		ON CONFLICT (tenant_id, canonical_id) DO UPDATE SET
			name = EXCLUDED.name, description = EXCLUDED.description, owner = EXCLUDED.owner,
			status = EXCLUDED.status, governance_ref = EXCLUDED.governance_ref,
			metadata = EXCLUDED.metadata, updated_at = EXCLUDED.updated_at`,
		o.ID, o.TenantID, o.CanonicalID, o.Name, o.Description, o.Owner, o.Status, o.GovernanceRef,
		o.StartDate, o.EndDate, string(meta), o.CreatedAt, o.UpdatedAt)
	if err != nil {
		log.Printf("phase12: persist objective failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}