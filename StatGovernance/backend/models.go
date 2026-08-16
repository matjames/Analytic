package main

import "time"

// ─── Domain Models ───────────────────────────────────────────────

type Policy struct {
	ID                 string    `json:"id"`
	PolicyNumber       string    `json:"policy_number"`
	Title              string    `json:"title"`
	Summary            string    `json:"summary"`
	Content            string    `json:"content"`
	Category           string    `json:"category"`
	Scope              string    `json:"scope"`
	Classification     string    `json:"classification"`
	Version            string    `json:"version"`
	Status             string    `json:"status"` // Draft, Under Review, Approval, Approved, Published, Active, Under Revision, Superseded, Archived
	Owner              string    `json:"owner"`
	OwnerID            string    `json:"owner_id,omitempty"`
	Department         string    `json:"department"`
	EffectiveDate      string    `json:"effective_date,omitempty"`
	ReviewDate         string    `json:"review_date,omitempty"`
	ExpiryDate         string    `json:"expiry_date,omitempty"`
	TenantID           string    `json:"tenant_id"`
	OrgID              string    `json:"org_id,omitempty"`
	RelatedRegulations []string  `json:"related_regulations,omitempty"`
	RelatedRisks       []string  `json:"related_risks,omitempty"`
	RelatedControls    []string  `json:"related_controls,omitempty"`
	RelatedProjects    []string  `json:"related_projects,omitempty"`
	RelatedResearch    []string  `json:"related_research,omitempty"`
	RelatedDatasets    []string  `json:"related_datasets,omitempty"`
	ApprovedBy         string    `json:"approved_by,omitempty"`
	ApprovedAt         string    `json:"approved_at,omitempty"`
	PublishedAt        string    `json:"published_at,omitempty"`
	CreatedTime        time.Time `json:"created_time"`
	UpdatedTime        time.Time `json:"updated_time"`
}

type PolicyVersion struct {
	ID            string    `json:"id"`
	PolicyID      string    `json:"policy_id"`
	Version       string    `json:"version"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	ChangeSummary string    `json:"change_summary"`
	ChangedBy     string    `json:"changed_by"`
	ApprovedBy    string    `json:"approved_by,omitempty"`
	Status        string    `json:"status"`
	CreatedTime   time.Time `json:"created_time"`
}

type SOP struct {
	ID               string    `json:"id"`
	SOPNumber        string    `json:"sop_number"`
	Title            string    `json:"title"`
	Process          string    `json:"process"`
	Purpose          string    `json:"purpose"`
	Scope            string    `json:"scope"`
	Responsibilities string    `json:"responsibilities"`
	ProcedureSteps   []SOPStep `json:"procedure_steps"`
	RequiredEvidence []string  `json:"required_evidence,omitempty"`
	RelatedPolicyID  string    `json:"related_policy_id,omitempty"`
	RelatedControlID string    `json:"related_control_id,omitempty"`
	ReviewFrequency  string    `json:"review_frequency"`
	Version          string    `json:"version"`
	Status           string    `json:"status"`
	Owner            string    `json:"owner"`
	Department       string    `json:"department"`
	TenantID         string    `json:"tenant_id"`
	ApprovedBy       string    `json:"approved_by,omitempty"`
	ApprovedAt       string    `json:"approved_at,omitempty"`
	CreatedTime      time.Time `json:"created_time"`
	UpdatedTime      time.Time `json:"updated_time"`
}

type SOPStep struct {
	StepNumber  int    `json:"step_number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Role        string `json:"role"`
	Evidence    string `json:"evidence,omitempty"`
}

type Regulation struct {
	ID                  string    `json:"id"`
	Code                string    `json:"code"`
	Name                string    `json:"name"`
	RegulatoryAuthority string    `json:"regulatory_authority"`
	Jurisdiction        string    `json:"jurisdiction"`
	Category            string    `json:"category"`
	Description         string    `json:"description"`
	SourceURL           string    `json:"source_url,omitempty"`
	EffectiveDate       string    `json:"effective_date,omitempty"`
	TenantID            string    `json:"tenant_id"`
	CreatedTime         time.Time `json:"created_time"`
	UpdatedTime         time.Time `json:"updated_time"`
}

