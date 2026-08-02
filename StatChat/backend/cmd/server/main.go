package main

import (
	"log"
	"net/http"
	"os"

	"statchat/pkg/api"
	"statchat/pkg/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		// Try common paths relative to backend/cmd/server
		_ = godotenv.Load("../.env", "../../.env", "../../../.env")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/statchat?sslmode=disable"
	}

	if err := store.Init(dsn); err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	router := mux.NewRouter()
	api.RegisterRoutes(router)

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "4000"
	}

	log.Printf("StatChat Go backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
