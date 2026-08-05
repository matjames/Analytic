# StatChat Fresh Line-by-Line Audit Report

**Date:** 2026-08-05  
**Scope:** Line-by-line analysis of all backend and frontend source files  
**Method:** Systematic review of every function, handler, and component

---

## Executive Summary

StatChat is an enterprise messaging gateway with a Go backend (PostgreSQL + WebSocket + WebRTC) and React/TypeScript frontend. The core messaging functionality is genuinely implemented with real-time WebSocket messaging, PostgreSQL persistence, JWT authentication, and native WebRTC video/voice calls.

**Overall verdict:** ~80% of advertised features are fully functional. The remaining 20% are either partially implemented, simulated, or missing.

### Recently Fixed Issues
- ✅ JWT auth middleware now extracts user ID from token (`sub`/`user_id` claims)
- ✅ Most REST handlers use `requestUserID(r)` instead of hardcoded `"user-001"`
- ✅ Frontend API client sends `Authorization: Bearer <token>` header via `apiFetch()` wrapper
- ✅ Theme loads from user settings on app startup
- ✅ Voice/video call buttons now functional with `useCall` hook and `CallOverlay` component
- ✅ Threaded replies now pass `parentMessageId` and `threadRootId` via `replyContext`
- ✅ Reactions sync now uses `currentUserId` instead of `currentUserName`
- ✅ Read receipts now mark ALL unread messages, not just the last one
- ✅ `currentUserId` passed to `ChatWorkspace` component
- ✅ New conferencing feature (call sessions, WebRTC signaling relay, recordings)
- ✅ New test for user ID extraction from JWT token

---

## 1. Backend Issues

### 1.1 `store.go` — `GetCurrentUser()` Still Hardcodes "user-001"
**File:** `backend/pkg/store/store.go`  
**Code:** `func GetCurrentUser() (model.User, error) { return GetUserByID("user-001") }`  
**Impact:** `GetCurrentUser()` is still called by `conferencing_handlers.go` in `createCallSessionHandler`, `joinCallSessionHandler`, `getMeetingSessionHandler`, and by `currentUserFallbackName()` in `handlers.go`. Call sessions are always created with "user-001" as the host.  
**Fix:** Replace `store.GetCurrentUser()` with `store.GetUserByID(requestUserID(r))` in all callers.

### 1.2 `store.go` — `GetConversations()` Unread Count Hardcoded to "user-001"
**File:** `backend/pkg/store/store.go`  
**Code:** `WHERE rr.message_id = unread_m.id AND rr.user_id = 'user-001'`  
**Impact:** The unread count in the conversation list is always calculated for "user-001". Other users see incorrect unread counts.  
**Fix:** `GetConversations()` should accept a `userID` parameter and use it in the subquery.

### 1.3 `collaboration.go` — `GetPosts()` `likedByMe` Hardcoded to "user-001"
**File:** `backend/pkg/store/collaboration.go`  
**Code:** `liked, err := IsPostLikedByUser(posts[i].ID, "user-001")`  
**Impact:** The `likedByMe` field on posts is always calculated for "user-001". Other users see incorrect like states.  
**Fix:** `GetPosts()` should accept a `userID` parameter and pass it to `IsPostLikedByUser`.

### 1.4 `conferencing_handlers.go` — Uses `store.GetCurrentUser()` Instead of `requestUserID(r)`
**File:** `backend/pkg/api/conferencing_handlers.go`  
**Lines:** 31, 129, 307  
**Impact:** Call sessions are always created/joined with "user-001" as the host/participant, even when auth is enabled.  
**Fix:** Replace `store.GetCurrentUser()` with `store.GetUserByID(requestUserID(r))`.

### 1.5 `conferencing_handlers.go` — `leaveCallSessionHandler` Hardcodes "user-001" Fallback
**File:** `backend/pkg/api/conferencing_handlers.go`, line 166  
**Code:** `if req.UserID == "" { req.UserID = "user-001" }`  
**Fix:** `if req.UserID == "" { req.UserID = requestUserID(r) }`

### 1.6 `handlers.go` — WebSocket Actions Hardcode "user-001" Fallbacks
**File:** `backend/pkg/api/handlers.go`, lines 757, 773, 782  
**Impact:** WebSocket `join-call`, `leave-call`, and `signal` actions default to "user-001" when userId is not provided.  
**Fix:** Implement WebSocket authentication and use the authenticated user ID.

