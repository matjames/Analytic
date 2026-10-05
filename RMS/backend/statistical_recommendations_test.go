package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildStatisticalRecommendationsWithoutDataset(t *testing.T) {
	recommendations := buildStatisticalRecommendations(nil, statisticalRecommendationRequest{OutcomeType: "binary"})
	if len(recommendations) != 1 || recommendations[0].Key != "register-dataset" {
		t.Fatalf("recommendations = %#v, want dataset blocker", recommendations)
	}
}

func TestBuildStatisticalRecommendationsForSurveyBinaryDataset(t *testing.T) {
	recommendations := buildStatisticalRecommendations([]statisticalRecommendationDataset{{
		ID: "dataset-1", Name: "Household survey", CollectionMethod: "ODK survey", SourceType: "sample", MetadataComplete: true,
	}}, statisticalRecommendationRequest{OutcomeType: "binary"})
	keys := make(map[string]bool, len(recommendations))
	for _, recommendation := range recommendations {
		keys[recommendation.Key] = true
	}
	if !keys["binary-outcome"] || !keys["survey-design-dataset-1"] || !keys["sensitivity-and-missingness"] {
		t.Fatalf("recommendation keys = %#v", keys)
	}
	for _, recommendation := range recommendations {
		if strings.TrimSpace(recommendation.Method) == "" || len(recommendation.DataRequirements) == 0 || len(recommendation.Assumptions) == 0 {
			t.Fatalf("recommendation lacks transparent planning fields: %#v", recommendation)
		}
	}
}

func TestNormalizeOutcomeType(t *testing.T) {
	if got := normalizeOutcomeType("TIME-TO-EVENT"); got != "time-to-event" {
		t.Fatalf("normalized outcome = %q", got)
	}
	if got := normalizeOutcomeType("unsupported"); got != "continuous" {
		t.Fatalf("fallback outcome = %q", got)
	}
}

func TestExecuteAnalyticsTabulationPreservesWorkspaceContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/statistics/tabulate" || r.Method != http.MethodPost {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-StatGate-Internal-Key") != "internal-test" || r.Header.Get("X-Workspace-ID") != "workspace-test" || r.Header.Get("X-Tenant-ID") != "tenant-test" || r.Header.Get("Authorization") != "Bearer caller" {
			t.Fatalf("forwarded headers missing: %+v", r.Header)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["dataset_id"] != "core-dataset-1" || payload["row_variable"] != "sex" {
			t.Fatalf("payload = %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"title":"tabulation","total_record_count":12}`)
	}))
	defer server.Close()

	result, err := executeAnalyticsTabulation(context.Background(), server.URL, map[string]interface{}{
		"dataset_id": "core-dataset-1", "row_variable": "sex", "aggregation": "COUNT",
	}, "Bearer caller", "tenant-test", "workspace-test", "internal-test")
	if err != nil {
		t.Fatalf("execute tabulation: %v", err)
	}
	if !strings.Contains(string(result), "total_record_count") {
		t.Fatalf("result = %s", result)
	}
}

func TestExecuteAnalyticsTabulationRequiresInternalKey(t *testing.T) {
	_, err := executeAnalyticsTabulation(context.Background(), "http://127.0.0.1:1", map[string]interface{}{}, "", "", "", "")
	if err == nil {
		t.Fatal("expected missing internal key error")
	}
	if got := analyticsTabulationErrorCode(err); got != http.StatusServiceUnavailable {
		t.Fatalf("error code = %d, want %d", got, http.StatusServiceUnavailable)
	}
}
