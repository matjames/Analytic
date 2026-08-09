package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func handleListCalendarEvents(c *gin.Context) {
	category := c.Query("category")
	source := c.Query("source")
	project := c.Query("project_id")
	start := c.Query("start")
	end := c.Query("end")
	limit := parseIntDefault(c.Query("limit"), 100)

	events := fetchCalendarEvents(limit)
	filtered := make([]CalendarEvent, 0, len(events))
	for _, e := range events {
		if category != "" && e.Category != category {
			continue
		}
		if source != "" && e.SourceApp != source {
			continue
		}
		if project != "" && e.ProjectID != project {
			continue
		}
		if start != "" && e.Start < start {
			continue
		}
		if end != "" && e.Start > end {
			continue
		}
		filtered = append(filtered, e)
		if len(filtered) >= limit {
			break
		}
	}
	c.JSON(200, gin.H{"count": len(filtered), "events": filtered})
}

func fetchCalendarEvents(limit int) []CalendarEvent {
	if redisClient == nil {
		return []CalendarEvent{}
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:calendar", 0, int64(limit-1)).Result()
	events := make([]CalendarEvent, 0, len(raw))
	for _, item := range raw {
		var e CalendarEvent
		if err := json.Unmarshal([]byte(item), &e); err == nil {
			events = append(events, e)
		}
	}
	return events
}

func handleGetCalendarEvent(c *gin.Context) {
	id := c.Param("id")
	events := fetchCalendarEvents(500)
	for _, e := range events {
		if e.ID == id {
			c.JSON(200, e)
			return
		}
	}
	c.JSON(404, gin.H{"error": "calendar event not found"})
}

func handleCreateCalendarEvent(c *gin.Context) {
	var e CalendarEvent
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(400, gin.H{"error": "invalid calendar event"})
		return
	}
	if e.ID == "" {
		e.ID = fmt.Sprintf("cal_%d", time.Now().UnixNano())
	}
	if e.CreatedAt == "" {
		e.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if redisClient != nil {
		data, _ := json.Marshal(e)
		redisClient.RPush(context.Background(), "statgate:calendar", string(data))
	}
	publishEvent(DomainEvent{
		EventType:  "calendar.event.created",
		Source:     e.SourceApp,
		ObjectType: "calendar",
		ObjectID:   e.ID,
		ProjectID:  e.ProjectID,
		Payload:    map[string]interface{}{"title": e.Title, "category": e.Category, "start": e.Start},
	})
	c.JSON(201, e)
}

func handleUpdateCalendarEvent(c *gin.Context) {
	id := c.Param("id")
	events := fetchCalendarEvents(500)
	for _, e := range events {
		if e.ID == id {
			var updates CalendarEvent
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(400, gin.H{"error": "invalid calendar event"})
				return
			}
			if updates.Title != "" {
				e.Title = updates.Title
			}
			if updates.Start != "" {
				e.Start = updates.Start
			}
			if updates.End != "" {
				e.End = updates.End
			}
			if updates.Location != "" {
				e.Location = updates.Location
			}
			persistCalendarEvent(id, e)
			c.JSON(200, e)
			return
		}
	}
	c.JSON(404, gin.H{"error": "calendar event not found"})
}

func handleDeleteCalendarEvent(c *gin.Context) {
	id := c.Param("id")
	if redisClient != nil {
		ctx := context.Background()
		raw, _ := redisClient.LRange(ctx, "statgate:calendar", 0, -1).Result()
		redisClient.Del(ctx, "statgate:calendar")
		for _, item := range raw {
			var e CalendarEvent
			if err := json.Unmarshal([]byte(item), &e); err == nil {
				if e.ID != id {
					redisClient.RPush(ctx, "statgate:calendar", item)
				}
			}
		}
	}
	c.JSON(200, gin.H{"status": "ok", "id": id, "deleted": true})
}

func persistCalendarEvent(id string, updated CalendarEvent) {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:calendar", 0, -1).Result()
	redisClient.Del(ctx, "statgate:calendar")
	for _, item := range raw {
		var e CalendarEvent
		if err := json.Unmarshal([]byte(item), &e); err == nil {
			if e.ID == id {
				e = updated
			}
			data, _ := json.Marshal(e)
			redisClient.RPush(ctx, "statgate:calendar", string(data))
		}
	}
}

// calendarFromEvent creates a calendar entry from a domain event.
func calendarFromEvent(ev DomainEvent, category, prefix string) {
	start := ev.Timestamp
	end := ""
	if t, err := time.Parse(time.RFC3339, start); err == nil {
		end = t.Add(1 * time.Hour).Format(time.RFC3339)
	}
	title := prefix
	if n, ok := ev.Payload["name"].(string); ok && n != "" {
		title = fmt.Sprintf("%s: %s", prefix, n)
	} else if t, ok := ev.Payload["title"].(string); ok && t != "" {
		title = fmt.Sprintf("%s: %s", prefix, t)
	}
	if s, ok := ev.Payload["start"].(string); ok && s != "" {
		start = s
	}
	if e, ok := ev.Payload["end"].(string); ok && e != "" {
		end = e
	}
	calEvt := CalendarEvent{
		ID:          fmt.Sprintf("cal_%d", time.Now().UnixNano()),
		Title:       title,
		Description: ev.EventType,
		Start:       start,
		End:         end,
		Category:    category,
		SourceApp:   ev.Source,
		EntityID:    ev.ObjectID,
		ProjectID:   ev.ProjectID,
		Metadata:    ev.Payload,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if redisClient != nil {
		data, _ := json.Marshal(calEvt)
		redisClient.RPush(context.Background(), "statgate:calendar", string(data))
		redisClient.LTrim(context.Background(), "statgate:calendar", 0, 1999)
	}
}
