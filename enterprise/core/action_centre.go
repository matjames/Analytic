package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Action Centre & My Work ───────────────────────────────────────────
// Phase V: The Action Centre shows users everything requiring their
// attention. My Work is the consolidated work queue across all
// applications. Together they answer: "What do I need to do now?"

// handleActionCentre returns all action items requiring the user's attention.
func handleActionCentre(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	actionCentre := map[string]interface{}{
		"user_id":          userID,
		"approvals":        userApprovals(userID),
		"tasks":            userTasks(userID),
		"alerts":           userAlerts(userID),
		"escalations":      userEscalations(userID),
		"reviews":          userReviews(userID),
		"data_quality":     userDataQualityIssues(userID),
		"helpdesk":         userHelpDeskActions(userID),
		"project_actions":  userProjectActions(userID),
		"research_actions": userResearchActions(userID),
		"governance_actions": userGovernanceActions(userID),
		"survey_actions":   userSurveyActions(userID),
		"workflows":        userWorkflows(userID),
		"summary": map[string]interface{}{
			"total_items":    0,
			"critical_count": 0,
			"high_count":     0,
			"due_soon":       0,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Build summary
	total := 0
	critical := 0
	high := 0
	dueSoon := 0

	if approvals, ok := actionCentre["approvals"].([]map[string]interface{}); ok {
		total += len(approvals)
	}
	if tasks, ok := actionCentre["tasks"].([]map[string]interface{}); ok {
		total += len(tasks)
		// Count overdue/due soon
		now := time.Now()
		for _, t := range tasks {
			if due, ok := t["due_at"].(string); ok && due != "" {
				if dueTime, err := time.Parse(time.RFC3339, due); err == nil {
					if dueTime.Before(now.Add(24 * time.Hour)) {
						dueSoon++
					}
				}
			}
			if p, ok := t["priority"].(string); ok && p == "critical" {
				critical++
			}
			if p, ok := t["priority"].(string); ok && p == "high" {
				high++
			}
		}
	}
	if alerts, ok := actionCentre["alerts"].([]map[string]interface{}); ok {
		total += len(alerts)
		for _, a := range alerts {
			if p, ok := a["priority"].(string); ok && p == "critical" {
				critical++
			}
			if p, ok := a["priority"].(string); ok && p == "high" {
				high++
			}
		}
	}
	if escalations, ok := actionCentre["escalations"].([]map[string]interface{}); ok {
		total += len(escalations)
	}
	if reviews, ok := actionCentre["reviews"].([]map[string]interface{}); ok {
		total += len(reviews)
	}
	if dataQuality, ok := actionCentre["data_quality"].([]map[string]interface{}); ok {
		total += len(dataQuality)
	}
	if helpdesk, ok := actionCentre["helpdesk"].([]map[string]interface{}); ok {
		total += len(helpdesk)
	}
	if workflows, ok := actionCentre["workflows"].([]map[string]interface{}); ok {
		total += len(workflows)
	}

	summary := actionCentre["summary"].(map[string]interface{})
	summary["total_items"] = total
	summary["critical_count"] = critical
	summary["high_count"] = high
	summary["due_soon"] = dueSoon
	actionCentre["summary"] = summary

	c.JSON(200, actionCentre)
}

// handleMyWork returns the consolidated work queue for a user.
func handleMyWork(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	// Gather from all applications
	workspace := map[string]interface{}{
		"user_id":       userID,
		"tasks":         userTasks(userID),
		"projects":      fetchMyProjects(userID),
		"tickets":       fetchMyTickets(userID),
		"surveys":       fetchMySurveys(userID),
		"research":      fetchMyResearch(userID),
		"approvals":     userApprovals(userID),
		"meetings":      fetchMyMeetings(userID),
		"assignments":   enterpriseAssignments(userID),
		"follow_ups":    userFollowUps(userID),
		"notifications": fetchMyNotifications(userID),
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(200, workspace)
}

// ─── User-Specific Data Fetchers ─────────────────────────────────

func userApprovals(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	requests := listApprovalRequests(userID, "pending")
	for _, req := range requests {
		out = append(out, map[string]interface{}{
			"id":          req.ID,
			"title":       req.Title,
			"description": req.Description,
			"type":        req.Type,
			"status":      req.Status,
			"entity_type": req.EntityType,
			"entity_id":   req.EntityID,
			"project_id":  req.ProjectID,
			"expires_at":  req.ExpiresAt,
			"created_at":  req.CreatedAt,
			"source":      "enterprise",
		})
	}
	// Add external approvals from PMS/RMS
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); data != nil {
		for _, p := range data {
			if stage, ok := p["stage"].(string); ok && stage == "Approval" {
				out = append(out, map[string]interface{}{
					"id": p["id"], "title": p["name"], "type": "project",
					"status": "pending", "source": "pms",
				})
			}
		}
	}
	if data := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/research"); data != nil {
		for _, r := range data {
			if stage, ok := r["stage"].(string); ok {
				if stage == "Proposal" || stage == "Ethics Submission" || stage == "Funding Approval" {
					out = append(out, map[string]interface{}{
						"id": r["id"], "title": r["name"], "type": "research",
						"status": "pending", "source": "rms",
					})
				}
			}
		}
	}
	return out
}

func userTasks(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	tasks := listEnterpriseTasks(userID, "", "", 50)
	for _, task := range tasks {
		out = append(out, map[string]interface{}{
			"id":          task.ID,
			"title":       task.Title,
			"description": task.Description,
			"status":      task.Status,
			"priority":    task.Priority,
			"project_id":  task.ProjectID,
			"due_at":      task.DueAt,
			"source":      "enterprise",
			"workflow_id": task.WorkflowID,
		})
	}
	// PMS tasks
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/tasks?limit=20"); data != nil {
		for _, t := range data {
			out = append(out, map[string]interface{}{
				"id": t["id"], "title": t["title"], "status": t["status"],
				"project_id": t["project_id"], "due": t["end_date"], "source": "pms",
			})
		}
	}
	// RMS tasks
	if data := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/tasks?limit=20"); data != nil {
		for _, t := range data {
			out = append(out, map[string]interface{}{
				"id": t["id"], "title": t["title"], "status": t["status"],
				"project_id": t["research_id"], "due": t["end_date"], "source": "rms",
			})
		}
	}
	return out
}

