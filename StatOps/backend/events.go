package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/matjames/statgate-lib/events"
	"github.com/redis/go-redis/v9"
)

var eventBus *events.EventBus
var legacyRDB *redis.Client

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func initRedis() error {
	bus, err := events.InitFromEnv("statops")
	if err != nil {
		return err
	}
	eventBus = bus

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if !bus.RedisAvailable(ctx) {
		log.Println("Redis unavailable; StatOps durable consumer is waiting for the next process restart")
		return nil
	}

	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	legacyRDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	log.Println("Connected to Redis for StatOps durable Event Bus")

	// The durable stream is authoritative. The compatibility subscriber only
	// accepts the pre-canonical envelope used by older StatGate publishers.
	go startEventSubscriber()
	go startLegacyEventSubscriber()
	return nil
}

func publishEvent(eventType, entityType, entityID, actorID, tenantID string, payload map[string]interface{}) {
	if eventBus == nil {
		return
	}
	if payload == nil {
		payload = make(map[string]interface{})
	}
	event := events.EnterpriseEvent{
		EventType:  eventType,
		Source:     "statops",
		ObjectType: entityType,
		ObjectID:   entityID,
		UserID:     actorID,
		TenantID:   tenantID,
		Payload:    payload,
		Timestamp:  time.Now().UTC(),
		Version:    "1.0",
	}
	if err := eventBus.PublishDurable(context.Background(), event); err != nil {
		log.Printf("Failed to publish durable event %s: %v", eventType, err)
	}
}

func startEventSubscriber() {
	if eventBus == nil {
		return
	}
	consumer := getEnv("STATGATE_EVENT_CONSUMER", "statops-runtime")
	if err := eventBus.SubscribeDurable(context.Background(), "statops", consumer, handleCanonicalEvent); err != nil {
		log.Printf("Failed to start StatOps durable event subscriber: %v", err)
	}
}

func handleCanonicalEvent(_ context.Context, evt events.EnterpriseEvent) error {
	if strings.EqualFold(evt.Source, "statops") || evt.EventType == "" {
		return nil
	}

	metadata := map[string]interface{}{
		"event_id":       evt.EventID,
		"event_type":     evt.EventType,
		"source":         evt.Source,
		"object_type":    evt.ObjectType,
		"object_id":      evt.ObjectID,
		"tenant_id":      evt.TenantID,
		"user_id":        evt.UserID,
		"correlation_id": evt.CorrelationID,
		"timestamp":      evt.Timestamp,
		"version":        evt.Version,
		"payload":        evt.Payload,
	}
	appendEventLog(evt.EventID, evt.EventType, evt.Source, evt.TenantID, workspaceFromPayload(evt.Payload), metadata)
	return nil
}

// startLegacyEventSubscriber keeps older non-canonical publishers visible while
// they are migrated. Canonical events are ignored here to avoid double logging.
func startLegacyEventSubscriber() {
	if legacyRDB == nil {
		return
	}

	pubsub := legacyRDB.Subscribe(context.Background(), "statgate:events")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &evt); err == nil {
			if _, canonical := evt["event_type"]; canonical {
				continue
			}
			eventType, _ := evt["type"].(string)
			source, _ := evt["service"].(string)
			if eventType != "" && !strings.EqualFold(source, "StatOps") {
				tenantID, _ := evt["tenant_id"].(string)
				workspaceID := ""
				if payload, ok := evt["payload"].(map[string]interface{}); ok {
					workspaceID = workspaceFromPayload(payload)
				}
				appendEventLog(fmt.Sprintf("legacy-%d", time.Now().UnixNano()), eventType, source, tenantID, workspaceID, evt)
			}
		}
	}
}

func workspaceFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	if workspaceID, ok := payload["workspace_id"].(string); ok {
		return workspaceID
	}
	if workspaceID, ok := payload["workspaceId"].(string); ok {
		return workspaceID
	}
	return ""
}

func appendEventLog(eventID, eventType, source, tenantID, workspaceID string, metadata map[string]interface{}) {
	if globalStore == nil {
		return
	}
	logEntry := CentralizedLog{
		ID:          fmt.Sprintf("log_%s", eventID),
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		Timestamp:   time.Now().UTC(),
		Level:       "INFO",
		SourceApp:   source,
		Message:     fmt.Sprintf("Platform event received: %s from %s", eventType, source),
		Metadata:    metadata,
	}
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	globalStore.logs = append(globalStore.logs, logEntry)
	if len(globalStore.logs) > 200 {
		globalStore.logs = globalStore.logs[len(globalStore.logs)-200:]
	}
}
