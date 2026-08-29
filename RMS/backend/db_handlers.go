package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// ─────────────────────────────────────────────────────────
//  Helpers
// ─────────────────────────────────────────────────────────

func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func writeAudit(researchID, action, user, details string) {
	id := newID() + "a"
	DB.Exec(`INSERT INTO rms.audit_logs (id, research_id, action, performed_by, details) VALUES ($1,$2,$3,$4,$5)`,
		id, researchID, action, user, details)
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullDate(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ─────────────────────────────────────────────────────────
//  Research Projects
// ─────────────────────────────────────────────────────────

func dbGetResearchProjects(c *gin.Context) {
	q := `SELECT id, code, name, description, type, stage, progress,
		principal_investigator, owner, organisation, portfolio, programme,
		COALESCE(start_date::TEXT,''), COALESCE(end_date::TEXT,''),
		COALESCE(target_geo,''), budget_total, spent_total,
		COALESCE(tags, '{}'), COALESCE(pms_project_id,''), COALESCE(statchat_room_id,''),
		created_time, updated_time, COALESCE(workspace_id, '')
		FROM rms.research_projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL) ORDER BY created_time DESC`

	rows, err := DB.Query(q, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []ResearchProject
	for rows.Next() {
		var r ResearchProject
		var tags pq.StringArray
		err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.Description, &r.Type, &r.Stage, &r.Progress,
			&r.PrincipalInvestigator, &r.Owner, &r.Organisation, &r.Portfolio, &r.Programme,
			&r.StartDate, &r.EndDate, &r.TargetGeo, &r.BudgetTotal, &r.SpentTotal,
			&tags, &r.PmsProjectID, &r.StatchatRoomID, &r.CreatedTime, &r.UpdatedTime, &r.WorkspaceID)
		if err != nil {
			continue
		}
		r.Tags = tags
		results = append(results, r)
	}
	if results == nil {
		results = []ResearchProject{}
	}
	c.JSON(200, results)
}

func dbCreateResearch(c *gin.Context) {
	var req CreateResearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	id := newID()
	if req.Type == "" {
		req.Type = "Research Project"
	}
	tags := pq.StringArray(req.Tags)

	_, err := DB.Exec(`
		INSERT INTO rms.research_projects
		(id, code, name, description, type, stage, principal_investigator, owner,
		organisation, portfolio, programme, start_date, end_date, target_geo,
		budget_total, tags, pms_project_id, workspace_id)
		VALUES ($1,$2,$3,$4,$5,'Research Idea',$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NULLIF($17,''))`,
		id, req.Code, req.Name, req.Description, req.Type,
		req.PrincipalInvestigator, req.Owner, req.Organisation,
		req.Portfolio, req.Programme,
		nullDate(req.StartDate), nullDate(req.EndDate),
		req.TargetGeo, req.BudgetTotal, tags, nullStr(req.PmsProjectID), workspaceIDContext(c))

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	writeAudit(id, "Research Study Created", req.Owner, req.Name)
	publishEvent("research.created", "research", id, map[string]interface{}{
		"code":  req.Code,
		"name":  req.Name,
		"owner": req.Owner,
	})

	// Return the created research
	var r ResearchProject
	var rtags pq.StringArray
	DB.QueryRow(`SELECT id, code, name, description, type, stage, progress,
		principal_investigator, owner, organisation, portfolio, programme,
		COALESCE(start_date::TEXT,''), COALESCE(end_date::TEXT,''),
		COALESCE(target_geo,''), budget_total, spent_total,
		COALESCE(tags, '{}'), COALESCE(pms_project_id,''), COALESCE(statchat_room_id,''),
		created_time, updated_time, COALESCE(workspace_id, '') FROM rms.research_projects WHERE id=$1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)`, id, workspaceIDContext(c)).Scan(
		&r.ID, &r.Code, &r.Name, &r.Description, &r.Type, &r.Stage, &r.Progress,
		&r.PrincipalInvestigator, &r.Owner, &r.Organisation, &r.Portfolio, &r.Programme,
		&r.StartDate, &r.EndDate, &r.TargetGeo, &r.BudgetTotal, &r.SpentTotal,
		&rtags, &r.PmsProjectID, &r.StatchatRoomID, &r.CreatedTime, &r.UpdatedTime, &r.WorkspaceID)
	r.Tags = rtags
	c.JSON(201, r)
}

func dbGetResearchWorkspace(c *gin.Context) {
	id := c.Param("id")

	var r ResearchProject
	var tags pq.StringArray
	err := DB.QueryRow(`SELECT id, code, name, description, type, stage, progress,
		principal_investigator, owner, organisation, portfolio, programme,
		COALESCE(start_date::TEXT,''), COALESCE(end_date::TEXT,''),
		COALESCE(target_geo,''), budget_total, spent_total,
		COALESCE(tags, '{}'), COALESCE(pms_project_id,''), COALESCE(statchat_room_id,''),
		created_time, updated_time, COALESCE(workspace_id, '') FROM rms.research_projects WHERE id=$1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)`, id, workspaceIDContext(c)).Scan(
		&r.ID, &r.Code, &r.Name, &r.Description, &r.Type, &r.Stage, &r.Progress,
		&r.PrincipalInvestigator, &r.Owner, &r.Organisation, &r.Portfolio, &r.Programme,
		&r.StartDate, &r.EndDate, &r.TargetGeo, &r.BudgetTotal, &r.SpentTotal,
		&tags, &r.PmsProjectID, &r.StatchatRoomID, &r.CreatedTime, &r.UpdatedTime, &r.WorkspaceID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Research study not found"})
		return
	}
	r.Tags = tags

	workspace := ResearchWorkspace{Research: &r}
	workspace.Members = fetchMembers(id)
	workspace.Proposals = fetchProposals(id)
	workspace.Ethics = fetchEthics(id)
	workspace.Grants = fetchGrants(id)
	workspace.Literature = fetchLiterature(id)
	workspace.Datasets = fetchDatasets(id)
	workspace.Publications = fetchPublications(id)
	workspace.Tasks = fetchTasks(id)
	workspace.Meetings = fetchMeetings(id)
	workspace.Risks = fetchRisks(id)
	workspace.ChatMessages, err = loadResearchDiscussion(c, r)
	workspace.ChatIntegrationReady = err == nil
	if workspace.ChatMessages == nil {
		workspace.ChatMessages = []ChatMessage{}
	}
	workspace.AuditLogs = fetchAuditLogs(id)

	c.JSON(200, workspace)
}

func dbUpdateResearch(c *gin.Context) {
	id := c.Param("id")
	var req UpdateResearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	tags := pq.StringArray(req.Tags)
	_, err := DB.Exec(`UPDATE rms.research_projects SET
		name=$1, description=$2, type=$3, principal_investigator=$4, owner=$5,
		organisation=$6, portfolio=$7, programme=$8,
		start_date=$9, end_date=$10, target_geo=$11,
		budget_total=$12, progress=$13, tags=$14, updated_time=NOW()
		WHERE id=$15 AND (workspace_id = NULLIF($16, '') OR NULLIF($16, '') IS NULL)`,
		req.Name, req.Description, req.Type, req.PrincipalInvestigator, req.Owner,
		req.Organisation, req.Portfolio, req.Programme,
		nullDate(req.StartDate), nullDate(req.EndDate), req.TargetGeo,
		req.BudgetTotal, req.Progress, tags, id, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	writeAudit(id, "Research Study Updated", req.Owner, "")
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteResearch(c *gin.Context) {
	id := c.Param("id")
	_, err := DB.Exec(`DELETE FROM rms.research_projects WHERE id=$1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)`, id, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func dbUpdateResearchStage(c *gin.Context) {
	id := c.Param("id")
	var req StageTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Calculate progress from stage
	stages := []string{
		"Research Idea", "Concept Note", "Research Proposal", "Protocol Development",
		"Internal Review", "Ethics Submission", "Ethics Approval", "Funding Approval",
		"Project Activation", "Survey Design", "Field Data Collection", "Data Validation",
		"Statistical Analysis", "Interpretation", "Report Writing", "Publication",
		"Knowledge Repository", "Archive",
	}
	progress := 0.0
	for i, s := range stages {
		if s == req.Stage {
			progress = float64(i+1) / float64(len(stages)) * 100
			break
		}
	}

	_, err := DB.Exec(`UPDATE rms.research_projects SET stage=$1, progress=$2, updated_time=NOW() WHERE id=$3 AND (workspace_id = NULLIF($4, '') OR NULLIF($4, '') IS NULL)`,
		req.Stage, progress, id, workspaceIDContext(c))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	user := req.User
	if user == "" {
		user = "System"
	}
	writeAudit(id, "Stage Transition", user, "→ "+req.Stage)
	publishEvent("research.stage_changed", "research", id, map[string]interface{}{
		"from_stage": "",
		"to_stage":   req.Stage,
		"progress":   progress,
		"user":       user,
	})
	c.JSON(200, gin.H{"success": true, "stage": req.Stage, "progress": progress})
}

func dbGetResearchDashboard(c *gin.Context) {
	id := c.Param("id")

	var memberCount, proposalCount, ethicsCount, grantCount, literatureCount, datasetCount, publicationCount, taskCount, riskCount int
	DB.QueryRow(`SELECT COUNT(*) FROM rms.research_members WHERE research_id=$1`, id).Scan(&memberCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.proposals WHERE research_id=$1`, id).Scan(&proposalCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.ethics_applications WHERE research_id=$1`, id).Scan(&ethicsCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.grants WHERE research_id=$1`, id).Scan(&grantCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.literature WHERE research_id=$1`, id).Scan(&literatureCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.datasets WHERE research_id=$1`, id).Scan(&datasetCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.publications WHERE research_id=$1`, id).Scan(&publicationCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.tasks WHERE research_id=$1`, id).Scan(&taskCount)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.risks WHERE research_id=$1 AND status='Open'`, id).Scan(&riskCount)

	var grantTotal, grantSpent float64
	DB.QueryRow(`SELECT COALESCE(SUM(budget_allocated),0), COALESCE(SUM(spent),0) FROM rms.grants WHERE research_id=$1`, id).Scan(&grantTotal, &grantSpent)

	c.JSON(200, gin.H{
		"members":      memberCount,
		"proposals":    proposalCount,
		"ethics":       ethicsCount,
		"grants":       grantCount,
		"literature":   literatureCount,
		"datasets":     datasetCount,
		"publications": publicationCount,
		"tasks":        taskCount,
		"openRisks":    riskCount,
		"grantTotal":   grantTotal,
		"grantSpent":   grantSpent,
	})
}

// ─────────────────────────────────────────────────────────
//  Global Dashboard
// ─────────────────────────────────────────────────────────

func dbGetDashboard(c *gin.Context) {
	var summary DashboardSummary
	workspaceID := workspaceIDContext(c)

	DB.QueryRow(`SELECT COUNT(*) FROM rms.research_projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&summary.TotalResearch)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.research_projects WHERE stage NOT IN ('Archive','Knowledge Repository') AND (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&summary.ActiveStudies)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.proposals x JOIN rms.research_projects p ON p.id=x.research_id WHERE x.status IN ('Draft','Under Review','Submitted') AND (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&summary.ProposalsPending)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.ethics_applications x JOIN rms.research_projects p ON p.id=x.research_id WHERE x.status IN ('Pending','Under Review') AND (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&summary.EthicsPending)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.publications x JOIN rms.research_projects p ON p.id=x.research_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&summary.TotalPublications)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.datasets x JOIN rms.research_projects p ON p.id=x.research_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&summary.TotalDatasets)
	DB.QueryRow(`SELECT COUNT(*) FROM rms.literature x JOIN rms.research_projects p ON p.id=x.research_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).Scan(&summary.TotalLiterature)
	DB.QueryRow(`SELECT COALESCE(SUM(budget_total),0), COALESCE(SUM(spent_total),0), COALESCE(AVG(progress),0) FROM rms.research_projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL)`, workspaceID).
		Scan(&summary.TotalBudget, &summary.TotalSpent, &summary.AvgProgress)

	// By stage breakdown
	rows, _ := DB.Query(`SELECT stage, COUNT(*) FROM rms.research_projects WHERE (workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL) GROUP BY stage ORDER BY COUNT(*) DESC`, workspaceID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var sc StageCount
			rows.Scan(&sc.Stage, &sc.Count)
			summary.ByStage = append(summary.ByStage, sc)
		}
	}
	if summary.ByStage == nil {
		summary.ByStage = []StageCount{}
	}

	// Recent activity
	summary.RecentActivity = []AuditLog{}
	arows, _ := DB.Query(`SELECT a.id, a.research_id, a.action, a.performed_by, COALESCE(a.details,''), a.created_time FROM rms.audit_logs a JOIN rms.research_projects p ON p.id=a.research_id WHERE (p.workspace_id = NULLIF($1, '') OR NULLIF($1, '') IS NULL) ORDER BY a.created_time DESC LIMIT 10`, workspaceID)
	if arows != nil {
		defer arows.Close()
		for arows.Next() {
			var al AuditLog
			arows.Scan(&al.ID, &al.ResearchID, &al.Action, &al.PerformedBy, &al.Details, &al.CreatedTime)
			summary.RecentActivity = append(summary.RecentActivity, al)
		}
	}

	c.JSON(200, summary)
}

// ─────────────────────────────────────────────────────────
//  Members
// ─────────────────────────────────────────────────────────

func fetchMembers(researchID string) []ResearchMember {
	rows, err := DB.Query(`SELECT id, research_id, name, role, email, COALESCE(avatar_url,''),
		COALESCE(department,''), COALESCE(phone,''), COALESCE(location,''),
		COALESCE(since::TEXT,''), created_time FROM rms.research_members WHERE research_id=$1`, researchID)
	if err != nil {
		return []ResearchMember{}
	}
	defer rows.Close()
	var results []ResearchMember
	for rows.Next() {
		var m ResearchMember
		rows.Scan(&m.ID, &m.ResearchID, &m.Name, &m.Role, &m.Email, &m.AvatarURL,
			&m.Department, &m.Phone, &m.Location, &m.Since, &m.CreatedTime)
		results = append(results, m)
	}
	if results == nil {
		return []ResearchMember{}
	}
	return results
}

func dbAddMember(c *gin.Context) {
	var req struct {
		ResearchID string `json:"researchId" binding:"required"`
		Name       string `json:"name" binding:"required"`
		Role       string `json:"role"`
		Email      string `json:"email"`
		Department string `json:"department"`
		Phone      string `json:"phone"`
		Location   string `json:"location"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := newID() + "m"
	_, err := DB.Exec(`INSERT INTO rms.research_members (id, research_id, name, role, email, department, phone, location)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, req.ResearchID, req.Name, req.Role, req.Email, req.Department, req.Phone, req.Location)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	writeAudit(req.ResearchID, "Member Added", "System", req.Name+" ("+req.Role+")")
	publishEvent("research.member_added", "research", req.ResearchID, map[string]interface{}{
		"member_id": id,
		"name":      req.Name,
		"role":      req.Role,
	})
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateMember(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name       string `json:"name"`
		Role       string `json:"role"`
		Email      string `json:"email"`
		Department string `json:"department"`
		Phone      string `json:"phone"`
		Location   string `json:"location"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.research_members SET name=$1, role=$2, email=$3, department=$4, phone=$5, location=$6 WHERE id=$7`,
		req.Name, req.Role, req.Email, req.Department, req.Phone, req.Location, id)
	c.JSON(200, gin.H{"success": true})
}

func dbRemoveMember(c *gin.Context) {
	id := c.Param("id")
	DB.Exec(`DELETE FROM rms.research_members WHERE id=$1`, id)
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Proposals
// ─────────────────────────────────────────────────────────

func fetchProposals(researchID string) []Proposal {
	rows, err := DB.Query(`SELECT id, research_id, title, version, status,
		COALESCE(description,''), COALESCE(draft_content,''), COALESCE(background,''),
		COALESCE(objectives,''), COALESCE(methodology,''),
		COALESCE(timeline_details,''), COALESCE(budget_details,''),
		COALESCE(reviewer_comments,''), COALESCE(submitted_by,''),
		COALESCE(reviewed_by,''), COALESCE(approved_by,''),
		created_time, updated_time FROM rms.proposals WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Proposal{}
	}
	defer rows.Close()
	var results []Proposal
	for rows.Next() {
		var p Proposal
		rows.Scan(&p.ID, &p.ResearchID, &p.Title, &p.Version, &p.Status,
			&p.Description, &p.DraftContent, &p.Background, &p.Objectives, &p.Methodology,
			&p.TimelineDetails, &p.BudgetDetails, &p.ReviewerComments,
			&p.SubmittedBy, &p.ReviewedBy, &p.ApprovedBy,
			&p.CreatedTime, &p.UpdatedTime)
		results = append(results, p)
	}
	if results == nil {
		return []Proposal{}
	}
	return results
}

func dbGetProposals(c *gin.Context) {
	c.JSON(200, fetchProposals(c.Param("id")))
}

func dbCreateProposal(c *gin.Context) {
	var req struct {
		ResearchID      string `json:"researchId" binding:"required"`
		Title           string `json:"title" binding:"required"`
		Description     string `json:"description"`
		DraftContent    string `json:"draftContent"`
		Background      string `json:"background"`
		Objectives      string `json:"objectives"`
		Methodology     string `json:"methodology"`
		TimelineDetails string `json:"timelineDetails"`
		BudgetDetails   string `json:"budgetDetails"`
		SubmittedBy     string `json:"submittedBy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := newID() + "p"
	_, err := DB.Exec(`INSERT INTO rms.proposals
		(id, research_id, title, description, draft_content, background, objectives, methodology, timeline_details, budget_details, submitted_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, req.ResearchID, req.Title, req.Description, req.DraftContent,
		req.Background, req.Objectives, req.Methodology,
		req.TimelineDetails, req.BudgetDetails, req.SubmittedBy)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	writeAudit(req.ResearchID, "Proposal Created", req.SubmittedBy, req.Title)
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateProposal(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status           string `json:"status"`
		Title            string `json:"title"`
		DraftContent     string `json:"draftContent"`
		Background       string `json:"background"`
		Objectives       string `json:"objectives"`
		Methodology      string `json:"methodology"`
		TimelineDetails  string `json:"timelineDetails"`
		BudgetDetails    string `json:"budgetDetails"`
		ReviewerComments string `json:"reviewerComments"`
		ReviewedBy       string `json:"reviewedBy"`
		ApprovedBy       string `json:"approvedBy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.proposals SET status=$1, title=$2, draft_content=$3, background=$4,
		objectives=$5, methodology=$6, timeline_details=$7, budget_details=$8,
		reviewer_comments=$9, reviewed_by=$10, approved_by=$11, updated_time=NOW() WHERE id=$12`,
		req.Status, req.Title, req.DraftContent, req.Background, req.Objectives,
		req.Methodology, req.TimelineDetails, req.BudgetDetails,
		req.ReviewerComments, req.ReviewedBy, req.ApprovedBy, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteProposal(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.proposals WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Ethics
// ─────────────────────────────────────────────────────────

func fetchEthics(researchID string) []EthicsApp {
	rows, err := DB.Query(`SELECT id, research_id, irb_name, status,
		COALESCE(submission_date::TEXT,''), COALESCE(approval_date::TEXT,''), COALESCE(expiry_date::TEXT,''),
		COALESCE(certificate_number,''), COALESCE(comments,''), COALESCE(amendment_notes,''),
		COALESCE(renewal_notes,''), COALESCE(compliance_notes,''),
		created_time, updated_time FROM rms.ethics_applications WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []EthicsApp{}
	}
	defer rows.Close()
	var results []EthicsApp
	for rows.Next() {
		var e EthicsApp
		rows.Scan(&e.ID, &e.ResearchID, &e.IRBName, &e.Status,
			&e.SubmissionDate, &e.ApprovalDate, &e.ExpiryDate,
			&e.CertificateNumber, &e.Comments, &e.AmendmentNotes,
			&e.RenewalNotes, &e.ComplianceNotes, &e.CreatedTime, &e.UpdatedTime)
		results = append(results, e)
	}
	if results == nil {
		return []EthicsApp{}
	}
	return results
}

func dbGetEthics(c *gin.Context) {
	c.JSON(200, fetchEthics(c.Param("id")))
}

func dbCreateEthics(c *gin.Context) {
	var req struct {
		ResearchID        string `json:"researchId" binding:"required"`
		IRBName           string `json:"irbName" binding:"required"`
		SubmissionDate    string `json:"submissionDate"`
		CertificateNumber string `json:"certificateNumber"`
		ExpiryDate        string `json:"expiryDate"`
		Comments          string `json:"comments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := newID() + "e"
	_, err := DB.Exec(`INSERT INTO rms.ethics_applications
		(id, research_id, irb_name, submission_date, certificate_number, expiry_date, comments)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, req.ResearchID, req.IRBName, nullDate(req.SubmissionDate),
		req.CertificateNumber, nullDate(req.ExpiryDate), req.Comments)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	writeAudit(req.ResearchID, "Ethics Application Created", "System", req.IRBName)
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateEthics(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status            string `json:"status"`
		IRBName           string `json:"irbName"`
		ApprovalDate      string `json:"approvalDate"`
		ExpiryDate        string `json:"expiryDate"`
		CertificateNumber string `json:"certificateNumber"`
		Comments          string `json:"comments"`
		AmendmentNotes    string `json:"amendmentNotes"`
		RenewalNotes      string `json:"renewalNotes"`
		ComplianceNotes   string `json:"complianceNotes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.ethics_applications SET status=$1, irb_name=$2,
		approval_date=$3, expiry_date=$4, certificate_number=$5, comments=$6,
		amendment_notes=$7, renewal_notes=$8, compliance_notes=$9, updated_time=NOW() WHERE id=$10`,
		req.Status, req.IRBName, nullDate(req.ApprovalDate), nullDate(req.ExpiryDate),
		req.CertificateNumber, req.Comments, req.AmendmentNotes, req.RenewalNotes,
		req.ComplianceNotes, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteEthics(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.ethics_applications WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Grants
// ─────────────────────────────────────────────────────────

func fetchGrants(researchID string) []Grant {
	rows, err := DB.Query(`SELECT id, research_id, opportunity_name, status,
		COALESCE(donor_name,''), COALESCE(contract_number,''),
		budget_allocated, spent, COALESCE(currency,'USD'),
		COALESCE(start_date::TEXT,''), COALESCE(end_date::TEXT,''),
		COALESCE(reporting_schedule,''), COALESCE(deliverables,''), COALESCE(notes,''),
		created_time, updated_time FROM rms.grants WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Grant{}
	}
	defer rows.Close()
	var results []Grant
	for rows.Next() {
		var g Grant
		rows.Scan(&g.ID, &g.ResearchID, &g.OpportunityName, &g.Status,
			&g.DonorName, &g.ContractNumber, &g.BudgetAllocated, &g.Spent, &g.Currency,
			&g.StartDate, &g.EndDate, &g.ReportingSchedule, &g.Deliverables, &g.Notes,
			&g.CreatedTime, &g.UpdatedTime)
		results = append(results, g)
	}
	if results == nil {
		return []Grant{}
	}
	return results
}

func dbGetGrants(c *gin.Context) {
	c.JSON(200, fetchGrants(c.Param("id")))
}

func dbCreateGrant(c *gin.Context) {
	var req struct {
		ResearchID      string  `json:"researchId" binding:"required"`
		OpportunityName string  `json:"opportunityName" binding:"required"`
		DonorName       string  `json:"donorName"`
		ContractNumber  string  `json:"contractNumber"`
		BudgetAllocated float64 `json:"budgetAllocated"`
		Currency        string  `json:"currency"`
		StartDate       string  `json:"startDate"`
		EndDate         string  `json:"endDate"`
		Notes           string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	id := newID() + "g"
	_, err := DB.Exec(`INSERT INTO rms.grants
		(id, research_id, opportunity_name, donor_name, contract_number, budget_allocated, currency, start_date, end_date, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, req.ResearchID, req.OpportunityName, req.DonorName,
		req.ContractNumber, req.BudgetAllocated, req.Currency,
		nullDate(req.StartDate), nullDate(req.EndDate), req.Notes)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// Update project budget total
	DB.Exec(`UPDATE rms.research_projects SET budget_total = (SELECT COALESCE(SUM(budget_allocated),0) FROM rms.grants WHERE research_id=$1) WHERE id=$1`, req.ResearchID)
	writeAudit(req.ResearchID, "Grant Created", "System", req.OpportunityName)
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateGrant(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status          string  `json:"status"`
		OpportunityName string  `json:"opportunityName"`
		DonorName       string  `json:"donorName"`
		ContractNumber  string  `json:"contractNumber"`
		BudgetAllocated float64 `json:"budgetAllocated"`
		Spent           float64 `json:"spent"`
		Notes           string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.grants SET status=$1, opportunity_name=$2, donor_name=$3,
		contract_number=$4, budget_allocated=$5, spent=$6, notes=$7, updated_time=NOW() WHERE id=$8`,
		req.Status, req.OpportunityName, req.DonorName, req.ContractNumber,
		req.BudgetAllocated, req.Spent, req.Notes, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteGrant(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.grants WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Literature
// ─────────────────────────────────────────────────────────

func fetchLiterature(researchID string) []LiteratureItem {
	rows, err := DB.Query(`SELECT id, research_id, title, COALESCE(authors,''),
		COALESCE(journal,''), COALESCE(doi,''), COALESCE(pub_year,0),
		COALESCE(citation,''), COALESCE(keywords,''), COALESCE(category,''),
		COALESCE(tags,'{}'), COALESCE(notes,''), COALESCE(url,''),
		reading_status, created_time FROM rms.literature WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []LiteratureItem{}
	}
	defer rows.Close()
	var results []LiteratureItem
	for rows.Next() {
		var l LiteratureItem
		var tags pq.StringArray
		rows.Scan(&l.ID, &l.ResearchID, &l.Title, &l.Authors, &l.Journal, &l.DOI, &l.PubYear,
			&l.Citation, &l.Keywords, &l.Category, &tags, &l.Notes, &l.URL, &l.ReadingStatus, &l.CreatedTime)
		l.Tags = tags
		results = append(results, l)
	}
	if results == nil {
		return []LiteratureItem{}
	}
	return results
}

func dbGetLiterature(c *gin.Context) {
	c.JSON(200, fetchLiterature(c.Param("id")))
}

func dbCreateLiterature(c *gin.Context) {
	var req struct {
		ResearchID string   `json:"researchId" binding:"required"`
		Title      string   `json:"title" binding:"required"`
		Authors    string   `json:"authors"`
		Journal    string   `json:"journal"`
		DOI        string   `json:"doi"`
		PubYear    int      `json:"pubYear"`
		Citation   string   `json:"citation"`
		Keywords   string   `json:"keywords"`
		Category   string   `json:"category"`
		Tags       []string `json:"tags"`
		Notes      string   `json:"notes"`
		URL        string   `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := newID() + "l"
	tags := pq.StringArray(req.Tags)
	_, err := DB.Exec(`INSERT INTO rms.literature
		(id, research_id, title, authors, journal, doi, pub_year, citation, keywords, category, tags, notes, url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		id, req.ResearchID, req.Title, req.Authors, req.Journal, req.DOI,
		req.PubYear, req.Citation, req.Keywords, req.Category, tags, req.Notes, req.URL)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	writeAudit(req.ResearchID, "Literature Added", "System", req.Title)
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateLiterature(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		ReadingStatus string `json:"readingStatus"`
		Notes         string `json:"notes"`
		Keywords      string `json:"keywords"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.literature SET reading_status=$1, notes=$2, keywords=$3 WHERE id=$4`,
		req.ReadingStatus, req.Notes, req.Keywords, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteLiterature(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.literature WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Datasets
// ─────────────────────────────────────────────────────────

func fetchDatasets(researchID string) []Dataset {
	rows, err := DB.Query(`SELECT id, research_id, name, COALESCE(description,''),
		version, status, COALESCE(source_type,''), COALESCE(collection_method,''),
		COALESCE(metadata_info,''), COALESCE(variables_dict,''),
		COALESCE(access_level,'Internal'), COALESCE(download_url,''),
		COALESCE(statcollect_id,''), created_time, updated_time
		FROM rms.datasets WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Dataset{}
	}
	defer rows.Close()
	var results []Dataset
	for rows.Next() {
		var d Dataset
		rows.Scan(&d.ID, &d.ResearchID, &d.Name, &d.Description, &d.Version, &d.Status,
			&d.SourceType, &d.CollectionMethod, &d.MetadataInfo, &d.VariablesDict,
			&d.AccessLevel, &d.DownloadURL, &d.StatcollectID, &d.CreatedTime, &d.UpdatedTime)
		results = append(results, d)
	}
	if results == nil {
		return []Dataset{}
	}
	return results
}

func dbGetDatasets(c *gin.Context) {
	c.JSON(200, fetchDatasets(c.Param("id")))
}

func dbCreateDataset(c *gin.Context) {
	var req struct {
		ResearchID       string `json:"researchId" binding:"required"`
		Name             string `json:"name" binding:"required"`
		Description      string `json:"description"`
		SourceType       string `json:"sourceType"`
		CollectionMethod string `json:"collectionMethod"`
		MetadataInfo     string `json:"metadataInfo"`
		VariablesDict    string `json:"variablesDict"`
		AccessLevel      string `json:"accessLevel"`
		StatcollectID    string `json:"statcollectId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.AccessLevel == "" {
		req.AccessLevel = "Internal"
	}
	id := newID() + "d"
	_, err := DB.Exec(`INSERT INTO rms.datasets
		(id, research_id, name, description, source_type, collection_method, metadata_info, variables_dict, access_level, statcollect_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, req.ResearchID, req.Name, req.Description, req.SourceType, req.CollectionMethod,
		req.MetadataInfo, req.VariablesDict, req.AccessLevel, req.StatcollectID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	writeAudit(req.ResearchID, "Dataset Registered", "System", req.Name)
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateDataset(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status        string `json:"status"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		MetadataInfo  string `json:"metadataInfo"`
		VariablesDict string `json:"variablesDict"`
		AccessLevel   string `json:"accessLevel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.datasets SET status=$1, name=$2, description=$3, metadata_info=$4,
		variables_dict=$5, access_level=$6, updated_time=NOW() WHERE id=$7`,
		req.Status, req.Name, req.Description, req.MetadataInfo, req.VariablesDict, req.AccessLevel, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteDataset(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.datasets WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Publications
// ─────────────────────────────────────────────────────────

func fetchPublications(researchID string) []Publication {
	rows, err := DB.Query(`SELECT id, research_id, title, COALESCE(pub_type,'Manuscript'),
		COALESCE(authors,''), COALESCE(journal,''), status,
		COALESCE(peer_review_comments,''), COALESCE(revision_history,''),
		COALESCE(doi,''), COALESCE(acceptance_date::TEXT,''), COALESCE(published_date::TEXT,''),
		COALESCE(affiliation,''), created_time, updated_time
		FROM rms.publications WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Publication{}
	}
	defer rows.Close()
	var results []Publication
	for rows.Next() {
		var p Publication
		rows.Scan(&p.ID, &p.ResearchID, &p.Title, &p.PubType, &p.Authors, &p.Journal, &p.Status,
			&p.PeerReviewComments, &p.RevisionHistory, &p.DOI,
			&p.AcceptanceDate, &p.PublishedDate, &p.Affiliation,
			&p.CreatedTime, &p.UpdatedTime)
		results = append(results, p)
	}
	if results == nil {
		return []Publication{}
	}
	return results
}

func dbGetPublications(c *gin.Context) {
	c.JSON(200, fetchPublications(c.Param("id")))
}

func dbCreatePublication(c *gin.Context) {
	var req struct {
		ResearchID  string `json:"researchId" binding:"required"`
		Title       string `json:"title" binding:"required"`
		PubType     string `json:"pubType"`
		Authors     string `json:"authors"`
		Journal     string `json:"journal"`
		Affiliation string `json:"affiliation"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.PubType == "" {
		req.PubType = "Manuscript"
	}
	id := newID() + "pub"
	_, err := DB.Exec(`INSERT INTO rms.publications (id, research_id, title, pub_type, authors, journal, affiliation)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, req.ResearchID, req.Title, req.PubType, req.Authors, req.Journal, req.Affiliation)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	writeAudit(req.ResearchID, "Publication Created", "System", req.Title)
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdatePublication(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status             string `json:"status"`
		PeerReviewComments string `json:"peerReviewComments"`
		DOI                string `json:"doi"`
		AcceptanceDate     string `json:"acceptanceDate"`
		PublishedDate      string `json:"publishedDate"`
		Journal            string `json:"journal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.publications SET status=$1, peer_review_comments=$2, doi=$3,
		acceptance_date=$4, published_date=$5, journal=$6, updated_time=NOW() WHERE id=$7`,
		req.Status, req.PeerReviewComments, req.DOI,
		nullDate(req.AcceptanceDate), nullDate(req.PublishedDate), req.Journal, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeletePublication(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.publications WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Tasks
// ─────────────────────────────────────────────────────────

func fetchTasks(researchID string) []ResearchTask {
	rows, err := DB.Query(`SELECT id, research_id, title, COALESCE(description,''),
		status, priority, COALESCE(start_date::TEXT,''), COALESCE(end_date::TEXT,''),
		progress, COALESCE(assigned_to,''), created_time, updated_time
		FROM rms.tasks WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []ResearchTask{}
	}
	defer rows.Close()
	var results []ResearchTask
	for rows.Next() {
		var t ResearchTask
		rows.Scan(&t.ID, &t.ResearchID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.StartDate, &t.EndDate, &t.Progress, &t.AssignedTo, &t.CreatedTime, &t.UpdatedTime)
		results = append(results, t)
	}
	if results == nil {
		return []ResearchTask{}
	}
	return results
}

func dbCreateTask(c *gin.Context) {
	var req struct {
		ResearchID  string  `json:"researchId" binding:"required"`
		Title       string  `json:"title" binding:"required"`
		Description string  `json:"description"`
		Status      string  `json:"status"`
		Priority    string  `json:"priority"`
		StartDate   string  `json:"startDate"`
		EndDate     string  `json:"endDate"`
		AssignedTo  string  `json:"assignedTo"`
		Progress    float64 `json:"progress"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "Todo"
	}
	if req.Priority == "" {
		req.Priority = "Medium"
	}
	id := newID() + "t"
	_, err := DB.Exec(`INSERT INTO rms.tasks
		(id, research_id, title, description, status, priority, start_date, end_date, assigned_to, progress)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, req.ResearchID, req.Title, req.Description, req.Status, req.Priority,
		nullDate(req.StartDate), nullDate(req.EndDate), req.AssignedTo, req.Progress)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateTask(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status     string  `json:"status"`
		Priority   string  `json:"priority"`
		Progress   float64 `json:"progress"`
		AssignedTo string  `json:"assignedTo"`
		Title      string  `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.tasks SET status=$1, priority=$2, progress=$3, assigned_to=$4, title=$5, updated_time=NOW() WHERE id=$6`,
		req.Status, req.Priority, req.Progress, req.AssignedTo, req.Title, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteTask(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.tasks WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Meetings
// ─────────────────────────────────────────────────────────

func fetchMeetings(researchID string) []Meeting {
	rows, err := DB.Query(`SELECT id, research_id, title,
		COALESCE(date_time::TEXT,''), COALESCE(location,''),
		COALESCE(agenda,''), COALESCE(decisions,'{}'), COALESCE(attendees,'{}'),
		status, created_time FROM rms.meetings WHERE research_id=$1 ORDER BY date_time DESC`, researchID)
	if err != nil {
		return []Meeting{}
	}
	defer rows.Close()
	var results []Meeting
	for rows.Next() {
		var m Meeting
		var decisions, attendees pq.StringArray
		rows.Scan(&m.ID, &m.ResearchID, &m.Title, &m.DateTime, &m.Location,
			&m.Agenda, &decisions, &attendees, &m.Status, &m.CreatedTime)
		m.Decisions = decisions
		m.Attendees = attendees
		results = append(results, m)
	}
	if results == nil {
		return []Meeting{}
	}
	return results
}

func dbCreateMeeting(c *gin.Context) {
	var req struct {
		ResearchID string   `json:"researchId" binding:"required"`
		Title      string   `json:"title" binding:"required"`
		DateTime   string   `json:"dateTime"`
		Location   string   `json:"location"`
		Agenda     string   `json:"agenda"`
		Attendees  []string `json:"attendees"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := newID() + "mt"
	attendees := pq.StringArray(req.Attendees)
	_, err := DB.Exec(`INSERT INTO rms.meetings (id, research_id, title, date_time, location, agenda, attendees)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, req.ResearchID, req.Title, nullDate(req.DateTime), req.Location, req.Agenda, attendees)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateMeeting(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status    string   `json:"status"`
		Decisions []string `json:"decisions"`
		Agenda    string   `json:"agenda"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	decisions := pq.StringArray(req.Decisions)
	DB.Exec(`UPDATE rms.meetings SET status=$1, decisions=$2, agenda=$3 WHERE id=$4`,
		req.Status, decisions, req.Agenda, id)
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Risks
// ─────────────────────────────────────────────────────────

func fetchRisks(researchID string) []Risk {
	rows, err := DB.Query(`SELECT id, research_id, description, category,
		COALESCE(probability,''), COALESCE(impact,''), COALESCE(mitigation,''),
		status, COALESCE(owner,''), COALESCE(due_date::TEXT,''),
		created_time FROM rms.risks WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Risk{}
	}
	defer rows.Close()
	var results []Risk
	for rows.Next() {
		var r Risk
		rows.Scan(&r.ID, &r.ResearchID, &r.Description, &r.Category,
			&r.Probability, &r.Impact, &r.Mitigation,
			&r.Status, &r.Owner, &r.DueDate, &r.CreatedTime)
		results = append(results, r)
	}
	if results == nil {
		return []Risk{}
	}
	return results
}

func dbCreateRisk(c *gin.Context) {
	var req struct {
		ResearchID  string `json:"researchId" binding:"required"`
		Description string `json:"description" binding:"required"`
		Category    string `json:"category"`
		Probability string `json:"probability"`
		Impact      string `json:"impact"`
		Mitigation  string `json:"mitigation"`
		Owner       string `json:"owner"`
		DueDate     string `json:"dueDate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Impact == "" {
		req.Impact = "Medium"
	}
	if req.Probability == "" {
		req.Probability = "Medium"
	}
	id := newID() + "r"
	_, err := DB.Exec(`INSERT INTO rms.risks (id, research_id, description, category, probability, impact, mitigation, owner, due_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, req.ResearchID, req.Description, req.Category, req.Probability, req.Impact,
		req.Mitigation, req.Owner, nullDate(req.DueDate))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateRisk(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status     string `json:"status"`
		Mitigation string `json:"mitigation"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.risks SET status=$1, mitigation=$2 WHERE id=$3`, req.Status, req.Mitigation, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteRisk(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.risks WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Issues
// ─────────────────────────────────────────────────────────

func fetchIssues(researchID string) []Issue {
	rows, err := DB.Query(`SELECT id, research_id, title, COALESCE(description,''),
		COALESCE(category,''), COALESCE(priority,'Medium'), status,
		COALESCE(owner,''), COALESCE(due_date::TEXT,''), COALESCE(resolution,''),
		created_time, updated_time FROM rms.issues WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Issue{}
	}
	defer rows.Close()
	var results []Issue
	for rows.Next() {
		var i Issue
		rows.Scan(&i.ID, &i.ResearchID, &i.Title, &i.Description, &i.Category,
			&i.Priority, &i.Status, &i.Owner, &i.DueDate, &i.Resolution,
			&i.CreatedTime, &i.UpdatedTime)
		results = append(results, i)
	}
	if results == nil {
		return []Issue{}
	}
	return results
}

func dbGetIssues(c *gin.Context) {
	c.JSON(200, fetchIssues(c.Param("id")))
}

func dbCreateIssue(c *gin.Context) {
	var req struct {
		ResearchID  string `json:"researchId" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
		Status      string `json:"status"`
		Owner       string `json:"owner"`
		DueDate     string `json:"dueDate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Priority == "" {
		req.Priority = "Medium"
	}
	if req.Status == "" {
		req.Status = "Open"
	}
	id := newID() + "i"
	_, err := DB.Exec(`INSERT INTO rms.issues (id, research_id, title, description, category, priority, status, owner, due_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, req.ResearchID, req.Title, req.Description, req.Category, req.Priority, req.Status, req.Owner, nullDate(req.DueDate))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateIssue(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status     string `json:"status"`
		Priority   string `json:"priority"`
		Resolution string `json:"resolution"`
		Owner      string `json:"owner"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.issues SET status=$1, priority=$2, resolution=$3, owner=$4, updated_time=NOW() WHERE id=$5`,
		req.Status, req.Priority, req.Resolution, req.Owner, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteIssue(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.issues WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Documents
// ─────────────────────────────────────────────────────────

func fetchDocuments(researchID string) []Document {
	rows, err := DB.Query(`SELECT id, research_id, name, COALESCE(type,''), COALESCE(size,''),
		COALESCE(uploaded_by,''), uploaded_at, COALESCE(url,''), COALESCE(status,'Draft'),
		created_time FROM rms.documents WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Document{}
	}
	defer rows.Close()
	var results []Document
	for rows.Next() {
		var d Document
		rows.Scan(&d.ID, &d.ResearchID, &d.Name, &d.Type, &d.Size,
			&d.UploadedBy, &d.UploadedAt, &d.URL, &d.Status, &d.CreatedTime)
		results = append(results, d)
	}
	if results == nil {
		return []Document{}
	}
	return results
}

func dbGetDocuments(c *gin.Context) {
	c.JSON(200, fetchDocuments(c.Param("id")))
}

func dbCreateDocument(c *gin.Context) {
	var req struct {
		ResearchID string `json:"researchId" binding:"required"`
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type"`
		Size       string `json:"size"`
		UploadedBy string `json:"uploadedBy"`
		URL        string `json:"url"`
		Status     string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "Draft"
	}
	id := newID() + "doc"
	_, err := DB.Exec(`INSERT INTO rms.documents (id, research_id, name, type, size, uploaded_by, url, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, req.ResearchID, req.Name, req.Type, req.Size, req.UploadedBy, req.URL, req.Status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbDeleteDocument(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.documents WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Surveys
// ─────────────────────────────────────────────────────────

func fetchSurveys(researchID string) []Survey {
	rows, err := DB.Query(`SELECT id, research_id, name, status,
		target_sample, submissions, progress, created_time
		FROM rms.surveys WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Survey{}
	}
	defer rows.Close()
	var results []Survey
	for rows.Next() {
		var s Survey
		rows.Scan(&s.ID, &s.ResearchID, &s.Name, &s.Status,
			&s.TargetSample, &s.Submissions, &s.Progress, &s.CreatedTime)
		results = append(results, s)
	}
	if results == nil {
		return []Survey{}
	}
	return results
}

func dbGetSurveys(c *gin.Context) {
	c.JSON(200, fetchSurveys(c.Param("id")))
}

func dbCreateSurvey(c *gin.Context) {
	var req struct {
		ResearchID   string `json:"researchId" binding:"required"`
		Name         string `json:"name" binding:"required"`
		Status       string `json:"status"`
		TargetSample int    `json:"targetSample"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "Template"
	}
	if req.TargetSample == 0 {
		req.TargetSample = 1000
	}
	id := newID() + "s"
	_, err := DB.Exec(`INSERT INTO rms.surveys (id, research_id, name, status, target_sample)
		VALUES ($1,$2,$3,$4,$5)`,
		id, req.ResearchID, req.Name, req.Status, req.TargetSample)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateSurvey(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status       string  `json:"status"`
		Submissions  int     `json:"submissions"`
		Progress     float64 `json:"progress"`
		TargetSample int     `json:"targetSample"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	DB.Exec(`UPDATE rms.surveys SET status=$1, submissions=$2, progress=$3, target_sample=$4 WHERE id=$5`,
		req.Status, req.Submissions, req.Progress, req.TargetSample, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteSurvey(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.surveys WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Reports
// ─────────────────────────────────────────────────────────

func fetchReports(researchID string) []Report {
	rows, err := DB.Query(`SELECT id, research_id, title, COALESCE(type,''), COALESCE(format,'PDF'),
		COALESCE(generated_by,''), generated_at, COALESCE(content,''), COALESCE(status,'Generated'),
		created_time FROM rms.reports WHERE research_id=$1 ORDER BY created_time DESC`, researchID)
	if err != nil {
		return []Report{}
	}
	defer rows.Close()
	var results []Report
	for rows.Next() {
		var r Report
		rows.Scan(&r.ID, &r.ResearchID, &r.Title, &r.Type, &r.Format,
			&r.GeneratedBy, &r.GeneratedAt, &r.Content, &r.Status, &r.CreatedTime)
		results = append(results, r)
	}
	if results == nil {
		return []Report{}
	}
	return results
}

func dbGetReports(c *gin.Context) {
	c.JSON(200, fetchReports(c.Param("id")))
}

func dbCreateReport(c *gin.Context) {
	var req struct {
		ResearchID  string `json:"researchId" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Type        string `json:"type"`
		Format      string `json:"format"`
		GeneratedBy string `json:"generatedBy"`
		Content     string `json:"content"`
		Status      string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Format == "" {
		req.Format = "PDF"
	}
	if req.Status == "" {
		req.Status = "Generated"
	}
	id := newID() + "rep"
	_, err := DB.Exec(`INSERT INTO rms.reports (id, research_id, title, type, format, generated_by, content, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, req.ResearchID, req.Title, req.Type, req.Format, req.GeneratedBy, req.Content, req.Status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbDeleteReport(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.reports WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Calendar Events
// ─────────────────────────────────────────────────────────

func fetchCalendarEvents(researchID string) []CalendarEvent {
	rows, err := DB.Query(`SELECT id, research_id, title, COALESCE(description,''),
		COALESCE(event_type,''), COALESCE(start_time::TEXT,''), COALESCE(end_time::TEXT,''),
		COALESCE(location,''), COALESCE(attendees,'{}'),
		created_time FROM rms.calendar_events WHERE research_id=$1 ORDER BY start_time ASC`, researchID)
	if err != nil {
		return []CalendarEvent{}
	}
	defer rows.Close()
	var results []CalendarEvent
	for rows.Next() {
		var ce CalendarEvent
		var attendees pq.StringArray
		rows.Scan(&ce.ID, &ce.ResearchID, &ce.Title, &ce.Description, &ce.EventType,
			&ce.StartTime, &ce.EndTime, &ce.Location, &attendees, &ce.CreatedTime)
		ce.Attendees = attendees
		results = append(results, ce)
	}
	if results == nil {
		return []CalendarEvent{}
	}
	return results
}

func dbGetCalendarEvents(c *gin.Context) {
	c.JSON(200, fetchCalendarEvents(c.Param("id")))
}

func dbCreateCalendarEvent(c *gin.Context) {
	var req struct {
		ResearchID  string   `json:"researchId" binding:"required"`
		Title       string   `json:"title" binding:"required"`
		Description string   `json:"description"`
		EventType   string   `json:"eventType"`
		StartTime   string   `json:"startTime"`
		EndTime     string   `json:"endTime"`
		Location    string   `json:"location"`
		Attendees   []string `json:"attendees"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := newID() + "ev"
	attendees := pq.StringArray(req.Attendees)
	_, err := DB.Exec(`INSERT INTO rms.calendar_events (id, research_id, title, description, event_type, start_time, end_time, location, attendees)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, req.ResearchID, req.Title, req.Description, req.EventType, req.StartTime, req.EndTime, req.Location, attendees)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "success": true})
}

func dbUpdateCalendarEvent(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		EventType   string   `json:"eventType"`
		StartTime   string   `json:"startTime"`
		EndTime     string   `json:"endTime"`
		Location    string   `json:"location"`
		Attendees   []string `json:"attendees"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	attendees := pq.StringArray(req.Attendees)
	DB.Exec(`UPDATE rms.calendar_events SET title=$1, description=$2, event_type=$3,
		start_time=$4, end_time=$5, location=$6, attendees=$7 WHERE id=$8`,
		req.Title, req.Description, req.EventType, req.StartTime, req.EndTime, req.Location, attendees, id)
	c.JSON(200, gin.H{"success": true})
}

func dbDeleteCalendarEvent(c *gin.Context) {
	DB.Exec(`DELETE FROM rms.calendar_events WHERE id=$1`, c.Param("id"))
	c.JSON(200, gin.H{"success": true})
}

// ─────────────────────────────────────────────────────────
//  Chat Messages
// ─────────────────────────────────────────────────────────

func fetchChatMessages(researchID string) []ChatMessage {
	rows, err := DB.Query(`SELECT id, research_id, sender, channel, message, created_time
		FROM rms.chat_messages WHERE research_id=$1 ORDER BY created_time ASC LIMIT 200`, researchID)
	if err != nil {
		return []ChatMessage{}
	}
	defer rows.Close()
	var results []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		rows.Scan(&msg.ID, &msg.ResearchID, &msg.Sender, &msg.Channel, &msg.Message, &msg.CreatedTime)
		results = append(results, msg)
	}
	if results == nil {
		return []ChatMessage{}
	}
	return results
}

func dbGetActivityTimeline(c *gin.Context) {
	rows, err := DB.Query(`
		SELECT id, research_id, action, performed_by, COALESCE(details,''), created_time
		FROM rms.audit_logs
		ORDER BY created_time DESC
		LIMIT 50
	`)
	if err != nil {
		c.JSON(200, []interface{}{})
		return
	}
	defer rows.Close()

	type ActivityItem struct {
		ID         string    `json:"id"`
		ResearchID string    `json:"researchId"`
		User       string    `json:"user"`
		Action     string    `json:"action"`
		Details    string    `json:"details"`
		Timestamp  time.Time `json:"timestamp"`
	}

	var items []ActivityItem
	for rows.Next() {
		var item ActivityItem
		if err := rows.Scan(&item.ID, &item.ResearchID, &item.Action, &item.User, &item.Details, &item.Timestamp); err != nil {
			continue
		}
		items = append(items, item)
	}
	if items == nil {
		items = []ActivityItem{}
	}
	c.JSON(200, items)
}

func dbPostChatMessage(c *gin.Context) {
	var req struct {
		ResearchID string `json:"researchId" binding:"required"`
		Sender     string `json:"sender" binding:"required"`
		Channel    string `json:"channel"`
		Message    string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Channel == "" {
		req.Channel = "general"
	}
	message, err := sendResearchDiscussionMessage(c, req.ResearchID, req.Channel, req.Message)
	if err != nil {
		writeResearchDiscussionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, message)
}

// ─────────────────────────────────────────────────────────
//  Audit Logs
// ─────────────────────────────────────────────────────────

func fetchAuditLogs(researchID string) []AuditLog {
	rows, err := DB.Query(`SELECT id, research_id, action, performed_by, COALESCE(details,''), created_time
		FROM rms.audit_logs WHERE research_id=$1 ORDER BY created_time DESC LIMIT 100`, researchID)
	if err != nil {
		return []AuditLog{}
	}
	defer rows.Close()
	var results []AuditLog
	for rows.Next() {
		var al AuditLog
		rows.Scan(&al.ID, &al.ResearchID, &al.Action, &al.PerformedBy, &al.Details, &al.CreatedTime)
		results = append(results, al)
	}
	if results == nil {
		return []AuditLog{}
	}
	return results
}

func dbGetAuditLogs(c *gin.Context) {
	c.JSON(200, fetchAuditLogs(c.Param("id")))
}

// ─────────────────────────────────────────────────────────
//  Search
// ─────────────────────────────────────────────────────────

func dbSearch(c *gin.Context) {
	q := "%" + strings.ToLower(c.Query("q")) + "%"

	type SearchResult struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Name string `json:"name"`
		Desc string `json:"desc"`
	}

	var results []SearchResult

	rows, _ := DB.Query(`SELECT id, name, COALESCE(description,'') FROM rms.research_projects WHERE (LOWER(name) LIKE $1 OR LOWER(description) LIKE $1) AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL) LIMIT 10`, q, workspaceIDContext(c))
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var r SearchResult
			r.Type = "research"
			rows.Scan(&r.ID, &r.Name, &r.Desc)
			results = append(results, r)
		}
	}

	rows2, _ := DB.Query(`SELECT l.id, l.title, COALESCE(l.authors,'') FROM rms.literature l JOIN rms.research_projects p ON p.id=l.research_id WHERE (LOWER(l.title) LIKE $1 OR LOWER(l.authors) LIKE $1) AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL) LIMIT 10`, q, workspaceIDContext(c))
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var r SearchResult
			r.Type = "literature"
			rows2.Scan(&r.ID, &r.Name, &r.Desc)
			results = append(results, r)
		}
	}

	rows3, _ := DB.Query(`SELECT r.id, r.title, COALESCE(r.authors,'') FROM rms.publications r JOIN rms.research_projects p ON p.id=r.research_id WHERE LOWER(r.title) LIKE $1 AND (p.workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL) LIMIT 10`, q, workspaceIDContext(c))
	if rows3 != nil {
		defer rows3.Close()
		for rows3.Next() {
			var r SearchResult
			r.Type = "publication"
			rows3.Scan(&r.ID, &r.Name, &r.Desc)
			results = append(results, r)
		}
	}

	if results == nil {
		results = []SearchResult{}
	}
	c.JSON(http.StatusOK, results)
}

// ─── Phase 5 Handlers: Ethics Committees, DOI, Citation, Open Science ───

func dbGetEthicsCommittees(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, name, institution, COALESCE(chair_person,''), COALESCE(email,''), members, active, created_time FROM rms.ethics_committees ORDER BY name ASC`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var committees []EthicsCommittee
	for rows.Next() {
		var ec EthicsCommittee
		var members pq.StringArray
		rows.Scan(&ec.ID, &ec.Name, &ec.Institution, &ec.ChairPerson, &ec.Email, &members, &ec.Active, &ec.CreatedTime)
		ec.Members = members
		committees = append(committees, ec)
	}
	c.JSON(200, committees)
}

func dbCreateEthicsCommittee(c *gin.Context) {
	var ec EthicsCommittee
	if err := c.ShouldBindJSON(&ec); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if ec.ID == "" {
		ec.ID = "irb-" + newID()
	}
	ec.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO rms.ethics_committees (id, name, institution, chair_person, email, members, active, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		ec.ID, ec.Name, ec.Institution, ec.ChairPerson, ec.Email, pq.Array(ec.Members), ec.Active, ec.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, ec)
}

func dbGetDOIRecords(c *gin.Context) {
	researchID := c.Param("id")
	rows, err := DB.Query(`SELECT id, research_id, COALESCE(publication_id,''), doi, title, authors, year, publisher, COALESCE(url,''), status, created_time 
		FROM rms.doi_records WHERE research_id = $1`, researchID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var dois []DOIRecord
	for rows.Next() {
		var d DOIRecord
		var authors pq.StringArray
		rows.Scan(&d.ID, &d.ResearchID, &d.PublicationID, &d.DOI, &d.Title, &authors, &d.Year, &d.Publisher, &d.URL, &d.Status, &d.CreatedTime)
		d.Authors = authors
		dois = append(dois, d)
	}
	c.JSON(200, dois)
}

func dbCreateDOIRecord(c *gin.Context) {
	var d DOIRecord
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if d.ID == "" {
		d.ID = "doi-" + newID()
	}
	if d.DOI == "" {
		d.DOI = fmt.Sprintf("10.5849/statgate.%s", newID()[:8])
	}
	if d.Publisher == "" {
		d.Publisher = "StatGate Open Science"
	}
	if d.Year == 0 {
		d.Year = time.Now().Year()
	}
	d.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO rms.doi_records (id, research_id, publication_id, doi, title, authors, year, publisher, url, status, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		d.ID, d.ResearchID, d.PublicationID, d.DOI, d.Title, pq.Array(d.Authors), d.Year, d.Publisher, d.URL, d.Status, d.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, d)
}

func dbFormatCitation(c *gin.Context) {
	var req struct {
		Authors string `json:"authors"`
		Title   string `json:"title"`
		Journal string `json:"journal"`
		Year    string `json:"year"`
		Volume  string `json:"volume"`
		Issue   string `json:"issue"`
		Pages   string `json:"pages"`
		DOI     string `json:"doi"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Year == "" {
		req.Year = "2026"
	}
	if req.Authors == "" {
		req.Authors = "StatGate Research Team"
	}

	apa := fmt.Sprintf("%s (%s). %s. %s, %s(%s), %s. https://doi.org/%s",
		req.Authors, req.Year, req.Title, req.Journal, req.Volume, req.Issue, req.Pages, req.DOI)
	chicago := fmt.Sprintf("%s. \"%s.\" %s %s, no. %s (%s): %s. https://doi.org/%s.",
		req.Authors, req.Title, req.Journal, req.Volume, req.Issue, req.Year, req.Pages, req.DOI)
	harvard := fmt.Sprintf("%s, %s. %s. %s, %s(%s), pp.%s.",
		req.Authors, req.Year, req.Title, req.Journal, req.Volume, req.Issue, req.Pages)
	vancouver := fmt.Sprintf("%s. %s. %s. %s;%s(%s):%s.",
		req.Authors, req.Title, req.Journal, req.Year, req.Volume, req.Issue, req.Pages)
	bibtex := fmt.Sprintf("@article{statgate_%s,\n  author = {%s},\n  title = {%s},\n  journal = {%s},\n  year = {%s},\n  volume = {%s},\n  number = {%s},\n  pages = {%s},\n  doi = {%s}\n}",
		newID()[:6], req.Authors, req.Title, req.Journal, req.Year, req.Volume, req.Issue, req.Pages, req.DOI)

	c.JSON(200, CitationOutput{
		APA:       apa,
		Chicago:   chicago,
		Harvard:   harvard,
		Vancouver: vancouver,
		BibTeX:    bibtex,
	})
}

func dbGetOpenAccessRepo(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, research_id, title, COALESCE(abstract,''), license, COALESCE(access_url,''), COALESCE(download_url,''), COALESCE(file_size,''), format, views, downloads, created_time FROM rms.open_access_repo ORDER BY created_time DESC`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var items []OpenAccessRepoItem
	for rows.Next() {
		var item OpenAccessRepoItem
		rows.Scan(&item.ID, &item.ResearchID, &item.Title, &item.Abstract, &item.License, &item.AccessURL, &item.DownloadURL, &item.FileSize, &item.Format, &item.Views, &item.Downloads, &item.CreatedTime)
		items = append(items, item)
	}
	c.JSON(200, items)
}

func dbCreateOpenAccessRepo(c *gin.Context) {
	var item OpenAccessRepoItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if item.ID == "" {
		item.ID = "repo-" + newID()
	}
	if item.License == "" {
		item.License = "CC-BY-4.0"
	}
	item.CreatedTime = time.Now()

	_, err := DB.Exec(`INSERT INTO rms.open_access_repo (id, research_id, title, abstract, license, access_url, download_url, file_size, format, views, downloads, created_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		item.ID, item.ResearchID, item.Title, item.Abstract, item.License, item.AccessURL, item.DownloadURL, item.FileSize, item.Format, 0, 0, item.CreatedTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, item)
}
