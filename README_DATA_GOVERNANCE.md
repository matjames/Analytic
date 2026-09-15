# Data Governance

StatGate treats evidence as tenant-owned data with explicit provenance, access, retention, and lifecycle controls.

## Minimum rules

- Identify the tenant and workspace for every persisted business object.
- Enforce authorization in the API and database query path; UI filtering is not a security boundary.
- Capture provenance for collected, imported, transformed, calculated, and AI-assisted data.
- Classify sensitive data before exposing it through search, exports, events, logs, or analytics.
- Apply retention and legal holds before deletion; make destructive operations auditable.
- Use pseudonymous identifiers in telemetry and avoid secrets or raw personal data in logs.
- Export only data the authenticated user is allowed to access.

Schema and operational details are in [docs/database/README.md](docs/database/README.md), [docs/security/README.md](docs/security/README.md), and [Roadmap/phase7.md](Roadmap/phase7.md).
