package api

import (
	"encoding/json"
	"net/http"

	"learningcrm/pkg/model"
	"learningcrm/pkg/store"
)

// ─── Stakeholders (P35) ──────────────────────────────────────────────────────

func ListStakeholdersHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListStakeholders(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateStakeholderHandler(w http.ResponseWriter, r *http.Request) {
	var s model.Stakeholder
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if s.Name == "" || s.Kind == "" {
		writeError(w, http.StatusBadRequest, "name and kind are required")
		return
	}
	s.TenantID = actorTenant(r)
	if err := store.CreateStakeholder(r.Context(), &s); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func GetStakeholderHandler(w http.ResponseWriter, r *http.Request) {
	s, err := store.GetStakeholder(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "stakeholder not found")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func DeleteStakeholderHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteStakeholder(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Engagements (P35) ───────────────────────────────────────────────────────

func ListEngagementsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListEngagements(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateEngagementHandler(w http.ResponseWriter, r *http.Request) {
	var e model.Engagement
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if e.StakeholderID == "" || e.Title == "" {
		writeError(w, http.StatusBadRequest, "stakeholder_id and title are required")
		return
	}
	if _, err := store.GetStakeholder(r.Context(), e.StakeholderID); err != nil {
		writeError(w, http.StatusBadRequest, "stakeholder does not exist")
		return
	}
	e.TenantID = actorTenant(r)
	if err := store.CreateEngagement(r.Context(), &e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// ─── Stewardship Actions (P35) ───────────────────────────────────────────────

func ListStewardshipHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListStewardshipActions(r.Context(), actorTenant(r), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateStewardshipHandler(w http.ResponseWriter, r *http.Request) {
	var a model.StewardshipAction
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if a.Action == "" {
		writeError(w, http.StatusBadRequest, "action is required")
		return
	}
	a.TenantID = actorTenant(r)
	if err := store.CreateStewardshipAction(r.Context(), &a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func UpdateStewardshipHandler(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetStewardshipAction(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "stewardship action not found")
		return
	}
	var body model.StewardshipAction
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	a.Status, a.Owner, a.Action = body.Status, body.Owner, body.Action
	a.DueAt, a.EntityType, a.EntityID = body.DueAt, body.EntityType, body.EntityID
	if err := store.UpdateStewardshipAction(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// ─── Cross-app object linkage ────────────────────────────────────────────────

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

// ─── Overview ────────────────────────────────────────────────────────────────

func SummaryHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service":   SourceApplication,
		"tenant_id": actorTenant(r),
		"endpoints": []string{"/courses", "/certificates", "/badges", "/leads", "/accounts", "/opportunities", "/partners", "/service-requests", "/invoices", "/stakeholders", "/engagements", "/stewardship"},
	})
}