### 1.7 `handlers.go` — WebSocket Has No Authentication
**File:** `backend/pkg/api/handlers.go`, line 872  
**Impact:** The `authMiddleware` skips all `/ws` paths. Any client can connect, send messages, join calls, and relay WebRTC signals without authentication.  
**Severity:** CRITICAL  
**Fix:** Implement WebSocket authentication via token in query parameter or initial message.

### 1.8 `conferencing.go` — `JoinCallSession` Meeting Participant Count Update Bug
**File:** `backend/pkg/store/conferencing.go`, lines 178-180  
**Code:** `WHERE room = $2`, `sessionID, ""`  
**Impact:** The `WHERE room = $2` with `$2 = ""` (empty string) will never match any meeting row. The meeting participant count is never updated.  
**Fix:** Pass the room name as `$2` instead of an empty string.

### 1.9 `ws_hub.go` — `SignalRelay` Has Dead Code
**File:** `backend/pkg/store/ws_hub.go`, lines 27, 66  
**Code:** `roomKey` is computed but never used (`_ = roomKey`).  
**Fix:** Remove the dead code or use `roomKey` for its intended purpose.

### 1.10 `conferencing_handlers.go` — Recording URL Mismatch
**File:** `backend/pkg/api/conferencing_handlers.go`, line 260  
**Code:** `URL: fmt.Sprintf("/uploads/%s", safeName)`  
**Impact:** Call recordings are saved to the `recordings` directory (line 228) but their URL points to `/uploads/`. Recordings won't be accessible via the URL.  
**Fix:** Save recordings to the `uploads` directory, or add a `/recordings/` file server route.

### 1.11 `feature_handlers.go` — `addPostCommentHandler` Hardcodes "StatChat User" Fallback
**File:** `backend/pkg/api/feature_handlers.go`, line 125  
**Code:** `if req.Author == "" { req.Author = "StatChat User" }`  
**Fix:** `if req.Author == "" { req.Author = requestUserID(r) }` or look up the user's name.

### 1.12 `store.go` — `StoreMessageAttachment` Passes `nil` for Metadata
**File:** `backend/pkg/store/store.go`, line 1176  
**Impact:** The `metadata` column in `message_attachments` is always NULL.  
**Fix:** Accept metadata as a parameter or extract it from the file.

### 1.13 `store.go` — `GetMessageAttachments` Doesn't Query `mime_type`
**File:** `backend/pkg/store/store.go`, line 1183  
**Impact:** The `mime_type` column is not queried. The code sets `attachment.MimeType = attachment.FileType` as a workaround.  
**Fix:** Add `mime_type` to the SELECT query.

### 1.14 `store.go` — `GetMessagesForTenant` N+1 Query Problem
**File:** `backend/pkg/store/store.go`, lines 1116-1140  
**Impact:** For each message, 4 separate queries are executed (attachments, reactions, pinned, read-by). For 50 messages, this is 201 queries.  
**Fix:** Use JOINs or batch queries.

### 1.15 `store.go` — Unread Count Excludes `sender != 'StatChat User'`
**File:** `backend/pkg/store/store.go`, line 1005  
**Code:** `AND unread_m.sender != 'StatChat User'`  
**Impact:** Messages from "StatChat User" are excluded from unread count. Any user named "StatChat User" won't have their messages counted as unread.  
**Fix:** Use the actual user ID instead of the sender name, or add an `is_system` flag.

### 1.16 `handlers.go` — `createMessageHandler` Doesn't Generate Notifications
**File:** `backend/pkg/api/handlers.go`, lines 608-660  
**Impact:** When a new message is created, no notification is generated for conversation participants. `store.CreateNotification()` exists but is never called.  
**Fix:** After `store.StoreMessage(message)`, call `store.CreateNotification()` for each conversation member.

### 1.17 `handlers.go` — No Typing Indicator WebSocket Action
**File:** `backend/pkg/api/handlers.go`, lines 730-800  
**Impact:** The WebSocket handler only supports `join-conversation`, `send-message`, `join-call`, `leave-call`, and `signal` actions. There is no `typing` or `stop-typing` action.  
**Fix:** Add `typing` and `stop-typing` WebSocket actions.

### 1.18 `handlers.go` — No Search API Endpoint
**File:** `backend/pkg/api/handlers.go`, `RegisterRoutes()`  
**Impact:** There is no `/search` route. The advertised "message search" feature has no server-side implementation.  
**Fix:** Add a `searchHandler` that queries messages, users, conversations, etc. by keyword.

