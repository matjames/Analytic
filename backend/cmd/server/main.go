package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"statgate/internal/abac"
	"statgate/internal/assets"
	"statgate/internal/lakehouse"
	"statgate/internal/semantic"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "statgate",
		Name:      "http_requests_total",
		Help:      "HTTP requests processed, labeled by method, endpoint and status",
	}, []string{"method", "endpoint", "status"})

	reqLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "statgate",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latencies in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "endpoint"})

	ingestCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "statgate",
		Name:      "ingest_events_total",
		Help:      "Total number of ingest events received",
	})

	abacDenials = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "statgate",
		Name:      "abac_denials_total",
		Help:      "ABAC policy denials, labeled by action",
	}, []string{"action"})
)

type Server struct {
	storageEngine  *lakehouse.StorageEngine
	semRegistry    *semantic.Registry
	abacEngine     *abac.Engine
	assetManager   *assets.Manager
	startTime      time.Time
	httpClient     *http.Client
	internalAPIKey string
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Prefer environment variable, then Docker secret file at /run/secrets/STATGATE_INTERNAL_API_KEY
	internalAPIKey := os.Getenv("STATGATE_INTERNAL_API_KEY")
	if internalAPIKey == "" {
		secretPath := "/run/secrets/STATGATE_INTERNAL_API_KEY"
		if b, err := os.ReadFile(secretPath); err == nil {
			internalAPIKey = strings.TrimSpace(string(b))
		}
	}
	if os.Getenv("STATGATE_ENV") == "production" && internalAPIKey == "" {
		log.Fatal("STATGATE_INTERNAL_API_KEY must be set when STATGATE_ENV=production")
	}

	// ── Database & Asset Manager Initialization ──
	// Connect to PostgreSQL so the Go Core can persist analytical assets
	// alongside the Flask layer.  The DSN is built from the same env vars
	// used by docker-compose (KAGGLE_DB_*).
	var assetMgr *assets.Manager
	dbDSN := os.Getenv("STATGATE_CORE_DB_DSN")
	if dbDSN == "" {
		dbHost := os.Getenv("KAGGLE_DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := os.Getenv("KAGGLE_DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("KAGGLE_DB_USER")
		if dbUser == "" {
			dbUser = "Kaggle"
		}
		dbPass := os.Getenv("KAGGLE_DB_PASSWORD")
		// No hardcoded fallback credential (SG-SEC-2026-08): if the password is
		// missing in production the core engine fails closed at startup.
		if dbPass == "" && os.Getenv("STATGATE_ENV") == "production" {
			log.Fatal("KAGGLE_DB_PASSWORD must be set when STATGATE_ENV=production")
		}
		dbName := os.Getenv("KAGGLE_DB_NAME")
		if dbName == "" {
			dbName = "statgate_ml_staging"
		}
		dbSSL := os.Getenv("KAGGLE_DB_SSLMODE")
		if dbSSL == "" {
			dbSSL = "disable"
		}
		dbDSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSSL)
	}

	sqlDB, err := sql.Open("pgx", dbDSN)
	if err != nil {
		log.Printf("[StatGate Core] PostgreSQL connection note: %v (asset persistence disabled)", err)
	} else {
		if pingErr := sqlDB.Ping(); pingErr != nil {
			log.Printf("[StatGate Core] PostgreSQL ping note: %v (asset persistence disabled)", pingErr)
			sqlDB.Close()
			sqlDB = nil
		} else {
			log.Printf("[StatGate Core] PostgreSQL connected — asset persistence enabled")
		}
	}

	if sqlDB != nil {
		assetMgr, err = assets.NewManager(sqlDB)
		if err != nil {
			log.Printf("[StatGate Core] Asset manager init note: %v", err)
		}
	}

	srv := &Server{
		abacEngine:     abac.NewEngine(),
		storageEngine:  lakehouse.NewStorageEngine(),
		semRegistry:    semantic.NewRegistry(),
		assetManager:   assetMgr,
		startTime:      time.Now(),
		httpClient:     &http.Client{Timeout: 5 * time.Second},
		internalAPIKey: internalAPIKey,
	}

	mux := http.NewServeMux()
	// Expose metrics endpoint (runtime + default collectors)
	mux.Handle("/metrics/prometheus", promhttp.Handler())
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/ready", srv.handleReady)
	mux.HandleFunc("/api/v1/ingest", srv.handleIngest)
	mux.HandleFunc("/api/v1/query", srv.handleQuery)
	mux.HandleFunc("/api/v1/stats", srv.handleStats)
	mux.HandleFunc("/api/v1/indicators", srv.handleIndicators)
	mux.HandleFunc("/api/v1/anomalies", srv.handleAnomalies)
	mux.HandleFunc("/api/v1/policies", srv.handlePolicies)
	mux.HandleFunc("/api/v1/kaggle/datasets", srv.handleKaggleDatasets)
	mux.HandleFunc("/api/v1/kaggle/schema", srv.handleKaggleSchema)
	mux.HandleFunc("/api/v1/datasets/fetch/", srv.handleFetchDataset)
	mux.HandleFunc("/api/v1/assets/save", srv.handleSaveAsset)
	mux.HandleFunc("/api/v1/assets/get/", srv.handleGetAsset)
	mux.HandleFunc("/api/v1/assets/list", srv.handleListAssets)
	mux.HandleFunc("/api/v1/assets/history/", srv.handleAssetHistory)
	mux.HandleFunc("/api/v1/alerts", srv.handleAlerts)
	mux.HandleFunc("/api/v1/agent/dispatch", srv.handleAgentDispatch)
	mux.HandleFunc("/api/v1/agent/actions", srv.handleAgentActions)

	bindAddress := os.Getenv("BIND_ADDRESS")
	if bindAddress == "" {
		bindAddress = "0.0.0.0"
	}
	// Wrap the composed handler (no promhttp instrumentation to avoid label misconfig)
	baseHandler := enableCORS(srv.requireInternalKey(mux))

	httpServer := &http.Server{
		Addr:              bindAddress + ":" + port,
		Handler:           baseHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("StatGate Go Analytical Orchestration Engine starting on %s", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// logAudit writes an ABAC audit record in JSON to the main logger (append-only)
func (s *Server) logAudit(uCtx abac.UserAttributes, action, resource, outcome string) {
	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"user_id":   uCtx.UserID,
		"tenant_id": uCtx.TenantID,
		"role":      uCtx.Role,
		"action":    action,
		"resource":  resource,
		"outcome":   outcome,
	}
	b, _ := json.Marshal(entry)
	log.Printf("%s", b)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "http://localhost:5000"
		}
		if origin := r.Header.Get("Origin"); origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Tenant-ID, X-User-Role, X-User-Clearance, X-StatGate-Internal-Key")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireInternalKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Probe endpoints (health/ready/metrics) are always allowed.
		if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/metrics/prometheus" {
			next.ServeHTTP(w, r)
			return
		}
		// Fail closed (SG-SEC-2026-08): if the internal API key is not
		// configured, every protected request is rejected. There is NO
		// silent downgrade to caller-supplied identity headers.
		if s.internalAPIKey == "" {
			http.Error(w, `{"error":"service_misconfigured","message":"STATGATE_INTERNAL_API_KEY is not configured"}`, http.StatusServiceUnavailable)
			return
		}
		provided := r.Header.Get("X-StatGate-Internal-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.internalAPIKey)) != 1 {
			http.Error(w, `{"error":"unauthorized internal request"}`, http.StatusUnauthorized)
			return
		}
		// Mark the request as an authenticated service-to-service call so the
		// identity layer may accept trusted-intermediary forwarding headers.
		r = r.WithContext(context.WithValue(r.Context(), internalKeyContextKey{}, true))
		next.ServeHTTP(w, r)
	})
}

