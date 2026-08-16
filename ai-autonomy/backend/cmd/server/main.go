package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"aiengines/pkg/api"
	"aiengines/pkg/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env", "../../.env", "../../../.env")
	}

	dsn := os.Getenv("AIENG_DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("AIENG_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("AIENG_DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("AIENG_DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("AIENG_DB_PASSWORD")
		name := os.Getenv("AIENG_DB_NAME")
		if name == "" {
			name = "ai_intelligence"
		}
		sslmode := os.Getenv("AIENG_DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
	}

	if err := store.Init(dsn); err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	// Enterprise convergence: shared Event Bus + Audit Service. When Redis is
	// absent, the shared in-memory broker still delivers cross-app events.
	if err := api.InitEnterprise(); err != nil {
		log.Printf("warning: failed to initialise enterprise services: %v", err)
	}

	// Mandatory system hook: listen on statgate:events to auto-trigger agent
	// tasks and simulations from other apps' domain events.
	api.StartEventConsumer(context.Background())

	router := mux.NewRouter()
	api.RegisterRoutes(router)

	port := os.Getenv("AIENG_PORT")
	if port == "" {
		port = "8101"
	}

	log.Printf("AI & Autonomy backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}