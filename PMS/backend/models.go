package main

import (
	"database/sql/driver"
	"strings"
	"time"
)

type LifecycleStage string

const (
	StageConcept        LifecycleStage = "Concept"
	StageProposal       LifecycleStage = "Proposal"
	StagePlanning       LifecycleStage = "Planning"
	StageApproval       LifecycleStage = "Approval"
	StageFunding        LifecycleStage = "Funding"
	StageImplementation LifecycleStage = "Implementation"
	StageMonitoring     LifecycleStage = "Monitoring"
	StageEvaluation     LifecycleStage = "Evaluation"
	StageClosure        LifecycleStage = "Closure"
	StageArchive        LifecycleStage = "Archive"
)

// ─── Enterprise Hierarchy ───────────────────────────────────────────

type Organization struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	ParentID    string    `json:"parentId,omitempty"`
	CreatedTime time.Time `json:"createdTime"`
}

type Portfolio struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"orgId"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Owner       string    `json:"owner"`
	CreatedTime time.Time `json:"createdTime"`
}

type Programme struct {
	ID          string    `json:"id"`
	PortfolioID string    `json:"portfolioId"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Owner       string    `json:"owner"`
	BudgetTotal float64   `json:"budgetTotal"`
	CreatedTime time.Time `json:"createdTime"`
}

type Component struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	ParentID    string    `json:"parentId,omitempty"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Order       int       `json:"order"`
	CreatedTime time.Time `json:"createdTime"`
}

type Activity struct {
	ID          string    `json:"id"`
	ComponentID string    `json:"componentId"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	StartDate   string    `json:"startDate"`
	EndDate     string    `json:"endDate"`
	Status      string    `json:"status"`
	Progress    float64   `json:"progress"`
	CreatedTime time.Time `json:"createdTime"`
}

type Deliverable struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	TaskID      string    `json:"taskId,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	DueDate     string    `json:"dueDate"`
	Owner       string    `json:"owner"`
	CreatedTime time.Time `json:"createdTime"`
}

type Milestone struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	DueDate     string    `json:"dueDate"`
	Status      string    `json:"status"`
	Owner       string    `json:"owner"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Project ────────────────────────────────────────────────────────

type Project struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Stage       LifecycleStage `json:"stage"`
	Progress    float64        `json:"progress"`
	Org         string         `json:"org"`
	Portfolio   string         `json:"portfolio"`
	Programme   string         `json:"programme"`
	Owner       string         `json:"owner"`
	StartDate   string         `json:"startDate"`
	EndDate     string         `json:"endDate"`
	TargetGeo   string         `json:"targetGeo"`
	Tags        StringArray    `json:"tags"`
	BudgetTotal float64        `json:"budgetTotal"`
	SpentTotal  float64        `json:"spentTotal"`
	RisksCount  int            `json:"risksCount"`
	IssuesCount int            `json:"issuesCount"`
	CreatedTime time.Time      `json:"createdTime"`
	UpdatedTime time.Time      `json:"updatedTime"`
	WorkspaceID string         `json:"workspaceId,omitempty"`
}

type ProjectMember struct {
	ID         string `json:"id"`
	ProjectID  string `json:"projectId"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Email      string `json:"email"`
	AvatarUrl  string `json:"avatarUrl"`
	Department string `json:"department,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Location   string `json:"location,omitempty"`
	Since      string `json:"since,omitempty"`
}

type Task struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"projectId"`
	ParentID      string   `json:"parentId,omitempty"`
	WBSCode       string   `json:"wbsCode"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	StartDate     string   `json:"startDate"`
	EndDate       string   `json:"endDate"`
	Progress      float64  `json:"progress"`
	AssignedTo    string   `json:"assignedTo"`
	Dependencies  []string `json:"dependencies"`
	IsMilestone   bool     `json:"isMilestone,omitempty"`
	IsDeliverable bool     `json:"isDeliverable,omitempty"`
}

// ─── Budget & Finance ──────────────────────────────────────────────

type BudgetLine struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"projectId"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Source      string  `json:"source"`
	Amount      float64 `json:"amount"`
	Spent       float64 `json:"spent"`
}

type FundingSource struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Donor       string    `json:"donor"`
	Amount      float64   `json:"amount"`
	Received    float64   `json:"received"`
	Currency    string    `json:"currency"`
	StartDate   string    `json:"startDate"`
	EndDate     string    `json:"endDate"`
	Status      string    `json:"status"`
	CreatedTime time.Time `json:"createdTime"`
}

type CostCentre struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Budget      float64   `json:"budget"`
	Spent       float64   `json:"spent"`
	CreatedTime time.Time `json:"createdTime"`
}

type BudgetRevision struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Version     int       `json:"version"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	ApprovedBy  string    `json:"approvedBy"`
	ApprovedAt  time.Time `json:"approvedAt"`
	Status      string    `json:"status"`
	CreatedTime time.Time `json:"createdTime"`
}

