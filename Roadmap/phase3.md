# PHASE 3 — COMMUNICATION, COLLABORATION & PRODUCTIVITY ECOSYSTEM

## Objective

Develop the complete collaboration platform for StatGate.

StatChat is the official communication and collaboration backbone. Users should never need to leave StatGate to communicate, collaborate, share files, conduct meetings, assign tasks, or coordinate projects.

Every module developed in subsequent phases leverages this communication infrastructure — conversation threads auto-created for platform objects, notifications routed through it, and real-time presence shared across the platform.

---

## Vision

StatChat functions as a hybrid of WhatsApp, Slack, Microsoft Teams, and Discord — while remaining tightly integrated with statistical, research, and governance workflows.

---

## Service: StatChat

**Repository:** `StatChat/`  
**Backend:** Go + Gorilla Mux + WebSocket → `StatChat/backend/` (:4000)  
**Frontend:** React 18 + Vite + TypeScript → `StatChat/frontend/` (:3009)  
**Database:** `statchat` (PostgreSQL 15)  
**Real-time:** WebSocket hub with room-based broadcasting  
**Auth:** Registry JWT (always required — `authRequired()` permanently returns `true`)

---

## Backend Package Structure

```
StatChat/backend/
├── cmd/                          ← entry point
├── pkg/
│   ├── api/
│   │   ├── handlers.go                   ← core chat handlers
│   │   ├── chat_completeness_handlers.go ← favourites, mute, clear, presence
│   │   ├── conferencing_handlers.go      ← voice/video call sessions
│   │   ├── feature_handlers.go           ← channels, tasks, calendar, meetings
│   │   ├── registry_directory.go         ← sync from Registry user directory
│   │   └── gateway_router.go             ← WebSocket gateway routing
│   ├── model/                    ← data structures
│   └── store/                    ← PostgreSQL data access layer
```

---

## What Exists ✅

### Core Messaging

**HTTP REST API**
```
POST   /api/v1/chat/conversations                        ← create conversation (DM or group)
GET    /api/v1/chat/conversations                        ← list my conversations
GET    /api/v1/chat/conversations/:id                    ← conversation detail
GET    /api/v1/chat/conversations/:id/messages           ← message history
POST   /api/v1/chat/conversations/:id/messages           ← send message
PUT    /api/v1/chat/messages/:id                         ← edit message
DELETE /api/v1/chat/messages/:id                         ← delete message
POST   /api/v1/chat/messages/:id/react                   ← add reaction
DELETE /api/v1/chat/messages/:id/react                   ← remove reaction
POST   /api/v1/chat/messages/:id/pin                     ← pin message
DELETE /api/v1/chat/messages/:id/pin                     ← unpin message
POST   /api/v1/chat/messages/:id/forward                 ← forward message
DELETE /api/v1/chat/conversations/:id/messages           ← clear chat history

GET    /api/v1/chat/conversations/:id/members            ← conversation members
POST   /api/v1/chat/conversations/:id/members            ← add member to group
DELETE /api/v1/chat/conversations/:id/members/:userId    ← remove member

POST   /api/v1/chat/conversations/:id/favourite          ← toggle favourite
POST   /api/v1/chat/conversations/:id/mute               ← mute conversation
POST   /api/v1/chat/conversations/:id/unmute             ← unmute conversation
```

**Channels**
```
POST   /api/v1/chat/channels                  ← create channel
GET    /api/v1/chat/channels                  ← list channels
GET    /api/v1/chat/channels/:id              ← channel detail + messages
POST   /api/v1/chat/channels/:id/join         ← join channel
POST   /api/v1/chat/channels/:id/leave        ← leave channel
```

**Tasks**
```
POST   /api/v1/chat/tasks                     ← create task
GET    /api/v1/chat/tasks                     ← list my tasks
PUT    /api/v1/chat/tasks/:id                 ← update task
DELETE /api/v1/chat/tasks/:id                 ← delete task
```

