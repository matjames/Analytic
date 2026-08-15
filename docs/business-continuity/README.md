# StatGate Business Continuity & Sovereign Resilience

## Institutional Reliability Targets
- **Tier 0 (Enterprise Core, Registry)**: RTO <= 60s, RPO = 0s. Zero data loss tolerance.
- **Tier 1 (StatCollect, PMS, StatGovernance)**: RTO <= 300s, RPO <= 60s.
- **Tier 2 (RMS, StatChat, HelpDesk)**: RTO <= 600s, RPO <= 300s.

## Failure Injection Strategy
Simulated drills are scheduled regularly across:
1. Redis pub/sub broker disconnection.
2. PostgreSQL database pool saturation and failover.
3. Microservice container termination and probe recovery.
4. Event consumer lag and dead-letter replay.

Evidence of every drill is cryptographically signed and archived in the Institutional Evidence Vault.