type ProcurementReference struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Reference   string    `json:"reference"`
	Description string    `json:"description"`
	Vendor      string    `json:"vendor"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Risk & Issue Management ───────────────────────────────────────

type Risk struct {
	ID          string `json:"id"`
	ProjectID   string `json:"projectId"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Probability string `json:"probability"`
	Impact      string `json:"impact"`
	Mitigation  string `json:"mitigation"`
	Status      string `json:"status"`
	Owner       string `json:"owner"`
	DueDate     string `json:"dueDate,omitempty"`
}

type Issue struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Owner       string    `json:"owner"`
	DueDate     string    `json:"dueDate"`
	Resolution  string    `json:"resolution,omitempty"`
	CreatedTime time.Time `json:"createdTime"`
	UpdatedTime time.Time `json:"updatedTime"`
}

type Assumption struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	Owner       string    `json:"owner"`
	CreatedTime time.Time `json:"createdTime"`
}

type LessonLearned struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Impact      string    `json:"impact"`
	Owner       string    `json:"owner"`
	CreatedTime time.Time `json:"createdTime"`
}

type CorrectiveAction struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	IssueID     string    `json:"issueId,omitempty"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Owner       string    `json:"owner"`
	DueDate     string    `json:"dueDate"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Documents ─────────────────────────────────────────────────────

type Document struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"projectId"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Size       string    `json:"size"`
	UploadedBy string    `json:"uploadedBy"`
	UploadedAt time.Time `json:"uploadedAt"`
	Url        string    `json:"url"`
	Status     string    `json:"status,omitempty"`
}

// ─── Meetings ──────────────────────────────────────────────────────

type Meeting struct {
	ID          string   `json:"id"`
	ProjectID   string   `json:"projectId"`
	Title       string   `json:"title"`
	DateTime    string   `json:"dateTime"`
	Location    string   `json:"location"`
	Attendees   []string `json:"attendees"`
	Agenda      string   `json:"agenda"`
	Minutes     string   `json:"minutes"`
	ActionItems []string `json:"actionItems"`
	Status      string   `json:"status,omitempty"`
	Decisions   []string `json:"decisions,omitempty"`
}

// ─── Surveys ───────────────────────────────────────────────────────

type Survey struct {
	ID           string  `json:"id"`
	ProjectID    string  `json:"projectId"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	TargetSample int     `json:"targetSample"`
	Submissions  int     `json:"submissions"`
	Progress     float64 `json:"progress"`
}

// ─── Chat ──────────────────────────────────────────────────────────

type ChatMessage struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Channel   string    `json:"channel"`
	Sender    string    `json:"sender"`
	Role      string    `json:"role"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ─── HelpDesk ──────────────────────────────────────────────────────

type HelpDeskTicket struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ─── Workflow & Permissions ────────────────────────────────────────

type WorkflowRule struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	FromStage    string      `json:"fromStage"`
	ToStage      string      `json:"toStage"`
	RequiredRole string      `json:"requiredRole"`
	AutoActions  StringArray `json:"autoActions"`
	Enabled      bool        `json:"enabled"`
	CreatedTime  time.Time   `json:"createdTime"`
}

