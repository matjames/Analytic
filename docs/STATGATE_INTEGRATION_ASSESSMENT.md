# StatGate Platform Integration Assessment & Unification Plan

> Scope: verify how far StatGate already behaves as **one system** and identify the
> concrete gaps that still make it present as **separate apps**. This document is a
> read-only assessment — no code was changed to produce it. It is the prerequisite
> for any further app development, per the engineering directive.

Status: **Implementation applied** — Phases A (complete service registry), B (env-driven launcher auth),
C (uniform `statgate_token` SSO across all module UIs), D (platform-wide health at launcher `/api/health`),
and E (launcher consumes the single registry; module app-switchers re-synced to it) were implemented and
validated with production builds. Remaining Phase-E follow-up: adopt the shared design-system chrome
across every module (recommended as a separate, UI-only refactor).

---

## 1. Executive summary

StatGate's **infrastructure and backend are already a single, coherent system**:
one Docker network, one PostgreSQL (per-module databases), one shared Redis bus, one
shared Registry‑issued JWT validated by every backend through `statgate-lib`, cross-module
event publishing, and cross-module object linkage. All of this is launched by a single
`docker-compose.yml`.

However, at the **user‑facing layer the platform still feels like separate apps**. Eight
concrete problems (Section 3) are responsible. Each is individually small; none require
a rewrite of any module. Address them in the priority order in Section 4 and StatGate
will behave as a single, navigable, single-sign-on platform.

---

## 2. What is ALREADY unified (verified)

| Concern | Mechanism | Where |
| --- | --- | --- |
| One network + one compose launch | `statgate-network` bridge; single root `docker-compose.yml` (760 lines) starts every service | `docker-compose.yml` |
| Shared databases | PostgreSQL 15, per-module DBs created by init scripts | `docker/postgres-init/*.sql` |
| Shared cache / event bus | Redis 7; cross-module events on `statgate:events` | `frontend/services/event_bus.py`, compose |
| Shared identity (backend) | Every Go backend (PMS, RMS, Governance, Spatial, StatChat, Registry) validates the same Registry JWT via `statgate-lib/auth`, fail-closed, tenant-isolated | `PMS/backend/middleware.go`, `statgate-lib/auth` |
| Cross-module linkage | shared `object_links` table | `frontend/services/object_links.py` |
| Collaboration backbone | StatChat auto-creates conversations for platform objects (`obj:{type}:{id}`) | `StatChat/`, event bus consumers |
| Live app catalogue (partial) | Launcher fetches `/api/services` from Analytics instead of a hardcoded list | `appluancher/src/pages/api/applications.ts` → `frontend/app.py:1332` |
| Bearer-token API calls | UIs attach `Authorization: Bearer <token>` to their module APIs | each `frontend/src/**` client |

Conclusion: the **plumbing is unified**. The remaining work is about **consistency of the
front-end layer** — single source of service truth, a uniform SSO hand-off, a shared
navigation shell, and a platform-wide status view.

---

## 3. Gaps that make it feel like separate apps (with evidence)

### G1 — No single origin / no unified ingress proxy
Every module is a standalone host:port web app (`:5000`, `:3007`, `:3005`, `:3009`,
`:3010`, `:3011`, `:3012`, `:4200`). There is **no reverse proxy / ingress** that serves
all modules under one domain (e.g. `analytics.<host>`, `pm.<host>`) or under path prefixes.
Because `localStorage` is scoped per origin, a token stored on the launcher origin is not
visible on any module origin. This is the architectural root of the "separate apps" feel.

### G2 — Inconsistent SSO token parameter names
When the launcher launches an app it forwards the token as `?registry_token=<jwt>`
(`appluancher/src/pages/index.tsx:40`). Consumers are inconsistent:

- **StatGovernance** reads `registry_token` (also `registry_jwt`) — **works**.
- **StatChat** reads `statgate_token` **or** `access_token`
  (`StatChat/frontend/src/api/client.ts:47`) — **does NOT match** `registry_token`,
  so SSO silently fails for StatChat.
- **PMS, RMS, Helpdesk, Registry** read neither a URL param nor a shared key — each
  requires a separate sign-in.

### G3 — Token localStorage keys differ across modules
The platform stores the bearer token under different keys per module:
`registry_jwt` (launcher/others), `statchat_token` (StatChat), `registry_jwt`/URL param
(Governance). A single signed-in user is therefore represented by different credentials
per app, undermining "log in once".

### G4 — Insecure `demo_token` fallback (Governance)
`StatGovernance/frontend/src/App.jsx` falls back to a literal `'demo_token'` when no
token is present. In a shared deployment this can produce an unauthenticated session
that the (correctly fail-closed) backend would still reject, or worse, appear to work
against permissive endpoints. Remove the fallback and treat a missing token as signed-out.

### G5 — Hardcoded `localhost` URLs scattered everywhere
The external address of every service is hardcoded in many places instead of derived from
one configuration:
- `appluancher/src/context/AuthContext.tsx:6-7` — Registry URL hardcoded to
  `http://localhost:9090/api`.
- `appluancher/src/utils/mockData.ts` — every app `url`/`healthEndpoint` hardcoded to
  `http://localhost:<port>`.
- `PMS/frontend/.env` — `VITE_PMS_API_URL=http://localhost:8091`.
- `PMS/frontend/src/components/StatGateHeader.jsx:4-14` — duplicate hardcoded app list.
- `frontend/config/services.json` + `frontend/services/service_health.py` fallback —
  the same list duplicated (and internally inconsistent between `localhost` UI URLs and
  Docker `statgate-*` health URLs).