### 1.19 `handlers.go` — `buildMessageFromPayload` Doesn't Set `ParentMessageID` or `ThreadRootID`
**File:** `backend/pkg/api/handlers.go`, lines 811-857  
**Impact:** WebSocket messages can't be threaded replies because the builder doesn't extract `parentMessageId` or `threadRootId` from the payload.  
**Fix:** Extract these fields from the payload and include them in the message.

### 1.20 `handlers.go` — `editMessageHandler` Doesn't Broadcast the Update
**File:** `backend/pkg/api/handlers.go`, lines 674-697  
**Impact:** When a message is edited via REST API, the updated message is not broadcast to other clients via WebSocket.  
**Fix:** Call `store.BroadcastMessage(message)` after `store.UpdateMessage(message)`.

### 1.21 `handlers.go` — `deleteMessageHandler` Doesn't Broadcast the Deletion
**File:** `backend/pkg/api/handlers.go`, lines 699-707  
**Impact:** When a message is deleted via REST API, the deletion is not broadcast to other clients.  
**Fix:** Broadcast a deletion event to the conversation room.

### 1.22 `handlers.go` — `uploadAttachmentHandler` Doesn't Set `TenantID`
**File:** `backend/pkg/api/handlers.go`, lines 585-593  
**Impact:** Messages created via file upload don't have a `TenantID` set. They won't be filtered correctly by `GetMessagesForTenant`.  
**Fix:** Extract the tenant ID from the form data or the authenticated user's organization.

---

## 2. Frontend Issues

### 2.1 `client.ts` — No Token-Setting Mechanism
**File:** `frontend/src/api/client.ts`, lines 5-10  
**Impact:** `getStoredToken()` reads the JWT from `localStorage`, but no code in the frontend ever writes to `localStorage.setItem('statchat_token', ...)`. The token will always be empty unless set externally.  
**Fix:** Add a `setAuthToken(token: string)` function and call it during login/bootstrap.

### 2.2 `App.tsx` — WebSocket Message Handler Only Handles `Message` Type
**File:** `frontend/src/App.tsx`, lines 153-161  
**Code:** `const message = JSON.parse(event.data) as Message;`  
**Impact:** The WebSocket message handler assumes all incoming WebSocket messages are chat `Message` objects. Call signals, call state updates, and other event types will be incorrectly added to the messages array.  
**Fix:** Parse the event type from a `GatewayEnvelope` wrapper and route to the appropriate handler.

### 2.3 `App.tsx` — Tenant ID Still Hardcoded
**File:** `frontend/src/App.tsx`, line 140  
**Code:** `const tenantId = 'statgate-uganda';`  
**Fix:** Use `user?.organizationId ?? 'default'` as the tenant ID.

### 2.4 `App.tsx` — Global Search Still Has No Server-Side API
**File:** `frontend/src/App.tsx`, lines 372-380  
**Impact:** The header search input only filters conversations client-side. No search API is called.  
**Fix:** Add a `searchAPI()` function in `client.ts` and call it when the search query changes.

### 2.5 `ChatWorkspace.tsx` — Typing Indicators Still Simulated
**File:** `frontend/src/components/ChatWorkspace.tsx`, lines 189-197  
**Impact:** The typing indicator is still a fake 2-second animation triggered by receiving a message, not a real typing event from the other user.  
**Fix:** Send `typing`/`stop-typing` WebSocket events and display the indicator when receiving these events.

### 2.6 `ChatWorkspace.tsx` — "Clear Chat" Only Deletes Last Message
**File:** `frontend/src/components/ChatWorkspace.tsx`, lines 577-579  
**Impact:** The "Clear chat" menu item only deletes the last message, not all messages.  
**Fix:** Loop through all messages and delete each, or add a backend API to clear all messages.

### 2.7 `ChatWorkspace.tsx` — "Mute Notifications" and "Wallpaper" Are Placeholder Menu Items
**File:** `frontend/src/components/ChatWorkspace.tsx`, lines 570-571  
**Impact:** These menu items only show a toast message, no actual functionality.  
**Fix:** Implement the handlers or remove the menu items.

### 2.8 `ChatSidebarList.tsx` — Presence Fetched Once, No Polling
**File:** `frontend/src/components/ChatSidebarList.tsx`, lines 102-115  
**Impact:** Presence is fetched once on mount with no polling or real-time updates. Online status indicators become stale.  
**Fix:** Poll presence every 30-60 seconds, or receive presence updates via WebSocket.

