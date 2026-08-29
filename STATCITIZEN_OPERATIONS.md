# StatCitizen — Operations & Deployment Runbook

## 1. Local Development Execution

### Running StatCitizen Backend
```powershell
cd StatCitizen\backend
go run .
```

### Running Automated Test Suite
```powershell
cd StatCitizen\backend
go test -v ./...
```

---

## 2. Docker & Container Deployment

StatCitizen is configured in `docker-compose.yml`:
```yaml
statcitizen-api:
  build:
    context: ./StatCitizen
  ports:
    - "8097:8097"
  environment:
    - STATGATE_ENV=production
    - STATCITIZEN_DB_HOST=postgres
    - STATCITIZEN_DB_PASSWORD=${STATCITIZEN_DB_PASSWORD}
    - REDIS_HOST=redis
```

To build and start with Docker Compose:
```bash
docker compose up -d statcitizen-api
```

---

## 3. Database Migrations

Database migrations execute automatically upon server start via `runMigrations()` in `persistence.go`.  
Migration history is tracked in table `statcitizen_migrations`.

---

## 4. Health & Monitoring

- **Health Check:** `http://localhost:8097/health`
- **Readiness Check:** `http://localhost:8097/ready`
- **Prometheus Metrics:** `http://localhost:8097/metrics`
