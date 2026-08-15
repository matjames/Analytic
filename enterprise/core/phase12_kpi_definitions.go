package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — KPI DEFINITIONS
//
// KPI data provenance quality is tracked via KpiDataStatus:
// ACTUAL | ESTIMATED | STALE | MISSING | INVALID. Stale data is never
// presented as current data.
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"context"
	"fmt"
	"log"
	"time"
)

var validMeasurementPeriods = map[string]bool{
	"DAILY": true, "WEEKLY": true, "MONTHLY": true, "QUARTERLY": true, "ANNUAL": true,
}

func (s *Phase12Store) UpsertKPI(k *KPI) error {
	if k == nil {
		return fmt.Errorf("kpi is nil")
	}
	if k.TenantID == "" || k.Name == "" {
		return fmt.Errorf("kpi requires tenant_id and name")
	}
	if !validMeasurementPeriods[k.MeasurementPeriod] {
		return fmt.Errorf("invalid measurement_period %q", k.MeasurementPeriod)
	}
	if k.Status == "" {
		k.Status = "ACTIVE"
	}
	if k.FreshnessWindowH <= 0 {
		k.FreshnessWindowH = 168 // 7 days
	}
	now := utcNow()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = now
	}
	k.UpdatedAt = now
	if k.ID == "" {
		k.ID = fmt.Sprintf("kpi_%d", time.Now().UnixNano())
	}
	if k.CanonicalID == "" {
		k.CanonicalID = "uoi_kpi_" + k.ID
	}

	k.DataStatus = k.computeDataStatus()
	s.mu.Lock()
	s.kpis[objectKey(k.TenantID, k.ID)] = k
	s.mu.Unlock()
	persistPhase12KPI(k)
	return nil
}

func (s *Phase12Store) ListKPIs(tenantID string) []*KPI {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*KPI
	for _, k := range s.kpis {
		if k.TenantID == tenantID {
			out = append(out, k)
		}
	}
	return out
}

// computeDataStatus derives ACTUAL/ESTIMATED/STALE/MISSING/INVALID from the
// last measurement. STALE is never presented as current data.
func (k *KPI) computeDataStatus() KpiDataStatus {
	if k.LastMeasuredAt == nil {
		return KpiDataMissing
	}
	window := time.Duration(k.FreshnessWindowH) * time.Hour
	if time.Since(*k.LastMeasuredAt) > window {
		return KpiDataStale
	}
	if k.ActualValue == nil {
		return KpiDataMissing
	}
	return KpiDataActual
}

func persistPhase12KPI(k *KPI) {
	if dbPool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := dbPool.ExecContext(ctx, `INSERT INTO kpis
		(id, tenant_id, objective_id, canonical_id, name, definition, owner, target,
		 measurement_period, actual_value, unit, source, status, data_status,
		 freshness_window_h, last_measured_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (tenant_id, canonical_id) DO UPDATE SET
			name = EXCLUDED.name, definition = EXCLUDED.definition, owner = EXCLUDED.owner,
			target = EXCLUDED.target, measurement_period = EXCLUDED.measurement_period,
			actual_value = EXCLUDED.actual_value, unit = EXCLUDED.unit, source = EXCLUDED.source,
			status = EXCLUDED.status, data_status = EXCLUDED.data_status,
			freshness_window_h = EXCLUDED.freshness_window_h, last_measured_at = EXCLUDED.last_measured_at,
			updated_at = EXCLUDED.updated_at`,
		k.ID, k.TenantID, k.ObjectiveID, k.CanonicalID, k.Name, k.Definition, k.Owner, k.Target,
		k.MeasurementPeriod, k.ActualValue, k.Unit, k.Source, k.Status, string(k.DataStatus),
		k.FreshnessWindowH, k.LastMeasuredAt, k.CreatedAt, k.UpdatedAt)
	if err != nil {
		log.Printf("phase12: persist kpi failed: %v", err)
		return
	}
	metricsCollector.incDBQuery()
}