### 2.9 `ChatSidebarList.tsx` — "Favourites" Filter Hardcoded
**File:** `frontend/src/components/ChatSidebarList.tsx`, lines 153-155  
**Code:** `if (activeFilter === 'Favourites') { return conversation.id === 'general' || conversation.type === 'direct'; }`  
**Fix:** Add a `favourite` flag to conversations and a UI to toggle it.

### 2.10 `CalendarPanel.tsx` — All Meeting Buttons Still Disabled
**File:** `frontend/src/components/CalendarPanel.tsx`  
**Impact:** Despite the backend having call session APIs and the frontend having `useCall` hook and `CallOverlay` component, the CalendarPanel still has all join/schedule/playback buttons disabled with "Unavailable" text. The new conferencing UI is not wired to the CalendarPanel.  
**Fix:** Wire the meeting join buttons to `fetchMeetingCallSession()` API and use the `CallOverlay` component.

### 2.11 `CollaborationPanel.tsx` — Non-Functional Buttons
**File:** `frontend/src/components/CollaborationPanel.tsx`  
**Impact:** The "Share" button on posts, "Photo/Video/Article/Poll" buttons in create post, "Learn More" on opportunities, and "Apply" on jobs all have no `onClick` handlers.  
**Fix:** Implement the handlers or remove the buttons.

### 2.12 `IntegrationPanel.tsx` — Placeholder Views
**File:** `frontend/src/components/IntegrationPanel.tsx`  
**Impact:** The "Projects", "Research", "Tasks", "Contacts", and "Communities" sidebar views still render placeholder panels with "This workspace is under active development" text.  
**Fix:** Implement real components for these views or remove them from the sidebar.

### 2.13 `App.tsx` — No Presence Update on Tab Close/Idle
**File:** `frontend/src/App.tsx`, line 181  
**Impact:** Presence is set to "online" on mount but never updated to "away" or "offline" when the user goes idle or closes the tab.  
**Fix:** Add `beforeunload` event listener and use the Page Visibility API.

### 2.14 `App.tsx` — WebSocket Reconnection Not Implemented
**File:** `frontend/src/App.tsx`, lines 147-161  
**Impact:** When the WebSocket connection drops, there's no reconnection logic.  
**Fix:** Implement exponential backoff reconnection logic.

### 2.15 `App.tsx` — `sendMessage` WebSocket Fallback Doesn't Include Thread Fields
**File:** `frontend/src/App.tsx`, line 267  
**Code:** `socket.send(JSON.stringify({ action: 'send-message', ...payload }));`  
**Impact:** The WebSocket fallback includes `parentMessageId` and `threadRootId` in the payload, but the backend's `buildMessageFromPayload` doesn't extract them (see issue 1.19).  
**Fix:** Fix the backend `buildMessageFromPayload` to extract thread fields.

### 2.16 `useCall.ts` — Hardcoded "user-001" Fallback
**File:** `frontend/src/hooks/useCall.ts`, line 47  
**Code:** `const getUserId = useCallback(() => userRef.current?.id ?? 'user-001', []);`  
**Impact:** If the user ID is not available, the call system defaults to "user-001".  
**Fix:** Require a valid user ID before allowing call operations.

### 2.17 `useCall.ts` — TURN Server Defaults to `turn.example.com`
**File:** `frontend/src/hooks/useCall.ts`, lines 22-25  
**Code:** `import.meta.env.VITE_TURN_URL ?? 'turn.example.com:3478'`  
**Impact:** If TURN environment variables are not set, the TURN server defaults to a non-functional domain. WebRTC connections will fail for users behind symmetric NAT.  
**Fix:** Fail fast if TURN is not configured, or use a known-good public TURN server.

### 2.18 `useCall.ts` — `startCall` Doesn't Call `initiateCall` for Other Participants
**File:** `frontend/src/hooks/useCall.ts`, lines 213-254  
**Impact:** When starting a call, the host joins the call session but doesn't initiate WebRTC offers to other participants. Other participants would need to join and create their own offers. The `initiateCall` function exists but is never called automatically.  
**Fix:** After joining the call, automatically initiate offers to all existing participants.

