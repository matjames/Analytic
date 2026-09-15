package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"bpmhub/pkg/api"
	"bpmhub/pkg/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env", "../../.env", "../../../.env")
	}

	dsn := os.Getenv("BPM_DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("BPM_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("BPM_DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("BPM_DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("BPM_DB_PASSWORD")
		name := os.Getenv("BPM_DB_NAME")
		if name == "" {
			name = "bpm_hub"
		}
		sslmode := os.Getenv("BPM_DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
	}

	if err := store.Init(dsn); err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	if err := api.InitEnterprise(); err != nil {
		log.Printf("warning: failed to initialise enterprise services: %v", err)
	}
	api.ConfigureStatChat(os.Getenv("STATCHAT_API_URL"), os.Getenv("STATGATE_INTERNAL_API_KEY"))

	router := mux.NewRouter()
	api.RegisterRoutes(router)

	port := os.Getenv("BPM_PORT")
	if port == "" {
		port = "8104"
	}

	log.Printf("BPM / Workflow backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
