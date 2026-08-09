package main

import (
	"database/sql/driver"
	"strings"
	"time"
)

// ─── Research Lifecycle Stages ─────────────────────────────
type ResearchStage string

const (
	StageResearchIdea      ResearchStage = "Research Idea"
	StageConceptNote       ResearchStage = "Concept Note"
	StageResearchProposal  ResearchStage = "Research Proposal"
	StageProtocolDev       ResearchStage = "Protocol Development"
	StageInternalReview    ResearchStage = "Internal Review"
	StageEthicsSubmission  ResearchStage = "Ethics Submission"
	StageEthicsApproval    ResearchStage = "Ethics Approval"
	StageFundingApproval   ResearchStage = "Funding Approval"
	StageProjectActivation ResearchStage = "Project Activation"
	StageSurveyDesign      ResearchStage = "Survey Design"
	StageFieldCollection   ResearchStage = "Field Data Collection"
	StageDataValidation    ResearchStage = "Data Validation"
	StageStatAnalysis      ResearchStage = "Statistical Analysis"
	StageInterpretation    ResearchStage = "Interpretation"
	StageReportWriting     ResearchStage = "Report Writing"
	StagePublication       ResearchStage = "Publication"
	StageKnowledgeRepo     ResearchStage = "Knowledge Repository"
	StageArchive           ResearchStage = "Archive"
)

// StringArray is a helper type for PostgreSQL TEXT[] columns
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return "{" + strings.Join(a, ",") + "}", nil
}

