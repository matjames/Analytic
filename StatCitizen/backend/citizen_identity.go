package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Citizen Identity Handlers ────────────────────────────────────────────────

// handleCreateSession creates a new anonymous citizen session.
// No PII required — this is the entry point for all citizen interactions.
func handleCreateSession(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)

	token, err := generateSessionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "session_error", Message: "Failed to create session"})
		return
	}

	id := fmt.Sprintf("cs_%d", time.Now().UnixNano())
	ipHash := hashIP(c.ClientIP())
	expiresAt := time.Now().Add(24 * time.Hour)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err = dbPool.ExecContext(ctx,
			`INSERT INTO citizen_sessions (id, token, tenant_id, ip_hash, user_agent, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			id, token, tenantID, ipHash, c.GetHeader("User-Agent"), expiresAt,
		)
		if err != nil {
			log.Printf("session: db error: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "session_error", Message: "Failed to persist session"})
			return
		}
	}

	// Audit
	recordCitizenAudit(c, cfg, "citizen", "session.created", "session", id, map[string]interface{}{
		"correlation_id": corrID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"session_id":  id,
		"token":       token,
		"expires_at":  expiresAt.Format(time.RFC3339),
		"tenant_id":   tenantID,
	})
}

// handleRegisterCitizen creates a persistent citizen profile.
// Data minimization: only stores what is explicitly provided.
func handleRegisterCitizen(c *gin.Context, cfg *Config) {
	var req struct {
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		OrgID       string `json:"org_id"`
		District    string `json:"district"`
		ConsentGranted bool `json:"consent_granted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}
	if !req.ConsentGranted {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "consent_required", Message: "Explicit consent is required to create a citizen profile."})
		return
	}

	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	id := fmt.Sprintf("cit_%d", time.Now().UnixNano())

	// Hash PII — never store raw
	emailHash := ""
	if req.Email != "" {
		emailHash = hashPII(req.Email)
	}
	phoneHash := ""
	if req.Phone != "" {
		phoneHash = hashPII(req.Phone)
	}

	citizen := RegisteredCitizen{
		ID:           id,
		TenantID:     tenantID,
		IdentityType: IdentityRegistered,
		DisplayName:  req.DisplayName,
		EmailHash:    emailHash,
		PhoneHash:    phoneHash,
		OrgID:        req.OrgID,
		District:     req.District,
		Status:       "active",
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO registered_citizens
			(id, tenant_id, identity_type, display_name, email_hash, phone_hash, org_id, district, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			citizen.ID, citizen.TenantID, citizen.IdentityType, citizen.DisplayName,
			citizen.EmailHash, citizen.PhoneHash, citizen.OrgID, citizen.District, citizen.Status,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error", Message: "Failed to register citizen"})
			return
		}
	}

	// Publish enterprise event
	publishCitizenEvent(EventCitizenRegistered, "citizen", id, tenantID, id, corrID, map[string]interface{}{
		"identity_type": string(citizen.IdentityType),
		"has_org":       citizen.OrgID != "",
		"district":      citizen.District,
	})

	recordCitizenAudit(c, cfg, "citizen", "citizen.registered", "citizen", id, nil)

	c.JSON(http.StatusCreated, gin.H{
		"id":            citizen.ID,
		"identity_type": citizen.IdentityType,
		"display_name":  citizen.DisplayName,
		"status":        citizen.Status,
		"created_at":    citizen.CreatedAt,
	})
}

// ─── Feedback Handlers ────────────────────────────────────────────────────────

