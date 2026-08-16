package models

import (
	"time"
)

// NodeType represents categories of federated participants
type NodeType string

const (
	NodeTypeNSSAgency        NodeType = "nss_agency"
	NodeTypeMinistry         NodeType = "ministry"
	NodeTypeDistrict         NodeType = "district"
	NodeTypeRegionalBloc     NodeType = "regional_bloc"
	NodeTypeInternationalOrg NodeType = "international_org"
	NodeTypeAcademicPartner  NodeType = "academic_partner"
)

// HealthStatus represents real-time operational state of a node
type HealthStatus string

const (
	HealthStatusHealthy     HealthStatus = "HEALTHY"
	HealthStatusDegraded    HealthStatus = "DEGRADED"
	HealthStatusUnreachable HealthStatus = "UNREACHABLE"
	HealthStatusMaintenance HealthStatus = "MAINTENANCE"
)

// TrustLevel defines sovereign security classification
type TrustLevel string

const (
	TrustLevelTier1Sovereign       TrustLevel = "TIER_1_SOVEREIGN"
	TrustLevelTier2DomesticAgency  TrustLevel = "TIER_2_DOMESTIC_AGENCY"
	TrustLevelTier3RegionalPartner TrustLevel = "TIER_3_REGIONAL_PARTNER"
	TrustLevelTier4Unrestricted    TrustLevel = "TIER_4_UNRESTRICTED"
)

// FederatedNode represents a registered node in NSS or global diplomacy network
type FederatedNode struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Code         string       `json:"code"`
	NodeType     NodeType     `json:"node_type"`
	Jurisdiction string       `json:"jurisdiction"`
	EndpointURL  string       `json:"endpoint_url"`
	HealthStatus HealthStatus `json:"health_status"`
	TrustLevel   TrustLevel   `json:"trust_level"`
	PublicKey    string       `json:"public_key,omitempty"`
	Protocols    []string     `json:"protocols"`
	Capabilities []string     `json:"capabilities"`
	TenantID     string       `json:"tenant_id"`
	ContactEmail string       `json:"contact_email,omitempty"`
	LastHeartbeat *time.Time  `json:"last_heartbeat,omitempty"`
	LatencyMs    int          `json:"latency_ms"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// DSAStatus defines the legal lifecycle of a Data Sharing Agreement
type DSAStatus string

const (
	DSAStatusDraft        DSAStatus = "DRAFT"
	DSAStatusUnderReview  DSAStatus = "UNDER_REVIEW"
	DSAStatusActive       DSAStatus = "ACTIVE"
	DSAStatusSuspended    DSAStatus = "SUSPENDED"
	DSAStatusExpired      DSAStatus = "EXPIRED"
	DSAStatusRevoked      DSAStatus = "REVOKED"
)

// DataSharingAgreement models legal, bilateral data governance boundaries
type DataSharingAgreement struct {
	ID                    string     `json:"id"`
	DSANumber             string     `json:"dsa_number"`
	Title                 string     `json:"title"`
	ProviderNodeID        string     `json:"provider_node_id"`
	ConsumerNodeID        string     `json:"consumer_node_id"`
	Status                DSAStatus  `json:"status"`
	AccessTier            string     `json:"access_tier"`
	PermittedDomains      []string   `json:"permitted_domains"`
	ClassificationAllowed string     `json:"classification_allowed"`
	RequiresApproval      bool       `json:"requires_approval"`
	Purpose               string     `json:"purpose"`
	ValidFrom             time.Time  `json:"valid_from"`
	ValidUntil            time.Time  `json:"valid_until"`
	RateLimitPerMin       int        `json:"rate_limit_per_min"`
	DailyQuota            int        `json:"daily_quota"`
	CurrentDailyUsage     int        `json:"current_daily_usage"`
	GovernanceApprovedBy  string     `json:"governance_approved_by,omitempty"`
	GovernanceApprovedAt  *time.Time `json:"governance_approved_at,omitempty"`
	TenantID              string     `json:"tenant_id"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// NationalIndicator represents a standard official statistic or SDG metric