// internalKeyContextKey marks a request that presented a valid internal API key.
type internalKeyContextKey struct{}

// validCoreClaims mirrors the canonical StatGate JWT claim names
// (docs/security/JWT_STANDARD.md) for the core engine.
type validCoreClaims struct {
	Sub      string `json:"sub"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	Exp      int64  `json:"exp"`
	Nbf      int64  `json:"nbf"`
	Iss      string `json:"iss"`
	Aud      string `json:"aud"`
}

// parseCoreBearerJWT verifies a compact JWS (HS256) against the shared
// STATGATE_REGISTRY_JWT_SECRET and enforces the canonical token contract.
func parseCoreBearerJWT(tokenStr, secret string) (*validCoreClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed token")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("malformed header")
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("unparseable header")
	}
	if header.Alg != "HS256" {
		return nil, fmt.Errorf("unsupported algorithm")
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid signature")
	}

	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("malformed claims")
	}
	var claims validCoreClaims
	if err := json.Unmarshal(claimBytes, &claims); err != nil {
		return nil, fmt.Errorf("unparseable claims")
	}

	now := time.Now().Unix()
	if claims.Exp == 0 || now > claims.Exp {
		return nil, fmt.Errorf("invalid or expired token")
	}
	if claims.Nbf != 0 && now < claims.Nbf {
		return nil, fmt.Errorf("token not yet valid")
	}
	issuer := getCoreEnv("STATGATE_JWT_ISSUER", "statgate-registry")
	audience := getCoreEnv("STATGATE_JWT_AUDIENCE", "statgate")
	if claims.Iss != issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	if claims.Aud != audience {
		return nil, fmt.Errorf("invalid audience")
	}
	userID := claims.UserID
	if userID == "" {
		userID = claims.Sub
	}
	if userID == "" {
		return nil, fmt.Errorf("token missing subject")
	}
	if claims.TenantID == "" {
		return nil, fmt.Errorf("token missing tenant")
	}
	claims.UserID = userID
	return &claims, nil
}

func getCoreEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// clearanceForRole maps the canonical StatGate roles to ABAC clearance levels.
func clearanceForRole(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", "superadmin", "platform_admin":
		return 5
	case "tenant_admin", "district_admin", "manager":
		return 4
	case "analyst", "editor", "operator", "governance_officer":
		return 3
	case "viewer", "agent", "district":
		return 2
	}
	return 1
}

func isSafeIdentifier(value string) bool {
	if value == "" || len(value) > 63 {
		return false
	}
	for i, char := range value {
		if !(char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || i > 0 && char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

// getUserContext derives the request identity from a verified StatGate JWT
// (Authorization: Bearer). Client-supplied identity headers (X-User-ID,
// X-User-Role, X-User-Clearance) are ONLY honoured when the request has
// already passed the internal service key gate (authenticated intermediary)
// AND no bearer JWT is present. Without either, an empty identity is
// returned so every data-access handler fails closed (SG-SEC-2026-08).
func getUserContext(r *http.Request) abac.UserAttributes {
	empty := abac.UserAttributes{}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	internalCall := false
	if v, ok := r.Context().Value(internalKeyContextKey{}).(bool); ok && v {
		internalCall = true
	}

	// Prefer the canonical bearer JWT (verified against the shared secret).
	if strings.HasPrefix(authHeader, "Bearer ") {
		secret := strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_JWT_SECRET"))
		if secret == "" {
			return empty // fail closed: no signing secret configured
		}
		claims, err := parseCoreBearerJWT(strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")), secret)
		if err != nil {
			return empty
		}
		return abac.UserAttributes{
			UserID:    claims.UserID,
			TenantID:  claims.TenantID,
			Role:      claims.Role,
			Clearance: clearanceForRole(claims.Role),
		}
	}

	// Authenticated service-to-service calls (valid internal key) may forward
	// tenant scoping for the downstream principal. This is NOT a client
	// identity path — the internal key is required.
	if internalCall {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
		role := strings.TrimSpace(r.Header.Get("X-User-Role"))
		if role == "" {
			role = "operator"
		}
		clearanceStr := r.Header.Get("X-User-Clearance")
		clearance := clearanceForRole(role)
		if clearanceStr != "" {
			if c, err := strconv.Atoi(clearanceStr); err == nil && c >= 0 && c <= 5 {
				clearance = c
			}
		}
		return abac.UserAttributes{
			UserID:    strings.TrimSpace(r.Header.Get("X-User-ID")),
			TenantID:  tenantID,
			Role:      role,
			Clearance: clearance,
		}
	}

	// No verified principal — fail closed.
	return empty
}

func (s *Server) requireUserAccess(w http.ResponseWriter, r *http.Request, resource, action string) (abac.UserAttributes, bool) {
	uCtx := getUserContext(r)
	if err := s.abacEngine.Evaluate(uCtx, resource, action, uCtx.TenantID); err != nil {
		s.logAudit(uCtx, action, resource, "denied")
		abacDenials.WithLabelValues(action).Inc()
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusForbidden)
		return uCtx, false
	}
	return uCtx, true
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"engine":  "StatGate High-Concurrency Core",
		"uptime":  time.Since(s.startTime).String(),
		"version": "1.0.0",
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ready",
		"engine":  "StatGate High-Concurrency Core",
		"uptime":  time.Since(s.startTime).String(),
		"version": "1.0.0",
	})
}

func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	if err := s.abacEngine.Evaluate(uCtx, "telemetry", "write", uCtx.TenantID); err != nil {
		s.logAudit(uCtx, "telemetry.write", "ingest", "denied")
		abacDenials.WithLabelValues("telemetry.write").Inc()
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusForbidden)
		return
	}

	var payload struct {
		ID     string                 `json:"id"`
		Source string                 `json:"source"`
		Data   map[string]interface{} `json:"data"`
		Value  float64                `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error": "invalid payload"}`, http.StatusBadRequest)
		return
	}

	record := lakehouse.EventRecord{
		ID:        payload.ID,
		TenantID:  uCtx.TenantID,
		Source:    payload.Source,
		Payload:   payload.Data,
		Timestamp: time.Now(),
		Value:     payload.Value,
	}
	if record.ID == "" {
		record.ID = fmt.Sprintf("ingest-%d", time.Now().UnixNano())
	}

	alert := s.storageEngine.Ingest(record)

	// Prometheus business metric: ingest event
	ingestCounter.Inc()

	// Audit allowed ingest
	s.logAudit(uCtx, "telemetry.write", record.ID, "allowed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "ingested",
		"record_id":     record.ID,
		"tenant_id":     record.TenantID,
		"anomaly_alert": alert,
	})
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	if err := s.abacEngine.Evaluate(uCtx, "telemetry", "read", uCtx.TenantID); err != nil {
		s.logAudit(uCtx, "telemetry.read", "query", "denied")
		abacDenials.WithLabelValues("telemetry.read").Inc()
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusForbidden)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	data := s.storageEngine.QueryTenantData(uCtx.TenantID, limit)

	s.logAudit(uCtx, "telemetry.read", "query", "allowed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": uCtx.TenantID,
		"count":     len(data),
		"records":   data,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	uCtx, ok := s.requireUserAccess(w, r, "telemetry", "read")
	if !ok {
		return
	}
	stats := s.storageEngine.GetStats(uCtx.TenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleIndicators(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.handleCreateIndicator(w, r)
		return
	}
	uCtx, ok := s.requireUserAccess(w, r, "telemetry", "read")
	if !ok {
		return
	}
	inds := s.semRegistry.ListByTenant(uCtx.TenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":  uCtx.TenantID,
		"indicators": inds,
	})
}