// ─── Core Research Project ────────────────────────────────
type ResearchProject struct {
	ID                    string    `json:"id"`
	Code                  string    `json:"code"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	Type                  string    `json:"type"`
	Stage                 string    `json:"stage"`
	Progress              float64   `json:"progress"`
	PrincipalInvestigator string    `json:"principalInvestigator"`
	Owner                 string    `json:"owner"`
	Organisation          string    `json:"organisation"`
	Portfolio             string    `json:"portfolio"`
	Programme             string    `json:"programme"`
	StartDate             string    `json:"startDate"`
	EndDate               string    `json:"endDate"`
	TargetGeo             string    `json:"targetGeo"`
	BudgetTotal           float64   `json:"budgetTotal"`
	SpentTotal            float64   `json:"spentTotal"`
	Tags                  []string  `json:"tags"`
	PmsProjectID          string    `json:"pmsProjectId,omitempty"`
	StatchatRoomID        string    `json:"statchatRoomId,omitempty"`
	CreatedTime           time.Time `json:"createdTime"`
	UpdatedTime           time.Time `json:"updatedTime"`
}

// ─── Research Workspace (all related data) ────────────────
type ResearchWorkspace struct {
	Research       *ResearchProject `json:"research"`
	Members        []ResearchMember `json:"members"`
	Proposals      []Proposal       `json:"proposals"`
	Ethics         []EthicsApp      `json:"ethics"`
	Grants         []Grant          `json:"grants"`
	Literature     []LiteratureItem `json:"literature"`
	Datasets       []Dataset        `json:"datasets"`
	Publications   []Publication    `json:"publications"`
	Tasks          []ResearchTask   `json:"tasks"`
	Meetings       []Meeting        `json:"meetings"`
	Risks          []Risk           `json:"risks"`
	Issues         []Issue          `json:"issues"`
	Documents      []Document       `json:"documents"`
	Surveys        []Survey         `json:"surveys"`
	Reports        []Report         `json:"reports"`
	ChatMessages   []ChatMessage    `json:"chatMessages"`
	AuditLogs      []AuditLog       `json:"auditLogs"`
	CalendarEvents []CalendarEvent  `json:"calendarEvents"`
}

// ─── Research Member ──────────────────────────────────────
type ResearchMember struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatarUrl"`
	Department  string    `json:"department"`
	Phone       string    `json:"phone"`
	Location    string    `json:"location"`
	Since       string    `json:"since"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Proposal ─────────────────────────────────────────────
type Proposal struct {
	ID               string    `json:"id"`
	ResearchID       string    `json:"researchId"`
	Title            string    `json:"title"`
	Version          string    `json:"version"`
	Status           string    `json:"status"`
	Description      string    `json:"description"`
	DraftContent     string    `json:"draftContent"`
	Background       string    `json:"background"`
	Objectives       string    `json:"objectives"`
	Methodology      string    `json:"methodology"`
	TimelineDetails  string    `json:"timelineDetails"`
	BudgetDetails    string    `json:"budgetDetails"`
	ReviewerComments string    `json:"reviewerComments"`
	SubmittedBy      string    `json:"submittedBy"`
	ReviewedBy       string    `json:"reviewedBy"`
	ApprovedBy       string    `json:"approvedBy"`
	CreatedTime      time.Time `json:"createdTime"`
	UpdatedTime      time.Time `json:"updatedTime"`
}

// ─── Ethics Application ───────────────────────────────────
type EthicsApp struct {
	ID                string    `json:"id"`
	ResearchID        string    `json:"researchId"`
	IRBName           string    `json:"irbName"`
	Status            string    `json:"status"`
	SubmissionDate    string    `json:"submissionDate"`
	ApprovalDate      string    `json:"approvalDate"`
	ExpiryDate        string    `json:"expiryDate"`
	CertificateNumber string    `json:"certificateNumber"`
	Comments          string    `json:"comments"`
	AmendmentNotes    string    `json:"amendmentNotes"`
	RenewalNotes      string    `json:"renewalNotes"`
	ComplianceNotes   string    `json:"complianceNotes"`
	CreatedTime       time.Time `json:"createdTime"`
	UpdatedTime       time.Time `json:"updatedTime"`
}

// ─── Grant / Funding ──────────────────────────────────────
type Grant struct {
	ID                string    `json:"id"`
	ResearchID        string    `json:"researchId"`
	OpportunityName   string    `json:"opportunityName"`
	Status            string    `json:"status"`
	DonorName         string    `json:"donorName"`
	ContractNumber    string    `json:"contractNumber"`
	BudgetAllocated   float64   `json:"budgetAllocated"`
	Spent             float64   `json:"spent"`
	Currency          string    `json:"currency"`
	StartDate         string    `json:"startDate"`
	EndDate           string    `json:"endDate"`
	ReportingSchedule string    `json:"reportingSchedule"`
	Deliverables      string    `json:"deliverables"`
	Notes             string    `json:"notes"`
	CreatedTime       time.Time `json:"createdTime"`
	UpdatedTime       time.Time `json:"updatedTime"`
}

// ─── Literature Item ──────────────────────────────────────
type LiteratureItem struct {
	ID            string    `json:"id"`
	ResearchID    string    `json:"researchId"`
	Title         string    `json:"title"`
	Authors       string    `json:"authors"`
	Journal       string    `json:"journal"`
	DOI           string    `json:"doi"`
	PubYear       int       `json:"pubYear"`
	Citation      string    `json:"citation"`
	Keywords      string    `json:"keywords"`
	Category      string    `json:"category"`
	Tags          []string  `json:"tags"`
	Notes         string    `json:"notes"`
	URL           string    `json:"url"`
	ReadingStatus string    `json:"readingStatus"`
	CreatedTime   time.Time `json:"createdTime"`
}

// ─── Dataset ──────────────────────────────────────────────
type Dataset struct {
	ID               string    `json:"id"`
	ResearchID       string    `json:"researchId"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Version          string    `json:"version"`
	Status           string    `json:"status"`
	SourceType       string    `json:"sourceType"`
	CollectionMethod string    `json:"collectionMethod"`
	MetadataInfo     string    `json:"metadataInfo"`
	VariablesDict    string    `json:"variablesDict"`
	AccessLevel      string    `json:"accessLevel"`
	DownloadURL      string    `json:"downloadUrl"`
	StatcollectID    string    `json:"statcollectId"`
	CreatedTime      time.Time `json:"createdTime"`
	UpdatedTime      time.Time `json:"updatedTime"`
}

// ─── Publication ──────────────────────────────────────────
type Publication struct {
	ID                 string    `json:"id"`
	ResearchID         string    `json:"researchId"`
	Title              string    `json:"title"`
	PubType            string    `json:"pubType"`
	Authors            string    `json:"authors"`
	Journal            string    `json:"journal"`
	Status             string    `json:"status"`
	PeerReviewComments string    `json:"peerReviewComments"`
	RevisionHistory    string    `json:"revisionHistory"`
	DOI                string    `json:"doi"`
	AcceptanceDate     string    `json:"acceptanceDate"`
	PublishedDate      string    `json:"publishedDate"`
	Affiliation        string    `json:"affiliation"`
	CreatedTime        time.Time `json:"createdTime"`
	UpdatedTime        time.Time `json:"updatedTime"`
}

// ─── Research Task ────────────────────────────────────────
type ResearchTask struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	StartDate   string    `json:"startDate"`
	EndDate     string    `json:"endDate"`
	Progress    float64   `json:"progress"`
	AssignedTo  string    `json:"assignedTo"`
	CreatedTime time.Time `json:"createdTime"`
	UpdatedTime time.Time `json:"updatedTime"`
}

