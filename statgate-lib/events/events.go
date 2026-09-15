package events

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	DefaultChannel = "statgate:events"
	DLQChannel     = "statgate:events:dlq"
	DefaultStream  = "statgate:events:stream"
	DLQStream      = "statgate:events:dlq:stream"
)

// ── In-process pub/sub broker ──────────────────────────────────────────────
// When Redis is unavailable (local development, CI, tests) the EventBus
// falls back to this process-local broker so that all app instances running
// in the same process can STILL exchange domain events on the canonical
// channel. This makes cross-application communication verifiable everywhere,
// not only in full-stack deployments.
type memoryBroker struct {
	mu   sync.Mutex
	subs map[string]map[chan EnterpriseEvent]struct{}
}

var inMemoryBroker = &memoryBroker{subs: make(map[string]map[chan EnterpriseEvent]struct{})}

func (b *memoryBroker) publish(channel string, evt EnterpriseEvent) {
	b.mu.Lock()
	targets := make([]chan EnterpriseEvent, 0, 8)
	for c := range b.subs[channel] {
		targets = append(targets, c)
	}
	b.mu.Unlock()
	for _, c := range targets {
		select {
		case c <- evt:
		default:
			log.Printf("[EventBus] in-memory subscriber queue full on %s; dropping event %s", channel, evt.EventID)
		}
	}
}

func (b *memoryBroker) subscribe(channel string) chan EnterpriseEvent {
	c := make(chan EnterpriseEvent, 256)
	b.mu.Lock()
	if b.subs[channel] == nil {
		b.subs[channel] = make(map[chan EnterpriseEvent]struct{})
	}
	b.subs[channel][c] = struct{}{}
	b.mu.Unlock()
	return c
}

func (b *memoryBroker) unsubscribe(channel string, c chan EnterpriseEvent) {
	b.mu.Lock()
	if set := b.subs[channel]; set != nil {
		delete(set, c)
	}
	b.mu.Unlock()
}

// Standard Event Type Constants across StatGate
const (
	// Registry
	EventUserCreated         = "user.created"
	EventUserUpdated         = "user.updated"
	EventFacilityCreated     = "facility.created"
	EventFacilityUpdated     = "facility.updated"
	EventOrganizationCreated = "organization.created"

	// PMS
	EventProjectCreated          = "project.created"
	EventProjectUpdated          = "project.updated"
	EventProjectMilestoneReached = "project.milestone_reached"
	EventProjectCompleted        = "project.completed"
	EventProjectDelayed          = "project.delayed"

	// RMS
	EventResearchCreated      = "research.created"
	EventResearchUpdated      = "research.updated"
	EventResearchStageChanged = "research.stage_changed"
	EventResearchApproved     = "research.approved"
	EventResearchPublished    = "research.published"

	// StatCollect
	EventSurveyCreated       = "survey.created"
	EventSurveyPublished     = "survey.published"
	EventSubmissionReceived  = "submission.received"
	EventSubmissionValidated = "submission.validated"
	EventSubmissionRejected  = "submission.rejected"
	EventFieldDataSubmitted  = "field_data.submitted"

	// HelpDesk
	EventTicketCreated     = "ticket.created"
	EventTicketAssigned    = "ticket.assigned"
	EventTicketUpdated     = "ticket.updated"
	EventTicketEscalated   = "ticket.escalated"
	EventTicketResolved    = "ticket.resolved"
	EventTicketSLABreached = "ticket.sla_breached"

	// StatChat
	EventMessageCreated    = "message.created"
	EventDiscussionCreated = "discussion.created"
	EventDiscussionUpdated = "discussion.updated"

	// Analytics
	EventDatasetCreated    = "dataset.created"
	EventDatasetUpdated    = "dataset.updated"
	EventTelemetryIngested = "telemetry.ingested"
	EventReportGenerated   = "report.generated"
	EventAnomalyDetected   = "anomaly.detected"

	// StatGovernance
	EventRiskCreated       = "risk.created"
	EventFindingCreated    = "finding.created"
	EventControlCreated    = "control.created"
	EventComplianceUpdated = "compliance.updated"

	// StatSpatial
	EventSpatialLayerCreated   = "spatial.layer.created"
	EventSpatialFeatureUpdated = "spatial.feature.updated"
	EventSpatialNodeSynced     = "spatial.node.synced"

	// Knowledge Portal (content / open data / library)
	EventContentPublished    = "content.published"
	EventDatasetPublished    = "dataset.published"
	EventRepositoryArchived  = "repository.item.archived"
	EventSubscriptionCreated = "subscription.created"
	EventFeedbackReceived    = "feedback.received"
	EventObjectLinkCreated   = "object.link.created"

	// AI & Autonomy (App 5: P22 Digital Twins & Analytics, P31 Agents, P39 Knowledge Graph)
	EventPredictionGenerated   = "prediction.generated"
	EventSimulationCompleted   = "simulation.completed"
	EventAgentTaskCreated      = "agent.task.created"
	EventAgentTaskCompleted    = "agent.task.completed"
	EventAgentActionTaken      = "agent.action.taken"
	EventGraphEntityRegistered = "graph.entity.registered"

	// Learning, Community & Commercial (App 7: P24 LMS/CPD, P26 CRM, P35 Stewardship)
	EventCourseCompleted       = "course.completed"
	EventCertificateIssued     = "certificate.issued"
	EventBadgeIssued           = "badge.issued"
	EventEnrollmentCreated     = "enrollment.created"
	EventLeadConverted         = "lead.converted"
	EventPartnerRegistered     = "partner.registered"
	EventServiceRequestCreated = "service_request.created"

	// Geospatial & Remote Sensing (App 9: P44 GIS / Drone / Remote Sensing)
	EventSceneIngested        = "scene.ingested"
	EventDroneFlightCompleted = "drone.flight.completed"
	EventSpatialAnalysisDone  = "spatial.analysis.completed"
	EventMapTileRegistered    = "tile.registered"

	// Business Process Management (App 11: P48 BPM / Case / Process Mining)
	EventProcessStarted      = "process.started"
	EventProcessCompleted    = "process.completed"
	EventWorkItemCreated     = "task.created"
	EventWorkItemCompleted   = "task.completed"
	EventCaseCreated         = "case.created"
	EventAutomationTriggered = "automation.triggered"
)

