# StatGate Enterprise Event Catalog & Consumption Topology

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Event Specification & Taxonomy  
**Transport:** Redis 7 Pub/Sub (`statgate:events`) plus Redis Streams durable ingress (`statgate:events:stream`) with consumer groups and DLQ (`statgate:events:dlq`, `statgate:events:dlq:stream`).

---

## 1. Universal Enterprise Event Schema

All domain events emitted across StatGate conform to the standardized JSON structure:

```json
{
  "event_id": "evt-7a91bf30-4e58-48b0-8f92-913a8ef5a021",
  "event_type": "submission.received",
  "source": "statcollect",
  "object_type": "submission",
  "object_id": "sub-10029",
  "tenant_id": "uganda-national",
  "org_id": "org-ministry-of-health",
  "user_id": "usr-field-agent-44",
  "correlation_id": "corr-8f0a2139",
  "payload": {
    "survey_id": "surv-malaria-prevalence-2026",
    "district_id": "dist-gulu",
    "sample_count": 150,
    "quality_score": 0.98
  },
  "timestamp": "2026-08-15T18:00:00Z",
  "version": "1.0"
}
```

---

## 2. Event Registry & Publication Taxonomy

### 2.1 Registry Domain
- `user.created` — Emitted when a new institutional identity is provisioned.
- `user.updated` — Profile, tenant affiliation, or credential updates.
- `facility.created` — Master facility register additions.
- `facility.updated` — Facility capacity or location modification.
- `organization.created` — Ministry, district, or NGO registration.

### 2.2 PMS Domain
- `project.created` — New capital project initiated.
- `project.updated` — Scope, timeline, or allocation revisions.
- `project.milestone_reached` — Milestone completed and validated.
- `project.completed` — Project officially commissioned.
- `project.delayed` — Project flagged as overdue or blocked.

### 2.3 RMS Domain
- `research.created` — Research protocol registered.
- `research.updated` — Methodology or investigator updates.
- `research.stage_changed` — State transition (e.g. Protocol -> IRB Review).
- `research.approved` — Institutional Review Board ethics approval granted.
- `research.published` — Findings published to national repository.

### 2.4 StatCollect Domain
- `survey.created` — Questionnaire instrument designed.
- `survey.published` — Survey deployed to field enumerators.
- `submission.received` — Enumerator data packet synchronized from mobile.
- `submission.validated` — Field submission passed automated logic checks.
- `submission.rejected` — Submission failed validation gates.
- `field_data.submitted` — Batch field dataset synchronized.

### 2.5 HelpDesk Domain
- `ticket.created` — Incident or service request logged.
- `ticket.assigned` — Ticket allocated to support specialist.
- `ticket.updated` — Comment or status change.
- `ticket.escalated` — Escalation to senior administrator.
- `ticket.resolved` — Issue resolved and closed.
- `ticket.sla_breached` — Resolution SLA threshold breached.

### 2.6 StatChat Domain
- `message.created` — Chat message posted in channel or object context.
- `discussion.created` — Contextual discussion thread spawned for object.
- `discussion.updated` — Discussion metadata or participant change.

### 2.7 Analytics Domain
- `dataset.created` — Analytical dataset compiled and published.
- `dataset.updated` — Analytical data refreshed.
- `report.generated` — Executive report compiled.
- `anomaly.detected` — Statistical or quality anomaly identified.

### 2.8 StatGovernance Domain
- `risk.created` — Risk registered in institutional risk matrix.
- `finding.created` — Internal or external audit finding recorded.
- `control.created` — Risk mitigation control established.
- `compliance.updated` — Regulatory compliance score adjusted.

### 2.9 StatSpatial Domain
- `spatial.layer.created` — GIS layer ingested.
- `spatial.feature.updated` — Boundary or polygon coordinates modified.
- `spatial.node.synced` — Federated district or agency data synchronized.

### 2.10 StatData Domain (P37 Data Engineering, P38 Scientific Computing)
- `dataset.imported` — Dataset created or refreshed through the ingestion pipeline.
- `pipeline.completed` — Batch pipeline run finished (payload carries status).
- `model.registered` — ML model registered in the model registry.
- `cdc.table.mutation` / `cdc.event` — Change-data-capture stream mutation.

### 2.11 StatIoT Domain (P27 IoT, P43 Field Ops)
- `telemetry.ingested` — Authenticated device telemetry batch processed. Consumed by: Analytics (index refresh). NOTE: was historically mis-emitted as `dataset.updated`; corrected 2026-08-29.
- `anomaly.detected` — Sensor threshold breach (also published by Analytics).
- `field_data.submitted` — Mobile field form submission synchronized (also StatCollect).

### 2.12 GeoIntel Domain (P44)
- `spatial.layer.created` / `spatial.layer.deleted` — GIS layer lifecycle.
- `tile.registered` / `tile.published` — XYZ tile generated / published.
- `drone.flight.completed` — Drone flight finished.

### 2.13 BPM-Hub Domain (P48)
- `process.started` / `process.completed` — Workflow instance lifecycle.
- `task.created` / `task.completed` — Human work items.
- `case.created` — Case instance opened.
- `automation.triggered` — Automation rule fired.

### 2.14 Learning-CRM Domain (P24/P26)
- `lead.converted` — Lead marked won; emits object link to account.
- `partner.registered` / `service_request.created` — Community/partner lifecycle.
- `enrollment.created` / `course.completed` / `certificate.issued` — LMS progression.

### 2.15 AI-Autonomy Domain (P22/P31)
- `agent.task.created` — Agent task spawned by an event trigger. Consumers: ai-autonomy internal workers.
- `twin.registered` — Digital twin instantiated for an object.
- `graph.entity.registered` — Knowledge-graph entity registered.

### 2.16 Knowledge-Portal Domain (P17/P40/P41)
- `content.published` / `dataset.published` — Public portal content/dataset publication.
- `repository.item.archived` — Repository item archived.
- `subscription.created` — Public subscription registered.
- `feedback.received` — Public feedback submitted.


---

## 3. Event Consumption Matrix

```text
Event: submission.received (StatCollect)
 └── Enterprise Core
      ├── Data Quality Engine (computes validation score)
      ├── Analytics Pipeline (updates real-time collection metrics)
      ├── Anomaly Detector (checks variance thresholds)
      ├── Timeline Generator (adds entry to national timeline)
      ├── Action Centre (creates validation task if anomalies detected)
      └── Notification Dispatcher (notifies survey coordinator)

Event: research.stage_changed (RMS)
 └── Enterprise Core
      ├── Workflow Engine (advances ethics approval state machine)
      ├── Knowledge Graph (links protocol to review committee)
      ├── Timeline Generator (records research stage transition)
      └── Command Centre (updates institutional dashboard progress)
```
