package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// processEvent derives timeline, notifications, calendar, analytics,
// workflows, and fabric updates from domain events.
func processEvent(ev DomainEvent) {
	recordTimelineFromEvent(ev)
	// Phase VI: events keep the enterprise knowledge graph synchronized
	processKnowledgeForEvent(ev)

	// Phase IX: events update canonical objects, catalogue freshness & fabric
	processFabricForEvent(ev)

	// ── Phase IV: Feed the enterprise data layer & analytics ──
	processAnalyticsForEvent(ev)

	// ── Phase V: Trigger enterprise workflows ──
	processWorkflowForEvent(ev)

	// ── Phase XII: Institutional intelligence signal & condition derivation ──
	processIntelligenceForEvent(ev)


	switch ev.EventType {
	case "user.created", "user.updated":
		notifyFromEvent(ev, "system", "User Account Update", "A user account was updated.", "medium")
	case "project.created":
		notifyFromEvent(ev, "project", "New Project", fmt.Sprintf("Project %s has been created.", ev.Payload["name"]), "high")
		calendarFromEvent(ev, "milestone", "Project Created")
		// Auto-provision project file storage, workspace, and downstream resources
		provisionProjectWorkspace(ev)
	case "project.closed":
		notifyFromEvent(ev, "project", "Project Closed", fmt.Sprintf("Project %s has been closed.", ev.Payload["name"]), "high")
	case "project.milestone":
		calendarFromEvent(ev, "milestone", "Project Milestone")
	case "survey.published":
		notifyFromEvent(ev, "survey", "Survey Published", fmt.Sprintf("Survey %s has been published.", ev.Payload["name"]), "high")
		calendarFromEvent(ev, "milestone", "Survey Published")
	case "survey.submitted":
		notifyFromEvent(ev, "survey", "Survey Submitted", fmt.Sprintf("Survey submission received for %s.", ev.Payload["name"]), "medium")
	case "survey.created":
		notifyFromEvent(ev, "survey", "Survey Created", fmt.Sprintf("Survey %s has been created.", ev.Payload["name"]), "medium")
	case "dataset.created":
		notifyFromEvent(ev, "system", "Dataset Created", fmt.Sprintf("Dataset %s has been created.", ev.Payload["name"]), "medium")
	case "dataset.updated":
		notifyFromEvent(ev, "system", "Dataset Updated", fmt.Sprintf("Dataset %s has been updated.", ev.Payload["name"]), "medium")
	case "submission.received":
		notifyFromEvent(ev, "survey", "Field Data Submitted", fmt.Sprintf("Field data submitted for %s.", ev.Payload["name"]), "medium")
	case "submission.approved":
		notifyFromEvent(ev, "survey", "Submission Approved", fmt.Sprintf("Submission for %s has been approved.", ev.Payload["name"]), "medium")
	case "ticket.assigned":
		notifyFromEvent(ev, "helpdesk", "Ticket Assigned", fmt.Sprintf("HelpDesk ticket %s has been assigned.", ev.ObjectID), "high")
	case "ticket.updated":
		notifyFromEvent(ev, "helpdesk", "Ticket Updated", fmt.Sprintf("HelpDesk ticket %s has been updated.", ev.ObjectID), "medium")
	case "research.created":
		notifyFromEvent(ev, "research", "Research Created", fmt.Sprintf("Research study %s has been created.", ev.Payload["name"]), "medium")
	case "research.stage_changed":
		notifyFromEvent(ev, "research", "Research Stage Changed", fmt.Sprintf("Research %s moved to %s.", ev.Payload["name"], ev.Payload["to_stage"]), "medium")
	case "field_data.submitted":
		notifyFromEvent(ev, "system", "Field Data Submitted", "Field data has been submitted.", "medium")
		calendarFromEvent(ev, "milestone", "Field Data Submitted")
	case "research.approved":
		notifyFromEvent(ev, "research", "Research Approved", fmt.Sprintf("Research %s has been approved.", ev.Payload["name"]), "high")
	case "meeting.scheduled":
		notifyFromEvent(ev, "meeting", "Meeting Scheduled", fmt.Sprintf("Meeting %s has been scheduled.", ev.Payload["title"]), "medium")
		calendarFromEvent(ev, "meeting", "Meeting")
	case "ticket.raised":
		notifyFromEvent(ev, "helpdesk", "Ticket Raised", fmt.Sprintf("HelpDesk ticket %s has been raised.", ev.Payload["title"]), "high")
		calendarFromEvent(ev, "sla", "Ticket SLA")
	case "ticket.closed":
		notifyFromEvent(ev, "helpdesk", "Ticket Closed", fmt.Sprintf("HelpDesk ticket %s has been closed.", ev.ObjectID), "medium")
	case "report.generated":
		notifyFromEvent(ev, "report", "Report Generated", fmt.Sprintf("Report %s has been generated.", ev.Payload["title"]), "medium")
	case "dashboard.refreshed":
		notifyFromEvent(ev, "system", "Dashboard Refreshed", "A dashboard has been refreshed.", "low")
	}
}

