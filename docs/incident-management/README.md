# StatGate Autonomous Incident Lifecycle & Management

## 8-Stage Formal State Machine
Every incident follows an audited, timestamped 8-stage transition flow:

```
  DETECTED ──► TRIAGED ──► ACKNOWLEDGED ──► MITIGATING
                                               │
                                               ▼
  CLOSED  ◄── RESOLVED ◄──  VALIDATING  ◄── RECOVERING
```

## Autonomous Incident Detection Rules
- **Probe Failure**: Consecutive failures on `/live` or `/ready` triggers automated `CRITICAL` incident.
- **DLQ Backlog**: Dead-letter count > 50 triggers `WARNING`, > 200 triggers `HIGH`.
- **RTO/RPO Breaches**: Measured recovery latency exceeding defined service tier SLA triggers `CRITICAL`.
- **DB Connection Pool Saturation**: Active connections >= 90% capacity triggers `HIGH`.

## Audit & Non-Repudiation
Every state transition logs an `incident_events` record with:
- Actor / System agent identity
- Previous and new status
- Transition reason and attached verification evidence
- Transition timestamp and correlation ID
