package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Default: DB stays nil — no database required (skipIfNoDb tests skip).
	// Set CI_INTEGRATION_DB=1 with a valid .env to run DB-backed tests (e.g. GitHub Actions integration job).
	if os.Getenv("CI_INTEGRATION_DB") == "1" {
		InitDB()
		if DB != nil {
			seedPublishValidationTestTable()
		}
		defer func() {
			if DB != nil {
				_ = DB.Close()
				DB = nil
			}
		}()
	}
	os.Exit(m.Run())
}

// seedPublishValidationTestTable creates report.test_table with all columns
// referenced by the publish validation test fixtures. It is dropped and
// re-created on each test run so the schema stays in sync with the fixtures.
func seedPublishValidationTestTable() {
	_, err := DB.Exec(`
		CREATE SCHEMA IF NOT EXISTS report;
		DROP TABLE IF EXISTS report.test_table;
		CREATE TABLE report.test_table (
			cases       numeric,
			tested      numeric,
			positive    numeric,
			period_year text
		);
	`)
	if err != nil {
		panic("seedPublishValidationTestTable: " + err.Error())
	}
}
