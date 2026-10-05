package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProcessingRun records a durable, scoped GSBPM process operation.
type ProcessingRun struct {
	ID             string                 `json:"id"`
	FormID         string                 `json:"form_id"`
	SurveyID       string                 `json:"survey_id,omitempty"`
	Method         string                 `json:"method"`
	WeightVariable string                 `json:"weight_variable,omitempty"`
	Configuration  map[string]interface{} `json:"configuration"`
	InputCount     int                    `json:"input_count"`
	ApprovedCount  int                    `json:"approved_count"`
	OutputCount    int                    `json:"output_count"`
	Status         string                 `json:"status"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	TenantID       string                 `json:"tenant_id"`
	WorkspaceID    string                 `json:"workspace_id"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
}

type ProcessingRunRecord struct {
	RunID                string                 `json:"run_id"`
	SubmissionInstanceID string                 `json:"submission_instance_id"`
	FormID               string                 `json:"form_id"`
	Values               map[string]interface{} `json:"values"`
	Weight               float64                `json:"weight"`
}

type processingWeightRequest struct {
	FormID         string `json:"form_id"`
	SurveyID       string `json:"survey_id,omitempty"`
	WeightVariable string `json:"weight_variable,omitempty"`
	Method         string `json:"method,omitempty"`
}

type processingTransformRequest struct {
	FormID        string                 `json:"form_id"`
	SurveyID      string                 `json:"survey_id,omitempty"`
	Method        string                 `json:"method"`
	Field         string                 `json:"field"`
	Strategy      string                 `json:"strategy,omitempty"`
	Constant      interface{}            `json:"constant,omitempty"`
	Mapping       map[string]interface{} `json:"mapping,omitempty"`
	Required      bool                   `json:"required,omitempty"`
	Min           *float64               `json:"min,omitempty"`
	Max           *float64               `json:"max,omitempty"`
	AllowedValues []string               `json:"allowed_values,omitempty"`
	OnError       string                 `json:"on_error,omitempty"`
}