func notifyFromEvent(ev DomainEvent, category, title, body, priority string) {
	recipient := ""
	if ev.Actor != "" {
		recipient = ev.Actor
	} else if u, ok := ev.Payload["owner"].(string); ok {
		recipient = u
	} else if u, ok := ev.Payload["user"].(string); ok {
		recipient = u
	}
	if recipient == "" {
		return
	}
	_, _ = createNotificationRecord(Notification{
		UserID:         recipient,
		Title:          title,
		Body:           body,
		Priority:       priority,
		Category:       category,
		SourceApp:      ev.Source,
		SourceEntity:   ev.ObjectType,
		SourceEntityID: ev.ObjectID,
		DeepLink:       buildDeepLink(ev),
		Metadata: map[string]interface{}{
			"event_id": ev.ID, "event_type": ev.EventType, "object_id": ev.ObjectID,
		},
	})
}

func buildDeepLink(ev DomainEvent) string {
	appURLs := map[string]string{
		"pms":       getEnv("PMS_UI_URL", "http://localhost:3010"),
		"rms":       getEnv("RMS_UI_URL", "http://localhost:3011"),
		"registry":  getEnv("REGISTRY_UI_URL", "http://localhost:3007"),
		"statchat":  getEnv("STATCHAT_UI_URL", "http://localhost:3009"),
		"helpdesk":  getEnv("HELPDESK_UI_URL", "http://localhost:3005"),
		"analytics": getEnv("ANALYTICS_UI_URL", "http://localhost:5000"),
	}
	base, ok := appURLs[ev.Source]
	if !ok {
		return ""
	}
	switch ev.Source {
	case "pms":
		switch ev.ObjectType {
		case "project":
			return base + "/projects/" + ev.ObjectID
		case "survey":
			return base + "/surveys/" + ev.ObjectID
		case "meeting":
			return base + "/meetings/" + ev.ObjectID
		}
		return base + "/projects"
	case "rms":
		switch ev.ObjectType {
		case "research":
			return base + "/research/" + ev.ObjectID
		case "dataset":
			return base + "/datasets/" + ev.ObjectID
		}
		return base + "/research"
	case "helpdesk":
		return base + "/tickets/" + ev.ObjectID
	case "statchat":
		return base + "/chats" + ev.ObjectID
	}
	return base
}

// recordTimelineFromEvent adds a timeline entry from the domain event.
func recordTimelineFromEvent(ev DomainEvent) {
	description := ev.EventType
	if d, ok := ev.Payload["description"].(string); ok && d != "" {
		description = d
	} else if n, ok := ev.Payload["name"].(string); ok {
		description = fmt.Sprintf("%s: %s", ev.EventType, n)
	}
	entry := TimelineEntry{
		ID:          fmt.Sprintf("tl_%d", time.Now().UnixNano()),
		User:        ev.Actor,
		Application: ev.Source,
		Entity:      ev.ObjectType,
		EntityID:    ev.ObjectID,
		Action:      ev.EventType,
		Description: description,
		ProjectID:   ev.ProjectID,
		OrgID:       ev.OrgID,
		Metadata:    ev.Payload,
		Timestamp:   ev.Timestamp,
	}
	if redisClient != nil {
		data, _ := json.Marshal(entry)
		ctx := context.Background()
		redisClient.LPush(ctx, "statgate:timeline", string(data))
		redisClient.LTrim(ctx, "statgate:timeline", 0, 4999)
	}
}

func processFabricForEvent(ev DomainEvent) {
	if ev.ObjectType == "" || ev.ObjectID == "" {
		return
	}

	canonicalID := fmt.Sprintf("%s:%s:%s:%s", ev.TenantID, ev.Source, ev.ObjectType, ev.ObjectID)
	if ev.TenantID == "" {
		canonicalID = fmt.Sprintf("tenant_uganda_inst:%s:%s:%s", ev.Source, ev.ObjectType, ev.ObjectID)
	}

	title := fmt.Sprintf("%s %s", ev.ObjectType, ev.ObjectID)
	if t, ok := ev.Payload["name"].(string); ok && t != "" {
		title = t
	} else if t, ok := ev.Payload["title"].(string); ok && t != "" {
		title = t
	}

	fabricStore.Lock()
	fabricStore.objects[canonicalID] = CanonicalObject{
		CanonicalID:       canonicalID,
		ObjectID:          ev.ObjectID,
		ObjectType:        ev.ObjectType,
		SourceApplication: ev.Source,
		TenantID:          "tenant_uganda_inst",
		OrganizationID:    "MOH_UG",
		ProjectID:         ev.ProjectID,
		Title:             title,
		CreatedBy:         ev.Actor,
		CreatedAt:         nowUTC(),
		UpdatedAt:         nowUTC(),
		Version:           1,
		Status:            "active",
		Classification:    "verified",
		Sensitivity:       "official",
		CanonicalURL:      buildDeepLink(ev),
	}
	fabricStore.Unlock()

	// Update catalogue freshness if dataset
	if ev.ObjectType == "dataset" {
		fabricStore.Lock()
		if ds, ok := fabricStore.catalogue[ev.ObjectID]; ok {
			ds.LastRefresh = nowUTC()
			ds.FreshnessStatus = "live"
			fabricStore.catalogue[ev.ObjectID] = ds
		}
		fabricStore.Unlock()
	}
}

