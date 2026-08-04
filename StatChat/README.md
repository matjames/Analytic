# StatChat

StatChat is the **enterprise communication and collaboration platform** for StatGate — the official collaboration backbone of the StatGate system. It is a fully functional messaging gateway (not a demo), deeply integrated with PostgreSQL persistence, real-time WebSockets, JWT authentication, and the broader StatGate module ecosystem.

## Features

### Core Messaging
- Direct messages (1:1)
- Teams and channels (public/private)
- Group chat with member management
- Real-time messaging over WebSockets (`/ws`)
- Message edit, delete (soft), and soft-delete recovery
- Message reactions (add/remove)
- Threaded replies and reply-to
- Read receipts and delivery status
- Typing indicators
- Message pinning
- Message search (header + per-conversation)
- Presence (online/busy/away/offline)

### Collaboration & Productivity
- File / attachment upload with in-chat preview (audio, video, image, documents)
- Real voice-note recording via the MediaRecorder API
- Tasks (create from conversation, assign, priority, status)
- Notifications (in-app, mark-read / read-all)
- Meetings (schedule, rooms, recordings)
- Collaboration feed (posts, connections, opportunities, jobs)
- Wellness feed and Knowledge hub (experts, articles, ideas, posts)
- Dark / light theme
- Responsive, mobile-friendly layout

### Integration & Security
- JWT bearer-token auth middleware (`STATCHAT_AUTH_REQUIRED`)
- Multi-tenant support (`tenantId`) with conversation routing
- PostgreSQL persistence with auto-migration and seed data
- CORS enabled
- StatGate launcher integration (no local login required)

## Architecture

```
StatChat/
├── backend/                 # Go service (API + WebSocket + PostgreSQL)
│   ├── cmd/server/          # entry point
│   └── pkg/
│       ├── api/             # REST handlers, WebSocket hub, auth, CORS
│       ├── store/           # PostgreSQL store, schema, seed, real-time clients
│       ├── model/           # shared domain models
│       └── types/           # shared types
├── frontend/                # React + TypeScript (Vite)
│   └── src/
│       ├── api/             # API clients
│       ├── components/      # UI components
│       └── types.ts         # shared TypeScript models
├── docker-compose.yml       # PostgreSQL 16 backing store
└── STATCHAT_ENTERPRISE_COLLABORATION_PLATFORM.md   # product directive
```

## Run locally

### 1. Start PostgreSQL

```bash
cd StatChat
docker compose up -d
```

### 2. Configure the backend

```bash
cd StatChat/backend
cp .env.example .env   # adjust as needed
```

### 3. Start the backend

```bash
cd StatChat/backend
go run ./cmd/server
```

The backend listens on `:4000` by default (`BACKEND_PORT`).

### 4. Start the frontend

```bash
cd StatChat/frontend
npm install
npm run dev
```

The frontend runs on `http://localhost:3009` and proxies `/api`, `/uploads`, and `/ws` to the backend.

### Quick start (Windows)

```powershell
cd StatChat
.\start-all.ps1     # launches backend + frontend in separate windows
```

## Database tables

The backend auto-creates and seeds the following tables on first run:

`users`, `user_settings`, `channels`, `conversations`, `messages`, `message_reactions`, `message_attachments`, `pinned_messages`, `read_receipts`, `tasks`, `notifications`, `user_presence`, plus collaboration tables (posts, connections, opportunities, jobs, meetings, wellness, knowledge).

## Environment variables

### Backend (`backend/.env`)
- `DATABASE_URL` — full Postgres DSN (takes precedence)
- `STATCHAT_DB_HOST` / `STATCHAT_DB_PORT` / `STATCHAT_DB_USER` / `STATCHAT_DB_PASSWORD` / `STATCHAT_DB_NAME` / `STATCHAT_DB_SSLMODE`
- `BACKEND_PORT` — HTTP listen port (default `4000`)
- `STATCHAT_AUTH_REQUIRED` — set `true` to enforce JWT auth
- `STATCHAT_JWT_SECRET` — JWT signing secret
- `STATCHAT_UPLOAD_DIR` — upload directory (default `uploads`)

### Frontend (`frontend/.env`)
- `VITE_API_BASE_URL` — backend API base URL (default `/api`)
- `VITE_WS_URL` — WebSocket endpoint (default derived from host)

## Verification

```bash
cd StatChat/backend
go build ./...    # passes
go vet ./...      # passes
go test ./...     # passes

cd StatChat/frontend
npm run build     # passes