func userAlerts(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	alertMu.RLock()
	for _, alert := range analyticalAlerts {
		if alert.Status == "resolved" {
			continue
		}
		// Check if the user is associated with the alert scope
		if alert.ScopeID == userID || alert.ScopeID == "" {
			out = append(out, map[string]interface{}{
				"id": alert.ID, "title": alert.Title, "message": alert.Message,
				"priority": alert.Priority, "status": alert.Status, "metric": alert.Metric,
				"value": alert.Value, "threshold": alert.Threshold, "created_at": alert.CreatedAt,
				"deep_link": alert.DeepLink, "source": "analytics",
			})
		}
	}
	alertMu.RUnlock()
	return out
}

func userEscalations(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	instances := listWorkflowInstances("escalated")
	for _, inst := range instances {
		if inst.Assignee == userID || inst.Assignee == "" {
			out = append(out, map[string]interface{}{
				"id": inst.ID, "title": inst.WorkflowName, "status": inst.Status,
				"escalation_level": inst.EscalationLevel, "due_at": inst.DueAt,
				"started_at": inst.StartedAt, "source": "enterprise",
			})
		}
	}
	return out
}

func userReviews(userID string) []map[string]interface{} {
	// Pending reviews from thresholds/approvals
	out := []map[string]interface{}{}
	// Reviews expected from data quality issues
	if dataQualityIssues := userDataQualityIssues(userID); len(dataQualityIssues) > 0 {
		out = append(out, dataQualityIssues...)
	}
	return out
}

