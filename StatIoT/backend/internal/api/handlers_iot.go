package api

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"statiot-backend/internal/iot"
	"statiot-backend/internal/models"
	"statiot-backend/internal/store"
)

type IoTHandlers struct {
	store   store.Store
	gateway *iot.GatewayEngine
}

func NewIoTHandlers(st store.Store, gw *iot.GatewayEngine) *IoTHandlers {
	return &IoTHandlers{
		store:   st,
		gateway: gw,
	}
}

// RegisterGateway handles provisioning a new physical or virtual IoT Gateway.
// getWorkspaceID extracts the selected workspace (Stage 2) set by
// tenant.GinWorkspaceContext on the request path.
func getWorkspaceID(c *gin.Context) string {
	if w, exists := c.Get("workspace_id"); exists {
		if s, ok := w.(string); ok {
			return s
		}
	}
	return ""
}

// DeviceAuth is a fail-closed middleware for device-facing routes. It requires
// X-Device-UID and X-Device-Token headers and verifies the token hash via the
// gateway engine. On success the device identity is stored in the context.
func (h *IoTHandlers) DeviceAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetHeader("X-Device-UID")
		token := c.GetHeader("X-Device-Token")
		if uid == "" || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "device_credentials_required",
				"message": "X-Device-UID and X-Device-Token headers are required",
			})
			return
		}
		dev, err := h.gateway.AuthenticateDevice(c.Request.Context(), uid, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "device_authentication_failed",
				"details": err.Error(),
			})
			return
		}
		c.Set("device_id", dev.ID)
		c.Set("device_uid", uid)
		c.Set("workspace_id", dev.WorkspaceID)
		c.Next()
	}
}

// GatewaySecretAuth protects gateway-originated routes (e.g. heartbeats) with
// the shared STATIOT_GATEWAY_SECRET delivered in the X-Gateway-Secret header.
func (h *IoTHandlers) GatewaySecretAuth() gin.HandlerFunc {
	expected := os.Getenv("STATIOT_GATEWAY_SECRET")
	return func(c *gin.Context) {
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":   "gateway_auth_unavailable",
				"message": "STATIOT_GATEWAY_SECRET is not configured",
			})
			return
		}
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Gateway-Secret")), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "gateway_authentication_failed"})
			return
		}
		c.Next()
	}
}

func (h *IoTHandlers) RegisterGateway(c *gin.Context) {
	var req models.IoTGateway
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	if req.GatewayCode == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway_code and name are required"})
		return
	}

	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.Status == "" {
		req.Status = models.GatewayStatusOnline
	}
	req.TenantID = c.GetString("tenant_id")
	if req.TenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated tenant is required"})
		return
	}
	req.WorkspaceID = getWorkspaceID(c)

	if err := h.store.CreateGateway(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create gateway", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Gateway registered successfully", "gateway": req})
}

// ListGateways lists all gateways for a tenant, scoped to the selected workspace.
func (h *IoTHandlers) ListGateways(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workspaceID := getWorkspaceID(c)
	gateways, err := h.store.ListGateways(c.Request.Context(), tenantID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve gateways", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gateways": gateways, "count": len(gateways)})
}

// GetGateway returns a single gateway by ID.
func (h *IoTHandlers) GetGateway(c *gin.Context) {
	id := c.Param("id")
	gw, err := h.store.GetGateway(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gateway not found"})
		return
	}
	c.JSON(http.StatusOK, gw)
}

