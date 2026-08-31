package server

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/matjames/statgate-lib/events"
)

// EventBus is the StatCollect adapter for the canonical shared StatGate bus.
// The shared library provides the standard envelope and in-process fallback;
// this wrapper preserves the existing submission lifecycle API.
type EventBus struct {
	bus     *events.EventBus
	enabled bool
}

var eventBus *EventBus

// InitEventBus initializes the Redis-backed event bus.
func InitEventBus(addr, password string, enabled bool) *EventBus {
	if !enabled {
		eventBus = &EventBus{enabled: false}
		return eventBus
	}
	bus, err := events.NewEventBus(events.Config{
		RedisAddr: addr,
		Password:  password,
		Source:    "statcollect",
	})
	if err != nil {
		log.Printf("warning: shared event bus unavailable: %v", err)
		eventBus = &EventBus{enabled: false}
		return eventBus
	}
	eventBus = &EventBus{bus: bus, enabled: true}
	log.Printf("StatGate shared event bus initialized for statcollect")
	return eventBus
}

// Publish sends an event to the StatGate event channel.
func (eb *EventBus) Publish(eventType, objectType, objectID string, payload interface{}) {
	if eb == nil || !eb.enabled || eb.bus == nil {
		return
	}
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	err := eb.bus.Publish(context.Background(), events.EnterpriseEvent{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		Source:     "statcollect",
		ObjectType: objectType,
		ObjectID:   objectID,
		TenantID:   tenantID,
		Payload:    payloadMap(payload),
		Timestamp:  time.Now().UTC(),
		Version:    "1.0",
	})
	if err != nil {
		log.Printf("event publish failed: %v", err)
		return
	}
	log.Printf("published event: %s (%s:%s)", eventType, objectType, objectID)
}

// Close closes the Redis connection.
func (eb *EventBus) Close() {
	if eb != nil && eb.bus != nil {
		_ = eb.bus.Close()
	}
}

func payloadMap(payload interface{}) map[string]interface{} {
	if payload == nil {
		return map[string]interface{}{}
	}
	if values, ok := payload.(map[string]interface{}); ok {
		return values
	}
	encoded, err := json.Marshal(payload)
	if err == nil {
		var values map[string]interface{}
		if json.Unmarshal(encoded, &values) == nil && values != nil {
			return values
		}
	}
	return map[string]interface{}{"data": payload}
}

// EventType constants for StatCollect events
const (
	EventSubmissionReceived  = "submission.received"
	EventSubmissionValidated = "submission.validated"
	EventSubmissionRejected  = "submission.rejected"
	EventSubmissionLinked    = "submission.linked"
)

// SubmissionEventPayload carries submission metadata for downstream consumers.
type SubmissionEventPayload struct {
	InstanceID string   `json:"instance_id"`
	FormID     string   `json:"form_id"`
	Files      []string `json:"files"`
	ReceivedAt string   `json:"received_at"`
	Source     string   `json:"source"`
}

// PublishSubmissionReceived publishes a submission.received event.
func PublishSubmissionReceived(instanceID, formID string, files []string) {
	if eventBus == nil {
		return
	}
	payload := SubmissionEventPayload{
		InstanceID: instanceID,
		FormID:     formID,
		Files:      files,
		ReceivedAt: time.Now().Format(time.RFC3339),
		Source:     "odk",
	}
	eventBus.Publish(EventSubmissionReceived, "submission", instanceID, payload)
}

// PublishSubmissionValidated publishes a submission.validated event.
func PublishSubmissionValidated(instanceID, formID string) {
	if eventBus == nil {
		return
	}
	eventBus.Publish(EventSubmissionValidated, "submission", instanceID, map[string]string{
		"instance_id": instanceID,
		"form_id":     formID,
	})
}

// PublishSubmissionRejected publishes a submission.rejected event.
func PublishSubmissionRejected(instanceID, reason string) {
	if eventBus == nil {
		return
	}
	eventBus.Publish(EventSubmissionRejected, "submission", instanceID, map[string]string{
		"instance_id": instanceID,
		"reason":      reason,
	})
}

// PublishSubmissionLinked publishes a submission.linked event when a submission
// is linked to a StatChat object discussion.
func PublishSubmissionLinked(instanceID, conversationID string) {
	if eventBus == nil {
		return
	}
	eventBus.Publish(EventSubmissionLinked, "submission", instanceID, map[string]string{
		"instance_id":     instanceID,
		"conversation_id": conversationID,
	})
}

// EnsureEventBus returns the current event bus instance.
func EnsureEventBus() *EventBus {
	return eventBus
}

// String returns a human-readable description of the event bus state.
func (eb *EventBus) String() string {
	if eb == nil || !eb.enabled {
		return "disabled"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if eb.bus.RedisAvailable(ctx) {
		return "enabled (shared Redis event bus)"
	}
	return "enabled (shared event bus fallback)"
}
