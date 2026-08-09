package main

import "time"

// ─── Phase IV: Enterprise Data Intelligence Models ─────────────────
// This file defines the core data structures for the StatGate
// Enterprise Data Layer, KPI Engine, Alert Engine, Data Quality,
// Anomaly Detection, Reporting, Export and Lineage services.

// ─── Enterprise Data Layer ─────────────────────────────────────────
// EnterpriseRecord preserves the original source of every data element.
// The source application remains the system of record; this layer
// provides the intelligence and aggregation view.

type EnterpriseRecord struct {
	ID           string                 `json:"id"`
	SourceApp    string                 `json:"source_app"`
	SourceEntity string                 `json:"source_entity"`
	SourceID     string                 `json:"source_id"`
	TenantID     string                 `json:"tenant_id"`
	OrgID        string                 `json:"organization_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	Region       string                 `json:"region,omitempty"`
	District     string                 `json:"district,omitempty"`
	FacilityID   string                 `json:"facility_id,omitempty"`
	EventType    string                 `json:"event_type"`
	DataVersion  string                 `json:"data_version,omitempty"`
	Correlation  string                 `json:"correlation_id,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Value        float64                `json:"value,omitempty"`
	TextValue    string                 `json:"text_value,omitempty"`
	GeoLat       float64                `json:"geo_lat,omitempty"`
	GeoLng       float64                `json:"geo_lng,omitempty"`
	Timestamp    string                 `json:"timestamp"`
	ReceivedAt   string                 `json:"received_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Dataset describes an analytical dataset derived from source systems.
type Dataset struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SourceApp    string   `json:"source_app"`
	SourceEntity string   `json:"source_entity,omitempty"`
	Fields       []string `json:"fields"`
	RecordCount  int64    `json:"record_count"`
	Status       string   `json:"status"`
	TenantID     string   `json:"tenant_id"`
	ProjectID    string   `json:"project_id,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
	LastUpdated  string   `json:"last_updated,omitempty"`
}

// ─── KPI Engine ────────────────────────────────────────────────────
// Every KPI has a definition that makes the calculation explainable.
// KPI values are always computed — never hardcoded.

type KPIDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Formula     string                 `json:"formula"`
	SourceApp   string                 `json:"source_app"`
	Scope       string                 `json:"scope"` // organization | project | survey | facility | region
	ScopeID     string                 `json:"scope_id,omitempty"`
	Period      string                 `json:"period"` // current_period | daily | weekly | monthly | quarterly | yearly
	PeriodValue string                 `json:"period_value,omitempty"`
	Unit        string                 `json:"unit,omitempty"`
	Threshold   float64                `json:"threshold,omitempty"`
	ThresholdOp string                 `json:"threshold_op,omitempty"` // below | above | equals
	Version     int                    `json:"version"`
	Status      string                 `json:"status"` // active | deprecated | draft
	Filters     map[string]interface{} `json:"filters,omitempty"`
	OwnerID     string                 `json:"owner_id,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

// KPIValue is a computed KPI result with full lineage.
type KPIValue struct {
	KPIID        string         `json:"kpi_id"`
	KPI          string         `json:"kpi_name"`
	Value        float64        `json:"value"`
	DisplayValue string         `json:"display_value"`
	Unit         string         `json:"unit"`
	Status       string         `json:"status"` // healthy | warning | critical | unknown
	Period       string         `json:"period"`
	PeriodStart  string         `json:"period_start"`
	PeriodEnd    string         `json:"period_end"`
	Scope        string         `json:"scope"`
	ScopeID      string         `json:"scope_id,omitempty"`
	SourceApp    string         `json:"source_app"`
	ComputedAt   string         `json:"computed_at"`
	Breakdown    []KPIComponent `json:"breakdown,omitempty"`
	Lineage      *DataLineage   `json:"lineage,omitempty"`
}

// KPIComponent represents one element in a KPI drill-down breakdown.
type KPIComponent struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"` // project | survey | region | facility | enumerator | status
	Value    float64                `json:"value"`
	Count    int64                  `json:"count"`
	Expected int64                  `json:"expected,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

// DataLineage records where a number came from.
type DataLineage struct {
	KPIID             string                 `json:"kpi_id"`
	KPIFormula        string                 `json:"kpi_formula"`
	SourceApp         string                 `json:"source_app"`
	SourceDatasets    []string               `json:"source_datasets"`
	Calculation       string                 `json:"calculation"`
	ReportingPeriod   string                 `json:"reporting_period"`
	Filters           map[string]interface{} `json:"filters,omitempty"`
	LastUpdated       string                 `json:"last_updated"`
	ResponsibleSystem string                 `json:"responsible_system"`
	RawRecordCount    int64                  `json:"raw_record_count"`
}

// ─── Data Quality Intelligence ─────────────────────────────────────

type DataQualityReport struct {
	ID                     string             `json:"id"`
	Scope                  string             `json:"scope"`
	ScopeID                string             `json:"scope_id,omitempty"`
	TenantID               string             `json:"tenant_id"`
	Score                  float64            `json:"score"`
	CompletenessScore      float64            `json:"completeness_score"`
	DuplicateCount         int64              `json:"duplicate_count"`
	MissingValueCount      int64              `json:"missing_value_count"`
	InvalidDateCount       int64              `json:"invalid_date_count"`
	InvalidCoordCount      int64              `json:"invalid_coord_count"`
	OutlierCount           int64              `json:"outlier_count"`
	InconsistentCount      int64              `json:"inconsistent_count"`
	MissingIdentifierCount int64              `json:"missing_identifier_count"`
	LowReportingFlag       bool               `json:"low_reporting_flag"`
	TotalRecords           int64              `json:"total_records"`
	Issues                 []DataQualityIssue `json:"issues,omitempty"`
	GeneratedAt            string             `json:"generated_at"`
}

type DataQualityIssue struct {
	Type     string                 `json:"type"`
	Severity string                 `json:"severity"`
	Message  string                 `json:"message"`
	EntityID string                 `json:"entity_id,omitempty"`
	Entity   string                 `json:"entity,omitempty"`
	Field    string                 `json:"field,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// ─── Anomaly Detection ─────────────────────────────────────────────

