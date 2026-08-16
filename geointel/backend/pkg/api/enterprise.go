package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"geointel/pkg/store"

	"github.com/matjames/statgate-lib/audit"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/events"
)

// sourceApplication is the canonical StatGate application identifier.
const SourceApplication = "geointel"

var (
	EnterpriseBus   *events.EventBus
	EnterpriseAudit *audit.Service
)

func InitEnterprise() error {
	bus, err := events.InitFromEnv(SourceApplication)
	if err != nil {
		return err
	}
	EnterpriseBus = bus
	EnterpriseAudit = audit.NewService(store.DB(), SourceApplication)
	return nil
}

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
	if err := EnterpriseBus.Publish(ctx, evt); err != nil {
		log.Printf("[%s] event publish failed (%s %s:%s): %v", SourceApplication, eventType, objectType, objectID, err)
	}
}

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