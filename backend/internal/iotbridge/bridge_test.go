package iotbridge

import (
	"context"
	"testing"
	"time"
)

func TestIoTBridgeIngestAndAnomalyDetection(t *testing.T) {
	bridge := NewBridge()
	ctx := context.Background()

	// Ingest baseline normal readings (around 22.0°C)
	for i := 0; i < 20; i++ {
		reading := TelemetryReading{
			DeviceID:   "sensor-temp-001",
			SensorType: "temperature",
			Value:      22.0 + float64(i%3)*0.5,
			Unit:       "°C",
			TenantID:   "tenant-alpha",
			Timestamp:  time.Now().UTC().Add(-time.Duration(20-i) * time.Minute),
		}
		anomaly := bridge.IngestReading(ctx, reading)
		if anomaly != nil {
			t.Errorf("Unexpected anomaly during baseline ingestion: %v", anomaly)
		}
	}

	// Ingest an extreme outlier (> 3 sigma above baseline ~22.5°C)
	spikeReading := TelemetryReading{
		DeviceID:   "sensor-temp-001",
		SensorType: "temperature",
		Value:      85.0, // High temperature anomaly
		Unit:       "°C",
		TenantID:   "tenant-alpha",
		Timestamp:  time.Now().UTC(),
	}

	anomaly := bridge.IngestReading(ctx, spikeReading)
	if anomaly == nil {
		t.Fatalf("Expected 3-sigma anomaly detection for 85°C spike, got nil")
	}

	if anomaly.Severity != "CRITICAL" && anomaly.Severity != "WARNING" {
		t.Errorf("Expected WARNING or CRITICAL anomaly severity, got %s", anomaly.Severity)
	}

	// Verify indicator computation
	indicators := bridge.ComputeIndicators(ctx)
	if len(indicators) == 0 {
		t.Errorf("Expected computed IoT indicators")
	}

	foundTemp := false
	for _, ind := range indicators {
		if ind.IndicatorCode == "ENV_TEMP_AVG" {
			foundTemp = true
			if ind.DeviceCount != 1 {
				t.Errorf("Expected 1 device, got %d", ind.DeviceCount)
			}
			if ind.AnomalyCount == 0 {
				t.Errorf("Expected recorded anomaly count > 0")
			}
		}
	}
	if !foundTemp {
		t.Errorf("ENV_TEMP_AVG indicator not found in results")
	}
}
