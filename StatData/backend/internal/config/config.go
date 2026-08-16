package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds runtime configuration parameters for StatData backend
type Config struct {
	Env               string
	Port              string
	NodeID            string
	DBHost            string
	DBPort            int
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	JWTSecret         string
	JWTIssuer         string
	JWTAudience       string
	CORSAllowedOrigin string
	DefaultTenantID   string
	MaxWorkers        int
}

// LoadConfig loads environment variables with sensible production and local fallbacks
func LoadConfig() (*Config, error) {
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	maxWorkers, err := strconv.Atoi(getEnv("MAX_WORKERS", "16"))
	if err != nil {
		maxWorkers = 16
	}

	cfg := &Config{
		Env:               getEnv("APP_ENV", "development"),
		Port:              getEnv("PORT", "8100"),
		NodeID:            getEnv("STATGATE_NODE_ID", "statdata-node-01"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            dbPort,
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_NAME", "statgate"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           redisDB,
		JWTSecret:         getEnv("JWT_SECRET", ""),
		JWTIssuer:         getEnv("JWT_ISSUER", "statgate-identity"),
		JWTAudience:       getEnv("JWT_AUDIENCE", "statgate-platform"),
		CORSAllowedOrigin: getEnv("CORS_ALLOWED_ORIGIN", "*"),
		DefaultTenantID:   getEnv("DEFAULT_TENANT_ID", "default"),
		MaxWorkers:        maxWorkers,
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
