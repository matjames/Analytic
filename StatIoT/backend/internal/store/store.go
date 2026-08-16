package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"statiot-backend/internal/models"
)

type Store interface {
	// Gateways
	CreateGateway(ctx context.Context, gw *models.IoTGateway) error
	GetGateway(ctx context.Context, id string) (*models.IoTGateway, error)
	ListGateways(ctx context.Context, tenantID string) ([]models.IoTGateway, error)
	UpdateGatewayHeartbeat(ctx context.Context, id string, status models.GatewayStatus) error

	// Devices & Sensors
	RegisterDevice(ctx context.Context, dev *models.IoTDevice) error
	GetDevice(ctx context.Context, id string) (*models.IoTDevice, error)
	GetDeviceByUID(ctx context.Context, uid string) (*models.IoTDevice, error)
	ListDevices(ctx context.Context, tenantID string) ([]models.IoTDevice, error)
	UpdateDeviceHeartbeat(ctx context.Context, id string, battery *float64, signal *int, lat *float64, lng *float64) error
	RegisterSensor(ctx context.Context, s *models.SensorRegistryItem) error
	ListSensorsByDevice(ctx context.Context, deviceID string) ([]models.SensorRegistryItem, error)

	// Telemetry & Aggregates
	InsertTelemetryBatch(ctx context.Context, readings []models.TelemetryRecord) error
	GetLatestTelemetry(ctx context.Context, deviceID string, limit int) ([]models.TelemetryRecord, error)
	GetTelemetryAggregates(ctx context.Context, deviceID string, metricName string, from time.Time, to time.Time) (*models.TelemetryAggregate, error)

	// Edge Config & Firmware
	SetDesiredConfig(ctx context.Context, cfg *models.EdgeConfiguration) error
	GetDeviceConfig(ctx context.Context, deviceID string) (*models.EdgeConfiguration, error)
	ReportAppliedConfig(ctx context.Context, deviceID string, reported map[string]interface{}) error
	CreateFirmware(ctx context.Context, fw *models.FirmwareRelease) error
	GetLatestFirmware(ctx context.Context, deviceType string) (*models.FirmwareRelease, error)

	// Alerts
	CreateAlert(ctx context.Context, alert *models.IoTAlert) error
	ListAlerts(ctx context.Context, tenantID string, status string) ([]models.IoTAlert, error)
	ResolveAlert(ctx context.Context, alertID string, resolvedBy string) error

	// Mobile FieldOps: Workers & Devices
	CreateFieldWorker(ctx context.Context, w *models.FieldWorker) error
	GetFieldWorker(ctx context.Context, id string) (*models.FieldWorker, error)
	ListFieldWorkers(ctx context.Context, tenantID string) ([]models.FieldWorker, error)
	RegisterMobileDevice(ctx context.Context, dev *models.MobileDevice) error
	GetMobileDevice(ctx context.Context, uuid string) (*models.MobileDevice, error)

	// Field Visits & Tasks
	CreateFieldVisit(ctx context.Context, v *models.FieldVisit) error
	GetFieldVisit(ctx context.Context, id string) (*models.FieldVisit, error)
	ListVisitsByWorker(ctx context.Context, workerID string) ([]models.FieldVisit, error)
	UpdateVisitStatus(ctx context.Context, id string, status string, start *time.Time, end *time.Time) error
	CreateVisitTask(ctx context.Context, t *models.VisitTask) error
	ListTasksByVisit(ctx context.Context, visitID string) ([]models.VisitTask, error)
	UpdateVisitTask(ctx context.Context, t *models.VisitTask) error

	// Forms & Submissions
	CreateFormDefinition(ctx context.Context, form *models.MobileFormDefinition) error
	GetFormDefinition(ctx context.Context, id string) (*models.MobileFormDefinition, error)
	GetFormByCode(ctx context.Context, code string) (*models.MobileFormDefinition, error)
	ListFormDefinitions(ctx context.Context, tenantID string) ([]models.MobileFormDefinition, error)
	InsertMobileSubmission(ctx context.Context, sub *models.MobileSubmission) error
	GetSubmissionByClientID(ctx context.Context, clientID string) (*models.MobileSubmission, error)
	ListSubmissions(ctx context.Context, formID string, workerID string) ([]models.MobileSubmission, error)

	// GPS Breadcrumbs & Geofencing
	InsertGPSBreadcrumbs(ctx context.Context, crumbs []models.GPSBreadcrumb) error
	GetRecentBreadcrumbs(ctx context.Context, workerID string, limit int) ([]models.GPSBreadcrumb, error)
	CreateGeofence(ctx context.Context, g *models.GeofenceZone) error
	ListGeofences(ctx context.Context, tenantID string) ([]models.GeofenceZone, error)

	// Sync & Conflicts
	RecordSyncTransaction(ctx context.Context, tx *models.SyncPushResponse, req *models.SyncPushRequest) error
	RecordConflict(ctx context.Context, c *models.ConflictRecord) error
	GetConflict(ctx context.Context, id string) (*models.ConflictRecord, error)
	ListConflicts(ctx context.Context, tenantID string, status string) ([]models.ConflictRecord, error)
	ResolveConflict(ctx context.Context, id string, resolvedPayload map[string]interface{}, strategy models.ConflictResolutionStrategy, resolvedBy string) error

	// Object Links
	CreateObjectLink(ctx context.Context, link *models.ObjectLink) error
	ListObjectLinks(ctx context.Context, srcType, srcID string) ([]models.ObjectLink, error)
}

// ──────────────────────────────────────────────────────────────────────
// IN-MEMORY STORE IMPLEMENTATION (FOR TEST SUITE & ISOLATED ENVIRONMENTS)
// ──────────────────────────────────────────────────────────────────────

type MemStore struct {
	mu          sync.RWMutex
	gateways    map[string]*models.IoTGateway
	devices     map[string]*models.IoTDevice
	devicesUID  map[string]string // UID -> ID
	sensors     map[string][]models.SensorRegistryItem
	telemetry   map[string][]models.TelemetryRecord // DeviceID -> Records
	configs     map[string]*models.EdgeConfiguration // DeviceID -> Config
	firmwares   map[string]*models.FirmwareRelease   // DeviceType -> Release
	alerts      map[string]*models.IoTAlert
	workers     map[string]*models.FieldWorker
	mobileDevs  map[string]*models.MobileDevice // UUID -> MobileDevice
	visits      map[string]*models.FieldVisit
	tasks       map[string]*models.VisitTask
	forms       map[string]*models.MobileFormDefinition
	formsCode   map[string]string // Code -> ID
	submissions map[string]*models.MobileSubmission // ClientSubmissionID -> Submission
	breadcrumbs map[string][]models.GPSBreadcrumb // WorkerID -> Breadcrumbs
	geofences   map[string]*models.GeofenceZone
	conflicts   map[string]*models.ConflictRecord
	objectLinks []models.ObjectLink
}

func NewMemStore() *MemStore {
	return &MemStore{
		gateways:    make(map[string]*models.IoTGateway),
		devices:     make(map[string]*models.IoTDevice),
		devicesUID:  make(map[string]string),
		sensors:     make(map[string][]models.SensorRegistryItem),
		telemetry:   make(map[string][]models.TelemetryRecord),
		configs:     make(map[string]*models.EdgeConfiguration),
		firmwares:   make(map[string]*models.FirmwareRelease),
		alerts:      make(map[string]*models.IoTAlert),
		workers:     make(map[string]*models.FieldWorker),
		mobileDevs:  make(map[string]*models.MobileDevice),
		visits:      make(map[string]*models.FieldVisit),
		tasks:       make(map[string]*models.VisitTask),
		forms:       make(map[string]*models.MobileFormDefinition),
		formsCode:   make(map[string]string),
		submissions: make(map[string]*models.MobileSubmission),
		breadcrumbs: make(map[string][]models.GPSBreadcrumb),
		geofences:   make(map[string]*models.GeofenceZone),
		conflicts:   make(map[string]*models.ConflictRecord),
		objectLinks: make([]models.ObjectLink, 0),
	}
}

func (m *MemStore) CreateGateway(ctx context.Context, gw *models.IoTGateway) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if gw.ID == "" {
		gw.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	gw.CreatedAt = now
	gw.UpdatedAt = now
	m.gateways[gw.ID] = gw
	return nil
}

