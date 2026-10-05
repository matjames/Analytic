package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type draftItem struct {
	ID            string
	TenantID      string
	SessionID     string
	DraftType     string
	PayloadJSON   []byte
	CorrelationID string
}

// ProcessPendingOfflineDrafts processes pending offline submission drafts.
// Returns the number of successfully processed drafts.
func ProcessPendingOfflineDrafts(ctx context.Context) (int, error) {
	if dbPool == nil {
		return 0, nil
	}

	qCtx, qCancel := context.WithTimeout(ctx, 10*time.Second)
	defer qCancel()

	rows, err := dbPool.QueryContext(qCtx,
		`SELECT id, tenant_id, session_id, draft_type, payload, correlation_id
		 FROM offline_drafts
		 WHERE status = 'pending' AND retry_count < 5
		 ORDER BY created_at ASC LIMIT 20`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var drafts []draftItem
	for rows.Next() {
		var d draftItem
		if err := rows.Scan(&d.ID, &d.TenantID, &d.SessionID, &d.DraftType, &d.PayloadJSON, &d.CorrelationID); err == nil {
			drafts = append(drafts, d)
		}
	}

	processedCount := 0
	for _, d := range drafts {
		uCtx, uCancel := context.WithTimeout(ctx, 8*time.Second)
		err := dispatchDraft(uCtx, dbPool, d)
		if err != nil {
			log.Printf("offline_sync: failed to process draft %s (%s): %v", d.ID, d.DraftType, err)
			_, _ = dbPool.ExecContext(uCtx,
				`UPDATE offline_drafts
				 SET retry_count = retry_count + 1,
				     last_error = $2,
				     status = CASE WHEN retry_count + 1 >= 5 THEN 'failed' ELSE 'pending' END,
				     updated_at = NOW()
				 WHERE id = $1`, d.ID, err.Error())
		} else {
			_, _ = dbPool.ExecContext(uCtx,
				`UPDATE offline_drafts
				 SET status = 'completed',
				     last_error = NULL,
				     updated_at = NOW()
				 WHERE id = $1`, d.ID)
			processedCount++
			log.Printf("offline_sync: successfully processed and synced draft %s (%s)", d.ID, d.DraftType)
		}
		uCancel()
	}

	return processedCount, nil
}

// dispatchDraft routes an offline draft payload to the appropriate table and emits events.
func dispatchDraft(ctx context.Context, db *sql.DB, d draftItem) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(d.PayloadJSON, &payload); err != nil {
		return fmt.Errorf("malformed draft payload JSON: %w", err)
	}

	tenantID := d.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	sessionID := d.SessionID
	corrID := d.CorrelationID
	if corrID == "" {
		corrID = fmt.Sprintf("sync_%d", time.Now().UnixNano())
	}

	switch d.DraftType {
	case "feedback":
		return syncFeedbackDraft(ctx, db, tenantID, sessionID, corrID, payload)
	case "report":
		return syncReportDraft(ctx, db, tenantID, sessionID, corrID, payload)
	case "rating":
		return syncRatingDraft(ctx, db, tenantID, sessionID, corrID, payload)
	case "consultation_response":
		return syncConsultationDraft(ctx, db, tenantID, sessionID, corrID, payload)
	default:
		return fmt.Errorf("unsupported draft_type: %s", d.DraftType)
	}
}

