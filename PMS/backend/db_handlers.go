package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func generateUUID() string {
	return uuid.New().String()
}

// ═══════════════════════════════════════════════════════════════════
// PROJECTS
// ═══════════════════════════════════════════════════════════════════

func dbGetProjects(c *gin.Context) {
	rows, err := DB.Query(`
		SELECT id, code, name, description, stage, progress, org, portfolio, programme,
		       owner, start_date, end_date, target_geo, tags, budget_total, spent_total,
		       risks_count, issues_count, created_time, updated_time, COALESCE(workspace_id, '')
		FROM pms.projects
		WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)
		ORDER BY created_time DESC
	`, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.Stage, &p.Progress,
			&p.Org, &p.Portfolio, &p.Programme, &p.Owner, &p.StartDate, &p.EndDate,
			&p.TargetGeo, &p.Tags, &p.BudgetTotal, &p.SpentTotal, &p.RisksCount, &p.IssuesCount, &p.CreatedTime, &p.UpdatedTime, &p.WorkspaceID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		projects = append(projects, p)
	}

	c.JSON(200, projects)
}

func dbGetProjectWorkspace(c *gin.Context) {
	id := c.Param("id")

	// Get project
	var p Project
	err := DB.QueryRow(`
		SELECT id, code, name, description, stage, progress, org, portfolio, programme,
		       owner, start_date, end_date, target_geo, tags, budget_total, spent_total,
		       risks_count, issues_count, created_time, updated_time, COALESCE(workspace_id, '')
		FROM pms.projects
		WHERE id = $1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, id, workspaceIDContext(c)).Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.Stage, &p.Progress,
		&p.Org, &p.Portfolio, &p.Programme, &p.Owner, &p.StartDate, &p.EndDate,
		&p.TargetGeo, &p.Tags, &p.BudgetTotal, &p.SpentTotal, &p.RisksCount, &p.IssuesCount, &p.CreatedTime, &p.UpdatedTime, &p.WorkspaceID)

	if err == sql.ErrNoRows {
		c.JSON(404, gin.H{"error": "Project not found"})
		return
	} else if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Get all related data
	members, _ := dbGetMembers(id)
	tasks, _ := dbGetTasks(id)
	budgets, _ := dbGetBudgets(id)
	risks, _ := dbGetRisks(id)
	documents, _ := dbGetDocuments(id)
	meetings, _ := dbGetMeetings(id)
	surveys, _ := dbGetSurveys(id)
	chats, chatErr := loadProjectDiscussion(c, p)
	if chats == nil {
		chats = []ChatMessage{}
	}
	helpdesk, _ := dbGetHelpdesk(id)
	components, _ := dbGetComponentsByProject(id)
	activities, _ := dbGetActivitiesByProject(id)
	deliverables, _ := dbGetDeliverablesByProject(id)
	milestones, _ := dbGetMilestonesByProject(id)
	issues, _ := dbGetIssuesByProject(id)
	assumptions, _ := dbGetAssumptionsByProject(id)
	lessons, _ := dbGetLessonsByProject(id)
	correctiveActions, _ := dbGetCorrectiveActionsByProject(id)
	fundingSources, _ := dbGetFundingSourcesByProject(id)
	costCentres, _ := dbGetCostCentresByProject(id)
	budgetRevisions, _ := dbGetBudgetRevisionsByProject(id)
	procurementRefs, _ := dbGetProcurementRefsByProject(id)
	auditLogs, _ := dbGetAuditLogsByProject(id)
	calendarEvents, _ := dbGetCalendarByProject(id)
	reports, _ := dbGetReportsByProject(id)

	response := map[string]interface{}{
		"project":              p,
		"members":              members,
		"tasks":                tasks,
		"budgets":              budgets,
		"risks":                risks,
		"documents":            documents,
		"meetings":             meetings,
		"surveys":              surveys,
		"chats":                chats,
		"helpdesk":             helpdesk,
		"components":           components,
		"activities":           activities,
		"deliverables":         deliverables,
		"milestones":           milestones,
		"issues":               issues,
		"assumptions":          assumptions,
		"lessons":              lessons,
		"correctiveActions":    correctiveActions,
		"fundingSources":       fundingSources,
		"costCentres":          costCentres,
		"budgetRevisions":      budgetRevisions,
		"procurementRefs":      procurementRefs,
		"auditLogs":            auditLogs,
		"calendarEvents":       calendarEvents,
		"reports":              reports,
		"chatIntegrationReady": chatErr == nil,
	}

	c.JSON(200, response)
}

func dbCreateProject(c *gin.Context) {
	var req struct {
		Name        string  `json:"name"`
		Code        string  `json:"code"`
		Description string  `json:"description"`
		Owner       string  `json:"owner"`
		Org         string  `json:"org"`
		Portfolio   string  `json:"portfolio"`
		Programme   string  `json:"programme"`
		StartDate   string  `json:"startDate"`
		EndDate     string  `json:"endDate"`
		TargetGeo   string  `json:"targetGeo"`
		BudgetTotal float64 `json:"budgetTotal"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	id := fmt.Sprintf("proj-%d", rand.Intn(100000))
	now := time.Now()

	_, err := DB.Exec(`
		INSERT INTO pms.projects (id, code, name, description, stage, progress, org, portfolio, programme, owner, start_date, end_date, target_geo, tags, budget_total, spent_total, risks_count, issues_count, created_time, updated_time, workspace_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NULLIF($21, ''))
	`, id, req.Code, req.Name, req.Description, "Concept", 0.0, req.Org, req.Portfolio, req.Programme,
		req.Owner, req.StartDate, req.EndDate, req.TargetGeo, pq.Array([]string{"New", "StatGate"}),
		req.BudgetTotal, 0.0, 0, 0, now, now, workspaceIDContext(c))

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Initialize default workspace assets
	dbInitDefaultWorkspace(id, req.Owner, req.Name, req.StartDate, req.BudgetTotal)

	// Log audit
	dbLogAudit(id, req.Owner, "CREATE", "project", id, "Project created and workspace initialized")

	publishEvent("project.created", "project", id, map[string]interface{}{
		"name":  req.Name,
		"owner": req.Owner,
		"code":  req.Code,
		"org":   req.Org,
	})

	c.JSON(201, gin.H{"id": id, "message": "Project created successfully"})
}

func dbUpdateProject(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Owner       string  `json:"owner"`
		Org         string  `json:"org"`
		Portfolio   string  `json:"portfolio"`
		Programme   string  `json:"programme"`
		StartDate   string  `json:"startDate"`
		EndDate     string  `json:"endDate"`
		TargetGeo   string  `json:"targetGeo"`
		BudgetTotal float64 `json:"budgetTotal"`
		Progress    float64 `json:"progress"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`
		UPDATE pms.projects SET
			name = COALESCE(NULLIF($1, ''), name),
			description = COALESCE(NULLIF($2, ''), description),
			owner = COALESCE(NULLIF($3, ''), owner),
			org = COALESCE(NULLIF($4, ''), org),
			portfolio = COALESCE(NULLIF($5, ''), portfolio),
			programme = COALESCE(NULLIF($6, ''), programme),
			start_date = COALESCE(NULLIF($7, ''), start_date),
			end_date = COALESCE(NULLIF($8, ''), end_date),
			target_geo = COALESCE(NULLIF($9, ''), target_geo),
			budget_total = CASE WHEN $10 > 0 THEN $10 ELSE budget_total END,
			progress = CASE WHEN $11 >= 0 THEN $11 ELSE progress END,
			updated_time = $12
		WHERE id = $13 AND (workspace_id = NULLIF($14, '') OR NULLIF($14, '') IS NULL)
	`, req.Name, req.Description, req.Owner, req.Org, req.Portfolio, req.Programme,
		req.StartDate, req.EndDate, req.TargetGeo, req.BudgetTotal, req.Progress, time.Now(), id, workspaceIDContext(c))

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	dbLogAudit(id, req.Owner, "UPDATE", "project", id, "Project details updated")

	publishEvent("project.updated", "project", id, map[string]interface{}{
		"name":     req.Name,
		"progress": req.Progress,
		"owner":    req.Owner,
	})

	c.JSON(200, gin.H{"message": "Project updated successfully"})
}

func dbDeleteProject(c *gin.Context) {
	id := c.Param("id")

	_, err := DB.Exec(`DELETE FROM pms.projects WHERE id = $1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)`, id, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Project deleted successfully"})
}

func dbInitDefaultWorkspace(projectID, owner, name, startDate string, budgetTotal float64) {
	now := time.Now()

	// Add owner as member
	DB.Exec(`INSERT INTO pms.project_members (id, project_id, name, role, email, avatar_url, department, phone, location, since)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		fmt.Sprintf("m-auto-%d", rand.Intn(100000)), projectID, owner, "Project Manager",
		"manager@statgate.gov", "https://api.dicebear.com/7.x/adventurer/svg?seed=pm",
		"Project Leadership", "+256 700 000 000", "Kampala HQ", now)

	// Add initial task
	DB.Exec(`INSERT INTO pms.tasks (id, project_id, wbs_code, title, description, status, priority, start_date, end_date, progress, assigned_to, created_time, updated_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		fmt.Sprintf("t-%d", rand.Intn(100000)), projectID, "1.1", "Initial Project Briefing",
		"Establish goals, context and assign team roles.", "Todo", "Medium", startDate, startDate, 0, owner, now, now)

	// Add budget lines
	DB.Exec(`INSERT INTO pms.budget_lines (id, project_id, category, description, source, amount, spent, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fmt.Sprintf("b-%d", rand.Intn(100000)), projectID, "Personnel", "Project Coordinator allowance",
		"Internal Fund", budgetTotal*0.6, 0, now)

	DB.Exec(`INSERT INTO pms.budget_lines (id, project_id, category, description, source, amount, spent, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fmt.Sprintf("b-%d", rand.Intn(100000)), projectID, "Travel", "Site planning travels",
		"Internal Fund", budgetTotal*0.4, 0, now)

	// Add survey
	DB.Exec(`INSERT INTO pms.surveys (id, project_id, name, status, target_sample, submissions, progress, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fmt.Sprintf("s-%d", rand.Intn(100000)), projectID, fmt.Sprintf("%s Field Questionnaire", name),
		"Template", 1000, 0, 0.0, now)

	// Add default component
	DB.Exec(`INSERT INTO pms.components (id, project_id, name, code, description, type, sort_order, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fmt.Sprintf("c-%d", rand.Intn(100000)), projectID, "Core Activities", "C1",
		"Primary project activities and deliverables", "Core", 1, now)

	// Add default milestone
	DB.Exec(`INSERT INTO pms.milestones (id, project_id, name, description, due_date, status, owner, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fmt.Sprintf("ms-%d", rand.Intn(100000)), projectID, "Project Kickoff",
		"Initial project kickoff and team briefing", startDate, "Pending", owner, now)

	// Add welcome chat message
	DB.Exec(`INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		fmt.Sprintf("ch-%d", rand.Intn(100000)), projectID, "general", "StatGate System", "Workflow Engine",
		fmt.Sprintf("Workspace for project [%s] automatically initialized successfully.", name), now)

	// Add announcement
	DB.Exec(`INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		fmt.Sprintf("ch-%d", rand.Intn(100000)), projectID, "announcements", "StatGate System", "Workflow Engine",
		"Project workspace, dashboard, calendar, and reporting modules have been automatically provisioned.", now)
}

func dbUpdateProjectStage(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Stage string `json:"stage"`
		User  string `json:"user"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.projects SET stage = $1, updated_time = $2 WHERE id = $3 AND (workspace_id = NULLIF($4, '') OR NULLIF($4, '') IS NULL)`,
		req.Stage, time.Now(), id, workspaceIDContext(c))

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Add transition notification to chat
	sysMsg := fmt.Sprintf("Project status updated to [%s] by workflow engine.", req.Stage)
	DB.Exec(`INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		fmt.Sprintf("ch-%d", rand.Intn(100000)), id, "announcements", "Workflow Engine", "System Announcement",
		sysMsg, time.Now())

	// Log audit
	user := req.User
	if user == "" {
		user = "System"
	}
	dbLogAudit(id, user, "STAGE_CHANGE", "project", id, fmt.Sprintf("Project stage changed to %s", req.Stage))

	// Execute workflow automation based on stage
	dbExecuteWorkflowAutomation(id, req.Stage)

	c.JSON(200, gin.H{"message": "Stage updated", "stage": req.Stage})
}

func dbExecuteWorkflowAutomation(projectID, stage string) {
	now := time.Now()

	switch stage {
	case "Approval":
		// Create approval notification
		DB.Exec(`INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			fmt.Sprintf("ch-%d", rand.Intn(100000)), projectID, "announcements", "Workflow Engine", "System Announcement",
			"Project submitted for formal approval. Programme Director notified.", now)
	case "Funding":
		// Create default funding source
		DB.Exec(`INSERT INTO pms.funding_sources (id, project_id, name, type, donor, amount, received, currency, status, created_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			fmt.Sprintf("fs-%d", rand.Intn(100000)), projectID, "Primary Funding Source", "Government",
			"State Fund", 0, 0, "USD", "Pending", now)
	case "Implementation":
		// Create dashboard and analytics notification
		DB.Exec(`INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			fmt.Sprintf("ch-%d", rand.Intn(100000)), projectID, "announcements", "Workflow Engine", "System Announcement",
			"Project dashboard, analytics, and reporting modules activated. Ready for implementation.", now)
	case "Closure":
		// Create final report entry
		DB.Exec(`INSERT INTO pms.reports (id, project_id, title, type, format, generated_by, generated_at, content, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			fmt.Sprintf("rp-%d", rand.Intn(100000)), projectID, "Project Final Report", "Final",
			"PDF", "Workflow Engine", now, "Final project report generated automatically", "Generated")
	case "Archive":
		DB.Exec(`INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			fmt.Sprintf("ch-%d", rand.Intn(100000)), projectID, "announcements", "Workflow Engine", "System Announcement",
			"Project archived. All records and documents have been preserved.", now)
	}
}

