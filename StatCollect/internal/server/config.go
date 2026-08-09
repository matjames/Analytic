package server

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Config holds configurable parameters for StatCollect
type Config struct {
	Port        string
	DataDir     string
	DBDSN       string
	S3Bucket    string
	S3Region    string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	UseS3       bool
	APIKeys     []string
	AdminKeys   []string

	// StatGate Platform Integration
	RegistryJWTSecret string
	RedisAddr         string
	RedisPassword     string
	StatChatURL       string
	RegistryURL       string
	InternalAPIKey    string
	InternalKey       string // alias used by platform_integration.go
	TenantID          string
	EnableEvents      bool
	EnableStatChat    bool

	// Directive 22 — Cross-Module Endpoint URLs
	// Set these in .env to enable live publishing to StatGate modules.
	// Leave empty ("-") to disable publishing to a specific module.
	ResearchURL   string // StatGate Research module ingest endpoint
	ProjectsURL   string // StatGate Projects module ingest endpoint
	StatisticsURL string // StatGate Statistics module ingest endpoint
	ReportingURL  string // StatGate Reporting module ingest endpoint
	GISURL        string // StatGate GIS module ingest endpoint
	DocumentsURL  string // StatGate Document Management ingest endpoint
}

// LoadConfig reads environment variables and returns a Config with defaults
func LoadConfig() *Config {
	port := os.Getenv("STATCOLLECT_PORT")
	if port == "" {
		port = ":8080"
	}
	data := os.Getenv("STATCOLLECT_DATA_DIR")
	if data == "" {
		data = "data"
	}
	dsn := os.Getenv("STATCOLLECT_DB_DSN")
	if dsn == "" {
		user := os.Getenv("STATCOLLECT_DB_USER")
		if user == "" {
			log.Fatal("STATCOLLECT_DB_USER must be set")
		}
		pass := os.Getenv("STATCOLLECT_DB_PASSWORD")
		if pass == "" {
			log.Fatal("STATCOLLECT_DB_PASSWORD must be set")
		}
		db := os.Getenv("STATCOLLECT_DB_NAME")
		if db == "" {
			db = "statcollect"
		}
		host := os.Getenv("STATCOLLECT_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", user, pass, host, db)
	}

	s3b := os.Getenv("STATCOLLECT_S3_BUCKET")
	s3r := os.Getenv("STATCOLLECT_S3_REGION")
	s3e := os.Getenv("STATCOLLECT_S3_ENDPOINT")
	s3k := os.Getenv("STATCOLLECT_S3_KEY")
	s3s := os.Getenv("STATCOLLECT_S3_SECRET")
	useS3 := os.Getenv("STATCOLLECT_USE_S3") == "true"

	// load API keys (comma-separated) or single key
	keysEnv := os.Getenv("STATCOLLECT_API_KEYS")
	if keysEnv == "" {
		single := os.Getenv("STATCOLLECT_API_KEY")
		if single != "" {
			keysEnv = single
		}
	}
	var keys []string
	if keysEnv != "" {
		for _, k := range strings.Split(keysEnv, ",") {
			kk := strings.TrimSpace(k)
			if kk != "" {
				keys = append(keys, kk)
			}
		}
	}

	// admin keys
	adminEnv := os.Getenv("STATCOLLECT_ADMIN_KEYS")
	var adminKeys []string
	if adminEnv != "" {
		for _, k := range strings.Split(adminEnv, ",") {
			kk := strings.TrimSpace(k)
			if kk != "" {
				adminKeys = append(adminKeys, kk)
			}
		}
	}

	// load persisted keys from data/keys.json if present
	keysFile := filepath.Join(data, "keys.json")
	if b, err := os.ReadFile(keysFile); err == nil {
		var m map[string][]string
		if err := json.Unmarshal(b, &m); err == nil {
			if ak, ok := m["api_keys"]; ok && len(ak) > 0 {
				keys = ak
			}
			if ad, ok := m["admin_keys"]; ok && len(ad) > 0 {
				adminKeys = ad
			}
		}
	}

	// if Vault configured, attempt to load keys from Vault (overrides file)
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultPath := os.Getenv("STATCOLLECT_VAULT_PATH")
	if vaultAddr != "" && vaultPath != "" {
		if vapiKeys, vadminKeys, err := LoadKeysFromVault(vaultAddr, os.Getenv("VAULT_TOKEN"), vaultPath); err == nil {
			if len(vapiKeys) > 0 {
				keys = vapiKeys
			}
			if len(vadminKeys) > 0 {
				adminKeys = vadminKeys
			}
		}
	}

	// StatGate Platform Integration settings
	registrySecret := os.Getenv("STATGATE_REGISTRY_JWT_SECRET")
	if registrySecret == "" {
		registrySecret = os.Getenv("JWT_SECRET")
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_HOST")
		if redisAddr == "" {
			redisAddr = "localhost:6379"
		} else {
			redisPort := os.Getenv("REDIS_PORT")
			if redisPort == "" {
				redisPort = "6379"
			}
			redisAddr = fmt.Sprintf("%s:%s", redisAddr, redisPort)
		}
	}
	statchatURL := os.Getenv("STATCHAT_API_URL")
	if statchatURL == "" {
		statchatURL = "http://localhost:4000"
	}
	registryURL := os.Getenv("STATGATE_REGISTRY_API_URL")
	if registryURL == "" {
		registryURL = "http://localhost:9090/api"
	}
	internalKey := os.Getenv("STATGATE_INTERNAL_API_KEY")
	if internalKey == "" {
		log.Fatal("STATGATE_INTERNAL_API_KEY must be set")
	}
	tenantID := os.Getenv("STATGATE_TENANT_ID")
	if tenantID == "" {
		tenantID = "default"
	}

	return &Config{
		Port:              port,
		DataDir:           data,
		DBDSN:             dsn,
		S3Bucket:          s3b,
		S3Region:          s3r,
		S3Endpoint:        s3e,
		S3AccessKey:       s3k,
		S3SecretKey:       s3s,
		UseS3:             useS3,
		APIKeys:           keys,
		AdminKeys:         adminKeys,
		RegistryJWTSecret: registrySecret,
		RedisAddr:         redisAddr,
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		StatChatURL:       statchatURL,
		RegistryURL:       registryURL,
		InternalAPIKey:    internalKey,
		InternalKey:       internalKey,
		TenantID:          tenantID,
		EnableEvents:      os.Getenv("STATCOLLECT_ENABLE_EVENTS") == "true",
		EnableStatChat:    os.Getenv("STATCOLLECT_ENABLE_STATCHAT") == "true",
		// Directive 22 module endpoints (default to "-" = disabled until configured)
		ResearchURL:   getEnvOrDefault("STATGATE_RESEARCH_URL", "-"),
		ProjectsURL:   getEnvOrDefault("STATGATE_PROJECTS_URL", "-"),
		StatisticsURL: getEnvOrDefault("STATGATE_STATISTICS_URL", "-"),
		ReportingURL:  getEnvOrDefault("STATGATE_REPORTING_URL", "-"),
		GISURL:        getEnvOrDefault("STATGATE_GIS_URL", "-"),
		DocumentsURL:  getEnvOrDefault("STATGATE_DOCUMENTS_URL", "-"),
	}
}

// getEnvOrDefault returns the env var value or the provided default.
func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