type NationalIndicator struct {
	ID                       string     `json:"id"`
	Code                     string     `json:"code"`
	Title                    string     `json:"title"`
	Domain                   string     `json:"domain"`
	SDMXDimension            string     `json:"sdmx_dimension,omitempty"`
	LeadAgencyID             string     `json:"lead_agency_id,omitempty"`
	LeadAgencyName           string     `json:"lead_agency_name,omitempty"`
	CalculationMethod        string     `json:"calculation_method,omitempty"`
	Frequency                string     `json:"frequency"`
	TargetValue              *float64   `json:"target_value,omitempty"`
	CurrentValue             *float64   `json:"current_value,omitempty"`
	BaselineValue            *float64   `json:"baseline_value,omitempty"`
	BaselineYear             *int       `json:"baseline_year,omitempty"`
	UnitOfMeasure            string     `json:"unit_of_measure,omitempty"`
	Tier                     string     `json:"tier"`
	DisaggregationDimensions []string   `json:"disaggregation_dimensions"`
	IsOfficialStatistic      bool       `json:"is_official_statistic"`
	CalendarReleaseDate      *time.Time `json:"calendar_release_date,omitempty"`
	TenantID                 string     `json:"tenant_id"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// QueryStatus tracks distributed execution state
type QueryStatus string

const (
	QueryStatusPending        QueryStatus = "PENDING"
	QueryStatusDispatched     QueryStatus = "DISPATCHED"
	QueryStatusPartialSuccess QueryStatus = "PARTIAL_SUCCESS"
	QueryStatusCompleted      QueryStatus = "COMPLETED"
	QueryStatusFailed         QueryStatus = "FAILED"
	QueryStatusTimedOut       QueryStatus = "TIMED_OUT"
)

// DistributedQueryRequest encapsulates an incoming cross-agency query
type DistributedQueryRequest struct {
	QueryName         string                 `json:"query_name"`
	TargetNodeIDs     []string               `json:"target_node_ids"`
	TargetDomain      string                 `json:"target_domain"`
	FilterCriteria    map[string]interface{} `json:"filter_criteria"`
	Aggregations      []string               `json:"aggregations,omitempty"`
	TimeoutSeconds    int                    `json:"timeout_seconds,omitempty"`
	MaxRecordsPerNode int                    `json:"max_records_per_node,omitempty"`
}

// DistributedQueryRecord represents query state ledger
type DistributedQueryRecord struct {
	ID                   string                 `json:"id"`
	QueryName            string                 `json:"query_name"`
	InitiatorUserID      string                 `json:"initiator_user_id"`
	InitiatorTenantID    string                 `json:"initiator_tenant_id"`
	TargetNodes          []string               `json:"target_nodes"`
	QuerySyntax          map[string]interface{} `json:"query_syntax"`
	ExecutionStrategy    string                 `json:"execution_strategy"`
	Status               QueryStatus            `json:"status"`
	DispatchTimestamp    time.Time              `json:"dispatch_timestamp"`
	CompletedTimestamp   *time.Time             `json:"completed_timestamp,omitempty"`
	TotalRecordsRetrieved int                   `json:"total_records_retrieved"`
	ExecutionTimeMs      int                    `json:"execution_time_ms"`
	NodeResponses        map[string]interface{} `json:"node_responses"`
	ErrorSummary         string                 `json:"error_summary,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
}

// MetadataVocabulary encapsulates cross-agency semantic harmonization
type MetadataVocabulary struct {
	ID                       string                 `json:"id"`
	VocabularyName           string                 `json:"vocabulary_name"`
	StandardFramework        string                 `json:"standard_framework"`
	SourceAgency             string                 `json:"source_agency"`
	TargetCanonicalConcept   string                 `json:"target_canonical_concept"`
	SourceConceptTerm        string                 `json:"source_concept_term"`
	MappingRules             map[string]interface{} `json:"mapping_rules"`
	TransformationExpression string                 `json:"transformation_expression,omitempty"`
	Status                   string                 `json:"status"`
	TenantID                 string                 `json:"tenant_id"`
	CreatedAt                time.Time              `json:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at"`
}

