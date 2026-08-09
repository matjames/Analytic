package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Enterprise Data Layer ──────────────────────────────────────────
// Consumes information from all approved systems and preserves the
// original source of every data element. The source application
// remains the system of record; this layer provides the intelligence
// and aggregation view.

var (
	dataLayerMu       sync.RWMutex
	enterpriseRecords = make(map[string]EnterpriseRecord) // key: source:entity:id
	datasets          = make(map[string]*Dataset)
)

const maxEnterpriseRecords = 10000

// ─── Dataset Registry ──────────────────────────────────────────────

func registerDataset(ds *Dataset) {
	dataLayerMu.Lock()
	if datasets[ds.ID] == nil {
		datasets[ds.ID] = ds
	} else {
		existing := datasets[ds.ID]
		existing.RecordCount = ds.RecordCount
		existing.LastUpdated = nowUTC()
		existing.UpdatedAt = nowUTC()
	}
	dataLayerMu.Unlock()
}

func getDataset(id string) *Dataset {
	dataLayerMu.RLock()
	defer dataLayerMu.RUnlock()
	return datasets[id]
}

func listDatasets() []Dataset {
	dataLayerMu.RLock()
	defer dataLayerMu.RUnlock()
	out := make([]Dataset, 0, len(datasets))
	for _, ds := range datasets {
		out = append(out, *ds)
	}
	return out
}

// ─── Enterprise Record Ingestion ───────────────────────────────────
// ingestEnterpriseRecord stores a record in the enterprise layer,
// preserving lineage to the source application.

func ingestEnterpriseRecord(rec EnterpriseRecord) {
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("er_%d", time.Now().UnixNano())
	}
	if rec.ReceivedAt == "" {
		rec.ReceivedAt = nowUTC()
	}
	if rec.Timestamp == "" {
		rec.Timestamp = nowUTC()
	}
	if rec.TenantID == "" {
		rec.TenantID = getEnvValue("STATGATE_TENANT_ID")
		if rec.TenantID == "" {
			rec.TenantID = "statgate"
		}
	}

	key := fmt.Sprintf("%s:%s:%s", rec.SourceApp, rec.SourceEntity, rec.SourceID)

	dataLayerMu.Lock()
	enterpriseRecords[key] = rec
	// Trim if map grows too large (simple LRU-ish behaviour)
	if len(enterpriseRecords) > maxEnterpriseRecords {
		i := 0
		for k := range enterpriseRecords {
			if i >= len(enterpriseRecords)-maxEnterpriseRecords {
				break
			}
			delete(enterpriseRecords, k)
			i++
		}
	}
	// Update dataset counters
	if ds := datasets[rec.SourceApp+":"+rec.SourceEntity]; ds != nil {
		ds.RecordCount++
		ds.LastUpdated = nowUTC()
	}
	dataLayerMu.Unlock()

	// Persist to Redis for recovery
	persistEnterpriseRecord(rec)
}

func persistEnterpriseRecord(rec EnterpriseRecord) {
	if redisClient == nil {
		return
	}
	data, _ := json.Marshal(rec)
	ctx := context.Background()
	key := fmt.Sprintf("statgate:enterprise:%s:%s:%s", rec.SourceApp, rec.SourceEntity, rec.SourceID)
	redisClient.Set(ctx, key, string(data), 7*24*time.Hour)
}