func (s *Server) handleCreateIndicator(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	var ind semantic.Indicator
	if err := json.NewDecoder(r.Body).Decode(&ind); err != nil {
		http.Error(w, `{"error": "invalid payload"}`, http.StatusBadRequest)
		return
	}

	ind.TenantID = uCtx.TenantID
	if ind.ID == "" {
		ind.ID = fmt.Sprintf("ind-%d", time.Now().UnixNano())
	}

	s.semRegistry.Register(ind)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ind)
}

func (s *Server) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	anomalies := s.storageEngine.GetAnomalies(uCtx.TenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": uCtx.TenantID,
		"anomalies": anomalies,
	})
}

func (s *Server) handlePolicies(w http.ResponseWriter, r *http.Request) {
	policies := s.abacEngine.GetPolicies()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policies)
}

func (s *Server) handleKaggleDatasets(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	if err := s.abacEngine.Evaluate(uCtx, "telemetry", "read", uCtx.TenantID); err != nil {
		s.logAudit(uCtx, "telemetry.read", "kaggle.datasets", "denied")
		abacDenials.WithLabelValues("telemetry.read").Inc()
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusForbidden)
		return
	}

	// Proxy live table discovery from the Python Flask layer which reads information_schema
	flaskURL := os.Getenv("FLASK_BACKEND_URL")
	if flaskURL == "" {
		flaskURL = "http://localhost:5000"
	}

	resp, err := s.httpClient.Get(flaskURL + "/api/kaggle/datasets")
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
		return
	}

	// Fallback: hardcoded real table names from statgate_ml_staging
	tables := []string{
		"billionaire_list_20yrs", "business_retailsales", "business_retailsales_1",
		"climate_vulnerability_and_readiness_2015", "co2_emissions_by_country",
		"combined_data", "county_statistics", "covid_19_data",
		"csv_building_damage_assessment", "data", "dataset",
		"dataset_metadata_catalog", "deaths_by_particulate_matter_air_pollution_vs_pm25_by_country",
		"education_by_region", "fossil_fuel_prices_1989_2019", "gdp_by_region",
		"gdp_vs_pollution_rates_by_country", "gini_by_country", "global_health",
		"green_growth_indicators_by_country", "health_kaggle_dataset", "hotel_bookings_raw",
		"hra_qna_scores", "income_per_capita_by_region", "insurance", "insurance_1",
		"master", "medquad", "percentage_of_energy_consumption_by_country",
		"percentage_of_energy_consumption_global", "pollution_emissions_by_region",
		"pollution_rate_vs_gdp_by_country", "renewable_energy_jobs_by_country",
		"renewable_energy_statistics_2010_2019", "test", "the_titanic_dataset",
		"unemployment_by_region", "world_development_data_imputed", "wri_sustainable_fiananciing_2019",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schema": "ml_staging",
		"count":  len(tables),
		"tables": tables,
	})
}

