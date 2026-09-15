# Architecture

StatGate is a multi-tenant evidence platform composed of independently deployable services connected through shared identity, PostgreSQL, Redis, object storage, observability, and StatChat collaboration.

## Boundaries

- Registry signs JWTs; services validate the shared registry secret and derive the authenticated user and tenant from claims.
- Each service owns its database or schema and exposes health, readiness, metrics, and authenticated resource routes.
- Workspace-scoped resources require an explicit workspace identifier and must enforce membership and tenant ownership server-side.
- Redis Streams provide durable cross-module events with consumer groups, bounded retries, idempotency, reclaim, and dead-letter handling.
- Enterprise Core provides integration services such as registration, timelines, files, workflow, permissions, and notifications.

## Main layers

The App Launcher is the entry portal. Analytics uses Flask plus the Go core. Registry, StatChat, PMS, RMS, Governance, Spatial, Citizen, and later services expose their own APIs and UIs. PostgreSQL and Redis are shared infrastructure; service data remains isolated by database/schema and tenant/workspace policy.

See [README.md](README.md), [docs/architecture/README.md](docs/architecture/README.md), and [Roadmap/PHASE_SERVICE_MATRIX.md](Roadmap/PHASE_SERVICE_MATRIX.md) for the current topology and certification state.
