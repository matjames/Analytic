# Database Guidelines

- PostgreSQL is the default transactional store; Redis is for cache, coordination, and durable events, not the only copy of business records.
- Every service owns its migrations and uses idempotent migration steps.
- Add tenant and workspace scope to every multi-tenant business table and index common authorization predicates.
- Use parameterized queries and transactions for related writes.
- Define foreign keys, uniqueness, check constraints, timestamps, and deletion behavior explicitly.
- Avoid storing secrets or unnecessary personal data; hash or pseudonymize values when possible.
- Design indexes from real access paths and review query plans for large collections.
- Backups, restore drills, retention, legal holds, and migration rollback plans are part of feature completion.

See [docs/database/README.md](docs/database/README.md) and the bootstrap scripts under `docker/postgres-init/`.
