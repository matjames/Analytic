package api

import (
	"github.com/gorilla/mux"
)

// RegisterRoutes registers all StatSpatial REST endpoints onto the gorilla router.
func RegisterRoutes(r *mux.Router) {
	// Probes (no auth required)
	r.HandleFunc("/health", HealthHandler).Methods("GET")
	r.HandleFunc("/ready", ReadyHandler).Methods("GET")
	r.HandleFunc("/live", ReadyHandler).Methods("GET")

	apiV1 := r.PathPrefix("/api/v1").Subrouter()
	apiV1.Use(CORSMiddleware)
	apiV1.Use(LoggingMiddleware)
	apiV1.Use(AuthMiddleware)

	// Admin Units
	apiV1.HandleFunc("/admin-units", ListAdminUnitsHandler).Methods("GET")
	apiV1.HandleFunc("/admin-units", CreateAdminUnitHandler).Methods("POST")
	apiV1.HandleFunc("/admin-units/tree", GetAdminUnitTreeHandler).Methods("GET")
	apiV1.HandleFunc("/admin-units/{id}", GetAdminUnitHandler).Methods("GET")
	apiV1.HandleFunc("/admin-units/{id}", UpdateAdminUnitHandler).Methods("PUT")
	apiV1.HandleFunc("/admin-units/{id}", DeleteAdminUnitHandler).Methods("DELETE")

	// Organizations
	apiV1.HandleFunc("/organizations", ListOrganizationsHandler).Methods("GET")
	apiV1.HandleFunc("/organizations", CreateOrganizationHandler).Methods("POST")
	apiV1.HandleFunc("/organizations/{id}", GetOrganizationHandler).Methods("GET")
	apiV1.HandleFunc("/organizations/{id}", UpdateOrganizationHandler).Methods("PUT")
	apiV1.HandleFunc("/organizations/{id}", DeleteOrganizationHandler).Methods("DELETE")

	// Classifications
	apiV1.HandleFunc("/classifications", ListClassificationsHandler).Methods("GET")
	apiV1.HandleFunc("/classifications", CreateClassificationHandler).Methods("POST")

	// Geo Layers & Features
	apiV1.HandleFunc("/layers", ListGeoLayersHandler).Methods("GET")
	apiV1.HandleFunc("/layers", CreateGeoLayerHandler).Methods("POST")
	apiV1.HandleFunc("/features", ListGeoFeaturesHandler).Methods("GET")
	apiV1.HandleFunc("/features", CreateGeoFeatureHandler).Methods("POST")

	// Spatial Index
	apiV1.HandleFunc("/spatial-index", ListSpatialIndexHandler).Methods("GET")
	apiV1.HandleFunc("/spatial-index", CreateSpatialIndexHandler).Methods("POST")

	// Federated Nodes
	apiV1.HandleFunc("/nodes", ListFederatedNodesHandler).Methods("GET")
	apiV1.HandleFunc("/nodes", CreateFederatedNodeHandler).Methods("POST")
	apiV1.HandleFunc("/nodes/{id}", GetFederatedNodeHandler).Methods("GET")
	apiV1.HandleFunc("/nodes/{id}", UpdateFederatedNodeHandler).Methods("PUT")
	apiV1.HandleFunc("/nodes/{id}", DeleteFederatedNodeHandler).Methods("DELETE")

	// Data Sharing Agreements
	apiV1.HandleFunc("/agreements", ListAgreementsHandler).Methods("GET")
	apiV1.HandleFunc("/agreements", CreateAgreementHandler).Methods("POST")

	// Federated Datasets & Logs
	apiV1.HandleFunc("/datasets", ListFederatedDatasetsHandler).Methods("GET")
	apiV1.HandleFunc("/datasets", CreateFederatedDatasetHandler).Methods("POST")
	apiV1.HandleFunc("/sync-logs", ListSyncLogsHandler).Methods("GET")
	apiV1.HandleFunc("/sync-logs", CreateSyncLogHandler).Methods("POST")

	// Node Links
	apiV1.HandleFunc("/node-links", ListNodeLinksHandler).Methods("GET")
	apiV1.HandleFunc("/node-links", CreateNodeLinkHandler).Methods("POST")

	// Summary & Intelligence
	apiV1.HandleFunc("/summary", GetSummaryHandler).Methods("GET")

	// Spatial Analysis Operations
	apiV1.HandleFunc("/spatial/buffer", SpatialBufferHandler).Methods("POST")
	apiV1.HandleFunc("/spatial/bbox", SpatialBBoxHandler).Methods("GET")
	apiV1.HandleFunc("/spatial/point-in-polygon", SpatialPointInPolygonHandler).Methods("POST")
	apiV1.HandleFunc("/spatial/area/{id}", SpatialAreaCalcHandler).Methods("GET")

	// Direct /api/spatial aliases for convenient frontend mapping
	apiSpatial := r.PathPrefix("/api/spatial").Subrouter()
	apiSpatial.Use(CORSMiddleware)
	apiSpatial.Use(LoggingMiddleware)
	apiSpatial.Use(AuthMiddleware)

	apiSpatial.HandleFunc("/layers", ListGeoLayersHandler).Methods("GET")
	apiSpatial.HandleFunc("/layers", CreateGeoLayerHandler).Methods("POST")
	apiSpatial.HandleFunc("/features", ListGeoFeaturesHandler).Methods("GET")
	apiSpatial.HandleFunc("/features", CreateGeoFeatureHandler).Methods("POST")
	apiSpatial.HandleFunc("/admin-units", ListAdminUnitsHandler).Methods("GET")
	apiSpatial.HandleFunc("/buffer", SpatialBufferHandler).Methods("POST")
	apiSpatial.HandleFunc("/bbox", SpatialBBoxHandler).Methods("GET")
	apiSpatial.HandleFunc("/point-in-polygon", SpatialPointInPolygonHandler).Methods("POST")
	apiSpatial.HandleFunc("/area/{id}", SpatialAreaCalcHandler).Methods("GET")
	apiSpatial.HandleFunc("/summary", GetSummaryHandler).Methods("GET")
}

