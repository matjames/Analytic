# StatGate Current State and Way Forward

**As of:** 29 August 2026  
**Purpose:** Establish a practical engineering baseline from the roadmap and the implementation currently present in the repository.

## Executive Position

StatGate has moved beyond the initial platform concept. The repository contains a working foundation, several usable business modules, shared identity and event-bus patterns, multiple frontends, and a growing set of later-phase services.

The platform is not yet at the point where the entire roadmap can be called complete. The main constraint is no longer the absence of code. It is integration, operational proof, security hardening, and completion of the business workflows that sit above the core CRUD foundations.

The immediate objective should be to make the existing platform dependable as one product before adding more roadmap domains.

## Current State by Roadmap Area

### Foundation and Identity: Phases 1-2

The repository has the expected platform spine:

- Go services use a shared service pattern.
- PostgreSQL, Redis, monitoring, and Compose infrastructure are present.
- Registry authentication and JWT validation are used across services.
- App Launcher, analytics UI, Registry, PMS, RMS, StatChat, Governance, and monitoring components are present.
- Audit, ABAC, notifications, facility, organization-unit, and administrative-boundary capabilities exist.

The foundation is not fully closed. The remaining concerns are whole-stack startup and health certification, production SMTP provider certification, OIDC provider certification, Redis verification for every dependent service, CI proof, and completion of advanced administration workflows. Refresh-token issuance/validation with server-side rotation/revocation, self-service MFA enrollment with one-time recovery codes, invitation acceptance, email verification with single-use expiring tokens, SMTP-backed verification and invitation email delivery with fake-SMTP tests, optional OIDC authorization-code login with fake-provider redirect/state/token/userinfo tests, Redis-backed rate limiting for sensitive Registry endpoints, session/device visibility with privacy-preserving metadata, country/region login hints plus cross-country anomaly flags, tenant-scoped organization branding, authenticated-identity-derived organization enforcement for user and invitation administration, handler-level Registry directory authorization tests, a canonical Enterprise Core role/permission catalog aligned with Registry JWT roles, personal user preferences, administrator-only user management, roles/policies administration, and an administrator-facing tenant-scoped audit log viewer are now implemented across Registry and Enterprise Core; they still need full deployment-path verification.

Enterprise Core now has tenant-scoped workspace persistence, membership APIs, membership administration UI, and membership validation for `X-Workspace-ID`. The shared Registry shell provides workspace selection for authenticated roles and carries the selected context on Registry requests. Registry admin/initiator headers, App Launcher, PMS, RMS, StatGovernance, and StatChat now consume tenant branding from Registry instead of relying only on static shell identity. PMS, RMS, and StatChat frontends now carry the same context on all frontend API calls, and their backend auth boundaries validate workspace ID shape and verify selected-workspace membership through Enterprise Core. PMS projects and RMS research projects now persist and enforce selected workspace scope on root list/read/create/update/delete/stage operations, with child routes, JSON body references, and direct child `PUT`/`DELETE` IDs guarded by root workspace ownership. PMS/RMS primary dashboards and search paths now filter through workspace-scoped roots. StatGovernance validates selected-workspace membership and its frontend propagates the workspace header. Analytics Core now stores and filters telemetry, anomaly, persistent analytical assets, platform projects, reports, research studies, object links, workflow instances, command-centre aggregates, and dataset metadata by tenant plus workspace; authenticated legacy Analytics routes now derive tenant scope from the verified identity instead of caller-supplied query/body values; legacy Analytics dataset refresh now derives tenant/workspace scope and writes scoped metadata plus dataset-imported events; the Python analytics UI uses workspace-specific dashboard storage and forwards the same context, and its runtime now passes compile, Flask smoke, and pytest certification; the App Launcher forwards the selected workspace during module handoff; StatOps pipeline/deployment APIs now carry and filter selected workspace context, and its frontend production build is certified; StatSpatial validates and propagates workspace context through its authenticated API and frontend, its mutable spatial/federation/link store records now carry tenant plus workspace ownership, and its live PostgreSQL ownership migration plus scoped-store behavior are certified; StatTrust validates workspace context, propagates it from the frontend, derives incident/ledger event tenant IDs from authenticated context, scopes in-memory incidents, ledger blocks, and provenance records by tenant plus workspace, and its scoped ledger/provenance PostgreSQL persistence plus startup reload are certified; and StatFederation now validates workspace context, propagates it through federation-node boundaries, applies workspace-aware PostgreSQL storage predicates across all migrated federation domains, and has live PostgreSQL migration/object-link isolation certification. Later-service workspace selection remains a release gate, so this is not yet an operational cross-module completion.

