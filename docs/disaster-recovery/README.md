# StatGate Disaster Recovery & Business Continuity

## Targets
- **Recovery Point Objective (RPO)**: < 5 minutes for core domain events (backed by write-ahead logs and PostgreSQL event persistence).
- **Recovery Time Objective (RTO)**: < 15 minutes for full platform reconstitution via docker compose / Kubernetes orchestration.

## Verification & Recovery Drill
1. Take automated snapshot of PostgreSQL cluster and Redis state.
2. In a clean environment, instantiate containers with existing database volumes.
3. Validate service health via `GET /health` and `GET /ready`.
4. Trigger `POST /api/events/replay` if broker state reconstruction is needed.
5. "A backup that has never been restored is not considered verified."
