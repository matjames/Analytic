package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"learningcrm/pkg/api"
	"learningcrm/pkg/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env", "../../.env", "../../../.env")
	}

	dsn := os.Getenv("LMS_DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("LMS_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("LMS_DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("LMS_DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("LMS_DB_PASSWORD")
		name := os.Getenv("LMS_DB_NAME")
		if name == "" {
			name = "learning_crm"
		}
		sslmode := os.Getenv("LMS_DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
	}

	if err := store.Init(dsn); err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	// Enterprise convergence: shared Event Bus + Audit Service.
	if err := api.InitEnterprise(); err != nil {
		log.Printf("warning: failed to initialise enterprise services: %v", err)
	}

	router := mux.NewRouter()
	api.RegisterRoutes(router)

	port := os.Getenv("LMS_PORT")
	if port == "" {
		port = "8102"
	}

	log.Printf("Learning, Community & Commercial backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}