# StatGate Sovereign Platform Architecture

## Executive Overview
StatGate operates as a sovereign institutional intelligence infrastructure. Rather than operating as isolated micro-applications, all enterprise modules (Registry, PMS, RMS, StatChat, StatCollect, HelpDesk, StatGovernance) function as organs of a unified institutional environment interconnected through the Enterprise Core.

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
  │                    Sovereign Platform Hardening & Security (Phase X)                   │
  │  [Strict JWT Verification] [Tenant Isolation] [PG State Store] [Service Registry]     │
  │  [Lock-Free Metrics]       [Dead-Letter Queue] [Fail-Fast Secrets] [Immutable Audit]   │
  └────────────────────────────────────────────────────────────────────────────────────────┘
```

## Key Architectural Principles (Phase X)
1. **Zero Unauthenticated Access**: All platform APIs require cryptographic JWT validation against `STATGATE_REGISTRY_JWT_SECRET`.
2. **Fail-Fast Configuration**: In production (`STATGATE_ENV=production`), services reject launch if mandatory secrets are missing.
3. **Durable Ingress**: Domain events are persisted to PostgreSQL before publishing to Redis, guaranteeing zero message loss across broker restarts.
4. **Tenant Isolation**: Deep header and token inspection rejects cross-tenant context injections.
5. **Observability & Probes**: Standardized lock-free metrics, `/ready`, `/live`, and `/health` endpoints enable autonomous infrastructure lifecycle management.