func fetchEnterpriseRecords(sourceApp, sourceEntity, projectID, orgID, region string, limit int) []EnterpriseRecord {
	dataLayerMu.RLock()
	defer dataLayerMu.RUnlock()

	out := make([]EnterpriseRecord, 0, limit)
	for _, rec := range enterpriseRecords {
		if sourceApp != "" && rec.SourceApp != sourceApp {
			continue
		}
		if sourceEntity != "" && rec.SourceEntity != sourceEntity {
			continue
		}
		if projectID != "" && rec.ProjectID != projectID {
			continue
		}
		if orgID != "" && rec.OrgID != orgID && rec.OrganizationID() != orgID {
			continue
		}
		if region != "" && rec.Region != region {
			continue
		}
		out = append(out, rec)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func fetchEnterpriseRecord(sourceApp, sourceEntity, sourceID string) (EnterpriseRecord, bool) {
	dataLayerMu.RLock()
	defer dataLayerMu.RUnlock()
	rec, ok := enterpriseRecords[fmt.Sprintf("%s:%s:%s", sourceApp, sourceEntity, sourceID)]
	return rec, ok
}

func (r EnterpriseRecord) OrganizationID() string {
	if r.OrgID != "" {
		return r.OrgID
	}
	if v, ok := r.Metadata["organization_id"].(string); ok {
		return v
	}
	if v, ok := r.Metadata["_msh_organization"].(string); ok {
		return v
	}
	return ""
}

// ─── Event → Enterprise Record Mapping ─────────────────────────────
// Converts domain events into structured enterprise records.

func eventToEnterpriseRecord(ev DomainEvent) *EnterpriseRecord {
	rec := &EnterpriseRecord{
		SourceApp:    ev.Source,
		SourceEntity: ev.ObjectType,
		SourceID:     ev.ObjectID,
		TenantID:     ev.TenantID,
		OrgID:        ev.OrgID,
		ProjectID:    ev.ProjectID,
		UserID:       ev.Actor,
		EventType:    ev.EventType,
		Correlation:  ev.Correlation,
		Timestamp:    ev.Timestamp,
		ReceivedAt:   nowUTC(),
		Metadata:     ev.Payload,
		DataVersion:  "1.0",
	}

	// Extract known metadata fields
	if p := ev.Payload; p != nil {
		if v, ok := p["project_id"].(string); ok && rec.ProjectID == "" {
			rec.ProjectID = v
		}
		if v, ok := p["_msh_project"].(string); ok && rec.ProjectID == "" {
			rec.ProjectID = v
		}
		if v, ok := p["organization_id"].(string); ok && rec.OrgID == "" {
			rec.OrgID = v
		}
		if v, ok := p["_msh_organization"].(string); ok && rec.OrgID == "" {
			rec.OrgID = v
		}
		if v, ok := p["region"].(string); ok {
			rec.Region = v
		}
		if v, ok := p["_msh_region"].(string); ok {
			rec.Region = v
		}
		if v, ok := p["district"].(string); ok {
			rec.District = v
		}
		if v, ok := p["_msh_district"].(string); ok {
			rec.District = v
		}
		if v, ok := p["facility_id"].(string); ok {
			rec.FacilityID = v
		}
		if v, ok := p["status"].(string); ok {
			rec.Status = v
		}
		// GPS extraction
		if v, ok := p["gps_coordinates"].(string); ok {
			lat, lng := parseGPSCoordinates(v)
			rec.GeoLat, rec.GeoLng = lat, lng
		}
		if v, ok := p["gps_lat"].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				rec.GeoLat = f
			}
		}
		if v, ok := p["gps_lng"].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				rec.GeoLng = f
			}
		}
		if v, ok := p["enumerator_id"].(string); ok {
			if rec.Metadata == nil {
				rec.Metadata = make(map[string]interface{})
			}
			rec.Metadata["enumerator_id"] = v
		}
		if v, ok := p["_msh_enumerator_id"].(string); ok {
			if rec.Metadata == nil {
				rec.Metadata = make(map[string]interface{})
			}
			rec.Metadata["enumerator_id"] = v
		}
	}
	return rec
}

