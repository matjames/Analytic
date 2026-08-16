# StatGate — Public Knowledge Portal (App 2)

Go backend for the **Public Knowledge & Open Data** group of phases:

| Phase | Domain |
|---|---|
| **P17** | Public Portals, CMS, Publications & Knowledge Dissemination |
| **P40** | National Digital Library, Knowledge Repository & Preservation |
| **P41** | Open Data Platform, Public Evidence Portal & Transparency |

It is a **first-class StatGate microservice**: Registry-JWT identity, Redis event
bus (`statgate:events`), immutable audit (`enterprise_audit_log`), dedicated
PostgreSQL schema (`knowledge` in the `knowledge_portal` database), and
`/health` `/ready` probes — the standard converged app pattern.

## Architecture

```
Browser
  -> App Launcher (:3006, tile "Public Knowledge Portal")
       `-> Knowledge Portal API (Go :8099)
             -> PostgreSQL  (knowledge_portal DB, knowledge schema)
             -> Redis      (publish domain events on statgate:events)
             -> Audit      (immutable enterprise_audit_log)
             -> object_links (cross-app linkage with PMS/RMS/StatCollect/etc.)
```

## Modules & data model

- **CMS content** — `content_items` (articles, news, blogs, publications, policies; draft → in_review → published workflow).
- **Open data** — `public_datasets` (machine-readable catalogue: license, format, version, source object linkage).
- **Digital library / repository** — `repository_items` (books, theses, journals, working papers, protocols; DOI/ISBN/ISSN).
- **Persistent identifiers** — `persistent_identifiers` (DOI / ORCID / ISBN / ISSN / handle registry).
- **Engagement** — `subscriptions` (newsletter) and `feedback` (feedback, FOI requests, data requests, citizen reports).
- **Cross-app linkage** — `object_links` mirroring the shared platform contract.

## API surface

### Public (unauthenticated, `GET /api/public/...`)
- `GET /api/public/search?q=...` — cross-catalog search (content + datasets + repository)
- `GET /api/public/content`, `/content/{id}`, `/datasets`, `/datasets/{id}`, `/repository`, `/repository/{id}`
- `POST /api/public/subscriptions`, `POST /api/public/feedback`

### Admin / CMS (Registry JWT required, `X-Tenant-ID` isolation)
- `POST|GET|PUT|DELETE /api/v1/content[/{id}]`, `POST /api/v1/content/{id}/publish`
- `POST|GET|PUT|DELETE /api/v1/datasets[/{id}]`, `POST /api/v1/datasets/{id}/publish`
- `POST|GET|PUT|DELETE /api/v1/repository[/{id}]`
- `POST /api/v1/identifiers`, `POST|GET /api/v1/links`
- `GET /api/v1/subscriptions`, `/api/v1/feedback`, `/api/v1/summary`

## Communication contracts (how this app interconnects)

1. **Events (async).** `content.published`, `dataset.published`,
   `repository.item.archived`, `subscription.created`, `feedback.received`,
   `object.link.created` — published via `statgate-lib/events` to
   `statgate:events` (standard schema), so any other app can react.
2. **Object links.** Every public artifact can be linked to source objects
   (e.g. a PMS project, RMS dataset, StatCollect submission) via the shared
   `object_links` contract used across the platform.
3. **Identity.** Registry-issued JWT validated by `statgate-lib/auth`
   (fail-closed zero-default); tenant isolation via `X-Tenant-ID` vs token.
4. **Audit.** Every write emits an immutable record to
   `enterprise_audit_log` via `statgate-lib/audit`.
5. **Observability.** `/health`, `/ready`, `/live` probes for the platform
   monitoring stack.

## Run

```powershell
# 1. Provision the database (docker-compose mounts this file automatically):
#    docker/postgres-init/16-create-knowledge-db.sql
# 2. Configure .env (see backend/.env.example), then:
cd backend
go run ./cmd/server      # listens on :8099 (KNOWLEDGE_PORT)
curl http://localhost:8099/health
```

The service is registered in `docker-compose.yml` as `knowledge-portal`
(host port `8099`, joins `statgate-network`), with its database role provisioned
by `docker/postgres-init/16-create-knowledge-db.sql`.

## Verify

```powershell
cd backend
go build ./...
go test  ./...
go vet   ./...
```