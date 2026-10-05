package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SetupOfficialStatisticsRoutes registers all Phase 6 Official Statistics endpoints.
func SetupOfficialStatisticsRoutes() {
	// Question Bank & Survey Projects
	http.HandleFunc("/admin/question-bank", adminQuestionBankHandler)
	http.HandleFunc("/admin/survey-projects", adminSurveyProjectsHandler)
	http.HandleFunc("/admin/survey-projects/phase", adminSurveyProjectPhaseHandler)
	http.HandleFunc("/admin/questionnaires/xform", adminQuestionnaireXFormHandler)

	// Sampling & Enumeration Areas
	http.HandleFunc("/admin/sampling/eas", adminEnumerationAreasHandler)
	http.HandleFunc("/admin/sampling/frames", adminSampleFramesHandler)
	http.HandleFunc("/admin/sampling/pps-cluster", adminPPSClusterHandler)

	// Field Workforce & Assignments
	http.HandleFunc("/admin/workforce/supervisors", adminSupervisorsHandler)
	http.HandleFunc("/admin/workforce/enumerators", adminEnumeratorsHandler)
	http.HandleFunc("/admin/workforce/assignments", adminAssignmentPlansHandler)

	// Supervisor Field Monitoring & Reviews
	http.HandleFunc("/admin/supervisor/dashboard", adminSupervisorDashboardHandler)
	http.HandleFunc("/admin/supervisor/review", adminSupervisorReviewHandler)

	// Official Statistics Tabulation Engine
	http.HandleFunc("/admin/statistics/tabulate", adminOfficialTabulateHandler)
	http.HandleFunc("/admin/statistics/tables/export", adminOfficialTabulateExportHandler)
	http.HandleFunc("/admin/statistics/processing/weight", adminProcessingWeightHandler)
	http.HandleFunc("/admin/statistics/processing/transform", adminProcessingTransformHandler)
	http.HandleFunc("/admin/statistics/processing/runs", adminProcessingRunsHandler)
	http.HandleFunc("/admin/statistics/processing/schedules", adminProcessingSchedulesHandler)
	http.HandleFunc("/admin/statistics/processing/schedule-runs", adminProcessingScheduleRunsHandler)
	http.HandleFunc("/admin/statistics/processing/schedules/run", adminRunProcessingScheduleHandler)
	http.HandleFunc("/admin/statistics/datasets/handoff", adminDatasetHandoffHandler)
	http.HandleFunc("/admin/statistics/datasets/handoffs", adminDatasetHandoffsHandler)
	http.HandleFunc("/internal/statistics/datasets/refresh", internalDatasetRefreshHandler)

	// SDMX 2.1 Metadata & Registry
	http.HandleFunc("/admin/sdmx/dataflow", adminSDMXDataflowHandler)
	http.HandleFunc("/admin/sdmx/datastructure", adminSDMXDataStructureHandler)
	http.HandleFunc("/admin/sdmx/codelist", adminSDMXCodelistHandler)
	http.HandleFunc("/admin/sdmx/data", adminSDMXDataHandler)

	// Census Operations & Post-Enumeration Survey (PES)
	http.HandleFunc("/admin/census/rounds", adminCensusRoundsHandler)
	http.HandleFunc("/admin/census/pes/calculate", adminCensusPESHandler)
	http.HandleFunc("/admin/census/demographics/pyramid", adminDemographicPyramidHandler)

	// UN Sustainable Development Goals (SDG) Monitoring
	http.HandleFunc("/admin/sdg/goals", adminSDGGoalsHandler)
	http.HandleFunc("/admin/sdg/indicators", adminSDGIndicatorsHandler)
	http.HandleFunc("/admin/sdg/dashboard", adminSDGDashboardHandler)
	http.HandleFunc("/admin/sdg/observations/handoffs", adminIndicatorObservationHandoffsHandler)
	http.HandleFunc("/admin/sdg/observations/calculate", adminIndicatorObservationCalculateHandler)
	http.HandleFunc("/admin/sdg/observations/validate", adminIndicatorObservationValidateHandler)
	http.HandleFunc("/admin/sdg/observations/publish", adminIndicatorObservationPublishHandler)
	http.HandleFunc("/admin/sdg/observations", adminIndicatorObservationsHandler)

	// Dissemination Portal & Open Data API
	http.HandleFunc("/admin/dissemination/publications", adminDisseminationPublicationsHandler)
	http.HandleFunc("/api/v1/open-data/indicators", openDataIndicatorsHandler)
	http.HandleFunc("/api/v1/open-data/observations", openDataObservationsHandler)
	http.HandleFunc("/api/v1/open-data/datasets", openDataDatasetsHandler)
}

