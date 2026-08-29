package intelligence

import (
	"context"
	"math"
	"time"
)

// ConditionLevel represents the institutional health assessment level.
type ConditionLevel string

const (
	LevelOptimal   ConditionLevel = "OPTIMAL"
	LevelNominal   ConditionLevel = "NOMINAL"
	LevelAttention ConditionLevel = "ATTENTION"
	LevelElevated  ConditionLevel = "ELEVATED"
	LevelCritical  ConditionLevel = "CRITICAL"
	LevelEmergency ConditionLevel = "EMERGENCY"
)

// DomainHealthScore represents a single operational domain score (0.0 to 100.0).
type DomainHealthScore struct {
	DomainName    string  `json:"domain_name"`
	Score         float64 `json:"score"`         // 0 - 100
	Weight        float64 `json:"weight"`        // 0.0 - 1.0
	Status        string  `json:"status"`        // HEALTHY, WARNING, CRITICAL
	AnomalyCount  int     `json:"anomaly_count"`
	ActiveRiskCount int   `json:"active_risk_count"`
	Notes         string  `json:"notes,omitempty"`
}

// InstitutionalCondition is the aggregated composite state of the entire institution.
type InstitutionalCondition struct {
	Level           ConditionLevel      `json:"level"`
	CompositeScore  float64             `json:"composite_score"` // 0 - 100
	CalculatedAt    time.Time           `json:"calculated_at"`
	TenantID        string              `json:"tenant_id"`
	DomainScores    []DomainHealthScore `json:"domain_scores"`
	Recommendations []string            `json:"recommendations"`
	CriticalAlerts  []string            `json:"critical_alerts"`
}

// ConditionCalculator aggregates cross-domain signals to evaluate the institutional condition.
type ConditionCalculator struct{}

// NewConditionCalculator creates a new condition calculator.
func NewConditionCalculator() *ConditionCalculator {
	return &ConditionCalculator{}
}

// EvaluateCondition evaluates institutional domain health based on input metrics.
func (c *ConditionCalculator) EvaluateCondition(
	ctx context.Context,
	tenantID string,
	anomaliesCount int,
	activeRisksCount int,
	schemaQualityScore float64,
	kpiAttainmentRate float64,
	slaComplianceRate float64,
) *InstitutionalCondition {
	now := time.Now().UTC()

	// 1. Data Quality & Schema Integrity Domain (Weight: 0.25)
	dataScore := schemaQualityScore
	if dataScore <= 0 {
		dataScore = 92.0 // Default healthy baseline
	}
	dataStatus := "HEALTHY"
	if dataScore < 70.0 {
		dataStatus = "CRITICAL"
	} else if dataScore < 85.0 {
		dataStatus = "WARNING"
	}

	// 2. Statistical & Anomaly Domain (Weight: 0.25)
	statScore := math.Max(0, 100.0-float64(anomaliesCount)*8.0)
	statStatus := "HEALTHY"
	if anomaliesCount > 5 {
		statStatus = "CRITICAL"
	} else if anomaliesCount > 2 {
		statStatus = "WARNING"
	}

	// 3. Strategic Objectives & KPI Domain (Weight: 0.25)
	kpiScore := kpiAttainmentRate
	if kpiScore <= 0 {
		kpiScore = 88.5
	}
	kpiStatus := "HEALTHY"
	if kpiScore < 60.0 {
		kpiStatus = "CRITICAL"
	} else if kpiScore < 80.0 {
		kpiStatus = "WARNING"
	}

	// 4. Governance & Risk Domain (Weight: 0.25)
	riskScore := math.Max(0, 100.0-float64(activeRisksCount)*12.0)
	if slaComplianceRate > 0 {
		riskScore = (riskScore + slaComplianceRate) / 2.0
	}
	riskStatus := "HEALTHY"
	if activeRisksCount > 3 {
		riskStatus = "CRITICAL"
	} else if activeRisksCount > 0 {
		riskStatus = "WARNING"
	}

	domains := []DomainHealthScore{
		{
			DomainName:    "Data Quality & Schema Health",
			Score:         math.Round(dataScore*10) / 10,
			Weight:        0.25,
			Status:        dataStatus,
			Notes:         "Evaluates completeness, freshness, and schema drift across datasets.",
		},
		{
			DomainName:    "Statistical Integrity & Anomalies",
			Score:         math.Round(statScore*10) / 10,
			Weight:        0.25,
			Status:        statStatus,
			AnomalyCount:  anomaliesCount,
			Notes:         "Rolling 3-sigma telemetry and dataset outlier detection.",
		},
		{
			DomainName:    "Strategic Objectives & KPI Attainment",
			Score:         math.Round(kpiScore*10) / 10,
			Weight:        0.25,
			Status:        kpiStatus,
			Notes:         "Progress tracking against national and organizational milestones.",
		},
		{
			DomainName:      "Governance, Risk & Compliance",
			Score:           math.Round(riskScore*10) / 10,
			Weight:          0.25,
			Status:          riskStatus,
			ActiveRiskCount: activeRisksCount,
			Notes:           "Audit trails, institutional risks, and SLA adherence.",
		},
	}

	// Calculate weighted composite score
	composite := 0.0
	for _, d := range domains {
		composite += d.Score * d.Weight
	}
	composite = math.Round(composite*10) / 10

	// Determine condition level
	var level ConditionLevel
	recommendations := make([]string, 0)
	alerts := make([]string, 0)

	switch {
	case composite >= 90.0 && anomaliesCount == 0 && activeRisksCount == 0:
		level = LevelOptimal
		recommendations = append(recommendations, "All institutional domains operating within optimal parameters.")
	case composite >= 80.0:
		level = LevelNominal
		recommendations = append(recommendations, "Nominal operations. Routine monitoring recommended.")
	case composite >= 68.0:
		level = LevelAttention
		recommendations = append(recommendations, "Review active warning indicators in underperforming domains.")
		if anomaliesCount > 0 {
			alerts = append(alerts, "Detected statistical anomalies requiring analyst investigation.")
		}
	case composite >= 50.0:
		level = LevelElevated
		recommendations = append(recommendations, "Elevated institutional risk. Convene domain leads for remediation.")
		if activeRisksCount > 0 {
			alerts = append(alerts, "Unmitigated active governance risks present.")
		}
	case composite >= 30.0:
		level = LevelCritical
		recommendations = append(recommendations, "Immediate executive intervention required to avert mission failure.")
		alerts = append(alerts, "Multiple institutional domains in critical status.")
	default:
		level = LevelEmergency
		recommendations = append(recommendations, "Institutional continuity protocol activation required.")
		alerts = append(alerts, "Severe compromise across core data, risk, and operational pillars.")
	}

	return &InstitutionalCondition{
		Level:           level,
		CompositeScore:  composite,
		CalculatedAt:    now,
		TenantID:        tenantID,
		DomainScores:    domains,
		Recommendations: recommendations,
		CriticalAlerts:  alerts,
	}
}
