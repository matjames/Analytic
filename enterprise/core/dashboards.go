package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func handleListDashboards(c *gin.Context) {
	ownerID := c.Query("owner_id")
	ownerType := c.Query("owner_type")
	limit := parseIntDefault(c.Query("limit"), 50)
	dashboards := fetchDashboards(ownerID, ownerType, limit)
	c.JSON(200, gin.H{"count": len(dashboards), "dashboards": dashboards})
}

func fetchDashboards(ownerID, ownerType string, limit int) []DashboardDefinition {
	if redisClient == nil {
		return []DashboardDefinition{}
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:dashboards", 0, -1).Result()
	dashboards := make([]DashboardDefinition, 0, len(raw))
	for _, item := range raw {
		var d DashboardDefinition
		if err := json.Unmarshal([]byte(item), &d); err == nil {
			if ownerID != "" && d.OwnerID != ownerID && d.OwnerType != "organization" && d.OwnerType != "system" {
				continue
			}
			if ownerType != "" && d.OwnerType != ownerType {
				continue
			}
			dashboards = append(dashboards, d)
			if len(dashboards) >= limit {
				break
			}
		}
	}
	return dashboards
}

func handleGetDashboard(c *gin.Context) {
	id := c.Param("id")
	dashboards := fetchDashboards("", "", 500)
	for _, d := range dashboards {
		if d.ID == id {
			c.JSON(200, d)
			return
		}
	}
	c.JSON(404, gin.H{"error": "dashboard not found"})
}

func handleCreateDashboard(c *gin.Context) {
	var d DashboardDefinition
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(400, gin.H{"error": "invalid dashboard", "detail": err.Error()})
		return
	}
	if d.ID == "" {
		d.ID = fmt.Sprintf("dash_%d", time.Now().UnixNano())
	}
	if d.CreatedAt == "" {
		d.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	d.UpdatedAt = d.CreatedAt
	if d.OwnerID == "" {
		d.OwnerID = c.GetHeader("X-User-ID")
	}
	if d.OwnerType == "" {
		d.OwnerType = "user"
	}
	if redisClient != nil {
		data, _ := json.Marshal(d)
		redisClient.RPush(context.Background(), "statgate:dashboards", string(data))
	}
	recordAudit("dashboard.create", "enterprise", d.OwnerID, map[string]interface{}{"dashboard_id": d.ID, "name": d.Name})
	c.JSON(201, d)
}

func handleUpdateDashboard(c *gin.Context) {
	id := c.Param("id")
	dashboards := fetchDashboards("", "", 500)
	for _, d := range dashboards {
		if d.ID == id {
			var updates DashboardDefinition
			if err := c.ShouldBindJSON(&updates); err != nil {
				c.JSON(400, gin.H{"error": "invalid dashboard"})
				return
			}
			if updates.Name != "" {
				d.Name = updates.Name
			}
			if updates.Description != "" {
				d.Description = updates.Description
			}
			if updates.Layout != nil {
				d.Layout = updates.Layout
			}
			d.Shared = updates.Shared
			d.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			persistDashboard(id, d)
			recordAudit("dashboard.update", "enterprise", d.OwnerID, map[string]interface{}{"dashboard_id": id})
			c.JSON(200, d)
			return
		}
	}
	c.JSON(404, gin.H{"error": "dashboard not found"})
}

func handleDeleteDashboard(c *gin.Context) {
	id := c.Param("id")
	if redisClient != nil {
		ctx := context.Background()
		raw, _ := redisClient.LRange(ctx, "statgate:dashboards", 0, -1).Result()
		redisClient.Del(ctx, "statgate:dashboards")
		for _, item := range raw {
			var d DashboardDefinition
			if err := json.Unmarshal([]byte(item), &d); err == nil {
				if d.ID != id {
					redisClient.RPush(ctx, "statgate:dashboards", item)
				}
			}
		}
	}
	c.JSON(200, gin.H{"status": "ok", "id": id, "deleted": true})
}

func handleShareDashboard(c *gin.Context) {
	id := c.Param("id")
	dashboards := fetchDashboards("", "", 500)
	for _, d := range dashboards {
		if d.ID == id {
			d.Shared = true
			persistDashboard(id, d)
			c.JSON(200, gin.H{"status": "shared", "id": id})
			return
		}
	}
	c.JSON(404, gin.H{"error": "dashboard not found"})
}

func persistDashboard(id string, updated DashboardDefinition) {
	if redisClient == nil {
		return
	}
	ctx := context.Background()
	raw, _ := redisClient.LRange(ctx, "statgate:dashboards", 0, -1).Result()
	redisClient.Del(ctx, "statgate:dashboards")
	for _, item := range raw {
		var d DashboardDefinition
		if err := json.Unmarshal([]byte(item), &d); err == nil {
			if d.ID == id {
				d = updated
			}
			data, _ := json.Marshal(d)
			redisClient.RPush(ctx, "statgate:dashboards", string(data))
		}
	}
}

// handleWidgetData fetches live data for a given widget type.
func handleWidgetData(c *gin.Context) {
	widgetID := c.Param("widgetId")
	widgetType := c.Query("type")
	if widgetType == "" {
		widgetType = widgetID
	}
	data := fetchWidgetData(widgetType)
	c.JSON(200, gin.H{"widget_id": widgetID, "type": widgetType, "data": data, "timestamp": time.Now().UTC().Format(time.RFC3339)})
}

func fetchWidgetData(widgetType string) map[string]interface{} {
	switch widgetType {
	case "kpi":
		return map[string]interface{}{"kpis": fetchAggregatedKPIs()}
	case "trend":
		return map[string]interface{}{"trend": fetchAggregatedTrend()}
	case "table":
		return map[string]interface{}{"table": fetchAggregatedTable()}
	case "timeline":
		return map[string]interface{}{"timeline": fetchTimeline(20)}
	case "task_list":
		return map[string]interface{}{"tasks": fetchAggregatedTasks()}
	case "activity_feed":
		return map[string]interface{}{"feed": fetchTimeline(20)}
	case "calendar":
		return map[string]interface{}{"calendar": fetchCalendarEvents(30)}
	case "gauge":
		return map[string]interface{}{"gauge": fetchAggregatedGauge()}
	case "metric_tile":
		return map[string]interface{}{"metrics": fetchAggregatedMetrics()}
	case "heat_map":
		return map[string]interface{}{"heatmap": fetchAggregatedHeatmap()}
	default:
		return map[string]interface{}{"kpis": fetchAggregatedKPIs(), "trend": fetchAggregatedTrend()}
	}
}

func fetchAggregatedKPIs() []map[string]interface{} {
	kpis := []map[string]interface{}{}
	if data := fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/dashboard"); data != nil {
		kpis = append(kpis, map[string]interface{}{"label": "PMS Projects", "value": data["totalProjects"], "source": "pms", "icon": "🏗️"})
		kpis = append(kpis, map[string]interface{}{"label": "PMS Surveys", "value": data["totalSurveys"], "source": "pms", "icon": "📋"})
	}
	if data := fetchJSON(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/dashboard"); data != nil {
		kpis = append(kpis, map[string]interface{}{"label": "RMS Studies", "value": data["totalResearch"], "source": "rms", "icon": "🔬"})
	}
	return kpis
}

