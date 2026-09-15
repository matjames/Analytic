# StatChat Phase 3 Implementation Matrix

Status is evidence-based: a feature is complete only after UI, API, authorization, persistence, real-time behavior where applicable, and automated tests are verified.

## Batch A — Secure messaging foundation

| Capability | Status | Remaining evidence |
|---|---|---|
| Registry JWT HTTP authentication | Implemented | Expand issuer/audience tests |
| Browser WebSocket authentication | Implemented and live-tested | Browser UI test |
| Tenant and membership authorization | Implemented | Broader integration tests |
| Direct and group conversations | Implemented and live-tested | Browser UI E2E verification |
| Message send/edit/delete/reactions/read/pins | Implemented | Full UI E2E suite |
| Object conversations | Operational for PMS/RMS; Enterprise workflow contract implemented | Browser E2E and later-module adoption |
| Durable uploads | Container deployment implemented | Object-storage production profile |

## Batch B — Core productivity

| Capability | Status | Remaining evidence |
|---|---|---|
| Channel create/list/detail/join/leave/archive | Implemented and build/test-verified | Browser UI E2E verification |
| Conversation member list/add/remove | Operational: persisted ownership, owner/canonical-admin authorization, tenant-scoped directory, safe UI, and live API proof | Browser UI E2E verification |
| Task create/list/update/status/delete | Implemented in source and UI | Browser UI E2E verification |
| Calendar event create/list/update/delete | Live-verified with PostgreSQL; full UI wired | Browser UI E2E verification |
| Message forwarding | Operational: dual-conversation authorization, transactional text/attachment copy, visible provenance, authenticated forwarder identity, realtime delivery, typed UI, and live read-back proof | Browser UI E2E verification |
| Scheduled messages | Operational: durable UTC scheduling, sender list/cancel UI, delivery-time authorization, restart-safe multi-instance worker, realtime publication, notifications, and live restart proof | Browser UI E2E verification |

## Batch C — Competitive messaging

| Capability | Status |
|---|---|
| Polls and voting | Implemented and live-verified with PostgreSQL |
| GIF/sticker search and sending | Operational: searchable built-in animated reactions/stickers, no vendor keys or tracking, strict catalog validation, atomic persistence, authorization, realtime delivery, forwarding, and live proof; external provider integration is intentionally not required |
| Location sharing | Operational: explicit browser permission and confirmation, strict coordinate/accuracy validation, atomic structured persistence, conversation authorization, realtime delivery, privacy-safe rendering, forwarding, and live PostgreSQL proof; browser UI E2E remains |
| Mentions and notification routing | Operational: stable-ID member autocomplete, structured persistence, target validation, manager-only `@all`, mute and per-category/previews preferences, realtime notification envelopes, ownership-safe reads, and live multi-user proof; browser UI E2E remains |
| Saved messages and advanced search | Operational: private idempotent saves, access-revocation filtering, independent removal, text/conversation/sender/date/attachment/saved-only filters, exact-result navigation across chat types, tenant-safe fallback search, and repeatable live PostgreSQL certification; browser UI E2E remains |
| Retention, export, and legal hold | Operational: authenticated JSON/CSV member exports, administrator-only deleted-record exports, formula-safe CSV, disabled-by-default tenant policies, hourly/manual hard-deletion enforcement, tenant/conversation legal holds, atomic immutable audits, administrator UI, and repeatable live PostgreSQL lifecycle certification; browser UI E2E remains |
| End-to-end encryption | Not implemented; UI now accurately describes tenant access control and makes no E2EE claim |

## Batch D — Meetings and calls

