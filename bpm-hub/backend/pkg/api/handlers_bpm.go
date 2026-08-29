package api

import (
	"encoding/json"
	"net/http"
	"time"

	"bpmhub/pkg/model"
	"bpmhub/pkg/store"
)

// ─── Process Definitions (P48) ───────────────────────────────────────────────

func ListProcessesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListProcessDefinitions(r.Context(), actorTenant(r), actorWorkspace(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateProcessHandler(w http.ResponseWriter, r *http.Request) {
	var p model.ProcessDefinition
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if p.Name == "" || p.Key == "" || p.StartNode == "" {
		writeError(w, http.StatusBadRequest, "name, key and start_node are required")
		return
	}
	p.TenantID = actorTenant(r)
	p.WorkspaceID = actorWorkspace(r)
	p.CreatedBy = actorID(r)
	if err := store.CreateProcessDefinition(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "bpm.definition.created", "process_definition", p.ID, map[string]interface{}{"key": p.Key})
	writeJSON(w, http.StatusCreated, p)
}

func GetProcessHandler(w http.ResponseWriter, r *http.Request) {
	p, err := store.GetProcessDefinition(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "process definition not found")
		return
	}
	if !workspaceAllowsRead(p.WorkspaceID, actorWorkspace(r)) {
		writeError(w, http.StatusNotFound, "process definition not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func PublishProcessHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetProcessDefinition(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "process definition not found")
		return
	}
	if err := store.UpdateProcessDefinitionStatus(r.Context(), id, "active"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}

// ─── Instances (P48 workflow engine) ─────────────────────────────────────────

// StartInstanceHandler boots a process instance at the definition start node,
// creates a work item for it and emits process.started.
func StartInstanceHandler(w http.ResponseWriter, r *http.Request) {
	defID := varsOf(r)["id"]
	def, err := store.GetProcessDefinition(r.Context(), defID)
	if err != nil {
		writeError(w, http.StatusNotFound, "process definition not found")
		return
	}
	if def.Status != "active" {
		writeError(w, http.StatusBadRequest, "process definition is not active")
		return
	}
	var body struct {
		Context  map[string]interface{} `json:"context"`
		Assignee string                 `json:"assignee"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	now := time.Now().UTC()
	inst := model.ProcessInstance{
		TenantID:     actorTenant(r),
		WorkspaceID:  def.WorkspaceID,
		DefinitionID: defID,
		Status:       "running",
		CurrentNode:  def.StartNode,
		Context:      body.Context,
		StartedAt:    &now,
		CreatedBy:    actorID(r),
	}
	if err := store.CreateInstance(r.Context(), &inst); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	wi := model.WorkItem{
		TenantID: inst.TenantID, InstanceID: inst.ID, NodeID: def.StartNode,
		Name: "Start: " + def.StartNode, Assignee: body.Assignee, Status: "todo",
	}
	if err := store.CreateWorkItem(r.Context(), &wi); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "process.started", "process_instance", inst.ID,
		map[string]interface{}{"definition_id": defID, "start_node": def.StartNode})
	_ = store.LogActivity(r.Context(), &model.ActivityLog{
		TenantID: inst.TenantID, InstanceID: inst.ID, Action: "instance.start", NodeID: def.StartNode, Actor: actorID(r),
	})
	writeJSON(w, http.StatusCreated, map[string]interface{}{"instance": inst, "work_item": wi})
}

func ListInstancesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := store.ListInstances(r.Context(), actorTenant(r), q.Get("definition_id"), q.Get("status"), actorWorkspace(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func GetInstanceHandler(w http.ResponseWriter, r *http.Request) {
	inst, err := store.GetInstance(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	writeJSON(w, http.StatusOK, inst)
}

func ListInstanceTasksHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetInstance(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	items, err := store.ListWorkItemsByInstance(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func ListInstanceLogHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	items, err := store.ListActivity(r.Context(), actorTenant(r), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// AdvanceInstanceHandler moves an instance along its transitions. When a node
// has no outgoing transition the process completes and process.completed is
// emitted.
func AdvanceInstanceHandler(w http.ResponseWriter, r *http.Request) {
	instID := varsOf(r)["id"]
	inst, err := store.GetInstance(r.Context(), instID)
	if err != nil {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	def, err := store.GetProcessDefinition(r.Context(), inst.DefinitionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "process definition not found")
		return
	}
	next := NextNode(def.Transitions, inst.CurrentNode)
	if next == "" {
		if err := store.CompleteInstance(r.Context(), instID, "completed"); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		emitEvent(r.Context(), "process.completed", "process_instance", instID,
			map[string]interface{}{"definition_id": inst.DefinitionID})
		_ = store.LogActivity(r.Context(), &model.ActivityLog{
			TenantID: inst.TenantID, InstanceID: instID, Action: "instance.completed", NodeID: inst.CurrentNode, Actor: actorID(r),
		})
		inst.Status = "completed"
		writeJSON(w, http.StatusOK, map[string]interface{}{"instance": inst, "completed": true})
		return
	}
	if err := store.AdvanceInstance(r.Context(), instID, next); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	wi := model.WorkItem{
		TenantID: inst.TenantID, InstanceID: instID, NodeID: next,
		Name: next, Status: "todo",
	}
	if err := store.CreateWorkItem(r.Context(), &wi); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "task.created", "work_item", wi.ID,
		map[string]interface{}{"instance_id": instID, "node_id": next})
	_ = store.LogActivity(r.Context(), &model.ActivityLog{
		TenantID: inst.TenantID, InstanceID: instID, Action: "advance", NodeID: next, Actor: actorID(r),
	})
	inst.CurrentNode = next
	writeJSON(w, http.StatusOK, map[string]interface{}{"instance": inst, "next_node": next, "work_item": wi})
}

func CancelInstanceHandler(w http.ResponseWriter, r *http.Request) {
	instID := varsOf(r)["id"]
	if err := store.CompleteInstance(r.Context(), instID, "cancelled"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// NextNode resolves the target of a transition map {"from": "to"}.
func NextNode(transitions map[string]interface{}, from string) string {
	if transitions == nil {
		return ""
	}
	if v, ok := transitions[from]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}