package main

// ─── StatCitizen Domain Models ────────────────────────────────────────────────
//
// These models define the citizen-specific domain.
// They do NOT duplicate Enterprise Core models (workflows, permissions, analytics, etc.).
// Enterprise interactions use canonical object IDs and event publishing.

// ─── Citizen Identity ─────────────────────────────────────────────────────────

// CitizenSession represents an anonymous or transient citizen session.
// No PII is stored here; it is a correlation token only.
type CitizenSession struct {
	ID        string `json:"id" db:"id"`
	Token     string `json:"token,omitempty" db:"token"`
	TenantID  string `json:"tenant_id" db:"tenant_id"`
	IPHash    string `json:"-" db:"ip_hash"`  // hashed, never raw IP stored permanently
	UserAgent string `json:"-" db:"user_agent"`
	CreatedAt string `json:"created_at" db:"created_at"`
	ExpiresAt string `json:"expires_at" db:"expires_at"`
	LastSeen  string `json:"last_seen" db:"last_seen"`
}

// CitizenIdentityType defines the level of identity provided by the citizen.
type CitizenIdentityType string

const (
	IdentityAnonymous    CitizenIdentityType = "anonymous"
	IdentityVerified     CitizenIdentityType = "verified_contact"
	IdentityRegistered   CitizenIdentityType = "registered"
	IdentityOrgRep       CitizenIdentityType = "org_representative"
)

// RegisteredCitizen holds a persistent citizen profile.
// Data minimization: only collect what the citizen explicitly provides for the purpose.
type RegisteredCitizen struct {
	ID           string              `json:"id" db:"id"`
	TenantID     string              `json:"tenant_id" db:"tenant_id"`
	IdentityType CitizenIdentityType `json:"identity_type" db:"identity_type"`
	DisplayName  string              `json:"display_name,omitempty" db:"display_name"`
	EmailHash    string              `json:"-" db:"email_hash"`       // hashed, not stored raw unless consented
	PhoneHash    string              `json:"-" db:"phone_hash"`       // hashed
	EmailVerified bool               `json:"email_verified" db:"email_verified"`
	PhoneVerified bool               `json:"phone_verified" db:"phone_verified"`
	OrgID        string              `json:"org_id,omitempty" db:"org_id"`        // linked to Registry org
	District     string              `json:"district,omitempty" db:"district"`
	Status       string              `json:"status" db:"status"` // active | suspended
	CreatedAt    string              `json:"created_at" db:"created_at"`
	UpdatedAt    string              `json:"updated_at" db:"updated_at"`
}

// CitizenConsent records explicit consent for each interaction.
// Privacy-by-design: every data collection event requires a linked consent record.
type CitizenConsent struct {
	ID             string                 `json:"id" db:"id"`
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	SessionID      string                 `json:"session_id,omitempty" db:"session_id"`
	CitizenID      string                 `json:"citizen_id,omitempty" db:"citizen_id"`
	Purpose        string                 `json:"purpose" db:"purpose"`                // e.g., "feedback_submission", "report_submission"
	DataCategories []string               `json:"data_categories" db:"data_categories"` // what data is collected
	Visibility     string                 `json:"visibility" db:"visibility"`           // anonymous | institution | public
	Retention      string                 `json:"retention" db:"retention"`             // e.g., "90_days", "1_year", "permanent"
	Sensitivity    string                 `json:"sensitivity" db:"sensitivity"`         // public | internal | confidential
	Granted        bool                   `json:"granted" db:"granted"`
	GrantedAt      string                 `json:"granted_at,omitempty" db:"granted_at"`
	WithdrawnAt    string                 `json:"withdrawn_at,omitempty" db:"withdrawn_at"`
	ExpiresAt      string                 `json:"expires_at,omitempty" db:"expires_at"`
	DownstreamUse  []string               `json:"downstream_use,omitempty" db:"downstream_use"` // permitted uses
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      string                 `json:"created_at" db:"created_at"`
}

// ─── Feedback ─────────────────────────────────────────────────────────────────

