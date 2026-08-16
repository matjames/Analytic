package models

import (
	"time"
)

// ──────────────────────────────────────────────────────────────────────
// IoT & EDGE COMPUTING MODELS (P27)
// ──────────────────────────────────────────────────────────────────────

type GatewayStatus string

const (
	GatewayStatusOnline      GatewayStatus = "ONLINE"
	GatewayStatusOffline     GatewayStatus = "OFFLINE"
	GatewayStatusDegraded    GatewayStatus = "DEGRADED"
	GatewayStatusMaintenance GatewayStatus = "MAINTENANCE"
)

type IoTGateway struct {
	ID              string                 `json:"id" db:"id"`
	Name            string                 `json:"name" db:"name"`
	GatewayCode     string                 `json:"gateway_code" db:"gateway_code"`
	IPAddress       string                 `json:"ip_address,omitempty" db:"ip_address"`
	MACAddress      string                 `json:"mac_address,omitempty" db:"mac_address"`
	FirmwareVersion string                 `json:"firmware_version" db:"firmware_version"`
	Status          GatewayStatus          `json:"status" db:"status"`
	Latitude        *float64               `json:"latitude,omitempty" db:"latitude"`
	Longitude       *float64               `json:"longitude,omitempty" db:"longitude"`
	LocationName    string                 `json:"location_name,omitempty" db:"location_name"`
	TenantID        string                 `json:"tenant_id" db:"tenant_id"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	LastHeartbeat   *time.Time             `json:"last_heartbeat,omitempty" db:"last_heartbeat"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

type DeviceStatus string

const (
	DeviceStatusRegistered    DeviceStatus = "REGISTERED"
	DeviceStatusActive        DeviceStatus = "ACTIVE"
	DeviceStatusInactive      DeviceStatus = "INACTIVE"
	DeviceStatusDecommissioned DeviceStatus = "DECOMMISSIONED"
)

type IoTDevice struct {
	ID               string                 `json:"id" db:"id"`
	DeviceUID        string                 `json:"device_uid" db:"device_uid"`
	Name             string                 `json:"name" db:"name"`
	DeviceType       string                 `json:"device_type" db:"device_type"`
	Protocol         string                 `json:"protocol" db:"protocol"`
	GatewayID        *string                `json:"gateway_id,omitempty" db:"gateway_id"`
	AuthTokenHash    string                 `json:"-" db:"auth_token_hash"`
	FirmwareVersion  string                 `json:"firmware_version" db:"firmware_version"`
	Status           DeviceStatus           `json:"status" db:"status"`
	BatteryLevel     *float64               `json:"battery_level,omitempty" db:"battery_level"`
	SignalStrengthDBM *int                  `json:"signal_strength_dbm,omitempty" db:"signal_strength_dbm"`
	Latitude         *float64               `json:"latitude,omitempty" db:"latitude"`
	Longitude        *float64               `json:"longitude,omitempty" db:"longitude"`
	Altitude         *float64               `json:"altitude,omitempty" db:"altitude"`
	TenantID         string                 `json:"tenant_id" db:"tenant_id"`
	OrgID            string                 `json:"org_id,omitempty" db:"org_id"`
	ConfigPayload    map[string]interface{} `json:"config_payload,omitempty" db:"config_payload"`
	Tags             []string               `json:"tags,omitempty" db:"tags"`
	LastSeenAt       *time.Time             `json:"last_seen_at,omitempty" db:"last_seen_at"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

type SensorRegistryItem struct {
	ID                string   `json:"id" db:"id"`
	DeviceID          string   `json:"device_id" db:"device_id"`
	SensorCode        string   `json:"sensor_code" db:"sensor_code"`
	MetricName        string   `json:"metric_name" db:"metric_name"`
	UnitOfMeasure     string   `json:"unit_of_measure" db:"unit_of_measure"`
	MinThreshold      *float64 `json:"min_threshold,omitempty" db:"min_threshold"`
	MaxThreshold      *float64 `json:"max_threshold,omitempty" db:"max_threshold"`
	CalibrationFactor float64  `json:"calibration_factor" db:"calibration_factor"`
	CalibrationOffset float64  `json:"calibration_offset" db:"calibration_offset"`
	Status            string   `json:"status" db:"status"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

type TelemetryRecord struct {
	ID           string                 `json:"id" db:"id"`
	DeviceID     string                 `json:"device_id" db:"device_id"`
	SensorCode   string                 `json:"sensor_code" db:"sensor_code"`
	MetricName   string                 `json:"metric_name" db:"metric_name"`
	Value        float64                `json:"value" db:"value"`
	RawPayload   map[string]interface{} `json:"raw_payload,omitempty" db:"raw_payload"`
	QualityScore float64                `json:"quality_score" db:"quality_score"`
	IsAnomaly    bool                   `json:"is_anomaly" db:"is_anomaly"`
	TenantID     string                 `json:"tenant_id" db:"tenant_id"`
	RecordedAt   time.Time              `json:"recorded_at" db:"recorded_at"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

type TelemetryBatchIngestRequest struct {
	DeviceUID  string            `json:"device_uid"`
	AuthToken  string            `json:"auth_token,omitempty"`
	Readings   []TelemetryRecord `json:"readings"`
	GatewayID  string            `json:"gateway_id,omitempty"`
	RecordedAt *time.Time        `json:"recorded_at,omitempty"`
}

type TelemetryAggregate struct {
	MetricName string    `json:"metric_name"`
	Count      int64     `json:"count"`
	Min        float64   `json:"min"`
	Max        float64   `json:"max"`
	Avg        float64   `json:"avg"`
	FirstTime  time.Time `json:"first_time"`
	LastTime   time.Time `json:"last_time"`
}

type EdgeConfiguration struct {
	ID             string                 `json:"id" db:"id"`
	DeviceID       string                 `json:"device_id" db:"device_id"`
	Version        int                    `json:"version" db:"version"`
	DesiredConfig  map[string]interface{} `json:"desired_config" db:"desired_config"`
	ReportedConfig map[string]interface{} `json:"reported_config,omitempty" db:"reported_config"`
	SyncStatus     string                 `json:"sync_status" db:"sync_status"` // PENDING, APPLIED, FAILED
	AppliedAt      *time.Time             `json:"applied_at,omitempty" db:"applied_at"`
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

type FirmwareRelease struct {
	ID             string    `json:"id" db:"id"`
	DeviceType     string    `json:"device_type" db:"device_type"`
	Version        string    `json:"version" db:"version"`
	BinaryURL      string    `json:"binary_url" db:"binary_url"`
	ChecksumSHA256 string    `json:"checksum_sha256" db:"checksum_sha256"`
	Changelog      string    `json:"changelog,omitempty" db:"changelog"`
	IsCritical     bool      `json:"is_critical" db:"is_critical"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	ReleasedAt     time.Time `json:"released_at" db:"released_at"`
}

type IoTAlert struct {
	ID             string     `json:"id" db:"id"`
	DeviceID       string     `json:"device_id" db:"device_id"`
	SensorCode     string     `json:"sensor_code,omitempty" db:"sensor_code"`
	AlertType      string     `json:"alert_type" db:"alert_type"`
	Severity       string     `json:"severity" db:"severity"` // INFO, WARNING, CRITICAL, EMERGENCY
	Message        string     `json:"message" db:"message"`
	Value          *float64   `json:"value,omitempty" db:"value"`
	Threshold      *float64   `json:"threshold,omitempty" db:"threshold"`
	Status         string     `json:"status" db:"status"` // ACTIVE, ACKNOWLEDGED, RESOLVED
	AcknowledgedBy *string    `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	TenantID       string     `json:"tenant_id" db:"tenant_id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// ──────────────────────────────────────────────────────────────────────
// MOBILE FIELD OPERATIONS & OFFLINE MODELS (P43)
// ──────────────────────────────────────────────────────────────────────

type FieldWorker struct {
	ID               string    `json:"id" db:"id"`
	UserID           string    `json:"user_id" db:"user_id"`
	FullName         string    `json:"full_name" db:"full_name"`
	PhoneNumber      string    `json:"phone_number,omitempty" db:"phone_number"`
	TeamName         string    `json:"team_name,omitempty" db:"team_name"`
	Role             string    `json:"role" db:"role"` // ENUMERATOR, SUPERVISOR, INSPECTOR, TECHNICIAN
	AssignedDistrict string    `json:"assigned_district,omitempty" db:"assigned_district"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	OrgID            string    `json:"org_id,omitempty" db:"org_id"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type MobileDevice struct {
	ID               string     `json:"id" db:"id"`
	DeviceUUID       string     `json:"device_uuid" db:"device_uuid"`
	AssignedWorkerID *string    `json:"assigned_worker_id,omitempty" db:"assigned_worker_id"`
	Platform         string     `json:"platform" db:"platform"` // ANDROID, IOS, PWA
	AppVersion       string     `json:"app_version" db:"app_version"`
	OSVersion        string     `json:"os_version,omitempty" db:"os_version"`
	BatteryLevel     *float64   `json:"battery_level,omitempty" db:"battery_level"`
	StorageFreeMB    *int64     `json:"storage_free_mb,omitempty" db:"storage_free_mb"`
	LastSyncAt       *time.Time `json:"last_sync_at,omitempty" db:"last_sync_at"`
	TenantID         string     `json:"tenant_id" db:"tenant_id"`
	Status           string     `json:"status" db:"status"` // ACTIVE, LOCKED, WIPED
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type FieldVisit struct {
	ID               string     `json:"id" db:"id"`
	WorkerID         string     `json:"worker_id" db:"worker_id"`
	Title            string     `json:"title" db:"title"`
	TargetEntityType string     `json:"target_entity_type" db:"target_entity_type"`
	TargetEntityID   string     `json:"target_entity_id" db:"target_entity_id"`
	ScheduledStart   time.Time  `json:"scheduled_start" db:"scheduled_start"`
	ScheduledEnd     time.Time  `json:"scheduled_end" db:"scheduled_end"`
	ActualStart      *time.Time `json:"actual_start,omitempty" db:"actual_start"`
	ActualEnd        *time.Time `json:"actual_end,omitempty" db:"actual_end"`
	Status           string     `json:"status" db:"status"` // SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED
	Latitude         *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude        *float64   `json:"longitude,omitempty" db:"longitude"`
	Notes            string     `json:"notes,omitempty" db:"notes"`
	TenantID         string     `json:"tenant_id" db:"tenant_id"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type VisitTask struct {
	ID            string                 `json:"id" db:"id"`
	VisitID       string                 `json:"visit_id" db:"visit_id"`
	Title         string                 `json:"title" db:"title"`
	TaskType      string                 `json:"task_type" db:"task_type"` // FORM_COLLECT, SENSOR_INSPECT, CALIBRATE, PHOTO_AUDIT
	FormID        *string                `json:"form_id,omitempty" db:"form_id"`
	Status        string                 `json:"status" db:"status"` // PENDING, COMPLETED, SKIPPED
	ResultSummary map[string]interface{} `json:"result_summary,omitempty" db:"result_summary"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

type MobileFormDefinition struct {
	ID               string                 `json:"id" db:"id"`
	FormCode         string                 `json:"form_code" db:"form_code"`
	Title            string                 `json:"title" db:"title"`
	Version          int                    `json:"version" db:"version"`
	SchemaDefinition map[string]interface{} `json:"schema_definition" db:"schema_definition"`
	IsActive         bool                   `json:"is_active" db:"is_active"`
	TenantID         string                 `json:"tenant_id" db:"tenant_id"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

type MobileSubmission struct {
	ID                 string                 `json:"id" db:"id"`
	ClientSubmissionID string                 `json:"client_submission_id" db:"client_submission_id"`
	FormID             string                 `json:"form_id" db:"form_id"`
	WorkerID           string                 `json:"worker_id" db:"worker_id"`
	VisitID            *string                `json:"visit_id,omitempty" db:"visit_id"`
	DataPayload        map[string]interface{} `json:"data_payload" db:"data_payload"`
	GeoPoint           map[string]interface{} `json:"geo_point,omitempty" db:"geo_point"`
	CollectedAt        time.Time              `json:"collected_at" db:"collected_at"`
	SyncedAt           time.Time              `json:"synced_at" db:"synced_at"`
	SyncStatus         string                 `json:"sync_status" db:"sync_status"`
	TenantID           string                 `json:"tenant_id" db:"tenant_id"`
	Version            int                    `json:"version" db:"version"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
}

type GPSBreadcrumb struct {
	ID             string    `json:"id" db:"id"`
	WorkerID       string    `json:"worker_id" db:"worker_id"`
	DeviceUUID     string    `json:"device_uuid" db:"device_uuid"`
	Latitude       float64   `json:"latitude" db:"latitude"`
	Longitude      float64   `json:"longitude" db:"longitude"`
	Altitude       *float64  `json:"altitude,omitempty" db:"altitude"`
	AccuracyMeters *float64  `json:"accuracy_meters,omitempty" db:"accuracy_meters"`
	SpeedMPS       *float64  `json:"speed_mps,omitempty" db:"speed_mps"`
	HeadingDeg     *float64  `json:"heading_deg,omitempty" db:"heading_deg"`
	BatteryLevel   *float64  `json:"battery_level,omitempty" db:"battery_level"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	RecordedAt     time.Time `json:"recorded_at" db:"recorded_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type GeofenceZone struct {
	ID             string                 `json:"id" db:"id"`
	Name           string                 `json:"name" db:"name"`
	ZoneType       string                 `json:"zone_type" db:"zone_type"`
	CenterLat      float64                `json:"center_lat" db:"center_lat"`
	CenterLng      float64                `json:"center_lng" db:"center_lng"`
	RadiusMeters   float64                `json:"radius_meters" db:"radius_meters"`
	PolygonGeoJSON map[string]interface{} `json:"polygon_geojson,omitempty" db:"polygon_geojson"`
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

type SyncPushRequest struct {
	DeviceUUID   string             `json:"device_uuid"`
	WorkerID     string             `json:"worker_id"`
	Submissions  []MobileSubmission `json:"submissions"`
	Breadcrumbs  []GPSBreadcrumb    `json:"breadcrumbs"`
	UpdatedTasks []VisitTask        `json:"updated_tasks"`
	ClientTime   time.Time          `json:"client_time"`
}

type SyncPushResponse struct {
	TransactionID      string   `json:"transaction_id"`
	CommittedCount     int      `json:"committed_count"`
	ConflictsCount     int      `json:"conflicts_count"`
	ServerSyncTime     time.Time `json:"server_sync_time"`
	ConflictIDs        []string `json:"conflict_ids,omitempty"`
	AcknowledgedClientIDs []string `json:"acknowledged_client_ids"`
}

type SyncPullRequest struct {
	DeviceUUID     string    `json:"device_uuid"`
	WorkerID       string    `json:"worker_id"`
	LastSyncedTime time.Time `json:"last_synced_time"`
}

type SyncPullResponse struct {
	Forms          []MobileFormDefinition `json:"forms"`
	AssignedVisits []FieldVisit           `json:"assigned_visits"`
	Tasks          []VisitTask            `json:"tasks"`
	ServerTime     time.Time              `json:"server_time"`
}

type ConflictResolutionStrategy string

const (
	StrategyLastWriteWins   ConflictResolutionStrategy = "LAST_WRITE_WINS"
	StrategyFieldLevelMerge ConflictResolutionStrategy = "FIELD_LEVEL_MERGE"
	StrategyManualReview    ConflictResolutionStrategy = "MANUAL_SUPERVISOR"
)

type ConflictRecord struct {
	ID                 string                     `json:"id" db:"id"`
	EntityType         string                     `json:"entity_type" db:"entity_type"`
	EntityID           string                     `json:"entity_id" db:"entity_id"`
	ClientVersion      int                        `json:"client_version" db:"client_version"`
	ServerVersion      int                        `json:"server_version" db:"server_version"`
	ClientPayload      map[string]interface{}     `json:"client_payload" db:"client_payload"`
	ServerPayload      map[string]interface{}     `json:"server_payload" db:"server_payload"`
	ResolutionStrategy ConflictResolutionStrategy `json:"resolution_strategy" db:"resolution_strategy"`
	ResolvedPayload    map[string]interface{}     `json:"resolved_payload,omitempty" db:"resolved_payload"`
	Status             string                     `json:"status" db:"status"` // PENDING_MANUAL, RESOLVED_AUTO, RESOLVED_MANUAL
	ResolvedBy         *string                    `json:"resolved_by,omitempty" db:"resolved_by"`
	TenantID           string                     `json:"tenant_id" db:"tenant_id"`
	CreatedAt          time.Time                  `json:"created_at" db:"created_at"`
	ResolvedAt         *time.Time                 `json:"resolved_at,omitempty" db:"resolved_at"`
}

type ObjectLink struct {
	ID           int       `json:"id" db:"id"`
	SourceType   string    `json:"source_type" db:"source_type"`
	SourceID     string    `json:"source_id" db:"source_id"`
	TargetType   string    `json:"target_type" db:"target_type"`
	TargetID     string    `json:"target_id" db:"target_id"`
	Relationship string    `json:"relationship" db:"relationship"`
	TenantID     string    `json:"tenant_id" db:"tenant_id"`
	CreatedBy    string    `json:"created_by,omitempty" db:"created_by"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