type AnomalyRule struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Type          string                 `json:"type"` // drop | spike | inactivity | delay | pattern
	Metric        string                 `json:"metric"`
	WindowMinutes int                    `json:"window_minutes"`
	ThresholdPct  float64                `json:"threshold_pct"`
	Severity      string                 `json:"severity"` // low | medium | high | critical
	Enabled       bool                   `json:"enabled"`
	Scope         string                 `json:"scope,omitempty"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
	CreatedAt     string                 `json:"created_at"`
}

type AnomalyRecord struct {
	ID            string                 `json:"id"`
	RuleID        string                 `json:"rule_id"`
	RuleName      string                 `json:"rule_name"`
	Type          string                 `json:"type"`
	Severity      string                 `json:"severity"`
	Scope         string                 `json:"scope,omitempty"`
	ScopeID       string                 `json:"scope_id,omitempty"`
	ProjectID     string                 `json:"project_id,omitempty"`
	Description   string                 `json:"description"`
	CurrentValue  float64                `json:"current_value"`
	ExpectedValue float64                `json:"expected_value"`
	WindowStart   string                 `json:"window_start"`
	WindowEnd     string                 `json:"window_end"`
	Status        string                 `json:"status"` // detected | investigating | resolved
	Meta          map[string]interface{} `json:"meta,omitempty"`
	DetectedAt    string                 `json:"detected_at"`
}

// ─── Alert Engine ──────────────────────────────────────────────────

type AlertRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Metric      string                 `json:"metric"`
	Condition   string                 `json:"condition"` // below | above | equals | overdue | expiring
	Threshold   float64                `json:"threshold"`
	Priority    string                 `json:"priority"` // low | medium | high | critical
	Scope       string                 `json:"scope,omitempty"`
	ScopeID     string                 `json:"scope_id,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Recipients  []string               `json:"recipients,omitempty"`
	NotifTitle  string                 `json:"notif_title,omitempty"`
	NotifBody   string                 `json:"notif_body,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	CreatedAt   string                 `json:"created_at"`
}

type AnalyticalAlert struct {
	ID         string                 `json:"id"`
	RuleID     string                 `json:"rule_id"`
	RuleName   string                 `json:"rule_name"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Priority   string                 `json:"priority"`
	Status     string                 `json:"status"` // open | acknowledged | resolved
	Metric     string                 `json:"metric"`
	Value      float64                `json:"value"`
	Threshold  float64                `json:"threshold"`
	Scope      string                 `json:"scope,omitempty"`
	ScopeID    string                 `json:"scope_id,omitempty"`
	ProjectID  string                 `json:"project_id,omitempty"`
	EntityType string                 `json:"entity_type,omitempty"`
	EntityID   string                 `json:"entity_id,omitempty"`
	DeepLink   string                 `json:"deep_link,omitempty"`
	Actions    []AlertAction          `json:"actions,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	ResolvedAt string                 `json:"resolved_at,omitempty"`
}

type AlertAction struct {
	Label string                 `json:"label"`
	Type  string                 `json:"type"` // deep_link | task | ticket | chat
	URL   string                 `json:"url,omitempty"`
	Meta  map[string]interface{} `json:"meta,omitempty"`
}

// ─── Report Builder ────────────────────────────────────────────────

type ReportDefinition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	DataSource   string                 `json:"data_source"`
	SourceApp    string                 `json:"source_app,omitempty"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	DateRange    DateRange              `json:"date_range,omitempty"`
	Grouping     string                 `json:"grouping,omitempty"`
	Aggregation  string                 `json:"aggregation,omitempty"`
	ChartType    string                 `json:"chart_type,omitempty"`
	Columns      []string               `json:"columns,omitempty"`
	Narrative    string                 `json:"narrative,omitempty"`
	Branding     string                 `json:"branding,omitempty"`
	ExportFormat string                 `json:"export_format,omitempty"`
	Schedule     *ScheduleConfig        `json:"schedule,omitempty"`
	CreatedBy    string                 `json:"created_by"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

type DateRange struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
	Type  string `json:"type,omitempty"` // yesterday | this_week | last_week | this_month | last_month | custom
}

type ScheduleConfig struct {
	Frequency  string   `json:"frequency"` // daily | weekly | monthly
	Time       string   `json:"time,omitempty"`
	Day        string   `json:"day,omitempty"`
	Channel    string   `json:"channel"` // workspace | notification | statchat | email
	Recipients []string `json:"recipients,omitempty"`
	Enabled    bool     `json:"enabled"`
}

type ReportRun struct {
	ID          string                 `json:"id"`
	ReportID    string                 `json:"report_id"`
	ReportName  string                 `json:"report_name"`
	Status      string                 `json:"status"` // queued | running | completed | failed
	GenBy       string                 `json:"generated_by,omitempty"`
	Schedule    string                 `json:"schedule,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RecordCount int64                  `json:"record_count,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	CompletedAt string                 `json:"completed_at,omitempty"`
	DataSummary map[string]interface{} `json:"data_summary,omitempty"`
}

