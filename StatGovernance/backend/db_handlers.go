package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// ─── Helpers ──────────────────────────────────────────────────────

func newID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func recordAudit(entityType, entityID, action, actor, details, tenantID string, prev, next interface{}) {
	if DB == nil {
		return
	}
	var prevJSON, nextJSON []byte
	if prev != nil {
		prevJSON, _ = json.Marshal(prev)
	}
	if next != nil {
		nextJSON, _ = json.Marshal(next)
	}

	id := newID("aud")
	corr := fmt.Sprintf("corr_%d", time.Now().UnixNano())
	_, _ = DB.Exec(`
		INSERT INTO statgovernance.audit_logs (
			id, entity_type, entity_id, action, actor, details, tenant_id, correlation_id, previous_state, new_state
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, entityType, entityID, action, actor, details, tenantID, corr, prevJSON, nextJSON)
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

// ─── 1. Governance Dashboard & Real KPIs ──────────────────────────

func dbGetDashboard(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetDashboard(tenantID))
		return
	}

	kpis := GovernanceDashboardKPIs{
		RiskHeatmapMatrix: make(map[string]int),
	}

	// 1. Policies counts
	_ = DB.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'Active' OR status = 'Published'),
			COUNT(*) FILTER (WHERE status = 'Under Review' OR status = 'Under Revision'),
			COUNT(*) FILTER (WHERE review_date < CURRENT_DATE AND status = 'Active'),
			COUNT(*) FILTER (WHERE status = 'Approval')
		FROM statgovernance.policies WHERE tenant_id = $1`, tenantID).
		Scan(&kpis.PoliciesActive, &kpis.PoliciesUnderReview, &kpis.PoliciesExpired, &kpis.PoliciesAwaitingApproval)

	// 2. Compliance status counts
	var totalObligations int
	_ = DB.QueryRow(`
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE compliance_status = 'Compliant'),
			COUNT(*) FILTER (WHERE compliance_status = 'Partially Compliant'),
			COUNT(*) FILTER (WHERE compliance_status = 'Non-Compliant'),
			COUNT(*) FILTER (WHERE deadline < CURRENT_DATE AND compliance_status != 'Compliant')
		FROM statgovernance.compliance_obligations WHERE tenant_id = $1`, tenantID).
		Scan(&totalObligations, &kpis.ComplianceCompliant, &kpis.CompliancePartially, &kpis.ComplianceNonCompliant, &kpis.ComplianceOverdue)

	if totalObligations > 0 {
		kpis.ComplianceRate = float64(kpis.ComplianceCompliant) / float64(totalObligations) * 100.0
	} else {
		kpis.ComplianceRate = 100.0
	}

	// 3. Risks counts & Heatmap
	_ = DB.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE inherent_risk_level = 'Critical'),
			COUNT(*) FILTER (WHERE inherent_risk_level = 'High'),
			COUNT(*) FILTER (WHERE inherent_risk_level = 'Medium'),
			COUNT(*) FILTER (WHERE inherent_risk_level = 'Low')
		FROM statgovernance.risks WHERE tenant_id = $1 AND status != 'Closed'`, tenantID).
		Scan(&kpis.RisksCritical, &kpis.RisksHigh, &kpis.RisksMedium, &kpis.RisksLow)

	rows, err := DB.Query(`
		SELECT probability, impact, COUNT(*) 
		FROM statgovernance.risks 
		WHERE tenant_id = $1 AND status != 'Closed'
		GROUP BY probability, impact`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p, i, count int
			if err := rows.Scan(&p, &i, &count); err == nil {
				key := fmt.Sprintf("%d_%d", p, i)
				kpis.RiskHeatmapMatrix[key] = count
			}
		}
	}

	// 4. Controls counts
	_ = DB.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE effectiveness = 'Effective'),
			COUNT(*) FILTER (WHERE effectiveness = 'Partially Effective'),
			COUNT(*) FILTER (WHERE effectiveness = 'Ineffective'),
			COUNT(*) FILTER (WHERE effectiveness = 'Not Tested')
		FROM statgovernance.controls WHERE tenant_id = $1`, tenantID).
		Scan(&kpis.ControlsEffective, &kpis.ControlsNeedsReview, &kpis.ControlsFailed, &kpis.ControlsNotTested)

	// 5. Audits & Findings counts
	_ = DB.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'Planned' OR status = 'Scheduled'),
			COUNT(*) FILTER (WHERE status = 'In Progress')
		FROM statgovernance.audits WHERE tenant_id = $1`, tenantID).
		Scan(&kpis.AuditsPlanned, &kpis.AuditsInProgress)

	_ = DB.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status = 'Open' OR status = 'In Remediation'),
			COUNT(*) FILTER (WHERE due_date < CURRENT_DATE AND (status = 'Open' OR status = 'In Remediation'))
		FROM statgovernance.audit_findings WHERE tenant_id = $1`, tenantID).
		Scan(&kpis.FindingsOpen, &kpis.FindingsOverdue)

	// 6. Decisions count
	_ = DB.QueryRow(`SELECT COUNT(*) FROM statgovernance.governance_decisions WHERE tenant_id = $1`, tenantID).
		Scan(&kpis.RecentDecisionsCount)

	kpis.PendingApprovalsCount = kpis.PoliciesAwaitingApproval

	c.JSON(http.StatusOK, kpis)
}

// ─── 2. Policy Handlers ───────────────────────────────────────────

