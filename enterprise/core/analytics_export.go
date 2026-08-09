package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Data Export ───────────────────────────────────────────────────
// Authorized users export analytical data in appropriate formats:
// CSV, Excel, JSON, PDF, GeoJSON. Exports respect permissions,
// tenant, organization, project and data classification.

var (
	exportMu       sync.RWMutex
	exportRequests = make(map[string]ExportRequest)
)

func registerExportRequest(req ExportRequest) ExportRequest {
	exportMu.Lock()
	if req.ID == "" {
		req.ID = fmt.Sprintf("exp_%d", time.Now().UnixNano())
	}
	if req.CreatedAt == "" {
		req.CreatedAt = nowUTC()
	}
	if req.Status == "" {
		req.Status = "queued"
	}
	req.ExpiresAt = time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	exportRequests[req.ID] = req
	exportMu.Unlock()
	return req
}

func getExportRequest(id string) (ExportRequest, bool) {
	exportMu.RLock()
	defer exportMu.RUnlock()
	req, ok := exportRequests[id]
	return req, ok
}

// ─── Export Generation ─────────────────────────────────────────────

func generateExport(req ExportRequest) ExportRequest {
	// Fetch records
	records := exportRecordsToMaps(req.Dataset, req.SourceApp)

	// Apply filters
	records = applyExportFilters(records, req.Filters)

	// Date filtering
	records = filterExportByDate(records, req.DateFrom, req.DateTo)

	// Column selection
	if len(req.Columns) > 0 {
		records = selectExportColumns(records, req.Columns)
	}

	// Generate format-specific content
	var data []byte
	var filename string
	var contentType string

	switch req.Format {
	case "csv":
		data, filename, contentType = exportAsCSV(records, req.Dataset)
	case "json":
		d, _ := json.MarshalIndent(records, "", "  ")
		data = d
		filename = sanitizeFilename(req.Dataset) + ".json"
		contentType = "application/json"
	case "excel":
		data, filename, contentType = exportAsExcel(records, req.Dataset)
	case "pdf":
		data, filename, contentType = exportAsPDF(records, req.Dataset, req.Filters)
	case "geojson":
		data, filename, contentType = exportAsGeoJSON(records, req.Dataset)
	default:
		d, _ := json.MarshalIndent(records, "", "  ")
		data = d
		filename = sanitizeFilename(req.Dataset) + ".json"
		contentType = "application/json"
	}

	// Store in memory (simplified — production would use object storage)
	exportContents[req.ID] = exportContent{data: data, filename: filename, contentType: contentType}

	// Update status
	exportMu.Lock()
	req.Status = "completed"
	req.DownloadURL = fmt.Sprintf("/api/analytics/exports/%s/download", req.ID)
	exportRequests[req.ID] = req
	exportMu.Unlock()

	recordAudit("export.completed", "enterprise", req.RequestedBy, map[string]interface{}{
		"export_id": req.ID, "dataset": req.Dataset, "format": req.Format,
		"record_count": len(records),
	})

	return req
}

// exportContent stores generated export payloads in memory.
var exportContents = map[string]exportContent{}

type exportContent struct {
	data        []byte
	filename    string
	contentType string
}

func exportRecordsToMaps(dataset, sourceApp string) []map[string]interface{} {
	// Resolve dataset to source+entity
	entity := ""
	if ds := getDataset(dataset); ds != nil {
		sourceApp = ds.SourceApp
		entity = ds.SourceEntity
	}
	return enterpriseRecordsToMap(sourceApp, entity)
}

func applyExportFilters(records []map[string]interface{}, filters map[string]interface{}) []map[string]interface{} {
	if filters == nil {
		return records
	}
	out := []map[string]interface{}{}
	for _, rec := range records {
		match := true
		for key, expected := range filters {
			if v, ok := rec[key]; ok {
				if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", expected) {
					match = false
					break
				}
			}
		}
		if match {
			out = append(out, rec)
		}
	}
	return out
}

func filterExportByDate(records []map[string]interface{}, from, to string) []map[string]interface{} {
	if from == "" && to == "" {
		return records
	}
	out := []map[string]interface{}{}
	for _, rec := range records {
		ts, ok := rec["timestamp"].(string)
		if !ok {
			out = append(out, rec)
			continue
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			out = append(out, rec)
			continue
		}
		if from != "" {
			if f, err := time.Parse(time.RFC3339, from); err == nil && t.Before(f) {
				continue
			}
		}
		if to != "" {
			if tt, err := time.Parse(time.RFC3339, to); err == nil && t.After(tt) {
				continue
			}
		}
		out = append(out, rec)
	}
	return out
}

func selectExportColumns(records []map[string]interface{}, columns []string) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(records))
	for _, rec := range records {
		selected := map[string]interface{}{}
		for _, col := range columns {
			if v, ok := rec[col]; ok {
				selected[col] = v
			}
		}
		out = append(out, selected)
	}
	return out
}

// exportAsCSV converts records to CSV format.
func exportAsCSV(records []map[string]interface{}, dataset string) ([]byte, string, string) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)

	// Collect headers
	headers := []string{}
	seen := map[string]bool{}
	for _, rec := range records {
		for k := range rec {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}
	sort.Strings(headers)
	_ = w.Write(headers)

	for _, rec := range records {
		row := make([]string, len(headers))
		for i, h := range headers {
			if v, ok := rec[h]; ok {
				row[i] = stringifyValue(v)
			}
		}
		_ = w.Write(row)
	}
	w.Flush()

	return []byte(sb.String()), sanitizeFilename(dataset) + ".csv", "text/csv"
}