type ComplianceObligation struct {
	ID                  string    `json:"id"`
	ObligationCode      string    `json:"obligation_code"`
	RegulationID        string    `json:"regulation_id"`
	RegulationName      string    `json:"regulation_name,omitempty"`
	Requirement         string    `json:"requirement"`
	ApplicableScope     string    `json:"applicable_scope"`
	ResponsibleDept     string    `json:"responsible_dept"`
	ResponsiblePerson   string    `json:"responsible_person"`
	Frequency           string    `json:"frequency"`
	ComplianceStatus    string    `json:"compliance_status"` // Compliant, Partially Compliant, Non-Compliant, Under Review, Not Assessed
	EvidenceRequirement string    `json:"evidence_requirement"`
	EvidenceLocation    string    `json:"evidence_location,omitempty"`
	ViolationRisk       string    `json:"violation_risk"`
	CorrectiveAction    string    `json:"corrective_action,omitempty"`
	Deadline            string    `json:"deadline,omitempty"`
	ReviewDate          string    `json:"review_date,omitempty"`
	TenantID            string    `json:"tenant_id"`
	CreatedTime         time.Time `json:"created_time"`
	UpdatedTime         time.Time `json:"updated_time"`
}

type ComplianceAssessment struct {
	ID              string           `json:"id"`
	AssessmentCode  string           `json:"assessment_code"`
	Title           string           `json:"title"`
	Scope           string           `json:"scope"`
	Assessor        string           `json:"assessor"`
	AssessorID      string           `json:"assessor_id,omitempty"`
	Status          string           `json:"status"` // Created, Assigned, Evidence Collection, Assessment, Review, Completed
	ScorePercentage float64          `json:"score_percentage"`
	FindingsCount   int              `json:"findings_count"`
	Recommendations string           `json:"recommendations"`
	StartDate       string           `json:"start_date,omitempty"`
	CompletionDate  string           `json:"completion_date,omitempty"`
	Items           []AssessmentItem `json:"items,omitempty"`
	TenantID        string           `json:"tenant_id"`
	CreatedTime     time.Time        `json:"created_time"`
	UpdatedTime     time.Time        `json:"updated_time"`
}

type AssessmentItem struct {
	ID             string    `json:"id"`
	AssessmentID   string    `json:"assessment_id"`
	ObligationID   string    `json:"obligation_id"`
	ObligationCode string    `json:"obligation_code,omitempty"`
	Requirement    string    `json:"requirement,omitempty"`
	Status         string    `json:"status"`
	Notes          string    `json:"notes"`
	EvidenceRef    string    `json:"evidence_ref,omitempty"`
	Findings       string    `json:"findings,omitempty"`
	CreatedTime    time.Time `json:"created_time"`
}

type Risk struct {
	ID                  string          `json:"id"`
	RiskCode            string          `json:"risk_code"`
	Title               string          `json:"title"`
	Description         string          `json:"description"`
	Category            string          `json:"category"` // Strategic, Operational, Compliance, Information Security, Financial, Reputational, Data Integrity
	Owner               string          `json:"owner"`
	OwnerID             string          `json:"owner_id,omitempty"`
	Organization        string          `json:"organization"`
	Department          string          `json:"department"`
	ProjectID           string          `json:"project_id,omitempty"`
	ResearchID          string          `json:"research_id,omitempty"`
	Probability         int             `json:"probability"` // 1-5
	Impact              int             `json:"impact"`      // 1-5
	InherentRiskScore   int             `json:"inherent_risk_score"`
	InherentRiskLevel   string          `json:"inherent_risk_level"` // Low, Medium, High, Critical
	TreatmentStrategy   string          `json:"treatment_strategy"`  // Mitigate, Avoid, Transfer, Accept
	MitigationActions   string          `json:"mitigation_actions"`
	ResidualProbability int             `json:"residual_probability,omitempty"`
	ResidualImpact      int             `json:"residual_impact,omitempty"`
	ResidualRiskScore   int             `json:"residual_risk_score,omitempty"`
	ResidualRiskLevel   string          `json:"residual_risk_level,omitempty"`
	Status              string          `json:"status"` // Identified, Assessed, In Treatment, Monitored, Closed
	TargetDate          string          `json:"target_date,omitempty"`
	ReviewDate          string          `json:"review_date,omitempty"`
	Treatments          []RiskTreatment `json:"treatments,omitempty"`
	TenantID            string          `json:"tenant_id"`
	CreatedTime         time.Time       `json:"created_time"`
	UpdatedTime         time.Time       `json:"updated_time"`
}

