# Phase VII Operationalization Increment

## Enterprise Workspace

The App Launcher workspace now includes an **Enterprise Intelligence** panel. It submits requests only to Enterprise Core (`/api/ai/v1/query`), displays the returned evidence state and source records, presents the personal daily briefing, and offers **Create investigation** only after the user sees grounded evidence. It is not a separate AI application.

The panel accepts optional `application`, `object_type`, `object_id`, and `project_id` context. Application UIs can reuse this panel or call the same governed API; no domain application needs its own intelligence engine. Evidence sources include a configured deep-link to the originating application when one is available.

## Investigation APIs

All routes require `X-User-ID` and are tenant-scoped through Enterprise Core.

| API | Purpose |
|---|---|
| `GET /api/ai/v1/investigations` | Lists the current user's investigations; administrators may request a specific owner. |
| `POST /api/ai/v1/investigations` | Creates an investigation from a user-reviewed AI response or accepted recommendation. Creates a linked Enterprise Task and notification. |
| `GET /api/ai/v1/investigations/:id` | Returns an authorized investigation and its evidence/action references. |
| `PUT /api/ai/v1/investigations/:id` | Updates governed status, outcome, or linked Decision Record. |
| `POST /api/ai/v1/investigations/:id/decisions` | Explicit human action that creates a Phase V Decision Record and links it to the investigation. |
| `GET /api/ai/v1/briefings` | Returns a personal briefing from the user's tasks, approvals, and analytical alerts. |

## Events and audit chain

Creating an investigation emits `ai.investigation.created` with the investigation ID as its correlation ID. It also creates an Enterprise Task, notification, timeline entry, knowledge relationships, and audit record. This preserves the chain from AI response to human-controlled action without allowing AI to execute a consequential action on its own.

## Durable investigation state

`enterprise_ai_investigations` is an Enterprise Core PostgreSQL table provisioned by `docker/postgres-init/07-create-enterprise-ai.sql`. Core hydrates investigations at startup and upserts lifecycle changes to this table. Redis continues to provide event/caching support only and is not the authoritative investigation store.

## Status model

Investigations use `open`, `in_progress`, `awaiting_evidence`, `awaiting_decision`, `action_in_progress`, `monitoring`, `resolved`, and `closed`. Decisions continue to use the existing Phase V Decision Record lifecycle and tasks use the existing workflow/task services.

The lifecycle also recognizes `investigating`, `findings_ready`, `action_required`, `verifying`, and `dismissed` so the investigation state remains distinct from a recommendation or an organizational decision. Investigation updates emit `investigation.updated`; user-created decisions emit `decision.created`, both with the investigation correlation ID.
