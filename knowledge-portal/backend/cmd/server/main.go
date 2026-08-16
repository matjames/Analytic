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

	port := os.Getenv("KNOWLEDGE_PORT")
	if port == "" {
		port = "8099"
	}

	log.Printf("Knowledge Portal backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
