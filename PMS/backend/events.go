package main

import (
	"context"
	"time"

	"github.com/matjames/statgate-lib/events"
)

// eventBus is the shared Enterprise Event Bus (statgate-lib). Redis-backed with
// an in-memory fallback so publishing is graceful when Redis is unavailable.
var eventBus *events.EventBus

func initRedis() error {
	bus, err := events.InitFromEnv("pms")
	if err != nil {
		return err
	}
	eventBus = bus
	return nil
}

func publishEvent(eventType, objectType, objectID string, payload map[string]interface{}) {
	if eventBus == nil {
		return
	}
	evt := events.EnterpriseEvent{
		EventType:  eventType,
		Source:     "pms",
		ObjectType: objectType,
		ObjectID:   objectID,
		TenantID:   getEnv("PMS_TENANT_ID", "tenant-alpha"),
		Payload:    payload,
		Timestamp:  time.Now().UTC(),
		Version:    "1.0",
	}
	_ = eventBus.Publish(context.Background(), evt)
}