// ═══════════════════════════════════════════════════════════════════
// ENTERPRISE HIERARCHY
// ═══════════════════════════════════════════════════════════════════

func dbGetOrganizations(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, name, code, description, type, parent_id, created_time FROM pms.organizations ORDER BY name`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		var parentID sql.NullString
		rows.Scan(&o.ID, &o.Name, &o.Code, &o.Description, &o.Type, &parentID, &o.CreatedTime)
		o.ParentID = parentID.String
		orgs = append(orgs, o)
	}
	c.JSON(200, orgs)
}

func dbCreateOrganization(c *gin.Context) {
	var req Organization
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("org-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.organizations (id, name, code, description, type, parent_id, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		req.ID, req.Name, req.Code, req.Description, req.Type, req.ParentID, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetPortfolios(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, org_id, name, code, description, owner, created_time FROM pms.portfolios ORDER BY name`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var portfolios []Portfolio
	for rows.Next() {
		var p Portfolio
		rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Code, &p.Description, &p.Owner, &p.CreatedTime)
		portfolios = append(portfolios, p)
	}
	c.JSON(200, portfolios)
}

func dbCreatePortfolio(c *gin.Context) {
	var req Portfolio
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("pf-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.portfolios (id, org_id, name, code, description, owner, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		req.ID, req.OrgID, req.Name, req.Code, req.Description, req.Owner, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetProgrammes(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, portfolio_id, name, code, description, owner, budget_total, created_time FROM pms.programmes ORDER BY name`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var programmes []Programme
	for rows.Next() {
		var p Programme
		rows.Scan(&p.ID, &p.PortfolioID, &p.Name, &p.Code, &p.Description, &p.Owner, &p.BudgetTotal, &p.CreatedTime)
		programmes = append(programmes, p)
	}
	c.JSON(200, programmes)
}

func dbCreateProgramme(c *gin.Context) {
	var req Programme
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("pg-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.programmes (id, portfolio_id, name, code, description, owner, budget_total, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.PortfolioID, req.Name, req.Code, req.Description, req.Owner, req.BudgetTotal, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetComponents(c *gin.Context) {
	projectID := c.Param("id")
	components, err := dbGetComponentsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, components)
}

func dbGetComponentsByProject(projectID string) ([]Component, error) {
	rows, err := DB.Query(`SELECT id, project_id, parent_id, name, code, description, type, sort_order, created_time FROM pms.components WHERE project_id = $1 ORDER BY sort_order`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []Component
	for rows.Next() {
		var comp Component
		var parentID sql.NullString
		rows.Scan(&comp.ID, &comp.ProjectID, &parentID, &comp.Name, &comp.Code, &comp.Description, &comp.Type, &comp.Order, &comp.CreatedTime)
		comp.ParentID = parentID.String
		components = append(components, comp)
	}
	return components, nil
}

func dbCreateComponent(c *gin.Context) {
	var req Component
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("c-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.components (id, project_id, parent_id, name, code, description, type, sort_order, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.ID, req.ProjectID, req.ParentID, req.Name, req.Code, req.Description, req.Type, req.Order, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetActivities(c *gin.Context) {
	projectID := c.Param("id")
	activities, err := dbGetActivitiesByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, activities)
}

func dbGetActivitiesByProject(projectID string) ([]Activity, error) {
	rows, err := DB.Query(`SELECT id, component_id, project_id, name, code, description, start_date, end_date, status, progress, created_time FROM pms.activities WHERE project_id = $1 ORDER BY created_time`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		rows.Scan(&a.ID, &a.ComponentID, &a.ProjectID, &a.Name, &a.Code, &a.Description, &a.StartDate, &a.EndDate, &a.Status, &a.Progress, &a.CreatedTime)
		activities = append(activities, a)
	}
	return activities, nil
}

func dbCreateActivity(c *gin.Context) {
	var req Activity
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("a-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.activities (id, component_id, project_id, name, code, description, start_date, end_date, status, progress, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		req.ID, req.ComponentID, req.ProjectID, req.Name, req.Code, req.Description, req.StartDate, req.EndDate, req.Status, req.Progress, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetDeliverables(c *gin.Context) {
	projectID := c.Param("id")
	deliverables, err := dbGetDeliverablesByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, deliverables)
}

func dbGetDeliverablesByProject(projectID string) ([]Deliverable, error) {
	rows, err := DB.Query(`SELECT id, project_id, task_id, name, description, type, status, due_date, owner, created_time FROM pms.deliverables WHERE project_id = $1 ORDER BY due_date`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliverables []Deliverable
	for rows.Next() {
		var d Deliverable
		var taskID sql.NullString
		rows.Scan(&d.ID, &d.ProjectID, &taskID, &d.Name, &d.Description, &d.Type, &d.Status, &d.DueDate, &d.Owner, &d.CreatedTime)
		d.TaskID = taskID.String
		deliverables = append(deliverables, d)
	}
	return deliverables, nil
}

func dbCreateDeliverable(c *gin.Context) {
	var req Deliverable
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("d-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.deliverables (id, project_id, task_id, name, description, type, status, due_date, owner, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		req.ID, req.ProjectID, req.TaskID, req.Name, req.Description, req.Type, req.Status, req.DueDate, req.Owner, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetMilestones(c *gin.Context) {
	projectID := c.Param("id")
	milestones, err := dbGetMilestonesByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, milestones)
}

func dbGetMilestonesByProject(projectID string) ([]Milestone, error) {
	rows, err := DB.Query(`SELECT id, project_id, name, description, due_date, status, owner, created_time FROM pms.milestones WHERE project_id = $1 ORDER BY due_date`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var milestones []Milestone
	for rows.Next() {
		var m Milestone
		rows.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Description, &m.DueDate, &m.Status, &m.Owner, &m.CreatedTime)
		milestones = append(milestones, m)
	}
	return milestones, nil
}

func dbCreateMilestone(c *gin.Context) {
	var req Milestone
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("ms-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.milestones (id, project_id, name, description, due_date, status, owner, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.Name, req.Description, req.DueDate, req.Status, req.Owner, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

// ═══════════════════════════════════════════════════════════════════
// TASKS
// ═══════════════════════════════════════════════════════════════════

func dbCreateTask(c *gin.Context) {
	var req Task
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.ID = fmt.Sprintf("t-%d", rand.Intn(100000))
	if req.Status == "" {
		req.Status = "Todo"
	}

	_, err := DB.Exec(`INSERT INTO pms.tasks (id, project_id, parent_id, wbs_code, title, description, status, priority, start_date, end_date, progress, assigned_to, dependencies, is_milestone, is_deliverable, created_time, updated_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		req.ID, req.ProjectID, req.ParentID, req.WBSCode, req.Title, req.Description,
		req.Status, req.Priority, req.StartDate, req.EndDate, req.Progress, req.AssignedTo,
		req.Dependencies, req.IsMilestone, req.IsDeliverable, time.Now(), time.Now())

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	publishEvent("task.created", "task", req.ID, map[string]interface{}{
		"project_id":  req.ProjectID,
		"title":       req.Title,
		"status":      req.Status,
		"assigned_to": req.AssignedTo,
	})

	c.JSON(201, req)
}

func dbUpdateTask(c *gin.Context) {
	taskID := c.Param("id")

	var req Task
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.tasks SET
		title = COALESCE(NULLIF($1, ''), title),
		description = COALESCE(NULLIF($2, ''), description),
		status = COALESCE(NULLIF($3, ''), status),
		priority = COALESCE(NULLIF($4, ''), priority),
		start_date = COALESCE(NULLIF($5, ''), start_date),
		end_date = COALESCE(NULLIF($6, ''), end_date),
		progress = CASE WHEN $7 >= 0 THEN $7 ELSE progress END,
		assigned_to = COALESCE(NULLIF($8, ''), assigned_to),
		updated_time = $9
		WHERE id = $10`,
		req.Title, req.Description, req.Status, req.Priority, req.StartDate, req.EndDate,
		req.Progress, req.AssignedTo, time.Now(), taskID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Task updated successfully"})
}

func dbDeleteTask(c *gin.Context) {
	taskID := c.Param("id")

	_, err := DB.Exec(`DELETE FROM pms.tasks WHERE id = $1`, taskID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Task deleted successfully"})
}

func dbUpdateTaskStatus(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		Status   string  `json:"status"`
		Progress float64 `json:"progress"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var updatedTask Task
	err := DB.QueryRow(`UPDATE pms.tasks SET status = $1, progress = $2, updated_time = $3 WHERE id = $4
		RETURNING id, project_id, parent_id, wbs_code, title, description, status, priority, start_date, end_date, progress, assigned_to, dependencies`,
		req.Status, req.Progress, time.Now(), taskID).Scan(
		&updatedTask.ID, &updatedTask.ProjectID, &updatedTask.ParentID, &updatedTask.WBSCode,
		&updatedTask.Title, &updatedTask.Description, &updatedTask.Status, &updatedTask.Priority,
		&updatedTask.StartDate, &updatedTask.EndDate, &updatedTask.Progress, &updatedTask.AssignedTo,
		&updatedTask.Dependencies)

	if err == sql.ErrNoRows {
		c.JSON(404, gin.H{"error": "Task not found"})
		return
	} else if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	publishEvent("task.updated", "task", updatedTask.ID, map[string]interface{}{
		"project_id":  updatedTask.ProjectID,
		"title":       updatedTask.Title,
		"status":      updatedTask.Status,
		"progress":    updatedTask.Progress,
		"assigned_to": updatedTask.AssignedTo,
	})

	c.JSON(200, updatedTask)
}

// ═══════════════════════════════════════════════════════════════════
// TEAM MEMBERS
// ═══════════════════════════════════════════════════════════════════

func dbAddMember(c *gin.Context) {
	var req ProjectMember
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.ID = fmt.Sprintf("m-%d", rand.Intn(100000))

	_, err := DB.Exec(`INSERT INTO pms.project_members (id, project_id, name, role, email, avatar_url, department, phone, location, since, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		req.ID, req.ProjectID, req.Name, req.Role, req.Email, req.AvatarUrl,
		req.Department, req.Phone, req.Location, req.Since, time.Now())

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	publishEvent("member.added", "project", req.ProjectID, map[string]interface{}{
		"member_id": req.ID,
		"name":      req.Name,
		"role":      req.Role,
	})

	c.JSON(201, req)
}

func dbRemoveMember(c *gin.Context) {
	memberID := c.Param("id")

	_, err := DB.Exec(`DELETE FROM pms.project_members WHERE id = $1`, memberID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Member removed successfully"})
}

// ═══════════════════════════════════════════════════════════════════
// BUDGET & FINANCE
// ═══════════════════════════════════════════════════════════════════

func dbCreateBudgetLine(c *gin.Context) {
	var req BudgetLine
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.ID = fmt.Sprintf("b-%d", rand.Intn(100000))

	_, err := DB.Exec(`INSERT INTO pms.budget_lines (id, project_id, category, description, source, amount, spent, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.Category, req.Description, req.Source, req.Amount, req.Spent, time.Now())

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Update project totals
	dbUpdateProjectBudgetTotals(req.ProjectID)

	c.JSON(201, req)
}

func dbUpdateBudgetLine(c *gin.Context) {
	budgetID := c.Param("id")

	var req BudgetLine
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.budget_lines SET
		category = COALESCE(NULLIF($1, ''), category),
		description = COALESCE(NULLIF($2, ''), description),
		source = COALESCE(NULLIF($3, ''), source),
		amount = CASE WHEN $4 >= 0 THEN $4 ELSE amount END,
		spent = CASE WHEN $5 >= 0 THEN $5 ELSE spent END
		WHERE id = $6`,
		req.Category, req.Description, req.Source, req.Amount, req.Spent, budgetID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Get project ID for this budget line
	var projectID string
	DB.QueryRow(`SELECT project_id FROM pms.budget_lines WHERE id = $1`, budgetID).Scan(&projectID)
	if projectID != "" {
		dbUpdateProjectBudgetTotals(projectID)
	}

	c.JSON(200, gin.H{"message": "Budget line updated successfully"})
}

func dbDeleteBudgetLine(c *gin.Context) {
	budgetID := c.Param("id")

	var projectID string
	DB.QueryRow(`SELECT project_id FROM pms.budget_lines WHERE id = $1`, budgetID).Scan(&projectID)

	_, err := DB.Exec(`DELETE FROM pms.budget_lines WHERE id = $1`, budgetID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if projectID != "" {
		dbUpdateProjectBudgetTotals(projectID)
	}

	c.JSON(200, gin.H{"message": "Budget line deleted successfully"})
}

func dbUpdateProjectBudgetTotals(projectID string) {
	var total, spent float64
	DB.QueryRow(`SELECT COALESCE(SUM(amount), 0), COALESCE(SUM(spent), 0) FROM pms.budget_lines WHERE project_id = $1`, projectID).Scan(&total, &spent)
	DB.Exec(`UPDATE pms.projects SET budget_total = $1, spent_total = $2, updated_time = $3 WHERE id = $4`,
		total, spent, time.Now(), projectID)
}

func dbGetFundingSources(c *gin.Context) {
	projectID := c.Param("id")
	funding, err := dbGetFundingSourcesByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, funding)
}

func dbGetFundingSourcesByProject(projectID string) ([]FundingSource, error) {
	rows, err := DB.Query(`SELECT id, project_id, name, type, donor, amount, received, currency, start_date, end_date, status, created_time FROM pms.funding_sources WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funding []FundingSource
	for rows.Next() {
		var f FundingSource
		rows.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Type, &f.Donor, &f.Amount, &f.Received, &f.Currency, &f.StartDate, &f.EndDate, &f.Status, &f.CreatedTime)
		funding = append(funding, f)
	}
	return funding, nil
}

func dbCreateFundingSource(c *gin.Context) {
	var req FundingSource
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("fs-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.funding_sources (id, project_id, name, type, donor, amount, received, currency, start_date, end_date, status, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		req.ID, req.ProjectID, req.Name, req.Type, req.Donor, req.Amount, req.Received, req.Currency, req.StartDate, req.EndDate, req.Status, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetCostCentres(c *gin.Context) {
	projectID := c.Param("id")
	centres, err := dbGetCostCentresByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, centres)
}

func dbGetCostCentresByProject(projectID string) ([]CostCentre, error) {
	rows, err := DB.Query(`SELECT id, project_id, name, code, description, budget, spent, created_time FROM pms.cost_centres WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var centres []CostCentre
	for rows.Next() {
		var cc CostCentre
		rows.Scan(&cc.ID, &cc.ProjectID, &cc.Name, &cc.Code, &cc.Description, &cc.Budget, &cc.Spent, &cc.CreatedTime)
		centres = append(centres, cc)
	}
	return centres, nil
}

func dbCreateCostCentre(c *gin.Context) {
	var req CostCentre
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("cc-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.cost_centres (id, project_id, name, code, description, budget, spent, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.Name, req.Code, req.Description, req.Budget, req.Spent, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetBudgetRevisions(c *gin.Context) {
	projectID := c.Param("id")
	revisions, err := dbGetBudgetRevisionsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, revisions)
}

func dbGetBudgetRevisionsByProject(projectID string) ([]BudgetRevision, error) {
	rows, err := DB.Query(`SELECT id, project_id, version, description, amount, approved_by, approved_at, status, created_time FROM pms.budget_revisions WHERE project_id = $1 ORDER BY version DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var revisions []BudgetRevision
	for rows.Next() {
		var br BudgetRevision
		var approvedAt sql.NullTime
		rows.Scan(&br.ID, &br.ProjectID, &br.Version, &br.Description, &br.Amount, &br.ApprovedBy, &approvedAt, &br.Status, &br.CreatedTime)
		if approvedAt.Valid {
			br.ApprovedAt = approvedAt.Time
		}
		revisions = append(revisions, br)
	}
	return revisions, nil
}

func dbCreateBudgetRevision(c *gin.Context) {
	var req BudgetRevision
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("br-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	// Get next version number
	var nextVersion int
	DB.QueryRow(`SELECT COALESCE(MAX(version), 0) + 1 FROM pms.budget_revisions WHERE project_id = $1`, req.ProjectID).Scan(&nextVersion)
	req.Version = nextVersion

	_, err := DB.Exec(`INSERT INTO pms.budget_revisions (id, project_id, version, description, amount, approved_by, approved_at, status, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.ID, req.ProjectID, req.Version, req.Description, req.Amount, req.ApprovedBy, req.ApprovedAt, req.Status, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetProcurementRefs(c *gin.Context) {
	projectID := c.Param("id")
	refs, err := dbGetProcurementRefsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, refs)
}

func dbGetProcurementRefsByProject(projectID string) ([]ProcurementReference, error) {
	rows, err := DB.Query(`SELECT id, project_id, reference, description, vendor, amount, status, created_time FROM pms.procurement_refs WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []ProcurementReference
	for rows.Next() {
		var pr ProcurementReference
		rows.Scan(&pr.ID, &pr.ProjectID, &pr.Reference, &pr.Description, &pr.Vendor, &pr.Amount, &pr.Status, &pr.CreatedTime)
		refs = append(refs, pr)
	}
	return refs, nil
}

func dbCreateProcurementRef(c *gin.Context) {
	var req ProcurementReference
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("pr-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.procurement_refs (id, project_id, reference, description, vendor, amount, status, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.Reference, req.Description, req.Vendor, req.Amount, req.Status, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

// ═══════════════════════════════════════════════════════════════════
// RISKS & ISSUES
// ═══════════════════════════════════════════════════════════════════

func dbCreateRisk(c *gin.Context) {
	var req Risk
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	req.ID = fmt.Sprintf("r-%d", rand.Intn(100000))
	req.Status = "Active"

	_, err := DB.Exec(`INSERT INTO pms.risks (id, project_id, description, category, probability, impact, mitigation, status, owner, due_date, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		req.ID, req.ProjectID, req.Description, req.Category, req.Probability, req.Impact,
		req.Mitigation, req.Status, req.Owner, req.DueDate, time.Now())

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Increment risk count in project
	DB.Exec(`UPDATE pms.projects SET risks_count = risks_count + 1, updated_time = $1 WHERE id = $2`,
		time.Now(), req.ProjectID)

	c.JSON(201, req)
}

func dbUpdateRisk(c *gin.Context) {
	riskID := c.Param("id")

	var req Risk
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.risks SET
		description = COALESCE(NULLIF($1, ''), description),
		category = COALESCE(NULLIF($2, ''), category),
		probability = COALESCE(NULLIF($3, ''), probability),
		impact = COALESCE(NULLIF($4, ''), impact),
		mitigation = COALESCE(NULLIF($5, ''), mitigation),
		status = COALESCE(NULLIF($6, ''), status),
		owner = COALESCE(NULLIF($7, ''), owner),
		due_date = COALESCE(NULLIF($8, ''), due_date)
		WHERE id = $9`,
		req.Description, req.Category, req.Probability, req.Impact, req.Mitigation, req.Status, req.Owner, req.DueDate, riskID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Risk updated successfully"})
}

func dbDeleteRisk(c *gin.Context) {
	riskID := c.Param("id")

	var projectID string
	DB.QueryRow(`SELECT project_id FROM pms.risks WHERE id = $1`, riskID).Scan(&projectID)

	_, err := DB.Exec(`DELETE FROM pms.risks WHERE id = $1`, riskID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if projectID != "" {
		DB.Exec(`UPDATE pms.projects SET risks_count = GREATEST(risks_count - 1, 0), updated_time = $1 WHERE id = $2`,
			time.Now(), projectID)
	}

	c.JSON(200, gin.H{"message": "Risk deleted successfully"})
}

func dbGetIssues(c *gin.Context) {
	projectID := c.Param("id")
	issues, err := dbGetIssuesByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, issues)
}

func dbGetIssuesByProject(projectID string) ([]Issue, error) {
	rows, err := DB.Query(`SELECT id, project_id, title, description, category, priority, status, owner, due_date, resolution, created_time, updated_time FROM pms.issues WHERE project_id = $1 ORDER BY created_time DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []Issue
	for rows.Next() {
		var i Issue
		rows.Scan(&i.ID, &i.ProjectID, &i.Title, &i.Description, &i.Category, &i.Priority, &i.Status, &i.Owner, &i.DueDate, &i.Resolution, &i.CreatedTime, &i.UpdatedTime)
		issues = append(issues, i)
	}
	return issues, nil
}

func dbCreateIssue(c *gin.Context) {
	var req Issue
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("i-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()
	req.UpdatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.issues (id, project_id, title, description, category, priority, status, owner, due_date, resolution, created_time, updated_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		req.ID, req.ProjectID, req.Title, req.Description, req.Category, req.Priority, req.Status, req.Owner, req.DueDate, req.Resolution, req.CreatedTime, req.UpdatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Increment issue count
	DB.Exec(`UPDATE pms.projects SET issues_count = issues_count + 1, updated_time = $1 WHERE id = $2`,
		time.Now(), req.ProjectID)

	c.JSON(201, req)
}

func dbUpdateIssue(c *gin.Context) {
	issueID := c.Param("id")

	var req Issue
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.issues SET
		title = COALESCE(NULLIF($1, ''), title),
		description = COALESCE(NULLIF($2, ''), description),
		category = COALESCE(NULLIF($3, ''), category),
		priority = COALESCE(NULLIF($4, ''), priority),
		status = COALESCE(NULLIF($5, ''), status),
		owner = COALESCE(NULLIF($6, ''), owner),
		due_date = COALESCE(NULLIF($7, ''), due_date),
		resolution = COALESCE(NULLIF($8, ''), resolution),
		updated_time = $9
		WHERE id = $10`,
		req.Title, req.Description, req.Category, req.Priority, req.Status, req.Owner, req.DueDate, req.Resolution, time.Now(), issueID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Issue updated successfully"})
}

func dbGetAssumptions(c *gin.Context) {
	projectID := c.Param("id")
	assumptions, err := dbGetAssumptionsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, assumptions)
}

func dbGetAssumptionsByProject(projectID string) ([]Assumption, error) {
	rows, err := DB.Query(`SELECT id, project_id, description, category, status, owner, created_time FROM pms.assumptions WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assumptions []Assumption
	for rows.Next() {
		var a Assumption
		rows.Scan(&a.ID, &a.ProjectID, &a.Description, &a.Category, &a.Status, &a.Owner, &a.CreatedTime)
		assumptions = append(assumptions, a)
	}
	return assumptions, nil
}

func dbCreateAssumption(c *gin.Context) {
	var req Assumption
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("as-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.assumptions (id, project_id, description, category, status, owner, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		req.ID, req.ProjectID, req.Description, req.Category, req.Status, req.Owner, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetLessonsLearned(c *gin.Context) {
	projectID := c.Param("id")
	lessons, err := dbGetLessonsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, lessons)
}

func dbGetLessonsByProject(projectID string) ([]LessonLearned, error) {
	rows, err := DB.Query(`SELECT id, project_id, title, description, category, impact, owner, created_time FROM pms.lessons_learned WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []LessonLearned
	for rows.Next() {
		var l LessonLearned
		rows.Scan(&l.ID, &l.ProjectID, &l.Title, &l.Description, &l.Category, &l.Impact, &l.Owner, &l.CreatedTime)
		lessons = append(lessons, l)
	}
	return lessons, nil
}

func dbCreateLessonLearned(c *gin.Context) {
	var req LessonLearned
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("ll-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.lessons_learned (id, project_id, title, description, category, impact, owner, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.Title, req.Description, req.Category, req.Impact, req.Owner, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetCorrectiveActions(c *gin.Context) {
	projectID := c.Param("id")
	actions, err := dbGetCorrectiveActionsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, actions)
}

func dbGetCorrectiveActionsByProject(projectID string) ([]CorrectiveAction, error) {
	rows, err := DB.Query(`SELECT id, project_id, issue_id, description, status, owner, due_date, created_time FROM pms.corrective_actions WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []CorrectiveAction
	for rows.Next() {
		var ca CorrectiveAction
		var issueID sql.NullString
		rows.Scan(&ca.ID, &ca.ProjectID, &issueID, &ca.Description, &ca.Status, &ca.Owner, &ca.DueDate, &ca.CreatedTime)
		ca.IssueID = issueID.String
		actions = append(actions, ca)
	}
	return actions, nil
}

func dbCreateCorrectiveAction(c *gin.Context) {
	var req CorrectiveAction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("ca-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.corrective_actions (id, project_id, issue_id, description, status, owner, due_date, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.IssueID, req.Description, req.Status, req.Owner, req.DueDate, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

// ═══════════════════════════════════════════════════════════════════
// DOCUMENTS
// ═══════════════════════════════════════════════════════════════════

func dbCreateDocument(c *gin.Context) {
	var req Document
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("doc-%d", rand.Intn(100000))
	req.UploadedAt = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.documents (id, project_id, name, type, size, uploaded_by, uploaded_at, url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.ID, req.ProjectID, req.Name, req.Type, req.Size, req.UploadedBy, req.UploadedAt, req.Url, req.Status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbDeleteDocument(c *gin.Context) {
	docID := c.Param("id")

	_, err := DB.Exec(`DELETE FROM pms.documents WHERE id = $1`, docID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Document deleted successfully"})
}

// ═══════════════════════════════════════════════════════════════════
// MEETINGS
// ═══════════════════════════════════════════════════════════════════

func dbCreateMeeting(c *gin.Context) {
	var req Meeting
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("mt-%d", rand.Intn(100000))
	if req.Status == "" {
		req.Status = "Scheduled"
	}

	_, err := DB.Exec(`INSERT INTO pms.meetings (id, project_id, title, date_time, location, attendees, agenda, minutes, action_items, status, decisions, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		req.ID, req.ProjectID, req.Title, req.DateTime, req.Location, req.Attendees, req.Agenda, req.Minutes, req.ActionItems, req.Status, req.Decisions, time.Now())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Add calendar event
	DB.Exec(`INSERT INTO pms.calendar_events (id, project_id, title, description, event_type, start_time, end_time, location, attendees, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		fmt.Sprintf("ev-%d", rand.Intn(100000)), req.ProjectID, req.Title, req.Agenda, "Meeting",
		req.DateTime, req.DateTime, req.Location, req.Attendees, time.Now())

	// Notify in chat
	DB.Exec(`INSERT INTO pms.chat_messages (id, project_id, channel, sender, role, message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		fmt.Sprintf("ch-%d", rand.Intn(100000)), req.ProjectID, "meetings", "Meeting Scheduler", "System",
		fmt.Sprintf("Meeting scheduled: %s", req.Title), time.Now())

	c.JSON(201, req)
}

func dbUpdateMeeting(c *gin.Context) {
	meetingID := c.Param("id")

	var req Meeting
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.meetings SET
		title = COALESCE(NULLIF($1, ''), title),
		date_time = COALESCE(NULLIF($2, ''), date_time),
		location = COALESCE(NULLIF($3, ''), location),
		agenda = COALESCE(NULLIF($4, ''), agenda),
		minutes = COALESCE(NULLIF($5, ''), minutes),
		status = COALESCE(NULLIF($6, ''), status)
		WHERE id = $7`,
		req.Title, req.DateTime, req.Location, req.Agenda, req.Minutes, req.Status, meetingID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Meeting updated successfully"})
}

// ═══════════════════════════════════════════════════════════════════
// SURVEYS
// ═══════════════════════════════════════════════════════════════════

func dbCreateSurvey(c *gin.Context) {
	var req Survey
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("s-%d", rand.Intn(100000))

	_, err := DB.Exec(`INSERT INTO pms.surveys (id, project_id, name, status, target_sample, submissions, progress, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.Name, req.Status, req.TargetSample, req.Submissions, req.Progress, time.Now())
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbUpdateSurvey(c *gin.Context) {
	surveyID := c.Param("id")

	var req Survey
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.surveys SET
		name = COALESCE(NULLIF($1, ''), name),
		status = COALESCE(NULLIF($2, ''), status),
		target_sample = CASE WHEN $3 > 0 THEN $3 ELSE target_sample END,
		submissions = CASE WHEN $4 >= 0 THEN $4 ELSE submissions END,
		progress = CASE WHEN $5 >= 0 THEN $5 ELSE progress END
		WHERE id = $6`,
		req.Name, req.Status, req.TargetSample, req.Submissions, req.Progress, surveyID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Survey updated successfully"})
}

// ═══════════════════════════════════════════════════════════════════
// CHAT
// ═══════════════════════════════════════════════════════════════════

func dbPostChatMessage(c *gin.Context) {
	var req ChatMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "projectId and message are required"})
		return
	}
	message, err := sendProjectDiscussionMessage(c, req)
	if err != nil {
		writeProjectDiscussionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, message)
}

// ═══════════════════════════════════════════════════════════════════
// HELPDESK
// ═══════════════════════════════════════════════════════════════════

func dbCreateHelpDeskTicket(c *gin.Context) {
	var req HelpDeskTicket
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("hd-%d", rand.Intn(100000))
	req.CreatedAt = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.helpdesk_tickets (id, project_id, title, description, status, priority, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.ID, req.ProjectID, req.Title, req.Description, req.Status, req.Priority, req.CreatedBy, req.CreatedAt)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbUpdateHelpDeskTicket(c *gin.Context) {
	ticketID := c.Param("id")

	var req HelpDeskTicket
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`UPDATE pms.helpdesk_tickets SET
		title = COALESCE(NULLIF($1, ''), title),
		description = COALESCE(NULLIF($2, ''), description),
		status = COALESCE(NULLIF($3, ''), status),
		priority = COALESCE(NULLIF($4, ''), priority),
		updated_at = $5
		WHERE id = $6`,
		req.Title, req.Description, req.Status, req.Priority, time.Now(), ticketID)

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Ticket updated successfully"})
}

// ═══════════════════════════════════════════════════════════════════
// WORKFLOW & PERMISSIONS
// ═══════════════════════════════════════════════════════════════════

func dbGetWorkflowRules(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, name, description, from_stage, to_stage, required_role, auto_actions, enabled, created_time FROM pms.workflow_rules ORDER BY from_stage`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rules []WorkflowRule
	for rows.Next() {
		var wf WorkflowRule
		rows.Scan(&wf.ID, &wf.Name, &wf.Description, &wf.FromStage, &wf.ToStage, &wf.RequiredRole, &wf.AutoActions, &wf.Enabled, &wf.CreatedTime)
		rules = append(rules, wf)
	}
	c.JSON(200, rules)
}

func dbCreateWorkflowRule(c *gin.Context) {
	var req WorkflowRule
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("wf-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.workflow_rules (id, name, description, from_stage, to_stage, required_role, auto_actions, enabled, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.ID, req.Name, req.Description, req.FromStage, req.ToStage, req.RequiredRole, req.AutoActions, req.Enabled, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

func dbGetPermissions(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, role, resource, action, created_time FROM pms.permissions ORDER BY role`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var p Permission
		rows.Scan(&p.ID, &p.Role, &p.Resource, &p.Action, &p.CreatedTime)
		permissions = append(permissions, p)
	}
	c.JSON(200, permissions)
}

func dbCreatePermission(c *gin.Context) {
	var req Permission
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("perm-%d", rand.Intn(100000))
	req.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.permissions (id, role, resource, action, created_time)
		VALUES ($1, $2, $3, $4, $5)`,
		req.ID, req.Role, req.Resource, req.Action, req.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, req)
}

// ═══════════════════════════════════════════════════════════════════
// AUDIT LOG
// ═══════════════════════════════════════════════════════════════════

func dbLogAudit(projectID, user, action, entity, entityID, details string) {
	DB.Exec(`INSERT INTO pms.audit_logs (id, project_id, user_name, action, entity, entity_id, details, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		fmt.Sprintf("al-%d", rand.Intn(100000)), projectID, user, action, entity, entityID, details, time.Now())
}

func dbGetProjectAuditLogs(c *gin.Context) {
	projectID := c.Param("id")
	logs, err := dbGetAuditLogsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, logs)
}

func dbGetAuditLogsByProject(projectID string) ([]AuditLog, error) {
	rows, err := DB.Query(`SELECT id, project_id, user_name, action, entity, entity_id, details, timestamp FROM pms.audit_logs WHERE project_id = $1 ORDER BY timestamp DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var al AuditLog
		var entityID sql.NullString
		rows.Scan(&al.ID, &al.ProjectID, &al.User, &al.Action, &al.Entity, &entityID, &al.Details, &al.Timestamp)
		al.EntityID = entityID.String
		logs = append(logs, al)
	}
	return logs, nil
}

// ═══════════════════════════════════════════════════════════════════
// CALENDAR
// ═══════════════════════════════════════════════════════════════════

func dbGetProjectCalendar(c *gin.Context) {
	projectID := c.Param("id")
	events, err := dbGetCalendarByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, events)
}

func dbGetCalendarByProject(projectID string) ([]CalendarEvent, error) {
	rows, err := DB.Query(`SELECT id, project_id, title, description, event_type, start_time, end_time, location, attendees, created_time FROM pms.calendar_events WHERE project_id = $1 ORDER BY start_time`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var ev CalendarEvent
		rows.Scan(&ev.ID, &ev.ProjectID, &ev.Title, &ev.Description, &ev.EventType, &ev.StartTime, &ev.EndTime, &ev.Location, &ev.Attendees, &ev.CreatedTime)
		events = append(events, ev)
	}
	return events, nil
}

// ═══════════════════════════════════════════════════════════════════
// REPORTS
// ═══════════════════════════════════════════════════════════════════

func dbGetProjectReports(c *gin.Context) {
	projectID := c.Param("id")
	reports, err := dbGetReportsByProject(projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, reports)
}

func dbGetReportsByProject(projectID string) ([]Report, error) {
	rows, err := DB.Query(`SELECT id, project_id, title, type, format, generated_by, generated_at, content, status FROM pms.reports WHERE project_id = $1 ORDER BY generated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []Report
	for rows.Next() {
		var r Report
		rows.Scan(&r.ID, &r.ProjectID, &r.Title, &r.Type, &r.Format, &r.GeneratedBy, &r.GeneratedAt, &r.Content, &r.Status)
		reports = append(reports, r)
	}
	return reports, nil
}

func dbGenerateReport(c *gin.Context) {
	projectID := c.Param("id")

	var req struct {
		Title       string `json:"title"`
		Type        string `json:"type"`
		Format      string `json:"format"`
		GeneratedBy string `json:"generatedBy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	reportID := fmt.Sprintf("rp-%d", rand.Intn(100000))
	now := time.Now()

	// Generate report content from live data
	content := dbBuildReportContent(projectID, req.Type)

	_, err := DB.Exec(`INSERT INTO pms.reports (id, project_id, title, type, format, generated_by, generated_at, content, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		reportID, projectID, req.Title, req.Type, req.Format, req.GeneratedBy, now, content, "Generated")

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"id": reportID, "message": "Report generated successfully", "content": content})
}

func dbBuildReportContent(projectID, reportType string) string {
	var p Project
	DB.QueryRow(`SELECT id, code, name, description, stage, progress, org, portfolio, programme, owner, start_date, end_date, target_geo, budget_total, spent_total, risks_count, issues_count FROM pms.projects WHERE id = $1`, projectID).Scan(
		&p.ID, &p.Code, &p.Name, &p.Description, &p.Stage, &p.Progress, &p.Org, &p.Portfolio, &p.Programme,
		&p.Owner, &p.StartDate, &p.EndDate, &p.TargetGeo, &p.BudgetTotal, &p.SpentTotal, &p.RisksCount, &p.IssuesCount)

	var taskCount, doneTasks, inProgressTasks int
	DB.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'Done'), COUNT(*) FILTER (WHERE status = 'In Progress') FROM pms.tasks WHERE project_id = $1`, projectID).Scan(&taskCount, &doneTasks, &inProgressTasks)

	var surveyCount, totalSubmissions, totalTarget int
	DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(submissions), 0), COALESCE(SUM(target_sample), 0) FROM pms.surveys WHERE project_id = $1`, projectID).Scan(&surveyCount, &totalSubmissions, &totalTarget)

	var memberCount int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.project_members WHERE project_id = $1`, projectID).Scan(&memberCount)

	var meetingCount int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.meetings WHERE project_id = $1`, projectID).Scan(&meetingCount)

	var docCount int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.documents WHERE project_id = $1`, projectID).Scan(&docCount)

	var issueCount int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.issues WHERE project_id = $1 AND status != 'Resolved'`, projectID).Scan(&issueCount)

	budgetUtil := 0.0
	if p.BudgetTotal > 0 {
		budgetUtil = (p.SpentTotal / p.BudgetTotal) * 100
	}

	surveyProgress := 0.0
	if totalTarget > 0 {
		surveyProgress = (float64(totalSubmissions) / float64(totalTarget)) * 100
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s REPORT ===\n", strings.ToUpper(reportType)))
	sb.WriteString(fmt.Sprintf("Project: %s (%s)\n", p.Name, p.Code))
	sb.WriteString(fmt.Sprintf("Stage: %s | Progress: %.1f%%\n", p.Stage, p.Progress))
	sb.WriteString(fmt.Sprintf("Organization: %s | Portfolio: %s | Programme: %s\n", p.Org, p.Portfolio, p.Programme))
	sb.WriteString(fmt.Sprintf("Owner: %s | Period: %s to %s\n", p.Owner, p.StartDate, p.EndDate))
	sb.WriteString(fmt.Sprintf("Target Region: %s\n", p.TargetGeo))
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("Budget: $%.2f allocated | $%.2f spent (%.1f%%)\n", p.BudgetTotal, p.SpentTotal, budgetUtil))
	sb.WriteString(fmt.Sprintf("Tasks: %d total | %d done | %d in progress\n", taskCount, doneTasks, inProgressTasks))
	sb.WriteString(fmt.Sprintf("Surveys: %d | Submissions: %d/%d (%.1f%%)\n", surveyCount, totalSubmissions, totalTarget, surveyProgress))
	sb.WriteString(fmt.Sprintf("Team: %d members | Meetings: %d | Documents: %d\n", memberCount, meetingCount, docCount))
	sb.WriteString(fmt.Sprintf("Risks: %d | Open Issues: %d\n", p.RisksCount, issueCount))
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return sb.String()
}

// ═══════════════════════════════════════════════════════════════════
// RELATIONSHIPS
// ═══════════════════════════════════════════════════════════════════

func dbGetProjectRelationships(c *gin.Context) {
	projectID := c.Param("id")

	relationships := map[string]interface{}{}

	// Count all related objects
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.tasks WHERE project_id = $1`, projectID).Scan(&count)
	relationships["tasks"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.project_members WHERE project_id = $1`, projectID).Scan(&count)
	relationships["members"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.budget_lines WHERE project_id = $1`, projectID).Scan(&count)
	relationships["budgetLines"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.risks WHERE project_id = $1`, projectID).Scan(&count)
	relationships["risks"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.issues WHERE project_id = $1`, projectID).Scan(&count)
	relationships["issues"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.documents WHERE project_id = $1`, projectID).Scan(&count)
	relationships["documents"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.meetings WHERE project_id = $1`, projectID).Scan(&count)
	relationships["meetings"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.surveys WHERE project_id = $1`, projectID).Scan(&count)
	relationships["surveys"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.chat_messages WHERE project_id = $1`, projectID).Scan(&count)
	relationships["chatMessages"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.helpdesk_tickets WHERE project_id = $1`, projectID).Scan(&count)
	relationships["helpdeskTickets"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.components WHERE project_id = $1`, projectID).Scan(&count)
	relationships["components"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.activities WHERE project_id = $1`, projectID).Scan(&count)
	relationships["activities"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.deliverables WHERE project_id = $1`, projectID).Scan(&count)
	relationships["deliverables"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.milestones WHERE project_id = $1`, projectID).Scan(&count)
	relationships["milestones"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.funding_sources WHERE project_id = $1`, projectID).Scan(&count)
	relationships["fundingSources"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.cost_centres WHERE project_id = $1`, projectID).Scan(&count)
	relationships["costCentres"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.calendar_events WHERE project_id = $1`, projectID).Scan(&count)
	relationships["calendarEvents"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.reports WHERE project_id = $1`, projectID).Scan(&count)
	relationships["reports"] = count
	DB.QueryRow(`SELECT COUNT(*) FROM pms.audit_logs WHERE project_id = $1`, projectID).Scan(&count)
	relationships["auditLogs"] = count

	c.JSON(200, relationships)
}

// ═══════════════════════════════════════════════════════════════════
// SEARCH
// ═══════════════════════════════════════════════════════════════════

func dbSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"error": "Search query required"})
		return
	}

	searchTerm := "%" + strings.ToLower(query) + "%"
	var results []SearchResult

	// Search projects
	rows, err := DB.Query(`
		SELECT id, name, description, code FROM pms.projects
		WHERE (LOWER(name) LIKE $1 OR LOWER(code) LIKE $1 OR LOWER(description) LIKE $1 OR LOWER(programme) LIKE $1 OR LOWER(owner) LIKE $1 OR LOWER(org) LIKE $1 OR LOWER(portfolio) LIKE $1 OR LOWER(target_geo) LIKE $1)
		AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var code string
			rows.Scan(&r.ID, &r.Title, &r.Description, &code)
			r.Type = "project"
			r.Meta = code
			results = append(results, r)
		}
	}

	// Search tasks
	rows, err = DB.Query(`
		SELECT t.id, t.title, t.description, p.name FROM pms.tasks t
		JOIN pms.projects p ON p.id = t.project_id
		WHERE (LOWER(t.title) LIKE $1 OR LOWER(t.description) LIKE $1 OR LOWER(t.assigned_to) LIKE $1)
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "task"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	// Search members
	rows, err = DB.Query(`
		SELECT m.id, m.name, m.role, p.name FROM pms.project_members m
		JOIN pms.projects p ON p.id = m.project_id
		WHERE (LOWER(m.name) LIKE $1 OR LOWER(m.role) LIKE $1)
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "member"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	// Search documents
	rows, err = DB.Query(`
		SELECT d.id, d.name, d.type, p.name FROM pms.documents d
		JOIN pms.projects p ON p.id = d.project_id
		WHERE (LOWER(d.name) LIKE $1 OR LOWER(d.type) LIKE $1)
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "document"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	// Search meetings
	rows, err = DB.Query(`
		SELECT m.id, m.title, m.agenda, p.name FROM pms.meetings m
		JOIN pms.projects p ON p.id = m.project_id
		WHERE (LOWER(m.title) LIKE $1 OR LOWER(m.agenda) LIKE $1)
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "meeting"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	// Search risks
	rows, err = DB.Query(`
		SELECT r.id, r.description, r.category, p.name FROM pms.risks r
		JOIN pms.projects p ON p.id = r.project_id
		WHERE (LOWER(r.description) LIKE $1 OR LOWER(r.category) LIKE $1)
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "risk"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	// Search issues
	rows, err = DB.Query(`
		SELECT i.id, i.title, i.description, p.name FROM pms.issues i
		JOIN pms.projects p ON p.id = i.project_id
		WHERE (LOWER(i.title) LIKE $1 OR LOWER(i.description) LIKE $1)
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "issue"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	// Search surveys
	rows, err = DB.Query(`
		SELECT s.id, s.name, s.status, p.name FROM pms.surveys s
		JOIN pms.projects p ON p.id = s.project_id
		WHERE LOWER(s.name) LIKE $1
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "survey"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	// Search milestones
	rows, err = DB.Query(`
		SELECT m.id, m.name, m.description, p.name FROM pms.milestones m
		JOIN pms.projects p ON p.id = m.project_id
		WHERE LOWER(m.name) LIKE $1
		AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, searchTerm, workspaceIDContext(c))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			var projectName string
			rows.Scan(&r.ID, &r.Title, &r.Description, &projectName)
			r.Type = "milestone"
			r.Meta = projectName
			results = append(results, r)
		}
	}

	c.JSON(200, results)
}

// ═══════════════════════════════════════════════════════════════════
// DASHBOARDS
// ═══════════════════════════════════════════════════════════════════

func dbGetDashboard(c *gin.Context) {
	// Global dashboard with aggregate stats
	var totalProjects, totalMembers, totalTasks, totalDoneTasks, totalSurveys, totalSubmissions, totalTarget int
	var totalBudget, totalSpent float64

	workspaceID := workspaceIDContext(c)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalProjects)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.project_members m JOIN pms.projects p ON p.id=m.project_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalMembers)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.tasks t JOIN pms.projects p ON p.id=t.project_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalTasks)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.tasks t JOIN pms.projects p ON p.id=t.project_id WHERE t.status = 'Done' AND (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalDoneTasks)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.surveys s JOIN pms.projects p ON p.id=s.project_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalSurveys)
	DB.QueryRow(`SELECT COALESCE(SUM(s.submissions), 0) FROM pms.surveys s JOIN pms.projects p ON p.id=s.project_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalSubmissions)
	DB.QueryRow(`SELECT COALESCE(SUM(s.target_sample), 0) FROM pms.surveys s JOIN pms.projects p ON p.id=s.project_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalTarget)
	DB.QueryRow(`SELECT COALESCE(SUM(budget_total), 0) FROM pms.projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalBudget)
	DB.QueryRow(`SELECT COALESCE(SUM(spent_total), 0) FROM pms.projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&totalSpent)

	// Stage distribution
	stageRows, _ := DB.Query(`SELECT stage, COUNT(*) FROM pms.projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL) GROUP BY stage`, workspaceID)
	defer stageRows.Close()
	stageDist := map[string]int{}
	for stageRows.Next() {
		var stage string
		var count int
		stageRows.Scan(&stage, &count)
		stageDist[stage] = count
	}

	// Recent projects
	recentRows, _ := DB.Query(`SELECT id, code, name, stage, progress, owner, created_time FROM pms.projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL) ORDER BY created_time DESC LIMIT 5`, workspaceID)
	defer recentRows.Close()
	var recentProjects []map[string]interface{}
	for recentRows.Next() {
		var id, code, name, stage, owner string
		var progress float64
		var created time.Time
		recentRows.Scan(&id, &code, &name, &stage, &progress, &owner, &created)
		recentProjects = append(recentProjects, map[string]interface{}{
			"id": id, "code": code, "name": name, "stage": stage, "progress": progress, "owner": owner, "createdTime": created,
		})
	}

	c.JSON(200, gin.H{
		"totalProjects":     totalProjects,
		"totalMembers":      totalMembers,
		"totalTasks":        totalTasks,
		"totalDoneTasks":    totalDoneTasks,
		"totalSurveys":      totalSurveys,
		"totalSubmissions":  totalSubmissions,
		"totalTarget":       totalTarget,
		"totalBudget":       totalBudget,
		"totalSpent":        totalSpent,
		"stageDistribution": stageDist,
		"recentProjects":    recentProjects,
	})
}

func dbGetPortfolioDashboard(c *gin.Context) {
	workspaceID := workspaceIDContext(c)
	portfolio := strings.TrimSpace(c.Query("portfolio"))
	programme := strings.TrimSpace(c.Query("programme"))

	where := "WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)"
	args := []interface{}{workspaceID}
	if portfolio != "" {
		where += fmt.Sprintf(" AND p.portfolio = $%d", len(args)+1)
		args = append(args, portfolio)
	}
	if programme != "" {
		where += fmt.Sprintf(" AND p.programme = $%d", len(args)+1)
		args = append(args, programme)
	}

	var summary struct {
		ProjectCount    int     `json:"projectCount"`
		AverageProgress float64 `json:"averageProgress"`
		BudgetTotal     float64 `json:"budgetTotal"`
		SpentTotal      float64 `json:"spentTotal"`
		RisksTotal      int     `json:"risksTotal"`
		IssuesTotal     int     `json:"issuesTotal"`
	}
	if err := DB.QueryRow(fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(AVG(COALESCE(p.progress, 0)), 0),
		       COALESCE(SUM(p.budget_total), 0), COALESCE(SUM(p.spent_total), 0),
		       COALESCE(SUM(p.risks_count), 0), COALESCE(SUM(p.issues_count), 0)
		FROM pms.projects p %s
	`, where), args...).Scan(&summary.ProjectCount, &summary.AverageProgress, &summary.BudgetTotal,
		&summary.SpentTotal, &summary.RisksTotal, &summary.IssuesTotal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
		return
	}

	stageDistribution := map[string]int{}
	stageRows, err := DB.Query(fmt.Sprintf(`
		SELECT COALESCE(p.stage, 'Unassigned'), COUNT(*)
		FROM pms.projects p %s
		GROUP BY p.stage ORDER BY COUNT(*) DESC, p.stage ASC
	`, where), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
		return
	}
	for stageRows.Next() {
		var stage string
		var count int
		if err := stageRows.Scan(&stage, &count); err == nil {
			stageDistribution[stage] = count
		}
	}
	if err := stageRows.Err(); err != nil {
		stageRows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
		return
	}
	stageRows.Close()

	type groupSummary struct {
		Name            string  `json:"name"`
		ProjectCount    int     `json:"projectCount"`
		AverageProgress float64 `json:"averageProgress"`
		BudgetTotal     float64 `json:"budgetTotal"`
		SpentTotal      float64 `json:"spentTotal"`
		RisksTotal      int     `json:"risksTotal"`
		IssuesTotal     int     `json:"issuesTotal"`
	}
	loadGroups := func(column string) ([]groupSummary, error) {
		// Column names are selected from this fixed list, never from user input.
		query := fmt.Sprintf(`
			SELECT COALESCE(NULLIF(p.%s, ''), 'Unassigned'), COUNT(*),
			       COALESCE(AVG(COALESCE(p.progress, 0)), 0), COALESCE(SUM(p.budget_total), 0),
			       COALESCE(SUM(p.spent_total), 0), COALESCE(SUM(p.risks_count), 0),
			       COALESCE(SUM(p.issues_count), 0)
			FROM pms.projects p %s
			GROUP BY p.%s ORDER BY COUNT(*) DESC, p.%s ASC
		`, column, where, column, column)
		rows, err := DB.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		groups := []groupSummary{}
		for rows.Next() {
			var group groupSummary
			if err := rows.Scan(&group.Name, &group.ProjectCount, &group.AverageProgress, &group.BudgetTotal,
				&group.SpentTotal, &group.RisksTotal, &group.IssuesTotal); err != nil {
				return nil, err
			}
			groups = append(groups, group)
		}
		return groups, rows.Err()
	}

	portfolioBreakdown, err := loadGroups("portfolio")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
		return
	}
	programmeBreakdown, err := loadGroups("programme")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
		return
	}

	projectRows, err := DB.Query(fmt.Sprintf(`
		SELECT p.id, p.code, p.name, COALESCE(p.portfolio, ''), COALESCE(p.programme, ''),
		       COALESCE(p.stage, 'Unassigned'), COALESCE(p.progress, 0), COALESCE(p.budget_total, 0),
		       COALESCE(p.spent_total, 0), COALESCE(p.risks_count, 0), COALESCE(p.issues_count, 0)
		FROM pms.projects p %s ORDER BY p.updated_time DESC NULLS LAST, p.name ASC
	`, where), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
		return
	}
	type projectSummary struct {
		ID          string  `json:"id"`
		Code        string  `json:"code"`
		Name        string  `json:"name"`
		Portfolio   string  `json:"portfolio"`
		Programme   string  `json:"programme"`
		Stage       string  `json:"stage"`
		Progress    float64 `json:"progress"`
		BudgetTotal float64 `json:"budgetTotal"`
		SpentTotal  float64 `json:"spentTotal"`
		RisksCount  int     `json:"risksCount"`
		IssuesCount int     `json:"issuesCount"`
	}
	projects := []projectSummary{}
	for projectRows.Next() {
		var project projectSummary
		if err := projectRows.Scan(&project.ID, &project.Code, &project.Name, &project.Portfolio, &project.Programme,
			&project.Stage, &project.Progress, &project.BudgetTotal, &project.SpentTotal, &project.RisksCount, &project.IssuesCount); err != nil {
			projectRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
			return
		}
		projects = append(projects, project)
	}
	if err := projectRows.Err(); err != nil {
		projectRows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "portfolio dashboard unavailable"})
		return
	}
	projectRows.Close()

	c.JSON(http.StatusOK, gin.H{
		"scope":              gin.H{"workspaceId": workspaceID, "portfolio": portfolio, "programme": programme},
		"summary":            summary,
		"stageDistribution":  stageDistribution,
		"portfolioBreakdown": portfolioBreakdown,
		"programmeBreakdown": programmeBreakdown,
		"projects":           projects,
	})
}

func dbGetProjectHealthAssistant(c *gin.Context) {
	projectID := c.Param("id")
	var project struct {
		ID       string  `json:"projectId"`
		Name     string  `json:"projectName"`
		Budget   float64 `json:"budget"`
		Spent    float64 `json:"spent"`
		Progress float64 `json:"progress"`
	}
	if err := DB.QueryRow(`
		SELECT id, name, COALESCE(budget_total, 0), COALESCE(spent_total, 0), COALESCE(progress, 0)
		FROM pms.projects
		WHERE id = $1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, projectID, workspaceIDContext(c)).Scan(&project.ID, &project.Name, &project.Budget, &project.Spent, &project.Progress); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	var input struct {
		Action string `json:"action"`
	}
	if c.Request.Body != nil {
		if err := json.NewDecoder(c.Request.Body).Decode(&input); err != nil && err != io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assistant request"})
			return
		}
	}
	if input.Action == "" {
		input.Action = "health_review"
	}
	allowedActions := map[string]bool{
		"health_review":         true,
		"risk_prediction":       true,
		"schedule_optimization": true,
		"milestone_review":      true,
		"budget_forecast":       true,
	}
	if !allowedActions[input.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported assistant action"})
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"action":     input.Action,
		"project_id": project.ID,
		"budget":     project.Budget,
		"spent":      project.Spent,
		"progress":   project.Progress,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "assistant request unavailable"})
		return
	}
	baseURL := strings.TrimRight(getEnv("STATGATE_ENTERPRISE_API_URL", ""), "/")
	if baseURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "assistant service unavailable"})
		return
	}
	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, baseURL+"/ai/project/assist", bytes.NewReader(payload))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "assistant service unavailable"})
		return
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	request.Header.Set("X-Workspace-ID", workspaceIDContext(c))
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "assistant service unavailable"})
		return
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		c.JSON(http.StatusBadGateway, gin.H{"error": "assistant service rejected the request"})
		return
	}
	var assistant map[string]interface{}
	if err := json.NewDecoder(response.Body).Decode(&assistant); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid assistant response"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"project":          project,
		"assistant":        assistant,
		"governanceNotice": "AI advisory only. Consequential decisions require human authorization.",
	})
}

func dbGetProjectProgressSummary(c *gin.Context) {
	projectID := c.Param("id")
	workspaceID := workspaceIDContext(c)
	var project struct {
		ID       string  `json:"projectId"`
		Name     string  `json:"projectName"`
		Stage    string  `json:"stage"`
		Progress float64 `json:"progress"`
		Budget   float64 `json:"budget"`
		Spent    float64 `json:"spent"`
	}
	if err := DB.QueryRow(`
		SELECT id, name, stage, COALESCE(progress, 0), COALESCE(budget_total, 0), COALESCE(spent_total, 0)
		FROM pms.projects
		WHERE id = $1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)
	`, projectID, workspaceID).Scan(&project.ID, &project.Name, &project.Stage, &project.Progress, &project.Budget, &project.Spent); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	var taskCount, completedTasks, inProgressTasks int
	var taskProgress float64
	if err := DB.QueryRow(`
		SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'Done'), COUNT(*) FILTER (WHERE status = 'In Progress'), COALESCE(AVG(progress), 0)
		FROM pms.tasks WHERE project_id = $1
	`, projectID).Scan(&taskCount, &completedTasks, &inProgressTasks, &taskProgress); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "progress summary unavailable"})
		return
	}

	var milestoneCount, completedMilestones int
	if err := DB.QueryRow(`
		SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'Completed')
		FROM pms.milestones WHERE project_id = $1
	`, projectID).Scan(&milestoneCount, &completedMilestones); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "progress summary unavailable"})
		return
	}

	var openRisks int
	if err := DB.QueryRow(`
		SELECT COUNT(*) FROM pms.risks
		WHERE project_id = $1 AND COALESCE(status, '') NOT IN ('Closed', 'Resolved')
	`, projectID).Scan(&openRisks); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "progress summary unavailable"})
		return
	}

	var nextMilestoneName sql.NullString
	var nextMilestoneDue sql.NullTime
	if err := DB.QueryRow(`
		SELECT name, due_date FROM pms.milestones
		WHERE project_id = $1 AND COALESCE(status, '') <> 'Completed'
		ORDER BY due_date NULLS LAST LIMIT 1
	`, projectID).Scan(&nextMilestoneName, &nextMilestoneDue); err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "progress summary unavailable"})
		return
	}

	utilization := 0.0
	if project.Budget > 0 {
		utilization = project.Spent / project.Budget * 100
	}
	deliveryProgress := taskProgress
	if taskCount == 0 {
		deliveryProgress = project.Progress
	}

	recommendations := make([]string, 0, 4)
	if taskCount > 0 && completedTasks < taskCount {
		recommendations = append(recommendations, fmt.Sprintf("%d of %d tasks are complete; confirm owners and due dates for the remaining work.", completedTasks, taskCount))
	}
	if milestoneCount > 0 && completedMilestones < milestoneCount {
		recommendations = append(recommendations, fmt.Sprintf("%d of %d milestones are complete; review the next milestone before the next status meeting.", completedMilestones, milestoneCount))
	}
	if openRisks > 0 {
		recommendations = append(recommendations, fmt.Sprintf("%d open risks require an owner, mitigation, and review date.", openRisks))
	}
	if utilization >= 90 {
		recommendations = append(recommendations, "Budget utilization is at least 90%; complete a finance-owner review before committing new spend.")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No immediate delivery exception was found in the current PMS records.")
	}

	summary := fmt.Sprintf("%s is %.1f%% complete in the %s stage, with %d tasks and %d milestones tracked.", project.Name, deliveryProgress, project.Stage, taskCount, milestoneCount)
	if openRisks > 0 {
		summary += fmt.Sprintf(" %d open risks remain.", openRisks)
	}

	nextMilestone := gin.H{"name": nil, "dueDate": nil}
	if nextMilestoneName.Valid {
		nextMilestone["name"] = nextMilestoneName.String
	}
	if nextMilestoneDue.Valid {
		nextMilestone["dueDate"] = nextMilestoneDue.Time.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{
		"scope":   gin.H{"workspaceId": workspaceID, "projectId": projectID},
		"project": project,
		"summary": summary,
		"delivery": gin.H{
			"progress":            deliveryProgress,
			"taskCount":           taskCount,
			"completedTasks":      completedTasks,
			"inProgressTasks":     inProgressTasks,
			"milestoneCount":      milestoneCount,
			"completedMilestones": completedMilestones,
			"nextMilestone":       nextMilestone,
		},
		"budget": gin.H{
			"budget":             project.Budget,
			"spent":              project.Spent,
			"remaining":          project.Budget - project.Spent,
			"utilizationPercent": utilization,
		},
		"openRisks":       openRisks,
		"recommendations": recommendations,
		"generatedFrom":   "workspace-scoped PMS project, task, milestone, risk, and budget records",
	})
}

func dbGetProjectCriticalPath(c *gin.Context) {
	projectID := c.Param("id")
	workspaceID := workspaceIDContext(c)
	var projectExists bool
	if err := DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM pms.projects WHERE id = $1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL))
	`, projectID, workspaceID).Scan(&projectExists); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "critical path unavailable"})
		return
	}
	if !projectExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	type pathTask struct {
		ID           string
		WBSCode      string
		Title        string
		StartDate    sql.NullTime
		EndDate      sql.NullTime
		Dependencies []string
	}
	rows, err := DB.Query(`
		SELECT id, COALESCE(wbs_code, ''), title, start_date, end_date, COALESCE(dependencies, '{}')
		FROM pms.tasks WHERE project_id = $1 ORDER BY start_date NULLS LAST, id
	`, projectID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "critical path unavailable"})
		return
	}
	defer rows.Close()

	tasks := make([]pathTask, 0)
	byID := make(map[string]pathTask)
	byWBS := make(map[string]string)
	for rows.Next() {
		var task pathTask
		if err := rows.Scan(&task.ID, &task.WBSCode, &task.Title, &task.StartDate, &task.EndDate, pq.Array(&task.Dependencies)); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "critical path unavailable"})
			return
		}
		tasks = append(tasks, task)
		byID[task.ID] = task
		if task.WBSCode != "" {
			byWBS[task.WBSCode] = task.ID
		}
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "critical path unavailable"})
		return
	}

	resolveDependency := func(value string) string {
		value = strings.TrimSpace(value)
		if _, ok := byID[value]; ok {
			return value
		}
		return byWBS[value]
	}
	durationDays := func(task pathTask) int {
		if !task.StartDate.Valid || !task.EndDate.Valid || task.EndDate.Time.Before(task.StartDate.Time) {
			return 1
		}
		days := int(task.EndDate.Time.Sub(task.StartDate.Time).Hours()/24) + 1
		if days < 1 {
			return 1
		}
		return days
	}

	state := make(map[string]int)
	memoDuration := make(map[string]int)
	memoChain := make(map[string][]string)
	missingDependencies := make(map[string]bool)
	var longest func(string) (int, []string, error)
	longest = func(taskID string) (int, []string, error) {
		switch state[taskID] {
		case 1:
			return 0, nil, fmt.Errorf("dependency cycle detected at %s", taskID)
		case 2:
			return memoDuration[taskID], memoChain[taskID], nil
		}
		state[taskID] = 1
		task := byID[taskID]
		bestDuration := 0
		var bestChain []string
		for _, dependency := range task.Dependencies {
			dependencyID := resolveDependency(dependency)
			if dependencyID == "" {
				missingDependencies[strings.TrimSpace(dependency)] = true
				continue
			}
			dependencyDuration, dependencyChain, err := longest(dependencyID)
			if err != nil {
				return 0, nil, err
			}
			if dependencyDuration > bestDuration {
				bestDuration = dependencyDuration
				bestChain = dependencyChain
			}
		}
		bestChain = append(append([]string{}, bestChain...), taskID)
		bestDuration += durationDays(task)
		state[taskID] = 2
		memoDuration[taskID] = bestDuration
		memoChain[taskID] = bestChain
		return bestDuration, bestChain, nil
	}

	criticalDuration := 0
	var criticalIDs []string
	for _, task := range tasks {
		duration, chain, err := longest(task.ID)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "task dependency cycle detected", "detail": err.Error()})
			return
		}
		if duration > criticalDuration {
			criticalDuration = duration
			criticalIDs = chain
		}
	}

	criticalTasks := make([]gin.H, 0, len(criticalIDs))
	for position, taskID := range criticalIDs {
		task := byID[taskID]
		criticalTasks = append(criticalTasks, gin.H{
			"position":     position + 1,
			"id":           task.ID,
			"wbsCode":      task.WBSCode,
			"title":        task.Title,
			"durationDays": durationDays(task),
			"dependencies": task.Dependencies,
		})
	}
	missing := make([]string, 0, len(missingDependencies))
	for dependency := range missingDependencies {
		if dependency != "" {
			missing = append(missing, dependency)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"scope": gin.H{"workspaceId": workspaceID, "projectId": projectID},
		"criticalPath": gin.H{
			"durationDays": criticalDuration,
			"taskIds":      criticalIDs,
			"tasks":        criticalTasks,
		},
		"missingDependencies": missing,
		"taskCount":           len(tasks),
		"method":              "longest dependency chain using inclusive task date durations; undated tasks count as one day",
	})
}

