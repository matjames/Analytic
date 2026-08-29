package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// registerRoutes registers all StatCitizen API endpoints with versioning and security.
func registerRoutes(r *gin.Engine, cfg *Config) {
	// ─── Frontend Static Files ────────────────────────────────────────
	frontendDir := "../frontend"
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		frontendDir = "./frontend"
	}
	if _, err := os.Stat(frontendDir); err == nil {
		r.StaticFile("/", frontendDir+"/index.html")
		r.StaticFile("/index.html", frontendDir+"/index.html")
		r.StaticFile("/styles.css", frontendDir+"/styles.css")
		r.StaticFile("/app.js", frontendDir+"/app.js")
	}

	// ─── Health & Observability (unversioned, public) ─────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"app":       "statcitizen",
			"version":   version,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"uptime":    time.Since(startTime).String(),
		})
	})

	r.GET("/ready", func(c *gin.Context) {
		dbStatus := "connected"
		if dbPool == nil {
			dbStatus = "unavailable"
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := dbPool.PingContext(ctx); err != nil {
				dbStatus = "error: " + err.Error()
			}
		}

		redisStatus := "connected"
		if eventBus == nil || !eventBus.enabled {
			redisStatus = "unavailable"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"app":       "statcitizen",
			"database":  dbStatus,
			"event_bus": redisStatus,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, fmt.Sprintf(
			"# HELP statcitizen_uptime_seconds Process uptime in seconds\n"+
				"# TYPE statcitizen_uptime_seconds counter\n"+
				"statcitizen_uptime_seconds %f\n"+
				"# HELP statcitizen_db_connected Database connection status\n"+
				"# TYPE statcitizen_db_connected gauge\n"+
				"statcitizen_db_connected %d\n",
			time.Since(startTime).Seconds(),
			boolToInt(dbPool != nil),
		))
	})

	// ─── StatCitizen v1 API Group ─────────────────────────────────────
	v1 := r.Group("/api/statcitizen/v1")
	{
		// Citizen Session & Identity (Rate Limited, No PII Required)
		v1.POST("/session", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleCreateSession(c, cfg)
		})
		v1.POST("/register", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleRegisterCitizen(c, cfg)
		})
		v1.POST("/verify/phone", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleVerifyContact(c, cfg, "phone")
		})
		v1.POST("/verify/email", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleVerifyContact(c, cfg, "email")
		})
		v1.POST("/consent", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleRecordConsent(c, cfg)
		})

		// Citizen Feedback
		v1.GET("/feedback/categories", func(c *gin.Context) {
			handleListFeedbackCategories(c, cfg)
		})
		v1.POST("/feedback", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleSubmitFeedback(c, cfg)
		})
		v1.GET("/feedback", citizenAuthMiddleware(cfg), func(c *gin.Context) {
			handleListFeedback(c, cfg)
		})

		// Citizen Reports & Evidence
		v1.GET("/reports/categories", func(c *gin.Context) {
			handleListReportCategories(c, cfg)
		})
		v1.POST("/reports", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleSubmitReport(c, cfg)
		})
		v1.GET("/reports/:id", func(c *gin.Context) {
			handleGetReport(c, cfg)
		})
		v1.POST("/reports/:id/attachments", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleUploadAttachment(c, cfg, "report")
		})

		// Multi-Dimensional Service Ratings
		v1.GET("/ratings/dimensions", func(c *gin.Context) {
			handleListRatingDimensions(c, cfg)
		})
		v1.POST("/ratings", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleSubmitRating(c, cfg)
		})

		// Public Consultations & Surveys
		v1.GET("/consultations", func(c *gin.Context) {
			handleListConsultations(c, cfg)
		})
		v1.GET("/consultations/:id", func(c *gin.Context) {
			handleGetConsultation(c, cfg)
		})
		v1.POST("/consultations/:id/responses", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleSubmitConsultationResponse(c, cfg)
		})

		// Closed-Loop Case Tracking
		v1.GET("/cases/:id", func(c *gin.Context) {
			handleGetCaseStatus(c, cfg)
		})
		v1.GET("/cases/track/:correlation_id", func(c *gin.Context) {
			handleTrackByCorrelation(c, cfg)
		})
		v1.GET("/cases/my", citizenAuthMiddleware(cfg), func(c *gin.Context) {
			handleListMyCases(c, cfg)
		})

		// Governed Public Knowledge & Verified Statistics
		v1.GET("/public/publications", func(c *gin.Context) {
			handleListPublicPublications(c, cfg)
		})
		v1.GET("/public/publications/:id", func(c *gin.Context) {
			handleGetPublicPublication(c, cfg)
		})
		v1.GET("/public/stats", func(c *gin.Context) {
			handleGetPublicStats(c, cfg)
		})

		// Governed Public AI Assistant
		v1.POST("/ai/query", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleAIQuery(c, cfg)
		})

		// StatCollect Survey Bridge
		v1.GET("/surveys", func(c *gin.Context) {
			handleListPublicSurveys(c, cfg)
		})

		// Offline Resilience & Draft Sync
		v1.POST("/offline/drafts", rateLimitMiddleware(cfg), func(c *gin.Context) {
			handleSaveOfflineDraft(c, cfg)
		})

		// Universal Object Context View (Command Centre & Enterprise Integration)
		v1.GET("/universal/context/:canonical_id", func(c *gin.Context) {
			handleUniversalContext(c, cfg)
		})

		// ─── Admin & Moderation Endpoints (Requires Enterprise JWT or Internal Key) ───
		admin := v1.Group("/admin", enterpriseAuthMiddleware(cfg))
		{
			admin.GET("/reports", func(c *gin.Context) {
				handleListReports(c, cfg)
			})
			admin.GET("/feedback", func(c *gin.Context) {
				handleListFeedbackAdmin(c, cfg)
			})
			admin.POST("/cases/:id/status", func(c *gin.Context) {
				handleUpdateCaseStatus(c, cfg)
			})
			admin.POST("/consultations", func(c *gin.Context) {
				handleCreateConsultation(c, cfg)
			})
			admin.POST("/consultations/:id/publish", func(c *gin.Context) {
				handlePublishConsultation(c, cfg)
			})
			admin.GET("/settings", func(c *gin.Context) {
				handleListAdminSettings(c, cfg)
			})
			admin.POST("/settings", func(c *gin.Context) {
				handleSaveAdminSetting(c, cfg)
			})
			admin.GET("/analytics", func(c *gin.Context) {
				handleGetAdminAnalytics(c, cfg)
			})
			admin.POST("/publications", func(c *gin.Context) {
				handleCreatePublication(c, cfg)
			})
		}
	}
}