### Collaboration: Phase 3

StatChat has a substantial working core:

- Direct and group messaging
- File sharing and persistent uploads
- Presence and typing indicators
- Favourites, mute, search, read receipts, reactions, and notifications
- Basic voice and video sessions
- Registry directory synchronization
- Authenticated WebSocket communication
- Tenant-scoped object conversations

PMS and RMS now consume the shared object-conversation API using canonical project and research references, exact deep links, module-to-StatChat write/read behavior, and trusted second-user enrollment. Enterprise workflow discussion creation now uses the supported object-conversation contract and is HTTP contract-tested. The repeatable frontend route audit verifies 155 typed operations against 167 backend routes, with only infrastructure, legacy aliases, upload serving, and duplicate task aliases outside the typed client. Channel and task workflows are implemented and build/test-covered. Poll creation/listing/voting, full calendar CRUD, grounded conversation assistance, call participant counts and host roles, host termination, and host-only WebM recording upload/listing are verified against the live PostgreSQL deployment. Conversation-member administration now has persisted ownership, tenant-scoped Registry selection, owner/canonical-admin authorization, direct-chat immutability, owner protection, a permission-aware UI, automated coverage, and live API proof. Message forwarding now enforces source and destination access, preserves visible provenance, transactionally copies durable attachment and structured location references, uses the authenticated forwarder identity, emits realtime delivery and notifications, and is live read-back verified. Scheduled messages now persist UTC delivery state across restarts, support sender-only list/cancel controls, recheck authorization at dispatch, use multi-instance-safe row locking, and publish through realtime and notification paths. A searchable built-in animated reaction/sticker catalog now provides keyless, tracking-free media for every authenticated user with validated IDs, atomic persistence, realtime delivery, and forwarding. Privacy-conscious location sharing now requires explicit browser permission and confirmation, validates and atomically persists structured coordinates, enforces conversation access, avoids passive map-provider contact, and is live send/read/forward verified. Stable-ID mentions and manager-only `@all` now persist structured targets, enforce conversation membership, route notifications through mute and user preferences, protect previews, update connected recipients in realtime, and prevent cross-user notification mutation. Private saved messages and tenant-safe advanced search now support access-revocation filtering, exact source navigation, and conversation/sender/date/attachment/saved-only filters with repeatable live certification. TURN is running in Compose with fail-closed credentials and initialized relay ports. Remaining Phase 3 gates are later-module/event-bus adoption, browser E2E, and real multi-browser NAT, recovery, and production TURN public-address/TLS certification.

The subsequent retention/export/legal-hold completion advances the current StatChat audit to 110 typed operations against 122 backend routes. This capability is operational with authenticated member JSON/CSV exports, administrator-only deleted-record exports and controls, disabled-by-default 1-3650 day tenant policies, hourly/manual hard deletion, tenant-wide and conversation legal-hold overrides, immutable audit records, and repeatable live PostgreSQL certification. This 110/122 figure supersedes the 102/114 checkpoint above. Browser UI E2E remains outstanding because this session has no browser binding.

Screen sharing is now implemented for every active voice/video participant using explicit `getDisplayMedia` permission, peer video-track replacement/addition, late-join presenter state, browser-initiated stop recovery, camera restoration, presentation-focused rendering, mobile-safe controls, and presentation-aware host recording. The supporting call boundary was hardened at the same time: sessions persist tenant ownership; conversation membership controls creation/discovery/join/read; JWT identity overrides spoofed participant details; leave cannot target another user; and WebSocket signaling requires active participants, active targets, and an SDP/ICE/presenter whitelist. The repeatable live authorization certification passes all nine scenarios, while real two-browser screen-media proof remains blocked because this session exposes no browser binding.

Call recovery and quality telemetry are now implemented. Signaling reconnects with capped exponential backoff, browser online/offline transitions trigger recovery, peer disconnects receive a grace period and bounded ICE restart before eviction, and known participants are re-registered and reoffered after signaling recovery. Every caller sees accessible connection/quality state backed by WebRTC RTT, jitter, packet-loss, and bitrate sampling. Active-participant-only writes, authorized reads, server-derived quality, and 30-day indexed persistence are live-certified; the current route audit is 112 typed operations against 124 backend routes. Real multi-browser interruption, external NAT, and production TURN public-address/TLS proof remain release gates.

