# StatGate — Learning, Community & Commercial (App 7)

Go backend for the **Learning, Community & Commercial** group of phases:

| Phase | Domain |
|---|---|
| **P24** | Learning Management System, Course Builder, Exams, CPD Certification & Digital Badges |
| **P26** | Commercial CRM, Partner Portal, Service Requests, Billing/Invoice Sync |
| **P35** | Stakeholder Engagement Platform & Governance Stewardship |

A **first-class StatGate microservice**: Registry-JWT identity
(`statgate-lib/auth`), Redis event bus (`statgate:events`), immutable audit
(`enterprise_audit_log`), dedicated schema (`learning` in the `learning_crm`
database), and `/health` `/ready` `/metrics` probes.

## Architecture

```
App Launcher (:3006) -> Learning, Community & Commercial (:8102)
   ├─ P24 LMS & CPD         courses, modules, enrollments, assessments,
   │                        attempts, certificates, badges
   ├─ P26 CRM & Commercial  leads, accounts, opportunities, partners,
   │                        service_requests, invoices (billing sync flag)
   └─ P35 Stewardship       stakeholders, engagements, stewardship_actions
   │  Events out: enrollment.created, course.completed, certificate.issued,
   │              badge.issued, lead.converted, partner.registered,
   │              service_request.created, object.link.created
   ▼
statgate:events -> Enterprise Core & other apps react
```

## Mandatory system hooks (implemented)

1. **Emits** to `statgate:events`: `course.completed`, `certificate.issued`,
   `badge.issued`, `lead.converted`, `partner.registered`,
   `service_request.created`, `enrollment.created`, `object.link.created`.
2. **Links entities in `object_links`** — completed learner→course links and
   converted lead→account links, plus a generic `/api/v1/links` endpoint.
3. **`statgate-lib/auth`** JWT enforcement (fail-closed, `X-Tenant-ID`
   isolation), `/health` `/ready` `/metrics`.

## API surface (all under `/api/v1`, Registry JWT required)

- **LMS**: `GET|POST /courses`, `GET|PUT|DELETE /courses/{id}`,
  `POST /courses/{id}/publish|enroll|complete`,
  `GET|POST /courses/{id}/modules`, `GET|POST /courses/{id}/assessments`,
  `POST /assessments/{id}/attempts`, `GET /courses/{id}/attempts`,
  `GET /enrollments`, `GET|GET /certificates[/{id}]`,
  `GET /certificates/verify/check?number=`, `GET /badges`
- **CRM**: `GET|POST /leads`, `GET|PUT /leads/{id}`,
  `POST /leads/{id}/convert`, `GET|POST /accounts[/{id}]`,
  `GET|POST|PUT|DELETE /opportunities[/{id}]`,
  `GET|POST|PUT|DELETE /partners[/{id}]`,
  `GET|POST|PUT /service-requests[/{id}]`,
  `GET|POST|PUT|DELETE /invoices[/{id}]`, `POST /billing/sync/{id}`
- **Stewardship**: `GET|POST /stakeholders`, `GET|DELETE /stakeholders/{id}`,
  `GET|POST /engagements`, `GET|POST /stewardship`, `PUT /stewardship/{id}`
- **Cross-app links**: `POST /links`, `GET /links?type=&id=`

## Integration contracts

| Contract | Mechanism |
|---|---|
| Events out | `emitEvent` → `statgate:events` canonical channel |
| Identity | `statgate-lib/auth` (Registry JWT, fail-closed, tenant isolation) |
| Audit | `statgate-lib/audit` → `enterprise_audit_log` |
| Entity linkage | shared `object_links` table |
| Observability | `/health` `/ready` `/metrics` |

## Run

```powershell
# 1. DB provisioned by docker/postgres-init/18-create-learning-db.sql (compose mount)
# 2. Configure .env (see backend/.env.example), then:
cd backend
go run ./cmd/server      # :8102 (LMS_PORT)
curl http://localhost:8102/health
```

Registered in `docker-compose.yml` as `learning-crm` (host `8102`,
`statgate-network`).

## Verify

```powershell
cd backend
go build ./...
go test  ./...
go vet   ./...
```