package main

import (
	"fmt"
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
		host := os.Getenv("STATCHAT_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("STATCHAT_DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("STATCHAT_DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("STATCHAT_DB_PASSWORD")
		if password == "" {
			password = "postgres"
		}
		name := os.Getenv("STATCHAT_DB_NAME")
		if name == "" {
			name = "statchat"
		}
		sslmode := os.Getenv("STATCHAT_DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
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