// DiplomaticTreaty represents international evidence exchange frameworks
type DiplomaticTreaty struct {
	ID                       string                 `json:"id"`
	TreatyCode               string                 `json:"treaty_code"`
	Title                    string                 `json:"title"`
	PartnerStates            []string               `json:"partner_states"`
	Jurisdiction             string                 `json:"jurisdiction"`
	FrameworkType            string                 `json:"framework_type"`
	Status                   string                 `json:"status"`
	RatificationDate         *time.Time             `json:"ratification_date,omitempty"`
	ExpiryDate               *time.Time             `json:"expiry_date,omitempty"`
	GoverningBody            string                 `json:"governing_body"`
	ComplianceRules          []map[string]interface{}`json:"compliance_rules"`
	DataLocalizationRequired bool                   `json:"data_localization_required"`
	EncryptionStandard       string                 `json:"encryption_standard"`
	TenantID                 string                 `json:"tenant_id"`
	CreatedAt                time.Time              `json:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at"`
}

// InternationalReport represents multilateral submission records (SDG, AU, EAC, UN)
type InternationalReport struct {
	ID                    string                 `json:"id"`
	ReportTitle           string                 `json:"report_title"`
	DestinationBody       string                 `json:"destination_body"`
	ReportingPeriod       string                 `json:"reporting_period"`
	Status                string                 `json:"status"`
	SubmissionHash        string                 `json:"submission_hash,omitempty"`
	TransferredIndicators []map[string]interface{}`json:"transferred_indicators"`
	CompliancePassed      bool                   `json:"compliance_passed"`
	ComplianceNotes       string                 `json:"compliance_notes,omitempty"`
	SubmittedBy           string                 `json:"submitted_by,omitempty"`
	SubmittedAt           *time.Time             `json:"submitted_at,omitempty"`
	AcknowledgementReceipt map[string]interface{} `json:"acknowledgement_receipt,omitempty"`
	TenantID              string                 `json:"tenant_id"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// FederatedSearchResult represents a cross-node discovered evidence item
type FederatedSearchResult struct {
	NodeID             string    `json:"node_id"`
	NodeName           string    `json:"node_name"`
	ResourceType       string    `json:"resource_type"`
	RemoteResourceID   string    `json:"remote_resource_id"`
	Title              string    `json:"title"`
	Abstract           string    `json:"abstract"`
	Keywords           []string  `json:"keywords"`
	Classification     string    `json:"classification"`
	TemporalCoverage   string    `json:"temporal_coverage,omitempty"`
	SpatialCoverage    string    `json:"spatial_coverage,omitempty"`
	DirectAccessURL    string    `json:"direct_access_url,omitempty"`
	Score              float64   `json:"score"`
}

// ComplianceAuditLog represents immutable sovereign transboundary evaluation logs
type ComplianceAuditLog struct {
	ID                 string                 `json:"id"`
	EventTimestamp     time.Time              `json:"event_timestamp"`
	ActorUserID        string                 `json:"actor_user_id"`
	ActorTenantID      string                 `json:"actor_tenant_id"`
	Action             string                 `json:"action"`
	SourceJurisdiction string                 `json:"source_jurisdiction"`
	TargetJurisdiction string                 `json:"target_jurisdiction"`
	ResourceType       string                 `json:"resource_type"`
	ResourceID         string                 `json:"resource_id"`
	Decision           string                 `json:"decision"`
	AppliedRules       []string               `json:"applied_rules"`
	RedactedFields     []string               `json:"redacted_fields"`
	PolicyHash         string                 `json:"policy_hash"`
	Reason             string                 `json:"reason,omitempty"`
}

// ObjectLink mirrors the universal StatGate cross-entity contract
type ObjectLink struct {
	ID           int       `json:"id,omitempty"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	Relationship string    `json:"relationship"`
	TenantID     string    `json:"tenant_id"`
	CreatedBy    string    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