// parseGPSCoordinates parses "lat,lng", "lat lng", or "lat, lng" strings.
func parseGPSCoordinates(s string) (float64, float64) {
	s = strings.TrimSpace(s)
	// Normalize: replace all spaces with commas, then collapse consecutive commas
	s = strings.ReplaceAll(s, " ", ",")
	for strings.Contains(s, ",,") {
		s = strings.ReplaceAll(s, ",,", ",")
	}
	parts := strings.Split(s, ",")
	if len(parts) < 2 {
		return 0, 0
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return lat, lng
}

// ─── Event-Driven Data Layer Processing ────────────────────────────
// Invoked from processEvent to feed the analytics layer in real time.

func processAnalyticsForEvent(ev DomainEvent) {
	// 1. Convert event to enterprise record
	rec := eventToEnterpriseRecord(ev)
	if rec != nil {
		ingestEnterpriseRecord(*rec)
	}

	// 2. Ensure dataset registry knows about this data source
	ensureDatasetForEvent(ev)

	// 3. Generate real-time analytics update (triggers KPI calc + alerts)
	go pushAnalyticsUpdate(ev)
}

// ensureDatasetForEvent registers a dataset entry for the event source/entity.
func ensureDatasetForEvent(ev DomainEvent) {
	dsID := fmt.Sprintf("%s:%s", ev.Source, ev.ObjectType)
	if getDataset(dsID) == nil {
		ds := &Dataset{
			ID:           dsID,
			Name:         fmt.Sprintf("%s %s", strings.ToUpper(ev.Source), ev.ObjectType),
			Description:  fmt.Sprintf("Enterprise records from %s (%s)", ev.Source, ev.ObjectType),
			SourceApp:    ev.Source,
			SourceEntity: ev.ObjectType,
			Fields:       []string{"id", "source_id", "project_id", "status", "timestamp", "region", "metadata"},
			RecordCount:  0,
			Status:       "active",
			TenantID:     getEnvValue("STATGATE_TENANT_ID"),
			CreatedAt:    nowUTC(),
			UpdatedAt:    nowUTC(),
		}
		if ds.TenantID == "" {
			ds.TenantID = "statgate"
		}
		registerDataset(ds)
	}
}

// ─── Analytics Update Pusher ───────────────────────────────────────
// Computes relevant KPIs for the event and publishes a live update
// to the SSE-style broker so dashboards refresh automatically.

func pushAnalyticsUpdate(ev DomainEvent) {
	alertContext := processAlertRulesForEvent(ev)

	update := map[string]interface{}{
		"event_type":   ev.EventType,
		"source":       ev.Source,
		"object_type":  ev.ObjectType,
		"object_id":    ev.ObjectID,
		"project_id":   ev.ProjectID,
		"timestamp":    nowUTC(),
		"alerts":       alertContext,
		"freshness":    computeFreshnessForEvent(ev),
		"data_quality": runQuickQualityCheck(ev),
		"kpi_refresh":  true,
	}
	publishAnalyticsEvent("analytics.update", update)
}

// publishAnalyticsEvent broadcasts an analytics event to connected clients.
// Uses Redis pub/sub so all StatGate components can consume it.
func publishAnalyticsEvent(eventType string, payload map[string]interface{}) {
	if redisClient == nil {
		return
	}
	evt := map[string]interface{}{
		"id":      fmt.Sprintf("an_%d", time.Now().UnixNano()),
		"type":    eventType,
		"payload": payload,
		"ts":      time.Now().Unix(),
	}
	data, _ := json.Marshal(evt)
	ctx := context.Background()
	redisClient.Publish(ctx, "statgate:analytics", string(data))
	// Store for analytics search
	redisClient.LPush(ctx, "statgate:analytics:history", string(data))
	redisClient.LTrim(ctx, "statgate:analytics:history", 0, 999)
}

// ─── Freshness Computation ─────────────────────────────────────────

func computeFreshnessForEvent(ev DomainEvent) map[string]interface{} {
	dsID := fmt.Sprintf("%s:%s", ev.Source, ev.ObjectType)
	ds := getDataset(dsID)
	lastUpdated := nowUTC()
	recordCount := int64(1)
	if ds != nil {
		lastUpdated = ds.LastUpdated
		recordCount = ds.RecordCount
	}
	return map[string]interface{}{
		"dataset_id":        dsID,
		"dataset_name":      fmt.Sprintf("%s %s", strings.ToUpper(ev.Source), ev.ObjectType),
		"source_app":        ev.Source,
		"last_updated":      lastUpdated,
		"record_count":      recordCount,
		"processing_status": "live",
		"is_fresh":          true,
		"age_seconds":       0,
	}
}

// ─── Quick Data Quality Check ──────────────────────────────────────
// Applies basic quality rules to a single event payload.

func runQuickQualityCheck(ev DomainEvent) map[string]interface{} {
	issues := []map[string]interface{}{}
	payload := ev.Payload

	// GPS check
	hasGPS := false
	if v, ok := payload["gps_coordinates"].(string); ok {
		lat, lng := parseGPSCoordinates(v)
		if lat != 0 && lng != 0 {
			hasGPS = true
		}
	}
	if v, ok := payload["gps_lat"].(string); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f != 0 {
			hasGPS = true
		}
	}
	if ev.Source == "statcollect" && !hasGPS {
		issues = append(issues, map[string]interface{}{
			"type": "missing_coordinates", "severity": "warning",
			"message": "Submission missing GPS coordinates",
		})
	}

	// Timestamp check
	if ts, ok := payload["timestamp"].(string); ok && ts != "" {
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			issues = append(issues, map[string]interface{}{
				"type": "invalid_date", "severity": "warning",
				"message": "Invalid timestamp format",
			})
		}
	}

	score := 100.0
	if len(issues) > 0 {
		score -= float64(len(issues)) * 5
	}
	return map[string]interface{}{
		"score":      math.Max(score, 0),
		"issues":     issues,
		"checked_at": nowUTC(),
	}
}

