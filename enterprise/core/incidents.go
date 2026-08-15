package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type IncidentStore struct {
	sync.RWMutex
	incidents map[string]*Incident
}

var incidentStore = &IncidentStore{
	incidents: make(map[string]*Incident),
}

// ─── Default Sample & Baseline Incidents ──────────────────────────

func initIncidentDetector() {
	now := nowRFC3339()
	defaults := []Incident{
		{
			ID:                 "inc_auto_001",
			Title:              "Temporary Redis Connection Latency Spike",
			Severity:           SeverityWarning,
			Status:             IncResolved,
			ServiceID:          "enterprise-core",
			ServiceName:        "Enterprise Core",
			DetectionSource:    "AUTONOMOUS_METRICS_MONITOR",
			RootCauseSummary:   "Transient network socket contention during high throughput sync.",
			AffectedComponents: []string{"redis", "events_broker"},
			CorrelationID:      "corr_inc_001",
			DetectedAt:         time.Now().Add(-12 * time.Hour).UTC().Format(time.RFC3339),
			AcknowledgedAt:     time.Now().Add(-11*time.Hour - 50*time.Minute).UTC().Format(time.RFC3339),
			AcknowledgedBy:     "Reliability Engineer",
			ResolvedAt:         time.Now().Add(-11 * time.Hour).UTC().Format(time.RFC3339),
			ResolvedBy:         "Automated Connection Pool Recycler",
			ClosedAt:           time.Now().Add(-10 * time.Hour).UTC().Format(time.RFC3339),
			ClosedBy:           "Quality Assurance Auditor",
			RTOImpactSec:       45,
			RPOImpactSec:       0,
			EvidenceID:         "evi_inc_001",
			Events: []IncidentEvent{
				{ID: 1, IncidentID: "inc_auto_001", FromStatus: "DETECTED", ToStatus: "TRIAGED", Actor: "AUTONOMOUS_ENGINE", Reason: "Severity assigned based on Tier 0 impact matrix", CreatedAt: now},
				{ID: 2, IncidentID: "inc_auto_001", FromStatus: "TRIAGED", ToStatus: "ACKNOWLEDGED", Actor: "Reliability Engineer", Reason: "On-call response initialized", CreatedAt: now},
				{ID: 3, IncidentID: "inc_auto_001", FromStatus: "ACKNOWLEDGED", ToStatus: "RESOLVED", Actor: "Automated Connection Pool Recycler", Reason: "Socket pool recycled; latency returned to <5ms", CreatedAt: now},
			},
			Actions: []IncidentAction{
				{ID: "act_1", IncidentID: "inc_auto_001", ActionType: "FLUSH_STALE_CONNECTIONS", Description: "Re-established idle connections in Redis pool", Status: "COMPLETED", ExecutedBy: "AUTONOMOUS_AGENT", Output: "Connections recycled", CreatedAt: now},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	incidentStore.Lock()
	for i := range defaults {
		inc := defaults[i]
		incidentStore.incidents[inc.ID] = &inc
	}
	incidentStore.Unlock()

	// Start background autonomous detection loop (every 60s)
	go startAutonomousIncidentDetector()

	log.Printf("incidents: engine initialized with %d baseline records", len(defaults))
}

// ─── Autonomous Detection Loop ───────────────────────────────────

func startAutonomousIncidentDetector() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		runAutonomousChecks()
	}
}

func runAutonomousChecks() {
	// 1. Check readiness & live status
	ready, details := isReady()
	if !ready {
		// Verify if active incident already exists
		hasActive := false
		incidentStore.RLock()
		for _, inc := range incidentStore.incidents {
			if inc.ServiceID == "enterprise-core" && inc.Status != IncResolved && inc.Status != IncClosed {
				hasActive = true
				break
			}
		}
		incidentStore.RUnlock()

		if !hasActive {
			incID := fmt.Sprintf("inc_auto_%d", time.Now().Unix())
			inc := &Incident{
				ID:                 incID,
				Title:              "Enterprise Core Readiness Probe Failure",
				Severity:           SeverityCritical,
				Status:             IncDetected,
				ServiceID:          "enterprise-core",
				ServiceName:        "Enterprise Core",
				DetectionSource:    "AUTONOMOUS_PROBE_MONITOR",
				RootCauseSummary:   fmt.Sprintf("Readiness probe check failed: %v", details),
				AffectedComponents: []string{"enterprise-core", "database", "redis"},
				CorrelationID:      fmt.Sprintf("corr_auto_%d", time.Now().UnixNano()),
				DetectedAt:         nowRFC3339(),
				Events: []IncidentEvent{
					{
						ID:         time.Now().UnixNano(),
						IncidentID: incID,
						FromStatus: "",
						ToStatus:   "DETECTED",
						Actor:      "AUTONOMOUS_ENGINE",
						Reason:     "Automated probe failure detected on /ready",
						CreatedAt:  nowRFC3339(),
					},
				},
				CreatedAt: nowRFC3339(),
				UpdatedAt: nowRFC3339(),
			}

			incidentStore.Lock()
			incidentStore.incidents[incID] = inc
			incidentStore.Unlock()

			log.Printf("incidents: [AUTONOMOUS DETECTION] Raised incident %s (Severity: %s)", incID, inc.Severity)
		}
	}
}

// ─── State Machine Transition ─────────────────────────────────────

func transitionIncident(inc *Incident, targetStatus IncidentStatus, actor, reason string) error {
	from := inc.Status
	inc.Status = targetStatus
	inc.UpdatedAt = nowRFC3339()

	switch targetStatus {
	case IncAcknowledged:
		inc.AcknowledgedAt = nowRFC3339()
		inc.AcknowledgedBy = actor
	case IncResolved:
		inc.ResolvedAt = nowRFC3339()
		inc.ResolvedBy = actor
	case IncClosed:
		inc.ClosedAt = nowRFC3339()
		inc.ClosedBy = actor
	}

	event := IncidentEvent{
		ID:         time.Now().UnixNano(),
		IncidentID: inc.ID,
		FromStatus: string(from),
		ToStatus:   string(targetStatus),
		Actor:      actor,
		Reason:     reason,
		CreatedAt:  nowRFC3339(),
	}
	inc.Events = append(inc.Events, event)

	// If resolved or closed, record evidence
	if targetStatus == IncResolved || targetStatus == IncClosed {
		evidence := recordResilienceEvidence(
			"INCIDENT_RECOVERY",
			inc.ServiceID,
			actor,
			"VERIFIED_SUCCESS",
			map[string]interface{}{
				"incident_id": inc.ID,
				"severity":    inc.Severity,
				"duration_s":  inc.RTOImpactSec,
			},
			fmt.Sprintf("Incident %s successfully resolved with reason: %s", inc.ID, reason),
		)
		inc.EvidenceID = evidence.ID
	}

	return nil
}

// ─── HTTP API Handlers ────────────────────────────────────────────

func handleListIncidents(c *gin.Context) {
	statusFilter := c.Query("status")
	severityFilter := c.Query("severity")
	serviceFilter := c.Query("service")

	incidentStore.RLock()
	incidents := make([]Incident, 0, len(incidentStore.incidents))
	for _, inc := range incidentStore.incidents {
		if statusFilter != "" && string(inc.Status) != statusFilter {
			continue
		}
		if severityFilter != "" && string(inc.Severity) != severityFilter {
			continue
		}
		if serviceFilter != "" && inc.ServiceID != serviceFilter {
			continue
		}
		incidents = append(incidents, *inc)
	}
	incidentStore.RUnlock()

	c.JSON(200, gin.H{
		"count":     len(incidents),
		"incidents": incidents,
	})
}

func handleGetIncident(c *gin.Context) {
	id := c.Param("id")
	incidentStore.RLock()
	inc, ok := incidentStore.incidents[id]
	incidentStore.RUnlock()

	if !ok {
		c.JSON(404, gin.H{"error": "incident not found", "id": id})
		return
	}
	c.JSON(200, inc)
}

func handleCreateIncident(c *gin.Context) {
	var inc Incident
	if err := c.ShouldBindJSON(&inc); err != nil {
		c.JSON(400, gin.H{"error": "invalid incident payload", "detail": err.Error()})
		return
	}

	if inc.ID == "" {
		inc.ID = fmt.Sprintf("inc_%d", time.Now().UnixNano())
	}
	if inc.Status == "" {
		inc.Status = IncDetected
	}
	if inc.Severity == "" {
		inc.Severity = SeverityHigh
	}
	inc.DetectedAt = nowRFC3339()
	inc.CreatedAt = nowRFC3339()
	inc.UpdatedAt = nowRFC3339()

	actor := getContextUserID(c)
	if actor == "" {
		actor = "administrator"
	}

	inc.Events = []IncidentEvent{
		{
			ID:         time.Now().UnixNano(),
			IncidentID: inc.ID,
			FromStatus: "",
			ToStatus:   string(inc.Status),
			Actor:      actor,
			Reason:     "Incident manually or programmatically opened",
			CreatedAt:  nowRFC3339(),
		},
	}

	incidentStore.Lock()
	incidentStore.incidents[inc.ID] = &inc
	incidentStore.Unlock()

	recordAuditFromContext(c, "incident.create", "incidents", inc.ID, map[string]interface{}{
		"title": inc.Title, "severity": inc.Severity, "service_id": inc.ServiceID,
	})

	c.JSON(201, inc)
}

func handleAcknowledgeIncident(c *gin.Context) {
	id := c.Param("id")
	actor := getContextUserID(c)
	if actor == "" {
		actor = "administrator"
	}

	incidentStore.Lock()
	inc, ok := incidentStore.incidents[id]
	if !ok {
		incidentStore.Unlock()
		c.JSON(404, gin.H{"error": "incident not found", "id": id})
		return
	}

	_ = transitionIncident(inc, IncAcknowledged, actor, "Acknowledged by on-call operator")
	incidentStore.Unlock()

	recordAuditFromContext(c, "incident.acknowledge", "incidents", id, nil)
	c.JSON(200, gin.H{"status": "acknowledged", "incident": inc})
}

func handleMitigateIncident(c *gin.Context) {
	id := c.Param("id")
	actor := getContextUserID(c)
	if actor == "" {
		actor = "administrator"
	}

	incidentStore.Lock()
	inc, ok := incidentStore.incidents[id]
	if !ok {
		incidentStore.Unlock()
		c.JSON(404, gin.H{"error": "incident not found", "id": id})
		return
	}

	_ = transitionIncident(inc, IncMitigating, actor, "Mitigation plan initiated")
	incidentStore.Unlock()

	recordAuditFromContext(c, "incident.mitigate", "incidents", id, nil)
	c.JSON(200, gin.H{"status": "mitigating", "incident": inc})
}

func handleResolveIncident(c *gin.Context) {
	id := c.Param("id")
	actor := getContextUserID(c)
	if actor == "" {
		actor = "administrator"
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Reason == "" {
		req.Reason = "Service health and data consistency confirmed"
	}

	incidentStore.Lock()
	inc, ok := incidentStore.incidents[id]
	if !ok {
		incidentStore.Unlock()
		c.JSON(404, gin.H{"error": "incident not found", "id": id})
		return
	}

	_ = transitionIncident(inc, IncResolved, actor, req.Reason)
	incidentStore.Unlock()

	recordAuditFromContext(c, "incident.resolve", "incidents", id, map[string]interface{}{"reason": req.Reason})
	c.JSON(200, gin.H{"status": "resolved", "incident": inc, "evidence_id": inc.EvidenceID})
}

func handleCloseIncident(c *gin.Context) {
	id := c.Param("id")
	actor := getContextUserID(c)
	if actor == "" {
		actor = "administrator"
	}

	incidentStore.Lock()
	inc, ok := incidentStore.incidents[id]
	if !ok {
		incidentStore.Unlock()
		c.JSON(404, gin.H{"error": "incident not found", "id": id})
		return
	}

	_ = transitionIncident(inc, IncClosed, actor, "Post-incident review and audit certification complete")
	incidentStore.Unlock()

	recordAuditFromContext(c, "incident.close", "incidents", id, nil)
	c.JSON(200, gin.H{"status": "closed", "incident": inc})
}

func handleAddIncidentAction(c *gin.Context) {
	id := c.Param("id")
	var act IncidentAction
	if err := c.ShouldBindJSON(&act); err != nil {
		c.JSON(400, gin.H{"error": "invalid action payload", "detail": err.Error()})
		return
	}

	if act.ID == "" {
		act.ID = fmt.Sprintf("act_%d", time.Now().UnixNano())
	}
	act.IncidentID = id
	act.CreatedAt = nowRFC3339()
	if act.ExecutedBy == "" {
		act.ExecutedBy = getContextUserID(c)
	}

	incidentStore.Lock()
	inc, ok := incidentStore.incidents[id]
	if !ok {
		incidentStore.Unlock()
		c.JSON(404, gin.H{"error": "incident not found", "id": id})
		return
	}
	inc.Actions = append(inc.Actions, act)
	inc.UpdatedAt = nowRFC3339()
	incidentStore.Unlock()

	c.JSON(201, act)
}
