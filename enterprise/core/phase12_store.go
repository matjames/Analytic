package main

// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•
// PHASE XII â€” INTELLIGENCE STORE
//
// The Phase XII store is a tenant-scoped projection store. In-memory for
// deterministic behaviour and offline/test operation, with write-through to
// PostgreSQL when dbPool is configured (production).
//
// Every accessor forces tenant_id. There is NO anonymous fallback and NO
// tenant fallback: missing tenant context fails closed.
// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Phase12Store struct {
	mu sync.RWMutex

	objects         map[string]*InstitutionalObject // key tenantID + "|" + canonicalID
	edges           map[string][]*GraphEdge         // key tenantID
	signals         map[string][]*IntelligenceSignal
	conditions      map[string]*InstitutionalCondition // latest per tenant
	objectives      map[string]*InstitutionalObjective
	kpis            map[string]*KPI
	measurements    map[string][]*KPIMeasurement
	riskEvents      map[string][]*RiskEvent
	recommendations map[string]*IntelligenceRecommendation
	aiAudit         []*AIAuditEntry
	nextMeasID      int64
}

func newPhase12Store() *Phase12Store {
	return &Phase12Store{
		objects:         map[string]*InstitutionalObject{},
		edges:           map[string][]*GraphEdge{},
		signals:         map[string][]*IntelligenceSignal{},
		conditions:      map[string]*InstitutionalCondition{},
		objectives:      map[string]*InstitutionalObjective{},
		kpis:            map[string]*KPI{},
		measurements:    map[string][]*KPIMeasurement{},
		riskEvents:      map[string][]*RiskEvent{},
		recommendations: map[string]*IntelligenceRecommendation{},
		aiAudit:         []*AIAuditEntry{},
	}
}

// Global Phase XII store instance.
var phase12 = newPhase12Store()

func objectKey(tenantID, canonicalID string) string {
	return tenantID + "|" + canonicalID
}

// utcNow returns the current UTC time as a time.Time. (The pre-existing
// nowUTC() in analytics_models.go returns a formatted string; this one is
// used where time.Time is required.)
func utcNow() time.Time {
	return time.Now().UTC()
}

// â”€â”€â”€ Object Registry â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// ProjectObject upserts a UOI-canonical object projection. Idempotent:
// projecting the same canonical_id twice does not create a duplicate.
func (s *Phase12Store) ProjectObject(obj *InstitutionalObject) error {
	if obj == nil {
		return fmt.Errorf("institutional object is nil")
	}
	if obj.TenantID == "" || obj.CanonicalID == "" {
		return fmt.Errorf("institutional object requires tenant_id and canonical_id")
	}
	if obj.SourceSystem == "" || obj.SourceObjectID == "" {
		return fmt.Errorf("institutional object requires source_system and source_object_id")
	}
	if !validObjectType(obj.ObjectType) {
		return fmt.Errorf("invalid object_type %q", obj.ObjectType)
	}
	if obj.ProjectionStatus == "" {
		obj.ProjectionStatus = "ACTIVE"
	}
	now := utcNow()
	if obj.CreatedAt.IsZero() {
		obj.CreatedAt = now
	}
	obj.UpdatedAt = now

	s.mu.Lock()
	key := objectKey(obj.TenantID, obj.CanonicalID)
	if existing, ok := s.objects[key]; ok {
		existing.ObjectType = obj.ObjectType
		existing.SourceSystem = obj.SourceSystem
		existing.SourceObjectID = obj.SourceObjectID
		existing.DisplayName = obj.DisplayName
		existing.ProjectionStatus = obj.ProjectionStatus
		existing.Metadata = obj.Metadata
		existing.UpdatedAt = now
		s.mu.Unlock()
		persistPhase12Object(existing)
		return nil
	}
	if obj.ID == "" {
		obj.ID = fmt.Sprintf("iobj_%d", time.Now().UnixNano())
	}
	s.objects[key] = obj
	s.mu.Unlock()
	persistPhase12Object(obj)
	return nil
}

// GetObject returns the projected object for a tenant, or nil.
func (s *Phase12Store) GetObject(tenantID, canonicalID string) *InstitutionalObject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.objects[objectKey(tenantID, canonicalID)]
}

// ListObjects returns tenant-scoped objects, optionally filtered by object_type.
func (s *Phase12Store) ListObjects(tenantID, objectType string) []*InstitutionalObject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*InstitutionalObject
	for _, o := range s.objects {
		if o.TenantID != tenantID {
			continue
		}
		if objectType != "" && o.ObjectType != objectType {
			continue
		}
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CanonicalID < out[j].CanonicalID })
	return out
}
