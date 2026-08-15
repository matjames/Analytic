package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

func handleTimeline(c *gin.Context) {
	app := c.Query("application")
	entity := c.Query("entity")
	user := c.Query("user")
	project := c.Query("project")
	limit := parseIntDefault(c.Query("limit"), 50)

	entries := fetchTimeline(limit)
	filtered := make([]TimelineEntry, 0, len(entries))
	for _, e := range entries {
		if app != "" && e.Application != app {
			continue
		}
		if entity != "" && e.Entity != entity {
			continue
		}
		if user != "" && e.User != user {
			continue
		}
		if project != "" && e.ProjectID != project {
			continue
		}
		filtered = append(filtered, e)
		if len(filtered) >= limit {
			break
		}
	}
	c.JSON(200, gin.H{"count": len(filtered), "entries": filtered})
}

func handleTimelineByEntity(c *gin.Context) {
	entity := c.Param("entity")
	id := c.Param("id")
	limit := parseIntDefault(c.Query("limit"), 50)

	entries := fetchTimeline(limit)
	filtered := make([]TimelineEntry, 0)
	for _, e := range entries {
		if e.Entity == entity && e.EntityID == id {
			filtered = append(filtered, e)
			if len(filtered) >= limit {
				break
			}
		}
	}
	c.JSON(200, gin.H{"count": len(filtered), "entries": filtered})
}

func handleTimelineByUser(c *gin.Context) {
	user := c.Param("user")
	limit := parseIntDefault(c.Query("limit"), 50)

	entries := fetchTimeline(limit)
	filtered := make([]TimelineEntry, 0)
	for _, e := range entries {
		if e.User == user {
			filtered = append(filtered, e)
			if len(filtered) >= limit {
				break
			}
		}
	}
	c.JSON(200, gin.H{"count": len(filtered), "entries": filtered})
}

func fetchTimeline(limit int) []TimelineEntry {
	if redisClient != nil {
		ctx := context.Background()
		raw, err := redisClient.LRange(ctx, "statgate:timeline", 0, int64(limit-1)).Result()
		if err == nil && len(raw) > 0 {
			entries := make([]TimelineEntry, 0, len(raw))
			for _, item := range raw {
				var e TimelineEntry
				if err := json.Unmarshal([]byte(item), &e); err == nil {
					entries = append(entries, e)
				}
			}
			// Sort newest first
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Timestamp > entries[j].Timestamp
			})
			return entries
		}
	}

	// Fallback to PostgreSQL
	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rows, err := dbPool.QueryContext(ctx,
			`SELECT id, user_id, tenant_id, application, entity, entity_id, action, description, project_id, org_id, metadata, timestamp
			FROM platform_timeline ORDER BY timestamp DESC LIMIT $1`, limit)
		if err == nil {
			defer rows.Close()
			entries := make([]TimelineEntry, 0)
			for rows.Next() {
				var e TimelineEntry
				var metaJSON []byte
				var ts time.Time
				if err := rows.Scan(&e.ID, &e.User, &e.TenantID, &e.Application, &e.Entity, &e.EntityID, &e.Action, &e.Description, &e.ProjectID, &e.OrgID, &metaJSON, &ts); err == nil {
					e.Timestamp = ts.UTC().Format(time.RFC3339)
					if len(metaJSON) > 0 {
						_ = json.Unmarshal(metaJSON, &e.Metadata)
					}
					entries = append(entries, e)
				}
			}
			return entries
		}
	}
	return []TimelineEntry{}
}

func handleCreateTimelineEntry(c *gin.Context) {
	var entry TimelineEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(400, gin.H{"error": "invalid timeline entry", "detail": err.Error()})
		return
	}
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("tl_%d", time.Now().UnixNano())
	}
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	// Persist to PostgreSQL
	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		metaJSON, _ := json.Marshal(entry.Metadata)
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO platform_timeline 
				(id, user_id, tenant_id, application, entity, entity_id, action, description, project_id, org_id, metadata, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12)
			ON CONFLICT (id) DO NOTHING`,
			entry.ID, entry.User, entry.TenantID, entry.Application, entry.Entity, entry.EntityID,
			entry.Action, entry.Description, entry.ProjectID, entry.OrgID, string(metaJSON), entry.Timestamp,
		)
		if err != nil {
			log.Printf("timeline: DB write error: %v", err)
		}
	}

	if redisClient != nil {
		data, _ := json.Marshal(entry)
		redisClient.LPush(context.Background(), "statgate:timeline", string(data))
		redisClient.LTrim(context.Background(), "statgate:timeline", 0, 4999)
	}
	c.JSON(201, entry)
}
