package model

import "time"

// Course is an LMS offering with optional CPD credits (P24).
type Course struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"`
	Language    string    `json:"language,omitempty"`
	Credits     int       `json:"credits,omitempty"` // CPD points
	Status      string    `json:"status"`            // draft|published|archived
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Module is a lesson block inside a course (P24).
type Module struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	CourseID    string                 `json:"course_id"`
	Title       string                 `json:"title"`
	OrderIndex  int                    `json:"order_index"`
	Content     map[string]interface{} `json:"content,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// Enrollment tracks a user's progression through a course (P24).
type Enrollment struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	CourseID    string     `json:"course_id"`
	UserID      string     `json:"user_id"`
	Status      string     `json:"status"` // enrolled|in_progress|completed
	Progress    float64    `json:"progress"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Assessment is an exam/quiz attached to a course (P24).
type Assessment struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	CourseID     string                 `json:"course_id"`
	Title        string                 `json:"title"`
	Kind         string                 `json:"kind"` // quiz|exam|practical
	Config       map[string]interface{} `json:"config,omitempty"`
	PassingScore float64                `json:"passing_score"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Attempt is a learner's submission to an assessment (P24).
type Attempt struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	AssessmentID string                `json:"assessment_id"`
	UserID      string                 `json:"user_id"`
	Answers     map[string]interface{} `json:"answers,omitempty"`
	Score       float64                `json:"score"`
	Passed      bool                   `json:"passed"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// Certificate is issued on course completion for CPD credit (P24).
type Certificate struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	CourseID    string     `json:"course_id"`
	UserID      string     `json:"user_id"`
	Number      string     `json:"number"`
	Kind        string     `json:"kind"` // cpd|course
	Credits     int        `json:"credits"`
	IssuedAt    time.Time  `json:"issued_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	VerifyHash  string     `json:"verify_hash"`
}

// Badge is a digital credential issued on completion (P24).
type Badge struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	CourseID  string    `json:"course_id"`
	UserID    string    `json:"user_id"`
	BadgeType string    `json:"badge_type"`
	IssuedAt  time.Time `json:"issued_at"`
}

// ─── Commercial CRM (P26) ────────────────────────────────────────────────────

// Lead is a prospective customer tracked through a sales pipeline (P26).
type Lead struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	WorkspaceID string     `json:"workspace_id,omitempty"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Company     string     `json:"company,omitempty"`
	Source      string     `json:"source,omitempty"`
	Stage       string     `json:"stage"` // lead|qualified|proposal|won|lost
	Value       float64    `json:"value"`
	OwnerID     string     `json:"owner_id,omitempty"`
	Converted   bool       `json:"converted"`
	ConvertedAt *time.Time `json:"converted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Account is an organization in the CRM (P26).
type Account struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type,omitempty"` // government|ngo|private|donor
	Industry  string    `json:"industry,omitempty"`
	Website   string    `json:"website,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Opportunity is a sales opportunity linked to an account/lead (P26).
type Opportunity struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	AccountID     string     `json:"account_id,omitempty"`
	LeadID        string     `json:"lead_id,omitempty"`
	Name          string     `json:"name"`
	Stage         string     `json:"stage"` // discovery|proposal|negotiation|closed_won|closed_lost
	Amount        float64    `json:"amount"`
	OwnerID       string     `json:"owner_id,omitempty"`
	ExpectedClose *time.Time `json:"expected_close,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Partner is a commercial / delivery partner (P26).
type Partner struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	Name         string    `json:"name"`
	Kind         string    `json:"kind"` // reseller|implementer|trainer|advisory
	Status       string    `json:"status"` // active|probation|suspended
	ContactEmail string    `json:"contact_email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ServiceRequest is a service request / ticket in the partner console (P26).
type ServiceRequest struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"` // support|onboarding|training|consultancy
	Requester string    `json:"requester"`
	Priority  string    `json:"priority"` // low|medium|high|critical
	Status    string    `json:"status"`   // open|in_progress|resolved|closed
	Assignee  string    `json:"assignee,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Invoice is a billing record synchronized with the finance backend (P26).
type Invoice struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id"`
	Number    string     `json:"number"`
	AccountID string     `json:"account_id,omitempty"`
	PartnerID string     `json:"partner_id,omitempty"`
	Currency  string     `json:"currency"`
	Amount    float64    `json:"amount"`
	Status    string     `json:"status"` // draft|sent|paid|overdue
	Synced    bool       `json:"synced"`
	IssuedAt  time.Time  `json:"issued_at"`
	DueAt     *time.Time `json:"due_at,omitempty"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ─── Stakeholder Engagement & Governance Stewardship (P35) ───────────────────

// Stakeholder is an engaged party tracked by the Stewardship layer (P35).
type Stakeholder struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"` // government|donor|community|ngo|partner|customer
	Tier           string    `json:"tier,omitempty"`
	EngagementLevel string   `json:"engagement_level,omitempty"`
	ContactEmail   string    `json:"contact_email,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Engagement is a stakeholder touchpoint: forum, workshop, briefing (P35).
type Engagement struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	StakeholderID string     `json:"stakeholder_id"`
	Kind          string     `json:"kind"` // forum|workshop|briefing|survey|newsletter
	Title         string     `json:"title"`
	HappenedAt    time.Time  `json:"happened_at"`
	Notes         string     `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// StewardshipAction tracks a governance action over platform assets (P35).
type StewardshipAction struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Action     string     `json:"action"`
	EntityType string     `json:"entity_type,omitempty"`
	EntityID   string     `json:"entity_id,omitempty"`
	Owner      string     `json:"owner,omitempty"`
	Status     string     `json:"status"` // open|in_progress|done|overdue
	DueAt      *time.Time `json:"due_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ObjectLink mirrors the shared object_links contract (cross-app linkage).
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