type RiskTreatment struct {
	ID                string    `json:"id"`
	RiskID            string    `json:"risk_id"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	ActionType        string    `json:"action_type"`
	ResponsiblePerson string    `json:"responsible_person"`
	DueDate           string    `json:"due_date,omitempty"`
	Status            string    `json:"status"`
	EnterpriseTaskID  string    `json:"enterprise_task_id,omitempty"`
	CompletionDate    string    `json:"completion_date,omitempty"`
	CreatedTime       time.Time `json:"created_time"`
}

type Control struct {
	ID                  string        `json:"id"`
	ControlCode         string        `json:"control_code"`
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	Objective           string        `json:"objective"`
	Owner               string        `json:"owner"`
	Department          string        `json:"department"`
	ControlType         string        `json:"control_type"` // Preventive, Detective, Corrective, Directive, Compensating
	Frequency           string        `json:"frequency"`    // Continuous, Daily, Weekly, Monthly, Quarterly, Annual
	TestMethod          string        `json:"test_method"`
	EvidenceRequirement string        `json:"evidence_requirement"`
	Effectiveness       string        `json:"effectiveness"` // Effective, Partially Effective, Ineffective, Not Tested
	RelatedRiskID       string        `json:"related_risk_id,omitempty"`
	RelatedPolicyID     string        `json:"related_policy_id,omitempty"`
	LastTested          string        `json:"last_tested,omitempty"`
	NextTestDate        string        `json:"next_test_date,omitempty"`
	Tests               []ControlTest `json:"tests,omitempty"`
	TenantID            string        `json:"tenant_id"`
	CreatedTime         time.Time     `json:"created_time"`
	UpdatedTime         time.Time     `json:"updated_time"`
}

type ControlTest struct {
	ID                string    `json:"id"`
	ControlID         string    `json:"control_id"`
	Tester            string    `json:"tester"`
	TestDate          time.Time `json:"test_date"`
	Result            string    `json:"result"`        // Pass, Partial Pass, Fail
	Effectiveness     string    `json:"effectiveness"` // Effective, Partially Effective, Ineffective
	Findings          string    `json:"findings"`
	EvidenceRef       string    `json:"evidence_ref,omitempty"`
	RemediationTaskID string    `json:"remediation_task_id,omitempty"`
	CreatedTime       time.Time `json:"created_time"`
}

type Audit struct {
	ID               string         `json:"id"`
	AuditCode        string         `json:"audit_code"`
	Title            string         `json:"title"`
	AuditType        string         `json:"audit_type"` // Internal Audit, External Compliance, ISO/IEC 27001, Data Protection, Financial, Operational
	Scope            string         `json:"scope"`
	Objectives       string         `json:"objectives"`
	LeadAuditor      string         `json:"lead_auditor"`
	Auditors         []string       `json:"auditors"`
	Auditees         []string       `json:"auditees"`
	Department       string         `json:"department"`
	StartDate        string         `json:"start_date,omitempty"`
	EndDate          string         `json:"end_date,omitempty"`
	Status           string         `json:"status"` // Planned, Scheduled, In Progress, Findings, Management Response, Corrective Action, Verification, Closed
	FindingsCount    int            `json:"findings_count"`
	CriticalFindings int            `json:"critical_findings"`
	Summary          string         `json:"summary"`
	ReportFileID     string         `json:"report_file_id,omitempty"`
	Findings         []AuditFinding `json:"findings,omitempty"`
	TenantID         string         `json:"tenant_id"`
	CreatedTime      time.Time      `json:"created_time"`
	UpdatedTime      time.Time      `json:"updated_time"`
}

type AuditFinding struct {
	ID                 string    `json:"id"`
	FindingCode        string    `json:"finding_code"`
	AuditID            string    `json:"audit_id,omitempty"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Severity           string    `json:"severity"` // Critical, High, Medium, Low, Observation
	Source             string    `json:"source"`
	Owner              string    `json:"owner"`
	Department         string    `json:"department"`
	PolicyID           string    `json:"policy_id,omitempty"`
	ControlID          string    `json:"control_id,omitempty"`
	RiskID             string    `json:"risk_id,omitempty"`
	Recommendation     string    `json:"recommendation"`
	ManagementResponse string    `json:"management_response,omitempty"`
	DueDate            string    `json:"due_date"`
	Status             string    `json:"status"` // Open, In Remediation, Remediation Submitted, Verified, Closed
	EnterpriseTaskID   string    `json:"enterprise_task_id,omitempty"`
	ClosureEvidence    string    `json:"closure_evidence,omitempty"`
	ClosedBy           string    `json:"closed_by,omitempty"`
	ClosedAt           string    `json:"closed_at,omitempty"`
	TenantID           string    `json:"tenant_id"`
	CreatedTime        time.Time `json:"created_time"`
	UpdatedTime        time.Time `json:"updated_time"`
}