// FeedbackRecord represents structured citizen feedback.
type FeedbackRecord struct {
	ID                  string                 `json:"id" db:"id"`
	CanonicalID         string                 `json:"canonical_id" db:"canonical_id"` // {tenant}:statcitizen:feedback:{id}
	TenantID            string                 `json:"tenant_id" db:"tenant_id"`
	SessionID           string                 `json:"session_id,omitempty" db:"session_id"`
	CitizenID           string                 `json:"citizen_id,omitempty" db:"citizen_id"`
	ConsentID           string                 `json:"consent_id" db:"consent_id"`
	CategoryID          string                 `json:"category_id" db:"category_id"` // configurable, not hardcoded
	Subject             string                 `json:"subject" db:"subject"`
	Description         string                 `json:"description" db:"description"`
	District            string                 `json:"district,omitempty" db:"district"`
	FacilityID          string                 `json:"facility_id,omitempty" db:"facility_id"`     // Registry facility
	ProjectID           string                 `json:"project_id,omitempty" db:"project_id"`       // PMS project
	ServiceID           string                 `json:"service_id,omitempty" db:"service_id"`
	Priority            string                 `json:"priority" db:"priority"`       // low | medium | high | critical
	Sensitivity         string                 `json:"sensitivity" db:"sensitivity"` // public | internal | confidential
	Status              string                 `json:"status" db:"status"`           // submitted | received | under_review | assigned | investigating | action_taken | resolved | closed
	AssignedInstitution string                 `json:"assigned_institution,omitempty" db:"assigned_institution"`
	Source              string                 `json:"source" db:"source"` // web | mobile | api
	Anonymous           bool                   `json:"anonymous" db:"anonymous"`
	Visibility          string                 `json:"visibility" db:"visibility"` // citizen | institution | public
	Resolution          string                 `json:"resolution,omitempty" db:"resolution"`
	Response            string                 `json:"response,omitempty" db:"response"`       // institutional response to citizen
	CorrelationID       string                 `json:"correlation_id" db:"correlation_id"`
	Metadata            map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt           string                 `json:"created_at" db:"created_at"`
	UpdatedAt           string                 `json:"updated_at" db:"updated_at"`
	ResolvedAt          string                 `json:"resolved_at,omitempty" db:"resolved_at"`
}

// FeedbackCategory is configurable — not hardcoded.
type FeedbackCategory struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	Name        string `json:"name" db:"name"`
	Slug        string `json:"slug" db:"slug"`
	Description string `json:"description,omitempty" db:"description"`
	ParentID    string `json:"parent_id,omitempty" db:"parent_id"`
	Active      bool   `json:"active" db:"active"`
	SortOrder   int    `json:"sort_order" db:"sort_order"`
	CreatedAt   string `json:"created_at" db:"created_at"`
}

// ─── Reports ──────────────────────────────────────────────────────────────────

// CitizenReport is a structured report submitted by a citizen.
type CitizenReport struct {
	ID             string                 `json:"id" db:"id"`
	CanonicalID    string                 `json:"canonical_id" db:"canonical_id"` // {tenant}:statcitizen:report:{id}
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	SessionID      string                 `json:"session_id,omitempty" db:"session_id"`
	CitizenID      string                 `json:"citizen_id,omitempty" db:"citizen_id"`
	ConsentID      string                 `json:"consent_id" db:"consent_id"`
	CategoryID     string                 `json:"category_id" db:"category_id"` // configurable
	Title          string                 `json:"title" db:"title"`
	Description    string                 `json:"description" db:"description"`
	LocationText   string                 `json:"location_text,omitempty" db:"location_text"`
	District       string                 `json:"district,omitempty" db:"district"`
	Subcounty      string                 `json:"subcounty,omitempty" db:"subcounty"`
	Parish         string                 `json:"parish,omitempty" db:"parish"`
	Latitude       *float64               `json:"latitude,omitempty" db:"latitude"`
	Longitude      *float64               `json:"longitude,omitempty" db:"longitude"`
	LocationConsent bool                  `json:"location_consent" db:"location_consent"` // explicit GPS consent
	FacilityID     string                 `json:"facility_id,omitempty" db:"facility_id"` // Registry
	ProjectID      string                 `json:"project_id,omitempty" db:"project_id"`   // PMS
	ServiceID      string                 `json:"service_id,omitempty" db:"service_id"`
	Priority       string                 `json:"priority" db:"priority"`
	Severity       string                 `json:"severity" db:"severity"` // low | medium | high | critical
	Anonymous      bool                   `json:"anonymous" db:"anonymous"`
	ContactPref    string                 `json:"contact_preference,omitempty" db:"contact_preference"` // none | email | phone | in_app
	Status         string                 `json:"status" db:"status"` // submitted | received | assigned | investigating | resolved | closed
	HelpDeskTicket string                 `json:"helpdesk_ticket_id,omitempty" db:"helpdesk_ticket_id"`
	EnterpriseCase string                 `json:"enterprise_case_id,omitempty" db:"enterprise_case_id"`
	CorrelationID  string                 `json:"correlation_id" db:"correlation_id"`
	Source         string                 `json:"source" db:"source"` // web | mobile | api
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      string                 `json:"created_at" db:"created_at"`
	UpdatedAt      string                 `json:"updated_at" db:"updated_at"`
	ResolvedAt     string                 `json:"resolved_at,omitempty" db:"resolved_at"`
}

