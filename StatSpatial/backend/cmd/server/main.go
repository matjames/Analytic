package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"statspatial/pkg/api"
	"statspatial/pkg/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env", "../../.env", "../../../.env")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("STATSPATIAL_DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("STATSPATIAL_DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("STATSPATIAL_DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("STATSPATIAL_DB_PASSWORD")
		if password == "" {
			password = "postgres"
		}
		name := os.Getenv("STATSPATIAL_DB_NAME")
		if name == "" {
			name = "statspatial"
		}
		sslmode := os.Getenv("STATSPATIAL_DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
	}

	if err := store.Init(dsn); err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	// Enterprise convergence: attach StatSpatial to the shared Event Bus and
	// Audit Service (statgate-lib). InitFromEnv only returns an error on
	// invalid configuration; an unreachable Redis falls back to in-memory.
	if err := api.InitEnterprise(); err != nil {
		log.Printf("warning: failed to initialise enterprise services: %v", err)
	}
	api.ConfigureStatChat(os.Getenv("STATCHAT_API_URL"), os.Getenv("STATGATE_INTERNAL_API_KEY"))

	router := mux.NewRouter()
	api.RegisterRoutes(router)

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "4200"
	}

	log.Printf("StatSpatial backend running on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
