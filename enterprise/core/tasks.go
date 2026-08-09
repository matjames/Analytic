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

// ─── Enterprise Task Management ───────────────────────────────────────
// Phase V: The enterprise task system supports create, assign, reassign,
// prioritize, start, pause, complete, cancel, comment, attach files,
// add deadlines, and add dependencies. Tasks are linked to their
// originating entity (Task → Project → Survey → Facility).

// createTaskRecord stores a new enterprise task.
func createTaskRecord(task EnterpriseTask) {
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	if task.CreatedAt == "" {
		task.CreatedAt = nowUTC()
	}
	task.UpdatedAt = nowUTC()

	workflowMu.Lock()
	enterpriseTasks[task.ID] = task
	workflowMu.Unlock()

	if task.SourceEntity != "" && task.SourceEntityID != "" {
		createKnowledgeRelationship(KnowledgeRelationship{FromType: "task", FromID: task.ID, ToType: task.SourceEntity, ToID: task.SourceEntityID, Relation: "action_for", Source: "enterprise", CreatedBy: task.Assigner})
	} else if task.ProjectID != "" {
		createKnowledgeRelationship(KnowledgeRelationship{FromType: "task", FromID: task.ID, ToType: "project", ToID: task.ProjectID, Relation: "action_for", Source: "enterprise", CreatedBy: task.Assigner})
	}

	// Persist to Redis
	persistEnterpriseTask(task)

	// Record audit
	recordAudit("task.created", "enterprise", task.Assigner, map[string]interface{}{
		"task_id": task.ID, "title": task.Title, "assignee": task.Assignee,
		"project_id": task.ProjectID, "priority": task.Priority,
		"source_entity": task.SourceEntity, "source_entity_id": task.SourceEntityID,
	})

	// Create timeline entry
	recordTimelineEntry(TimelineEntry{
		ID:          fmt.Sprintf("tl_%d", time.Now().UnixNano()),
		User:        task.Assigner,
		Application: "enterprise",
		Entity:      "task",
		EntityID:    task.ID,
		Action:      "task.created",
		Description: fmt.Sprintf("Task created: %s", task.Title),
		ProjectID:   task.ProjectID,
		OrgID:       task.OrgID,
		Metadata: map[string]interface{}{
			"task_id": task.ID, "assignee": task.Assignee, "priority": task.Priority,
			"workflow_id": task.WorkflowID, "instance_id": task.InstanceID,
			"correlation_id": task.CorrelationID,
		},
		Timestamp: nowUTC(),
	})

	// Notify assignee
	if task.Assignee != "" {
		_, _ = createNotificationRecord(Notification{
			UserID:         task.Assignee,
			Title:          "Task Assigned",
			Body:           task.Title,
			Priority:       task.Priority,
			Category:       "task",
			SourceApp:      "enterprise",
			SourceEntity:   "task",
			SourceEntityID: task.ID,
			DeepLink:       "/my-work/tasks/" + task.ID,
			Metadata: map[string]interface{}{
				"task_id": task.ID, "project_id": task.ProjectID, "priority": task.Priority,
				"due_at": task.DueAt, "workflow_id": task.WorkflowID,
			},
		})
	}

	// Publish analytics event
	publishAnalyticsEvent("task.created", map[string]interface{}{
		"task_id": task.ID, "title": task.Title, "assignee": task.Assignee,
		"priority": task.Priority, "project_id": task.ProjectID,
		"source_entity": task.SourceEntity, "source_entity_id": task.SourceEntityID,
		"workflow_id": task.WorkflowID,
	})

	// Publish domain event for downstream consumers
	publishEvent(DomainEvent{
		ID:          fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType:   "task.created",
		Source:      "enterprise",
		ObjectType:  "task",
		ObjectID:    task.ID,
		Actor:       task.Assigner,
		ProjectID:   task.ProjectID,
		OrgID:       task.OrgID,
		Payload:     map[string]interface{}{"title": task.Title, "assignee": task.Assignee, "priority": task.Priority},
		Timestamp:   nowUTC(),
		Correlation: task.CorrelationID,
	})
}

func persistEnterpriseTask(task EnterpriseTask) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(task)
	ctx := context.Background()
	key := fmt.Sprintf("statgate:task:%s", task.ID)
	redisClient.Set(ctx, key, string(data), 30*24*time.Hour)
	redisClient.LPush(ctx, "statgate:tasks", task.ID)
	redisClient.LTrim(ctx, "statgate:tasks", 0, 4999)
}

func getEnterpriseTask(id string) (EnterpriseTask, bool) {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	task, ok := enterpriseTasks[id]
	return task, ok
}

