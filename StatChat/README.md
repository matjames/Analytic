# StatChat

StatChat is the enterprise communication and collaboration platform for StatGate.

## Initial MVP

- Direct messages
- Teams and channels
- Authentication and role-aware access
- Real-time messaging
- Basic message persistence

## Structure

- `backend/`
  - `cmd/server/` - backend server entry point
  - `pkg/api/` - REST and WebSocket route handlers
  - `pkg/store/` - in-memory data store and real-time client management
  - `pkg/model/` - shared domain models
- `frontend/`
  - `src/api/` - frontend API clients and service helpers
  - `src/components/` - UI components
  - `src/types.ts` - shared TypeScript models
- `STATCHAT_ENTERPRISE_COLLABORATION_PLATFORM.md` - product directive

## Run locally

Backend:

```bash
cd backend
go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

## Notes

- Auth is currently placeholder UI; actual authentication is expected to be handled by the StatGate launcher.
- Real-time messaging is powered by WebSockets and the `/ws` endpoint.
- Messages are stored in memory for Milestone 1 proof of concept.

## Environment

The frontend reads Vite environment variables from `frontend/.env`:

- `VITE_API_BASE_URL` - backend API base URL
- `VITE_WS_URL` - WebSocket endpoint URL

The backend reads `BACKEND_PORT` from the environment and defaults to `4000` if unset.
