package api

import (
	"encoding/json"
	"net/http"

	"knowledgeportal/pkg/model"
	"knowledgeportal/pkg/store"
)

// ─── Cross-application object links ────────────────────────────────────────

func CreateLinkHandler(w http.ResponseWriter, r *http.Request) {
	var l model.ObjectLink
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if l.SourceType == "" || l.SourceID == "" || l.TargetType == "" || l.TargetID == "" {
		writeError(w, http.StatusBadRequest, "source_type/source_id and target_type/target_id are required")
		return
	}
	l.TenantID = actorTenant(r)
	if err := store.CreateObjectLink(r.Context(), &l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "object.link.created", "object_link", l.ID, map[string]interface{}{"source": l.SourceType + ":" + l.SourceID, "target": l.TargetType + ":" + l.TargetID})
	recordAudit(r, "object.link.created", "object_link", l.ID, map[string]interface{}{"source": l.SourceType, "target": l.TargetType})
	writeJSON(w, http.StatusCreated, l)
}

func ListLinksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	objectType := q.Get("type")
	objectID := q.Get("id")
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

// ─── Subscriptions & feedback consoles ─────────────────────────────────────

func ListSubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	subs, err := store.ListSubscriptions(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, subs)
}

func ListFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListFeedback(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func SummaryHandler(w http.ResponseWriter, r *http.Request) {
	s, err := store.PortalSummary(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}
