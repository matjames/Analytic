package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type statCollectLink struct {
	ID              string                 `json:"id"`
	ResearchID      string                 `json:"researchId"`
	SubmissionID    string                 `json:"submissionId"`
	FormID          string                 `json:"formId"`
	Relationship    string                 `json:"relationship"`
	Status          string                 `json:"status"`
	SourceEventType string                 `json:"sourceEventType"`
	TenantID        string                 `json:"tenantId"`
	WorkspaceID     string                 `json:"workspaceId"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedBy       string                 `json:"createdBy"`
	CreatedTime     time.Time              `json:"createdTime"`
	UpdatedTime     time.Time              `json:"updatedTime"`
}

type statCollectLinkRequest struct {
	SubmissionID string                 `json:"submissionId"`
	FormID       string                 `json:"formId"`
	Relationship string                 `json:"relationship"`
	Status       string                 `json:"status"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type statCollectIngestEvent struct {
	EventType    string                 `json:"event_type"`
	SourceModule string                 `json:"source_module"`
	SourceType   string                 `json:"source_type"`
	SourceID     string                 `json:"source_id"`
	TenantID     string                 `json:"tenant_id"`
	WorkspaceID  string                 `json:"workspace_id"`
	ResearchID   string                 `json:"research_id"`
	Payload      map[string]interface{} `json:"payload"`
}

func dbGetStatCollectLinks(c *gin.Context) {
	rows, err := DB.Query(`
		SELECT id, research_id, submission_id, COALESCE(form_id,''), relationship, status,
		       COALESCE(source_event_type,''), tenant_id, workspace_id, metadata,
		       COALESCE(created_by,''), created_time, updated_time
		FROM rms.statcollect_links
		WHERE research_id=$1 AND tenant_id=$2 AND workspace_id=$3
		ORDER BY created_time DESC`, c.Param("id"), tenantIDContext(c), workspaceIDContext(c))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not load StatCollect links"})
		return
	}
	defer rows.Close()
	links := make([]statCollectLink, 0)
	for rows.Next() {
		link, err := scanStatCollectLink(rows)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not read StatCollect links"})
			return
		}
		links = append(links, link)
	}
	c.JSON(http.StatusOK, links)
}

func dbCreateStatCollectLink(c *gin.Context) {
	var request statCollectLinkRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid StatCollect link request"})
		return
	}
	request.SubmissionID = strings.TrimSpace(request.SubmissionID)
	request.FormID = strings.TrimSpace(request.FormID)
	if request.SubmissionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "submissionId is required"})
		return
	}
	if request.Relationship == "" {
		request.Relationship = "collected-submission"
	}
	if request.Status == "" {
		request.Status = "received"
	}
	link, err := saveStatCollectLink(c, c.Param("id"), request.SubmissionID, request.FormID, request.Relationship, request.Status, "user-linked", request.Metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "research study not found in workspace"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not save StatCollect link"})
		return
	}
	c.JSON(http.StatusOK, link)
}

func dbDeleteStatCollectLink(c *gin.Context) {
	result, err := DB.Exec(`DELETE FROM rms.statcollect_links WHERE id=$1 AND research_id=$2 AND tenant_id=$3 AND workspace_id=$4`, c.Param("linkId"), c.Param("id"), tenantIDContext(c), workspaceIDContext(c))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not remove StatCollect link"})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "StatCollect link not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func dbIngestStatCollectEvent(c *gin.Context) {
	if c.GetString("user_role") != "service" {
		c.JSON(http.StatusForbidden, gin.H{"error": "internal service authentication required"})
		return
	}
	var event statCollectIngestEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid StatCollect event"})
		return
	}
	if strings.TrimSpace(event.SourceID) == "" || strings.TrimSpace(event.SourceType) != "submission" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "submission source_id and source_type are required"})
		return
	}
	researchID := statCollectEventValue(event.ResearchID, event.Payload, "research_id", "researchId", "_msh_research_id")
	workspaceID := statCollectEventValue(event.WorkspaceID, event.Payload, "workspace_id", "workspaceId", "_msh_workspace_id")
	if researchID == "" {
		c.JSON(http.StatusAccepted, gin.H{"status": "ignored", "reason": "research_id_not_provided", "submissionId": event.SourceID})
		return
	}
	if workspaceID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "workspace_id is required for an RMS link"})
		return
	}
	if workspaceIDContext(c) != workspaceID {
		c.JSON(http.StatusForbidden, gin.H{"error": "workspace context mismatch"})
		return
	}
	status := eventValue("", event.Payload, "status")
	if status == "" {
		status = "received"
	}
	link, err := saveStatCollectLinkWithContext(researchID, event.SourceID, eventValue("", event.Payload, "form_id", "formId"), "collected-submission", status, event.EventType, event.Payload, tenantIDContext(c), workspaceID, "statcollect-service")
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "research study not found in workspace"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not ingest StatCollect link"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "linked", "link": link})
}

