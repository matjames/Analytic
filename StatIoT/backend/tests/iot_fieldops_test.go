package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matjames/statgate-lib/health"
	"github.com/matjames/statgate-lib/metrics"

	"statiot-backend/internal/api"
	iotevents "statiot-backend/internal/events"
	"statiot-backend/internal/fieldops"
	"statiot-backend/internal/iot"
	"statiot-backend/internal/models"
	"statiot-backend/internal/store"
)

var (
	testMetricsOnce sync.Once
	testMetricsInst *metrics.Metrics
)

func getTestMetrics() *metrics.Metrics {
	testMetricsOnce.Do(func() {
		testMetricsInst = metrics.NewMetrics("statiot_test")
	})
	return testMetricsInst
}

func setupTestRouter(t *testing.T) (*store.MemStore, http.Handler) {
	t.Helper()
	os.Setenv("STATIOT_GATEWAY_SECRET", "test-gateway-secret")
	memStore := store.NewMemStore()
	eventWorker := iotevents.NewEventWorker(nil, memStore)
	gatewayEngine := iot.NewGatewayEngine(memStore, eventWorker)
	syncEngine := fieldops.NewSyncEngine(memStore, eventWorker)
	spatialTracker := fieldops.NewSpatialTracker(memStore)

	promMetrics := getTestMetrics()
	healthChecker := health.NewChecker("statiot_test", nil)

	iotHandlers := api.NewIoTHandlers(memStore, gatewayEngine)
	fieldHandlers := api.NewFieldOpsHandlers(memStore, syncEngine, spatialTracker)

	router := api.SetupRouter(api.RouterConfig{
		IoTHandlers:      iotHandlers,
		FieldOpsHandlers: fieldHandlers,
		HealthChecker:    healthChecker,
		Metrics:          promMetrics,
		AuthValidator:    nil,
		CORSOrigin:       "*",
		Env:              "test",
	})

	// Admin routes enforce tenant context (tenant.GinTenantIsolation). The
	// test suite runs without a JWT validator, so inject a tenant header for
	// every request the way an authenticated caller would carry one.
	return memStore, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Tenant-ID") == "" {
			r.Header.Set("X-Tenant-ID", "default")
		}
		router.ServeHTTP(w, r)
	})
}

func TestHealthAndMetricsEndpoints(t *testing.T) {
	_, router := setupTestRouter(t)

	// 1. Health Probe
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /health, got %d", w.Code)
	}

	// 2. Ready Probe
	req, _ = http.NewRequest(http.MethodGet, "/ready", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /ready, got %d", w.Code)
	}

	// 3. Metrics Probe
	req, _ = http.NewRequest(http.MethodGet, "/metrics", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /metrics, got %d", w.Code)
	}

	// 4. API Info
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/info", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /api/v1/info, got %d", w.Code)
	}
}