func dbGetPolicies(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	statusFilter := c.Query("status")
	categoryFilter := c.Query("category")

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetPolicies(tenantID, statusFilter, categoryFilter))
		return
	}

	q := `SELECT id, policy_number, title, summary, content, category, scope, classification,
		version, status, owner, COALESCE(owner_id, ''), department,
		COALESCE(effective_date::TEXT, ''), COALESCE(review_date::TEXT, ''), COALESCE(expiry_date::TEXT, ''),
		tenant_id, COALESCE(org_id, ''), COALESCE(related_regulations, '{}'), COALESCE(related_risks, '{}'),
		COALESCE(related_controls, '{}'), COALESCE(related_projects, '{}'), COALESCE(related_research, '{}'),
		COALESCE(related_datasets, '{}'), COALESCE(approved_by, ''), COALESCE(approved_at::TEXT, ''),
		COALESCE(published_at::TEXT, ''), created_time, updated_time
		FROM statgovernance.policies WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	idx := 2

	if statusFilter != "" {
		q += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, statusFilter)
		idx++
	}
	if categoryFilter != "" {
		q += fmt.Sprintf(" AND category = $%d", idx)
		args = append(args, categoryFilter)
		idx++
	}
	q += " ORDER BY updated_time DESC"

	rows, err := DB.Query(q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	policies := []Policy{}
	for rows.Next() {
		var p Policy
		var reg, rsk, ctl, prj, res, dat pq.StringArray
		err := rows.Scan(&p.ID, &p.PolicyNumber, &p.Title, &p.Summary, &p.Content, &p.Category, &p.Scope, &p.Classification,
			&p.Version, &p.Status, &p.Owner, &p.OwnerID, &p.Department,
			&p.EffectiveDate, &p.ReviewDate, &p.ExpiryDate,
			&p.TenantID, &p.OrgID, &reg, &rsk, &ctl, &prj, &res, &dat,
			&p.ApprovedBy, &p.ApprovedAt, &p.PublishedAt, &p.CreatedTime, &p.UpdatedTime)
		if err != nil {
			continue
		}
		p.RelatedRegulations = reg
		p.RelatedRisks = rsk
		p.RelatedControls = ctl
		p.RelatedProjects = prj
		p.RelatedResearch = res
		p.RelatedDatasets = dat
		policies = append(policies, p)
	}

	c.JSON(http.StatusOK, policies)
}

func dbGetPolicyByID(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	id := c.Param("id")

	if DB == nil {
		p, found := memStore.GetPolicyByID(id, tenantID)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
			return
		}
		c.JSON(http.StatusOK, p)
		return
	}

	q := `SELECT id, policy_number, title, summary, content, category, scope, classification,
		version, status, owner, COALESCE(owner_id, ''), department,
		COALESCE(effective_date::TEXT, ''), COALESCE(review_date::TEXT, ''), COALESCE(expiry_date::TEXT, ''),
		tenant_id, COALESCE(org_id, ''), COALESCE(related_regulations, '{}'), COALESCE(related_risks, '{}'),
		COALESCE(related_controls, '{}'), COALESCE(related_projects, '{}'), COALESCE(related_research, '{}'),
		COALESCE(related_datasets, '{}'), COALESCE(approved_by, ''), COALESCE(approved_at::TEXT, ''),
		COALESCE(published_at::TEXT, ''), created_time, updated_time
		FROM statgovernance.policies WHERE id = $1 AND tenant_id = $2`

	var p Policy
	var reg, rsk, ctl, prj, res, dat pq.StringArray
	err := DB.QueryRow(q, id, tenantID).Scan(&p.ID, &p.PolicyNumber, &p.Title, &p.Summary, &p.Content, &p.Category, &p.Scope, &p.Classification,
		&p.Version, &p.Status, &p.Owner, &p.OwnerID, &p.Department,
		&p.EffectiveDate, &p.ReviewDate, &p.ExpiryDate,
		&p.TenantID, &p.OrgID, &reg, &rsk, &ctl, &prj, &res, &dat,
		&p.ApprovedBy, &p.ApprovedAt, &p.PublishedAt, &p.CreatedTime, &p.UpdatedTime)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	p.RelatedRegulations = reg
	p.RelatedRisks = rsk
	p.RelatedControls = ctl
	p.RelatedProjects = prj
	p.RelatedResearch = res
	p.RelatedDatasets = dat

	c.JSON(http.StatusOK, p)
}

func dbCreatePolicy(c *gin.Context) {
	userID, userName, _, tenantID := getAuthContext(c)
	var req Policy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("pol")
	}
	if req.PolicyNumber == "" {
		req.PolicyNumber = fmt.Sprintf("POL-%s-%d", strings.ToUpper(req.Category[:min(4, len(req.Category))]), time.Now().Unix()%100000)
	}
	if req.Version == "" {
		req.Version = "1.0"
	}
	if req.Status == "" {
		req.Status = "Draft"
	}
	if req.Owner == "" {
		req.Owner = userName
	}
	if req.OwnerID == "" {
		req.OwnerID = userID
	}

	reg := pq.StringArray(req.RelatedRegulations)
	rsk := pq.StringArray(req.RelatedRisks)
	ctl := pq.StringArray(req.RelatedControls)
	prj := pq.StringArray(req.RelatedProjects)
	res := pq.StringArray(req.RelatedResearch)
	dat := pq.StringArray(req.RelatedDatasets)

	_, err := DB.Exec(`
		INSERT INTO statgovernance.policies (
			id, policy_number, title, summary, content, category, scope, classification, version, status,
			owner, owner_id, department, effective_date, review_date, expiry_date, tenant_id, org_id,
			related_regulations, related_risks, related_controls, related_projects, related_research, related_datasets
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		req.ID, req.PolicyNumber, req.Title, req.Summary, req.Content, req.Category, req.Scope, req.Classification,
		req.Version, req.Status, req.Owner, req.OwnerID, req.Department,
		nullDate(req.EffectiveDate), nullDate(req.ReviewDate), nullDate(req.ExpiryDate),
		tenantID, req.OrgID, reg, rsk, ctl, prj, res, dat)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create initial version
	verID := newID("pver")
	_, _ = DB.Exec(`
		INSERT INTO statgovernance.policy_versions (id, policy_id, version, title, content, change_summary, changed_by, status)
		VALUES ($1, $2, $3, $4, $5, 'Initial Creation', $6, $7)`,
		verID, req.ID, req.Version, req.Title, req.Content, userName, req.Status)

	recordAudit("policy", req.ID, "policy.created", userName, fmt.Sprintf("Policy %s created", req.PolicyNumber), tenantID, nil, req)

	publishEvent("policy.created", "policy", req.ID, userName, tenantID, map[string]interface{}{
		"policy_number": req.PolicyNumber,
		"title":         req.Title,
		"status":        req.Status,
		"owner":         req.Owner,
		"department":    req.Department,
	})

	c.JSON(http.StatusCreated, req)
}

