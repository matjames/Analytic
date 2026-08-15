# StatGate Service Security Matrix (SG-SEC-2026-08)

Machine-readable copy: docs/security/service-security-matrix.json

| Service | Port | DB | JWT | Internal API | Redis | Event channel | Health | Ready | Metrics | Uploads |
|---------|------|----|-----|--------------|-------|---------------|--------|-------|---------|---------|
| Registry (Go) | 9090 | kaggle | sign+verify | yes | no | n/a | /health | /ready | /metrics | documents |
| Core engine | 8080 | ml_staging | verify | yes | no | n/a | /health | /ready | /metrics/prometheus | n/a |
| Analytics (Flask) | 5000 | ml_staging | verify | yes | redis | n/a | /health | n/a | /metrics | n/a |
| PMS | 8091 | pms | verify | no | n/a | pulses | /health | n/a | /metrics | n/a |
| RMS | 8092 | rms | verify | no | n/a | pulses | /health | n/a | /metrics | n/a |
| Governance | 8093 | statgovernance | verify | no | yes | pulses | /health | /ready | /metrics | n/a |
| Enterprise core | 8096 | enterprise | verify | yes | yes | statgate:events | /health | /ready | /metrics | files |
| Enterprise search | 8095 | ml_staging | verify | no | no | n/a | /health | n/a | n/a | n/a |
| StatChat | 4000 | statchat | verify | yes | no | n/a | /health | /readyz | n/a | uploads |
| Helpdesk | 5006 | statgate | verify | no | no | n/a | /api-docs | n/a | n/a | uploads |
| StatCollect | 8080 | statcollect | verify | yes | yes | opt-in | /health | n/a | /metrics | data dir |

Service, ports, databases, health, readiness and metrics are defined for the
Docker Compose deployment; any change must be reflected here first.
---
## Phase XII — Enterprise Core namespaces

Enterprise Core (8096) now also exposes the Phase XII namespaces under the same JWT + tenant-isolation + rate-limit middleware:

| Namespace | Auth | Privileged actions |
|---|---|---|
| /api/intelligence/* | JWT + tenant | signal ingest (admin/institutional_lead) |
| /api/graph/objects, /edges | JWT + tenant | create/delete (admin/institutional_lead); queries all-authenticated |
| /api/objectives, /kpis | JWT + tenant | create/measure (admin/institutional_lead) |
| /api/risks/events | JWT + tenant | record/detect (admin/institutional_lead) |
| /api/ai/recommendations, /audit | JWT + tenant | review/execute (admin/institutional_lead) |