// ─── Dashboard Definitions (Phase IV) ──────────────────────────────

type AnalyticsDashboard struct {
	ID          string             `json:"id"`
	Type        string             `json:"type"` // executive | management | project | field
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Scopes      []string           `json:"scopes,omitempty"`
	Sections    []DashboardSection `json:"sections,omitempty"`
	CreatedAt   string             `json:"created_at"`
}

type DashboardSection struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Type  string   `json:"type"`            // kpis | charts | tables | maps | quality | alerts | timeline
	Items []string `json:"items,omitempty"` // KPI IDs or widget IDs
}

// ─── Export Options ────────────────────────────────────────────────

type ExportRequest struct {
	ID          string                 `json:"id"`
	Dataset     string                 `json:"dataset"`
	SourceApp   string                 `json:"source_app,omitempty"`
	Format      string                 `json:"format"` // csv | excel | json | pdf | geojson
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Columns     []string               `json:"columns,omitempty"`
	DateFrom    string                 `json:"date_from,omitempty"`
	DateTo      string                 `json:"date_to,omitempty"`
	RequestedBy string                 `json:"requested_by"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	Status      string                 `json:"status"`
	DownloadURL string                 `json:"download_url,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	ExpiresAt   string                 `json:"expires_at,omitempty"`
}

// ─── Analytics Search ──────────────────────────────────────────────

type AnalyticsSearchResult struct {
	Type        string                 `json:"type"` // dashboard | kpi | report | dataset | alert
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Score       int                    `json:"score"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	URL         string                 `json:"url,omitempty"`
}

// ─── Freshness ─────────────────────────────────────────────────────

type DataFreshness struct {
	DatasetID        string `json:"dataset_id"`
	DatasetName      string `json:"dataset_name"`
	SourceApp        string `json:"source_app"`
	LastUpdated      string `json:"last_updated"`
	RecordCount      int64  `json:"record_count"`
	ProcessingStatus string `json:"processing_status"`
	AgeSeconds       int64  `json:"age_seconds"`
	IsFresh          bool   `json:"is_fresh"`
}

// now returns the current UTC timestamp (helper for models)
func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
