# StatGate Phase VII Technical Implementation Review

## Existing capabilities to reuse

Enterprise Core already supplies the cross-application event bus, enterprise data layer, KPI and anomaly engines, data-quality reports, analytics lineage, workflow engine, tasks, approvals, decision records, notifications, reports, permissions, audit records, and Phase VI knowledge context. PMS, RMS, Registry, StatCollect, HelpDesk and StatChat remain systems of record and are reached through their existing APIs/events.

## Phase VII implementation approach

The AI layer is implemented inside `enterprise/core` as a versioned API and never reads application databases directly. A request resolves an authenticated user, tenant and optional organization/project/geographic scope, checks the existing permission service, and retrieves only matching Enterprise Data Layer records and approved knowledge. Every returned finding retains source application, entity and record identifiers.

The initial increment adds context, source traceability, grounded analysis, anomaly investigation, recommendation creation, feedback, evaluation placeholders, model registry and usage endpoints. Recommendations reuse the existing Phase V recommendation and decision lifecycle; they do not create decisions or actions automatically. Workflow actions continue to use the existing task/approval/decision APIs.

## Persistence and security

Operational AI state is retained through the existing Redis-backed enterprise persistence and audit mechanism; PostgreSQL persistence is the next migration before production retention guarantees. Requests require `X-User-ID`, use the existing `evaluatePermission` policy, constrain records to the request tenant, and do not return hidden context. The model interface is provider-neutral; no provider is called unless configured. When no evidence exists the API returns `insufficient_evidence` rather than inventing a response.

## API contracts

The increment exposes `/api/ai/v1/context`, `/query`, `/analyze`, `/investigate`, `/recommend`, `/sources/:id`, `/feedback`, `/evaluations`, `/models`, `/usage`, and `/briefings`. Inputs and outputs include source references, confidence categories, audit IDs and explicit fact/inference/recommendation separation.

## Validation plan

Automated tests cover an unauthenticated request rejection, tenant-scoped evidence retrieval, insufficient-evidence handling, traceable investigation, and the rule that recommendations require an explicit human workflow/decision action. Later increments will add real-provider contract tests, service failure handling, action authorization, and the full StatCollect-to-outcome end-to-end scenario.