func (m *MemStore) GetGateway(ctx context.Context, id string) (*models.IoTGateway, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gw, exists := m.gateways[id]
	if !exists {
		return nil, errors.New("gateway not found")
	}
	return gw, nil
}

func (m *MemStore) ListGateways(ctx context.Context, tenantID string) ([]models.IoTGateway, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.IoTGateway, 0, len(m.gateways))
	for _, gw := range m.gateways {
		if tenantID == "" || gw.TenantID == tenantID {
			res = append(res, *gw)
		}
	}
	return res, nil
}

func (m *MemStore) UpdateGatewayHeartbeat(ctx context.Context, id string, status models.GatewayStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	gw, exists := m.gateways[id]
	if !exists {
		return errors.New("gateway not found")
	}
	now := time.Now().UTC()
	gw.Status = status
	gw.LastHeartbeat = &now
	gw.UpdatedAt = now
	return nil
}

func (m *MemStore) RegisterDevice(ctx context.Context, dev *models.IoTDevice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if dev.ID == "" {
		dev.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	dev.CreatedAt = now
	dev.UpdatedAt = now
	dev.LastSeenAt = &now
	m.devices[dev.ID] = dev
	m.devicesUID[dev.DeviceUID] = dev.ID
	return nil
}

func (m *MemStore) GetDevice(ctx context.Context, id string) (*models.IoTDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	dev, exists := m.devices[id]
	if !exists {
		return nil, errors.New("device not found")
	}
	return dev, nil
}

func (m *MemStore) GetDeviceByUID(ctx context.Context, uid string) (*models.IoTDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, exists := m.devicesUID[uid]
	if !exists {
		return nil, errors.New("device UID not found")
	}
	return m.devices[id], nil
}

func (m *MemStore) ListDevices(ctx context.Context, tenantID string) ([]models.IoTDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.IoTDevice, 0, len(m.devices))
	for _, dev := range m.devices {
		if tenantID == "" || dev.TenantID == tenantID {
			res = append(res, *dev)
		}
	}
	return res, nil
}

func (m *MemStore) UpdateDeviceHeartbeat(ctx context.Context, id string, battery *float64, signal *int, lat *float64, lng *float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	dev, exists := m.devices[id]
	if !exists {
		return errors.New("device not found")
	}
	now := time.Now().UTC()
	dev.LastSeenAt = &now
	dev.Status = models.DeviceStatusActive
	if battery != nil {
		dev.BatteryLevel = battery
	}
	if signal != nil {
		dev.SignalStrengthDBM = signal
	}
	if lat != nil {
		dev.Latitude = lat
	}
	if lng != nil {
		dev.Longitude = lng
	}
	dev.UpdatedAt = now
	return nil
}

func (m *MemStore) RegisterSensor(ctx context.Context, s *models.SensorRegistryItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.CreatedAt = time.Now().UTC()
	m.sensors[s.DeviceID] = append(m.sensors[s.DeviceID], *s)
	return nil
}

func (m *MemStore) ListSensorsByDevice(ctx context.Context, deviceID string) ([]models.SensorRegistryItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sensors[deviceID], nil
}

func (m *MemStore) InsertTelemetryBatch(ctx context.Context, readings []models.TelemetryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for i := range readings {
		if readings[i].ID == "" {
			readings[i].ID = uuid.New().String()
		}
		if readings[i].CreatedAt.IsZero() {
			readings[i].CreatedAt = now
		}
		if readings[i].RecordedAt.IsZero() {
			readings[i].RecordedAt = now
		}
		m.telemetry[readings[i].DeviceID] = append(m.telemetry[readings[i].DeviceID], readings[i])
	}
	return nil
}

func (m *MemStore) GetLatestTelemetry(ctx context.Context, deviceID string, limit int) ([]models.TelemetryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	recs := m.telemetry[deviceID]
	if len(recs) == 0 {
		return []models.TelemetryRecord{}, nil
	}
	if limit <= 0 || limit > len(recs) {
		limit = len(recs)
	}
	start := len(recs) - limit
	res := make([]models.TelemetryRecord, limit)
	copy(res, recs[start:])
	return res, nil
}

func (m *MemStore) GetTelemetryAggregates(ctx context.Context, deviceID string, metricName string, from time.Time, to time.Time) (*models.TelemetryAggregate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	recs := m.telemetry[deviceID]
	var count int64
	var sum, minVal, maxVal float64
	var first, last time.Time
	minVal = math.MaxFloat64
	maxVal = -math.MaxFloat64

	for _, r := range recs {
		if r.MetricName == metricName {
			if (!from.IsZero() && r.RecordedAt.Before(from)) || (!to.IsZero() && r.RecordedAt.After(to)) {
				continue
			}
			count++
			sum += r.Value
			if r.Value < minVal {
				minVal = r.Value
			}
			if r.Value > maxVal {
				maxVal = r.Value
			}
			if first.IsZero() || r.RecordedAt.Before(first) {
				first = r.RecordedAt
			}
			if last.IsZero() || r.RecordedAt.After(last) {
				last = r.RecordedAt
			}
		}
	}

	if count == 0 {
		return &models.TelemetryAggregate{
			MetricName: metricName,
			Count:      0,
		}, nil
	}

	return &models.TelemetryAggregate{
		MetricName: metricName,
		Count:      count,
		Min:        minVal,
		Max:        maxVal,
		Avg:        sum / float64(count),
		FirstTime:  first,
		LastTime:   last,
	}, nil
}

func (m *MemStore) SetDesiredConfig(ctx context.Context, cfg *models.EdgeConfiguration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now
	cfg.SyncStatus = "PENDING"
	m.configs[cfg.DeviceID] = cfg
	return nil
}

func (m *MemStore) GetDeviceConfig(ctx context.Context, deviceID string) (*models.EdgeConfiguration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, exists := m.configs[deviceID]
	if !exists {
		return &models.EdgeConfiguration{
			DeviceID:      deviceID,
			Version:       1,
			DesiredConfig: map[string]interface{}{},
			SyncStatus:    "APPLIED",
		}, nil
	}
	return cfg, nil
}

func (m *MemStore) ReportAppliedConfig(ctx context.Context, deviceID string, reported map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, exists := m.configs[deviceID]
	if !exists {
		cfg = &models.EdgeConfiguration{
			ID:            uuid.New().String(),
			DeviceID:      deviceID,
			Version:       1,
			DesiredConfig: reported,
			TenantID:      "default",
			CreatedAt:     time.Now().UTC(),
		}
		m.configs[deviceID] = cfg
	}
	now := time.Now().UTC()
	cfg.ReportedConfig = reported
	cfg.SyncStatus = "APPLIED"
	cfg.AppliedAt = &now
	cfg.UpdatedAt = now
	return nil
}

func (m *MemStore) CreateFirmware(ctx context.Context, fw *models.FirmwareRelease) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fw.ID == "" {
		fw.ID = uuid.New().String()
	}
	fw.ReleasedAt = time.Now().UTC()
	m.firmwares[fw.DeviceType] = fw
	return nil
}

func (m *MemStore) GetLatestFirmware(ctx context.Context, deviceType string) (*models.FirmwareRelease, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	fw, exists := m.firmwares[deviceType]
	if !exists {
		return nil, errors.New("no firmware release found for device type")
	}
	return fw, nil
}

func (m *MemStore) CreateAlert(ctx context.Context, alert *models.IoTAlert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}
	alert.CreatedAt = time.Now().UTC()
	alert.Status = "ACTIVE"
	m.alerts[alert.ID] = alert
	return nil
}

func (m *MemStore) ListAlerts(ctx context.Context, tenantID string, status string) ([]models.IoTAlert, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.IoTAlert, 0, len(m.alerts))
	for _, a := range m.alerts {
		if (tenantID == "" || a.TenantID == tenantID) && (status == "" || a.Status == status) {
			res = append(res, *a)
		}
	}
	return res, nil
}

func (m *MemStore) ResolveAlert(ctx context.Context, alertID string, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, exists := m.alerts[alertID]
	if !exists {
		return errors.New("alert not found")
	}
	now := time.Now().UTC()
	a.Status = "RESOLVED"
	a.AcknowledgedBy = &resolvedBy
	a.ResolvedAt = &now
	return nil
}