// ReportCategory is configurable — not hardcoded.
type ReportCategory struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	Name        string `json:"name" db:"name"`
	Slug        string `json:"slug" db:"slug"`
	Description string `json:"description,omitempty" db:"description"`
	Active      bool   `json:"active" db:"active"`
	SortOrder   int    `json:"sort_order" db:"sort_order"`
	CreatedAt   string `json:"created_at" db:"created_at"`
}

// ─── Attachments ─────────────────────────────────────────────────────────────

// AttachmentRecord tracks files attached to feedback, reports, or consultations.
type AttachmentRecord struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	EntityType  string `json:"entity_type" db:"entity_type"` // feedback | report | consultation_response
	EntityID    string `json:"entity_id" db:"entity_id"`
	FileName    string `json:"file_name" db:"file_name"`
	MimeType    string `json:"mime_type" db:"mime_type"`
	SizeBytes   int64  `json:"size_bytes" db:"size_bytes"`
	StoragePath string `json:"-" db:"storage_path"` // not exposed to clients
	Checksum    string `json:"checksum,omitempty" db:"checksum"`
	ScanStatus  string `json:"scan_status" db:"scan_status"` // pending | clean | quarantined
	UploadedBy  string `json:"uploaded_by,omitempty" db:"uploaded_by"` // session or citizen id
	CreatedAt   string `json:"created_at" db:"created_at"`
}

// ─── Consultations ────────────────────────────────────────────────────────────

// Consultation is a public consultation published by an institution.
type Consultation struct {
	ID                string                 `json:"id" db:"id"`
	CanonicalID       string                 `json:"canonical_id" db:"canonical_id"`
	TenantID          string                 `json:"tenant_id" db:"tenant_id"`
	InstitutionID     string                 `json:"institution_id" db:"institution_id"` // Registry org
	Title             string                 `json:"title" db:"title"`
	Description       string                 `json:"description" db:"description"`
	Category          string                 `json:"category" db:"category"` // configurable
	OpenDate          string                 `json:"open_date" db:"open_date"`
	CloseDate         string                 `json:"close_date" db:"close_date"`
	Status            string                 `json:"status" db:"status"` // draft | published | closed | archived
	AllowAnonymous    bool                   `json:"allow_anonymous" db:"allow_anonymous"`
	RequireVerified   bool                   `json:"require_verified" db:"require_verified"`
	StatCollectFormID string                 `json:"statcollect_form_id,omitempty" db:"statcollect_form_id"` // linked survey
	Questions         []ConsultationQuestion `json:"questions,omitempty"`
	ParticipantCount  int                    `json:"participant_count" db:"participant_count"`
	ResponseCount     int                    `json:"response_count" db:"response_count"`
	CreatedBy         string                 `json:"created_by" db:"created_by"`
	PublishedBy       string                 `json:"published_by,omitempty" db:"published_by"`
	PublishedAt       string                 `json:"published_at,omitempty" db:"published_at"`
	CorrelationID     string                 `json:"correlation_id" db:"correlation_id"`
	Metadata          map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt         string                 `json:"created_at" db:"created_at"`
	UpdatedAt         string                 `json:"updated_at" db:"updated_at"`
}

// ConsultationQuestion is a question within a consultation.
type ConsultationQuestion struct {
	ID          string   `json:"id" db:"id"`
	ConsultID   string   `json:"consultation_id" db:"consultation_id"`
	TenantID    string   `json:"tenant_id" db:"tenant_id"`
	Text        string   `json:"text" db:"text"`
	Type        string   `json:"type" db:"type"` // text | choice | rating | boolean | file
	Options     []string `json:"options,omitempty" db:"options"`
	Required    bool     `json:"required" db:"required"`
	SortOrder   int      `json:"sort_order" db:"sort_order"`
}

// ConsultationResponse is a citizen's response to a consultation.
type ConsultationResponse struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	ConsultID     string                 `json:"consultation_id" db:"consultation_id"`
	SessionID     string                 `json:"session_id,omitempty" db:"session_id"`
	CitizenID     string                 `json:"citizen_id,omitempty" db:"citizen_id"`
	ConsentID     string                 `json:"consent_id" db:"consent_id"`
	Anonymous     bool                   `json:"anonymous" db:"anonymous"`
	Answers       map[string]interface{} `json:"answers" db:"answers"`
	Comment       string                 `json:"comment,omitempty" db:"comment"`
	CorrelationID string                 `json:"correlation_id" db:"correlation_id"`
	CreatedAt     string                 `json:"created_at" db:"created_at"`
}

