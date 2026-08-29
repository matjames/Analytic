# Report server — convenience targets (no Docker; local PostgreSQL required for DB tasks)
.PHONY: help run test-db test test-integration

# Default database name for local DWH-shaped test data (override: make test-db TEST_DB_NAME=mydb)
TEST_DB_NAME ?= dwh_test

help:
	@echo "Targets:"
	@echo "  make run                Stop running server, rebuild, start in background, health-check"
	@echo "                          (delegates to ./start.sh)"
	@echo "  make test-db            Create DB (if needed), run migrations + seeds under database/testdb/"
	@echo "                          Uses TEST_DB_NAME=$(TEST_DB_NAME)"
	@echo "                          Connection: PGHOST PGPORT PGUSER (defaults: localhost, 5432, current user)"
	@echo "                          Then: cp .env.localdb .env  (fill in your PG username)"
	@echo "  make test               Run the fast unit test suite (no DB required)"
	@echo "  make test-integration   Run end-to-end tests against the seeded test DB"
	@echo "                          Requires: make test-db && .env pointing at that DB"
	@echo "  make help               Show this message"

run:
	./start.sh

test-db:
	./database/testdb/setup_test_db.sh $(TEST_DB_NAME)

test:
	go test ./server

test-integration:
	CI_INTEGRATION_DB=1 go test -tags=integration ./server
