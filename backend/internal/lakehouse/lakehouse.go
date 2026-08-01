package lakehouse

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type EventRecord struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Source    string                 `json:"source"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
}

type AnomalyAlert struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	TenantID    string    `json:"tenant_id"`
	MetricName  string    `json:"metric_name"`
	Dataset     string    `json:"dataset"`
	Value       float64   `json:"value"`
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"std_dev"`
	ZScore      float64   `json:"z_score"`
	SigmaScore  float64   `json:"sigma_score"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
	NotebookURL string    `json:"notebook_url"`
}

type StorageEngine struct {
	mu            sync.RWMutex
	records       []EventRecord
	tenantRecords map[string][]EventRecord
	anomalies     []AnomalyAlert
}

func NewStorageEngine() *StorageEngine {
	se := &StorageEngine{
		records:       make([]EventRecord, 0),
		tenantRecords: make(map[string][]EventRecord),
		anomalies:     make([]AnomalyAlert, 0),
	}
	se.seedDemoData()
	return se
}

func (se *StorageEngine) seedDemoData() {
	tenants := []string{"tenant-alpha", "tenant-beta"}
	now := time.Now()

	for _, t := range tenants {
		for i := 0; i < 50; i++ {
			val := 100.0 + float64(i%10) + float64(i*2%5)
			rec := EventRecord{
				ID:        fmt.Sprintf("rec-%s-%d", t, i),
				TenantID:  t,
				Source:    "webhook_sensor",
				Payload:   map[string]interface{}{"temperature": val, "status": "active"},
				Timestamp: now.Add(-time.Duration(50-i) * time.Second),
				Value:     val,
			}
			se.Ingest(rec)
		}
	}
}

func (se *StorageEngine) Ingest(rec EventRecord) AnomalyAlert {
	se.mu.Lock()
	defer se.mu.Unlock()

	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}

	se.records = append(se.records, rec)
	se.tenantRecords[rec.TenantID] = append(se.tenantRecords[rec.TenantID], rec)

	// Anomaly detection (+/- 3 sigma thresholding)
	var alert AnomalyAlert
	tenantList := se.tenantRecords[rec.TenantID]
	if len(tenantList) >= 10 {
		var sum float64
		for _, r := range tenantList {
			sum += r.Value
		}
		mean := sum / float64(len(tenantList))

		var varianceSum float64
		for _, r := range tenantList {
			varianceSum += math.Pow(r.Value-mean, 2)
		}
		stdDev := math.Sqrt(varianceSum / float64(len(tenantList)))

		if stdDev > 0 {
			zScore := (rec.Value - mean) / stdDev
			if math.Abs(zScore) >= 3.0 {
				severity := "HIGH"
				if math.Abs(zScore) >= 3.5 {
					severity = "CRITICAL"
				}
				alert = AnomalyAlert{
					ID:          fmt.Sprintf("alert_%d", rec.Timestamp.UnixNano()),
					EventID:     rec.ID,
					TenantID:    rec.TenantID,
					MetricName:  rec.Source,
					Dataset:     "live_telemetry_stream",
					Value:       rec.Value,
					Mean:        mean,
					StdDev:      stdDev,
					ZScore:      zScore,
					SigmaScore:  math.Round(zScore*100) / 100,
					Severity:    severity,
					Timestamp:   rec.Timestamp,
					Message:     fmt.Sprintf("3σ Anomaly: %s = %.2f (σ=%.2f, μ=%.2f)", rec.Source, rec.Value, zScore, mean),
					NotebookURL: "/notebook",
				}
				se.anomalies = append(se.anomalies, alert)
			}
		}
	}

	return alert
}

func (se *StorageEngine) QueryTenantData(tenantID string, limit int) []EventRecord {
	se.mu.RLock()
	defer se.mu.RUnlock()

	records := se.tenantRecords[tenantID]
	if limit > 0 && len(records) > limit {
		return records[len(records)-limit:]
	}
	return records
}

func (se *StorageEngine) GetAnomalies(tenantID string) []AnomalyAlert {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var result []AnomalyAlert
	for _, a := range se.anomalies {
		if tenantID == "" || a.TenantID == tenantID {
			result = append(result, a)
		}
	}
	return result
}

func (se *StorageEngine) GetStats(tenantID string) map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	recs := se.tenantRecords[tenantID]
	count := len(recs)
	var totalVal float64
	for _, r := range recs {
		totalVal += r.Value
	}

	avg := 0.0
	if count > 0 {
		avg = totalVal / float64(count)
	}

	return map[string]interface{}{
		"tenant_id":        tenantID,
		"total_records":    count,
		"avg_value":        avg,
		"anomalies_count":  len(se.GetAnomalies(tenantID)),
		"last_ingested_at": time.Now().Format(time.RFC3339),
	}
}
