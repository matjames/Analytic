package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Event Persistence ────────────────────────────────────────────
// Every domain event is written to PostgreSQL BEFORE being published to Redis.
// This ensures that a Redis outage cannot silently destroy institutional events.
// Redis remains the real-time delivery mechanism; PostgreSQL is the source of truth.

// persistEvent writes the event to platform_events.
// Failure is logged but non-fatal — the event will still be published to Redis.
func persistEvent(ev DomainEvent) {
	if dbPool == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	payloadJSON := "{}"
	if ev.Payload != nil {
		if b, err := json.Marshal(ev.Payload); err == nil {
			payloadJSON = string(b)
		}
	}

	_, err := dbPool.ExecContext(ctx,
		`INSERT INTO platform_events
			(id, event_type, source, object_type, object_id, actor, tenant_id, project_id, org_id,
			 correlation_id, causation_id, schema_version, payload, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14,NOW())
		ON CONFLICT (id) DO NOTHING`,
		ev.ID, ev.EventType, ev.Source, ev.ObjectType, ev.ObjectID, ev.Actor,
		ev.TenantID, ev.ProjectID, ev.OrgID, ev.Correlation,
		"", // causation_id — populated downstream when available
		1,  // schema_version v1
		payloadJSON, "published",
	)
	if err != nil {
		log.Printf("event_persistence: failed to persist event %s (%s): %v", ev.ID, ev.EventType, err)
	}
	metricsCollector.incDBQuery()
	metricsCollector.incEvents()
}

// markEventFailed updates an event's status to 'failed' in the persistent store.
func markEventFailed(eventID, reason string, retryCount int) {
	if dbPool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = dbPool.ExecContext(ctx,
		`UPDATE platform_events SET status='failed', last_error=$1, retry_count=$2 WHERE id=$3`,
		reason, retryCount, eventID,
	)
	metricsCollector.incDBQuery()
}

// markEventProcessed updates an event's status to 'processed' in the persistent store.
func markEventProcessedDB(eventID string) {
	if dbPool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = dbPool.ExecContext(ctx,
		`UPDATE platform_events SET status='processed', processed_at=NOW() WHERE id=$1`,
		eventID,
	)
	metricsCollector.incDBQuery()
}

// ─── Event Replay API ─────────────────────────────────────────────
// POST /api/events/replay — admin-only, re-publishes a range of persisted events.

func handleEventReplay(c *gin.Context) {
	if !requireRole(c, "admin", "superadmin") {
		return
	}
	var req struct {
		EventType string `json:"event_type"`
		FromTime  string `json:"from"`
		ToTime    string `json:"to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request", "detail": err.Error()})
		return
	}
	if dbPool == nil {
		c.JSON(503, gin.H{"error": "event persistence not configured"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT id, event_type, source, object_type, object_id, actor, tenant_id,
		project_id, org_id, correlation_id, payload, created_at
		FROM platform_events WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if req.EventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argIdx)
		args = append(args, req.EventType)
		argIdx++
	}
	if req.FromTime != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, req.FromTime)
		argIdx++
	}
	if req.ToTime != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, req.ToTime)
		argIdx++
	}
	query += " ORDER BY created_at ASC LIMIT 500"

	rows, err := dbPool.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(500, gin.H{"error": "replay query failed", "detail": err.Error()})
		return
	}
	defer rows.Close()

	replayed := 0
	for rows.Next() {
		var ev DomainEvent
		var payloadJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.Source, &ev.ObjectType,
			&ev.ObjectID, &ev.Actor, &ev.TenantID, &ev.ProjectID, &ev.OrgID,
			&ev.Correlation, &payloadJSON, &createdAt); err != nil {
			continue
		}
		ev.Timestamp = createdAt.UTC().Format(time.RFC3339)
		if len(payloadJSON) > 0 {
			_ = json.Unmarshal(payloadJSON, &ev.Payload)
		}
		// Re-publish; idempotency key will be cleared to allow reprocessing
		// by generating a new replay-prefixed ID
		ev.ID = fmt.Sprintf("replay_%s", ev.ID)
		publishEvent(ev)
		replayed++
	}

	recordAuditFromContext(c, "events.replay", "platform_events", "", map[string]interface{}{
		"event_type": req.EventType, "from": req.FromTime, "to": req.ToTime, "replayed": replayed,
	})
	c.JSON(200, gin.H{"status": "replayed", "count": replayed})
}

// ─── Dead Letter Queue — Extended API ────────────────────────────
// GET  /api/events/dead-letter     — list DLQ from DB when available, Redis as fallback
// POST /api/events/dead-letter/:id/retry — retry a specific DLQ entry

func handleListDeadLetterDB(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 50)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rows, err := dbPool.QueryContext(ctx,
			`SELECT id, event_type, source, last_error, retry_count, created_at
			FROM platform_events WHERE status='failed' ORDER BY created_at DESC LIMIT $1`, limit)
		if err == nil {
			defer rows.Close()
			entries := []map[string]interface{}{}
			for rows.Next() {
				var id, etype, source, lastErr string
				var retryCount int
				var createdAt time.Time
				if err := rows.Scan(&id, &etype, &source, &lastErr, &retryCount, &createdAt); err == nil {
					entries = append(entries, map[string]interface{}{
						"event_id":    id,
						"event_type":  etype,
						"source":      source,
						"last_error":  lastErr,
						"retry_count": retryCount,
						"created_at":  createdAt.UTC().Format(time.RFC3339),
					})
				}
			}
			c.JSON(200, gin.H{"count": len(entries), "entries": entries, "source": "postgresql"})
			return
		}
	}
	// Fall back to Redis DLQ
	handleDeadLetterQueue(c)
}

func handleRetryDeadLetterEvent(c *gin.Context) {
	if !requireRole(c, "admin", "superadmin") {
		return
	}
	id := c.Param("id")
	if dbPool == nil {
		c.JSON(503, gin.H{"error": "event persistence not configured"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var ev DomainEvent
	var payloadJSON []byte
	err := dbPool.QueryRowContext(ctx,
		`SELECT id, event_type, source, object_type, object_id, actor, tenant_id, correlation_id, payload
		FROM platform_events WHERE id=$1 AND status='failed'`, id).
		Scan(&ev.ID, &ev.EventType, &ev.Source, &ev.ObjectType, &ev.ObjectID,
			&ev.Actor, &ev.TenantID, &ev.Correlation, &payloadJSON)
	if err != nil {
		c.JSON(404, gin.H{"error": "failed event not found", "id": id})
		return
	}
	if len(payloadJSON) > 0 {
		_ = json.Unmarshal(payloadJSON, &ev.Payload)
	}
	ev.ID = fmt.Sprintf("retry_%s", ev.ID)
	publishEvent(ev)

	// Mark original as retried
	_, _ = dbPool.ExecContext(ctx,
		`UPDATE platform_events SET status='retried', processed_at=NOW() WHERE id=$1`, id)

	recordAuditFromContext(c, "events.dlq.retry", "platform_events", id, nil)
	c.JSON(200, gin.H{"status": "retried", "original_id": id, "new_id": ev.ID})
}
