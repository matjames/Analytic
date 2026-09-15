package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"aiengines/pkg/store"

	"github.com/matjames/statgate-lib/audit"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/events"
)

// sourceApplication is the canonical StatGate application identifier.
const SourceApplication = "ai-autonomy"

var (
	// EnterpriseBus is the shared statgate-lib Event Bus (Redis-backed with an
	// in-memory broker fallback when Redis is unavailable).
	EnterpriseBus *events.EventBus
	// EnterpriseAudit writes to the authoritative enterprise_audit_log.
	EnterpriseAudit *audit.Service
)

// InitEnterprise wires this app onto the shared Event Bus and Audit Service.
func InitEnterprise() error {
	bus, err := events.InitFromEnv(SourceApplication)
	if err != nil {
		return err
	}
	EnterpriseBus = bus
	EnterpriseAudit = audit.NewService(store.DB(), SourceApplication)
	return nil
}

// StartEventConsumer subscribes to the canonical channel and automatically
// triggers agent tasks / simulations from platform domain events (P31 hook).
// Safe to run in any environment: Redis backing or in-memory broker.
func StartEventConsumer(ctx context.Context) {
	if EnterpriseBus == nil {
		log.Printf("[%s] event consumer disabled (bus not initialised)", SourceApplication)
		return
	}
	go func() {
		err := EnterpriseBus.SubscribeDurable(ctx, "ai-autonomy", "runtime", func(ctx context.Context, evt events.EnterpriseEvent) error {
			return HandleDomainEvent(ctx, evt)
		})
		if err != nil {
			log.Printf("[%s] event consumer failed to subscribe: %v", SourceApplication, err)
		}
	}()
	log.Printf("[%s] listening on %s for domain events", SourceApplication, events.DefaultChannel)
}

// emitEvent publishes a domain event onto the Enterprise Event Bus.
func emitEvent(ctx context.Context, eventType, objectType, objectID string, payload map[string]interface{}) {
	if EnterpriseBus == nil {
		return
	}
	var tenantID, userID string
	if u, ok := auth.GetUserFromContext(ctx); ok && u != nil {
		tenantID = u.TenantID
		userID = u.UserID
	}
	evt := events.EnterpriseEvent{
		EventType:  eventType,
		Source:     SourceApplication,
		ObjectType: objectType,
		ObjectID:   objectID,
		TenantID:   tenantID,
		UserID:     userID,
		Payload:    payload,
		Timestamp:  time.Now().UTC(),
		Version:    "1.0",
	}
	if err := EnterpriseBus.PublishDurable(ctx, evt); err != nil {
		log.Printf("[%s] enterprise event publish failed (%s %s:%s): %v", SourceApplication, eventType, objectType, objectID, err)
	}
}

// recordAudit writes an immutable enterprise audit record.
func recordAudit(r *http.Request, action, objectType, objectID string, newState map[string]interface{}) {
	if EnterpriseAudit == nil {
		return
	}
	var who, tenantID string
	if u, ok := auth.GetUserFromContext(r.Context()); ok && u != nil {
		who = u.UserID
		tenantID = u.TenantID
	}
	ip, userAgent := audit.ExtractContext(r)
	rec := audit.Record{
		Who:         who,
		What:        action,
		Where:       r.URL.Path,
		Application: SourceApplication,
		TenantID:    tenantID,
		ObjectID:    objectID,
		ObjectType:  objectType,
		NewState:    newState,
		RequestID:   r.Header.Get("X-Request-ID"),
		IPAddress:   ip,
		UserAgent:   userAgent,
	}
	if err := EnterpriseAudit.Log(r.Context(), rec); err != nil {
		log.Printf("[%s] audit record failed (%s %s:%s): %v", SourceApplication, action, objectType, objectID, err)
	}
}