func syncFeedbackDraft(ctx context.Context, db *sql.DB, tenantID, sessionID, corrID string, p map[string]interface{}) error {
	fbID := fmt.Sprintf("fb_sync_%d", time.Now().UnixNano())
	canonicalID := fmt.Sprintf("%s:statcitizen:feedback:%s", tenantID, fbID)

	categoryID, _ := p["category_id"].(string)
	subject, _ := p["subject"].(string)
	description, _ := p["description"].(string)
	district, _ := p["district"].(string)
	facilityID, _ := p["facility_id"].(string)
	projectID, _ := p["project_id"].(string)
	serviceID, _ := p["service_id"].(string)
	priority, _ := p["priority"].(string)
	if priority == "" {
		priority = "medium"
	}
	visibility, _ := p["visibility"].(string)
	if visibility == "" {
		visibility = "institution"
	}
	anonymous, _ := p["anonymous"].(bool)

	if subject == "" || description == "" {
		return fmt.Errorf("feedback draft missing required subject or description")
	}

	consentID := recordConsent(tenantID, sessionID, "", "feedback_submission",
		[]string{"feedback_content", "location"}, visibility, "1_year", "internal", corrID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO feedback_records
		 (id, canonical_id, tenant_id, session_id, consent_id, category_id, subject, description,
		  district, facility_id, project_id, service_id, priority, sensitivity, status, source,
		  anonymous, visibility, correlation_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,'internal','submitted','offline_sync',$14,$15,$16)`,
		fbID, canonicalID, tenantID, sessionID, consentID, categoryID, subject, description,
		district, facilityID, projectID, serviceID, priority, anonymous, visibility, corrID)
	if err != nil {
		return fmt.Errorf("db insert feedback: %w", err)
	}

	createCitizenCase(tenantID, sessionID, "", "feedback", fbID, "submitted",
		"Your offline feedback has been synchronized.", corrID)

	publishCitizenEvent(EventFeedbackCreated, "feedback", fbID, tenantID, sessionID, corrID, map[string]interface{}{
		"canonical_id": canonicalID,
		"category_id":  categoryID,
		"subject":      subject,
		"priority":     priority,
		"offline_sync": true,
	})
	return nil
}

func syncReportDraft(ctx context.Context, db *sql.DB, tenantID, sessionID, corrID string, p map[string]interface{}) error {
	rptID := fmt.Sprintf("rpt_sync_%d", time.Now().UnixNano())
	canonicalID := fmt.Sprintf("%s:statcitizen:report:%s", tenantID, rptID)

	categoryID, _ := p["category_id"].(string)
	title, _ := p["title"].(string)
	description, _ := p["description"].(string)
	locationText, _ := p["location_text"].(string)
	district, _ := p["district"].(string)
	priority, _ := p["priority"].(string)
	if priority == "" {
		priority = "medium"
	}
	severity, _ := p["severity"].(string)
	if severity == "" {
		severity = "moderate"
	}
	anonymous, _ := p["anonymous"].(bool)
	locationConsent, _ := p["location_consent"].(bool)

	var lat, lon *float64
	if latVal, ok := p["latitude"].(float64); ok && locationConsent {
		lat = &latVal
	}
	if lonVal, ok := p["longitude"].(float64); ok && locationConsent {
		lon = &lonVal
	}

	if title == "" || description == "" {
		return fmt.Errorf("report draft missing required title or description")
	}

	consentID := recordConsent(tenantID, sessionID, "", "incident_reporting",
		[]string{"report_details", "location"}, "institution", "3_years", "internal", corrID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO citizen_reports
		 (id, canonical_id, tenant_id, session_id, consent_id, category_id, title, description,
		  location_text, district, latitude, longitude, location_consent, priority, severity,
		  anonymous, status, source, correlation_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,'submitted','offline_sync',$17)`,
		rptID, canonicalID, tenantID, sessionID, consentID, categoryID, title, description,
		locationText, district, lat, lon, locationConsent, priority, severity, anonymous, corrID)
	if err != nil {
		return fmt.Errorf("db insert report: %w", err)
	}

	if lat != nil && lon != nil {
		geoID := fmt.Sprintf("geo_%d", time.Now().UnixNano())
		_, _ = db.ExecContext(ctx,
			`INSERT INTO geo_locations (id, tenant_id, entity_type, entity_id, latitude, longitude, address)
			 VALUES ($1,$2,'report',$3,$4,$5,$6)`,
			geoID, tenantID, rptID, *lat, *lon, locationText)
	}

	createCitizenCase(tenantID, sessionID, "", "report", rptID, "submitted",
		"Your offline report has been synchronized and logged.", corrID)

	publishCitizenEvent(EventReportCreated, "report", rptID, tenantID, sessionID, corrID, map[string]interface{}{
		"canonical_id": canonicalID,
		"category_id":  categoryID,
		"title":        title,
		"priority":     priority,
		"offline_sync": true,
	})
	return nil
}

