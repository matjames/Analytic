package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════
// PHASE 12 — MONITORING & EVALUATION (M&E) HANDLERS
// ═══════════════════════════════════════════════════════════════════

type MELogframe struct {
	ID               string           `json:"id"`
	EntityType       string           `json:"entity_type"` // project, programme, portfolio, strategy
	EntityID         string           `json:"entity_id"`
	Title            string           `json:"title"`
	NarrativeSummary string           `json:"narrative_summary"`
	Items            []MELogframeItem `json:"items"`
	CreatedTime      time.Time        `json:"created_time"`
}

type MELogframeItem struct {
	ID          string   `json:"id"`
	LogframeID  string   `json:"logframe_id"`
	Level       string   `json:"level"` // Impact, Outcome, Output, Activity
	Code        string   `json:"code"`
	Description string   `json:"description"`
	Indicators  []string `json:"indicators"`
	MeansOfVer  []string `json:"means_of_verification"`
	Assumptions []string `json:"assumptions"`
}

type MEEvaluation struct {
	ID             string    `json:"id"`
	EntityID       string    `json:"entity_id"`
	EntityType     string    `json:"entity_type"`
	Type           string    `json:"type"` // Baseline, Midline, Endline, Impact, Thematic
	LeadEvaluator  string    `json:"lead_evaluator"`
	TermsOfRef     string    `json:"terms_of_reference"`
	Budget         float64   `json:"budget"`
	StartDate      string    `json:"start_date"`
	EndDate        string    `json:"end_date"`
	Status         string    `json:"status"` // Planned, In_Progress, Completed, Published
	ReportURL      string    `json:"report_url,omitempty"`
	RecommendationsCount int `json:"recommendations_count"`
	CreatedTime    time.Time `json:"created_time"`
}

type MERecommendation struct {
	ID               string    `json:"id"`
	EvaluationID     string    `json:"evaluation_id"`
	RecommendationText string  `json:"recommendation_text"`
	Priority         string    `json:"priority"` // High, Medium, Low
	ResponsibleUnit  string    `json:"responsible_unit"`
	ActionPlan       string    `json:"action_plan"`
	Status           string    `json:"status"` // Open, In_Progress, Addressed, Rejected
	Deadline         string    `json:"deadline"`
	CreatedTime      time.Time `json:"created_time"`
}

func handleListLogframes(c *gin.Context) {
	entityID := c.Query("entity_id")
	entityType := c.Query("entity_type")

	// Demo / Seed M&E LogFrame
	logframes := []MELogframe{
		{
			ID:               "lf-001",
			EntityType:       "project",
			EntityID:         entityID,
			Title:            "Evidence for Sustainable Development LogFrame",
			NarrativeSummary: "Strengthening statistical evidence systems to monitor national social-economic transformation.",
			Items: []MELogframeItem{
				{
					ID:          "lfi-001",
					LogframeID:  "lf-001",
					Level:       "Impact",
					Code:        "IMP-1",
					Description: "Improved evidence-based policy formulation across participating ministries.",
					Indicators:  []string{"Proportion of national budget allocations aligned to official statistical evidence (Target: 85%)"},
					MeansOfVer:  []string{"Annual National Budget Performance Reports"},
					Assumptions: []string{"Continued political commitment to statistical integrity"},
				},
				{
					ID:          "lfi-002",
					LogframeID:  "lf-001",
					Level:       "Outcome",
					Code:        "OUT-1",
					Description: "Real-time survey collection and automated indicator tabulation operational in all target districts.",
					Indicators:  []string{"Survey data latency reduced from 90 days to < 7 days"},
					MeansOfVer:  []string{"StatCollect Sync Logs & Analytics Lakehouse Timestamp Audits"},
					Assumptions: []string{"Cellular / satellite field connectivity available"},
				},
			},
			CreatedTime: time.Now().Add(-30 * 24 * time.Hour),
		},
	}

	if entityType != "" {
		_ = entityType
	}

	c.JSON(http.StatusOK, logframes)
}