func TestIoTGatewayAndDeviceFlow(t *testing.T) {
	_, router := setupTestRouter(t)

	// 1. Register Gateway
	gwPayload := models.IoTGateway{
		Name:        "Kampala Metro Environmental Gateway 01",
		GatewayCode: "GW-KLA-01",
		IPAddress:   "10.20.30.40",
		Status:      models.GatewayStatusOnline,
	}
	body, _ := json.Marshal(gwPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/iot/gateways", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create gateway, status %d: %s", w.Code, w.Body.String())
	}

	var gwResp struct {
		Gateway models.IoTGateway `json:"gateway"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &gwResp)
	gwID := gwResp.Gateway.ID

	// 2. Gateway Heartbeat (requires the shared gateway secret)
	hbReq, _ := http.NewRequest(http.MethodPost, "/api/v1/iot/gateways/"+gwID+"/heartbeat", bytes.NewBuffer([]byte(`{"status":"ONLINE"}`)))
	hbReq.Header.Set("Content-Type", "application/json")
	hbReq.Header.Set("X-Gateway-Secret", "test-gateway-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, hbReq)
	if w.Code != http.StatusOK {
		t.Fatalf("gateway heartbeat failed: %s", w.Body.String())
	}

	// 3. Register Device
	devPayload := models.IoTDevice{
		DeviceUID:   "DEV-WEATHER-001",
		Name:        "Nakasero Weather Station Alpha",
		DeviceType:  "WEATHER_STATION",
		Protocol:    "MQTT",
		GatewayID:   &gwID,
		FirmwareVersion: "1.2.0",
	}
	body, _ = json.Marshal(devPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/iot/devices", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to register device: %s", w.Body.String())
	}

	var devResp struct {
		Device      models.IoTDevice `json:"device"`
		DeviceToken string           `json:"device_token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &devResp)
	deviceID := devResp.Device.ID
	deviceToken := devResp.DeviceToken

	if deviceToken == "" {
		t.Fatalf("expected generated device_token on registration")
	}

	// 4. Attach Sensor with Threshold
	minT := 10.0
	maxT := 40.0
	sensorPayload := models.SensorRegistryItem{
		SensorCode:        "TEMP_S1",
		MetricName:        "temperature",
		UnitOfMeasure:     "celsius",
		MinThreshold:      &minT,
		MaxThreshold:      &maxT,
		CalibrationFactor: 1.0,
		CalibrationOffset: 0.0,
	}
	body, _ = json.Marshal(sensorPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/iot/devices/"+deviceID+"/sensors", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to register sensor: %s", w.Body.String())
	}

	// 5. Ingest Telemetry Batch (with 1 Normal and 1 Anomaly Threshold Breach)
	now := time.Now().UTC()
	ingestReq := models.TelemetryBatchIngestRequest{
		DeviceUID: "DEV-WEATHER-001",
		AuthToken: deviceToken,
		Readings: []models.TelemetryRecord{
			{
				SensorCode: "TEMP_S1",
				MetricName: "temperature",
				Value:      28.5, // Normal
				RecordedAt: now.Add(-5 * time.Minute),
			},
			{
				SensorCode: "TEMP_S1",
				MetricName: "temperature",
				Value:      48.2, // Anomaly > 40.0 threshold
				RecordedAt: now,
			},
		},
	}
	body, _ = json.Marshal(ingestReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/iot/telemetry/ingest", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Device-facing routes require device credentials (Stage 2 auth).
	req.Header.Set("X-Device-UID", "DEV-WEATHER-001")
	req.Header.Set("X-Device-Token", deviceToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("telemetry ingestion failed: %s", w.Body.String())
	}

	var ingestResp struct {
		RecordsStored int               `json:"records_stored"`
		AlertsCreated int               `json:"alerts_created"`
		Alerts        []models.IoTAlert `json:"alerts"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &ingestResp)
	if ingestResp.RecordsStored != 2 {
		t.Fatalf("expected 2 records stored, got %d", ingestResp.RecordsStored)
	}
	if ingestResp.AlertsCreated != 1 {
		t.Fatalf("expected 1 anomaly alert created for threshold breach, got %d", ingestResp.AlertsCreated)
	}

	// 6. Query Telemetry Aggregates (device credentials required)
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/iot/telemetry/"+deviceID+"/aggregates?metric_name=temperature", nil)
	req.Header.Set("X-Device-UID", "DEV-WEATHER-001")
	req.Header.Set("X-Device-Token", deviceToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("telemetry aggregate query failed: %s", w.Body.String())
	}

	var agg models.TelemetryAggregate
	_ = json.Unmarshal(w.Body.Bytes(), &agg)
	if agg.Count != 2 || agg.Min != 28.5 || agg.Max != 48.2 {
		t.Fatalf("unexpected aggregates: count=%d, min=%.2f, max=%.2f", agg.Count, agg.Min, agg.Max)
	}

	// 7. Verify Alerts List & Resolution
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/iot/alerts", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("alerts list failed: %s", w.Body.String())
	}
	var alertsResp struct {
		Alerts []models.IoTAlert `json:"alerts"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &alertsResp)
	if len(alertsResp.Alerts) != 1 {
		t.Fatalf("expected 1 alert in list, got %d", len(alertsResp.Alerts))
	}

	// Resolve the alert
	alertID := alertsResp.Alerts[0].ID
	resolveReq, _ := http.NewRequest(http.MethodPost, "/api/v1/iot/alerts/"+alertID+"/resolve", bytes.NewBuffer([]byte(`{"resolved_by":"operator-sarah"}`)))
	resolveReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, resolveReq)
	if w.Code != http.StatusOK {
		t.Fatalf("alert resolve failed: %s", w.Body.String())
	}
}

