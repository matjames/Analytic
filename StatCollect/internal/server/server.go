package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Meta struct {
	FormID     string   `json:"form_id"`
	InstanceID string   `json:"instance_id"`
	Files      []string `json:"files"`
	ReceivedAt string   `json:"received_at"`
}

// package-level store
var store Store
var cfg *Config

// Prometheus metrics
var (
	submissionsReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "statcollect_submissions_received_total",
		Help: "Total submissions received",
	})
	submissionsDuplicate = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "statcollect_submissions_duplicates_total",
		Help: "Total duplicate submissions skipped",
	})
	submissionsFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "statcollect_submissions_failed_total",
		Help: "Total failed submissions",
	})
	submissionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "statcollect_submission_duration_seconds",
		Help:    "Submission processing duration (seconds)",
		Buckets: prometheus.DefBuckets,
	})
	submissionsLinked = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "statcollect_submissions_linked_total",
		Help: "Total submissions linked to StatChat discussions",
	})
)

// Run starts the server
func Run() {
	cfg = LoadConfig()

	// init DB if configured
	if err := InitDB(cfg.DBDSN); err != nil {
		log.Printf("warning: DB init failed: %v", err)
	} else {
		defer CloseDB()
	}

	// init StatGate Platform Integrations
	InitEventBus(cfg.RedisAddr, cfg.RedisPassword, cfg.EnableEvents)
	InitStatChat(cfg.StatChatURL, cfg.EnableStatChat)
	InitRegistry(cfg.RegistryJWTSecret, cfg.RegistryURL, cfg.RegistryJWTSecret != "")

	// seed default questionnaire templates (database-backed, not hardcoded)
	if err := SeedDefaultTemplates(); err != nil {
		log.Printf("warning: failed to seed templates: %v", err)
	} else {
		log.Printf("questionnaire templates initialized")
	}

	// init store
	var storeBackend Store
	if cfg.UseS3 && cfg.S3Bucket != "" {
		s3s, err := NewS3Store(cfg.S3Region, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket)
		if err != nil {
			log.Fatalf("failed to init s3 store: %v", err)
		}
		storeBackend = s3s
	} else {
		storeBackend = &FSStore{BaseDir: cfg.DataDir}
	}

	// make store globally accessible via package-level variable
	store = storeBackend

	// serve admin static files (moved to frontend/admin)
	fs := http.FileServer(http.Dir("./frontend/admin"))
	http.Handle("/admin/", http.StripPrefix("/admin/", fs))

	// serve field agent static files (moved to frontend/field)
	ffs := http.FileServer(http.Dir("./frontend/field"))
	http.Handle("/field/", http.StripPrefix("/field/", ffs))

	// serve StatCollect AI app (frontend/ai)
	aifs := http.FileServer(http.Dir("./frontend/ai"))
	http.Handle("/ai/", http.StripPrefix("/ai/", aifs))

	// register prometheus metrics
	prometheus.MustRegister(submissionsReceived, submissionsDuplicate, submissionsFailed, submissionDuration, submissionsLinked)
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/health/full", fullHealthHandler)
	http.HandleFunc("/submission", submissionHandler)

	// Admin API - Core
	http.HandleFunc("/admin/submissions", adminListHandler)
	http.HandleFunc("/admin/submission", adminGetHandler)
	http.HandleFunc("/admin/keys", adminRotateKeysHandler)

	// Admin API - StatGate Platform Integration
	http.HandleFunc("/admin/submission/validate", adminValidateHandler)
	http.HandleFunc("/admin/events", adminEventsHandler)
	http.HandleFunc("/admin/objects/links", adminObjectLinksHandler)
	http.HandleFunc("/admin/submission/discussion", adminDiscussionHandler)

	// Template Management API (database-backed, not hardcoded)
	http.HandleFunc("/templates", templatesHandler)
	http.HandleFunc("/templates/library", templateLibraryHandler)
	http.HandleFunc("/templates/import", templateImportHandler)
	http.HandleFunc("/templates/export", templateExportHandler)
	http.HandleFunc("/templates/versions", templateVersionsHandler)
	http.HandleFunc("/templates/share", templateShareHandler)
	http.HandleFunc("/templates/clone", templateCloneHandler)
	http.HandleFunc("/templates/", templateDetailHandler)

	// Admin API - Data Quality & Export
	http.HandleFunc("/admin/submissions/export", adminExportHandler)
	http.HandleFunc("/admin/submissions/qa", adminQAHandler)
	http.HandleFunc("/admin/submission/review", adminReviewHandler)

	// Phase Y Enterprise Core APIs
	http.HandleFunc("/admin/devices", adminDevicesHandler)
	http.HandleFunc("/admin/assignments", adminAssignmentsHandler)
	http.HandleFunc("/admin/registries", adminRegistriesHandler)
	http.HandleFunc("/admin/sampling", adminSamplingHandler)
	http.HandleFunc("/admin/workflows", adminWorkflowsHandler)
	http.HandleFunc("/admin/schedules", adminSchedulesHandler)
	http.HandleFunc("/admin/comments", adminCommentsHandler)
	http.HandleFunc("/admin/notifications", adminNotificationsHandler)
	http.HandleFunc("/admin/ai/analyze", adminAIAnalyzeHandler)
	http.HandleFunc("/admin/ai/outliers", adminAIOutliersHandler)

	// Phase Y Production Additions
	http.HandleFunc("/admin/lineage", adminLineageHandler)
	http.HandleFunc("/admin/submission/approve", adminApproveHandler)
	http.HandleFunc("/admin/submission/reject", adminRejectHandler)
	http.HandleFunc("/admin/submission/pipeline", adminPipelineHandler)
	http.HandleFunc("/admin/ai/summary", adminAISummaryHandler)
	http.HandleFunc("/admin/ai/translate", adminAITranslateHandler)
	http.HandleFunc("/admin/ai/quality", adminAIQualityHandler)
	http.HandleFunc("/admin/integrations", adminIntegrationsHandler)
	http.HandleFunc("/admin/plugins", adminPluginsHandler)
	http.HandleFunc("/admin/rules", adminRulesHandler)
	http.HandleFunc("/templates/rollback", templateRollbackHandler)

	// Directive 21 — Mandatory Survey Header (MSH) API
	http.HandleFunc("/admin/msh", adminMSHHandler)
	http.HandleFunc("/admin/msh/preview", adminMSHPreviewHandler)
	http.HandleFunc("/admin/respondent-types", adminRespondentTypesHandler)
	http.HandleFunc("/admin/admin-levels", adminAdminLevelsHandler)

	// Directive 22 — Unified Platform Integration API
	http.HandleFunc("/admin/platform/links", adminPlatformLinksHandler)
	http.HandleFunc("/admin/platform/tasks", adminWorkflowTasksHandler)
	http.HandleFunc("/admin/platform/publish", adminPlatformPublishHandler)
	http.HandleFunc("/admin/platform/gis-queue", adminGISSyncQueueHandler)
	http.HandleFunc("/admin/platform/stats-exports", adminStatsExportsHandler)

	// Phase X — Real-Time Analytics Engine API
	SetupAnalyticsRoutes()
	http.HandleFunc("/api/analytics/ai", handleGetAIIntelligence)

	// Survey Intelligence Engine — auto field classification, descriptive stats,
	// missing values, outlier detection, cross-tabulation, frequency analysis
	SetupIntelligenceRoutes()

	// Phase X — Start SSE event broker for live analytics stream
	StartSSEBroker()

	go NotificationDispatcher()

	// OpenAPI Specification API
	http.HandleFunc("/openapi.json", openAPIHandler)

	addr := cfg.Port
	srv := &http.Server{Addr: addr}

	go func() {
		log.Printf("StatCollect adapter listening on %s", addr)
		log.Printf("  Events: %s", eventBus.String())
		log.Printf("  StatChat: %s", statChat.String())
		log.Printf("  Registry: %s", registry.String())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Println("shutting down")
	if eventBus != nil {
		eventBus.Close()
	}
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func submissionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// API key check (StatGate internal key or configured API key)
	if !checkAPIKey(r) && !checkInternalKey(r) {
		// Also allow Registry JWT for authenticated service calls
		if checkRegistryToken(r) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// measure duration
	timer := prometheus.NewTimer(submissionDuration)
	defer timer.ObserveDuration()

	// limit memory for parsing
	err := r.ParseMultipartForm(200 << 20) // 200MB
	if err != nil {
		submissionsFailed.Inc()
		http.Error(w, "failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// find XML submission content
	xmlData, formID, instanceID := extractXMLSubmission(r.MultipartForm)
	// allow client to provide instance id in header for idempotency
	if hdr := r.Header.Get("X-Instance-ID"); hdr != "" {
		instanceID = hdr
	}
	if len(instanceID) == 0 {
		// fallback to timestamp-based id
		instanceID = fmt.Sprintf("inst-%d", time.Now().UnixNano())
	}
	// idempotency: if submission already exists, return 200
	if exists, err := SubmissionExists(instanceID); err == nil && exists {
		submissionsDuplicate.Inc()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("already received"))
		return
	}

	// persist submission via configured store
	savedFiles := []string{}

	if len(xmlData) > 0 {
		if err := store.SaveSubmission(instanceID, "submission.xml", xmlData); err != nil {
			log.Printf("failed to save submission xml: %v", err)
		} else {
			savedFiles = append(savedFiles, "submission.xml")
		}
	}

	// save any uploaded files via store
	hasVideoAttachment := false
	for _, fhs := range r.MultipartForm.File {
		for _, fh := range fhs {
			// Track whether a field_video was uploaded
			if strings.HasPrefix(fh.Filename, "field_video") || fh.Filename == "field_video" {
				hasVideoAttachment = true
			}
			key, err := store.SaveFile(instanceID, fh)
			if err != nil {
				log.Printf("failed to save uploaded file: %v", err)
				continue
			}
			savedFiles = append(savedFiles, key)
			// record attachment in DB (size available in header)
			_ = SaveAttachmentToDB(instanceID, filepath.Base(key), key, fh.Size)
		}
	}

	// ── GPS ENFORCEMENT ──
	// Every submission MUST contain GPS coordinates in the XML.
	// Reject the submission if gps_coordinates is missing or empty.
	if len(xmlData) > 0 {
		xmlStr := string(xmlData)
		hasGPS := strings.Contains(xmlStr, "<gps_coordinates>") &&
			!strings.Contains(xmlStr, "<gps_coordinates></gps_coordinates>")
		if !hasGPS {
			submissionsFailed.Inc()
			log.Printf("submission %s rejected: missing GPS coordinates", instanceID)
			http.Error(w, "GPS coordinates are mandatory for all field submissions", http.StatusBadRequest)
			return
		}
	}

	// ── VIDEO EVIDENCE QA FLAG ──
	// Log a QA flag if no field_video was attached; submission is still accepted
	// but flagged so supervisor can follow up.
	if !hasVideoAttachment {
		log.Printf("qa_flag: submission %s is missing field video evidence", instanceID)
		_ = LogEvent("submission.qa_flag", "statcollect", "submission", instanceID, map[string]string{
			"flag":   "MISSING_FIELD_VIDEO",
			"detail": "No field_video attachment was submitted with this record",
		})
	}

	meta := Meta{
		FormID:     formID,
		InstanceID: instanceID,
		Files:      savedFiles,
		ReceivedAt: time.Now().Format(time.RFC3339),
	}
	metaB, _ := json.MarshalIndent(meta, "", "  ")

	// save meta to storage as well
	_ = store.SaveSubmission(instanceID, "meta.json", metaB)

	// capture submitting identity from Registry JWT if present
	submittedBy := ""
	if ident := checkRegistryToken(r); ident != nil {
		submittedBy = ident.UserID
		_ = LogEvent(EventSubmissionReceived, "statcollect", "submission", instanceID, map[string]interface{}{
			"instance_id":  instanceID,
			"form_id":      formID,
			"submitted_by": ident.UserID,
			"files":        savedFiles,
		})
	}

	// persist submission record in DB (xml as text)
	if err := SaveSubmissionToDB(instanceID, formID, map[string]interface{}{"files": savedFiles}, string(xmlData)); err != nil {
		submissionsFailed.Inc()
		log.Printf("failed to save submission to db: %v", err)
		http.Error(w, "failed to save submission", http.StatusInternalServerError)
		return
	}
	submissionsReceived.Inc()

	// ── StatGate Platform Integration ──
	// 1. Publish submission.received event to the event bus
	PublishSubmissionReceived(instanceID, formID, savedFiles)
	// 2. Log event for audit
	_ = LogEvent(EventSubmissionReceived, "statcollect", "submission", instanceID, map[string]interface{}{
		"instance_id":  instanceID,
		"form_id":      formID,
		"files":        savedFiles,
		"submitted_by": submittedBy,
	})
	// 3. Link submission to StatChat for discussion (communication backbone)
	if statChat != nil && statChat.IsEnabled() {
		go LinkSubmission(instanceID, formID)
		submissionsLinked.Inc()
	}
	// 4. Directive 22 — fire cross-module platform event bus
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	go PublishSubmissionEvent("submission.received", instanceID, formID, tenantID, map[string]interface{}{
		"instance_id":  instanceID,
		"form_id":      formID,
		"files":        savedFiles,
		"submitted_by": submittedBy,
	})

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("ok"))
}

func saveUploadedFile(baseDir string, fh *multipart.FileHeader) (string, error) {
	file, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	outPath := filepath.Join(baseDir, filepath.Base(fh.Filename))
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return "", err
	}
	return filepath.Base(outPath), nil
}

func extractXMLSubmission(mf *multipart.Form) ([]byte, string, string) {
	// common ODK uses field name "xml_submission_file" or they post an xml file
	if mf == nil {
		return nil, "", ""
	}
	// check values first (some clients may post XML as a form value)
	for k, vals := range mf.Value {
		if k == "xml_submission_file" {
			return []byte(vals[0]), "", instanceIDFromXML(vals[0])
		}
	}

	// check files
	for key, fhs := range mf.File {
		for _, fh := range fhs {
			if isXMLFilename(fh.Filename) || key == "xml_submission_file" {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				b, err := io.ReadAll(f)
				f.Close()
				if err != nil {
					continue
				}
				return b, formIDFromXML(b), instanceIDFromXML(string(b))
			}
		}
	}

	return nil, "", ""
}

func isXMLFilename(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".xml" || ext == ".XML"
}

var instanceRe = regexp.MustCompile(`<instanceID>([^<]+)</instanceID>`)
var formIDRe = regexp.MustCompile(`<([a-zA-Z0-9_\-:]+)`)

func instanceIDFromXML(s string) string {
	m := instanceRe.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func formIDFromXML(b []byte) string {
	// attempt to find root element name as a form id (simple heuristic)
	m := formIDRe.FindSubmatch(b)
	if len(m) >= 2 {
		return string(m[1])
	}
	return ""
}

func apiKeyFromRequest(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return r.Header.Get("X-API-Key")
}

func checkAPIKey(r *http.Request) bool {
	if cfg == nil {
		return false
	}
	key := apiKeyFromRequest(r)
	if key == "" {
		return false
	}
	for _, valid := range cfg.APIKeys {
		if valid != "" && key == valid {
			return true
		}
	}
	return false
}

// checkInternalKey allows the StatGate internal service key for
// service-to-service communication.
func checkInternalKey(r *http.Request) bool {
	if cfg == nil || cfg.InternalAPIKey == "" {
		return false
	}
	key := r.Header.Get("X-StatGate-Internal-Key")
	if key == "" {
		key = r.Header.Get("X-Internal-Key")
	}
	return key != "" && key == cfg.InternalAPIKey
}

func checkAdminKey(r *http.Request) bool {
	if cfg == nil {
		return false
	}
	key := apiKeyFromRequest(r)
	if key == "" {
		return false
	}
	for _, valid := range cfg.AdminKeys {
		if valid != "" && key == valid {
			return true
		}
	}
	// Also allow Registry JWT for admin operations
	if registry != nil && registry.IsEnabled() {
		if ident := checkRegistryToken(r); ident != nil {
			for _, role := range ident.Roles {
				if role == "admin" || role == "system_admin" || role == "super_admin" {
					return true
				}
			}
		}
	}
	return false
}

// admin list submissions
func adminListHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// parse limit
	perPage := 50
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
		if page < 1 {
			page = 1
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		fmt.Sscanf(pp, "%d", &perPage)
		if perPage < 1 {
			perPage = 50
		}
	}
	offset := (page - 1) * perPage
	subs, err := ListSubmissions(perPage, offset)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out := map[string]interface{}{"page": page, "per_page": perPage, "items": subs}
	b, _ := json.MarshalIndent(out, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// admin get submission by instance_id
func adminGetHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := r.URL.Query().Get("instance_id")
	if id == "" {
		http.Error(w, "instance_id required", http.StatusBadRequest)
		return
	}
	s, xml, err := GetSubmission(id)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if s == nil {
		http.Error(w, "submission not found", http.StatusNotFound)
		return
	}
	// include StatChat link if available
	var discussion string
	if convID, err := GetStatChatLink("submission", id); err == nil {
		discussion = convID
	}
	out := map[string]interface{}{"summary": s, "xml": xml, "discussion": discussion}
	b, _ := json.MarshalIndent(out, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// admin rotate keys (persist to disk)
func adminRotateKeysHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		APIKeys   []string `json:"api_keys"`
		AdminKeys []string `json:"admin_keys"`
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	// update in-memory
	cfg.APIKeys = body.APIKeys
	cfg.AdminKeys = body.AdminKeys
	// persist to data/keys.json
	k := map[string][]string{"api_keys": cfg.APIKeys, "admin_keys": cfg.AdminKeys}
	b, _ := json.MarshalIndent(k, "", "  ")
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = "data"
	}
	os.MkdirAll(dataDir, 0o755)
	if err := os.WriteFile(filepath.Join(dataDir, "keys.json"), b, 0o600); err != nil {
		http.Error(w, "failed to save keys: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// attempt to persist to Vault if configured
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultPath := os.Getenv("STATCOLLECT_VAULT_PATH")
	if vaultAddr != "" && vaultPath != "" {
		if err := SaveKeysToVault(vaultAddr, os.Getenv("VAULT_TOKEN"), vaultPath, cfg.APIKeys, cfg.AdminKeys); err != nil {
			log.Printf("warning: failed to save keys to vault: %v", err)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// ── StatGate Platform Integration Admin Handlers ──

// adminValidateHandler approves/rejects a submission (workflow).
// POST /admin/submission/validate?instance_id=...&status=approved|rejected
func adminValidateHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	instanceID := r.URL.Query().Get("instance_id")
	status := r.URL.Query().Get("status")
	notes := r.URL.Query().Get("notes")
	if instanceID == "" {
		http.Error(w, "instance_id required", http.StatusBadRequest)
		return
	}
	if status == "" {
		status = "approved"
	}
	validator := "admin"
	if ident := checkRegistryToken(r); ident != nil && ident.UserID != "" {
		validator = ident.UserID
	}
	if err := CreateValidation(instanceID, validator, status, notes); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// ── Full enterprise cascade on quick-review ──
	// Derive form_id from the submission record for the cascade payload
	formIDForCascade := ""
	if sub, _, err2 := GetSubmission(instanceID); err2 == nil && sub != nil {
		formIDForCascade = sub.FormID
	}
	tenantIDForCascade := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantIDForCascade = cfg.TenantID
	}
	cascadePayload := map[string]interface{}{
		"instance_id": instanceID,
		"form_id":     formIDForCascade,
		"status":      status,
		"validator":   validator,
		"notes":       notes,
	}
	if status == "approved" {
		// Fire full platform cascade: GIS queue, stats export, workflow tasks, StatChat, SSE
		go PublishSubmissionEvent("submission.approved", instanceID, formIDForCascade, tenantIDForCascade, cascadePayload)
	} else {
		go PublishSubmissionEvent("submission.rejected", instanceID, formIDForCascade, tenantIDForCascade, cascadePayload)
	}
	_ = LogEvent("submission."+status, "statcollect", "submission", instanceID, cascadePayload)
	b, _ := json.MarshalIndent(map[string]string{"instance_id": instanceID, "status": status, "validator": validator}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// adminEventsHandler lists events from the audit log.
// GET /admin/events?object_type=submission&object_id=...&limit=50
func adminEventsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	objectType := r.URL.Query().Get("object_type")
	objectID := r.URL.Query().Get("object_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	events, err := ListEvents(objectType, objectID, limit)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	b, _ := json.MarshalIndent(map[string]interface{}{"events": events, "count": len(events)}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// adminObjectLinksHandler lists object connections for a submission.
// GET /admin/objects/links?object_type=submission&object_id=...
func adminObjectLinksHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	objectType := r.URL.Query().Get("object_type")
	objectID := r.URL.Query().Get("object_id")
	if objectType == "" || objectID == "" {
		http.Error(w, "object_type and object_id required", http.StatusBadRequest)
		return
	}
	links, err := ListObjectLinks(objectType, objectID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	b, _ := json.MarshalIndent(map[string]interface{}{"links": links, "count": len(links)}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// adminDiscussionHandler returns the StatChat discussion link for a submission.
// GET /admin/submission/discussion?instance_id=...&name=...&create=true
func adminDiscussionHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		http.Error(w, "instance_id required", http.StatusBadRequest)
		return
	}
	// check if link already exists
	convID, err := GetStatChatLink("submission", instanceID)
	if err == nil && convID != "" {
		b, _ := json.MarshalIndent(map[string]string{"conversation_id": convID, "instance_id": instanceID}, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		return
	}
	// check if creation requested
	if r.URL.Query().Get("create") != "true" {
		http.Error(w, "no discussion link found", http.StatusNotFound)
		return
	}
	// create the object discussion
	if statChat == nil || !statChat.IsEnabled() {
		http.Error(w, "statchat integration disabled", http.StatusServiceUnavailable)
		return
	}
	formID := ""
	if s, _, err := GetSubmission(instanceID); err == nil && s != nil {
		formID = s.FormID
	}
	name := fmt.Sprintf("Submission %s (%s)", instanceID, formID)
	convID, err = statChat.EnsureObjectDiscussion("submission", instanceID, name)
	if err != nil {
		http.Error(w, "failed to create discussion: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = SaveStatChatLink("submission", instanceID, convID)
	b, _ := json.MarshalIndent(map[string]string{"conversation_id": convID, "instance_id": instanceID}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// ── Template Management API Handlers ──
//
// Templates are stored in PostgreSQL — fully dynamic, not hardcoded.
// This allows admins to create, edit, activate, and archive questionnaires
// without code changes.

// templatesHandler handles GET /templates (list) and POST /templates (create).
func templatesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Any valid API key can list active templates
		status := r.URL.Query().Get("status")
		templates, err := ListTemplates(status)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		b, _ := json.MarshalIndent(map[string]interface{}{"templates": templates, "count": len(templates)}, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)

	case http.MethodPost:
		// Creating templates requires admin key
		if !checkAdminKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var t Template
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&t); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if t.ID == "" || t.Name == "" {
			http.Error(w, "id and name are required", http.StatusBadRequest)
			return
		}
		if t.Status == "" {
			t.Status = "active"
		}
		if t.Version == "" {
			t.Version = "1.0"
		}
		// At minimum validate schema is valid JSON
		var schemaCheck map[string]interface{}
		if err := json.Unmarshal(t.Schema, &schemaCheck); err != nil {
			http.Error(w, "invalid schema: "+err.Error(), http.StatusBadRequest)
			return
		}
		t.TenantID = cfg.TenantID
		if err := SaveTemplate(t); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("template.created", "statcollect", "template", t.ID, map[string]string{
			"template_id": t.ID,
			"name":        t.Name,
		})
		b, _ := json.MarshalIndent(t, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(b)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// templateDetailHandler handles GET /templates/{id}, PUT /templates/{id},
// DELETE /templates/{id}, and PATCH /templates/{id}/status.
func templateDetailHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/templates/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "template id required", http.StatusBadRequest)
		return
	}
	id := parts[0]

	switch r.Method {
	case http.MethodGet:
		t, err := GetTemplate(id)
		if err != nil {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}
		b, _ := json.MarshalIndent(t, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)

	case http.MethodPut:
		if !checkAdminKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var t Template
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&t); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		t.ID = id
		if t.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if t.Status == "" {
			t.Status = "active"
		}
		if t.Version == "" {
			t.Version = "1.0"
		}
		t.TenantID = cfg.TenantID
		if err := SaveTemplate(t); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("template.updated", "statcollect", "template", id, map[string]string{
			"template_id": id,
			"name":        t.Name,
		})
		b, _ := json.MarshalIndent(t, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)

	case http.MethodDelete:
		if !checkAdminKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := DeleteTemplate(id); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("template.deleted", "statcollect", "template", id, nil)
		w.WriteHeader(http.StatusNoContent)

	case http.MethodPatch:
		if !checkAdminKey(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Check if this is a status update: /templates/{id}/status
		if len(parts) >= 2 && parts[1] == "status" {
			var body struct {
				Status string `json:"status"`
			}
			dec := json.NewDecoder(r.Body)
			if err := dec.Decode(&body); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if body.Status == "" {
				http.Error(w, "status is required", http.StatusBadRequest)
				return
			}
			if err := UpdateTemplateStatus(id, body.Status); err != nil {
				http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			b, _ := json.MarshalIndent(map[string]string{"id": id, "status": body.Status}, "", "  ")
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
			return
		}
		http.Error(w, "invalid patch path", http.StatusBadRequest)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// templateVersionsHandler lists version history for a template.
// GET /templates/versions?template_id=...
func templateVersionsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	templateID := r.URL.Query().Get("template_id")
	if templateID == "" {
		http.Error(w, "template_id required", http.StatusBadRequest)
		return
	}
	versions, err := ListTemplateVersions(templateID)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	b, _ := json.MarshalIndent(map[string]interface{}{"versions": versions, "count": len(versions)}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// templateShareHandler shares a template across tenants.
// POST /templates/share {template_id, new_id, target_tenant_id}
// DELETE /templates/share {template_id}
func templateShareHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodPost:
		var body struct {
			TemplateID     string `json:"template_id"`
			NewID          string `json:"new_id"`
			TargetTenantID string `json:"target_tenant_id"`
		}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if body.TemplateID == "" {
			http.Error(w, "template_id is required", http.StatusBadRequest)
			return
		}
		if err := ShareTemplate(body.TemplateID, body.NewID, body.TargetTenantID); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		b, _ := json.MarshalIndent(map[string]string{"template_id": body.TemplateID, "status": "shared"}, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)

	case http.MethodDelete:
		var body struct {
			TemplateID string `json:"template_id"`
		}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if body.TemplateID == "" {
			http.Error(w, "template_id is required", http.StatusBadRequest)
			return
		}
		if err := UnshareTemplate(body.TemplateID); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		b, _ := json.MarshalIndent(map[string]string{"template_id": body.TemplateID, "status": "unshared"}, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(b)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// full health check
func fullHealthHandler(w http.ResponseWriter, r *http.Request) {
	res := map[string]interface{}{"ok": true}
	// check DB
	if dbPool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := dbPool.Ping(ctx); err != nil {
			res["db"] = map[string]interface{}{"ok": false, "error": err.Error()}
			res["ok"] = false
		} else {
			res["db"] = map[string]interface{}{"ok": true}
		}
	} else {
		res["db"] = map[string]interface{}{"ok": false, "error": "no db configured"}
		res["ok"] = false
	}
	// check S3 if configured
	if s3s, ok := store.(*S3Store); ok {
		// attempt head bucket
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := s3s.Client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s3s.Bucket})
		if err != nil {
			res["s3"] = map[string]interface{}{"ok": false, "error": err.Error()}
			res["ok"] = false
		} else {
			res["s3"] = map[string]interface{}{"ok": true}
		}
	} else {
		res["s3"] = map[string]interface{}{"ok": false, "info": "not configured"}
	}
	// StatGate Platform Integration status
	res["integrations"] = map[string]interface{}{
		"event_bus": eventBus.String(),
		"statchat":  statChat.String(),
		"registry":  registry.String(),
		"tenant_id": cfg.TenantID,
	}
	b, _ := json.MarshalIndent(res, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// templateCloneHandler handles POST /templates/clone {source_id, new_id, name}
func templateCloneHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SourceID string `json:"source_id"`
		NewID    string `json:"new_id"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SourceID == "" || req.NewID == "" {
		http.Error(w, "source_id and new_id are required", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = req.NewID
	}
	tmpl, err := CloneTemplate(req.SourceID, req.NewID, req.Name)
	if err != nil {
		http.Error(w, "clone failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	b, _ := json.MarshalIndent(tmpl, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(b)
}

// adminExportHandler handles GET /admin/submissions/export?form_id=...&format=csv|json|geojson
func adminExportHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	formID := r.URL.Query().Get("form_id")
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}
	subs, err := GetSubmissionsByFormID(formID, 1000)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"export_%s.csv\"", formID))
		w.Write([]byte("instance_id,form_id,submitted_by,status,received_at\n"))
		for _, s := range subs {
			w.Write([]byte(fmt.Sprintf("%s,%s,%s,%s,%s\n", s.InstanceID, s.FormID, s.SubmittedBy, s.Status, s.ReceivedAt.Format(time.RFC3339))))
		}

	case "geojson":
		w.Header().Set("Content-Type", "application/geo+json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="export_%s.geojson"`, formID))
		features := []map[string]interface{}{}
		for _, s := range subs {
			// Attempt to extract GPS from the submission XML stored in meta or DB
			var geometry interface{}
			if xmlStr, _, err2 := GetSubmission(s.InstanceID); err2 == nil && xmlStr != nil {
				if lat, lng, ok := extractGPSFromXML(xmlStr.Meta); ok {
					geometry = map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{lng, lat},
					}
				}
			}
			features = append(features, map[string]interface{}{
				"type": "Feature",
				"properties": map[string]interface{}{
					"instance_id":  s.InstanceID,
					"form_id":      s.FormID,
					"status":       s.Status,
					"submitted_by": s.SubmittedBy,
					"received_at":  s.ReceivedAt,
				},
				"geometry": geometry,
			})
		}
		res := map[string]interface{}{
			"type":     "FeatureCollection",
			"features": features,
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		w.Write(b)

	default: // json
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.MarshalIndent(subs, "", "  ")
		w.Write(b)
	}
}

// adminQAHandler handles GET /admin/submissions/qa?form_id=...
func adminQAHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	formID := r.URL.Query().Get("form_id")
	flags, err := ScanSubmissionsQA(formID)
	if err != nil {
		http.Error(w, "qa scan error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	res := map[string]interface{}{
		"form_id":     formID,
		"flagged":     flags,
		"total_flags": len(flags),
	}
	b, _ := json.MarshalIndent(res, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// adminReviewHandler handles POST /admin/submission/review {instance_id, status, notes}
func adminReviewHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req struct {
		InstanceID string `json:"instance_id"`
		Status     string `json:"status"` // approved, rejected, flagged
		Notes      string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InstanceID == "" || req.Status == "" {
		http.Error(w, "instance_id and status are required", http.StatusBadRequest)
		return
	}
	if err := CreateValidation(req.InstanceID, "admin", req.Status, req.Notes); err != nil {
		http.Error(w, "failed to update review status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = LogEvent("submission.reviewed", "statcollect", "submission", req.InstanceID, map[string]string{
		"status": req.Status,
		"notes":  req.Notes,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"success"}`))
}

// openAPIHandler serves the OpenAPI 3.0 specification for StatCollect APIs.
func openAPIHandler(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile("openapi.json")
	if err != nil {
		http.Error(w, "openapi specification not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// ── Template Library / Import / Export Handlers ──────────────

// templateLibraryHandler serves the built-in template catalog.
// GET /templates/library
func templateLibraryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entries := GetTemplateLibrary()
	b, _ := json.MarshalIndent(map[string]interface{}{
		"library": entries,
		"count":   len(entries),
	}, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// templateImportHandler handles POST /templates/import
// Body: multipart form with 'schema' file (JSON) or raw JSON body;
//
//	plus form fields: id, name, description.
func templateImportHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	var id, name, description string
	var schemaBytes []byte

	if strings.Contains(contentType, "multipart/form-data") {
		// Multipart upload
		if err := r.ParseMultipartForm(16 << 20); err != nil {
			http.Error(w, "invalid multipart: "+err.Error(), http.StatusBadRequest)
			return
		}
		id = r.FormValue("id")
		name = r.FormValue("name")
		description = r.FormValue("description")
		file, _, err := r.FormFile("schema")
		if err != nil {
			http.Error(w, "schema file required", http.StatusBadRequest)
			return
		}
		defer file.Close()
		schemaBytes, err = io.ReadAll(file)
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
	} else {
		// Raw JSON body: {id, name, description, schema: {...}}
		var body struct {
			ID          string          `json:"id"`
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Schema      json.RawMessage `json:"schema"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		id = body.ID
		name = body.Name
		description = body.Description
		schemaBytes = body.Schema
	}

	if id == "" || name == "" {
		http.Error(w, "id and name are required", http.StatusBadRequest)
		return
	}

	tmpl, err := ImportTemplateFromJSON(id, name, description, schemaBytes)
	if err != nil {
		http.Error(w, "import error: "+err.Error(), http.StatusBadRequest)
		return
	}
	tmpl.TenantID = cfg.TenantID
	if err := SaveTemplate(*tmpl); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = LogEvent("template.imported", "statcollect", "template", tmpl.ID, map[string]string{
		"template_id": tmpl.ID, "name": tmpl.Name,
	})
	b, _ := json.MarshalIndent(tmpl, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(b)
}

// templateExportHandler handles GET /templates/export?id=...&format=json|xlsform|odkxml
func templateExportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	templateID := r.URL.Query().Get("id")
	format := strings.ToLower(r.URL.Query().Get("format"))
	if templateID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if format == "" {
		format = "json"
	}

	t, err := GetTemplate(templateID)
	if err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	switch format {
	case "xlsform":
		content, err := ExportTemplateAsXLSForm(t)
		if err != nil {
			http.Error(w, "export error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/tab-separated-values")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_xlsform.txt"`, templateID))
		w.Write([]byte(content))

	case "odkxml":
		content, err := ExportTemplateAsODKXML(t)
		if err != nil {
			http.Error(w, "export error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xml"`, templateID))
		w.Write([]byte(content))

	default: // json
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.json"`, templateID))
		b, _ := json.MarshalIndent(t, "", "  ")
		w.Write(b)
	}
}

// extractGPSFromXML tries to find lat/lng from a submission's meta JSONB.
// The mandatory GPS block writes coordinates as "lat lon alt acc".
func extractGPSFromXML(meta map[string]interface{}) (lat, lng float64, ok bool) {
	if meta == nil {
		return 0, 0, false
	}
	// Look for a gps-style string in meta values
	for _, v := range meta {
		if s, isStr := v.(string); isStr {
			var la, lo float64
			if n, _ := fmt.Sscanf(s, "%f %f", &la, &lo); n == 2 {
				if la >= -90 && la <= 90 && lo >= -180 && lo <= 180 {
					return la, lo, true
				}
			}
			// Try comma-separated
			if n, _ := fmt.Sscanf(strings.ReplaceAll(s, ",", " "), "%f %f", &la, &lo); n == 2 {
				if la >= -90 && la <= 90 && lo >= -180 && lo <= 180 {
					return la, lo, true
				}
			}
		}
	}
	return 0, 0, false
}

// ── Phase Y Handlers ──

func adminDevicesHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		devices, err := ListDevices()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"devices": devices})

	case http.MethodPost:
		var d Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if d.DeviceID == "" {
			http.Error(w, "device_id is required", http.StatusBadRequest)
			return
		}
		if d.Status == "" {
			d.Status = "Active"
		}
		d.LastSyncAt = time.Now()
		if err := SaveDevice(d); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("device.registered", "statcollect", "device", d.DeviceID, d)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(d)

	case http.MethodDelete:
		id := r.URL.Query().Get("device_id")
		if id == "" {
			http.Error(w, "device_id is required", http.StatusBadRequest)
			return
		}
		if err := DeleteDevice(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("device.deleted", "statcollect", "device", id, nil)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminAssignmentsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		surveyID := r.URL.Query().Get("survey_id")
		list, err := ListAssignments(surveyID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"assignments": list})

	case http.MethodPost:
		var a Assignment
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if a.SurveyID == "" || a.TargetType == "" || a.TargetID == "" {
			http.Error(w, "survey_id, target_type, and target_id are required", http.StatusBadRequest)
			return
		}
		if a.Status == "" {
			a.Status = "Active"
		}
		if err := SaveAssignment(a); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("assignment.created", "statcollect", "assignment", a.SurveyID, a)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(a)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			http.Error(w, "valid id is required", http.StatusBadRequest)
			return
		}
		if err := DeleteAssignment(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminRegistriesHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		regType := r.URL.Query().Get("registry_type")
		histID := r.URL.Query().Get("history_id")
		if histID != "" {
			hist, err := GetRegistryLongitudinalHistory(histID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"history": hist})
			return
		}
		list, err := ListRegistries(regType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"registries": list})

	case http.MethodPost:
		var reg Registry
		if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if reg.ID == "" || reg.RegistryType == "" || reg.Name == "" {
			http.Error(w, "id, registry_type, and name are required", http.StatusBadRequest)
			return
		}
		if reg.TenantID == "" {
			reg.TenantID = cfg.TenantID
		}
		if err := SaveRegistry(reg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("registry.updated", "statcollect", "registry", reg.ID, reg)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(reg)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminSamplingHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := ListSamplingDesigns()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"sampling": list})

	case http.MethodPost:
		// Execute sampling algorithm
		var req struct {
			Name       string          `json:"name"`
			Method     string          `json:"method"`
			SampleSize int             `json:"sample_size"`
			Seed       int64           `json:"seed"`
			FrameData  json.RawMessage `json:"frame_data"` // array of items
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.Method == "" || req.SampleSize <= 0 {
			http.Error(w, "name, method, and sample_size are required", http.StatusBadRequest)
			return
		}
		var frame []interface{}
		if err := json.Unmarshal(req.FrameData, &frame); err != nil || len(frame) == 0 {
			http.Error(w, "frame_data must be a non-empty array", http.StatusBadRequest)
			return
		}

		// Simple seed-based random picker to guarantee reproducibility
		if req.Seed == 0 {
			req.Seed = time.Now().UnixNano()
		}

		// Deterministic random selection using standard formula: (seed * prime) % length
		selected := []interface{}{}
		size := req.SampleSize
		if size > len(frame) {
			size = len(frame)
		}

		currentSeed := req.Seed
		chosenIndices := make(map[int]bool)
		for len(selected) < size {
			currentSeed = (currentSeed*1103515245 + 12345) & 0x7fffffff
			idx := int(currentSeed) % len(frame)
			if !chosenIndices[idx] {
				chosenIndices[idx] = true
				selected = append(selected, frame[idx])
			}
		}

		selectedB, _ := json.Marshal(selected)
		auditLog := fmt.Sprintf("Sampled %d items from %d frame using %s. Seed preserved: %d.", len(selected), len(frame), req.Method, req.Seed)

		design := SamplingDesign{
			Name:        req.Name,
			Method:      req.Method,
			SampleSize:  len(selected),
			Seed:        req.Seed,
			FrameData:   req.FrameData,
			SelectedIDs: json.RawMessage(selectedB),
			AuditTrail:  auditLog,
		}

		if err := SaveSamplingDesign(design); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("sampling.designed", "statcollect", "sampling", req.Name, design)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(design)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		surveyID := r.URL.Query().Get("survey_id")
		if surveyID == "" {
			http.Error(w, "survey_id is required", http.StatusBadRequest)
			return
		}
		wfl, err := GetWorkflow(surveyID)
		if err != nil {
			http.Error(w, "workflow not found: "+err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wfl)

	case http.MethodPost:
		var wf Workflow
		if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if wf.SurveyID == "" {
			http.Error(w, "survey_id is required", http.StatusBadRequest)
			return
		}
		if wf.CurrentStage == "" {
			wf.CurrentStage = "draft"
		}
		if err := SaveWorkflow(wf); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("workflow.saved", "statcollect", "workflow", wf.SurveyID, wf)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(wf)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := ListSchedules()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"schedules": list})

	case http.MethodPost:
		var s Schedule
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if s.SurveyID == "" || s.CronExpression == "" {
			http.Error(w, "survey_id and cron_expression are required", http.StatusBadRequest)
			return
		}
		s.Status = "Active"
		t := time.Now().Add(24 * time.Hour) // placeholder simulated next run time
		s.NextRunAt = &t
		if err := SaveSchedule(s); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("schedule.created", "statcollect", "schedule", s.SurveyID, s)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(s)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminCommentsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		objType := r.URL.Query().Get("object_type")
		objID := r.URL.Query().Get("object_id")
		if objType == "" || objID == "" {
			http.Error(w, "object_type and object_id are required", http.StatusBadRequest)
			return
		}
		list, err := ListComments(objType, objID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"comments": list})

	case http.MethodPost:
		var c Comment
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if c.ObjectType == "" || c.ObjectID == "" || c.Comment == "" {
			http.Error(w, "object_type, object_id, and comment are required", http.StatusBadRequest)
			return
		}
		if c.Author == "" {
			c.Author = "Supervisor"
		}
		if err := SaveComment(c); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("comment.posted", "statcollect", "comment", c.ObjectID, c)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(c)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		userID := r.URL.Query().Get("user_id")
		list, err := ListNotifications(userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"notifications": list})

	case http.MethodPost:
		var n Notification
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if n.UserID == "" || n.Message == "" {
			http.Error(w, "user_id and message are required", http.StatusBadRequest)
			return
		}
		if n.Channel == "" {
			n.Channel = "app"
		}
		n.Status = "Pending"
		if err := SaveNotification(n); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(n)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func adminAIAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TemplateID string `json:"template_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TemplateID == "" {
		http.Error(w, "template_id is required", http.StatusBadRequest)
		return
	}

	t, err := GetTemplate(req.TemplateID)
	if err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	// Dynamic AI rules check simulator (checks labels, duplication, formatting, recommends enhancements)
	suggestions := []string{}
	var schema map[string]interface{}
	_ = json.Unmarshal(t.Schema, &schema)
	sections, _ := schema["sections"].([]interface{})

	fieldsCount := 0
	requiredCount := 0
	hasGPS := false
	duplicateLabels := make(map[string]bool)

	for _, sec := range sections {
		s, _ := sec.(map[string]interface{})
		fields, _ := s["fields"].([]interface{})
		for _, fld := range fields {
			f, _ := fld.(map[string]interface{})
			fieldsCount++
			lbl, _ := f["label"].(string)
			reqVal, _ := f["required"].(bool)
			typ, _ := f["type"].(string)

			if reqVal {
				requiredCount++
			}
			if typ == "gps" {
				hasGPS = true
			}
			if duplicateLabels[lbl] {
				suggestions = append(suggestions, fmt.Sprintf("AI Alert: Duplicate label text observed in field '%s'. Consider refining label texts for clarity.", lbl))
			} else {
				duplicateLabels[lbl] = true
			}
		}
	}

	if fieldsCount > 0 && float64(requiredCount)/float64(fieldsCount) < 0.2 {
		suggestions = append(suggestions, "AI Optimization: Relatively few fields are marked as 'Required'. Consider marking key demographic parameters as required to avoid data gaps.")
	}
	if !hasGPS {
		suggestions = append(suggestions, "AI Alert: Survey is missing a GPS coordinate field. Adding geolocation is highly recommended for field verification audit trails.")
	}
	if fieldsCount > 30 {
		suggestions = append(suggestions, "AI Optimization: Long questionnaire design (>30 questions). Consider splitting elements across more separate sections to minimize scroll fatigue.")
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "AI Review: Questionnaire design meets all standards. Duplicate questions check passed.")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"template_id": req.TemplateID,
		"suggestions": suggestions,
		"field_count": fieldsCount,
	})
}

func adminAIOutliersHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FormID string `json:"form_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FormID == "" {
		http.Error(w, "form_id is required", http.StatusBadRequest)
		return
	}

	subs, err := GetSubmissionsByFormID(req.FormID, 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Simple AI statistical outlier scanner
	// It parses any numbers in the meta logs, calculates bounds, and flags records outside average ranges
	outliers := []map[string]interface{}{}
	numberValues := make(map[string][]float64)

	for _, s := range subs {
		for k, v := range s.Meta {
			if num, ok := v.(float64); ok {
				numberValues[k] = append(numberValues[k], num)
			}
		}
	}

	// Calculate average and standard deviations
	stats := make(map[string]struct{ avg, std float64 })
	for k, vals := range numberValues {
		if len(vals) < 3 {
			continue // need enough data points
		}
		var sum float64
		for _, val := range vals {
			sum += val
		}
		avg := sum / float64(len(vals))
		var variance float64
		for _, val := range vals {
			variance += (val - avg) * (val - avg)
		}
		std := math.Sqrt(variance / float64(len(vals)))
		stats[k] = struct{ avg, std float64 }{avg, std}
	}

	// Scan outlier items
	for _, s := range subs {
		flaggedKeys := []string{}
		for k, v := range s.Meta {
			if num, ok := v.(float64); ok {
				if st, ok2 := stats[k]; ok2 && st.std > 0 {
					zScore := math.Abs(num-st.avg) / st.std
					if zScore > 2.0 { // outside 2 std deviations is mapped as statistical outlier
						flaggedKeys = append(flaggedKeys, fmt.Sprintf("%s = %v (Z-Score: %.2f)", k, num, zScore))
					}
				}
			}
		}
		if len(flaggedKeys) > 0 {
			outliers = append(outliers, map[string]interface{}{
				"instance_id":  s.InstanceID,
				"submitted_by": s.SubmittedBy,
				"received_at":  s.ReceivedAt,
				"outliers":     flaggedKeys,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"form_id":        req.FormID,
		"outliers_found": outliers,
		"count":          len(outliers),
	})
}