// ─── Meeting ──────────────────────────────────────────────
type Meeting struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Title       string    `json:"title"`
	DateTime    string    `json:"dateTime"`
	Location    string    `json:"location"`
	Agenda      string    `json:"agenda"`
	Decisions   []string  `json:"decisions"`
	Attendees   []string  `json:"attendees"`
	Status      string    `json:"status"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Risk ─────────────────────────────────────────────────
type Risk struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Probability string    `json:"probability"`
	Impact      string    `json:"impact"`
	Mitigation  string    `json:"mitigation"`
	Status      string    `json:"status"`
	Owner       string    `json:"owner"`
	DueDate     string    `json:"dueDate"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Issue ─────────────────────────────────────────────────
type Issue struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Owner       string    `json:"owner"`
	DueDate     string    `json:"dueDate"`
	Resolution  string    `json:"resolution"`
	CreatedTime time.Time `json:"createdTime"`
	UpdatedTime time.Time `json:"updatedTime"`
}

// ─── Document ─────────────────────────────────────────────
type Document struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Size        string    `json:"size"`
	UploadedBy  string    `json:"uploadedBy"`
	UploadedAt  time.Time `json:"uploadedAt"`
	URL         string    `json:"url"`
	Status      string    `json:"status"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Survey ───────────────────────────────────────────────
type Survey struct {
	ID           string    `json:"id"`
	ResearchID   string    `json:"researchId"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	TargetSample int       `json:"targetSample"`
	Submissions  int       `json:"submissions"`
	Progress     float64   `json:"progress"`
	CreatedTime  time.Time `json:"createdTime"`
}

// ─── Report ───────────────────────────────────────────────
type Report struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Format      string    `json:"format"`
	GeneratedBy string    `json:"generatedBy"`
	GeneratedAt time.Time `json:"generatedAt"`
	Content     string    `json:"content"`
	Status      string    `json:"status"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Calendar Event ───────────────────────────────────────
type CalendarEvent struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventType   string    `json:"eventType"`
	StartTime   string    `json:"startTime"`
	EndTime     string    `json:"endTime"`
	Location    string    `json:"location"`
	Attendees   []string  `json:"attendees"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Chat Message ─────────────────────────────────────────
type ChatMessage struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Sender      string    `json:"sender"`
	Channel     string    `json:"channel"`
	Message     string    `json:"message"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Audit Log ────────────────────────────────────────────
type AuditLog struct {
	ID          string    `json:"id"`
	ResearchID  string    `json:"researchId"`
	Action      string    `json:"action"`
	PerformedBy string    `json:"performedBy"`
	Details     string    `json:"details"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Dashboard Summary ────────────────────────────────────
type DashboardSummary struct {
	TotalResearch     int          `json:"totalResearch"`
	ActiveStudies     int          `json:"activeStudies"`
	ProposalsPending  int          `json:"proposalsPending"`
	EthicsPending     int          `json:"ethicsPending"`
	TotalPublications int          `json:"totalPublications"`
	TotalDatasets     int          `json:"totalDatasets"`
	TotalLiterature   int          `json:"totalLiterature"`
	TotalBudget       float64      `json:"totalBudget"`
	TotalSpent        float64      `json:"totalSpent"`
	AvgProgress       float64      `json:"avgProgress"`
	ByStage           []StageCount `json:"byStage"`
	RecentActivity    []AuditLog   `json:"recentActivity"`
}

type StageCount struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
}

// ─── Request Bodies ───────────────────────────────────────
type CreateResearchRequest struct {
	Code                  string   `json:"code" binding:"required"`
	Name                  string   `json:"name" binding:"required"`
	Description           string   `json:"description"`
	Type                  string   `json:"type"`
	PrincipalInvestigator string   `json:"principalInvestigator"`
	Owner                 string   `json:"owner"`
	Organisation          string   `json:"organisation"`
	Portfolio             string   `json:"portfolio"`
	Programme             string   `json:"programme"`
	StartDate             string   `json:"startDate"`
	EndDate               string   `json:"endDate"`
	TargetGeo             string   `json:"targetGeo"`
	BudgetTotal           float64  `json:"budgetTotal"`
	Tags                  []string `json:"tags"`
	PmsProjectID          string   `json:"pmsProjectId"`
}

type UpdateResearchRequest struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Type                  string   `json:"type"`
	PrincipalInvestigator string   `json:"principalInvestigator"`
	Owner                 string   `json:"owner"`
	Organisation          string   `json:"organisation"`
	Portfolio             string   `json:"portfolio"`
	Programme             string   `json:"programme"`
	StartDate             string   `json:"startDate"`
	EndDate               string   `json:"endDate"`
	TargetGeo             string   `json:"targetGeo"`
	BudgetTotal           float64  `json:"budgetTotal"`
	Progress              float64  `json:"progress"`
	Tags                  []string `json:"tags"`
}

type StageTransitionRequest struct {
	Stage string `json:"stage" binding:"required"`
	User  string `json:"user"`
}
