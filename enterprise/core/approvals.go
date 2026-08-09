package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Approval Engine ──────────────────────────────────────────────────
// Phase V: A reusable enterprise approval system supporting single,
// multiple, sequential, parallel, conditional, and delegated approvals.
// Approval actions are recorded in the enterprise audit trail.

// createApprovalRequest creates and stores a new approval request.
// Returns the created request (with generated ID) so callers receive
// the full persisted state.
func createApprovalRequest(req ApprovalRequest) ApprovalRequest {
	if req.ID == "" {
		req.ID = fmt.Sprintf("apr_%d", time.Now().UnixNano())
	}
	if req.CreatedAt == "" {
		req.CreatedAt = nowUTC()
	}
	req.UpdatedAt = nowUTC()
	if req.Status == "" {
		req.Status = "pending"
	}

	workflowMu.Lock()
	approvalRequests[req.ID] = req
	workflowMu.Unlock()

	// Persist to Redis
	persistApprovalRequest(req)

	// Record audit
	recordAudit("approval.requested", "enterprise", req.Requester, map[string]interface{}{
		"approval_id": req.ID, "title": req.Title, "type": req.Type,
		"approvers": req.Approvers, "entity_type": req.EntityType, "entity_id": req.EntityID,
		"project_id": req.ProjectID, "workflow_id": req.WorkflowID, "instance_id": req.InstanceID,
	})

	// Create timeline entry
	recordTimelineEntry(TimelineEntry{
		ID:          fmt.Sprintf("tl_%d", time.Now().UnixNano()),
		User:        req.Requester,
		Application: "enterprise",
		Entity:      "approval",
		EntityID:    req.ID,
		Action:      "approval.requested",
		Description: fmt.Sprintf("Approval requested: %s", req.Title),
		ProjectID:   req.ProjectID,
		OrgID:       req.OrgID,
		Metadata: map[string]interface{}{
			"approval_id": req.ID, "type": req.Type, "approvers": req.Approvers,
			"workflow_id": req.WorkflowID, "instance_id": req.InstanceID,
		},
		Timestamp: nowUTC(),
	})

	// Publish analytics event
	publishAnalyticsEvent("approval.requested", map[string]interface{}{
		"approval_id": req.ID, "title": req.Title, "type": req.Type,
		"approvers": req.Approvers, "project_id": req.ProjectID,
		"workflow_id": req.WorkflowID, "instance_id": req.InstanceID,
	})

	return req
}

func persistApprovalRequest(req ApprovalRequest) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(req)
	ctx := context.Background()
	key := fmt.Sprintf("statgate:approval:%s", req.ID)
	redisClient.Set(ctx, key, string(data), 30*24*time.Hour)
	redisClient.LPush(ctx, "statgate:approvals", req.ID)
	redisClient.LTrim(ctx, "statgate:approvals", 0, 1999)
}

func getApprovalRequest(id string) (ApprovalRequest, bool) {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	req, ok := approvalRequests[id]
	return req, ok
}