type Permission struct {
	ID          string    `json:"id"`
	Role        string    `json:"role"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Audit Log ─────────────────────────────────────────────────────

type AuditLog struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId,omitempty"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entityId,omitempty"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ─── Reports ───────────────────────────────────────────────────────

type Report struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Format      string    `json:"format"`
	GeneratedBy string    `json:"generatedBy"`
	GeneratedAt time.Time `json:"generatedAt"`
	Content     string    `json:"content,omitempty"`
	Status      string    `json:"status"`
}

// ─── Calendar Events ───────────────────────────────────────────────

type CalendarEvent struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventType   string    `json:"eventType"`
	StartTime   string    `json:"startTime"`
	EndTime     string    `json:"endTime"`
	Location    string    `json:"location"`
	Attendees   []string  `json:"attendees"`
	CreatedTime time.Time `json:"createdTime"`
}

// ─── Search Results ────────────────────────────────────────────────

type SearchResult struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	ProjectID   string `json:"projectId,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Meta        string `json:"meta,omitempty"`
}

// NullString handles NULL string values from database
type NullString struct {
	String string
	Valid  bool
}

func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.String = ""
		ns.Valid = false
		return nil
	}
	ns.String = value.(string)
	ns.Valid = true
	return nil
}

// StringArray handles PostgreSQL text array
type StringArray []string

func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = nil
		return nil
	}

	var raw string
	switch v := value.(type) {
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return nil
	}

	// Parse PostgreSQL array format: {elem1,elem2,"elem with spaces"}
	raw = strings.TrimPrefix(raw, "{")
	raw = strings.TrimSuffix(raw, "}")

	if raw == "" {
		*sa = []string{}
		return nil
	}

	var arr []string
	var current strings.Builder
	inQuotes := false
	for _, r := range raw {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			arr = append(arr, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	arr = append(arr, current.String())

	*sa = arr
	return nil
}

func (sa StringArray) Value() (driver.Value, error) {
	if sa == nil {
		return nil, nil
	}
	return "{" + strings.Join(sa, ",") + "}", nil
}

// ─── Phase 4 Extensions: LogFrame, Theory of Change, Donors ─────────

type LogFrame struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"projectId"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Items       []LogFrameItem `json:"items,omitempty"`
	CreatedTime time.Time      `json:"createdTime"`
}

type LogFrameItem struct {
	ID          string      `json:"id"`
	LogFrameID  string      `json:"logframeId"`
	Level       string      `json:"level"` // Goal, Outcome, Output, Activity
	Code        string      `json:"code"`
	Description string      `json:"description"`
	Indicators  StringArray `json:"indicators"`
	MeansOfVer  StringArray `json:"meansOfVerification"`
	Assumptions StringArray `json:"assumptions"`
	CreatedTime time.Time   `json:"createdTime"`
}

type TheoryOfChange struct {
	ID                string      `json:"id"`
	ProjectID         string      `json:"projectId"`
	Title             string      `json:"title"`
	Narrative         string      `json:"narrative"`
	Inputs            StringArray `json:"inputs"`
	Activities        StringArray `json:"activities"`
	Outputs           StringArray `json:"outputs"`
	ShortTermOutcomes StringArray `json:"shortTermOutcomes"`
	LongTermOutcomes  StringArray `json:"longTermOutcomes"`
	Impact            StringArray `json:"impact"`
	Assumptions       StringArray `json:"assumptions"`
	CreatedTime       time.Time   `json:"createdTime"`
}

type Donor struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	Type          string    `json:"type"` // Bilateral, Multilateral, Foundation, NGO, Private
	ContactPerson string    `json:"contactPerson"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	Website       string    `json:"website"`
	TotalFunding  float64   `json:"totalFunding"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedTime   time.Time `json:"createdTime"`
}
