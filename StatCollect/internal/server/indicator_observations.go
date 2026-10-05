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

type IndicatorObservation struct {
	ID              string                 `json:"id"`
	IndicatorCode   string                 `json:"indicator_code"`
	Period          string                 `json:"period"`
	GeographyCode   string                 `json:"geography_code"`
	Value           float64                `json:"value"`
	Unit            string                 `json:"unit"`
	Version         int                    `json:"version"`
	IsCurrent       bool                   `json:"is_current"`
	Status          string                 `json:"status"`
	SourceFormID    string                 `json:"source_form_id"`
	ProcessingRunID string                 `json:"processing_run_id"`
	SourceCount     int                    `json:"source_count"`
	Methodology     map[string]interface{} `json:"methodology"`
	PublishedBy     string                 `json:"published_by"`
	TenantID        string                 `json:"tenant_id"`
	WorkspaceID     string                 `json:"workspace_id"`
	PublishedAt     time.Time              `json:"published_at"`
	CreatedAt       time.Time              `json:"created_at"`
}

type PublishIndicatorObservationRequest struct {
	IndicatorCode   string                 `json:"indicator_code"`
	Period          string                 `json:"period"`
	GeographyCode   string                 `json:"geography_code,omitempty"`
	Value           *float64               `json:"value"`
	Unit            string                 `json:"unit,omitempty"`
	SourceFormID    string                 `json:"source_form_id,omitempty"`
	ProcessingRunID string                 `json:"processing_run_id"`
	Methodology     map[string]interface{} `json:"methodology,omitempty"`
}

type CalculateIndicatorObservationRequest struct {
	IndicatorCode   string   `json:"indicator_code"`
	Period          string   `json:"period"`
	GeographyCode   string   `json:"geography_code,omitempty"`
	ProcessingRunID string   `json:"processing_run_id"`
	MeasureVariable string   `json:"measure_variable,omitempty"`
	Aggregation     string   `json:"aggregation"`
	ConditionField  string   `json:"condition_field,omitempty"`
	ConditionValue  string   `json:"condition_value,omitempty"`
	Unit            string   `json:"unit,omitempty"`
	ExpectedMin     *float64 `json:"expected_min,omitempty"`
	ExpectedMax     *float64 `json:"expected_max,omitempty"`
}

type IndicatorCalculation struct {
	Aggregation     string  `json:"aggregation"`
	MeasureVariable string  `json:"measure_variable,omitempty"`
	ConditionField  string  `json:"condition_field,omitempty"`
	ConditionValue  string  `json:"condition_value,omitempty"`
	RecordCount     int     `json:"record_count"`
	NumericCount    int     `json:"numeric_count"`
	MatchedCount    int     `json:"matched_count,omitempty"`
	Value           float64 `json:"value"`
	Validation      string  `json:"validation"`
}

type IndicatorQualityCheck struct {
	Status   string   `json:"status"`
	Checks   []string `json:"checks"`
	Warnings []string `json:"warnings,omitempty"`
}

type ValidateIndicatorObservationRequest struct {
	ObservationID string `json:"observation_id"`
}

type AnalyticsHandoff struct {
	ID            int64     `json:"id"`
	ObservationID string    `json:"observation_id"`
	TargetModule  string    `json:"target_module"`
	TargetAssetID string    `json:"target_asset_id"`
	Status        string    `json:"status"`
	HTTPStatus    int       `json:"http_status"`
	Attempts      int       `json:"attempts"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	TenantID      string    `json:"tenant_id"`
	WorkspaceID   string    `json:"workspace_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func validObservationPeriod(period string) bool {
	period = strings.TrimSpace(period)
	if len(period) != 4 && len(period) != 7 {
		return false
	}
	if _, err := strconv.Atoi(period[:4]); err != nil {
		return false
	}
	if len(period) == 4 {
		return true
	}
	if period[4] == '-' {
		if period[5] == 'Q' && period[6] >= '1' && period[6] <= '4' {
			return true
		}
		if period[5] >= '0' && period[5] <= '1' && period[6] >= '0' && period[6] <= '9' {
			month, _ := strconv.Atoi(period[5:])
			return month >= 1 && month <= 12
		}
	}
	return false
}