func fetchAggregatedTrend() []map[string]interface{} {
	// Real data from PMS project creation timestamps
	trend := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); data != nil {
		monthly := map[string]int{}
		for _, p := range data {
			if ts, ok := p["created_time"].(string); ok {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					key := t.Format("Jan")
					monthly[key]++
				}
			}
		}
		// Sort months chronologically
		months := []string{}
		for i := 5; i >= 0; i-- {
			months = append(months, time.Now().AddDate(0, -i, 0).Format("Jan"))
		}
		for _, m := range months {
			trend = append(trend, map[string]interface{}{
				"period": m, "value": monthly[m], "source": "pms",
			})
		}
	}
	return trend
}

func fetchAggregatedTable() []map[string]interface{} {
	// Real data from PMS projects
	table := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); data != nil {
		for _, p := range data {
			table = append(table, map[string]interface{}{
				"entity": "Project",
				"name":   p["name"],
				"status": p["stage"],
				"owner":  p["owner"],
				"id":     p["id"],
			})
		}
	}
	// Add RMS research
	if data := fetchJSONArray(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/research"); data != nil {
		for _, r := range data {
			table = append(table, map[string]interface{}{
				"entity": "Research",
				"name":   r["name"],
				"status": r["stage"],
				"owner":  r["owner"],
				"id":     r["id"],
			})
		}
	}
	return table
}

func fetchAggregatedTasks() []map[string]interface{} {
	tasks := []map[string]interface{}{}
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/tasks?limit=10"); data != nil {
		for _, t := range data {
			if title, ok := t["title"]; ok {
				tasks = append(tasks, map[string]interface{}{
					"id": t["id"], "title": title, "status": t["status"], "source": "pms", "due": t["end_date"],
				})
			}
		}
	}
	return tasks
}

func fetchAggregatedMetrics() []map[string]interface{} {
	metrics := []map[string]interface{}{}

	// Real data from PMS
	if data := fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/dashboard"); data != nil {
		if v, ok := data["totalProjects"]; ok {
			metrics = append(metrics, map[string]interface{}{"label": "Total Projects", "value": v, "source": "pms"})
		}
		if v, ok := data["totalSurveys"]; ok {
			metrics = append(metrics, map[string]interface{}{"label": "Total Surveys", "value": v, "source": "pms"})
		}
	}

	// Real data from RMS
	if data := fetchJSON(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/dashboard"); data != nil {
		if v, ok := data["totalResearch"]; ok {
			metrics = append(metrics, map[string]interface{}{"label": "Research Studies", "value": v, "source": "rms"})
		}
	}

	// Real data from Registry
	if data := fetchJSON(getEnv("REGISTRY_API_URL", "http://localhost:9090") + "/api/dashboard/stats"); data != nil {
		if v, ok := data["totalFacilities"]; ok {
			metrics = append(metrics, map[string]interface{}{"label": "Facilities", "value": v, "source": "registry"})
		}
	}

	// Real data from HelpDesk
	if data := fetchJSON(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/dashboard"); data != nil {
		if v, ok := data["openTickets"]; ok {
			metrics = append(metrics, map[string]interface{}{"label": "Open Tickets", "value": v, "source": "helpdesk"})
		}
	}

	return metrics
}