// GatewayHeartbeat updates the gateway status and last heartbeat timestamp.
func (h *IoTHandlers) GatewayHeartbeat(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status models.GatewayStatus `json:"status"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Status == "" {
		req.Status = models.GatewayStatusOnline
	}

	if err := h.store.UpdateGatewayHeartbeat(c.Request.Context(), id, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update heartbeat", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat recorded", "status": req.Status})
}

// RegisterDevice registers a sensor node, smart device, or drone.
func (h *IoTHandlers) RegisterDevice(c *gin.Context) {
	var req models.IoTDevice
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device payload", "details": err.Error()})
		return
	}

	if req.DeviceUID == "" || req.Name == "" || req.DeviceType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_uid, name, and device_type are required"})
		return
	}

	token, tokenHash := iot.GenerateDeviceToken("")
	req.AuthTokenHash = tokenHash
	if req.Status == "" {
		req.Status = models.DeviceStatusActive
	}
	req.TenantID = c.GetString("tenant_id")
	if req.TenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated tenant is required"})
		return
	}
	req.WorkspaceID = getWorkspaceID(c)

	if err := h.store.RegisterDevice(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Device registered successfully",
		"device":       req,
		"device_token": token, // Delivered only once on provisioning
	})
}

// ListDevices returns all registered devices, scoped to the selected workspace.
func (h *IoTHandlers) ListDevices(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	workspaceID := getWorkspaceID(c)
	devs, err := h.store.ListDevices(c.Request.Context(), tenantID, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve devices", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devs, "count": len(devs)})
}

// GetDevice returns details for a single device.
func (h *IoTHandlers) GetDevice(c *gin.Context) {
	id := c.Param("id")
	dev, err := h.store.GetDevice(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}
	c.JSON(http.StatusOK, dev)
}

// RegisterSensor attaches a sensor definition to a device.
func (h *IoTHandlers) RegisterSensor(c *gin.Context) {
	deviceID := c.Param("id")
	var req models.SensorRegistryItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor payload", "details": err.Error()})
		return
	}

	req.DeviceID = deviceID
	if req.SensorCode == "" || req.MetricName == "" || req.UnitOfMeasure == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sensor_code, metric_name, and unit_of_measure are required"})
		return
	}
	if req.CalibrationFactor == 0 {
		req.CalibrationFactor = 1.0
	}
	if req.Status == "" {
		req.Status = "ACTIVE"
	}

	if err := h.store.RegisterSensor(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register sensor", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Sensor registered successfully", "sensor": req})
}

// ListSensors returns all sensors attached to a device.
func (h *IoTHandlers) ListSensors(c *gin.Context) {
	deviceID := c.Param("id")
	sensors, err := h.store.ListSensorsByDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sensors", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sensors": sensors, "count": len(sensors)})
}

// IngestTelemetry ingests a batch of time-series telemetry from HTTP or MQTT forwarder.
func (h *IoTHandlers) IngestTelemetry(c *gin.Context) {
	var req models.TelemetryBatchIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid telemetry payload", "details": err.Error()})
		return
	}

	if req.DeviceUID == "" || len(req.Readings) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_uid and at least one reading are required"})
		return
	}

	// Device identity: the DeviceAuth middleware has already verified the
	// device token and stored the authenticated UID in the context. The
	// authenticated identity always wins over any body-supplied device_uid
	// (prevents cross-device spoofing).
	if authedUID := c.GetString("device_uid"); authedUID != "" {
		req.DeviceUID = authedUID
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "device_credentials_required",
			"message": "device authentication is required for telemetry ingestion",
		})
		return
	}

	count, alerts, err := h.gateway.ProcessTelemetryBatch(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process telemetry batch", "details": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":        "Telemetry batch accepted",
		"records_stored": count,
		"alerts_created": len(alerts),
		"alerts":         alerts,
	})
}

// GetLatestTelemetry queries recent telemetry for a device.
func (h *IoTHandlers) GetLatestTelemetry(c *gin.Context) {
	deviceID := c.Param("device_id")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	readings, err := h.store.GetLatestTelemetry(c.Request.Context(), deviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve telemetry", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_id": deviceID, "readings": readings, "count": len(readings)})
}

// GetTelemetryAggregates computes min, max, avg, count over a time window.
func (h *IoTHandlers) GetTelemetryAggregates(c *gin.Context) {
	deviceID := c.Param("device_id")
	metricName := c.Query("metric_name")
	if metricName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric_name query param is required"})
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	if fromStr != "" {
		from, _ = time.Parse(time.RFC3339, fromStr)
	}
	if toStr != "" {
		to, _ = time.Parse(time.RFC3339, toStr)
	}

	agg, err := h.store.GetTelemetryAggregates(c.Request.Context(), deviceID, metricName, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute aggregates", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, agg)
}

// GetDeviceConfig returns the desired configuration for edge synchronization.
func (h *IoTHandlers) GetDeviceConfig(c *gin.Context) {
	deviceID := c.Param("id")
	cfg, err := h.store.GetDeviceConfig(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch config", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// SetDesiredConfig updates the target state configuration for an edge device.
func (h *IoTHandlers) SetDesiredConfig(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		DesiredConfig map[string]interface{} `json:"desired_config"`
		Version       int                    `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config payload", "details": err.Error()})
		return
	}

	edgeCfg := &models.EdgeConfiguration{
		DeviceID:      deviceID,
		DesiredConfig: req.DesiredConfig,
		Version:       req.Version,
		TenantID:      c.GetString("tenant_id"),
	}

	if err := h.store.SetDesiredConfig(c.Request.Context(), edgeCfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set desired config", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Desired configuration updated", "config": edgeCfg})
}

// ReportAppliedConfig allows the edge device to acknowledge applied configuration.
func (h *IoTHandlers) ReportAppliedConfig(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		ReportedConfig map[string]interface{} `json:"reported_config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report payload", "details": err.Error()})
		return
	}

	if err := h.store.ReportAppliedConfig(c.Request.Context(), deviceID, req.ReportedConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record applied config", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration status acknowledged as APPLIED"})
}

// CreateFirmware uploads a new firmware release for OTA distribution.
func (h *IoTHandlers) CreateFirmware(c *gin.Context) {
	var req models.FirmwareRelease
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firmware payload", "details": err.Error()})
		return
	}

	if req.DeviceType == "" || req.Version == "" || req.BinaryURL == "" || req.ChecksumSHA256 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_type, version, binary_url, and checksum_sha256 are required"})
		return
	}

	if err := h.store.CreateFirmware(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create firmware release", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Firmware release published", "firmware": req})
}

// GetLatestFirmware returns the latest OTA firmware for a given device type.
func (h *IoTHandlers) GetLatestFirmware(c *gin.Context) {
	deviceType := c.Query("device_type")
	if deviceType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_type query param is required"})
		return
	}

	fw, err := h.store.GetLatestFirmware(c.Request.Context(), deviceType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No firmware available for this device type"})
		return
	}
	c.JSON(http.StatusOK, fw)
}

// ListAlerts returns active and resolved IoT alerts.
func (h *IoTHandlers) ListAlerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	status := c.Query("status")
	workspaceID := getWorkspaceID(c)

	alerts, err := h.store.ListAlerts(c.Request.Context(), tenantID, status, workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve alerts", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "count": len(alerts)})
}

// ResolveAlert marks an active alert as resolved.
func (h *IoTHandlers) ResolveAlert(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		ResolvedBy string `json:"resolved_by"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.ResolvedBy == "" {
		req.ResolvedBy = "operator"
	}

	if err := h.store.ResolveAlert(c.Request.Context(), id, req.ResolvedBy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve alert", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved successfully"})
}