func dbGetActivityTimeline(c *gin.Context) {
	rows, err := DB.Query(`
		SELECT a.id, a.project_id, a.user_name, a.action, a.entity, a.entity_id, a.details, a.timestamp
		FROM pms.audit_logs a
		ORDER BY a.timestamp DESC
		LIMIT 50
	`)
	if err != nil {
		c.JSON(200, []interface{}{})
		return
	}
	defer rows.Close()

	type ActivityItem struct {
		ID        string    `json:"id"`
		ProjectID string    `json:"projectId"`
		User      string    `json:"user"`
		Action    string    `json:"action"`
		Entity    string    `json:"entity"`
		EntityID  string    `json:"entityId"`
		Details   string    `json:"details"`
		Timestamp time.Time `json:"timestamp"`
	}

	var items []ActivityItem
	for rows.Next() {
		var item ActivityItem
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.User, &item.Action, &item.Entity, &item.EntityID, &item.Details, &item.Timestamp); err != nil {
			continue
		}
		items = append(items, item)
	}
	if items == nil {
		items = []ActivityItem{}
	}
	c.JSON(200, items)
}

func dbGetProjectDashboard(c *gin.Context) {
	projectID := c.Param("id")

	var p Project
	err := DB.QueryRow(`
		SELECT id, code, name, description, stage, progress, org, portfolio, programme,
		       owner, start_date, end_date, target_geo, tags, budget_total, spent_total,
		       risks_count, issues_count, created_time
		FROM pms.projects WHERE id = $1
	`, projectID).Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.Stage, &p.Progress,
		&p.Org, &p.Portfolio, &p.Programme, &p.Owner, &p.StartDate, &p.EndDate,
		&p.TargetGeo, &p.Tags, &p.BudgetTotal, &p.SpentTotal, &p.RisksCount, &p.IssuesCount, &p.CreatedTime)

	if err != nil {
		c.JSON(404, gin.H{"error": "Project not found"})
		return
	}

	// Task stats
	var totalTasks, doneTasks, inProgressTasks, todoTasks int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.tasks WHERE project_id = $1`, projectID).Scan(&totalTasks)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.tasks WHERE project_id = $1 AND status = 'Done'`, projectID).Scan(&doneTasks)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.tasks WHERE project_id = $1 AND status = 'In Progress'`, projectID).Scan(&inProgressTasks)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.tasks WHERE project_id = $1 AND status = 'Todo'`, projectID).Scan(&todoTasks)

	// Survey stats
	var totalSurveys, totalSubmissions, totalTarget int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.surveys WHERE project_id = $1`, projectID).Scan(&totalSurveys)
	DB.QueryRow(`SELECT COALESCE(SUM(submissions), 0) FROM pms.surveys WHERE project_id = $1`, projectID).Scan(&totalSubmissions)
	DB.QueryRow(`SELECT COALESCE(SUM(target_sample), 0) FROM pms.surveys WHERE project_id = $1`, projectID).Scan(&totalTarget)

	// Team stats
	var totalMembers int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.project_members WHERE project_id = $1`, projectID).Scan(&totalMembers)

	// Meeting stats
	var totalMeetings, completedMeetings int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.meetings WHERE project_id = $1`, projectID).Scan(&totalMeetings)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.meetings WHERE project_id = $1 AND status = 'Completed'`, projectID).Scan(&completedMeetings)

	// Document stats
	var totalDocuments int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.documents WHERE project_id = $1`, projectID).Scan(&totalDocuments)

	// Issue stats
	var openIssues, resolvedIssues int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.issues WHERE project_id = $1 AND status != 'Resolved'`, projectID).Scan(&openIssues)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.issues WHERE project_id = $1 AND status = 'Resolved'`, projectID).Scan(&resolvedIssues)

	// Milestone stats
	var totalMilestones, completedMilestones int
	DB.QueryRow(`SELECT COUNT(*) FROM pms.milestones WHERE project_id = $1`, projectID).Scan(&totalMilestones)
	DB.QueryRow(`SELECT COUNT(*) FROM pms.milestones WHERE project_id = $1 AND status = 'Completed'`, projectID).Scan(&completedMilestones)

	// Budget utilization
	budgetUtil := 0.0
	if p.BudgetTotal > 0 {
		budgetUtil = (p.SpentTotal / p.BudgetTotal) * 100
	}

	// Survey progress
	surveyProgress := 0.0
	if totalTarget > 0 {
		surveyProgress = (float64(totalSubmissions) / float64(totalTarget)) * 100
	}

	// Timeline adherence
	timelineAdherence := 0.0
	startMs := time.Now()
	endMs := time.Now()
	if p.StartDate != "" {
		if t, err := time.Parse("2006-01-02", p.StartDate); err == nil {
			startMs = t
		}
	}
	if p.EndDate != "" {
		if t, err := time.Parse("2006-01-02", p.EndDate); err == nil {
			endMs = t
		}
	}
	totalDuration := endMs.Sub(startMs).Hours() / 24
	elapsed := time.Since(startMs).Hours() / 24
	if totalDuration > 0 && elapsed > 0 {
		elapsedPct := (elapsed / totalDuration) * 100
		if elapsedPct > 0 {
			timelineAdherence = (p.Progress / elapsedPct) * 100
			if timelineAdherence > 100 {
				timelineAdherence = 100
			}
		}
	}

	c.JSON(200, gin.H{
		"project":             p,
		"totalTasks":          totalTasks,
		"doneTasks":           doneTasks,
		"inProgressTasks":     inProgressTasks,
		"todoTasks":           todoTasks,
		"totalSurveys":        totalSurveys,
		"totalSubmissions":    totalSubmissions,
		"totalTarget":         totalTarget,
		"surveyProgress":      surveyProgress,
		"totalMembers":        totalMembers,
		"totalMeetings":       totalMeetings,
		"completedMeetings":   completedMeetings,
		"totalDocuments":      totalDocuments,
		"openIssues":          openIssues,
		"resolvedIssues":      resolvedIssues,
		"totalMilestones":     totalMilestones,
		"completedMilestones": completedMilestones,
		"budgetUtilization":   budgetUtil,
		"timelineAdherence":   timelineAdherence,
	})
}

// ═══════════════════════════════════════════════════════════════════
// HELPER FUNCTIONS
// ═══════════════════════════════════════════════════════════════════

func dbGetMembers(projectID string) ([]ProjectMember, error) {
	rows, err := DB.Query(`SELECT id, project_id, name, role, email, avatar_url, department, phone, location, since FROM pms.project_members WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []ProjectMember
	for rows.Next() {
		var m ProjectMember
		var since sql.NullString
		rows.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Role, &m.Email, &m.AvatarUrl, &m.Department, &m.Phone, &m.Location, &since)
		m.Since = since.String
		members = append(members, m)
	}
	return members, nil
}

