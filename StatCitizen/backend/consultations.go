package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Consultation Handlers ────────────────────────────────────────────────────

// handleListConsultations returns all published consultations (public endpoint).
func handleListConsultations(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	page := parseIntDefault(c.Query("page"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	if dbPool == nil {
		c.JSON(http.StatusOK, PaginatedResponse{Data: []Consultation{}, Total: 0, Page: page, PerPage: perPage})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.QueryContext(ctx,
		`SELECT id,canonical_id,tenant_id,institution_id,title,description,category,
		 open_date,close_date,status,allow_anonymous,require_verified,
		 participant_count,response_count,published_at,created_at
		 FROM consultations
		 WHERE tenant_id=$1 AND status='published' AND close_date > NOW()
		 ORDER BY open_date DESC LIMIT $2 OFFSET $3`,
		tenantID, perPage, offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}
	defer rows.Close()

	var consultations []Consultation
	for rows.Next() {
		var cons Consultation
		var instID, category, publishedAt *string
		_ = rows.Scan(&cons.ID, &cons.CanonicalID, &cons.TenantID, &instID, &cons.Title,
			&cons.Description, &category, &cons.OpenDate, &cons.CloseDate,
			&cons.Status, &cons.AllowAnonymous, &cons.RequireVerified,
			&cons.ParticipantCount, &cons.ResponseCount, &publishedAt, &cons.CreatedAt)
		if instID != nil {
			cons.InstitutionID = *instID
		}
		if category != nil {
			cons.Category = *category
		}
		if publishedAt != nil {
			cons.PublishedAt = *publishedAt
		}
		consultations = append(consultations, cons)
	}
	if consultations == nil {
		consultations = []Consultation{}
	}

	var total int
	_ = dbPool.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM consultations WHERE tenant_id=$1 AND status='published' AND close_date > NOW()`,
		tenantID).Scan(&total)

	pages := (total + perPage - 1) / perPage
	c.JSON(http.StatusOK, PaginatedResponse{Data: consultations, Total: total, Page: page, PerPage: perPage, Pages: pages})
}

// handleGetConsultation returns a single consultation with its questions.
func handleGetConsultation(c *gin.Context, cfg *Config) {
	id := c.Param("id")
	tenantID := getTenantID(c, cfg)

	if dbPool == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := dbPool.QueryRowContext(ctx,
		`SELECT id,canonical_id,tenant_id,institution_id,title,description,category,
		 open_date,close_date,status,allow_anonymous,require_verified,statcollect_form_id,
		 participant_count,response_count,published_at,created_at,updated_at
		 FROM consultations WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
	)

	var cons Consultation
	var instID, category, formID, publishedAt *string
	if err := row.Scan(&cons.ID, &cons.CanonicalID, &cons.TenantID, &instID, &cons.Title,
		&cons.Description, &category, &cons.OpenDate, &cons.CloseDate, &cons.Status,
		&cons.AllowAnonymous, &cons.RequireVerified, &formID, &cons.ParticipantCount,
		&cons.ResponseCount, &publishedAt, &cons.CreatedAt, &cons.UpdatedAt); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "not_found"})
		return
	}
	if instID != nil {
		cons.InstitutionID = *instID
	}
	if category != nil {
		cons.Category = *category
	}
	if formID != nil {
		cons.StatCollectFormID = *formID
	}
	if publishedAt != nil {
		cons.PublishedAt = *publishedAt
	}

	// Load questions
	qRows, err := dbPool.QueryContext(ctx,
		`SELECT id,consultation_id,tenant_id,text,type,options,required,sort_order
		 FROM consultation_questions WHERE consultation_id=$1 ORDER BY sort_order`, id)
	if err == nil {
		defer qRows.Close()
		for qRows.Next() {
			var q ConsultationQuestion
			var opts []string
			_ = qRows.Scan(&q.ID, &q.ConsultID, &q.TenantID, &q.Text, &q.Type, &opts, &q.Required, &q.SortOrder)
			q.Options = opts
			cons.Questions = append(cons.Questions, q)
		}
	}

	c.JSON(http.StatusOK, cons)
}