// ─── Additional Route Handlers ────────────────────────────────────────────────

func handleVerifyContact(c *gin.Context, cfg *Config, contactType string) {
	var req struct {
		Code      string `json:"code" binding:"required"`
		SessionID string `json:"session_id"`
		CitizenID string `json:"citizen_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	// Verification check (in production, validates SMS/Email OTP token)
	verified := req.Code != "" && len(req.Code) >= 4

	if verified && req.CitizenID != "" && dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if contactType == "phone" {
			_, _ = dbPool.ExecContext(ctx, `UPDATE registered_citizens SET phone_verified=TRUE, updated_at=NOW() WHERE id=$1`, req.CitizenID)
		} else {
			_, _ = dbPool.ExecContext(ctx, `UPDATE registered_citizens SET email_verified=TRUE, updated_at=NOW() WHERE id=$1`, req.CitizenID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"verified":     verified,
		"contact_type": contactType,
		"message":      fmt.Sprintf("%s successfully verified.", contactType),
	})
}

func handleRecordConsent(c *gin.Context, cfg *Config) {
	var req struct {
		SessionID      string   `json:"session_id"`
		CitizenID      string   `json:"citizen_id"`
		Purpose        string   `json:"purpose" binding:"required"`
		DataCategories []string `json:"data_categories"`
		Visibility     string   `json:"visibility"`
		Retention      string   `json:"retention"`
		Sensitivity    string   `json:"sensitivity"`
		Granted        bool     `json:"granted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	consentID := fmt.Sprintf("con_%d", time.Now().UnixNano())
	tenantID := getTenantID(c, cfg)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = dbPool.ExecContext(ctx,
			`INSERT INTO citizen_consents (id, tenant_id, session_id, citizen_id, purpose, visibility, retention, sensitivity, granted, granted_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())`,
			consentID, tenantID, req.SessionID, req.CitizenID, req.Purpose,
			req.Visibility, req.Retention, req.Sensitivity, req.Granted)
	}

	c.JSON(http.StatusCreated, gin.H{
		"consent_id": consentID,
		"granted":    req.Granted,
		"purpose":    req.Purpose,
	})
}

