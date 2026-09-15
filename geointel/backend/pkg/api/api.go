package api

import (
	"github.com/gorilla/mux"
	"github.com/matjames/statgate-lib/tenant"
)

// RegisterRoutes mounts Geospatial & Remote Sensing (App 9) endpoints.
// Probes: /health /ready /live /metrics (public). All /api/v1/* require JWT.
func RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/health", HealthHandler).Methods("GET")
	r.HandleFunc("/ready", ReadyHandler).Methods("GET")
	r.HandleFunc("/live", ReadyHandler).Methods("GET")
	r.Handle("/metrics", MetricsHandler())

	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(CORSMiddleware)
	api.Use(LoggingMiddleware)
	api.Use(AuthMiddleware)

	// Stage 2: workspace context + Enterprise Core membership enforcement.
	api.Use(tenant.WorkspaceContext)
	api.Use(tenant.WorkspaceMembership("", nil))
	if discussionHandler != nil {
		api.HandleFunc("/discussions", discussionHandler.Create).Methods("POST")
	}

	// ── GIS Service & Vector Engine (P44) ──
	api.HandleFunc("/layers", ListLayersHandler).Methods("GET")
	api.HandleFunc("/layers", CreateLayerHandler).Methods("POST")
	api.HandleFunc("/layers/{id}", GetLayerHandler).Methods("GET")
	api.HandleFunc("/layers/{id}", DeleteLayerHandler).Methods("DELETE")
	api.HandleFunc("/layers/{id}/features", ListLayerFeaturesHandler).Methods("GET")
	api.HandleFunc("/layers/{id}/features", CreateFeatureHandler).Methods("POST")
	api.HandleFunc("/features/{id}", GetFeatureHandler).Methods("GET")
	api.HandleFunc("/features/{id}", DeleteFeatureHandler).Methods("DELETE")

	// ── Spatial Analytics Engine (P44) ──
	api.HandleFunc("/analysis/buffer", BufferHandler).Methods("POST")
	api.HandleFunc("/analysis/contains", ContainsHandler).Methods("POST")
	api.HandleFunc("/analysis/distance", DistanceHandler).Methods("POST")
	api.HandleFunc("/analysis/search", SpatialSearchHandler).Methods("GET")
	api.HandleFunc("/analysis/summary/{layerId}", LayerSummaryHandler).Methods("GET")

	// ── Remote Sensing & Raster Engine (P44) ──
	api.HandleFunc("/rasters", ListRastersHandler).Methods("GET")
	api.HandleFunc("/rasters", CreateRasterHandler).Methods("POST")
	api.HandleFunc("/scenes", ListScenesHandler).Methods("GET")
	api.HandleFunc("/scenes", CreateSceneHandler).Methods("POST")
	api.HandleFunc("/scenes/{id}", GetSceneHandler).Methods("GET")
	api.HandleFunc("/scenes/{id}/process", ProcessSceneHandler).Methods("POST")
	api.HandleFunc("/indices", ListIndicesHandler).Methods("GET")

	// ── Drone Integration Service (P44) ──
	api.HandleFunc("/drones", ListDronesHandler).Methods("GET")
	api.HandleFunc("/drones", CreateDroneHandler).Methods("POST")
	api.HandleFunc("/drones/{id}", GetDroneHandler).Methods("GET")
	api.HandleFunc("/drones/{id}/status", DroneStatusHandler).Methods("PUT")
	api.HandleFunc("/flight-plans", ListFlightPlansHandler).Methods("GET")
	api.HandleFunc("/flight-plans", CreateFlightPlanHandler).Methods("POST")
	api.HandleFunc("/flights", ListFlightsHandler).Methods("GET")
	api.HandleFunc("/flights", StartFlightHandler).Methods("POST")
	api.HandleFunc("/flights/{id}/telemetry", FlightTelemetryHandler).Methods("POST")
	api.HandleFunc("/flights/{id}/telemetry", ListFlightTelemetryHandler).Methods("GET")
	api.HandleFunc("/flights/{id}/end", EndFlightHandler).Methods("POST")

	// ── Map Tile Service & Geocoding (P44) ──
	api.HandleFunc("/tiles", ListTilesHandler).Methods("GET")
	api.HandleFunc("/tiles", RegisterTileHandler).Methods("POST")
	api.HandleFunc("/geocode", GeocodeHandler).Methods("POST")
	api.HandleFunc("/geocode/reverse", ReverseGeocodeHandler).Methods("GET")
	api.HandleFunc("/geocode/cache", GeocodeCacheHandler).Methods("GET")

	// ── Cross-app object linkage + overview ──
	api.HandleFunc("/links", CreateLinkHandler).Methods("POST")
	api.HandleFunc("/links", ListLinksHandler).Methods("GET")
	api.HandleFunc("/summary", SummaryHandler).Methods("GET")
}

var discussionHandler *DiscussionHandler

// ConfigureStatChat enables the canonical object-discussion boundary.
func ConfigureStatChat(baseURL, internalKey string) {
	discussionHandler = NewDiscussionHandler(NewStatChatIntegration(baseURL, internalKey))
}
