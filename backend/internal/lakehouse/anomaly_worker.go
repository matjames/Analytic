package lakehouse

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// AnomalyDetector is a proactive 3-sigma monitoring engine.
// It maintains an in-memory alert ring buffer accessible to the HTTP layer.
// AnomalyAlert is defined in lakehouse.go (same package).

type AnomalyDetector struct {
	mu     sync.RWMutex
	alerts []AnomalyAlert
}

var globalDetector *AnomalyDetector

// GetAnomalyDetector returns the singleton detector instance.
func GetAnomalyDetector() *AnomalyDetector {
	if globalDetector == nil {
		globalDetector = &AnomalyDetector{
			alerts: make([]AnomalyAlert, 0),
		}
	}
	return globalDetector
}

// EvaluateDataPoint calculates z-score against the provided history and
// auto-creates an alert if the value exceeds ±2.5σ.
func (ad *AnomalyDetector) EvaluateDataPoint(tenantID, dataset, metricName string, val float64, history []float64) *AnomalyAlert {
	if len(history) < 2 {
		return nil
	}

	var sum float64
	for _, v := range history {
		sum += v
	}
	mean := sum / float64(len(history))

	var varianceSum float64
	for _, v := range history {
		varianceSum += math.Pow(v-mean, 2)
	}
	stdDev := math.Sqrt(varianceSum / float64(len(history)))
	if stdDev == 0 {
		return nil
	}

	sigmaScore := (val - mean) / stdDev

	if math.Abs(sigmaScore) >= 2.5 {
		severity := "HIGH"
		if math.Abs(sigmaScore) >= 3.5 {
			severity = "CRITICAL"
		}

		alert := AnomalyAlert{
			ID:          fmt.Sprintf("alert_%d", time.Now().UnixNano()),
			EventID:     fmt.Sprintf("evt_%s_%d", dataset, time.Now().UnixNano()),
			TenantID:    tenantID,
			MetricName:  metricName,
			Dataset:     dataset,
			Value:       val,
			Mean:        math.Round(mean*100) / 100,
			StdDev:      math.Round(stdDev*100) / 100,
			ZScore:      math.Round(sigmaScore*100) / 100,
			SigmaScore:  math.Round(sigmaScore*100) / 100,
			Severity:    severity,
			Message:     fmt.Sprintf("3σ Anomaly Detected: %s = %.2f (%.2fσ from μ=%.2f)", metricName, val, sigmaScore, mean),
			NotebookURL: fmt.Sprintf("/notebook?dataset=%s", dataset),
			Timestamp:   time.Now().UTC(),
		}

		ad.mu.Lock()
		ad.alerts = append([]AnomalyAlert{alert}, ad.alerts...)
		if len(ad.alerts) > 50 {
			ad.alerts = ad.alerts[:50]
		}
		ad.mu.Unlock()

		return &alert
	}

	return nil
}

// GetAlerts returns all recent alerts for a given tenantID.
func (ad *AnomalyDetector) GetAlerts(tenantID string) []AnomalyAlert {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	if tenantID == "" || tenantID == "all" {
		result := make([]AnomalyAlert, len(ad.alerts))
		copy(result, ad.alerts)
		return result
	}

	filtered := make([]AnomalyAlert, 0)
	for _, a := range ad.alerts {
		if a.TenantID == tenantID {
			filtered = append(filtered, a)
		}
	}
	return filtered
}