func listApprovalRequests(approver, status string) []ApprovalRequest {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]ApprovalRequest, 0, len(approvalRequests))
	for _, req := range approvalRequests {
		if approver != "" {
			found := false
			for _, a := range req.Approvers {
				if a == approver {
					found = true
					break
				}
			}
			if !found && req.Requester != approver {
				continue
			}
		}
		if status != "" && req.Status != status {
			continue
		}
		out = append(out, req)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func updateApprovalRequest(id string, mutate func(*ApprovalRequest)) (ApprovalRequest, bool) {
	workflowMu.Lock()
	req, ok := approvalRequests[id]
	if !ok {
		workflowMu.Unlock()
		return req, false
	}
	mutate(&req)
	req.UpdatedAt = nowUTC()
	approvalRequests[id] = req
	workflowMu.Unlock()
	persistApprovalRequest(req)
	return req, true
}

// approveApproval handles an approve decision.
func approveApproval(id, approver, comment string) (ApprovalRequest, error) {
	req, ok := getApprovalRequest(id)
	if !ok {
		return req, fmt.Errorf("approval request not found")
	}
	if req.Status != "pending" {
		return req, fmt.Errorf("approval already resolved")
	}

	// Check if approver is valid
	if !isAuthorizedApprover(req, approver) {
		return req, fmt.Errorf("user %s is not an authorized approver", approver)
	}

	decision := ApprovalDecision{
		Approver:  approver,
		Decision:  "approved",
		Comment:   comment,
		Timestamp: nowUTC(),
	}

	var shouldComplete bool
	var updated ApprovalRequest
	updated, _ = updateApprovalRequest(id, func(r *ApprovalRequest) {
		r.Approvals = append(r.Approvals, decision)
		approvalCount := countApprovals(r.Approvals)
		if r.Type == "single" || r.Type == "conditional" {
			r.Status = "approved"
			shouldComplete = true
		} else if r.Type == "parallel" {
			if approvalCount >= r.MinApprovals || approvalCount >= len(r.Approvers) {
				r.Status = "approved"
				shouldComplete = true
			}
		} else if r.Type == "sequential" {
			// Move to next approver or complete
			idx := -1
			for i, a := range r.Approvers {
				if a == approver {
					idx = i
					break
				}
			}
			if idx >= 0 && idx < len(r.Approvers)-1 {
				r.CurrentApprover = r.Approvers[idx+1]
			} else {
				r.Status = "approved"
				shouldComplete = true
			}
		} else {
			if approvalCount >= len(r.Approvers) {
				r.Status = "approved"
				shouldComplete = true
			}
		}
	})
	if shouldComplete {
		onApprovalComplete(updated)
	}

	return updated, nil
}

// rejectApproval handles a rejection decision.
func rejectApproval(id, approver, comment string) (ApprovalRequest, error) {
	req, ok := getApprovalRequest(id)
	if !ok {
		return req, fmt.Errorf("approval request not found")
	}
	if req.Status != "pending" {
		return req, fmt.Errorf("approval already resolved")
	}
	if !isAuthorizedApprover(req, approver) {
		return req, fmt.Errorf("user %s is not an authorized approver", approver)
	}

	decision := ApprovalDecision{
		Approver:  approver,
		Decision:  "rejected",
		Comment:   comment,
		Timestamp: nowUTC(),
	}

	updated, _ := updateApprovalRequest(id, func(r *ApprovalRequest) {
		r.Approvals = append(r.Approvals, decision)
		r.Status = "rejected"
	})
	onApprovalRejected(updated)

	return updated, nil
}

// requestApprovalChanges handles a request-for-changes decision.
func requestApprovalChanges(id, approver, comment string) (ApprovalRequest, error) {
	req, ok := getApprovalRequest(id)
	if !ok {
		return req, fmt.Errorf("approval request not found")
	}
	if req.Status != "pending" {
		return req, fmt.Errorf("approval already resolved")
	}
	if !isAuthorizedApprover(req, approver) {
		return req, fmt.Errorf("user %s is not an authorized approver", approver)
	}

	decision := ApprovalDecision{
		Approver:  approver,
		Decision:  "changes_requested",
		Comment:   comment,
		Timestamp: nowUTC(),
	}

	updated, _ := updateApprovalRequest(id, func(r *ApprovalRequest) {
		r.Approvals = append(r.Approvals, decision)
		r.Status = "changes_requested"
	})

	return updated, nil
}

// delegateApproval handles delegated approval.
func delegateApproval(id, approver, delegateTo, comment string) (ApprovalRequest, error) {
	req, ok := getApprovalRequest(id)
	if !ok {
		return req, fmt.Errorf("approval request not found")
	}
	if req.Status != "pending" {
		return req, fmt.Errorf("approval already resolved")
	}
	if !isAuthorizedApprover(req, approver) {
		return req, fmt.Errorf("user %s is not an authorized approver", approver)
	}
	if delegateTo == "" {
		return req, fmt.Errorf("delegate_to required")
	}

	decision := ApprovalDecision{
		Approver:    approver,
		Decision:    "delegated",
		Comment:     comment,
		DelegatedTo: delegateTo,
		Timestamp:   nowUTC(),
	}

	updated, _ := updateApprovalRequest(id, func(r *ApprovalRequest) {
		r.Approvals = append(r.Approvals, decision)
		r.Approvers = append(r.Approvers, delegateTo)
		if r.CurrentApprover == approver {
			r.CurrentApprover = delegateTo
		}
	})

	// Notify delegate
	_, _ = createNotificationRecord(Notification{
		UserID:         delegateTo,
		Title:          "Approval Delegated to You",
		Body:           fmt.Sprintf("%s has delegated approval %s to you", approver, req.Title),
		Priority:       "high",
		Category:       "approval",
		SourceApp:      "enterprise",
		SourceEntity:   "approval",
		SourceEntityID: req.ID,
		DeepLink:       "/action-centre/approvals/" + req.ID,
		Metadata: map[string]interface{}{
			"approval_id": req.ID, "delegated_by": approver,
		},
	})

	return updated, nil
}

// escalateApproval escalates an approval to the configured escalate-to user.
func escalateApproval(id, reason string) (ApprovalRequest, error) {
	req, ok := getApprovalRequest(id)
	if !ok {
		return req, fmt.Errorf("approval request not found")
	}
	if req.Status != "pending" {
		return req, fmt.Errorf("approval already resolved")
	}

	escalateTo := req.EscalatedTo
	if escalateTo == "" {
		// Look up the workflow definition for escalation target
		if req.WorkflowID != "" {
			if def, ok := getWorkflowDefinition(req.WorkflowID); ok && def.Approval != nil {
				escalateTo = def.Approval.EscalateTo
			}
		}
	}
	if escalateTo == "" {
		return req, fmt.Errorf("no escalation target configured")
	}

	updated, _ := updateApprovalRequest(id, func(r *ApprovalRequest) {
		r.Status = "escalated"
		r.EscalatedTo = escalateTo
		r.Context = map[string]interface{}{"escalation_reason": reason}
	})

	// Notify escalation target
	_, _ = createNotificationRecord(Notification{
		UserID:         escalateTo,
		Title:          "Approval Escalated",
		Body:           fmt.Sprintf("Approval %s has been escalated: %s", req.Title, reason),
		Priority:       "critical",
		Category:       "approval",
		SourceApp:      "enterprise",
		SourceEntity:   "approval",
		SourceEntityID: req.ID,
		DeepLink:       "/action-centre/approvals/" + req.ID,
		Metadata: map[string]interface{}{
			"approval_id": req.ID, "escalation_reason": reason,
		},
	})

	return updated, nil
}

// expireApproval marks an approval as expired and escalates if configured.
func expireApproval(id string) {
	req, ok := getApprovalRequest(id)
	if !ok {
		return
	}
	if req.Status != "pending" {
		return
	}

	updated, _ := updateApprovalRequest(id, func(r *ApprovalRequest) {
		r.Status = "expired"
	})
	onApprovalExpired(updated)
}

// ─── Approval Helpers ─────────────────────────────────────────────

func isAuthorizedApprover(req ApprovalRequest, user string) bool {
	if user == "" {
		return false
	}
	for _, a := range req.Approvers {
		if a == user {
			return true
		}
	}
	// Check grants
	grants := fetchGrants(user)
	for _, g := range grants {
		if g.Status == "active" && g.Resource == "approval" {
			return true
		}
	}
	return false
}

func countApprovals(decisions []ApprovalDecision) int {
	count := 0
	for _, d := range decisions {
		if d.Decision == "approved" {
			count++
		}
	}
	return count
}

// onApprovalComplete continues a workflow that was waiting for approval.
func onApprovalComplete(req ApprovalRequest) {
	// Record audit
	recordAudit("approval.approved", "enterprise", req.Requester, map[string]interface{}{
		"approval_id": req.ID, "title": req.Title, "project_id": req.ProjectID,
		"workflow_id": req.WorkflowID, "instance_id": req.InstanceID,
	})

	// Create timeline
	recordTimelineEntry(TimelineEntry{
		ID:          fmt.Sprintf("tl_%d", time.Now().UnixNano()),
		User:        req.Requester,
		Application: "enterprise",
		Entity:      "approval",
		EntityID:    req.ID,
		Action:      "approval.approved",
		Description: fmt.Sprintf("Approval completed: %s", req.Title),
		ProjectID:   req.ProjectID,
		OrgID:       req.OrgID,
		Metadata: map[string]interface{}{
			"approval_id": req.ID, "workflow_id": req.WorkflowID,
		},
		Timestamp: nowUTC(),
	})

	// Notify requester
	if req.Requester != "" {
		_, _ = createNotificationRecord(Notification{
			UserID:         req.Requester,
			Title:          "Approval Approved",
			Body:           req.Title,
			Priority:       "high",
			Category:       "approval",
			SourceApp:      "enterprise",
			SourceEntity:   "approval",
			SourceEntityID: req.ID,
			DeepLink:       "/action-centre/approvals/" + req.ID,
		})
	}

	// Publish event
	correlationID := ""
	if req.Context != nil {
		if v, ok := req.Context["correlation_id"].(string); ok {
			correlationID = v
		}
	}
	publishEvent(DomainEvent{
		ID:          fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType:   "approval.approved",
		Source:      "enterprise",
		ObjectType:  "approval",
		ObjectID:    req.ID,
		Actor:       req.Requester,
		ProjectID:   req.ProjectID,
		OrgID:       req.OrgID,
		Payload:     map[string]interface{}{"title": req.Title, "approval_id": req.ID},
		Timestamp:   nowUTC(),
		Correlation: correlationID,
	})

	// If this approval is linked to a workflow that was waiting, resume it
	if req.InstanceID != "" {
		if instance, ok := getWorkflowInstance(req.InstanceID); ok {
			if instance.Status == "waiting_approval" {
				inst := instance
				inst.Status = "running"
				updateWorkflowInstance(inst)
				go executeWorkflowSteps(inst)
			}
		}
	}

	// Publish analytics event
	publishAnalyticsEvent("approval.approved", map[string]interface{}{
		"approval_id": req.ID, "title": req.Title, "workflow_id": req.WorkflowID,
		"instance_id": req.InstanceID, "project_id": req.ProjectID,
	})
}

// onApprovalRejected handles rejection.
func onApprovalRejected(req ApprovalRequest) {
	// Record audit
	recordAudit("approval.rejected", "enterprise", req.Requester, map[string]interface{}{
		"approval_id": req.ID, "title": req.Title, "project_id": req.ProjectID,
		"workflow_id": req.WorkflowID, "instance_id": req.InstanceID,
	})

	// Notify requester
	if req.Requester != "" {
		_, _ = createNotificationRecord(Notification{
			UserID:         req.Requester,
			Title:          "Approval Rejected",
			Body:           req.Title,
			Priority:       "high",
			Category:       "approval",
			SourceApp:      "enterprise",
			SourceEntity:   "approval",
			SourceEntityID: req.ID,
			DeepLink:       "/action-centre/approvals/" + req.ID,
		})
	}

	// Publish event
	correlationID := ""
	if req.Context != nil {
		if v, ok := req.Context["correlation_id"].(string); ok {
			correlationID = v
		}
	}
	publishEvent(DomainEvent{
		ID:          fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType:   "approval.rejected",
		Source:      "enterprise",
		ObjectType:  "approval",
		ObjectID:    req.ID,
		Actor:       req.Requester,
		ProjectID:   req.ProjectID,
		OrgID:       req.OrgID,
		Payload:     map[string]interface{}{"title": req.Title, "approval_id": req.ID},
		Timestamp:   nowUTC(),
		Correlation: correlationID,
	})

	// If linked to workflow, mark the workflow as failed on rejection
	if req.InstanceID != "" {
		failWorkflowInstance(req.InstanceID, fmt.Sprintf("approval %s was rejected", req.ID))
	}
}

// onApprovalExpired handles approval expiry.
func onApprovalExpired(req ApprovalRequest) {
	// Record audit
	recordAudit("approval.expired", "enterprise", req.Requester, map[string]interface{}{
		"approval_id": req.ID, "title": req.Title, "project_id": req.ProjectID,
	})

	// If workflow is linked and escalation is configured, escalate
	if req.InstanceID != "" {
		// Check if escalation is configured on the workflow
		if req.WorkflowID != "" {
			if def, ok := getWorkflowDefinition(req.WorkflowID); ok && def.Approval != nil {
				if def.Approval.EscalateTo != "" {
					_, _ = escalateApproval(req.ID, "Approval expired without resolution")
					return
				}
			}
		}
		// Otherwise mark workflow as cancelled due to expiry
		updateWorkflowStatus(req.InstanceID, "cancelled", "")
	}
}

// ─── Scheduled Approval Expiry Sweep ─────────────────────────────────

func runApprovalExpirySweep() {
	now := time.Now()
	workflowMu.RLock()
	pending := make([]ApprovalRequest, 0)
	for _, req := range approvalRequests {
		if req.Status == "pending" && req.ExpiresAt != "" {
			if expiry, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
				if now.After(expiry) {
					pending = append(pending, req)
				}
			}
		}
	}
	workflowMu.RUnlock()

	for _, req := range pending {
		expireApproval(req.ID)
		log.Printf("approval: expired approval %s (%s)", req.ID, req.Title)
	}
}

