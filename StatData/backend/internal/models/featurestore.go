package models

import "time"

// FeatureView represents a curated set of features grouped by entity
type FeatureView struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	EntityName  string                 `json:"entity_name"` // e.g., citizen_id, facility_id, household_id
	Description string                 `json:"description"`
	TTLSeconds  int64                  `json:"ttl_seconds"`
	Features    []FeatureDefinition    `json:"features"`
	SourceQuery string                 `json:"source_query,omitempty"`
	OnlineStore bool                   `json:"online_store"`
	OfflineSink string                 `json:"offline_sink,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	TenantID    string                 `json:"tenant_id"`
	WorkspaceID string                 `json:"workspace_id,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// FeatureDefinition defines a single feature parameter
type FeatureDefinition struct {
	Name        string `json:"name"`
	DataType    string `json:"data_type"` // FLOAT, INTEGER, STRING, BOOLEAN, VECTOR
	Description string `json:"description,omitempty"`
}

// FeatureEntity represents an identifiable domain entity (e.g. Household, Hospital, CensusDistrict)
type FeatureEntity struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	JoinKey     string    `json:"join_key"`
	Description string    `json:"description"`
	TenantID    string    `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// FeatureRecord represents stored feature values for an entity instance
type FeatureRecord struct {
	EntityKey     string                 `json:"entity_key"`
	FeatureViewID string                 `json:"feature_view_id"`
	Values        map[string]interface{} `json:"values"`
	Timestamp     time.Time              `json:"timestamp"`
	TenantID      string                 `json:"tenant_id"`
}

// FeatureVector represents an online retrieval request and vector payload
type FeatureVector struct {
	EntityKey  string                 `json:"entity_key"`
	Features   map[string]interface{} `json:"features"`
	RetrievedAt time.Time             `json:"retrieved_at"`
}