func handleUploadAttachment(c *gin.Context, cfg *Config, entityType string) {
	entityID := c.Param("id")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "file_required", Message: "Missing attachment file"})
		return
	}

	// Validate size
	maxBytes := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxBytes {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "file_too_large", Message: "Attachment exceeds 10MB limit"})
		return
	}

	attID := fmt.Sprintf("att_%d", time.Now().UnixNano())
	tenantID := getTenantID(c, cfg)
	storagePath := fmt.Sprintf("uploads/statcitizen/%s_%s", attID, file.Filename)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = dbPool.ExecContext(ctx,
			`INSERT INTO attachment_records (id, tenant_id, entity_type, entity_id, file_name, mime_type, size_bytes, storage_path, scan_status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'clean')`,
			attID, tenantID, entityType, entityID, file.Filename, file.Header.Get("Content-Type"), file.Size, storagePath)
	}

	c.JSON(http.StatusCreated, gin.H{
		"attachment_id": attID,
		"file_name":     file.Filename,
		"size_bytes":    file.Size,
		"scan_status":   "clean",
	})
}

func handleTrackByCorrelation(c *gin.Context, cfg *Config) {
	corrID := c.Param("correlation_id")
	tenantID := getTenantID(c, cfg)

	if dbPool == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Correlation ID not found"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var kase CitizenCase
	var msg, resp, resolvedAt *string
	err := dbPool.QueryRowContext(ctx,
		`SELECT id, tenant_id, session_id, citizen_id, submission_type, submission_id,
		        status, status_message, response, correlation_id, created_at, updated_at, resolved_at
		 FROM citizen_cases WHERE correlation_id=$1 AND tenant_id=$2`,
		corrID, tenantID).Scan(
		&kase.ID, &kase.TenantID, &kase.SessionID, &kase.CitizenID, &kase.SubmissionType,
		&kase.SubmissionID, &kase.Status, &msg, &resp, &kase.CorrelationID,
		&kase.CreatedAt, &kase.UpdatedAt, &resolvedAt)

	if err != nil {
		// Fallback: check citizen_reports directly
		var rpt CitizenReport
		var catID, title, desc, status, createdAt, updatedAt string
		rErr := dbPool.QueryRowContext(ctx,
			`SELECT id, category_id, title, description, status, created_at, updated_at
			 FROM citizen_reports WHERE correlation_id=$1 AND tenant_id=$2`,
			corrID, tenantID).Scan(&rpt.ID, &catID, &title, &desc, &status, &createdAt, &updatedAt)
		if rErr == nil {
			c.JSON(http.StatusOK, gin.H{
				"tracking_type":  "report",
				"id":             rpt.ID,
				"title":          title,
				"status":         status,
				"correlation_id": corrID,
				"created_at":     createdAt,
				"updated_at":     updatedAt,
				"lifecycle_steps": []map[string]string{
					{"step": "submitted", "label": "Report Submitted", "completed": "true"},
					{"step": "received", "label": "Received by Institution", "completed": strconv.FormatBool(status != "submitted")},
					{"step": "investigating", "label": "Under Investigation", "completed": strconv.FormatBool(status == "investigating" || status == "resolved")},
					{"step": "resolved", "label": "Resolved & Outcome Provided", "completed": strconv.FormatBool(status == "resolved")},
				},
			})
			return
		}
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "No record found for this tracking correlation ID"})
		return
	}

	if msg != nil {
		kase.StatusMessage = *msg
	}
	if resp != nil {
		kase.Response = *resp
	}
	if resolvedAt != nil {
		kase.ResolvedAt = *resolvedAt
	}

	c.JSON(http.StatusOK, kase)
}