// ─── API Handlers ──────────────────────────────────────────────────

func handleListEnterpriseRecords(c *gin.Context) {
	sourceApp := c.Query("source")
	sourceEntity := c.Query("entity")
	projectID := c.Query("project_id")
	orgID := c.Query("organization_id")
	region := c.Query("region")
	limit := parseIntDefault(c.Query("limit"), 100)

	records := fetchEnterpriseRecords(sourceApp, sourceEntity, projectID, orgID, region, limit)
	c.JSON(200, gin.H{"count": len(records), "records": records, "timestamp": nowUTC()})
}

func handleGetEnterpriseRecord(c *gin.Context) {
	sourceApp := c.Param("source")
	sourceEntity := c.Param("entity")
	sourceID := c.Param("id")

	rec, ok := fetchEnterpriseRecord(sourceApp, sourceEntity, sourceID)
	if !ok {
		c.JSON(404, gin.H{"error": "enterprise record not found"})
		return
	}
	c.JSON(200, rec)
}

func handleListDatasets(c *gin.Context) {
	ds := listDatasets()
	c.JSON(200, gin.H{"count": len(ds), "datasets": ds})
}

func handleDatasetDetail(c *gin.Context) {
	id := c.Param("id")
	ds := getDataset(id)
	if ds == nil {
		c.JSON(404, gin.H{"error": "dataset not found"})
		return
	}
	// Include freshness
	freshness := DataFreshness{
		DatasetID:        ds.ID,
		DatasetName:      ds.Name,
		SourceApp:        ds.SourceApp,
		LastUpdated:      ds.LastUpdated,
		RecordCount:      ds.RecordCount,
		ProcessingStatus: "active",
	}
	if t, err := time.Parse(time.RFC3339, ds.LastUpdated); err == nil {
		freshness.AgeSeconds = int64(time.Since(t).Seconds())
		freshness.IsFresh = freshness.AgeSeconds < 24*3600
	}
	c.JSON(200, gin.H{"dataset": ds, "freshness": freshness})
}

// ─── Data Layer Bootstrap ──────────────────────────────────────────
// Registers known datasets at startup and performs initial snapshot.
func bootstrapDataLayer() {
	log.Println("analytics: bootstrapping enterprise data layer")

	// Register known datasets without hardcoding KPI values.
	// These are metadata definitions, not business numbers.
	sources := []struct {
		app, entity, name string
	}{
		{"pms", "project", "PMS Projects"},
		{"pms", "survey", "PMS Surveys"},
		{"pms", "task", "PMS Tasks"},
		{"pms", "meeting", "PMS Meetings"},
		{"pms", "approval", "PMS Approvals"},
		{"rms", "research", "RMS Research"},
		{"rms", "dataset", "RMS Datasets"},
		{"rms", "proposal", "RMS Proposals"},
		{"registry", "facility", "Registry Facilities"},
		{"registry", "staff", "Registry Staff"},
		{"registry", "organization", "Registry Organizations"},
		{"registry", "region", "Registry Regions"},
		{"statcollect", "submission", "StatCollect Submissions"},
		{"statcollect", "form", "StatCollect Forms"},
		{"helpdesk", "ticket", "HelpDesk Tickets"},
		{"statchat", "message", "StatChat Messages"},
		{"statchat", "space", "StatChat Spaces"},
	}
	for _, s := range sources {
		ds := &Dataset{
			ID:           fmt.Sprintf("%s:%s", s.app, s.entity),
			Name:         s.name,
			Description:  fmt.Sprintf("Enterprise records from %s (%s)", s.app, s.entity),
			SourceApp:    s.app,
			SourceEntity: s.entity,
			Fields:       []string{"id", "source_id", "project_id", "status", "timestamp", "region"},
			RecordCount:  0,
			Status:       "active",
			TenantID:     getEnvValue("STATGATE_TENANT_ID"),
			CreatedAt:    nowUTC(),
			UpdatedAt:    nowUTC(),
		}
		if ds.TenantID == "" {
			ds.TenantID = "statgate"
		}
		registerDataset(ds)
	}

	// Initial snapshot from source systems (non-hardcoded counts)
	go asyncInitialSnapshot()
}

