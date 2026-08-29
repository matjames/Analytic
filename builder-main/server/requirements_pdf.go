package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// RequirementsSpecPDFHandler handles both HTML and PDF views of a single requirements spec.
// Route: GET /api/requirements-specs/{id}       — returns the entry as JSON
// Route: PUT /api/requirements-specs/{id}       — updates an existing entry
// Route: GET /api/requirements-specs/{id}/html  — renders the spec as HTML
// Route: GET /api/requirements-specs/{id}/pdf   — renders the spec as PDF
func RequirementsSpecPDFHandler(w http.ResponseWriter, r *http.Request) {
	prefix := basePath + "/api/requirements-specs/"
	trimmed := strings.TrimPrefix(r.URL.Path, prefix)

	// Plain /{id} with no sub-path — GET (JSON) or PUT (update)
	if !strings.Contains(trimmed, "/") {
		switch r.Method {
		case http.MethodGet:
			handleRequirementsGetOne(w, r, trimmed)
		case http.MethodPut:
			handleRequirementsUpdateOne(w, r, trimmed)
		default:
			w.Header().Set("Allow", "GET, PUT")
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// /validate sub-path
	if strings.HasSuffix(trimmed, "/validate") {
		idStr := strings.TrimSuffix(trimmed, "/validate")
		handleValidateRequirementsSpec(w, r, idStr)
		return
	}

	// Extract numeric ID and format from: /api/requirements-specs/{id}/pdf|html
	var idStr, format string
	if strings.HasSuffix(trimmed, "/html") {
		idStr = strings.TrimSuffix(trimmed, "/html")
		format = "html"
	} else {
		idStr = strings.TrimSuffix(trimmed, "/pdf")
		format = "pdf"
	}
	if idStr == trimmed || idStr == "" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid requirements spec ID")
		return
	}

	ctx := r.Context()
	entry, err := GetRequirementsSpecByID(ctx, id)
	if err != nil {
		logErrorCtx(ctx, "Requirements PDF: fetch id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch requirements spec")
		return
	}
	if entry == nil {
		writeJSONError(w, http.StatusNotFound, "requirements spec not found")
		return
	}

	fullHTML, err := buildRequirementsPDFHTML(entry)
	if err != nil {
		logErrorCtx(ctx, "Requirements PDF: build HTML id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to build PDF content")
		return
	}

	// Serve HTML directly — no Chrome required.
	if format == "html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fullHTML))
		logInfoCtx(ctx, "Requirements HTML served: id=%d", id)
		return
	}

	// Rate limiting — reuses global pdfSemaphore and pdfPerIP from handlers.go
	ip := getClientIP(r)
	counterVal, _ := pdfPerIP.LoadOrStore(ip, &atomic.Int32{})
	ipCounter := counterVal.(*atomic.Int32)
	if ipCounter.Add(1) > 2 {
		ipCounter.Add(-1)
		writeJSONError(w, http.StatusTooManyRequests, "too many concurrent PDF requests")
		return
	}
	select {
	case pdfSemaphore <- struct{}{}:
		defer func() { <-pdfSemaphore; ipCounter.Add(-1) }()
	case <-time.After(30 * time.Second):
		ipCounter.Add(-1)
		writeJSONError(w, http.StatusServiceUnavailable, "PDF generation queue full, try again later")
		return
	}

	pdfCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(pdfCtx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
		)...,
	)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	tmp, err := os.CreateTemp("", "req-pdf-*.html")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to create temp file")
		return
	}
	defer os.Remove(tmp.Name())

	if _, err = tmp.WriteString(fullHTML); err != nil {
		tmp.Close()
		writeJSONError(w, http.StatusInternalServerError, "failed to write temp file")
		return
	}
	tmp.Close()

	var pdfBuf []byte
	err = chromedp.Run(browserCtx,
		chromedp.EmulateViewport(1200, 900),
		chromedp.Navigate("file://"+tmp.Name()),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var e error
			pdfBuf, _, e = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0.5).
				WithMarginBottom(0.6).
				WithMarginLeft(0.5).
				WithMarginRight(0.5).
				WithPreferCSSPageSize(false).
				WithScale(0.85).
				WithDisplayHeaderFooter(true).
				WithHeaderTemplate(`<div style="font-size:7px;color:#888;width:100%;text-align:right;padding-right:20px;">Digital Reporting Requirements Specification — Confidential</div>`).
				WithFooterTemplate(`<div style="font-size:7px;text-align:center;width:100%;color:#888;">Page <span class="pageNumber"></span> of <span class="totalPages"></span></div>`).
				Do(ctx)
			return e
		}),
	)
	if err != nil {
		logErrorCtx(r.Context(), "Requirements PDF: chromedp error id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to generate PDF")
		return
	}

	fname := "requirements-" + reqSanitizeFilename(entry.ReportName) + ".pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+fname+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBuf)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBuf)
	logInfoCtx(r.Context(), "Requirements PDF served: id=%d bytes=%d", id, len(pdfBuf))
}

