package events

import (
	"context"
	"testing"
	"time"
)

// TestInMemoryCrossAppCommunication proves the core communication guarantee:
// two independent app EventBus instances (simulating distinct StatGate apps)
// exchange domain events over the canonical channel — without Redis, using
// the in-process broker. This is the same mechanism Redis provides in
// production deployments.
func TestInMemoryCrossAppCommunication(t *testing.T) {
	ctx := context.Background()

	appPublishing := "knowledge-portal"
	appConsuming := "enterprise-core"

	producer, err := NewEventBus(Config{Source: appPublishing, Channel: DefaultChannel})
	if err != nil {
		t.Fatalf("producer bus: %v", err)
	}
	consumer, err := NewEventBus(Config{Source: appConsuming, Channel: DefaultChannel})
	if err != nil {
		t.Fatalf("consumer bus: %v", err)
	}

	received := make(chan EnterpriseEvent, 8)
	if err := consumer.Subscribe(ctx, func(_ context.Context, evt EnterpriseEvent) error {
		received <- evt
		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Give the in-memory subscription a moment to register.
	time.Sleep(50 * time.Millisecond)

	// App A (knowledge-portal) publishes a domain event on the shared channel.
	if err := producer.Publish(ctx, EnterpriseEvent{
		EventType:  "dataset.published",
		Source:     appPublishing,
		ObjectType: "dataset",
		ObjectID:   "ds-ug-001",
		TenantID:   "uganda-national",
		Payload:    map[string]interface{}{"title": "Malaria Surveillance 2026"},
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case evt := <-received:
		if evt.EventType == "" {
			t.Errorf("event_type lost during transport")
		}
		if evt.Source != appPublishing {
			t.Errorf("expected source %q, got %q", appPublishing, evt.Source)
		}
		if evt.ObjectID != "ds-ug-001" {
			t.Errorf("expected object_id ds-ug-001, got %q", evt.ObjectID)
		}
		if evt.TenantID != "uganda-national" {
			t.Errorf("tenant context lost: %q", evt.TenantID)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("consumer app never received the event published by producer app (cross-app broker failure)")
	}

	// Reverse direction: consumer → producer (both directions must work).
	producerReceived := make(chan EnterpriseEvent, 8)
	if err := producer.Subscribe(ctx, func(_ context.Context, evt EnterpriseEvent) error {
		producerReceived <- evt
		return nil
	}); err != nil {
		t.Fatalf("producer subscribe: %v", err)
	}
	time.Sleep(25 * time.Millisecond)

	if err := consumer.Publish(ctx, EnterpriseEvent{
		EventType:  "feedback.received",
		Source:     appConsuming,
		ObjectType: "feedback",
		ObjectID:   "fb-001",
		TenantID:   "uganda-national",
	}); err != nil {
		t.Fatalf("reverse publish: %v", err)
	}

	select {
	case evt := <-producerReceived:
		if evt.EventType != "feedback.received" {
			t.Errorf("expected feedback.received, got %q", evt.EventType)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("producer app never received the event published by consumer app")
	}
}

// TestDefaultChannelIsCanonical guards the single-source-of-truth channel name.
func TestDefaultChannelIsCanonical(t *testing.T) {
	if DefaultChannel != "statgate:events" {
		t.Fatalf("canonical event channel drifted: %q", DefaultChannel)
	}
	if DLQChannel != "statgate:events:dlq" {
		t.Fatalf("canonical DLQ channel drifted: %q", DLQChannel)
	}
}