// ─── API Handlers ─────────────────────────────────────────────────

func handleListApprovals(c *gin.Context) {
	approver := c.Query("approver")
	if approver == "" {
		approver = c.GetHeader("X-User-ID")
	}
	status := c.Query("status")
	requests := listApprovalRequests(approver, status)
	c.JSON(200, gin.H{"count": len(requests), "approvals": requests})
}

func handleGetApproval(c *gin.Context) {
	id := c.Param("id")
	req, ok := getApprovalRequest(id)
	if !ok {
		c.JSON(404, gin.H{"error": "approval not found"})
		return
	}
	c.JSON(200, req)
}

func handleCreateApproval(c *gin.Context) {
	var req ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid approval request", "detail": err.Error()})
		return
	}
	if len(req.Approvers) == 0 {
		c.JSON(400, gin.H{"error": "at least one approver required"})
		return
	}
	requester := c.GetHeader("X-User-ID")
	if requester == "" {
		requester = c.Query("requester")
	}
	req.Requester = requester
	created := createApprovalRequest(req)
	c.JSON(201, created)
}

func handleApprovalAction(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}

	var body struct {
		Comment    string `json:"comment"`
		DelegateTo string `json:"delegate_to"`
		Reason     string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	var result interface{}
	var err error

	switch action {
	case "approve":
		result, err = approveApproval(id, userID, body.Comment)
	case "reject":
		result, err = rejectApproval(id, userID, body.Comment)
	case "changes":
		result, err = requestApprovalChanges(id, userID, body.Comment)
	case "delegate":
		result, err = delegateApproval(id, userID, body.DelegateTo, body.Comment)
	case "escalate":
		result, err = escalateApproval(id, body.Reason)
	default:
		c.JSON(400, gin.H{"error": "unsupported action", "supported": []string{"approve", "reject", "changes", "delegate", "escalate"}})
		return
	}

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, result)
}
