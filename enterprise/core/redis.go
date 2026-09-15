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

var redisClient *redis.Client
var eventBus *events.EventBus

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

	bus, err := events.InitFromEnv("enterprise-core")
	if err != nil {
		log.Printf("events: shared durable bus initialization failed: %v", err)
		return
	}
	eventBus = bus
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
	if eventBus != nil {
		ctx := context.Background()
		if err := eventBus.PublishDurable(ctx, enterpriseEventFromDomain(ev)); err != nil {
			log.Printf("events: durable publish failed for %s: %v", ev.ID, err)
		}
	} else if redisClient != nil {
		data, _ := json.Marshal(ev)
		_ = redisClient.Publish(context.Background(), EventChannel, string(data)).Err()
	}
	// Store event history with correlation support (7 day retention)
	storeEventWithCorrelation(ev)
}

// consumeEvents listens on the event bus and derives downstream artifacts.
func consumeEvents() {
	if eventBus == nil {
		log.Println("events: shared durable bus unavailable; consumer disabled")
		return
	}
	consumer := getEnvValue("STATGATE_EVENT_CONSUMER")
	if consumer == "" {
		consumer = "enterprise-core-runtime"
	}
	if err := eventBus.SubscribeDurable(context.Background(), "enterprise-core", consumer, handleDurableEvent); err != nil {
		log.Printf("events: durable consumer failed to start: %v", err)
		return
	}
	// Keep older publishers visible while they migrate to the canonical envelope.
	go consumeLegacyEvents()
	log.Printf("events: consuming durable stream as %s", consumer)
}

func handleDurableEvent(_ context.Context, evt events.EnterpriseEvent) error {
	if strings.EqualFold(evt.Source, "enterprise-core") {
		return nil
	}
	processEventIdempotent(domainEventFromEnterprise(evt))
	updateConsumerState()
	return nil
}

func consumeLegacyEvents() {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	pubsub := redisClient.Subscribe(ctx, EventChannel)
	defer pubsub.Close()
	log.Printf("events: consuming on %s", EventChannel)

	for msg := range pubsub.Channel() {
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Payload), &raw); err != nil {
			continue
		}
		// PublishDurable emits the same event on Pub/Sub for low latency. The
		// durable consumer owns canonical envelopes, so do not process twice.
		if _, canonical := raw["event_id"]; canonical {
			continue
		}
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

func enterpriseEventFromDomain(ev DomainEvent) events.EnterpriseEvent {
	timestamp, err := time.Parse(time.RFC3339, ev.Timestamp)
	if err != nil {
		timestamp = time.Now().UTC()
	}
	return events.EnterpriseEvent{
		EventID:       ev.ID,
		EventType:     ev.EventType,
		Source:        ev.Source,
		ObjectType:    ev.ObjectType,
		ObjectID:      ev.ObjectID,
		TenantID:      ev.TenantID,
		UserID:        ev.Actor,
		CorrelationID: ev.Correlation,
		Payload:       ev.Payload,
		Timestamp:     timestamp,
		Version:       "1.0",
	}
}

func domainEventFromEnterprise(evt events.EnterpriseEvent) DomainEvent {
	timestamp := evt.Timestamp.UTC().Format(time.RFC3339)
	if evt.Timestamp.IsZero() {
		timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return DomainEvent{
		ID:          evt.EventID,
		EventType:   evt.EventType,
		Source:      evt.Source,
		ObjectType:  evt.ObjectType,
		ObjectID:    evt.ObjectID,
		Actor:       evt.UserID,
		TenantID:    evt.TenantID,
		Payload:     evt.Payload,
		Timestamp:   timestamp,
		Correlation: evt.CorrelationID,
	}
}
