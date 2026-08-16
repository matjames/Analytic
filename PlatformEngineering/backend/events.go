package main

import (
	"context"
	"github.com/matjames/statgate-lib/events"
	"log"
)

func publish(bus *events.EventBus, eventType, objectType, objectID, tenantID string, payload map[string]any) {
	if bus == nil {
		return
	}
	err := bus.Publish(context.Background(), events.EnterpriseEvent{EventType: eventType, ObjectType: objectType, ObjectID: objectID, TenantID: tenantID, Payload: payload})
	if err != nil {
		log.Printf("RunOps event publish failed: %v", err)
	}
}
