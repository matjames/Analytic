package events

import (
	"context"
	"log"
	"sync"

	"github.com/matjames/statgate-lib/events"
	"statfederation-backend/internal/engine"
	"statfederation-backend/internal/store"
)

// Event types consumed by the AutoPushConsumer. The first is emitted by
// Analytics Core whenever an indicator value is recomputed; the second is the
// explicit cross-hub push request. Task 1.4 wires these so that approved
// indicator values are automatically transmitted to nodes with an active DSA.
const (
	// EventIndicatorComputed is the enterprise event Analytics Core emits when a
	// NationalIndicator value has been recomputed.
	EventIndicatorComputed = "indicator.computed"
	// EventIndicatorPushRequested requests an explicit outbound federation push
	// to one or more target hubs.
	EventIndicatorPushRequested = "federation.indicator.push.requested"
)

// autoPushQueueSize bounds the in-memory auto-push work queue.
const autoPushQueueSize = 256

// PushDispatcher executes outbound indicator pushes. It is implemented by
// engine.PushProtocol and is extracted as an interface so the consumer can be
// tested in isolation (without a live peer hub).
type PushDispatcher interface {
	PushIndicators(ctx context.Context, req engine.IndicatorPushRequest, senderName, tenantID string) (*engine.IndicatorPushResult, error)
}

// pushJob is a single queued outbound indicator-push work item.
type pushJob struct {
	sourceNodeID string
	targetNodeID string
	indicatorIDs []string
	domain       string
	senderName   string
	tenantID     string
}

// AutoPushConsumer watches the enterprise event bus and automatically queues
// approved indicator values for transmission to peer hubs covered by an active
// Data Sharing Agreement (DSA).
type AutoPushConsumer struct {
	store      store.Store
	dispatcher PushDispatcher
	queue      chan pushJob
	startOnce  sync.Once
}

// NewAutoPushConsumer constructs the auto-push consumer.
func NewAutoPushConsumer(s store.Store, d PushDispatcher) *AutoPushConsumer {
	return &AutoPushConsumer{
		store:      s,
		dispatcher: d,
		queue:      make(chan pushJob, autoPushQueueSize),
	}
}

// Start launches the background queue worker. Safe to call more than once.
func (a *AutoPushConsumer) Start(ctx context.Context) {
	a.startOnce.Do(func() {
		go a.run(ctx)
		log.Println("[AutoPushConsumer] queue worker started")
	})
}

// HandleEnterpriseEvent routes an incoming bus event to the correct handler.
// It returns true if the event was consumed by this consumer.
func (a *AutoPushConsumer) HandleEnterpriseEvent(ctx context.Context, evt events.EnterpriseEvent) bool {
	switch evt.EventType {
	case EventIndicatorComputed:
		a.OnIndicatorComputed(ctx, evt)
		return true
	case EventIndicatorPushRequested:
		a.OnIndicatorPushRequested(ctx, evt)
		return true
	default:
		return false
	}
}

// OnIndicatorComputed handles an Analytics "indicator.computed" event. It
// resolves the source node, discovers every peer hub authorised by an active
// DSA for the indicator's domain, and enqueues a push to each of them.
func (a *AutoPushConsumer) OnIndicatorComputed(ctx context.Context, evt events.EnterpriseEvent) {
	tenantID := evt.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	domain := payloadString(evt.Payload, "domain_id", "domain", "indicator_domain")
	if domain == "" {
		log.Printf("[AutoPushConsumer] indicator.computed event %s has no domain; skipping", evt.EventID)
		return
	}

	sourceNodeID := payloadString(evt.Payload, "source_node_id", "provider_node_id")
	if sourceNodeID == "" {
		log.Printf("[AutoPushConsumer] indicator.computed event %s has no source node; skipping", evt.EventID)
		return
	}

	indicatorID := payloadString(evt.Payload, "indicator_id")
	if indicatorID == "" {
		// Fall back to resolving the indicator by its code.
		if code := payloadString(evt.Payload, "indicator_code", "code"); code != "" {
			if ind, err := a.store.GetIndicatorByCode(ctx, code); err == nil {
				indicatorID = ind.ID
			}
		}
	}
	indicatorIDs := make([]string, 0, 1)
	if indicatorID != "" {
		indicatorIDs = append(indicatorIDs, indicatorID)
	}

	senderName := senderNameFor(tenantID)
	targets := a.resolvePushTargets(ctx, sourceNodeID, domain, tenantID)
	if len(targets) == 0 {
		log.Printf("[AutoPushConsumer] no DSA-authorised target for indicator %q domain %q", indicatorID, domain)
		return
	}

	for _, t := range targets {
		a.enqueue(ctx, pushJob{
			sourceNodeID: sourceNodeID,
			targetNodeID: t,
			indicatorIDs: indicatorIDs,
			domain:       domain,
			senderName:   senderName,
			tenantID:     tenantID,
		})
	}
	log.Printf("[AutoPushConsumer] queued indicator %q push to %d node(s) for domain %q", indicatorID, len(targets), domain)
}

