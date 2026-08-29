package api

import (
	"github.com/gorilla/mux"
	"github.com/matjames/statgate-lib/tenant"
)

// RegisterRoutes mounts every Learning, Community & Commercial endpoint.
// Probes: /health /ready /live /metrics (public).
// All /api/v1/* endpoints require a Registry-issued JWT.
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

	// ── P24: LMS, Courses, Exams, CPD Certificates & Badges ──
	api.HandleFunc("/courses", ListCoursesHandler).Methods("GET")
	api.HandleFunc("/courses", CreateCourseHandler).Methods("POST")
	api.HandleFunc("/courses/{id}", GetCourseHandler).Methods("GET")
	api.HandleFunc("/courses/{id}", UpdateCourseHandler).Methods("PUT")
	api.HandleFunc("/courses/{id}", DeleteCourseHandler).Methods("DELETE")
	api.HandleFunc("/courses/{id}/publish", PublishCourseHandler).Methods("POST")
	api.HandleFunc("/courses/{id}/modules", ListCourseModulesHandler).Methods("GET")
	api.HandleFunc("/courses/{id}/modules", CreateCourseModuleHandler).Methods("POST")
	api.HandleFunc("/courses/{id}/assessments", ListCourseAssessmentsHandler).Methods("GET")
	api.HandleFunc("/courses/{id}/assessments", CreateAssessmentHandler).Methods("POST")
	api.HandleFunc("/assessments/{id}/attempts", SubmitAssessmentAttemptHandler).Methods("POST")
	api.HandleFunc("/courses/{id}/enroll", EnrollHandler).Methods("POST")
	api.HandleFunc("/courses/{id}/attempts", ListCourseAttemptsHandler).Methods("GET")
	api.HandleFunc("/courses/{id}/complete", CompleteCourseHandler).Methods("POST")
	api.HandleFunc("/enrollments", ListEnrollmentsHandler).Methods("GET")
	api.HandleFunc("/certificates", ListCertificatesHandler).Methods("GET")
	api.HandleFunc("/certificates/{id}", GetCertificateHandler).Methods("GET")
	api.HandleFunc("/certificates/verify/check", VerifyCertificateHandler).Methods("GET")
	api.HandleFunc("/badges", ListBadgesHandler).Methods("GET")

	// ── P26: CRM, Partners & Billing ──
	api.HandleFunc("/leads", ListLeadsHandler).Methods("GET")
	api.HandleFunc("/leads", CreateLeadHandler).Methods("POST")
	api.HandleFunc("/leads/{id}", GetLeadHandler).Methods("GET")
	api.HandleFunc("/leads/{id}", UpdateLeadHandler).Methods("PUT")
	api.HandleFunc("/leads/{id}/convert", ConvertLeadHandler).Methods("POST")
	api.HandleFunc("/accounts", ListAccountsHandler).Methods("GET")
	api.HandleFunc("/accounts", CreateAccountHandler).Methods("POST")
	api.HandleFunc("/accounts/{id}", GetAccountHandler).Methods("GET")
	api.HandleFunc("/accounts/{id}", DeleteAccountHandler).Methods("DELETE")
	api.HandleFunc("/opportunities", ListOpportunitiesHandler).Methods("GET")
	api.HandleFunc("/opportunities", CreateOpportunityHandler).Methods("POST")
	api.HandleFunc("/opportunities/{id}", GetOpportunityHandler).Methods("GET")
	api.HandleFunc("/opportunities/{id}", UpdateOpportunityHandler).Methods("PUT")
	api.HandleFunc("/opportunities/{id}", DeleteOpportunityHandler).Methods("DELETE")
	api.HandleFunc("/partners", ListPartnersHandler).Methods("GET")
	api.HandleFunc("/partners", CreatePartnerHandler).Methods("POST")
	api.HandleFunc("/partners/{id}", GetPartnerHandler).Methods("GET")
	api.HandleFunc("/partners/{id}", UpdatePartnerHandler).Methods("PUT")
	api.HandleFunc("/partners/{id}", DeletePartnerHandler).Methods("DELETE")
	api.HandleFunc("/service-requests", ListServiceRequestsHandler).Methods("GET")
	api.HandleFunc("/service-requests", CreateServiceRequestHandler).Methods("POST")
	api.HandleFunc("/service-requests/{id}", GetServiceRequestHandler).Methods("GET")
	api.HandleFunc("/service-requests/{id}", UpdateServiceRequestHandler).Methods("PUT")
	api.HandleFunc("/invoices", ListInvoicesHandler).Methods("GET")
	api.HandleFunc("/invoices", CreateInvoiceHandler).Methods("POST")
	api.HandleFunc("/invoices/{id}", GetInvoiceHandler).Methods("GET")
	api.HandleFunc("/invoices/{id}", UpdateInvoiceHandler).Methods("PUT")
	api.HandleFunc("/invoices/{id}", DeleteInvoiceHandler).Methods("DELETE")
	api.HandleFunc("/billing/sync/{id}", SyncInvoiceHandler).Methods("POST")

	// ── P35: Stakeholders, Engagements & Stewardship ──
	api.HandleFunc("/stakeholders", ListStakeholdersHandler).Methods("GET")
	api.HandleFunc("/stakeholders", CreateStakeholderHandler).Methods("POST")
	api.HandleFunc("/stakeholders/{id}", GetStakeholderHandler).Methods("GET")
	api.HandleFunc("/stakeholders/{id}", DeleteStakeholderHandler).Methods("DELETE")
	api.HandleFunc("/engagements", ListEngagementsHandler).Methods("GET")
	api.HandleFunc("/engagements", CreateEngagementHandler).Methods("POST")
	api.HandleFunc("/stewardship", ListStewardshipHandler).Methods("GET")
	api.HandleFunc("/stewardship", CreateStewardshipHandler).Methods("POST")
	api.HandleFunc("/stewardship/{id}", UpdateStewardshipHandler).Methods("PUT")

	// ── Cross-app object linkage ──
	api.HandleFunc("/links", CreateLinkHandler).Methods("POST")
	api.HandleFunc("/links", ListLinksHandler).Methods("GET")

	// ── Overview ──
	api.HandleFunc("/summary", SummaryHandler).Methods("GET")
}