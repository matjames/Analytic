package configs

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func ConnectDB() {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("REGISTRY_DB_HOST"),
		os.Getenv("REGISTRY_DB_PORT"),
		os.Getenv("REGISTRY_DB_USER"),
		os.Getenv("REGISTRY_DB_PASSWORD"),
		os.Getenv("REGISTRY_DB_NAME"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	DB = db
	fmt.Println("✅ Database connected")
}