func userDataQualityIssues(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("ANALYTICS_API_URL", "http://localhost:5000") + "/api/quality/issues?limit=10"); data != nil {
		for _, issue := range data {
			out = append(out, map[string]interface{}{
				"id":          issue["id"],
				"title":       "Data Quality Issue",
				"type":        issue["type"],
				"severity":    issue["severity"],
				"entity_id":   issue["entity_id"],
				"message":     issue["message"],
				"source":      "analytics",
				"description": fmt.Sprintf("%v", issue["message"]),
			})
		}
	}
	return out
}

func userHelpDeskActions(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/tickets?assignee=" + userID + "&status=open&limit=10"); data != nil {
		for _, ticket := range data {
			out = append(out, map[string]interface{}{
				"id": ticket["id"], "title": ticket["title"], "status": ticket["status"],
				"priority": ticket["priority"], "source": "helpdesk",
				"description": fmt.Sprintf("HelpDesk ticket %v", ticket["id"]),
			})
		}
	}
	return out
}

func userProjectActions(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects?owner=" + userID + "&limit=10"); data != nil {
		for _, project := range data {
			// Identify projects requiring action
			progress := 0.0
			if v, ok := project["progress"].(float64); ok {
				progress = v
			}
			if progress > 0 && progress < 50 {
				out = append(out, map[string]interface{}{
					"id": project["id"], "title": fmt.Sprintf("Project behind schedule: %v", project["name"]),
					"project_id": project["id"], "source": "pms",
					"description": fmt.Sprintf("Project progress is %.1f%%", progress),
					"priority":    "high",
				})
			}
		}
	}
	return out
}

func userResearchActions(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/research?owner=" + userID + "&limit=10"); data != nil {
		for _, research := range data {
			if stage, ok := research["stage"].(string); ok && stage == "Proposal" {
				out = append(out, map[string]interface{}{
					"id": research["id"], "title": fmt.Sprintf("Research requires review: %v", research["name"]),
					"source": "rms", "description": "Research is at Proposal stage",
					"priority": "high",
				})
			}
		}
	}
	return out
}

func userGovernanceActions(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	govURL := getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093")
	if data := fetchJSONArray(govURL + "/api/findings?limit=10"); data != nil {
		for _, finding := range data {
			if status, ok := finding["status"].(string); ok && (status == "Open" || status == "In Remediation") {
				severity, _ := finding["severity"].(string)
				priority := "medium"
				if severity == "Critical" {
					priority = "critical"
				} else if severity == "High" {
					priority = "high"
				}
				out = append(out, map[string]interface{}{
					"id": finding["id"], "title": fmt.Sprintf("Governance Action: %v", finding["title"]),
					"source": "statgovernance", "description": fmt.Sprintf("Audit finding remediation due: %v", finding["due_date"]),
					"priority": priority,
				})
			}
		}
	}
	return out
}

func userSurveyActions(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/surveys?limit=10"); data != nil {
		for _, survey := range data {
			status, _ := survey["status"].(string)
			if status == "Data Collection" {
				out = append(out, map[string]interface{}{
					"id": survey["id"], "title": fmt.Sprintf("Survey data collection in progress: %v", survey["name"]),
					"source": "pms", "description": "Survey is in data collection phase",
					"priority": "medium",
				})
			}
		}
	}
	return out
}

func userWorkflows(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	instances := listWorkflowInstances("")
	for _, inst := range instances {
		if inst.Assignee == userID || inst.Assignee == "" {
			out = append(out, map[string]interface{}{
				"id": inst.ID, "title": inst.WorkflowName, "status": inst.Status,
				"current_step": inst.CurrentStep, "due_at": inst.DueAt,
				"started_at": inst.StartedAt, "source": "enterprise",
			})
		}
	}
	return out
}

