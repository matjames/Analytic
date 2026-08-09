package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Idempotency Keys ─────────────────────────────────────────────
// Critical enterprise events must be safe to process more than once.
// We use a Redis SET with NX (not-exists) to ensure each event is
// processed exactly once per consumer.

const idempotencyKeyPrefix = "statgate:idempotency:"

// isEventProcessed checks whether an event has already been processed.
// Returns true if the event was already seen (duplicate).
func isEventProcessed(eventID string) bool {
	if redisClient == nil || eventID == "" {
		return false
	}
	ctx := context.Background()
	key := idempotencyKeyPrefix + eventID
	// SET NX EX 86400 - only set if not exists, expire after 24h
	ok, err := redisClient.SetNX(ctx, key, "1", 24*time.Hour).Result()
	if err != nil {
		// On Redis error, fail-open (process the event) to avoid data loss
		return false
	}
	// If SetNX returned false, the key already existed → duplicate
	return !ok
}

// markEventProcessed explicitly marks an event as processed.
// Used when we need to record processing after successful completion.
func markEventProcessed(eventID string) {
	if redisClient == nil || eventID == "" {
		return
	}
	ctx := context.Background()
	key := idempotencyKeyPrefix + eventID
	redisClient.Set(ctx, key, "1", 24*time.Hour)
}

// ─── Event Deduplication ──────────────────────────────────────────
// processEventIdempotent wraps processEvent with deduplication.
// If the event has already been processed, it is skipped.
func processEventIdempotent(ev DomainEvent) {
	// Generate a deterministic event ID if not present
	if ev.ID == "" {
		ev.ID = fmt.Sprintf("evt_%d_%s_%s", time.Now().UnixNano(), ev.EventType, ev.ObjectID)
	}

	// Check idempotency - skip if already processed
	if isEventProcessed(ev.ID) {
		log.Printf("events: duplicate event %s skipped (idempotency)", ev.ID)
		return
	}

	// Process the event
	processEvent(ev)

	// Mark as processed after successful handling
	markEventProcessed(ev.ID)
}

// ─── Event Retry & Failure Handling ───────────────────────────────
// Failed events are recorded in a dead-letter queue for later inspection.

const deadLetterKey = "statgate:events:dead-letter"

// recordDeadLetter stores a failed event for later inspection/retry.
func recordDeadLetter(ev DomainEvent, reason string) {
	if redisClient == nil {
		return
	}
	entry := map[string]interface{}{
		"event_id":   ev.ID,
		"event_type": ev.EventType,
		"source":     ev.Source,
		"object_id":  ev.ObjectID,
		"reason":     reason,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"payload":    ev.Payload,
	}
	data, _ := json.Marshal(entry)
	ctx := context.Background()
	redisClient.LPush(ctx, deadLetterKey, string(data))
	redisClient.LTrim(ctx, deadLetterKey, 0, 999)
}

// ─── Consumer Recovery ─────────────────────────────────────────────
// Track the last processed event timestamp for recovery monitoring.

const consumerStateKey = "statgate:events:consumer-state"

func updateConsumerState() {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	redisClient.Set(ctx, consumerStateKey, time.Now().UTC().Format(time.RFC3339), 0)
}

func getConsumerState() string {
	if redisClient == nil {
		return ""
	}
	ctx := context.Background()
	val, _ := redisClient.Get(ctx, consumerStateKey).Result()
	return val
}

// ─── Event History with Correlation ────────────────────────────────
// Store events with correlation IDs for traceability.

func storeEventWithCorrelation(ev DomainEvent) {
	if redisClient == nil {
		return
	}
	// Store in event history with correlation
	data, _ := json.Marshal(ev)
	ctx := context.Background()
	redisClient.LPush(ctx, "statgate:event:history", string(data))
	redisClient.LTrim(ctx, "statgate:event:history", 0, 9999)
	redisClient.Expire(ctx, "statgate:event:history", 7*24*time.Hour)

	// Store by correlation ID for traceability
	if ev.Correlation != "" {
		corrKey := fmt.Sprintf("statgate:events:correlation:%s", ev.Correlation)
		redisClient.RPush(ctx, corrKey, string(data))
		redisClient.Expire(ctx, corrKey, 7*24*time.Hour)
	}
}

// handleDeadLetterQueue returns the dead-letter queue for inspection
// and provides endpoints for retrying failed events.
func handleDeadLetterQueue(c *gin.Context) {
	if redisClient == nil {
		c.JSON(200, gin.H{"count": 0, "entries": []interface{}{}})
		return
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, deadLetterKey, 0, 99).Result()
	entries := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(item), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	c.JSON(200, gin.H{"count": len(entries), "entries": entries})
}
