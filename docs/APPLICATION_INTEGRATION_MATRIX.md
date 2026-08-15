# StatGate Enterprise Application Integration Matrix

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Enterprise Adoption Audit  
**Scope:** Evaluation of all 11 StatGate applications across 16 core enterprise capabilities.

---

## 1. Enterprise Capability Matrix

| Application | Registry Auth | Tenant Context | Permissions (RBAC) | Events Publish | Events Consume | Search Indexing | Notification Sink | Enterprise Timeline | Enterprise Audit | MinIO / S3 Files | Workflow Engine | Enterprise Analytics | Universal Object ID | Knowledge Graph | Command Centre Deep Links | StatChat Discussion | AI Context Pipeline |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **Registry** | Authoritative | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **PMS** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **RMS** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **StatCollect** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **HelpDesk** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **StatChat** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **StatGovernance** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **StatSpatial** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **Analytics Engine** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES |
| **Enterprise Core** | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub | Hub |
| **Command Centre** | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | YES | Gateway | YES | YES |

---

## 2. Detailed Application Integration Profiles

### 2.1 Registry (`stage_register`)
- **Role:** Identity Provider & Trust Anchor.
- **Port:** Backend :8081, Frontend :3006.
- **Identity Scope:** Issues high-entropy HS256 JWT tokens containing `userId`, `tenant_id`, `org_id`, `role`, `email`.
- **Event Outflow:** `user.created`, `user.updated`, `organization.created`, `facility.created`.
- **Object Resolution:** `tenant:registry:user:<id>`, `tenant:registry:org:<id>`, `tenant:registry:facility:<id>`.

### 2.2 PMS (Project Management System)
- **Role:** Capital investment, project milestones, infrastructure delivery tracking.
- **Port:** Backend :8091, Frontend :3010.
- **Event Outflow:** `project.created`, `project.updated`, `project.milestone_reached`, `project.completed`, `project.delayed`.
- **Object Resolution:** `tenant:pms:project:<id>`, `tenant:pms:milestone:<id>`, `tenant:pms:contract:<id>`.
- **StatChat Context:** Direct discussion linking on `/projects/:id/discussion`.

### 2.3 RMS (Research Management System)
- **Role:** Academic protocols, ethics board approvals, publication lifecycles.
- **Port:** Backend :8092, Frontend :3011.
- **Event Outflow:** `research.created`, `research.updated`, `research.stage_changed`, `research.approved`, `research.published`.
- **Object Resolution:** `tenant:rms:research:<id>`, `tenant:rms:protocol:<id>`, `tenant:rms:dataset:<id>`.
- **Workflow Hook:** Automated progression through Ethics Review -> Data Collection -> Peer Review -> Publication.

### 2.4 StatCollect
- **Role:** Field data collection, ODK / offline enumerator mobile sync.
- **Port:** Backend :8082, Frontend :3007.
- **Event Outflow:** `survey.created`, `survey.published`, `submission.received`, `submission.validated`, `submission.rejected`, `field_data.submitted`.
- **Object Resolution:** `tenant:statcollect:survey:<id>`, `tenant:statcollect:submission:<id>`.
- **Real-Time Integration:** Streaming submissions directly trigger Data Quality alerts and Anomaly Detection.

### 2.5 HelpDesk
- **Role:** Institutional ticket resolution, SLAs, stakeholder incident handling.
- **Port:** Backend :8083, Frontend :3008.
- **Event Outflow:** `ticket.created`, `ticket.assigned`, `ticket.updated`, `ticket.escalated`, `ticket.resolved`, `ticket.sla_breached`.
- **Object Resolution:** `tenant:helpdesk:ticket:<id>`.

### 2.6 StatChat
- **Role:** Enterprise communication backbone and universal context collaboration.
- **Port:** Backend :8084, Frontend :3009.
- **Event Outflow:** `message.created`, `discussion.created`, `discussion.updated`.
- **Deep Linking:** Any universal object ID (`tenant:app:type:id`) can open an immediate threaded room.

### 2.7 StatGovernance
- **Role:** Enterprise risk registers, audit findings, control compliance.
- **Port:** Backend :8085, Frontend :3012.
- **Event Outflow:** `risk.created`, `finding.created`, `control.created`, `compliance.updated`.
- **Object Resolution:** `tenant:statgovernance:risk:<id>`, `tenant:statgovernance:control:<id>`.

### 2.8 StatSpatial
- **Role:** Geospatial intelligence, administrative boundaries, federated data node exchange.
- **Port:** Backend :4200.
- **Event Outflow:** `spatial.layer.created`, `spatial.feature.updated`, `spatial.node.synced`.
- **Object Resolution:** `tenant:statspatial:layer:<id>`, `tenant:statspatial:node:<id>`.

### 2.9 Enterprise Core & Search
- **Role:** Central backbone for durable event persistence, workflow execution, timeline generation, knowledge graph, and federated search.
- **Port:** Core :8080, Search :8090.