The call slice has advanced again: hosts can assign moderator roles, moderators can remove ordinary participants only, hosts and moderators can send explicit mute requests that recipients must accept, and all role/removal changes persist and broadcast to peers with host protection and post-removal telemetry denial. The live authorization certification passes all fourteen assertions, and the live WebSocket mute certification passes ordinary-participant denial, moderator request delivery, and recipient-consent delivery. The route audit is now 114 typed operations against 126 backend routes. Production TURN/TLS setup is documented in `StatChat/docs/CALL_DEPLOYMENT.md`; real multi-browser screen capture, interruption, and external NAT proof remain release gates because no browser binding is available in this session.

The existing collaboration feed is now tenant-safe at its core: posts and comments use authenticated identity rather than request-supplied profile fields, feeds and comments are tenant-scoped, and likes, shares, and comments reject cross-tenant targets. The social graph now uses explicit pending/accepted/declined requests, recipient-only response controls, reciprocal accepted edges, idempotent duplicate handling, reciprocal removal, and self/cross-tenant isolation, with live certification. Wellness now provides tenant-scoped authenticated publishing, default seeded content, durable comments, per-user idempotent likes/bookmarks, durable shares, and live certification. Photo/video/article posts now provide authenticated typed publishing, article title validation, URL-backed media rendering, same-tenant visibility, foreign-tenant isolation, and live certification. Communities and forums now support tenant-scoped public/private communities, creator ownership, owner-managed invitations, public join/leave, member-only topic/reply workflows, authenticated authorship, private visibility, owner/member moderation boundaries, ownership transfer, frontend workspace access, typed client coverage, and live certification. Knowledge Hub/wiki workflows now support tenant-scoped posts, authenticated authorship, expanded article content, per-user idempotent expert follows and idea upvotes, frontend workflows, and live certification. Collaborative documents now support tenant-scoped shared pages, authenticated ownership, editor/viewer membership, optimistic conflict protection, revision history, frontend editing, and live certification. Whiteboards now support tenant-scoped canvas boards, authenticated ownership, editor/viewer membership, persisted stroke data, optimistic conflict protection, revision history, frontend canvas editing, and live certification. Translation now supports authenticated language discovery, validated source/target translation, same-language identity handling, frontend workspace access, and live certification. Remaining Phase 3 gates are platform event-bus and later-module adoption, browser E2E, dedicated media-provider/upload UX, and production NAT/TURN validation.

### Business Modules: Phases 4-5

PMS and RMS contain broad operational foundations. Projects, programmes, portfolios, research projects, proposals, grants, ethics records, publications, datasets, documents, tasks, meetings, risks, dashboards, timelines, audit trails, and StatChat integrations are represented.

The major gap is depth of workflow. LogFrames, Theory of Change, donor management, critical-path and resource planning, DOI and citation workflows, formal IRB administration, journal submission, open science, and module-specific AI assistants remain future work.

### Statistics, Data, and Analytics: Phases 6-8

StatCollect, Analytics Core, Analytics UI, Enterprise Core, and Enterprise Search provide a meaningful statistics and analytics base:

- ODK/OpenRosa submission handling
- Submission validation and lifecycle events
- Object links to platform entities
- Dataset catalog and schema health checks
- Indicator registry
- Anomaly detection
- Dashboards, KPIs, reports, exports, alerts, and command-centre views
- Data catalog, lineage, quality, backups, resilience, and knowledge-graph foundations
- Keyword enterprise search

The platform still lacks the higher-order statistical and analytical layer: questionnaire and sampling designers, enumeration and supervisor workflows, census management, tabulation, SDMX metadata, open-data dissemination, forecasting, ad-hoc query building, NLQ, conversational analytics, scheduled reports, semantic search, classification, retention enforcement, OCR, automated metadata, and ontology management.

### GIS, Field Operations, and Governance: Phases 10-13

StatSpatial and StatIoT code is present, including spatial UI work, device and sensor management, telemetry, field workers, visits, mobile devices, offline synchronization, GPS breadcrumbs, forms, geofences, and conflict handling.

StatGovernance and Enterprise Core also contain substantial governance, risk, controls, audit, committee, evidence, workflow, SLA, and governed-AI foundations.

