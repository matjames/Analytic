# Contributing

## Before changing code

1. Read [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md), the relevant phase file in `Roadmap/`, and the service README.
2. Confirm the change has an explicit tenant, workspace, and authenticated-user boundary where applicable.
3. Keep secrets in `.env` or an external secret store. Never commit credentials, generated binaries, uploads, or local databases.

## Change expectations

- Prefer small, reviewable changes that preserve the standard service pattern.
- Add or update tests for authorization, workspace isolation, persistence, and failure paths.
- Keep API changes backward-compatible or document the migration.
- Use ASCII for new source and documentation unless the existing file requires otherwise.
- Update the relevant roadmap and changelog when behavior or certification status changes.

## Verification

Run the narrowest relevant checks first, then the service checks. For platform changes, also run:

```powershell
docker compose config --quiet
git diff --check
```

Go modules should pass `go test ./...` and `go vet ./...`; frontends should pass their locked install and production build. Do not report a check as passing unless it was run.

## Pull requests

Describe the user impact, security boundary, migration needs, tests run, and any external gate that remains. Include screenshots for meaningful UI changes and identify any intentionally deferred work.
