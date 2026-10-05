package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type statisticalRecommendationDataset struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	SourceType       string `json:"sourceType"`
	CollectionMethod string `json:"collectionMethod"`
	MetadataComplete bool   `json:"metadataComplete"`
	StatcollectID    string `json:"statcollectId,omitempty"`
}

type statisticalRecommendationRequest struct {
	DatasetID   string `json:"datasetId"`
	OutcomeType string `json:"outcomeType"`
	Question    string `json:"question"`
}

type statisticalTabulationRequest struct {
	DatasetID       string `json:"datasetId"`
	Title           string `json:"title"`
	RowVariable     string `json:"rowVariable"`
	ColVariable     string `json:"colVariable"`
	MeasureVariable string `json:"measureVariable"`
	Aggregation     string `json:"aggregation"`
	WeightVariable  string `json:"weightVariable"`
}

type statisticalRecommendation struct {
	Key              string   `json:"key"`
	Method           string   `json:"method"`
	Rationale        string   `json:"rationale"`
	DataRequirements []string `json:"dataRequirements"`
	Assumptions      []string `json:"assumptions"`
	Priority         string   `json:"priority"`
}

func dbGetStatisticalRecommendations(c *gin.Context) {
	buildStatisticalRecommendationsResponse(c, statisticalRecommendationRequest{})
}

func dbPostStatisticalRecommendations(c *gin.Context) {
	var request statisticalRecommendationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid statistical recommendation request"})
		return
	}
	buildStatisticalRecommendationsResponse(c, request)
}

func dbRunStatisticalTabulation(c *gin.Context) {
	var request statisticalTabulationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tabulation request"})
		return
	}
	request.DatasetID = strings.TrimSpace(request.DatasetID)
	request.RowVariable = strings.TrimSpace(request.RowVariable)
	if request.DatasetID == "" || request.RowVariable == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datasetId and rowVariable are required"})
		return
	}
	datasets, err := loadStatisticalRecommendationDatasets(c, request.DatasetID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found in this workspace"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "dataset execution context unavailable"})
		return
	}
	dataset := datasets[0]
	if strings.TrimSpace(dataset.StatcollectID) == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "dataset is not bound to a StatCollect or Analytics Core dataset identifier"})
		return
	}
	endpoint := strings.TrimRight(strings.TrimSpace(getEnv("RMS_ANALYTICS_CORE_URL", "")), "/")
	if endpoint == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Analytics Core integration is not configured"})
		return
	}
	if request.Title == "" {
		request.Title = dataset.Name + " statistical analysis"
	}
	if request.Aggregation == "" {
		request.Aggregation = "COUNT"
	}
	payload := map[string]interface{}{
		"title":            request.Title,
		"dataset_id":       dataset.StatcollectID,
		"row_variable":     request.RowVariable,
		"col_variable":     request.ColVariable,
		"measure_variable": request.MeasureVariable,
		"aggregation":      strings.ToUpper(request.Aggregation),
		"weight_variable":  request.WeightVariable,
	}
	result, err := executeAnalyticsTabulation(c.Request.Context(), endpoint, payload, c.GetHeader("Authorization"), c.GetHeader("X-Tenant-ID"), workspaceIDContext(c), getEnv("STATGATE_INTERNAL_API_KEY", ""))
	if err != nil {
		c.JSON(analyticsTabulationErrorCode(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mode":          "analytics-core-tabulation",
		"researchId":    c.Param("id"),
		"datasetId":     dataset.ID,
		"coreDatasetId": dataset.StatcollectID,
		"result":        json.RawMessage(result),
		"generatedAt":   time.Now().UTC(),
	})
}