**Calendar & Meetings**
```
POST   /api/v1/chat/calendar                  ← create calendar event
GET    /api/v1/chat/calendar                  ← list calendar events
PUT    /api/v1/chat/calendar/:id              ← update calendar event
DELETE /api/v1/chat/calendar/:id              ← delete calendar event
```

**Voice / Video Conferencing**
```
POST   /api/v1/chat/calls                     ← initiate call session
GET    /api/v1/chat/calls/:id                 ← get call session
POST   /api/v1/chat/calls/:id/join            ← join call
POST   /api/v1/chat/calls/:id/leave           ← leave call
POST   /api/v1/chat/calls/:id/signal          ← WebRTC signalling
GET    /api/v1/chat/calls/:id/participants    ← active participants
POST   /api/v1/chat/calls/:id/record          ← start recording
POST   /api/v1/chat/calls/:id/stop-recording  ← stop recording
```

**File Sharing**
```
POST   /api/v1/chat/upload                    ← upload file attachment
GET    /api/v1/uploads/:filename              ← serve uploaded file
```

**Profile & Presence**
```
GET    /api/v1/chat/profile                   ← my profile
PUT    /api/v1/chat/profile                   ← update profile
GET    /api/v1/chat/users                     ← user directory (from Registry)
GET    /api/v1/chat/users/:id                 ← user profile
GET    /api/v1/chat/presence                  ← presence status list
```

**Search**
```
GET    /api/v1/chat/search?q=                 ← search messages, channels, users
```

**WebSocket Gateway** (`ws://localhost:4000/ws`)

Actions broadcasted in real-time:
- `message` — new message
- `typing` / `stop-typing` — typing indicators
- `presence-update` — user online/offline/away
- `reaction` — message reaction added/removed
- `channel-join` / `channel-leave`
- `call-signal` — WebRTC SDP offer/answer/ICE candidates

### Registry Directory Sync
```
GET  /api/v1/internal/sync-users       ← pull users from Registry into StatChat store
```

---

## What is Missing ❌

### Core Completeness
- **Later-module adoption** — PMS and RMS now use canonical StatChat object conversations, and Enterprise workflow creation uses the supported contract; field, survey, governance, and later modules still need adoption.
- **Platform event bus** — publish/consume durable conversation, message, notification, and presence events across services.

### Frontend Gaps
- **Remaining object entry points** — PMS/RMS exact-reference deep links are implemented; workflow, survey, governance, and later-module entry points remain.
- **Competitive collaboration tools** — collaborative documents, whiteboards, and translation are implemented; broader social-graph policy remains.

### Conferencing Gaps (Phase D)
- Configure the Compose-provided TURN service with a public relay address and TLS certificate for production NAT traversal.
- Add real multi-browser reliability, network recovery, moderation, and end-to-end NAT tests.

### Features Not Yet Built
- AI meeting transcription and minutes (grounded chat assistance is implemented)
- SMS / push / mobile push notifications
- End-to-end encryption

---

## Database Tables (`statchat` database)

```sql
-- Core messaging
conversations, conversation_members, conversation_roles, messages, message_attachments, message_locations, message_mentions, saved_messages
message_reactions, message_pins, threads
-- Compliance
retention_policies, legal_holds, compliance_audit_events
-- Channels
channels, channel_members
-- Presence & settings
user_presence, conversation_mutes, favourite_conversations, user_settings
-- Calls
call_sessions, call_participants, call_recordings
-- Collaboration
tasks, calendar_events, polls, poll_options, poll_votes, meetings, meeting_recordings
-- Notifications
notifications
```

---

## Integration with Other Modules

All StatGate modules can request StatChat to create a conversation thread for a platform object:

```
POST /v1/chat/conversations/object
{
  "objectRef": "obj:pms:project:<id>",
  "name": "Project Alpha Discussion",
  "memberIds": ["user-uuid-1", "user-uuid-2"]
}
```

