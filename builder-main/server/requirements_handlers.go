package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	requirementsMaxBodyBytes = 1 << 20 // 1 MiB
)

type requirementsSubmitResponse struct {
	ID        int64  `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	Message   string `json:"message"`
}

type requirementsListResponse struct {
	Entries []RequirementsSpecEntry `json:"entries"`
	HasMore bool                    `json:"hasMore"`
	NextCur int64                   `json:"nextCursor,omitempty"`
}

// RequirementsSpecHandler accepts digital reporting requirements submissions.
// Route: POST /api/requirements-specs
// Route: GET  /api/requirements-specs
func RequirementsSpecHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleRequirementsSubmit(w, r)
	case http.MethodGet:
		handleRequirementsList(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleRequirementsSubmit(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, requirementsMaxBodyBytes))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "request body too large or unreadable")
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	docControl := asMap(payload["documentControl"])
	reportOverview := asMap(payload["reportOverview"])

	reportName := strings.TrimSpace(readString(docControl, "reportName"))
	businessObjective := strings.TrimSpace(readString(reportOverview, "businessObjective"))
	if reportName == "" {
		writeJSONError(w, http.StatusBadRequest, "documentControl.reportName is required")
		return
	}
	if businessObjective == "" {
		writeJSONError(w, http.StatusBadRequest, "reportOverview.businessObjective is required")
		return
	}

	// Server-owned metadata prevents clients from spoofing submit time.
	payload["serverMetadata"] = map[string]interface{}{
		"submittedAt": time.Now().UTC().Format(time.RFC3339),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		logErrorCtx(ctx, "Requirements payload marshal failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to process submission")
		return
	}

	claims := GetUserClaims(r)
	userSub, username, email := identityFromClaims(claims, r)

	entry := RequirementsSpecEntry{
		ReportName:           reportName,
		ReportID:             strings.TrimSpace(readString(docControl, "reportId")),
		ProgramArea:          strings.TrimSpace(readString(docControl, "programArea")),
		RequestingDepartment: strings.TrimSpace(readString(docControl, "requestingDepartment")),
		BusinessOwner:        strings.TrimSpace(readString(docControl, "businessOwner")),
		DataAnalyst:          strings.TrimSpace(readString(docControl, "dataAnalyst")),
		ReportDeveloper:      strings.TrimSpace(readString(docControl, "reportDeveloper")),
		DateRequested:        strings.TrimSpace(readString(docControl, "dateRequested")),
		Version:              strings.TrimSpace(readString(docControl, "version")),
		Status:               strings.TrimSpace(readString(docControl, "status")),
		SubmittedBySub:       userSub,
		SubmittedByUsername:  username,
		SubmittedByEmail:     email,
		PayloadJSON:          string(payloadJSON),
	}

	id, err := InsertRequirementsSpec(ctx, entry)
	if err != nil {
		logErrorCtx(ctx, "Requirements insert failed (report=%s user=%s): %v", reportName, username, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to save requirements")
		return
	}

	logInfoCtx(ctx, "Requirements saved: id=%d report=%s user=%s", id, reportName, username)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(requirementsSubmitResponse{
		ID:        id,
		CreatedAt: time.Now().Unix(),
		Message:   "requirements submitted successfully",
	})
}

func handleRequirementsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	var before int64
	if v := r.URL.Query().Get("before"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			before = n
		}
	}

	entries, err := ListRequirementsSpecs(ctx, limit+1, before)
	if err != nil {
		logErrorCtx(ctx, "Requirements list failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load requirements")
		return
	}

	resp := requirementsListResponse{Entries: entries}
	if len(entries) > limit {
		resp.Entries = entries[:limit]
		resp.HasMore = true
		resp.NextCur = resp.Entries[len(resp.Entries)-1].CreatedAt
	}
	if resp.Entries == nil {
		resp.Entries = []RequirementsSpecEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func readString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// handleRequirementsGetOne returns a single requirements spec as JSON.
// Called by RequirementsSpecPDFHandler for GET /api/requirements-specs/{id}
func handleRequirementsGetOne(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid requirements spec ID")
		return
	}
	ctx := r.Context()
	entry, err := GetRequirementsSpecByID(ctx, id)
	if err != nil {
		logErrorCtx(ctx, "Requirements GET id=%d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to fetch requirements spec")
		return
	}
	if entry == nil {
		writeJSONError(w, http.StatusNotFound, "requirements spec not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entry)
}

// handleRequirementsUpdateOne updates an existing requirements spec.
// Called by RequirementsSpecPDFHandler for PUT /api/requirements-specs/{id}
func handleRequirementsUpdateOne(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid requirements spec ID")
		return
	}

	ctx := r.Context()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, requirementsMaxBodyBytes))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "request body too large or unreadable")
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	docControl := asMap(payload["documentControl"])
	reportOverview := asMap(payload["reportOverview"])

	reportName := strings.TrimSpace(readString(docControl, "reportName"))
	businessObjective := strings.TrimSpace(readString(reportOverview, "businessObjective"))
	if reportName == "" {
		writeJSONError(w, http.StatusBadRequest, "documentControl.reportName is required")
		return
	}
	if businessObjective == "" {
		writeJSONError(w, http.StatusBadRequest, "reportOverview.businessObjective is required")
		return
	}

	// Preserve original server metadata; add an updatedAt timestamp.
	if sm, ok := payload["serverMetadata"].(map[string]interface{}); ok {
		sm["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
	} else {
		payload["serverMetadata"] = map[string]interface{}{
			"updatedAt": time.Now().UTC().Format(time.RFC3339),
		}
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		logErrorCtx(ctx, "Requirements update marshal failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to process update")
		return
	}

	claims := GetUserClaims(r)
	_, username, _ := identityFromClaims(claims, r)

	entry := RequirementsSpecEntry{
		ReportName:           reportName,
		ReportID:             strings.TrimSpace(readString(docControl, "reportId")),
		ProgramArea:          strings.TrimSpace(readString(docControl, "programArea")),
		RequestingDepartment: strings.TrimSpace(readString(docControl, "requestingDepartment")),
		BusinessOwner:        strings.TrimSpace(readString(docControl, "businessOwner")),
		DataAnalyst:          strings.TrimSpace(readString(docControl, "dataAnalyst")),
		ReportDeveloper:      strings.TrimSpace(readString(docControl, "reportDeveloper")),
		DateRequested:        strings.TrimSpace(readString(docControl, "dateRequested")),
		Version:              strings.TrimSpace(readString(docControl, "version")),
		Status:               strings.TrimSpace(readString(docControl, "status")),
		PayloadJSON:          string(payloadJSON),
	}

	found, err := UpdateRequirementsSpec(ctx, id, entry)
	if err != nil {
		logErrorCtx(ctx, "Requirements update failed (id=%d user=%s): %v", id, username, err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update requirements")
		return
	}
	if !found {
		writeJSONError(w, http.StatusNotFound, "requirements spec not found")
		return
	}

	logInfoCtx(ctx, "Requirements updated: id=%d report=%s user=%s", id, reportName, username)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"message": "requirements updated successfully",
	})
}
