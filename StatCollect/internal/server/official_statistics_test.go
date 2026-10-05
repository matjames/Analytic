package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateOpenRosaXForm(t *testing.T) {
	schema := QuestionnaireSchema{
		FormID:  "test_survey_2026",
		Title:   "National Sample Survey 2026",
		Version: "1.0",
		Sections: []DesignerSection{
			{
				ID:    "demographics",
				Title: "Household Roster",
				Questions: []DesignerQuestion{
					{
						ID:       "q1",
						Code:     "AGE",
						Label:    "Age in completed years",
						Type:     "integer",
						Required: true,
					},
					{
						ID:       "q2",
						Code:     "SEX",
						Label:    "Sex of respondent",
						Type:     "select_one",
						Required: true,
						Options: []struct {
							Value string `json:"value"`
							Label string `json:"label"`
						}{
							{"M", "Male"},
							{"F", "Female"},
						},
					},
				},
			},
		},
	}

	xmlDoc, err := GenerateOpenRosaXForm(schema)
	if err != nil {
		t.Fatalf("unexpected error generating XForm: %v", err)
	}

	// Verify standard OpenRosa XForm tags
	if !strings.Contains(xmlDoc, `<h:html xmlns="http://www.w3.org/2002/xforms"`) {
		t.Errorf("missing OpenRosa namespace")
	}
	if !strings.Contains(xmlDoc, `<data id="test_survey_2026" version="1.0">`) {
		t.Errorf("missing form instance root")
	}
	if !strings.Contains(xmlDoc, `<bind nodeset="/data/meta/instanceID"`) {
		t.Errorf("missing OpenRosa meta instanceID bind")
	}
	if !strings.Contains(xmlDoc, `<bind nodeset="/data/demographics/AGE" type="int" required="true()"/>`) {
		t.Errorf("missing AGE bind with int type and required")
	}
	if !strings.Contains(xmlDoc, `<select1 ref="/data/demographics/SEX">`) {
		t.Errorf("missing select1 control for SEX")
	}
}

func TestPPSClusterSamplingMath(t *testing.T) {
	eas := []EnumerationArea{
		{ID: "ea1", EACode: "EA-001", Name: "Central A", District: "Kampala", UrbanRural: "Urban", EstimatedHouseholds: 200},
		{ID: "ea2", EACode: "EA-002", Name: "Central B", District: "Kampala", UrbanRural: "Urban", EstimatedHouseholds: 400},
		{ID: "ea3", EACode: "EA-003", Name: "Eastern A", District: "Jinja", UrbanRural: "Urban", EstimatedHouseholds: 300},
		{ID: "ea4", EACode: "EA-004", Name: "Western A", District: "Mbarara", UrbanRural: "Rural", EstimatedHouseholds: 100},
	}

	req := PPSClusterSelectionRequest{
		SurveyTitle:          "UNHS Cluster Test",
		NumberOfClusters:     2,
		HouseholdsPerCluster: 25,
		Seed:                 42,
		AvailableEAs:         eas,
	}

	res, err := GeneratePPSClusterSample(req)
	if err != nil {
		t.Fatalf("unexpected error in PPS sampling: %v", err)
	}

	if res.TotalMeasureOfSize != 1000 {
		t.Errorf("expected total measure of size 1000, got %d", res.TotalMeasureOfSize)
	}
	if res.ClustersSelectedCount != 2 {
		t.Errorf("expected 2 selected clusters, got %d", res.ClustersSelectedCount)
	}
	if res.TargetHouseholdsCount != 50 {
		t.Errorf("expected 50 target households, got %d", res.TargetHouseholdsCount)
	}

	for _, c := range res.SelectedClusters {
		if c.BaseDesignWeight <= 0 {
			t.Errorf("expected positive design weight, got %f", c.BaseDesignWeight)
		}
		if c.ProbabilityStage1 <= 0 || c.ProbabilityStage1 > 1.0 {
			t.Errorf("invalid Stage 1 inclusion probability: %f", c.ProbabilityStage1)
		}
	}
}

func TestPESDualSystemEstimation(t *testing.T) {
	// Census: 40,000,000 counted; PES estimated 180,000 in sample; Matched: 168,000
	censusPop := int64(40000000)
	pesCount := int64(180000)
	matched := int64(168000)

	pesRes, err := CalculatePESDualSystemEstimation("census_2026", censusPop, pesCount, matched, 300)
	if err != nil {
		t.Fatalf("unexpected error calculating PES dual system: %v", err)
	}

	// Chandra-Sekar-Deming: N = (N1 * N2) / M = (40M * 180K) / 168K = ~42,857,142
	if pesRes.EstimatedTruePopulation <= censusPop {
		t.Errorf("expected estimated true pop to exceed census count, got %d", pesRes.EstimatedTruePopulation)
	}
	if pesRes.NetUndercountRate <= 0 || pesRes.NetUndercountRate > 20.0 {
		t.Errorf("unexpected net undercount rate: %f", pesRes.NetUndercountRate)
	}
	if pesRes.CoverageRate <= 0 || pesRes.CoverageRate > 100.0 {
		t.Errorf("unexpected coverage rate: %f", pesRes.CoverageRate)
	}
	if pesRes.GrossOmissionRate <= 0 || pesRes.GrossOmissionRate > 20.0 {
		t.Errorf("unexpected omission rate: %f", pesRes.GrossOmissionRate)
	}
}