func executeAnalyticsTabulation(ctx context.Context, endpoint string, payload map[string]interface{}, authorization, tenantID, workspaceID, internalKey string) ([]byte, error) {
	if strings.TrimSpace(internalKey) == "" {
		return nil, &analyticsTabulationError{Code: http.StatusServiceUnavailable, Err: fmt.Errorf("Analytics Core internal service key is not configured")}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/api/v1/statistics/tabulate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-StatGate-Internal-Key", internalKey)
	if strings.TrimSpace(authorization) != "" {
		req.Header.Set("Authorization", authorization)
	}
	if strings.TrimSpace(tenantID) != "" {
		req.Header.Set("X-Tenant-ID", tenantID)
	}
	if strings.TrimSpace(workspaceID) != "" {
		req.Header.Set("X-Workspace-ID", workspaceID)
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, &analyticsTabulationError{Code: http.StatusBadGateway, Err: fmt.Errorf("Analytics Core request failed: %w", err)}
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := http.StatusBadGateway
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			code = resp.StatusCode
		}
		return nil, &analyticsTabulationError{Code: code, Err: fmt.Errorf("Analytics Core returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))}
	}
	var valid json.RawMessage
	if err := json.Unmarshal(responseBody, &valid); err != nil {
		return nil, &analyticsTabulationError{Code: http.StatusBadGateway, Err: fmt.Errorf("Analytics Core returned invalid JSON: %w", err)}
	}
	return responseBody, nil
}

type analyticsTabulationError struct {
	Code int
	Err  error
}

func (e *analyticsTabulationError) Error() string { return e.Err.Error() }

func analyticsTabulationErrorCode(err error) int {
	if providerErr, ok := err.(*analyticsTabulationError); ok {
		return providerErr.Code
	}
	return http.StatusBadGateway
}

func buildStatisticalRecommendationsResponse(c *gin.Context, request statisticalRecommendationRequest) {
	evidence, err := loadResearchQualityEvidence(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "research study not found"})
		return
	}
	datasets, err := loadStatisticalRecommendationDatasets(c, request.DatasetID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "dataset not found in this workspace"})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "dataset recommendation context unavailable"})
		return
	}
	recommendations := buildStatisticalRecommendations(datasets, request)
	analyticsURL := strings.TrimSpace(getEnv("RMS_ANALYTICS_CORE_URL", ""))
	analyticsState := "not-configured"
	analyticsNextStep := "Configure RMS_ANALYTICS_CORE_URL and bind a StatCollect or Analytics Core dataset before running a registered analysis."
	if analyticsURL != "" {
		analyticsState = "configured"
		analyticsNextStep = "Use the dataset identifier and recommended design in Analytics Core tabulation or sampling workflows; RMS does not execute an analysis from metadata alone."
	}
	c.JSON(http.StatusOK, gin.H{
		"mode":            "governed-statistical-recommendations",
		"researchId":      c.Param("id"),
		"researchName":    evidence.Name,
		"question":        strings.TrimSpace(request.Question),
		"outcomeType":     normalizeOutcomeType(request.OutcomeType),
		"datasets":        datasets,
		"recommendations": recommendations,
		"analyticsCore": gin.H{
			"status":     analyticsState,
			"nextStep":   analyticsNextStep,
			"configured": analyticsURL != "",
		},
		"disclaimer":  "These are transparent planning recommendations, not executed statistical tests. Confirm the design, estimand, assumptions, weights, missing-data handling, and results with a qualified statistician.",
		"generatedAt": time.Now().UTC(),
	})
}

func loadStatisticalRecommendationDatasets(c *gin.Context, datasetID string) ([]statisticalRecommendationDataset, error) {
	query := `SELECT d.id, d.name, COALESCE(d.status,'Draft'), COALESCE(d.source_type,''),
		COALESCE(d.collection_method,''), COALESCE(d.metadata_info,''), COALESCE(d.variables_dict,''), COALESCE(d.statcollect_id,'')
		FROM rms.datasets d JOIN rms.research_projects p ON p.id=d.research_id
		WHERE d.research_id=$1 AND (p.workspace_id = NULLIF($2,'') OR NULLIF($2,'') IS NULL)`
	args := []interface{}{c.Param("id"), workspaceIDContext(c)}
	if strings.TrimSpace(datasetID) != "" {
		query += " AND d.id=$3"
		args = append(args, strings.TrimSpace(datasetID))
	}
	query += " ORDER BY d.created_time DESC"
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make([]statisticalRecommendationDataset, 0)
	for rows.Next() {
		var item statisticalRecommendationDataset
		var metadataInfo, variablesDict string
		if err := rows.Scan(&item.ID, &item.Name, &item.Status, &item.SourceType, &item.CollectionMethod, &metadataInfo, &variablesDict, &item.StatcollectID); err != nil {
			return nil, err
		}
		item.MetadataComplete = strings.TrimSpace(metadataInfo) != "" && strings.TrimSpace(variablesDict) != ""
		results = append(results, item)
	}
	if strings.TrimSpace(datasetID) != "" && len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results, rows.Err()
}

