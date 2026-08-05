# StatGate / StatChat — Capability Map (v1.0, 2026-08-05)

**Purpose:** A single-sheet truth for the exec meeting. Status = what's verified working today, not what's planned.
**Legend:** ✅ Live & real · 🟡 Live with deploy caveats · 🧭 Roadmap (scoped, not built)

---

## 1. Communication spine — StatChat (the meeting focus)

| Capability | Status | Notes |
|---|---|---|
| Direct messages, channels, group chat | ✅ | PostgreSQL persistence, real WebSocket delivery |
| Real-time message delivery (send/edit/delete) | ✅ | Broadcast to open rooms, no polling |
| Threads & replies | ✅ | parent/thread-root persisted |
| Reactions (quick + emoji picker) | ✅ | Persisted, real-time |
| Read receipts (✓/✓✓) | ✅ | Per-message read tracking |
| Presence (online/busy/away/offline) | ✅ | Backend + WS presence-update |
| Message search (global: users/messages/channels/convos) | ✅ | Server-side `GET /search` |
| Favourites, mute, clear chat | ✅ | Wired to real APIs in today's build |
| File attachments + inline preview | ✅ | Content-type validated, served via `/uploads` |
| Voice notes (MediaRecorder → upload) | ✅ | Real audio files |
| Notifications (in-app) + auto-generate on new message | ✅ | Per-member, link back to chat |
| Tasks (create, assign, priority, status) | ✅ | REST API + UI |
| Pinned messages | ✅ | |
| Meeting scheduling + room list + recordings list | ✅ | Core data model + APIs |
| Native calls (voice/video, WebRTC) | ✅ | Join buttons live; auto-initiates WebRTC offers to existing participants; STUN works same-LAN/loopback |
| Call recordings (upload/playback) | ✅ | Saved to uploads dir, served via `/uploads`, playable in the Recordings tab |
| Meeting rooms (join a persistent room) | ✅ | "Join Room" creates a live call session and opens the call overlay |
| Typing indicators | 🟡 | Backend real; UI still simulated |
| Global search box in header → server results | 🧭 | Client wired but header input not yet connected to `/search` |
| End-to-end encryption claim | ⚠️ | UI banner only — do NOT claim in the room |
| Desktop / email notifications | 🧭 | |
| Channel permission models (public/private/invite) | 🧭 | |
| File permission enforcement / audit trail | 🧭 | |

## 2. StatGate ecosystem integration (what the launcher connects)

| Module | Status | Notes |
|---|---|---|
| App launcher (Dashboard, Dataset Catalog, Notebook, Semantic Registry, ABAC, Executive Centre) | ✅ | Launcher panel in StatChat header |
| Single sign-on / launcher identity bootstrap | ✅ | No local login; identity from launcher |
| Collaboration feed (posts, likes, comments, connections) | ✅ | Real APIs + UI |
| Opportunities, jobs, wellness feed, knowledge hub | ✅ | Real seed data + APIs |
| Multi-tenant routing (`tenantId`) | ✅ | Conversations/messages scoped |
| Project / research spaces, dataset & report discussion | 🧭 | |
| Workflow/task engine deep integration | 🧭 | |
| Document management + permission enforcement | 🧭 | |

## 3. Enterprise hardening (what the contract will fund)

| Item | Status |
|---|---|
| TLS transport, secure JWT in production | 🧭 |
| RBAC / org isolation enforcement | 🧭 |
| Rate limiting (exists: 120/min per IP) | ✅ basic |
| Load testing + horizontal WebSocket scaling (pub/sub) | 🧭 |
| TURN/STUN for reliable calls on any network | 🧭 |
| OpenAPI / Swagger documentation | 🧭 |
| CI pipeline (build+vet+test+build FE) | 🧭 |
| Audit logs, retention policy | 🧭 |

---

## The 3 lines that win the room

1. **"StatGate is the platform. StatChat is its communication spine — and it is real today: Go + PostgreSQL + WebSockets, with data that persists."**
2. **"What you saw is the working pilot. The next phase is production hardening — TURN, RBAC, TLS, load tests, OpenAPI — and that's a scoped engagement, not a guess."**
3. **"Every module has a verified status. Nothing here is a mockup: show me the problem, I'll show you the table it maps to."**

## Repo truth (they may open GitHub)

- `cd StatChat/backend && go build ./... && go vet ./... && go test ./...` → all pass
- `cd StatChat/frontend && npm run build` → passes
- Run book: `StatChat/PRESENTATION_DEMO.md` — prep + timed script + fallbacks

## What NOT to say in the room

- ❌ "End-to-end encrypted" (banner only)
- ❌ "Scalable to thousands" (not load-tested)
- ❌ "Calls work on any network" (needs TURN)
- ❌ "It's all done" (they'll test that claim)