package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/matjames/statgate-lib/events"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type TelemetryCollector struct {
	db      *sql.DB
	bus     *events.EventBus
	client  *http.Client
	targets []ServiceTarget
	mu      sync.RWMutex
}

func NewTelemetryCollector(db *sql.DB, bus *events.EventBus, targets []ServiceTarget) *TelemetryCollector {
	return &TelemetryCollector{db: db, bus: bus, targets: targets, client: &http.Client{Timeout: 5 * time.Second}}
}
func (t *TelemetryCollector) Start(ctx context.Context, every time.Duration) {
	t.ProbeAll(ctx)
	ticker := time.NewTicker(every)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t.ProbeAll(ctx)
			}
		}
	}()
}
func (t *TelemetryCollector) ProbeAll(ctx context.Context) {
	t.mu.RLock()
	targets := append([]ServiceTarget(nil), t.targets...)
	t.mu.RUnlock()
	for _, target := range targets {
		if target.Enabled {
			for _, endpoint := range []string{"/health", "/ready", "/metrics"} {
				t.probe(ctx, target, endpoint)
			}
		}
	}
}
func (t *TelemetryCollector) probe(ctx context.Context, target ServiceTarget, endpoint string) {
	started := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(target.BaseURL, "/")+endpoint, nil)
	response, err := t.client.Do(req)
	result := ProbeResult{ServiceID: target.ID, Endpoint: endpoint, CheckedAt: time.Now(), LatencyMS: time.Since(started).Milliseconds(), Status: "healthy"}
	if err != nil {
		result.Status = "unreachable"
		result.Detail = err.Error()
	} else {
		defer response.Body.Close()
		result.HTTPStatus = response.StatusCode
		if response.StatusCode >= 400 {
			result.Status = "unhealthy"
			result.Detail = response.Status
		}
		if endpoint == "/metrics" && result.Status == "healthy" {
			t.ingestPrometheus(target.ID, response.Body, result.CheckedAt)
		}
	}
	if t.db != nil {
		_, _ = t.db.ExecContext(ctx, "INSERT INTO runops.service_probes(service_id,endpoint,status,http_status,latency_ms,detail,checked_at) VALUES($1,$2,$3,$4,$5,$6,$7)", result.ServiceID, result.Endpoint, result.Status, result.HTTPStatus, result.LatencyMS, result.Detail, result.CheckedAt)
	}
	if result.Status != "healthy" {
		fingerprint := fmt.Sprintf("probe:%s:%s", target.ID, endpoint)
		t.raiseAlert(ctx, Alert{ID: fmt.Sprintf("alt-%d", time.Now().UnixNano()), Fingerprint: fingerprint, Severity: severity(target.Criticality), Status: "open", Source: "probe", Summary: fmt.Sprintf("%s %s is %s", target.Name, endpoint, result.Status), Labels: map[string]string{"service_id": target.ID, "endpoint": endpoint}, StartsAt: result.CheckedAt})
	}
}
func severity(criticality string) string {
	if criticality == "tier-0" {
		return "critical"
	}
	return "warning"
}
func (t *TelemetryCollector) ingestPrometheus(serviceID string, r io.Reader, at time.Time) {
	raw, _ := io.ReadAll(io.LimitReader(r, 2<<20))
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.HasPrefix(line, "#") {
			continue
		}
		var value float64
		if _, err := fmt.Sscanf(fields[1], "%f", &value); err == nil && t.db != nil {
			name := strings.Split(fields[0], "{")[0]
			_, _ = t.db.Exec("INSERT INTO runops.metric_samples(service_id,metric_name,value,observed_at) VALUES($1,$2,$3,$4)", serviceID, name, value, at)
		}
	}
}
func (t *TelemetryCollector) raiseAlert(ctx context.Context, a Alert) {
	if t.db != nil {
		_, _ = t.db.ExecContext(ctx, "INSERT INTO runops.alerts(id,fingerprint,severity,status,source,summary,labels,starts_at) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8) ON CONFLICT(fingerprint) DO UPDATE SET status='open',summary=EXCLUDED.summary", a.ID, a.Fingerprint, a.Severity, a.Status, a.Source, a.Summary, jsonString(a.Labels), a.StartsAt)
	}
	publish(t.bus, "runops.alert.raised", "alert", a.Fingerprint, "system", map[string]any{"severity": a.Severity, "summary": a.Summary, "labels": a.Labels})
}