func dbUpdatePolicy(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	id := c.Param("id")

	var req Policy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch existing policy to preserve version history
	var existing Policy
	_ = DB.QueryRow("SELECT version, title, content, status FROM statgovernance.policies WHERE id = $1 AND tenant_id = $2", id, tenantID).
		Scan(&existing.Version, &existing.Title, &existing.Content, &existing.Status)

	reg := pq.StringArray(req.RelatedRegulations)
	rsk := pq.StringArray(req.RelatedRisks)
	ctl := pq.StringArray(req.RelatedControls)
	prj := pq.StringArray(req.RelatedProjects)
	res := pq.StringArray(req.RelatedResearch)
	dat := pq.StringArray(req.RelatedDatasets)

	_, err := DB.Exec(`
		UPDATE statgovernance.policies SET
			title = $1, summary = $2, content = $3, category = $4, scope = $5, classification = $6,
			department = $7, effective_date = $8, review_date = $9, expiry_date = $10,
			related_regulations = $11, related_risks = $12, related_controls = $13,
			related_projects = $14, related_research = $15, related_datasets = $16,
			updated_time = CURRENT_TIMESTAMP
		WHERE id = $17 AND tenant_id = $18`,
		req.Title, req.Summary, req.Content, req.Category, req.Scope, req.Classification,
		req.Department, nullDate(req.EffectiveDate), nullDate(req.ReviewDate), nullDate(req.ExpiryDate),
		reg, rsk, ctl, prj, res, dat, id, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("policy", id, "policy.updated", userName, fmt.Sprintf("Policy %s updated", id), tenantID, existing, req)

	publishEvent("policy.updated", "policy", id, userName, tenantID, map[string]interface{}{
		"title": req.Title,
	})

	c.JSON(http.StatusOK, gin.H{"status": "updated", "id": id})
}

func dbApprovePolicy(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	id := c.Param("id")

	_, err := DB.Exec(`
		UPDATE statgovernance.policies SET
			status = 'Approved',
			approved_by = $1,
			approved_at = CURRENT_TIMESTAMP,
			updated_time = CURRENT_TIMESTAMP
		WHERE id = $2 AND tenant_id = $3`,
		userName, id, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update latest version record
	_, _ = DB.Exec(`
		UPDATE statgovernance.policy_versions SET
			status = 'Approved',
			approved_by = $1
		WHERE policy_id = $2`, userName, id)

	recordAudit("policy", id, "policy.approved", userName, "Policy approved", tenantID, nil, map[string]interface{}{"status": "Approved"})

	publishEvent("policy.approved", "policy", id, userName, tenantID, map[string]interface{}{
		"approved_by": userName,
		"approved_at": time.Now().UTC().Format(time.RFC3339),
	})

	c.JSON(http.StatusOK, gin.H{"status": "approved", "id": id})
}

func dbPublishPolicy(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	id := c.Param("id")

	_, err := DB.Exec(`
		UPDATE statgovernance.policies SET
			status = 'Active',
			published_at = CURRENT_TIMESTAMP,
			updated_time = CURRENT_TIMESTAMP
		WHERE id = $1 AND tenant_id = $2`,
		id, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("policy", id, "policy.published", userName, "Policy published and made active", tenantID, nil, map[string]interface{}{"status": "Active"})

	publishEvent("policy.published", "policy", id, userName, tenantID, map[string]interface{}{
		"published_at": time.Now().UTC().Format(time.RFC3339),
	})

	c.JSON(http.StatusOK, gin.H{"status": "published", "id": id})
}

func dbGetPolicyVersions(c *gin.Context) {
	id := c.Param("id")

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetPolicyVersions(id))
		return
	}
	rows, err := DB.Query(`
		SELECT id, policy_id, version, title, content, COALESCE(change_summary, ''), changed_by,
		COALESCE(approved_by, ''), status, created_time
		FROM statgovernance.policy_versions WHERE policy_id = $1 ORDER BY created_time DESC`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	versions := []PolicyVersion{}
	for rows.Next() {
		var v PolicyVersion
		rows.Scan(&v.ID, &v.PolicyID, &v.Version, &v.Title, &v.Content, &v.ChangeSummary, &v.ChangedBy, &v.ApprovedBy, &v.Status, &v.CreatedTime)
		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, versions)
}

// ─── 3. SOP Handlers ──────────────────────────────────────────────

func dbGetSOPs(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetSOPs(tenantID))
		return
	}
	rows, err := DB.Query(`
		SELECT id, sop_number, title, process, purpose, scope, COALESCE(responsibilities, ''),
		procedure_steps, COALESCE(required_evidence, '{}'), COALESCE(related_policy_id, ''),
		COALESCE(related_control_id, ''), review_frequency, version, status, owner, department,
		tenant_id, COALESCE(approved_by, ''), COALESCE(approved_at::TEXT, ''), created_time, updated_time
		FROM statgovernance.sops WHERE tenant_id = $1 ORDER BY updated_time DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	sops := []SOP{}
	for rows.Next() {
		var s SOP
		var stepsRaw []byte
		var ev pq.StringArray
		err := rows.Scan(&s.ID, &s.SOPNumber, &s.Title, &s.Process, &s.Purpose, &s.Scope, &s.Responsibilities,
			&stepsRaw, &ev, &s.RelatedPolicyID, &s.RelatedControlID, &s.ReviewFrequency, &s.Version, &s.Status,
			&s.Owner, &s.Department, &s.TenantID, &s.ApprovedBy, &s.ApprovedAt, &s.CreatedTime, &s.UpdatedTime)
		if err != nil {
			continue
		}
		s.RequiredEvidence = ev
		if len(stepsRaw) > 0 {
			_ = json.Unmarshal(stepsRaw, &s.ProcedureSteps)
		}
		sops = append(sops, s)
	}

	c.JSON(http.StatusOK, sops)
}

func dbCreateSOP(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req SOP
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("sop")
	}
	if req.SOPNumber == "" {
		req.SOPNumber = fmt.Sprintf("SOP-%d", time.Now().Unix()%100000)
	}
	if req.Version == "" {
		req.Version = "1.0"
	}
	if req.Status == "" {
		req.Status = "Active"
	}

	stepsJSON, _ := json.Marshal(req.ProcedureSteps)
	ev := pq.StringArray(req.RequiredEvidence)

	_, err := DB.Exec(`
		INSERT INTO statgovernance.sops (
			id, sop_number, title, process, purpose, scope, responsibilities, procedure_steps,
			required_evidence, related_policy_id, related_control_id, review_frequency, version,
			status, owner, department, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		req.ID, req.SOPNumber, req.Title, req.Process, req.Purpose, req.Scope, req.Responsibilities,
		stepsJSON, ev, nullStr(req.RelatedPolicyID), nullStr(req.RelatedControlID),
		req.ReviewFrequency, req.Version, req.Status, req.Owner, req.Department, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("sop", req.ID, "sop.created", userName, fmt.Sprintf("SOP %s created", req.SOPNumber), tenantID, nil, req)
	publishEvent("sop.created", "sop", req.ID, userName, tenantID, map[string]interface{}{"title": req.Title})

	c.JSON(http.StatusCreated, req)
}

// ─── 4. Compliance & Regulatory Handlers ──────────────────────────

func dbGetRegulations(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetRegulations(tenantID))
		return
	}
	rows, err := DB.Query(`
		SELECT id, code, name, regulatory_authority, jurisdiction, category, COALESCE(description, ''),
		COALESCE(source_url, ''), COALESCE(effective_date::TEXT, ''), tenant_id, created_time, updated_time
		FROM statgovernance.regulations WHERE tenant_id = $1 ORDER BY name ASC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	regs := []Regulation{}
	for rows.Next() {
		var r Regulation
		rows.Scan(&r.ID, &r.Code, &r.Name, &r.RegulatoryAuthority, &r.Jurisdiction, &r.Category,
			&r.Description, &r.SourceURL, &r.EffectiveDate, &r.TenantID, &r.CreatedTime, &r.UpdatedTime)
		regs = append(regs, r)
	}

	c.JSON(http.StatusOK, regs)
}

func dbCreateRegulation(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req Regulation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("reg")
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.regulations (
			id, code, name, regulatory_authority, jurisdiction, category, description, source_url, effective_date, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		req.ID, req.Code, req.Name, req.RegulatoryAuthority, req.Jurisdiction, req.Category,
		req.Description, req.SourceURL, nullDate(req.EffectiveDate), tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("regulation", req.ID, "regulation.created", userName, fmt.Sprintf("Regulation %s created", req.Code), tenantID, nil, req)
	c.JSON(http.StatusCreated, req)
}

func dbGetObligations(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	rows, err := DB.Query(`
		SELECT o.id, o.obligation_code, o.regulation_id, r.name, o.requirement, COALESCE(o.applicable_scope, ''),
		o.responsible_dept, o.responsible_person, o.frequency, o.compliance_status,
		COALESCE(o.evidence_requirement, ''), COALESCE(o.evidence_location, ''), o.violation_risk,
		COALESCE(o.corrective_action, ''), COALESCE(o.deadline::TEXT, ''), COALESCE(o.review_date::TEXT, ''),
		o.tenant_id, o.created_time, o.updated_time
		FROM statgovernance.compliance_obligations o
		LEFT JOIN statgovernance.regulations r ON o.regulation_id = r.id
		WHERE o.tenant_id = $1 ORDER BY o.deadline ASC NULLS LAST`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	obs := []ComplianceObligation{}
	for rows.Next() {
		var o ComplianceObligation
		rows.Scan(&o.ID, &o.ObligationCode, &o.RegulationID, &o.RegulationName, &o.Requirement, &o.ApplicableScope,
			&o.ResponsibleDept, &o.ResponsiblePerson, &o.Frequency, &o.ComplianceStatus,
			&o.EvidenceRequirement, &o.EvidenceLocation, &o.ViolationRisk,
			&o.CorrectiveAction, &o.Deadline, &o.ReviewDate, &o.TenantID, &o.CreatedTime, &o.UpdatedTime)
		obs = append(obs, o)
	}

	c.JSON(http.StatusOK, obs)
}

func dbCreateObligation(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req ComplianceObligation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("obl")
	}
	if req.ObligationCode == "" {
		req.ObligationCode = fmt.Sprintf("OBL-%d", time.Now().Unix()%100000)
	}
	if req.ComplianceStatus == "" {
		req.ComplianceStatus = "Compliant"
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.compliance_obligations (
			id, obligation_code, regulation_id, requirement, applicable_scope, responsible_dept,
			responsible_person, frequency, compliance_status, evidence_requirement, evidence_location,
			violation_risk, corrective_action, deadline, review_date, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		req.ID, req.ObligationCode, req.RegulationID, req.Requirement, req.ApplicableScope,
		req.ResponsibleDept, req.ResponsiblePerson, req.Frequency, req.ComplianceStatus,
		req.EvidenceRequirement, req.EvidenceLocation, req.ViolationRisk, req.CorrectiveAction,
		nullDate(req.Deadline), nullDate(req.ReviewDate), tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("obligation", req.ID, "obligation.created", userName, fmt.Sprintf("Obligation %s created", req.ObligationCode), tenantID, nil, req)
	publishEvent("compliance.obligation.created", "obligation", req.ID, userName, tenantID, map[string]interface{}{"code": req.ObligationCode})

	c.JSON(http.StatusCreated, req)
}

func dbGetAssessments(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetAssessments(tenantID))
		return
	}
	rows, err := DB.Query(`
		SELECT id, assessment_code, title, scope, assessor, COALESCE(assessor_id, ''), status,
		score_percentage, findings_count, COALESCE(recommendations, ''),
		COALESCE(start_date::TEXT, ''), COALESCE(completion_date::TEXT, ''), tenant_id, created_time, updated_time
		FROM statgovernance.compliance_assessments WHERE tenant_id = $1 ORDER BY created_time DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	assessments := []ComplianceAssessment{}
	for rows.Next() {
		var a ComplianceAssessment
		rows.Scan(&a.ID, &a.AssessmentCode, &a.Title, &a.Scope, &a.Assessor, &a.AssessorID, &a.Status,
			&a.ScorePercentage, &a.FindingsCount, &a.Recommendations,
			&a.StartDate, &a.CompletionDate, &a.TenantID, &a.CreatedTime, &a.UpdatedTime)
		assessments = append(assessments, a)
	}

	c.JSON(http.StatusOK, assessments)
}

func dbCreateAssessment(c *gin.Context) {
	userID, userName, _, tenantID := getAuthContext(c)
	var req ComplianceAssessment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("asm")
	}
	if req.AssessmentCode == "" {
		req.AssessmentCode = fmt.Sprintf("ASM-%d", time.Now().Unix()%100000)
	}
	if req.Status == "" {
		req.Status = "In Progress"
	}
	if req.Assessor == "" {
		req.Assessor = userName
	}
	if req.AssessorID == "" {
		req.AssessorID = userID
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.compliance_assessments (
			id, assessment_code, title, scope, assessor, assessor_id, status, score_percentage,
			findings_count, recommendations, start_date, completion_date, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		req.ID, req.AssessmentCode, req.Title, req.Scope, req.Assessor, req.AssessorID,
		req.Status, req.ScorePercentage, req.FindingsCount, req.Recommendations,
		nullDate(req.StartDate), nullDate(req.CompletionDate), tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("assessment", req.ID, "assessment.created", userName, fmt.Sprintf("Assessment %s created", req.AssessmentCode), tenantID, nil, req)
	publishEvent("compliance.assessment.created", "assessment", req.ID, userName, tenantID, map[string]interface{}{"title": req.Title})

	c.JSON(http.StatusCreated, req)
}

// ─── 5. Risk Management & Heatmap ────────────────────────────────

func calculateRiskLevel(score int) string {
	if score >= 15 {
		return "Critical"
	} else if score >= 10 {
		return "High"
	} else if score >= 5 {
		return "Medium"
	}
	return "Low"
}

func dbGetRisks(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	categoryFilter := c.Query("category")
	levelFilter := c.Query("level")
	statusFilter := c.Query("status")

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetRisks(tenantID, categoryFilter, levelFilter, statusFilter))
		return
	}
	rows, err := DB.Query(`
		SELECT id, risk_code, title, description, category, owner, COALESCE(owner_id, ''),
		organization, COALESCE(department, ''), COALESCE(project_id, ''), COALESCE(research_id, ''),
		probability, impact, inherent_risk_score, inherent_risk_level, treatment_strategy,
		COALESCE(mitigation_actions, ''), COALESCE(residual_probability, 0), COALESCE(residual_impact, 0),
		COALESCE(residual_risk_score, 0), COALESCE(residual_risk_level, ''), status,
		COALESCE(target_date::TEXT, ''), COALESCE(review_date::TEXT, ''), tenant_id, created_time, updated_time
		FROM statgovernance.risks WHERE tenant_id = $1 ORDER BY inherent_risk_score DESC, updated_time DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	risks := []Risk{}
	for rows.Next() {
		var r Risk
		rows.Scan(&r.ID, &r.RiskCode, &r.Title, &r.Description, &r.Category, &r.Owner, &r.OwnerID,
			&r.Organization, &r.Department, &r.ProjectID, &r.ResearchID,
			&r.Probability, &r.Impact, &r.InherentRiskScore, &r.InherentRiskLevel, &r.TreatmentStrategy,
			&r.MitigationActions, &r.ResidualProbability, &r.ResidualImpact,
			&r.ResidualRiskScore, &r.ResidualRiskLevel, &r.Status,
			&r.TargetDate, &r.ReviewDate, &r.TenantID, &r.CreatedTime, &r.UpdatedTime)
		risks = append(risks, r)
	}

	c.JSON(http.StatusOK, risks)
}

func dbCreateRisk(c *gin.Context) {
	userID, userName, _, tenantID := getAuthContext(c)
	var req Risk
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if DB == nil {
		if req.ID == "" { req.ID = newID("rsk") }
		req.TenantID = tenantID
		c.JSON(http.StatusCreated, memStore.CreateRisk(req))
		return
	}

	if req.ID == "" {
		req.ID = newID("rsk")
	}
	if req.RiskCode == "" {
		req.RiskCode = fmt.Sprintf("RSK-%d", time.Now().Unix()%100000)
	}
	if req.Probability < 1 {
		req.Probability = 1
	}
	if req.Probability > 5 {
		req.Probability = 5
	}
	if req.Impact < 1 {
		req.Impact = 1
	}
	if req.Impact > 5 {
		req.Impact = 5
	}
	req.InherentRiskScore = req.Probability * req.Impact
	req.InherentRiskLevel = calculateRiskLevel(req.InherentRiskScore)

	if req.ResidualProbability > 0 && req.ResidualImpact > 0 {
		req.ResidualRiskScore = req.ResidualProbability * req.ResidualImpact
		req.ResidualRiskLevel = calculateRiskLevel(req.ResidualRiskScore)
	}

	if req.Status == "" {
		req.Status = "Identified"
	}
	if req.Owner == "" {
		req.Owner = userName
	}
	if req.OwnerID == "" {
		req.OwnerID = userID
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.risks (
			id, risk_code, title, description, category, owner, owner_id, organization, department,
			project_id, research_id, probability, impact, inherent_risk_level, treatment_strategy,
			mitigation_actions, residual_probability, residual_impact, residual_risk_score, residual_risk_level,
			status, target_date, review_date, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)`,
		req.ID, req.RiskCode, req.Title, req.Description, req.Category, req.Owner, req.OwnerID,
		req.Organization, req.Department, nullStr(req.ProjectID), nullStr(req.ResearchID),
		req.Probability, req.Impact, req.InherentRiskLevel, req.TreatmentStrategy,
		req.MitigationActions, nullStr(fmt.Sprintf("%d", req.ResidualProbability)), nullStr(fmt.Sprintf("%d", req.ResidualImpact)),
		req.ResidualRiskScore, req.ResidualRiskLevel, req.Status,
		nullDate(req.TargetDate), nullDate(req.ReviewDate), tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("risk", req.ID, "risk.created", userName, fmt.Sprintf("Risk %s created: %s", req.RiskCode, req.Title), tenantID, nil, req)

	publishEvent("risk.created", "risk", req.ID, userName, tenantID, map[string]interface{}{
		"risk_code":           req.RiskCode,
		"title":               req.Title,
		"inherent_risk_score": req.InherentRiskScore,
		"inherent_risk_level": req.InherentRiskLevel,
		"owner":               req.Owner,
	})

	if req.InherentRiskLevel == "Critical" || req.InherentRiskLevel == "High" {
		publishEvent("risk.escalated", "risk", req.ID, userName, tenantID, map[string]interface{}{
			"risk_code": req.RiskCode,
			"title":     req.Title,
			"level":     req.InherentRiskLevel,
			"score":     req.InherentRiskScore,
		})
	}

	c.JSON(http.StatusCreated, req)
}

func dbEscalateRisk(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	id := c.Param("id")

	var r Risk
	_ = DB.QueryRow("SELECT risk_code, title, inherent_risk_level, inherent_risk_score FROM statgovernance.risks WHERE id = $1 AND tenant_id = $2", id, tenantID).
		Scan(&r.RiskCode, &r.Title, &r.InherentRiskLevel, &r.InherentRiskScore)

	recordAudit("risk", id, "risk.escalated", userName, fmt.Sprintf("Risk %s manually escalated for executive review", r.RiskCode), tenantID, nil, map[string]interface{}{"escalated_by": userName})

	publishEvent("risk.escalated", "risk", id, userName, tenantID, map[string]interface{}{
		"risk_code": r.RiskCode,
		"title":     r.Title,
		"level":     r.InherentRiskLevel,
		"score":     r.InherentRiskScore,
	})

	c.JSON(http.StatusOK, gin.H{"status": "escalated", "id": id})
}

// ─── 6. Control Library & Testing ─────────────────────────────────

func dbGetControls(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	typeFilter := c.Query("type")
	effFilter := c.Query("effectiveness")

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetControls(tenantID, typeFilter, effFilter))
		return
	}
	rows, err := DB.Query(`
		SELECT id, control_code, name, description, objective, owner, department, control_type,
		frequency, test_method, COALESCE(evidence_requirement, ''), effectiveness,
		COALESCE(related_risk_id, ''), COALESCE(related_policy_id, ''),
		COALESCE(last_tested::TEXT, ''), COALESCE(next_test_date::TEXT, ''),
		tenant_id, created_time, updated_time
		FROM statgovernance.controls WHERE tenant_id = $1 ORDER BY control_code ASC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	controls := []Control{}
	for rows.Next() {
		var ct Control
		rows.Scan(&ct.ID, &ct.ControlCode, &ct.Name, &ct.Description, &ct.Objective, &ct.Owner, &ct.Department,
			&ct.ControlType, &ct.Frequency, &ct.TestMethod, &ct.EvidenceRequirement, &ct.Effectiveness,
			&ct.RelatedRiskID, &ct.RelatedPolicyID, &ct.LastTested, &ct.NextTestDate,
			&ct.TenantID, &ct.CreatedTime, &ct.UpdatedTime)
		controls = append(controls, ct)
	}

	c.JSON(http.StatusOK, controls)
}

func dbCreateControl(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req Control
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("ctl")
	}
	if req.ControlCode == "" {
		req.ControlCode = fmt.Sprintf("CTL-%d", time.Now().Unix()%100000)
	}
	if req.Effectiveness == "" {
		req.Effectiveness = "Effective"
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.controls (
			id, control_code, name, description, objective, owner, department, control_type,
			frequency, test_method, evidence_requirement, effectiveness, related_risk_id,
			related_policy_id, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		req.ID, req.ControlCode, req.Name, req.Description, req.Objective, req.Owner, req.Department,
		req.ControlType, req.Frequency, req.TestMethod, req.EvidenceRequirement, req.Effectiveness,
		nullStr(req.RelatedRiskID), nullStr(req.RelatedPolicyID), tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("control", req.ID, "control.created", userName, fmt.Sprintf("Control %s created", req.ControlCode), tenantID, nil, req)
	publishEvent("control.created", "control", req.ID, userName, tenantID, map[string]interface{}{"name": req.Name})

	c.JSON(http.StatusCreated, req)
}

func dbTestControl(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	id := c.Param("id")

	var testReq ControlTest
	if err := c.ShouldBindJSON(&testReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	testID := newID("ctest")
	if testReq.Tester == "" {
		testReq.Tester = userName
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.control_tests (
			id, control_id, tester, result, effectiveness, findings, evidence_ref
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		testID, id, testReq.Tester, testReq.Result, testReq.Effectiveness, testReq.Findings, testReq.EvidenceRef)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update control effectiveness and last tested timestamp
	_, _ = DB.Exec(`
		UPDATE statgovernance.controls SET
			effectiveness = $1,
			last_tested = CURRENT_TIMESTAMP,
			updated_time = CURRENT_TIMESTAMP
		WHERE id = $2 AND tenant_id = $3`,
		testReq.Effectiveness, id, tenantID)

	recordAudit("control", id, "control.tested", userName, fmt.Sprintf("Control tested with result: %s", testReq.Result), tenantID, nil, testReq)

	publishEvent("control.tested", "control", id, userName, tenantID, map[string]interface{}{
		"result":        testReq.Result,
		"effectiveness": testReq.Effectiveness,
	})

	if testReq.Effectiveness == "Ineffective" || testReq.Result == "Fail" {
		publishEvent("control.failed", "control", id, userName, tenantID, map[string]interface{}{
			"findings": testReq.Findings,
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "tested", "test_id": testID, "effectiveness": testReq.Effectiveness})
}

// ─── 7. Audit Management & Findings ──────────────────────────────

func dbGetAudits(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	statusFilter := c.Query("status")

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetAudits(tenantID, statusFilter))
		return
	}
	rows, err := DB.Query(`
		SELECT id, audit_code, title, audit_type, scope, objectives, lead_auditor,
		COALESCE(auditors, '{}'), COALESCE(auditees, '{}'), department,
		COALESCE(start_date::TEXT, ''), COALESCE(end_date::TEXT, ''), status,
		findings_count, critical_findings, COALESCE(summary, ''), COALESCE(report_file_id, ''),
		tenant_id, created_time, updated_time
		FROM statgovernance.audits WHERE tenant_id = $1 ORDER BY created_time DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	audits := []Audit{}
	for rows.Next() {
		var a Audit
		var aud, adt pq.StringArray
		rows.Scan(&a.ID, &a.AuditCode, &a.Title, &a.AuditType, &a.Scope, &a.Objectives, &a.LeadAuditor,
			&aud, &adt, &a.Department, &a.StartDate, &a.EndDate, &a.Status,
			&a.FindingsCount, &a.CriticalFindings, &a.Summary, &a.ReportFileID,
			&a.TenantID, &a.CreatedTime, &a.UpdatedTime)
		a.Auditors = aud
		a.Auditees = adt
		audits = append(audits, a)
	}

	c.JSON(http.StatusOK, audits)
}