// handleSubmitConsultationResponse submits a citizen response to a consultation.
func handleSubmitConsultationResponse(c *gin.Context, cfg *Config) {
	consultID := c.Param("id")
	var req struct {
		Answers        map[string]interface{} `json:"answers" binding:"required"`
		Comment        string                 `json:"comment"`
		Anonymous      bool                   `json:"anonymous"`
		ConsentGranted bool                   `json:"consent_granted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}
	if !req.ConsentGranted {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "consent_required"})
		return
	}

	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	session := getCitizenSession(c)

	// Verify consultation exists and is open
	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var status string
		var closeDate string
		err := dbPool.QueryRowContext(ctx,
			`SELECT status, close_date FROM consultations WHERE id=$1 AND tenant_id=$2`,
			consultID, tenantID,
		).Scan(&status, &closeDate)
		if err != nil {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "consultation_not_found"})
			return
		}
		if status != "published" {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "consultation_not_open", Message: "This consultation is not currently accepting responses."})
			return
		}
	}

	consentID := recordConsent(tenantID, getSessionID(session), "", "consultation_participation",
		[]string{"consultation_response"}, "institution", "5_years", "internal", corrID)

	id := fmt.Sprintf("cr_%d", time.Now().UnixNano())

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Check for duplicate submission from same session
		if session != nil {
			var existCount int
			_ = dbPool.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM consultation_responses WHERE consultation_id=$1 AND session_id=$2`,
				consultID, session.ID,
			).Scan(&existCount)
			if existCount > 0 {
				c.JSON(http.StatusConflict, ErrorResponse{Error: "duplicate_submission", Message: "You have already submitted a response to this consultation."})
				return
			}
		}

		answers := "{}"
		if req.Answers != nil {
			import_json_b, _ := answerJSON(req.Answers)
			answers = import_json_b
		}

		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO consultation_responses
			(id,tenant_id,consultation_id,session_id,consent_id,anonymous,answers,comment,correlation_id)
			VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9)`,
			id, tenantID, consultID, getSessionID(session), consentID,
			req.Anonymous, answers, req.Comment, corrID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
			return
		}

		// Update participation count
		_, _ = dbPool.ExecContext(ctx,
			`UPDATE consultations SET response_count=response_count+1, participant_count=participant_count+1 WHERE id=$1`,
			consultID,
		)
	}

	// Publish enterprise event
	publishCitizenEvent(EventConsultationResponded, "consultation_response", id, tenantID, getSessionID(session), corrID, map[string]interface{}{
		"consultation_id": consultID,
		"anonymous":       req.Anonymous,
		"has_comment":     req.Comment != "",
	})

	recordCitizenAudit(c, cfg, "citizen", "consultation.responded", "consultation", consultID, nil)

	c.JSON(http.StatusCreated, gin.H{
		"id":             id,
		"consultation_id": consultID,
		"status":         "submitted",
		"correlation_id": corrID,
		"message":        "Your response has been submitted. Thank you for participating.",
	})
}

// handleCreateConsultation creates a new consultation (admin only).
func handleCreateConsultation(c *gin.Context, cfg *Config) {
	var req struct {
		InstitutionID   string                  `json:"institution_id"`
		Title           string                  `json:"title" binding:"required"`
		Description     string                  `json:"description" binding:"required"`
		Category        string                  `json:"category"`
		OpenDate        string                  `json:"open_date" binding:"required"`
		CloseDate       string                  `json:"close_date" binding:"required"`
		AllowAnonymous  bool                    `json:"allow_anonymous"`
		RequireVerified bool                    `json:"require_verified"`
		StatCollectFormID string               `json:"statcollect_form_id"`
		Questions       []ConsultationQuestion  `json:"questions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}

	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	adminUser := getEnterpriseUser(c)
	id := fmt.Sprintf("cns_%d", time.Now().UnixNano())
	canonicalID := fmt.Sprintf("%s:statcitizen:consultation:%s", tenantID, id)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO consultations
			(id,canonical_id,tenant_id,institution_id,title,description,category,open_date,close_date,
			 status,allow_anonymous,require_verified,statcollect_form_id,created_by,correlation_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'draft',$10,$11,$12,$13,$14)`,
			id, canonicalID, tenantID, req.InstitutionID, req.Title, req.Description,
			req.Category, req.OpenDate, req.CloseDate, req.AllowAnonymous, req.RequireVerified,
			req.StatCollectFormID, adminUser, corrID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
			return
		}

		// Insert questions
		for i, q := range req.Questions {
			qID := fmt.Sprintf("cq_%d_%d", time.Now().UnixNano(), i)
			optJSON, _ := answerJSON(map[string]interface{}{"opts": q.Options})
			_, _ = dbPool.ExecContext(ctx,
				`INSERT INTO consultation_questions (id,consultation_id,tenant_id,text,type,required,sort_order)
				VALUES($1,$2,$3,$4,$5,$6,$7)`,
				qID, id, tenantID, q.Text, q.Type, q.Required, i+1,
			)
			_ = optJSON
		}
	}

	recordCitizenAudit(c, cfg, "admin", "consultation.created", "consultation", id, nil)

	c.JSON(http.StatusCreated, gin.H{
		"id":           id,
		"canonical_id": canonicalID,
		"status":       "draft",
		"created_at":   nowUTC(),
	})
}

// handlePublishConsultation publishes a draft consultation (admin only).
func handlePublishConsultation(c *gin.Context, cfg *Config) {
	id := c.Param("id")
	tenantID := getTenantID(c, cfg)
	adminUser := getEnterpriseUser(c)
	corrID := getCorrelationID(c)

	if dbPool == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "unavailable"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx,
		`UPDATE consultations SET status='published', published_by=$2, published_at=NOW(), updated_at=NOW()
		 WHERE id=$1 AND tenant_id=$3 AND status='draft'`,
		id, adminUser, tenantID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
		return
	}

	publishCitizenEvent(EventPublicationPublished, "consultation", id, tenantID, adminUser, corrID, map[string]interface{}{
		"consultation_id": id,
		"published_by":    adminUser,
	})

	recordCitizenAudit(c, cfg, "admin", "consultation.published", "consultation", id, nil)
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "published"})
}

// ─── Service Ratings Handlers ─────────────────────────────────────────────────

func handleListRatingDimensions(c *gin.Context, cfg *Config) {
	tenantID := getTenantID(c, cfg)
	defaultDims := []RatingDimension{
		{ID: "rd_satisfaction", Slug: "satisfaction", Label: "Overall Satisfaction", Description: "How satisfied were you overall?", MinScore: 1, MaxScore: 5, Active: true, SortOrder: 1},
		{ID: "rd_accessibility", Slug: "accessibility", Label: "Accessibility", Description: "How accessible was the service?", MinScore: 1, MaxScore: 5, Active: true, SortOrder: 2},
		{ID: "rd_wait_time", Slug: "wait_time", Label: "Waiting Time", Description: "How reasonable was the waiting time?", MinScore: 1, MaxScore: 5, Active: true, SortOrder: 3},
		{ID: "rd_availability", Slug: "availability", Label: "Service Availability", Description: "Was the service available when needed?", MinScore: 1, MaxScore: 5, Active: true, SortOrder: 4},
		{ID: "rd_staff", Slug: "staff", Label: "Staff Experience", Description: "How was your experience with staff?", MinScore: 1, MaxScore: 5, Active: true, SortOrder: 5},
		{ID: "rd_quality", Slug: "quality", Label: "Quality", Description: "How would you rate the quality?", MinScore: 1, MaxScore: 5, Active: true, SortOrder: 6},
		{ID: "rd_outcome", Slug: "outcome", Label: "Outcome", Description: "Did you achieve your intended outcome?", MinScore: 1, MaxScore: 5, Active: true, SortOrder: 7},
	}

	if dbPool == nil {
		c.JSON(http.StatusOK, gin.H{"dimensions": defaultDims})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := dbPool.QueryContext(ctx,
		`SELECT id,tenant_id,slug,label,description,min_score,max_score,active,sort_order
		 FROM rating_dimensions WHERE (tenant_id=$1 OR tenant_id='') AND active=true ORDER BY sort_order`,
		tenantID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"dimensions": defaultDims})
		return
	}
	defer rows.Close()
	var dims []RatingDimension
	for rows.Next() {
		var d RatingDimension
		var desc *string
		_ = rows.Scan(&d.ID, &d.TenantID, &d.Slug, &d.Label, &desc, &d.MinScore, &d.MaxScore, &d.Active, &d.SortOrder)
		if desc != nil {
			d.Description = *desc
		}
		dims = append(dims, d)
	}
	if len(dims) == 0 {
		dims = defaultDims
	}
	c.JSON(http.StatusOK, gin.H{"dimensions": dims})
}

func handleSubmitRating(c *gin.Context, cfg *Config) {
	var req struct {
		ServiceID      string         `json:"service_id" binding:"required"`
		ServiceName    string         `json:"service_name" binding:"required"`
		FacilityID     string         `json:"facility_id"`
		District       string         `json:"district"`
		Ratings        map[string]int `json:"ratings" binding:"required"`
		Comment        string         `json:"comment"`
		Anonymous      bool           `json:"anonymous"`
		ConsentGranted bool           `json:"consent_granted"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: err.Error()})
		return
	}
	if !req.ConsentGranted {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "consent_required"})
		return
	}
	if len(req.Ratings) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "ratings_required", Message: "At least one rating dimension is required."})
		return
	}

	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)
	session := getCitizenSession(c)

	// Compute overall score
	total := 0
	for _, v := range req.Ratings {
		total += v
	}
	overallScore := float64(total) / float64(len(req.Ratings))

	id := fmt.Sprintf("sr_%d", time.Now().UnixNano())
	canonicalID := fmt.Sprintf("%s:statcitizen:rating:%s", tenantID, id)
	consentID := recordConsent(tenantID, getSessionID(session), "", "service_rating",
		[]string{"rating_scores", "service_feedback"}, "institution", "3_years", "internal", corrID)

	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		ratingsJSON, _ := answerJSON(map[string]interface{}{"r": req.Ratings})
		_, err := dbPool.ExecContext(ctx,
			`INSERT INTO service_ratings
			(id,canonical_id,tenant_id,session_id,consent_id,service_id,service_name,facility_id,
			 district,ratings,overall_score,comment,anonymous,source,correlation_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11,$12,$13,'web',$14)`,
			id, canonicalID, tenantID, getSessionID(session), consentID,
			req.ServiceID, req.ServiceName, req.FacilityID, req.District,
			ratingsJSON, overallScore, req.Comment, req.Anonymous, corrID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db_error"})
			return
		}
	}

	publishCitizenEvent(EventServiceRatingCreated, "service_rating", id, tenantID, getSessionID(session), corrID, map[string]interface{}{
		"canonical_id":  canonicalID,
		"service_id":    req.ServiceID,
		"service_name":  req.ServiceName,
		"facility_id":   req.FacilityID,
		"district":      req.District,
		"overall_score": overallScore,
		"anonymous":     req.Anonymous,
	})

	recordCitizenAudit(c, cfg, "citizen", "rating.submitted", "service_rating", id, nil)

	c.JSON(http.StatusCreated, gin.H{
		"id":            id,
		"overall_score": overallScore,
		"status":        "submitted",
		"message":       "Thank you for rating this service.",
	})
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func answerJSON(data interface{}) (string, error) {
	import_json_b, err := marshalJSON(data)
	return import_json_b, err
}

func marshalJSON(v interface{}) (string, error) {
	import_json := jsonMarshal(v)
	return import_json, nil
}

func jsonMarshal(v interface{}) string {
	import_json_b, _ := import_json_marshal(v)
	return import_json_b
}

func import_json_marshal(v interface{}) (string, error) {
	import_json_b := fmt.Sprintf(`{}`)
	switch tv := v.(type) {
	case map[string]interface{}:
		// Use built-in encoding
		return formatJSONMap(tv), nil
	case map[string]int:
		return formatJSONMapInt(tv), nil
	}
	return import_json_b, nil
}

func formatJSONMap(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}
	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":%v`, k, formatVal(v))
		first = false
	}
	return result + "}"
}

func formatJSONMapInt(m map[string]int) string {
	if len(m) == 0 {
		return "{}"
	}
	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":%d`, k, v)
		first = false
	}
	return result + "}"
}

func formatVal(v interface{}) string {
	switch tv := v.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, tv)
	case bool:
		if tv {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", tv)
	case float64:
		return fmt.Sprintf("%g", tv)
	default:
		return fmt.Sprintf(`"%v"`, tv)
	}
}
