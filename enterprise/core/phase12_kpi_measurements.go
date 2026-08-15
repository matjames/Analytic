package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — KPI MEASUREMENTS
//
// Append-only measurement history with indexed measurements. estimated=true
// produces ESTIMATED status. Recorded measurements refresh the KPI actual
// value and data status and emit a kpi.measured event for the pipeline.
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"fmt"
	"log"
	"time"
)

func (s *Phase12Store) RecordMeasurement(tenantID, kpiID string, value float64, unit, source string, estimated bool, measuredAt time.Time) error {
	key := objectKey(tenantID, kpiID)
	s.mu.RLock()
	k, ok := s.kpis[key]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("kpi %q not found for tenant", kpiID)
	}
	if measuredAt.IsZero() {
		measuredAt = utcNow()
	}
	status := "ACTUAL"
	if estimated {
		status = "ESTIMATED"
	}
	m := &KPIMeasurement{
		KpiID:      kpiID,
		TenantID:   tenantID,
		Value:      value,
		Unit:       unit,
		Source:     source,
		Estimated:  estimated,
		Status:     status,
		MeasuredAt: measuredAt,
		RecordedAt: utcNow(),
	}
	s.mu.Lock()
	s.nextMeasID++
	m.ID = s.nextMeasID
	if s.measurements[tenantID] == nil {
		s.measurements[tenantID] = []*KPIMeasurement{}
	}
	s.measurements[tenantID] = append(s.measurements[tenantID], m)

	k.ActualValue = &value
	k.Unit = unit
	k.Source = source
	k.LastMeasuredAt = &measuredAt
	k.DataStatus = k.computeDataStatus()
	s.mu.Unlock()

	persistPhase12Measurement(m)
	persistPhase12KPI(k)

	publishEvent(DomainEvent{
		EventType:  "kpi.measured",
		Source:     "intelligence_kpi",
		ObjectType: "kpi",
		ObjectID:   kpiID,
		TenantID:   tenantID,
		Payload: map[string]interface{}{
			"value":       value,
			"unit":        unit,
			"source":      source,
			"estimated":   estimated,
			"data_status": string(k.DataStatus),
		},
	})
	return nil
}

func (s *Phase12Store) ListMeasurements(tenantID, kpiID string, limit int) []*KPIMeasurement {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*KPIMeasurement
	for _, m := range s.measurements[tenantID] {
		if m.KpiID == kpiID {
			out = append(out, m)
		}
	}
	for i := 0; i < len(out)/2; i++ {
		j := len(out) - 1 - i
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func persistPhase12Measurement(m *KPIMeasurement) {
	if dbPool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO kpi_measurements
		(kpi_id, tenant_id, value, unit, source, estimated, status, measured_at, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		m.KpiID, m.TenantID, m.Value, m.Unit, m.Source, m.Estimated, m.Status, m.MeasuredAt, m.RecordedAt)
	if err != nil {
		log.Printf("phase12: persist measurement failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}