func TestMobileFieldOpsAndDeltaSync(t *testing.T) {
	_, router := setupTestRouter(t)

	// 1. Create Field Worker
	workerPayload := models.FieldWorker{
		UserID:           "usr-field-101",
		FullName:         "David Okello",
		Role:             "ENUMERATOR",
		AssignedDistrict: "Gulu",
	}
	body, _ := json.Marshal(workerPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/fieldops/workers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create worker: %s", w.Body.String())
	}

	var wResp struct {
		Worker models.FieldWorker `json:"worker"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &wResp)
	workerID := wResp.Worker.ID

	// 2. Register Mobile Device
	devUUID := "TAB-SAMSUNG-GL-09"
	mobileDevPayload := models.MobileDevice{
		DeviceUUID:       devUUID,
		AssignedWorkerID: &workerID,
		Platform:         "ANDROID",
		AppVersion:       "2.4.1",
	}
	body, _ = json.Marshal(mobileDevPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/fieldops/devices/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to register mobile device: %s", w.Body.String())
	}

	// 3. Create Mobile Form Definition
	formPayload := models.MobileFormDefinition{
		FormCode: "AGRI_FARM_CENSUS_V1",
		Title:    "Agricultural Household Census 2026",
		Version:  1,
		SchemaDefinition: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"farmer_name": map[string]interface{}{"type": "string"},
				"crop_type":   map[string]interface{}{"type": "string"},
				"acreage":     map[string]interface{}{"type": "number"},
			},
		},
	}
	body, _ = json.Marshal(formPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/fieldops/forms", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create form definition: %s", w.Body.String())
	}

	var formResp struct {
		Form models.MobileFormDefinition `json:"form"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &formResp)
	formID := formResp.Form.ID

	// 4. Test Pull Sync (Downloading active forms and assigned visits)
	pullReqPayload := models.SyncPullRequest{
		DeviceUUID:     devUUID,
		WorkerID:       workerID,
		LastSyncedTime: time.Now().Add(-24 * time.Hour),
	}
	body, _ = json.Marshal(pullReqPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/fieldops/sync/pull", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("sync pull failed: %s", w.Body.String())
	}

	var pullResp models.SyncPullResponse
	_ = json.Unmarshal(w.Body.Bytes(), &pullResp)
	if len(pullResp.Forms) == 0 {
		t.Fatalf("expected at least 1 form in sync pull, got 0")
	}

	// 5. Test Push Sync (Uploading offline collected submissions and GPS breadcrumbs)
	clientSubID := uuid.New().String()
	pushReqPayload := models.SyncPushRequest{
		DeviceUUID: devUUID,
		WorkerID:   workerID,
		ClientTime: time.Now().UTC(),
		Submissions: []models.MobileSubmission{
			{
				ClientSubmissionID: clientSubID,
				FormID:             formID,
				WorkerID:           workerID,
				Version:            1,
				CollectedAt:        time.Now().Add(-30 * time.Minute),
				DataPayload: map[string]interface{}{
					"farmer_name": "Akol Grace",
					"crop_type":   "Maize",
					"acreage":     4.5,
				},
				GeoPoint: map[string]interface{}{
					"lat": 2.7745,
					"lng": 32.2990,
				},
			},
		},
		Breadcrumbs: []models.GPSBreadcrumb{
			{
				Latitude:   2.7745,
				Longitude:  32.2990,
				RecordedAt: time.Now().Add(-10 * time.Minute),
			},
		},
	}
	body, _ = json.Marshal(pushReqPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/fieldops/sync/push", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("sync push failed: %s", w.Body.String())
	}

	var pushResp models.SyncPushResponse
	_ = json.Unmarshal(w.Body.Bytes(), &pushResp)
	if pushResp.CommittedCount < 2 {
		t.Fatalf("expected at least 2 committed records (1 sub + 1 breadcrumb), got %d", pushResp.CommittedCount)
	}

	// 6. Test Conflict Resolution Flow (Pushing updated version with divergence)
	conflictPushReq := models.SyncPushRequest{
		DeviceUUID: devUUID,
		WorkerID:   workerID,
		ClientTime: time.Now().UTC(),
		Submissions: []models.MobileSubmission{
			{
				ClientSubmissionID: clientSubID,
				FormID:             formID,
				WorkerID:           workerID,
				Version:            2, // Higher version triggers conflict resolution
				CollectedAt:        time.Now().Add(-5 * time.Minute),
				DataPayload: map[string]interface{}{
					"farmer_name": "Akol Grace",
					"crop_type":   "Maize & Sorghum",
					"acreage":     5.0,
				},
			},
		},
	}
	body, _ = json.Marshal(conflictPushReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/fieldops/sync/push", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("conflict push failed: %s", w.Body.String())
	}

	_ = json.Unmarshal(w.Body.Bytes(), &pushResp)
	if pushResp.ConflictsCount != 1 {
		t.Fatalf("expected 1 conflict detected and auto-merged, got %d", pushResp.ConflictsCount)
	}

	// 7. Verify Geofence Evaluation & GPS Tracking
	geofencePayload := models.GeofenceZone{
		Name:         "Gulu Demonstration Farm Zone",
		ZoneType:     "PROJECT_SITE",
		CenterLat:    2.7745,
		CenterLng:    32.2990,
		RadiusMeters: 500.0, // 500m radius
	}
	body, _ = json.Marshal(geofencePayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/fieldops/geofences", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create geofence failed: %s", w.Body.String())
	}

	// Stream GPS point directly inside the geofence
	gpsPayload := map[string]interface{}{
		"worker_id":   workerID,
		"device_uuid": devUUID,
		"breadcrumbs": []models.GPSBreadcrumb{
			{
				Latitude:   2.7746,
				Longitude:  32.2991, // ~15 meters from center
				RecordedAt: time.Now().UTC(),
			},
		},
	}
	body, _ = json.Marshal(gpsPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/fieldops/gps/track", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("gps track ingestion failed: %s", w.Body.String())
	}

	var gpsResp struct {
		ActiveGeofenceZones []models.GeofenceZone `json:"active_geofence_zones"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &gpsResp)
	if len(gpsResp.ActiveGeofenceZones) == 0 {
		t.Fatalf("expected GPS breadcrumb to trigger inside the active geofence zone")
	}
}

func TestObjectLinksIntegration(t *testing.T) {
	memStore, _ := setupTestRouter(t)
	eventWorker := iotevents.NewEventWorker(nil, memStore)

	alert := &models.IoTAlert{
		ID:         "alt-999",
		DeviceID:   "dev-temp-01",
		AlertType:  "THRESHOLD_BREACH",
		Severity:   "CRITICAL",
		Message:    "Temp critical breach",
		TenantID:   "default",
		SensorCode: "TEMP_01",
	}

	// Publish device alert should automatically register an object link
	err := eventWorker.PublishDeviceAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("failed to publish device alert: %v", err)
	}

	links, err := memStore.ListObjectLinks(context.Background(), "iot_device", "dev-temp-01")
	if err != nil || len(links) == 0 {
		t.Fatalf("expected object_link created for iot_device -> alert, got %d links", len(links))
	}

	if links[0].TargetID != "alt-999" || links[0].Relationship != "generated_alert" {
		t.Fatalf("unexpected link content: %+v", links[0])
	}
}
