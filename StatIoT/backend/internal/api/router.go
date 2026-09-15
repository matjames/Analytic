package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/health"
	"github.com/matjames/statgate-lib/metrics"
	"github.com/matjames/statgate-lib/tenant"
)

type RouterConfig struct {
	IoTHandlers       *IoTHandlers
	FieldOpsHandlers  *FieldOpsHandlers
	DiscussionHandler *DiscussionHandler
	HealthChecker     *health.Checker
	Metrics           *metrics.Metrics
	AuthValidator     *auth.Validator
	CORSOrigin        string
	Env               string
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	if strings.EqualFold(cfg.Env, "production") {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// 1. CORS Configuration
	corsCfg := cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Tenant-ID", "X-Workspace-ID", "X-Request-ID", "X-Device-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsCfg))

	// 2. Metrics Middleware
	if cfg.Metrics != nil {
		r.Use(cfg.Metrics.GinMiddleware())
	}

	// 3. Health & Observability Endpoints
	if cfg.HealthChecker != nil {
		cfg.HealthChecker.RegisterGinRoutes(r)
	} else {
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "statiot-backend"})
		})
		r.GET("/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "statiot-backend"})
		})
		r.GET("/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "live", "service": "statiot-backend"})
		})
	}

	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	// Service Info
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "StatIoT & Mobile Field Ops (App 8)",
			"phases":    []string{"P27 (IoT & Edge)", "P43 (Mobile Field Ops)"},
			"status":    "OPERATIONAL",
			"version":   "1.0.0",
			"timestamp": time.Now().UTC(),
		})
	})

	apiV1 := r.Group("/api/v1")
	// Stage 2: workspace context propagation. Membership enforcement applies
	// only to the admin group below (device/gateway routes authenticate via
	// device tokens / gateway secret instead).
	apiV1.Use(tenant.GinWorkspaceContext())
	{
		apiV1.GET("/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"name":       "StatGate IoT, Sensors & Mobile Field Ops",
				"app_number": 8,
				"phases":     []string{"P27", "P43"},
				"protocols":  []string{"MQTT", "HTTP", "CoAP", "LoRaWAN", "ODK/Sync"},
				"status":     "UP",
			})
		})

		// ── DEVICE-FACING ROUTES (device token authentication, fail-closed) ──
		device := apiV1.Group("", cfg.IoTHandlers.DeviceAuth())
		{
			device.POST("/iot/telemetry/ingest", cfg.IoTHandlers.IngestTelemetry)
			device.GET("/iot/telemetry/:device_id/latest", cfg.IoTHandlers.GetLatestTelemetry)
			device.GET("/iot/telemetry/:device_id/aggregates", cfg.IoTHandlers.GetTelemetryAggregates)
			device.GET("/iot/devices/:id/config", cfg.IoTHandlers.GetDeviceConfig)
			device.POST("/iot/devices/:id/config/report", cfg.IoTHandlers.ReportAppliedConfig)
			device.GET("/iot/firmware/latest", cfg.IoTHandlers.GetLatestFirmware)
		}

		// ── GATEWAY ROUTES (shared gateway secret) ──
		gwGroup := apiV1.Group("", cfg.IoTHandlers.GatewaySecretAuth())
		{
			gwGroup.POST("/iot/gateways/:id/heartbeat", cfg.IoTHandlers.GatewayHeartbeat)
		}

		// ── ADMIN ROUTES (Registry JWT required) ──
		admin := apiV1.Group("")
		if cfg.AuthValidator != nil {
			admin.Use(cfg.AuthValidator.GinMiddleware())
		}
		admin.Use(tenant.GinTenantIsolation())
		admin.Use(tenant.GinWorkspaceMembership("", nil))
		if cfg.DiscussionHandler != nil {
			admin.POST("/discussions", cfg.DiscussionHandler.Create)
		}

		// ── IoT & SENSORS (P27) ─────────────────────────────────────────
		iotGroup := admin.Group("/iot")
		{
			// Gateways
			iotGroup.POST("/gateways", cfg.IoTHandlers.RegisterGateway)
			iotGroup.GET("/gateways", cfg.IoTHandlers.ListGateways)
			iotGroup.GET("/gateways/:id", cfg.IoTHandlers.GetGateway)

			// Devices & Sensors
			iotGroup.POST("/devices", cfg.IoTHandlers.RegisterDevice)
			iotGroup.GET("/devices", cfg.IoTHandlers.ListDevices)
			iotGroup.GET("/devices/:id", cfg.IoTHandlers.GetDevice)
			iotGroup.POST("/devices/:id/sensors", cfg.IoTHandlers.RegisterSensor)
			iotGroup.GET("/devices/:id/sensors", cfg.IoTHandlers.ListSensors)

			// Edge Configuration & Firmware (admin side)
			iotGroup.POST("/devices/:id/config", cfg.IoTHandlers.SetDesiredConfig)
			iotGroup.POST("/firmware", cfg.IoTHandlers.CreateFirmware)

			// Alerts
			iotGroup.GET("/alerts", cfg.IoTHandlers.ListAlerts)
			iotGroup.POST("/alerts/:id/resolve", cfg.IoTHandlers.ResolveAlert)
		}

		// ── MOBILE FIELD OPERATIONS & OFFLINE (P43) ────────────────────
		fieldGroup := admin.Group("/fieldops")
		{
			// Workers & Mobile Devices
			fieldGroup.POST("/workers", cfg.FieldOpsHandlers.CreateWorker)
			fieldGroup.GET("/workers", cfg.FieldOpsHandlers.ListWorkers)
			fieldGroup.GET("/workers/:id", cfg.FieldOpsHandlers.GetWorker)
			fieldGroup.POST("/devices/register", cfg.FieldOpsHandlers.RegisterMobileDevice)

			// Visits & Tasks
			fieldGroup.POST("/visits", cfg.FieldOpsHandlers.CreateVisit)
			fieldGroup.GET("/visits", cfg.FieldOpsHandlers.ListVisits)
			fieldGroup.POST("/visits/:id/tasks", cfg.FieldOpsHandlers.CreateVisitTask)
			fieldGroup.GET("/visits/:id/tasks", cfg.FieldOpsHandlers.ListVisitTasks)

			// Forms
			fieldGroup.POST("/forms", cfg.FieldOpsHandlers.CreateFormDefinition)
			fieldGroup.GET("/forms", cfg.FieldOpsHandlers.ListFormDefinitions)

			// Delta Synchronization (Push & Pull)
			fieldGroup.POST("/sync/push", cfg.FieldOpsHandlers.SyncPush)
			fieldGroup.POST("/sync/pull", cfg.FieldOpsHandlers.SyncPull)

			// GPS & Geofencing
			fieldGroup.POST("/gps/track", cfg.FieldOpsHandlers.IngestGPSTrack)
			fieldGroup.GET("/gps/breadcrumbs/:worker_id", cfg.FieldOpsHandlers.GetRecentBreadcrumbs)
			fieldGroup.POST("/geofences", cfg.FieldOpsHandlers.CreateGeofence)
			fieldGroup.GET("/geofences", cfg.FieldOpsHandlers.ListGeofences)

			// Bidirectional Conflicts
			fieldGroup.GET("/conflicts", cfg.FieldOpsHandlers.ListConflicts)
			fieldGroup.POST("/conflicts/:id/resolve", cfg.FieldOpsHandlers.ResolveConflict)
		}
	}

	return r
}
