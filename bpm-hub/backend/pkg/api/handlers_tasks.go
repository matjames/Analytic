package api

import (
	"encoding/json"
	"net/http"

	"bpmhub/pkg/model"
	"bpmhub/pkg/store"
)

// ─── Task Management (P48) ───────────────────────────────────────────────────

func ListTasksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := store.ListWorkItems(r.Context(), actorTenant(r), q.Get("assignee"), q.Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	item, err := store.GetWorkItem(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// UpdateTaskHandler transitions a task; completing emits task.completed.
func UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	item, err := store.GetWorkItem(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	var body model.WorkItem
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.Status != "" {
		item.Status = body.Status
	}
	if body.Assignee != "" {
		item.Assignee = body.Assignee
	}
	item.Name = body.Name
	item.Payload = body.Payload
	if err := store.UpdateWorkItem(r.Context(), item); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if item.Status == "done" {
		emitEvent(r.Context(), "task.completed", "work_item", item.ID,
			map[string]interface{}{"instance_id": item.InstanceID, "node_id": item.NodeID})
		_ = store.LogActivity(r.Context(), &model.ActivityLog{
			TenantID: item.TenantID, InstanceID: item.InstanceID, Action: "task.completed", NodeID: item.NodeID, Actor: actorID(r),
		})
		recordAudit(r, "task.completed", "work_item", item.ID, map[string]interface{}{"assignee": item.Assignee})
	}
	writeJSON(w, http.StatusOK, item)
}

// ─── Automation Service (P48) ────────────────────────────────────────────────

func ListAutomationHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListAutomationRules(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateAutomationHandler(w http.ResponseWriter, r *http.Request) {
	var rule model.AutomationRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if rule.Name == "" || rule.TriggerEvent == "" {
		writeError(w, http.StatusBadRequest, "name and trigger_event are required")
		return
	}
	rule.TenantID = actorTenant(r)
	if err := store.CreateAutomationRule(r.Context(), &rule); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func ToggleAutomationHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	var body struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := store.SetAutomationRuleEnabled(r.Context(), id, body.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": body.Enabled})
}

// EvaluateAutomationHandler applies enabled rules for an event type, emitting
// automation.triggered for each match (event-driven automation).
func EvaluateAutomationHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EventType string                 `json:"event_type"`
		Payload   map[string]interface{} `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.EventType == "" {
		writeError(w, http.StatusBadRequest, "event_type is required")
		return
	}
	rules, err := store.FindAutomationRulesForEvent(r.Context(), actorTenant(r), body.EventType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	matched := make([]string, 0, len(rules))
	for _, rule := range rules {
		emitEvent(r.Context(), "automation.triggered", "automation_rule", rule.ID,
			map[string]interface{}{"event_type": body.EventType, "actions": rule.Actions})
		matched = append(matched, rule.ID)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"event_type": body.EventType, "matched_rules": matched, "count": len(matched)})
}