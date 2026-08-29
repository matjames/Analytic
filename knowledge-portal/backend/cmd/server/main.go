package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"knowledgeportal/pkg/api"
	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env", "../../.env", "../../../.env")
	}

	dsn := os.Getenv("KNOWLEDGE_DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("KNOWLEDGE_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("KNOWLEDGE_DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("KNOWLEDGE_DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("KNOWLEDGE_DB_PASSWORD")
		name := os.Getenv("KNOWLEDGE_DB_NAME")
		if name == "" {
			name = "knowledge_portal"
		}
		sslmode := os.Getenv("KNOWLEDGE_DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
	}

	if err := store.Init(dsn); err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	// Enterprise convergence: attach this app to the shared Event Bus and
	// Audit Service (statgate-lib). An unreachable Redis falls back to
	// in-memory so local development still works.
	if err := api.InitEnterprise(); err != nil {
		log.Printf("warning: failed to initialise enterprise services: %v", err)
	}

	router := mux.NewRouter()
	api.RegisterRoutes(router)

	// Serve the public Open Data Portal UI (no-build HTML/JS) from the frontend
	// directory. This keeps the portal self-contained within the Go binary with
	// no separate UI container. Falls back gracefully when the dir is absent.
	servePublicPortal(router)

	port := os.Getenv("KNOWLEDGE_PORT")
	if port == "" {
		port = "8099"
	}

	log.Printf("Knowledge Portal backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// servePublicPortal mounts the no-build Open Data Portal at the service root.
// The frontend location is resolved left-to-right:
//
//  1. $KNOWLEDGE_FRONTEND_DIR (explicit override)
//  2. ./frontend (default working directory / Docker context)
//  3. /frontend (Docker image copy path)
func servePublicPortal(r *mux.Router) {
	candidates := []string{
		os.Getenv("KNOWLEDGE_FRONTEND_DIR"),
		"frontend",
		"../../frontend",
		"../../../frontend",
		"/frontend",
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			continue
		}
		fs := http.FileServer(http.Dir(dir))
		r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
		// Mount the single-page portal at the root.
		r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/" {
				// Serve any real file directly; index.html otherwise.
				target := dir + req.URL.Path
				if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
					http.ServeFile(w, req, target)
					return
				}
			}
			http.ServeFile(w, req, dir+"/index.html")
		})
		log.Printf("[Knowledge Portal] serving public portal UI from %s", dir)
		return
	}
	log.Printf("[Knowledge Portal] public portal frontend not found; API-only mode")
}
