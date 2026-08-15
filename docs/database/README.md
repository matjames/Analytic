# StatGate Database Governance & Schema Architecture

## Schema Ownership & Structure
- **Analytics DB (`statgate_ml_staging`)**: Kaggle staging and ML feature pipelines.
- **HelpDesk DB (`statgate`)**: Support tickets, SLAs, and service desk interactions.
- **Registry DB (`kaggle`)**: Authoritative identity, organizations, roles, and SSO credentials.
- **StatGovernance DB (`statgovernance`)**: Risk registers, compliance assessments, meeting governance, audit programs.
- **Enterprise Platform DB (`statgate_enterprise`)**: Durable platform state managed by Enterprise Core:
  - `platform_notifications`
  - `platform_timeline`
  - `platform_audit_log`
  - `platform_events`
  - `platform_service_registry`

## Migration & Versioning Strategy
- Database initialization is managed via numbered, idempotent scripts in `docker/postgres-init/`.
- Enterprise Core automatically applies necessary schema assertions on boot via `ensurePlatformSchema` in `persistence.go`.
