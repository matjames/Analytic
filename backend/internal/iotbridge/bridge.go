package iotbridge

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// TelemetryReading represents a single IoT sensor measurement from StatIoT.
type TelemetryReading struct {
	DeviceID    string    `json:"device_id"`
	SensorType  string    `json:"sensor_type"`   // "temperature", "humidity", "air_quality", "water_level"
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`           // "°C", "%RH", "AQI", "m"
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	TenantID    string    `json:"tenant_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// IndicatorMapping maps IoT sensor types to statistical indicators.
type IndicatorMapping struct {
	SensorType    string  `json:"sensor_type"`
	IndicatorCode string  `json:"indicator_code"`
	IndicatorName string  `json:"indicator_name"`
	LowerThreshold float64 `json:"lower_threshold"` // Acceptable range low
	UpperThreshold float64 `json:"upper_threshold"` // Acceptable range high
	AnomalySigma   float64 `json:"anomaly_sigma"`   // Sigma for 3-sigma anomaly detection
}

// IoTAnomaly represents an anomaly detected in IoT telemetry.
type IoTAnomaly struct {
	DeviceID      string    `json:"device_id"`
	SensorType    string    `json:"sensor_type"`
	Value         float64   `json:"value"`
	Mean          float64   `json:"mean"`
	StdDev        float64   `json:"std_dev"`
	ZScore        float64   `json:"z_score"`
	Severity      string    `json:"severity"` // "INFO", "WARNING", "CRITICAL"
	Message       string    `json:"message"`
	DetectedAt    time.Time `json:"detected_at"`
}

// IoTIndicatorValue is the computed indicator derived from IoT telemetry.
type IoTIndicatorValue struct {
	IndicatorCode  string    `json:"indicator_code"`
	IndicatorName  string    `json:"indicator_name"`
	DeviceCount    int       `json:"device_count"`
	Mean           float64   `json:"mean"`
	StdDev         float64   `json:"std_dev"`
	Min            float64   `json:"min"`
	Max            float64   `json:"max"`
	LatestValue    float64   `json:"latest_value"`
	ReadingCount   int       `json:"reading_count"`
	AnomalyCount   int       `json:"anomaly_count"`
	ComputedAt     time.Time `json:"computed_at"`
}

// Bridge connects StatIoT telemetry streams to Analytics Core indicator computation.
type Bridge struct {
	mu            sync.RWMutex
	mappings      []IndicatorMapping
	readingBuffer map[string][]TelemetryReading // sensorType -> recent readings
	anomalies     []IoTAnomaly
	bufferSize    int
}

// NewBridge creates a new IoT-to-Analytics bridge.
func NewBridge() *Bridge {
	return &Bridge{
		mappings: defaultMappings(),
		readingBuffer: make(map[string][]TelemetryReading),
		anomalies:     make([]IoTAnomaly, 0),
		bufferSize:    1000,
	}
}

// defaultMappings returns built-in sensor-to-indicator mappings.
func defaultMappings() []IndicatorMapping {
	return []IndicatorMapping{
		{SensorType: "temperature", IndicatorCode: "ENV_TEMP_AVG", IndicatorName: "Average Environmental Temperature", LowerThreshold: -10, UpperThreshold: 50, AnomalySigma: 3.0},
		{SensorType: "humidity", IndicatorCode: "ENV_HUMIDITY_AVG", IndicatorName: "Average Relative Humidity", LowerThreshold: 10, UpperThreshold: 100, AnomalySigma: 3.0},
		{SensorType: "air_quality", IndicatorCode: "ENV_AQI", IndicatorName: "Air Quality Index", LowerThreshold: 0, UpperThreshold: 300, AnomalySigma: 2.5},
		{SensorType: "water_level", IndicatorCode: "ENV_WATER_LEVEL", IndicatorName: "Water Level Monitor", LowerThreshold: 0, UpperThreshold: 20, AnomalySigma: 3.0},
		{SensorType: "rainfall", IndicatorCode: "ENV_RAINFALL_MM", IndicatorName: "Rainfall Accumulation (mm)", LowerThreshold: 0, UpperThreshold: 300, AnomalySigma: 2.5},
	}
}

// IngestReading processes a single telemetry reading: buffers it, runs anomaly detection, and returns any anomaly.
func (b *Bridge) IngestReading(ctx context.Context, reading TelemetryReading) *IoTAnomaly {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := reading.SensorType
	b.readingBuffer[key] = append(b.readingBuffer[key], reading)

	// Trim buffer
	if len(b.readingBuffer[key]) > b.bufferSize {
		b.readingBuffer[key] = b.readingBuffer[key][len(b.readingBuffer[key])-b.bufferSize:]
	}

	// 3-sigma anomaly detection
	readings := b.readingBuffer[key]
	if len(readings) < 10 {
		return nil // Need sufficient history
	}

	mean, stddev := computeStats(readings)
	if stddev == 0 {
		return nil
	}

	zscore := (reading.Value - mean) / stddev
	absZ := math.Abs(zscore)

	// Find matching mapping
	sigma := 3.0
	for _, m := range b.mappings {
		if m.SensorType == reading.SensorType {
			sigma = m.AnomalySigma
			break
		}
	}

	if absZ >= sigma {
		severity := "WARNING"
		if absZ >= sigma+1 {
			severity = "CRITICAL"
		}

		anomaly := IoTAnomaly{
			DeviceID:   reading.DeviceID,
			SensorType: reading.SensorType,
			Value:      reading.Value,
			Mean:       math.Round(mean*100) / 100,
			StdDev:     math.Round(stddev*100) / 100,
			ZScore:     math.Round(zscore*100) / 100,
			Severity:   severity,
			Message:    fmt.Sprintf("IoT device %s sensor %s reading %.2f deviates %.1fσ from mean %.2f", reading.DeviceID, reading.SensorType, reading.Value, absZ, mean),
			DetectedAt: time.Now().UTC(),
		}
		b.anomalies = append(b.anomalies, anomaly)
		return &anomaly
	}

	return nil
}

// ComputeIndicators aggregates buffered readings into indicator values.
func (b *Bridge) ComputeIndicators(ctx context.Context) []IoTIndicatorValue {
	b.mu.RLock()
	defer b.mu.RUnlock()

	indicators := make([]IoTIndicatorValue, 0)

	for _, mapping := range b.mappings {
		readings := b.readingBuffer[mapping.SensorType]
		if len(readings) == 0 {
			continue
		}

		mean, stddev := computeStats(readings)

		// Count unique devices
		deviceSet := make(map[string]bool)
		minVal := math.MaxFloat64
		maxVal := -math.MaxFloat64
		var latest TelemetryReading

		anomalyCnt := 0
		for _, r := range readings {
			deviceSet[r.DeviceID] = true
			if r.Value < minVal {
				minVal = r.Value
			}
			if r.Value > maxVal {
				maxVal = r.Value
			}
			if r.Timestamp.After(latest.Timestamp) {
				latest = r
			}
			z := math.Abs((r.Value - mean) / (stddev + 0.0001))
			if z >= mapping.AnomalySigma {
				anomalyCnt++
			}
		}

		indicators = append(indicators, IoTIndicatorValue{
			IndicatorCode: mapping.IndicatorCode,
			IndicatorName: mapping.IndicatorName,
			DeviceCount:   len(deviceSet),
			Mean:          math.Round(mean*100) / 100,
			StdDev:        math.Round(stddev*100) / 100,
			Min:           math.Round(minVal*100) / 100,
			Max:           math.Round(maxVal*100) / 100,
			LatestValue:   latest.Value,
			ReadingCount:  len(readings),
			AnomalyCount:  anomalyCnt,
			ComputedAt:    time.Now().UTC(),
		})
	}

	return indicators
}

// GetAnomalies returns all detected IoT anomalies.
func (b *Bridge) GetAnomalies() []IoTAnomaly {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]IoTAnomaly, len(b.anomalies))
	copy(result, b.anomalies)
	return result
}

// GetMappings returns the current sensor-to-indicator mapping table.
func (b *Bridge) GetMappings() []IndicatorMapping {
	return b.mappings
}

// computeStats returns mean and standard deviation of readings.
func computeStats(readings []TelemetryReading) (float64, float64) {
	if len(readings) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, r := range readings {
		sum += r.Value
	}
	mean := sum / float64(len(readings))

	variance := 0.0
	for _, r := range readings {
		diff := r.Value - mean
		variance += diff * diff
	}
	variance /= float64(len(readings))
	return mean, math.Sqrt(variance)
}
