package api

import (
	"encoding/json"
	"net/http"
	"time"

	"aiengines/pkg/model"
	"aiengines/pkg/store"
)

// ─── Agents (P31) ────────────────────────────────────────────────────────────

func ListAgentsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListAgents(r.Context(), actorTenant(r), r.URL.Query().Get("status"), actorWorkspace(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateAgentHandler(w http.ResponseWriter, r *http.Request) {
	var a model.Agent
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if a.Name == "" || a.Role == "" {
		writeError(w, http.StatusBadRequest, "name and role are required")
		return
	}
	a.TenantID = actorTenant(r)
	a.WorkspaceID = actorWorkspace(r)
	a.CreatedBy = actorID(r)
	if err := store.CreateAgent(r.Context(), &a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "agent.registered", "agent", a.ID, map[string]interface{}{"name": a.Name, "role": a.Role})
	writeJSON(w, http.StatusCreated, a)
}

func GetAgentHandler(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetAgent(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if !workspaceAllowsRead(a.WorkspaceID, actorWorkspace(r)) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func UpdateAgentHandler(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetAgent(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	var body model.Agent
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	a.Name, a.Role = body.Name, body.Role
	a.Persona, a.Capabilities, a.SafetyPolicy = body.Persona, body.Capabilities, body.SafetyPolicy
	a.Status = body.Status
	if err := store.UpdateAgent(r.Context(), a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "agent.updated", "agent", a.ID, map[string]interface{}{"status": a.Status})
	writeJSON(w, http.StatusOK, a)
}

// ─── Agent Tasks & orchestration (P31) ───────────────────────────────────────

func ListAgentTasksHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListTasks(r.Context(), varsOf(r)["id"], r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateAgentTaskHandler enqueues a new agent task, executes it with the
// deterministic runtime, emits agent.task.created and agent.task.completed.
func CreateAgentTaskHandler(w http.ResponseWriter, r *http.Request) {
	agentID := varsOf(r)["id"]
	if _, err := store.GetAgent(r.Context(), agentID); err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	var body model.AgentTask
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.Name == "" || body.TaskType == "" {
		writeError(w, http.StatusBadRequest, "name and task_type are required")
		return
	}
	body.TenantID = actorTenant(r)
	body.AgentID = agentID
	body.TriggerType = "manual"
	if err := store.CreateTask(r.Context(), &body); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "agent.task.created", "agent_task", body.ID, map[string]interface{}{"agent_id": agentID, "task_type": body.TaskType})

	now := time.Now().UTC()
	body.StartedAt = &now
	_ = store.StartTask(r.Context(), body.ID)

	// Deterministic agent execution: echo task type + a canned result.
	result := map[string]interface{}{"completed_by": "runtime", "task_type": body.TaskType, "summary": "processed"}
	if err := store.CompleteTask(r.Context(), body.ID, "completed", result, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	body.Status = "completed"
	body.Result = result
	f := time.Now().UTC()
	body.FinishedAt = &f

	emitEvent(r.Context(), "agent.task.completed", "agent_task", body.ID, map[string]interface{}{"agent_id": agentID})
	recordAudit(r, "agent.task.completed", "agent_task", body.ID, map[string]interface{}{"agent_id": agentID})
	writeJSON(w, http.StatusCreated, body)
}

// ─── Agent Memory (P31) ──────────────────────────────────────────────────────

func ListAgentMemoryHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListMemory(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateAgentMemoryHandler(w http.ResponseWriter, r *http.Request) {
	agentID := varsOf(r)["id"]
	if _, err := store.GetAgent(r.Context(), agentID); err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	var m model.AgentMemory
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if m.Content == nil {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	m.TenantID = actorTenant(r)
	m.AgentID = agentID
	if err := store.CreateMemory(r.Context(), &m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// ─── Agent Communications Bus (P31) ──────────────────────────────────────────

func SendAgentMessageHandler(w http.ResponseWriter, r *http.Request) {
	var m model.AgentMessage
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if m.FromAgent == "" || m.ToAgent == "" {
		writeError(w, http.StatusBadRequest, "from_agent and to_agent are required")
		return
	}
	m.TenantID = actorTenant(r)
	if err := store.CreateMessage(r.Context(), &m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "agent.action.taken", "agent_message", m.ID, map[string]interface{}{
		"from_agent": m.FromAgent, "to_agent": m.ToAgent, "type": m.Type})
	writeJSON(w, http.StatusCreated, m)
}

func ListAgentMessagesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListMessages(r.Context(), actorTenant(r), r.URL.Query().Get("agent_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── AI Safety & Governance Guardrails (P31) ─────────────────────────────────

func ListGovernanceHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListGovernance(r.Context(), actorTenant(r), r.URL.Query().Get("verdict"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func ReviewGovernanceHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	var body struct {
		Verdict string `json:"verdict"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.Verdict != "auto_approved" && body.Verdict != "blocked" && body.Verdict != "human_review" {
		writeError(w, http.StatusBadRequest, "verdict must be auto_approved|blocked|human_review")
		return
	}
	if err := store.ReviewGovernance(r.Context(), id, body.Verdict, actorID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "governance.reviewed", "governance_action", id, map[string]interface{}{"verdict": body.Verdict})
	writeJSON(w, http.StatusOK, map[string]string{"status": "reviewed", "verdict": body.Verdict})
}

// AgentRuntimeStatsHandler returns a compact runtime view for orchestration tooling.
func AgentRuntimeStatsHandler(w http.ResponseWriter, r *http.Request) {
	agents, err := store.ListAgents(r.Context(), actorTenant(r), "", actorWorkspace(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	active, paused := 0, 0
	for _, a := range agents {
		if a.Status == "active" {
			active++
		} else if a.Status == "paused" {
			paused++
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":  len(agents),
		"active": active,
		"paused": paused,
	})
}