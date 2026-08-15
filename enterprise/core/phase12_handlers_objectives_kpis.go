package main

// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•
// PHASE XII â€” OBJECTIVES & KPI API HANDLERS
// â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func handleListObjectives(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	objs := phase12.ListObjectives(tenantID)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(objs), "objectives": objs})
}

func handleCreateObjective(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	var o InstitutionalObjective
	if err := c.ShouldBindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_objective", "message": err.Error()})
		return
	}
	o.TenantID = tenantID
	if err := phase12.UpsertObjective(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "objective_rejected", "message": err.Error()})
		return
	}
	recordAuditFromContext(c, "objective.create", "institutional_objectives", o.ID, map[string]interface{}{"name": o.Name})
	c.JSON(http.StatusCreated, gin.H{"id": o.ID, "canonical_id": o.CanonicalID, "status": "created"})
}

func handlePhase12ListKPIs(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	kpis := phase12.ListKPIs(tenantID)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(kpis), "kpis": kpis})
}

func handleCreateKPI(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	var k KPI
	if err := c.ShouldBindJSON(&k); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_kpi", "message": err.Error()})
		return
	}
	k.TenantID = tenantID
	if err := phase12.UpsertKPI(&k); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kpi_rejected", "message": err.Error()})
		return
	}
	recordAuditFromContext(c, "kpi.create", "kpis", k.ID, map[string]interface{}{"name": k.Name})
	c.JSON(http.StatusCreated, gin.H{"id": k.ID, "canonical_id": k.CanonicalID, "status": "created"})
}

func handleListKPIMeasurements(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	kpiID := c.Param("id")
	limit := parseIntDefault(c.Query("limit"), 50)
	meas := phase12.ListMeasurements(tenantID, kpiID, limit)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "kpi_id": kpiID, "count": len(meas), "measurements": meas})
}

func handleRecordKPIMeasurement(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	kpiID := c.Param("id")
	var body struct {
		Value      float64 `json:"value"`
		Unit       string  `json:"unit"`
		Source     string  `json:"source"`
		Estimated  bool    `json:"estimated"`
		MeasuredAt string  `json:"measured_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_measurement", "message": err.Error()})
		return
	}
	var measuredAt time.Time
	if body.MeasuredAt != "" {
		parsed, err := time.Parse(time.RFC3339, body.MeasuredAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_measured_at", "message": err.Error()})
			return
		}
		measuredAt = parsed
	}
	if err := phase12.RecordMeasurement(tenantID, kpiID, body.Value, body.Unit, body.Source, body.Estimated, measuredAt); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "measurement_rejected", "message": err.Error()})
		return
	}
	recordAuditFromContext(c, "kpi.measurement.recorded", "kpi_measurements", kpiID, map[string]interface{}{
		"value": body.Value, "estimated": body.Estimated,
	})
	c.JSON(http.StatusCreated, gin.H{"status": "measurement_recorded", "kpi_id": kpiID})
}
