package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Audit Log ────────────────────────────────────────────────────
// Immutable, append-only audit record for all significant institutional actions.
// Writes to PostgreSQL when available; logs to stderr as fallback.

type AuditEntry struct {
	Actor         string                 `json:"actor"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	ResourceID    string                 `json:"resource_id"`
	TenantID      string                 `json:"tenant_id"`
	CorrelationID string                 `json:"correlation_id"`
	RequestID     string                 `json:"request_id"`
	Outcome       string                 `json:"outcome"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     string                 `json:"created_at"`
}

// recordAuditEvent persists an audit event to PostgreSQL.
// Falls back to structured log if database is unavailable.
func recordAuditEvent(actor, action, resource, resourceID, tenantID, corrID, reqID string, meta map[string]interface{}) {
	entry := AuditEntry{
		Actor:         actor,
		Action:        action,
		Resource:      resource,
		ResourceID:    resourceID,
		TenantID:      tenantID,
		CorrelationID: corrID,
		RequestID:     reqID,
		Outcome:       "success",
		Metadata:      meta,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		metaJSON := "{}"
		if meta != nil {
			if b, err := json.Marshal(meta); err == nil {
				metaJSON = string(b)
			}
		}
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO platform_audit_log
				(actor, action, resource, resource_id, tenant_id, correlation_id, request_id, outcome, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)`,
			entry.Actor, entry.Action, entry.Resource, entry.ResourceID,
			entry.TenantID, entry.CorrelationID, entry.RequestID, entry.Outcome, metaJSON,
		)
		if err != nil {
			log.Printf("audit: DB write failed for %s/%s: %v", action, resource, err)
		}
		metricsCollector.incDBQuery()
		return
	}

	// Fallback: structured log
	log.Printf("audit: actor=%s action=%s resource=%s/%s tenant=%s corr=%s",
		actor, action, resource, resourceID, tenantID, corrID)
}

// recordAuditFromContext is a helper that extracts correlation/request IDs from
// the Gin context automatically, reducing boilerplate in handlers.
func recordAuditFromContext(c *gin.Context, action, resource, resourceID string, meta map[string]interface{}) {
	actor := getContextUserID(c)
	tenantID := getContextTenantID(c)
	corrID, _ := c.Get("correlation_id")
	reqID, _ := c.Get("request_id")
	corrStr, _ := corrID.(string)
	reqStr, _ := reqID.(string)
	recordAuditEvent(actor, action, resource, resourceID, tenantID, corrStr, reqStr, meta)
}

// ─── Audit Log API Handler ────────────────────────────────────────
// GET /api/apis/audit  (replaces the stub from governance.go)
// Returns paginated audit log from PostgreSQL when available.

func handleAuditLog(c *gin.Context) {
	if !requireRole(c, "admin", "superadmin", "platform_admin", "tenant_admin", "governance_officer") {
		return
	}
	limit := parseIntDefault(c.Query("limit"), 100)
	offset := parseIntDefault(c.Query("offset"), 0)
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	actor := c.Query("actor")
	action := c.Query("action")
	resource := c.Query("resource")

	if dbPool != nil {
		entries, total, err := queryAuditLog(limit, offset, actor, action, resource, getContextTenantID(c))
		if err != nil {
			log.Printf("audit: query failed: %v", err)
			c.JSON(500, gin.H{"error": "audit log query failed"})
			return
		}
		c.JSON(200, gin.H{
			"total":   total,
			"count":   len(entries),
			"offset":  offset,
			"entries": entries,
			"source":  "postgresql",
		})
		return
	}

	// Fallback: return empty when no DB
	c.JSON(200, gin.H{
		"total":   0,
		"count":   0,
		"offset":  offset,
		"entries": []interface{}{},
		"source":  "unavailable",
		"message": "Audit log database is not configured.",
	})
}

func queryAuditLog(limit, offset int, actor, action, resource, tenantID string) ([]AuditEntry, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build dynamic WHERE clause
	where := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argIdx := 1
	if actor != "" {
		argIdx++
		where += fmt.Sprintf(" AND actor = $%d", argIdx)
		args = append(args, actor)
		argIdx++
	}
	if action != "" {
		argIdx++
		where += fmt.Sprintf(" AND action ILIKE $%d", argIdx)
		args = append(args, "%"+action+"%")
		argIdx++
	}
	if resource != "" {
		argIdx++
		where += fmt.Sprintf(" AND resource = $%d", argIdx)
		args = append(args, resource)
		argIdx++
	}

	// Count
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := dbPool.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM platform_audit_log "+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data
	args = append(args, limit, offset)
	rows, err := dbPool.QueryContext(ctx,
		fmt.Sprintf(`SELECT actor, action, resource, resource_id, tenant_id, correlation_id, request_id, outcome, metadata, created_at
		FROM platform_audit_log %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1),
		args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	metricsCollector.incDBQuery()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var metaJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&e.Actor, &e.Action, &e.Resource, &e.ResourceID,
			&e.TenantID, &e.CorrelationID, &e.RequestID, &e.Outcome, &metaJSON, &createdAt); err != nil {
			continue
		}
		e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &e.Metadata)
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []AuditEntry{}
	}
	return entries, total, nil
}

// ─── Legacy shim ─────────────────────────────────────────────────
// recordAudit is called throughout routes.go using the old 4-arg signature.
// This shim bridges it to the new structured audit event AND the in-memory
// auditLog store (used by monitoring endpoints for supplementary display).

func recordAudit(action, resource, actor string, meta map[string]interface{}) {
	resourceID := ""
	if id, ok := meta["id"].(string); ok {
		resourceID = id
	} else if id, ok := meta["object_id"].(string); ok {
		resourceID = id
	}
	recordAuditEvent(actor, action, resource, resourceID, "", "", "", meta)
	// Also write to supplementary in-memory log for backwards compat
	appendAuditRecord(action, resource, actor, meta)
}

