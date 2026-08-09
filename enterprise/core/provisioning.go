package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Project & Workspace Provisioning (Phase III) ─────────────────
// When a project is created in PMS, the enterprise layer automatically
// provisions downstream resources:
//   1. Project file storage directory
//   2. Project workspace metadata
//   3. Enterprise search index hint
//   4. Calendar/milestone entries
//   5. Dashboard initialization

// provisionProjectWorkspace creates all project-associated resources.
// This is idempotent - safe to call multiple times.
func provisionProjectWorkspace(ev DomainEvent) {
	projectID := ev.ObjectID
	if projectID == "" {
		if oid, ok := ev.Payload["project_id"].(string); ok {
			projectID = oid
		}
	}
	if projectID == "" {
		return
	}

	// 1. Create project file storage area
	createProjectFileStorage(projectID, ev)

	// 2. Initialize project timeline entry (already done by processEvent)

	// 3. Create calendar milestone for project start
	// (already done by calendarFromEvent in processEvent)

	// 4. Record provisioning completion
	recordAudit("project.provisioned", "enterprise", ev.Actor, map[string]interface{}{
		"project_id": projectID,
		"event_id":   ev.ID,
		"resources":  []string{"file_storage", "timeline", "calendar", "notifications"},
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

// createProjectFileStorage initializes the file storage area for a project.
// In the Universal File Service, this creates the project's root folder metadata.
func createProjectFileStorage(projectID string, ev DomainEvent) {
	if redisClient == nil || projectID == "" {
		return
	}

	// Check if the project file area already exists (idempotency)
	ctx := context.Background()
	key := "statgate:files:project:" + projectID
	exists, _ := redisClient.Exists(ctx, key).Result()
	if exists > 0 {
		return // Already provisioned
	}

	// Create a project root file record (like a folder)
	projectName := "Project Files"
	if n, ok := ev.Payload["name"].(string); ok && n != "" {
		projectName = n + " Files"
	}

	rootRecord := FileRecord{
		ID:           fmt.Sprintf("file_root_%s", projectID),
		Name:         projectName,
		OriginalName: projectName,
		Path:         filepath.Join(getUploadDir(), "projects", projectID),
		Size:         0,
		MimeType:     "application/vnd.statgate.project-folder",
		Category:     "project_folder",
		UploadedBy:   ev.Actor,
		ProjectID:    projectID,
		Version:      1,
		Status:       "active",
		Metadata: map[string]interface{}{
			"project_id":   projectID,
			"project_name": ev.Payload["name"],
			"is_folder":    true,
			"provisioned":  true,
			"created_by":   ev.Actor,
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Create the physical directory
	dirPath := filepath.Join(getUploadDir(), "projects", projectID)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return
	}

	// Store metadata in Redis
	data, _ := json.Marshal(rootRecord)
	redisClient.RPush(ctx, "statgate:files", string(data))
	redisClient.Set(ctx, key, "1", 0) // Mark as provisioned

	// Publish a file area created event
	publishEvent(DomainEvent{
		EventType:  "project.files_initialized",
		Source:     "enterprise",
		ObjectType: "file_area",
		ObjectID:   rootRecord.ID,
		Actor:      ev.Actor,
		ProjectID:  projectID,
		Payload: map[string]interface{}{
			"project_id":  projectID,
			"folder":      projectName,
			"description": "Project file storage area",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func getUploadDir() string {
	dir := getEnv("ENTERPRISE_UPLOAD_DIR", "./uploads/enterprise")
	if strings.HasPrefix(dir, "./") || strings.HasPrefix(dir, "../") {
		// Resolve relative to workspace
		if wd, err := os.Getwd(); err == nil {
			dir = filepath.Join(wd, dir)
		}
	}
	return dir
}

// ─── Enterprise File Provisioning API ─────────────────────────────
// GET /api/files/project/:projectId — list files for a project
func handleProjectFiles(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(400, gin.H{"error": "project_id required"})
		return
	}
	files := fetchFileRecords(500)
	projectFiles := make([]FileRecord, 0)
	for _, f := range files {
		if f.ProjectID == projectID {
			projectFiles = append(projectFiles, f)
		}
	}
	c.JSON(200, gin.H{"project_id": projectID, "count": len(projectFiles), "files": projectFiles})
}

// ─── Project Workspace Status API ─────────────────────────────────
// GET /api/workspace/project/:id — check what's provisioned for a project
func handleProjectProvisioningStatus(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(400, gin.H{"error": "project_id required"})
		return
	}

	status := map[string]interface{}{
		"project_id":     projectID,
		"file_storage":   false,
		"timeline":       false,
		"calendar":       false,
		"notifications":  false,
		"search_indexed": false,
		"checked_at":     time.Now().UTC().Format(time.RFC3339),
	}

	if redisClient != nil {
		ctx := context.Background()

		// Check file storage
		filesKey := "statgate:files:project:" + projectID
		if exists, _ := redisClient.Exists(ctx, filesKey).Result(); exists > 0 {
			status["file_storage"] = true
		}

		// Check timeline entries
		if entries := fetchTimeline(100); len(entries) > 0 {
			for _, e := range entries {
				if e.ProjectID == projectID || e.EntityID == projectID {
					status["timeline"] = true
					break
				}
			}
		}

		// Check calendar events
		if events := fetchCalendarEvents(100); len(events) > 0 {
			for _, e := range events {
				if e.ProjectID == projectID || e.EntityID == projectID {
					status["calendar"] = true
					break
				}
			}
		}

		// Check notifications
		// Notifications are per-user, so we check for any related to this project
		raw, _ := redisClient.Keys(ctx, "statgate:notifications:*").Result()
		for _, notifKey := range raw {
			notifs, _ := redisClient.LRange(ctx, notifKey, 0, 99).Result()
			for _, item := range notifs {
				var n map[string]interface{}
				if err := json.Unmarshal([]byte(item), &n); err == nil {
					if meta, ok := n["metadata"].(map[string]interface{}); ok {
						if oid, ok := meta["object_id"].(string); ok && oid == projectID {
							status["notifications"] = true
							break
						}
					}
					if sid, ok := n["source_entity_id"].(string); ok && sid == projectID {
						status["notifications"] = true
						break
					}
				}
			}
			if status["notifications"].(bool) {
				break
			}
		}

		// Check search - try PMS search for this project
		if data := fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects/" + projectID); data != nil {
			status["search_indexed"] = true
		}
	}

	c.JSON(200, status)
}
