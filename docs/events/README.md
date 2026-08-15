# StatGate Event Bus & Reliability Architecture

## Durable Event Lifecycle
1. **Ingress**: When `publishEvent` is called, the domain event is assigned a unique ID, timestamp, and tenant context.
2. **Durable Persistence**: The event is synchronously committed to the `platform_events` table in PostgreSQL.
3. **Broker Distribution**: The event is published to Redis pub/sub (`statgate:events`).
4. **Idempotent Consumption**: Subscribing nodes process events using deduplication keys (`statgate:idempotency:<id>`).
5. **Dead-Letter Handling**: Any failed consumer executions record the failure in both Redis (`statgate:events:dead-letter`) and PostgreSQL (`platform_events.status = 'failed'`).

## Event Replay & Remediation
- Administrators can replay events using `POST /api/events/replay` with temporal and event-type filters.
- Dead-letter entries can be inspected (`GET /api/events/dead-letter`) and retried (`POST /api/events/dead-letter/:id/retry`).
