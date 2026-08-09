package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const EventChannel = "statgate:events"

var (
	version   = "7.0.0"
	startTime = time.Now()
)

// ─── Models ──────────────────────────────────────────────────────

type DomainEvent struct {
	ID          string                 `json:"id"`
	EventType   string                 `json:"event_type"`
	Source      string                 `json:"source"`
	ObjectType  string                 `json:"object_type"`
	ObjectID    string                 `json:"object_id"`
	Actor       string                 `json:"actor"`
	TenantID    string                 `json:"tenant_id"`
	ProjectID   string                 `json:"project_id,omitempty"`
	OrgID       string                 `json:"organization_id,omitempty"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   string                 `json:"timestamp"`
	Correlation string                 `json:"correlation_id,omitempty"`
}

type Notification struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	Priority       string                 `json:"priority"`
	Category       string                 `json:"category"`
	SourceApp      string                 `json:"source_app"`
	SourceEntity   string                 `json:"source_entity,omitempty"`
	SourceEntityID string                 `json:"source_entity_id,omitempty"`
	Read           bool                   `json:"read"`
	Archived       bool                   `json:"archived"`
	DeepLink       string                 `json:"deep_link,omitempty"`
	Actions        []NotificationAction   `json:"actions,omitempty"`
	Attachments    []AttachmentRef        `json:"attachments,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      string                 `json:"created_at"`
}

type NotificationAction struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

type AttachmentRef struct {
	FileID   string `json:"file_id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
}

type TimelineEntry struct {
	ID          string                 `json:"id"`
	User        string                 `json:"user"`
	Application string                 `json:"application"`
	Entity      string                 `json:"entity"`
	EntityID    string                 `json:"entity_id"`
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	ProjectID   string                 `json:"project_id,omitempty"`
	OrgID       string                 `json:"organization_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   string                 `json:"timestamp"`
}

type DashboardDefinition struct {
	ID          string         `json:"id"`
	OwnerID     string         `json:"owner_id"`
	OwnerType   string         `json:"owner_type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Layout      []WidgetLayout `json:"layout"`
	Shared      bool           `json:"shared"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

type WidgetLayout struct {
	ID       string                 `json:"id"`
	WidgetID string                 `json:"widget_id"`
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	X        int                    `json:"x"`
	Y        int                    `json:"y"`
	W        int                    `json:"w"`
	H        int                    `json:"h"`
	Config   map[string]interface{} `json:"config"`
}

type FileRecord struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	OriginalName string                 `json:"original_name"`
	Path         string                 `json:"path"`
	Size         int64                  `json:"size"`
	MimeType     string                 `json:"mime_type"`
	Category     string                 `json:"category"`
	UploadedBy   string                 `json:"uploaded_by"`
	ProjectID    string                 `json:"project_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Version      int                    `json:"version"`
	PreviewURL   string                 `json:"preview_url,omitempty"`
	Checksum     string                 `json:"checksum,omitempty"`
	Status       string                 `json:"status"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

type CalendarEvent struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Start        string                 `json:"start"`
	End          string                 `json:"end"`
	AllDay       bool                   `json:"all_day"`
	Category     string                 `json:"category"`
	SourceApp    string                 `json:"source_app"`
	EntityID     string                 `json:"entity_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	Participants []string               `json:"participants,omitempty"`
	Location     string                 `json:"location,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    string                 `json:"created_at"`
}

type ReportRequest struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Type        string                 `json:"type"`
	Format      string                 `json:"format"`
	SourceApp   string                 `json:"source_app"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Query       map[string]interface{} `json:"query,omitempty"`
	Template    string                 `json:"template,omitempty"`
	CreatedBy   string                 `json:"created_by"`
	Status      string                 `json:"status"`
	DownloadURL string                 `json:"download_url,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	CompletedAt string                 `json:"completed_at,omitempty"`
}

type ServiceHealth struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	URL       string                 `json:"url"`
	LatencyMs int64                  `json:"latency_ms"`
	Version   string                 `json:"version,omitempty"`
	LastCheck string                 `json:"last_check"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// ─── Main ────────────────────────────────────────────────────────

func main() {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")
	initAIPersistence()

	initRedis()
	go consumeEvents()
	startScheduler()

	// ── Phase IV: Enterprise Data Intelligence & Real-Time Analytics ──
	bootstrapDataLayer()
	bootstrapKPIs()
	bootstrapAlertRules()
	bootstrapAnomalyRules()
	bootstrapAnalyticsDashboards()
	startAnalyticsStreamBroker()
	startAnalyticsEventSubscriber()

	// ── Phase V: Enterprise Decision & Workflow Automation ──
	bootstrapWorkflowTemplates()
	bootstrapEscalationPolicies()
	bootstrapSLAs()

	// Phase VI: Enterprise Knowledge, Collaboration & Institutional Intelligence
	bootstrapKnowledge()

	r := setupRouter()
	registerRoutes(r)

	port := getEnv("ENTERPRISE_CORE_PORT", "8096")
	log.Printf("StatGate Enterprise Core v%s starting on :%s...", version, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000", "http://localhost:3003", "http://localhost:3005",
			"http://localhost:3006", "http://localhost:3007", "http://localhost:3009",
			"http://localhost:3010", "http://localhost:3011", "http://localhost:5000",
			"http://localhost:8088", "http://host.docker.internal:3006",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Request-ID", "X-Correlation-ID", "X-User-ID", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Disposition", "X-Report-URL"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))
	r.GET("/health", handleHealth)
	r.GET("/api/info", handleInfo)
	return r
}

func getEnv(key, def string) string {
	if v := getEnvValue(key); v != "" {
		return v
	}
	return def
}

func handleHealth(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":   "healthy",
		"service":  "statgate-enterprise-core",
		"version":  version,
		"uptime_s": int(time.Since(startTime).Seconds()),
		"redis":    redisClient != nil,
	})
}

func handleInfo(c *gin.Context) {
	c.JSON(200, gin.H{
		"name":    "StatGate Enterprise Core",
		"version": version,
		"phase":   "PHASE_VII",
		"capabilities": []string{
			"event_bus", "notifications", "timeline", "dashboards", "widgets",
			"file_service", "permissions", "calendar", "reporting", "ai_preparation",
			"api_governance", "monitoring", "workspace", "idempotency",
			"dead_letter_queue", "correlation_ids",
			"enterprise_data_layer", "kpi_engine", "kpi_drill_down", "data_quality",
			"anomaly_detection", "alert_engine", "analytics_dashboards",
			"cross_app_intelligence", "report_builder", "scheduled_reports",
			"data_export", "analytics_search", "ai_ready_analytics",
			"data_lineage", "real_time_stream",
			"enterprise_workflow_engine", "workflow_builder", "approval_engine",
			"enterprise_ai_context", "grounded_ai_query", "ai_source_traceability",
			"ai_investigations", "ai_recommendations", "ai_feedback", "ai_audit",
			"action_centre", "my_work", "enterprise_tasks", "task_dependencies",
			"escalation_engine", "sla_management", "cross_app_actions",
			"workflow_audit_trails", "failure_retry", "dead_letter_workflows",
			"workflow_templates", "role_aware_workflows", "organizational_scoping",
			"workflow_dashboard", "process_analytics", "human_approval_controls",
			"ai_recommendations", "decision_records", "closed_loop_management",
			"enterprise_knowledge_graph", "entity_context", "knowledge_search",
			"knowledge_articles", "knowledge_versioning", "data_dictionary",
			"permission_aware_knowledge", "sourced_ai_context", "knowledge_audit",
		},
		"started_at": startTime.UTC().Format(time.RFC3339),
	})
}