func syncRatingDraft(ctx context.Context, db *sql.DB, tenantID, sessionID, corrID string, p map[string]interface{}) error {
	ratID := fmt.Sprintf("rat_sync_%d", time.Now().UnixNano())
	canonicalID := fmt.Sprintf("%s:statcitizen:rating:%s", tenantID, ratID)

	serviceID, _ := p["service_id"].(string)
	serviceName, _ := p["service_name"].(string)
	if serviceName == "" {
		serviceName = serviceID
	}
	comment, _ := p["comment"].(string)
	anonymous, _ := p["anonymous"].(bool)

	overallScore := 0.0
	if s, ok := p["overall_score"].(float64); ok {
		overallScore = s
	}

	ratingsJSON := "{}"
	if r, ok := p["dimension_scores"]; ok {
		if b, err := json.Marshal(r); err == nil {
			ratingsJSON = string(b)
		}
	} else if r, ok := p["ratings"]; ok {
		if b, err := json.Marshal(r); err == nil {
			ratingsJSON = string(b)
		}
	}

	if serviceID == "" {
		return fmt.Errorf("rating draft missing required service_id")
	}

	consentID := recordConsent(tenantID, sessionID, "", "service_rating",
		[]string{"rating_score", "service_feedback"}, "public", "3_years", "public", corrID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO service_ratings
		 (id, canonical_id, tenant_id, session_id, consent_id, service_id, service_name,
		  ratings, overall_score, comment, anonymous, source, correlation_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,'offline_sync',$12)`,
		ratID, canonicalID, tenantID, sessionID, consentID, serviceID, serviceName,
		ratingsJSON, overallScore, comment, anonymous, corrID)
	if err != nil {
		return fmt.Errorf("db insert rating: %w", err)
	}

	publishCitizenEvent(EventServiceRatingCreated, "service_rating", ratID, tenantID, sessionID, corrID, map[string]interface{}{
		"service_id":    serviceID,
		"overall_score": overallScore,
		"offline_sync":  true,
	})
	return nil
}

func syncConsultationDraft(ctx context.Context, db *sql.DB, tenantID, sessionID, corrID string, p map[string]interface{}) error {
	crID := fmt.Sprintf("cr_sync_%d", time.Now().UnixNano())

	consultID, _ := p["consultation_id"].(string)
	comment, _ := p["comment"].(string)
	anonymous, _ := p["anonymous"].(bool)

	answersJSON := "{}"
	if a, ok := p["answers"]; ok {
		if b, err := json.Marshal(a); err == nil {
			answersJSON = string(b)
		}
	}

	if consultID == "" {
		return fmt.Errorf("consultation draft missing required consultation_id")
	}

	consentID := recordConsent(tenantID, sessionID, "", "consultation_participation",
		[]string{"consultation_response"}, "institution", "5_years", "internal", corrID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO consultation_responses
		 (id, tenant_id, consultation_id, session_id, consent_id, anonymous, answers, comment, correlation_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9)`,
		crID, tenantID, consultID, sessionID, consentID, anonymous, answersJSON, comment, corrID)
	if err != nil {
		return fmt.Errorf("db insert consultation response: %w", err)
	}

	_, _ = db.ExecContext(ctx,
		`UPDATE consultations SET response_count=response_count+1, participant_count=participant_count+1 WHERE id=$1`,
		consultID)

	publishCitizenEvent(EventConsultationResponded, "consultation_response", crID, tenantID, sessionID, corrID, map[string]interface{}{
		"consultation_id": consultID,
		"offline_sync":    true,
	})
	return nil
}

// startOfflineSyncWorker periodically processes pending offline submission drafts.
func startOfflineSyncWorker() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if dbPool == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, _ = ProcessPendingOfflineDrafts(ctx)
		cancel()
	}
}