type CorrectiveAction struct {
	ID                string    `json:"id"`
	ActionCode        string    `json:"action_code"`
	Title             string    `json:"title"`
	Description       string    `json:"description"`
	SourceType        string    `json:"source_type"` // Finding, Risk, Control Failure, Incident, Assessment
	SourceID          string    `json:"source_id"`
	Owner             string    `json:"owner"`
	Priority          string    `json:"priority"` // Critical, High, Medium, Low
	DueDate           string    `json:"due_date"`
	EnterpriseTaskID  string    `json:"enterprise_task_id,omitempty"`
	Status            string    `json:"status"` // Pending, In Progress, Completed, Verified, Closed
	CompletionDate    string    `json:"completion_date,omitempty"`
	VerificationNotes string    `json:"verification_notes,omitempty"`
	VerifiedBy        string    `json:"verified_by,omitempty"`
	VerifiedAt        string    `json:"verified_at,omitempty"`
	TenantID          string    `json:"tenant_id"`
	CreatedTime       time.Time `json:"created_time"`
	UpdatedTime       time.Time `json:"updated_time"`
}

type Committee struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Code              string            `json:"code"`
	Mandate           string            `json:"mandate"`
	Chairperson       string            `json:"chairperson"`
	Secretary         string            `json:"secretary"`
	MeetingFrequency  string            `json:"meeting_frequency"`
	StatChatChannelID string            `json:"statchat_channel_id,omitempty"`
	Members           []CommitteeMember `json:"members,omitempty"`
	Meetings          []Meeting         `json:"meetings,omitempty"`
	TenantID          string            `json:"tenant_id"`
	CreatedTime       time.Time         `json:"created_time"`
	UpdatedTime       time.Time         `json:"updated_time"`
}

type CommitteeMember struct {
	ID          string    `json:"id"`
	CommitteeID string    `json:"committee_id"`
	Name        string    `json:"name"`
	Role        string    `json:"role"` // Chairperson, Secretary, Member, Technical Advisor, Observer
	Email       string    `json:"email"`
	Department  string    `json:"department"`
	JoinedDate  string    `json:"joined_date,omitempty"`
	CreatedTime time.Time `json:"created_time"`
}

type Meeting struct {
	ID              string    `json:"id"`
	CommitteeID     string    `json:"committee_id"`
	CommitteeName   string    `json:"committee_name,omitempty"`
	Title           string    `json:"title"`
	MeetingDate     time.Time `json:"meeting_date"`
	Location        string    `json:"location"`
	CalendarEventID string    `json:"calendar_event_id,omitempty"`
	Status          string    `json:"status"` // Scheduled, In Progress, Completed, Cancelled
	Agenda          string    `json:"agenda"`
	Minutes         string    `json:"minutes"`
	Attendees       []string  `json:"attendees"`
	DecisionsCount  int       `json:"decisions_count"`
	TenantID        string    `json:"tenant_id"`
	CreatedTime     time.Time `json:"created_time"`
	UpdatedTime     time.Time `json:"updated_time"`
}