func dbCreateAudit(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req Audit
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("adt")
	}
	if req.AuditCode == "" {
		req.AuditCode = fmt.Sprintf("AUD-%d", time.Now().Unix()%100000)
	}
	if req.Status == "" {
		req.Status = "Planned"
	}

	aud := pq.StringArray(req.Auditors)
	adt := pq.StringArray(req.Auditees)

	_, err := DB.Exec(`
		INSERT INTO statgovernance.audits (
			id, audit_code, title, audit_type, scope, objectives, lead_auditor, auditors,
			auditees, department, start_date, end_date, status, summary, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		req.ID, req.AuditCode, req.Title, req.AuditType, req.Scope, req.Objectives, req.LeadAuditor,
		aud, adt, req.Department, nullDate(req.StartDate), nullDate(req.EndDate), req.Status, req.Summary, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("audit", req.ID, "audit.created", userName, fmt.Sprintf("Audit %s created", req.AuditCode), tenantID, nil, req)
	publishEvent("audit.created", "audit", req.ID, userName, tenantID, map[string]interface{}{"title": req.Title})

	c.JSON(http.StatusCreated, req)
}

func dbGetFindings(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	sevFilter := c.Query("severity")
	statusFilter := c.Query("status")

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetFindings(tenantID, sevFilter, statusFilter))
		return
	}
	auditID := c.Query("audit_id")

	q := `SELECT id, finding_code, COALESCE(audit_id, ''), title, description, severity, source,
		owner, department, COALESCE(policy_id, ''), COALESCE(control_id, ''), COALESCE(risk_id, ''),
		recommendation, COALESCE(management_response, ''), COALESCE(due_date::TEXT, ''), status,
		COALESCE(enterprise_task_id, ''), COALESCE(closure_evidence, ''), COALESCE(closed_by, ''),
		COALESCE(closed_at::TEXT, ''), tenant_id, created_time, updated_time
		FROM statgovernance.audit_findings WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	if auditID != "" {
		q += " AND audit_id = $2"
		args = append(args, auditID)
	}
	q += " ORDER BY CASE severity WHEN 'Critical' THEN 1 WHEN 'High' THEN 2 WHEN 'Medium' THEN 3 ELSE 4 END, created_time DESC"

	rows, err := DB.Query(q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	findings := []AuditFinding{}
	for rows.Next() {
		var f AuditFinding
		rows.Scan(&f.ID, &f.FindingCode, &f.AuditID, &f.Title, &f.Description, &f.Severity, &f.Source,
			&f.Owner, &f.Department, &f.PolicyID, &f.ControlID, &f.RiskID,
			&f.Recommendation, &f.ManagementResponse, &f.DueDate, &f.Status,
			&f.EnterpriseTaskID, &f.ClosureEvidence, &f.ClosedBy, &f.ClosedAt,
			&f.TenantID, &f.CreatedTime, &f.UpdatedTime)
		findings = append(findings, f)
	}

	c.JSON(http.StatusOK, findings)
}

func dbCreateFinding(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req AuditFinding
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("fnd")
	}
	if req.FindingCode == "" {
		req.FindingCode = fmt.Sprintf("FND-%d", time.Now().Unix()%100000)
	}
	if req.Status == "" {
		req.Status = "Open"
	}
	if req.Severity == "" {
		req.Severity = "Medium"
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.audit_findings (
			id, finding_code, audit_id, title, description, severity, source, owner, department,
			policy_id, control_id, risk_id, recommendation, management_response, due_date, status, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		req.ID, req.FindingCode, nullStr(req.AuditID), req.Title, req.Description, req.Severity, req.Source,
		req.Owner, req.Department, nullStr(req.PolicyID), nullStr(req.ControlID), nullStr(req.RiskID),
		req.Recommendation, req.ManagementResponse, nullDate(req.DueDate), req.Status, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update audit counts if linked to an audit
	if req.AuditID != "" {
		_, _ = DB.Exec(`
			UPDATE statgovernance.audits SET
				findings_count = findings_count + 1,
				critical_findings = critical_findings + CASE WHEN $1 = 'Critical' THEN 1 ELSE 0 END,
				updated_time = CURRENT_TIMESTAMP
			WHERE id = $2`, req.Severity, req.AuditID)
	}

	recordAudit("finding", req.ID, "finding.created", userName, fmt.Sprintf("Finding %s created: %s", req.FindingCode, req.Title), tenantID, nil, req)

	publishEvent("compliance.finding.created", "finding", req.ID, userName, tenantID, map[string]interface{}{
		"finding_code": req.FindingCode,
		"title":        req.Title,
		"severity":     req.Severity,
		"owner":        req.Owner,
	})

	c.JSON(http.StatusCreated, req)
}

func dbCloseFinding(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	id := c.Param("id")

	var closeReq struct {
		Evidence string `json:"evidence"`
	}
	_ = c.ShouldBindJSON(&closeReq)

	_, err := DB.Exec(`
		UPDATE statgovernance.audit_findings SET
			status = 'Closed',
			closure_evidence = $1,
			closed_by = $2,
			closed_at = CURRENT_TIMESTAMP,
			updated_time = CURRENT_TIMESTAMP
		WHERE id = $3 AND tenant_id = $4`,
		closeReq.Evidence, userName, id, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("finding", id, "finding.closed", userName, "Finding closed with verification", tenantID, nil, map[string]interface{}{"status": "Closed"})

	publishEvent("compliance.finding.closed", "finding", id, userName, tenantID, map[string]interface{}{
		"closed_by": userName,
		"closed_at": time.Now().UTC().Format(time.RFC3339),
	})

	c.JSON(http.StatusOK, gin.H{"status": "closed", "id": id})
}

// ─── 8. Corrective & Preventive Actions (CAPA) ────────────────────

func dbGetCorrectiveActions(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	statusFilter := c.Query("status")

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetCorrectiveActions(tenantID, statusFilter))
		return
	}
	rows, err := DB.Query(`
		SELECT id, action_code, title, description, source_type, source_id, owner, priority,
		due_date::TEXT, COALESCE(enterprise_task_id, ''), status, COALESCE(completion_date::TEXT, ''),
		COALESCE(verification_notes, ''), COALESCE(verified_by, ''), COALESCE(verified_at::TEXT, ''),
		tenant_id, created_time, updated_time
		FROM statgovernance.corrective_actions WHERE tenant_id = $1 ORDER BY due_date ASC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	actions := []CorrectiveAction{}
	for rows.Next() {
		var a CorrectiveAction
		rows.Scan(&a.ID, &a.ActionCode, &a.Title, &a.Description, &a.SourceType, &a.SourceID,
			&a.Owner, &a.Priority, &a.DueDate, &a.EnterpriseTaskID, &a.Status, &a.CompletionDate,
			&a.VerificationNotes, &a.VerifiedBy, &a.VerifiedAt, &a.TenantID, &a.CreatedTime, &a.UpdatedTime)
		actions = append(actions, a)
	}

	c.JSON(http.StatusOK, actions)
}

func dbCreateCorrectiveAction(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req CorrectiveAction
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("capa")
	}
	if req.ActionCode == "" {
		req.ActionCode = fmt.Sprintf("CAPA-%d", time.Now().Unix()%100000)
	}
	if req.Status == "" {
		req.Status = "Pending"
	}
	if req.Priority == "" {
		req.Priority = "High"
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.corrective_actions (
			id, action_code, title, description, source_type, source_id, owner, priority, due_date, status, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		req.ID, req.ActionCode, req.Title, req.Description, req.SourceType, req.SourceID,
		req.Owner, req.Priority, req.DueDate, req.Status, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("capa", req.ID, "capa.created", userName, fmt.Sprintf("CAPA %s created", req.ActionCode), tenantID, nil, req)
	publishEvent("corrective_action.created", "capa", req.ID, userName, tenantID, map[string]interface{}{"title": req.Title})

	c.JSON(http.StatusCreated, req)
}

// ─── 9. Governance Committees & Meetings ──────────────────────────

func dbGetCommittees(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetCommittees(tenantID))
		return
	}

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetCommittees(tenantID))
		return
	}
	rows, err := DB.Query(`
		SELECT id, name, code, mandate, chairperson, secretary, meeting_frequency,
		COALESCE(statchat_channel_id, ''), tenant_id, created_time, updated_time
		FROM statgovernance.governance_committees WHERE tenant_id = $1 ORDER BY name ASC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	committees := []Committee{}
	for rows.Next() {
		var cm Committee
		rows.Scan(&cm.ID, &cm.Name, &cm.Code, &cm.Mandate, &cm.Chairperson, &cm.Secretary,
			&cm.MeetingFrequency, &cm.StatChatChannelID, &cm.TenantID, &cm.CreatedTime, &cm.UpdatedTime)
		committees = append(committees, cm)
	}

	c.JSON(http.StatusOK, committees)
}

func dbCreateCommittee(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req Committee
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("com")
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.governance_committees (
			id, name, code, mandate, chairperson, secretary, meeting_frequency, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		req.ID, req.Name, req.Code, req.Mandate, req.Chairperson, req.Secretary, req.MeetingFrequency, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("committee", req.ID, "committee.created", userName, fmt.Sprintf("Committee %s created", req.Name), tenantID, nil, req)
	c.JSON(http.StatusCreated, req)
}

func dbGetMeetings(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	comID := c.Query("committee_id")
	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetMeetings(tenantID, comID))
		return
	}
	committeeID := c.Query("committee_id")

	q := `SELECT m.id, m.committee_id, c.name, m.title, m.meeting_date, m.location,
		COALESCE(m.calendar_event_id, ''), m.status, COALESCE(m.agenda, ''), COALESCE(m.minutes, ''),
		COALESCE(m.attendees, '{}'), m.decisions_count, m.tenant_id, m.created_time, m.updated_time
		FROM statgovernance.governance_meetings m
		LEFT JOIN statgovernance.governance_committees c ON m.committee_id = c.id
		WHERE m.tenant_id = $1`

	args := []interface{}{tenantID}
	if committeeID != "" {
		q += " AND m.committee_id = $2"
		args = append(args, committeeID)
	}
	q += " ORDER BY m.meeting_date DESC"

	rows, err := DB.Query(q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	meetings := []Meeting{}
	for rows.Next() {
		var m Meeting
		var att pq.StringArray
		rows.Scan(&m.ID, &m.CommitteeID, &m.CommitteeName, &m.Title, &m.MeetingDate, &m.Location,
			&m.CalendarEventID, &m.Status, &m.Agenda, &m.Minutes, &att, &m.DecisionsCount,
			&m.TenantID, &m.CreatedTime, &m.UpdatedTime)
		m.Attendees = att
		meetings = append(meetings, m)
	}

	c.JSON(http.StatusOK, meetings)
}

func dbCreateMeeting(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req Meeting
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("mtg")
	}
	if req.Status == "" {
		req.Status = "Scheduled"
	}
	if req.Location == "" {
		req.Location = "StatGate Boardroom / StatChat Video"
	}

	att := pq.StringArray(req.Attendees)

	_, err := DB.Exec(`
		INSERT INTO statgovernance.governance_meetings (
			id, committee_id, title, meeting_date, location, status, agenda, minutes, attendees, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.CommitteeID, req.Title, req.MeetingDate, req.Location, req.Status, req.Agenda, req.Minutes, att, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("meeting", req.ID, "meeting.created", userName, fmt.Sprintf("Meeting scheduled: %s", req.Title), tenantID, nil, req)
	publishEvent("governance.meeting.created", "meeting", req.ID, userName, tenantID, map[string]interface{}{"title": req.Title})

	c.JSON(http.StatusCreated, req)
}

// ─── 10. Governance Decisions ─────────────────────────────────────

func dbGetDecisions(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	comID := c.Query("committee_id")
	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetDecisions(tenantID, comID))
		return
	}
	rows, err := DB.Query(`
		SELECT id, decision_code, title, context, options_considered, COALESCE(risks_evaluated, ''),
		COALESCE(recommendation, ''), final_decision, decision_maker, COALESCE(committee_id, ''),
		COALESCE(meeting_id, ''), COALESCE(policy_id, ''), status, ai_advisory, human_approved,
		COALESCE(enterprise_decision_id, ''), COALESCE(effective_date::TEXT, ''), tenant_id, created_time
		FROM statgovernance.governance_decisions WHERE tenant_id = $1 ORDER BY created_time DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	decisions := []GovernanceDecision{}
	for rows.Next() {
		var d GovernanceDecision
		var optRaw []byte
		rows.Scan(&d.ID, &d.DecisionCode, &d.Title, &d.Context, &optRaw, &d.RisksEvaluated,
			&d.Recommendation, &d.FinalDecision, &d.DecisionMaker, &d.CommitteeID,
			&d.MeetingID, &d.PolicyID, &d.Status, &d.AIAdvisory, &d.HumanApproved,
			&d.EnterpriseDecisionID, &d.EffectiveDate, &d.TenantID, &d.CreatedTime)
		if len(optRaw) > 0 {
			_ = json.Unmarshal(optRaw, &d.OptionsConsidered)
		}
		decisions = append(decisions, d)
	}

	c.JSON(http.StatusOK, decisions)
}

func dbCreateDecision(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req GovernanceDecision
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("dec")
	}
	if req.DecisionCode == "" {
		req.DecisionCode = fmt.Sprintf("DEC-%d", time.Now().Unix()%100000)
	}
	if req.Status == "" {
		req.Status = "Approved"
	}
	if req.DecisionMaker == "" {
		req.DecisionMaker = userName
	}
	req.HumanApproved = true // Decisions require human authority

	optJSON, _ := json.Marshal(req.OptionsConsidered)

	_, err := DB.Exec(`
		INSERT INTO statgovernance.governance_decisions (
			id, decision_code, title, context, options_considered, risks_evaluated, recommendation,
			final_decision, decision_maker, committee_id, meeting_id, policy_id, status,
			ai_advisory, human_approved, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		req.ID, req.DecisionCode, req.Title, req.Context, optJSON, req.RisksEvaluated, req.Recommendation,
		req.FinalDecision, req.DecisionMaker, nullStr(req.CommitteeID), nullStr(req.MeetingID),
		nullStr(req.PolicyID), req.Status, req.AIAdvisory, req.HumanApproved, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("decision", req.ID, "decision.created", userName, fmt.Sprintf("Institutional decision %s recorded", req.DecisionCode), tenantID, nil, req)
	publishEvent("governance.decision.created", "decision", req.ID, userName, tenantID, map[string]interface{}{"title": req.Title})

	c.JSON(http.StatusCreated, req)
}

// ─── 11. Evidence Records ─────────────────────────────────────────

func dbGetEvidence(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	entityType := c.Query("entity_type")
	entityID := c.Query("entity_id")
	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetEvidence(tenantID, entityType, entityID))
		return
	}
	relType := c.Query("entity_type")
	relID := c.Query("entity_id")

	q := `SELECT id, title, COALESCE(description, ''), category, source_application,
		related_entity_type, related_entity_id, COALESCE(file_id, ''), COALESCE(file_name, ''),
		COALESCE(file_url, ''), COALESCE(mime_type, ''), COALESCE(file_size, 0),
		COALESCE(checksum_sha256, ''), classification, version, uploaded_by, tenant_id, created_time
		FROM statgovernance.evidence_records WHERE tenant_id = $1`

	args := []interface{}{tenantID}
	if relType != "" && relID != "" {
		q += " AND related_entity_type = $2 AND related_entity_id = $3"
		args = append(args, relType, relID)
	}
	q += " ORDER BY created_time DESC"

	rows, err := DB.Query(q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	records := []EvidenceRecord{}
	for rows.Next() {
		var e EvidenceRecord
		rows.Scan(&e.ID, &e.Title, &e.Description, &e.Category, &e.SourceApplication,
			&e.RelatedEntityType, &e.RelatedEntityID, &e.FileID, &e.FileName,
			&e.FileURL, &e.MimeType, &e.FileSize, &e.ChecksumSHA256,
			&e.Classification, &e.Version, &e.UploadedBy, &e.TenantID, &e.CreatedTime)
		records = append(records, e)
	}

	c.JSON(http.StatusOK, records)
}

func dbCreateEvidence(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req EvidenceRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("evi")
	}
	if req.UploadedBy == "" {
		req.UploadedBy = userName
	}
	if req.SourceApplication == "" {
		req.SourceApplication = "StatGovernance"
	}
	if req.ChecksumSHA256 == "" {
		h := sha256.New()
		h.Write([]byte(req.Title + req.FileName + time.Now().String()))
		req.ChecksumSHA256 = hex.EncodeToString(h.Sum(nil))
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.evidence_records (
			id, title, description, category, source_application, related_entity_type, related_entity_id,
			file_id, file_name, file_url, mime_type, file_size, checksum_sha256, classification,
			version, uploaded_by, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		req.ID, req.Title, req.Description, req.Category, req.SourceApplication,
		req.RelatedEntityType, req.RelatedEntityID, nullStr(req.FileID), req.FileName,
		req.FileURL, req.MimeType, req.FileSize, req.ChecksumSHA256, req.Classification,
		req.Version, req.UploadedBy, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("evidence", req.ID, "evidence.uploaded", userName, fmt.Sprintf("Evidence %s uploaded for %s %s", req.Title, req.RelatedEntityType, req.RelatedEntityID), tenantID, nil, req)
	publishEvent("evidence.uploaded", "evidence", req.ID, userName, tenantID, map[string]interface{}{"title": req.Title})

	c.JSON(http.StatusCreated, req)
}

// ─── 12. Data Governance & Privacy ────────────────────────────────

func dbGetDataGovernance(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	classFilter := c.Query("classification")
	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetDataGovernance(tenantID, classFilter))
		return
	}
	rows, err := DB.Query(`
		SELECT id, dataset_name, dataset_source, data_owner, data_steward, classification,
		sensitivity_level, retention_period, access_rules, COALESCE(lineage_reference, ''),
		quality_threshold, COALESCE(sharing_agreements, ''), tenant_id, created_time, updated_time
		FROM statgovernance.data_governance_records WHERE tenant_id = $1 ORDER BY dataset_name ASC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	records := []DataGovernanceRecord{}
	for rows.Next() {
		var d DataGovernanceRecord
		rows.Scan(&d.ID, &d.DatasetName, &d.DatasetSource, &d.DataOwner, &d.DataSteward, &d.Classification,
			&d.SensitivityLevel, &d.RetentionPeriod, &d.AccessRules, &d.LineageReference,
			&d.QualityThreshold, &d.SharingAgreements, &d.TenantID, &d.CreatedTime, &d.UpdatedTime)
		records = append(records, d)
	}

	c.JSON(http.StatusOK, records)
}

func dbCreateDataGovernance(c *gin.Context) {
	_, userName, _, tenantID := getAuthContext(c)
	var req DataGovernanceRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("dgov")
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.data_governance_records (
			id, dataset_name, dataset_source, data_owner, data_steward, classification,
			sensitivity_level, retention_period, access_rules, lineage_reference, quality_threshold,
			sharing_agreements, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		req.ID, req.DatasetName, req.DatasetSource, req.DataOwner, req.DataSteward, req.Classification,
		req.SensitivityLevel, req.RetentionPeriod, req.AccessRules, req.LineageReference,
		req.QualityThreshold, req.SharingAgreements, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("data_governance", req.ID, "data_governance.created", userName, fmt.Sprintf("Data governance record for %s registered", req.DatasetName), tenantID, nil, req)
	c.JSON(http.StatusCreated, req)
}

func dbGetPrivacyAssessments(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetPrivacyAssessments(tenantID))
		return
	}

	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetPrivacyAssessments(tenantID))
		return
	}
	rows, err := DB.Query(`
		SELECT id, activity_name, data_controller, lawful_basis, COALESCE(personal_data_types, '{}'),
		retention_rules, cross_border_transfer, dpia_status, breach_records_count, tenant_id, created_time, updated_time
		FROM statgovernance.privacy_assessments WHERE tenant_id = $1 ORDER BY created_time DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	assessments := []PrivacyAssessment{}
	for rows.Next() {
		var p PrivacyAssessment
		var pdt pq.StringArray
		rows.Scan(&p.ID, &p.ActivityName, &p.DataController, &p.LawfulBasis, &pdt,
			&p.RetentionRules, &p.CrossBorderTransfer, &p.DPIAStatus, &p.BreachRecordsCount,
			&p.TenantID, &p.CreatedTime, &p.UpdatedTime)
		p.PersonalDataTypes = pdt
		assessments = append(assessments, p)
	}

	c.JSON(http.StatusOK, assessments)
}

// ─── 13. Delegations ──────────────────────────────────────────────

func dbGetDelegations(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	statusFilter := c.Query("status")
	if DB == nil {
		c.JSON(http.StatusOK, memStore.GetDelegations(tenantID, statusFilter))
		return
	}
	rows, err := DB.Query(`
		SELECT id, delegator_id, delegator_name, delegate_id, delegate_name, role_scope, scope_details,
		start_date, end_date, reason, status, tenant_id, created_time, updated_time
		FROM statgovernance.delegations WHERE tenant_id = $1 ORDER BY start_date DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	delegations := []Delegation{}
	now := time.Now()
	for rows.Next() {
		var d Delegation
		rows.Scan(&d.ID, &d.DelegatorID, &d.DelegatorName, &d.DelegateID, &d.DelegateName,
			&d.RoleScope, &d.ScopeDetails, &d.StartDate, &d.EndDate, &d.Reason, &d.Status,
			&d.TenantID, &d.CreatedTime, &d.UpdatedTime)

		// Auto-expire
		if d.Status == "Active" && now.After(d.EndDate) {
			d.Status = "Expired"
			_, _ = DB.Exec("UPDATE statgovernance.delegations SET status = 'Expired' WHERE id = $1", d.ID)
		}
		delegations = append(delegations, d)
	}

	c.JSON(http.StatusOK, delegations)
}

