package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/matjames/statgate-lib/events"
	"statfederation-backend/internal/engine"
	"statfederation-backend/internal/store"
)

// fakeDispatcher records every outbound push it receives and reports the count.
type fakeDispatcher struct {
	mu     sync.Mutex
	called []engine.IndicatorPushRequest
}

func (f *fakeDispatcher) PushIndicators(_ context.Context, req engine.IndicatorPushRequest, _ string, _ string) (*engine.IndicatorPushResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = append(f.called, req)
	return &engine.IndicatorPushResult{
		PushID:         "push-test",
		SourceNodeID:   req.SourceNodeID,
		TargetNodeID:   req.TargetNodeID,
		IndicatorCount: len(req.IndicatorIDs),
		Status:         "SUCCEEDED",
		HTTPStatus:     200,
		PushedAt:       time.Now().UTC(),
	}, nil
}

func (f *fakeDispatcher) requests() []engine.IndicatorPushRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]engine.IndicatorPushRequest, len(f.called))
	copy(out, f.called)
	return out
}

// waitFor blocks until len() satisfies cond or the timeout elapses.
func waitFor(t *testing.T, timeout time.Duration, cond func(int) bool, getLen func() int) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond(getLen()) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for condition (len=%d)", getLen())
}

// TestAutoPushIndicatorComputed verifies that an Analytics "indicator.computed"
// event is routed to every node with an active DSA for the indicator's domain.
func TestAutoPushIndicatorComputed(t *testing.T) {
	mem := store.NewMemStore()
	disp := &fakeDispatcher{}
	consumer := NewAutoPushConsumer(mem, disp)

	ctx := context.Background()
	evt := events.EnterpriseEvent{
		EventID:   "evt-computed-001",
		EventType: EventIndicatorComputed,
		TenantID:  "default",
		Payload: map[string]interface{}{
			"indicator_id":   "ind-sdg-311",
			"domain":         "health",
			"source_node_id": "node-moh-002",
		},
	}

	if !consumer.HandleEnterpriseEvent(ctx, evt) {
		t.Fatal("expected HandleEnterpriseEvent to consume the event")
	}
	waitFor(t, 2*time.Second, func(n int) bool { return n >= 1 }, func() int { return len(disp.requests()) })

	reqs := disp.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected exactly 1 push, got %d", len(reqs))
	}
	r := reqs[0]
	if r.TargetNodeID != "node-nss-001" {
		t.Errorf("expected target node-nss-001 (DSA consumer), got %q", r.TargetNodeID)
	}
	if r.SourceNodeID != "node-moh-002" {
		t.Errorf("expected source node-moh-002, got %q", r.SourceNodeID)
	}
	if len(r.IndicatorIDs) != 1 || r.IndicatorIDs[0] != "ind-sdg-311" {
		t.Errorf("expected indicator ind-sdg-311, got %v", r.IndicatorIDs)
	}
	if r.Domain != "health" {
		t.Errorf("expected domain health, got %q", r.Domain)
	}
}

// TestAutoConsumerResolveByCode verifies the fallback of resolving an indicator
// by its SDMX code when only a code is supplied.
func TestAutoConsumerResolveByCode(t *testing.T) {
	mem := store.NewMemStore()
	disp := &fakeDispatcher{}
	consumer := NewAutoPushConsumer(mem, disp)

	ctx := context.Background()
	evt := events.EnterpriseEvent{
		EventID:   "evt-computed-002",
		EventType: EventIndicatorComputed,
		TenantID:  "default",
		Payload: map[string]interface{}{
			"indicator_code":   "IND-SDG-3.1.1",
			"indicator_domain": "health",
			"source_node_id":   "node-moh-002",
		},
	}
	if !consumer.HandleEnterpriseEvent(ctx, evt) {
		t.Fatal("expected event to be consumed")
	}
	waitFor(t, 2*time.Second, func(n int) bool { return n >= 1 }, func() int { return len(disp.requests()) })

	reqs := disp.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 push, got %d", len(reqs))
	}
	if len(reqs[0].IndicatorIDs) != 1 || reqs[0].IndicatorIDs[0] != "ind-sdg-311" {
		t.Errorf("expected code-resolved indicator ind-sdg-311, got %v", reqs[0].IndicatorIDs)
	}
}

// TestAutoConsumerNoDSA: without an active DSA between source and any peer,
// an indicator.computed event must not trigger any push.
func TestAutoConsumerNoDSA(t *testing.T) {
	mem := store.NewMemStore()
	disp := &fakeDispatcher{}
	consumer := NewAutoPushConsumer(mem, disp)

	// node-au-003 is the source; no active DSA authorises it to push to peers.
	evt := events.EnterpriseEvent{
		EventID:   "evt-computed-003",
		EventType: EventIndicatorComputed,
		TenantID:  "default",
		Payload: map[string]interface{}{
			"indicator_id":   "ind-sdg-311",
			"domain":         "health",
			"source_node_id": "node-au-003",
		},
	}
	if !consumer.HandleEnterpriseEvent(context.Background(), evt) {
		t.Fatal("expected event to be consumed")
	}

	// Give the queue a moment to drain; the resolved target list is empty.
	time.Sleep(150 * time.Millisecond)
	if got := len(disp.requests()); got != 0 {
		t.Fatalf("expected 0 pushes without an active DSA, got %d", got)
	}
}

// TestAutoConsumerPushRequested handles an explicit push.request event with a
// target node and a list of indicator IDs.
func TestAutoConsumerPushRequested(t *testing.T) {
	mem := store.NewMemStore()
	disp := &fakeDispatcher{}
	consumer := NewAutoPushConsumer(mem, disp)

	ctx := context.Background()
	evt := events.EnterpriseEvent{
		EventID:   "evt-req-004",
		EventType: EventIndicatorPushRequested,
		TenantID:  "default",
		Payload: map[string]interface{}{
			"source_node_id": "node-moh-002",
			"target_node_id": "node-nss-001",
			"indicator_ids":  []interface{}{"ind-sdg-311", "ind-2"},
			"domain":         "health",
		},
	}
	if !consumer.HandleEnterpriseEvent(ctx, evt) {
		t.Fatal("expected push.requested event to be consumed")
	}
	waitFor(t, 2*time.Second, func(n int) bool { return n >= 1 }, func() int { return len(disp.requests()) })

	reqs := disp.requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 push, got %d", len(reqs))
	}
	if reqs[0].TargetNodeID != "node-nss-001" {
		t.Errorf("expected target node-nss-001, got %q", reqs[0].TargetNodeID)
	}
	if len(reqs[0].IndicatorIDs) != 2 {
		t.Errorf("expected 2 indicators, got %v", reqs[0].IndicatorIDs)
	}
}

// TestAutoConsumerUnrelatedEvent confirms the consumer returns false for events
// it does not handle, leaving the worker switch open to other handlers.
func TestAutoConsumerUnrelatedEvent(t *testing.T) {
	mem := store.NewMemStore()
	consumer := NewAutoPushConsumer(mem, &fakeDispatcher{})

	evt := events.EnterpriseEvent{
		EventID:   "evt-other",
		EventType: "something.else",
	}
	if consumer.HandleEnterpriseEvent(context.Background(), evt) {
		t.Fatal("expected unrelated event to return false")
	}
}