func handleCreateLogframe(c *gin.Context) {
	var lf MELogframe
	if err := c.ShouldBindJSON(&lf); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if lf.ID == "" {
		lf.ID = fmt.Sprintf("lf-%d", time.Now().Unix())
	}
	lf.CreatedTime = time.Now()
	c.JSON(http.StatusCreated, lf)
}

func handleListEvaluations(c *gin.Context) {
	evaluations := []MEEvaluation{
		{
			ID:                   "eval-001",
			EntityID:             "proj-national-stats",
			EntityType:           "project",
			Type:                 "Midline",
			LeadEvaluator:        "Dr. Grace Nakato",
			TermsOfRef:           "Mid-term performance assessment of electronic field data collection rollout.",
			Budget:               45000.0,
			StartDate:            "2026-03-01",
			EndDate:              "2026-06-30",
			Status:               "Completed",
			ReportURL:            "/api/files/doc-eval-001/download",
			RecommendationsCount: 4,
			CreatedTime:          time.Now().Add(-60 * 24 * time.Hour),
		},
	}
	c.JSON(http.StatusOK, evaluations)
}

func handleCreateEvaluation(c *gin.Context) {
	var ev MEEvaluation
	if err := c.ShouldBindJSON(&ev); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ev.ID == "" {
		ev.ID = fmt.Sprintf("eval-%d", time.Now().Unix())
	}
	ev.CreatedTime = time.Now()
	c.JSON(http.StatusCreated, ev)
}

func handleListMERecommendations(c *gin.Context) {
	recommendations := []MERecommendation{
		{
			ID:                 "rec-001",
			EvaluationID:       "eval-001",
			RecommendationText: "Incorporate offline SQLite validation rules in collect-master to reduce post-collection cleaning workload.",
			Priority:           "High",
			ResponsibleUnit:    "Field Operations & ICT",
			ActionPlan:         "Implement XForm constraint engine update by Q3 2026.",
			Status:             "In_Progress",
			Deadline:           "2026-09-30",
			CreatedTime:        time.Now().Add(-45 * 24 * time.Hour),
		},
		{
			ID:                 "rec-002",
			EvaluationID:       "eval-001",
			RecommendationText: "Establish weekly supervisor quota reconciliation meetings over StatChat channels.",
			Priority:           "Medium",
			ResponsibleUnit:    "Survey Management",
			ActionPlan:         "Create automated Monday morning StatChat broadcast bot.",
			Status:             "Addressed",
			Deadline:           "2026-07-15",
			CreatedTime:        time.Now().Add(-40 * 24 * time.Hour),
		},
	}
	c.JSON(http.StatusOK, recommendations)
}

func handleCreateMERecommendation(c *gin.Context) {
	var rec MERecommendation
	if err := c.ShouldBindJSON(&rec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if rec.ID == "" {
		rec.ID = fmt.Sprintf("rec-%d", time.Now().Unix())
	}
	rec.CreatedTime = time.Now()
	c.JSON(http.StatusCreated, rec)
}


func handleListSDGs(c *gin.Context) {
	sdgs := []map[string]interface{}{
		{"number": 1, "title": "No Poverty", "target_indicators": []string{"1.1.1", "1.2.1", "1.3.1"}, "tracking_count": 8},
		{"number": 2, "title": "Zero Hunger", "target_indicators": []string{"2.1.1", "2.1.2", "2.2.1"}, "tracking_count": 6},
		{"number": 3, "title": "Good Health and Well-being", "target_indicators": []string{"3.1.1", "3.2.1", "3.7.1"}, "tracking_count": 14},
		{"number": 4, "title": "Quality Education", "target_indicators": []string{"4.1.1", "4.2.1"}, "tracking_count": 10},
		{"number": 6, "title": "Clean Water and Sanitation", "target_indicators": []string{"6.1.1", "6.2.1"}, "tracking_count": 5},
		{"number": 8, "title": "Decent Work and Economic Growth", "target_indicators": []string{"8.1.1", "8.5.1"}, "tracking_count": 9},
		{"number": 17, "title": "Partnerships for the Goals", "target_indicators": []string{"17.18.1", "17.19.2"}, "tracking_count": 12},
	}
	c.JSON(http.StatusOK, gin.H{"sdgs": sdgs, "total_tracked_indicators": 64})
}
