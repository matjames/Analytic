package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AIInvestigation is the governed container that joins evidence, a human
// decision and the actions/outcomes managed by existing Enterprise Core APIs.
// It intentionally does not duplicate workflow, task, notification or
// knowledge storage.
type AIInvestigation struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Question         string     `json:"question"`
	Owner            string     `json:"owner"`
	TenantID         string     `json:"tenant_id"`
	OrganizationID   string     `json:"organization_id,omitempty"`
	ProjectID        string     `json:"project_id,omitempty"`
	SourceApp        string     `json:"source_application,omitempty"`
	SourceObjectType string     `json:"source_object_type,omitempty"`
	SourceObjectID   string     `json:"source_object_id,omitempty"`
	Priority         string     `json:"priority,omitempty"`
	CorrelationID    string     `json:"correlation_id"`
	Status           string     `json:"status"`
	Evidence         []AISource `json:"evidence"`
	ResponseID       string     `json:"response_id,omitempty"`
	RecommendationID string     `json:"recommendation_id,omitempty"`
	TaskID           string     `json:"task_id,omitempty"`
	WorkflowID       string     `json:"workflow_id,omitempty"`
	DecisionID       string     `json:"decision_id,omitempty"`
	Outcome          string     `json:"outcome,omitempty"`
	CreatedAt        string     `json:"created_at"`
	UpdatedAt        string     `json:"updated_at"`
}

var investigationStore = struct {
	sync.RWMutex
	items map[string]AIInvestigation
}{items: map[string]AIInvestigation{}}

func validInvestigationStatus(status string) bool {
	for _, candidate := range []string{"open", "investigating", "findings_ready", "action_required", "in_progress", "verifying", "awaiting_evidence", "awaiting_decision", "action_in_progress", "monitoring", "resolved", "closed", "dismissed"} {
		if status == candidate {
			return true
		}
	}
	return false
}

func investigationStatus(status string) string { return strings.ToLower(strings.TrimSpace(status)) }

func saveInvestigation(item AIInvestigation) {
	investigationStore.Lock()
	investigationStore.items[item.ID] = item
	investigationStore.Unlock()
	if err := persistInvestigationToDB(item); err != nil {
		// The in-memory view remains available for the current request, but the
		// operation is auditable as a persistence failure and must be remediated.
		recordAudit("ai.investigation.persistence_failed", "enterprise", item.Owner, map[string]interface{}{"investigation_id": item.ID, "error": err.Error()})
	}
	if redisClient != nil {
		data, _ := jsonMarshal(item)
		redisClient.Set(context.Background(), "statgate:ai:investigation:"+item.ID, string(data), 7*24*time.Hour)
	}
}

func listInvestigations(tenant, owner, status string) []AIInvestigation {
	investigationStore.RLock()
	defer investigationStore.RUnlock()
	items := make([]AIInvestigation, 0, len(investigationStore.items))
	for _, item := range investigationStore.items {
		if item.TenantID != tenant || (owner != "" && item.Owner != owner) || (status != "" && item.Status != status) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	return items
}

func handleAIInvestigationsV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	owner := user
	if evaluatePermission(user, "ai", "admin") && c.Query("owner") != "" {
		owner = c.Query("owner")
	}
	c.JSON(200, gin.H{"investigations": listInvestigations(aiTenant(c), owner, c.Query("status")), "requested_by": user})
}