| Capability | Status |
|---|---|
| Basic WebRTC voice/video | Partial |
| TURN deployment and NAT traversal | Compose deployment operational; production TLS/public-address configuration is documented; external NAT verification remains |
| Correct host/moderator controls | Implemented: immutable host, host-managed moderator roles, moderator-limited participant removal, explicit consent-based mute requests, host protection, tenant/conversation access, spoof-resistant participant identity, persistent participant state, and realtime role/removal propagation are build/test/live-certified; browser E2E remains |
| Screen sharing | Implemented: every active voice/video participant can explicitly request browser capture, replace/add the outgoing video track, present to late joiners, stop from StatChat or browser chrome, restore the prior camera, expose presenter layouts, include the local presentation in host recording, and receive unsupported/denied states; tenant/conversation/active-participant signaling boundaries are test/build/live-API verified, while real two-browser media proof remains blocked by unavailable browser binding |
| Multi-party recovery and quality telemetry | Implemented: capped exponential signaling reconnect, online/offline recovery, grace-period peer handling, bounded ICE restart, participant re-registration/reoffer, accessible recovery state, 10-second WebRTC RTT/jitter/loss/bitrate sampling, server-derived quality, active-participant writes, authorized diagnostics reads, and 30-day bounded persistence are build/test/live-API verified; real multi-browser interruption/NAT proof remains |
| Recording lifecycle | Host-only WebM upload/listing and persistent storage live-verified; browser capture E2E remains |
| AI minutes and action extraction | Grounded chat summary/action extraction implemented; call transcription/minutes not started |

## Batch E — Collaboration suite

| Capability | Status |
|---|---|
| Social posts/comments/connections | Operational: authenticated author identity, tenant-scoped post/comment/feed and connection access, tenant-isolated likes/shares/comments, explicit pending/accepted/declined requests, idempotent duplicate handling, recipient-only accept/decline, reciprocal accepted edges, reciprocal removal, self/cross-tenant denial, and live certification are implemented |
| Photo/video/article posts | Operational: authenticated tenant-scoped typed posts support article titles, validated photo/video media URLs, authenticated multipart image/video upload up to 16 MB, same-tenant feed visibility, frontend composers/rendering, and live certification |
| Wellness comments/shares/bookmarks persistence | Operational: authenticated tenant-scoped wellness authorship, durable comments, per-user idempotent likes/bookmarks, durable shares, default seeded content visibility, frontend controls, and live certification are implemented |
| Communities and forums | Operational: tenant-scoped public/private communities, creator ownership, owner-managed invitations, public join/leave, member-only topics/replies, authenticated authorship, private visibility, owner/member moderation boundaries, ownership transfer, frontend workspace, typed client coverage, and live certification are implemented |
| Collaborative documents | Operational: tenant-scoped shared documents, authenticated ownership, editor/viewer membership, optimistic version protection, revision history, owner-managed access, frontend editor workspace, and live certification are implemented |
| Whiteboards | Operational: tenant-scoped canvas boards, authenticated ownership, editor/viewer membership, persisted stroke data, optimistic version protection, revision history, owner-managed access, frontend canvas workspace, and live certification are implemented |
| Knowledge wiki | Operational: tenant-scoped knowledge posts, authenticated authorship, expanded article content, per-user expert follows and idea upvotes, idempotent interaction counts, frontend workflow, and live certification are implemented |
| Translation | Operational: authenticated language discovery, validated source/target translation, same-language identity handling, local fallback provider, frontend workspace, and live certification are implemented |

## Batch F — Platform integration

