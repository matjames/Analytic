# StatCitizen — Institutional Integration Map

## 1. Integration Topology

StatCitizen connects directly with the StatGate ecosystem:

| Target System | Protocol / Channel | Purpose |
|---|---|---|
| **Enterprise Core** (`:8096`) | HTTP (`/api/registry/services`) | Live service discovery, heartbeat & health reporting |
| **Enterprise Event Bus** | Redis (`statgate:events`) | Canonical domain event publishing for closed-loop lifecycle |
| **HelpDesk API** (`:5006`) | HTTP REST (`/api/tickets`) | Asynchronous operational ticket generation from citizen reports |
| **Field Operations Registry** (`:9090`) | HTTP REST | Organizational and facility identity reference |
| **StatCollect** (`:5050`) | HTTP REST | Survey definition ingestion and participation bridge |
| **Command Centre** (`:3006`) | Universal Context (`:8097`) | 8-facet object view introspection for institutional operators |

---

## 2. Canonical Enterprise Events

Published to Redis channel `statgate:events`:

- `citizen.registered`
- `citizen.feedback.created`
- `citizen.feedback.resolved`
- `citizen.report.created`
- `citizen.report.assigned`
- `citizen.report.resolved`
- `citizen.consultation.joined`
- `citizen.consultation.response_submitted`
- `citizen.service_rating.created`
- `citizen.case.created`
- `citizen.case.updated`
- `citizen.case.closed`
- `citizen.publication.published`

---

## 3. Universal Object Identity Namespace

Format: `{tenant}:statcitizen:{type}:{id}`

Examples:
- `tenant_default:statcitizen:report:rpt_17249382`
- `tenant_default:statcitizen:feedback:fb_17249383`
- `tenant_default:statcitizen:consultation:cons_health_1`
- `tenant_default:statcitizen:publication:pub_101`