These areas should be treated as implementation in progress until their database dependencies, security boundaries, integrations, and end-to-end workflows are certified. The remaining roadmap gaps include PostGIS-backed spatial analysis, geocoding, vector tiles, supervisor operations, location streaming, fraud detection, LogFrame and evaluation workflows, whistleblower intake, conflict-of-interest management, board packs, and centralized configuration.

### Later Roadmap Phases: 14-50

The repository contains prototypes or foundations associated with several later domains, including data engineering, federation, IoT, trust, operations, workflow, low-code tooling, geospatial intelligence, learning, knowledge, and AI autonomy.

These should not yet be described as completed roadmap phases. The later roadmap documents describe target capabilities and use completion marks inconsistently. They do not establish that the services are integrated, production-ready, security-reviewed, or validated against their acceptance criteria.

## Reality-Based Status Model

For future planning, every roadmap capability should use one of these states:

1. **Operational:** implemented, integrated, tested, and usable through the supported deployment path.
2. **Implemented:** code exists and local or unit-level behavior is present, but integration or production proof is incomplete.
3. **In progress:** a meaningful foundation exists, but core workflow behavior is still being built.
4. **Specified:** described in the roadmap but not represented by a complete implementation.
5. **Deferred:** intentionally postponed until a dependency or platform gate is complete.

This avoids treating a directory, API stub, or prototype as an operational product capability.

## Critical Gaps Before Expansion

The following gaps affect multiple phases and should be addressed before declaring later phases complete:

- Full Compose startup and health verification for the complete service inventory
- One repeatable CI pipeline covering Go, Python, frontend builds, migrations, and integration tests
- Consistent Redis authentication and event-bus behavior across all dependent services
- Centralized tenant isolation and authorization enforcement
- End-to-end Registry JWT validation and authorization tests for every service
- Shared object linking and event-bus adoption across modules
- Database migration ordering and clean-environment bootstrap verification
- Persistent storage, backup, restore, and disaster-recovery validation
- Standard health, readiness, metrics, logging, and tracing behavior
- Frontend-to-backend completion for every capability marked operational
- Clear ownership and acceptance tests for each roadmap phase

## Way Forward

### Stage 1: Establish the Release Baseline

Freeze the current scope and define a supported platform slice consisting of Registry, App Launcher, Analytics, StatChat, PMS, RMS, Governance, PostgreSQL, Redis, and monitoring.

Complete the following before adding major new domains:

- Start the supported Compose stack from a clean environment.
- Verify health and readiness endpoints for every service.
- Verify Redis authentication and event delivery.
- Run all Go, Python, frontend, migration, and integration tests in CI.
- Document required environment variables, secrets, ports, volumes, and startup order.

### Stage 2: Close Identity and Security Foundations

- Redis-backed Registry rate limiting is now implemented and certified against live Redis for sensitive endpoints; continue extending the same pattern to any new public authentication surfaces.
- SMTP-backed Registry verification and invitation delivery are implemented and fake-SMTP certified; configure the production SMTP provider and run a provider smoke test before release.
- Optional Registry OIDC authorization-code login is implemented and fake-provider redirect/state/token/userinfo behavior is certified; configure the external identity provider and run callback/userinfo/claim-mapping smoke tests before release.
- Registry session/device intelligence now records privacy-preserving country/region hints and flags cross-country anomalies; production reverse proxies should pass trusted geo headers if this feature is required operationally.
- Registry user and invitation administration now derives tenant-admin organization context from the authenticated JWT instead of caller-supplied identifiers; platform admins retain explicit cross-organization administration.
- Registry directory and tenant-scope authorization tests now cover the user/invitation administration boundary; Enterprise Core workspace manager role-matrix tests cover workspace owner/admin/platform-admin decisions; and Enterprise Core file records now carry tenant ownership with tests for tenant matching, cross-tenant denial, platform override, and tenantless legacy denial. Extend the same test pattern to messages, object links, and every supported service.
- Enterprise Core now exposes a canonical platform role and permission catalog aligned with Registry JWT roles, with tests proving catalog/JWT consistency and default permission behavior.
- Tenant branding is now consumed by Registry, App Launcher, PMS, RMS, StatGovernance, and StatChat shells; additional later-phase services still need adoption as they enter the supported deployment slice.
- Propagate workspace context through supported modules and provide workspace selection for every applicable user role.
- StatSpatial ownership migrations and scoped store behavior are now certified against live PostgreSQL; keep the opt-in integration test in CI once database services are available.
- StatTrust PostgreSQL persistence now keeps storage identity separate from each workspace's ledger-chain index and has live cross-workspace reload certification.
- StatFederation workspace migrations and scoped object-link behavior are now certified against live PostgreSQL.
- All StatFederation persisted domains now have workspace-aware PostgreSQL predicates and scans, including nodes, DSAs, indicators, queries, vocabularies, diplomacy, reports, compliance logs, search indices, and object links.