func (m *MemStore) CreateFieldWorker(ctx context.Context, w *models.FieldWorker) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now
	w.IsActive = true
	m.workers[w.ID] = w
	return nil
}

func (m *MemStore) GetFieldWorker(ctx context.Context, id string) (*models.FieldWorker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, exists := m.workers[id]
	if !exists {
		return nil, errors.New("field worker not found")
	}
	return w, nil
}

func (m *MemStore) ListFieldWorkers(ctx context.Context, tenantID string) ([]models.FieldWorker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.FieldWorker, 0, len(m.workers))
	for _, w := range m.workers {
		if tenantID == "" || w.TenantID == tenantID {
			res = append(res, *w)
		}
	}
	return res, nil
}

func (m *MemStore) RegisterMobileDevice(ctx context.Context, dev *models.MobileDevice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if dev.ID == "" {
		dev.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	dev.CreatedAt = now
	dev.UpdatedAt = now
	dev.Status = "ACTIVE"
	m.mobileDevs[dev.DeviceUUID] = dev
	return nil
}

func (m *MemStore) GetMobileDevice(ctx context.Context, devUUID string) (*models.MobileDevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	dev, exists := m.mobileDevs[devUUID]
	if !exists {
		return nil, errors.New("mobile device not found")
	}
	return dev, nil
}

func (m *MemStore) CreateFieldVisit(ctx context.Context, v *models.FieldVisit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now
	m.visits[v.ID] = v
	return nil
}

func (m *MemStore) GetFieldVisit(ctx context.Context, id string) (*models.FieldVisit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, exists := m.visits[id]
	if !exists {
		return nil, errors.New("field visit not found")
	}
	return v, nil
}

func (m *MemStore) ListVisitsByWorker(ctx context.Context, workerID string) ([]models.FieldVisit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.FieldVisit, 0)
	for _, v := range m.visits {
		if v.WorkerID == workerID {
			res = append(res, *v)
		}
	}
	return res, nil
}

func (m *MemStore) UpdateVisitStatus(ctx context.Context, id string, status string, start *time.Time, end *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, exists := m.visits[id]
	if !exists {
		return errors.New("visit not found")
	}
	v.Status = status
	if start != nil {
		v.ActualStart = start
	}
	if end != nil {
		v.ActualEnd = end
	}
	v.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemStore) CreateVisitTask(ctx context.Context, t *models.VisitTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.CreatedAt = time.Now().UTC()
	m.tasks[t.ID] = t
	return nil
}

func (m *MemStore) ListTasksByVisit(ctx context.Context, visitID string) ([]models.VisitTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.VisitTask, 0)
	for _, t := range m.tasks {
		if t.VisitID == visitID {
			res = append(res, *t)
		}
	}
	return res, nil
}

func (m *MemStore) UpdateVisitTask(ctx context.Context, t *models.VisitTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, exists := m.tasks[t.ID]
	if !exists {
		m.tasks[t.ID] = t
		return nil
	}
	existing.Status = t.Status
	existing.ResultSummary = t.ResultSummary
	existing.CompletedAt = t.CompletedAt
	return nil
}

func (m *MemStore) CreateFormDefinition(ctx context.Context, form *models.MobileFormDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if form.ID == "" {
		form.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	form.CreatedAt = now
	form.UpdatedAt = now
	m.forms[form.ID] = form
	m.formsCode[form.FormCode] = form.ID
	return nil
}

func (m *MemStore) GetFormDefinition(ctx context.Context, id string) (*models.MobileFormDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, exists := m.forms[id]
	if !exists {
		return nil, errors.New("form not found")
	}
	return f, nil
}

func (m *MemStore) GetFormByCode(ctx context.Context, code string) (*models.MobileFormDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, exists := m.formsCode[code]
	if !exists {
		return nil, errors.New("form code not found")
	}
	return m.forms[id], nil
}

func (m *MemStore) ListFormDefinitions(ctx context.Context, tenantID string) ([]models.MobileFormDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.MobileFormDefinition, 0, len(m.forms))
	for _, f := range m.forms {
		if tenantID == "" || f.TenantID == tenantID {
			res = append(res, *f)
		}
	}
	return res, nil
}

func (m *MemStore) InsertMobileSubmission(ctx context.Context, sub *models.MobileSubmission) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	sub.CreatedAt = now
	sub.SyncedAt = now
	m.submissions[sub.ClientSubmissionID] = sub
	return nil
}

func (m *MemStore) GetSubmissionByClientID(ctx context.Context, clientID string) (*models.MobileSubmission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sub, exists := m.submissions[clientID]
	if !exists {
		return nil, errors.New("submission not found")
	}
	return sub, nil
}

func (m *MemStore) ListSubmissions(ctx context.Context, formID string, workerID string) ([]models.MobileSubmission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.MobileSubmission, 0)
	for _, s := range m.submissions {
		if (formID == "" || s.FormID == formID) && (workerID == "" || s.WorkerID == workerID) {
			res = append(res, *s)
		}
	}
	return res, nil
}

func (m *MemStore) InsertGPSBreadcrumbs(ctx context.Context, crumbs []models.GPSBreadcrumb) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	for i := range crumbs {
		if crumbs[i].ID == "" {
			crumbs[i].ID = uuid.New().String()
		}
		if crumbs[i].CreatedAt.IsZero() {
			crumbs[i].CreatedAt = now
		}
		if crumbs[i].RecordedAt.IsZero() {
			crumbs[i].RecordedAt = now
		}
		m.breadcrumbs[crumbs[i].WorkerID] = append(m.breadcrumbs[crumbs[i].WorkerID], crumbs[i])
	}
	return nil
}

func (m *MemStore) GetRecentBreadcrumbs(ctx context.Context, workerID string, limit int) ([]models.GPSBreadcrumb, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	crumbs := m.breadcrumbs[workerID]
	if len(crumbs) == 0 {
		return []models.GPSBreadcrumb{}, nil
	}
	if limit <= 0 || limit > len(crumbs) {
		limit = len(crumbs)
	}
	start := len(crumbs) - limit
	res := make([]models.GPSBreadcrumb, limit)
	copy(res, crumbs[start:])
	return res, nil
}

func (m *MemStore) CreateGeofence(ctx context.Context, g *models.GeofenceZone) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	g.CreatedAt = time.Now().UTC()
	m.geofences[g.ID] = g
	return nil
}

func (m *MemStore) ListGeofences(ctx context.Context, tenantID string) ([]models.GeofenceZone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.GeofenceZone, 0, len(m.geofences))
	for _, g := range m.geofences {
		if tenantID == "" || g.TenantID == tenantID {
			res = append(res, *g)
		}
	}
	return res, nil
}

func (m *MemStore) RecordSyncTransaction(ctx context.Context, tx *models.SyncPushResponse, req *models.SyncPushRequest) error {
	// MemStore recorded in-memory
	return nil
}

func (m *MemStore) RecordConflict(ctx context.Context, c *models.ConflictRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	c.CreatedAt = time.Now().UTC()
	m.conflicts[c.ID] = c
	return nil
}

func (m *MemStore) GetConflict(ctx context.Context, id string) (*models.ConflictRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, exists := m.conflicts[id]
	if !exists {
		return nil, errors.New("conflict record not found")
	}
	return c, nil
}

func (m *MemStore) ListConflicts(ctx context.Context, tenantID string, status string) ([]models.ConflictRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.ConflictRecord, 0, len(m.conflicts))
	for _, c := range m.conflicts {
		if (tenantID == "" || c.TenantID == tenantID) && (status == "" || c.Status == status) {
			res = append(res, *c)
		}
	}
	return res, nil
}

func (m *MemStore) ResolveConflict(ctx context.Context, id string, resolvedPayload map[string]interface{}, strategy models.ConflictResolutionStrategy, resolvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, exists := m.conflicts[id]
	if !exists {
		return errors.New("conflict record not found")
	}
	now := time.Now().UTC()
	c.ResolvedPayload = resolvedPayload
	c.ResolutionStrategy = strategy
	c.Status = "RESOLVED_MANUAL"
	c.ResolvedBy = &resolvedBy
	c.ResolvedAt = &now
	return nil
}

