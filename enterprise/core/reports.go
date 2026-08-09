package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var reportStore = struct {
	sync.RWMutex
	reports map[string]ReportRequest
}{reports: map[string]ReportRequest{}}

func handleListReports(c *gin.Context) {
	status := c.Query("status")
	source := c.Query("source")
	reportStore.RLock()
	reports := make([]ReportRequest, 0, len(reportStore.reports))
	for _, r := range reportStore.reports {
		if status != "" && r.Status != status {
			continue
		}
		if source != "" && r.SourceApp != source {
			continue
		}
		reports = append(reports, r)
	}
	reportStore.RUnlock()
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].CreatedAt > reports[j].CreatedAt
	})
	c.JSON(200, gin.H{"count": len(reports), "reports": reports})
}

func handleGetReport(c *gin.Context) {
	id := c.Param("id")
	reportStore.RLock()
	r, ok := reportStore.reports[id]
	reportStore.RUnlock()
	if !ok {
		c.JSON(404, gin.H{"error": "report not found"})
		return
	}
	c.JSON(200, r)
}

func handleCreateReport(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid report request"})
		return
	}
	if req.ID == "" {
		req.ID = fmt.Sprintf("rpt_%d", time.Now().UnixNano())
	}
	if req.Status == "" {
		req.Status = "queued"
	}
	if req.CreatedAt == "" {
		req.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if req.CreatedBy == "" {
		req.CreatedBy = c.GetHeader("X-User-ID")
	}
	if req.Format == "" {
		req.Format = "json"
	}
	reportStore.Lock()
	reportStore.reports[req.ID] = req
	reportStore.Unlock()

	go generateReport(req.ID)

	publishEvent(DomainEvent{
		EventType:  "report.requested",
		Source:     req.SourceApp,
		ObjectType: "report",
		ObjectID:   req.ID,
		Actor:      req.CreatedBy,
		Payload:    map[string]interface{}{"title": req.Title, "type": req.Type, "format": req.Format},
	})

	c.JSON(202, req)
}

func generateReport(id string) {
	reportStore.RLock()
	req, ok := reportStore.reports[id]
	reportStore.RUnlock()
	if !ok {
		return
	}

	time.Sleep(2 * time.Second)

	reportStore.Lock()
	req.Status = "completed"
	req.DownloadURL = fmt.Sprintf("/api/reports/%s/download", id)
	req.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	reportStore.reports[id] = req
	reportStore.Unlock()

	publishEvent(DomainEvent{
		EventType:  "report.generated",
		Source:     req.SourceApp,
		ObjectType: "report",
		ObjectID:   id,
		Actor:      req.CreatedBy,
		Payload:    map[string]interface{}{"title": req.Title, "format": req.Format},
	})

	_, _ = createNotificationRecord(Notification{
		UserID:         req.CreatedBy,
		Title:          "Report Ready",
		Body:           fmt.Sprintf("Report %s is ready to download.", req.Title),
		Priority:       "medium",
		Category:       "report",
		SourceApp:      req.SourceApp,
		SourceEntity:   "report",
		SourceEntityID: id,
		DeepLink:       fmt.Sprintf("/api/reports/%s/download", id),
	})
}

func handleDownloadReport(c *gin.Context) {
	id := c.Param("id")
	reportStore.RLock()
	r, ok := reportStore.reports[id]
	reportStore.RUnlock()
	if !ok {
		c.JSON(404, gin.H{"error": "report not found"})
		return
	}
	if r.Status != "completed" {
		c.JSON(400, gin.H{"error": "report not ready", "status": r.Status})
		return
	}
	filename := sanitizeFilename(r.Title) + "." + r.Format
	var data []byte
	switch r.Format {
	case "csv":
		data = []byte(fmt.Sprintf("id,title,type,format,status,created_at\n%s,%s,%s,%s,%s,%s\n",
			r.ID, r.Title, r.Type, r.Format, r.Status, r.CreatedAt))
		filename = sanitizeFilename(r.Title) + ".csv"
	case "pdf":
		data = []byte(fmt.Sprintf("%%PDF-1.4\nReport: %s\nType: %s\nCreated: %s\n", r.Title, r.Type, r.CreatedAt))
	case "excel":
		filename = sanitizeFilename(r.Title) + ".xlsx"
		data = []byte(fmt.Sprintf("Report: %s\nType: %s\n", r.Title, r.Type))
	default:
		data, _ = json.MarshalIndent(r, "", "  ")
		filename = sanitizeFilename(r.Title) + ".json"
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(200, "application/octet-stream", data)
}

func handleDeleteReport(c *gin.Context) {
	id := c.Param("id")
	reportStore.Lock()
	delete(reportStore.reports, id)
	reportStore.Unlock()
	c.JSON(200, gin.H{"status": "ok", "id": id, "deleted": true})
}