func (s *Server) handleKaggleSchema(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	if err := s.abacEngine.Evaluate(uCtx, "telemetry", "read", uCtx.TenantID); err != nil {
		s.logAudit(uCtx, "telemetry.read", "kaggle.schema", "denied")
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusForbidden)
		return
	}

	// Proxy to Flask which runs real information_schema profiling via StatGateAnalysisEngine
	tableName := r.URL.Query().Get("table_name")
	if tableName == "" {
		tableName = "covid_19_data"
	}
	if !isSafeIdentifier(tableName) {
		http.Error(w, `{"error":"invalid table name"}`, http.StatusBadRequest)
		return
	}

	flaskURL := os.Getenv("FLASK_BACKEND_URL")
	if flaskURL == "" {
		flaskURL = "http://localhost:5000"
	}

	resp, err := s.httpClient.Get(flaskURL + "/api/kaggle/schema/" + tableName)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
		return
	}

	// Fallback if Flask is unavailable
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"table_name": tableName,
		"error":      "Flask analytical engine unavailable",
	})
}

func (s *Server) handleFetchDataset(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	if err := s.abacEngine.Evaluate(uCtx, "telemetry", "read", uCtx.TenantID); err != nil {
		s.logAudit(uCtx, "telemetry.read", "kaggle.fetch", "denied")
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusForbidden)
		return
	}

	tableName := strings.TrimPrefix(r.URL.Path, "/api/v1/datasets/fetch/")
	if tableName == "" {
		tableName = "covid_19_data"
	}
	if !isSafeIdentifier(tableName) {
		http.Error(w, `{"error":"invalid table name"}`, http.StatusBadRequest)
		return
	}

	// Proxy live SELECT * FROM ml_staging.<table> LIMIT 100 via Flask connector
	flaskURL := os.Getenv("FLASK_BACKEND_URL")
	if flaskURL == "" {
		flaskURL = "http://localhost:5000"
	}

	resp, err := s.httpClient.Get(flaskURL + "/api/v1/datasets/fetch/" + tableName)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"table_name": tableName,
		"count":      0,
		"records":    []interface{}{},
		"error":      "Flask analytical engine unavailable",
	})
}

