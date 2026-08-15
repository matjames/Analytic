# StatGate Sovereign Enterprise Architecture

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Enterprise Architecture Standard  
**Security Classification:** Sovereign Official-Sensitive  
**Scope:** Institutional Cross-Application Infrastructure

---

## 1. Executive Summary

StatGate is a sovereign intelligence and institutional operational platform. Rather than maintaining isolated silos or disconnected single-purpose web tools, StatGate unifies 11 core applications under a robust, resilient, multi-tenant Enterprise Core layer.

Applications retain domain ownership and modularity, while sharing **identity, security, data governance, event distribution, declarative workflows, object storage, unified search, real-time analytics, knowledge graph linkage, notification dispatch, immutable audit logging, and design systems.**

```
                     ┌────────────────────────────────────────────────────────┐
                     │          Institutional Command Centre (Phase VIII)     │
                     └───────────────▲────────────────────────▲───────────────┘
                                     │                        │
     ┌───────────────────────────────┴────────────────────────┴───────────────────────────────┐
     │                 Institutional Data, Knowledge & Interoperability Fabric (Phase IX)     │
     │  [Universal Object Identity] [Knowledge Graph] [Data Catalogue] [Data Dictionary]       │
     └───────────────────────────────▲────────────────────────▲───────────────────────────────┘
                                     │                        │
     ┌───────────────────────────────┴────────────────────────┴───────────────────────────────┐
     │                        Enterprise Core Backbone (Phase I - VII)                        │
     │  [Durable Event Bus] [Workflow Engine] [Decision Records] [Action Centre] [AI Gateway] │
     └───────────────────────────────▲────────────────────────▲───────────────────────────────┘
                                     │                        │
     ┌───────────────────────────────┴────────────────────────┴───────────────────────────────┐
     │                  Shared Platform Library & Convergence Layer (Phase y)                 │
     │  [statgate-lib] [Strict JWT/RBAC] [Tenant Isolation] [Immutable Audit] [S3/MinIO]     │
     └───────────────────────────────▲────────────────────────▲───────────────────────────────┘
                                     │
         ┌────────────┬──────────────┼──────────────┬─────────────┬────────────┐
         │            │              │              │             │            │
      Registry       PMS            RMS        StatCollect    HelpDesk      StatChat
         │            │              │              │             │            │
    StatGovernance StatSpatial   Enterprise   Command Centre  Analytics  JupyterHub
```

---

## 2. Core Architectural Principles

### 2.1 Unified Identity & Context Propagation
- **Authoritative Provider:** StatGate Registry (`stage_register`) is the single source of cryptographic truth for user authentication and token issuance.
- **JWT Standard:** High-entropy HMAC-SHA256 tokens (`STATGATE_REGISTRY_JWT_SECRET`) encoding `userId`, `email`, `tenant_id`, `org_id`, `role`, and expiration claims.
- **Fail-Fast Zero Defaults:** In production, missing identity secrets immediately abort startup. Mock users and anonymous bypasses are prohibited.

### 2.2 Application-First Domain Modularity
Applications maintain their specialized transactional schemas and localized workflows, but communicate across domain boundaries exclusively through the Enterprise Event Bus and Object Fabric.
- **Registry:** Identities, organizational units, facilities, user credentials.
- **PMS:** Project tracking, capital allocations, milestones, contractors.
- **RMS:** Research protocols, ethics approvals, investigators, datasets.
- **StatCollect:** Field surveys, ODK sync, validation gates, enumerator submissions.
- **HelpDesk:** Service requests, incident lifecycles, SLAs, escalations.
- **StatChat:** Real-time channels, direct discussions, universal object context threads.
- **StatGovernance:** Compliance audits, risk registers, internal controls.
- **StatSpatial:** Administrative boundaries, GIS layers, spatial index, federated nodes.
- **Enterprise Core:** Shared workflows, timelines, decision records, cross-app search, knowledge graph.

### 2.3 Universal Object Identity
Every institutional entity is assigned a canonical Universal Identifier:
```text
tenant:application:object_type:object_id
```
Examples:
- `uganda-national:pms:project:proj-9821`
- `uganda-national:rms:research:res-4402`
- `uganda-national:statcollect:survey:surv-8101`
- `uganda-national:statspatial:layer:layer-district-boundaries`

### 2.4 Enterprise Communication & Event Sinks
All state transitions publish domain events to Redis channel `statgate:events` with PostgreSQL durable ingress fallback. Enterprise Core fans out events to Analytics, Data Quality, Timeline, Notifications, and AI Investigation pipelines.

---

## 3. Data & Storage Tiering

1. **Relational / Transactional:** PostgreSQL with per-application database schemas and shared connection pooling (`statgate-lib/database`).
2. **Event Broker & Caching:** Redis 7 with pub/sub clustering and dead-letter routing (`statgate:events:dlq`).
3. **Object & Document Storage:** MinIO / S3 cluster with SHA-256 integrity verification, tenant prefixes, and metadata governance (`statgate-lib/storage`).
4. **Knowledge & Interoperability:** Graph schema and Object Resolver linking relational entities across systems.

---

## 4. Governance & Resilience Assurance

- **Zero Data Loss:** Durable transaction boundaries before external event publication.
- **Idempotency:** SHA-256 deterministic payload hashing and event ID deduplication caching.
- **Automated Health Probes:** Standard `/health`, `/ready`, and `/live` endpoints with active database and broker pings.
- **Traceability:** Distributed request and correlation ID propagation (`X-Request-ID`, `X-Correlation-ID`).
