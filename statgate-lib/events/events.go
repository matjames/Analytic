package events

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	DefaultChannel = "statgate:events"
	DLQChannel     = "statgate:events:dlq"
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
}

// Config holds EventBus initialization options.
type Config struct {
	RedisAddr  string
	Password   string
	DB         int
	Source     string
	Channel    string
	DLQChannel string
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
	}

	return bus, nil
}

// Publish creates and broadcasts an event to the Enterprise Event Bus.
func (b *EventBus) Publish(ctx context.Context, evt EnterpriseEvent) error {
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

	return NewEventBus(Config{
		RedisAddr:  addr,
		Password:   os.Getenv("REDIS_PASSWORD"),
		Source:     source,
		Channel:    os.Getenv("STATGATE_EVENT_CHANNEL"),
		DLQChannel: os.Getenv("STATGATE_EVENT_DLQ_CHANNEL"),
	})
}