type GovernanceDecision struct {
	ID                   string                 `json:"id"`
	DecisionCode         string                 `json:"decision_code"`
	Title                string                 `json:"title"`
	Context              string                 `json:"context"`
	OptionsConsidered    []OptionConsidered     `json:"options_considered"`
	RisksEvaluated       string                 `json:"risks_evaluated"`
	Recommendation       string                 `json:"recommendation"`
	FinalDecision        string                 `json:"final_decision"`
	DecisionMaker        string                 `json:"decision_maker"`
	CommitteeID          string                 `json:"committee_id,omitempty"`
	MeetingID            string                 `json:"meeting_id,omitempty"`
	PolicyID             string                 `json:"policy_id,omitempty"`
	Status               string                 `json:"status"`
	AIAdvisory           bool                   `json:"ai_advisory"`
	HumanApproved        bool                   `json:"human_approved"`
	EnterpriseDecisionID string                 `json:"enterprise_decision_id,omitempty"`
	EffectiveDate        string                 `json:"effective_date,omitempty"`
	TenantID             string                 `json:"tenant_id"`
	CreatedTime          time.Time              `json:"created_time"`
}

type OptionConsidered struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Pros        string `json:"pros"`
	Cons        string `json:"cons"`
}

type EvidenceRecord struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Category           string    `json:"category"`
	SourceApplication  string    `json:"source_application"`
	RelatedEntityType  string    `json:"related_entity_type"`
	RelatedEntityID    string    `json:"related_entity_id"`
	FileID             string    `json:"file_id,omitempty"`
	FileName           string    `json:"file_name,omitempty"`
	FileURL            string    `json:"file_url,omitempty"`
	MimeType           string    `json:"mime_type,omitempty"`
	FileSize           int64     `json:"file_size,omitempty"`
	ChecksumSHA256     string    `json:"checksum_sha256,omitempty"`
	Classification     string    `json:"classification"`
	Version            int       `json:"version"`
	UploadedBy         string    `json:"uploaded_by"`
	TenantID           string    `json:"tenant_id"`
	CreatedTime        time.Time `json:"created_time"`
}

type DataGovernanceRecord struct {
	ID                string    `json:"id"`
	DatasetName       string    `json:"dataset_name"`
	DatasetSource     string    `json:"dataset_source"`
	DataOwner         string    `json:"data_owner"`
	DataSteward       string    `json:"data_steward"`
	Classification    string    `json:"classification"`
	SensitivityLevel  string    `json:"sensitivity_level"`
	RetentionPeriod   string    `json:"retention_period"`
	AccessRules       string    `json:"access_rules"`
	LineageReference  string    `json:"lineage_reference,omitempty"`
	QualityThreshold  float64   `json:"quality_threshold"`
	SharingAgreements string    `json:"sharing_agreements,omitempty"`
	TenantID          string    `json:"tenant_id"`
	CreatedTime       time.Time `json:"created_time"`
	UpdatedTime       time.Time `json:"updated_time"`
}

type PrivacyAssessment struct {
	ID                  string    `json:"id"`
	ActivityName        string    `json:"activity_name"`
	DataController      string    `json:"data_controller"`
	LawfulBasis         string    `json:"lawful_basis"`
	PersonalDataTypes   []string  `json:"personal_data_types"`
	RetentionRules      string    `json:"retention_rules"`
	CrossBorderTransfer bool      `json:"cross_border_transfer"`
	DPIAStatus          string    `json:"dpia_status"`
	BreachRecordsCount  int       `json:"breach_records_count"`
	TenantID            string    `json:"tenant_id"`
	CreatedTime         time.Time `json:"created_time"`
	UpdatedTime         time.Time `json:"updated_time"`
}