func asyncInitialSnapshot() {
	time.Sleep(3 * time.Second)
	snapshotSourceCounts()
}

// snapshotSourceCounts pulls live record counts from source APIs.
func snapshotSourceCounts() {
	// PMS projects
	if data := fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/dashboard"); data != nil {
		if v, ok := data["totalProjects"].(float64); ok {
			if ds := getDataset("pms:project"); ds != nil {
				ds.RecordCount = int64(v)
				ds.LastUpdated = nowUTC()
				ds.UpdatedAt = nowUTC()
			}
		}
		if v, ok := data["totalSurveys"].(float64); ok {
			if ds := getDataset("pms:survey"); ds != nil {
				ds.RecordCount = int64(v)
				ds.LastUpdated = nowUTC()
				ds.UpdatedAt = nowUTC()
			}
		}
	}

	// RMS research
	if data := fetchJSON(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/dashboard"); data != nil {
		if v, ok := data["totalResearch"].(float64); ok {
			if ds := getDataset("rms:research"); ds != nil {
				ds.RecordCount = int64(v)
				ds.LastUpdated = nowUTC()
				ds.UpdatedAt = nowUTC()
			}
		}
	}

	// Registry facilities
	if data := fetchJSON(getEnv("REGISTRY_API_URL", "http://localhost:9090") + "/api/dashboard/stats"); data != nil {
		if v, ok := data["totalFacilities"].(float64); ok {
			if ds := getDataset("registry:facility"); ds != nil {
				ds.RecordCount = int64(v)
				ds.LastUpdated = nowUTC()
				ds.UpdatedAt = nowUTC()
			}
		}
	}

	// HelpDesk tickets
	if data := fetchJSON(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/dashboard"); data != nil {
		if v, ok := data["openTickets"].(float64); ok {
			if ds := getDataset("helpdesk:ticket"); ds != nil {
				ds.RecordCount = int64(v)
				ds.LastUpdated = nowUTC()
				ds.UpdatedAt = nowUTC()
			}
		}
	}

	// StatChat messages
	if data := fetchJSONArray(getEnv("STATCHAT_API_URL", "http://localhost:4000") + "/v1/messages?limit=1"); data != nil {
		if ds := getDataset("statchat:message"); ds != nil {
			ds.LastUpdated = nowUTC()
			ds.UpdatedAt = nowUTC()
		}
	}

	log.Println("analytics: initial source snapshot completed")
}

// handleListAnalyticsEvents returns recent analytics updates.
func handleListAnalyticsEvents(c *gin.Context) {
	if redisClient == nil {
		c.JSON(200, gin.H{"count": 0, "events": []interface{}{}})
		return
	}
	limit := parseIntDefault(c.Query("limit"), 50)
	raw, _ := redisClient.LRange(context.Background(), "statgate:analytics:history", 0, int64(limit-1)).Result()
	events := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		var ev map[string]interface{}
		if err := json.Unmarshal([]byte(item), &ev); err == nil {
			events = append(events, ev)
		}
	}
	c.JSON(200, gin.H{"count": len(events), "events": events})
}

// handleDataFreshness returns freshness for all registered datasets.
func handleDataFreshness(c *gin.Context) {
	dsList := listDatasets()
	freshnessList := make([]DataFreshness, 0, len(dsList))
	now := time.Now()
	for _, ds := range dsList {
		f := DataFreshness{
			DatasetID:        ds.ID,
			DatasetName:      ds.Name,
			SourceApp:        ds.SourceApp,
			LastUpdated:      ds.LastUpdated,
			RecordCount:      ds.RecordCount,
			ProcessingStatus: ds.Status,
		}
		if t, err := time.Parse(time.RFC3339, ds.LastUpdated); err == nil {
			f.AgeSeconds = int64(now.Sub(t).Seconds())
			f.IsFresh = f.AgeSeconds < 3600 // < 1 hour = fresh for live data
		} else {
			f.IsFresh = false
		}
		freshnessList = append(freshnessList, f)
	}
	c.JSON(200, gin.H{"count": len(freshnessList), "freshness": freshnessList})
}