func (s *Server) handleSaveAsset(w http.ResponseWriter, r *http.Request) {
	if s.assetManager == nil {
		http.Error(w, `{"error":"asset database is not configured; use the Flask asset API or configure persistent storage"}`, http.StatusServiceUnavailable)
		return
	}
	uCtx := getUserContext(r)
	var asset assets.Asset
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		http.Error(w, `{"error": "Invalid asset JSON payload"}`, http.StatusBadRequest)
		return
	}

	asset.OwnerID = uCtx.TenantID
	if err := s.assetManager.SaveAsset(asset); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "saved",
		"id":         asset.ID,
		"owner_id":   asset.OwnerID,
		"version":    asset.VersionTag,
		"updated_at": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleGetAsset(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/get/")
	if s.assetManager != nil {
		asset, err := s.assetManager.GetAsset(id)
		if err == nil && asset != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(asset)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":                 id,
		"asset_type":         "dashboard",
		"content_definition": map[string]interface{}{"widgets": []interface{}{}},
		"owner_id":           "tenant-alpha",
		"version_tag":        "1.0.0",
	})
}

func (s *Server) handleListAssets(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)
	assetType := r.URL.Query().Get("type")

	if s.assetManager != nil {
		list, err := s.assetManager.ListAssets(uCtx.TenantID, assetType)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"owner_id": uCtx.TenantID,
				"count":    len(list),
				"assets":   list,
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"owner_id": uCtx.TenantID,
		"count":    0,
		"assets":   []interface{}{},
	})
}

