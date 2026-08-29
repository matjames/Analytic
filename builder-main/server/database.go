package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// ComponentQueryTimeout is the maximum time allowed for a single component query.
// Configurable via COMPONENT_TIMEOUT_SECONDS environment variable.
// Prevents runaway queries from blocking the entire report generation.
var ComponentQueryTimeout = 30 * time.Second

func init() {
	if t := os.Getenv("COMPONENT_TIMEOUT_SECONDS"); t != "" {
		if sec, err := strconv.Atoi(t); err == nil && sec > 0 {
			ComponentQueryTimeout = time.Duration(sec) * time.Second
			log.Printf("ComponentQueryTimeout set to %v from COMPONENT_TIMEOUT_SECONDS", ComponentQueryTimeout)
		} else {
			log.Printf("Warning: invalid COMPONENT_TIMEOUT_SECONDS value %q, using default 30s", t)
		}
	}
}

// Connection pool configuration constants
const (
	// DBMaxOpenConns is the maximum number of open connections to the database.
	// Prevents resource exhaustion under high load. Suitable for ~100 concurrent users.
	DBMaxOpenConns = 50

	// DBMaxIdleConns is the number of idle connections kept in the pool.
	// Reduces connection latency for subsequent requests.
	DBMaxIdleConns = 5

	// DBConnMaxLifetime is how long a connection can be reused before being closed.
	// Helps with load balancing and recovering from network issues.
	DBConnMaxLifetime = 5 * time.Minute

	// DBConnMaxIdleTime is how long an idle connection stays in the pool.
	// Frees resources when traffic is low.
	DBConnMaxIdleTime = 2 * time.Minute
)

var DB *sql.DB

// DBConfig holds PostgreSQL connection configuration
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Schema   string
}

// parseEnvLine parses a single .env line, supporting both "key:value" and "key=value" formats.
// Returns key, value, and whether the line was successfully parsed.
func parseEnvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}

	// Try "=" delimiter first (standard format), then ":" (legacy format)
	var parts []string
	if idx := strings.Index(line, "="); idx > 0 {
		parts = []string{line[:idx], line[idx+1:]}
	} else if idx := strings.Index(line, ":"); idx > 0 {
		parts = []string{line[:idx], line[idx+1:]}
	} else {
		return "", "", false
	}

	key = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])

	if key == "" {
		return "", "", false
	}

	return key, value, true
}

// LoadDBConfig reads database configuration from .env file.
// Returns an error if the file is missing or required fields are absent,
// so callers (tests, etc.) can handle the absence gracefully.
// Supports both "key:value" (legacy) and "key=value" (standard) formats.
func LoadDBConfig() (DBConfig, error) {
	config := DBConfig{}

	// Check current directory first, then parent (for test environments where
	// Go changes CWD to the package directory).
	envPath := ".env"
	if _, statErr := os.Stat(envPath); os.IsNotExist(statErr) {
		envPath = "../.env"
	}
	file, err := os.Open(envPath)
	if err != nil {
		return DBConfig{}, fmt.Errorf(".env file is required but not found: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := parseEnvLine(scanner.Text())
		if !ok {
			continue
		}

		// Normalize key to lowercase for case-insensitive matching
		keyLower := strings.ToLower(key)

		switch keyLower {
		// Database configuration (set in config struct)
		case "host", "db_host":
			config.Host = value
		case "port", "db_port":
			config.Port = value
		case "username", "db_username", "db_user":
			config.User = value
		case "password", "db_password":
			config.Password = value
		case "database", "db_database", "db_name":
			config.Database = value
		case "schema", "db_schema":
			config.Schema = value

		// Auth and other configuration (set as environment variables)
		case "auth_mode", "keycloak_url", "keycloak_realm", "keycloak_client_id",
			"keycloak_roles_client_id",
			"keycloak_admin_id", "keycloak_admin_secret", "keycloak_admin_role",
			"base_path", "server_port", "rate_limit_per_sec", "component_timeout_seconds",
			"gemini_api_key", "gemini_model", "claude_api_key", "claude_model",
			"ask_enabled", "portal_api_client_url":
			// Set as environment variable (uppercase) so os.Getenv() can read it
			envKey := strings.ToUpper(key)
			// Map server_port to PORT for backwards compatibility
			if keyLower == "server_port" {
				envKey = "PORT"
			}
			if os.Getenv(envKey) == "" { // Don't override if already set in shell
				os.Setenv(envKey, value)
			}
		}
	}

	// Validate that ALL required fields are present
	if config.Host == "" {
		return DBConfig{}, fmt.Errorf(".env must contain 'host' configuration (e.g., host=localhost)")
	}
	if config.Port == "" {
		return DBConfig{}, fmt.Errorf(".env must contain 'port' configuration (e.g., port=5432)")
	}
	if config.User == "" {
		return DBConfig{}, fmt.Errorf(".env must contain 'username' configuration (e.g., username=postgres)")
	}
	if config.Password == "" {
		return DBConfig{}, fmt.Errorf(".env must contain 'password' configuration (e.g., password=YourPassword)")
	}
	if config.Database == "" {
		return DBConfig{}, fmt.Errorf(".env must contain 'database' configuration (e.g., database=uganda_dwh)")
	}
	if config.Schema == "" {
		return DBConfig{}, fmt.Errorf(".env must contain 'schema' configuration (e.g., schema=report)")
	}

	return config, nil
}

func InitDB() {
	var err error
	config, err := LoadDBConfig()
	if err != nil {
		log.Fatalf("FATAL: %v\nPlease create a .env file with database credentials (host, port, username, password, database, schema)", err)
	}

	// Build PostgreSQL connection string
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.Database)

	DB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %v", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(DBMaxOpenConns)
	DB.SetMaxIdleConns(DBMaxIdleConns)
	DB.SetConnMaxLifetime(DBConnMaxLifetime)
	DB.SetConnMaxIdleTime(DBConnMaxIdleTime)

	log.Printf("PostgreSQL connection established. Host: %s, Database: %s, Schema: %s", config.Host, config.Database, config.Schema)
	log.Printf("PostgreSQL connection pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
		DBMaxOpenConns, DBMaxIdleConns, DBConnMaxLifetime, DBConnMaxIdleTime)
}

// monthAbbreviations maps month numbers to abbreviated names
var monthAbbreviations = []string{
	"Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

// monthNames maps month numbers to full names
var monthNames = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

// FormatMonthYYYYMM converts a month number (1-12) to YYYY-MM format
// For example: 6 becomes "2024-06"
func FormatMonthYYYYMM(month string) string {
	monthNum, err := strconv.Atoi(month)
	if err != nil || monthNum < 1 || monthNum > 12 {
		return ""
	}
	return fmt.Sprintf("2024-%02d", monthNum)
}

// FormatMonthAbbreviated converts a month number (1-12) to abbreviated name
// For example: 6 becomes "Jun"
func FormatMonthAbbreviated(month string) string {
	monthNum, err := strconv.Atoi(month)
	if err != nil || monthNum < 1 || monthNum > 12 {
		return ""
	}
	return monthAbbreviations[monthNum-1]
}

// GetMonthAbbreviation gets the abbreviated month name for a number
func GetMonthAbbreviation(monthNum int) string {
	if monthNum < 1 || monthNum > 12 {
		return ""
	}
	return monthAbbreviations[monthNum-1]
}