func listEnterpriseTasks(assignee, status, priority string, limit int) []EnterpriseTask {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	out := make([]EnterpriseTask, 0, len(enterpriseTasks))
	for _, task := range enterpriseTasks {
		if assignee != "" && task.Assignee != assignee {
			continue
		}
		if status != "" && task.Status != status {
			continue
		}
		if priority != "" && task.Priority != priority {
			continue
		}
		out = append(out, task)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func updateEnterpriseTask(id string, mutate func(*EnterpriseTask)) (EnterpriseTask, bool) {
	workflowMu.Lock()
	task, ok := enterpriseTasks[id]
	if !ok {
		workflowMu.Unlock()
		return task, false
	}
	mutate(&task)
	task.UpdatedAt = nowUTC()
	enterpriseTasks[id] = task
	workflowMu.Unlock()
	persistEnterpriseTask(task)
	return task, true
}

// checkTaskDependencies determines if a task can be started.
// Returns blocked=true if a required dependency is not yet completed.
func checkTaskDependencies(task EnterpriseTask) ([]string, bool) {
	blockers := []string{}
	if len(task.Dependencies) == 0 {
		return blockers, false
	}
	for _, dep := range task.Dependencies {
		if dep.Type != "blocked_by" && dep.Type != "" {
			continue
		}
		depTask, ok := getEnterpriseTask(dep.TaskID)
		if !ok {
			continue
		}
		if depTask.Status != "completed" {
			blockers = append(blockers, depTask.Title)
		}
	}
	return blockers, len(blockers) > 0
}

// ─── API Handlers ─────────────────────────────────────────────────

func handleListTasks(c *gin.Context) {
	assignee := c.Query("assignee")
	if assignee == "" {
		assignee = c.GetHeader("X-User-ID")
	}
	status := c.Query("status")
	priority := c.Query("priority")
	limit := parseIntDefault(c.Query("limit"), 50)

	tasks := listEnterpriseTasks(assignee, status, priority, limit)
	c.JSON(200, gin.H{"count": len(tasks), "tasks": tasks})
}

func handleGetTask(c *gin.Context) {
	id := c.Param("id")
	task, ok := getEnterpriseTask(id)
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}
	c.JSON(200, task)
}

func handleCreateTask(c *gin.Context) {
	var body struct {
		Title          string                 `json:"title"`
		Description    string                 `json:"description"`
		Priority       string                 `json:"priority"`
		Assignee       string                 `json:"assignee"`
		ProjectID      string                 `json:"project_id"`
		OrgID          string                 `json:"organization_id"`
		FacilityID     string                 `json:"facility_id"`
		Region         string                 `json:"region"`
		District       string                 `json:"district"`
		DueAt          string                 `json:"due_at"`
		SourceApp      string                 `json:"source_app"`
		SourceEntity   string                 `json:"source_entity"`
		SourceEntityID string                 `json:"source_entity_id"`
		Dependencies   []TaskDependency       `json:"dependencies,omitempty"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid task", "detail": err.Error()})
		return
	}

	assigner := c.GetHeader("X-User-ID")
	if assigner == "" {
		assigner = c.Query("actor")
	}

	task := EnterpriseTask{
		ID:             fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Title:          body.Title,
		Description:    body.Description,
		Status:         "pending",
		Priority:       body.Priority,
		Assignee:       body.Assignee,
		Assigner:       assigner,
		SourceApp:      body.SourceApp,
		SourceEntity:   body.SourceEntity,
		SourceEntityID: body.SourceEntityID,
		ProjectID:      body.ProjectID,
		OrgID:          body.OrgID,
		FacilityID:     body.FacilityID,
		Region:         body.Region,
		District:       body.District,
		DueAt:          body.DueAt,
		Dependencies:   body.Dependencies,
		Metadata:       body.Metadata,
		CreatedAt:      nowUTC(),
		UpdatedAt:      nowUTC(),
	}
	if task.Priority == "" {
		task.Priority = "medium"
	}
	if task.Status == "" {
		task.Status = "pending"
	}
	createTaskRecord(task)
	c.JSON(201, task)
}

func handleUpdateTask(c *gin.Context) {
	id := c.Param("id")
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "invalid task update"})
		return
	}

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}

	_, ok := updateEnterpriseTask(id, func(task *EnterpriseTask) {
		if v, ok := body["title"].(string); ok {
			task.Title = v
		}
		if v, ok := body["description"].(string); ok {
			task.Description = v
		}
		if v, ok := body["priority"].(string); ok {
			task.Priority = v
		}
		if v, ok := body["assignee"].(string); ok {
			task.Assignee = v
		}
		if v, ok := body["due_at"].(string); ok {
			task.DueAt = v
		}
		if v, ok := body["status"].(string); ok {
			task.Status = v
		}
	})
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}

	// Record audit
	recordAudit("task.updated", "enterprise", userID, map[string]interface{}{
		"task_id": id, "updates": body,
	})
	c.JSON(200, gin.H{"status": "updated", "task_id": id})
}

func handleTaskAction(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")

	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}

	var task EnterpriseTask
	var ok bool

	switch action {
	case "assign":
		var body struct {
			Assignee string `json:"assignee"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "assignee required"})
			return
		}
		task, ok = updateEnterpriseTask(id, func(t *EnterpriseTask) {
			t.Assignee = body.Assignee
		})
	case "reassign":
		var body struct {
			Assignee string `json:"assignee"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "assignee required"})
			return
		}
		task, ok = updateEnterpriseTask(id, func(t *EnterpriseTask) {
			t.Assignee = body.Assignee
		})
	case "start":
		// Check dependencies first
		current, _ := getEnterpriseTask(id)
		blockers, blocked := checkTaskDependencies(current)
		if blocked {
			c.JSON(409, gin.H{"error": "task is blocked by dependencies", "blockers": blockers})
			return
		}
		task, ok = updateEnterpriseTask(id, func(t *EnterpriseTask) {
			t.Status = "in_progress"
			t.StartedAt = nowUTC()
		})
	case "pause":
		task, ok = updateEnterpriseTask(id, func(t *EnterpriseTask) {
			t.Status = "paused"
			t.PausedAt = nowUTC()
		})
	case "complete":
		task, ok = updateEnterpriseTask(id, func(t *EnterpriseTask) {
			t.Status = "completed"
			t.CompletedAt = nowUTC()
		})
		if ok {
			// Check downstream tasks that this task was blocking (outside the lock)
			checkDownstreamTasks(id)
			onTaskCompleted(task)
		}
	case "cancel":
		task, ok = updateEnterpriseTask(id, func(t *EnterpriseTask) {
			t.Status = "cancelled"
			t.CancelledAt = nowUTC()
		})
	default:
		c.JSON(400, gin.H{"error": "unsupported action", "supported": []string{"assign", "reassign", "start", "pause", "complete", "cancel"}})
		return
	}

	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}

	recordAudit("task."+action, "enterprise", userID, map[string]interface{}{
		"task_id": id, "status": task.Status,
	})
	c.JSON(200, task)
}

func handleTaskComment(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}

	var body struct {
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "comment required"})
		return
	}

	_, ok := updateEnterpriseTask(id, func(t *EnterpriseTask) {
		t.Comments = append(t.Comments, TaskComment{
			User:      userID,
			Comment:   body.Comment,
			Timestamp: nowUTC(),
		})
	})
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}
	c.JSON(200, gin.H{"status": "commented", "task_id": id})
}

func handleTaskDependencies(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Dependencies []TaskDependency `json:"dependencies"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "dependencies required"})
		return
	}

	_, ok := updateEnterpriseTask(id, func(t *EnterpriseTask) {
		t.Dependencies = body.Dependencies
	})
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}
	c.JSON(200, gin.H{"status": "dependencies_updated", "task_id": id})
}