func (s *Server) handleAssetHistory(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/assets/history/")
	if s.assetManager != nil {
		hist, err := s.assetManager.GetAssetHistory(id)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"asset_id": id,
				"count":    len(hist),
				"history":  hist,
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"asset_id": id,
		"count":    0,
		"history":  []interface{}{},
	})
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)

	// Fetch 3-sigma alerts from detector
	detector := lakehouse.GetAnomalyDetector()
	alerts := detector.GetAlerts(uCtx.TenantID)

	// NOTE: demo/seed alert generation was removed (SG-SEC-2026-08). The
	// production path must never fabricate operational alerts.

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": uCtx.TenantID,
		"count":     len(alerts),
		"alerts":    alerts,
	})
}

// handleAgentActions returns all registered ABAC-approved webhook actions.
func (s *Server) handleAgentActions(w http.ResponseWriter, r *http.Request) {
	worker := lakehouse.GetAgenticWorker()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"actions": worker.ListActions(),
		"count":   len(worker.ListActions()),
	})
}

// handleAgentDispatch fires a policy-approved webhook for a confirmed anomaly.
func (s *Server) handleAgentDispatch(w http.ResponseWriter, r *http.Request) {
	uCtx := getUserContext(r)

	var payload struct {
		AlertID    string  `json:"alert_id"`
		ActionID   string  `json:"action_id"`
		MetricName string  `json:"metric_name"`
		Dataset    string  `json:"dataset"`
		Value      float64 `json:"value"`
		Sigma      float64 `json:"sigma"`
		Severity   string  `json:"severity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	alert := lakehouse.AnomalyAlert{
		ID:         payload.AlertID,
		TenantID:   uCtx.TenantID,
		MetricName: payload.MetricName,
		Dataset:    payload.Dataset,
		Value:      payload.Value,
		SigmaScore: payload.Sigma,
		Severity:   payload.Severity,
		Message:    payload.MetricName + " triggered auto-action",
	}

	worker := lakehouse.GetAgenticWorker()
	err := worker.DispatchIfApproved(alert, payload.ActionID, uCtx.Clearance)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "denied",
			"reason": err.Error(),
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "dispatched",
		"action_id": payload.ActionID,
		"tenant_id": uCtx.TenantID,
	})
}
