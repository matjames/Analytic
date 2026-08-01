# Deployment guide

## Required production configuration

Copy `.env.example` to your deployment secret store and set every database value. Never commit `.env`.

Set `STATGATE_ENV=production` and a strong `STATGATE_INTERNAL_API_KEY`. The Go service refuses to start in production without this value; Flask forwards it on internal API calls. Keep Go on a private network or bind it to loopback, and publish Flask through an authenticated TLS reverse proxy.

Set `CORS_ALLOWED_ORIGIN` to the exact browser origin. The Go service does not accept wildcard origins.

Leave `ENABLE_NOTEBOOK_EXECUTION=false` unless access is restricted to fully trusted users. The notebook runs submitted Python in the Flask process.

## Service topology

Run Go on a private address and Flask behind the reverse proxy:

```text
Internet -> TLS reverse proxy with authentication -> Flask (:5000) -> Go (:8080)
                                                -> PostgreSQL
```

The Go core must not be exposed directly to the Internet. Request tenant/role/clearance headers are currently trusted application input; the reverse proxy or application authentication layer must establish those values before Flask processes requests.

## Start commands

```powershell
cd backend
go build -o bin/server.exe ./cmd/server
./bin/server.exe

cd ../frontend
python -m pip install -r requirements.txt
python app.py
```

Verify `GET /health` on the Go service, then check the Flask dashboard and catalog. The dataset catalog now queries PostgreSQL directly from Flask, avoiding the former Flask-to-Go-to-Flask proxy loop.

For container deployment, use `docker compose up --build` from the repository root. It requires a populated `.env`, keeps notebook execution disabled, and mounts `data/` into the Analytics UI container for durable local asset storage. It uses host ports 5001 (UI) and 8081 (core) by default; set `ANALYTICS_UI_PORT` and `ANALYTICS_CORE_PORT` to override them.

## Current persistence model

Dashboard metadata, feedback, and schema snapshots are local JSON files under `data/`. Mount durable storage for that directory, or migrate these records to a managed database before horizontal scaling. Go's database-backed asset manager is not wired into the current bootstrap and its save endpoint returns `503` until a database-backed deployment is configured.
