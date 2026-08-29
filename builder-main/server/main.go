// Package main: StatGate Report Builder - READ-ONLY SERVICE
// This application provides read-only access to report data from PostgreSQL
// No data modifications (INSERT, UPDATE, DELETE) are possible
// All database operations are parameterized SELECT queries with SQL injection protection
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// basePath holds the URL path prefix for subpath deployments (e.g., "/dashboards/viz")
// Empty string means root deployment (default)
var basePath string

// assetVersion is set at startup for cache-busting CSS/JS references in HTML.
// Changes on every server restart so browsers re-fetch updated static assets.
var assetVersion string

// localAssetRe matches local (relative) CSS/JS asset references in HTML.
// Captures: href="css/style.css" or src="js/app.js" (but not https:// CDN URLs).
var localAssetRe = regexp.MustCompile(`((?:href|src)="(?:css|js)/[^"]+\.(?:css|js))"`)

func main() {
	// Initialize server logger (logs to both stdout and logs/server.log)
	if err := InitServerLogger(); err != nil {
		log.Printf("Warning: Could not initialize server logger: %v", err)
	}
	defer CloseServerLogger()

	// Initialize metrics store and load persisted data from previous sessions
	InitMetrics()
	LoadMetrics()
	stopPeriodicSave := StartPeriodicSave()

	// Load cached component results from previous session
	LoadCache()
	stopPeriodicCacheSave := StartPeriodicCacheSave()

	// REQUIRED: .env file must exist with database credentials.
	// Service will NOT start if the connection fails.
	InitDB()

	// SQLite write stores (feedback + requirements). Scoped write exception — see CLAUDE.md.
	if err := InitFeedbackDB(); err != nil {
		log.Fatalf("Feedback DB init failed: %v", err)
	}
	InitFeedbackConfig()
	if err := InitRequirementsDB(); err != nil {
		log.Fatalf("Requirements DB init failed: %v", err)
	}
	defer CloseRequirementsDB()

	// Build CSP now that .env vars (KEYCLOAK_URL etc.) are available
	InitCSP()

	// Initialize Keycloak JWT authentication
	// Reads KEYCLOAK_URL, KEYCLOAK_REALM, KEYCLOAK_CLIENT_ID, AUTH_MODE (off/on) from env
	InitAuth()

	// Initialize AI provider for chat feature (Claude or Gemini, based on .env)
	InitAI()

	// Initialize SQL logger
	if err := InitSQLLogger(); err != nil {
		log.Printf("Warning: Could not initialize SQL logger: %v", err)
	}
	defer CloseSQLLogger()

	// Get the directory of the current executable
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	exeDir := filepath.Dir(exePath)

	// Read BASE_PATH for subpath deployment (e.g., "/dashboards/viz")
	// Empty or unset means root deployment
	basePath = strings.TrimSuffix(os.Getenv("BASE_PATH"), "/")
	if basePath != "" {
		log.Printf("BASE_PATH configured: %s", basePath)
	}

	// Generate asset version from startup time for cache-busting
	assetVersion = fmt.Sprintf("%d", time.Now().Unix())

	// Serve static files from public/ directory
	publicDir := filepath.Join(exeDir, "public")
	log.Printf("Serving static files from: %s", publicDir)

	// API routes - Auth config (PUBLIC - no auth required)
	// Frontend needs this to determine whether to initialize Keycloak
	http.HandleFunc(basePath+"/api/auth/config", WithRequestID(GetAuthConfigHandler))

	// Frontend parameters
	http.HandleFunc(basePath+"/api/frontend/config", WithRequestID(GetFrontendConfigHandler))

	// Current user profile compatible with StatGate Portal /api/v1/auth/me
	http.HandleFunc(basePath+"/api/me", WithRequestID(AuthMiddleware(HandleMe)))

	// Current user permissions for menu visibility and guarded UI
	http.HandleFunc(basePath+"/api/me/permissions", WithRequestID(AuthMiddleware(GetMePermissionsHandler)))

	// API routes - Report serving (with base path prefix)
	// All API handlers wrapped with WithRequestID for request tracing
	// AuthMiddleware validates JWT tokens based on AUTH_MODE (off/on)
	http.HandleFunc(basePath+"/api/reports", WithRequestID(AuthMiddleware(ListReportsHandler)))
	http.HandleFunc(basePath+"/api/report/pdf", WithRequestID(AuthMiddleware(GeneratePDFHandler)))
	http.HandleFunc(basePath+"/api/report/builder-preview", WithRequestID(AuthMiddleware(PreviewReportHandler)))
	// Report source endpoint for editing (must be before generic /api/report/ handler)
	http.HandleFunc(basePath+"/api/report/source/", WithRequestID(AuthMiddleware(GetReportSourceHandler)))
	// Unlimited CSV download for table_advanced components (no AdvancedSQLMaxRows cap)
	http.HandleFunc(basePath+"/api/download/csv", WithRequestID(AuthMiddleware(DownloadTableCSVHandler)))
	// Wrap report handler with metrics middleware to track usage
	http.HandleFunc(basePath+"/api/report/", WithRequestID(AuthMiddleware(MetricsMiddleware(GetReportHandler, extractReportIDFromRequest))))

	// API routes - Metrics & Cache
	// Metrics endpoint is PUBLIC (operational stats, no user data)
	http.HandleFunc(basePath+"/api/metrics/csv", WithRequestID(RecordCSVDownloadHandler))
	http.HandleFunc(basePath+"/api/metrics/ui-event", WithRequestID(PostUIEventHandler))
	http.HandleFunc(basePath+"/api/metrics", WithRequestID(GetMetricsHandler))
	http.HandleFunc(basePath+"/api/cache/stats", WithRequestID(AuthMiddleware(GetCacheStatsHandler)))
	http.HandleFunc(basePath+"/api/cache/clear", WithRequestID(RequireRole("admin", ClearCacheHandler)))

	// Keycloak Admin API proxy for user management
	http.HandleFunc(basePath+"/api/keycloak/", WithRequestID(RequireKeycloakAdminRole(KeycloakAdminHandler)))

	// API routes - Filters (require valid token)
	http.HandleFunc(basePath+"/api/filters/districts", WithRequestID(AuthMiddleware(GetDistrictsHandler)))
	http.HandleFunc(basePath+"/api/filters/years", WithRequestID(AuthMiddleware(GetYearsHandler)))
	http.HandleFunc(basePath+"/api/filters/months", WithRequestID(AuthMiddleware(GetMonthsHandler)))
	http.HandleFunc(basePath+"/api/filters/quarters", WithRequestID(AuthMiddleware(GetQuartersHandler)))
	http.HandleFunc(basePath+"/api/filters/weeks", WithRequestID(AuthMiddleware(GetWeeksHandler)))
	http.HandleFunc(basePath+"/api/filters/regions", WithRequestID(AuthMiddleware(GetRegionsHandler)))
	http.HandleFunc(basePath+"/api/filters/facilities", WithRequestID(AuthMiddleware(GetFacilitiesHandler)))
	http.HandleFunc(basePath+"/api/filters/custom", WithRequestID(AuthMiddleware(GetCustomFilterHandler)))

	// API routes - Schema introspection & builder (require authentication)
	// All authenticated users can view schemas, build reports, and preview
	// Publishing is blocked server-side when AUTH_MODE=on (see PublishReportHandler)
	http.HandleFunc(basePath+"/api/schema/schemas", WithRequestID(AuthMiddleware(GetAvailableSchemasHandler)))
	http.HandleFunc(basePath+"/api/schema/tables", WithRequestID(AuthMiddleware(GetSchemaTablesHandler)))
	http.HandleFunc(basePath+"/api/schema/table/", WithRequestID(AuthMiddleware(GetTableColumnsHandler)))
	http.HandleFunc(basePath+"/api/query/validate", WithRequestID(AuthMiddleware(ValidateQueryHandler)))
	http.HandleFunc(basePath+"/api/query/preview", WithRequestID(AuthMiddleware(PreviewQueryHandler)))
	http.HandleFunc(basePath+"/api/query/preview-sql", WithRequestID(AuthMiddleware(PreviewAdvancedSQLHandler)))
	http.HandleFunc(basePath+"/api/report/generate", WithRequestID(AuthMiddleware(GenerateYAMLHandler)))
	http.HandleFunc(basePath+"/api/report/parse", WithRequestID(AuthMiddleware(ParseYAMLHandler)))
	http.HandleFunc(basePath+"/api/builder/categories", WithRequestID(AuthMiddleware(GetBuilderCategoriesHandler)))
	// Publish endpoint - blocked when AUTH_MODE=on (returns 403 in handler)
	http.HandleFunc(basePath+"/api/report/publish", WithRequestID(AuthMiddleware(PublishReportHandler)))
	http.HandleFunc(basePath+"/api/requirements-specs", WithRequestID(AuthMiddleware(RequirementsSpecHandler)))
	http.HandleFunc(basePath+"/api/requirements-specs/", WithRequestID(AuthMiddleware(RequirementsSpecPDFHandler)))

	// Feedback endpoints
	// POST /api/feedback/{reportID} — submit feedback (always requires auth)
	// GET  /api/feedback/{reportID} — list feedback for one report (newest-first, keyset paged)
	// GET  /api/feedback           — list feedback across all reports (newest-first, keyset paged)
	// When FEEDBACK_PUBLIC_READ=true, GETs are reachable without auth and
	// username/email are redacted in the response. POST remains authed.
	if feedbackPublicRead {
		http.HandleFunc(basePath+"/api/feedback/", WithRequestID(feedbackConditionalAuth(FeedbackHandler)))
		http.HandleFunc(basePath+"/api/feedback", WithRequestID(feedbackConditionalAuth(FeedbackListAllHandler)))
	} else {
		http.HandleFunc(basePath+"/api/feedback/", WithRequestID(AuthMiddleware(FeedbackHandler)))
		http.HandleFunc(basePath+"/api/feedback", WithRequestID(AuthMiddleware(FeedbackListAllHandler)))
	}

	// Chat endpoints (require authentication)
	http.HandleFunc(basePath+"/api/ask/query", WithRequestID(AuthMiddleware(ChatAskHandler)))
	http.HandleFunc(basePath+"/api/ask/execute", WithRequestID(AuthMiddleware(ChatExecuteHandler)))
	http.HandleFunc(basePath+"/api/ask/catalog", WithRequestID(AuthMiddleware(ChatCatalogHandler)))

	// Builder route - serve builder.html with base path injection
	http.HandleFunc(basePath+"/builder", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "builder.html"))
	})

	// User management route - initial HTML load stays unauthed; API calls are role-protected
	http.HandleFunc(basePath+"/user-management", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "user-management.html"))
	})

	// Requirements documents list route
	http.HandleFunc(basePath+"/requirements", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "requirements-list.html"))
	})

	// Requirements form route
	http.HandleFunc(basePath+"/requirements/form", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "requirements-spec-form.html"))
	})

	// Metrics route - serve metrics.html with base path injection
	http.HandleFunc(basePath+"/metrics", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "metrics.html"))
	})

	// Handle index.html explicitly (with and without .html extension)
	http.HandleFunc(basePath+"/index.html", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "index.html"))
	})

	// Handle builder.html explicitly
	http.HandleFunc(basePath+"/builder.html", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "builder.html"))
	})

	// Handle user-management.html explicitly
	http.HandleFunc(basePath+"/user-management.html", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "user-management.html"))
	})

	// Handle requirements-spec-form.html explicitly
	http.HandleFunc(basePath+"/requirements-spec-form.html", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "requirements-spec-form.html"))
	})

	// Handle requirements-list.html explicitly
	http.HandleFunc(basePath+"/requirements-list.html", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "requirements-list.html"))
	})

	// Handle metrics.html explicitly
	http.HandleFunc(basePath+"/metrics.html", func(w http.ResponseWriter, r *http.Request) {
		serveHTMLWithBasePath(w, filepath.Join(publicDir, "metrics.html"))
	})

	// Service worker: no-cache so browser always checks for updates
	http.HandleFunc(basePath+"/sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Service-Worker-Allowed", basePath+"/")
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, filepath.Join(publicDir, "sw.js"))
	})

	// PWA manifest: inject BASE_PATH into start_url and scope
	http.HandleFunc(basePath+"/manifest.json", func(w http.ResponseWriter, r *http.Request) {
		content, err := os.ReadFile(filepath.Join(publicDir, "manifest.json"))
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		manifest := string(content)
		// Replace relative "./" with absolute basePath + "/"
		prefix := basePath + "/"
		if prefix == "/" {
			prefix = "./"
		}
		manifest = strings.ReplaceAll(manifest, `"./"`, `"`+prefix+`"`)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(manifest))
	})

	// Serve index.html at root with base path injection, static files for other paths
	http.HandleFunc(basePath+"/", func(w http.ResponseWriter, r *http.Request) {
		// Root path serves index.html
		if r.URL.Path == basePath+"/" || r.URL.Path == basePath {
			serveHTMLWithBasePath(w, filepath.Join(publicDir, "index.html"))
			return
		}
		staticPath := strings.TrimPrefix(r.URL.Path, basePath)
		filePath := filepath.Join(publicDir, staticPath)

		// GeoJSON files: long cache (rarely change) + serve pre-compressed if available.
		// Pre-compressed .geojson.gz files are generated by running:
		//   gzip -k -9 public/assets/*.geojson
		if strings.HasSuffix(staticPath, ".geojson") {
			w.Header().Set("Cache-Control", "public, max-age=604800")
			w.Header().Set("Vary", "Accept-Encoding")
			w.Header().Set("Content-Type", "application/geo+json")
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				gzPath := filePath + ".gz"
				if _, err := os.Stat(gzPath); err == nil {
					w.Header().Set("Content-Encoding", "gzip")
					http.ServeFile(w, r, gzPath)
					return
				}
			}
			http.ServeFile(w, r, filePath)
			return
		}

		// All other static assets: short cache + revalidation.
		// The ?v= query parameter from cache-busting ensures fresh fetches on deploy.
		w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
		http.ServeFile(w, r, filePath)
	})

	// Start server - read port from PORT environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// Ensure port doesn't have a leading colon
	port = strings.TrimPrefix(port, ":")

	// Wrap default mux with global middleware
	// Order: RateLimit -> Recovery -> routes
	// Recovery is inner so it catches panics from rate-limited requests too
	// Note: Security headers are handled by nginx in production
	wrappedMux := RateLimitMiddleware(RecoveryMiddleware(http.DefaultServeMux))

	// Health check sits outside rate limiting/auth so load balancers can probe freely
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == basePath+"/health" {
			healthCheckHandler(w, r)
			return
		}
		wrappedMux.ServeHTTP(w, r)
	})

	// Configure HTTP server with timeouts
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second, // reports can be slow
		IdleTimeout:       60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		fmt.Printf("Server starting on http://localhost:%s\n", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal (SIGINT or SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Printf("Received signal %v, shutting down gracefully...", sig)

	// Create deadline for graceful shutdown (30 seconds for in-flight requests)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server - stops accepting new connections, waits for in-flight requests
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Save metrics before closing
	close(stopPeriodicSave)
	if err := SaveMetrics(); err != nil {
		log.Printf("Metrics save error: %v", err)
	} else {
		log.Println("Metrics saved to disk")
	}

	// Save cache before closing
	close(stopPeriodicCacheSave)
	if err := SaveCache(); err != nil {
		log.Printf("Cache save error: %v", err)
	} else {
		log.Println("Cache saved to disk")
	}

	// Close feedback DB
	CloseFeedbackDB()

	// Close database pool
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Database close error: %v", err)
		} else {
			log.Println("Database connections closed")
		}
	}

	// Note: Loggers are closed by deferred calls at the top of main()
	log.Println("Server stopped")
}

