package main

import "time"

// ─── Objectives & KPIs ───────────────────────────────────────────────────────

type InstitutionalObjective struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	CanonicalID   string                 `json:"canonical_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Owner         string                 `json:"owner"`
	Status        string                 `json:"status"`
	GovernanceRef string                 `json:"governance_ref"` // reference to StatGovernance objective
	StartDate     *time.Time             `json:"start_date,omitempty"`
	EndDate       *time.Time             `json:"end_date,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// KpiDataStatus distinguishes data provenance quality. NEVER display stale
// data as current data.
type KpiDataStatus string

const (
	KpiDataActual    KpiDataStatus = "ACTUAL"
	KpiDataEstimated KpiDataStatus = "ESTIMATED"
	KpiDataStale     KpiDataStatus = "STALE"
	KpiDataMissing   KpiDataStatus = "MISSING"
	KpiDataInvalid   KpiDataStatus = "INVALID"
)

type KPI struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	ObjectiveID       string                 `json:"objective_id,omitempty"`
	CanonicalID       string                 `json:"canonical_id"`
	Name              string                 `json:"name"`
	Definition        string                 `json:"definition"`
	Owner             string                 `json:"owner"`
	Target            float64                `json:"target"`
	MeasurementPeriod string                 `json:"measurement_period"`
	ActualValue       *float64               `json:"actual_value,omitempty"`
	Unit              string                 `json:"unit"`
	Source            string                 `json:"source"`
	Status            string                 `json:"status"`
	DataStatus        KpiDataStatus          `json:"data_status"`
	FreshnessWindowH  int                    `json:"freshness_window_h"`
	LastMeasuredAt    *time.Time             `json:"last_measured_at,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type KPIMeasurement struct {
	ID         int64     `json:"id"`
	KpiID      string    `json:"kpi_id"`
	TenantID   string    `json:"tenant_id"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Source     string    `json:"source"`
	Estimated  bool      `json:"estimated"`
	Status     string    `json:"status"`
	MeasuredAt time.Time `json:"measured_at"`
	RecordedAt time.Time `json:"recorded_at"`
}

// ─── Risk Intelligence ───────────────────────────────────────────────────────

type RiskEvent struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	EventType         string                 `json:"event_type"`
	SourceSystem      string                 `json:"source_system"`
	SourceEventID     string                 `json:"source_event_id,omitempty"`
	RiskReference     string                 `json:"risk_reference,omitempty"` // authoritative register ref (StatGovernance)
	SeverityEstimate  int                    `json:"severity_estimate"`        // 0-100
	CorrelatedObjects []string               `json:"correlated_objects"`
	SignalID          string                 `json:"signal_id,omitempty"`
	Evidence          string                 `json:"evidence"`
	CorrelationID     string                 `json:"correlation_id"`
	Status            string                 `json:"status"`
	CreatedAt         time.Time              `json:"created_at"`
}

// ─── Governed AI ─────────────────────────────────────────────────────────────

// AIInput is the minimum-necessary-data contract sent to a reasoning
// provider. Sensitive information is never sent unless classification
// permits it (RESTRICTED/SENSITIVE blocked for external providers).
type AIInput struct {
	TenantID              string                   `json:"tenant_id"`
	Context               string                   `json:"context"`
	InstitutionalSnapshot map[string]interface{}   `json:"institutional_snapshot"`
	SupportingEvidence    []string                 `json:"supporting_evidence"`
	GraphRelationships    []GraphEdgeRef           `json:"graph_relationships,omitempty"`
	RelevantMetrics       []map[string]interface{} `json:"relevant_metrics,omitempty"`
	DataClassification    string                   `json:"data_classification"`
	CorrelationID         string                   `json:"correlation_id"`
}

// AIOutput is the mandatory structured output contract (directive §18).
type AIOutput struct {
	Recommendation     string   `json:"recommendation"`
	Confidence         float64  `json:"confidence"`
	ReasoningSummary   string   `json:"reasoning_summary"`
	SupportingEvidence []string `json:"supporting_evidence"`
	AffectedObjects    []string `json:"affected_objects"`
	RiskLevel          string   `json:"risk_level"`
	RecommendedActions []string `json:"recommended_actions"`
	Limitations        []string `json:"limitations"`
}