func dbGetTasks(projectID string) ([]Task, error) {
	rows, err := DB.Query(`SELECT id, project_id, parent_id, wbs_code, title, description, status, priority, start_date, end_date, progress, assigned_to, dependencies, COALESCE(is_milestone, FALSE), COALESCE(is_deliverable, FALSE) FROM pms.tasks WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.ParentID, &t.WBSCode, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.StartDate, &t.EndDate, &t.Progress, &t.AssignedTo, &t.Dependencies, &t.IsMilestone, &t.IsDeliverable); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func dbGetBudgets(projectID string) ([]BudgetLine, error) {
	rows, err := DB.Query(`SELECT id, project_id, category, description, source, amount, spent FROM pms.budget_lines WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var budgets []BudgetLine
	for rows.Next() {
		var b BudgetLine
		rows.Scan(&b.ID, &b.ProjectID, &b.Category, &b.Description, &b.Source, &b.Amount, &b.Spent)
		budgets = append(budgets, b)
	}
	return budgets, nil
}

func dbGetRisks(projectID string) ([]Risk, error) {
	rows, err := DB.Query(`SELECT id, project_id, description, category, probability, impact, mitigation, status, owner, due_date FROM pms.risks WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var risks []Risk
	for rows.Next() {
		var r Risk
		var dueDate sql.NullString
		rows.Scan(&r.ID, &r.ProjectID, &r.Description, &r.Category, &r.Probability, &r.Impact, &r.Mitigation, &r.Status, &r.Owner, &dueDate)
		r.DueDate = dueDate.String
		risks = append(risks, r)
	}
	return risks, nil
}

func dbGetDocuments(projectID string) ([]Document, error) {
	rows, err := DB.Query(`SELECT id, project_id, name, type, size, uploaded_by, uploaded_at, url, status FROM pms.documents WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		rows.Scan(&d.ID, &d.ProjectID, &d.Name, &d.Type, &d.Size, &d.UploadedBy, &d.UploadedAt, &d.Url, &d.Status)
		docs = append(docs, d)
	}
	return docs, nil
}

func dbGetMeetings(projectID string) ([]Meeting, error) {
	rows, err := DB.Query(`SELECT id, project_id, title, date_time, location, attendees, agenda, minutes, action_items, status, decisions FROM pms.meetings WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []Meeting
	for rows.Next() {
		var m Meeting
		rows.Scan(&m.ID, &m.ProjectID, &m.Title, &m.DateTime, &m.Location, &m.Attendees, &m.Agenda, &m.Minutes, &m.ActionItems, &m.Status, &m.Decisions)
		meetings = append(meetings, m)
	}
	return meetings, nil
}

func dbGetSurveys(projectID string) ([]Survey, error) {
	rows, err := DB.Query(`SELECT id, project_id, name, status, target_sample, submissions, progress FROM pms.surveys WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var surveys []Survey
	for rows.Next() {
		var s Survey
		rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Status, &s.TargetSample, &s.Submissions, &s.Progress)
		surveys = append(surveys, s)
	}
	return surveys, nil
}

func dbGetChats(projectID string) ([]ChatMessage, error) {
	rows, err := DB.Query(`SELECT id, project_id, channel, sender, role, message, timestamp FROM pms.chat_messages WHERE project_id = $1 ORDER BY timestamp ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []ChatMessage
	for rows.Next() {
		var ch ChatMessage
		rows.Scan(&ch.ID, &ch.ProjectID, &ch.Channel, &ch.Sender, &ch.Role, &ch.Message, &ch.Timestamp)
		chats = append(chats, ch)
	}
	return chats, nil
}

func dbGetHelpdesk(projectID string) ([]HelpDeskTicket, error) {
	rows, err := DB.Query(`SELECT id, project_id, title, description, status, priority, created_by, created_at FROM pms.helpdesk_tickets WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []HelpDeskTicket
	for rows.Next() {
		var h HelpDeskTicket
		rows.Scan(&h.ID, &h.ProjectID, &h.Title, &h.Description, &h.Status, &h.Priority, &h.CreatedBy, &h.CreatedAt)
		tickets = append(tickets, h)
	}
	return tickets, nil
}

// ─── Phase 4 Handlers: LogFrame, Theory of Change, Donors ─────────

func workspaceLogFrameReference(c *gin.Context, logFrameID string) bool {
	workspaceID := workspaceIDContext(c)
	if workspaceID == "" {
		return true
	}
	var exists bool
	err := DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM pms.logframes lf
			JOIN pms.projects p ON p.id = lf.project_id
			WHERE lf.id = $1 AND p.workspace_id = $2
		)`, logFrameID, workspaceID).Scan(&exists)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "workspace scope unavailable"})
		return false
	}
	if !exists {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "logframe not found in workspace"})
		return false
	}
	return true
}

func dbGetLogFrames(c *gin.Context) {
	projectID := c.Param("id")
	rows, err := DB.Query(`SELECT id, project_id, title, description, created_time FROM pms.logframes WHERE project_id = $1`, projectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logframes []LogFrame
	for rows.Next() {
		var lf LogFrame
		rows.Scan(&lf.ID, &lf.ProjectID, &lf.Title, &lf.Description, &lf.CreatedTime)
		// Fetch items for this logframe
		itemRows, err := DB.Query(`SELECT id, logframe_id, level, code, description, indicators, means_of_verification, assumptions, created_time FROM pms.logframe_items WHERE logframe_id = $1`, lf.ID)
		if err == nil {
			for itemRows.Next() {
				var item LogFrameItem
				var code sql.NullString
				itemRows.Scan(&item.ID, &item.LogFrameID, &item.Level, &code, &item.Description, &item.Indicators, &item.MeansOfVer, &item.Assumptions, &item.CreatedTime)
				item.Code = code.String
				lf.Items = append(lf.Items, item)
			}
			itemRows.Close()
		}
		logframes = append(logframes, lf)
	}
	c.JSON(200, logframes)
}

func dbCreateLogFrame(c *gin.Context) {
	var lf LogFrame
	if err := c.ShouldBindJSON(&lf); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(lf.ProjectID) == "" || strings.TrimSpace(lf.Title) == "" {
		c.JSON(400, gin.H{"error": "projectId and title are required"})
		return
	}
	if lf.ID == "" {
		lf.ID = generateUUID()
	}
	lf.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.logframes (id, project_id, title, description, created_time, workspace_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		lf.ID, lf.ProjectID, lf.Title, lf.Description, lf.CreatedTime, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, lf)
}

func dbCreateLogFrameItem(c *gin.Context) {
	var item LogFrameItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(item.LogFrameID) == "" || strings.TrimSpace(item.Level) == "" || strings.TrimSpace(item.Description) == "" {
		c.JSON(400, gin.H{"error": "logframeId, level, and description are required"})
		return
	}
	if !workspaceLogFrameReference(c, item.LogFrameID) {
		return
	}
	if item.ID == "" {
		item.ID = generateUUID()
	}
	item.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.logframe_items (id, logframe_id, level, code, description, indicators, means_of_verification, assumptions, created_time, workspace_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		item.ID, item.LogFrameID, item.Level, item.Code, item.Description, item.Indicators, item.MeansOfVer, item.Assumptions, item.CreatedTime, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, item)
}

func dbGetTheoryOfChange(c *gin.Context) {
	projectID := c.Param("id")
	var toc TheoryOfChange
	err := DB.QueryRow(`SELECT id, project_id, title, narrative, inputs, activities, outputs, short_term_outcomes, long_term_outcomes, impact, assumptions, created_time 
		FROM pms.theory_of_change WHERE project_id = $1`, projectID).
		Scan(&toc.ID, &toc.ProjectID, &toc.Title, &toc.Narrative, &toc.Inputs, &toc.Activities, &toc.Outputs, &toc.ShortTermOutcomes, &toc.LongTermOutcomes, &toc.Impact, &toc.Assumptions, &toc.CreatedTime)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(200, gin.H{"message": "No Theory of Change defined for this project yet", "projectId": projectID})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, toc)
}

func dbSaveTheoryOfChange(c *gin.Context) {
	var toc TheoryOfChange
	if err := c.ShouldBindJSON(&toc); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(toc.ProjectID) == "" || strings.TrimSpace(toc.Title) == "" {
		c.JSON(400, gin.H{"error": "projectId and title are required"})
		return
	}
	if toc.ID == "" {
		toc.ID = generateUUID()
	} else {
		var existingProjectID string
		err := DB.QueryRow(`SELECT project_id FROM pms.theory_of_change WHERE id = $1`, toc.ID).Scan(&existingProjectID)
		if err == nil && existingProjectID != toc.ProjectID {
			c.JSON(http.StatusNotFound, gin.H{"error": "theory of change not found for project"})
			return
		}
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "theory of change scope unavailable"})
			return
		}
	}
	toc.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO pms.theory_of_change (id, project_id, title, narrative, inputs, activities, outputs, short_term_outcomes, long_term_outcomes, impact, assumptions, created_time, workspace_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			narrative = EXCLUDED.narrative,
			inputs = EXCLUDED.inputs,
			activities = EXCLUDED.activities,
			outputs = EXCLUDED.outputs,
			short_term_outcomes = EXCLUDED.short_term_outcomes,
			long_term_outcomes = EXCLUDED.long_term_outcomes,
			impact = EXCLUDED.impact,
			assumptions = EXCLUDED.assumptions,
			workspace_id = EXCLUDED.workspace_id`,
		toc.ID, toc.ProjectID, toc.Title, toc.Narrative, toc.Inputs, toc.Activities, toc.Outputs, toc.ShortTermOutcomes, toc.LongTermOutcomes, toc.Impact, toc.Assumptions, toc.CreatedTime, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, toc)
}

