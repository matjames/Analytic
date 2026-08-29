package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ─── Enterprise Service Registry Integration ──────────────────────────────────

// registerWithEnterpriseFabric registers StatCitizen with Enterprise Core (:8096)
// and periodically sends a heartbeat so Command Centre reflects live status.
func registerWithEnterpriseFabric(cfg *Config) {
	time.Sleep(2 * time.Second) // wait for server to bind

	regURL := fmt.Sprintf("%s/api/registry/services", cfg.EnterpriseCoreURL)
	payload := map[string]interface{}{
		"id":           "statcitizen",
		"name":         "statcitizen",
		"display_name": "StatCitizen",
		"description":  "Citizen participation, public evidence, consultations & institutional feedback platform",
		"api_url":      fmt.Sprintf("http://localhost:%s/api/statcitizen/v1", cfg.Port),
		"ui_url":       fmt.Sprintf("http://localhost:%s", cfg.UIPort),
		"health_url":   fmt.Sprintf("http://localhost:%s/health", cfg.Port),
		"version":      version,
		"capabilities": []string{
			"citizen_participation",
			"structured_reporting",
			"service_ratings",
			"public_consultations",
			"governed_public_knowledge",
			"citizen_ai_assistant",
			"closed_loop_tracking",
			"statcollect_participation_bridge",
		},
		"status": "active",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("fabric_registration: marshal error: %v", err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}

	// Initial registration attempt with retry
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest(http.MethodPost, regURL, bytes.NewBuffer(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			if cfg.InternalAPIKey != "" {
				req.Header.Set("X-Internal-API-Key", cfg.InternalAPIKey)
			}
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode < 300 {
					log.Printf("fabric_registration: successfully registered StatCitizen with Enterprise Core at %s", regURL)
					break
				}
			}
		}
		time.Sleep(3 * time.Second)
	}

	// Periodic heartbeat
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		req, err := http.NewRequest(http.MethodPost, regURL, bytes.NewBuffer(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			if cfg.InternalAPIKey != "" {
				req.Header.Set("X-Internal-API-Key", cfg.InternalAPIKey)
			}
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
			}
		}
	}
}

// ─── HelpDesk Integration ─────────────────────────────────────────────────────

// createHelpDeskTicket forwards a citizen report to HelpDesk API (:5006)
// to instantiate an operational institutional case.
func createHelpDeskTicket(cfg *Config, reportID, canonicalID, title, description, category, priority, severity, district, facilityID, corrID string) {
	if cfg.HelpDeskAPIURL == "" {
		return
	}

	url := fmt.Sprintf("%s/api/tickets", cfg.HelpDeskAPIURL)
	payload := map[string]interface{}{
		"title":          fmt.Sprintf("[Citizen Report] %s", title),
		"description":    fmt.Sprintf("%s\n\n---\nSource: StatCitizen\nCanonical ID: %s\nDistrict: %s\nFacility ID: %s\nCorrelation ID: %s", description, canonicalID, district, facilityID, corrID),
		"category":       category,
		"priority":       priority,
		"severity":       severity,
		"source":         "statcitizen",
		"external_id":    reportID,
		"correlation_id": corrID,
		"metadata": map[string]interface{}{
			"statcitizen_report_id": reportID,
			"canonical_id":          canonicalID,
			"district":              district,
			"facility_id":           facilityID,
		},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalAPIKey != "" {
		req.Header.Set("X-Internal-API-Key", cfg.InternalAPIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("helpdesk_integration: [NOTICE] HelpDesk ticket creation deferred (service offline or unreachable): %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var res struct {
			ID       string `json:"id"`
			TicketID string `json:"ticket_id"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		ticketID := res.ID
		if ticketID == "" {
			ticketID = res.TicketID
		}

		if ticketID != "" && dbPool != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, _ = dbPool.ExecContext(ctx,
				`UPDATE citizen_reports SET helpdesk_ticket_id=$2, updated_at=NOW() WHERE id=$1`,
				reportID, ticketID)
		}
		log.Printf("helpdesk_integration: linked HelpDesk ticket %s to report %s", ticketID, reportID)
	}
}

// ─── Universal Object Context ─────────────────────────────────────────────────

// UniversalObjectContext represents the standard enterprise 8-facet view.
type UniversalObjectContext struct {
	CanonicalID   string                 `json:"canonical_id"`
	Overview      map[string]interface{} `json:"overview"`
	Activity      []map[string]interface{} `json:"activity"`
	Relationships []map[string]interface{} `json:"relationships"`
	Documents     []map[string]interface{} `json:"documents"`
	Workflow      map[string]interface{} `json:"workflow"`
	Decisions     []map[string]interface{} `json:"decisions"`
	AIIntelligence map[string]interface{} `json:"ai_intelligence"`
	Audit         []map[string]interface{} `json:"audit"`
}

// buildUniversalContext constructs the 8-facet context for any StatCitizen canonical object.
func buildUniversalContext(canonicalID string, tenantID string) (*UniversalObjectContext, error) {
	ctx := &UniversalObjectContext{
		CanonicalID: canonicalID,
		Overview: map[string]interface{}{
			"canonical_id": canonicalID,
			"tenant_id":    tenantID,
			"system":       "statcitizen",
			"status":       "active",
		},
		Activity:      []map[string]interface{}{},
		Relationships: []map[string]interface{}{},
		Documents:     []map[string]interface{}{},
		Workflow: map[string]interface{}{
			"current_state": "active",
			"locked":        false,
		},
		Decisions: []map[string]interface{}{},
		AIIntelligence: map[string]interface{}{
			"summary":    "StatCitizen governed citizen evidence record.",
			"confidence": 0.95,
		},
		Audit: []map[string]interface{}{},
	}

	if dbPool != nil {
		cCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Fetch audit trail
		rows, err := dbPool.QueryContext(cCtx,
			`SELECT action, actor, actor_type, outcome, created_at
			 FROM citizen_audit_log
			 WHERE resource_id=$1 OR metadata->>'canonical_id'=$1
			 ORDER BY created_at DESC LIMIT 10`, canonicalID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var action, actor, actorType, outcome, createdAt string
				if err := rows.Scan(&action, &actor, &actorType, &outcome, &createdAt); err == nil {
					ctx.Audit = append(ctx.Audit, map[string]interface{}{
						"action":     action,
						"actor":      actor,
						"actor_type": actorType,
						"outcome":    outcome,
						"timestamp":  createdAt,
					})
				}
			}
		}
	}

	return ctx, nil
}
