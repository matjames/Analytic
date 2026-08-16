package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"statfederation-backend/internal/diplomacy"
	"statfederation-backend/internal/engine"
	"statfederation-backend/internal/events"
	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// Handlers encapsulates all HTTP request controllers for App 6
type Handlers struct {
	store        store.Store
	fedEngine    *engine.FederationEngine
	queryRouter  *engine.QueryRouter
	harmonizer   *engine.Harmonizer
	diplomacyGW  *diplomacy.DiplomacyGateway
	compliance   *diplomacy.ComplianceEvaluator
	fedSearch    *diplomacy.FederatedSearchHub
	eventWorker  *events.EventWorker
}

// NewHandlers constructs API handlers
func NewHandlers(
	s store.Store,
	fe *engine.FederationEngine,
	qr *engine.QueryRouter,
	h *engine.Harmonizer,
	dg *diplomacy.DiplomacyGateway,
	ce *diplomacy.ComplianceEvaluator,
	fs *diplomacy.FederatedSearchHub,
	ew *events.EventWorker,
) *Handlers {
	return &Handlers{
		store:        s,
		fedEngine:    fe,
		queryRouter:  qr,
		harmonizer:   h,
		diplomacyGW:  dg,
		compliance:   ce,
		fedSearch:    fs,
		eventWorker:  ew,
	}
}

// ─── NSS Node Handlers ───────────────────────────────────────────────────────

func (h *Handlers) ListNodes(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	nodeType := c.Query("type")
	jurisdiction := c.Query("jurisdiction")

	nodes, err := h.store.ListNodes(c.Request.Context(), tenantIDStr, nodeType, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": nodes, "count": len(nodes)})
}

func (h *Handlers) RegisterNode(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var node models.FederatedNode
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}
	if node.TenantID == "" {
		node.TenantID = tenantIDStr
	}

	if err := h.fedEngine.RegisterNSSNode(c.Request.Context(), &node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Publish event
	_ = h.eventWorker.PublishFederationEvent(
		c.Request.Context(),
		events.EventNodeRegistered, "federated_node", node.ID, node.TenantID, "",
		map[string]interface{}{"node_name": node.Name, "code": node.Code, "type": node.NodeType},
	)

	c.JSON(http.StatusCreated, gin.H{"message": "Node registered successfully", "node": node})
}

func (h *Handlers) GetNode(c *gin.Context) {
	id := c.Param("id")
	node, err := h.store.GetNodeByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	c.JSON(http.StatusOK, node)
}

func (h *Handlers) NodeHeartbeat(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status    models.HealthStatus `json:"status"`
		LatencyMs int                 `json:"latency_ms"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Status = models.HealthStatusHealthy
		req.LatencyMs = 20
	}
	if err := h.fedEngine.ProcessHeartbeat(c.Request.Context(), id, req.Status, req.LatencyMs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "heartbeat_acknowledged", "node_id": id, "health": req.Status})
}

func (h *Handlers) ProbeNodes(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	nodes, err := h.fedEngine.ProbeAllNodes(c.Request.Context(), tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Probe completed across all federated participants", "nodes": nodes})
}

// ─── Distributed Query Handlers ─────────────────────────────────────────────

func (h *Handlers) DispatchQuery(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var req models.DistributedQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query payload: " + err.Error()})
		return
	}

	result, err := h.queryRouter.DispatchDistributedQuery(c.Request.Context(), req, userIDStr, tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Publish Event
	_ = h.eventWorker.PublishFederationEvent(
		c.Request.Context(),
		events.EventQueryDispatched, "distributed_query", result.QueryID, tenantIDStr, userIDStr,
		map[string]interface{}{
			"query_name":   result.QueryName,
			"target_nodes": result.TotalNodesTargeted,
			"records":      result.TotalRecords,
		},
	)

	c.JSON(http.StatusOK, result)
}

func (h *Handlers) ListQueries(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	queries, err := h.store.ListQueries(c.Request.Context(), tenantIDStr, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queries": queries, "count": len(queries)})
}

func (h *Handlers) GetQuery(c *gin.Context) {
	id := c.Param("id")
	q, err := h.store.GetQueryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Query record not found"})
		return
	}
	c.JSON(http.StatusOK, q)
}

// ─── Data Sharing Agreement Handlers ────────────────────────────────────────

func (h *Handlers) ListDSAs(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	status := c.Query("status")
	dsas, err := h.store.ListDSAs(c.Request.Context(), tenantIDStr, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data_sharing_agreements": dsas, "count": len(dsas)})
}

func (h *Handlers) CreateDSA(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var dsa models.DataSharingAgreement
	if err := c.ShouldBindJSON(&dsa); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSA payload: " + err.Error()})
		return
	}
	if dsa.TenantID == "" {
		dsa.TenantID = tenantIDStr
	}
	if dsa.ValidFrom.IsZero() {
		dsa.ValidFrom = time.Now().UTC()
	}
	if dsa.ValidUntil.IsZero() {
		dsa.ValidUntil = time.Now().UTC().Add(365 * 24 * time.Hour)
	}

	if err := h.store.CreateDSA(c.Request.Context(), &dsa); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Data sharing agreement drafted", "dsa": dsa})
}

func (h *Handlers) GetDSA(c *gin.Context) {
	id := c.Param("id")
	dsa, err := h.store.GetDSAByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DSA not found"})
		return
	}
	c.JSON(http.StatusOK, dsa)
}

func (h *Handlers) ApproveDSA(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	if err := h.fedEngine.ApproveDSA(c.Request.Context(), id, userIDStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.eventWorker.PublishFederationEvent(
		c.Request.Context(),
		events.EventDSAApproved, "data_sharing_agreement", id, "default", userIDStr,
		map[string]interface{}{"approved_by": userIDStr},
	)

	c.JSON(http.StatusOK, gin.H{"status": "approved", "dsa_id": id})
}

func (h *Handlers) RevokeDSA(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	if err := h.fedEngine.RevokeDSA(c.Request.Context(), id, userIDStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.eventWorker.PublishFederationEvent(
		c.Request.Context(),
		events.EventDSARevoked, "data_sharing_agreement", id, "default", userIDStr,
		map[string]interface{}{"revoked_by": userIDStr},
	)

	c.JSON(http.StatusOK, gin.H{"status": "revoked", "dsa_id": id})
}

// ─── National Indicator Handlers ────────────────────────────────────────────

func (h *Handlers) ListIndicators(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	domain := c.Query("domain")
	inds, err := h.store.ListIndicators(c.Request.Context(), tenantIDStr, domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"national_indicators": inds, "count": len(inds)})
}

func (h *Handlers) CreateIndicator(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var ind models.NationalIndicator
	if err := c.ShouldBindJSON(&ind); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid indicator payload: " + err.Error()})
		return
	}
	if ind.TenantID == "" {
		ind.TenantID = tenantIDStr
	}

	if err := h.store.CreateIndicator(c.Request.Context(), &ind); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Indicator registered in national registry", "indicator": ind})
}

func (h *Handlers) GetIndicator(c *gin.Context) {
	id := c.Param("id")
	ind, err := h.store.GetIndicatorByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Indicator not found"})
		return
	}
	c.JSON(http.StatusOK, ind)
}

func (h *Handlers) UpdateIndicatorValue(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		CurrentValue float64 `json:"current_value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateIndicatorValues(c.Request.Context(), id, &req.CurrentValue); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated", "indicator_id": id, "current_value": req.CurrentValue})
}