This is how the platform achieves threaded, searchable discussion on datasets, projects, research, surveys, and incidents without each module building its own messaging system.

---

## Acceptance Criteria

- [x] Direct messaging operational
- [x] Group chat operational
- [x] Channels operational — tenant-scoped create, discovery, join, leave, owner-only archive, and live conversation entry points verified by backend suite and frontend production build
- [x] Tasks operational — tenant/member-scoped CRUD, status transitions, creator deletion, and validated status/priority contracts covered by backend tests and frontend production build
- [x] Calendar operational — tenant/attendee-scoped create, list, edit, and delete are wired, build/test-covered, and verified against the live PostgreSQL deployment
- [x] File sharing operational
- [x] Presence real-time operational
- [x] Typing indicators operational
- [x] Favourites and mute operational
- [x] Global and advanced search operational — tenant/member-scoped message search supports text, conversation, sender, date range, attachment, and saved-only filters with exact-result navigation; local user/channel fallback results remain tenant-scoped.
- [x] Voice/video call sessions operational (basic)
- [x] Registry user directory sync operational
- [x] Message notifications working (all members notified)
- [x] WS envelope events for update/delete/read
- [x] Frontend wired to supported product APIs — repeatable audit verifies 92 typed frontend operations against 104 routes; the 12 uncovered routes are health/readiness, legacy aliases, upload serving, and duplicate task aliases. Secure group/channel member administration is exposed only when the server reports owner/admin management permission.
- [ ] Conferencing bugs resolved — authenticated host controls, immutable host plus moderator roles, moderator-limited removal, consent-based mute requests, participant counts, call termination, host-only WebM recording upload/listing, tenant/conversation-scoped call discovery and signaling, spoof-resistant join/leave identity, quality telemetry, and TURN deployment are live-verified. Screen sharing is implemented for every active voice/video participant. Signaling now reconnects with capped backoff; peer failures use a grace period and bounded ICE restart; participants are re-registered/reoffered; online/offline recovery is visible; and RTT, jitter, loss, bitrate, and server-derived quality are sampled with active-participant authorization and 30-day persistence. Production TURN TLS/public-address configuration is documented. Real multi-browser screen/recovery/NAT proof remains.
- [x] Polls operational — tenant-scoped creation/listing, unique option validation, one-vote-per-user persistence, collaboration-feed voting UI, and live PostgreSQL create/list/vote verification completed
- [x] AI chat assistant operational — authenticated conversation summaries, action extraction, and review-before-send reply drafting use only tenant-scoped messages the caller may access
- [x] PMS and RMS object discussions operational — live writes through each module are readable from both the module workspace and canonical StatChat object conversation; a second authenticated user is enrolled and can read both.
- [x] Enterprise workflow object-conversation contract corrected and covered by an HTTP contract test.
- [x] Conversation-member administration operational — persisted owner roles, tenant-scoped Registry directory selection, owner/canonical-admin authorization, direct-chat immutability, owner-removal protection, and permission-aware UI are test/build-covered and live-verified.
- [x] Message forwarding operational — source and destination access are independently enforced, forwarded text and attachments are copied transactionally, source provenance is visible, the authenticated forwarder remains the new sender, realtime delivery/notifications are emitted, and destination read-back is live-verified.
- [x] Scheduled messages operational — RFC3339 input is normalized to UTC, pending jobs persist through restarts, sender-only list/cancel controls are wired, delivery claims use transactional row locks for multi-instance safety, conversation access is rechecked at dispatch, and delivered messages use realtime/notification paths.
- [x] Built-in GIF/sticker search and sending operational — all authenticated users receive a searchable privacy-safe catalog without vendor keys or tracking, catalog IDs are server-validated, message/attachment persistence is atomic, conversation access and authenticated sender identity are enforced, and media survives forwarding.
- [x] Location sharing operational — browser permission is requested only after explicit user action, coordinates and accuracy are shown for confirmation before sending, structured location data is range-validated and persisted atomically, conversation membership and authenticated sender identity are enforced, forwarding preserves the location, and no map provider is contacted until the recipient deliberately opens the map.
- [x] Mentions and notification routing operational — member autocomplete carries stable account IDs, mention targets are conversation-validated and persisted atomically, manager-only `@all` prevents broadcast abuse, mute and message/group/mention/preview preferences are enforced, explicit mentions can route through mute when enabled, realtime notification envelopes update connected users, and notification read ownership is enforced.
- [x] Saved messages operational — private per-user saves are idempotent, accessible from every chat workspace, removable independently, searchable with other advanced filters, hidden immediately after conversation access is lost, and exact saved results navigate back to the source message across direct, group, and channel conversations.