These work **only** for a local Docker host, and they mean the "system" has no single,
deployable address model.

### G6 — Service registry is incomplete
`frontend/config/services.json` lists analytics, core, registry, postgres, redis, minio,
prometheus, grafana, alertmanager, statchat, helpdesk, pms, rms. It **omits**:
StatGovernance UI/API (`:3012`/`8093`), StatSpatial (`:4200`), StatCollect (`:8080`),
Enterprise search/core (`:8095`/`:8096`). Because the launcher renders the live list from
this registry, the "whole system" view is incomplete in the portal.

### G7 — No prominent platform-wide health view
Health endpoints exist per service, and Analytics aggregates them at `/api/services/health`
(`frontend/app.py:1338`), but the launcher's own `/api/health`
(`appluancher/src/pages/api/health.ts`) reports only the *launcher's* uptime. There is no
single "entire platform" status tile in the command centre / launcher home that surfaces
degraded modules.

### G8 — Duplicated, divergent app switchers and chrome
Each module ships its own header with its own hardcoded app switcher (e.g.
`PMS/.../StatGateHeader.jsx`), which duplicates and can drift from the launcher's
`mockData.ts` and the Analytics `services.json`. There is no shared navigation model or
shared UI component source of truth (the `enterprise/design-system` exists but is not used
consistently across the module UIs).

---

## 4. Prioritized unification plan (no changes applied yet)

### Phase A — Single source of truth for the service catalogue
1. Make `frontend/config/services.json` the **authoritative** registry and add the missing
   entries: `governance`, `governance-api`, `statspatial`, `statcollect`, `enterprise`,
   `enterprise-core` — using the exact host ports from `docker-compose.yml`
   (`8093`/`3012`, `4200`, `8080`, `8095`/`8096`).
2. Keep `frontend/services/service_health.py` reading the same file (it already does);
   delete/ignore its duplicate fallback list or source it from the same place.
3. Have the launcher (`appluancher/src/pages/api/applications.ts`) and any module
   switcher consume this single list (optionally served by Analytics `/api/services`),
   removing per-module hardcoded app lists.

### Phase B — Env-driven, deployable addressing
Replace hardcoded `localhost` with configuration for every service address:
- `appluancher/src/context/AuthContext.tsx`: read Registry API URL from
  `NEXT_PUBLIC_REGISTRY_API_URL` (default `http://localhost:9090/api`).
- `appluancher/src/utils/mockData.ts`: serve as a **fallback only**, with URLs from
  `NEXT_PUBLIC_*` placeholders.
- `PMS/frontend/.env` and any other `.env`: keep `VITE_*` but ensure compose injects the
  container service names so builds are portable.
- Centralize base URLs in one shared module (or the `enterprise/design-system`) consumed
  by the launcher and the switchers.

### Phase C — Uniform cross-app single sign-on
Standardize the SSO hand-off so "log in once" actually works across **different origins**:
1. **Adopt one token param name** (recommend `statgate_token`) everywhere the launcher
   navigates, and have **every** module UI read it, store it under its own key, and strip
   it from the URL immediately (pattern already present in StatChat's
   `bootstrapSharedSignOn`) — so it is not retained in history/shared links.
2. Accept `registry_token` as an alias in each module so current launcher behaviour keeps
   working, then migrate the launcher to the canonical name.
3. Align localStorage key names where feasible, or accept per-origin keys but make each
   module's bootstrap consume the hand-off param (origin isolation means the param is the
   only cross-origin channel).
4. Remove the `'demo_token'` fallback in StatGovernance (G4) — missing token ⇒ signed out.
5. Consider a lightweight **token-passer proxy** for the *development* deployment so one
   origin (the launcher) can embed module UIs, collapsing the multi-origin problem without
   a full ingress migration.

### Phase D — Unified system health view
1. Make `appluancher/src/pages/api/health.ts` proxy Analytics `/api/services/health` and
   return an aggregate status (healthy / degraded / down) with per-service detail.
2. Surface it in the launcher home / CommandCentre (a "Platform Status" tile showing each
   module's health), replacing today's launcher-only uptime.

### Phase E — Shared navigation shell
1. Standardize one app-switcher component (sourced from the Phase-A single registry) used
   by every module header, removing `StatGateHeader.jsx`-style hardcoded lists.
2. Adopt `enterprise/design-system` as the shared source for this shell so all UIs render
   the same chrome and app list.

---

## 5. Verification plan (after each phase)

- `cd backend && go test ./...` and `cd StatChat && go test ./...`
  (`StatChat/backend`, `enterprise/core`) — ensure backend suites still pass.
- `docker compose config` — validates the compose file after registry/env edits.
- Build each touched UI: `npm run build` (launcher), `npm run build` (Vite module UIs);
  fix any type/import errors before proceeding.
- Manual SSO walk: sign in once on the launcher, launch → Governance, StatChat, PMS, RMS —
  assert no module requires a second login; assert `?statgate_token` is stripped from the
  address bar.
- Manual health walk: open the Platform Status tile; confirm every service listed in
  Section 2 of this doc reports ok when running; confirm a stopped container shows
  degraded/down.

---

## 6. Risks & notes

- **Do not** turn on `ENABLE_NOTEBOOK_EXECUTION=true`; notebook execution runs submitted
  Python in the Flask process (security).
- Cross-origin SSO via URL query parameter is acceptable for this internal platform but
  only if each module **strips the parameter** after storing the token (StatChat already
  does); avoid cookie-based SSO until TLS + a single domain is provisioned.
- Phases only touch configuration/front-end wiring and health aggregation — they do not
  alter backend schemas or service contracts, so regression risk is low and verification
  is mostly UI/build-level.