package main

import "time"

// ─── Institutional Object Registry (projection) ──────────────────────────────

// InstitutionalObject is a projection of a UOI-canonical object into the
// intelligence layer. It is an identity/relationship reference, NOT a copy
// of the source record. Every object MUST carry the fields below.
type InstitutionalObject struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	CanonicalID      string                 `json:"canonical_id"`
	ObjectType       string                 `json:"object_type"`
	SourceSystem     string                 `json:"source_system"`
	SourceObjectID   string                 `json:"source_object_id"`
	DisplayName      string                 `json:"display_name"`
	ProjectionStatus string                 `json:"projection_status"` // ACTIVE | STALE | DELETED
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// ─── Knowledge Graph Edge ────────────────────────────────────────────────────

type GraphEdge struct {
	ID                 string                 `json:"id"`
	TenantID           string                 `json:"tenant_id"`
	SubjectCanonicalID string                 `json:"subject_canonical_id"`
	RelationshipType   string                 `json:"relationship_type"`
	ObjectCanonicalID  string                 `json:"object_canonical_id"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	ProvenanceType     ProvenanceType         `json:"provenance_type"`
	Confidence         float64                `json:"confidence"`
	SourceEventID      string                 `json:"source_event_id,omitempty"`
	ValidFrom          time.Time              `json:"valid_from"`
	ValidUntil         *time.Time             `json:"valid_until,omitempty"` // nil = still valid
	CreatedBy          string                 `json:"created_by"`
	CreatedAt          time.Time              `json:"created_at"`
}

// GraphTraversalPath is one result of a depth-limited graph traversal.
type GraphTraversalPath struct {
	Hops  int            `json:"hops"`
	Nodes []string       `json:"nodes"` // ordered canonical ids
	Edges []GraphEdgeRef `json:"edges"`
}

type GraphEdgeRef struct {
	ID               string `json:"id"`
	RelationshipType string `json:"relationship_type"`
	From             string `json:"from"`
	To               string `json:"to"`
}

// ─── Intelligence Signals ────────────────────────────────────────────────────

type IntelligenceSignal struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	SignalType    string                 `json:"signal_type"`
	Domain        string                 `json:"domain"`
	Severity      int                    `json:"severity"` // 0-100
	SourceSystem  string                 `json:"source_system"`
	SourceEventID string                 `json:"source_event_id,omitempty"`
	Status        string                 `json:"status"` // ACTIVE | SUPERSEDED
	Payload       map[string]interface{} `json:"payload,omitempty"`
	CorrelationID string                 `json:"correlation_id"`
	Evidence      string                 `json:"evidence"`
	CreatedAt     time.Time              `json:"created_at"`
	EvaluatedAt   *time.Time             `json:"evaluated_at,omitempty"`
}

// ─── Institutional Condition ─────────────────────────────────────────────────

// SupportingSignalRef is the evidence pointer inside a condition record.
type SupportingSignalRef struct {
	SignalID   string `json:"signal_id"`
	SignalType string `json:"signal_type"`
	Domain     string `json:"domain"`
	Severity   int    `json:"severity"`
	Evidence   string `json:"evidence"`
}

type AffectedDomain struct {
	Domain      string `json:"domain"`
	Severity    int    `json:"severity"` // max severity contributing to this domain
	SignalCount int    `json:"signal_count"`
}

type InstitutionalCondition struct {
	ID                   string                `json:"id"`
	TenantID             string                `json:"tenant_id"`
	ConditionLevel       ConditionLevel        `json:"condition_level"`
	CompositeScore       int                   `json:"composite_score"` // 0-100
	CalculationTimestamp time.Time             `json:"calculation_timestamp"`
	SupportingSignals    []SupportingSignalRef `json:"supporting_signals"`
	AffectedDomains      []AffectedDomain      `json:"affected_domains"`
	Explanation          string                `json:"explanation"`
	CalculationVersion   string                `json:"calculation_version"`
	CorrelationID        string                `json:"correlation_id"`
}