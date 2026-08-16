package models

import "time"

// DataClassification defines confidentiality tier
type DataClassification string

const (
	ClassificationPublic       DataClassification = "PUBLIC"
	ClassificationInternal     DataClassification = "INTERNAL"
	ClassificationConfidential DataClassification = "CONFIDENTIAL"
	ClassificationRestricted   DataClassification = "RESTRICTED"
)

// Dataset represents a governed dataset entity in the central data catalog
type Dataset struct {
	ID             string                 `json:"id"`
	URN            string                 `json:"urn"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Domain         string                 `json:"domain"` // e.g., health, economics, census, agriculture
	Classification DataClassification     `json:"classification"`
	OwnerTeam      string                 `json:"owner_team"`
	OwnerEmail     string                 `json:"owner_email"`
	Format         string                 `json:"format"` // PARQUET, CSV, JSON, DELTA, POSTGRES
	StorageURI     string                 `json:"storage_uri"`
	SchemaID       string                 `json:"schema_id,omitempty"`
	Version        string                 `json:"version"`
	QualityScore   float64                `json:"quality_score"` // 0.0 - 100.0
	RowCount       int64                  `json:"row_count"`
	SizeBytes      int64                  `json:"size_bytes"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	TenantID       string                 `json:"tenant_id"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// DataSource represents an external or internal source system (Postgres, S3, ODK, Kafka, etc.)
type DataSource struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	SourceType     string                 `json:"source_type"` // POSTGRESQL, MYSQL, S3, KAFKA, ODK, DHIS2, REST_API
	ConnectionURI  string                 `json:"connection_uri,omitempty"`
	AuthType       string                 `json:"auth_type"` // NONE, BASIC, API_KEY, OAUTH2, IAM
	Credentials    map[string]interface{} `json:"credentials,omitempty"`
	Status         string                 `json:"status"` // ACTIVE, INACTIVE, ERROR
	TenantID       string                 `json:"tenant_id"`
	LastTestedAt   *time.Time             `json:"last_tested_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// SchemaDefinition represents registered schemas for data contracts & validation
type SchemaDefinition struct {
	ID            string                 `json:"id"`
	Subject       string                 `json:"subject"` // e.g., national_health_surveys, cdc_demographics
	Version       int                    `json:"version"`
	SchemaType    string                 `json:"schema_type"` // JSON_SCHEMA, AVRO, PROTOBUF
	SchemaContent string                 `json:"schema_content"`
	Compatibility string                 `json:"compatibility"` // BACKWARD, FORWARD, FULL, NONE
	Description   string                 `json:"description,omitempty"`
	Fields        []SchemaField          `json:"fields,omitempty"`
	TenantID      string                 `json:"tenant_id"`
	CreatedBy     string                 `json:"created_by"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// SchemaField defines a structured field in a dataset schema
type SchemaField struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // STRING, INTEGER, FLOAT, BOOLEAN, TIMESTAMP, GEOJSON, ARRAY, OBJECT
	Nullable    bool   `json:"nullable"`
	Description string `json:"description,omitempty"`
	IsPII       bool   `json:"is_pii"`
}

// DataContract defines SLA and guarantees between data producers and consumers
type DataContract struct {
	ID             string                 `json:"id"`
	DatasetID      string                 `json:"dataset_id"`
	Title          string                 `json:"title"`
	ProducerTeam   string                 `json:"producer_team"`
	ConsumerTeam   string                 `json:"consumer_team"`
	Status         string                 `json:"status"` // DRAFT, ACTIVE, DEPRECATED, TERMINATED
	MaxLatencyMins int                    `json:"max_latency_mins"`
	MinQualityRate float64                `json:"min_quality_rate"`
	ExpectedVolume int64                  `json:"expected_volume"`
	SLAConfig      map[string]interface{} `json:"sla_config,omitempty"`
	TenantID       string                 `json:"tenant_id"`
	ValidFrom      time.Time              `json:"valid_from"`
	ValidUntil     time.Time              `json:"valid_until"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// DataQualityRule defines an automated validation assertion
type DataQualityRule struct {
	ID          string                 `json:"id"`
	DatasetID   string                 `json:"dataset_id"`
	RuleName    string                 `json:"rule_name"`
	RuleType    string                 `json:"rule_type"` // NOT_NULL, UNIQUE, RANGE, REGEX, FRESHNESS, SQL_ASSERTION
	TargetField string                 `json:"target_field,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Severity    string                 `json:"severity"` // ERROR, WARNING, INFO
	IsEnabled   bool                   `json:"is_enabled"`
	TenantID    string                 `json:"tenant_id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// DataQualityReport stores the evaluation outcome of data quality rules
type DataQualityReport struct {
	ID           string                 `json:"id"`
	DatasetID    string                 `json:"dataset_id"`
	PipelineRunID string                `json:"pipeline_run_id,omitempty"`
	Status       string                 `json:"status"` // PASSED, FAILED, WARNING
	QualityScore float64                `json:"quality_score"`
	TotalRules   int                    `json:"total_rules"`
	PassedRules  int                    `json:"passed_rules"`
	FailedRules  int                    `json:"failed_rules"`
	RuleResults  []RuleExecutionResult  `json:"rule_results"`
	EvaluatedAt  time.Time              `json:"evaluated_at"`
	TenantID     string                 `json:"tenant_id"`
}

// RuleExecutionResult represents a single rule test outcome
type RuleExecutionResult struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Passed      bool                   `json:"passed"`
	Severity    string                 `json:"severity"`
	TargetField string                 `json:"target_field,omitempty"`
	ActualValue interface{}            `json:"actual_value,omitempty"`
	Message     string                 `json:"message,omitempty"`
	SampleErrors []map[string]interface{} `json:"sample_errors,omitempty"`
}

// LineageNode represents an entity in the data lineage dependency graph
type LineageNode struct {
	ID          string                 `json:"id"`
	URN         string                 `json:"urn"`
	Type        string                 `json:"type"` // DATASET, PIPELINE, ML_MODEL, DASHBOARD, FEATURE_VIEW
	Name        string                 `json:"name"`
	Domain      string                 `json:"domain"`
	TenantID    string                 `json:"tenant_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LineageEdge represents a directional data flow relationship
type LineageEdge struct {
	ID             string                 `json:"id"`
	SourceNodeID   string                 `json:"source_node_id"`
	TargetNodeID   string                 `json:"target_node_id"`
	RelationType   string                 `json:"relation_type"` // READS_FROM, TRANSFORMS_TO, GENERATES, CONSUMES
	PipelineID     string                 `json:"pipeline_id,omitempty"`
	Transformation string                 `json:"transformation,omitempty"`
	TenantID       string                 `json:"tenant_id"`
	CreatedAt      time.Time              `json:"created_at"`
}

// LineageGraph aggregates nodes and edges for visual lineage graph rendering
type LineageGraph struct {
	RootID string        `json:"root_id"`
	Nodes  []LineageNode `json:"nodes"`
	Edges  []LineageEdge `json:"edges"`
}