func fetchAggregatedGauge() map[string]interface{} {
	// Real data: project completion rate from PMS
	value := 0.0
	if data := fetchJSONArray(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/projects"); data != nil {
		total := len(data)
		completed := 0
		for _, p := range data {
			if stage, ok := p["stage"].(string); ok {
				if stage == "Completed" || stage == "Closed" {
					completed++
				}
			}
		}
		if total > 0 {
			value = float64(completed) / float64(total) * 100
		}
	}
	return map[string]interface{}{
		"value": int(value), "min": 0, "max": 100, "label": "Project Completion Rate",
	}
}

func fetchAggregatedHeatmap() []map[string]interface{} {
	// Real data from timeline activity by day/hour
	heatmap := []map[string]interface{}{}
	if entries := fetchTimeline(500); len(entries) > 0 {
		dayHour := map[string]int{}
		for _, e := range entries {
			if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
				key := fmt.Sprintf("%s:%d", t.Weekday().String()[:3], t.Hour())
				dayHour[key]++
			}
		}
		days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		for _, d := range days {
			for h := 8; h <= 18; h++ {
				key := fmt.Sprintf("%s:%d", d, h)
				heatmap = append(heatmap, map[string]interface{}{
					"day": d, "hour": h, "value": dayHour[key],
				})
			}
		}
	}
	return heatmap
}

// ─── Widget Library ───────────────────────────────────────────────

type WidgetDefinition struct {
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Icon         string   `json:"icon"`
	DefaultW     int      `json:"default_w"`
	DefaultH     int      `json:"default_h"`
	MaxW         int      `json:"max_w"`
	MaxH         int      `json:"max_h"`
	Configurable []string `json:"configurable"`
}

func handleListWidgets(c *gin.Context) {
	widgets := []WidgetDefinition{
		{Type: "kpi_card", Name: "KPI Card", Description: "Display a single key performance indicator", Icon: "📊", DefaultW: 2, DefaultH: 2, MaxW: 4, MaxH: 4, Configurable: []string{"label", "source", "aggregation"}},
		{Type: "trend_chart", Name: "Trend Chart", Description: "Line chart showing trends over time", Icon: "📈", DefaultW: 4, DefaultH: 3, MaxW: 6, MaxH: 6, Configurable: []string{"metric", "period", "source"}},
		{Type: "table", Name: "Table", Description: "Data table with filtering and sorting", Icon: "📋", DefaultW: 4, DefaultH: 4, MaxW: 6, MaxH: 8, Configurable: []string{"columns", "source", "filter"}},
		{Type: "map", Name: "Map", Description: "Geographic distribution view", Icon: "🗺️", DefaultW: 4, DefaultH: 4, MaxW: 6, MaxH: 6, Configurable: []string{"latitude_field", "longitude_field", "marker_field"}},
		{Type: "timeline", Name: "Timeline", Description: "Chronological activity stream", Icon: "⏱️", DefaultW: 4, DefaultH: 4, MaxW: 6, MaxH: 8, Configurable: []string{"applications", "entity_types"}},
		{Type: "task_list", Name: "Task List", Description: "Tasks and assignments", Icon: "✅", DefaultW: 3, DefaultH: 3, MaxW: 6, MaxH: 6, Configurable: []string{"assignee", "status"}},
		{Type: "activity_feed", Name: "Activity Feed", Description: "Recent user activity", Icon: "🔔", DefaultW: 3, DefaultH: 3, MaxW: 6, MaxH: 6, Configurable: []string{"users", "applications"}},
		{Type: "calendar", Name: "Calendar", Description: "Upcoming events and deadlines", Icon: "📅", DefaultW: 4, DefaultH: 4, MaxW: 6, MaxH: 8, Configurable: []string{"categories", "projects"}},
		{Type: "heat_map", Name: "Heat Map", Description: "Activity density heatmap", Icon: "🔥", DefaultW: 4, DefaultH: 3, MaxW: 6, MaxH: 6, Configurable: []string{"grouping", "metric"}},
		{Type: "gauge", Name: "Gauge", Description: "Progress gauge for targets", Icon: "🎯", DefaultW: 2, DefaultH: 2, MaxW: 4, MaxH: 4, Configurable: []string{"min", "max", "target"}},
		{Type: "metric_tile", Name: "Metric Tile", Description: "Compact metric with delta", Icon: "🔢", DefaultW: 2, DefaultH: 1, MaxW: 4, MaxH: 2, Configurable: []string{"metric", "format"}},
	}
	c.JSON(200, gin.H{"widgets": widgets, "count": len(widgets)})
}

func handleWidgetPreview(c *gin.Context) {
	widgetType := c.Param("type")
	data := fetchWidgetData(widgetType)
	c.JSON(200, gin.H{"type": widgetType, "preview": data})
}
