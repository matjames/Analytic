package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config represents runtime configuration for the StatFederation microservice.
type Config struct {
	Port              string
	Env               string
	DBHost            string
	DBPort            string
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
	NodeID            string
	Jurisdiction      string
}

// LoadConfig retrieves configuration parameters with zero-default safety checks.
func LoadConfig() (*Config, error) {
	env := getEnv("STATGATE_ENV", "development")
	port := getEnv("PORT", "8099")

	dbHost := getEnv("STATFEDERATION_DB_HOST", getEnv("KAGGLE_DB_HOST", "localhost"))
	dbPort := getEnv("STATFEDERATION_DB_PORT", getEnv("KAGGLE_DB_PORT", "5432"))
	dbUser := getEnv("STATFEDERATION_DB_USER", getEnv("KAGGLE_DB_USER", "postgres"))
	dbPassword := os.Getenv("STATFEDERATION_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = os.Getenv("KAGGLE_DB_PASSWORD")
	}
	dbName := getEnv("STATFEDERATION_DB_NAME", "statfederation")
	dbSSLMode := getEnv("STATFEDERATION_DB_SSLMODE", "disable")

	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBStr := getEnv("REDIS_DB", "0")
	redisDB, _ := strconv.Atoi(redisDBStr)

	jwtSecret := os.Getenv("STATGATE_REGISTRY_JWT_SECRET")
	jwtIssuer := getEnv("STATGATE_JWT_ISSUER", "statgate-registry")
	jwtAudience := getEnv("STATGATE_JWT_AUDIENCE", "statgate")

	corsOrigin := getEnv("CORS_ALLOWED_ORIGIN", "*")
	nodeID := getEnv("STATGATE_NODE_ID", "NODE-NSS-HQ-001")
	jurisdiction := getEnv("STATGATE_JURISDICTION", "NATIONAL")

	// Fail closed in production if critical security tokens are missing
	if strings.EqualFold(env, "production") {
		if jwtSecret == "" {
			return nil, errors.New("STATGATE_REGISTRY_JWT_SECRET is strictly required in production")
		}
		if dbPassword == "" {
			return nil, errors.New("STATFEDERATION_DB_PASSWORD is strictly required in production")
		}
	}

	return &Config{
		Port:              port,
		Env:               env,
		DBHost:            dbHost,
		DBPort:            dbPort,
		DBUser:            dbUser,
		DBPassword:        dbPassword,
		DBName:            dbName,
		DBSSLMode:         dbSSLMode,
		RedisAddr:         redisAddr,
		RedisPassword:     redisPassword,
		RedisDB:           redisDB,
		JWTSecret:         jwtSecret,
		JWTIssuer:         jwtIssuer,
		JWTAudience:       jwtAudience,
		CORSAllowedOrigin: corsOrigin,
		NodeID:            nodeID,
		Jurisdiction:      jurisdiction,
	}, nil
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
