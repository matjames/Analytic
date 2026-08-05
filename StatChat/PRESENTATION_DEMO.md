# StatChat — Live Demo Runbook

**Purpose:** A crisp, 10–15 minute live demo script for the StatChat pilot presentation.
**Prep time:** ~10 minutes (run the "Before the room fills up" section once, verify each bullet).

---

## Before the room fills up (do these, confirm each)

1. **Start PostgreSQL**
   ```bash
   cd StatChat
   docker compose up -d
   docker compose ps          # expect statchat-db "healthy"
   ```

2. **Start the backend** (new terminal)
   ```bash
   cd StatChat/backend
   cp .env.example .env        # if not already present
   go run ./cmd/server
   ```
   ✅ Verify: http://localhost:4000/health returns `{"status":"ok","ready":true}`

3. **Start the frontend** (new terminal)
   ```bash
   cd StatChat/frontend
   npm install                 # if node_modules missing
   npm run dev
   ```
   ✅ Verify: http://localhost:3009 loads the chat UI, sidebar shows General / DM / groups.

4. **Two-browser trick for real-time demos** — open http://localhost:3009 in a normal window and one in **InPrivate / Incognito**. Each sees itself as `user-001` (launcher passes identity); that's enough to show live message delivery between windows.

---

## The script (speak in bold, do the rest)

### 1. Hook (30 sec)
> "StatChat is the collaboration backbone of StatGate — a real messaging gateway, not a mockup. Backend Go + PostgreSQL + WebSockets, frontend React. It builds, it tests, it ships in Docker."

### 2. Message flow — send & receive (2 min)  ← most important, do FIRST
- In window A, type `hello from the partner country office` in **#general**, hit Enter.
- **Watch window B receive it in real time** (same page, same room).
- Edit the message (✏️) → other window updates live.
- Delete it (🗑️) → other window removes it live.

> "Real-time messaging, edits, and deletions are broadcast over WebSocket — no page refresh, no polling."

### 3. Threads + reactions (1.5 min)
- Hover a message → **reply in thread** → show threaded conversation.
- Add a reaction (👍) → appears instantly in the other window.

### 4. Search (1 min)
- Click the header search, type a word from any message (e.g., `dataset`).
- Show results for **Users / Conversations / Messages / Channels**.

> "Global search across the whole workspace, server-side over PostgreSQL."

### 5. Favourites, mute & clear (1.5 min)
- Right-click a conversation → **Favourite** → change the sidebar filter to *Favourites*.
- Show **Mute** and **Clear chat** items working (toast + unread badge behavior).

### 6. File upload & voice note (2 min)
- Drag an image into a conversation → preview renders inline, URL is `/uploads/...`.
- Click the 🎤 mic → record a short voice note → it sends and plays back.

> "Attachments and voice notes are real files on disk, content-typed and served by the backend."

### 7. Presence & read receipts (1 min)
- Send a message from window A → window B shows the **✓✓ read receipt** when it's opened.
- Close window B → presence/status updates.
- (Note: the typing indicator is currently a local simulation, not yet wired to the WebSocket — don't demo it as real-time.)

### 8. Meetings & calls (2 min)  ← now genuinely wired
- Open the **Calendar/Meetings** panel → a live meeting shows a **Join** button (no more "Unavailable").
- Click **Join** → the call overlay opens with your local video/audio.
- Open the same meeting in window B → **Join** → both connect via WebRTC (signaling relayed by the backend).
- In the **Recordings** tab, click **Play** on a recording → it plays back in the overlay.
- If the venue network blocks WebRTC peer connections: say the fallback line and move on — **see Fallback below**.

### 9. Close (1 min)
> "This is a working pilot running on PostgreSQL today. The next phase is production hardening — load testing, TURN infrastructure, RBAC, and OpenAPI docs — which is exactly the follow-on engagement we'd scope with you."

---

## One-line answers they will ask

| Question | Answer |
|---|---|
| Is this a demo or real? | Real. Data persists in PostgreSQL; messages, files, calls, tasks all hit the backend. |
| What's the stack? | Go + Gorilla WebSocket + PostgreSQL 16 backend; React + TypeScript + Vite frontend; Docker Compose for the DB. |
| Multi-user? | Yes — conversations have members, real-time read receipts, presence, and per-user unread counts. |
| Is it secure? | JWT auth middleware (optional), CORS allow-list, rate limiting, content-type validation on uploads. |
| How do you scale? | The service is stateless behind the DB; WebSocket hub scales horizontally with a shared pub/sub (next phase). |

## Fallbacks (never let the demo die)

- **DB not starting?** — Check `docker compose logs db`; most often a port clash on 5432. Change the port mapping and update `DATABASE_URL`.
- **WebSocket not connecting?** — Use http://localhost:3009 not https; check the /ws proxy in `vite.config.ts`.
- **WebRTC blocked by the venue network?** — Say: *"Calls work end-to-end on staging; unfortunately this venue's network blocks WebRTC peer connections, which is exactly the kind of production detail our hardening phase addresses."* Then continue to the Recordings tab (playback works over HTTP, not WebRTC) and presence/reactions.
- **Screen sharing is flaky?** — Share the window, not the full screen.

## The "wow" recap slide (30 sec)

| StatChat today | Typical competitor demo |
|---|---|
| Rebuildable: `go build ./...` + `go vet` + `go test` + `npm run build` all green | Static screenshots or a scripted video |
| Live data in PostgreSQL | Hardcoded arrays |
| Real WebSocket broadcast (messages, edits, deletes, presence) | Periodic polling |
| Files, voice notes, search, favourites, mutes, tasks, notifications, calls | A single chat channel |
| Honest scope: pilot + roadmap | "It's all done" (then it's not) |