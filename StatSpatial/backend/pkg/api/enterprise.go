package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"statspatial/pkg/store"

	"github.com/matjames/statgate-lib/audit"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/events"
)

// sourceApplication is the canonical StatGate application identifier used in
// audit records and event provenance.
const sourceApplication = "statspatial"

var (
	// EnterpriseBus is the shared statgate-lib Event Bus (Redis-backed with an
	// in-memory fallback when Redis is unavailable).
	EnterpriseBus *events.EventBus
	// EnterpriseAudit is the shared statgate-lib Audit Service writing the
	// authoritative enterprise_audit_log schema.
	EnterpriseAudit *audit.Service
)

// InitEnterprise wires StatSpatial onto the shared Enterprise Event Bus and
// Audit Service from statgate-lib, fulfilling the Phase y convergence
// directive (Object 19: applications consume the shared platform library).
func InitEnterprise() error {
	bus, err := events.InitFromEnv(sourceApplication)
	if err != nil {
		return err
	}
	EnterpriseBus = bus
	// The audit service reuses the StatSpatial connection pool; when it is nil
	// (e.g. not yet initialised) records are streamed as structured JSON.
	EnterpriseAudit = audit.NewService(store.DB(), sourceApplication)
	return nil
}

// emitEvent publishes a domain event onto the Enterprise Event Bus. It is a
// no-op when the bus is not initialised (e.g. during unit tests).
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
		Source:     sourceApplication,
		ObjectType: objectType,
		ObjectID:   objectID,
		TenantID:   tenantID,
		UserID:     userID,
		Payload:    payload,
		Timestamp:  time.Now().UTC(),
		Version:    "1.0",
	}

	if err := EnterpriseBus.PublishDurable(ctx, evt); err != nil {
		log.Printf("[StatSpatial] enterprise event publish failed (%s %s:%s): %v", eventType, objectType, objectID, err)
	}
}

// recordAudit writes an immutable enterprise audit record via statgate-lib.
// It is a no-op when the audit service is not initialised.
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
		Application: sourceApplication,
		TenantID:    tenantID,
		ObjectID:    objectID,
		ObjectType:  objectType,
		NewState:    newState,
		RequestID:   r.Header.Get("X-Request-ID"),
		IPAddress:   ip,
		UserAgent:   userAgent,
	}

	if err := EnterpriseAudit.Log(r.Context(), rec); err != nil {
		log.Printf("[StatSpatial] audit record failed (%s %s:%s): %v", action, objectType, objectID, err)
	}
}