func (m *MemStore) CreateObjectLink(ctx context.Context, link *models.ObjectLink) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	link.ID = len(m.objectLinks) + 1
	link.CreatedAt = time.Now().UTC()
	m.objectLinks = append(m.objectLinks, *link)
	return nil
}

func (m *MemStore) ListObjectLinks(ctx context.Context, srcType, srcID string) ([]models.ObjectLink, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]models.ObjectLink, 0)
	for _, l := range m.objectLinks {
		if l.SourceType == srcType && l.SourceID == srcID {
			res = append(res, l)
		}
	}
	return res, nil
}

// ──────────────────────────────────────────────────────────────────────
// POSTGRESQL STORE IMPLEMENTATION
// ──────────────────────────────────────────────────────────────────────

type PGStore struct {
	db *sql.DB
}

func NewPGStore(db *sql.DB) *PGStore {
	return &PGStore{db: db}
}

func (p *PGStore) CreateGateway(ctx context.Context, gw *models.IoTGateway) error {
	if gw.ID == "" {
		gw.ID = uuid.New().String()
	}
	metaJSON, _ := json.Marshal(gw.Metadata)
	query := `
		INSERT INTO statiot.iot_gateways (
			id, name, gateway_code, ip_address, mac_address, firmware_version,
			status, latitude, longitude, location_name, tenant_id, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		gw.ID, gw.Name, gw.GatewayCode, gw.IPAddress, gw.MACAddress, gw.FirmwareVersion,
		gw.Status, gw.Latitude, gw.Longitude, gw.LocationName, gw.TenantID, metaJSON,
	)
	return err
}

func (p *PGStore) GetGateway(ctx context.Context, id string) (*models.IoTGateway, error) {
	query := `
		SELECT id, name, gateway_code, ip_address, mac_address, firmware_version,
		       status, latitude, longitude, location_name, tenant_id, metadata, last_heartbeat, created_at, updated_at
		FROM statiot.iot_gateways WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var gw models.IoTGateway
	var metaJSON []byte
	err := row.Scan(
		&gw.ID, &gw.Name, &gw.GatewayCode, &gw.IPAddress, &gw.MACAddress, &gw.FirmwareVersion,
		&gw.Status, &gw.Latitude, &gw.Longitude, &gw.LocationName, &gw.TenantID, &metaJSON,
		&gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(metaJSON) > 0 {
		_ = json.Unmarshal(metaJSON, &gw.Metadata)
	}
	return &gw, nil
}

func (p *PGStore) ListGateways(ctx context.Context, tenantID string) ([]models.IoTGateway, error) {
	query := `
		SELECT id, name, gateway_code, ip_address, mac_address, firmware_version,
		       status, latitude, longitude, location_name, tenant_id, metadata, last_heartbeat, created_at, updated_at
		FROM statiot.iot_gateways WHERE ($1 = '' OR tenant_id = $1)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.IoTGateway
	for rows.Next() {
		var gw models.IoTGateway
		var metaJSON []byte
		if err := rows.Scan(
			&gw.ID, &gw.Name, &gw.GatewayCode, &gw.IPAddress, &gw.MACAddress, &gw.FirmwareVersion,
			&gw.Status, &gw.Latitude, &gw.Longitude, &gw.LocationName, &gw.TenantID, &metaJSON,
			&gw.LastHeartbeat, &gw.CreatedAt, &gw.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &gw.Metadata)
		}
		result = append(result, gw)
	}
	return result, nil
}

func (p *PGStore) UpdateGatewayHeartbeat(ctx context.Context, id string, status models.GatewayStatus) error {
	query := `
		UPDATE statiot.iot_gateways
		SET status = $2, last_heartbeat = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, status)
	return err
}

func (p *PGStore) RegisterDevice(ctx context.Context, dev *models.IoTDevice) error {
	if dev.ID == "" {
		dev.ID = uuid.New().String()
	}
	cfgJSON, _ := json.Marshal(dev.ConfigPayload)
	query := `
		INSERT INTO statiot.iot_devices (
			id, device_uid, name, device_type, protocol, gateway_id, auth_token_hash,
			firmware_version, status, battery_level, signal_strength_dbm, latitude,
			longitude, altitude, tenant_id, org_id, config_payload, last_seen_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (device_uid) DO UPDATE SET
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			firmware_version = EXCLUDED.firmware_version,
			last_seen_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		dev.ID, dev.DeviceUID, dev.Name, dev.DeviceType, dev.Protocol, dev.GatewayID, dev.AuthTokenHash,
		dev.FirmwareVersion, dev.Status, dev.BatteryLevel, dev.SignalStrengthDBM, dev.Latitude,
		dev.Longitude, dev.Altitude, dev.TenantID, dev.OrgID, cfgJSON,
	)
	return err
}

func (p *PGStore) GetDevice(ctx context.Context, id string) (*models.IoTDevice, error) {
	query := `
		SELECT id, device_uid, name, device_type, protocol, gateway_id, firmware_version,
		       status, battery_level, signal_strength_dbm, latitude, longitude, altitude,
		       tenant_id, org_id, config_payload, last_seen_at, created_at, updated_at
		FROM statiot.iot_devices WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var dev models.IoTDevice
	var cfgJSON []byte
	err := row.Scan(
		&dev.ID, &dev.DeviceUID, &dev.Name, &dev.DeviceType, &dev.Protocol, &dev.GatewayID,
		&dev.FirmwareVersion, &dev.Status, &dev.BatteryLevel, &dev.SignalStrengthDBM,
		&dev.Latitude, &dev.Longitude, &dev.Altitude, &dev.TenantID, &dev.OrgID,
		&cfgJSON, &dev.LastSeenAt, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(cfgJSON) > 0 {
		_ = json.Unmarshal(cfgJSON, &dev.ConfigPayload)
	}
	return &dev, nil
}

func (p *PGStore) GetDeviceByUID(ctx context.Context, uid string) (*models.IoTDevice, error) {
	query := `
		SELECT id, device_uid, name, device_type, protocol, gateway_id, firmware_version,
		       status, battery_level, signal_strength_dbm, latitude, longitude, altitude,
		       tenant_id, org_id, config_payload, last_seen_at, created_at, updated_at
		FROM statiot.iot_devices WHERE device_uid = $1
	`
	row := p.db.QueryRowContext(ctx, query, uid)
	var dev models.IoTDevice
	var cfgJSON []byte
	err := row.Scan(
		&dev.ID, &dev.DeviceUID, &dev.Name, &dev.DeviceType, &dev.Protocol, &dev.GatewayID,
		&dev.FirmwareVersion, &dev.Status, &dev.BatteryLevel, &dev.SignalStrengthDBM,
		&dev.Latitude, &dev.Longitude, &dev.Altitude, &dev.TenantID, &dev.OrgID,
		&cfgJSON, &dev.LastSeenAt, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(cfgJSON) > 0 {
		_ = json.Unmarshal(cfgJSON, &dev.ConfigPayload)
	}
	return &dev, nil
}

func (p *PGStore) ListDevices(ctx context.Context, tenantID string) ([]models.IoTDevice, error) {
	query := `
		SELECT id, device_uid, name, device_type, protocol, gateway_id, firmware_version,
		       status, battery_level, signal_strength_dbm, latitude, longitude, altitude,
		       tenant_id, org_id, config_payload, last_seen_at, created_at, updated_at
		FROM statiot.iot_devices WHERE ($1 = '' OR tenant_id = $1)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.IoTDevice
	for rows.Next() {
		var dev models.IoTDevice
		var cfgJSON []byte
		if err := rows.Scan(
			&dev.ID, &dev.DeviceUID, &dev.Name, &dev.DeviceType, &dev.Protocol, &dev.GatewayID,
			&dev.FirmwareVersion, &dev.Status, &dev.BatteryLevel, &dev.SignalStrengthDBM,
			&dev.Latitude, &dev.Longitude, &dev.Altitude, &dev.TenantID, &dev.OrgID,
			&cfgJSON, &dev.LastSeenAt, &dev.CreatedAt, &dev.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(cfgJSON) > 0 {
			_ = json.Unmarshal(cfgJSON, &dev.ConfigPayload)
		}
		res = append(res, dev)
	}
	return res, nil
}

func (p *PGStore) UpdateDeviceHeartbeat(ctx context.Context, id string, battery *float64, signal *int, lat *float64, lng *float64) error {
	query := `
		UPDATE statiot.iot_devices
		SET last_seen_at = CURRENT_TIMESTAMP,
		    status = 'ACTIVE',
		    battery_level = COALESCE($2, battery_level),
		    signal_strength_dbm = COALESCE($3, signal_strength_dbm),
		    latitude = COALESCE($4, latitude),
		    longitude = COALESCE($5, longitude),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, battery, signal, lat, lng)
	return err
}

func (p *PGStore) RegisterSensor(ctx context.Context, s *models.SensorRegistryItem) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	query := `
		INSERT INTO statiot.sensor_registry (
			id, device_id, sensor_code, metric_name, unit_of_measure,
			min_threshold, max_threshold, calibration_factor, calibration_offset, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP)
		ON CONFLICT (device_id, sensor_code) DO UPDATE SET
			metric_name = EXCLUDED.metric_name,
			unit_of_measure = EXCLUDED.unit_of_measure,
			min_threshold = EXCLUDED.min_threshold,
			max_threshold = EXCLUDED.max_threshold
	`
	_, err := p.db.ExecContext(ctx, query,
		s.ID, s.DeviceID, s.SensorCode, s.MetricName, s.UnitOfMeasure,
		s.MinThreshold, s.MaxThreshold, s.CalibrationFactor, s.CalibrationOffset, s.Status,
	)
	return err
}