func handleListFeedbackCategories(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	rows, err := queryFeedbackCategories(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": rows})
}

func handleSubmitFeedback(c *gin.Context, cfg *Config) {
	var req struct {
		CategoryID  string `json:"category_id" binding:"required"`
		Subject     string `json:"subject" binding:"required"`
		Description string `json:"description" binding:"required"`
		District    string `json:"district"`
		FacilityID  string `json:"facility_id"`
		ProjectID   string `json:"project_id"`
		ServiceID   string `json:"service_id"`
		Priority    string `json:"priority"`
		Anonymous   bool   `json:"anonymous"`
		ConsentGranted bool `json:"consent_granted"`
		Visibility  string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}
	if !req.ConsentGranted {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "consent_required", Message: "Consent is required to submit feedback."})
		return
	}

	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	session := getCitizenSession(c)

	id := fmt.Sprintf("fb_%d", time.Now().UnixNano())
	canonicalID := fmt.Sprintf("%s:statcitizen:feedback:%s", tenantID, id)
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = "institution"
	}

	// Record consent
	consentID := recordConsent(tenantID, getSessionID(session), "", "feedback_submission",
		[]string{"feedback_content", "location"}, visibility, "1_year", "internal", corrID)

	fb := FeedbackRecord{
		ID:          id,
		CanonicalID: canonicalID,
		TenantID:    tenantID,
		ConsentID:   consentID,
		CategoryID:  req.CategoryID,
		Subject:     req.Subject,
		Description: req.Description,
		District:    req.District,
		FacilityID:  req.FacilityID,
		ProjectID:   req.ProjectID,
		ServiceID:   req.ServiceID,
		Priority:    priority,
		Sensitivity: "internal",
		Status:      "submitted",
		Source:      "web",
		Anonymous:   req.Anonymous,
		Visibility:  visibility,
		CorrelationID: corrID,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if session != nil {
		fb.SessionID = session.ID
	}

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO feedback_records
			(id,canonical_id,tenant_id,session_id,consent_id,category_id,subject,description,
			 district,facility_id,project_id,service_id,priority,sensitivity,status,source,
			 anonymous,visibility,correlation_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
			fb.ID, fb.CanonicalID, fb.TenantID, fb.SessionID, fb.ConsentID,
			fb.CategoryID, fb.Subject, fb.Description, fb.District, fb.FacilityID,
			fb.ProjectID, fb.ServiceID, fb.Priority, fb.Sensitivity, fb.Status,
			fb.Source, fb.Anonymous, fb.Visibility, fb.CorrelationID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error", Message: "Failed to submit feedback"})
			return
		}
	}

	// Create citizen case for tracking
	createCitizenCase(tenantID, getSessionID(session), "", "feedback", id, "submitted",
		"Your feedback has been received.", corrID)

	// Publish enterprise event
	publishCitizenEvent(EventFeedbackCreated, "feedback", id, tenantID, getSessionID(session), corrID, map[string]interface{}{
		"canonical_id": canonicalID,
		"category_id":  req.CategoryID,
		"subject":      req.Subject,
		"priority":     priority,
		"district":     req.District,
		"facility_id":  req.FacilityID,
		"project_id":   req.ProjectID,
		"anonymous":    req.Anonymous,
	})

	recordCitizenAudit(c, cfg, "citizen", "feedback.submitted", "feedback", id, nil)

	c.JSON(http.StatusCreated, gin.H{
		"id":             id,
		"canonical_id":   canonicalID,
		"status":         "submitted",
		"correlation_id": corrID,
		"message":        "Your feedback has been received. Thank you.",
	})
}