- [x] Retention, export, and legal hold operational — members receive authenticated server-generated JSON/CSV exports only for conversations they can access; deleted-record exports and compliance controls require tenant administration; retention is disabled by default, supports 1-3650 day policies, runs hourly or on demand, permanently removes eligible message dependencies and local uploads, and is blocked by active tenant-wide or conversation holds. Policy changes, hold lifecycle, enforcement, and exports are durably audited.

### Verified foundation update — 21 August 2026

- Browser WebSockets authenticate with the Registry JWT through the WebSocket subprotocol; long-lived credentials are no longer placed in URLs.
- Direct/group conversations are membership-filtered, and message, attachment, reaction, read receipt, pin, favourite, mute, clear, typing, and WebSocket-send operations enforce conversation access.
- Message read and reaction changes are broadcast and consumed in real time.
- Tenant-scoped object conversations are available through `GET/POST /v1/chat/conversations/object` using `obj:<module>:<entity>:<id>` references.
- Attachment and recording storage is mounted on the persistent `statchat_uploads` volume.
- PMS and RMS no longer write new UI discussion messages to their local chat tables. Enterprise workflows now target the object-conversation API. Remaining integration work is later-module adoption and durable platform event publication.

### Verification update — 26 August 2026

- StatChat route audit: 92 typed frontend operations verified against 104 backend routes; no typed-client/backend mismatch.
- Full Go test and vet suites passed for StatChat, PMS, RMS, Enterprise Core, and the shared StatChat client.
- Production frontend builds passed for StatChat, PMS, and RMS.
- Live PostgreSQL/Redis deployment verified PMS `obj:pms:project:proj-2` and RMS `obj:rms:research:1787776372788082174` write/read paths, canonical lookups, and second-user enrollment.
- RMS certification workspace `1787776372788082174` was created because the existing RMS volume contained no research workspace.
- PMS/RMS Compose now receives Redis, Registry JWT, Enterprise workspace, and internal StatChat configuration; both APIs report healthy database and Redis dependencies.
- Clean bootstrap ownership was corrected so PMS/RMS runtime roles can apply additive migrations, and migration errors are no longer silently ignored.
- Automated browser verification could not run because this session exposed no browser binding. Public TURN address/TLS and external multi-browser NAT/recovery certification remain release gates.

### Verification update — 27 August 2026

