package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

func initRedis() error {
	host := getEnv("REDIS_HOST", "redis")
	port := getEnv("REDIS_PORT", "6379")

	if host == "" {
		return nil
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed on %s:%s: %v", host, port, err)
		return err
	}

	log.Printf("Connected to Redis on %s:%s", host, port)
	return nil
}

// DomainEvent represents canonical StatGate Event Envelope
type DomainEvent struct {
	ID          string                 `json:"id"`
	EventType   string                 `json:"event_type"`
	Source      string                 `json:"source"`
	ObjectType  string                 `json:"object_type"`
	ObjectID    string                 `json:"object_id"`
	Actor       string                 `json:"actor"`
	TenantID    string                 `json:"tenant_id"`
	OrgID       string                 `json:"organization_id,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   string                 `json:"timestamp"`
	Correlation string                 `json:"correlation_id,omitempty"`
}

func publishEvent(eventType, objectType, objectID, actor, tenantID string, payload map[string]interface{}) {
	if redisClient == nil {
		return
	}

	eventID := fmt.Sprintf("evt_gov_%d", time.Now().UnixNano())
	if actor == "" {
		actor = "governance-system"
	}
	if tenantID == "" {
		tenantID = "tenant-alpha"
	}

	event := DomainEvent{
		ID:          eventID,
		EventType:   eventType,
		Source:      "statgovernance",
		ObjectType:  objectType,
		ObjectID:    objectID,
		Actor:       actor,
		TenantID:    tenantID,
		Payload:     payload,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Correlation: fmt.Sprintf("corr_%d", time.Now().UnixNano()),
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshalling event %s: %v", eventType, err)
		return
	}

	channel := getEnv("STATGATE_EVENT_CHANNEL", "statgate:events")
	ctx := context.Background()
	_ = redisClient.Publish(ctx, channel, string(data)).Err()
	// Store in recent history
	_ = redisClient.LPush(ctx, "statgate:event:history", string(data)).Err()
	_ = redisClient.LTrim(ctx, "statgate:event:history", 0, 999).Err()
}