func dbGetDonors(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, name, code, type, contact_person, email, phone, website, total_funding, currency, status, created_time, COALESCE(workspace_id, '') FROM pms.donors WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL) ORDER BY name ASC`, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var donors []Donor
	for rows.Next() {
		var d Donor
		var code, cp, email, phone, web sql.NullString
		rows.Scan(&d.ID, &d.Name, &code, &d.Type, &cp, &email, &phone, &web, &d.TotalFunding, &d.Currency, &d.Status, &d.CreatedTime, &d.WorkspaceID)
		d.Code = code.String
		d.ContactPerson = cp.String
		d.Email = email.String
		d.Phone = phone.String
		d.Website = web.String
		donors = append(donors, d)
	}
	c.JSON(200, donors)
}

func dbCreateDonor(c *gin.Context) {
	var d Donor
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if d.ID == "" {
		d.ID = generateUUID()
	}
	if d.Currency == "" {
		d.Currency = "USD"
	}
	if d.Status == "" {
		d.Status = "Active"
	}
	d.CreatedTime = time.Now()

	d.WorkspaceID = workspaceIDContext(c)
	_, err := DB.Exec(`INSERT INTO pms.donors (id, name, code, type, contact_person, email, phone, website, total_funding, currency, status, created_time, workspace_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		d.ID, d.Name, d.Code, d.Type, d.ContactPerson, d.Email, d.Phone, d.Website, d.TotalFunding, d.Currency, d.Status, d.CreatedTime, d.WorkspaceID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, d)
}

func dbUpdateDonor(c *gin.Context) {
	id := c.Param("id")
	var d Donor
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	d.WorkspaceID = workspaceIDContext(c)
	result, err := DB.Exec(`UPDATE pms.donors SET name=$1, code=$2, type=$3, contact_person=$4, email=$5, phone=$6, website=$7, total_funding=$8, currency=$9, status=$10 WHERE id=$11 AND (workspace_id = NULLIF($12, '') OR NULLIF($12, '') IS NULL)`,
		d.Name, d.Code, d.Type, d.ContactPerson, d.Email, d.Phone, d.Website, d.TotalFunding, d.Currency, d.Status, id, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "donor not found in workspace"})
		return
	}
	d.ID = id
	c.JSON(200, d)
}

func dbDeleteDonor(c *gin.Context) {
	id := c.Param("id")
	result, err := DB.Exec(`DELETE FROM pms.donors WHERE id = $1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)`, id, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "donor not found in workspace"})
		return
	}
	c.JSON(200, gin.H{"message": "Donor deleted successfully"})
}