// healthCheckHandler pings the database and returns health status.
// No auth, no rate limiting — designed for load balancer probes.
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if DB == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"reason": "database not initialized",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := DB.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"reason": "database ping failed",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// serveHTMLWithBasePath reads an HTML file and injects BASE_PATH configuration
// and cache-busting version parameters on local CSS/JS assets.
func serveHTMLWithBasePath(w http.ResponseWriter, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	html := string(content)

	// Inject BASE_PATH and <base href> right after <head> so relative assets
	// work even on nested routes like /requirements/form.
	baseHref := basePath + "/"
	if baseHref == "/" {
		baseHref = "/"
	}
	basePathScript := fmt.Sprintf(`<head>
    <base href="%s">
    <script>window.BASE_PATH = '%s';</script>`, baseHref, basePath)
	html = strings.Replace(html, "<head>", basePathScript, 1)

	// Append cache-busting version to local CSS and JS references.
	// Matches relative paths like href="css/style.css" and src="js/app.js"
	// but not CDN URLs (which use absolute https:// paths).
	// Forces browsers to re-fetch assets after each server restart.
	html = localAssetRe.ReplaceAllString(html, `${1}?v=`+assetVersion+`"`)

	// Cache-bust the user-management ES module import on the dedicated page.
	html = strings.Replace(html, "from './js/user-management.js'", "from './js/user-management.js?v="+assetVersion+"'", 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// extractReportIDFromRequest extracts the report ID from the request URL
// Used by metrics middleware to identify which report was requested
func extractReportIDFromRequest(r *http.Request) string {
	prefix := basePath + "/api/report/"
	if strings.HasPrefix(r.URL.Path, prefix) {
		return strings.TrimPrefix(r.URL.Path, prefix)
	}
	return "unknown"
}