func saveStatCollectLink(c *gin.Context, researchID, submissionID, formID, relationship, status, sourceEventType string, metadata map[string]interface{}) (statCollectLink, error) {
	return saveStatCollectLinkWithContext(researchID, submissionID, formID, relationship, status, sourceEventType, metadata, tenantIDContext(c), workspaceIDContext(c), c.GetString("user_id"))
}

func saveStatCollectLinkWithContext(researchID, submissionID, formID, relationship, status, sourceEventType string, metadata map[string]interface{}, tenantID, workspaceID, createdBy string) (statCollectLink, error) {
	if strings.TrimSpace(researchID) == "" || strings.TrimSpace(submissionID) == "" || strings.TrimSpace(workspaceID) == "" {
		return statCollectLink{}, sql.ErrNoRows
	}
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return statCollectLink{}, err
	}
	var link statCollectLink
	var metadataBytes []byte
	err = DB.QueryRow(`
		INSERT INTO rms.statcollect_links
			(id, research_id, submission_id, form_id, relationship, status, source_event_type, tenant_id, workspace_id, metadata, created_by, updated_time)
		SELECT $1, p.id, $2, NULLIF($3,''), $4, $5, NULLIF($6,''), $7, p.workspace_id, $8, $9, NOW()
		FROM rms.research_projects p
		WHERE p.id=$10 AND p.workspace_id=$11
		ON CONFLICT (research_id, submission_id) DO UPDATE SET
			form_id=EXCLUDED.form_id, relationship=EXCLUDED.relationship, status=EXCLUDED.status,
			source_event_type=EXCLUDED.source_event_type, metadata=EXCLUDED.metadata, updated_time=NOW()
		RETURNING id, research_id, submission_id, COALESCE(form_id,''), relationship, status,
		          COALESCE(source_event_type,''), tenant_id, workspace_id, metadata,
		          COALESCE(created_by,''), created_time, updated_time`,
		newID()+"sc", submissionID, formID, relationship, status, sourceEventType, tenantID, metadataJSON, createdBy, researchID, workspaceID).
		Scan(&link.ID, &link.ResearchID, &link.SubmissionID, &link.FormID, &link.Relationship, &link.Status,
			&link.SourceEventType, &link.TenantID, &link.WorkspaceID, &metadataBytes, &link.CreatedBy, &link.CreatedTime, &link.UpdatedTime)
	if err != nil {
		return statCollectLink{}, err
	}
	_ = json.Unmarshal(metadataBytes, &link.Metadata)
	return link, nil
}

type statCollectLinkScanner interface {
	Scan(dest ...interface{}) error
}

func scanStatCollectLink(scanner statCollectLinkScanner) (statCollectLink, error) {
	var link statCollectLink
	var metadataBytes []byte
	err := scanner.Scan(&link.ID, &link.ResearchID, &link.SubmissionID, &link.FormID, &link.Relationship, &link.Status,
		&link.SourceEventType, &link.TenantID, &link.WorkspaceID, &metadataBytes, &link.CreatedBy, &link.CreatedTime, &link.UpdatedTime)
	if err != nil {
		return link, err
	}
	_ = json.Unmarshal(metadataBytes, &link.Metadata)
	return link, nil
}

func tenantIDContext(c *gin.Context) string {
	if tenantID := strings.TrimSpace(c.GetString("tenant_id")); tenantID != "" {
		return tenantID
	}
	return "default"
}

func statCollectEventValue(primary string, payload map[string]interface{}, keys ...string) string {
	if value := strings.TrimSpace(primary); value != "" {
		return value
	}
	return eventValue("", payload, keys...)
}

func eventValue(primary string, payload map[string]interface{}, keys ...string) string {
	if value := strings.TrimSpace(primary); value != "" {
		return value
	}
	for _, key := range keys {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if nested, ok := payload["meta"].(map[string]interface{}); ok {
		for _, key := range keys {
			if value, ok := nested[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}