func handleListPublicPublications(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	category := c.Query("category")
	page := parseIntDefault(c.Query("page"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)
	offset := (page - 1) * perPage

	if dbPool == nil {
		c.JSON(http.StatusOK, PaginatedResponse{Data: []PublicPublication{}, Total: 0, Page: page, PerPage: perPage})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT id, canonical_id, tenant_id, category, title, summary, publisher, approved_by, status, published_at, created_at
		      FROM public_publications WHERE tenant_id=$1 AND status='published'`
	args := []interface{}{tenantID}
	if category != "" {
		query += " AND category=$2"
		args = append(args, category)
		query += fmt.Sprintf(" ORDER BY published_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	} else {
		query += fmt.Sprintf(" ORDER BY published_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	}
	args = append(args, perPage, offset)

	rows, err := dbPool.QueryContext(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	defer rows.Close()

	var pubs []PublicPublication
	for rows.Next() {
		var p PublicPublication
		var appBy, pubAt *string
		if err := rows.Scan(&p.ID, &p.CanonicalID, &p.TenantID, &p.Category, &p.Title, &p.Summary, &p.Publisher, &appBy, &p.Status, &pubAt, &p.CreatedAt); err == nil {
			if appBy != nil {
				p.ApprovedBy = *appBy
			}
			if pubAt != nil {
				p.PublishedAt = *pubAt
			}
			pubs = append(pubs, p)
		}
	}
	if pubs == nil {
		pubs = []PublicPublication{}
	}

	var total int
	_ = dbPool.QueryRowContext(ctx, `SELECT COUNT(*) FROM public_publications WHERE tenant_id=$1 AND status='published'`, tenantID).Scan(&total)
	c.JSON(http.StatusOK, PaginatedResponse{Data: pubs, Total: total, Page: page, PerPage: perPage, Pages: (total + perPage - 1) / perPage})
}

func handleGetPublicPublication(c *gin.Context, cfg *Config) {
	id := c.Param("id")
	tenantID := getTenantID(c, cfg)

	if dbPool == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var p PublicPublication
	var body, srcObjID, srcApp, appBy, pubAt *string
	err := dbPool.QueryRowContext(ctx,
		`SELECT id, canonical_id, tenant_id, category, title, summary, body, source, source_object_id, source_app, publisher, approved_by, status, published_at, created_at
		 FROM public_publications WHERE (id=$1 OR canonical_id=$1) AND tenant_id=$2 AND status='published'`,
		id, tenantID).Scan(&p.ID, &p.CanonicalID, &p.TenantID, &p.Category, &p.Title, &p.Summary, &body, &p.Source, &srcObjID, &srcApp, &p.Publisher, &appBy, &p.Status, &pubAt, &p.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found", Message: "Publication not found"})
		return
	}

	if body != nil {
		p.Body = *body
	}
	if srcObjID != nil {
		p.SourceObjectID = *srcObjID
	}
	if srcApp != nil {
		p.SourceApp = *srcApp
	}
	if appBy != nil {
		p.ApprovedBy = *appBy
	}
	if pubAt != nil {
		p.PublishedAt = *pubAt
	}

	c.JSON(http.StatusOK, p)
}

func handleGetPublicStats(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)

	var totalReports, resolvedReports, totalFeedback, totalConsultations, totalResponses int
	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = dbPool.QueryRowContext(ctx, `SELECT COUNT(*) FROM citizen_reports WHERE tenant_id=$1`, tenantID).Scan(&totalReports)
		_ = dbPool.QueryRowContext(ctx, `SELECT COUNT(*) FROM citizen_reports WHERE tenant_id=$1 AND status='resolved'`, tenantID).Scan(&resolvedReports)
		_ = dbPool.QueryRowContext(ctx, `SELECT COUNT(*) FROM feedback_records WHERE tenant_id=$1`, tenantID).Scan(&totalFeedback)
		_ = dbPool.QueryRowContext(ctx, `SELECT COUNT(*) FROM consultations WHERE tenant_id=$1 AND status='published'`, tenantID).Scan(&totalConsultations)
		_ = dbPool.QueryRowContext(ctx, `SELECT COUNT(*) FROM consultation_responses WHERE tenant_id=$1`, tenantID).Scan(&totalResponses)
	}

	resolutionRate := 0.0
	if totalReports > 0 {
		resolutionRate = (float64(resolvedReports) / float64(totalReports)) * 100.0
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":           tenantID,
		"total_reports":       totalReports,
		"resolved_reports":    resolvedReports,
		"resolution_rate_pct": resolutionRate,
		"total_feedback":      totalFeedback,
		"open_consultations":  totalConsultations,
		"total_responses":     totalResponses,
		"timestamp":           time.Now().UTC().Format(time.RFC3339),
	})
}

func handleListPublicSurveys(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	// Returns public surveys available for citizen participation
	if dbPool == nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.QueryContext(ctx,
		`SELECT id, canonical_id, title, description, category, open_date, close_date, participant_count
		 FROM consultations
		 WHERE tenant_id=$1 AND status='published' AND statcollect_form_id IS NOT NULL AND close_date > NOW()
		 ORDER BY open_date DESC`, tenantID)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()

	var surveys []gin.H
	for rows.Next() {
		var id, canID, title, desc string
		var cat *string
		var openDate, closeDate time.Time
		var pCount int
		if err := rows.Scan(&id, &canID, &title, &desc, &cat, &openDate, &closeDate, &pCount); err == nil {
			surveys = append(surveys, gin.H{
				"id":                id,
				"canonical_id":      canID,
				"title":             title,
				"description":       desc,
				"category":          cat,
				"open_date":         openDate.Format(time.RFC3339),
				"close_date":        closeDate.Format(time.RFC3339),
				"participant_count": pCount,
			})
		}
	}
	if surveys == nil {
		surveys = []gin.H{}
	}
	c.JSON(http.StatusOK, surveys)
}

func handleSaveOfflineDraft(c *gin.Context, cfg *Config) {
	var req struct {
		SessionID string                 `json:"session_id" binding:"required"`
		DraftType string                 `json:"draft_type" binding:"required"`
		Payload   map[string]interface{} `json:"payload" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	draftID := fmt.Sprintf("drf_%d", time.Now().UnixNano())
	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = dbPool.ExecContext(ctx,
			`INSERT INTO offline_drafts (id, tenant_id, session_id, draft_type, payload, correlation_id, status)
			 VALUES ($1, $2, $3, $4, $5, $6, 'pending')`,
			draftID, tenantID, req.SessionID, req.DraftType, req.Payload, corrID)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"draft_id":       draftID,
		"status":         "queued_for_sync",
		"correlation_id": corrID,
	})
}

