package api

import (
	"encoding/json"
	"net/http"

	"bpmhub/pkg/model"
	"bpmhub/pkg/store"
)

// ─── Process Mining / Operational Analytics (P48) ───────────────────────────

func MiningSummaryHandler(w http.ResponseWriter, r *http.Request) {
	summary, err := store.MiningSummary(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// ─── Cross-app object linkage ────────────────────────────────────────────────

func CreateLinkHandler(w http.ResponseWriter, r *http.Request) {
	var l model.ObjectLink
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if l.SourceType == "" || l.SourceID == "" || l.TargetType == "" || l.TargetID == "" {
		writeError(w, http.StatusBadRequest, "source and target are required")
		return
	}
	l.TenantID = actorTenant(r)
	if err := store.CreateObjectLink(r.Context(), &l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "object.link.created", "object_link", l.ID,
		map[string]interface{}{"source": l.SourceType + ":" + l.SourceID, "target": l.TargetType + ":" + l.TargetID})
	writeJSON(w, http.StatusCreated, l)
}

func ListLinksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	objectType, objectID := q.Get("type"), q.Get("id")
	if objectType == "" || objectID == "" {
		writeError(w, http.StatusBadRequest, "type and id query parameters are required")
		return
	}
	links, err := store.ListObjectLinks(r.Context(), objectType, objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, links)
}

func SummaryHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service":   SourceApplication,
		"tenant_id": actorTenant(r),
		"endpoints": []string{"/processes", "/instances", "/cases", "/tasks", "/automation", "/mining/summary", "/links"},
	})
}