package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func initRedis() error {
	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPass := getEnv("REDIS_PASSWORD", "")

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPass,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed (using in-memory event bus): %v", err)
		return err
	}

	log.Println("Connected to Redis for StatOps Event Bus")
	rdb = client

	// Start background subscriber
	go startEventSubscriber()
	return nil
}

func publishEvent(eventType, entityType, entityID, actorID, tenantID string, payload map[string]interface{}) {
	event := map[string]interface{}{
		"id":          fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		"type":        eventType,
		"entity_type": entityType,
		"entity_id":   entityID,
		"actor_id":    actorID,
		"tenant_id":   tenantID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"payload":     payload,
		"service":     "StatOps",
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	if rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rdb.Publish(ctx, "statgate:events", data).Err(); err != nil {
			log.Printf("Failed to publish event to Redis: %v", err)
		}
	}
}

func startEventSubscriber() {
	if rdb == nil {
		return
	}

	pubsub := rdb.Subscribe(context.Background(), "statgate:events")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &evt); err == nil {
			eventType, _ := evt["type"].(string)
			source, _ := evt["service"].(string)
			if eventType != "" && source != "StatOps" {
				// Record event in centralized logs for observability
				logEntry := CentralizedLog{
					ID:        fmt.Sprintf("log_%d", time.Now().UnixNano()),
					Timestamp: time.Now().UTC(),
					Level:     "INFO",
					SourceApp: source,
					Message:   fmt.Sprintf("Platform event received: %s from %s", eventType, source),
					Metadata:  evt,
				}
				if globalStore != nil {
					globalStore.mu.Lock()
					globalStore.logs = append(globalStore.logs, logEntry)
					if len(globalStore.logs) > 200 {
						globalStore.logs = globalStore.logs[len(globalStore.logs)-200:]
					}
					globalStore.mu.Unlock()
				}
			}
		}
	}
}
