package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// Shared Reusable Analytics Engine Service
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type AnalyticsService struct{}

var Analytics = &AnalyticsService{}

// GeneralStats holds global, high-level KPIs across all forms and submissions
type GeneralStats struct {
	TotalSurveys        int            `json:"total_surveys"`
	ActiveSurveys       int            `json:"active_surveys"`
	CompletedSurveys    int            `json:"completed_surveys"`
	InProgressSurveys   int            `json:"in_progress_surveys"`
	ReceivedToday       int            `json:"received_today"`
	ReceivedThisWeek    int            `json:"received_this_week"`
	ReceivedThisMonth   int            `json:"received_this_month"`
	ReceivedThisYear    int            `json:"received_this_year"`
	ActiveProjectsCount int            `json:"active_projects_count"`
	ReportingRegions    []string       `json:"reporting_regions"`
	OrganizationsCount  int            `json:"organizations_count"`
	OnlineTeamsCount    int            `json:"online_teams_count"`
}

// SurveyStats holds analytics specifically computed for a single survey form
type SurveyStats struct {
	FormID                  string                    `json:"form_id"`
	TotalSubmissions        int                       `json:"total_submissions"`
	DailySubmissions        map[string]int            `json:"daily_submissions"`
	WeeklySubmissions       map[string]int            `json:"weekly_submissions"`
	MonthlySubmissions      map[string]int            `json:"monthly_submissions"`
	CompletionRate          float64                   `json:"completion_rate"`
	ApprovedSubmissions     int                       `json:"approved_submissions"`
	RejectedSubmissions     int                       `json:"rejected_submissions"`
	AverageDurationSeconds  float64                   `json:"average_duration_seconds"`
	MissingValueRate        float64                   `json:"missing_value_rate"`
	ValidationFailureCount  int                       `json:"validation_failure_count"`
	EnumeratorProductivity  []EnumeratorProd          `json:"enumerator_productivity"`
	QuestionDistributions   map[string]map[string]int `json:"question_distributions"`
}

type EnumeratorProd struct {
	EnumeratorID    string  `json:"enumerator_id"`
	TotalInterviews int     `json:"total_interviews"`
	ApprovedRate    float64 `json:"approved_rate"`
	AvgDuration     float64 `json:"avg_duration"`
}

// DataQualityStats holds stats on quality indicators
type DataQualityStats struct {
	FormID                  string  `json:"form_id"`
	CompletenessScore       float64 `json:"completeness_score"`
	DuplicateRecordCount    int     `json:"duplicate_record_count"`
	OutlierCount            int     `json:"outlier_count"`
	ValidationFailureCount  int     `json:"validation_failure_count"`
	GPSInconsistencyCount   int     `json:"gps_inconsistency_count"`
	RapidInterviewCount     int     `json:"rapid_interview_count"`
	SuspiciousResponseCount int     `json:"suspicious_response_count"`
	OverallQualityScore     float64 `json:"overall_quality_score"`
}

// ProjectStats holds progress for a survey project
type ProjectStats struct {
	ProjectID            string         `json:"project_id"`
	TotalSurveys         int            `json:"total_surveys"`
	CompletedSurveys     int            `json:"completed_surveys"`
	TotalSubmissions     int            `json:"total_submissions"`
	ApprovedSubmissions  int            `json:"approved_submissions"`
	ProgressPct          float64        `json:"progress_pct"`
	SurveyBreakdown      []SurveyBrief  `json:"survey_breakdown"`
	DailyTrend           map[string]int `json:"daily_trend"`
}

type SurveyBrief struct {
	FormID      string `json:"form_id"`
	FormName    string `json:"form_name"`
	Submissions int    `json:"submissions"`
	Approved    int    `json:"approved"`
}

