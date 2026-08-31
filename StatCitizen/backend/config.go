package main

import (
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds all StatCitizen configuration.
// All values are sourced from environment variables — no hardcoded production values.
type Config struct {
	Env     string
	Port    string
	UIPort  string

	// Database
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	EventChannel  string

	// Security
	CitizenSessionSecret  string
	EnterpriseJWTSecret   string
	InternalAPIKey        string

	// Enterprise integration URLs
	EnterpriseCoreURL    string
	EnterpriseSearchURL  string
	HelpDeskAPIURL       string
	RegistryAPIURL       string
	StatCollectAPIURL    string
	PMSAPIURL            string
	RMSAPIURL            string
	GovernanceAPIURL     string

	// CORS
	CORSOrigins []string

	// Rate limiting
	RateLimitRPM   int
	MaxRequestKB   int64
	MaxAttachmentMB int64

	// Features
	CaptchaEnabled     bool
	CaptchaSecret      string
	OfflineSyncEnabled bool
	FailFastWithoutDB  bool

	// File storage
	UploadDir string
	MaxFileSize int64

	// Tenant defaults
	DefaultTenantID string
}

func loadConfig() *Config {
	return &Config{
		Env:    getEnv("STATGATE_ENV", "development"),
		Port:   getEnv("STATCITIZEN_API_PORT", "8115"),
		UIPort: getEnv("STATCITIZEN_UI_PORT", "8115"),

		DBHost:     getEnv("STATCITIZEN_DB_HOST", "localhost"),
		DBPort:     getEnv("STATCITIZEN_DB_PORT", "5432"),
		DBName:     getEnv("STATCITIZEN_DB_NAME", "statcitizen"),
		DBUser:     getEnv("STATCITIZEN_DB_USER", "statcitizen"),
		DBPassword: getEnv("STATCITIZEN_DB_PASSWORD", ""),
		DBSSLMode:  getEnv("STATCITIZEN_DB_SSLMODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		EventChannel:  getEnv("STATGATE_EVENT_CHANNEL", "statgate:events"),

		CitizenSessionSecret: getEnv("STATCITIZEN_CITIZEN_SESSION_SECRET", ""),
		EnterpriseJWTSecret:  getEnv("STATGATE_REGISTRY_JWT_SECRET", ""),
		InternalAPIKey:       getEnv("STATGATE_INTERNAL_API_KEY", ""),

		EnterpriseCoreURL:   getEnv("ENTERPRISE_CORE_URL", "http://localhost:8096"),
		EnterpriseSearchURL: getEnv("ENTERPRISE_SEARCH_URL", "http://localhost:8095"),
		HelpDeskAPIURL:      getEnv("HELPDESK_API_URL", "http://localhost:5006"),
		RegistryAPIURL:      getEnv("STATGATE_REGISTRY_API_URL", "http://localhost:9090/api"),
		StatCollectAPIURL:   getEnv("STATCOLLECT_API_URL", "http://localhost:8080"),
		PMSAPIURL:           getEnv("PMS_API_URL", "http://localhost:8091"),
		RMSAPIURL:           getEnv("RMS_API_URL", "http://localhost:8092"),
		GovernanceAPIURL:    getEnv("GOVERNANCE_API_URL", "http://localhost:8093"),

		CORSOrigins: parseCORSOrigins(getEnv("STATCITIZEN_CORS_ORIGINS",
			"http://localhost:3015,http://localhost:3006,http://localhost:3002")),

		RateLimitRPM:   parseInt(getEnv("STATCITIZEN_RATE_LIMIT_RPM", "60")),
		MaxRequestKB:   int64(parseInt(getEnv("STATCITIZEN_MAX_REQUEST_BODY_KB", "10240"))),
		MaxAttachmentMB: int64(parseInt(getEnv("STATCITIZEN_MAX_ATTACHMENT_MB", "10"))),

		CaptchaEnabled: getEnv("STATCITIZEN_CAPTCHA_ENABLED", "false") == "true",
		CaptchaSecret:  getEnv("STATCITIZEN_CAPTCHA_SECRET", ""),

		OfflineSyncEnabled: getEnv("STATCITIZEN_OFFLINE_SYNC_ENABLED", "true") == "true",
		FailFastWithoutDB:  getEnv("FAIL_FAST_WITHOUT_DB", "false") == "true",

		UploadDir:   getEnv("STATCITIZEN_UPLOAD_DIR", "./uploads"),
		MaxFileSize: int64(parseInt(getEnv("STATCITIZEN_MAX_ATTACHMENT_MB", "10"))) * 1024 * 1024,

		DefaultTenantID: getEnv("STATCITIZEN_DEFAULT_TENANT", "tenant_default"),
	}
}

func validateProductionSecrets(cfg *Config) {
	required := [][]string{
		{"STATCITIZEN_CITIZEN_SESSION_SECRET", cfg.CitizenSessionSecret},
		{"STATGATE_REGISTRY_JWT_SECRET", cfg.EnterpriseJWTSecret},
		{"STATGATE_INTERNAL_API_KEY", cfg.InternalAPIKey},
		{"STATCITIZEN_DB_PASSWORD", cfg.DBPassword},
		{"STATCITIZEN_DB_USER", cfg.DBUser},
	}
	var missing []string
	for _, pair := range required {
		if pair[1] == "" {
			missing = append(missing, pair[0])
		}
	}
	if len(missing) > 0 {
		log.Fatalf("[FATAL] Required production secrets missing: %v. Startup aborted.", missing)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

func parseCORSOrigins(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