// ─────────────────────────────────────────────────────────────────────────────
// Question Bank & Survey Project Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminQuestionBankHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		domain := r.URL.Query().Get("domain")
		search := r.URL.Query().Get("q")
		items, err := ListQuestionBank(domain, search, scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": len(items),
			"items": items,
		})

	case http.MethodPost:
		var q QuestionBankItem
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if q.Code == "" || q.Label == "" || q.Domain == "" || q.QuestionType == "" {
			http.Error(w, "code, label, domain, and question_type are required", http.StatusBadRequest)
			return
		}
		if err := SaveQuestionBankItem(q, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("question_bank.created", "statcollect", "question_bank", q.Code, q)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(q)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		if err := DeleteQuestionBankItem(id, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminSurveyProjectsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		if id != "" {
			proj, err := GetSurveyProject(id, scope)
			if err != nil {
				http.Error(w, "not found: "+err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(proj)
			return
		}
		status := r.URL.Query().Get("status")
		projects, err := ListSurveyProjects(status, scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":    len(projects),
			"projects": projects,
		})

	case http.MethodPost:
		var p SurveyProject
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if p.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}
		if err := SaveSurveyProject(p, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("survey_project.saved", "statcollect", "survey_project", p.ID, p)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminSurveyProjectPhaseHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID     string `json:"id"`
		Phase  string `json:"phase"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" || req.Phase == "" {
		http.Error(w, "id and phase are required", http.StatusBadRequest)
		return
	}
	if err := UpdateSurveyProjectPhase(req.ID, req.Phase, req.Status, scope); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = LogEvent("survey_project.phase_changed", "statcollect", "survey_project", req.ID, req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "phase": req.Phase})
}

func adminQuestionnaireXFormHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireOfficialStatisticsScope(w, r); !ok {
		return
	}
	var schema QuestionnaireSchema
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
	} else {
		// Provide default questionnaire if GET
		schema = QuestionnaireSchema{
			FormID:  "unhs_official_2026",
			Title:   "Uganda National Household Survey 2026",
			Version: "2026.1",
			Sections: []DesignerSection{
				{
					ID:    "sec_household_roster",
					Title: "Section 1: Household Demographic Roster",
					Questions: []DesignerQuestion{
						{ID: "q1", Code: "AGE", Label: "Age of member in completed years", Type: "integer", Required: true},
						{ID: "q2", Code: "SEX", Label: "Sex of household member", Type: "select_one", Required: true,
							Options: []struct {
								Value string `json:"value"`
								Label string `json:"label"`
							}{{"M", "Male"}, {"F", "Female"}}},
						{ID: "q3", Code: "MARITAL_STATUS", Label: "Marital status", Type: "select_one", Required: true, Relevant: "AGE &gt;= 12",
							Options: []struct {
								Value string `json:"value"`
								Label string `json:"label"`
							}{{"never", "Never Married"}, {"married", "Married"}, {"divorced", "Divorced"}, {"widowed", "Widowed"}}},
					},
				},
				{
					ID:    "sec_water_energy",
					Title: "Section 2: Water, Sanitation & Energy Access",
					Questions: []DesignerQuestion{
						{ID: "q4", Code: "WATER_SOURCE", Label: "Main source of drinking water", Type: "select_one", Required: true,
							Options: []struct {
								Value string `json:"value"`
								Label string `json:"label"`
							}{{"piped", "Piped Water"}, {"borehole", "Protected Borehole"}, {"spring", "Protected Spring"}, {"surface", "Surface Water"}}},
						{ID: "q5", Code: "LIGHTING_ENERGY", Label: "Primary source of lighting", Type: "select_one", Required: true,
							Options: []struct {
								Value string `json:"value"`
								Label string `json:"label"`
							}{{"grid", "National Grid"}, {"solar", "Solar Home System"}, {"kerosene", "Kerosene"}, {"candles", "Candles"}}},
					},
				},
			},
		}
	}

	xmlDoc, err := GenerateOpenRosaXForm(schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(xmlDoc))
}

// ─────────────────────────────────────────────────────────────────────────────
// Sampling & Enumeration Areas Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminEnumerationAreasHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		region := r.URL.Query().Get("region")
		district := r.URL.Query().Get("district")
		status := r.URL.Query().Get("status")
		eas, err := ListEnumerationAreas(region, district, status, scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": len(eas),
			"eas":   eas,
		})

	case http.MethodPost:
		var ea EnumerationArea
		if err := json.NewDecoder(r.Body).Decode(&ea); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if ea.EACode == "" || ea.Name == "" || ea.Region == "" || ea.District == "" {
			http.Error(w, "ea_code, name, region, and district are required", http.StatusBadRequest)
			return
		}
		if err := SaveEnumerationArea(ea, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ea)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminSampleFramesHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	frames, err := ListMasterSamplingFrames(scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":  len(frames),
		"frames": frames,
	})
}

func adminPPSClusterHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PPSClusterSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// If available EAs not supplied in payload, load from database
	if len(req.AvailableEAs) == 0 {
		eas, err := ListEnumerationAreas("", "", "", scope)
		if err != nil {
			http.Error(w, "failed to load EAs: "+err.Error(), http.StatusInternalServerError)
			return
		}
		req.AvailableEAs = eas
	}

	res, err := GeneratePPSClusterSample(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = LogEvent("sampling.pps_designed", "statcollect", "sampling", req.SurveyTitle, res)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// ─────────────────────────────────────────────────────────────────────────────
// Workforce & Assignment Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminSupervisorsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	sups, err := ListSupervisors(scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":       len(sups),
		"supervisors": sups,
	})
}

func adminEnumeratorsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	supID := r.URL.Query().Get("supervisor_id")
	status := r.URL.Query().Get("status")
	enums, err := ListEnumerators(supID, status, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":       len(enums),
		"enumerators": enums,
	})
}

func adminAssignmentPlansHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		surveyID := r.URL.Query().Get("survey_id")
		supID := r.URL.Query().Get("supervisor_id")
		status := r.URL.Query().Get("status")
		plans, err := ListAssignmentPlans(surveyID, supID, status, scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": len(plans),
			"plans": plans,
		})

	case http.MethodPost:
		var p AssignmentPlan
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if p.SurveyID == "" || p.EAID == "" {
			http.Error(w, "survey_id and ea_id are required", http.StatusBadRequest)
			return
		}
		if err := SaveAssignmentPlan(p, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("assignment.saved", "statcollect", "assignment", fmt.Sprintf("%d", p.ID), p)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Supervisor Field Monitoring & Reviews Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminSupervisorDashboardHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	kpis, err := GetSupervisorDashboardKPIs(scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kpis)
}

func adminSupervisorReviewHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var rev SupervisorReview
	if err := json.NewDecoder(r.Body).Decode(&rev); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if rev.SubmissionInstanceID == "" || rev.Decision == "" {
		http.Error(w, "submission_instance_id and decision are required", http.StatusBadRequest)
		return
	}
	if rev.SupervisorID == "" {
		rev.SupervisorID = "supervisor_field_lead"
	}
	if err := SaveSupervisorReview(rev, scope); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = LogEvent("submission.supervisor_reviewed", "statcollect", "submission", rev.SubmissionInstanceID, rev)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"decision": rev.Decision,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Official Statistics Tabulation Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminOfficialTabulateHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CrosstabCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ProcessingRunID != "" && len(req.Records) == 0 {
		run, loadErr := GetProcessingRun(req.ProcessingRunID, scope)
		if loadErr != nil || run.Status != "completed" {
			http.Error(w, "completed processing_run_id is required", http.StatusUnprocessableEntity)
			return
		}
		records, loadErr := GetProcessingRunRecords(req.ProcessingRunID, scope)
		if loadErr != nil || len(records) == 0 {
			http.Error(w, "processing run has no output records", http.StatusUnprocessableEntity)
			return
		}
		for _, record := range records {
			req.Records = append(req.Records, record.Values)
		}
		req.IsWeighted = true
		if req.WeightVariable == "" {
			req.WeightVariable = run.WeightVariable
		}
	} else if req.FormID != "" && len(req.Records) == 0 {
		subs, loadErr := GetSubmissionsByFormID(req.FormID, 5000, scope)
		if loadErr != nil {
			http.Error(w, loadErr.Error(), http.StatusInternalServerError)
			return
		}
		for _, sub := range subs {
			if sub.Status == "approved" && sub.Meta != nil {
				req.Records = append(req.Records, sub.Meta)
			}
		}
		if len(req.Records) == 0 {
			http.Error(w, "no approved submissions found for form_id", http.StatusUnprocessableEntity)
			return
		}
	}

	res, err := ComputeOfficialCrosstab(req, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func adminOfficialTabulateExportHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CrosstabCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ProcessingRunID != "" && len(req.Records) == 0 {
		run, loadErr := GetProcessingRun(req.ProcessingRunID, scope)
		if loadErr != nil || run.Status != "completed" {
			http.Error(w, "completed processing_run_id is required", http.StatusUnprocessableEntity)
			return
		}
		records, loadErr := GetProcessingRunRecords(req.ProcessingRunID, scope)
		if loadErr != nil || len(records) == 0 {
			http.Error(w, "processing run has no output records", http.StatusUnprocessableEntity)
			return
		}
		for _, record := range records {
			req.Records = append(req.Records, record.Values)
		}
		req.IsWeighted = true
		if req.WeightVariable == "" {
			req.WeightVariable = run.WeightVariable
		}
	} else if req.FormID != "" && len(req.Records) == 0 {
		subs, loadErr := GetSubmissionsByFormID(req.FormID, 5000, scope)
		if loadErr != nil {
			http.Error(w, loadErr.Error(), http.StatusInternalServerError)
			return
		}
		for _, sub := range subs {
			if sub.Status == "approved" && sub.Meta != nil {
				req.Records = append(req.Records, sub.Meta)
			}
		}
		if len(req.Records) == 0 {
			http.Error(w, "no approved submissions found for form_id", http.StatusUnprocessableEntity)
			return
		}
	}

	res, err := ComputeOfficialCrosstab(req, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "csv" {
		csvBytes, err := ExportTabulationCSV(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="tabulation_table.csv"`)
		w.Write(csvBytes)
		return
	}

	// Default to JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// ─────────────────────────────────────────────────────────────────────────────
// SDMX 2.1 Metadata Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminSDMXDataflowHandler(w http.ResponseWriter, r *http.Request) {
	dataflows := []map[string]interface{}{
		{
			"id":       "DF_UNHS_POVERTY",
			"agencyID": "UG_UBOS",
			"version":  "1.0",
			"name":     "National Poverty & Welfare Indicators",
			"dsdRef":   "DSD_POVERTY",
			"isFinal":  true,
		},
		{
			"id":       "DF_DEMOGRAPHY_CENSUS",
			"agencyID": "UG_UBOS",
			"version":  "2026.1",
			"name":     "National Population Census Demographics",
			"dsdRef":   "DSD_DEMOGRAPHY",
			"isFinal":  true,
		},
		{
			"id":       "DF_SDG_NATIONAL",
			"agencyID": "UG_UBOS",
			"version":  "1.0",
			"name":     "Sustainable Development Goals National Series",
			"dsdRef":   "DSD_SDG",
			"isFinal":  true,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"header": map[string]interface{}{
			"id":       "DF_LIST",
			"prepared": time.Now().UTC().Format(time.RFC3339),
			"sender":   map[string]string{"id": "STATGATE_COLLECT", "name": "StatGate Official Statistics Portal"},
		},
		"dataflows": dataflows,
	})
}

