package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Workflow API Handlers ─────────────────────────────────────────────
// Phase V: HTTP handlers for workflow definitions, instances, and
// execution management.

func handleListWorkflowDefinitions(c *gin.Context) {
	status := c.Query("status")
	defs := listWorkflowDefinitions(status)
	c.JSON(200, gin.H{"count": len(defs), "workflows": defs})
}

func handleCreateWorkflowDefinition(c *gin.Context) {
	var def WorkflowDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(400, gin.H{"error": "invalid workflow definition", "detail": err.Error()})
		return
	}
	if def.ID == "" {
		def.ID = fmt.Sprintf("wf_%d", time.Now().UnixNano())
	}
	if def.Status == "" {
		def.Status = "draft"
	}
	registerWorkflowDefinition(def)
	c.JSON(201, def)
}

func handleGetWorkflowDefinition(c *gin.Context) {
	id := c.Param("id")
	def, ok := getWorkflowDefinition(id)
	if !ok {
		c.JSON(404, gin.H{"error": "workflow definition not found"})
		return
	}
	c.JSON(200, def)
}

func handleUpdateWorkflowDefinition(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(400, gin.H{"error": "invalid workflow update"})
		return
	}

	workflowMu.Lock()
	def, ok := workflowDefinitions[id]
	if !ok {
		workflowMu.Unlock()
		c.JSON(404, gin.H{"error": "workflow definition not found"})
		return
	}
	if v, ok := updates["name"].(string); ok {
		def.Name = v
	}
	if v, ok := updates["description"].(string); ok {
		def.Description = v
	}
	if v, ok := updates["category"].(string); ok {
		def.Category = v
	}
	if v, ok := updates["status"].(string); ok {
		def.Status = v
	}
	def.UpdatedAt = nowUTC()
	workflowDefinitions[id] = def
	workflowMu.Unlock()

	c.JSON(200, def)
}

func handleActivateWorkflow(c *gin.Context) {
	id := c.Param("id")
	workflowMu.Lock()
	def, ok := workflowDefinitions[id]
	if !ok {
		workflowMu.Unlock()
		c.JSON(404, gin.H{"error": "workflow definition not found"})
		return
	}
	def.Status = "active"
	def.UpdatedAt = nowUTC()
	workflowDefinitions[id] = def
	workflowMu.Unlock()

	recordAudit("workflow.activated", "enterprise", "", map[string]interface{}{
		"workflow_id": id, "workflow_name": def.Name,
	})
	c.JSON(200, def)
}

func handleDeactivateWorkflow(c *gin.Context) {
	id := c.Param("id")
	workflowMu.Lock()
	def, ok := workflowDefinitions[id]
	if !ok {
		workflowMu.Unlock()
		c.JSON(404, gin.H{"error": "workflow definition not found"})
		return
	}
	def.Status = "draft"
	def.UpdatedAt = nowUTC()
	workflowDefinitions[id] = def
	workflowMu.Unlock()

	recordAudit("workflow.deactivated", "enterprise", "", map[string]interface{}{
		"workflow_id": id, "workflow_name": def.Name,
	})
	c.JSON(200, def)
}

func handleRunWorkflowManual(c *gin.Context) {
	id := c.Param("id")
	def, ok := getWorkflowDefinition(id)
	if !ok {
		c.JSON(404, gin.H{"error": "workflow definition not found"})
		return
	}

	var body struct {
		EntityType string                 `json:"entity_type"`
		EntityID   string                 `json:"entity_id"`
		ProjectID  string                 `json:"project_id"`
		OrgID      string                 `json:"organization_id"`
		Actor      string                 `json:"actor"`
		Context    map[string]interface{} `json:"context"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid run request", "detail": err.Error()})
		return
	}

	if body.Actor == "" {
		body.Actor = c.GetHeader("X-User-ID")
	}

	instance, err := startWorkflow(def, "manual", id, body.EntityType, body.EntityID, body.ProjectID, body.OrgID, body.Actor, body.Context)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to start workflow", "detail": err.Error()})
		return
	}
	c.JSON(201, gin.H{"status": "started", "instance_id": instance.ID})
}

func handleListWorkflowInstances(c *gin.Context) {
	status := c.Query("status")
	instances := listWorkflowInstances(status)
	c.JSON(200, gin.H{"count": len(instances), "instances": instances})
}

func handleGetWorkflowInstance(c *gin.Context) {
	id := c.Param("id")
	instance, ok := getWorkflowInstance(id)
	if !ok {
		// Try loading from Redis
		if redisClient != nil {
			ctx := context.Background()
			data, err := redisClient.Get(ctx, fmt.Sprintf("statgate:workflow:instance:%s", id)).Result()
			if err == nil {
				var inst WorkflowInstance
				if json.Unmarshal([]byte(data), &inst) == nil {
					c.JSON(200, inst)
					return
				}
			}
		}
		c.JSON(404, gin.H{"error": "workflow instance not found"})
		return
	}
	c.JSON(200, instance)
}

func handleCancelWorkflowInstance(c *gin.Context) {
	id := c.Param("id")
	instance, ok := getWorkflowInstance(id)
	if !ok {
		c.JSON(404, gin.H{"error": "workflow instance not found"})
		return
	}
	updateWorkflowStatus(id, "cancelled", "")

	recordAudit("workflow.cancelled", "enterprise", instance.Assignee, map[string]interface{}{
		"instance_id": id, "workflow_id": instance.WorkflowID,
	})
	c.JSON(200, gin.H{"status": "cancelled", "instance_id": id})
}

func handleRetryWorkflowInstance(c *gin.Context) {
	id := c.Param("id")
	instance, ok := getWorkflowInstance(id)
	if !ok {
		c.JSON(404, gin.H{"error": "workflow instance not found"})
		return
	}

	// Reset to pending and re-execute
	instance.Status = "pending"
	instance.Error = ""
	updateWorkflowInstance(instance)
	go executeWorkflowSteps(instance)

	recordAudit("workflow.retried", "enterprise", instance.Assignee, map[string]interface{}{
		"instance_id": id, "workflow_id": instance.WorkflowID,
	})
	c.JSON(200, gin.H{"status": "retrying", "instance_id": id})
}

// persistWorkflowInstanceData saves workflow data to Redis for recovery.
func persistWorkflowInstanceData(instance WorkflowInstance) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(instance)
	key := fmt.Sprintf("statgate:workflow:instance:%s", instance.ID)
	redisClient.Set(context.Background(), key, string(data), 30*24*time.Hour)
}

// loadWorkflowInstanceFromRedis loads a workflow instance from Redis.
func loadWorkflowInstanceFromRedis(id string) (WorkflowInstance, bool) {
	if redisClient == nil {
		return WorkflowInstance{}, false
	}
	data, err := redisClient.Get(context.Background(), fmt.Sprintf("statgate:workflow:instance:%s", id)).Result()
	if err != nil {
		return WorkflowInstance{}, false
	}
	var inst WorkflowInstance
	if err := json.Unmarshal([]byte(data), &inst); err != nil {
		return WorkflowInstance{}, false
	}
	return inst, true
}
