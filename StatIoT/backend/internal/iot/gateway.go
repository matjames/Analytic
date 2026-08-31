package iot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"statiot-backend/internal/models"
	"statiot-backend/internal/store"
)

type GatewayEngine struct {
	store          store.Store
	alertPublisher AlertPublisher
}

type AlertPublisher interface {
	PublishDeviceAlert(ctx context.Context, alert *models.IoTAlert) error
	PublishTelemetryIngested(ctx context.Context, deviceID, tenantID string, recordCount int) error
}

func NewGatewayEngine(st store.Store, pub AlertPublisher) *GatewayEngine {
	return &GatewayEngine{
		store:          st,
		alertPublisher: pub,
	}
}

// AuthenticateDevice verifies the device token against the stored hash or secret.
func (g *GatewayEngine) AuthenticateDevice(ctx context.Context, deviceUID, rawToken string) (*models.IoTDevice, error) {
	dev, err := g.store.GetDeviceByUID(ctx, deviceUID)
	if err != nil {
		return nil, fmt.Errorf("unknown device: %w", err)
	}

	if dev.Status == models.DeviceStatusDecommissioned || dev.Status == models.DeviceStatusInactive {
		return nil, fmt.Errorf("device is not in active state: %s", dev.Status)
	}

	if dev.AuthTokenHash != "" && rawToken != "" {
		hash := sha256.Sum256([]byte(rawToken))
		computed := hex.EncodeToString(hash[:])
		if dev.AuthTokenHash != computed && dev.AuthTokenHash != rawToken {
			return nil, fmt.Errorf("device authentication failed: invalid token")
		}
	}

	return dev, nil
}

// ProcessTelemetryBatch processes, calibrates, anomaly-checks, and stores telemetry records.
func (g *GatewayEngine) ProcessTelemetryBatch(ctx context.Context, req *models.TelemetryBatchIngestRequest) (int, []models.IoTAlert, error) {
	dev, err := g.store.GetDeviceByUID(ctx, req.DeviceUID)
	if err != nil {
		return 0, nil, fmt.Errorf("device not registered: %w", err)
	}

	sensors, _ := g.store.ListSensorsByDevice(ctx, dev.ID)
	sensorMap := make(map[string]models.SensorRegistryItem)
	for _, s := range sensors {
		sensorMap[s.SensorCode] = s
	}

	now := time.Now().UTC()
	var processedRecords []models.TelemetryRecord
	var generatedAlerts []models.IoTAlert

	for _, reading := range req.Readings {
		rec := reading
		rec.DeviceID = dev.ID
		rec.TenantID = dev.TenantID
		if rec.RecordedAt.IsZero() {
			if req.RecordedAt != nil {
				rec.RecordedAt = *req.RecordedAt
			} else {
				rec.RecordedAt = now
			}
		}

		// Apply sensor calibration if registered
		if sensor, found := sensorMap[rec.SensorCode]; found {
			if sensor.CalibrationFactor != 0 {
				rec.Value = (rec.Value * sensor.CalibrationFactor) + sensor.CalibrationOffset
			}

			// Threshold checking & Anomaly Detection
			var alertMsg string
			var severity = "WARNING"
			var breached = false

			if sensor.MaxThreshold != nil && rec.Value > *sensor.MaxThreshold {
				breached = true
				severity = "CRITICAL"
				alertMsg = fmt.Sprintf("High threshold breach on sensor %s (%s): value %.2f > max %.2f %s",
					sensor.SensorCode, sensor.MetricName, rec.Value, *sensor.MaxThreshold, sensor.UnitOfMeasure)
			} else if sensor.MinThreshold != nil && rec.Value < *sensor.MinThreshold {
				breached = true
				alertMsg = fmt.Sprintf("Low threshold breach on sensor %s (%s): value %.2f < min %.2f %s",
					sensor.SensorCode, sensor.MetricName, rec.Value, *sensor.MinThreshold, sensor.UnitOfMeasure)
			}

			if breached {
				rec.IsAnomaly = true
				alert := models.IoTAlert{
					ID:         uuid.New().String(),
					DeviceID:   dev.ID,
					SensorCode: sensor.SensorCode,
					AlertType:  "THRESHOLD_BREACH",
					Severity:   severity,
					Message:    alertMsg,
					Value:      &rec.Value,
					Threshold:  sensor.MaxThreshold,
					Status:     "ACTIVE",
					TenantID:   dev.TenantID,
					CreatedAt:  now,
				}
				_ = g.store.CreateAlert(ctx, &alert)
				generatedAlerts = append(generatedAlerts, alert)

				if g.alertPublisher != nil {
					_ = g.alertPublisher.PublishDeviceAlert(ctx, &alert)
				}
			}
		}

		processedRecords = append(processedRecords, rec)
	}

	// Batch insert into time-series store
	if err := g.store.InsertTelemetryBatch(ctx, processedRecords); err != nil {
		return 0, generatedAlerts, fmt.Errorf("failed to persist telemetry batch: %w", err)
	}

	// Update device heartbeat & last seen
	_ = g.store.UpdateDeviceHeartbeat(ctx, dev.ID, dev.BatteryLevel, dev.SignalStrengthDBM, dev.Latitude, dev.Longitude)

	if g.alertPublisher != nil && len(processedRecords) > 0 {
		_ = g.alertPublisher.PublishTelemetryIngested(ctx, dev.ID, dev.TenantID, len(processedRecords))
	}

	log.Printf("[IoTGateway] Successfully ingested %d telemetry records for device %s (%s)", len(processedRecords), dev.Name, dev.DeviceUID)
	return len(processedRecords), generatedAlerts, nil
}

// GenerateDeviceToken creates a SHA-256 token hash for secure device provisioning.
func GenerateDeviceToken(rawSecret string) (token string, hash string) {
	if rawSecret == "" {
		rawSecret = uuid.New().String() + "-" + uuid.New().String()
	}
	h := sha256.Sum256([]byte(rawSecret))
	return rawSecret, hex.EncodeToString(h[:])
}
