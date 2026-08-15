# StatGate Database Catalog & Convergence Strategy

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Database Schema Classification  
**Directives:** Section 18 Database Convergence, Zero Destructive Refactoring without Evidence

---

## 1. Executive Summary & Governance Policy

StatGate utilizes a multi-database architecture partitioned by domain responsibilities with an authoritative Enterprise Core database. Destructive merging of operational schemas is prohibited. Every database and table is classified below with its domain authority and target migration status.

---

## 2. Comprehensive Database & Table Catalog

### 2.1 Database: `statgate_core` (Enterprise Core Backbone)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `enterprise_events` | ENTERPRISE-OWNED | Authoritative durable store for incoming domain events across all apps. | Retain as permanent system of record. |
| `enterprise_audit_log` | ENTERPRISE-OWNED | Immutable audit history recording who, what, when, where, tenant, states. | Retain as permanent compliance record. |
| `workflows` | ENTERPRISE-OWNED | Declarative cross-application workflow state machines. | Target engine for all application workflows. |
| `workflow_instances` | ENTERPRISE-OWNED | Active executions of state machines with token positions. | Authoritative execution store. |
| `workflow_tasks` | ENTERPRISE-OWNED | User assignment tasks generated across workflows. | Unified task list for Command Centre. |
| `decisions` | ENTERPRISE-OWNED | Human-approved institutional decision records. | Authoritative institutional governance. |
| `action_items` | ENTERPRISE-OWNED | Post-decision and anomaly action trackers. | Enterprise action center. |
| `evidence_records` | ENTERPRISE-OWNED | Traceable supporting files, datasets, and reports for decisions. | Immutable evidentiary store. |
| `timeline_entries` | ENTERPRISE-OWNED | Unified temporal activity log fanned out from events. | Single source for Command Centre timeline. |
| `knowledge_nodes` | ENTERPRISE-OWNED | Institutional knowledge graph entities. | Phase IX / XII graph engine. |
| `knowledge_edges` | ENTERPRISE-OWNED | Relationships linking objects, owners, and decisions. | Interoperability fabric. |
| `analytics_datasets` | ENTERPRISE-OWNED | Catalog of verified analytical datasets. | Enterprise analytics metadata. |
| `analytics_kpis` | ENTERPRISE-OWNED | Authoritative institutional metrics and formulas. | Real-data executive KPIs. |
| `analytics_alerts` | ENTERPRISE-OWNED | Automated threshold and statistical anomaly detections. | Real-time monitoring feed. |
| `ai_investigations` | ENTERPRISE-OWNED | Evidence-backed AI analysis sessions with human checkpoints. | Governed AI pipeline. |

---

### 2.2 Database: `statgate_registry` (Identity & Directory)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `users` | DOMAIN-OWNED | Authoritative identity registry for credentials and profile. | Sole source of user identities across platform. |
| `tenants` | DOMAIN-OWNED | Multi-tenant organization boundaries and configurations. | Sovereign tenant directory. |
| `organizations` | DOMAIN-OWNED | Ministries, agencies, NGOs, district local governments. | Master institutional hierarchy. |
| `facilities` | DOMAIN-OWNED | Master facility list (health centers, schools, offices). | Standard facility repository. |
| `roles` | DOMAIN-OWNED | Platform RBAC role definitions. | Master authorization roles. |
| `user_roles` | DOMAIN-OWNED | Role assignments per tenant and org. | Identity mapping. |
| `local_audit_log` | DUPLICATE / DEPRECATED | Legacy application-level audit entries. | MIGRATION REQUIRED: Stream to `enterprise_audit_log`. |

---

### 2.3 Database: `statgate_pms` (Project Management)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `projects` | DOMAIN-OWNED | Capital projects, objectives, budget allocations. | Domain authority for capital works. |
| `milestones` | DOMAIN-OWNED | Deliverable phases, deadlines, completion percentages. | Domain tracking. |
| `budgets` | DOMAIN-OWNED | Financial allocations and expenditure logs. | Financial domain. |
| `contractors` | DOMAIN-OWNED | Vendor profiles, contracts, performance ratings. | Vendor management. |
| `project_documents` | LEGACY | File references stored on container disk. | MIGRATION REQUIRED: Migrate to MinIO S3 metadata. |
| `pms_notifications` | DUPLICATE / DEPRECATED | Local in-app notifications. | MIGRATION REQUIRED: Route through Enterprise Event Bus. |