type Delegation struct {
	ID            string    `json:"id"`
	DelegatorID   string    `json:"delegator_id"`
	DelegatorName string    `json:"delegator_name"`
	DelegateID    string    `json:"delegate_id"`
	DelegateName  string    `json:"delegate_name"`
	RoleScope     string    `json:"role_scope"`
	ScopeDetails  string    `json:"scope_details"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	Reason        string    `json:"reason"`
	Status        string    `json:"status"` // Active, Expired, Revoked
	TenantID      string    `json:"tenant_id"`
	CreatedTime   time.Time `json:"created_time"`
	UpdatedTime   time.Time `json:"updated_time"`
}

type GovernanceDashboardKPIs struct {
	PoliciesActive          int            `json:"policies_active"`
	PoliciesUnderReview     int            `json:"policies_under_review"`
	PoliciesExpired         int            `json:"policies_expired"`
	PoliciesAwaitingApproval int           `json:"policies_awaiting_approval"`
	ComplianceRate          float64        `json:"compliance_rate"`
	ComplianceCompliant     int            `json:"compliance_compliant"`
	CompliancePartially     int            `json:"compliance_partially"`
	ComplianceNonCompliant  int            `json:"compliance_non_compliant"`
	ComplianceOverdue       int            `json:"compliance_overdue"`
	RisksCritical           int            `json:"risks_critical"`
	RisksHigh               int            `json:"risks_high"`
	RisksMedium             int            `json:"risks_medium"`
	RisksLow                int            `json:"risks_low"`
	RiskHeatmapMatrix       map[string]int `json:"risk_heatmap_matrix"` // key: "prob_impact", value: count
	ControlsEffective       int            `json:"controls_effective"`
	ControlsNeedsReview     int            `json:"controls_needs_review"`
	ControlsFailed          int            `json:"controls_failed"`
	ControlsNotTested       int            `json:"controls_not_tested"`
	AuditsPlanned           int            `json:"audits_planned"`
	AuditsInProgress        int            `json:"audits_in_progress"`
	FindingsOpen            int            `json:"findings_open"`
	FindingsOverdue         int            `json:"findings_overdue"`
	PendingApprovalsCount   int            `json:"pending_approvals_count"`
	RecentDecisionsCount    int            `json:"recent_decisions_count"`
}

type SearchItem struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Source      string `json:"source"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
}

// ─── Phase 13 Extensions: Whistleblower, COI, Feature Flags, Parameters ───

type WhistleblowerReport struct {
	ID             string    `json:"id"`
	TicketNumber   string    `json:"ticket_number"`
	Title          string    `json:"title"`
	Category       string    `json:"category"` // Fraud, Corruption, Harassment, Safety, Breach, Other
	Description    string    `json:"description"`
	EvidenceFiles  []string  `json:"evidence_files,omitempty"`
	Status         string    `json:"status"` // Submitted, Under Investigation, Action Taken, Closed
	EncryptedNotes string    `json:"encrypted_notes,omitempty"`
	AssignedTo     string    `json:"assigned_to,omitempty"`
	TenantID       string    `json:"tenant_id"`
	CreatedTime    time.Time `json:"created_time"`
	UpdatedTime    time.Time `json:"updated_time"`
}

type ConflictDeclaration struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	UserName        string    `json:"user_name"`
	Department      string    `json:"department"`
	DeclarationType string    `json:"declaration_type"` // Annual, Ad-Hoc, Procurement, Recruitment
	EntityName      string    `json:"entity_name"`
	NatureOfInterest string   `json:"nature_of_interest"` // Financial, Directorship, Family, Shareholding
	Description     string    `json:"description"`
	MitigationPlan  string    `json:"mitigation_plan"`
	Status          string    `json:"status"` // Declared, Under Review, Approved, Mitigated, Rejected
	ReviewedBy      string    `json:"reviewed_by,omitempty"`
	ReviewedAt      string    `json:"reviewed_at,omitempty"`
	TenantID        string    `json:"tenant_id"`
	CreatedTime     time.Time `json:"created_time"`
}

type FeatureFlag struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	TenantID    string    `json:"tenant_id"`
	Module      string    `json:"module"`
	RolloutPct  int       `json:"rollout_pct"`
	CreatedTime time.Time `json:"created_time"`
}

type SystemParameter struct {
	ID          string    `json:"id"`
	ParamKey    string    `json:"param_key"`
	ParamValue  string    `json:"param_value"`
	Description string    `json:"description"`
	DataType    string    `json:"data_type"` // string, number, boolean, json
	Category    string    `json:"category"`
	TenantID    string    `json:"tenant_id"`
	UpdatedTime time.Time `json:"updated_time"`
}


