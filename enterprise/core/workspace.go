package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Enterprise Workspace API ─────────────────────────────────────
// The Workspace Dashboard is the primary user entry point.
// It answers: "What do I need to know and what do I need to do today?"

// handleWorkspace returns personalized workspace data for the authenticated user.
func handleWorkspace(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}

	// Gather personalized data from all services
	workspace := map[string]interface{}{
		"user_id":          userID,
		"my_tasks":         fetchMyTasks(userID),
		"my_projects":      fetchMyProjects(userID),
		"my_research":      fetchMyResearch(userID),
		"my_surveys":       fetchMySurveys(userID),
		"my_tickets":       fetchMyTickets(userID),
		"my_approvals":     fetchMyApprovals(userID),
		"my_messages":      fetchMyMessages(userID),
		"my_meetings":      fetchMyMeetings(userID),
		"my_notifications": fetchMyNotifications(userID),
		"recent_activity":  fetchMyActivity(userID),
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(200, workspace)
}

// ─── Personalized Data Fetchers ─────────────────────────────────

func fetchMyTasks(userID string) []map[string]interface{} {
	tasks := []map[string]interface{}{}
	// PMS tasks assigned to this user
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/tasks?limit=20"); data != nil {
		for _, t := range data {
			assignedTo := ""
			if v, ok := t["assignedTo"].(string); ok {
				assignedTo = v
			} else if v, ok := t["assigned_to"].(string); ok {
				assignedTo = v
			}
			if assignedTo == userID || assignedTo == "" {
				tasks = append(tasks, map[string]interface{}{
					"id": t["id"], "title": t["title"], "status": t["status"],
					"project_id": t["project_id"], "due": t["end_date"], "source": "pms",
				})
			}
		}
	}
	// RMS tasks
	if data := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/tasks?limit=20"); data != nil {
		for _, t := range data {
			assignedTo := ""
			if v, ok := t["assignedTo"].(string); ok {
				assignedTo = v
			} else if v, ok := t["assigned_to"].(string); ok {
				assignedTo = v
			}
			if assignedTo == userID || assignedTo == "" {
				tasks = append(tasks, map[string]interface{}{
					"id": t["id"], "title": t["title"], "status": t["status"],
					"project_id": t["research_id"], "due": t["end_date"], "source": "rms",
				})
			}
		}
	}
	return tasks
}

func fetchMyProjects(userID string) []map[string]interface{} {
	projects := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); data != nil {
		for _, p := range data {
			owner := ""
			if v, ok := p["owner"].(string); ok {
				owner = v
			}
			if owner == userID || owner == "" {
				projects = append(projects, map[string]interface{}{
					"id": p["id"], "name": p["name"], "code": p["code"],
					"stage": p["stage"], "progress": p["progress"], "source": "pms",
				})
			}
		}
	}
	return projects
}

func fetchMyResearch(userID string) []map[string]interface{} {
	research := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/research"); data != nil {
		for _, r := range data {
			owner := ""
			if v, ok := r["owner"].(string); ok {
				owner = v
			}
			if owner == userID || owner == "" {
				research = append(research, map[string]interface{}{
					"id": r["id"], "name": r["name"], "code": r["code"],
					"stage": r["stage"], "source": "rms",
				})
			}
		}
	}
	return research
}

func fetchMySurveys(userID string) []map[string]interface{} {
	surveys := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/surveys?limit=20"); data != nil {
		for _, s := range data {
			owner := ""
			if v, ok := s["owner"].(string); ok {
				owner = v
			}
			if owner == userID || owner == "" {
				surveys = append(surveys, map[string]interface{}{
					"id": s["id"], "name": s["name"], "status": s["status"], "source": "pms",
				})
			}
		}
	}
	return surveys
}

func fetchMyTickets(userID string) []map[string]interface{} {
	tickets := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/tickets?limit=20"); data != nil {
		for _, t := range data {
			assignee := ""
			if v, ok := t["assignee"].(string); ok {
				assignee = v
			} else if v, ok := t["assigned_to"].(string); ok {
				assignee = v
			}
			if assignee == userID || assignee == "" {
				tickets = append(tickets, map[string]interface{}{
					"id": t["id"], "title": t["title"], "status": t["status"],
					"priority": t["priority"], "source": "helpdesk",
				})
			}
		}
	}
	return tickets
}

func fetchMyApprovals(userID string) []map[string]interface{} {
	approvals := []map[string]interface{}{}
	// PMS projects pending approval
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); data != nil {
		for _, p := range data {
			if stage, ok := p["stage"].(string); ok && stage == "Approval" {
				approvals = append(approvals, map[string]interface{}{
					"id": p["id"], "title": p["name"], "type": "project",
					"status": "pending", "source": "pms",
				})
			}
		}
	}
	// RMS research pending approval
	if data := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/research"); data != nil {
		for _, r := range data {
			if stage, ok := r["stage"].(string); ok {
				if stage == "Proposal" || stage == "Ethics Submission" || stage == "Funding Approval" {
					approvals = append(approvals, map[string]interface{}{
						"id": r["id"], "title": r["name"], "type": "research",
						"status": "pending", "source": "rms",
					})
				}
			}
		}
	}
	return approvals
}

func fetchMyMessages(userID string) []map[string]interface{} {
	messages := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("STATCHAT_API_URL", "http://localhost:4000") + "/v1/chats?limit=10"); data != nil {
		for _, m := range data {
			messages = append(messages, map[string]interface{}{
				"id": m["id"], "sender": m["sender"], "message": m["message"],
				"timestamp": m["created_at"], "source": "statchat",
			})
		}
	}
	return messages
}

func fetchMyMeetings(userID string) []map[string]interface{} {
	meetings := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/calendar?limit=10"); data != nil {
		for _, m := range data {
			meetings = append(meetings, map[string]interface{}{
				"id": m["id"], "title": m["title"], "start": m["start_time"],
				"end": m["end_time"], "source": "pms",
			})
		}
	}
	return meetings
}

func fetchMyNotifications(userID string) []map[string]interface{} {
	notifications := []map[string]interface{}{}
	// From Enterprise Core notifications
	if redisClient != nil {
		key := fmt.Sprintf("statgate:notifications:%s", userID)
		ctx := context.Background()
		raw, _ := redisClient.LRange(ctx, key, 0, 19).Result()
		for _, item := range raw {
			var n Notification
			if err := json.Unmarshal([]byte(item), &n); err == nil {
				notifications = append(notifications, map[string]interface{}{
					"id": n.ID, "title": n.Title, "body": n.Body,
					"priority": n.Priority, "read": n.Read, "deep_link": n.DeepLink,
					"created_at": n.CreatedAt, "source": "enterprise",
				})
			}
		}
	}
	return notifications
}

func fetchMyActivity(userID string) []map[string]interface{} {
	activity := []map[string]interface{}{}
	// From Enterprise Timeline
	if entries := fetchTimeline(50); len(entries) > 0 {
		for _, e := range entries {
			if e.User == userID || e.User == "" {
				activity = append(activity, map[string]interface{}{
					"id": e.ID, "action": e.Action, "description": e.Description,
					"application": e.Application, "entity": e.Entity,
					"entity_id": e.EntityID, "timestamp": e.Timestamp, "source": "enterprise",
				})
			}
		}
	}
	return activity
}