### 2.19 `useCall.ts` — `joinCall` Doesn't Initiate Offers to Existing Participants
**File:** `frontend/src/hooks/useCall.ts`, lines 256-281  
**Impact:** When joining an existing call, the participant doesn't create offers to existing participants. They wait for incoming offers but nobody sends them (see issue 2.18).  
**Fix:** After joining, automatically initiate offers to all existing participants.

### 2.20 `CallOverlay.tsx` — `isHost` Logic Is Incorrect
**File:** `frontend/src/components/CallOverlay.tsx`, lines 51-53  
**Code:** `const isHost = participants.some((p) => p.role === 'host' && p.userId === participants[0]?.userId);`  
**Impact:** The `isHost` check compares the host's userId to `participants[0]?.userId`, but participants may not be ordered with the host first. The "End call for all" button may not appear for the actual host.  
**Fix:** Check if the current user's ID matches the session's `hostId` field.

### 2.21 `CallOverlay.tsx` — Recording Upload Silently Fails
**File:** `frontend/src/components/CallOverlay.tsx`, lines 98-102  
**Code:** `try { await uploadCallRecording(session.id, formData); } catch { // ignore }`  
**Impact:** If recording upload fails, the error is silently ignored. The user has no indication that the recording was not saved.  
**Fix:** Show an error message to the user.

---

## 3. Security Issues

### 3.1 WebSocket Connections Are Unauthenticated
**File:** `backend/pkg/api/handlers.go`, line 872  
**Severity:** CRITICAL  
**Impact:** Any client can connect to the WebSocket, send messages, join calls, and relay WebRTC signals without authentication.  
**Fix:** Implement WebSocket authentication via token in query parameter or initial message.

### 3.2 CORS Allows All Origins
**File:** `backend/pkg/api/handlers.go`, line 946  
**Code:** `w.Header().Set("Access-Control-Allow-Origin", "*")`  
**Fix:** Restrict CORS to known origins.

### 3.3 JWT Secret Defaults to "statchat-dev-secret"
**File:** `backend/pkg/api/handlers.go`, line 900  
**Fix:** Fail fast if `STATCHAT_AUTH_REQUIRED=true` and `STATCHAT_JWT_SECRET` is not set.

### 3.4 No Rate Limiting
**Impact:** No rate limiting on any API endpoint.  
**Fix:** Add rate limiting middleware.

### 3.5 No Input Validation on Message Text
**File:** `backend/pkg/api/handlers.go`, line 634  
**Fix:** Add maximum length validation and HTML sanitization.

### 3.6 File Upload Has No Type Restriction
**File:** `backend/pkg/api/handlers.go`, line 533  
**Impact:** Any file type is accepted (including executables). HTML files could be rendered in the browser (stored XSS).  
**Fix:** Validate file types, add `Content-Disposition: attachment` header for non-image files.

### 3.7 Meeting Room Passwords Exposed in API Response
**File:** `backend/pkg/store/collaboration.go`, `GetMeetingRooms()`  
**Fix:** Don't return passwords in the API response.

---

## 4. Data Integrity Issues

### 4.1 `call_participants` UNIQUE Constraint Includes `left_at`
**File:** `backend/pkg/store/conferencing.go`, line 38  
**Code:** `UNIQUE (session_id, user_id, left_at)`  
**Impact:** NULL values are distinct in PostgreSQL, so the same user can join multiple times without leaving.  
**Fix:** Use a partial unique index: `WHERE left_at IS NULL`.

### 4.2 `GetActiveParticipants` Returns `joined_at` as `LeftAt`
**File:** `backend/pkg/store/conferencing.go`, line 213  
**Code:** `SELECT id, session_id, user_id, user_name, role, joined_at, joined_at FROM call_participants`  
**Impact:** The query selects `joined_at` twice. Active participants will have `LeftAt` set to their `JoinedAt` time.  
**Fix:** Select `NULL` for `LeftAt`.

### 4.3 `GetCallParticipants` Uses `COALESCE(left_at, joined_at)` for `LeftAt`
**File:** `backend/pkg/store/conferencing.go`, line 193  
**Impact:** For participants who haven't left, `LeftAt` is set to `JoinedAt` instead of being zero/empty.  
**Fix:** Select `left_at` directly and handle NULL in the Go code.

### 4.4 `conversations` Table Has No `tenant_id` Column
**File:** `backend/pkg/store/store.go`, `ensureSchema()`  
**Fix:** Add `tenant_id TEXT NOT NULL DEFAULT 'default'` column.

