package api

import (
	"encoding/json"
	"net/http"

	"aiengines/pkg/model"
	"aiengines/pkg/store"
)

// ─── Digital Twins (P22) ─────────────────────────────────────────────────────

func ListTwinsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListTwins(r.Context(), actorTenant(r), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateTwinHandler(w http.ResponseWriter, r *http.Request) {
	var t model.DigitalTwin
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if t.Name == "" || t.EntityType == "" {
		writeError(w, http.StatusBadRequest, "name and entity_type are required")
		return
	}
	t.TenantID = actorTenant(r)
	t.CreatedBy = actorID(r)
	if err := store.CreateTwin(r.Context(), &t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "twin.created", "digital_twin", t.ID, map[string]interface{}{"name": t.Name, "entity_type": t.EntityType})
	writeJSON(w, http.StatusCreated, t)
}

func GetTwinHandler(w http.ResponseWriter, r *http.Request) {
	t, err := store.GetTwin(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "twin not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func UpdateTwinHandler(w http.ResponseWriter, r *http.Request) {
	t, err := store.GetTwin(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "twin not found")
		return
	}
	var body model.DigitalTwin
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	t.Name, t.Description = body.Name, body.Description
	t.EntityType, t.EntityID = body.EntityType, body.EntityID
	t.Parameters, t.State, t.Status = body.Parameters, body.State, body.Status
	if err := store.UpdateTwin(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "twin.updated", "digital_twin", t.ID, map[string]interface{}{"status": t.Status})
	writeJSON(w, http.StatusOK, t)
}

func DeleteTwinHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteTwin(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "twin.deleted", "digital_twin", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// SimulateTwinHandler runs a simulation synchronously using a deterministic
// scenario engine, marks it completed and emits simulation.completed.
func SimulateTwinHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	twin, err := store.GetTwin(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "twin not found")
		return
	}
	var body struct {
		Name     string                 `json:"name"`
		Scenario map[string]interface{} `json:"scenario"`
		Steps    int                    `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.Name == "" {
		body.Name = twin.Name + " simulation"
	}
	steps := body.Steps
	if steps <= 0 {
		steps = 10
	}
	sim := model.Simulation{
		TenantID:  actorTenant(r),
		TwinID:    twin.ID,
		Name:      body.Name,
		Scenario:  body.Scenario,
		Status:    "running",
		CreatedBy: actorID(r),
	}
	if err := store.CreateSimulation(r.Context(), &sim); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Deterministic step integrator over the twin's parameters.
	start := 0.0
	if v, ok := twin.State["value"].(float64); ok {
		start = v
	}
	step := 1.0
	if v, ok := twin.Parameters["growth_rate"].(float64); ok && v != 0 {
		step = v
	}
	series := make([]float64, 0, steps)
	val := start
	for i := 0; i < steps; i++ {
		if v, ok := body.Scenario["shock"].(float64); ok {
			val = val*(1+step) + v
		} else {
			val = val*(1+step) + 1
		}
		series = append(series, round2(val))
	}
	results := map[string]interface{}{"series": series, "steps": steps, "final": round2(val)}
	if err := store.CompleteSimulation(r.Context(), sim.ID, "completed", results, ""); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sim.Results = results
	sim.Status = "completed"
	emitEvent(r.Context(), "simulation.completed", "simulation", sim.ID, map[string]interface{}{"twin_id": twin.ID, "steps": steps})
	recordAudit(r, "simulation.completed", "simulation", sim.ID, map[string]interface{}{"twin_id": twin.ID})
	writeJSON(w, http.StatusCreated, sim)
}

func ListTwinSimulationsHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetTwin(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "twin not found")
		return
	}
	items, err := store.ListSimulations(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// ─── AI Models (P22) ─────────────────────────────────────────────────────────

func ListModelsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListModels(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateModelHandler(w http.ResponseWriter, r *http.Request) {
	var m model.AIModel
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if m.Name == "" || m.Kind == "" {
		writeError(w, http.StatusBadRequest, "name and kind are required")
		return
	}
	m.TenantID = actorTenant(r)
	m.CreatedBy = actorID(r)
	if err := store.CreateModel(r.Context(), &m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "model.registered", "ai_model", m.ID, map[string]interface{}{"name": m.Name, "kind": m.Kind})
	writeJSON(w, http.StatusCreated, m)
}

func GetModelHandler(w http.ResponseWriter, r *http.Request) {
	m, err := store.GetModel(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// PredictHandler runs the Predictive Analytics Runtime: registers a prediction,
// computes a deterministic result, completes it, emits prediction.generated and
// records a governance guardrail decision.
func DeleteModelHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteModel(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "model.retired", "ai_model", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func PredictHandler(w http.ResponseWriter, r *http.Request) {
	modelID := varsOf(r)["id"]
	if _, err := store.GetModel(r.Context(), modelID); err != nil {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}
	var body model.Prediction
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.TargetType == "" || body.TargetID == "" {
		writeError(w, http.StatusBadRequest, "target_type and target_id are required")
		return
	}
	body.TenantID = actorTenant(r)
	body.ModelID = modelID
	body.TriggeredBy = "manual"
	body.CreatedBy = actorID(r)
	if err := store.CreatePrediction(r.Context(), &body); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Deterministic predictive runtime (seeded by supplied field value).
	val := 0.5
	if v, ok := body.Fields["indicator"]; ok {
		if f, ok := v.(float64); ok {
			val = (0.5 + f/100) / 2
		}
	}
	if val < 0 {
		val = 0
	}
	if val > 1 {
		val = 1
	}
	result := map[string]interface{}{
		"point": round2(val),
		"band":  map[string]interface{}{"low": round2(val - 0.1), "high": round2(val + 0.1)},
	}
	if err := store.CompletePrediction(r.Context(), body.ID, "completed", result); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	body.Prediction = result
	body.Status = "completed"

	risk := 0.0
	if body.Confidence < 0.5 {
		risk = 0.8
	}
	verdict := "auto_approved"
	if risk >= 0.7 {
		verdict = "human_review"
	}
	gov := model.GovernanceAction{
		TenantID:  body.TenantID,
		Action:    "prediction_release",
		Decision:  map[string]interface{}{"prediction_id": body.ID, "model_id": modelID, "target": body.TargetID},
		RiskScore: risk,
		Verdict:   verdict,
	}
	_ = store.CreateGovernanceAction(r.Context(), &gov)

	emitEvent(r.Context(), "prediction.generated", "prediction", body.ID, map[string]interface{}{
		"model_id": modelID, "target_type": body.TargetType, "target_id": body.TargetID, "confidence": body.Confidence})
	recordAudit(r, "prediction.generated", "prediction", body.ID, map[string]interface{}{"model_id": modelID})
	writeJSON(w, http.StatusCreated, body)
}

func ListPredictionsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListPredictions(r.Context(), actorTenant(r), r.URL.Query().Get("model_id"), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func GetPredictionHandler(w http.ResponseWriter, r *http.Request) {
	p, err := store.GetPrediction(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "prediction not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// ─── Decision Pipelines (P22) ────────────────────────────────────────────────

func ListPipelinesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListPipelines(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreatePipelineHandler(w http.ResponseWriter, r *http.Request) {
	var p model.DecisionPipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if p.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p.TenantID = actorTenant(r)
	p.CreatedBy = actorID(r)
	if err := store.CreatePipeline(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "pipeline.created", "pipeline", p.ID, map[string]interface{}{"name": p.Name})
	writeJSON(w, http.StatusCreated, p)
}

// SummaryHandler returns a compact overview of App 5 for the Command Centre.
func SummaryHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service":   SourceApplication,
		"tenant_id": actorTenant(r),
		"endpoints": []string{"/twins", "/models", "/predictions", "/pipelines", "/agents", "/governance", "/graph/nodes", "/graph/statements", "/index/search"},
	})
}