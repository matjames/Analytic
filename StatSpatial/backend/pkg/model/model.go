package model

import "time"

// ─── Foundation ────────────────────────────────────────────────────────────

type AdminUnit struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	Level     string    `json:"level"` // country | region | district | sub_county | parish
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Organization struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`  // government | ministry | agency | ngo | research | private
	Level        string    `json:"level"` // national | regional | district | community
	ParentID     *string   `json:"parent_id,omitempty"`
	AdminUnitID  *string   `json:"admin_unit_id,omitempty"`
	ContactEmail string    `json:"contact_email,omitempty"`
	ContactPhone string    `json:"contact_phone,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Classification struct {
	ID          string    `json:"id"`
	Scheme      string    `json:"scheme"` // geographic_code | isic | institution_type
	Code        string    `json:"code"`
	Label       string    `json:"label"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── GIS ──────────────────────────────────────────────────────────────────

type GeoLayer struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id,omitempty"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	GeometryType string    `json:"geometry_type"` // polygon | point | line
	Source       string    `json:"source,omitempty"`
	AdminUnitID  *string   `json:"admin_unit_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type GeoFeature struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	WorkspaceID string                 `json:"workspace_id,omitempty"`
	LayerID     string                 `json:"layer_id"`
	AdminUnitID *string                `json:"admin_unit_id,omitempty"`
	Name        string                 `json:"name"`
	Geometry    string                 `json:"geometry"` // GeoJSON geometry string
	Properties  map[string]interface{} `json:"properties,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type SpatialIndex struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	AdminUnitID string    `json:"admin_unit_id"`
	TargetType  string    `json:"target_type"` // facility | dataset | indicator | organization | project
	TargetID    string    `json:"target_id"`
	Lat         float64   `json:"lat,omitempty"`
	Lng         float64   `json:"lng,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── Federation ────────────────────────────────────────────────────────────

type FederatedNode struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id,omitempty"`
	WorkspaceID  string     `json:"workspace_id,omitempty"`
	Name         string     `json:"name"`
	NodeType     string     `json:"node_type"` // district | ministry | agency | regional_body | international
	URL          string     `json:"url,omitempty"`
	APIKeyRef    string     `json:"api_key_ref,omitempty"`
	Status       string     `json:"status"` // active | paused | pending | disconnected
	ContactEmail string     `json:"contact_email,omitempty"`
	AdminUnitID  *string    `json:"admin_unit_id,omitempty"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type DataSharingAgreement struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id,omitempty"`
	WorkspaceID  string     `json:"workspace_id,omitempty"`
	Title        string     `json:"title"`
	SourceNodeID string     `json:"source_node_id"`
	TargetNodeID string     `json:"target_node_id"`
	DataTypes    []string   `json:"data_types"`
	Frequency    string     `json:"frequency"` // realtime | daily | weekly | monthly | quarterly | yearly
	Status       string     `json:"status"`    // draft | active | suspended | expired
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type FederatedDataset struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id,omitempty"`
	WorkspaceID string     `json:"workspace_id,omitempty"`
	NodeID      string     `json:"node_id"`
	ExternalID  string     `json:"external_id,omitempty"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"` // synced | pending | failed
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	RecordCount int64      `json:"record_count,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type SyncLog struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id,omitempty"`
	WorkspaceID   string     `json:"workspace_id,omitempty"`
	NodeID        string     `json:"node_id"`
	DatasetID     string     `json:"dataset_id,omitempty"`
	Status        string     `json:"status"` // success | partial | failed
	RecordsIn     int64      `json:"records_in,omitempty"`
	RecordsFailed int64      `json:"records_failed,omitempty"`
	Message       string     `json:"message,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

// ─── Shared ───────────────────────────────────────────────────────────────

type NodeLink struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	SourceType  string    `json:"source_type"`
	SourceID    string    `json:"source_id"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	CreatedAt   time.Time `json:"created_at"`
}
