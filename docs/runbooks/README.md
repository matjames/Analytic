# StatGate Operational Recovery Runbooks

## Machine-Readable Standard Operating Procedures
Every operational runbook provides:
1. **Incident Classification**: Target trigger condition and severity mapping.
2. **Affected Microservices**: Explicit dependency graph mapping.
3. **Immediate Containment Actions**: High-priority steps to prevent blast-radius propagation.
4. **Step-by-Step Recovery Actions**: Sequential instructions executable either via operator UI or autonomous orchestrator.
5. **Validation Checks**: Automated health, probe, and query assertions.
6. **Rollback Procedures**: Fail-safe instructions in case recovery steps fail.
7. **Escalation Path**: Assigned engineering and managerial leads.
8. **Evidence Token Requirement**: Automated output recorded to Institutional Evidence Vault.
