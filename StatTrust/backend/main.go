package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/matjames/statgate-lib/auth"
	"github.com/matjames/statgate-lib/tenant"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.Println("Starting StatGate App 3: Security, Trust & Identity (StatTrust)...")

	// Initialize In-Memory Store
	globalStore = NewMemStore()

	// Initialize Optional PostgreSQL and Redis (non-blocking)
	go func() {
		_, _ = initDB()
	}()
	go func() {
		_ = initRedis()
	}()

	router := gin.Default()

	// ── CORS ──────────────────────────────────────────────────────────────────
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Internal-API-Key", "X-Request-ID", "X-Tenant-ID"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ── Mandatory Platform Probes (no auth) ───────────────────────────────────
	router.GET("/health", HealthHandler)
	router.GET("/ready", ReadyHandler)
	router.GET("/readyz", ReadyHandler)
	router.GET("/live", HealthHandler)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ── Zero-Trust Auth Middleware ─────────────────────────────────────────────
	jwtValidator, err := auth.NewValidator(
		getEnv("STATGATE_REGISTRY_JWT_SECRET", ""),
		getEnv("STATGATE_JWT_ISSUER", "statgate-registry"),
		getEnv("STATGATE_JWT_AUDIENCE", "statgate"),
	)
	if err != nil {
		log.Printf("[WARN] JWT validator not configured — running in OPEN mode: %v", err)
	}

	// ── API v1 ─────────────────────────────────────────────────────────────────
	v1 := router.Group("/api/v1")

	// Apply auth + tenant isolation only when JWT is configured
	if jwtValidator != nil {
		v1.Use(jwtValidator.GinMiddleware())
		v1.Use(tenant.GinTenantIsolation())
	} else {
		// Inject a passthrough tenant from header for dev mode
		v1.Use(func(c *gin.Context) {
			hdr := c.GetHeader("X-Tenant-ID")
			if hdr == "" {
				hdr = "tenant-alpha"
			}
			c.Set("tenant_id", hdr)
			c.Next()
		})
	}

	{
		v1.GET("/summary", SummaryHandler)

		// ── SecOps & Cyber (P19) ───────────────────────────────────────────────
		secops := v1.Group("/secops")
		{
			secops.GET("/incidents", ListIncidentsHandler)
			secops.POST("/incidents", CreateIncidentHandler)
			secops.PUT("/incidents/:id/status", UpdateIncidentStatusHandler)

			// DLP
			secops.POST("/dlp/scan", DLPScanHandler)

			// Privacy & Consent
			secops.GET("/privacy/consents", ListConsentsHandler)
			secops.POST("/privacy/consents", CreateConsentHandler)

			// BCM / Disaster Recovery
			secops.GET("/bcm/checks", ListBCMChecksHandler)

			// Encryption Key Rotation (P19 — Automated Key Rotation)
			secops.GET("/keys", ListEncryptionKeysHandler)
			secops.POST("/keys/rotate", RotateEncryptionKeyHandler)
			secops.GET("/keys/:id", GetEncryptionKeyHandler)
		}

		// ── Digital Trust & Blockchain (P28) ──────────────────────────────────
		trust := v1.Group("/trust")
		{
			// Immutable Audit Ledger
			trust.GET("/ledger/records", ListLedgerHandler)
			trust.POST("/ledger/append", AppendLedgerHandler)

			// Verifiable Credentials
			trust.GET("/credentials", ListCredentialsHandler)
			trust.POST("/credentials/issue", IssueCredentialHandler)
			trust.POST("/credentials/verify", VerifyCredentialHandler)
			trust.DELETE("/credentials/:id/revoke", RevokeCredentialHandler)

			// Artifact Provenance & Chain of Custody
			trust.GET("/provenance", ListProvenanceHandler)
			trust.GET("/provenance/:id", GetProvenanceHandler)
			trust.POST("/provenance/register", RegisterProvenanceHandler)
			trust.POST("/provenance/:id/transfer", TransferProvenanceHandler)

			// Digital Signatures
			trust.POST("/signatures/sign", SignArtifactHandler)
			trust.POST("/signatures/verify", VerifySignatureHandler)

			// Trust Registry (P28 — Trust Registry)
			trust.GET("/registry", ListTrustRegistryHandler)
			trust.POST("/registry", RegisterTrustEntryHandler)
			trust.GET("/registry/:id", GetTrustEntryHandler)
			trust.PUT("/registry/:id/status", UpdateTrustEntryHandler)

			// PKI Certificate Authority (P28 — PKI Infrastructure)
			trust.POST("/pki/csr", IssueCertificateHandler)
			trust.GET("/pki/certificates", ListCertificatesHandler)
			trust.POST("/pki/certificates/:id/revoke", RevokeCertificateHandler)

			// Timestamp Authority (P28 — Timestamp Authority)
			trust.POST("/tsa/stamp", TimestampHandler)
			trust.GET("/tsa/verify/:id", VerifyTimestampHandler)
		}

		// ── Inter-App Ecosystem Mesh ───────────────────────────────────────────
		interop := v1.Group("/interop")
		{
			interop.GET("/apps", ListInterAppsHandler)
			interop.POST("/apps/probe", ProbeInterAppHandler)
		}
	}

	// ── Start Server ──────────────────────────────────────────────────────────
	port := getEnv("PORT", "8094")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("StatTrust Backend operational on port %s (auth_mode=%s)", port, authMode(jwtValidator))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down StatTrust service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("StatTrust service exited cleanly.")
}

func authMode(v *auth.Validator) string {
	if v != nil {
		return "ZERO-TRUST"
	}
	return "OPEN-DEV"
}