func TestDemographicProfileAndDependencyRatios(t *testing.T) {
	prof := GenerateAgeSexPyramid()
	if prof == nil {
		t.Fatalf("profile is nil")
	}

	if prof.TotalPopulation <= 0 {
		t.Errorf("expected positive total population, got %d", prof.TotalPopulation)
	}
	if len(prof.Cohorts) != 17 {
		t.Errorf("expected 17 5-year cohorts, got %d", len(prof.Cohorts))
	}
	if prof.ChildDependencyRatio <= 0 || prof.ChildDependencyRatio > 150 {
		t.Errorf("unexpected child dependency ratio: %f", prof.ChildDependencyRatio)
	}
	if prof.OldAgeDependencyRatio <= 0 || prof.OldAgeDependencyRatio > 50 {
		t.Errorf("unexpected old age dependency ratio: %f", prof.OldAgeDependencyRatio)
	}
	if prof.TotalDependencyRatio < prof.ChildDependencyRatio {
		t.Errorf("total dependency ratio should be greater than child dependency ratio")
	}
}

func TestOfficialCrosstabAndChiSquare(t *testing.T) {
	req := CrosstabCalculationRequest{
		Title:       "Water Source by Residence",
		TableNumber: "Table 2.1",
		RowVariable: "water_source",
		ColVariable: "residence",
		Aggregation: "COUNT",
		IsWeighted:  true,
		Records: []map[string]interface{}{
			{"water_source": "Piped", "residence": "Urban", "weight": 1.0},
			{"water_source": "Piped", "residence": "Urban", "weight": 1.0},
			{"water_source": "Piped", "residence": "Rural", "weight": 1.0},
			{"water_source": "Borehole", "residence": "Rural", "weight": 1.0},
			{"water_source": "Borehole", "residence": "Rural", "weight": 1.0},
			{"water_source": "Borehole", "residence": "Urban", "weight": 1.0},
		},
	}

	res, err := ComputeOfficialCrosstab(req)
	if err != nil {
		t.Fatalf("unexpected error computing crosstab: %v", err)
	}

	if res.GrandTotal != 6.0 {
		t.Errorf("expected grand total 6.0, got %f", res.GrandTotal)
	}
	if res.MatrixData["Piped"]["Urban"] != 2.0 {
		t.Errorf("expected 2 piped urban, got %f", res.MatrixData["Piped"]["Urban"])
	}
	if len(res.StatisticalTests) == 0 {
		t.Errorf("expected Chi-Square statistical test in output")
	}

	csvBytes, err := ExportTabulationCSV(res)
	if err != nil {
		t.Fatalf("unexpected error exporting CSV: %v", err)
	}
	csvStr := string(csvBytes)
	if !strings.Contains(csvStr, "water_source") || !strings.Contains(csvStr, "Total") {
		t.Errorf("CSV output missing headers: %s", csvStr)
	}
}

func TestSDMXEndpoints(t *testing.T) {
	// Test SDMX Codelist
	req := httptest.NewRequest(http.MethodGet, "/admin/sdmx/codelist?id=CL_SEX", nil)
	rr := httptest.NewRecorder()
	adminSDMXCodelistHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var cl struct {
		CodelistID string              `json:"codelist_id"`
		Codes      []map[string]string `json:"codes"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &cl); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if cl.CodelistID != "CL_SEX" || len(cl.Codes) < 2 {
		t.Errorf("unexpected codelist output: %+v", cl)
	}

	// Test SDMX Dataflow
	reqDf := httptest.NewRequest(http.MethodGet, "/admin/sdmx/dataflow", nil)
	rrDf := httptest.NewRecorder()
	adminSDMXDataflowHandler(rrDf, reqDf)
	if rrDf.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rrDf.Code)
	}
	if !strings.Contains(rrDf.Body.String(), "DF_UNHS_POVERTY") {
		t.Errorf("missing poverty dataflow in SDMX registry")
	}
}

func TestSupervisorDashboardHandlerUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/supervisor/dashboard", nil)
	rr := httptest.NewRecorder()
	adminSupervisorDashboardHandler(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized without admin key, got %d", rr.Code)
	}
}

func TestIndicatorObservationQualityChecks(t *testing.T) {
	indicator := SDGIndicator{IndicatorCode: "quality.test", Unit: "%"}
	quality, err := validateIndicatorObservationQuality(indicator, 42.5, "%", 10)
	if err != nil {
		t.Fatalf("expected valid percentage quality check: %v", err)
	}
	if quality.Status != "passed" || len(quality.Warnings) != 0 {
		t.Fatalf("unexpected quality result: %+v", quality)
	}

	warning, err := validateIndicatorObservationQuality(indicator, 42.5, "%", 2)
	if err != nil {
		t.Fatalf("expected small-source warning, got: %v", err)
	}
	if warning.Status != "passed_with_warnings" || len(warning.Warnings) != 1 || warning.Warnings[0] != "small_source_count" {
		t.Fatalf("expected small-source warning, got: %+v", warning)
	}

	if _, err := validateIndicatorObservationQuality(indicator, 101, "%", 10); err == nil {
		t.Fatal("expected percentage range validation error")
	}
	if _, err := validateIndicatorObservationQuality(SDGIndicator{Unit: "count"}, -1, "count", 10); err == nil {
		t.Fatal("expected negative count validation error")
	}
	if _, err := validateIndicatorObservationQuality(indicator, 42.5, "number", 10); err == nil {
		t.Fatal("expected unit mismatch validation error")
	}
}