### Stage 3: Complete Cross-Module Collaboration

- Extend the now-certified PMS/RMS and Enterprise workflow object-conversation pattern to field, survey, governance, and later modules.
- Standardize event names, payloads, retries, idempotency, and dead-letter handling.
- Finish remaining notification preferences, competitive collaboration features, and later-module frontend deep links; channels, tasks, calendar, polls, assistance, secure member controls, retention, exports, and legal holds are implemented.
- Add end-to-end tests for conversations created from project, research, survey, and governance contexts.

### Stage 4: Finish the Current Product Modules

- PMS: LogFrame, Theory of Change, portfolio KPIs, donor workflows, resources, and GIS links.
- RMS: DOI, citations, IRB workflow, journal tracking, repository, and research assistant.
- Statistics: questionnaire, sampling, enumeration, supervision, tabulation, census, SDMX, and dissemination.
- Analytics: ad-hoc querying, forecasting, scheduled reporting, geospatial analysis, and conversational analytics.
- Governance: whistleblower, conflict of interest, board packs, and centralized configuration.
- Field and GIS: PostGIS certification, spatial analysis, map services, GPS streaming, and supervision.

### Stage 5: Add Advanced Intelligence

Only after the data, identity, search, and event foundations are stable:

- Build the LLM gateway and provider abstraction.
- Add embeddings and vector storage.
- Implement governed RAG with tenant and document-level permissions.
- Add module-specific assistants for research, projects, surveys, and analytics.
- Add model registry, evaluation, cost controls, prompt governance, and human approval.

### Stage 6: Expand into Later Roadmap Domains

Prioritize later phases by dependency and user value, not by phase number. A sensible order is:

1. Data platform, semantic search, open data, and knowledge preservation
2. Integration platform and federation
3. Finance, procurement, documents, and enterprise operations
4. Low-code, marketplace, learning, and mobile expansion
5. IoT, digital identity, digital twins, advanced AI, and national-scale infrastructure

Each expansion phase must have a named service owner, a minimum deployable slice, migration plan, security review, integration tests, operational runbook, and explicit exit criteria.

## Definition of Done for a Roadmap Phase

A phase should only be marked complete when:

- Its backend and frontend workflows are implemented.
- Its database migrations work on a clean environment.
- Registry identity, tenant isolation, and permissions are enforced.
- Required event-bus and object-link integrations work.
- Unit, integration, and end-to-end tests pass.
- The feature is included in the supported deployment path.
- Health, metrics, logs, backups, and recovery behavior are documented.
- A user can complete the stated acceptance scenario without manual database intervention.

## Immediate Next Actions

The next practical release should focus on these five outputs:

1. A certified baseline Compose stack.
2. A passing CI pipeline with reproducible tests and builds.
3. A tenant and authorization test suite covering every current service.
4. Extend certified StatChat object conversations beyond PMS/RMS/Enterprise workflows and add durable event-bus integration.
5. Complete the remaining StatChat Phase 3 acceptance gaps: browser E2E and real multi-browser/NAT conferencing reliability with production TURN public-address/TLS.

Until these outputs exist, the platform should be described as a broad and active implementation with a strong foundation, not as a completed 50-phase product.

## Source Roadmap Documents

- [Phase 1 - Foundation](phase1.md)
- [Phase 2 - Identity](phase2.md)
- [Phase 3 - Collaboration](phase3.md)
- [Phase 4 - PMS](phase4.md)
- [Phase 5 - RMS](phase5.md)
- [Phase 6 - Statistics](phase6.md)
- [Phase 7 - Data Governance](phase7.md)
- [Phase 8 - Analytics](phase8.md)
- [Phase 9 - AI](phase9.md)
- [Phase 10 - GIS](phase10.md)
- [Phase 11 - Field Operations](phase11.md)
- [Phase 12 - M&E](phase12.md)
- [Phase 13 - Governance](phase13.md)