func enterpriseAssignments(userID string) []map[string]interface{} {
	// Tasks, approvals and workflows assigned to this user from enterprise
	out := []map[string]interface{}{}
	tasks := listEnterpriseTasks(userID, "", "", 20)
	for _, task := range tasks {
		out = append(out, map[string]interface{}{
			"id": task.ID, "title": task.Title, "type": "task", "status": task.Status,
			"due_at": task.DueAt, "source": "enterprise",
		})
	}
	return out
}

func userFollowUps(userID string) []map[string]interface{} {
	out := []map[string]interface{}{}
	// Overdue items requiring follow-up
	tasks := listEnterpriseTasks(userID, "in_progress", "", 20)
	now := time.Now()
	for _, task := range tasks {
		if task.DueAt != "" {
			if due, err := time.Parse(time.RFC3339, task.DueAt); err == nil {
				if now.After(due) {
					out = append(out, map[string]interface{}{
						"id": task.ID, "title": task.Title, "type": "overdue_task",
						"due_at": task.DueAt, "source": "enterprise",
					})
				}
			}
		}
	}
	// Pending decisions
	decisions := listDecisionRecords("open", userID, 20)
	for _, decision := range decisions {
		out = append(out, map[string]interface{}{
			"id": decision.ID, "title": decision.Decision, "type": "pending_decision",
			"action_required": decision.ActionRequired, "deadline": decision.Deadline,
			"source": "enterprise",
		})
	}
	return out
}

// ─── Workflow Dashboard ────────────────────────────────────────────────
// Provides workflow monitoring for administrators. Managers see only
// workflows within their scope.

func handleWorkflowDashboard(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}

	instances := listWorkflowInstances("")

	// Build dashboard metrics
	dashboard := map[string]interface{}{
		"active_workflows":              0,
		"completed_workflows":           0,
		"failed_workflows":              0,
		"overdue_workflows":             0,
		"escalated_workflows":           0,
		"waiting_approval":              0,
		"waiting_task":                  0,
		"average_completion_time_hours": 0.0,
		"bottlenecks":                   []map[string]interface{}{},
		"failure_rate":                  0.0,
		"instances":                     []map[string]interface{}{},
		"timestamp":                     nowUTC(),
	}

	now := time.Now()
	activeCount := 0
	completedCount := 0
	failedCount := 0
	overdueCount := 0
	escalatedCount := 0
	waitingApproval := 0
	waitingTask := 0
	totalCompletionHours := 0.0
	completionCount := 0

	type stepCount struct {
		ID    string
		Name  string
		Count int
	}
	stepCounts := map[string]*stepCount{}

	for _, inst := range instances {
		// If user is provided, filter to their scope
		if userID != "" && inst.Assignee != "" && inst.Assignee != userID {
			continue
		}

		switch inst.Status {
		case "running", "pending":
			activeCount++
		case "completed":
			completedCount++
			if start, err := time.Parse(time.RFC3339, inst.StartedAt); err == nil {
				totalCompletionHours += time.Since(start).Hours()
				completionCount++
			}
		case "failed":
			failedCount++
		case "escalated":
			escalatedCount++
			activeCount++
		case "waiting_approval":
			waitingApproval++
			activeCount++
		case "waiting_task":
			waitingTask++
			activeCount++
		}

		// Check overdue
		if inst.DueAt != "" && (inst.Status == "running" || inst.Status == "waiting_task" || inst.Status == "waiting_approval") {
			if due, err := time.Parse(time.RFC3339, inst.DueAt); err == nil {
				if now.After(due) {
					overdueCount++
				}
			}
		}

		// Track step counts for bottleneck detection
		stepID := inst.CurrentStep
		if stepID != "" {
			if stepCounts[stepID] == nil {
				stepCounts[stepID] = &stepCount{ID: stepID, Name: stepID}
			}
			stepCounts[stepID].Count++
		}
	}

	// Compute failure rate
	totalExecutions := completedCount + failedCount
	if totalExecutions > 0 {
		dashboard["failure_rate"] = float64(failedCount) / float64(totalExecutions) * 100
	}
	if completionCount > 0 {
		dashboard["average_completion_time_hours"] = totalCompletionHours / float64(completionCount)
	}

	// Detect bottlenecks: steps with the most active instances
	type stepRow struct {
		StepID string `json:"step_id"`
		Name   string `json:"name"`
		Count  int    `json:"count"`
	}
	rows := make([]stepRow, 0, len(stepCounts))
	for _, sc := range stepCounts {
		rows = append(rows, stepRow{StepID: sc.ID, Name: sc.Name, Count: sc.Count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Count > rows[j].Count })
	bottlenecks := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		if row.Count >= 2 {
			bottlenecks = append(bottlenecks, map[string]interface{}{
				"step_id": row.StepID, "name": row.Name, "count": row.Count,
			})
		}
	}
	dashboard["bottlenecks"] = bottlenecks

	dashboard["active_workflows"] = activeCount
	dashboard["completed_workflows"] = completedCount
	dashboard["failed_workflows"] = failedCount
	dashboard["overdue_workflows"] = overdueCount
	dashboard["escalated_workflows"] = escalatedCount
	dashboard["waiting_approval"] = waitingApproval
	dashboard["waiting_task"] = waitingTask

	// Add instance summaries
	instanceSummaries := make([]map[string]interface{}, 0, len(instances))
	for _, inst := range instances {
		if userID != "" && inst.Assignee != "" && inst.Assignee != userID {
			continue
		}
		instanceSummaries = append(instanceSummaries, map[string]interface{}{
			"id": inst.ID, "workflow_id": inst.WorkflowID, "workflow_name": inst.WorkflowName,
			"status": inst.Status, "current_step": inst.CurrentStep,
			"assignee": inst.Assignee, "due_at": inst.DueAt, "started_at": inst.StartedAt,
			"escalation_level": inst.EscalationLevel, "project_id": inst.ProjectID,
		})
	}
	dashboard["instances"] = instanceSummaries

	c.JSON(200, dashboard)
}

