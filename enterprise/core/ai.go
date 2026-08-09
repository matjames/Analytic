package main

import "github.com/gin-gonic/gin"

func handleAICatalog(c *gin.Context) {
	apps := []map[string]interface{}{
		{"app": "pms", "entities": []string{"projects", "surveys", "meetings", "tasks", "approvals", "documents"}},
		{"app": "rms", "entities": []string{"research", "proposals", "ethics", "grants", "datasets", "publications"}},
		{"app": "registry", "entities": []string{"staff", "facilities", "organizations", "regions"}},
		{"app": "statchat", "entities": []string{"messages", "spaces", "members"}},
		{"app": "helpdesk", "entities": []string{"tickets", "categories", "responses"}},
		{"app": "enterprise", "entities": []string{"events", "notifications", "timeline", "files", "calendar", "reports"}},
	}
	c.JSON(200, gin.H{"catalog": apps})
}

func handleAICatalogByApp(c *gin.Context) {
	app := c.Param("app")
	catalog := map[string]interface{}{
		"pms": map[string]interface{}{
			"entities": []string{"projects", "surveys", "meetings", "tasks", "approvals", "documents"},
			"events":   []string{"project.created", "project.closed", "survey.published", "meeting.scheduled", "ticket.raised"},
		},
		"rms": map[string]interface{}{
			"entities": []string{"research", "proposals", "ethics", "grants", "datasets", "publications"},
			"events":   []string{"research.created", "research.approved", "dataset.updated"},
		},
	}
	if data, ok := catalog[app]; ok {
		c.JSON(200, data)
		return
	}
	c.JSON(404, gin.H{"error": "application not in AI catalog"})
}

func handleAISchema(c *gin.Context) {
	schema := map[string]interface{}{
		"version": version,
		"entities": []map[string]interface{}{
			{"name": "event", "fields": []string{"id", "event_type", "source", "object_type", "object_id", "actor", "timestamp", "project_id", "organization_id"}},
			{"name": "notification", "fields": []string{"id", "user_id", "title", "body", "priority", "category", "source_app", "read", "created_at"}},
			{"name": "timeline_entry", "fields": []string{"id", "user", "application", "entity", "entity_id", "action", "description", "timestamp"}},
			{"name": "file", "fields": []string{"id", "name", "mime_type", "category", "size", "uploaded_by", "created_at"}},
			{"name": "calendar_event", "fields": []string{"id", "title", "start", "end", "category", "source_app", "project_id"}},
			{"name": "report", "fields": []string{"id", "title", "type", "format", "status", "created_by", "created_at"}},
		},
		"relationships": []map[string]string{
			{"from": "event", "to": "notification", "via": "source_entity_id"},
			{"from": "event", "to": "timeline_entry", "via": "event_id"},
			{"from": "timeline_entry", "to": "entity", "via": "entity_id"},
		},
	}
	c.JSON(200, schema)
}