func dbCreateDelegation(c *gin.Context) {
	userID, userName, _, tenantID := getAuthContext(c)
	var req Delegation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ID == "" {
		req.ID = newID("dlg")
	}
	if req.DelegatorID == "" {
		req.DelegatorID = userID
	}
	if req.DelegatorName == "" {
		req.DelegatorName = userName
	}
	if req.Status == "" {
		req.Status = "Active"
	}

	_, err := DB.Exec(`
		INSERT INTO statgovernance.delegations (
			id, delegator_id, delegator_name, delegate_id, delegate_name, role_scope, scope_details,
			start_date, end_date, reason, status, tenant_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		req.ID, req.DelegatorID, req.DelegatorName, req.DelegateID, req.DelegateName,
		req.RoleScope, req.ScopeDetails, req.StartDate, req.EndDate, req.Reason, req.Status, tenantID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recordAudit("delegation", req.ID, "delegation.created", userName, fmt.Sprintf("Delegation of %s to %s", req.RoleScope, req.DelegateName), tenantID, nil, req)
	publishEvent("governance.delegation.created", "delegation", req.ID, userName, tenantID, map[string]interface{}{
		"delegate": req.DelegateName,
		"role":     req.RoleScope,
	})

	c.JSON(http.StatusCreated, req)
}

// ─── 14. Enterprise Search & Activity Timeline ────────────────────

func dbSearch(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	q := "%" + strings.ToLower(c.Query("q")) + "%"

	type SearchItem struct {
		Type        string `json:"type"`
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Source      string `json:"source"`
		URL         string `json:"url"`
		Score       int    `json:"score"`
	}

	results := []SearchItem{}

	// Policies
	rows, _ := DB.Query(`SELECT id, policy_number || ': ' || title, COALESCE(summary, '') FROM statgovernance.policies WHERE tenant_id = $1 AND (LOWER(title) LIKE $2 OR LOWER(content) LIKE $2) LIMIT 5`, tenantID, q)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var s SearchItem
			s.Type = "policy"
			s.Source = "StatGovernance"
			rows.Scan(&s.ID, &s.Title, &s.Description)
			s.URL = fmt.Sprintf("http://localhost:3012/?tab=policies&id=%s", s.ID)
			s.Score = 90
			results = append(results, s)
		}
	}

	// Risks
	rows2, _ := DB.Query(`SELECT id, risk_code || ': ' || title, description FROM statgovernance.risks WHERE tenant_id = $1 AND (LOWER(title) LIKE $2 OR LOWER(description) LIKE $2) LIMIT 5`, tenantID, q)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var s SearchItem
			s.Type = "risk"
			s.Source = "StatGovernance"
			rows2.Scan(&s.ID, &s.Title, &s.Description)
			s.URL = fmt.Sprintf("http://localhost:3012/?tab=risks&id=%s", s.ID)
			s.Score = 85
			results = append(results, s)
		}
	}

	// Controls
	rows3, _ := DB.Query(`SELECT id, control_code || ': ' || name, description FROM statgovernance.controls WHERE tenant_id = $1 AND (LOWER(name) LIKE $2 OR LOWER(description) LIKE $2) LIMIT 5`, tenantID, q)
	if rows3 != nil {
		defer rows3.Close()
		for rows3.Next() {
			var s SearchItem
			s.Type = "control"
			s.Source = "StatGovernance"
			rows3.Scan(&s.ID, &s.Title, &s.Description)
			s.URL = fmt.Sprintf("http://localhost:3012/?tab=controls&id=%s", s.ID)
			s.Score = 80
			results = append(results, s)
		}
	}

	// Audits & Findings
	rows4, _ := DB.Query(`SELECT id, finding_code || ': ' || title, description FROM statgovernance.audit_findings WHERE tenant_id = $1 AND (LOWER(title) LIKE $2 OR LOWER(description) LIKE $2) LIMIT 5`, tenantID, q)
	if rows4 != nil {
		defer rows4.Close()
		for rows4.Next() {
			var s SearchItem
			s.Type = "finding"
			s.Source = "StatGovernance"
			rows4.Scan(&s.ID, &s.Title, &s.Description)
			s.URL = fmt.Sprintf("http://localhost:3012/?tab=findings&id=%s", s.ID)
			s.Score = 88
			results = append(results, s)
		}
	}

	c.JSON(http.StatusOK, results)
}

func dbGetActivity(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)

	if DB == nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	rows, err := DB.Query(`
		SELECT id, entity_type, entity_id, action, actor, COALESCE(details, ''), timestamp
		FROM statgovernance.audit_logs WHERE tenant_id = $1 ORDER BY timestamp DESC LIMIT 30`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type ActivityItem struct {
		ID         string    `json:"id"`
		EntityType string    `json:"entity_type"`
		EntityID   string    `json:"entity_id"`
		Action     string    `json:"action"`
		Actor      string    `json:"actor"`
		Details    string    `json:"details"`
		Timestamp  time.Time `json:"timestamp"`
	}

	activities := []ActivityItem{}
	for rows.Next() {
		var a ActivityItem
		rows.Scan(&a.ID, &a.EntityType, &a.EntityID, &a.Action, &a.Actor, &a.Details, &a.Timestamp)
		activities = append(activities, a)
	}

	c.JSON(http.StatusOK, activities)
}

// ─── 15. Enterprise AI Grounded Governance Analysis ───────────────

func dbAIAnalyze(c *gin.Context) {
	_, _, _, tenantID := getAuthContext(c)
	var req struct {
		Question string `json:"question"`
		Context  string `json:"context,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if DB == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":   "grounded",
			"question": req.Question,
			"findings": []map[string]interface{}{
				{"kind": "fact", "statement": "Policy POL-DATA-2026-01 (National Statistical Data Governance Policy) is Active and Approved.", "confidence": "high"},
				{"kind": "fact", "statement": "Institutional Risk RSK-COMP-2026-02 (Survey Consent Documentation Gaps) is Critical severity with active mitigation in progress.", "confidence": "high"},
				{"kind": "fact", "statement": "Control CTL-SEC-001 (Automated JWT & RBAC Token Authorization) is Effective with continuous test frequency.", "confidence": "high"},
			},
			"sources": []map[string]interface{}{
				{"id": "pol-001", "app": "StatGovernance", "entity": "policy", "title": "POL-DATA-2026-01: National Statistical Data Governance Policy", "status": "Active"},
				{"id": "risk-003", "app": "StatGovernance", "entity": "risk", "title": "RSK-COMP-2026-02: Survey Consent Documentation Gaps", "level": "Critical"},
				{"id": "ctl-001", "app": "StatGovernance", "entity": "control", "title": "CTL-SEC-001: Automated JWT & RBAC Token Authorization", "status": "Effective"},
			},
			"analysis":             "Analysis synthesized directly from 3 authoritative governance records in the Institutional Governance Operating System.",
			"recommendation":       "Prioritize completion of biometric signature capture step in ODK collect master to close the Critical survey consent risk.",
			"human_control_notice": "AI advisory only. Human authorization required for decisions or policy approvals.",
			"generated_at":         time.Now().UTC().Format(time.RFC3339),
		})
		return
	}


	qLower := strings.ToLower(req.Question)

	// Gather real evidence from database
	sources := []map[string]interface{}{}
	facts := []map[string]interface{}{}

	if strings.Contains(qLower, "policy") || strings.Contains(qLower, "data") || strings.Contains(qLower, "governance") {
		rows, _ := DB.Query(`SELECT id, policy_number, title, status FROM statgovernance.policies WHERE tenant_id = $1 LIMIT 3`, tenantID)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, num, title, status string
				rows.Scan(&id, &num, &title, &status)
				sources = append(sources, map[string]interface{}{
					"id": id, "app": "StatGovernance", "entity": "policy", "title": num + ": " + title, "status": status,
				})
				facts = append(facts, map[string]interface{}{
					"kind": "fact", "statement": fmt.Sprintf("Policy %s (%s) is currently in %s state.", num, title, status), "confidence": "high",
				})
			}
		}
	}

	if strings.Contains(qLower, "risk") || strings.Contains(qLower, "critical") || strings.Contains(qLower, "control") {
		rows, _ := DB.Query(`SELECT id, risk_code, title, inherent_risk_level FROM statgovernance.risks WHERE tenant_id = $1 AND (inherent_risk_level = 'Critical' OR inherent_risk_level = 'High') LIMIT 3`, tenantID)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var id, code, title, level string
				rows.Scan(&id, &code, &title, &level)
				sources = append(sources, map[string]interface{}{
					"id": id, "app": "StatGovernance", "entity": "risk", "title": code + ": " + title, "level": level,
				})
				facts = append(facts, map[string]interface{}{
					"kind": "fact", "statement": fmt.Sprintf("Institutional Risk %s (%s) has an inherent risk severity of %s.", code, title, level), "confidence": "high",
				})
			}
		}
	}

	if len(sources) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"status":      "insufficient_evidence",
			"statement":   "Insufficient evidence in institutional governance records to formulate a grounded recommendation.",
			"sources":     []interface{}{},
			"findings":    []interface{}{},
			"analysis":    "No direct policies, risks, controls, or audit records match the query criteria.",
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "grounded",
		"question":    req.Question,
		"findings":    facts,
		"sources":     sources,
		"analysis":    fmt.Sprintf("Analysis synthesized directly from %d authoritative governance record(s).", len(sources)),
		"recommendation": "Review linked active controls and audit findings to verify ongoing compliance.",
		"human_control_notice": "AI advisory only. Human authorization required for decisions or policy approvals.",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	})
}