func processingNumericValue(value interface{}) (float64, error) {
	var raw string
	switch typed := value.(type) {
	case float64:
		return typed, validateProcessingNumericNumeric(typed)
	case float32:
		parsed := float64(typed)
		return parsed, validateProcessingNumericNumeric(parsed)
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case json.Number:
		raw = typed.String()
	case string:
		raw = strings.TrimSpace(typed)
	default:
		raw = strings.TrimSpace(fmt.Sprint(value))
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("value must be numeric")
	}
	return parsed, validateProcessingNumericNumeric(parsed)
}

func validateProcessingNumericNumeric(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < -1e12 || value > 1e12 {
		return fmt.Errorf("value must be finite and within the supported range")
	}
	return nil
}

func validateIndicatorObservationQuality(indicator SDGIndicator, value float64, unit string, sourceCount int) (*IndicatorQualityCheck, error) {
	quality := &IndicatorQualityCheck{
		Status: "passed",
		Checks: []string{"registered_indicator", "unit_match", "finite_value", "source_records"},
	}
	if indicator.Unit != "" && unit != indicator.Unit {
		return nil, fmt.Errorf("unit %s does not match registered indicator unit %s", unit, indicator.Unit)
	}
	if err := validateProcessingNumericNumeric(value); err != nil {
		return nil, err
	}
	if unit == "%" && (value < 0 || value > 100) {
		return nil, fmt.Errorf("percentage observation must be between 0 and 100")
	}
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "count", "number", "persons", "households", "per 1k", "per 100k":
		if value < 0 {
			return nil, fmt.Errorf("%s observation cannot be negative", unit)
		}
	}
	if sourceCount <= 0 {
		return nil, fmt.Errorf("observation must have at least one source record")
	}
	if sourceCount < 5 {
		quality.Status = "passed_with_warnings"
		quality.Warnings = append(quality.Warnings, "small_source_count")
	}
	return quality, nil
}

