package server

import (
	"context"
	"testing"
	"time"

	"github.com/matjames/statgate-lib/events"
)

func TestStatCollectPublishesSharedTenantScopedEventEnvelope(t *testing.T) {
	previousConfig := cfg
	previousBus := eventBus
	defer func() {
		cfg = previousConfig
		eventBus = previousBus
	}()

	cfg = &Config{TenantID: "tenant-events"}
	consumer, err := events.NewEventBus(events.Config{Source: "test-consumer"})
	if err != nil {
		t.Fatalf("create consumer bus: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	received := make(chan events.EnterpriseEvent, 1)
	if err := consumer.Subscribe(ctx, func(_ context.Context, event events.EnterpriseEvent) error {
		received <- event
		return nil
	}); err != nil {
		t.Fatalf("subscribe consumer bus: %v", err)
	}
	defer consumer.Close()

	InitEventBus("", "", true).Publish(EventSubmissionReceived, "submission", "instance-events", map[string]interface{}{
		"form_id": "form-events",
	})

	select {
	case event := <-received:
		if event.EventType != EventSubmissionReceived || event.Source != "statcollect" || event.ObjectType != "submission" || event.ObjectID != "instance-events" || event.TenantID != "tenant-events" || event.Version != "1.0" || event.EventID == "" {
			t.Fatalf("unexpected shared event envelope: %+v", event)
		}
		if event.Payload["form_id"] != "form-events" {
			t.Fatalf("expected flattened submission payload, got %+v", event.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shared event")
	}
}
