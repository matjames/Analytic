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

var redisClient *redis.Client

func getEnvValue(key string) string {
	return os.Getenv(key)
}

func initRedis() {
	host := getEnvValue("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := getEnvValue("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: getEnvValue("REDIS_PASSWORD"),
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("events: Redis unavailable at %s:%s - %v", host, port, err)
		redisClient = nil
		return
	}
	log.Printf("events: Redis connected at %s:%s", host, port)
}

// publishEvent publishes a domain event to the enterprise event bus.
func publishEvent(ev DomainEvent) {
	if ev.ID == "" {
		ev.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	if ev.Timestamp == "" {
		ev.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if ev.TenantID == "" {
		ev.TenantID = getEnvValue("STATGATE_TENANT_ID")
		if ev.TenantID == "" {
			ev.TenantID = "statgate"
		}
	}

	// Phase X: Persist BEFORE publish — Redis outage cannot destroy events.
	persistEvent(ev)

	// Process locally first so the request has deterministic, immediately
	// visible effects. Redis remains the delivery mechanism for other service
	// instances; the shared idempotency key prevents them repeating the work.
	// This also preserves core behaviour during a broker outage.
	processKnowledgeForEvent(ev)
	processEventIdempotent(ev)
	if redisClient == nil {
		// Redis unavailable: event is already persisted, will be replayed later
		return
	}
	data, _ := json.Marshal(ev)
	ctx := context.Background()
	_ = redisClient.Publish(ctx, EventChannel, string(data)).Err()
	// Store event history with correlation support (7 day retention)
	storeEventWithCorrelation(ev)
}

// consumeEvents listens on the event bus and derives downstream artifacts.
func consumeEvents() {
	if redisClient == nil {
		log.Println("events: Redis not available; consumer disabled")
		return
	}
	ctx := context.Background()
	pubsub := redisClient.Subscribe(ctx, EventChannel)
	defer pubsub.Close()
	log.Printf("events: consuming on %s", EventChannel)

	for msg := range pubsub.Channel() {
		var ev DomainEvent
		if err := json.Unmarshal([]byte(msg.Payload), &ev); err != nil {
			continue
		}
		// Use idempotent processing to prevent duplicate side effects
		processEventIdempotent(ev)
		// Track consumer state for recovery monitoring
		updateConsumerState()
	}
}
