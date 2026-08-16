package events

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/matjames/statgate-lib/events"
	"statiot-backend/internal/models"
	"statiot-backend/internal/store"
)

type EventWorker struct {
	eventBus *events.EventBus
	store    store.Store
	source   string
	stopChan chan struct{}
}

func NewEventWorker(bus *events.EventBus, st store.Store) *EventWorker {
	return &EventWorker{
		eventBus: bus,
		store:    st,
		source:   "statiot-backend",
		stopChan: make(chan struct{}),
	}
}

// PublishDeviceAlert broadcasts an alert to the platform and registers object links.
func (w *EventWorker) PublishDeviceAlert(ctx context.Context, alert *models.IoTAlert) error {
	evt := events.EnterpriseEvent{
		EventID:    uuid.New().String(),
		EventType:  events.EventAnomalyDetected,
		Source:     w.source,
		ObjectType: "iot_device",
		ObjectID:   alert.DeviceID,
		TenantID:   alert.TenantID,
		Payload: map[string]interface{}{
			"alert_id":    alert.ID,
			"alert_type":  alert.AlertType,
			"severity":    alert.Severity,
			"message":     alert.Message,
			"value":       alert.Value,
			"threshold":   alert.Threshold,
			"sensor_code": alert.SensorCode,
		},
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}

	if w.eventBus != nil {
		_ = w.eventBus.Publish(ctx, evt)
	}

	// Register in object links
	_ = w.store.CreateObjectLink(ctx, &models.ObjectLink{
		SourceType:   "iot_device",
		SourceID:     alert.DeviceID,
		TargetType:   "alert",
		TargetID:     alert.ID,
		Relationship: "generated_alert",
		TenantID:     alert.TenantID,
		CreatedBy:    "statiot-gateway",
	})

	return nil
}

// PublishTelemetryIngested notifies the platform of high-throughput sensor updates.
func (w *EventWorker) PublishTelemetryIngested(ctx context.Context, deviceID string, recordCount int) error {
	evt := events.EnterpriseEvent{
		EventID:    uuid.New().String(),
		EventType:  events.EventDatasetUpdated,
		Source:     w.source,
		ObjectType: "iot_device",
		ObjectID:   deviceID,
		TenantID:   "default",
		Payload: map[string]interface{}{
			"device_id":    deviceID,
			"record_count": recordCount,
			"ingested_at":  time.Now().UTC().Format(time.RFC3339),
		},
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}

	if w.eventBus != nil {
		return w.eventBus.Publish(ctx, evt)
	}
	return nil
}

// PublishFieldSubmission publishes mobile field form submissions.
func (w *EventWorker) PublishFieldSubmission(ctx context.Context, sub *models.MobileSubmission) error {
	evt := events.EnterpriseEvent{
		EventID:    uuid.New().String(),
		EventType:  events.EventFieldDataSubmitted,
		Source:     w.source,
		ObjectType: "mobile_submission",
		ObjectID:   sub.ID,
		TenantID:   sub.TenantID,
		Payload: map[string]interface{}{
			"client_submission_id": sub.ClientSubmissionID,
			"form_id":              sub.FormID,
			"worker_id":            sub.WorkerID,
			"visit_id":             sub.VisitID,
			"collected_at":         sub.CollectedAt.Format(time.RFC3339),
		},
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}

	if w.eventBus != nil {
		_ = w.eventBus.Publish(ctx, evt)
	}

	// Register in object links
	_ = w.store.CreateObjectLink(ctx, &models.ObjectLink{
		SourceType:   "field_worker",
		SourceID:     sub.WorkerID,
		TargetType:   "mobile_submission",
		TargetID:     sub.ID,
		Relationship: "submitted_by",
		TenantID:     sub.TenantID,
		CreatedBy:    "statiot-sync",
	})

	return nil
}

// PublishSyncCompleted broadcasts successful synchronization transactions.
func (w *EventWorker) PublishSyncCompleted(ctx context.Context, workerID, deviceUUID string, recordsSynced int) error {
	evt := events.EnterpriseEvent{
		EventID:    uuid.New().String(),
		EventType:  "fieldops.sync.completed",
		Source:     w.source,
		ObjectType: "mobile_device",
		ObjectID:   deviceUUID,
		TenantID:   "default",
		Payload: map[string]interface{}{
			"worker_id":      workerID,
			"device_uuid":    deviceUUID,
			"records_synced": recordsSynced,
			"synced_at":      time.Now().UTC().Format(time.RFC3339),
		},
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}

	if w.eventBus != nil {
		return w.eventBus.Publish(ctx, evt)
	}
	return nil
}

// PublishConflictDetected alerts supervisors when offline data conflict occurs.
func (w *EventWorker) PublishConflictDetected(ctx context.Context, conflict *models.ConflictRecord) error {
	evt := events.EnterpriseEvent{
		EventID:    uuid.New().String(),
		EventType:  "fieldops.conflict.detected",
		Source:     w.source,
		ObjectType: "conflict_record",
		ObjectID:   conflict.ID,
		TenantID:   conflict.TenantID,
		Payload: map[string]interface{}{
			"conflict_id":    conflict.ID,
			"entity_type":    conflict.EntityType,
			"entity_id":      conflict.EntityID,
			"client_version": conflict.ClientVersion,
			"server_version": conflict.ServerVersion,
			"strategy":       conflict.ResolutionStrategy,
		},
		Timestamp: time.Now().UTC(),
		Version:   "1.0",
	}

	if w.eventBus != nil {
		return w.eventBus.Publish(ctx, evt)
	}
	return nil
}

// StartEventListener listens to cross-platform messages from the Redis bus.
func (w *EventWorker) StartEventListener(ctx context.Context) {
	if w.eventBus == nil {
		return
	}

	_ = w.eventBus.Subscribe(ctx, func(c context.Context, evt events.EnterpriseEvent) error {
		log.Printf("[EventWorker] StatIoT received event %s (%s) on object %s:%s",
			evt.EventType, evt.EventID, evt.ObjectType, evt.ObjectID)

		switch evt.EventType {
		case events.EventFacilityCreated:
			// Auto-link facility to field visit domain
			log.Printf("[EventWorker] New facility created: %s, updating field ops registry", evt.ObjectID)
		case events.EventSurveyPublished:
			// Sync new survey as active mobile form
			log.Printf("[EventWorker] New survey published: %s, provisioning mobile form definition", evt.ObjectID)
		}
		return nil
	})
}

func (w *EventWorker) Stop() {
	close(w.stopChan)
}
