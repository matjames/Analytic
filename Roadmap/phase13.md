# PHASE 13 — GOVERNANCE, COMPLIANCE, RISK MANAGEMENT & ENTERPRISE ADMINISTRATION

## Objective

Develop the enterprise governance framework that ensures StatGate operates securely, transparently, ethically, and in full compliance with institutional, national, and international standards.

This phase establishes the governance backbone responsible for policies, standard operating procedures (SOPs), risk registers, internal/external audits, corrective actions (CAPA), regulatory compliance, committee oversight, delegations of authority, and enterprise-wide administration.

---

## Vision

Governance shall be embedded into every workflow rather than being treated as an afterthought. Every action within StatGate shall be accountable, traceable, reviewable, and compliant.

---

## Services Involved

| Service | Technology | Port |
|---|---|---|
| **StatGovernance Backend** | Go + Gin | :8093 |
| **StatGovernance UI** | React 18 + Vite + TypeScript | :3012 |
| **Enterprise Core Workflow Engine** | Go + Gin | :8096 |

---

## Service: StatGovernance

**Repository:** `StatGovernance/`  
**Backend:** Go + Gin (`StatGovernance/backend/`, :8093)  
**Frontend:** React 18 + Vite + TypeScript (`StatGovernance/frontend/`, :3012)  
**Database:** `statgovernance` (PostgreSQL 15)  
**Auth:** Registry JWT Auth Middleware (`registryAuthMiddleware()`)  
**Redis:** Redis event bus for cross-module governance broadcasts

---

## What Exists ✅

### 1. StatGovernance API Routes (`StatGovernance/backend/routes.go`)

**Policy & SOP Management**
```
GET    /api/policies                           ← list institutional policies
POST   /api/policies                           ← create policy
GET    /api/policies/:id                       ← policy detail
PUT    /api/policies/:id                       ← update policy
POST   /api/policies/:id/approve               ← approve policy workflow
POST   /api/policies/:id/publish               ← publish policy
GET    /api/policies/:id/versions              ← version history
GET    /api/sops                               ← list standard operating procedures
POST   /api/sops                               ← create SOP
```

**Regulatory & Compliance Register**
```
GET    /api/regulations                        ← list applicable laws & regulations
POST   /api/regulations                        ← register regulation
GET    /api/compliance/obligations             ← list compliance obligations
POST   /api/compliance/obligations             ← create compliance obligation
GET    /api/compliance/assessments             ← compliance assessment reviews
POST   /api/compliance/assessments             ← record assessment
```

**Enterprise Risk Management & Controls**
```
GET    /api/risks                              ← enterprise risk register
POST   /api/risks                              ← create risk entry
POST   /api/risks/:id/escalate                 ← escalate risk to board/executive
GET    /api/controls                           ← internal controls library
POST   /api/controls                           ← register internal control
POST   /api/controls/:id/test                  ← record control effectiveness test
```

**Audits, Findings & CAPA**
```
GET    /api/audits                             ← internal & external audit engagements
POST   /api/audits                             ← schedule audit
GET    /api/findings                           ← audit findings & non-conformities
POST   /api/findings                           ← log finding
POST   /api/findings/:id/close                 ← close finding with verification
GET    /api/corrective-actions                 ← Corrective & Preventive Actions (CAPA)
POST   /api/corrective-actions                 ← create CAPA task
```

**Committees, Decisions & Delegations**
```
GET    /api/committees                         ← governance committees (Audit, Ethics, Board)
POST   /api/committees                         ← create committee
GET    /api/meetings                           ← committee meetings & agendas
POST   /api/meetings                           ← schedule meeting
GET    /api/decisions                          ← binding decision register
POST   /api/decisions                          ← record formal decision
GET    /api/delegations                        ← delegation of authority register
POST   /api/delegations                        ← register delegation
```

**Evidence Vault, Privacy & Grounded AI**
```
GET    /api/evidence                           ← immutable evidence vault
POST   /api/evidence                           ← store verified evidence record
GET    /api/data-governance                    ← data governance register
POST   /api/data-governance                    ← record data governance asset
GET    /api/data-governance/privacy            ← data protection & privacy impact assessments
POST   /api/ai/analyze                         ← grounded AI policy/compliance analysis
```