func handleDeleteTask(c *gin.Context) {
	id := c.Param("id")
	workflowMu.Lock()
	if _, ok := enterpriseTasks[id]; ok {
		delete(enterpriseTasks, id)
		workflowMu.Unlock()
		if redisClient != nil {
			ctx := context.Background()
			redisClient.Del(ctx, fmt.Sprintf("statgate:task:%s", id))
		}
		c.JSON(200, gin.H{"status": "deleted", "task_id": id})
		return
	}
	workflowMu.Unlock()
	c.JSON(404, gin.H{"error": "task not found"})
}

// ─── Task Dependency Helpers ─────────────────────────────────────

// checkDownstreamTasks checks if any tasks blocked by the given task
// can now be started.
func checkDownstreamTasks(taskID string) {
	workflowMu.RLock()
	all := make([]EnterpriseTask, 0, len(enterpriseTasks))
	for _, t := range enterpriseTasks {
		all = append(all, t)
	}
	workflowMu.RUnlock()

	for _, task := range all {
		for _, dep := range task.Dependencies {
			if dep.TaskID == taskID && dep.Type != "blocks" {
				blockers, blocked := checkTaskDependencies(task)
				if !blocked && task.Status == "pending" {
					_, _ = updateEnterpriseTask(task.ID, func(t *EnterpriseTask) {
						t.Status = "ready"
					})
					recordAudit("task.ready", "enterprise", "", map[string]interface{}{
						"task_id": task.ID, "title": task.Title,
					})
				}
				_ = blockers
				break
			}
		}
	}
}

// onTaskCompleted resumes a workflow that was waiting for task completion.
func onTaskCompleted(task EnterpriseTask) {
	if task.InstanceID == "" {
		return
	}

	// Find the workflow instance
	instance, ok := getWorkflowInstance(task.InstanceID)
	if !ok {
		return
	}

	// If workflow was waiting for this task, continue it
	if instance.Status == "waiting_task" {
		inst := instance
		inst.Status = "running"
		updateWorkflowInstance(inst)
		go executeWorkflowSteps(instance)
	}

	// Record audit
	recordAudit("task.completed", "enterprise", task.Assignee, map[string]interface{}{
		"task_id": task.ID, "title": task.Title, "workflow_id": task.WorkflowID,
		"instance_id": task.InstanceID, "correlation_id": task.CorrelationID,
	})

	// Publish analytics event
	publishAnalyticsEvent("task.completed", map[string]interface{}{
		"task_id": task.ID, "title": task.Title, "project_id": task.ProjectID,
		"workflow_id": task.WorkflowID, "instance_id": task.InstanceID,
	})

	// Log
	log.Printf("task: task %s completed, resuming workflow %s", task.ID, task.WorkflowID)
}
