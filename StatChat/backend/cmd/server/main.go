package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"statchat/pkg/api"
	"statchat/pkg/store"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load service settings first, then fill any missing shared StatGate values
	// (Registry URL, shared SSO secret, service credential) from the workspace.
	_ = godotenv.Load()
	loadWorkspaceEnvironment("../.env", "../../.env", "../../../.env")

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

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Printf("StatChat Go backend running on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignal
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func loadWorkspaceEnvironment(paths ...string) {
	for _, path := range paths {
		values, err := godotenv.Read(path)
		if err != nil {
			continue
		}
		for key, value := range values {
			if strings.TrimSpace(os.Getenv(key)) == "" && strings.TrimSpace(value) != "" {
				_ = os.Setenv(key, value)
			}
		}
	}
}
