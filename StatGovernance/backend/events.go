package main

import (
	"context"
	"fmt"
	"time"

	"github.com/matjames/statgate-lib/events"
)

var eventBus *events.EventBus

func initRedis() error {
	bus, err := events.InitFromEnv("statgovernance")
	if err != nil {
		return err
	}
	eventBus = bus
	return nil
}

func newGovernanceEvent(eventType, objectType, objectID, actor, tenantID string, payload map[string]interface{}) events.EnterpriseEvent {
	if actor == "" {
		actor = "governance-system"
	}
	if tenantID == "" {
		tenantID = "default"
	}
	return events.EnterpriseEvent{
		EventType:     eventType,
		Source:        "statgovernance",
		ObjectType:    objectType,
		ObjectID:      objectID,
		UserID:        actor,
		TenantID:      tenantID,
		Payload:       payload,
		Timestamp:     time.Now().UTC(),
		CorrelationID: fmt.Sprintf("corr_%d", time.Now().UnixNano()),
		Version:       "1.0",
	}
}

func publishEvent(eventType, objectType, objectID, actor, tenantID string, payload map[string]interface{}) {
	if eventBus == nil {
		return
	}
	_ = eventBus.Publish(context.Background(), newGovernanceEvent(eventType, objectType, objectID, actor, tenantID, payload))
}
