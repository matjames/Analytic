# StatGate Backup Assurance & Restore Verification

## The Non-Negotiable Principle
> "A backup that has never been restored and verified in a sandboxed environment is not considered compliant."

## Certification Flow
1. **BACKUP CREATED**: Automated snapshot, incremental WAL, or logical dump.
2. **CHECKSUM VERIFIED**: SHA-256 digest validation against storage location.
3. **RESTORE EXECUTED**: Automated restoration into isolated sandbox.
4. **DATA INTEGRITY CHECK**: Schema assertions and record count validation.
5. **SERVICE VALIDATED**: Health probe `/ready` verification on restored instance.
6. **RECOVERY CERTIFIED**: Cryptographic evidence token generated and recorded.
