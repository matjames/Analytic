package main

import (
	"log"
	"os"
)

// ─── Production Secret Validation ────────────────────────────────
// validateProductionSecrets checks that all mandatory secrets are
// populated when STATGATE_ENV=production.
//
// Any service configured for production that is missing a mandatory
// secret is immediately failed — a blank secret is worse than no service.

var mandatoryProductionSecrets = []struct {
	EnvVar  string
	Purpose string
}{
	{"STATGATE_REGISTRY_JWT_SECRET", "JWT token signing and verification"},
	{"STATGATE_INTERNAL_API_KEY", "Internal service-to-service authentication"},
}

func validateProductionSecrets() {
	env := os.Getenv("STATGATE_ENV")
	if env != "production" {
		// In development / staging, warn but do not block
		for _, s := range mandatoryProductionSecrets {
			if os.Getenv(s.EnvVar) == "" {
				log.Printf("startup: [WARNING] %s is not set (required in production) — purpose: %s", s.EnvVar, s.Purpose)
			}
		}
		return
	}

	// Production: any missing mandatory secret is fatal.
	failed := false
	for _, s := range mandatoryProductionSecrets {
		if os.Getenv(s.EnvVar) == "" {
			log.Printf("startup: [FATAL] mandatory secret %s is not set — %s", s.EnvVar, s.Purpose)
			failed = true
		}
	}
	if failed {
		log.Fatal("startup: refusing to start in production mode with missing secrets. Set all mandatory environment variables.")
	}

	log.Println("startup: production secret validation passed")
}