// ─── Service Ratings ──────────────────────────────────────────────────────────

// ServiceRating is a configurable multi-dimension rating of a service.
type ServiceRating struct {
	ID            string                 `json:"id" db:"id"`
	CanonicalID   string                 `json:"canonical_id" db:"canonical_id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	SessionID     string                 `json:"session_id,omitempty" db:"session_id"`
	CitizenID     string                 `json:"citizen_id,omitempty" db:"citizen_id"`
	ConsentID     string                 `json:"consent_id" db:"consent_id"`
	ServiceID     string                 `json:"service_id" db:"service_id"`
	ServiceName   string                 `json:"service_name" db:"service_name"`
	FacilityID    string                 `json:"facility_id,omitempty" db:"facility_id"`
	District      string                 `json:"district,omitempty" db:"district"`
	Ratings       map[string]int         `json:"ratings" db:"ratings"`      // configurable dimensions → 1-5 score
	OverallScore  float64                `json:"overall_score" db:"overall_score"`
	Comment       string                 `json:"comment,omitempty" db:"comment"`
	Anonymous     bool                   `json:"anonymous" db:"anonymous"`
	Source        string                 `json:"source" db:"source"`
	CorrelationID string                 `json:"correlation_id" db:"correlation_id"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     string                 `json:"created_at" db:"created_at"`
}

// RatingDimension defines a configurable rating dimension (not hardcoded).
type RatingDimension struct {
	ID          string `json:"id" db:"id"`
	TenantID    string `json:"tenant_id" db:"tenant_id"`
	Slug        string `json:"slug" db:"slug"`
	Label       string `json:"label" db:"label"`
	Description string `json:"description,omitempty" db:"description"`
	MinScore    int    `json:"min_score" db:"min_score"` // typically 1
	MaxScore    int    `json:"max_score" db:"max_score"` // typically 5
	Active      bool   `json:"active" db:"active"`
	SortOrder   int    `json:"sort_order" db:"sort_order"`
}

// ─── Public Publications ──────────────────────────────────────────────────────

// PublicPublication represents a governed public information item.
// All public data must pass through this publication gate — never auto-exposed.
type PublicPublication struct {
	ID             string                 `json:"id" db:"id"`
	CanonicalID    string                 `json:"canonical_id" db:"canonical_id"`
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	Category       string                 `json:"category" db:"category"` // statistics | report | dataset | research | policy | project | indicator | consultation_outcome
	Title          string                 `json:"title" db:"title"`
	Summary        string                 `json:"summary" db:"summary"`
	Body           string                 `json:"body,omitempty" db:"body"`
	Source         string                 `json:"source" db:"source"`             // always identifiable
	SourceObjectID string                 `json:"source_object_id,omitempty" db:"source_object_id"` // links to Enterprise object
	SourceApp      string                 `json:"source_app,omitempty" db:"source_app"`
	Publisher      string                 `json:"publisher" db:"publisher"`           // institution
	ApprovedBy     string                 `json:"approved_by,omitempty" db:"approved_by"` // governance gate
	Status         string                 `json:"status" db:"status"` // draft | pending_approval | published | archived | superseded
	PublishedAt    string                 `json:"published_at,omitempty" db:"published_at"`
	ExpiresAt      string                 `json:"expires_at,omitempty" db:"expires_at"`
	Tags           []string               `json:"tags,omitempty" db:"tags"`
	GeoScope       []string               `json:"geo_scope,omitempty" db:"geo_scope"` // which districts/regions this covers
	Language       string                 `json:"language" db:"language"`             // multilingual support
	Metadata       map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt      string                 `json:"created_at" db:"created_at"`
	UpdatedAt      string                 `json:"updated_at" db:"updated_at"`
}

// ─── Citizen Case Tracking ────────────────────────────────────────────────────

// CitizenCase is the citizen-visible status of a submission through the system.
// It does NOT expose internal enterprise details — only status appropriate for the citizen.
type CitizenCase struct {
	ID             string `json:"id" db:"id"`
	TenantID       string `json:"tenant_id" db:"tenant_id"`
	SessionID      string `json:"session_id,omitempty" db:"session_id"`
	CitizenID      string `json:"citizen_id,omitempty" db:"citizen_id"`
	SubmissionType string `json:"submission_type" db:"submission_type"` // feedback | report | consultation | rating
	SubmissionID   string `json:"submission_id" db:"submission_id"`
	Status         string `json:"status" db:"status"` // submitted | received | under_review | assigned | investigating | action_taken | resolved | closed
	StatusMessage  string `json:"status_message,omitempty" db:"status_message"` // citizen-friendly message
	Response       string `json:"response,omitempty" db:"response"` // institutional response
	CorrelationID  string `json:"correlation_id" db:"correlation_id"`
	CreatedAt      string `json:"created_at" db:"created_at"`
	UpdatedAt      string `json:"updated_at" db:"updated_at"`
	ResolvedAt     string `json:"resolved_at,omitempty" db:"resolved_at"`
}