// EnterpriseEvent represents the universal message schema.
type EnterpriseEvent struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	Source        string                 `json:"source"`
	ObjectType    string                 `json:"object_type"`
	ObjectID      string                 `json:"object_id"`
	TenantID      string                 `json:"tenant_id"`
	OrgID         string                 `json:"org_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
	Timestamp     time.Time              `json:"timestamp"`
	Version       string                 `json:"version"`
}

// EventBus provides publishing, subscription, deduplication, and dead-letter handling.
type EventBus struct {
	client       *redis.Client
	channel      string
	dlqChannel   string
	source       string
	seenCache    map[string]time.Time
	seenMu       sync.RWMutex
	pruneTimeout time.Duration
	localStop    chan struct{} // closes the in-memory consumer goroutine
	closeOnce    sync.Once
	closeErr     error
	stream       string
	dlqStream    string
	group        string
	consumer     string
	maxAttempts  int
	retryDelay   time.Duration
}

// Config holds EventBus initialization options.
type Config struct {
	RedisAddr   string
	Password    string
	DB          int
	Source      string
	Channel     string
	DLQChannel  string
	Stream      string
	DLQStream   string
	Group       string
	Consumer    string
	MaxAttempts int
	RetryDelay  time.Duration
}

// RedisAvailable reports whether the cross-process transport is reachable.
// An EventBus can still serve in-process subscribers when this returns false,
// but services should expose the distinction in health responses because the
// in-process broker cannot communicate across application boundaries.
func (b *EventBus) RedisAvailable(ctx context.Context) bool {
	if b == nil || b.client == nil {
		return false
	}
	return b.client.Ping(ctx).Err() == nil
}

// NewEventBus initializes a new EventBus.
func NewEventBus(cfg Config) (*EventBus, error) {
	if cfg.Channel == "" {
		cfg.Channel = DefaultChannel
	}
	if cfg.DLQChannel == "" {
		cfg.DLQChannel = DLQChannel
	}
	if cfg.Stream == "" {
		cfg.Stream = DefaultStream
	}
	if cfg.DLQStream == "" {
		cfg.DLQStream = DLQStream
	}
	if cfg.Group == "" {
		cfg.Group = "statgate-default"
	}
	if cfg.Consumer == "" {
		cfg.Consumer = uuid.NewString()
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = time.Second
	}
	if cfg.Source == "" {
		cfg.Source = "statgate-core"
	}

	var rClient *redis.Client
	if cfg.RedisAddr != "" {
		rClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := rClient.Ping(ctx).Err(); err != nil {
			log.Printf("[EventBus] Warning: Redis connection failed (%v). Bus running in in-memory fallback.", err)
		}
	}

	bus := &EventBus{
		client:       rClient,
		channel:      cfg.Channel,
		dlqChannel:   cfg.DLQChannel,
		source:       cfg.Source,
		seenCache:    make(map[string]time.Time),
		pruneTimeout: 10 * time.Minute,
		stream:       cfg.Stream,
		dlqStream:    cfg.DLQStream,
		group:        cfg.Group,
		consumer:     cfg.Consumer,
		maxAttempts:  cfg.MaxAttempts,
		retryDelay:   cfg.RetryDelay,
	}

	return bus, nil
}

// Publish creates and broadcasts an event to the Enterprise Event Bus.
func (b *EventBus) Publish(ctx context.Context, evt EnterpriseEvent) error {
	evt = normalizeEvent(b, evt)
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if b.client != nil {
		if err := b.client.Publish(ctx, b.channel, string(data)).Err(); err != nil {
			log.Printf("[EventBus] Redis publish error: %v. Delivering via in-memory broker.", err)
			inMemoryBroker.publish(b.channel, evt)
			return err
		}
	} else {
		// No Redis: still deliver in-process so sibling app instances can react.
		inMemoryBroker.publish(b.channel, evt)
		log.Printf("[EventBus:InMemory] broadcast event %s (%s) for object %s:%s", evt.EventType, evt.EventID, evt.ObjectType, evt.ObjectID)
	}

	return nil
}

// PublishDurable persists an event in Redis Streams and also publishes it on
// the live Pub/Sub channel for existing consumers. The stream is the recovery
// source; Pub/Sub remains the low-latency compatibility path.
func (b *EventBus) PublishDurable(ctx context.Context, evt EnterpriseEvent) error {
	evt = normalizeEvent(b, evt)
	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal durable event: %w", err)
	}
	if b.client == nil {
		inMemoryBroker.publish(b.channel, evt)
		return nil
	}
	publishedKey := fmt.Sprintf("%s:published:%s", b.stream, evt.EventID)
	claimed, err := b.client.SetNX(ctx, publishedKey, "1", 24*time.Hour).Result()
	if err != nil {
		return fmt.Errorf("claim durable event id: %w", err)
	}
	if !claimed {
		return nil
	}
	if err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: b.stream,
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{"event": string(data)},
	}).Err(); err != nil {
		_ = b.client.Del(ctx, publishedKey).Err()
		return fmt.Errorf("append durable event: %w", err)
	}
	if err := b.client.Publish(ctx, b.channel, string(data)).Err(); err != nil {
		return fmt.Errorf("publish live event after durable append: %w", err)
	}
	return nil
}

func normalizeEvent(b *EventBus, evt EnterpriseEvent) EnterpriseEvent {
	if evt.EventID == "" {
		evt.EventID = uuid.New().String()
	}
	if evt.Source == "" {
		evt.Source = b.source
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	if evt.Version == "" {
		evt.Version = "1.0"
	}
	if evt.TenantID == "" {
		evt.TenantID = "default"
	}
	return evt
}

// PublishDLQ sends a rejected or poisoned event to the Dead Letter Queue.
func (b *EventBus) PublishDLQ(ctx context.Context, evt EnterpriseEvent, failureReason string) error {
	if evt.Payload == nil {
		evt.Payload = make(map[string]interface{})
	}
	evt.Payload["_dlq_failure_reason"] = failureReason
	evt.Payload["_dlq_timestamp"] = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	if b.client != nil {
		return b.client.Publish(ctx, b.dlqChannel, string(data)).Err()
	}
	log.Printf("[EventBus:DLQ] Event %s sent to DLQ: %s", evt.EventID, failureReason)
	return nil
}

// IsDuplicate checks if an event ID or hash has already been processed within the deduplication window.
func (b *EventBus) IsDuplicate(eventID string) bool {
	b.seenMu.Lock()
	defer b.seenMu.Unlock()

	now := time.Now()
	// Prune old entries
	for id, t := range b.seenCache {
		if now.Sub(t) > b.pruneTimeout {
			delete(b.seenCache, id)
		}
	}

	if _, exists := b.seenCache[eventID]; exists {
		return true
	}

	b.seenCache[eventID] = now
	return false
}

// CalculateEventHash returns a deterministic SHA-256 fingerprint for idempotency checking.
func CalculateEventHash(tenantID, eventType, objectID string, timestamp time.Time) string {
	raw := fmt.Sprintf("%s:%s:%s:%d", tenantID, eventType, objectID, timestamp.Unix())
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// SubscribeHandler represents a callback for incoming domain events.
type SubscribeHandler func(ctx context.Context, evt EnterpriseEvent) error

// Subscribe listens to the canonical channel and executes handlers with
// deduplication. It works with a physical Redis pub/sub connection OR the
// in-process broker (when Redis is absent) so cross-app communication can be
// verified in any environment.
func (b *EventBus) Subscribe(ctx context.Context, handler SubscribeHandler) error {
	if b.client == nil {
		ch := inMemoryBroker.subscribe(b.channel)
		b.localStop = make(chan struct{})
		go func() {
			defer inMemoryBroker.unsubscribe(b.channel, ch)
			for {
				select {
				case <-b.localStop:
					return
				case evt := <-ch:
					if b.IsDuplicate(evt.EventID) {
						log.Printf("[EventBus] Skipping duplicate event %s (%s)", evt.EventID, evt.EventType)
						continue
					}
					if err := handler(ctx, evt); err != nil {
						log.Printf("[EventBus] Handler error on event %s: %v. Routing to DLQ.", evt.EventID, err)
						_ = b.PublishDLQ(ctx, evt, err.Error())
					}
				}
			}
		}()
		return nil
	}

	pubsub := b.client.Subscribe(ctx, b.channel)
	ch := pubsub.Channel()

	go func() {
		defer pubsub.Close()
		for msg := range ch {
			var evt EnterpriseEvent
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				log.Printf("[EventBus] Failed to parse incoming event payload: %v", err)
				continue
			}

			if b.IsDuplicate(evt.EventID) {
				log.Printf("[EventBus] Skipping duplicate event %s (%s)", evt.EventID, evt.EventType)
				continue
			}

			if err := handler(ctx, evt); err != nil {
				log.Printf("[EventBus] Handler error on event %s: %v. Routing to DLQ.", evt.EventID, err)
				_ = b.PublishDLQ(ctx, evt, err.Error())
			}
		}
	}()

	return nil
}

// SubscribeDurable consumes the Redis Stream with a consumer group. Failed
// handlers are retried until MaxAttempts, then acknowledged only after the
// event is written to the durable DLQ. Processed event IDs are retained in
// Redis so redelivery cannot run a successful handler twice.
func (b *EventBus) SubscribeDurable(ctx context.Context, group, consumer string, handler SubscribeHandler) error {
	if b == nil {
		return fmt.Errorf("event bus is nil")
	}
	if b.client == nil {
		return b.Subscribe(ctx, handler)
	}
	if strings.TrimSpace(group) == "" {
		group = b.group
	}
	if strings.TrimSpace(consumer) == "" {
		consumer = b.consumer
	}
	if err := b.client.XGroupCreateMkStream(ctx, b.stream, group, "$").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create event consumer group: %w", err)
	}

	go func() {
		for ctx.Err() == nil {
			claimed, _, err := b.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
				Stream:   b.stream,
				Group:    group,
				Consumer: consumer,
				MinIdle:  b.retryDelay,
				Start:    "-",
				Count:    10,
			}).Result()
			if err != nil && err != redis.Nil && ctx.Err() == nil {
				log.Printf("[EventBus] durable pending claim failed: %v", err)
			}
			for _, message := range claimed {
				b.processDurableMessage(ctx, group, message, handler)
			}

			blockDuration := time.Second
			if b.retryDelay > 0 && b.retryDelay < blockDuration {
				blockDuration = b.retryDelay
			}
			batches, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{b.stream, ">"},
				Count:    10,
				Block:    blockDuration,
			}).Result()
			if err == redis.Nil || (err != nil && ctx.Err() != nil) {
				continue
			}
			if err != nil {
				log.Printf("[EventBus] durable event read failed: %v", err)
				continue
			}
			for _, batch := range batches {
				for _, message := range batch.Messages {
					b.processDurableMessage(ctx, group, message, handler)
				}
			}
		}
	}()
	return nil
}

func (b *EventBus) processDurableMessage(ctx context.Context, group string, message redis.XMessage, handler SubscribeHandler) {
	raw, ok := message.Values["event"]
	if !ok {
		log.Printf("[EventBus] durable event %s has no event payload", message.ID)
		_ = b.client.XAck(ctx, b.stream, group, message.ID).Err()
		return
	}
	var event EnterpriseEvent
	if err := json.Unmarshal([]byte(fmt.Sprint(raw)), &event); err != nil {
		log.Printf("[EventBus] durable event %s could not be decoded: %v", message.ID, err)
		_ = b.publishDurableDLQ(ctx, event, "invalid event payload: "+err.Error())
		_ = b.client.XAck(ctx, b.stream, group, message.ID).Err()
		return
	}
	processedKey := fmt.Sprintf("%s:processed:%s:%s", b.stream, group, event.EventID)
	processed, err := b.client.Exists(ctx, processedKey).Result()
	if err != nil {
		log.Printf("[EventBus] durable idempotency check failed: %v", err)
		return
	}
	if processed > 0 {
		_ = b.client.XAck(ctx, b.stream, group, message.ID).Err()
		return
	}

	if err := handler(ctx, event); err == nil {
		if err := b.client.Set(ctx, processedKey, "1", 24*time.Hour).Err(); err != nil {
			log.Printf("[EventBus] durable idempotency mark failed: %v", err)
			return
		}
		_ = b.client.Del(ctx, fmt.Sprintf("%s:attempts:%s", b.stream, event.EventID)).Err()
		if err := b.client.XAck(ctx, b.stream, group, message.ID).Err(); err != nil {
			log.Printf("[EventBus] durable event acknowledgement failed: %v", err)
		}
		return
	} else {
		attemptKey := fmt.Sprintf("%s:attempts:%s", b.stream, event.EventID)
		attempt, incrementErr := b.client.Incr(ctx, attemptKey).Result()
		if incrementErr != nil {
			log.Printf("[EventBus] durable retry counter failed: %v", incrementErr)
			return
		}
		_ = b.client.Expire(ctx, attemptKey, 24*time.Hour).Err()
		if attempt >= int64(b.maxAttempts) {
			if dlqErr := b.publishDurableDLQOnce(ctx, event, err.Error()); dlqErr != nil {
				log.Printf("[EventBus] durable DLQ write failed: %v", dlqErr)
				return
			}
			_ = b.client.Del(ctx, attemptKey).Err()
			_ = b.client.XAck(ctx, b.stream, group, message.ID).Err()
			return
		}
		// Leave the message pending for XAutoClaim to retry after the backoff.
		timer := time.NewTimer(b.retryDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
	}
}

func (b *EventBus) publishDurableDLQ(ctx context.Context, event EnterpriseEvent, failureReason string) error {
	if event.Payload == nil {
		event.Payload = make(map[string]interface{})
	}
	event.Payload["_dlq_failure_reason"] = failureReason
	event.Payload["_dlq_timestamp"] = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: b.dlqStream,
		MaxLen: 10000,
		Approx: true,
		Values: map[string]interface{}{"event": string(data)},
	}).Err(); err != nil {
		return err
	}
	return b.client.Publish(ctx, b.dlqChannel, string(data)).Err()
}

func (b *EventBus) publishDurableDLQOnce(ctx context.Context, event EnterpriseEvent, failureReason string) error {
	dlqKey := fmt.Sprintf("%s:published:%s", b.dlqStream, event.EventID)
	claimed, err := b.client.SetNX(ctx, dlqKey, "1", 24*time.Hour).Result()
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	if err := b.publishDurableDLQ(ctx, event, failureReason); err != nil {
		_ = b.client.Del(ctx, dlqKey).Err()
		return err
	}
	return nil
}

// Close releases the transport and stops an in-process subscription, if one
// was started. It is safe to call more than once during service shutdown.
func (b *EventBus) Close() error {
	if b == nil {
		return nil
	}
	b.closeOnce.Do(func() {
		if b.localStop != nil {
			close(b.localStop)
		}
		if b.client != nil {
			b.closeErr = b.client.Close()
		}
	})
	return b.closeErr
}

// InitFromEnv creates an EventBus using standard StatGate environment variables.
func InitFromEnv(source string) (*EventBus, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	addr := ""
	if host != "" {
		addr = host + ":" + port
	}

	maxAttempts, _ := strconv.Atoi(os.Getenv("STATGATE_EVENT_MAX_ATTEMPTS"))
	retryDelay := time.Second
	if value, err := time.ParseDuration(strings.TrimSpace(os.Getenv("STATGATE_EVENT_RETRY_DELAY"))); err == nil && value > 0 {
		retryDelay = value
	}
	return NewEventBus(Config{
		RedisAddr:   addr,
		Password:    os.Getenv("REDIS_PASSWORD"),
		Source:      source,
		Channel:     os.Getenv("STATGATE_EVENT_CHANNEL"),
		DLQChannel:  os.Getenv("STATGATE_EVENT_DLQ_CHANNEL"),
		Stream:      os.Getenv("STATGATE_EVENT_STREAM"),
		DLQStream:   os.Getenv("STATGATE_EVENT_DLQ_STREAM"),
		Group:       os.Getenv("STATGATE_EVENT_GROUP"),
		Consumer:    os.Getenv("STATGATE_EVENT_CONSUMER"),
		MaxAttempts: maxAttempts,
		RetryDelay:  retryDelay,
	})
}