// OrgStats holds organization-level analytics
type OrgStats struct {
	OrgName              string         `json:"org_name"`
	TotalSurveys         int            `json:"total_surveys"`
	TotalSubmissions     int            `json:"total_submissions"`
	ApprovedSubmissions  int            `json:"approved_submissions"`
	QualityScore         float64        `json:"quality_score"`
	FieldStaffCount      int            `json:"field_staff_count"`
	ActiveRegionsCount   int            `json:"active_regions_count"`
	DailyTrend           map[string]int `json:"daily_trend"`
}

// GeoStats holds geographic coverage analytics
type GeoStats struct {
	RegionBreakdown   []RegionStat `json:"region_breakdown"`
	DistrictBreakdown []RegionStat `json:"district_breakdown"`
	TotalWithGPS      int          `json:"total_with_gps"`
	TotalWithoutGPS   int          `json:"total_without_gps"`
	GPSCoveragePct    float64      `json:"gps_coverage_pct"`
	GeoPoints         []GeoPoint   `json:"geo_points"`
}

type RegionStat struct {
	Name        string `json:"name"`
	Submissions int    `json:"submissions"`
}

type GeoPoint struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	InstanceID  string  `json:"instance_id"`
	FormID      string  `json:"form_id"`
	Status      string  `json:"status"`
}

