package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	platformevents "github.com/matjames/statgate-lib/events"
)

// ─── Enterprise Event Bus ─────────────────────────────────────────────────────
//
// StatCitizen publishes to the same Redis channel used by all StatGate applications.
// It does NOT create a second event bus — it reuses the enterprise infrastructure.
// Channel: STATGATE_EVENT_CHANNEL (default: "statgate:events")

// StatGateEvent is the canonical enterprise event envelope.
// Matches the DomainEvent struct in Enterprise Core (main.go).
type StatGateEvent struct {
	ID            string                 `json:"id"`
	EventType     string                 `json:"event_type"`
	Source        string                 `json:"source"` // always "statcitizen"
	ObjectType    string                 `json:"object_type"`
	ObjectID      string                 `json:"object_id"`
	Actor         string                 `json:"actor"`
	TenantID      string                 `json:"tenant_id"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
	Timestamp     string                 `json:"timestamp"`
}

// EventBus manages Redis-backed event publishing.
type EventBus struct {
	bus     *platformevents.EventBus
	enabled bool
}

var eventBus *EventBus

// initEventBus initializes the Redis-backed event bus.
func initEventBus(cfg *Config) {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	bus, err := platformevents.NewEventBus(platformevents.Config{
		RedisAddr: addr,
		Password:  cfg.RedisPassword,
		Source:    "statcitizen",
		Channel:   cfg.EventChannel,
	})
	if err != nil {
		log.Printf("[WARNING] Shared event bus unavailable at %s: %v — events will be queued", addr, err)
		eventBus = &EventBus{enabled: false}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	enabled := bus.RedisAvailable(ctx)
	eventBus = &EventBus{bus: bus, enabled: enabled}
	if enabled {
		log.Printf("event bus: connected to shared durable Redis bus at %s, channel=%s", addr, cfg.EventChannel)
	} else {
		log.Printf("[WARNING] Event bus (Redis) unavailable at %s: events will be queued", addr)
	}
}

func closeEventBus() {
	if eventBus != nil && eventBus.bus != nil {
		_ = eventBus.bus.Close()
	}
}

// Publish sends a StatCitizen event to the enterprise event bus.
// If Redis is unavailable, the event is written to the dead-letter table for retry.
func (eb *EventBus) Publish(eventType, objectType, objectID, tenantID, actor, correlationID string, payload map[string]interface{}) {
	if correlationID == "" {
		correlationID = fmt.Sprintf("corr_%d", time.Now().UnixNano())
	}

	if eb != nil && eb.enabled && eb.bus != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := eb.bus.PublishDurable(ctx, newCitizenEnterpriseEvent(eventType, objectType, objectID, tenantID, actor, correlationID, payload)); err != nil {
			log.Printf("event: publish failed for %s: %v — writing to dead letter", eventType, err)
			writeDeadLetter(eventType, tenantID, payload)
			return
		}
		log.Printf("event: published %s (object=%s:%s, corr=%s)", eventType, objectType, objectID, correlationID)
		return
	}

	// Redis unavailable — write to dead letter queue for later retry
	log.Printf("event: bus unavailable, queueing %s to dead letter", eventType)
	writeDeadLetter(eventType, tenantID, payload)
}

func newCitizenEnterpriseEvent(eventType, objectType, objectID, tenantID, actor, correlationID string, payload map[string]interface{}) platformevents.EnterpriseEvent {
	if correlationID == "" {
		correlationID = fmt.Sprintf("corr_%d", time.Now().UnixNano())
	}
	return platformevents.EnterpriseEvent{
		EventID:       fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType:     eventType,
		Source:        "statcitizen",
		ObjectType:    objectType,
		ObjectID:      objectID,
		UserID:        actor,
		TenantID:      tenantID,
		CorrelationID: correlationID,
		Payload:       payload,
		Timestamp:     time.Now().UTC(),
		Version:       "1.0",
	}
}

// writeDeadLetter persists a failed event for later retry.
func writeDeadLetter(eventType, tenantID string, payload map[string]interface{}) {
	if dbPool == nil {
		log.Printf("dead-letter: no DB available, event %s lost", eventType)
		return
	}
	id := fmt.Sprintf("dlq_%d", time.Now().UnixNano())
	payloadJSON, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx,
		`INSERT INTO event_dead_letter (id, tenant_id, event_type, payload, status)
		 VALUES ($1, $2, $3, $4::jsonb, 'pending')`,
		id, tenantID, eventType, string(payloadJSON),
	)
	if err != nil {
		log.Printf("dead-letter: failed to write to DB for event %s: %v", eventType, err)
	}
}

// startEventRetryWorker runs a background goroutine that retries dead-letter events.
func startEventRetryWorker() {
	if dbPool == nil {
		return
	}
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		retryDeadLetterEvents()
	}
}

func retryDeadLetterEvents() {
	if dbPool == nil || eventBus == nil || !eventBus.enabled {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := dbPool.QueryContext(ctx,
		`SELECT id, tenant_id, event_type, payload FROM event_dead_letter
		 WHERE status = 'pending' AND retry_count < 10
		 ORDER BY created_at LIMIT 10`)
	if err != nil {
		log.Printf("dead-letter retry: query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, tenantID, eventType, payloadStr string
		if err := rows.Scan(&id, &tenantID, &eventType, &payloadStr); err != nil {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			continue
		}

		pubCtx, pubCancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := eventBus.bus.PublishDurable(pubCtx, newCitizenEnterpriseEvent(eventType, "", "", tenantID, "", "", payload))
		pubCancel()

		if err == nil {
			dbPool.ExecContext(ctx, `UPDATE event_dead_letter SET status='completed', updated_at=NOW() WHERE id=$1`, id)
			log.Printf("dead-letter retry: succeeded for event %s (id=%s)", eventType, id)
		} else {
			dbPool.ExecContext(ctx,
				`UPDATE event_dead_letter SET retry_count=retry_count+1, last_error=$2, updated_at=NOW() WHERE id=$1`,
				id, err.Error())
		}
	}
}

// ─── Citizen Event Type Constants ─────────────────────────────────────────────

const (
	// Identity events
	EventCitizenRegistered = "citizen.registered"
	EventCitizenVerified   = "citizen.verified"

	// Feedback events
	EventFeedbackCreated  = "citizen.feedback.created"
	EventFeedbackUpdated  = "citizen.feedback.updated"
	EventFeedbackAssigned = "citizen.feedback.assigned"
	EventFeedbackResolved = "citizen.feedback.resolved"

	// Report events
	EventReportCreated  = "citizen.report.created"
	EventReportAssigned = "citizen.report.assigned"
	EventReportResolved = "citizen.report.resolved"

	// Consultation events
	EventConsultationJoined    = "citizen.consultation.joined"
	EventConsultationResponded = "citizen.consultation.response_submitted"

	// Rating events
	EventServiceRatingCreated = "citizen.service_rating.created"

	// Case events
	EventCaseCreated = "citizen.case.created"
	EventCaseUpdated = "citizen.case.updated"
	EventCaseClosed  = "citizen.case.closed"

	// Publication events
	EventPublicationCreated   = "citizen.publication.created"
	EventPublicationPublished = "citizen.publication.published"
)

// publishCitizenEvent is the standard way for all StatCitizen handlers to emit events.
func publishCitizenEvent(eventType, objectType, objectID, tenantID, actor, correlationID string, payload map[string]interface{}) {
	if eventBus == nil {
		log.Printf("event: bus not initialized, skipping %s", eventType)
		return
	}
	// Enrich payload with canonical object ID
	if objectID != "" && tenantID != "" {
		payload["canonical_id"] = fmt.Sprintf("%s:statcitizen:%s:%s", tenantID, objectType, objectID)
	}
	eventBus.Publish(eventType, objectType, objectID, tenantID, actor, correlationID, payload)
}
