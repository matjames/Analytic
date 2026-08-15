# StatGate Sovereign Deployment & Infrastructure Topology

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Deployment Specification  
**Infrastructure Target:** Docker Compose / Sovereign Bare-Metal / Kubernetes

---

## 1. Network & Port Allocation Topology

All 11 applications and enterprise shared services operate within the unified `statgate-network` bridge network:

| Service / Container | Role | Backend Port | Frontend Port | Database / Storage |
|---|---|---|---|---|
| **Enterprise Core** | Event Bus, Workflows, Fabric, Timeline | `8080` | — | PostgreSQL (`statgate_core`) |
| **Enterprise Search** | Federated Search Index | `8090` | — | In-Memory / Core DB |
| **Command Centre** | Operational Front Door / Executive Workspace | — | `3000` | Core API Gateway |
| **Registry** | Identity Provider / User Directory | `8081` | `3006` | PostgreSQL (`statgate_registry`) |
| **StatCollect** | Survey Engine & Field Sync | `8082` | `3007` | PostgreSQL (`statgate_statcollect`) |
| **HelpDesk** | Incident & Service Management | `8083` | `3008` | PostgreSQL (`statgate_helpdesk`) |
| **StatChat** | Real-Time Collaboration & Object Threads | `8084` | `3009` | PostgreSQL (`statgate_statchat`) |
| **PMS** | Project Management System | `8091` | `3010` | PostgreSQL (`statgate_pms`) |
| **RMS** | Research Management System | `8092` | `3011` | PostgreSQL (`statgate_rms`) |
| **StatGovernance** | Compliance & Risk Registers | `8085` | `3012` | PostgreSQL (`statgate_governance`) |
| **StatSpatial** | Geospatial Boundaries & GIS Nodes | `4200` | — | PostgreSQL (`statgate_spatial`) |
| **Analytics Engine** | Statistical Processing & KPIs | `5000` | — | Core & PostgreSQL |
| **PostgreSQL** | Relational Multi-Database Engine | `5432` | — | Persistent Volume Mount |
| **Redis 7** | Event Bus & In-Memory Cache | `6379` | — | Append-Only Persistence |
| **MinIO** | S3-Compatible Object Store | `9000` (API) | `9001` (Console) | S3 Volume Mount |

---

## 2. Fail-Fast Zero-Default Environment Matrix

All production environments require the following environment variables. Services fail startup if any required secret is missing:

```bash
# Security & Auth Anchor
STATGATE_ENV=production
STATGATE_REGISTRY_JWT_SECRET=<32-char-high-entropy-secret>
STATGATE_JWT_ISSUER=statgate-registry
STATGATE_JWT_AUDIENCE=statgate

# Database Cluster
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<db-cluster-password>

# Event Broker
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=<redis-auth-token>
STATGATE_EVENT_CHANNEL=statgate:events
STATGATE_EVENT_DLQ_CHANNEL=statgate:events:dlq

# MinIO S3 Object Storage
MINIO_ENDPOINT=minio:9000
MINIO_ROOT_USER=statgate-admin
MINIO_ROOT_PASSWORD=<minio-root-secret>
MINIO_DEFAULT_BUCKET=statgate-files
```

---

## 3. Health & Liveness Probing Protocol

Every service exposes `/health` and `/ready` endpoints:
- Liveness Probe: Returns `200 OK` if the process HTTP runtime is responsive.
- Readiness Probe: Returns `200 OK` if the database and required event broker connections are established. If any component is down, returns `503 Service Unavailable`.
