package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	platformevents "github.com/matjames/statgate-lib/events"
)

var trustEventBus *platformevents.EventBus

func initRedis() error {
	host := getEnv("REDIS_HOST", "redis")
	port := getEnv("REDIS_PORT", "6379")

	if host == "" {
		return nil
	}

	bus, err := platformevents.NewEventBus(platformevents.Config{
		RedisAddr: host + ":" + port,
		Source:    "stattrust",
		Group:     "stattrust",
		Consumer:  "runtime",
	})
	if err != nil {
		log.Printf("Warning: shared Redis event bus initialization failed: %v", err)
		return err
	}
	if !bus.RedisAvailable(context.Background()) {
		log.Printf("Warning: shared Redis event bus is unavailable on %s:%s", host, port)
		return fmt.Errorf("shared Redis event bus unavailable on %s:%s", host, port)
	}

	trustEventBus = bus
	log.Printf("Connected to Redis event bus on %s:%s", host, port)
	go startEventListener()
	return nil
}

func publishEvent(eventType, objectType, objectID, actor, tenantID string, payload map[string]interface{}) {
	if trustEventBus == nil {
		return
	}

	eventID := fmt.Sprintf("evt_trust_%d", time.Now().UnixNano())
	if actor == "" {
		actor = "statgate-trust-engine"
	}
	if tenantID == "" {
		tenantID = "tenant-alpha"
	}

	event := platformevents.EnterpriseEvent{
		EventID:       eventID,
		EventType:     eventType,
		Source:        "stattrust",
		ObjectType:    objectType,
		ObjectID:      objectID,
		UserID:        actor,
		TenantID:      tenantID,
		Payload:       payload,
		CorrelationID: fmt.Sprintf("corr_%d", time.Now().UnixNano()),
		Timestamp:     time.Now().UTC(),
		Version:       "1.0",
	}

	if err := trustEventBus.PublishDurable(context.Background(), event); err != nil {
		log.Printf("Error publishing durable event %s: %v", eventType, err)
	}
}

func startEventListener() {
	if trustEventBus == nil {
		return
	}

	log.Printf("StatTrust listening for durable enterprise events in group stattrust")
	if err := trustEventBus.SubscribeDurable(context.Background(), "stattrust", "runtime", func(ctx context.Context, evt platformevents.EnterpriseEvent) error {
		// Don't loop over own events.
		if evt.Source == "stattrust" {
			return nil
		}

		// High-integrity mutations are automatically notarized into the ledger.
		if strings.HasSuffix(evt.EventType, ".created") || strings.HasSuffix(evt.EventType, ".approved") || strings.HasSuffix(evt.EventType, ".published") {
			if globalStore != nil {
				globalStore.AppendLedger(evt.EventType, evt.Source, evt.UserID, evt.Payload)
			}
		}

		inspectEventForThreats(evt)
		return nil
	}); err != nil {
		log.Printf("StatTrust durable event listener failed: %v", err)
	}
}

func inspectEventForThreats(evt platformevents.EnterpriseEvent) {
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
				"event_id":   evt.EventID,
				"actor":      evt.UserID,
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
