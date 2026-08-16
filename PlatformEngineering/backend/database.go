package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func openDB(c Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec("SET search_path TO runops, public"); err != nil {
		db.Close()
		return nil, err
	}
	for _, name := range []string{"migrations/001_runops.sql"} {
		raw, readErr := migrationFS.ReadFile(name)
		if readErr != nil {
			return nil, readErr
		}
		if _, err = db.Exec(string(raw)); err != nil {
			return nil, fmt.Errorf("apply %s: %w", name, err)
		}
	}
	log.Println("RunOps database schema ready")
	return db, nil
}

func jsonString(v any) string {
	if v == nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}
