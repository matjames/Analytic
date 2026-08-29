# StatGate Deployment Readiness Plan

## Purpose

This plan converts the 50-phase product roadmap into a deployment-focused execution sequence. Roadmap completion percentages describe feature breadth; deployment readiness is determined by repeatable evidence from builds, tests, security checks, runtime health, recovery exercises, and operational acceptance.

The first supported deployment target is the root Docker Compose stack on a single controlled host. Kubernetes, multi-cloud, national, and global rollout targets remain later milestones and must not block certification of this first deployable release.

## Definition of ready

A release is ready for production only when all mandatory gates below have recorded evidence, an owner, a date, and a reproducible command or report. A documentation claim alone is not evidence.

## Gate 0: Scope and ownership

- [ ] Classify every Compose service as required, optional, experimental, or retired.
- [ ] Define the minimum production profile and supported optional profiles.
- [ ] Assign an owner and operational runbook to every required service.
- [ ] Freeze the release scope; new roadmap features do not enter the release until Gate 6.

Exit criterion: one approved service inventory and release scope.

## Gate 1: Engineering foundation (Roadmap Phase 1)

- [ ] All Go modules build and test in CI.
- [ ] The Flask test suite passes in CI.
- [ ] All production frontend packages build in CI.
- [ ] Docker Compose configuration validates with documented environment variables.
- [ ] Secret scanning passes and no credentials or compiled binaries are tracked.
- [ ] Documentation matches actual ports, routes, and configuration.

Exit criterion: the default branch is green from a clean checkout.

## Gate 2: Reproducible platform startup

- [ ] A clean host can start the minimum profile using documented commands.
- [ ] Every required service reaches healthy and ready states.
- [ ] Database initialization is idempotent and versioned.
- [ ] Redis, PostgreSQL, MinIO, and service dependencies reconnect after restart.
- [ ] Smoke tests cover login, tenant context, one create/read/update workflow, search, events, and analytics.

Exit criterion: automated startup and smoke test pass twice from clean state.

## Gate 3: Security and tenant isolation (Roadmap Phases 19 and 21)

- [ ] Authentication fails closed on all protected endpoints.
- [ ] Cross-tenant access tests cover every service and persistence layer.
- [ ] Default credentials, public database ports, debug modes, and unsafe CORS are absent from the production profile.
- [ ] Images and dependencies pass vulnerability and license policy checks.
- [ ] TLS termination, secret rotation, audit retention, and least-privilege database roles are verified.
- [ ] A threat model and security sign-off are recorded.

Exit criterion: no unresolved critical/high security finding and tenant-isolation suite passes.

## Gate 4: Data durability and recovery (Roadmap Phase 20)

- [ ] PostgreSQL, object storage, uploads, and required Redis data have explicit durability policies.
- [ ] Automated encrypted backups run with retention and off-host copies.
- [ ] Schema migration and rollback procedures are tested.
- [ ] Restore is exercised into a clean environment and data integrity is verified.
- [ ] Approved RPO and RTO targets are measured, not estimated.

Exit criterion: successful documented backup-and-restore exercise.

## Gate 5: Operability and performance (Roadmap Phases 20 and 49)

- [ ] Metrics, logs, traces or request IDs, dashboards, and actionable alerts cover required services.
- [ ] Resource requests/limits and capacity assumptions are documented.
- [ ] Load tests establish supported concurrency, throughput, and data volume.
- [ ] Graceful shutdown, dependency loss, disk pressure, and restart behavior are tested.
- [ ] Incident, rollback, maintenance, and escalation runbooks are exercised.

Exit criterion: performance targets pass and an operational game day completes.

## Gate 6: Staging certification and release (Roadmap Phase 34)

- [ ] Staging mirrors the production profile and configuration model.
- [ ] End-to-end, accessibility, browser, and user-acceptance tests pass.
- [ ] Release artifacts are immutable, versioned, signed, and traceable to a commit.
- [ ] Deployment and rollback are automated and rehearsed.
- [ ] Known limitations, support ownership, training, and release notes are approved.

Exit criterion: signed go-live decision with a tested rollback point.

## Immediate sprint: Foundation recovery

1. Repair CI job dependencies and Compose validation inputs.
2. Correct service credential mappings and reconcile `.env.example` with Compose.
3. Run all Go module builds/tests and address failures by service.
4. Restore a clean Python 3.12 test environment and run the Flask suite.
5. Add frontend production builds to CI.
6. Define Compose profiles and the minimum deployable service inventory.
7. Start the minimum profile, record health results, and add an automated smoke test.

## Progress reporting

`PROJECT_PROGRESS.md` should report these gates separately from feature-phase completion. Each gate uses only `Not started`, `In progress`, `Blocked`, or `Passed`, and links to its latest evidence. Production readiness must never be inferred from an average feature-completion percentage.
