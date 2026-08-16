package api

import (
	"context"
	"log"

	"aiengines/pkg/model"
	"aiengines/pkg/store"

	"github.com/matjames/statgate-lib/events"
)

// EventTrigger maps an inbound platform domain event to an agent task spec.
// Being exported keeps the mapping testable and lets integration tooling reuse
// it. The store-independent contract is: (name, taskType, role, makeTwin).
func EventTrigger(eventType string) (name, taskType, role string, createTwin bool) {
	switch eventType {
	case events.EventDatasetPublished:
		return "Analyze published dataset", "dataset_analysis", "analyst", false
	case events.EventSurveyPublished:
		return "Analyze published survey", "survey_analysis", "analyst", false
	case events.EventSubmissionReceived:
		return "Triage field submission", "submission_triage", "auditor", false
	case events.EventProjectCreated:
		return "Assess project risk", "project_risk", "analyst", true
	case events.EventResearchPublished:
		return "Summarize research output", "research_summary", "researcher", false
	case events.EventContentPublished:
		return "Curate published knowledge", "knowledge_curation", "researcher", false
	case events.EventPredictionGenerated:
		return "Evaluate generated prediction", "prediction_review", "auditor", false
	}
	return "", "", "", false
}

// HandleDomainEvent is the mandatory system hook: it reacts to platform domain
// events on `statgate:events` by (1) dispatching an agent task and (2) for
// project creation events, spawning a digital twin. Events are emitted back to
// the bus so the rest of the platform observes the agent activity.
func HandleDomainEvent(ctx context.Context, evt events.EnterpriseEvent) error {
	name, taskType, role, createTwin := EventTrigger(evt.EventType)
	if taskType == "" {
		return nil // not a domain event this app reacts to
	}
	tenantID := evt.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	// Twin registration for project events (idempotent).
	if createTwin {
		if err := createTwinForEvent(ctx, tenantID, evt); err != nil {
			log.Printf("[%s] twin creation for event %s failed: %v", SourceApplication, evt.EventType, err)
		}
	}

	// Find an active agent for the role; skip if none is provisioned yet.
	agent, err := store.FindActiveAgentByRole(ctx, tenantID, role)
	if err != nil || agent == nil {
		log.Printf("[%s] no active %s agent for event %s; skipping task", SourceApplication, role, evt.EventType)
		return nil
	}

	task := model.AgentTask{
		TenantID:    tenantID,
		AgentID:     agent.ID,
		Name:        name,
		TaskType:    taskType,
		Payload:     map[string]interface{}{"source_event": evt.EventType, "object_type": evt.ObjectType, "object_id": evt.ObjectID},
		TriggerType: "event",
		TriggeredBy: evt.EventType,
	}
	if err = store.CreateTask(ctx, &task); err != nil {
		log.Printf("[%s] event-driven task failed: %v", SourceApplication, err)
		return err
	}
	emitEvent(ctx, events.EventAgentTaskCreated, "agent_task", task.ID,
		map[string]interface{}{"agent_id": agent.ID, "task_type": taskType, "source_event": evt.EventType})
	return nil
}

// createTwinForEvent registers a digital twin for an object referenced by an
// inbound event (idempotent within the tenant + entity pair).
func createTwinForEvent(ctx context.Context, tenantID string, evt events.EnterpriseEvent) error {
	if exists, _ := store.TwinExists(ctx, tenantID, evt.ObjectType, evt.ObjectID); exists {
		return nil
	}
	twin := model.DigitalTwin{
		TenantID:   tenantID,
		Name:       "Twin: " + evt.ObjectType + ":" + evt.ObjectID,
		EntityType: evt.ObjectType,
		EntityID:   evt.ObjectID,
		Parameters: evt.Payload,
		CreatedBy:  "event",
		Status:     "active",
	}
	if err := store.CreateTwin(ctx, &twin); err != nil {
		log.Printf("[%s] twin creation failed: %v", SourceApplication, err)
		return err
	}
	emitEvent(ctx, "twin.registered", "digital_twin", twin.ID, map[string]interface{}{"source_event": evt.EventType})
	return nil
}