func adminSDMXDataStructureHandler(w http.ResponseWriter, r *http.Request) {
	dsdID := r.URL.Query().Get("id")
	if dsdID == "" {
		dsdID = "DSD_POVERTY"
	}

	dsd := map[string]interface{}{
		"id":       dsdID,
		"agencyID": "UG_UBOS",
		"version":  "1.0",
		"name":     "Official Statistics Data Structure Definition",
		"dimensions": []map[string]interface{}{
			{"id": "FREQ", "position": 1, "codelist": "CL_FREQ"},
			{"id": "REF_AREA", "position": 2, "codelist": "CL_GEO_UGANDA"},
			{"id": "SEX", "position": 3, "codelist": "CL_SEX"},
			{"id": "AGE_GROUP", "position": 4, "codelist": "CL_AGE_GROUP"},
			{"id": "INDICATOR", "position": 5, "codelist": "CL_INDICATOR"},
		},
		"primaryMeasure": map[string]interface{}{
			"id":      "OBS_VALUE",
			"concept": "OBS_VALUE",
		},
		"attributes": []map[string]interface{}{
			{"id": "UNIT_MEASURE", "assignmentStatus": "Mandatory", "codelist": "CL_UNIT"},
			{"id": "OBS_STATUS", "assignmentStatus": "Conditional", "codelist": "CL_OBS_STATUS"},
			{"id": "DECIMALS", "assignmentStatus": "Mandatory"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dsd)
}

func adminSDMXCodelistHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	codelists := map[string][]map[string]string{
		"CL_FREQ": {
			{"id": "A", "name": "Annual"},
			{"id": "Q", "name": "Quarterly"},
			{"id": "M", "name": "Monthly"},
		},
		"CL_SEX": {
			{"id": "T", "name": "Total / Both Sexes"},
			{"id": "M", "name": "Male"},
			{"id": "F", "name": "Female"},
		},
		"CL_GEO_UGANDA": {
			{"id": "UGA", "name": "Uganda (National)"},
			{"id": "UGA_CEN", "name": "Central Region"},
			{"id": "UGA_EAS", "name": "Eastern Region"},
			{"id": "UGA_WES", "name": "Western Region"},
			{"id": "UGA_NOR", "name": "Northern Region"},
			{"id": "UGA_KLA", "name": "Kampala Capital City"},
		},
		"CL_OBS_STATUS": {
			{"id": "A", "name": "Normal observation"},
			{"id": "E", "name": "Estimated value"},
			{"id": "P", "name": "Provisional value"},
			{"id": "F", "name": "Forecast value"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if id != "" && codelists[id] != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"codelist_id": id, "codes": codelists[id]})
	} else {
		json.NewEncoder(w).Encode(codelists)
	}
}

func adminSDMXDataHandler(w http.ResponseWriter, r *http.Request) {
	dataflow := r.URL.Query().Get("dataflow")
	if dataflow == "" {
		dataflow = "DF_UNHS_POVERTY"
	}

	payload := map[string]interface{}{
		"header": map[string]interface{}{
			"id":       "SDMX_MSG_" + fmt.Sprintf("%d", time.Now().Unix()),
			"test":     false,
			"prepared": time.Now().UTC().Format(time.RFC3339),
			"sender":   map[string]string{"id": "UG_UBOS", "name": "Uganda National Statistics Office"},
		},
		"dataSets": []map[string]interface{}{
			{
				"action":       "Information",
				"structureRef": "DSD_POVERTY",
				"series": map[string]interface{}{
					"A.UGA.T.TOTAL.POV_HEADCOUNT": map[string]interface{}{
						"attributes": []interface{}{"%", "A", 1},
						"observations": map[string]interface{}{
							"2015": []interface{}{28.0},
							"2020": []interface{}{24.5},
							"2026": []interface{}{20.3},
						},
					},
					"A.UGA_CEN.T.TOTAL.POV_HEADCOUNT": map[string]interface{}{
						"attributes": []interface{}{"%", "A", 1},
						"observations": map[string]interface{}{
							"2026": []interface{}{12.4},
						},
					},
					"A.UGA_NOR.T.TOTAL.POV_HEADCOUNT": map[string]interface{}{
						"attributes": []interface{}{"%", "A", 1},
						"observations": map[string]interface{}{
							"2026": []interface{}{31.8},
						},
					},
				},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

// ─────────────────────────────────────────────────────────────────────────────
// Census Operations & PES Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminCensusRoundsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	rounds, err := ListCensusRounds(scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":  len(rounds),
		"rounds": rounds,
	})
}

func adminCensusPESHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CensusRoundID string `json:"census_round_id"`
		CensusCount   int64  `json:"census_count"`
		PESCount      int64  `json:"pes_count"`
		MatchedCount  int64  `json:"matched_count"`
		PESSampleEAs  int    `json:"pes_sample_eas"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.CensusRoundID == "" {
		req.CensusRoundID = "census_2026"
	}
	if req.CensusCount == 0 {
		req.CensusCount = 42180000
		req.PESCount = 185000
		req.MatchedCount = 172050
		req.PESSampleEAs = 450
	}

	res, err := CalculatePESDualSystemEstimation(req.CensusRoundID, req.CensusCount, req.PESCount, req.MatchedCount, req.PESSampleEAs, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = LogEvent("census.pes_evaluated", "statcollect", "census", req.CensusRoundID, res)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func adminDemographicPyramidHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireOfficialStatisticsScope(w, r); !ok {
		return
	}
	pyramid := GenerateAgeSexPyramid()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pyramid)
}

// ─────────────────────────────────────────────────────────────────────────────
// UN Sustainable Development Goals (SDG) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminSDGGoalsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	dash, err := GetSDGDashboard(scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dash.GoalSummaries)
}

func adminSDGIndicatorsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	goalStr := r.URL.Query().Get("goal")
	tier := r.URL.Query().Get("tier")
	goalNum := 0
	if goalStr != "" {
		goalNum, _ = strconv.Atoi(goalStr)
	}

	indicators, err := ListSDGIndicators(goalNum, tier, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(indicators),
		"indicators": indicators,
	})
}

func adminSDGDashboardHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	dash, err := GetSDGDashboard(scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dash)
}

// ─────────────────────────────────────────────────────────────────────────────
// Dissemination & Open Data Handlers
// ─────────────────────────────────────────────────────────────────────────────

func adminDisseminationPublicationsHandler(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireOfficialStatisticsScope(w, r)
	if !ok {
		return
	}
	pubType := r.URL.Query().Get("type")
	domain := r.URL.Query().Get("domain")
	pubs, err := ListStatisticalPublications(pubType, domain, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":        len(pubs),
		"publications": pubs,
	})
}

func openDataIndicatorsHandler(w http.ResponseWriter, r *http.Request) {
	scope, err := officialStatisticsScope(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	indicators, err := ListSDGIndicators(0, "", scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"source":     "StatGate National Statistics Dissemination Engine",
		"license":    "Open Government Data License (OGDL)",
		"count":      len(indicators),
		"indicators": indicators,
	})
}

func openDataDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	scope, err := officialStatisticsScope(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pubs, err := ListStatisticalPublications("", "", scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"portal":   "StatGate Official Statistics Open Data Portal",
		"datasets": pubs,
	})
}