func persistProcessingRun(run ProcessingRun, records []ProcessingRunRecord, scopes ...OfficialStatisticsScope) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	scope := officialScopeOrDefault(scopes)
	run.TenantID = scope.TenantID
	run.WorkspaceID = scope.WorkspaceID
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	if run.Configuration == nil {
		run.Configuration = map[string]interface{}{}
	}
	configuration, err := json.Marshal(run.Configuration)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // no-op after commit

	_, err = tx.Exec(ctx, `INSERT INTO processing_runs
		(id, form_id, survey_id, method, weight_variable, configuration, input_count,
		 approved_count, output_count, status, error_message, tenant_id, workspace_id,
		 created_by, created_at, completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,COALESCE($15,now()),$16)`,
		run.ID, run.FormID, nullableString(run.SurveyID), run.Method, nullableString(run.WeightVariable),
		configuration, run.InputCount, run.ApprovedCount, run.OutputCount, run.Status,
		nullableString(run.ErrorMessage), scope.TenantID, scope.WorkspaceID, run.CreatedBy,
		run.CreatedAt, run.CompletedAt)
	if err != nil {
		return err
	}
	for _, record := range records {
		values, marshalErr := json.Marshal(record.Values)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.Exec(ctx, `INSERT INTO processing_run_records
			(processing_run_id, submission_instance_id, form_id, values, weight, tenant_id, workspace_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`, run.ID, record.SubmissionInstanceID, record.FormID,
			values, record.Weight, scope.TenantID, scope.WorkspaceID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func ListProcessingRuns(formID, status string, limit int, scopes ...OfficialStatisticsScope) ([]ProcessingRun, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	scope := officialScopeOrDefault(scopes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, form_id, COALESCE(survey_id,''), method, COALESCE(weight_variable,''),
		configuration, input_count, approved_count, output_count, status, COALESCE(error_message,''),
		tenant_id, workspace_id, created_by, created_at, completed_at
		FROM processing_runs WHERE tenant_id=$1 AND workspace_id=$2`
	args := []interface{}{scope.TenantID, scope.WorkspaceID}
	if formID != "" {
		query += " AND form_id=$3"
		args = append(args, formID)
	}
	if status != "" {
		query += fmt.Sprintf(" AND status=$%d", len(args)+1)
		args = append(args, status)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProcessingRun
	for rows.Next() {
		var run ProcessingRun
		var configuration []byte
		if err := rows.Scan(&run.ID, &run.FormID, &run.SurveyID, &run.Method, &run.WeightVariable,
			&configuration, &run.InputCount, &run.ApprovedCount, &run.OutputCount, &run.Status,
			&run.ErrorMessage, &run.TenantID, &run.WorkspaceID, &run.CreatedBy, &run.CreatedAt, &run.CompletedAt); err != nil {
			return nil, err
		}
		if len(configuration) > 0 {
			_ = json.Unmarshal(configuration, &run.Configuration)
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func GetProcessingRun(id string, scopes ...OfficialStatisticsScope) (*ProcessingRun, error) {
	runs, err := ListProcessingRuns("", "", 500, scopes...)
	if err != nil {
		return nil, err
	}
	for i := range runs {
		if runs[i].ID == id {
			return &runs[i], nil
		}
	}
	return nil, fmt.Errorf("processing run not found")
}

func GetProcessingRunRecords(runID string, scopes ...OfficialStatisticsScope) ([]ProcessingRunRecord, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	scope := officialScopeOrDefault(scopes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT submission_instance_id, form_id, values, weight
		FROM processing_run_records WHERE processing_run_id=$1 AND tenant_id=$2 AND workspace_id=$3
		ORDER BY id`, runID, scope.TenantID, scope.WorkspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProcessingRunRecord
	for rows.Next() {
		var record ProcessingRunRecord
		var values []byte
		if err := rows.Scan(&record.SubmissionInstanceID, &record.FormID, &values, &record.Weight); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(values, &record.Values); err != nil {
			return nil, err
		}
		record.RunID = runID
		out = append(out, record)
	}
	return out, rows.Err()
}

func processingWeight(value interface{}) (float64, error) {
	var raw string
	switch typed := value.(type) {
	case float64:
		return validateProcessingWeight(typed)
	case float32:
		return validateProcessingWeight(float64(typed))
	case int:
		return validateProcessingWeight(float64(typed))
	case int64:
		return validateProcessingWeight(float64(typed))
	case json.Number:
		raw = typed.String()
	case string:
		raw = strings.TrimSpace(typed)
	default:
		raw = strings.TrimSpace(fmt.Sprint(value))
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("weight must be numeric")
	}
	return validateProcessingWeight(parsed)
}

func validateProcessingWeight(value float64) (float64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 1e12 {
		return 0, fmt.Errorf("weight must be greater than zero and finite")
	}
	return value, nil
}

func processingFieldNameValid(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func adminProcessingWeightHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req processingWeightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.FormID = strings.TrimSpace(req.FormID)
	req.WeightVariable = strings.TrimSpace(req.WeightVariable)
	if req.WeightVariable == "" {
		req.WeightVariable = "weight"
	}
	if req.Method == "" {
		req.Method = "provided_weight"
	}
	if req.FormID == "" || !processingFieldNameValid(req.WeightVariable) {
		http.Error(w, "form_id and a valid weight_variable are required", http.StatusBadRequest)
		return
	}
	if req.Method != "provided_weight" {
		http.Error(w, "unsupported weighting method", http.StatusBadRequest)
		return
	}

	subs, err := GetSubmissionsByFormID(req.FormID, 5000, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	run := ProcessingRun{
		ID:             uuid.NewString(),
		FormID:         req.FormID,
		SurveyID:       req.SurveyID,
		Method:         req.Method,
		WeightVariable: req.WeightVariable,
		Configuration:  map[string]interface{}{"weight_variable": req.WeightVariable, "method": req.Method},
		InputCount:     len(subs),
		Status:         "completed",
		CreatedBy:      strings.TrimSpace(r.Header.Get("X-Actor-ID")),
		CreatedAt:      time.Now().UTC(),
		TenantID:       scope.TenantID,
		WorkspaceID:    scope.WorkspaceID,
	}
	if run.CreatedBy == "" {
		run.CreatedBy = "admin"
	}

	var records []ProcessingRunRecord
	for _, sub := range subs {
		if sub.Status != "approved" {
			continue
		}
		run.ApprovedCount++
		weight, weightErr := processingWeight(sub.Meta[req.WeightVariable])
		if weightErr != nil {
			run.Status = "failed"
			run.ErrorMessage = fmt.Sprintf("submission %s: %v", sub.InstanceID, weightErr)
			break
		}
		values := make(map[string]interface{}, len(sub.Meta)+1)
		for key, value := range sub.Meta {
			values[key] = value
		}
		values[req.WeightVariable] = weight
		records = append(records, ProcessingRunRecord{
			RunID:                run.ID,
			SubmissionInstanceID: sub.InstanceID,
			FormID:               sub.FormID,
			Values:               values,
			Weight:               weight,
		})
	}
	run.OutputCount = len(records)
	if run.ApprovedCount == 0 {
		run.Status = "failed"
		run.ErrorMessage = "no approved submissions found for form_id"
	}
	if run.Status == "failed" {
		records = nil
		run.OutputCount = 0
	}
	if err := persistProcessingRun(run, records, scope); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if run.Status == "failed" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(run)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(run)
}

func cloneProcessingValues(values map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func processingValueMissing(value interface{}) bool {
	if value == nil {
		return true
	}
	return strings.TrimSpace(fmt.Sprint(value)) == ""
}

func processingValueKey(value interface{}) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func processingTransformRun(req processingTransformRequest, subs []SubmissionSummary, scope OfficialStatisticsScope, actor string) ProcessingRun {
	return ProcessingRun{
		ID:            uuid.NewString(),
		FormID:        req.FormID,
		SurveyID:      req.SurveyID,
		Method:        req.Method,
		Configuration: map[string]interface{}{"method": req.Method, "field": req.Field, "strategy": req.Strategy, "required": req.Required, "min": req.Min, "max": req.Max, "allowed_values": req.AllowedValues, "on_error": req.OnError},
		InputCount:    len(subs),
		TenantID:      scope.TenantID,
		WorkspaceID:   scope.WorkspaceID,
		CreatedBy:     actor,
		CreatedAt:     time.Now().UTC(),
		Status:        "completed",
	}
}

func adminProcessingTransformHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req processingTransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.FormID = strings.TrimSpace(req.FormID)
	req.Method = strings.ToLower(strings.TrimSpace(req.Method))
	req.Field = strings.TrimSpace(req.Field)
	req.Strategy = strings.ToLower(strings.TrimSpace(req.Strategy))
	req.OnError = strings.ToLower(strings.TrimSpace(req.OnError))
	if req.FormID == "" || !processingFieldNameValid(req.Field) {
		http.Error(w, "form_id and a valid field are required", http.StatusBadRequest)
		return
	}
	if req.OnError == "" {
		req.OnError = "fail"
	}
	if req.OnError != "fail" && req.OnError != "drop" {
		http.Error(w, "on_error must be fail or drop", http.StatusBadRequest)
		return
	}
	validMethods := map[string]bool{"impute": true, "code": true, "edit": true}
	if !validMethods[req.Method] {
		http.Error(w, "method must be impute, code, or edit", http.StatusBadRequest)
		return
	}
	if req.Method == "impute" && req.Strategy == "" {
		req.Strategy = "constant"
	}
	if req.Method == "impute" && req.Strategy != "constant" && req.Strategy != "mean" && req.Strategy != "mode" {
		http.Error(w, "impute strategy must be constant, mean, or mode", http.StatusBadRequest)
		return
	}
	if req.Method == "code" && len(req.Mapping) == 0 {
		http.Error(w, "mapping is required for coding", http.StatusBadRequest)
		return
	}

	subs, err := GetSubmissionsByFormID(req.FormID, 5000, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	if actor == "" {
		actor = "admin"
	}
	run := processingTransformRun(req, subs, scope, actor)
	var approved []SubmissionSummary
	for _, sub := range subs {
		if sub.Status == "approved" {
			run.ApprovedCount++
			approved = append(approved, sub)
		}
	}
	if len(approved) == 0 {
		run.Status = "failed"
		run.ErrorMessage = "no approved submissions found for form_id"
	} else {
		var records []ProcessingRunRecord
		var imputeValue interface{}
		if req.Method == "impute" && req.Strategy == "mean" {
			var total float64
			var count int
			for _, sub := range approved {
				if processingValueMissing(sub.Meta[req.Field]) {
					continue
				}
				value, valueErr := processingWeight(sub.Meta[req.Field])
				if valueErr == nil {
					total += value
					count++
				}
			}
			if count == 0 {
				run.Status = "failed"
				run.ErrorMessage = "mean imputation requires at least one numeric observed value"
			} else {
				imputeValue = total / float64(count)
			}
		} else if req.Method == "impute" && req.Strategy == "mode" {
			counts := map[string]int{}
			values := map[string]interface{}{}
			for _, sub := range approved {
				if processingValueMissing(sub.Meta[req.Field]) {
					continue
				}
				key := processingValueKey(sub.Meta[req.Field])
				counts[key]++
				values[key] = sub.Meta[req.Field]
			}
			bestKey := ""
			for key, count := range counts {
				if bestKey == "" || count > counts[bestKey] || (count == counts[bestKey] && key < bestKey) {
					bestKey = key
				}
			}
			if bestKey == "" {
				run.Status = "failed"
				run.ErrorMessage = "mode imputation requires at least one observed value"
			} else {
				imputeValue = values[bestKey]
			}
		} else if req.Method == "impute" && req.Strategy == "constant" && processingValueMissing(req.Constant) {
			run.Status = "failed"
			run.ErrorMessage = "constant imputation requires constant"
		}

		if run.Status == "completed" {
			for _, sub := range approved {
				values := cloneProcessingValues(sub.Meta)
				valid := true
				if req.Method == "impute" && processingValueMissing(values[req.Field]) {
					values[req.Field] = imputeValue
				}
				if req.Method == "code" {
					if replacement, found := req.Mapping[processingValueKey(values[req.Field])]; found {
						values[req.Field] = replacement
					}
				}
				if req.Method == "edit" {
					if req.Required && processingValueMissing(values[req.Field]) {
						valid = false
					}
					if len(req.AllowedValues) > 0 && !processingAllowedValue(values[req.Field], req.AllowedValues) {
						valid = false
					}
					if req.Min != nil || req.Max != nil {
						numeric, numericErr := processingWeight(values[req.Field])
						if numericErr != nil || (req.Min != nil && numeric < *req.Min) || (req.Max != nil && numeric > *req.Max) {
							valid = false
						}
					}
				}
				if !valid {
					if req.OnError == "drop" {
						continue
					}
					run.Status = "failed"
					run.ErrorMessage = fmt.Sprintf("structured edit failed for submission %s", sub.InstanceID)
					break
				}
				records = append(records, ProcessingRunRecord{RunID: run.ID, SubmissionInstanceID: sub.InstanceID, FormID: sub.FormID, Values: values, Weight: 1})
			}
		}
		if run.Status == "completed" {
			run.OutputCount = len(records)
			if err := persistProcessingRun(run, records, scope); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	if run.Status == "failed" {
		if err := persistProcessingRun(run, nil, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(run)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(run)
}

func processingAllowedValue(value interface{}, allowed []string) bool {
	key := processingValueKey(value)
	for _, candidate := range allowed {
		if key == strings.TrimSpace(candidate) {
			return true
		}
	}
	return false
}

func adminProcessingRunsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	runs, err := ListProcessingRuns(r.URL.Query().Get("form_id"), r.URL.Query().Get("status"), limit, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"count": len(runs), "runs": runs})
}
