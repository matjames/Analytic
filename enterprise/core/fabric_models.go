package main

// Phase IX: Institutional Data, Knowledge & Interoperability Fabric models.
// Provides canonical object identity, multi-relational graphs, governed
// knowledge hierarchy, data catalogue, data dictionary, and application registry.

type CanonicalObject struct {
	CanonicalID        string                 `json:"canonical_id"`
	ObjectID           string                 `json:"object_id"`
	ObjectType         string                 `json:"object_type"`
	SourceApplication  string                 `json:"source_application"`
	TenantID           string                 `json:"tenant_id"`
	OrganizationID     string                 `json:"organization_id"`
	ProjectID          string                 `json:"project_id,omitempty"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description,omitempty"`
	CreatedBy          string                 `json:"created_by"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
	Version            int                    `json:"version"`
	Status             string                 `json:"status"`
	Classification     string                 `json:"classification"` // verified | derived | ai_recommendation | decision | unverified
	Sensitivity        string                 `json:"sensitivity"`    // public | internal | official | confidential | restricted
	CanonicalURL       string                 `json:"canonical_url"`
	CorrelationID      string                 `json:"correlation_id,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	RelationshipsCount int                    `json:"relationships_count"`
}

type FabricRelationship struct {
	ID              string   `json:"id"`
	FromType        string   `json:"from_type"`
	FromID          string   `json:"from_id"`
	ToType          string   `json:"to_type"`
	ToID            string   `json:"to_id"`
	RelationType    string   `json:"relation_type"` // belongs_to, derived_from, depends_on, documents, resolves, triggered_by, monitors, governed_by, assigned_to, located_at, verified_by, informs
	Confidence      float64  `json:"confidence"`    // 0.00 to 1.00
	Source          string   `json:"source"`
	Provenance      string   `json:"provenance,omitempty"`
	LifecycleStatus string   `json:"lifecycle_status"` // active | deprecated | revoked
	CreatedBy       string   `json:"created_by"`
	CreatedAt       string   `json:"created_at"`
	AuditHistory    []string `json:"audit_history,omitempty"`
	TenantID        string   `json:"tenant_id"`
}

type GovernedKnowledgeItem struct {
	ID               string                 `json:"id"`
	Title            string                 `json:"title"`
	Summary          string                 `json:"summary"`
	Content          string                 `json:"content"`
	Category         string                 `json:"category"`
	Classification   string                 `json:"classification"` // verified | derived | ai_recommendation | decision | unverified
	Author           string                 `json:"author"`
	Reviewer         string                 `json:"reviewer,omitempty"`
	ApprovalStatus   string                 `json:"approval_status"` // draft | review | approved | published | superseded | archived
	Version          int                    `json:"version"`
	EffectiveDate    string                 `json:"effective_date"`
	ExpiryDate       string                 `json:"expiry_date,omitempty"`
	SupersededBy     string                 `json:"superseded_by,omitempty"`
	SourceReferences []string               `json:"source_references,omitempty"`
	ChangeLog        []KnowledgeChangeEntry `json:"change_log,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	TenantID         string                 `json:"tenant_id"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

type KnowledgeChangeEntry struct {
	Version   int    `json:"version"`
	Timestamp string `json:"timestamp"`
	Author    string `json:"author"`
	Summary   string `json:"summary"`
}

type CatalogueDataset struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	Owner              string                 `json:"owner"`
	SourceApplication  string                 `json:"source_application"`
	OrganizationID     string                 `json:"organization_id"`
	DataDomain         string                 `json:"data_domain"`
	UpdateFrequency    string                 `json:"update_frequency"`
	LastRefresh        string                 `json:"last_refresh"`
	RecordCount        int64                  `json:"record_count"`
	QualityScore       float64                `json:"quality_score"`
	Completeness       float64                `json:"completeness"`
	Coverage           float64                `json:"coverage"`
	Sensitivity        string                 `json:"sensitivity"`
	GeographicCoverage string                 `json:"geographic_coverage"`
	TemporalCoverage   string                 `json:"temporal_coverage"`
	SchemaDefinition   []CatalogueVariable    `json:"schema_definition,omitempty"`
	VariablesCount     int                    `json:"variables_count"`
	LineageSource      string                 `json:"lineage_source,omitempty"`
	FreshnessStatus    string                 `json:"freshness_status"` // live | recent | delayed | stale | unavailable
	TenantID           string                 `json:"tenant_id"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

type CatalogueVariable struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Unit        string   `json:"unit,omitempty"`
	AllowedVals []string `json:"allowed_values,omitempty"`
}

type DictionaryVariable struct {
	ID                 string   `json:"id"`
	CanonicalName      string   `json:"canonical_name"`
	Aliases            []string `json:"aliases"`
	Definition         string   `json:"definition"`
	DataType           string   `json:"data_type"`
	Unit               string   `json:"unit,omitempty"`
	PermissibleValues  []string `json:"permissible_values,omitempty"`
	SourceApplications []string `json:"source_applications"`
	Owner              string   `json:"owner"`
	RelatedIndicators  []string `json:"related_indicators,omitempty"`
	RelatedDatasets    []string `json:"related_datasets,omitempty"`
	Version            int      `json:"version"`
	Status             string   `json:"status"`
	TenantID           string   `json:"tenant_id"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type SemanticConceptMapping struct {
	ID                string  `json:"id"`
	SourceTerm        string  `json:"source_term"`
	CanonicalConcept  string  `json:"canonical_concept"`
	SourceApplication string  `json:"source_application"`
	StandardCode      string  `json:"standard_code,omitempty"`
	Confidence        float64 `json:"confidence"`
	ApprovedBy        string  `json:"approved_by,omitempty"`
	TenantID          string  `json:"tenant_id"`
	CreatedAt         string  `json:"created_at"`
}

type RegisteredApplication struct {
	ApplicationID    string   `json:"application_id"`
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Owner            string   `json:"owner"`
	APIURL           string   `json:"api_url"`
	UIURL            string   `json:"ui_url"`
	HealthEndpoint   string   `json:"health_endpoint"`
	Capabilities     []string `json:"capabilities"`
	SupportedObjects []string `json:"supported_objects"`
	SupportedEvents  []string `json:"supported_events"`
	AuthMethod       string   `json:"auth_method"`
	Status           string   `json:"status"` // active | maintenance | offline
	LastHeartbeat    string   `json:"last_heartbeat"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type InstitutionalDecisionMemory struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Context            string   `json:"context"`
	ProblemStatement   string   `json:"problem_statement"`
	EvidenceSources    []string `json:"evidence_sources"`
	DecidedBy          string   `json:"decided_by"`
	DecisionTimestamp  string   `json:"decision_timestamp"`
	AlternativesTested []string `json:"alternatives_tested,omitempty"`
	ActionTasks        []string `json:"action_tasks,omitempty"`
	ExpectedOutcome    string   `json:"expected_outcome"`
	ActualOutcome      string   `json:"actual_outcome,omitempty"`
	OutcomeStatus      string   `json:"outcome_status"` // monitoring | achieved | partially_achieved | deviated | resolved
	VerifiedAt         string   `json:"verified_at,omitempty"`
	TenantID           string   `json:"tenant_id"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

type FabricLineageStep struct {
	Stage       string `json:"stage"` // source -> collection -> submission -> dataset -> transformation -> indicator -> report -> decision -> task -> outcome
	Label       string `json:"label"`
	SourceApp   string `json:"source_app"`
	EntityID    string `json:"entity_id"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
	Status      string `json:"status"`
}
