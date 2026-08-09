package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Decision Records ──────────────────────────────────────────────────
// Phase V: Important organizational decisions are recorded as decision
// records containing the decision, date, decision maker, context,
// supporting evidence, related project/data/analytics, action required,
// responsible person, deadline and outcome. This creates institutional
// memory and is fully auditable.

// createDecisionRecord stores a new decision record.
func createDecisionRecord(decision DecisionRecord) DecisionRecord {
	if decision.ID == "" {
		decision.ID = fmt.Sprintf("dec_%d", time.Now().UnixNano())
	}
	if decision.CreatedAt == "" {
		decision.CreatedAt = nowUTC()
	}
	decision.UpdatedAt = nowUTC()
	if decision.Status == "" {
		decision.Status = "open"
	}

	workflowMu.Lock()
	decisionRecords[decision.ID] = decision
	workflowMu.Unlock()

	// Preserve the decision's supporting project/data relationships in the
	// shared knowledge graph; source systems remain authoritative.
	if decision.RelatedProject != "" {
		createKnowledgeRelationship(KnowledgeRelationship{FromType: "decision", FromID: decision.ID, ToType: "project", ToID: decision.RelatedProject, Relation: "decides_for", Source: "enterprise", CreatedBy: decision.DecisionMaker})
	}

	// Persist to Redis
	persistDecisionRecord(decision)

	// Record audit
	recordAudit("decision.created", "enterprise", decision.DecisionMaker, map[string]interface{}{
		"decision_id": decision.ID, "decision": decision.Decision,
		"related_project": decision.RelatedProject, "responsible_person": decision.ResponsiblePerson,
		"workflow_id": decision.WorkflowID, "instance_id": decision.InstanceID,
		"correlation_id": decision.CorrelationID,
	})

	// Create timeline entry
	recordTimelineEntry(TimelineEntry{
		ID:          fmt.Sprintf("tl_%d", time.Now().UnixNano()),
		User:        decision.DecisionMaker,
		Application: "enterprise",
		Entity:      "decision",
		EntityID:    decision.ID,
		Action:      "decision.created",
		Description: fmt.Sprintf("Decision made: %s", decision.Decision),
		ProjectID:   decision.RelatedProject,
		Metadata: map[string]interface{}{
			"decision_id": decision.ID, "action_required": decision.ActionRequired,
			"responsible_person": decision.ResponsiblePerson, "deadline": decision.Deadline,
			"workflow_id": decision.WorkflowID, "instance_id": decision.InstanceID,
		},
		Timestamp: nowUTC(),
	})

	// Publish analytics event
	publishAnalyticsEvent("decision.created", map[string]interface{}{
		"decision_id": decision.ID, "decision": decision.Decision,
		"decision_maker": decision.DecisionMaker, "related_project": decision.RelatedProject,
		"action_required": decision.ActionRequired, "responsible_person": decision.ResponsiblePerson,
		"workflow_id": decision.WorkflowID, "instance_id": decision.InstanceID,
	})
	return decision
}

func persistDecisionRecord(decision DecisionRecord) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(decision)
	ctx := context.Background()
	key := fmt.Sprintf("statgate:decision:%s", decision.ID)
	redisClient.Set(ctx, key, string(data), 365*24*time.Hour)
	redisClient.LPush(ctx, "statgate:decisions", decision.ID)
	redisClient.LTrim(ctx, "statgate:decisions", 0, 999)
}

func getDecisionRecord(id string) (DecisionRecord, bool) {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	decision, ok := decisionRecords[id]
	return decision, ok
}

func listDecisionRecords(status, responsiblePerson string, limit int) []DecisionRecord {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]DecisionRecord, 0, len(decisionRecords))
	for _, decision := range decisionRecords {
		if status != "" && decision.Status != status {
			continue
		}
		if responsiblePerson != "" && decision.ResponsiblePerson != responsiblePerson {
			continue
		}
		out = append(out, decision)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func updateDecisionRecord(id string, mutate func(*DecisionRecord)) (DecisionRecord, bool) {
	workflowMu.Lock()
	decision, ok := decisionRecords[id]
	if !ok {
		workflowMu.Unlock()
		return decision, false
	}
	mutate(&decision)
	decision.UpdatedAt = nowUTC()
	decisionRecords[id] = decision
	workflowMu.Unlock()
	persistDecisionRecord(decision)
	return decision, true
}

// ─── AI Recommendations ─────────────────────────────────────────────────
// AI recommendations are always distinguishable from confirmed
// organizational decisions. They have a "recommendation" status and
// require human confirmation before becoming decisions.

func createAIRecommendation(rec AIRecommendation) {
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("air_%d", time.Now().UnixNano())
	}
	if rec.CreatedAt == "" {
		rec.CreatedAt = nowUTC()
	}
	if rec.Status == "" {
		rec.Status = "pending"
	}

	workflowMu.Lock()
	aiRecommendations[rec.ID] = rec
	workflowMu.Unlock()

	// Record audit
	recordAudit("ai.recommendation", "enterprise", "", map[string]interface{}{
		"recommendation_id": rec.ID, "type": rec.Type, "title": rec.Title,
		"confidence": rec.Confidence, "related_entity": rec.RelatedEntity,
		"related_entity_id": rec.RelatedEntityID, "project_id": rec.ProjectID,
	})

	// Publish analytics event
	publishAnalyticsEvent("ai.recommendation", map[string]interface{}{
		"recommendation_id": rec.ID, "type": rec.Type, "title": rec.Title,
		"confidence": rec.Confidence, "status": rec.Status,
		"related_entity": rec.RelatedEntity, "related_entity_id": rec.RelatedEntityID,
	})
}

