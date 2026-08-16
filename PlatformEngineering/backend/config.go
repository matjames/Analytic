package main

import (
	"encoding/json"
	"os"
	"strings"
)

type Config struct {
	Port, DBHost, DBPort, DBName, DBUser, DBPassword, DBSSLMode string
	RedisHost, RedisPort, EventChannel                          string
	ProbeIntervalSeconds                                        int
	ServiceTargets                                              []ServiceTarget
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() Config {
	c := Config{Port: env("PORT", "8080"), DBHost: env("RUNOPS_DB_HOST", "postgres"), DBPort: env("RUNOPS_DB_PORT", "5432"), DBName: env("RUNOPS_DB_NAME", "statgate_enterprise"), DBUser: env("RUNOPS_DB_USER", "Kaggle"), DBPassword: env("RUNOPS_DB_PASSWORD", ""), DBSSLMode: env("RUNOPS_DB_SSLMODE", "disable"), RedisHost: env("REDIS_HOST", "redis"), RedisPort: env("REDIS_PORT", "6379"), EventChannel: env("STATGATE_EVENT_CHANNEL", "statgate:events"), ProbeIntervalSeconds: 30}
	// Docker DNS names make these targets available to every RunOps instance on statgate-network.
	c.ServiceTargets = []ServiceTarget{{"enterprise-core", "Enterprise Core", "http://statgate-enterprise-core:8096", "tier-0", true}, {"governance", "StatGovernance", "http://statgate-governance-api:8080", "tier-1", true}, {"trust", "StatTrust", "http://statgate-trust-api:8080", "tier-1", true}, {"registry", "Registry", "http://statgate-registry-api:9090", "tier-0", true}, {"pms", "PMS", "http://statgate-pms-api:8080", "tier-1", true}, {"rms", "RMS", "http://statgate-rms-api:8080", "tier-1", true}}
	if raw := strings.TrimSpace(os.Getenv("RUNOPS_SERVICE_TARGETS")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &c.ServiceTargets)
	}
	return c
}
