package main

import (
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/events"
	"net/http"
	"time"
)

type API struct {
	db        *sql.DB
	telemetry *TelemetryCollector
	bus       *events.EventBus
}

func actor(c *gin.Context) string {
	if v := c.GetString("user_id"); v != "" {
		return v
	}
	return "runops-system"
}
func tenantID(c *gin.Context) string {
	if v := c.GetString("tenant_id"); v != "" {
		return v
	}
	return "system"
}
func (a *API) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"service": "statgate-runops", "status": "healthy", "version": "1.0.0", "timestamp": time.Now().UTC()})
}
func (a *API) ready(c *gin.Context) {
	status := "ready"
	code := http.StatusOK
	if a.db == nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, gin.H{"status": status, "database": a.db != nil, "telemetry_targets": len(a.telemetry.targets)})
}
func (a *API) summary(c *gin.Context) {
	var open int
	if a.db != nil {
		_ = a.db.QueryRow("SELECT COUNT(*) FROM runops.alerts WHERE status='open'").Scan(&open)
	}
	c.JSON(http.StatusOK, gin.H{"open_alerts": open, "targets": a.telemetry.targets, "components": []string{"cicd", "cloud", "clusters", "mesh", "scheduler", "cost", "ioc", "observability", "cmdb", "sre", "status", "dr"}})
}
func (a *API) targets(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"targets": a.telemetry.targets}) }
func (a *API) probe(c *gin.Context) {
	a.telemetry.ProbeAll(c.Request.Context())
	c.JSON(http.StatusAccepted, gin.H{"status": "probe_started"})
}
func (a *API) listAlerts(c *gin.Context) {
	rows, err := a.db.Query("SELECT id,fingerprint,severity,status,source,summary,starts_at FROM runops.alerts ORDER BY starts_at DESC LIMIT 100")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	out := []Alert{}
	for rows.Next() {
		var v Alert
		_ = rows.Scan(&v.ID, &v.Fingerprint, &v.Severity, &v.Status, &v.Source, &v.Summary, &v.StartsAt)
		out = append(out, v)
	}
	c.JSON(200, gin.H{"alerts": out})
}
func (a *API) acknowledgeAlert(c *gin.Context) {
	id := c.Param("id")
	r, err := a.db.Exec("UPDATE runops.alerts SET status='acknowledged',acknowledged_by=$1,acknowledged_at=NOW() WHERE id=$2", actor(c), id)
	n, _ := r.RowsAffected()
	if err != nil || n == 0 {
		c.JSON(404, gin.H{"error": "alert not found"})
		return
	}
	publish(a.bus, "runops.alert.acknowledged", "alert", id, tenantID(c), map[string]any{"actor": actor(c)})
	c.Status(204)
}
func (a *API) ingestLog(c *gin.Context) {
	var v TelemetryEnvelope
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	message := fmt.Sprint(v.Body["message"])
	severity := fmt.Sprint(v.Body["severity"])
	if severity == "" {
		severity = "INFO"
	}
	at := v.Timestamp
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := a.db.Exec("INSERT INTO runops.log_entries(service_id,severity,message,attributes,observed_at) VALUES($1,$2,$3,$4::jsonb,$5)", v.ServiceID, severity, message, jsonString(v.Attributes), at)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Status(202)
}
func (a *API) ingestMetric(c *gin.Context) {
	var v TelemetryEnvelope
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	name := fmt.Sprint(v.Body["name"])
	var value float64
	_, _ = fmt.Sscan(fmt.Sprint(v.Body["value"]), &value)
	at := v.Timestamp
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := a.db.Exec("INSERT INTO runops.metric_samples(service_id,metric_name,value,labels,observed_at) VALUES($1,$2,$3,$4::jsonb,$5)", v.ServiceID, name, value, jsonString(v.Attributes), at)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Status(202)
}
func (a *API) ingestTrace(c *gin.Context) {
	var v TelemetryEnvelope
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	traceID, spanID := fmt.Sprint(v.Body["trace_id"]), fmt.Sprint(v.Body["span_id"])
	at := v.Timestamp
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := a.db.Exec("INSERT INTO runops.trace_spans(trace_id,span_id,parent_span_id,service_id,operation,started_at,duration_ms,status,attributes) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb) ON CONFLICT DO NOTHING", traceID, spanID, fmt.Sprint(v.Body["parent_span_id"]), v.ServiceID, fmt.Sprint(v.Body["operation"]), at, v.Body["duration_ms"], fmt.Sprint(v.Body["status"]), jsonString(v.Attributes))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Status(202)
}
func (a *API) createDeployment(c *gin.Context) {
	var v struct {
		PipelineName string         `json:"pipeline_name" binding:"required"`
		ServiceID    string         `json:"service_id" binding:"required"`
		Environment  string         `json:"environment" binding:"required"`
		Revision     string         `json:"revision" binding:"required"`
		Metadata     map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := fmt.Sprintf("dep-%d", time.Now().UnixNano())
	_, err := a.db.Exec("INSERT INTO runops.deployments(id,pipeline_name,service_id,environment,revision,requested_by,metadata) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb)", id, v.PipelineName, v.ServiceID, v.Environment, v.Revision, actor(c), jsonString(v.Metadata))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	publish(a.bus, "runops.deployment.queued", "deployment", id, tenantID(c), map[string]any{"service_id": v.ServiceID, "revision": v.Revision})
	c.JSON(202, gin.H{"id": id, "status": "queued"})
}
func (a *API) createResource(c *gin.Context) {
	var v map[string]any
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := fmt.Sprintf("res-%d", time.Now().UnixNano())
	_, err := a.db.Exec("INSERT INTO runops.cloud_resources(id,provider,account_id,region,resource_type,name,tags) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb)", id, v["provider"], v["account_id"], v["region"], v["resource_type"], v["name"], jsonString(v["tags"]))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	publish(a.bus, "runops.resource.managed", "cloud_resource", id, tenantID(c), v)
	c.JSON(201, gin.H{"id": id})
}
func (a *API) createCluster(c *gin.Context) {
	var v map[string]any
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	id := fmt.Sprintf("clu-%d", time.Now().UnixNano())
	_, err := a.db.Exec("INSERT INTO runops.clusters(id,name,provider,region,version,policy) VALUES($1,$2,$3,$4,$5,$6::jsonb)", id, v["name"], v["provider"], v["region"], v["version"], jsonString(v["policy"]))
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	publish(a.bus, "runops.cluster.provisioning", "cluster", id, tenantID(c), v)
	c.JSON(202, gin.H{"id": id, "status": "provisioning"})
}