func (p *PGStore) ListSensorsByDevice(ctx context.Context, deviceID string) ([]models.SensorRegistryItem, error) {
	query := `
		SELECT id, device_id, sensor_code, metric_name, unit_of_measure,
		       min_threshold, max_threshold, calibration_factor, calibration_offset, status, created_at
		FROM statiot.sensor_registry WHERE device_id = $1
	`
	rows, err := p.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.SensorRegistryItem
	for rows.Next() {
		var s models.SensorRegistryItem
		if err := rows.Scan(
			&s.ID, &s.DeviceID, &s.SensorCode, &s.MetricName, &s.UnitOfMeasure,
			&s.MinThreshold, &s.MaxThreshold, &s.CalibrationFactor, &s.CalibrationOffset, &s.Status, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, nil
}

func (p *PGStore) InsertTelemetryBatch(ctx context.Context, readings []models.TelemetryRecord) error {
	if len(readings) == 0 {
		return nil
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO statiot.telemetry_records (
			id, device_id, sensor_code, metric_name, value, raw_payload, quality_score, is_anomaly, tenant_id, recorded_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range readings {
		if r.ID == "" {
			r.ID = uuid.New().String()
		}
		if r.RecordedAt.IsZero() {
			r.RecordedAt = time.Now().UTC()
		}
		rawJSON, _ := json.Marshal(r.RawPayload)
		if _, err := stmt.ExecContext(ctx,
			r.ID, r.DeviceID, r.SensorCode, r.MetricName, r.Value, rawJSON, r.QualityScore, r.IsAnomaly, r.TenantID, r.RecordedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *PGStore) GetLatestTelemetry(ctx context.Context, deviceID string, limit int) ([]models.TelemetryRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, device_id, sensor_code, metric_name, value, raw_payload, quality_score, is_anomaly, tenant_id, recorded_at, created_at
		FROM statiot.telemetry_records WHERE device_id = $1
		ORDER BY recorded_at DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.TelemetryRecord
	for rows.Next() {
		var r models.TelemetryRecord
		var rawJSON []byte
		if err := rows.Scan(
			&r.ID, &r.DeviceID, &r.SensorCode, &r.MetricName, &r.Value, &rawJSON,
			&r.QualityScore, &r.IsAnomaly, &r.TenantID, &r.RecordedAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(rawJSON) > 0 {
			_ = json.Unmarshal(rawJSON, &r.RawPayload)
		}
		res = append(res, r)
	}
	return res, nil
}

func (p *PGStore) GetTelemetryAggregates(ctx context.Context, deviceID string, metricName string, from time.Time, to time.Time) (*models.TelemetryAggregate, error) {
	query := `
		SELECT COUNT(*), COALESCE(MIN(value), 0), COALESCE(MAX(value), 0), COALESCE(AVG(value), 0),
		       COALESCE(MIN(recorded_at), CURRENT_TIMESTAMP), COALESCE(MAX(recorded_at), CURRENT_TIMESTAMP)
		FROM statiot.telemetry_records
		WHERE device_id = $1 AND metric_name = $2
		  AND recorded_at >= $3 AND recorded_at <= $4
	`
	row := p.db.QueryRowContext(ctx, query, deviceID, metricName, from, to)
	var agg models.TelemetryAggregate
	agg.MetricName = metricName
	err := row.Scan(&agg.Count, &agg.Min, &agg.Max, &agg.Avg, &agg.FirstTime, &agg.LastTime)
	if err != nil {
		return nil, err
	}
	return &agg, nil
}

func (p *PGStore) SetDesiredConfig(ctx context.Context, cfg *models.EdgeConfiguration) error {
	if cfg.ID == "" {
		cfg.ID = uuid.New().String()
	}
	desiredJSON, _ := json.Marshal(cfg.DesiredConfig)
	query := `
		INSERT INTO statiot.edge_configurations (
			id, device_id, version, desired_config, sync_status, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, 'PENDING', $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query, cfg.ID, cfg.DeviceID, cfg.Version, desiredJSON, cfg.TenantID)
	return err
}

func (p *PGStore) GetDeviceConfig(ctx context.Context, deviceID string) (*models.EdgeConfiguration, error) {
	query := `
		SELECT id, device_id, version, desired_config, reported_config, sync_status, applied_at, tenant_id, created_at, updated_at
		FROM statiot.edge_configurations WHERE device_id = $1
		ORDER BY created_at DESC LIMIT 1
	`
	row := p.db.QueryRowContext(ctx, query, deviceID)
	var cfg models.EdgeConfiguration
	var desiredJSON, reportedJSON []byte
	err := row.Scan(
		&cfg.ID, &cfg.DeviceID, &cfg.Version, &desiredJSON, &reportedJSON,
		&cfg.SyncStatus, &cfg.AppliedAt, &cfg.TenantID, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &models.EdgeConfiguration{
				DeviceID:      deviceID,
				Version:       1,
				DesiredConfig: map[string]interface{}{},
				SyncStatus:    "APPLIED",
			}, nil
		}
		return nil, err
	}
	if len(desiredJSON) > 0 {
		_ = json.Unmarshal(desiredJSON, &cfg.DesiredConfig)
	}
	if len(reportedJSON) > 0 {
		_ = json.Unmarshal(reportedJSON, &cfg.ReportedConfig)
	}
	return &cfg, nil
}

func (p *PGStore) ReportAppliedConfig(ctx context.Context, deviceID string, reported map[string]interface{}) error {
	repJSON, _ := json.Marshal(reported)
	query := `
		UPDATE statiot.edge_configurations
		SET reported_config = $2, sync_status = 'APPLIED', applied_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = (SELECT id FROM statiot.edge_configurations WHERE device_id = $1 ORDER BY created_at DESC LIMIT 1)
	`
	_, err := p.db.ExecContext(ctx, query, deviceID, repJSON)
	return err
}

func (p *PGStore) CreateFirmware(ctx context.Context, fw *models.FirmwareRelease) error {
	if fw.ID == "" {
		fw.ID = uuid.New().String()
	}
	query := `
		INSERT INTO statiot.firmware_releases (
			id, device_type, version, binary_url, checksum_sha256, changelog, is_critical, tenant_id, released_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		fw.ID, fw.DeviceType, fw.Version, fw.BinaryURL, fw.ChecksumSHA256, fw.Changelog, fw.IsCritical, fw.TenantID,
	)
	return err
}

func (p *PGStore) GetLatestFirmware(ctx context.Context, deviceType string) (*models.FirmwareRelease, error) {
	query := `
		SELECT id, device_type, version, binary_url, checksum_sha256, changelog, is_critical, tenant_id, released_at
		FROM statiot.firmware_releases WHERE device_type = $1
		ORDER BY released_at DESC LIMIT 1
	`
	row := p.db.QueryRowContext(ctx, query, deviceType)
	var fw models.FirmwareRelease
	err := row.Scan(
		&fw.ID, &fw.DeviceType, &fw.Version, &fw.BinaryURL, &fw.ChecksumSHA256,
		&fw.Changelog, &fw.IsCritical, &fw.TenantID, &fw.ReleasedAt,
	)
	if err != nil {
		return nil, err
	}
	return &fw, nil
}

func (p *PGStore) CreateAlert(ctx context.Context, alert *models.IoTAlert) error {
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}
	query := `
		INSERT INTO statiot.iot_alerts (
			id, device_id, sensor_code, alert_type, severity, message, value, threshold, status, tenant_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'ACTIVE', $9, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		alert.ID, alert.DeviceID, alert.SensorCode, alert.AlertType, alert.Severity,
		alert.Message, alert.Value, alert.Threshold, alert.TenantID,
	)
	return err
}

func (p *PGStore) ListAlerts(ctx context.Context, tenantID string, status string) ([]models.IoTAlert, error) {
	query := `
		SELECT id, device_id, sensor_code, alert_type, severity, message, value, threshold, status, acknowledged_by, tenant_id, created_at, resolved_at
		FROM statiot.iot_alerts
		WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.IoTAlert
	for rows.Next() {
		var a models.IoTAlert
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.SensorCode, &a.AlertType, &a.Severity, &a.Message,
			&a.Value, &a.Threshold, &a.Status, &a.AcknowledgedBy, &a.TenantID, &a.CreatedAt, &a.ResolvedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, nil
}

func (p *PGStore) ResolveAlert(ctx context.Context, alertID string, resolvedBy string) error {
	query := `
		UPDATE statiot.iot_alerts
		SET status = 'RESOLVED', acknowledged_by = $2, resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, alertID, resolvedBy)
	return err
}

func (p *PGStore) CreateFieldWorker(ctx context.Context, w *models.FieldWorker) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	query := `
		INSERT INTO statiot.field_workers (
			id, user_id, full_name, phone_number, team_name, role, assigned_district, tenant_id, org_id, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		w.ID, w.UserID, w.FullName, w.PhoneNumber, w.TeamName, w.Role, w.AssignedDistrict, w.TenantID, w.OrgID,
	)
	return err
}

func (p *PGStore) GetFieldWorker(ctx context.Context, id string) (*models.FieldWorker, error) {
	query := `
		SELECT id, user_id, full_name, phone_number, team_name, role, assigned_district, tenant_id, org_id, is_active, created_at, updated_at
		FROM statiot.field_workers WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var w models.FieldWorker
	err := row.Scan(
		&w.ID, &w.UserID, &w.FullName, &w.PhoneNumber, &w.TeamName, &w.Role,
		&w.AssignedDistrict, &w.TenantID, &w.OrgID, &w.IsActive, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (p *PGStore) ListFieldWorkers(ctx context.Context, tenantID string) ([]models.FieldWorker, error) {
	query := `
		SELECT id, user_id, full_name, phone_number, team_name, role, assigned_district, tenant_id, org_id, is_active, created_at, updated_at
		FROM statiot.field_workers WHERE ($1 = '' OR tenant_id = $1)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.FieldWorker
	for rows.Next() {
		var w models.FieldWorker
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.FullName, &w.PhoneNumber, &w.TeamName, &w.Role,
			&w.AssignedDistrict, &w.TenantID, &w.OrgID, &w.IsActive, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, w)
	}
	return res, nil
}

func (p *PGStore) RegisterMobileDevice(ctx context.Context, dev *models.MobileDevice) error {
	if dev.ID == "" {
		dev.ID = uuid.New().String()
	}
	query := `
		INSERT INTO statiot.mobile_devices (
			id, device_uuid, assigned_worker_id, platform, app_version, os_version,
			battery_level, storage_free_mb, last_sync_at, tenant_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, $9, 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (device_uuid) DO UPDATE SET
			app_version = EXCLUDED.app_version,
			battery_level = EXCLUDED.battery_level,
			storage_free_mb = EXCLUDED.storage_free_mb,
			last_sync_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		dev.ID, dev.DeviceUUID, dev.AssignedWorkerID, dev.Platform, dev.AppVersion, dev.OSVersion,
		dev.BatteryLevel, dev.StorageFreeMB, dev.TenantID,
	)
	return err
}

func (p *PGStore) GetMobileDevice(ctx context.Context, devUUID string) (*models.MobileDevice, error) {
	query := `
		SELECT id, device_uuid, assigned_worker_id, platform, app_version, os_version,
		       battery_level, storage_free_mb, last_sync_at, tenant_id, status, created_at, updated_at
		FROM statiot.mobile_devices WHERE device_uuid = $1
	`
	row := p.db.QueryRowContext(ctx, query, devUUID)
	var dev models.MobileDevice
	err := row.Scan(
		&dev.ID, &dev.DeviceUUID, &dev.AssignedWorkerID, &dev.Platform, &dev.AppVersion, &dev.OSVersion,
		&dev.BatteryLevel, &dev.StorageFreeMB, &dev.LastSyncAt, &dev.TenantID, &dev.Status, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

func (p *PGStore) CreateFieldVisit(ctx context.Context, v *models.FieldVisit) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	query := `
		INSERT INTO statiot.field_visits (
			id, worker_id, title, target_entity_type, target_entity_id, scheduled_start, scheduled_end,
			status, latitude, longitude, notes, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		v.ID, v.WorkerID, v.Title, v.TargetEntityType, v.TargetEntityID, v.ScheduledStart, v.ScheduledEnd,
		v.Status, v.Latitude, v.Longitude, v.Notes, v.TenantID,
	)
	return err
}

func (p *PGStore) GetFieldVisit(ctx context.Context, id string) (*models.FieldVisit, error) {
	query := `
		SELECT id, worker_id, title, target_entity_type, target_entity_id, scheduled_start, scheduled_end,
		       actual_start, actual_end, status, latitude, longitude, notes, tenant_id, created_at, updated_at
		FROM statiot.field_visits WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var v models.FieldVisit
	err := row.Scan(
		&v.ID, &v.WorkerID, &v.Title, &v.TargetEntityType, &v.TargetEntityID, &v.ScheduledStart, &v.ScheduledEnd,
		&v.ActualStart, &v.ActualEnd, &v.Status, &v.Latitude, &v.Longitude, &v.Notes, &v.TenantID, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (p *PGStore) ListVisitsByWorker(ctx context.Context, workerID string) ([]models.FieldVisit, error) {
	query := `
		SELECT id, worker_id, title, target_entity_type, target_entity_id, scheduled_start, scheduled_end,
		       actual_start, actual_end, status, latitude, longitude, notes, tenant_id, created_at, updated_at
		FROM statiot.field_visits WHERE worker_id = $1
		ORDER BY scheduled_start ASC
	`
	rows, err := p.db.QueryContext(ctx, query, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.FieldVisit
	for rows.Next() {
		var v models.FieldVisit
		if err := rows.Scan(
			&v.ID, &v.WorkerID, &v.Title, &v.TargetEntityType, &v.TargetEntityID, &v.ScheduledStart, &v.ScheduledEnd,
			&v.ActualStart, &v.ActualEnd, &v.Status, &v.Latitude, &v.Longitude, &v.Notes, &v.TenantID, &v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, nil
}

func (p *PGStore) UpdateVisitStatus(ctx context.Context, id string, status string, start *time.Time, end *time.Time) error {
	query := `
		UPDATE statiot.field_visits
		SET status = $2, actual_start = COALESCE($3, actual_start), actual_end = COALESCE($4, actual_end), updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, status, start, end)
	return err
}

func (p *PGStore) CreateVisitTask(ctx context.Context, t *models.VisitTask) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	sumJSON, _ := json.Marshal(t.ResultSummary)
	query := `
		INSERT INTO statiot.visit_tasks (
			id, visit_id, title, task_type, form_id, status, result_summary, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		t.ID, t.VisitID, t.Title, t.TaskType, t.FormID, t.Status, sumJSON,
	)
	return err
}

func (p *PGStore) ListTasksByVisit(ctx context.Context, visitID string) ([]models.VisitTask, error) {
	query := `
		SELECT id, visit_id, title, task_type, form_id, status, result_summary, completed_at, created_at
		FROM statiot.visit_tasks WHERE visit_id = $1
	`
	rows, err := p.db.QueryContext(ctx, query, visitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.VisitTask
	for rows.Next() {
		var t models.VisitTask
		var sumJSON []byte
		if err := rows.Scan(
			&t.ID, &t.VisitID, &t.Title, &t.TaskType, &t.FormID, &t.Status, &sumJSON, &t.CompletedAt, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(sumJSON) > 0 {
			_ = json.Unmarshal(sumJSON, &t.ResultSummary)
		}
		res = append(res, t)
	}
	return res, nil
}

func (p *PGStore) UpdateVisitTask(ctx context.Context, t *models.VisitTask) error {
	sumJSON, _ := json.Marshal(t.ResultSummary)
	query := `
		UPDATE statiot.visit_tasks
		SET status = $2, result_summary = $3, completed_at = $4
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, t.ID, t.Status, sumJSON, t.CompletedAt)
	return err
}

func (p *PGStore) CreateFormDefinition(ctx context.Context, form *models.MobileFormDefinition) error {
	if form.ID == "" {
		form.ID = uuid.New().String()
	}
	defJSON, _ := json.Marshal(form.SchemaDefinition)
	query := `
		INSERT INTO statiot.mobile_form_definitions (
			id, form_code, title, version, schema_definition, is_active, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (form_code) DO UPDATE SET
			title = EXCLUDED.title,
			version = EXCLUDED.version,
			schema_definition = EXCLUDED.schema_definition,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		form.ID, form.FormCode, form.Title, form.Version, defJSON, form.IsActive, form.TenantID,
	)
	return err
}

func (p *PGStore) GetFormDefinition(ctx context.Context, id string) (*models.MobileFormDefinition, error) {
	query := `
		SELECT id, form_code, title, version, schema_definition, is_active, tenant_id, created_at, updated_at
		FROM statiot.mobile_form_definitions WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var form models.MobileFormDefinition
	var defJSON []byte
	err := row.Scan(
		&form.ID, &form.FormCode, &form.Title, &form.Version, &defJSON,
		&form.IsActive, &form.TenantID, &form.CreatedAt, &form.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(defJSON) > 0 {
		_ = json.Unmarshal(defJSON, &form.SchemaDefinition)
	}
	return &form, nil
}

func (p *PGStore) GetFormByCode(ctx context.Context, code string) (*models.MobileFormDefinition, error) {
	query := `
		SELECT id, form_code, title, version, schema_definition, is_active, tenant_id, created_at, updated_at
		FROM statiot.mobile_form_definitions WHERE form_code = $1
	`
	row := p.db.QueryRowContext(ctx, query, code)
	var form models.MobileFormDefinition
	var defJSON []byte
	err := row.Scan(
		&form.ID, &form.FormCode, &form.Title, &form.Version, &defJSON,
		&form.IsActive, &form.TenantID, &form.CreatedAt, &form.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(defJSON) > 0 {
		_ = json.Unmarshal(defJSON, &form.SchemaDefinition)
	}
	return &form, nil
}

func (p *PGStore) ListFormDefinitions(ctx context.Context, tenantID string) ([]models.MobileFormDefinition, error) {
	query := `
		SELECT id, form_code, title, version, schema_definition, is_active, tenant_id, created_at, updated_at
		FROM statiot.mobile_form_definitions WHERE ($1 = '' OR tenant_id = $1)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.MobileFormDefinition
	for rows.Next() {
		var form models.MobileFormDefinition
		var defJSON []byte
		if err := rows.Scan(
			&form.ID, &form.FormCode, &form.Title, &form.Version, &defJSON,
			&form.IsActive, &form.TenantID, &form.CreatedAt, &form.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(defJSON) > 0 {
			_ = json.Unmarshal(defJSON, &form.SchemaDefinition)
		}
		res = append(res, form)
	}
	return res, nil
}

func (p *PGStore) InsertMobileSubmission(ctx context.Context, sub *models.MobileSubmission) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	dataJSON, _ := json.Marshal(sub.DataPayload)
	geoJSON, _ := json.Marshal(sub.GeoPoint)
	query := `
		INSERT INTO statiot.mobile_submissions (
			id, client_submission_id, form_id, worker_id, visit_id, data_payload,
			geo_point, collected_at, synced_at, sync_status, tenant_id, version, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, $9, $10, $11, CURRENT_TIMESTAMP)
		ON CONFLICT (client_submission_id) DO UPDATE SET
			sync_status = EXCLUDED.sync_status,
			version = EXCLUDED.version
	`
	_, err := p.db.ExecContext(ctx, query,
		sub.ID, sub.ClientSubmissionID, sub.FormID, sub.WorkerID, sub.VisitID, dataJSON,
		geoJSON, sub.CollectedAt, sub.SyncStatus, sub.TenantID, sub.Version,
	)
	return err
}

func (p *PGStore) GetSubmissionByClientID(ctx context.Context, clientID string) (*models.MobileSubmission, error) {
	query := `
		SELECT id, client_submission_id, form_id, worker_id, visit_id, data_payload,
		       geo_point, collected_at, synced_at, sync_status, tenant_id, version, created_at
		FROM statiot.mobile_submissions WHERE client_submission_id = $1
	`
	row := p.db.QueryRowContext(ctx, query, clientID)
	var sub models.MobileSubmission
	var dataJSON, geoJSON []byte
	err := row.Scan(
		&sub.ID, &sub.ClientSubmissionID, &sub.FormID, &sub.WorkerID, &sub.VisitID, &dataJSON,
		&geoJSON, &sub.CollectedAt, &sub.SyncedAt, &sub.SyncStatus, &sub.TenantID, &sub.Version, &sub.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(dataJSON) > 0 {
		_ = json.Unmarshal(dataJSON, &sub.DataPayload)
	}
	if len(geoJSON) > 0 {
		_ = json.Unmarshal(geoJSON, &sub.GeoPoint)
	}
	return &sub, nil
}

func (p *PGStore) ListSubmissions(ctx context.Context, formID string, workerID string) ([]models.MobileSubmission, error) {
	query := `
		SELECT id, client_submission_id, form_id, worker_id, visit_id, data_payload,
		       geo_point, collected_at, synced_at, sync_status, tenant_id, version, created_at
		FROM statiot.mobile_submissions
		WHERE ($1 = '' OR form_id = $1) AND ($2 = '' OR worker_id = $2)
		ORDER BY collected_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, formID, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.MobileSubmission
	for rows.Next() {
		var sub models.MobileSubmission
		var dataJSON, geoJSON []byte
		if err := rows.Scan(
			&sub.ID, &sub.ClientSubmissionID, &sub.FormID, &sub.WorkerID, &sub.VisitID, &dataJSON,
			&geoJSON, &sub.CollectedAt, &sub.SyncedAt, &sub.SyncStatus, &sub.TenantID, &sub.Version, &sub.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(dataJSON) > 0 {
			_ = json.Unmarshal(dataJSON, &sub.DataPayload)
		}
		if len(geoJSON) > 0 {
			_ = json.Unmarshal(geoJSON, &sub.GeoPoint)
		}
		res = append(res, sub)
	}
	return res, nil
}

func (p *PGStore) InsertGPSBreadcrumbs(ctx context.Context, crumbs []models.GPSBreadcrumb) error {
	if len(crumbs) == 0 {
		return nil
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO statiot.gps_breadcrumbs (
			id, worker_id, device_uuid, latitude, longitude, altitude, accuracy_meters,
			speed_mps, heading_deg, battery_level, tenant_id, recorded_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range crumbs {
		if c.ID == "" {
			c.ID = uuid.New().String()
		}
		if c.RecordedAt.IsZero() {
			c.RecordedAt = time.Now().UTC()
		}
		if _, err := stmt.ExecContext(ctx,
			c.ID, c.WorkerID, c.DeviceUUID, c.Latitude, c.Longitude, c.Altitude,
			c.AccuracyMeters, c.SpeedMPS, c.HeadingDeg, c.BatteryLevel, c.TenantID, c.RecordedAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *PGStore) GetRecentBreadcrumbs(ctx context.Context, workerID string, limit int) ([]models.GPSBreadcrumb, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, worker_id, device_uuid, latitude, longitude, altitude, accuracy_meters,
		       speed_mps, heading_deg, battery_level, tenant_id, recorded_at, created_at
		FROM statiot.gps_breadcrumbs WHERE worker_id = $1
		ORDER BY recorded_at DESC LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, workerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.GPSBreadcrumb
	for rows.Next() {
		var c models.GPSBreadcrumb
		if err := rows.Scan(
			&c.ID, &c.WorkerID, &c.DeviceUUID, &c.Latitude, &c.Longitude, &c.Altitude,
			&c.AccuracyMeters, &c.SpeedMPS, &c.HeadingDeg, &c.BatteryLevel, &c.TenantID, &c.RecordedAt, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, nil
}

func (p *PGStore) CreateGeofence(ctx context.Context, g *models.GeofenceZone) error {
	if g.ID == "" {
		g.ID = uuid.New().String()
	}
	polyJSON, _ := json.Marshal(g.PolygonGeoJSON)
	query := `
		INSERT INTO statiot.geofence_zones (
			id, name, zone_type, center_lat, center_lng, radius_meters, polygon_geojson, tenant_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		g.ID, g.Name, g.ZoneType, g.CenterLat, g.CenterLng, g.RadiusMeters, polyJSON, g.TenantID,
	)
	return err
}

func (p *PGStore) ListGeofences(ctx context.Context, tenantID string) ([]models.GeofenceZone, error) {
	query := `
		SELECT id, name, zone_type, center_lat, center_lng, radius_meters, polygon_geojson, tenant_id, created_at
		FROM statiot.geofence_zones WHERE ($1 = '' OR tenant_id = $1)
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.GeofenceZone
	for rows.Next() {
		var g models.GeofenceZone
		var polyJSON []byte
		if err := rows.Scan(
			&g.ID, &g.Name, &g.ZoneType, &g.CenterLat, &g.CenterLng, &g.RadiusMeters, &polyJSON, &g.TenantID, &g.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(polyJSON) > 0 {
			_ = json.Unmarshal(polyJSON, &g.PolygonGeoJSON)
		}
		res = append(res, g)
	}
	return res, nil
}

func (p *PGStore) RecordSyncTransaction(ctx context.Context, tx *models.SyncPushResponse, req *models.SyncPushRequest) error {
	query := `
		INSERT INTO statiot.sync_transactions (
			id, worker_id, device_uuid, sync_direction, records_synced, conflicts_detected, status, tenant_id, started_at, completed_at
		) VALUES ($1, $2, $3, 'PUSH', $4, $5, 'SUCCESS', 'default', $6, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		tx.TransactionID, req.WorkerID, req.DeviceUUID, tx.CommittedCount, tx.ConflictsCount, req.ClientTime,
	)
	return err
}

func (p *PGStore) RecordConflict(ctx context.Context, c *models.ConflictRecord) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	clientJSON, _ := json.Marshal(c.ClientPayload)
	serverJSON, _ := json.Marshal(c.ServerPayload)
	resolvedJSON, _ := json.Marshal(c.ResolvedPayload)

	query := `
		INSERT INTO statiot.conflict_records (
			id, entity_type, entity_id, client_version, server_version, client_payload, server_payload,
			resolution_strategy, resolved_payload, status, tenant_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP)
	`
	_, err := p.db.ExecContext(ctx, query,
		c.ID, c.EntityType, c.EntityID, c.ClientVersion, c.ServerVersion, clientJSON, serverJSON,
		c.ResolutionStrategy, resolvedJSON, c.Status, c.TenantID,
	)
	return err
}

func (p *PGStore) GetConflict(ctx context.Context, id string) (*models.ConflictRecord, error) {
	query := `
		SELECT id, entity_type, entity_id, client_version, server_version, client_payload, server_payload,
		       resolution_strategy, resolved_payload, status, resolved_by, tenant_id, created_at, resolved_at
		FROM statiot.conflict_records WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var c models.ConflictRecord
	var clientJSON, serverJSON, resolvedJSON []byte
	err := row.Scan(
		&c.ID, &c.EntityType, &c.EntityID, &c.ClientVersion, &c.ServerVersion,
		&clientJSON, &serverJSON, &c.ResolutionStrategy, &resolvedJSON,
		&c.Status, &c.ResolvedBy, &c.TenantID, &c.CreatedAt, &c.ResolvedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(clientJSON) > 0 {
		_ = json.Unmarshal(clientJSON, &c.ClientPayload)
	}
	if len(serverJSON) > 0 {
		_ = json.Unmarshal(serverJSON, &c.ServerPayload)
	}
	if len(resolvedJSON) > 0 {
		_ = json.Unmarshal(resolvedJSON, &c.ResolvedPayload)
	}
	return &c, nil
}

func (p *PGStore) ListConflicts(ctx context.Context, tenantID string, status string) ([]models.ConflictRecord, error) {
	query := `
		SELECT id, entity_type, entity_id, client_version, server_version, client_payload, server_payload,
		       resolution_strategy, resolved_payload, status, resolved_by, tenant_id, created_at, resolved_at
		FROM statiot.conflict_records
		WHERE ($1 = '' OR tenant_id = $1) AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.ConflictRecord
	for rows.Next() {
		var c models.ConflictRecord
		var clientJSON, serverJSON, resolvedJSON []byte
		if err := rows.Scan(
			&c.ID, &c.EntityType, &c.EntityID, &c.ClientVersion, &c.ServerVersion,
			&clientJSON, &serverJSON, &c.ResolutionStrategy, &resolvedJSON,
			&c.Status, &c.ResolvedBy, &c.TenantID, &c.CreatedAt, &c.ResolvedAt,
		); err != nil {
			return nil, err
		}
		if len(clientJSON) > 0 {
			_ = json.Unmarshal(clientJSON, &c.ClientPayload)
		}
		if len(serverJSON) > 0 {
			_ = json.Unmarshal(serverJSON, &c.ServerPayload)
		}
		if len(resolvedJSON) > 0 {
			_ = json.Unmarshal(resolvedJSON, &c.ResolvedPayload)
		}
		res = append(res, c)
	}
	return res, nil
}

func (p *PGStore) ResolveConflict(ctx context.Context, id string, resolvedPayload map[string]interface{}, strategy models.ConflictResolutionStrategy, resolvedBy string) error {
	resJSON, _ := json.Marshal(resolvedPayload)
	query := `
		UPDATE statiot.conflict_records
		SET resolved_payload = $2, resolution_strategy = $3, status = 'RESOLVED_MANUAL',
		    resolved_by = $4, resolved_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, resJSON, strategy, resolvedBy)
	return err
}

func (p *PGStore) CreateObjectLink(ctx context.Context, link *models.ObjectLink) error {
	query := `
		INSERT INTO statiot.object_links (
			source_type, source_id, target_type, target_id, relationship, tenant_id, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
		ON CONFLICT (source_type, source_id, target_type, target_id, relationship) DO NOTHING
	`
	_, err := p.db.ExecContext(ctx, query,
		link.SourceType, link.SourceID, link.TargetType, link.TargetID, link.Relationship, link.TenantID, link.CreatedBy,
	)
	if err != nil {
		// Also try global public.object_links if available
		_, _ = p.db.ExecContext(ctx, `
			INSERT INTO public.object_links (
				source_type, source_id, target_type, target_id, relationship, tenant_id, created_by, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
			ON CONFLICT (source_type, source_id, target_type, target_id, relationship) DO NOTHING
		`, link.SourceType, link.SourceID, link.TargetType, link.TargetID, link.Relationship, link.TenantID, link.CreatedBy)
	}
	return nil
}

func (p *PGStore) ListObjectLinks(ctx context.Context, srcType, srcID string) ([]models.ObjectLink, error) {
	query := `
		SELECT id, source_type, source_id, target_type, target_id, relationship, tenant_id, COALESCE(created_by, ''), created_at
		FROM statiot.object_links WHERE source_type = $1 AND source_id = $2
	`
	rows, err := p.db.QueryContext(ctx, query, srcType, srcID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.ObjectLink
	for rows.Next() {
		var l models.ObjectLink
		if err := rows.Scan(
			&l.ID, &l.SourceType, &l.SourceID, &l.TargetType, &l.TargetID, &l.Relationship, &l.TenantID, &l.CreatedBy, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		res = append(res, l)
	}
	return res, nil
}
