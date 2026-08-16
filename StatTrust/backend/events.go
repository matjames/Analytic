package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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
		log.Printf("Warning: Redis connection failed on %s:%s (continuing with in-memory bus): %v", host, port, err)
		return err
	}

	log.Printf("Connected to Redis event bus on %s:%s", host, port)
	go startEventListener()
	return nil
}

// EnterpriseEvent represents the universal message schema across StatGate.
type EnterpriseEvent struct {
	ID          string                 `json:"id"`
	EventType   string                 `json:"event_type"`
	Source      string                 `json:"source"`
	ObjectType  string                 `json:"object_type"`
	ObjectID    string                 `json:"object_id"`
	Actor       string                 `json:"actor"`
	TenantID    string                 `json:"tenant_id"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   string                 `json:"timestamp"`
	Correlation string                 `json:"correlation_id,omitempty"`
}

func publishEvent(eventType, objectType, objectID, actor, tenantID string, payload map[string]interface{}) {
	if redisClient == nil {
		return
	}

	eventID := fmt.Sprintf("evt_trust_%d", time.Now().UnixNano())
	if actor == "" {
		actor = "statgate-trust-engine"
	}
	if tenantID == "" {
		tenantID = "tenant-alpha"
	}

	event := EnterpriseEvent{
		ID:          eventID,
		EventType:   eventType,
		Source:      "stattrust",
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
	_ = redisClient.LPush(ctx, "statgate:event:history", string(data)).Err()
	_ = redisClient.LTrim(ctx, "statgate:event:history", 0, 999).Err()
}

func startEventListener() {
	if redisClient == nil {
		return
	}
	channel := getEnv("STATGATE_EVENT_CHANNEL", "statgate:events")
	pubsub := redisClient.Subscribe(context.Background(), channel)
	ch := pubsub.Channel()

	log.Printf("StatTrust listening for enterprise events on channel: %s", channel)

	for msg := range ch {
		var evt EnterpriseEvent
		if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
			continue
		}

		// Don't loop over own events
		if evt.Source == "stattrust" {
			continue
		}

		// Process cross-app events:
		// 1. High-integrity mutations are automatically notarized into the Immutable Audit Ledger
		if strings.HasSuffix(evt.EventType, ".created") || strings.HasSuffix(evt.EventType, ".approved") || strings.HasSuffix(evt.EventType, ".published") {
			if globalStore != nil {
				globalStore.AppendLedger(evt.EventType, evt.Source, evt.Actor, evt.Payload)
			}
		}

		// 2. Perform automated DLP & Anomaly inspection on incoming payloads
		inspectEventForThreats(evt)
	}
}

func inspectEventForThreats(evt EnterpriseEvent) {
	// Simple pattern scanning across event payloads
	b, _ := json.Marshal(evt.Payload)
	raw := string(b)

	// Check for secret keys or unprotected credentials
	if strings.Contains(strings.ToLower(raw), "password") || strings.Contains(strings.ToLower(raw), "secret_key") || strings.Contains(strings.ToLower(raw), "bearer ") {
		inc := SecurityIncident{
			ID:             fmt.Sprintf("INC-%d", time.Now().Unix()),
			Title:          fmt.Sprintf("Plaintext Token Detected in Event Stream from %s", evt.Source),
			Severity:       "HIGH",
			Status:         "OPEN",
			ThreatCategory: "Credential Leakage / DLP",
			SourceApp:      evt.Source,
			SourceIP:       "internal-event-bus",
			AffectedAsset:  evt.EventType,
			AssignedTo:     "Automated SecOps Guard",
			Details: map[string]interface{}{
				"event_id":   evt.ID,
				"actor":      evt.Actor,
				"rule_fired": "SECRET_IN_EVENT_PAYLOAD",
			},
			RemediationLog: []string{"Flagged for audit review", "Alert broadcasted to SecOps"},
			CreatedAt:      time.Now().UTC(),
		}
		if globalStore != nil {
			globalStore.AddIncident(inc)
		}
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