// ─── Geographic Reference ─────────────────────────────────────────────────────

// GeoLocation is a configurable geographic hierarchy (not Uganda-specific).
type GeoLocation struct {
	ID         string `json:"id" db:"id"`
	TenantID   string `json:"tenant_id" db:"tenant_id"`
	Level      string `json:"level" db:"level"` // country | region | district | subcounty | parish | village | facility
	Code       string `json:"code" db:"code"`
	Name       string `json:"name" db:"name"`
	ParentID   string `json:"parent_id,omitempty" db:"parent_id"`
	Latitude   *float64 `json:"latitude,omitempty" db:"latitude"`
	Longitude  *float64 `json:"longitude,omitempty" db:"longitude"`
	FacilityID string `json:"facility_id,omitempty" db:"facility_id"` // linked to Registry
	Active     bool   `json:"active" db:"active"`
}

// ─── Audit ────────────────────────────────────────────────────────────────────

// CitizenAuditEntry is an immutable audit record for StatCitizen actions.
// Citizen-specific audit is separate from enterprise audit for privacy isolation.
type CitizenAuditEntry struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	Actor         string                 `json:"actor" db:"actor"` // session_id, citizen_id, or "system"
	ActorType     string                 `json:"actor_type" db:"actor_type"` // citizen | admin | system
	Action        string                 `json:"action" db:"action"`
	ResourceType  string                 `json:"resource_type" db:"resource_type"`
	ResourceID    string                 `json:"resource_id" db:"resource_id"`
	CorrelationID string                 `json:"correlation_id" db:"correlation_id"`
	Outcome       string                 `json:"outcome" db:"outcome"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     string                 `json:"created_at" db:"created_at"`
}

// ─── Offline / Retry ──────────────────────────────────────────────────────────

// OfflineDraft is a submission draft held for later sync.
type OfflineDraft struct {
	ID            string                 `json:"id" db:"id"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	SessionID     string                 `json:"session_id" db:"session_id"`
	DraftType     string                 `json:"draft_type" db:"draft_type"` // feedback | report | consultation_response
	Payload       map[string]interface{} `json:"payload" db:"payload"`
	RetryCount    int                    `json:"retry_count" db:"retry_count"`
	LastError     string                 `json:"last_error,omitempty" db:"last_error"`
	Status        string                 `json:"status" db:"status"` // pending | processing | completed | failed
	CorrelationID string                 `json:"correlation_id" db:"correlation_id"`
	CreatedAt     string                 `json:"created_at" db:"created_at"`
	UpdatedAt     string                 `json:"updated_at" db:"updated_at"`
}

// EventDeadLetter holds events that failed to publish to the enterprise bus.
type EventDeadLetter struct {
	ID          string                 `json:"id" db:"id"`
	TenantID    string                 `json:"tenant_id" db:"tenant_id"`
	EventType   string                 `json:"event_type" db:"event_type"`
	Payload     map[string]interface{} `json:"payload" db:"payload"`
	RetryCount  int                    `json:"retry_count" db:"retry_count"`
	LastError   string                 `json:"last_error" db:"last_error"`
	Status      string                 `json:"status" db:"status"` // pending | retrying | failed
	CreatedAt   string                 `json:"created_at" db:"created_at"`
	UpdatedAt   string                 `json:"updated_at" db:"updated_at"`
}

// ─── Admin Configuration ──────────────────────────────────────────────────────

// AdminSetting is a key-value configuration item managed by administrators.
type AdminSetting struct {
	ID        string `json:"id" db:"id"`
	TenantID  string `json:"tenant_id" db:"tenant_id"`
	Key       string `json:"key" db:"key"`
	Value     string `json:"value" db:"value"`
	Category  string `json:"category" db:"category"` // moderation | notification | retention | geography | publication
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// ─── API Response Helpers ─────────────────────────────────────────────────────

// PaginatedResponse wraps a list result with pagination metadata.
type PaginatedResponse struct {
	Data    interface{} `json:"data"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Pages   int         `json:"pages"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// SuccessResponse is the standard success envelope.
type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