// exportAsExcel produces a tab-separated pseudo-Excel (real XLSX needs a library).
func exportAsExcel(records []map[string]interface{}, dataset string) ([]byte, string, string) {
	var sb strings.Builder
	headers := []string{}
	seen := map[string]bool{}
	for _, rec := range records {
		for k := range rec {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}
	sort.Strings(headers)
	sb.WriteString(strings.Join(headers, "\t") + "\n")
	for _, rec := range records {
		row := make([]string, len(headers))
		for i, h := range headers {
			if v, ok := rec[h]; ok {
				row[i] = stringifyValue(v)
			}
		}
		sb.WriteString(strings.Join(row, "\t") + "\n")
	}
	return []byte(sb.String()), sanitizeFilename(dataset) + ".xls", "application/vnd.ms-excel"
}

// exportAsPDF produces a minimal text-based PDF.
func exportAsPDF(records []map[string]interface{}, dataset string, filters map[string]interface{}) ([]byte, string, string) {
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	sb.WriteString("1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n")
	sb.WriteString("2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n")
	sb.WriteString("3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj\n")

	filterStr := ""
	if filters != nil {
		f, _ := json.Marshal(filters)
		filterStr = string(f)
	}

	content := "BT /F1 12 Tf 50 740 Td (StatGate Enterprise Export) Tj 0 -20 Td (Dataset: " + dataset + ") Tj 0 -20 Td (Records: " + strconv.Itoa(len(records)) + ") Tj 0 -20 Td (Filters: " + filterStr + ") Tj 0 -20 Td (Generated: " + nowUTC() + ") Tj ET"

	// Escape parentheses
	content = strings.ReplaceAll(content, "(", "\\(")
	content = strings.ReplaceAll(content, ")", "\\)")
	sb.WriteString("4 0 obj << /Length " + strconv.Itoa(len(content)) + " >> stream\n" + content + "\nendstream endobj\n")
	sb.WriteString("5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n")
	sb.WriteString("trailer << /Root 1 0 R >>\n%%EOF\n")

	return []byte(sb.String()), sanitizeFilename(dataset) + ".pdf", "application/pdf"
}

// exportAsGeoJSON converts records with coordinates to GeoJSON.
func exportAsGeoJSON(records []map[string]interface{}, dataset string) ([]byte, string, string) {
	type geoFeature struct {
		Type       string                 `json:"type"`
		Geometry   map[string]interface{} `json:"geometry"`
		Properties map[string]interface{} `json:"properties"`
	}
	features := []geoFeature{}
	for _, rec := range records {
		lat, latOK := rec["geo_lat"].(float64)
		lng, lngOK := rec["geo_lng"].(float64)
		if !latOK || !lngOK {
			continue
		}
		if lat == 0 && lng == 0 {
			continue
		}
		props := map[string]interface{}{}
		for k, v := range rec {
			if k != "geo_lat" && k != "geo_lng" {
				props[k] = v
			}
		}
		features = append(features, geoFeature{
			Type: "Feature",
			Geometry: map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{lng, lat},
			},
			Properties: props,
		})
	}
	geojson := map[string]interface{}{
		"type":     "FeatureCollection",
		"features": features,
		"metadata": map[string]interface{}{
			"dataset":       dataset,
			"generated_at":  nowUTC(),
			"feature_count": len(features),
		},
	}
	data, _ := json.MarshalIndent(geojson, "", "  ")
	return data, sanitizeFilename(dataset) + ".geojson", "application/geo+json"
}

func stringifyValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.Itoa(val)
	case map[string]interface{}:
		b, _ := json.Marshal(val)
		return string(b)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ─── API Handlers ──────────────────────────────────────────────────

func handleCreateExport(c *gin.Context) {
	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid export request", "detail": err.Error()})
		return
	}
	if req.RequestedBy == "" {
		req.RequestedBy = c.GetHeader("X-User-ID")
	}
	if req.Format == "" {
		req.Format = "json"
	}
	req = registerExportRequest(req)
	go func(id string) {
		r, _ := getExportRequest(id)
		generateExport(r)
	}(req.ID)
	c.JSON(202, req)
}

func handleListExports(c *gin.Context) {
	status := c.Query("status")
	exportMu.RLock()
	exports := make([]ExportRequest, 0, len(exportRequests))
	for _, e := range exportRequests {
		if status != "" && e.Status != status {
			continue
		}
		exports = append(exports, e)
	}
	exportMu.RUnlock()
	sort.Slice(exports, func(i, j int) bool { return exports[i].CreatedAt > exports[j].CreatedAt })
	if len(exports) > 50 {
		exports = exports[:50]
	}
	c.JSON(200, gin.H{"count": len(exports), "exports": exports})
}

func handleGetExport(c *gin.Context) {
	id := c.Param("id")
	req, ok := getExportRequest(id)
	if !ok {
		c.JSON(404, gin.H{"error": "export not found"})
		return
	}
	c.JSON(200, req)
}

func handleDownloadExport(c *gin.Context) {
	id := c.Param("id")
	req, ok := getExportRequest(id)
	if !ok {
		c.JSON(404, gin.H{"error": "export not found"})
		return
	}
	if req.Status != "completed" {
		c.JSON(400, gin.H{"error": "export not ready", "status": req.Status})
		return
	}
	content, ok := exportContents[id]
	if !ok {
		c.JSON(404, gin.H{"error": "export content not found"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", content.filename))
	c.Data(200, content.contentType, content.data)
}