func CalculateAndPublishIndicatorObservation(req CalculateIndicatorObservationRequest, actor string, scopes ...OfficialStatisticsScope) (*IndicatorObservation, *IndicatorCalculation, error) {
	scope := officialScopeOrDefault(scopes)
	req.IndicatorCode = strings.TrimSpace(req.IndicatorCode)
	req.Period = strings.TrimSpace(req.Period)
	req.GeographyCode = strings.TrimSpace(req.GeographyCode)
	req.ProcessingRunID = strings.TrimSpace(req.ProcessingRunID)
	req.MeasureVariable = strings.TrimSpace(req.MeasureVariable)
	req.Aggregation = strings.ToUpper(strings.TrimSpace(req.Aggregation))
	req.ConditionField = strings.TrimSpace(req.ConditionField)
	req.ConditionValue = strings.TrimSpace(req.ConditionValue)
	req.Unit = strings.TrimSpace(req.Unit)
	if req.GeographyCode == "" {
		req.GeographyCode = "UGA"
	}
	if !processingFieldNameValid(req.IndicatorCode) || !validObservationPeriod(req.Period) || !processingFieldNameValid(req.GeographyCode) {
		return nil, nil, fmt.Errorf("indicator_code, period, and geography_code are invalid")
	}
	if req.ProcessingRunID == "" {
		return nil, nil, fmt.Errorf("processing_run_id is required")
	}
	validAggregations := map[string]bool{"COUNT": true, "SUM": true, "MEAN": true, "PROPORTION": true, "WEIGHTED_COUNT": true}
	if !validAggregations[req.Aggregation] {
		return nil, nil, fmt.Errorf("aggregation must be COUNT, SUM, MEAN, PROPORTION, or WEIGHTED_COUNT")
	}
	if req.MeasureVariable != "" && !processingFieldNameValid(req.MeasureVariable) {
		return nil, nil, fmt.Errorf("measure_variable is invalid")
	}
	if req.ConditionField != "" && !processingFieldNameValid(req.ConditionField) {
		return nil, nil, fmt.Errorf("condition_field is invalid")
	}
	if req.Aggregation == "SUM" || req.Aggregation == "MEAN" {
		if req.MeasureVariable == "" {
			return nil, nil, fmt.Errorf("measure_variable is required for %s", req.Aggregation)
		}
	}
	if req.Aggregation == "PROPORTION" && (req.ConditionField == "" || req.ConditionValue == "") {
		return nil, nil, fmt.Errorf("condition_field and condition_value are required for PROPORTION")
	}
	if req.ExpectedMin != nil && validateProcessingNumericNumeric(*req.ExpectedMin) != nil {
		return nil, nil, fmt.Errorf("expected_min must be finite and within the supported range")
	}
	if req.ExpectedMax != nil && validateProcessingNumericNumeric(*req.ExpectedMax) != nil {
		return nil, nil, fmt.Errorf("expected_max must be finite and within the supported range")
	}
	if req.ExpectedMin != nil && req.ExpectedMax != nil && *req.ExpectedMin > *req.ExpectedMax {
		return nil, nil, fmt.Errorf("expected_min cannot exceed expected_max")
	}

	run, err := GetProcessingRun(req.ProcessingRunID, scope)
	if err != nil || run.Status != "completed" {
		return nil, nil, fmt.Errorf("a completed processing_run_id in the same tenant/workspace is required")
	}
	records, err := GetProcessingRunRecords(req.ProcessingRunID, scope)
	if err != nil {
		return nil, nil, err
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("processing run has no output records")
	}

	calculation := &IndicatorCalculation{
		Aggregation: req.Aggregation, MeasureVariable: req.MeasureVariable,
		ConditionField: req.ConditionField, ConditionValue: req.ConditionValue,
		RecordCount: len(records), Validation: "passed",
	}
	var total float64
	for _, record := range records {
		switch req.Aggregation {
		case "COUNT":
			calculation.NumericCount = len(records)
		case "WEIGHTED_COUNT":
			if err := validateProcessingNumericNumeric(record.Weight); err != nil {
				return nil, nil, fmt.Errorf("record %s has invalid weight: %v", record.SubmissionInstanceID, err)
			}
			total += record.Weight
		case "SUM", "MEAN":
			value, valueErr := processingNumericValue(record.Values[req.MeasureVariable])
			if valueErr != nil {
				return nil, nil, fmt.Errorf("record %s: %s must be numeric", record.SubmissionInstanceID, req.MeasureVariable)
			}
			calculation.NumericCount++
			total += value
		case "PROPORTION":
			if strings.TrimSpace(fmt.Sprint(record.Values[req.ConditionField])) == req.ConditionValue {
				calculation.MatchedCount++
			}
		}
	}
	if req.Aggregation == "COUNT" {
		total = float64(len(records))
		calculation.NumericCount = len(records)
	} else if req.Aggregation == "MEAN" {
		if calculation.NumericCount == 0 {
			return nil, nil, fmt.Errorf("mean requires at least one numeric value")
		}
		total /= float64(calculation.NumericCount)
	} else if req.Aggregation == "PROPORTION" {
		total = float64(calculation.MatchedCount) / float64(len(records)) * 100
	}
	if err := validateProcessingNumericNumeric(total); err != nil {
		return nil, nil, err
	}
	calculation.Value = total
	if req.ExpectedMin != nil && total < *req.ExpectedMin {
		return nil, nil, fmt.Errorf("calculated value is below expected_min")
	}
	if req.ExpectedMax != nil && total > *req.ExpectedMax {
		return nil, nil, fmt.Errorf("calculated value is above expected_max")
	}

	indicators, err := ListSDGIndicators(0, "", scope)
	if err != nil {
		return nil, nil, err
	}
	var indicator *SDGIndicator
	for i := range indicators {
		if indicators[i].IndicatorCode == req.IndicatorCode {
			indicator = &indicators[i]
			break
		}
	}
	if indicator == nil {
		return nil, nil, fmt.Errorf("indicator %s is not registered in the same tenant/workspace", req.IndicatorCode)
	}
	if req.Unit != "" && indicator.Unit != "" && req.Unit != indicator.Unit {
		return nil, nil, fmt.Errorf("unit %s does not match registered indicator unit %s", req.Unit, indicator.Unit)
	}
	if req.Unit == "" {
		req.Unit = indicator.Unit
	}
	methodology := map[string]interface{}{
		"calculation":       calculation,
		"processing_run_id": req.ProcessingRunID,
		"validation":        "passed",
	}
	if req.ExpectedMin != nil {
		methodology["expected_min"] = *req.ExpectedMin
	}
	if req.ExpectedMax != nil {
		methodology["expected_max"] = *req.ExpectedMax
	}
	observation, err := PublishIndicatorObservation(PublishIndicatorObservationRequest{
		IndicatorCode: req.IndicatorCode, Period: req.Period, GeographyCode: req.GeographyCode,
		Value: &total, Unit: req.Unit, ProcessingRunID: req.ProcessingRunID,
		Methodology: methodology,
	}, actor, scope)
	if err != nil {
		return nil, nil, err
	}
	return observation, calculation, nil
}