func buildStatisticalRecommendations(datasets []statisticalRecommendationDataset, request statisticalRecommendationRequest) []statisticalRecommendation {
	recommendations := make([]statisticalRecommendation, 0, 7)
	if len(datasets) == 0 {
		return append(recommendations, statisticalRecommendation{
			Key: "register-dataset", Method: "Dataset registration and metadata profiling", Priority: "blocker",
			Rationale:        "No dataset is registered for this study, so an analysis design cannot be tied to an auditable data source.",
			DataRequirements: []string{"Dataset identifier", "Data dictionary", "Unit of observation", "Collection and sampling documentation"},
			Assumptions:      []string{"The proposed study question is not evaluated until data and metadata are registered."},
		})
	}
	for _, dataset := range datasets {
		if !dataset.MetadataComplete {
			recommendations = append(recommendations, statisticalRecommendation{
				Key: "metadata-" + dataset.ID, Method: "Complete data dictionary and measurement metadata", Priority: "blocker",
				Rationale:        dataset.Name + " is missing metadata information or a variables dictionary.",
				DataRequirements: []string{"Variable names and labels", "Measurement levels", "Missing-value codes", "Units and valid ranges"},
				Assumptions:      []string{"Variable roles cannot be safely inferred from names alone."},
			})
		}
	}

	outcomeType := normalizeOutcomeType(request.OutcomeType)
	commonRequirements := []string{"Prespecified outcome and exposure variables", "Analysis population and inclusion criteria", "Missing-data and exclusion rules"}
	switch outcomeType {
	case "binary":
		recommendations = append(recommendations, statisticalRecommendation{Key: "binary-outcome", Method: "Proportions with confidence intervals; logistic regression for adjusted associations", Priority: "recommended", Rationale: "The selected outcome is binary.", DataRequirements: commonRequirements, Assumptions: []string{"Independent observations unless clustering is explicitly modeled", "Adequate outcome events for the planned model"}})
	case "categorical":
		recommendations = append(recommendations, statisticalRecommendation{Key: "categorical-outcome", Method: "Frequency tables and cross-tabulation; chi-square or exact tests where appropriate", Priority: "recommended", Rationale: "The selected outcome is categorical.", DataRequirements: commonRequirements, Assumptions: []string{"Categories are mutually defined and coded consistently", "Expected cell counts are checked before asymptotic tests"}})
	case "count":
		recommendations = append(recommendations, statisticalRecommendation{Key: "count-outcome", Method: "Rate summaries; Poisson or negative-binomial regression after dispersion assessment", Priority: "recommended", Rationale: "The selected outcome is a count.", DataRequirements: append(commonRequirements, "Exposure time or population denominator where applicable"), Assumptions: []string{"The observation window and denominator are defined", "Overdispersion and zero inflation are assessed"}})
	case "time-to-event":
		recommendations = append(recommendations, statisticalRecommendation{Key: "time-to-event", Method: "Kaplan-Meier summaries and Cox regression when proportional-hazards assumptions are defensible", Priority: "recommended", Rationale: "The selected outcome is time-to-event.", DataRequirements: append(commonRequirements, "Event indicator", "Time origin and censoring rule"), Assumptions: []string{"Censoring is documented and its mechanism is considered", "Proportional-hazards assumptions are assessed before interpretation"}})
	default:
		recommendations = append(recommendations, statisticalRecommendation{Key: "continuous-outcome", Method: "Descriptive summaries with confidence intervals; linear or robust regression after diagnostics", Priority: "recommended", Rationale: "No outcome type was selected, so a continuous-outcome planning baseline is shown for review.", DataRequirements: commonRequirements, Assumptions: []string{"Outcome scale and distribution are confirmed before selecting a final model", "Independence, linearity, and residual behavior are assessed"}})
	}

	for _, dataset := range datasets {
		method := strings.ToLower(dataset.CollectionMethod + " " + dataset.SourceType)
		if strings.Contains(method, "survey") || strings.Contains(method, "odk") || strings.Contains(method, "sample") {
			recommendations = append(recommendations, statisticalRecommendation{Key: "survey-design-" + dataset.ID, Method: "Design-aware estimates with weights, strata, clusters, and finite-population settings where applicable", Priority: "recommended", Rationale: dataset.Name + " appears to come from a survey or sample-based collection process.", DataRequirements: []string{"Sampling frame", "Selection probabilities or weights", "Strata and cluster identifiers", "Nonresponse and calibration rules"}, Assumptions: []string{"The declared sample design matches the collection protocol"}})
		}
	}
	recommendations = append(recommendations, statisticalRecommendation{Key: "sensitivity-and-missingness", Method: "Missingness assessment and sensitivity analysis", Priority: "required", Rationale: "Every research analysis needs an explicit missing-data decision before results are interpreted.", DataRequirements: []string{"Missingness by variable and study subgroup", "Reasons for missingness where available", "Primary and sensitivity analysis plan"}, Assumptions: []string{"Complete-case analysis is not automatically unbiased"}})
	return recommendations
}

func normalizeOutcomeType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "binary", "categorical", "count", "time-to-event":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "continuous"
	}
}