// ─── Process Analytics ─────────────────────────────────────────────────
// Measures organizational processes: approval times, resolution times,
// overdue tasks, workflow bottlenecks, etc.

func handleProcessAnalytics(c *gin.Context) {
	instances := listWorkflowInstances("")
	approvals := listApprovalRequests("", "")
	tasks := listEnterpriseTasks("", "", "", 100)

	process := map[string]interface{}{
		"processes":    []map[string]interface{}{},
		"generated_at": nowUTC(),
	}

	metrics := []map[string]interface{}{}

	// Workflow process metrics
	now := time.Now()

	// Average workflow completion time
	completedInstances := 0
	totalDuration := 0.0
	for _, inst := range instances {
		if inst.Status == "completed" && inst.CompletedAt != "" {
			if start, err := time.Parse(time.RFC3339, inst.StartedAt); err == nil {
				if end, err := time.Parse(time.RFC3339, inst.CompletedAt); err == nil {
					totalDuration += end.Sub(start).Hours()
					completedInstances++
				}
			}
		}
	}
	avgWorkflow := 0.0
	if completedInstances > 0 {
		avgWorkflow = totalDuration / float64(completedInstances)
	}
	metrics = append(metrics, map[string]interface{}{
		"category": "workflow", "name": "Average Workflow Completion Time",
		"average_hours": avgWorkflow, "count": completedInstances,
		"last_updated": nowUTC(),
	})

	// Overdue workflow count
	overdueCount := 0
	for _, inst := range instances {
		if inst.DueAt != "" && (inst.Status == "running" || inst.Status == "waiting_task" || inst.Status == "waiting_approval") {
			if due, err := time.Parse(time.RFC3339, inst.DueAt); err == nil {
				if now.After(due) {
					overdueCount++
				}
			}
		}
	}
	metrics = append(metrics, map[string]interface{}{
		"category": "workflow", "name": "Overdue Workflows",
		"count": overdueCount, "last_updated": nowUTC(),
	})

	// Approval process metrics
	approvedCount := 0
	rejectedCount := 0
	pendingCount := 0
	totalApprovalDuration := 0.0
	for _, req := range approvals {
		switch req.Status {
		case "approved":
			approvedCount++
			if created, err := time.Parse(time.RFC3339, req.CreatedAt); err == nil {
				if updated, err := time.Parse(time.RFC3339, req.UpdatedAt); err == nil {
					totalApprovalDuration += updated.Sub(created).Hours()
				}
			}
		case "rejected":
			rejectedCount++
		case "pending", "escalated", "expired":
			pendingCount++
		}
	}
	avgApproval := 0.0
	if approvedCount > 0 {
		avgApproval = totalApprovalDuration / float64(approvedCount)
	}
	metrics = append(metrics, map[string]interface{}{
		"category": "approval", "name": "Average Approval Time",
		"average_hours": avgApproval, "count": approvedCount, "last_updated": nowUTC(),
	})
	metrics = append(metrics, map[string]interface{}{
		"category": "approval", "name": "Pending Approvals",
		"count": pendingCount, "last_updated": nowUTC(),
	})

	// Task process metrics
	overdueTasks := 0
	completedTasks := 0
	for _, task := range tasks {
		if task.Status == "completed" {
			completedTasks++
		}
		if task.DueAt != "" && task.Status != "completed" && task.Status != "cancelled" {
			if due, err := time.Parse(time.RFC3339, task.DueAt); err == nil {
				if now.After(due) {
					overdueTasks++
				}
			}
		}
	}
	metrics = append(metrics, map[string]interface{}{
		"category": "task", "name": "Completed Tasks",
		"count": completedTasks, "last_updated": nowUTC(),
	})
	metrics = append(metrics, map[string]interface{}{
		"category": "task", "name": "Overdue Tasks",
		"count": overdueTasks, "last_updated": nowUTC(),
	})

	// Identify bottlenecks
	stepCounts := map[string]int{}
	for _, inst := range instances {
		if inst.Status == "running" || inst.Status == "waiting_task" || inst.Status == "waiting_approval" {
			if inst.CurrentStep != "" {
				stepCounts[inst.CurrentStep]++
			}
		}
	}
	for step, count := range stepCounts {
		if count >= 2 {
			metrics = append(metrics, map[string]interface{}{
				"category": "bottleneck", "name": fmt.Sprintf("Bottleneck at Step %s", step),
				"count": count, "bottleneck": true, "last_updated": nowUTC(),
			})
		}
	}

	process["processes"] = metrics
	c.JSON(200, process)
}