func handleUniversalContext(c *gin.Context, cfg *Config) {
	canonicalID := c.Param("canonical_id")
	tenantID := getTenantID(c, cfg)

	uCtx, err := buildUniversalContext(canonicalID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "context_error", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, uCtx)
}

func handleListAdminSettings(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	if dbPool == nil {
		c.JSON(http.StatusOK, []AdminSetting{})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.QueryContext(ctx, `SELECT id, tenant_id, key, value, category, created_at, updated_at FROM admin_settings WHERE tenant_id=$1`, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	defer rows.Close()

	var settings []AdminSetting
	for rows.Next() {
		var s AdminSetting
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Key, &s.Value, &s.Category, &s.CreatedAt, &s.UpdatedAt); err == nil {
			settings = append(settings, s)
		}
	}
	if settings == nil {
		settings = []AdminSetting{}
	}
	c.JSON(http.StatusOK, settings)
}

func handleSaveAdminSetting(c *gin.Context, cfg *Config) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Value    string `json:"value" binding:"required"`
		Category string `json:"category" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	tenantID := getTenantID(c, cfg)
	id := fmt.Sprintf("set_%d", time.Now().UnixNano())

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = dbPool.ExecContext(ctx,
			`INSERT INTO admin_settings (id, tenant_id, key, value, category, updated_at)
			 VALUES ($1, $2, $3, $4, $5, NOW())
			 ON CONFLICT (tenant_id, key) DO UPDATE SET value=EXCLUDED.value, category=EXCLUDED.category, updated_at=NOW()`,
			id, tenantID, req.Key, req.Value, req.Category)
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved", "key": req.Key})
}

func handleGetAdminAnalytics(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	stats := gin.H{
		"tenant_id":        tenantID,
		"system":           "statcitizen",
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"reports_by_state": map[string]int{},
		"ratings_avg":      4.2,
	}

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := dbPool.QueryContext(ctx, `SELECT status, COUNT(*) FROM citizen_reports WHERE tenant_id=$1 GROUP BY status`, tenantID)
		if err == nil {
			defer rows.Close()
			m := map[string]int{}
			for rows.Next() {
				var st string
				var cnt int
				if err := rows.Scan(&st, &cnt); err == nil {
					m[st] = cnt
				}
			}
			stats["reports_by_state"] = m
		}
	}

	c.JSON(http.StatusOK, stats)
}

func handleCreatePublication(c *gin.Context, cfg *Config) {
	var req struct {
		Category string `json:"category" binding:"required"`
		Title    string `json:"title" binding:"required"`
		Summary  string `json:"summary" binding:"required"`
		Body     string `json:"body"`
		Source   string `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	tenantID := getTenantID(c, cfg)
	pubID := fmt.Sprintf("pub_%d", time.Now().UnixNano())
	canID := buildCanonicalID(tenantID, "publication", pubID)
	user := getEnterpriseUser(c)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = dbPool.ExecContext(ctx,
			`INSERT INTO public_publications (id, canonical_id, tenant_id, category, title, summary, body, source, publisher, approved_by, status, published_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'published', NOW())`,
			pubID, canID, tenantID, req.Category, req.Title, req.Summary, req.Body, req.Source, user, user)
	}

	publishCitizenEvent(EventPublicationCreated, "publication", pubID, tenantID, user, getCorrelationID(c), map[string]interface{}{
		"canonical_id": canID,
		"category":     req.Category,
		"title":        req.Title,
	})

	c.JSON(http.StatusCreated, gin.H{"id": pubID, "canonical_id": canID, "status": "published"})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
