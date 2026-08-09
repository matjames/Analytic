package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// StatGateEvent is the standard event schema for cross-module communication.
// Per the StatGate Engineering Directive, every module must communicate
// through the event bus so actions in one module propagate to others.
type StatGateEvent struct {
	EventType  string      `json:"event_type"`
	Source     string      `json:"source"`
	ObjectType string      `json:"object_type"`
	ObjectID   string      `json:"object_id"`
	TenantID   string      `json:"tenant_id"`
	Payload    interface{} `json:"payload"`
	Timestamp  time.Time   `json:"timestamp"`
}

// EventBus publishes StatGate events to Redis for cross-module communication.
type EventBus struct {
	client  *redis.Client
	enabled bool
}

var eventBus *EventBus

// InitEventBus initializes the Redis-backed event bus.
func InitEventBus(addr, password string, enabled bool) *EventBus {
	if !enabled {
		eventBus = &EventBus{enabled: false}
		return eventBus
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("warning: event bus (Redis) unavailable at %s: %v", addr, err)
		eventBus = &EventBus{enabled: false}
		return eventBus
	}
	eventBus = &EventBus{client: client, enabled: true}
	log.Printf("StatGate event bus connected to Redis at %s", addr)
	return eventBus
}

// Publish sends an event to the StatGate event channel.
func (eb *EventBus) Publish(eventType, objectType, objectID string, payload interface{}) {
	if eb == nil || !eb.enabled || eb.client == nil {
		return
	}
	evt := StatGateEvent{
		EventType:  eventType,
		Source:     "statcollect",
		ObjectType: objectType,
		ObjectID:   objectID,
		TenantID:   cfg.TenantID,
		Payload:    payload,
		Timestamp:  time.Now().UTC(),
	}
	b, err := json.Marshal(evt)
	if err != nil {
		log.Printf("event marshal failed: %v", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := eb.client.Publish(ctx, "statgate:events", b).Err(); err != nil {
		log.Printf("event publish failed: %v", err)
		return
	}
	log.Printf("published event: %s (%s:%s)", eventType, objectType, objectID)
}

// Close closes the Redis connection.
func (eb *EventBus) Close() {
	if eb != nil && eb.client != nil {
		eb.client.Close()
	}
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
	return fmt.Sprintf("enabled (Redis at %s)", eb.client.Options().Addr)
}