| Capability | Status |
|---|---|
| PMS object discussions | Operational: canonical `obj:pms:project:<id>` conversations, live write/read proof, deep link, and second-user enrollment |
| RMS object discussions | Operational: canonical `obj:rms:research:<id>` conversations, live write/read proof, deep link, readiness flag, and second-user enrollment |
| Enterprise workflow discussions | Implemented and contract-tested against `POST /v1/chat/conversations/object` with tenant-scoped workflow references |
| StatCollect survey discussions | Operational: canonical `obj:statcollect:submission:<id>` conversations, shared service-authenticated create/message requests, optional tenant/workspace context, and isolated HTTP contract coverage are implemented |
| StatGovernance object discussions | Implemented for tenant-scoped policies, risks, controls, findings, and evidence: authenticated discussion endpoint, canonical `obj:statgovernance:<type>:<id>` references, workspace propagation, existence checks, and frontend deep links; live StatChat deployment proof remains |
| StatIoT/field object discussions | Implemented at the authenticated API boundary for workspace-filtered gateways, devices, alerts, workers, and forms using canonical `obj:statiot:<type>:<id>` references; isolated HTTP contract coverage passes; no checked-in StatIoT frontend source or live deployment proof yet |
| StatData object discussions | Implemented at the authenticated API boundary for tenant/workspace-filtered datasets, pipelines, data sources, feature views, notebooks, experiments, and models using canonical `obj:statdata:<type>:<id>` references; isolated HTTP contract coverage passes; no checked-in StatData frontend source or live deployment proof yet |
| GeoIntel object discussions | Implemented at the authenticated API boundary for tenant-owned layers, rasters, scenes, drones, flight plans, and flights using canonical `obj:geointel:<type>:<id>` references; isolated HTTP contract coverage and Compose credentials are present; no checked-in frontend source or live deployment proof yet |
| AI Autonomy object discussions | Implemented for tenant/workspace-filtered twins, models, decision pipelines, agents, and graph nodes using canonical `obj:ai-autonomy:<type>:<id>` references; isolated HTTP contract coverage and Compose credentials are present; no checked-in frontend source or live deployment proof yet |
| Learning/CRM object discussions | Implemented for tenant/workspace-filtered courses, CRM, partner, and stewardship objects using canonical `obj:learning-crm:<type>:<id>` references; isolated HTTP contract coverage and Compose credentials are present; no checked-in frontend source or live deployment proof yet |
| BPM object discussions | Implemented for tenant/workspace-filtered process definitions, instances, cases, work items, and automation rules using canonical `obj:bpm-hub:<type>:<id>` references; isolated HTTP contract coverage and Compose credentials are present; no checked-in frontend source or live deployment proof yet |
| StatFederation object discussions | Implemented for tenant/workspace-checked federation and diplomacy objects using canonical `obj:statfederation:<type>:<id>` references; isolated HTTP contract coverage and Compose credentials are present; no live deployment proof yet |
| StatSpatial object discussions | Implemented for tenant/workspace-checked GIS and federation objects using canonical `obj:statspatial:<type>:<id>` references; isolated HTTP contract coverage and Compose credentials are present; no live deployment proof yet |
| StatTrust object discussions | Implemented for tenant/workspace-scoped incidents, ledger blocks, provenance, and certificates using canonical `obj:stattrust:<type>:<id>` references; isolated HTTP contract coverage is present and Compose credentials were already configured; no live deployment proof yet |
| Redis/platform event publication | Partial: shared library durable Redis Streams consumer groups, bounded retries, Redis-backed idempotency, pending-message reclamation, and durable plus Pub/Sub DLQ publication are implemented; StatCollect, StatIoT, GeoIntel, AI Autonomy, Learning/CRM, BPM, Knowledge Portal, StatFederation, StatSpatial, and StatData are opted into durable publication or consumption as appropriate, while live Redis certification and remaining custom consumers remain |
| Shared notification routing | Partial |
| Cross-app presence | Not started |

## Batch G — Production readiness

| Capability | Status |
|---|---|
| Push/mobile/SMS notifications | Not started |
| S3/MinIO production media storage | Not started |
| RBAC for channel and group administration | Not started |
| Versioned database migrations | In progress; PMS/RMS additive migrations now fail visibly and clean bootstrap ownership is corrected |
| Backend authorization/integration coverage | Expanded across object enrollment, internal service scope, shared client, PMS/RMS, and Enterprise workflow contracts |
| Frontend component and E2E tests | Not started |
| Accurate legal/status/security UI | Not complete |

## Required execution order

1. Add browser E2E coverage for route-audited Batch B workflows, secure member administration, and object deep links.
2. Add browser E2E coverage to the operational Batch C workflows; end-to-end encryption requires a separately approved key-management design.
3. Harden calls in Batch D, including public TURN/TLS and external NAT/recovery certification.
4. Build collaboration and AI features in Batch E.
5. Complete production readiness in Batch G before deployment approval.
