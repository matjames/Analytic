package api

import (
	"github.com/gorilla/mux"
)

// RegisterRoutes registers all Knowledge Portal REST endpoints.
//
//   - Probes: /health /ready /live /metrics (no auth)
//   - Public portal (read/search + subscribe/feedback, no auth): /api/public
//   - Admin/CMS (Registry JWT auth): /api/v1
func RegisterRoutes(r *mux.Router) {
	// Probes (no auth required)
	r.HandleFunc("/health", HealthHandler).Methods("GET")
	r.HandleFunc("/ready", ReadyHandler).Methods("GET")
	r.HandleFunc("/live", ReadyHandler).Methods("GET")

	// ── Public, unauthenticated portal ────────────────────────────────
	pub := r.PathPrefix("/api/public").Subrouter()
	pub.Use(CORSMiddleware)
	pub.Use(LoggingMiddleware)
	pub.HandleFunc("/search", PublicSearchHandler).Methods("GET")
	pub.HandleFunc("/content", ListPublicContentHandler).Methods("GET")
	pub.HandleFunc("/content/{id}", GetPublicContentHandler).Methods("GET")
	pub.HandleFunc("/datasets", ListPublicDatasetsHandler).Methods("GET")
	pub.HandleFunc("/datasets/{id}", GetPublicDatasetHandler).Methods("GET")
	pub.HandleFunc("/repository", ListPublicRepositoryHandler).Methods("GET")
	pub.HandleFunc("/repository/{id}", GetPublicRepositoryHandler).Methods("GET")
	pub.HandleFunc("/subscriptions", CreateSubscriptionHandler).Methods("POST")
	pub.HandleFunc("/feedback", CreateFeedbackHandler).Methods("POST")

	// ── Admin / CMS (authenticated) ──────────────────────────────────
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(CORSMiddleware)
	api.Use(LoggingMiddleware)
	api.Use(AuthMiddleware)

	// Content (publications / articles / news / reports)
	api.HandleFunc("/content", ListContentHandler).Methods("GET")
	api.HandleFunc("/content", CreateContentHandler).Methods("POST")
	api.HandleFunc("/content/{id}", GetContentHandler).Methods("GET")
	api.HandleFunc("/content/{id}", UpdateContentHandler).Methods("PUT")
	api.HandleFunc("/content/{id}", DeleteContentHandler).Methods("DELETE")
	api.HandleFunc("/content/{id}/publish", PublishContentHandler).Methods("POST")

	// Open data sets
	api.HandleFunc("/datasets", ListDatasetsHandler).Methods("GET")
	api.HandleFunc("/datasets", CreateDatasetHandler).Methods("POST")
	api.HandleFunc("/datasets/{id}", GetDatasetHandler).Methods("GET")
	api.HandleFunc("/datasets/{id}", UpdateDatasetHandler).Methods("PUT")
	api.HandleFunc("/datasets/{id}", DeleteDatasetHandler).Methods("DELETE")
	api.HandleFunc("/datasets/{id}/publish", PublishDatasetHandler).Methods("POST")

	// Digital library / repository
	api.HandleFunc("/repository", ListRepositoryHandler).Methods("GET")
	api.HandleFunc("/repository", CreateRepositoryHandler).Methods("POST")
	api.HandleFunc("/repository/{id}", GetRepositoryHandler).Methods("GET")
	api.HandleFunc("/repository/{id}", UpdateRepositoryHandler).Methods("PUT")
	api.HandleFunc("/repository/{id}", DeleteRepositoryHandler).Methods("DELETE")

	// Persistent identifiers (DOI / ORCID / ISBN / ISSN)
	api.HandleFunc("/identifiers", CreateIdentifierHandler).Methods("POST")

	// Cross-application object linkage
	api.HandleFunc("/links", CreateLinkHandler).Methods("POST")
	api.HandleFunc("/links", ListLinksHandler).Methods("GET")

	// Admin consoles
	api.HandleFunc("/subscriptions", ListSubscriptionsHandler).Methods("GET")
	api.HandleFunc("/feedback", ListFeedbackHandler).Methods("GET")
	api.HandleFunc("/summary", SummaryHandler).Methods("GET")
}