func PublishIndicatorObservation(req PublishIndicatorObservationRequest, actor string, scopes ...OfficialStatisticsScope) (*IndicatorObservation, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	scope := officialScopeOrDefault(scopes)
	req.IndicatorCode = strings.TrimSpace(req.IndicatorCode)
	req.Period = strings.TrimSpace(req.Period)
	req.GeographyCode = strings.TrimSpace(req.GeographyCode)
	req.SourceFormID = strings.TrimSpace(req.SourceFormID)
	if req.GeographyCode == "" {
		req.GeographyCode = "UGA"
	}
	if !processingFieldNameValid(req.IndicatorCode) || !validObservationPeriod(req.Period) || !processingFieldNameValid(req.GeographyCode) {
		return nil, fmt.Errorf("indicator_code, period, and geography_code are invalid")
	}
	if req.Value == nil || math.IsNaN(*req.Value) || math.IsInf(*req.Value, 0) || *req.Value < -1e12 || *req.Value > 1e12 {
		return nil, fmt.Errorf("value must be finite and within the supported range")
	}
	run, err := GetProcessingRun(req.ProcessingRunID, scope)
	if err != nil || run.Status != "completed" {
		return nil, fmt.Errorf("a completed processing_run_id in the same tenant/workspace is required")
	}
	if req.SourceFormID != "" && req.SourceFormID != run.FormID {
		return nil, fmt.Errorf("source_form_id does not match the processing run")
	}
	req.SourceFormID = run.FormID
	indicators, err := ListSDGIndicators(0, "", scope)
	if err != nil {
		return nil, err
	}
	var indicator SDGIndicator
	found := false
	for _, candidate := range indicators {
		if candidate.IndicatorCode == req.IndicatorCode {
			indicator = candidate
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("indicator %s is not registered in the same tenant/workspace", req.IndicatorCode)
	}
	if req.Unit == "" {
		req.Unit = indicator.Unit
	}
	if req.Methodology == nil {
		req.Methodology = map[string]interface{}{}
	}
	quality, err := validateIndicatorObservationQuality(indicator, *req.Value, req.Unit, run.OutputCount)
	if err != nil {
		return nil, err
	}
	req.Methodology["quality_checks"] = quality
	if actor == "" {
		actor = "admin"
	}
	methodology, err := json.Marshal(req.Methodology)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var version int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM indicator_observations
		WHERE indicator_code=$1 AND period=$2 AND geography_code=$3 AND tenant_id=$4 AND workspace_id=$5`,
		req.IndicatorCode, req.Period, req.GeographyCode, scope.TenantID, scope.WorkspaceID).Scan(&version); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE indicator_observations SET is_current=false
		WHERE indicator_code=$1 AND period=$2 AND geography_code=$3 AND tenant_id=$4 AND workspace_id=$5`,
		req.IndicatorCode, req.Period, req.GeographyCode, scope.TenantID, scope.WorkspaceID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	observation := &IndicatorObservation{
		ID: uuid.NewString(), IndicatorCode: req.IndicatorCode, Period: req.Period,
		GeographyCode: req.GeographyCode, Value: *req.Value, Unit: req.Unit,
		Version: version, IsCurrent: true, Status: "published", SourceFormID: req.SourceFormID,
		ProcessingRunID: req.ProcessingRunID, SourceCount: run.OutputCount, Methodology: req.Methodology,
		PublishedBy: actor, TenantID: scope.TenantID, WorkspaceID: scope.WorkspaceID,
		PublishedAt: now, CreatedAt: now,
	}
	if _, err := tx.Exec(ctx, `INSERT INTO indicator_observations
		(id, indicator_code, period, geography_code, value, unit, version, is_current, status,
		 source_form_id, processing_run_id, source_count, methodology, published_by,
		 tenant_id, workspace_id, published_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		observation.ID, observation.IndicatorCode, observation.Period, observation.GeographyCode,
		observation.Value, observation.Unit, observation.Version, observation.Status,
		observation.SourceFormID, observation.ProcessingRunID, observation.SourceCount, methodology,
		observation.PublishedBy, scope.TenantID, scope.WorkspaceID, observation.PublishedAt, observation.CreatedAt); err != nil {
		return nil, err
	}
	latestYear, _ := strconv.Atoi(req.Period[:4])
	if _, err := tx.Exec(ctx, `UPDATE sdg_indicators SET latest_value=$1, latest_year=$2,
		data_source=$3, updated_at=now() WHERE indicator_code=$4 AND tenant_id=$5 AND workspace_id=$6`,
		observation.Value, latestYear, "statcollect:processing_run:"+req.ProcessingRunID,
		req.IndicatorCode, scope.TenantID, scope.WorkspaceID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return observation, nil
}

func recordAnalyticsHandoff(handoff AnalyticsHandoff) (*AnalyticsHandoff, error) {
	if dbPool == nil {
		return &handoff, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stored AnalyticsHandoff
	err := dbPool.QueryRow(ctx, `INSERT INTO analytics_handoffs
		(observation_id, target_module, target_asset_id, status, http_status, attempts,
		 error_message, tenant_id, workspace_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now(),now())
		ON CONFLICT (observation_id, target_module, tenant_id, workspace_id) DO UPDATE SET
		 target_asset_id=EXCLUDED.target_asset_id, status=EXCLUDED.status,
		 http_status=EXCLUDED.http_status, attempts=analytics_handoffs.attempts+1,
		 error_message=EXCLUDED.error_message, updated_at=now()
		RETURNING id, observation_id, target_module, target_asset_id, status, http_status,
		 attempts, COALESCE(error_message,''), tenant_id, workspace_id, created_at, updated_at`,
		handoff.ObservationID, handoff.TargetModule, handoff.TargetAssetID, handoff.Status,
		handoff.HTTPStatus, handoff.Attempts, nullableString(handoff.ErrorMessage),
		handoff.TenantID, handoff.WorkspaceID).Scan(
		&stored.ID, &stored.ObservationID, &stored.TargetModule, &stored.TargetAssetID,
		&stored.Status, &stored.HTTPStatus, &stored.Attempts, &stored.ErrorMessage,
		&stored.TenantID, &stored.WorkspaceID, &stored.CreatedAt, &stored.UpdatedAt)
	if err != nil {
		return &handoff, err
	}
	return &stored, nil
}

func handoffIndicatorObservation(observation *IndicatorObservation) *AnalyticsHandoff {
	targetAssetID := "official-observation-" + observation.ID
	handoff := AnalyticsHandoff{
		ObservationID: observation.ID,
		TargetModule:  "analytics_core",
		TargetAssetID: targetAssetID,
		Status:        "skipped",
		TenantID:      observation.TenantID,
		WorkspaceID:   observation.WorkspaceID,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	statisticsURL := ""
	internalKey := ""
	if cfg != nil {
		statisticsURL = strings.TrimRight(strings.TrimSpace(cfg.StatisticsURL), "/")
		internalKey = cfg.InternalKey
	}
	if statisticsURL == "" || statisticsURL == "-" {
		handoff.ErrorMessage = "statistics_url_not_configured"
		stored, err := recordAnalyticsHandoff(handoff)
		if err == nil {
			return stored
		}
		return &handoff
	}
	payload := map[string]interface{}{
		"id":          targetAssetID,
		"asset_type":  "official_indicator_observation",
		"version_tag": fmt.Sprintf("%s-v%d", observation.Period, observation.Version),
		"content_definition": map[string]interface{}{
			"indicator_code":    observation.IndicatorCode,
			"period":            observation.Period,
			"geography_code":    observation.GeographyCode,
			"value":             observation.Value,
			"unit":              observation.Unit,
			"status":            observation.Status,
			"source_form_id":    observation.SourceFormID,
			"processing_run_id": observation.ProcessingRunID,
			"source_count":      observation.SourceCount,
			"methodology":       observation.Methodology,
			"observation_id":    observation.ID,
		},
	}
	body, err := json.Marshal(payload)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, statisticsURL+"/api/v1/assets/save", strings.NewReader(string(body)))
		if requestErr == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-StatGate-Internal-Key", internalKey)
			req.Header.Set("X-Tenant-ID", observation.TenantID)
			req.Header.Set("X-Workspace-ID", observation.WorkspaceID)
			req.Header.Set("X-User-ID", "statcollect")
			req.Header.Set("X-User-Role", "operator")
			if resp, requestErr := http.DefaultClient.Do(req); requestErr == nil {
				handoff.HTTPStatus = resp.StatusCode
				handoff.Attempts = 1
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					handoff.Status = "published"
				} else {
					handoff.Status = "failed"
					handoff.ErrorMessage = "analytics_core_rejected_handoff"
				}
				resp.Body.Close()
			} else {
				handoff.Status = "failed"
				handoff.Attempts = 1
				handoff.ErrorMessage = requestErr.Error()
			}
		} else {
			handoff.Status = "failed"
			handoff.ErrorMessage = requestErr.Error()
		}
	} else {
		handoff.Status = "failed"
		handoff.ErrorMessage = err.Error()
	}
	stored, recordErr := recordAnalyticsHandoff(handoff)
	if recordErr == nil {
		return stored
	}
	return &handoff
}

func ListAnalyticsHandoffs(observationID string, limit int, scopes ...OfficialStatisticsScope) ([]AnalyticsHandoff, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	scope := officialScopeOrDefault(scopes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, observation_id, target_module, target_asset_id, status, http_status,
		 attempts, COALESCE(error_message,''), tenant_id, workspace_id, created_at, updated_at
		 FROM analytics_handoffs WHERE tenant_id=$1 AND workspace_id=$2`
	args := []interface{}{scope.TenantID, scope.WorkspaceID}
	if observationID != "" {
		query += " AND observation_id=$3"
		args = append(args, observationID)
	}
	query += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AnalyticsHandoff
	for rows.Next() {
		var handoff AnalyticsHandoff
		if err := rows.Scan(&handoff.ID, &handoff.ObservationID, &handoff.TargetModule,
			&handoff.TargetAssetID, &handoff.Status, &handoff.HTTPStatus, &handoff.Attempts,
			&handoff.ErrorMessage, &handoff.TenantID, &handoff.WorkspaceID,
			&handoff.CreatedAt, &handoff.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, handoff)
	}
	return out, rows.Err()
}

func ListIndicatorObservations(indicatorCode, period, geographyCode string, history bool, limit int, scopes ...OfficialStatisticsScope) ([]IndicatorObservation, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	scope := officialScopeOrDefault(scopes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, indicator_code, period, geography_code, value, unit, version, is_current,
		status, source_form_id, processing_run_id, source_count, methodology, published_by,
		tenant_id, workspace_id, published_at, created_at FROM indicator_observations
		WHERE tenant_id=$1 AND workspace_id=$2 AND status='published'`
	args := []interface{}{scope.TenantID, scope.WorkspaceID}
	arg := 3
	if indicatorCode != "" {
		query += fmt.Sprintf(" AND indicator_code=$%d", arg)
		args = append(args, indicatorCode)
		arg++
	}
	if period != "" {
		query += fmt.Sprintf(" AND period=$%d", arg)
		args = append(args, period)
		arg++
	}
	if geographyCode != "" {
		query += fmt.Sprintf(" AND geography_code=$%d", arg)
		args = append(args, geographyCode)
		arg++
	}
	if !history {
		query += " AND is_current=true"
	}
	query += fmt.Sprintf(" ORDER BY published_at DESC LIMIT $%d", arg)
	args = append(args, limit)
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IndicatorObservation
	for rows.Next() {
		var observation IndicatorObservation
		var methodology []byte
		if err := rows.Scan(&observation.ID, &observation.IndicatorCode, &observation.Period,
			&observation.GeographyCode, &observation.Value, &observation.Unit, &observation.Version,
			&observation.IsCurrent, &observation.Status, &observation.SourceFormID,
			&observation.ProcessingRunID, &observation.SourceCount, &methodology, &observation.PublishedBy,
			&observation.TenantID, &observation.WorkspaceID, &observation.PublishedAt, &observation.CreatedAt); err != nil {
			return nil, err
		}
		if len(methodology) > 0 {
			_ = json.Unmarshal(methodology, &observation.Methodology)
		}
		out = append(out, observation)
	}
	return out, rows.Err()
}

func adminIndicatorObservationPublishHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req PublishIndicatorObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	observation, err := PublishIndicatorObservation(req, actor, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	_ = LogEvent("indicator.observation_published", "statcollect", "indicator_observation", observation.ID, observation)
	handoff := handoffIndicatorObservation(observation)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"observation": observation, "analytics_handoff": handoff})
}

func adminIndicatorObservationCalculateHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CalculateIndicatorObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	actor := strings.TrimSpace(r.Header.Get("X-Actor-ID"))
	observation, calculation, err := CalculateAndPublishIndicatorObservation(req, actor, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	_ = LogEvent("indicator.observation_calculated", "statcollect", "indicator_observation", observation.ID, map[string]interface{}{
		"observation": observation, "calculation": calculation,
	})
	handoff := handoffIndicatorObservation(observation)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"observation": observation, "calculation": calculation, "analytics_handoff": handoff,
	})
}

func adminIndicatorObservationValidateHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ValidateIndicatorObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.ObservationID) == "" {
		http.Error(w, "observation_id is required", http.StatusBadRequest)
		return
	}
	observations, err := ListIndicatorObservations("", "", "", true, 500, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var observation *IndicatorObservation
	for i := range observations {
		if observations[i].ID == strings.TrimSpace(req.ObservationID) {
			observation = &observations[i]
			break
		}
	}
	if observation == nil {
		http.Error(w, "observation not found in the same tenant/workspace", http.StatusNotFound)
		return
	}
	run, err := GetProcessingRun(observation.ProcessingRunID, scope)
	if err != nil || run.Status != "completed" {
		http.Error(w, "observation source processing run is not completed", http.StatusUnprocessableEntity)
		return
	}
	indicators, err := ListSDGIndicators(0, "", scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var indicator SDGIndicator
	found := false
	for _, candidate := range indicators {
		if candidate.IndicatorCode == observation.IndicatorCode {
			indicator = candidate
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "indicator is not registered in the same tenant/workspace", http.StatusUnprocessableEntity)
		return
	}
	quality, err := validateIndicatorObservationQuality(indicator, observation.Value, observation.Unit, observation.SourceCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"observation": observation, "quality_checks": quality})
}

func adminIndicatorObservationsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	history := r.URL.Query().Get("history") == "true"
	observations, err := ListIndicatorObservations(r.URL.Query().Get("indicator_code"), r.URL.Query().Get("period"), r.URL.Query().Get("geography_code"), history, limit, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"count": len(observations), "observations": observations})
}

func adminIndicatorObservationHandoffsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	handoffs, err := ListAnalyticsHandoffs(r.URL.Query().Get("observation_id"), limit, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"count": len(handoffs), "handoffs": handoffs})
}

func openDataObservationsHandler(w http.ResponseWriter, r *http.Request) {
	scope, err := officialStatisticsScope(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	observations, err := ListIndicatorObservations(r.URL.Query().Get("indicator_code"), r.URL.Query().Get("period"), r.URL.Query().Get("geography_code"), false, 500, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"source":  "StatGate National Statistics Dissemination Engine",
		"license": "Open Government Data License (OGDL)",
		"count":   len(observations), "observations": observations,
	})
}