// OnIndicatorPushRequested handles an explicit "federation.indicator.push.requested"
// event containing a target node and a set of indicator IDs (or a single code).
func (a *AutoPushConsumer) OnIndicatorPushRequested(ctx context.Context, evt events.EnterpriseEvent) {
	tenantID := evt.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	sourceNodeID := payloadString(evt.Payload, "source_node_id", "provider_node_id")
	if sourceNodeID == "" {
		sourceNodeID = evt.Source
	}

	ids := payloadStringSlice(evt.Payload, "indicator_ids")
	if len(ids) == 0 {
		if code := payloadString(evt.Payload, "indicator_code", "code"); code != "" {
			if ind, err := a.store.GetIndicatorByCode(ctx, code); err == nil {
				ids = []string{ind.ID}
			}
		}
	}

	domain := payloadString(evt.Payload, "domain_id", "domain")
	senderName := payloadString(evt.Payload, "sender_name")
	if senderName == "" {
		senderName = senderNameFor(tenantID)
	}

	// Support a single target or an explicit list of targets.
	targets := payloadStringSlice(evt.Payload, "target_node_ids", "targets")
	if t := payloadString(evt.Payload, "target_node_id", "target_id"); t != "" {
		targets = append(targets, t)
	}
	if len(targets) == 0 {
		log.Printf("[AutoPushConsumer] push.requested event %s has no target node; skipping", evt.EventID)
		return
	}

	for _, t := range targets {
		a.enqueue(ctx, pushJob{
			sourceNodeID: sourceNodeID,
			targetNodeID: t,
			indicatorIDs: ids,
			domain:       domain,
			senderName:   senderName,
			tenantID:     tenantID,
		})
	}
	log.Printf("[AutoPushConsumer] queued %d indicator(s) to %d target(s)", len(ids), len(targets))
}

// enqueue places a push job on the queue without blocking callers; jobs are
// dropped (and logged) when the queue is full to protect the pipeline. The
// worker is started lazily on first enqueue so the consumer is self-contained.
func (a *AutoPushConsumer) enqueue(ctx context.Context, job pushJob) {
	if a.dispatcher == nil {
		log.Printf("[AutoPushConsumer] dispatcher not configured; dropping push job to %s", job.targetNodeID)
		return
	}
	a.startOnce.Do(func() {
		go a.run(ctx)
		log.Println("[AutoPushConsumer] queue worker started")
	})
	select {
	case a.queue <- job:
	default:
		log.Printf("[AutoPushConsumer] queue full; dropping push job to %s", job.targetNodeID)
	}
}

// run is the single queue-worker goroutine.
func (a *AutoPushConsumer) run(ctx context.Context) {
	for {
		select {
		case job := <-a.queue:
			a.dispatch(ctx, job)
		case <-ctx.Done():
			log.Println("[AutoPushConsumer] queue worker stopping")
			return
		}
	}
}

// dispatch executes one queued push via the configured dispatcher.
func (a *AutoPushConsumer) dispatch(ctx context.Context, job pushJob) {
	if a.dispatcher == nil {
		return // already rejected at enqueue
	}
	req := engine.IndicatorPushRequest{
		SourceNodeID: job.sourceNodeID,
		TargetNodeID: job.targetNodeID,
		IndicatorIDs: job.indicatorIDs,
		Domain:       job.domain,
	}
	res, err := a.dispatcher.PushIndicators(ctx, req, job.senderName, job.tenantID)
	if err != nil {
		log.Printf("[AutoPushConsumer:dispatch] push failed source=%s target=%s: %v", job.sourceNodeID, job.targetNodeID, err)
		return
	}
	log.Printf("[AutoPushConsumer:dispatch] push %s to %s (%d indicators)", res.Status, job.targetNodeID, res.IndicatorCount)
}

// resolvePushTargets returns the IDs of nodes that are authorised by an active
// DSA to consume the given domain from us (this hub acts as the provider).
func (a *AutoPushConsumer) resolvePushTargets(ctx context.Context, sourceNodeID, domain, tenantID string) []string {
	nodes, err := a.store.ListNodes(ctx, tenantID, "", "")
	if err != nil {
		return nil
	}
	var targets []string
	for _, n := range nodes {
		if n.ID == "" || n.ID == sourceNodeID {
			continue
		}
		if _, err := a.store.FindActiveDSA(ctx, sourceNodeID, n.ID, domain); err == nil {
			targets = append(targets, n.ID)
		}
	}
	return targets
}

// senderNameFor builds a human-readable sender label.
func senderNameFor(tenantID string) string {
	return "StatGate Hub (" + tenantID + ")"
}

// payloadString reads the first non-empty string for a set of candidate keys.
func payloadString(payload map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := payload[k]; ok && v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// payloadStringSlice reads a slice-of-strings value, tolerating both native
// []string and decoded []interface{} payloads.
func payloadStringSlice(payload map[string]interface{}, keys ...string) []string {
	for _, k := range keys {
		raw, ok := payload[k]
		if !ok || raw == nil {
			continue
		}
		switch v := raw.(type) {
		case []string:
			out := make([]string, 0, len(v))
			for _, s := range v {
				if s != "" {
					out = append(out, s)
				}
			}
			return out
		case []interface{}:
			out := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}