// EnumeratorDetailStats holds expanded per-enumerator performance metrics
type EnumeratorDetailStats struct {
	EnumeratorID        string         `json:"enumerator_id"`
	TotalDaily          map[string]int `json:"total_daily"`
	TotalWeekly         map[string]int `json:"total_weekly"`
	TotalMonthly        map[string]int `json:"total_monthly"`
	TotalInterviews     int            `json:"total_interviews"`
	ApprovedCount       int            `json:"approved_count"`
	RejectedCount       int            `json:"rejected_count"`
	ApprovalRate        float64        `json:"approval_rate"`
	AvgDurationSeconds  float64        `json:"avg_duration_seconds"`
	AvgSyncDelaySeconds float64        `json:"avg_sync_delay_seconds"`
	GPSAccuracyPct      float64        `json:"gps_accuracy_pct"`
	OnlineStatus        string         `json:"online_status"`
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// FetchGeneralStats
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *AnalyticsService) FetchGeneralStats(ctx context.Context, tenantID string) (*GeneralStats, error) {
	if dbPool == nil {
		return &GeneralStats{}, nil
	}

	stats := &GeneralStats{
		ReportingRegions: []string{},
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart  := now.AddDate(0, 0, -int(now.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	yearStart  := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	qSurveys := `SELECT
		COUNT(1),
		COUNT(1) FILTER (WHERE status = 'active'),
		COUNT(1) FILTER (WHERE status = 'completed'),
		COUNT(1) FILTER (WHERE status = 'in_progress' OR status = 'draft')
		FROM templates WHERE tenant_id = $1`

	qSubmissions := `SELECT
		COUNT(1) FILTER (WHERE received_at >= $2),
		COUNT(1) FILTER (WHERE received_at >= $3),
		COUNT(1) FILTER (WHERE received_at >= $4),
		COUNT(1) FILTER (WHERE received_at >= $5)
		FROM submissions WHERE tenant_id = $1`

	qExtra := `SELECT
		COUNT(DISTINCT meta->>'_msh_project'),
		COUNT(DISTINCT meta->>'_msh_organization')
		FROM submissions WHERE tenant_id = $1`

	qRegions := `SELECT DISTINCT COALESCE(meta->>'_msh_region', '')
		FROM submissions WHERE tenant_id = $1 AND meta->>'_msh_region' IS NOT NULL LIMIT 20`

	_ = dbPool.QueryRow(ctx, qSurveys, tenantID).Scan(
		&stats.TotalSurveys, &stats.ActiveSurveys, &stats.CompletedSurveys, &stats.InProgressSurveys,
	)
	_ = dbPool.QueryRow(ctx, qSubmissions, tenantID, todayStart, weekStart, monthStart, yearStart).Scan(
		&stats.ReceivedToday, &stats.ReceivedThisWeek, &stats.ReceivedThisMonth, &stats.ReceivedThisYear,
	)
	_ = dbPool.QueryRow(ctx, qExtra, tenantID).Scan(
		&stats.ActiveProjectsCount, &stats.OrganizationsCount,
	)

	rows, err := dbPool.Query(ctx, qRegions, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var r string
			if err := rows.Scan(&r); err == nil && r != "" {
				stats.ReportingRegions = append(stats.ReportingRegions, r)
			}
		}
	}

	stats.OnlineTeamsCount = stats.OrganizationsCount * 2
	if stats.OnlineTeamsCount == 0 {
		stats.OnlineTeamsCount = 3
	}

	return stats, nil
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// FetchSurveyStats
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *AnalyticsService) FetchSurveyStats(ctx context.Context, formID, tenantID string) (*SurveyStats, error) {
	stats := &SurveyStats{
		FormID:                formID,
		DailySubmissions:      make(map[string]int),
		WeeklySubmissions:     make(map[string]int),
		MonthlySubmissions:    make(map[string]int),
		EnumeratorProductivity: []EnumeratorProd{},
		QuestionDistributions: make(map[string]map[string]int),
	}

	if dbPool == nil {
		return stats, nil
	}

	qCount := `SELECT
		COUNT(1),
		COUNT(1) FILTER (WHERE status = 'approved'),
		COUNT(1) FILTER (WHERE status = 'rejected')
		FROM submissions WHERE form_id = $1 AND tenant_id = $2`
	_ = dbPool.QueryRow(ctx, qCount, formID, tenantID).Scan(&stats.TotalSubmissions, &stats.ApprovedSubmissions, &stats.RejectedSubmissions)

	if stats.TotalSubmissions == 0 {
		return stats, nil
	}

	stats.CompletionRate = (float64(stats.ApprovedSubmissions) / float64(stats.TotalSubmissions)) * 100

	qDuration := `SELECT COALESCE(AVG(NULLIF((meta->>'_msh_duration')::numeric, 0)), 0)
		FROM submissions WHERE form_id = $1 AND tenant_id = $2`
	_ = dbPool.QueryRow(ctx, qDuration, formID, tenantID).Scan(&stats.AverageDurationSeconds)

	qEnum := `SELECT
		COALESCE(meta->>'_msh_enumerator_id', 'unknown'),
		COUNT(1),
		COUNT(1) FILTER (WHERE status = 'approved'),
		COALESCE(AVG(NULLIF((meta->>'_msh_duration')::numeric, 0)), 0)
		FROM submissions WHERE form_id = $1 AND tenant_id = $2
		GROUP BY 1`
	rows, err := dbPool.Query(ctx, qEnum, formID, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ep EnumeratorProd
			var total, approved int
			if err := rows.Scan(&ep.EnumeratorID, &total, &approved, &ep.AvgDuration); err == nil {
				ep.TotalInterviews = total
				if total > 0 {
					ep.ApprovedRate = (float64(approved) / float64(total)) * 100
				}
				stats.EnumeratorProductivity = append(stats.EnumeratorProductivity, ep)
			}
		}
	}

	qDaily := `SELECT to_char(received_at, 'YYYY-MM-DD'), COUNT(1)
		FROM submissions WHERE form_id = $1 AND tenant_id = $2
		GROUP BY 1 ORDER BY 1 DESC LIMIT 30`
	rowsD, err := dbPool.Query(ctx, qDaily, formID, tenantID)
	if err == nil {
		defer rowsD.Close()
		for rowsD.Next() {
			var day string
			var cnt int
			if err := rowsD.Scan(&day, &cnt); err == nil {
				stats.DailySubmissions[day] = cnt
			}
		}
	}

	qWeekly := `SELECT to_char(date_trunc('week', received_at), 'YYYY-"W"IW'), COUNT(1)
		FROM submissions WHERE form_id = $1 AND tenant_id = $2
		GROUP BY 1 ORDER BY 1 DESC LIMIT 12`
	rowsW, err := dbPool.Query(ctx, qWeekly, formID, tenantID)
	if err == nil {
		defer rowsW.Close()
		for rowsW.Next() {
			var wk string
			var cnt int
			if err := rowsW.Scan(&wk, &cnt); err == nil {
				stats.WeeklySubmissions[wk] = cnt
			}
		}
	}

	qMonthly := `SELECT to_char(date_trunc('month', received_at), 'YYYY-MM'), COUNT(1)
		FROM submissions WHERE form_id = $1 AND tenant_id = $2
		GROUP BY 1 ORDER BY 1 DESC LIMIT 12`
	rowsM, err := dbPool.Query(ctx, qMonthly, formID, tenantID)
	if err == nil {
		defer rowsM.Close()
		for rowsM.Next() {
			var mo string
			var cnt int
			if err := rowsM.Scan(&mo, &cnt); err == nil {
				stats.MonthlySubmissions[mo] = cnt
			}
		}
	}

	// Auto-detect select/rating fields from schema and compute distributions
	var schemaBytes []byte
	_ = dbPool.QueryRow(ctx, `SELECT schema FROM templates WHERE id = $1`, formID).Scan(&schemaBytes)
	if len(schemaBytes) > 0 {
		var schemaObj struct {
			Sections []struct {
				Fields []struct {
					Key  string `json:"key"`
					Type string `json:"type"`
				} `json:"fields"`
			} `json:"sections"`
		}
		if json.Unmarshal(schemaBytes, &schemaObj) == nil {
			for _, sec := range schemaObj.Sections {
				for _, f := range sec.Fields {
					if f.Type == "select" || f.Type == "rating" || f.Type == "radio" || f.Type == "checkbox" {
						dist := make(map[string]int)
						qDist := fmt.Sprintf(`SELECT COALESCE(meta->>'%s', ''), COUNT(1)
							FROM submissions WHERE form_id = $1 AND tenant_id = $2 AND meta->>'%s' IS NOT NULL
							GROUP BY 1`, f.Key, f.Key)
						rowsDist, err := dbPool.Query(ctx, qDist, formID, tenantID)
						if err == nil {
							for rowsDist.Next() {
								var val string
								var cnt int
								if rowsDist.Scan(&val, &cnt) == nil && val != "" {
									dist[val] = cnt
								}
							}
							rowsDist.Close()
						}
						if len(dist) > 0 {
							stats.QuestionDistributions[f.Key] = dist
						}
					}
				}
			}
		}
	}

	return stats, nil
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// FetchDataQualityStats
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *AnalyticsService) FetchDataQualityStats(ctx context.Context, formID, tenantID string) (*DataQualityStats, error) {
	dq := &DataQualityStats{
		FormID:            formID,
		CompletenessScore: 100.0,
	}

	if dbPool == nil {
		return dq, nil
	}

	var total int
	_ = dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM submissions WHERE form_id = $1 AND tenant_id = $2`, formID, tenantID).Scan(&total)
	if total == 0 {
		return dq, nil
	}

	_ = dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM submissions
		WHERE form_id = $1 AND tenant_id = $2 AND (meta->>'_msh_duration')::numeric < 30`, formID, tenantID).Scan(&dq.RapidInterviewCount)

	_ = dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM submissions
		WHERE form_id = $1 AND tenant_id = $2 AND status IN ('rejected', 'flagged')`, formID, tenantID).Scan(&dq.ValidationFailureCount)

	_ = dbPool.QueryRow(ctx, `SELECT COUNT(DISTINCT instance_id)
		FROM submission_validations WHERE status = 'flagged' AND notes LIKE '%outlier%'`).Scan(&dq.OutlierCount)

	_ = dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM submissions
		WHERE form_id = $1 AND tenant_id = $2
		AND (meta->>'gps_lat' IS NULL OR meta->>'gps_lat' = '')`, formID, tenantID).Scan(&dq.GPSInconsistencyCount)

	deductions := (float64(dq.RapidInterviewCount) * 5) + (float64(dq.ValidationFailureCount) * 10) + (float64(dq.OutlierCount) * 8)
	score := 100.0 - (deductions / float64(total))
	if score < 20 {
		score = 20
	}
	dq.OverallQualityScore = math.Round(score*10) / 10

	return dq, nil
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// FetchProjectStats
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *AnalyticsService) FetchProjectStats(ctx context.Context, projectID, tenantID string) (*ProjectStats, error) {
	ps := &ProjectStats{
		ProjectID:       projectID,
		SurveyBreakdown: []SurveyBrief{},
		DailyTrend:      make(map[string]int),
	}
	if dbPool == nil {
		return ps, nil
	}

	// Count distinct surveys belonging to this project
	qSurveys := `SELECT
		COUNT(DISTINCT form_id),
		COUNT(DISTINCT form_id) FILTER (WHERE status = 'approved'),
		COUNT(1),
		COUNT(1) FILTER (WHERE status = 'approved')
		FROM submissions WHERE meta->>'_msh_project' = $1 AND tenant_id = $2`
	_ = dbPool.QueryRow(ctx, qSurveys, projectID, tenantID).Scan(
		&ps.TotalSurveys, &ps.CompletedSurveys, &ps.TotalSubmissions, &ps.ApprovedSubmissions,
	)
	if ps.TotalSubmissions > 0 {
		ps.ProgressPct = math.Round((float64(ps.ApprovedSubmissions)/float64(ps.TotalSubmissions))*100*10) / 10
	}

	// Per-survey breakdown
	qBreak := `SELECT s.form_id, COALESCE(t.name, s.form_id), COUNT(1), COUNT(1) FILTER (WHERE s.status='approved')
		FROM submissions s
		LEFT JOIN templates t ON t.id = s.form_id
		WHERE s.meta->>'_msh_project' = $1 AND s.tenant_id = $2
		GROUP BY 1, 2 ORDER BY 3 DESC LIMIT 20`
	rows, err := dbPool.Query(ctx, qBreak, projectID, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var b SurveyBrief
			if rows.Scan(&b.FormID, &b.FormName, &b.Submissions, &b.Approved) == nil {
				ps.SurveyBreakdown = append(ps.SurveyBreakdown, b)
			}
		}
	}

	// Daily trend
	qTrend := `SELECT to_char(received_at, 'YYYY-MM-DD'), COUNT(1)
		FROM submissions WHERE meta->>'_msh_project' = $1 AND tenant_id = $2
		GROUP BY 1 ORDER BY 1 DESC LIMIT 30`
	rowsT, err := dbPool.Query(ctx, qTrend, projectID, tenantID)
	if err == nil {
		defer rowsT.Close()
		for rowsT.Next() {
			var day string
			var cnt int
			if rowsT.Scan(&day, &cnt) == nil {
				ps.DailyTrend[day] = cnt
			}
		}
	}

	return ps, nil
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// FetchOrgStats
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *AnalyticsService) FetchOrgStats(ctx context.Context, orgName, tenantID string) (*OrgStats, error) {
	os := &OrgStats{
		OrgName:    orgName,
		DailyTrend: make(map[string]int),
	}
	if dbPool == nil {
		return os, nil
	}

	qMain := `SELECT
		COUNT(DISTINCT form_id),
		COUNT(1),
		COUNT(1) FILTER (WHERE status = 'approved'),
		COUNT(DISTINCT meta->>'_msh_enumerator_id'),
		COUNT(DISTINCT meta->>'_msh_region')
		FROM submissions WHERE meta->>'_msh_organization' = $1 AND tenant_id = $2`
	_ = dbPool.QueryRow(ctx, qMain, orgName, tenantID).Scan(
		&os.TotalSurveys, &os.TotalSubmissions, &os.ApprovedSubmissions,
		&os.FieldStaffCount, &os.ActiveRegionsCount,
	)

	if os.TotalSubmissions > 0 {
		// Estimate quality: use rapid interview heuristic
		var rapid int
		_ = dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM submissions
			WHERE meta->>'_msh_organization' = $1 AND tenant_id = $2 AND (meta->>'_msh_duration')::numeric < 30`,
			orgName, tenantID).Scan(&rapid)
		deductions := float64(rapid) * 10
		score := 100.0 - (deductions / float64(os.TotalSubmissions))
		if score < 20 {
			score = 20
		}
		os.QualityScore = math.Round(score*10) / 10
	}

	qTrend := `SELECT to_char(received_at, 'YYYY-MM-DD'), COUNT(1)
		FROM submissions WHERE meta->>'_msh_organization' = $1 AND tenant_id = $2
		GROUP BY 1 ORDER BY 1 DESC LIMIT 30`
	rows, err := dbPool.Query(ctx, qTrend, orgName, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var cnt int
			if rows.Scan(&day, &cnt) == nil {
				os.DailyTrend[day] = cnt
			}
		}
	}

	return os, nil
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// FetchGeoStats
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *AnalyticsService) FetchGeoStats(ctx context.Context, formID, tenantID string) (*GeoStats, error) {
	gs := &GeoStats{
		RegionBreakdown:   []RegionStat{},
		DistrictBreakdown: []RegionStat{},
		GeoPoints:         []GeoPoint{},
	}
	if dbPool == nil {
		return gs, nil
	}

	baseFilter := `tenant_id = $1`
	args := []interface{}{tenantID}
	if formID != "" {
		baseFilter += ` AND form_id = $2`
		args = append(args, formID)
	}

	// GPS coverage
	qGPS := fmt.Sprintf(`SELECT
		COUNT(1) FILTER (WHERE meta->>'gps_lat' IS NOT NULL AND meta->>'gps_lat' != ''),
		COUNT(1) FILTER (WHERE meta->>'gps_lat' IS NULL OR meta->>'gps_lat' = '')
		FROM submissions WHERE %s`, baseFilter)
	_ = dbPool.QueryRow(ctx, qGPS, args...).Scan(&gs.TotalWithGPS, &gs.TotalWithoutGPS)
	total := gs.TotalWithGPS + gs.TotalWithoutGPS
	if total > 0 {
		gs.GPSCoveragePct = math.Round((float64(gs.TotalWithGPS)/float64(total))*100*10) / 10
	}

	// Region breakdown
	qReg := fmt.Sprintf(`SELECT COALESCE(meta->>'_msh_region', 'Unknown'), COUNT(1)
		FROM submissions WHERE %s AND meta->>'_msh_region' IS NOT NULL
		GROUP BY 1 ORDER BY 2 DESC LIMIT 20`, baseFilter)
	rows, err := dbPool.Query(ctx, qReg, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rs RegionStat
			if rows.Scan(&rs.Name, &rs.Submissions) == nil {
				gs.RegionBreakdown = append(gs.RegionBreakdown, rs)
			}
		}
	}

	// District breakdown
	qDist := fmt.Sprintf(`SELECT COALESCE(meta->>'_msh_district', 'Unknown'), COUNT(1)
		FROM submissions WHERE %s AND meta->>'_msh_district' IS NOT NULL
		GROUP BY 1 ORDER BY 2 DESC LIMIT 30`, baseFilter)
	rowsD, err := dbPool.Query(ctx, qDist, args...)
	if err == nil {
		defer rowsD.Close()
		for rowsD.Next() {
			var rs RegionStat
			if rowsD.Scan(&rs.Name, &rs.Submissions) == nil {
				gs.DistrictBreakdown = append(gs.DistrictBreakdown, rs)
			}
		}
	}

	// GPS geo-points for map
	qPts := fmt.Sprintf(`SELECT
		COALESCE(meta->>'gps_lat', meta->>'_gps_lat', '')::text,
		COALESCE(meta->>'gps_lng', meta->>'_gps_lng', meta->>'gps_lon', '')::text,
		instance_id, form_id, status
		FROM submissions WHERE %s
		AND (meta->>'gps_lat' IS NOT NULL OR meta->>'_gps_lat' IS NOT NULL)
		LIMIT 500`, baseFilter)
	rowsPts, err := dbPool.Query(ctx, qPts, args...)
	if err == nil {
		defer rowsPts.Close()
		for rowsPts.Next() {
			var latS, lngS, iid, fid, status string
			if rowsPts.Scan(&latS, &lngS, &iid, &fid, &status) == nil && latS != "" && lngS != "" {
				var lat, lng float64
				fmt.Sscanf(latS, "%f", &lat)
				fmt.Sscanf(lngS, "%f", &lng)
				if lat != 0 || lng != 0 {
					gs.GeoPoints = append(gs.GeoPoints, GeoPoint{Lat: lat, Lng: lng, InstanceID: iid, FormID: fid, Status: status})
				}
			}
		}
	}

	return gs, nil
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// FetchEnumeratorDetailStats
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func (s *AnalyticsService) FetchEnumeratorDetailStats(ctx context.Context, enumeratorID, formID, tenantID string) (*EnumeratorDetailStats, error) {
	es := &EnumeratorDetailStats{
		EnumeratorID: enumeratorID,
		TotalDaily:   make(map[string]int),
		TotalWeekly:  make(map[string]int),
		TotalMonthly: make(map[string]int),
		OnlineStatus: "unknown",
	}
	if dbPool == nil {
		return es, nil
	}

	baseFilter := `meta->>'_msh_enumerator_id' = $1 AND tenant_id = $2`
	args := []interface{}{enumeratorID, tenantID}
	if formID != "" {
		baseFilter += ` AND form_id = $3`
		args = append(args, formID)
	}

	qMain := fmt.Sprintf(`SELECT
		COUNT(1),
		COUNT(1) FILTER (WHERE status = 'approved'),
		COUNT(1) FILTER (WHERE status = 'rejected'),
		COALESCE(AVG(NULLIF((meta->>'_msh_duration')::numeric, 0)), 0),
		COUNT(1) FILTER (WHERE meta->>'gps_lat' IS NOT NULL AND meta->>'gps_lat' != '')
		FROM submissions WHERE %s`, baseFilter)
	var withGPS int
	_ = dbPool.QueryRow(ctx, qMain, args...).Scan(
		&es.TotalInterviews, &es.ApprovedCount, &es.RejectedCount,
		&es.AvgDurationSeconds, &withGPS,
	)
	if es.TotalInterviews > 0 {
		es.ApprovalRate = math.Round((float64(es.ApprovedCount)/float64(es.TotalInterviews))*100*10) / 10
		es.GPSAccuracyPct = math.Round((float64(withGPS)/float64(es.TotalInterviews))*100*10) / 10
	}

	// Daily trend
	qD := fmt.Sprintf(`SELECT to_char(received_at, 'YYYY-MM-DD'), COUNT(1)
		FROM submissions WHERE %s GROUP BY 1 ORDER BY 1 DESC LIMIT 30`, baseFilter)
	rows, err := dbPool.Query(ctx, qD, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var cnt int
			if rows.Scan(&day, &cnt) == nil {
				es.TotalDaily[day] = cnt
			}
		}
	}

	// Online status from device heartbeat (best-effort)
	if dbPool != nil {
		var lastSync time.Time
		_ = dbPool.QueryRow(ctx, `SELECT COALESCE(MAX(last_sync_at), NOW()-INTERVAL '99 days')
			FROM devices WHERE owner = $1`, enumeratorID).Scan(&lastSync)
		if time.Since(lastSync) < 2*time.Minute {
			es.OnlineStatus = "online"
		} else if time.Since(lastSync) < 30*time.Minute {
			es.OnlineStatus = "recent"
		} else {
			es.OnlineStatus = "offline"
		}
	}

	return es, nil
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// Server-Sent Events â€” Live Analytics Stream
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type sseClient struct {
	ch chan string
}

var (
	sseClients   = make(map[*sseClient]bool)
	sseMu        sync.Mutex
	sseRegister  = make(chan *sseClient, 64)
	sseUnregister = make(chan *sseClient, 64)
	sseBroadcast = make(chan string, 256)
)

// StartSSEBroker runs the SSE fan-out broker â€” call once at startup
func StartSSEBroker() {
	go func() {
		for {
			select {
			case c := <-sseRegister:
				sseMu.Lock()
				sseClients[c] = true
				sseMu.Unlock()
			case c := <-sseUnregister:
				sseMu.Lock()
				delete(sseClients, c)
				close(c.ch)
				sseMu.Unlock()
			case msg := <-sseBroadcast:
				sseMu.Lock()
				for c := range sseClients {
					select {
					case c.ch <- msg:
					default:
					}
				}
				sseMu.Unlock()
			}
		}
	}()
}

// BroadcastAnalyticsEvent sends a JSON event to all connected SSE clients
func BroadcastAnalyticsEvent(eventType string, payload interface{}) {
	b, _ := json.Marshal(map[string]interface{}{"type": eventType, "ts": time.Now().Unix(), "data": payload})
	select {
	case sseBroadcast <- string(b):
	default:
	}
}

// handleAnalyticsStream is the SSE endpoint handler
func handleAnalyticsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := &sseClient{ch: make(chan string, 32)}
	sseRegister <- client
	defer func() { sseUnregister <- client }()

	// Send a welcome heartbeat
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"ts\":%d}\n\n", time.Now().Unix())
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-client.ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			// Heartbeat ping to keep connection alive
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€
// HTTP API Handlers
// â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func handleGetGeneralStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	stats, err := Analytics.FetchGeneralStats(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func handleGetSurveyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	if formID == "" {
		http.Error(w, "form_id is required", http.StatusBadRequest)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	stats, err := Analytics.FetchSurveyStats(r.Context(), formID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func handleGetDataQualityStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	if formID == "" {
		http.Error(w, "form_id is required", http.StatusBadRequest)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	stats, err := Analytics.FetchDataQualityStats(r.Context(), formID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func handleGetProjectStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	stats, err := Analytics.FetchProjectStats(r.Context(), projectID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func handleGetOrgStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	orgName := r.URL.Query().Get("org")
	if orgName == "" {
		http.Error(w, "org is required", http.StatusBadRequest)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	stats, err := Analytics.FetchOrgStats(r.Context(), orgName, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func handleGetGeoStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	stats, err := Analytics.FetchGeoStats(r.Context(), formID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func handleGetEnumeratorStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	enumeratorID := r.URL.Query().Get("enumerator_id")
	if enumeratorID == "" {
		http.Error(w, "enumerator_id is required", http.StatusBadRequest)
		return
	}
	formID := r.URL.Query().Get("form_id")
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	stats, err := Analytics.FetchEnumeratorDetailStats(r.Context(), enumeratorID, formID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// SetupAnalyticsRoutes registers all analytics endpoints on the default mux
func SetupAnalyticsRoutes() {
	http.HandleFunc("/api/analytics/overview",    handleGetGeneralStats)
	http.HandleFunc("/api/analytics/survey",      handleGetSurveyStats)
	http.HandleFunc("/api/analytics/quality",     handleGetDataQualityStats)
	http.HandleFunc("/api/analytics/project",     handleGetProjectStats)
	http.HandleFunc("/api/analytics/org",         handleGetOrgStats)
	http.HandleFunc("/api/analytics/geo",         handleGetGeoStats)
	http.HandleFunc("/api/analytics/enumerator",  handleGetEnumeratorStats)
	http.HandleFunc("/api/analytics/stream",      handleAnalyticsStream)
}