- Registry internal directory requests now accept an authenticated `X-Tenant-ID` scope and return only matching `organisation` users; live verification returned 200 scoped users and zero foreign users.
- Group/channel conversations persist an owner in `conversation_roles`; existing non-direct conversations receive a deterministic owner backfill.
- Live API verification proved owner management (`204`), ordinary-member denial (`403`), tenant-administrator management (`204`), owner-removal protection (`409`), and immutable direct-chat membership (`409`).
- The StatChat member manager lists members, identifies the owner, adds tenant-directory users, removes eligible users, and is shown only when the backend returns `canManage`.
- Registry and StatChat Go suites pass, StatChat passes `go vet`, the route contract remains 92/104 with no typed mismatch, and the production frontend build passes.
- Docker build contexts now exclude local caches, logs, uploads, and Windows executables; measured contexts fell to approximately 4.5 MB for Registry and 56 KB for StatChat.
- Message forwarding live verification returned `201`, persisted source-message and source-sender provenance, preserved text, and returned `403` when either source or destination access was missing. The typed route audit advanced to 93/105 with the same 12 intentional infrastructure/legacy aliases outside the product client.
- Scheduled-message live verification returned `201` for creation, survived a backend container restart, delivered once with the authenticated sender, returned `404` for another user's cancellation attempt, returned `204` for sender cancellation, rejected past time with `400`, and left zero pending records. The typed route audit advanced to 96/108.
- Built-in media live verification returned `200` for tag search, `400` for an unknown catalog ID, `403` for a nonmember send, and `201` for a member send and forward. Read-back preserved the authenticated sender, self-contained SVG MIME/data, and forwarded attachment. The typed route audit advanced to 98/110.
- Location-sharing live verification returned `400` for an out-of-range coordinate, `403` for a nonmember, `201` for member send and forward, and `200` for authorized source/destination read-back. Persistence preserved the authenticated sender, latitude, longitude, accuracy, label, forwarded location, and authenticated forwarder identity. Go tests/vet and the production frontend build pass; the typed route audit advanced to 99/111.
- Mention-routing live verification returned `400` for a spoofed recipient and ordinary-member `@all`, persisted valid stable-ID mentions, suppressed ordinary notifications for a muted conversation, delivered an enabled explicit mention through mute, suppressed mentions when the recipient disabled them, delivered owner-authorized `@all`, protected message previews by default, and returned `404` when another user attempted to mark a notification read. Authenticated identities now synchronize before every protected route, and the repaired settings/mute migrations are live-verified. The route audit remains 99/111 because the capability extends existing typed message and notification operations.
- Saved-message and advanced-search certification proved private and idempotent per-user saves, outsider denial, saved-only isolation, independent removal, automatic hiding after membership revocation, and conversation/sender/date/attachment filters against live PostgreSQL. Invalid dates/booleans returned `400`, inaccessible explicit-conversation filters returned `403`, frontend/backend tests and vet passed, and the production frontend build passed. The repeatable certification is `StatChat/scripts/certify_saved_search.ps1`; the typed route audit advanced to 102/114.
- Retention/export/legal-hold certification proved the safe disabled default, administrator boundaries, policy/hold validation, member JSON/CSV export, outsider denial, administrator-only deleted export, conversation and tenant hold protection, permanent purge after release, and required immutable audit actions against live PostgreSQL. CSV formula injection is neutralized, policy/hold mutations are atomically audited, Go tests/vet and the production frontend build pass, and the repeatable certification is `StatChat/scripts/certify_compliance.ps1`; the typed route audit advanced to 110/122.
- Collaboration-feed hardening now derives post and comment identity from the authenticated user, scopes posts/comments and all post interactions to the caller's tenant, and rejects cross-tenant reads, likes, shares, and comments. Connections are tenant-scoped, self-connections are rejected, and cross-tenant targets return not found. `StatChat/scripts/certify_collaboration_feed.ps1` passes authenticated-author, tenant read/write isolation, comment-authorship, and connection-boundary assertions; Go tests/vet and the frontend build pass.
- Communities and discussion forums now provide tenant-scoped public/private communities, creator ownership, owner-managed invitations, public join/leave, member-only topics and replies, authenticated topic/reply authorship, private visibility, owner/member moderation boundaries, ownership transfer, frontend workspace access, and typed client coverage. `StatChat/scripts/certify_communities.ps1` passes public creation, private scope, membership boundary, private invite, owner-only membership, authenticated topic/reply, cross-tenant write/read denial, moderation boundary, ownership transfer, and topic read-back assertions; Go tests/vet, the frontend build, and the route audit pass at 137/149.
- Knowledge Hub/wiki workflows now provide tenant-scoped knowledge posts, authenticated authorship, expanded article content, per-user expert follows and idea upvotes, idempotent interaction counts, frontend article/post workflows, and live certification. `StatChat/scripts/certify_knowledge.ps1` passes authenticated post authorship, tenant isolation, article content availability, idempotent follows/upvotes, per-user interaction state, and missing-target handling; Go tests/vet, the frontend build, and the route audit pass at 148/160.
- Collaborative documents now provide tenant-scoped shared pages, authenticated ownership, editor/viewer membership, optimistic version conflict protection, revision history, owner-managed access, frontend editing, and live certification. `StatChat/scripts/certify_documents.ps1` passes owner/authentication, tenant list/read isolation, member access boundaries, editor updates, stale-version conflict handling, viewer write denial, owner-only management/deletion, cross-tenant member denial, and revision history; Go tests/vet, the frontend build, and the route audit pass at 148/160.
- Whiteboards now provide tenant-scoped canvas boards, authenticated ownership, editor/viewer membership, persisted stroke data, optimistic version conflict protection, revision history, owner-managed access, frontend canvas editing, and live certification. `StatChat/scripts/certify_whiteboards.ps1` passes owner/authentication, tenant list/read isolation, member access boundaries, editor updates, stale-version conflict handling, viewer write denial, owner-only management/deletion, cross-tenant member denial, and revision history; Go tests/vet, the frontend build, and the route audit pass at 148/160.
- Translation now provides authenticated language discovery, validated source/target translation, same-language identity behavior, a local fallback provider, frontend workspace access, and live certification. `StatChat/scripts/certify_translation.ps1` passes language discovery, glossary translation, same-language handling, invalid-language and empty-input rejection, and authentication requirements; Go tests/vet, the frontend build, and the route audit pass at 148/160.
- Host moderation now includes immutable host protection, host-managed moderator roles, moderator-limited participant removal, explicit consent-based mute requests, persistent departure/role state, realtime propagation, and post-removal telemetry denial. The live authorization certification passes all fourteen assertions, and `StatChat/scripts/certify_call_mute.ps1` passes live WebSocket denial, request delivery, and recipient-consent assertions. Go tests/vet, the production frontend build, and the route audit pass at 114/126. The production TURN/TLS runbook is in `StatChat/docs/CALL_DEPLOYMENT.md`; real multi-browser interruption, screen capture, and external NAT proof remain release gates.
- Screen-sharing implementation now supports camera-track replacement, presentation tracks in voice calls, late joiners, automatic restoration when browser sharing ends, remote presenter emphasis, mobile-safe controls, and presentation-aware host recording. Call sessions now persist tenant ownership; create/read/list/join/leave/participants/recordings/end and WebSocket signaling enforce tenant, conversation, authenticated identity, active participation, target activity, and a strict SDP/ICE/presenter signal whitelist. Live certification proved nine multi-user authorization and spoof-resistance scenarios through `StatChat/scripts/certify_call_authorization.ps1`; Go tests/vet, the frontend production build, and the 110/122 route audit pass. Real screen capture remains uncertified because browser discovery returned no available binding.
- Multi-party recovery and quality telemetry now add capped exponential WebSocket reconnect, online/offline handling, delayed peer eviction, ICE restart and participant reoffer, ended-session/timer cleanup, accessible quality/recovery UI, and `getStats()` aggregation for RTT, jitter, packet loss, and bitrate. Active participants alone can submit bounded metrics; the server derives quality labels, authorized call members can read diagnostics, and indexed samples expire after 30 days. The expanded live certification passes thirteen authorization, identity, telemetry, validation, isolation, and lifecycle assertions. Go tests/vet, the production frontend build, and the route audit pass at 112/124; real multi-browser interruption/NAT proof remains blocked by the unavailable browser binding.

---

## Ports & Services

| Component | Port |
|---|---|
| StatChat API + WebSocket (Go) | :4000 |
| StatChat UI (React/Vite) | :3009 |

---

## Estimated Duration

8–10 weeks

## Milestone

Enterprise Collaboration Platform complete. Every StatGate module can route communication through StatChat.
