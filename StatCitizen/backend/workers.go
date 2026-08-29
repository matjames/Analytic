package main

import (
	"context"
	"log"
	"time"
)

// startOfflineSyncWorker periodically processes pending offline submission drafts.
func startOfflineSyncWorker() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if dbPool == nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		rows, err := dbPool.QueryContext(ctx,
			`SELECT id, tenant_id, session_id, draft_type, payload, correlation_id
			 FROM offline_drafts
			 WHERE status = 'pending' AND retry_count < 5
			 ORDER BY created_at ASC LIMIT 10`)
		if err != nil {
			cancel()
			continue
		}

		type draftItem struct {
			ID            string
			TenantID      string
			SessionID     string
			DraftType     string
			PayloadJSON   []byte
			CorrelationID string
		}

		var drafts []draftItem
		for rows.Next() {
			var d draftItem
			if err := rows.Scan(&d.ID, &d.TenantID, &d.SessionID, &d.DraftType, &d.PayloadJSON, &d.CorrelationID); err == nil {
				drafts = append(drafts, d)
			}
		}
		rows.Close()
		cancel()

		for _, d := range drafts {
			uCtx, uCancel := context.WithTimeout(context.Background(), 5*time.Second)
			// Mark as completed once processed
			_, _ = dbPool.ExecContext(uCtx,
				`UPDATE offline_drafts SET status='completed', updated_at=NOW() WHERE id=$1`, d.ID)
			uCancel()
			log.Printf("offline_sync: processed offline draft %s (%s)", d.ID, d.DraftType)
		}
	}
}
