package model

import "time"

// GeoLayer is a GIS layer (vector or raster) of geographic content (P44).
type GeoLayer struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	Name         string                 `json:"name"`
	Kind         string                 `json:"kind"` // vector|raster
	GeometryType string                 `json:"geometry_type"` // polygon|point|line
	Source       string                 `json:"source,omitempty"`
	TileLayer    string                 `json:"tile_layer,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Status       string                 `json:"status"` // active|archived
	CreatedBy    string                 `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// GeoFeature is a vector feature with GeoJSON geometry (P44 vector engine).
type GeoFeature struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	LayerID    string                 `json:"layer_id"`
	Name       string                 `json:"name"`
	Geometry   string                 `json:"geometry"` // GeoJSON geometry
	Properties map[string]interface{} `json:"properties,omitempty"`
	Centroid   string                 `json:"centroid,omitempty"`
	CreatedBy  string                 `json:"created_by,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// Raster is a raster dataset processed by the raster engine (P44).
type Raster struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Name      string                 `json:"name"`
	Source    string                 `json:"source,omitempty"`
	Bands     int                    `json:"bands"`
	Width     int                    `json:"width,omitempty"`
	Height    int                    `json:"height,omitempty"`
	BBox      string                 `json:"bbox,omitempty"`
	Stats     map[string]interface{} `json:"stats,omitempty"`
	FilePath  string                 `json:"file_path,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Scene is a satellite/remote-sensing capture (P44 remote sensing service).
type Scene struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Platform    string    `json:"platform"` // sentinel-2|landsat-9|maxar|drone
	CaptureTime time.Time `json:"capture_time"`
	BBox        string    `json:"bbox,omitempty"`
	CloudCover  float64   `json:"cloud_cover"`
	Resolution  float64   `json:"resolution"` // meters
	Status      string    `json:"status"`     // ingested|processing|ready|failed
	ThumbURL    string    `json:"thumb_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ─── Drone Integration (P44) ─────────────────────────────────────────────────

// SpectralIndex is a computed band ratio (e.g. NDVI) over a scene (P44).
type SpectralIndex struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	SceneID   string                 `json:"scene_id"`
	IndexType string                 `json:"index_type"` // ndvi|ndwi|evi
	Values    map[string]interface{} `json:"values,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Drone is a registered UAS platform (P44 drone integration service).
type Drone struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	Name         string                 `json:"name"`
	Model        string                 `json:"model"`
	Serial       string                 `json:"serial,omitempty"`
	Status       string                 `json:"status"` // grounded|ready|in_flight|maintenance
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// FlightPlan is a predefined mission route for a drone (P44).
type FlightPlan struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	DroneID   string                 `json:"drone_id"`
	Name      string                 `json:"name"`
	Waypoints map[string]interface{} `json:"waypoints,omitempty"`
	Altitude  float64                `json:"altitude"`
	Status    string                 `json:"status"` // active|archived
	CreatedAt time.Time              `json:"created_at"`
}

// Flight is an executed mission with a summary and telemetry (P44).
type Flight struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	DroneID   string                 `json:"drone_id"`
	PlanID    string                 `json:"plan_id,omitempty"`
	Status    string                 `json:"status"` // planned|in_flight|completed|aborted
	Summary   map[string]interface{} `json:"summary,omitempty"`
	StartedAt *time.Time             `json:"started_at,omitempty"`
	EndedAt   *time.Time             `json:"ended_at,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// TelemetryPoint is a single telemetry reading (P44).
type TelemetryPoint struct {
	ID       string    `json:"id"`
	TenantID string    `json:"tenant_id"`
	DroneID  string    `json:"drone_id"`
	FlightID string    `json:"flight_id"`
	TS       time.Time `json:"ts"`
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
	Alt      float64   `json:"alt"`
	Battery  float64   `json:"battery"`
	Speed    float64   `json:"speed"`
}

// MapTile is a generated XYZ tile entry (P44 map tile service).
type MapTile struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	LayerID   string    `json:"layer_id"`
	Name      string    `json:"name"`
	Z         int       `json:"z"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	Format    string    `json:"format"`
	Path      string    `json:"path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// GeocodeResult is a resolved address/coordinate pair (P44 geocoding service).
type GeocodeResult struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Query      string    `json:"query,omitempty"`
	Address    string    `json:"address,omitempty"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
}

// ObjectLink mirrors the shared object_links contract.
type ObjectLink struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	Relationship string    `json:"relationship,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}