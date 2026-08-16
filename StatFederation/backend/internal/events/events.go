package events

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/matjames/statgate-lib/events"
	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// Federation specific event types
const (
	EventNodeRegistered       = "federation.node.registered"
	EventNodeHeartbeat        = "federation.node.heartbeat"
	EventDSAApproved          = "federation.dsa.approved"
	EventDSARevoked           = "federation.dsa.revoked"
	EventQueryDispatched      = "federation.query.dispatched"
	EventQueryCompleted       = "federation.query.completed"
	EventDiplomacyReportSent  = "diplomacy.report.transmitted"
	EventComplianceViolation  = "diplomacy.compliance.violation"
)

// EventWorker coordinates event broadcasting, consumption, and DLQ handling
type EventWorker struct {
	bus     *events.EventBus
	store   store.Store
	nodeID  string
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewEventWorker creates the event worker
func NewEventWorker(bus *events.EventBus, s store.Store, nodeID string) *EventWorker {
	return &EventWorker{
		bus:    bus,
		store:  s,
		nodeID: nodeID,
		stopCh: make(chan struct{}),
	}
}

// PublishFederationEvent publishes standard EnterpriseEvent
func (ew *EventWorker) PublishFederationEvent(
	ctx context.Context,
	eventType, objectType, objectID, tenantID, userID string,
	payload map[string]interface{},
) error {
	evt := events.EnterpriseEvent{
		EventID:       "evt-" + uuid.New().String()[:8],
		EventType:     eventType,
		Source:        "statgate-federation",
		ObjectType:    objectType,
		ObjectID:      objectID,
		TenantID:      tenantID,
		UserID:        userID,
		CorrelationID: uuid.New().String(),
		Payload:       payload,
		Timestamp:     time.Now().UTC(),
		Version:       "1.0",
	}

	return ew.bus.Publish(ctx, evt)
}

// StartEventListener begins consuming platform events (dataset, survey, research) for cross-agency synchronization
func (ew *EventWorker) StartEventListener(ctx context.Context) {
	log.Println("[EventWorker] Subscribing to platform events on channel: statgate:events")

	err := ew.bus.Subscribe(ctx, func(handlerCtx context.Context, evt events.EnterpriseEvent) error {
		ew.processIncomingEvent(handlerCtx, evt)
		return nil
	})
	if err != nil {
		log.Printf("[EventWorker] Subscribe warning: %v", err)
	}
}

func (ew *EventWorker) processIncomingEvent(ctx context.Context, evt events.EnterpriseEvent) {
	log.Printf("[EventWorker] Received event %s (%s) for %s:%s", evt.EventType, evt.EventID, evt.ObjectType, evt.ObjectID)

	switch evt.EventType {
	case events.EventDatasetCreated, events.EventDatasetUpdated:
		// Automatically index newly published dataset into federated search index
		title, _ := evt.Payload["title"].(string)
		if title == "" {
			title = "Platform Dataset: " + evt.ObjectID
		}
		_ = ew.store.IndexRemoteResource(ctx, &models.FederatedSearchResult{
			NodeID:           ew.nodeID,
			NodeName:         "StatGate National Hub",
			ResourceType:     "DATASET",
			RemoteResourceID: evt.ObjectID,
			Title:            title,
			Abstract:         "Discovered from enterprise data fabric events.",
			Keywords:         []string{"Enterprise", "Dataset", evt.ObjectType},
			Classification:   "INTERNAL",
			Score:            0.9,
		}, evt.TenantID)

	case events.EventSurveyPublished:
		log.Printf("[EventWorker] Survey %s published. Updating NSS Official Statistics Calendar.", evt.ObjectID)

	case EventComplianceViolation:
		log.Printf("[EventWorker:ALERT] Transboundary compliance violation recorded for object %s!", evt.ObjectID)
	}
}

// Stop gracefully shuts down event workers
func (ew *EventWorker) Stop() {
	// Bus cleanup handled via context cancellation
}