**Cross-Platform Search & Activity Feed**
```
GET    /api/dashboard                          ← governance cockpit summary
GET    /api/search?q=                          ← full-text search across governance records
GET    /api/activity                           ← governance audit activity timeline
```

### 2. Enterprise Workflow Engine (`enterprise/core/workflow_engine.go`)
- Full BPMN-style multi-step approval workflow engine (54KB)
- Step transitions, conditions, timeout escalations, and SLA enforcement (`escalation_sla.go`)
- Incident tracking and lifecycle resolution (`incidents.go`)

### 3. Governed AI Framework (Phase XII)
- Mandatory human-in-the-loop review for AI reasoning requests (`phase12_ai.go`)
- Immutable AI audit logs (`phase12_ai_audit.go`) and AI model lifecycle registry

---

## What is Missing ❌

### 1. Ethics & Integrity Portals
- **Whistleblower & Anonymous Reporting Portal:**
  ```
  POST   /api/governance/whistleblower           ← encrypted anonymous intake
  GET    /api/governance/whistleblower/:ticket   ← encrypted follow-up portal
  ```
- **Conflict of Interest (COI) Registry:** Annual & ad-hoc declaration of interests, automated conflict screening against procurement/grant vendors.

### 2. Board & Corporate Secretarial Management
- Board pack generation (automated single-PDF compilation of meeting agendas, resolutions, and reports)
- Electronic resolution circularization & digital signatures
- Statutory registers (Directors, Beneficial Owners, Shareholdings)

### 3. Master Administration & System Governance
- Centralized Feature Flag Management Console (`GET/PUT /api/admin/features`)
- Centralized System Configuration Console (`GET/PUT /api/admin/parameters`)
- Business Continuity Planning (BCP) & Disaster Recovery (DR) Plan execution UI

---

## Database Tables (`statgovernance` database)

```sql
-- Policies & Compliance
policies, policy_versions, sops, regulations, compliance_obligations, compliance_assessments

-- Risk & Controls
enterprise_risks, internal_controls, control_tests, risk_escalations

-- Audits & CAPA
audit_engagements, audit_findings, corrective_actions, evidence_vault

-- Governance & Oversight
committees, committee_members, committee_meetings, decision_register, delegations_of_authority

-- Data Privacy & Governance
data_assets, privacy_assessments, governance_audit_logs

-- Missing / To Build
whistleblower_reports, conflict_declarations, statutory_registers, feature_flags, system_parameters
```

---

## Integration Points

| Module | Integration Flow |
|---|---|
| **PMS (:8091) & RMS (:8092)** | High-severity risks and non-compliance findings automatically escalate into StatGovernance |
| **Field Registry (:9090)** | Organization hierarchy and staff roles enforce Delegation of Authority approval matrices |
| **StatChat (:4000)** | Secure committee collaboration channels and automated CAPA reminder notifications |
| **Enterprise Core (:8096)** | Executes complex multi-step approval state machines across all modules |

---

## Acceptance Criteria

- [x] Policy and SOP lifecycle (draft, review, approve, publish, version) operational
- [x] Enterprise Risk Register with escalation operational
- [x] Controls library with test tracking operational
- [x] Audit engagements, findings, and CAPA tracking operational
- [x] Committee workspaces, meetings, and formal decision registers operational
- [x] Delegation of authority registry operational
- [x] Immutable evidence vault operational
- [x] Enterprise workflow engine and SLA escalation operational
- [x] Governed AI analysis with human approval and audit logging operational
- [ ] Whistleblower portal (anonymous encrypted intake) operational
- [ ] Conflict of Interest declaration and review registry operational
- [ ] Board pack generator and digital resolution sign-off operational
- [ ] Centralized feature flag and system configuration UI operational

---

## Ports & Services

| Component | Port |
|---|---|
| StatGovernance API (Go) | :8093 |
| StatGovernance UI (React/Vite) | :3012 |
| Enterprise Core Workflow Engine (Go) | :8096 |

---

## Estimated Duration

12 weeks

## Milestone

Enterprise Governance, Compliance & Administration Platform Complete. All organizational operations adhere to auditable, traceable, and legally sound governance frameworks.