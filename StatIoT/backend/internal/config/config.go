package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env               string
	Port              string
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
	GatewayAuthSecret string
}

func LoadConfig() (*Config, error) {
	env := getEnv("STATGATE_ENV", "development")

	port := getEnv("STATIOT_PORT", getEnv("PORT", "8103"))
	dbHost := getEnv("STATIOT_DB_HOST", getEnv("KAGGLE_DB_HOST", "localhost"))
	dbPort := getEnv("STATIOT_DB_PORT", getEnv("KAGGLE_DB_PORT", "5432"))
	dbUser := getEnv("STATIOT_DB_USER", getEnv("KAGGLE_DB_USER", "postgres"))
	dbPassword := os.Getenv("STATIOT_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = os.Getenv("KAGGLE_DB_PASSWORD")
	}
	dbName := getEnv("STATIOT_DB_NAME", "statiot")
	dbSSLMode := getEnv("STATIOT_DB_SSLMODE", "disable")

	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisAddr := ""
	if redisHost != "" {
		redisAddr = redisHost + ":" + redisPort
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBStr := getEnv("REDIS_DB", "0")
	redisDB, _ := strconv.Atoi(redisDBStr)

	jwtSecret := getEnv("STATGATE_REGISTRY_JWT_SECRET", "statgate-dev-secret-key-32bytes-long!")
	jwtIssuer := getEnv("STATGATE_JWT_ISSUER", "statgate-registry")
	jwtAudience := getEnv("STATGATE_JWT_AUDIENCE", "statgate")
	corsOrigin := getEnv("CORS_ALLOWED_ORIGIN", "*")
	gatewayAuthSecret := getEnv("STATIOT_GATEWAY_SECRET", "statiot-device-auth-secret-key-prod")

	if strings.EqualFold(env, "production") {
		if dbPassword == "" {
			return nil, errors.New("STATIOT_DB_PASSWORD is required in production mode (SG-SEC-ZERO-DEFAULT)")
		}
		if jwtSecret == "statgate-dev-secret-key-32bytes-long!" {
			return nil, errors.New("STATGATE_REGISTRY_JWT_SECRET must be explicitly configured in production mode")
		}
	}

	return &Config{
		Env:               env,
		Port:              port,
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
		GatewayAuthSecret: gatewayAuthSecret,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
