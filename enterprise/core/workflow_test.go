package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── Phase V: Automated Tests for Workflow & Decision Automation ─────
// Covers: workflow creation, execution, conditions, actions, approvals,
// escalations, permissions, idempotency, retries, failures, audit
// records, cross-application actions, task dependencies, SLA management,
// decision records, AI recommendations, Action Centre and My Work.

// ─── Workflow Definition Tests ─────────────────────────────────────

func TestWorkflowTemplatesBootstrapped(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/workflows", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count     int                  `json:"count"`
		Workflows []WorkflowDefinition `json:"workflows"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count < 5 {
		t.Fatalf("expected at least 5 workflow templates, got %d", resp.Count)
	}
}

func TestWorkflowDefinitionCreation(t *testing.T) {
	ts := setupTestRouter()
	body := `{
		"name": "Test Workflow",
		"description": "Test workflow definition",
		"category": "general",
		"status": "draft",
		"trigger": {"type": "manual"},
		"steps": [
			{"id": "s1", "name": "Notify", "type": "action", "action": "notify", "order": 1,
			 "params": {"title": "Test", "body": "Test notification", "priority": "medium"}}
		]
	}`
	w := ts.do("POST", "/api/workflows", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var def WorkflowDefinition
	if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if def.ID == "" {
		t.Fatal("expected workflow ID")
	}
	if def.Status != "draft" {
		t.Fatalf("expected draft status, got %s", def.Status)
	}
}

func TestWorkflowActivation(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("POST", "/api/workflows/wf_project_onboarding/activate", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var def WorkflowDefinition
	if err := json.Unmarshal(w.Body.Bytes(), &def); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if def.Status != "active" {
		t.Fatalf("expected active status, got %s", def.Status)
	}
}

func TestWorkflowConditionEvaluation(t *testing.T) {
	// Test equals
	cond := WorkflowCondition{Field: "status", Operator: "equals", Value: "active"}
	ctx := map[string]interface{}{"status": "active"}
	if !evaluateWorkflowCondition(cond.Field, cond.Operator, cond.Value, ctx) {
		t.Fatal("expected condition to pass for equals")
	}
	ctx2 := map[string]interface{}{"status": "inactive"}
	if evaluateWorkflowCondition(cond.Field, cond.Operator, cond.Value, ctx2) {
		t.Fatal("expected condition to fail for equals")
	}

	// Test greater_than
	cond2 := WorkflowCondition{Field: "value", Operator: "greater_than", Value: float64(50)}
	ctx3 := map[string]interface{}{"value": float64(75)}
	if !evaluateWorkflowCondition(cond2.Field, cond2.Operator, cond2.Value, ctx3) {
		t.Fatal("expected condition to pass for greater_than")
	}
	ctx4 := map[string]interface{}{"value": float64(25)}
	if evaluateWorkflowCondition(cond2.Field, cond2.Operator, cond2.Value, ctx4) {
		t.Fatal("expected condition to fail for greater_than")
	}

	// Test contains (case-sensitive matching)
	cond3 := WorkflowCondition{Field: "name", Operator: "contains", Value: "survey"}
	ctx5 := map[string]interface{}{"name": "survey completion report"}
	if !evaluateWorkflowCondition(cond3.Field, cond3.Operator, cond3.Value, ctx5) {
		t.Fatal("expected condition to pass for contains")
	}
	ctx6 := map[string]interface{}{"name": "Missing"}
	if evaluateWorkflowCondition(cond3.Field, cond3.Operator, cond3.Value, ctx6) {
		t.Fatal("expected condition to fail for contains when value not found")
	}
}

// ─── Workflow Instance Tests ───────────────────────────────────────

func TestWorkflowInstanceStart(t *testing.T) {
	ts := setupTestRouter()
	body := `{
		"entity_type": "project",
		"entity_id": "PRJ-TEST-1",
		"project_id": "PRJ-TEST-1",
		"actor": "test-user",
		"context": {"name": "Test Project"}
	}`
	w := ts.do("POST", "/api/workflows/wf_project_onboarding/run", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp struct {
		Status     string `json:"status"`
		InstanceID string `json:"instance_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Status != "started" {
		t.Fatalf("expected started status, got %s", resp.Status)
	}
	if resp.InstanceID == "" {
		t.Fatal("expected instance ID")
	}
}