func handleListFeedback(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	session := getCitizenSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "session_required"})
		return
	}

	page := parseIntDefault(c.Query("page"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	if dbPool == nil {
		c.JSON(http.StatusOK, PaginatedResponse{Data: []FeedbackRecord{}, Total: 0, Page: page, PerPage: perPage})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.QueryContext(ctx,
		`SELECT id,canonical_id,tenant_id,session_id,consent_id,category_id,subject,
		 description,district,facility_id,project_id,service_id,priority,sensitivity,
		 status,source,anonymous,visibility,correlation_id,created_at,updated_at
		 FROM feedback_records
		 WHERE tenant_id=$1 AND session_id=$2
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, session.ID, perPage, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	defer rows.Close()

	var records []FeedbackRecord
	for rows.Next() {
		var fb FeedbackRecord
		var sessionID, citizenID, consentID, catID, district, facilityID, projectID, serviceID *string
		_ = rows.Scan(&fb.ID, &fb.CanonicalID, &fb.TenantID, &sessionID, &consentID, &catID,
			&fb.Subject, &fb.Description, &district, &facilityID, &projectID, &serviceID,
			&fb.Priority, &fb.Sensitivity, &fb.Status, &fb.Source, &fb.Anonymous,
			&fb.Visibility, &fb.CorrelationID, &fb.CreatedAt, &fb.UpdatedAt)
		if sessionID != nil {
			fb.SessionID = *sessionID
		}
		if citizenID != nil {
			fb.CitizenID = *citizenID
		}
		if consentID != nil {
			fb.ConsentID = *consentID
		}
		if catID != nil {
			fb.CategoryID = *catID
		}
		if district != nil {
			fb.District = *district
		}
		if facilityID != nil {
			fb.FacilityID = *facilityID
		}
		if projectID != nil {
			fb.ProjectID = *projectID
		}
		if serviceID != nil {
			fb.ServiceID = *serviceID
		}
		records = append(records, fb)
	}
	if records == nil {
		records = []FeedbackRecord{}
	}

	var total int
	_ = dbPool.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM feedback_records WHERE tenant_id=$1 AND session_id=$2`,
		tenantID, session.ID,
	).Scan(&total)

	pages := (total + perPage - 1) / perPage
	c.JSON(http.StatusOK, PaginatedResponse{
		Data: records, Total: total, Page: page, PerPage: perPage, Pages: pages,
	})
}

// ─── Report Handlers ──────────────────────────────────────────────────────────

func handleListReportCategories(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	rows, err := queryReportCategories(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": rows})
}

func handleSubmitReport(c *gin.Context, cfg *Config) {
	var req struct {
		CategoryID      string   `json:"category_id" binding:"required"`
		Title           string   `json:"title" binding:"required"`
		Description     string   `json:"description" binding:"required"`
		LocationText    string   `json:"location_text"`
		District        string   `json:"district"`
		Subcounty       string   `json:"subcounty"`
		Parish          string   `json:"parish"`
		Latitude        *float64 `json:"latitude"`
		Longitude       *float64 `json:"longitude"`
		LocationConsent bool     `json:"location_consent"`
		FacilityID      string   `json:"facility_id"`
		ProjectID       string   `json:"project_id"`
		ServiceID       string   `json:"service_id"`
		Priority        string   `json:"priority"`
		Severity        string   `json:"severity"`
		Anonymous       bool     `json:"anonymous"`
		ContactPref     string   `json:"contact_preference"`
		ConsentGranted  bool     `json:"consent_granted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}
	if !req.ConsentGranted {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "consent_required", Message: "Consent is required to submit a report."})
		return
	}

	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	session := getCitizenSession(c)

	id := fmt.Sprintf("rpt_%d", time.Now().UnixNano())
	canonicalID := fmt.Sprintf("%s:statcitizen:report:%s", tenantID, id)

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}

	// Validate GPS only if consent given
	var lat, lon *float64
	if req.LocationConsent {
		lat = req.Latitude
		lon = req.Longitude
	}

	dataCategories := []string{"report_content", "location"}
	if !req.Anonymous && req.ContactPref != "none" {
		dataCategories = append(dataCategories, "contact_preference")
	}

	consentID := recordConsent(tenantID, getSessionID(session), "", "report_submission",
		dataCategories, "institution", "2_years", "internal", corrID)

	rpt := CitizenReport{
		ID:              id,
		CanonicalID:     canonicalID,
		TenantID:        tenantID,
		ConsentID:       consentID,
		CategoryID:      req.CategoryID,
		Title:           req.Title,
		Description:     req.Description,
		LocationText:    req.LocationText,
		District:        req.District,
		Subcounty:       req.Subcounty,
		Parish:          req.Parish,
		Latitude:        lat,
		Longitude:       lon,
		LocationConsent: req.LocationConsent,
		FacilityID:      req.FacilityID,
		ProjectID:       req.ProjectID,
		ServiceID:       req.ServiceID,
		Priority:        priority,
		Severity:        severity,
		Anonymous:       req.Anonymous,
		ContactPref:     req.ContactPref,
		Status:          "submitted",
		CorrelationID:   corrID,
		Source:          "web",
	}
	if session != nil {
		rpt.SessionID = session.ID
	}

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO citizen_reports
			(id,canonical_id,tenant_id,session_id,consent_id,category_id,title,description,
			 location_text,district,subcounty,parish,latitude,longitude,location_consent,
			 facility_id,project_id,service_id,priority,severity,anonymous,contact_preference,
			 status,source,correlation_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
			rpt.ID, rpt.CanonicalID, rpt.TenantID, rpt.SessionID, rpt.ConsentID,
			rpt.CategoryID, rpt.Title, rpt.Description, rpt.LocationText, rpt.District,
			rpt.Subcounty, rpt.Parish, rpt.Latitude, rpt.Longitude, rpt.LocationConsent,
			rpt.FacilityID, rpt.ProjectID, rpt.ServiceID, rpt.Priority, rpt.Severity,
			rpt.Anonymous, rpt.ContactPref, rpt.Status, rpt.Source, rpt.CorrelationID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error", Message: "Failed to submit report"})
			return
		}
	}

	// Create citizen case for tracking
	createCitizenCase(tenantID, getSessionID(session), "", "report", id, "submitted",
		"Your report has been received and is being processed.", corrID)

	// Publish enterprise event — triggers HelpDesk ticket creation via Enterprise Core workflow
	publishCitizenEvent(EventReportCreated, "report", id, tenantID, getSessionID(session), corrID, map[string]interface{}{
		"canonical_id":  canonicalID,
		"category_id":  req.CategoryID,
		"title":        req.Title,
		"district":     req.District,
		"facility_id":  req.FacilityID,
		"priority":     priority,
		"severity":     severity,
		"anonymous":    req.Anonymous,
		"has_location": req.LocationConsent && (lat != nil || req.LocationText != ""),
	})

	// Attempt to create HelpDesk ticket asynchronously
	go createHelpDeskTicket(cfg, id, canonicalID, req.Title, req.Description, req.CategoryID,
		priority, severity, req.District, req.FacilityID, corrID)

	recordCitizenAudit(c, cfg, "citizen", "report.submitted", "report", id, nil)

	c.JSON(http.StatusCreated, gin.H{
		"id":             id,
		"canonical_id":   canonicalID,
		"status":         "submitted",
		"correlation_id": corrID,
		"message":        "Your report has been received. You can track its status using your correlation ID.",
	})
}

func handleGetReport(c *gin.Context, cfg *Config) {
	id := c.Param("id")
	if dbPool == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := dbPool.QueryRowContext(ctx,
		`SELECT id,canonical_id,tenant_id,category_id,title,description,district,
		 facility_id,priority,severity,status,anonymous,correlation_id,created_at,updated_at
		 FROM citizen_reports WHERE id=$1 AND tenant_id=$2`,
		id, getTenantID(c, cfg),
	)
	var rpt CitizenReport
	var catID, district, facilityID *string
	if err := row.Scan(&rpt.ID, &rpt.CanonicalID, &rpt.TenantID, &catID, &rpt.Title,
		&rpt.Description, &district, &facilityID, &rpt.Priority, &rpt.Severity,
		&rpt.Status, &rpt.Anonymous, &rpt.CorrelationID, &rpt.CreatedAt, &rpt.UpdatedAt); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found"})
		return
	}
	if catID != nil {
		rpt.CategoryID = *catID
	}
	if district != nil {
		rpt.District = *district
	}
	if facilityID != nil {
		rpt.FacilityID = *facilityID
	}
	c.JSON(http.StatusOK, rpt)
}

// ─── Case Tracking Handlers ───────────────────────────────────────────────────

func handleGetCaseStatus(c *gin.Context, cfg *Config) {
	correlationID := c.Query("correlation_id")
	if correlationID == "" {
		correlationID = c.Param("correlation_id")
	}
	if correlationID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "correlation_id_required", Message: "Provide correlation_id to track your submission"})
		return
	}

	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := dbPool.QueryRowContext(ctx,
		`SELECT id,submission_type,submission_id,status,status_message,response,
		 correlation_id,created_at,updated_at,resolved_at
		 FROM citizen_cases WHERE correlation_id=$1 AND tenant_id=$2`,
		correlationID, getTenantID(c, cfg),
	)
	var cc CitizenCase
	var msg, resp, resolvedAt *string
	if err := row.Scan(&cc.ID, &cc.SubmissionType, &cc.SubmissionID, &cc.Status,
		&msg, &resp, &cc.CorrelationID, &cc.CreatedAt, &cc.UpdatedAt, &resolvedAt); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "No submission found with this tracking ID"})
		return
	}
	if msg != nil {
		cc.StatusMessage = *msg
	}
	if resp != nil {
		cc.Response = *resp
	}
	if resolvedAt != nil {
		cc.ResolvedAt = *resolvedAt
	}

	// Map status to citizen-friendly labels
	statusLabels := map[string]string{
		"submitted":    "Submitted — your submission has been received",
		"received":     "Received — your submission has been logged",
		"under_review": "Under Review — your submission is being reviewed",
		"assigned":     "Assigned — your submission has been assigned to a team",
		"investigating": "Investigating — an investigation is in progress",
		"action_taken": "Action Taken — action has been initiated",
		"resolved":     "Resolved — your submission has been resolved",
		"closed":       "Closed",
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              cc.ID,
		"submission_type": cc.SubmissionType,
		"status":          cc.Status,
		"status_label":    statusLabels[cc.Status],
		"status_message":  cc.StatusMessage,
		"response":        cc.Response,
		"correlation_id":  cc.CorrelationID,
		"created_at":      cc.CreatedAt,
		"updated_at":      cc.UpdatedAt,
		"resolved_at":     cc.ResolvedAt,
	})
}

func handleListMyCases(c *gin.Context, cfg *Config) {
	session := getCitizenSession(c)
	if session == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "session_required"})
		return
	}
	if dbPool == nil {
		c.JSON(http.StatusOK, gin.H{"cases": []interface{}{}})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.QueryContext(ctx,
		`SELECT id,submission_type,submission_id,status,status_message,correlation_id,created_at,updated_at
		 FROM citizen_cases WHERE session_id=$1 AND tenant_id=$2 ORDER BY created_at DESC LIMIT 50`,
		session.ID, getTenantID(c, cfg),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	defer rows.Close()
	var cases []CitizenCase
	for rows.Next() {
		var cc CitizenCase
		var msg *string
		_ = rows.Scan(&cc.ID, &cc.SubmissionType, &cc.SubmissionID, &cc.Status, &msg, &cc.CorrelationID, &cc.CreatedAt, &cc.UpdatedAt)
		if msg != nil {
			cc.StatusMessage = *msg
		}
		cases = append(cases, cc)
	}
	if cases == nil {
		cases = []CitizenCase{}
	}
	c.JSON(http.StatusOK, gin.H{"cases": cases})
}

// ─── Helper Functions ─────────────────────────────────────────────────────────

func recordConsent(tenantID, sessionID, citizenID, purpose string,
	dataCategories []string, visibility, retention, sensitivity, corrID string) string {
	id := fmt.Sprintf("cns_%d", time.Now().UnixNano())
	if dbPool == nil {
		return id
	}
	cats, _ := json.Marshal(dataCategories)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = dbPool.ExecContext(ctx,
		`INSERT INTO citizen_consents
		(id,tenant_id,session_id,citizen_id,purpose,data_categories,visibility,retention,
		 sensitivity,granted,granted_at,downstream_use)
		VALUES($1,$2,$3,$4,$5,$6::text[],$7,$8,$9,true,NOW(),'{}')`,
		id, tenantID, sessionID, citizenID, purpose, string(cats),
		visibility, retention, sensitivity,
	)
	return id
}

func createCitizenCase(tenantID, sessionID, citizenID, submissionType, submissionID, status, message, corrID string) string {
	id := fmt.Sprintf("cc_%d", time.Now().UnixNano())
	if dbPool == nil {
		return id
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = dbPool.ExecContext(ctx,
		`INSERT INTO citizen_cases (id,tenant_id,session_id,citizen_id,submission_type,submission_id,status,status_message,correlation_id)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, tenantID, sessionID, citizenID, submissionType, submissionID, status, message, corrID,
	)
	return id
}

func getSessionID(session *CitizenSession) string {
	if session == nil {
		return ""
	}
	return session.ID
}

func parseIntDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

func queryFeedbackCategories(tenantID string) ([]FeedbackCategory, error) {
	if dbPool == nil {
		return []FeedbackCategory{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := dbPool.QueryContext(ctx,
		`SELECT id,tenant_id,name,slug,description,active,sort_order
		 FROM feedback_categories
		 WHERE (tenant_id=$1 OR tenant_id='') AND active=true
		 ORDER BY sort_order`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []FeedbackCategory
	for rows.Next() {
		var cat FeedbackCategory
		var desc *string
		_ = rows.Scan(&cat.ID, &cat.TenantID, &cat.Name, &cat.Slug, &desc, &cat.Active, &cat.SortOrder)
		if desc != nil {
			cat.Description = *desc
		}
		cats = append(cats, cat)
	}
	if cats == nil {
		cats = []FeedbackCategory{}
	}
	return cats, nil
}

func queryReportCategories(tenantID string) ([]ReportCategory, error) {
	if dbPool == nil {
		return []ReportCategory{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := dbPool.QueryContext(ctx,
		`SELECT id,tenant_id,name,slug,description,active,sort_order
		 FROM report_categories
		 WHERE (tenant_id=$1 OR tenant_id='') AND active=true
		 ORDER BY sort_order`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []ReportCategory
	for rows.Next() {
		var cat ReportCategory
		var desc *string
		_ = rows.Scan(&cat.ID, &cat.TenantID, &cat.Name, &cat.Slug, &desc, &cat.Active, &cat.SortOrder)
		if desc != nil {
			cat.Description = *desc
		}
		cats = append(cats, cat)
	}
	if cats == nil {
		cats = []ReportCategory{}
	}
	return cats, nil
}

// handleUpdateCaseStatus is called internally (or by admin) to push status updates to citizens.
func handleUpdateCaseStatus(c *gin.Context, cfg *Config) {
	id := c.Param("id")
	var req struct {
		Status  string `json:"status" binding:"required"`
		Message string `json:"message"`
		Response string `json:"response"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	validStatuses := map[string]bool{
		"submitted": true, "received": true, "under_review": true,
		"assigned": true, "investigating": true, "action_taken": true,
		"resolved": true, "closed": true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_status"})
		return
	}

	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resolvedAt := ""
	if req.Status == "resolved" || req.Status == "closed" {
		resolvedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var result string
	if resolvedAt != "" {
		_, _ = dbPool.ExecContext(ctx,
			`UPDATE citizen_cases SET status=$2, status_message=$3, response=$4, resolved_at=$5, updated_at=NOW() WHERE id=$1`,
			id, req.Status, req.Message, req.Response, resolvedAt,
		)
	} else {
		_, _ = dbPool.ExecContext(ctx,
			`UPDATE citizen_cases SET status=$2, status_message=$3, response=$4, updated_at=NOW() WHERE id=$1`,
			id, req.Status, req.Message, req.Response,
		)
	}

	// Publish event
	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	publishCitizenEvent(EventCaseUpdated, "case", id, tenantID, getEnterpriseUser(c), corrID, map[string]interface{}{
		"status":   req.Status,
		"message":  req.Message,
		"response": req.Response,
	})

	recordCitizenAudit(c, cfg, "admin", "case.status_updated", "case", id, map[string]interface{}{
		"status": req.Status,
	})

	_ = result
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status, "updated": true})
}

// handleListReports lists reports for admin view (filtered by tenant)
func handleListReports(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	status := c.Query("status")
	district := c.Query("district")
	page := parseIntDefault(c.Query("page"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	if dbPool == nil {
		c.JSON(http.StatusOK, PaginatedResponse{Data: []CitizenReport{}, Total: 0, Page: page, PerPage: perPage})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT id,canonical_id,tenant_id,category_id,title,description,district,
		facility_id,priority,severity,status,anonymous,correlation_id,created_at,updated_at
		FROM citizen_reports WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if district != "" {
		query += fmt.Sprintf(" AND district=$%d", argIdx)
		args = append(args, district)
		argIdx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := dbPool.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	defer rows.Close()

	var reports []CitizenReport
	for rows.Next() {
		var rpt CitizenReport
		var catID, district2, facilityID *string
		_ = rows.Scan(&rpt.ID, &rpt.CanonicalID, &rpt.TenantID, &catID, &rpt.Title,
			&rpt.Description, &district2, &facilityID, &rpt.Priority, &rpt.Severity,
			&rpt.Status, &rpt.Anonymous, &rpt.CorrelationID, &rpt.CreatedAt, &rpt.UpdatedAt)
		if catID != nil {
			rpt.CategoryID = *catID
		}
		if district2 != nil {
			rpt.District = *district2
		}
		if facilityID != nil {
			rpt.FacilityID = *facilityID
		}
		reports = append(reports, rpt)
	}
	if reports == nil {
		reports = []CitizenReport{}
	}

	var total int
	countQ := `SELECT COUNT(*) FROM citizen_reports WHERE tenant_id=$1`
	countArgs := []interface{}{tenantID}
	if status != "" {
		countQ += " AND status=$2"
		countArgs = append(countArgs, status)
	}
	_ = dbPool.QueryRowContext(ctx, countQ, countArgs...).Scan(&total)

	pages := (total + perPage - 1) / perPage
	c.JSON(http.StatusOK, PaginatedResponse{Data: reports, Total: total, Page: page, PerPage: perPage, Pages: pages})
}

// handleListFeedbackAdmin lists all feedback records (admin only)
func handleListFeedbackAdmin(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	status := c.Query("status")
	page := parseIntDefault(c.Query("page"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	if dbPool == nil {
		c.JSON(http.StatusOK, PaginatedResponse{Data: []FeedbackRecord{}, Total: 0, Page: page, PerPage: perPage})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT id,canonical_id,tenant_id,session_id,category_id,subject,description,
		district,facility_id,priority,sensitivity,status,source,anonymous,visibility,
		correlation_id,created_at,updated_at
		FROM feedback_records WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argIdx := 2

	if status != "" {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, perPage, offset)

	rows, err := dbPool.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	defer rows.Close()

	var records []FeedbackRecord
	for rows.Next() {
		var fb FeedbackRecord
		var sessID, catID, district, facilityID *string
		_ = rows.Scan(&fb.ID, &fb.CanonicalID, &fb.TenantID, &sessID, &catID, &fb.Subject,
			&fb.Description, &district, &facilityID, &fb.Priority, &fb.Sensitivity, &fb.Status,
			&fb.Source, &fb.Anonymous, &fb.Visibility, &fb.CorrelationID, &fb.CreatedAt, &fb.UpdatedAt)
		if sessID != nil {
			fb.SessionID = *sessID
		}
		if catID != nil {
			fb.CategoryID = *catID
		}
		if district != nil {
			fb.District = *district
		}
		if facilityID != nil {
			fb.FacilityID = *facilityID
		}
		records = append(records, fb)
	}
	if records == nil {
		records = []FeedbackRecord{}
	}
	var total int
	_ = dbPool.QueryRowContext(ctx, `SELECT COUNT(*) FROM feedback_records WHERE tenant_id=$1`, tenantID).Scan(&total)
	pages := (total + perPage - 1) / perPage
	c.JSON(http.StatusOK, PaginatedResponse{Data: records, Total: total, Page: page, PerPage: perPage, Pages: pages})
}

// nowUTC returns current UTC time as RFC3339 string.
func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// dummy to use strings import
var _ = strings.Contains