### 4.5 `messages` Table `sender` Column Stores Name, Not ID
**File:** `backend/pkg/store/store.go`  
**Fix:** Add a `sender_id` column that stores the user ID.

---

## 5. Performance Issues

### 5.1 N+1 Query in `GetMessagesForTenant`
**File:** `backend/pkg/store/store.go`, lines 1116-1140  
**Fix:** Use JOINs or batch queries.

### 5.2 N+1 Query in `GetPosts`
**File:** `backend/pkg/store/collaboration.go`, lines 300-313  
**Fix:** Use JOINs or batch queries.

### 5.3 No Database Connection Pooling Configuration
**File:** `backend/pkg/store/store.go`, line 47  
**Fix:** Configure `db.SetMaxOpenConns()`, `db.SetMaxIdleConns()`, and `db.SetConnMaxLifetime()`.

### 5.4 No Database Indexes on Foreign Keys
**File:** `backend/pkg/store/store.go`, `ensureSchema()`  
**Fix:** Add indexes on all foreign key columns.

---

## 6. Test Coverage Issues

### 6.1 No Tests for Store Functions
**Fix:** Add unit tests for all store functions.

### 6.2 No Tests for Conferencing Handlers
**Fix:** Add tests for call session creation, joining, leaving, ending, and recording upload.

### 6.3 No Integration Tests
**Fix:** Add integration tests with a test HTTP server and WebSocket client.

### 6.4 No Frontend Tests
**Fix:** Add tests using Jest/React Testing Library.

### 6.5 Auth Test Only Tests `sub` Claim
**File:** `backend/pkg/api/handlers_test.go`, line 63  
**Fix:** Add tests for `user_id` claim fallback and missing-claims rejection.

---

## 7. Summary of Issues by Severity

### Critical (Security/Data Loss) — 7 issues
1. WebSocket connections are unauthenticated (3.1)
2. JWT secret defaults to known value (3.3)
3. `GetConversations()` unread count hardcoded to "user-001" (1.2)
4. `GetPosts()` `likedByMe` hardcoded to "user-001" (1.3)
5. Conferencing handlers use `GetCurrentUser()` instead of `requestUserID(r)` (1.4)
6. Meeting room passwords exposed in API (3.7)
7. File upload has no type restriction — stored XSS risk (3.6)

### High (Functional Bugs) — 9 issues
8. `JoinCallSession` meeting participant count update uses empty string (1.8)
9. Recording URL points to `/uploads/` but file saved to `recordings/` dir (1.10)
10. `GetActiveParticipants` returns `joined_at` as `LeftAt` (4.2)
11. `call_participants` UNIQUE constraint includes `left_at` (4.1)
12. `editMessageHandler` doesn't broadcast update (1.20)
13. `deleteMessageHandler` doesn't broadcast deletion (1.21)
14. WebSocket message handler only handles `Message` type (2.2)
15. No notification generation on new messages (1.16)
16. `useCall` doesn't auto-initiate offers to existing participants (2.18, 2.19)

### Medium (Missing Features/Incomplete) — 10 issues
17. No typing indicator WebSocket events (1.17)
18. No server-side search API (1.18)
19. `buildMessageFromPayload` doesn't set `ParentMessageID` (1.19)
20. `uploadAttachmentHandler` doesn't set `TenantID` (1.22)
21. CalendarPanel buttons still disabled despite call API (2.10)
22. No token-setting mechanism in frontend (2.1)
23. No rate limiting (3.4)
24. No input validation on message text (3.5)
25. No WebSocket reconnection logic (2.14)
26. TURN server defaults to non-functional domain (2.17)

### Low (Code Quality/Performance) — 14 issues
27. `SignalRelay` has dead code (1.9)
28. `StoreMessageAttachment` passes `nil` for metadata (1.12)
29. `GetMessageAttachments` doesn't query `mime_type` (1.13)
30. N+1 query in `GetMessagesForTenant` (1.14)
31. N+1 query in `GetPosts` (5.2)
32. No database connection pooling configuration (5.3)
33. No indexes on foreign keys (5.4)
34. `conversations` table has no `tenant_id` column (4.4)
35. `messages.sender` stores name, not ID (4.5)
36. `addPostCommentHandler` hardcodes "StatChat User" fallback (1.11)
37. Presence fetched once, no polling (2.8)
38. "Favourites" filter hardcoded (2.9)
39. No presence update on tab close/idle (2.13)
40. Placeholder views for Projects/Research/Tasks/Contacts/Communities (2.12)