func TestWorkflowInstanceList(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/workflows/instances", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count     int                `json:"count"`
		Instances []WorkflowInstance `json:"instances"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

func TestWorkflowCancel(t *testing.T) {
	ts := setupTestRouter()
	// Start a workflow first
	body := `{
		"entity_type": "project",
		"entity_id": "PRJ-CANCEL-1",
		"project_id": "PRJ-CANCEL-1",
		"actor": "test-user"
	}`
	w := ts.do("POST", "/api/workflows/wf_project_onboarding/run", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp struct {
		InstanceID string `json:"instance_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// Cancel it
	w2 := ts.do("POST", "/api/workflows/instances/"+resp.InstanceID+"/cancel", "")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}

// ─── Approval Engine Tests ─────────────────────────────────────────

func TestApprovalCreation(t *testing.T) {
	ts := setupTestRouter()
	body := `{
		"title": "Test Approval",
		"description": "Test approval request",
		"type": "single",
		"approvers": ["approver-1"],
		"requester": "requester-1"
	}`
	w := ts.do("POST", "/api/approvals", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var req ApprovalRequest
	if err := json.Unmarshal(w.Body.Bytes(), &req); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if req.ID == "" {
		t.Fatal("expected approval ID")
	}
	if req.Status != "pending" {
		t.Fatalf("expected pending status, got %s", req.Status)
	}
}

func TestApprovalApprove(t *testing.T) {
	ts := setupTestRouter()
	// Create approval
	body := `{
		"title": "Test Approval Approve",
		"description": "Test approval",
		"type": "single",
		"approvers": ["approver-2"],
		"requester": "requester-2"
	}`
	w := ts.do("POST", "/api/approvals", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var req ApprovalRequest
	_ = json.Unmarshal(w.Body.Bytes(), &req)

	// Approve it (acting as the configured approver via verified JWT)
	w2 := ts.doAs("POST", "/api/approvals/"+req.ID+"/approve", `{"comment": "Approved"}`, "approver-2")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var updated ApprovalRequest
	if err := json.Unmarshal(w2.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if updated.Status != "approved" {
		t.Fatalf("expected approved status, got %s", updated.Status)
	}
}

func TestApprovalReject(t *testing.T) {
	ts := setupTestRouter()
	body := `{
		"title": "Test Approval Reject",
		"description": "Test approval",
		"type": "single",
		"approvers": ["approver-3"],
		"requester": "requester-3"
	}`
	w := ts.do("POST", "/api/approvals", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var req ApprovalRequest
	_ = json.Unmarshal(w.Body.Bytes(), &req)

	w2 := ts.doAs("POST", "/api/approvals/"+req.ID+"/reject", `{"comment": "Rejected"}`, "approver-3")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
	var updated ApprovalRequest
	_ = json.Unmarshal(w2.Body.Bytes(), &updated)
	if updated.Status != "rejected" {
		t.Fatalf("expected rejected status, got %s", updated.Status)
	}
}

func TestApprovalUnauthorized(t *testing.T) {
	ts := setupTestRouter()
	body := `{
		"title": "Test Approval Unauthorized",
		"description": "Test approval",
		"type": "single",
		"approvers": ["approver-4"],
		"requester": "requester-4"
	}`
	w := ts.do("POST", "/api/approvals", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var req ApprovalRequest
	_ = json.Unmarshal(w.Body.Bytes(), &req)

	// Try to approve with a user who is not on the approver list
	w2 := ts.doAs("POST", "/api/approvals/"+req.ID+"/approve", `{"comment": "Approved"}`, "wrong-user")
	if w2.Code != 400 {
		t.Fatalf("expected 400 for unauthorized approver, got %d", w2.Code)
	}
}

// ─── Enterprise Task Tests ─────────────────────────────────────────

func TestTaskCreation(t *testing.T) {
	ts := setupTestRouter()
	body := `{
		"title": "Test Task",
		"description": "Test task description",
		"priority": "high",
		"assignee": "task-user-1",
		"project_id": "PRJ-TASK-1"
	}`
	w := ts.do("POST", "/api/tasks", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var task EnterpriseTask
	if err := json.Unmarshal(w.Body.Bytes(), &task); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if task.ID == "" {
		t.Fatal("expected task ID")
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending status, got %s", task.Status)
	}
}

func TestTaskLifecycle(t *testing.T) {
	ts := setupTestRouter()
	// Create task
	body := `{
		"title": "Lifecycle Task",
		"description": "Test task lifecycle",
		"priority": "medium",
		"assignee": "task-user-2"
	}`
	w := ts.do("POST", "/api/tasks", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var task EnterpriseTask
	_ = json.Unmarshal(w.Body.Bytes(), &task)

	// Start task
	w2 := ts.do("POST", "/api/tasks/"+task.ID+"/start", "")
	if w2.Code != 200 {
		t.Fatalf("expected 200 for start, got %d", w2.Code)
	}
	var started EnterpriseTask
	_ = json.Unmarshal(w2.Body.Bytes(), &started)
	if started.Status != "in_progress" {
		t.Fatalf("expected in_progress status, got %s", started.Status)
	}

	// Complete task
	w3 := ts.do("POST", "/api/tasks/"+task.ID+"/complete", "")
	if w3.Code != 200 {
		t.Fatalf("expected 200 for complete, got %d", w3.Code)
	}
	var completed EnterpriseTask
	_ = json.Unmarshal(w3.Body.Bytes(), &completed)
	if completed.Status != "completed" {
		t.Fatalf("expected completed status, got %s", completed.Status)
	}
}

func TestTaskDependencies(t *testing.T) {
	ts := setupTestRouter()
	// Create task A
	bodyA := `{"title": "Task A", "description": "Dependency task", "priority": "high", "assignee": "dep-user"}`
	wA := ts.do("POST", "/api/tasks", bodyA)
	if wA.Code != 201 {
		t.Fatalf("expected 201, got %d", wA.Code)
	}
	var taskA EnterpriseTask
	_ = json.Unmarshal(wA.Body.Bytes(), &taskA)

	// Create task B with dependency on A
	bodyB := `{"title": "Task B", "description": "Dependent task", "priority": "high", "assignee": "dep-user"}`
	wB := ts.do("POST", "/api/tasks", bodyB)
	if wB.Code != 201 {
		t.Fatalf("expected 201, got %d", wB.Code)
	}
	var taskB EnterpriseTask
	_ = json.Unmarshal(wB.Body.Bytes(), &taskB)

	// Add dependency: B blocked by A
	depBody := `{"dependencies": [{"task_id": "` + taskA.ID + `", "type": "blocked_by", "required": true}]}`
	wD := ts.do("PUT", "/api/tasks/"+taskB.ID+"/dependencies", depBody)
	if wD.Code != 200 {
		t.Fatalf("expected 200, got %d", wD.Code)
	}

	// Try to start B - should be blocked
	wS := ts.do("POST", "/api/tasks/"+taskB.ID+"/start", "")
	if wS.Code != 409 {
		t.Fatalf("expected 409 for blocked task, got %d", wS.Code)
	}

	// Complete A
	wC := ts.do("POST", "/api/tasks/"+taskA.ID+"/complete", "")
	if wC.Code != 200 {
		t.Fatalf("expected 200, got %d", wC.Code)
	}

	// Now B should be startable
	wS2 := ts.do("POST", "/api/tasks/"+taskB.ID+"/start", "")
	if wS2.Code != 200 {
		t.Fatalf("expected 200 for start after dependency completed, got %d", wS2.Code)
	}
}

// ─── Escalation & SLA Tests ────────────────────────────────────────

func TestEscalationPoliciesBootstrapped(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/escalations/policies", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count    int                `json:"count"`
		Policies []EscalationPolicy `json:"policies"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count < 2 {
		t.Fatalf("expected at least 2 escalation policies, got %d", resp.Count)
	}
}

func TestSLAsBootstrapped(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/slas", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count int         `json:"count"`
		SLAs  []SLAConfig `json:"slas"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Count < 4 {
		t.Fatalf("expected at least 4 SLA configs, got %d", resp.Count)
	}
}

// ─── Decision Records Tests ────────────────────────────────────────

func TestDecisionRecordCreation(t *testing.T) {
	ts := setupTestRouter()
	body := `{
		"decision": "Approve project expansion",
		"decision_maker": "manager-1",
		"context": "Project is performing well",
		"related_project": "PRJ-DEC-1",
		"action_required": "Notify stakeholders",
		"responsible_person": "manager-1"
	}`
	w := ts.do("POST", "/api/decisions", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var decision DecisionRecord
	if err := json.Unmarshal(w.Body.Bytes(), &decision); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if decision.ID == "" {
		t.Fatal("expected decision ID")
	}
	if decision.Status != "open" {
		t.Fatalf("expected open status, got %s", decision.Status)
	}
}

// ─── AI Recommendations Tests ──────────────────────────────────────

func TestAIRecommendationCreation(t *testing.T) {
	rec := AIRecommendation{
		Type:            "action",
		Title:           "Recommend intervention",
		Description:     "Facility reporting has dropped",
		Rationale:       "37% fewer submissions than previous period",
		Evidence:        []string{"Reporting rate: 61%", "2 unresolved connectivity tickets"},
		SuggestedAction: "create_task",
		Confidence:      0.85,
		Status:          "pending",
	}
	createAIRecommendation(rec)

	recs := listAIRecommendations("pending")
	if len(recs) == 0 {
		t.Fatal("expected at least one AI recommendation")
	}
	found := false
	for _, r := range recs {
		if r.Title == "Recommend intervention" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected recommendation to be found")
	}
}

func TestAIRecommendationDistinguishableFromDecision(t *testing.T) {
	// AI recommendations must remain distinguishable from confirmed decisions
	rec := AIRecommendation{
		Type:            "action",
		Title:           "Recommend escalation",
		Description:     "Project is behind schedule",
		Rationale:       "Progress below 50%",
		SuggestedAction: "escalate",
		Confidence:      0.7,
		Status:          "pending",
	}
	createAIRecommendation(rec)

	// Verify the recommendation has a "recommendation" status, not a decision status
	recs := listAIRecommendations("pending")
	for _, r := range recs {
		if r.Title == "Recommend escalation" {
			if r.Status != "pending" {
				t.Fatalf("expected pending status for AI recommendation, got %s", r.Status)
			}
			// Verify it's not in the decision records
			decisions := listDecisionRecords("", "", 100)
			for _, d := range decisions {
				if d.Decision == r.Title {
					t.Fatal("AI recommendation should not appear as a confirmed decision")
				}
			}
		}
	}
}

// ─── Action Centre Tests ───────────────────────────────────────────

func TestActionCentre(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/action-centre?user_id=test-user", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["approvals"]; !ok {
		t.Fatal("expected approvals in action centre")
	}
	if _, ok := resp["tasks"]; !ok {
		t.Fatal("expected tasks in action centre")
	}
	if _, ok := resp["summary"]; !ok {
		t.Fatal("expected summary in action centre")
	}
}

func TestMyWork(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/my-work?user_id=test-user", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["tasks"]; !ok {
		t.Fatal("expected tasks in my work")
	}
	if _, ok := resp["projects"]; !ok {
		t.Fatal("expected projects in my work")
	}
}

// ─── Workflow Dashboard Tests ──────────────────────────────────────

func TestWorkflowDashboard(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/workflows/dashboard", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["active_workflows"]; !ok {
		t.Fatal("expected active_workflows in dashboard")
	}
	if _, ok := resp["bottlenecks"]; !ok {
		t.Fatal("expected bottlenecks in dashboard")
	}
}

func TestProcessAnalytics(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/workflows/process-analytics", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if _, ok := resp["processes"]; !ok {
		t.Fatal("expected processes in process analytics")
	}
}

// ─── Workflow Audit Trail Tests ────────────────────────────────────

func TestWorkflowExecutionsAudit(t *testing.T) {
	ts := setupTestRouter()
	w := ts.do("GET", "/api/workflows/executions", "")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Count      int                 `json:"count"`
		Executions []WorkflowExecution `json:"executions"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
}

// ─── Idempotency & Retry Tests ─────────────────────────────────────

func TestWorkflowRetry(t *testing.T) {
	ts := setupTestRouter()
	// Start a workflow
	body := `{
		"entity_type": "project",
		"entity_id": "PRJ-RETRY-1",
		"project_id": "PRJ-RETRY-1",
		"actor": "test-user"
	}`
	w := ts.do("POST", "/api/workflows/wf_project_onboarding/run", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp struct {
		InstanceID string `json:"instance_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// Retry the workflow
	w2 := ts.do("POST", "/api/workflows/instances/"+resp.InstanceID+"/retry", "")
	if w2.Code != 200 {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}

// ─── Cross-Application Action Tests ────────────────────────────────

func TestWorkflowActionDispatch(t *testing.T) {
	// Verify that workflow actions map to the correct handlers.
	// Actions that call external services (create_ticket, create_chat, create_calendar)
	// may return connection errors in test environments; we only assert that
	// internal actions and the dispatcher itself do not panic.
	internalActions := map[string]map[string]interface{}{
		"create_task":     {"title": "Test Task", "assignee": "test-user"},
		"notify":          {"title": "Test", "body": "Test", "priority": "medium"},
		"create_decision": {"decision": "Test decision", "decision_maker": "test-user"},
		"create_approval": {"title": "Test", "approvers": []interface{}{"a1"}},
	}
	for action, params := range internalActions {
		step := WorkflowStep{ID: "s1", Name: "Test", Type: "action", Action: action, Params: params}
		inst := WorkflowInstance{ID: "test-inst", WorkflowID: "test-wf", Assignee: "test-user"}
		def := WorkflowDefinition{ID: "test-wf", Name: "Test Workflow"}
		err := performWorkflowAction(inst, def, step)
		if err != nil {
			t.Fatalf("action %s should not error in dispatch: %v", action, err)
		}
	}

	// External-service actions should be recognized by the dispatcher even if
	// the downstream service is unavailable in the test environment.
	externalActions := []string{"create_ticket", "create_chat", "create_calendar"}
	for _, action := range externalActions {
		step := WorkflowStep{ID: "s1", Name: "Test", Type: "action", Action: action}
		inst := WorkflowInstance{ID: "test-inst", WorkflowID: "test-wf", Assignee: "test-user"}
		def := WorkflowDefinition{ID: "test-wf", Name: "Test Workflow"}
		_ = performWorkflowAction(inst, def, step)
	}
}

func TestOpenStatChatForWorkflowUsesObjectConversationContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/conversations/object" {
			t.Fatalf("unexpected StatChat request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Internal-API-Key") != "internal-key" || r.Header.Get("X-StatGate-User-ID") != "enterprise-workflow" || r.Header.Get("X-Tenant-ID") != "tenant-1" {
			t.Fatalf("missing service identity headers")
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("invalid payload: %v", err)
		}
		if payload["objectRef"] != "obj:enterprise:workflow:instance-1" {
			t.Fatalf("unexpected object reference: %v", payload["objectRef"])
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "conversation-1"})
	}))
	defer server.Close()
	t.Setenv("STATCHAT_API_URL", server.URL)
	t.Setenv("STATGATE_INTERNAL_API_KEY", "internal-key")

	err := openStatChatForWorkflow(
		WorkflowInstance{ID: "instance-1", OrgID: "tenant-1", Assignee: "user-1"},
		WorkflowDefinition{ID: "workflow-1", Name: "Approval"},
		WorkflowStep{Name: "Coordinate", Params: map[string]interface{}{}},
	)
	if err != nil {
		t.Fatalf("expected object conversation creation to succeed: %v", err)
	}
}

// ─── Closed-Loop Workflow Test ─────────────────────────────────────

func TestClosedLoopWorkflow(t *testing.T) {
	// Verify the submission decline intervention workflow exists with
	// the complete closed-loop structure: detect → understand → decide →
	// assign → act → monitor → verify → close
	def, ok := getWorkflowDefinition("wf_submission_decline_intervention")
	if !ok {
		t.Fatal("expected submission decline intervention workflow")
	}
	if def.Status != "active" {
		t.Fatalf("expected active status, got %s", def.Status)
	}
	if len(def.Steps) < 7 {
		t.Fatalf("expected at least 7 steps for closed-loop workflow, got %d", len(def.Steps))
	}
	// Verify completion criteria
	if def.Completion == nil {
		t.Fatal("expected completion criteria for closed-loop workflow")
	}
	if def.Completion.Type != "metric_improved" {
		t.Fatalf("expected metric_improved completion, got %s", def.Completion.Type)
	}
}