---

### 2.4 Database: `statgate_rms` (Research Management)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `research_projects` | DOMAIN-OWNED | Research proposals, abstracts, principal investigators. | Academic research domain. |
| `ethics_reviews` | DOMAIN-OWNED | Institutional Review Board (IRB) ethics approvals. | Ethics authority. |
| `publications` | DOMAIN-OWNED | Papers, datasets, peer-reviewed artifacts. | Dissemination repository. |
| `investigators` | DOMAIN-OWNED | Academic credentials, affiliations. | Researcher directory. |
| `rms_workflows` | DUPLICATE | Custom embedded workflow transition state. | MIGRATION REQUIRED: Delegate to Enterprise Core Workflows. |

---

### 2.5 Database: `statgate_statcollect` (Field Collection & Surveys)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `surveys` | DOMAIN-OWNED | Questionnaires, schemas, version definitions. | Survey schema authority. |
| `submissions` | DOMAIN-OWNED | Raw and validated enumerator submissions. | Field data authority. |
| `field_teams` | DOMAIN-OWNED | Enumerators, supervisors, device identifiers. | Field operations. |
| `validation_rules` | DOMAIN-OWNED | Logic gates for field data validation. | Quality rules. |
| `offline_sync_queue` | DOMAIN-OWNED | Idempotent mobile sync buffer. | Mobile sync engine. |

---

### 2.6 Database: `statgate_helpdesk` (Service Desk & Support)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `tickets` | DOMAIN-OWNED | Incident and request tickets. | HelpDesk authority. |
| `ticket_comments` | DOMAIN-OWNED | Internal agent notes and requester replies. | Ticket thread. |
| `sla_policies` | DOMAIN-OWNED | Resolution time targets and escalation rules. | SLA engine. |
| `helpdesk_files` | LEGACY | Attachments stored in local container paths. | MIGRATION REQUIRED: MinIO S3 integration. |

---

### 2.7 Database: `statgate_statchat` (Collaboration)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `channels` | DOMAIN-OWNED | Public and team chat channels. | Chat domain. |
| `discussions` | DOMAIN-OWNED | Object-context threads (`tenant:app:type:id`). | Universal discussion backbone. |
| `messages` | DOMAIN-OWNED | Chat messages, markdown formatting, reactions. | Real-time messaging. |
| `chat_attachments` | LEGACY | Media attachments. | MIGRATION REQUIRED: MinIO S3 storage client. |

---

### 2.8 Database: `statgate_statspatial` (Geospatial & GIS)
| Table Name | Classification | Description & Authority | Target Roadmap |
|---|---|---|---|
| `admin_units` | DOMAIN-OWNED | Boundary polygons for country, regions, districts. | Spatial boundaries authority. |
| `geo_layers` | DOMAIN-OWNED | Map layers (infrastructure, health, topography). | GIS layers. |
| `geo_features` | DOMAIN-OWNED | GeoJSON geometries and polygon points. | Spatial geometries. |
| `spatial_indices` | DOMAIN-OWNED | Spatial indexing linking facilities to admin units. | Spatial search index. |
| `federated_nodes` | DOMAIN-OWNED | External district and ministry data exchange nodes. | Inter-agency federation. |
| `data_sharing_agreements`| DOMAIN-OWNED | Legal and protocol sharing agreements. | Spatial federation agreements. |
| `sync_logs` | DOMAIN-OWNED | Federation synchronization logs. | Node sync history. |

---

## 3. Convergence Action Plan

1. **Step 1 (Immediate):** Audit and log synchronization. All local audit hooks forward events to `enterprise_audit_log`.
2. **Step 2 (Phase y):** File persistence standardization. All uploaded files stream through `statgate-lib/storage` to MinIO S3 buckets with SHA-256 validation.
3. **Step 3 (Continuous):** Declarative workflow transition. Embedded workflow states in RMS, PMS, and HelpDesk delegate state transitions to Enterprise Core Workflow Engine.
4. **Step 4 (Read-Only Safety):** No tables will be dropped during Phase y. Legacy tables are marked read-only and shadowed by Enterprise Core tables.