// ─── Workflow Audit Trail Query ───────────────────────────────────────

func handleWorkflowExecutions(c *gin.Context) {
	instanceID := c.Query("instance_id")
	workflowID := c.Query("workflow_id")
	status := c.Query("status")
	limit := parseIntDefault(c.Query("limit"), 100)

	workflowMu.RLock()
	out := make([]WorkflowExecution, 0, len(workflowExecutions))
	for _, exec := range workflowExecutions {
		if instanceID != "" && exec.InstanceID != instanceID {
			continue
		}
		if workflowID != "" && exec.WorkflowID != workflowID {
			continue
		}
		if status != "" && exec.Status != status {
			continue
		}
		out = append(out, exec)
	}
	workflowMu.RUnlock()

	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp > out[j].Timestamp })
	if len(out) > limit {
		out = out[:limit]
	}
	c.JSON(200, gin.H{"count": len(out), "executions": out})
}

// ─── Stats (for tests) ─────────────────────────────────────────────

func countWorkflowDefinitions() int {
	return len(listWorkflowDefinitions(""))
}

func countApprovalRequests() int {
	return len(listApprovalRequests("", ""))
}

func countEnterpriseTasks() int {
	workflowMu.RLock()
	defer workflowMu.RUnlock()
	return len(enterpriseTasks)
}