func listAIRecommendations(status string) []AIRecommendation {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]AIRecommendation, 0, len(aiRecommendations))
	for _, rec := range aiRecommendations {
		if status != "" && rec.Status != status {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out
}

func updateAIRecommendation(id string, mutate func(*AIRecommendation)) (AIRecommendation, bool) {
	workflowMu.Lock()
	rec, ok := aiRecommendations[id]
	if !ok {
		workflowMu.Unlock()
		return rec, false
	}
	mutate(&rec)
	if rec.Status != "pending" && rec.ResolvedAt == "" {
		rec.ResolvedAt = nowUTC()
	}
	aiRecommendations[id] = rec
	workflowMu.Unlock()
	return rec, true
}

// ─── API Handlers ─────────────────────────────────────────────────

func handleListDecisions(c *gin.Context) {
	status := c.Query("status")
	responsible := c.Query("responsible_person")
	limit := parseIntDefault(c.Query("limit"), 50)
	decisions := listDecisionRecords(status, responsible, limit)
	c.JSON(200, gin.H{"count": len(decisions), "decisions": decisions})
}

func handleGetDecision(c *gin.Context) {
	id := c.Param("id")
	decision, ok := getDecisionRecord(id)
	if !ok {
		c.JSON(404, gin.H{"error": "decision not found"})
		return
	}
	c.JSON(200, decision)
}

func handleCreateDecision(c *gin.Context) {
	var decision DecisionRecord
	if err := c.ShouldBindJSON(&decision); err != nil {
		c.JSON(400, gin.H{"error": "invalid decision record", "detail": err.Error()})
		return
	}
	decision = createDecisionRecord(decision)
	c.JSON(201, decision)
}

func handleUpdateDecision(c *gin.Context) {
	id := c.Param("id")
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid decision update"})
		return
	}

	_, ok := updateDecisionRecord(id, func(d *DecisionRecord) {
		if v, ok := body["status"].(string); ok {
			d.Status = v
		}
		if v, ok := body["outcome"].(string); ok {
			d.Outcome = v
		}
		if v, ok := body["action_required"].(string); ok {
			d.ActionRequired = v
		}
		if v, ok := body["responsible_person"].(string); ok {
			d.ResponsiblePerson = v
		}
		if v, ok := body["deadline"].(string); ok {
			d.Deadline = v
		}
	})
	if !ok {
		c.JSON(404, gin.H{"error": "decision not found"})
		return
	}
	c.JSON(200, gin.H{"status": "updated", "decision_id": id})
}

func handleListAIRecommendations(c *gin.Context) {
	status := c.Query("status")
	recommendations := listAIRecommendations(status)
	c.JSON(200, gin.H{"count": len(recommendations), "recommendations": recommendations})
}

func handleAIRecommendationAction(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")

	var body struct {
		Decision string `json:"decision"`
	}
	_ = c.ShouldBindJSON(&body)

	switch action {
	case "accept":
		rec, ok := updateAIRecommendation(id, func(r *AIRecommendation) {
			r.Status = "accepted"
		})
		if !ok {
			c.JSON(404, gin.H{"error": "recommendation not found"})
			return
		}
		c.JSON(200, rec)
	case "dismiss":
		rec, ok := updateAIRecommendation(id, func(r *AIRecommendation) {
			r.Status = "dismissed"
		})
		if !ok {
			c.JSON(404, gin.H{"error": "recommendation not found"})
			return
		}
		c.JSON(200, rec)
	case "convert":
		// Convert an accepted recommendation into a confirmed decision
		rec, ok := getAIRecommendation(id)
		if !ok {
			c.JSON(404, gin.H{"error": "recommendation not found"})
			return
		}
		decision := DecisionRecord{
			ID:                 fmt.Sprintf("dec_%d", time.Now().UnixNano()),
			Decision:           body.Decision,
			Date:               nowUTC(),
			DecisionMaker:      rec.RelatedEntityID,
			Context:            rec.Description,
			SupportingEvidence: rec.Evidence,
			RelatedProject:     rec.ProjectID,
			Status:             "open",
			CreatedAt:          nowUTC(),
			UpdatedAt:          nowUTC(),
		}
		createDecisionRecord(decision)
		_, _ = updateAIRecommendation(id, func(r *AIRecommendation) {
			r.Status = "converted_to_decision"
		})
		c.JSON(201, decision)
	default:
		c.JSON(400, gin.H{"error": "unsupported action", "supported": []string{"accept", "dismiss", "convert"}})
	}
}

func getAIRecommendation(id string) (AIRecommendation, bool) {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	rec, ok := aiRecommendations[id]
	return rec, ok
}
