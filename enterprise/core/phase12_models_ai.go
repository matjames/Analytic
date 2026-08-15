package main

import "time"

// ReasoningProvider is the provider-agnostic interface. The platform must
// not become coupled to a single AI provider. Possible implementations:
// OpenAI, Google Gemini, private hosted model, local inference.
type ReasoningProvider interface {
	Name() string
	Generate(input AIInput) (*AIOutput, error)
}

// IntelligenceRecommendation is the lifecycle record. States:
// GENERATED -> PENDING_REVIEW -> AUTHORIZED|REJECTED -> EXECUTED|CANCELLED
type IntelligenceRecommendation struct {
	ID                   string     `json:"id"`
	TenantID             string     `json:"tenant_id"`
	Provider             string     `json:"provider"`
	Model                string     `json:"model"`
	RequestID            string     `json:"request_id"`
	CorrelationID        string     `json:"correlation_id"`
	Status               string     `json:"status"`
	Recommendation       string     `json:"recommendation"`
	Confidence           float64    `json:"confidence"`
	ReasoningSummary     string     `json:"reasoning_summary"`
	SupportingEvidence   []string   `json:"supporting_evidence"`
	AffectedObjects      []string   `json:"affected_objects"`
	RiskLevel            string     `json:"risk_level"`
	RecommendedActions   []string   `json:"recommended_actions"`
	Limitations          []string   `json:"limitations"`
	InputClassification  string     `json:"input_classification"`
	OutputClassification string     `json:"output_classification"`
	AILabel              string     `json:"ai_label"` // always AI_GENERATED
	GeneratedBy          string     `json:"generated_by"`
	ReviewedBy           string     `json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time `json:"reviewed_at,omitempty"`
	HumanDecision        string     `json:"human_decision,omitempty"`
	ExecutedAt           *time.Time `json:"executed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

// AIAuditEntry is the dedicated AI audit trail linked to the platform audit
// system via correlation_id / request_id / actor.
type AIAuditEntry struct {
	ID                   int64     `json:"id"`
	Provider             string    `json:"provider"`
	Model                string    `json:"model"`
	RequestID            string    `json:"request_id"`
	CorrelationID        string    `json:"correlation_id"`
	InputClassification  string    `json:"input_classification"`
	OutputClassification string    `json:"output_classification"`
	RecommendationID     string    `json:"recommendation_id,omitempty"`
	Recommendation       string    `json:"recommendation"`
	Confidence           float64   `json:"confidence"`
	Timestamp            time.Time `json:"timestamp"`
	Actor                string    `json:"actor"`
	TenantID             string    `json:"tenant_id"`
	AuthorizationStatus  string    `json:"authorization_status"`
	HumanDecision        string    `json:"human_decision,omitempty"`
}