func reqSanitizeFilename(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	result := b.String()
	if result == "" {
		return "spec"
	}
	return result
}

// buildRequirementsPDFHTML constructs a fully self-contained HTML document
// from the requirements spec payload that chromedp can render to PDF.
func buildRequirementsPDFHTML(e *RequirementsSpecEntry) (string, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(e.PayloadJSON), &payload); err != nil {
		return "", fmt.Errorf("unmarshal payload: %w", err)
	}

	esc := html.EscapeString
	docCtrl := asMap(payload["documentControl"])
	overview := asMap(payload["reportOverview"])
	dataReq := asMap(payload["dataRequirements"])
	filters := asMap(payload["reportFilters"])
	access := asMap(payload["reportAccessRequirements"])
	perf := asMap(payload["performanceRequirements"])
	_ = asMap(payload["geospatialRequirements"]) // retained in payload; not rendered (covered by §6)
	vizReq := asMap(payload["visualizationRequirements"])
	approvals := asMap(payload["approvals"])

	submittedAt := ""
	if sm := asMap(payload["serverMetadata"]); len(sm) > 0 {
		submittedAt = readString(sm, "submittedAt")
	}

	var sb strings.Builder

	// ── CSS ──────────────────────────────────────────────────────────────────
	sb.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
	font-family: 'Segoe UI', Arial, sans-serif;
	font-size: 11px;
	color: #1a2b3c;
	background: #fff;
	line-height: 1.5;
	padding: 20px 24px;
}
.doc-cover {
	border-bottom: 3px solid #183a3f;
	padding-bottom: 18px;
	margin-bottom: 22px;
}
.doc-eyebrow {
	font-size: 9px;
	font-weight: 700;
	letter-spacing: 1.5px;
	text-transform: uppercase;
	color: #1a5276;
	margin-bottom: 6px;
}
.doc-title {
	font-size: 24px;
	font-weight: 700;
	color: #183a3f;
	margin-bottom: 10px;
}
.doc-meta-row {
	display: flex;
	flex-wrap: wrap;
	gap: 6px 24px;
	font-size: 10px;
	color: #555;
}
.doc-meta-row span strong { color: #1a2b3c; }
.section {
	margin-bottom: 18px;
}
.section-heading {
	font-size: 11.5px;
	font-weight: 700;
	color: #fff;
	background: #1a5276;
	padding: 5px 10px;
	margin-bottom: 10px;
}
.field-grid {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	gap: 8px 16px;
}
.field-grid.two-col { grid-template-columns: repeat(2, 1fr); }
.field-grid.one-col { grid-template-columns: 1fr; }
.field { display: flex; flex-direction: column; }
.field-label {
	font-size: 8.5px;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.5px;
	color: #666;
	margin-bottom: 2px;
}
.field-value {
	font-size: 11px;
	color: #1a2b3c;
	background: #f4f6f8;
	padding: 4px 7px;
	border: 1px solid #dde3e6;
	border-radius: 3px;
	min-height: 24px;
}
.objective-text {
	font-size: 11px;
	font-style: italic;
	color: #2c3e50;
	background: #f4f6f8;
	border-left: 3px solid #1a5276;
	padding: 8px 12px;
	margin-top: 8px;
}
.tag-list {
	display: flex;
	flex-wrap: wrap;
	gap: 5px;
}
.tag {
	background: #e8f4f8;
	border: 1px solid #a9ccd9;
	color: #1a4f6e;
	padding: 2px 9px;
	border-radius: 12px;
	font-size: 10px;
	font-weight: 500;
}
.tag.check {
	background: #e8f5e9;
	border-color: #81c784;
	color: #2e7d32;
}
.tag.check::before { content: "✓  "; }
.approvals-table {
	width: 100%;
	border-collapse: collapse;
	font-size: 11px;
}
.approvals-table th {
	background: #eaf0f1;
	border: 1px solid #c8d6da;
	padding: 5px 10px;
	text-align: left;
	font-size: 9px;
	font-weight: 700;
	text-transform: uppercase;
	letter-spacing: 0.5px;
	color: #555;
}
.approvals-table td {
	border: 1px solid #dde3e5;
	padding: 6px 10px;
}
.vis-spec {
	border: 1px solid #b8cdd4;
	border-radius: 4px;
	margin-bottom: 12px;
	page-break-inside: avoid;
}
.vis-spec-head {
	background: #183a3f;
	color: #fff;
	padding: 7px 12px;
	display: flex;
	justify-content: space-between;
	align-items: center;
	border-radius: 3px 3px 0 0;
}
.vis-spec-title { font-size: 12px; font-weight: 700; }
.vis-spec-type {
	font-size: 9px;
	background: rgba(255,255,255,0.2);
	padding: 2px 9px;
	border-radius: 10px;
	text-transform: uppercase;
	letter-spacing: 0.5px;
}
.vis-spec-body { padding: 10px 12px; }
.vis-desc { font-size: 11px; color: #555; margin-bottom: 8px; font-style: italic; }
.indicator {
	border: 1px solid #dde8ec;
	border-radius: 3px;
	margin-bottom: 7px;
	overflow: hidden;
}
.indicator-head {
	background: #eef4f7;
	padding: 5px 10px;
	display: flex;
	justify-content: space-between;
	align-items: center;
	border-bottom: 1px solid #dde8ec;
}
.indicator-label { font-weight: 600; font-size: 11px; color: #1a2b3c; }
.indicator-mode {
	font-size: 9px;
	background: #d6e8f0;
	color: #1a4f6e;
	padding: 2px 8px;
	border-radius: 10px;
	text-transform: uppercase;
}
.de-body { padding: 6px 10px; display: flex; flex-direction: column; gap: 3px; }
.de-role {
	font-size: 8.5px;
	font-weight: 700;
	text-transform: uppercase;
	color: #888;
	letter-spacing: 0.5px;
	margin-top: 4px;
	margin-bottom: 2px;
}
.de-item {
	font-size: 10px;
	color: #333;
	padding: 2px 8px;
	background: #fafbfc;
	border: 1px solid #e8ecef;
	border-radius: 3px;
}
</style></head><body>
`)

	// ── Cover ─────────────────────────────────────────────────────────────────
	reportName := esc(readString(docCtrl, "reportName"))
	if reportName == "" {
		reportName = esc(e.ReportName)
	}
	version := readString(docCtrl, "version")
	if version == "" {
		version = "—"
	}
	status := readString(docCtrl, "status")
	if status == "" {
		status = "—"
	}
	dateReq := readString(docCtrl, "dateRequested")
	if dateReq == "" && len(submittedAt) >= 10 {
		dateReq = submittedAt[:10]
	}

	sb.WriteString(`<div class="doc-cover">`)
	sb.WriteString(`<div class="doc-eyebrow">Digital Reporting Requirements Specification</div>`)
	sb.WriteString(`<div class="doc-title">` + reportName + `</div>`)
	sb.WriteString(`<div class="doc-meta-row">`)
	sb.WriteString(`<span><strong>Version:</strong> ` + esc(version) + `</span>`)
	sb.WriteString(`<span><strong>Date Requested:</strong> ` + esc(dateReq) + `</span>`)
	sb.WriteString(`<span><strong>Status:</strong> ` + esc(status) + `</span>`)
	sb.WriteString(`<span><strong>Submitted By:</strong> ` + esc(e.SubmittedByUsername) + `</span>`)
	if submittedAt != "" {
		sb.WriteString(`<span><strong>Submitted At:</strong> ` + esc(submittedAt) + `</span>`)
	}
	sb.WriteString(`</div></div>`)

	// ── §1 Document Control ───────────────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">1 — Document Control</div>`)
	sb.WriteString(`<div class="field-grid">`)
	reqWriteFields(&sb, esc, [][2]string{
		{"Report ID", readString(docCtrl, "reportId")},
		{"Program Area", readString(docCtrl, "programArea")},
		{"Requesting Department", readString(docCtrl, "requestingDepartment")},
		{"Business Owner", readString(docCtrl, "businessOwner")},
		{"Data Analyst", readString(docCtrl, "dataAnalyst")},
		{"Report Developer", readString(docCtrl, "reportDeveloper")},
	})
	sb.WriteString(`</div></div>`)

	// ── §2 Report Overview ────────────────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">2 — Report Overview</div>`)
	sb.WriteString(`<div class="field-grid two-col">`)
	reqWriteFields(&sb, esc, [][2]string{
		{"Report Title", readString(overview, "reportTitle")},
		{"Reporting Frequency", readString(overview, "reportingFrequency")},
		{"Reporting Period", readString(overview, "reportingPeriod")},
		{"Target Audience", readString(overview, "targetAudience")},
	})
	sb.WriteString(`</div>`)
	if obj := strings.TrimSpace(readString(overview, "businessObjective")); obj != "" {
		sb.WriteString(`<div class="objective-text">` + esc(obj) + `</div>`)
	}
	sb.WriteString(`</div>`)

	// ── §3 Access & Security ──────────────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">3 — Access &amp; Security Requirements</div>`)
	sb.WriteString(`<div class="field-grid">`)
	reqWriteFields(&sb, esc, [][2]string{
		{"Authentication", readString(access, "authenticationRequirements")},
		{"User Roles", readString(access, "userRoles")},
		{"Permissions", readString(access, "permissions")},
	})
	sb.WriteString(`</div></div>`)

	// ── §4 Data Requirements ──────────────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">4 — Data Requirements</div>`)
	sb.WriteString(`<div class="field-grid two-col">`)
	reqWriteFields(&sb, esc, [][2]string{
		{"Data Sources", readString(dataReq, "dataSources")},
		{"Data Quality Requirements", readString(dataReq, "dataQualityRequirements")},
	})
	sb.WriteString(`</div>`)
	if br := strings.TrimSpace(readString(payload, "businessRules")); br != "" {
		sb.WriteString(`<div style="margin-top:8px;"><div class="field-label">Business Rules</div><div class="field-value" style="margin-top:2px;">` + esc(br) + `</div></div>`)
	}
	if ir := strings.TrimSpace(readString(payload, "indicatorRequirements")); ir != "" {
		sb.WriteString(`<div style="margin-top:8px;"><div class="field-label">Indicator Requirements</div><div class="field-value" style="margin-top:2px;">` + esc(ir) + `</div></div>`)
	}
	sb.WriteString(`</div>`)

	// ── §5 Report Filters ─────────────────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">5 — Report Filters</div>`)
	if selArr, ok := filters["selectedFilters"].([]interface{}); ok && len(selArr) > 0 {
		sb.WriteString(`<div class="field-label" style="margin-bottom:6px;">Selected Filters</div>`)
		sb.WriteString(`<div class="tag-list">`)
		for _, f := range selArr {
			sb.WriteString(`<span class="tag">` + esc(fmt.Sprint(f)) + `</span>`)
		}
		sb.WriteString(`</div>`)
	}
	if other := strings.TrimSpace(readString(filters, "otherFilters")); other != "" {
		sb.WriteString(`<div style="margin-top:8px;"><div class="field-label">Other Filters</div><div class="field-value" style="margin-top:2px;">` + esc(other) + `</div></div>`)
	}
	sb.WriteString(`</div>`)

	// ── §6 Visualization Specifications ───────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">6 — Visualization Specifications</div>`)
	if notes := strings.TrimSpace(readString(vizReq, "notes")); notes != "" {
		sb.WriteString(`<p style="margin-bottom:10px;font-style:italic;color:#555;">` + esc(notes) + `</p>`)
	}
	specs, _ := vizReq["specifications"].([]interface{})
	for si, specRaw := range specs {
		spec := asMap(specRaw)
		vTitle := readString(spec, "visualTitle")
		if vTitle == "" {
			vTitle = fmt.Sprintf("Visual %d", si+1)
		}
		vType := readString(spec, "visualType")
		vDesc := readString(spec, "visualDescription")

		sb.WriteString(`<div class="vis-spec">`)
		sb.WriteString(`<div class="vis-spec-head"><span class="vis-spec-title">` + esc(vTitle) + `</span><span class="vis-spec-type">` + esc(vType) + `</span></div>`)
		sb.WriteString(`<div class="vis-spec-body">`)
		if vDesc != "" {
			sb.WriteString(`<div class="vis-desc">` + esc(vDesc) + `</div>`)
		}

		indicators, _ := spec["indicators"].([]interface{})
		for _, indRaw := range indicators {
			ind := asMap(indRaw)
			iLabel := readString(ind, "indicatorLabel")
			if iLabel == "" {
				iLabel = "Indicator"
			}
			iMode := readString(ind, "mode")
			sb.WriteString(`<div class="indicator">`)
			sb.WriteString(`<div class="indicator-head"><span class="indicator-label">` + esc(iLabel) + `</span><span class="indicator-mode">` + esc(iMode) + `</span></div>`)
			sb.WriteString(`<div class="de-body">`)
			if iMode == "calculated" {
				numElems, _ := ind["numerator"].([]interface{})
				denElems, _ := ind["denominator"].([]interface{})
				numRole := "Numerator"
				denRole := "Denominator"
				if len(numElems) > 1 {
					numRole = "Numerator (SUM)"
				}
				if len(denElems) > 1 {
					denRole = "Denominator (SUM)"
				}
				sb.WriteString(`<div class="de-role">` + numRole + `</div>`)
				for _, de := range numElems {
					sb.WriteString(`<div class="de-item">` + esc(readString(asMap(de), "label")) + `</div>`)
				}
				sb.WriteString(`<div class="de-role">` + denRole + `</div>`)
				for _, de := range denElems {
					sb.WriteString(`<div class="de-item">` + esc(readString(asMap(de), "label")) + `</div>`)
				}
			} else {
				if elems, ok := ind["dataElements"].([]interface{}); ok {
					if len(elems) > 1 {
						sb.WriteString(`<div class="de-role">Data Elements (SUM)</div>`)
					}
					for _, de := range elems {
						sb.WriteString(`<div class="de-item">` + esc(readString(asMap(de), "label")) + `</div>`)
					}
				}
			}
			sb.WriteString(`</div></div>`) // de-body, indicator
		}
		sb.WriteString(`</div></div>`) // vis-spec-body, vis-spec
	}
	if len(specs) == 0 {
		sb.WriteString(`<p style="color:#888;font-style:italic;">No visualization specifications provided.</p>`)
	}
	sb.WriteString(`</div>`)

	// ── §7 Performance Requirements ───────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">7 — Performance Requirements</div>`)
	sb.WriteString(`<div class="field-grid">`)
	reqWriteFields(&sb, esc, [][2]string{
		{"Expected Users", readString(perf, "expectedUsers")},
		{"Load Time (seconds)", readString(perf, "loadTimeSeconds")},
		{"Export Time (seconds)", readString(perf, "exportTimeSeconds")},
	})
	sb.WriteString(`</div></div>`)

	// ── §8 Export Requirements ────────────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">8 — Export Requirements</div>`)
	if expArr, ok := payload["exportRequirements"].([]interface{}); ok && len(expArr) > 0 {
		sb.WriteString(`<div class="tag-list">`)
		for _, x := range expArr {
			sb.WriteString(`<span class="tag">` + esc(fmt.Sprint(x)) + `</span>`)
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)

	// ── §9 Geospatial Requirements — omitted: maps are fully described in §6 ──

	// ── §10 Validation Requirements ───────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">10 — Validation Requirements</div>`)
	if valArr, ok := payload["validationRequirements"].([]interface{}); ok && len(valArr) > 0 {
		sb.WriteString(`<div class="tag-list">`)
		for _, v := range valArr {
			sb.WriteString(`<span class="tag check">` + esc(fmt.Sprint(v)) + `</span>`)
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)

	// ── §11 Acceptance Criteria ───────────────────────────────────────────────
	sb.WriteString(`<div class="section"><div class="section-heading">11 — Acceptance Criteria</div>`)
	if acArr, ok := payload["acceptanceCriteria"].([]interface{}); ok && len(acArr) > 0 {
		sb.WriteString(`<div class="tag-list">`)
		for _, ac := range acArr {
			sb.WriteString(`<span class="tag check">` + esc(fmt.Sprint(ac)) + `</span>`)
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)

	// ── §12 Approvals ─────────────────────────────────────────────────────────
	sb.WriteString(`<div class="section" style="page-break-inside:avoid;"><div class="section-heading">12 — Approvals</div>`)
	sb.WriteString(`<table class="approvals-table"><thead><tr>`)
	sb.WriteString(`<th>Role</th><th>Name</th><th>Date</th><th>Signature</th>`)
	sb.WriteString(`</tr></thead><tbody>`)
	for _, row := range [][3]string{
		{"Prepared By", readString(approvals, "preparedBy"), readString(approvals, "preparedDate")},
		{"Reviewed By", readString(approvals, "reviewedBy"), readString(approvals, "reviewedDate")},
		{"Approved By", readString(approvals, "Commissioner"), readString(approvals, "approvedDate")},
	} {
		sb.WriteString(`<tr><td>` + esc(row[0]) + `</td><td>` + esc(row[1]) + `</td><td>` + esc(row[2]) + `</td><td></td></tr>`)
	}
	sb.WriteString(`</tbody></table></div>`)

	sb.WriteString(`</body></html>`)
	return sb.String(), nil
}

// reqWriteFields writes a list of label/value pairs as field-grid items.
func reqWriteFields(sb *strings.Builder, esc func(string) string, fields [][2]string) {
	for _, f := range fields {
		sb.WriteString(`<div class="field"><div class="field-label">` + esc(f[0]) + `</div>`)
		sb.WriteString(`<div class="field-value">` + esc(f[1]) + `</div></div>`)
	}
}
