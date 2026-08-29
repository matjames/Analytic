// dbprovision — local dev helper: probes Postgres auth, lists databases, and
// creates any StatGate app databases that are missing. Uses credentials from
// the process environment (KAGGLE_DB_* / POSTGRES_*), never hardcoded.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var wantDBs = []string{
	"statgate_ml_staging", "statcollect", "kaggle", "statgate",
	"knowledge_portal", "ai_intelligence", "learning_crm",
	"gis_intelligence", "bpm_hub",
	"statfederation", "statiot", "statdata",
	"pms", "rms", "statchat", "statgovernance", "statspatial", "stattrust",
}

func main() {
	host := envOr("KAGGLE_DB_HOST", "localhost")
	port := envOr("KAGGLE_DB_PORT", "5432")
	user := envOr("KAGGLE_DB_USER", envOr("POSTGRES_USER", "postgres"))
	password := os.Getenv("KAGGLE_DB_PASSWORD")
	if password == "" {
		password = os.Getenv("POSTGRES_PASSWORD")
	}
	sslmode := envOr("KAGGLE_DB_SSLMODE", "disable")

	var pool *sql.DB
	var lastErr error
	// Try list of candidate superusers so local dev works whichever role is used.
	candidates := []string{user, "postgres", "Kaggle"}
	seen := map[string]bool{}
	for _, u := range candidates {
		if seen[u] {
			continue
		}
		seen[u] = true
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s", host, port, u, password, sslmode)
		conn, err := sql.Open("pgx", dsn)
		if err != nil {
			lastErr = err
			continue
		}
		if err := conn.Ping(); err != nil {
			conn.Close()
			lastErr = err
			continue
		}
		pool = conn
		fmt.Printf("[ok] connected as %s\n", u)
		break
	}
	if pool == nil {
		fmt.Printf("[fail] cannot connect to postgres: %v\n", lastErr)
		fmt.Println("Set KAGGLE_DB_PASSWORD/POSTGRES_PASSWORD or run within the network.")
		os.Exit(1)
	}
	defer pool.Close()

	rows, err := pool.Query("SELECT datname FROM pg_database")
	if err != nil {
		fmt.Println("error listing databases:", err)
		os.Exit(1)
	}
	existing := map[string]bool{}
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			existing[n] = true
		}
	}
	rows.Close()

	for _, database := range wantDBs {
		if existing[database] {
			fmt.Printf("  exists: %s\n", database)
			continue
		}
		q := fmt.Sprintf("CREATE DATABASE %q", database)
		if _, err := pool.Exec(q); err != nil {
			fmt.Printf("  create %s FAILED: %v\n", database, err)
			continue
		}
		fmt.Printf("  created: %s\n", database)
	}
}

func envOr(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}