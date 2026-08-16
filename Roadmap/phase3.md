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
- **Message notifications** — `CreateNotification` not called after message store; other conversation members not notified
- **WS envelope events** — `message.update`, `message.delete`, `message.read` not broadcast over WebSocket
- **Wallpaper / theme settings** — `user_settings.wallpaper` field and handler not implemented

### Frontend Gaps
- **App.tsx GatewayEnvelope WS routing** — WS reconnection logic and global search dropdown not wired
- **api/client.ts** — `searchAPI`, `toggleFavourite`, `clearConversation`, `muteConversation` missing
- **ChatWorkspace.tsx** — fake typing timer not replaced with real WebSocket `typing` events
- **ChatSidebarList.tsx** — real favourites filter not wired to backend
- **types.ts** — `Favourite`, `SearchResult`, wallpaper types missing

### Conferencing Bugs (Phase D)
- Recording URL: recordings not saved into uploads directory
- `JoinCallSession` participant count bug: `WHERE room = $2` should resolve via session
- `GetActiveParticipants` / `GetCallParticipants`: `left_at` not handled correctly
- `call_participants` partial unique index migration missing
- `useCall.ts`: auto-offer not initiated to existing participants; TURN config guard missing
- `CallOverlay.tsx`: `isHost` derived from `session.hostId` not connected

### Features Not Yet Built
- Polls and voting
- Stickers / GIF support
- Message scheduling
- Location sharing
- Message export / legal hold
- Whiteboards
- Collaborative document editing
- Knowledge Hub / Wiki module
- Communities and discussion forums
- AI meeting assistant / chat assistant
- Translation services
- SMS / push / mobile push notifications
- End-to-end encryption

---

## Database Tables (`statchat` database)

```sql
-- Core messaging
conversations, conversation_members, messages, message_attachments
message_reactions, message_pins, threads
-- Channels
channels, channel_members
-- Presence & settings
user_presence, conversation_mutes, favourite_conversations, user_settings
-- Calls
call_sessions, call_participants, call_recordings
-- Collaboration
tasks, calendar_events, meetings, meeting_recordings
-- Notifications
notifications
```

---

## Integration with Other Modules

All StatGate modules can request StatChat to create a conversation thread for a platform object:

```
POST /api/v1/chat/conversations
{
  "type": "object",
  "object_ref": "obj:pms:project:<id>",
  "name": "Project Alpha Discussion",
  "members": ["user-uuid-1", "user-uuid-2"]
}
```

This is how the platform achieves threaded, searchable discussion on datasets, projects, research, surveys, and incidents without each module building its own messaging system.

---

## Acceptance Criteria

- [x] Direct messaging operational
- [x] Group chat operational
- [x] Channels operational
- [x] Tasks operational
- [x] Calendar operational
- [x] File sharing operational
- [x] Presence real-time operational
- [x] Typing indicators operational
- [x] Favourites and mute operational
- [x] Global search operational
- [x] Voice/video call sessions operational (basic)
- [x] Registry user directory sync operational
- [ ] Message notifications working (all members notified)
- [ ] WS envelope events for update/delete/read
- [ ] Frontend fully wired to all backend APIs
- [ ] Conferencing bugs resolved (recording, participant count, TURN)
- [ ] Polls operational
- [ ] AI chat assistant operational

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
