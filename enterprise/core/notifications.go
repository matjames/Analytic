package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func createNotificationRecord(n Notification) (string, error) {
	if n.ID == "" {
		n.ID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	}
	if n.CreatedAt == "" {
		n.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if n.Priority == "" {
		n.Priority = "medium"
	}
	if redisClient == nil || n.UserID == "" {
		return n.ID, nil
	}
	key := fmt.Sprintf("statgate:notifications:%s", n.UserID)
	data, _ := json.Marshal(n)
	ctx := context.Background()
	if err := redisClient.LPush(ctx, key, string(data)).Err(); err != nil {
		return n.ID, err
	}
	redisClient.LTrim(ctx, key, 0, 499)
	// Publish realtime notification event
	publishEvent(DomainEvent{
		ID:         fmt.Sprintf("notif_evt_%d", time.Now().UnixNano()),
		EventType:  "notification.created",
		Source:     "enterprise",
		ObjectType: "notification",
		ObjectID:   n.ID,
		Actor:      n.UserID,
		Payload:    map[string]interface{}{"title": n.Title, "body": n.Body, "category": n.Category, "priority": n.Priority},
		Timestamp:  n.CreatedAt,
	})
	return n.ID, nil
}

func handleListNotifications(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	includeArchived := c.Query("archived") == "true"
	includeRead := c.Query("read") == "true"
	limit := parseIntDefault(c.Query("limit"), 50)

	notifs := fetchNotifications(userID, limit)
	filtered := make([]Notification, 0, len(notifs))
	for _, n := range notifs {
		if !includeArchived && n.Archived {
			continue
		}
		if !includeRead && n.Read {
			continue
		}
		filtered = append(filtered, n)
		if len(filtered) >= limit {
			break
		}
	}
	c.JSON(200, gin.H{"count": len(filtered), "notifications": filtered})
}

func fetchNotifications(userID string, limit int) []Notification {
	if redisClient == nil || userID == "" {
		return []Notification{}
	}
	key := fmt.Sprintf("statgate:notifications:%s", userID)
	ctx := context.Background()
	raw, err := redisClient.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return []Notification{}
	}
	notifs := make([]Notification, 0, len(raw))
	for _, item := range raw {
		var n Notification
		if err := json.Unmarshal([]byte(item), &n); err == nil {
			notifs = append(notifs, n)
		}
	}
	return notifs
}

func handleGetNotification(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("user_id")
	notifs := fetchNotifications(userID, 500)
	for _, n := range notifs {
		if n.ID == id {
			c.JSON(200, n)
			return
		}
	}
	c.JSON(404, gin.H{"error": "notification not found"})
}

func handleMarkNotificationRead(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("user_id")
	if err := updateNotification(userID, id, func(n *Notification) { n.Read = true }); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "id": id, "read": true})
}

func handleMarkAllRead(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = c.GetHeader("X-User-ID")
	}
	if redisClient == nil || userID == "" {
		c.JSON(200, gin.H{"status": "ok", "count": 0})
		return
	}
	key := fmt.Sprintf("statgate:notifications:%s", userID)
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, key, 0, -1).Result()
	count := 0
	for i, item := range raw {
		var n Notification
		if err := json.Unmarshal([]byte(item), &n); err == nil {
			if !n.Read {
				n.Read = true
				count++
				data, _ := json.Marshal(n)
				redisClient.LSet(ctx, key, int64(i), string(data))
			}
		}
	}
	c.JSON(200, gin.H{"status": "ok", "count": count})
}

func handleArchiveNotification(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("user_id")
	if err := updateNotification(userID, id, func(n *Notification) { n.Archived = true }); err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok", "id": id, "archived": true})
}

func handleDeleteNotification(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("user_id")
	if redisClient == nil || userID == "" {
		c.JSON(400, gin.H{"error": "user_id required"})
		return
	}
	key := fmt.Sprintf("statgate:notifications:%s", userID)
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, key, 0, -1).Result()
	newList := make([]string, 0, len(raw))
	for _, item := range raw {
		var n Notification
		if err := json.Unmarshal([]byte(item), &n); err == nil {
			if n.ID != id {
				newList = append(newList, item)
			}
		}
	}
	redisClient.Del(ctx, key)
	if len(newList) > 0 {
		items := make([]interface{}, len(newList))
		for i, v := range newList {
			items[i] = v
		}
		redisClient.RPush(ctx, key, items...)
	}
	c.JSON(200, gin.H{"status": "ok", "id": id, "deleted": true})
}

func handleCreateNotification(c *gin.Context) {
	var n Notification
	if err := c.ShouldBindJSON(&n); err != nil {
		c.JSON(400, gin.H{"error": "invalid notification", "detail": err.Error()})
		return
	}
	id, err := createNotificationRecord(n)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to create notification", "detail": err.Error()})
		return
	}
	c.JSON(201, gin.H{"id": id, "status": "created"})
}

func updateNotification(userID, id string, mutate func(*Notification)) error {
	if redisClient == nil || userID == "" {
		return fmt.Errorf("notification service unavailable")
	}
	key := fmt.Sprintf("statgate:notifications:%s", userID)
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, key, 0, -1).Result()
	for i, item := range raw {
		var n Notification
		if err := json.Unmarshal([]byte(item), &n); err == nil {
			if n.ID == id {
				mutate(&n)
				data, _ := json.Marshal(n)
				return redisClient.LSet(ctx, key, int64(i), string(data)).Err()
			}
		}
	}
	return fmt.Errorf("notification not found")
}