func handleAICreateInvestigationV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "write")
	if !ok {
		return
	}
	var body struct {
		Title            string `json:"title"`
		Question         string `json:"question"`
		OrganizationID   string `json:"organization_id"`
		ProjectID        string `json:"project_id"`
		ResponseID       string `json:"response_id"`
		RecommendationID string `json:"recommendation_id"`
		Assignee         string `json:"assignee"`
		DueAt            string `json:"due_at"`
		WorkflowID       string `json:"workflow_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" || body.Question == "" {
		c.JSON(400, gin.H{"error": "title and question are required"})
		return
	}

	var evidence []AISource
	if body.ResponseID != "" {
		aiStore.RLock()
		response, found := aiStore.responses[body.ResponseID]
		aiStore.RUnlock()
		if !found {
			c.JSON(404, gin.H{"error": "AI response not found"})
			return
		}
		aiStore.RLock()
		responseContext, contextFound := aiStore.contexts[response.ContextID]
		aiStore.RUnlock()
		if !contextFound || responseContext.TenantID != aiTenant(c) || responseContext.UserID != user {
			c.JSON(404, gin.H{"error": "AI response not found"})
			return
		}
		evidence = response.Sources
	}
	if body.RecommendationID != "" {
		if rec, found := getAIRecommendation(body.RecommendationID); !found || rec.Status != "accepted" {
			c.JSON(409, gin.H{"error": "an investigation can only be created from an accepted recommendation"})
			return
		}
	}
	now := nowUTC()
	item := AIInvestigation{ID: fmt.Sprintf("aiinv_%d", time.Now().UnixNano()), Title: body.Title, Question: body.Question, Owner: user, TenantID: aiTenant(c), OrganizationID: body.OrganizationID, ProjectID: body.ProjectID, Status: "open", Evidence: evidence, ResponseID: body.ResponseID, RecommendationID: body.RecommendationID, WorkflowID: body.WorkflowID, Priority: "high", CreatedAt: now, UpdatedAt: now}
	item.CorrelationID = item.ID
	if len(evidence) > 0 {
		item.SourceApp, item.SourceObjectType, item.SourceObjectID = evidence[0].App, evidence[0].Entity, evidence[0].RecordID
	}
	assignee := body.Assignee
	if assignee == "" {
		assignee = user
	}
	task := EnterpriseTask{ID: fmt.Sprintf("task_%d", time.Now().UnixNano()), Title: "Investigate: " + item.Title, Description: item.Question, Status: "pending", Priority: "high", Assignee: assignee, Assigner: user, SourceApp: "enterprise_ai", SourceEntity: "ai_investigation", SourceEntityID: item.ID, ProjectID: item.ProjectID, OrgID: item.OrganizationID, DueAt: body.DueAt, WorkflowID: item.WorkflowID, CorrelationID: item.ID, CreatedAt: now, UpdatedAt: now}
	createTaskRecord(task)
	item.TaskID = task.ID
	saveInvestigation(item)
	createKnowledgeRelationship(KnowledgeRelationship{FromType: "ai_investigation", FromID: item.ID, ToType: "task", ToID: task.ID, Relation: "action_for", Source: "enterprise_ai", CreatedBy: user})
	if item.RecommendationID != "" {
		createKnowledgeRelationship(KnowledgeRelationship{FromType: "ai_investigation", FromID: item.ID, ToType: "ai_recommendation", ToID: item.RecommendationID, Relation: "investigates", Source: "enterprise_ai", CreatedBy: user})
	}
	_, _ = createNotificationRecord(Notification{UserID: assignee, Title: "Investigation assigned", Body: item.Title, Priority: "high", Category: "ai_investigation", SourceApp: "enterprise_ai", SourceEntity: "ai_investigation", SourceEntityID: item.ID, DeepLink: "/ai/investigations/" + item.ID})
	recordTimelineEntry(TimelineEntry{ID: fmt.Sprintf("tl_%d", time.Now().UnixNano()), User: user, Application: "enterprise_ai", Entity: "ai_investigation", EntityID: item.ID, Action: "ai.investigation.created", Description: item.Title, ProjectID: item.ProjectID, OrgID: item.OrganizationID, Metadata: map[string]interface{}{"task_id": task.ID, "correlation_id": item.ID}})
	publishEvent(DomainEvent{ID: fmt.Sprintf("aiev_%d", time.Now().UnixNano()), EventType: "investigation.created", Source: "enterprise_ai", ObjectType: "ai_investigation", ObjectID: item.ID, Actor: user, TenantID: item.TenantID, ProjectID: item.ProjectID, OrgID: item.OrganizationID, Payload: map[string]interface{}{"task_id": task.ID, "recommendation_id": item.RecommendationID}, Timestamp: now, Correlation: item.CorrelationID})
	recordAudit("ai.investigation.create", "enterprise", user, map[string]interface{}{"investigation_id": item.ID, "task_id": task.ID, "source_count": len(evidence), "correlation_id": item.ID})
	c.JSON(201, item)
}

func handleAIInvestigationV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	investigationStore.RLock()
	item, found := investigationStore.items[c.Param("id")]
	investigationStore.RUnlock()
	if !found || item.TenantID != aiTenant(c) || (item.Owner != user && !evaluatePermission(user, "ai", "admin")) {
		c.JSON(404, gin.H{"error": "investigation not found"})
		return
	}
	c.JSON(200, item)
}

func handleAIUpdateInvestigationV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "write")
	if !ok {
		return
	}
	var body struct{ Status, Outcome, DecisionID string }
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid investigation update"})
		return
	}
	investigationStore.Lock()
	item, found := investigationStore.items[c.Param("id")]
	if !found || item.TenantID != aiTenant(c) || (item.Owner != user && !evaluatePermission(user, "ai", "admin")) {
		investigationStore.Unlock()
		c.JSON(404, gin.H{"error": "investigation not found"})
		return
	}
	if body.Status != "" {
		body.Status = investigationStatus(body.Status)
	}
	if body.Status != "" && !validInvestigationStatus(body.Status) {
		investigationStore.Unlock()
		c.JSON(400, gin.H{"error": "invalid investigation status"})
		return
	}
	if body.Status != "" {
		item.Status = body.Status
	}
	if body.Outcome != "" {
		item.Outcome = body.Outcome
	}
	if body.DecisionID != "" {
		item.DecisionID = body.DecisionID
	}
	item.UpdatedAt = nowUTC()
	investigationStore.items[item.ID] = item
	investigationStore.Unlock()
	saveInvestigation(item)
	recordAudit("ai.investigation.update", "enterprise", user, map[string]interface{}{"investigation_id": item.ID, "status": item.Status, "decision_id": item.DecisionID})
	publishEvent(DomainEvent{ID: fmt.Sprintf("aiev_%d", time.Now().UnixNano()), EventType: "investigation.updated", Source: "enterprise_ai", ObjectType: "ai_investigation", ObjectID: item.ID, Actor: user, TenantID: item.TenantID, ProjectID: item.ProjectID, OrgID: item.OrganizationID, Payload: map[string]interface{}{"status": item.Status, "outcome": item.Outcome}, Timestamp: item.UpdatedAt, Correlation: item.CorrelationID})
	c.JSON(200, item)
}

// handleAIInvestigationDecisionV1 is an explicit human action. It creates a
// normal Phase V Decision Record and links it back to this investigation.
func handleAIInvestigationDecisionV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "write")
	if !ok {
		return
	}
	var body struct {
		Decision          string `json:"decision"`
		Reason            string `json:"reason"`
		ActionRequired    string `json:"action_required"`
		ResponsiblePerson string `json:"responsible_person"`
		Deadline          string `json:"deadline"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Decision == "" {
		c.JSON(400, gin.H{"error": "decision is required"})
		return
	}
	investigationStore.RLock()
	item, found := investigationStore.items[c.Param("id")]
	investigationStore.RUnlock()
	if !found || item.TenantID != aiTenant(c) || (item.Owner != user && !evaluatePermission(user, "ai", "admin")) {
		c.JSON(404, gin.H{"error": "investigation not found"})
		return
	}
	decision := createDecisionRecord(DecisionRecord{Decision: body.Decision, Date: nowUTC(), DecisionMaker: user, Context: body.Reason, SupportingEvidence: sourceIDs(item.Evidence), RelatedProject: item.ProjectID, ActionRequired: body.ActionRequired, ResponsiblePerson: body.ResponsiblePerson, Deadline: body.Deadline, Status: "open", CorrelationID: item.CorrelationID, CreatedAt: nowUTC(), UpdatedAt: nowUTC()})
	investigationStore.Lock()
	item.DecisionID, item.Status, item.UpdatedAt = decision.ID, "action_required", nowUTC()
	investigationStore.items[item.ID] = item
	investigationStore.Unlock()
	saveInvestigation(item)
	createKnowledgeRelationship(KnowledgeRelationship{FromType: "ai_investigation", FromID: item.ID, ToType: "decision", ToID: decision.ID, Relation: "resulted_in", Source: "enterprise_ai", CreatedBy: user})
	recordTimelineEntry(TimelineEntry{ID: fmt.Sprintf("tl_%d", time.Now().UnixNano()), User: user, Application: "enterprise_ai", Entity: "ai_investigation", EntityID: item.ID, Action: "investigation.decision_created", Description: decision.Decision, ProjectID: item.ProjectID, OrgID: item.OrganizationID, Metadata: map[string]interface{}{"decision_id": decision.ID, "correlation_id": item.CorrelationID}})
	publishEvent(DomainEvent{ID: fmt.Sprintf("aiev_%d", time.Now().UnixNano()), EventType: "decision.created", Source: "enterprise_ai", ObjectType: "decision", ObjectID: decision.ID, Actor: user, TenantID: item.TenantID, ProjectID: item.ProjectID, OrgID: item.OrganizationID, Payload: map[string]interface{}{"investigation_id": item.ID}, Timestamp: nowUTC(), Correlation: item.CorrelationID})
	recordAudit("ai.investigation.decision_created", "enterprise", user, map[string]interface{}{"investigation_id": item.ID, "decision_id": decision.ID, "correlation_id": item.CorrelationID})
	c.JSON(201, gin.H{"investigation": item, "decision": decision})
}