func (h *Handlers) OfficialStatisticsCalendar(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	inds, err := h.store.ListIndicators(c.Request.Context(), tenantIDStr, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	calendar := make([]map[string]interface{}, 0)
	for _, ind := range inds {
		calendar = append(calendar, map[string]interface{}{
			"indicator_code": ind.Code,
			"title":          ind.Title,
			"domain":         ind.Domain,
			"frequency":      ind.Frequency,
			"lead_agency":    ind.LeadAgencyName,
			"status":         "ON_SCHEDULE",
			"target_release": ind.CalendarReleaseDate,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"official_statistics_calendar": calendar,
		"total_scheduled":              len(calendar),
	})
}

// ─── Metadata Harmonizer Handlers ───────────────────────────────────────────

func (h *Handlers) HarmonizePayload(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var req struct {
		SourceAgency      string                 `json:"source_agency" binding:"required"`
		StandardFramework string                 `json:"standard_framework"`
		RawRecord         map[string]interface{} `json:"raw_record" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid harmonization request: " + err.Error()})
		return
	}

	res, err := h.harmonizer.HarmonizeRecord(c.Request.Context(), req.SourceAgency, req.StandardFramework, req.RawRecord, tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handlers) ListVocabularies(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	agency := c.Query("agency")
	vocabs, err := h.store.ListVocabularies(c.Request.Context(), tenantIDStr, agency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vocabularies": vocabs, "count": len(vocabs)})
}

func (h *Handlers) CreateVocabulary(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var v models.MetadataVocabulary
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vocabulary payload: " + err.Error()})
		return
	}
	if v.TenantID == "" {
		v.TenantID = tenantIDStr
	}

	if err := h.store.CreateVocabulary(c.Request.Context(), &v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Vocabulary mapping created", "vocabulary": v})
}

// ─── Global & Digital Diplomacy Handlers ────────────────────────────────────

func (h *Handlers) ListTreaties(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	jurisdiction := c.Query("jurisdiction")
	treaties, err := h.store.ListTreaties(c.Request.Context(), tenantIDStr, jurisdiction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"treaties": treaties, "count": len(treaties)})
}

func (h *Handlers) RegisterTreaty(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var t models.DiplomaticTreaty
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid treaty payload: " + err.Error()})
		return
	}
	if t.TenantID == "" {
		t.TenantID = tenantIDStr
	}

	if err := h.diplomacyGW.RegisterTreaty(c.Request.Context(), &t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Diplomatic treaty registered", "treaty": t})
}

func (h *Handlers) GetTreaty(c *gin.Context) {
	id := c.Param("id")
	t, err := h.store.GetTreatyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Treaty not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *Handlers) GenerateSDGReport(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	var req struct {
		Period string `json:"period"`
	}
	_ = c.ShouldBindJSON(&req)

	rep, err := h.diplomacyGW.GenerateSDGSubmission(c.Request.Context(), req.Period, tenantIDStr, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = h.eventWorker.PublishFederationEvent(
		c.Request.Context(),
		events.EventDiplomacyReportSent, "international_report", rep.ID, tenantIDStr, userIDStr,
		map[string]interface{}{"destination": rep.DestinationBody, "period": rep.ReportingPeriod},
	)

	c.JSON(http.StatusOK, gin.H{"message": "SDG submission dossier generated and transmitted", "report": rep})
}

func (h *Handlers) GenerateAUReport(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	var req struct {
		Period string `json:"period"`
	}
	_ = c.ShouldBindJSON(&req)

	rep, err := h.diplomacyGW.GenerateAUReport(c.Request.Context(), req.Period, tenantIDStr, userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "AU Agenda 2063 report compiled", "report": rep})
}

func (h *Handlers) ListInternationalReports(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	dest := c.Query("destination")
	reports, err := h.store.ListReports(c.Request.Context(), tenantIDStr, dest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"international_reports": reports, "count": len(reports)})
}

func (h *Handlers) GetInternationalReport(c *gin.Context) {
	id := c.Param("id")
	rep, err := h.store.GetReportByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}
	c.JSON(http.StatusOK, rep)
}

// ─── Federated Search Handlers ──────────────────────────────────────────────

func (h *Handlers) FederatedSearch(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	var q diplomacy.SearchQuery
	if err := c.ShouldBindJSON(&q); err != nil {
		q.Keyword = c.Query("q")
		q.ResourceType = c.Query("type")
	}

	results, err := h.fedSearch.ExecuteFederatedSearch(c.Request.Context(), q, tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

// ─── Transboundary Compliance Handlers ──────────────────────────────────────

func (h *Handlers) EvaluateCompliance(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	var req struct {
		SourceJurisdiction string                 `json:"source_jurisdiction" binding:"required"`
		TargetJurisdiction string                 `json:"target_jurisdiction" binding:"required"`
		ResourceType       string                 `json:"resource_type" binding:"required"`
		ResourceID         string                 `json:"resource_id" binding:"required"`
		Payload            map[string]interface{} `json:"payload" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evaluation payload: " + err.Error()})
		return
	}

	res, err := h.compliance.EvaluateCrossBorderTransmission(
		c.Request.Context(), userIDStr, tenantIDStr,
		req.SourceJurisdiction, req.TargetJurisdiction,
		req.ResourceType, req.ResourceID, req.Payload,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *Handlers) ListComplianceLogs(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.store.ListComplianceLogs(c.Request.Context(), tenantIDStr, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"compliance_logs": logs, "count": len(logs)})
}

// ─── Cross-Application Object Links ─────────────────────────────────────────

func (h *Handlers) ListObjectLinks(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	sourceType := c.Query("source_type")
	sourceID := c.Query("source_id")

	if sourceType == "" || sourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_type and source_id query parameters are required"})
		return
	}

	links, err := h.store.GetObjectLinks(c.Request.Context(), sourceType, sourceID, tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"object_links": links, "count": len(links)})
}

func (h *Handlers) CreateObjectLink(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)
	userID, _ := c.Get("user_id")
	userIDStr, _ := userID.(string)

	var link models.ObjectLink
	if err := c.ShouldBindJSON(&link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid object link: " + err.Error()})
		return
	}
	if link.TenantID == "" {
		link.TenantID = tenantIDStr
	}
	if link.CreatedBy == "" {
		link.CreatedBy = userIDStr
	}

	if err := h.store.CreateObjectLink(c.Request.Context(), &link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Object link created", "link": link})
}
