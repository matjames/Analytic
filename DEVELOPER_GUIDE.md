# Developer Guide

## Start here

1. Copy `.env.example` to `.env` and use local-only secrets.
2. Start dependencies with `docker compose up -d postgres redis` or start the complete stack with `docker compose up --build`.
3. Read the service README and locate its backend, frontend, migration, and test directories.
4. Use the Registry identity and workspace fixtures for authenticated tests.

## Service checklist

- Load configuration and fail safely when required production secrets are absent.
- Expose health, readiness, and metrics endpoints.
- Validate Registry JWTs and derive user and tenant identity from claims.
- Enforce workspace membership for workspace-owned objects.
- Use shared event contracts and durable publication for cross-module events.
- Add authorization, isolation, persistence, and failure-path tests.
- Update Compose, `.env.example`, security documentation, and the roadmap when configuration changes.

## Verification commands

```powershell
go test ./...
go vet ./...
docker compose config --quiet
git diff --check
```

Use `npm ci` followed by `npm run build` for React or Next.js frontends, and the documented Python test command for Flask. Keep generated dependencies and build outputs out of the index unless a reproducible vendored build explicitly requires them.
