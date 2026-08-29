# StatCitizen — Platform Architecture & Design

## Sovereign Citizen Participation, Public Evidence & Institutional Feedback Infrastructure

**Platform:** StatGate Sovereign Intelligence Infrastructure  
**Application:** StatCitizen  
**Status:** Operational Architecture & Edge Application  
**Ports:** API `:8097` | UI `:3015`  
**Universal Object Identity Namespace:** `{tenant}:statcitizen:{type}:{id}`

---

## 1. Executive Summary

StatCitizen establishes a formal, sovereign **Institution ↔ Citizen** interaction layer within the StatGate ecosystem. Rather than serving as an internal enterprise dashboard, StatCitizen operates as an external, highly accessible, mobile-first edge platform through which citizens participate in public consultations, submit structured evidence and service reports, evaluate public services across 7 configurable dimensions, track end-to-end institutional accountability, and consult governed public evidence through a source-aware Citizen AI Assistant.

---

## 2. Core Closed-Loop Lifecycle

```text
                             CITIZEN
                                │
                      [Submit Service Report]
                                │
                                ▼
                       STATCITIZEN EDGE
          (Assigns Canonical ID & Issues Correlation ID)
                                │
                                ▼
               CANONICAL ENTERPRISE EVENT BUS
              (Event: citizen.report.created)
                                │
                                ▼
                         ENTERPRISE CORE
               (Workflow Engine, Tasks, Decisions)
                                │
                ┌───────────────┼───────────────┐
                ▼               ▼               ▼
          HelpDesk API      PMS / RMS      Analytics
        (Ticket Created)   (Projects)     (Warehouse)
                │               │               │
                └───────────────┼───────────────┘
                                │
                                ▼
                      INSTITUTIONAL ACTION
              (Investigation → Decision → Action)
                                │
                                ▼
                        CITIZEN STATUS
              (Closed-Loop Complete Resolution)
```

---

## 3. High-Level System Architecture

```text
                    ┌────────────────────────────┐
                    │          CITIZENS          │
                    └─────────────┬──────────────┘
                                  │
                  HTTPS (Web / Mobile / Offline)
                                  │
                                  ▼
                    ┌────────────────────────────┐
                    │      STATCITIZEN EDGE      │
                    │   (Port 8097 / Port 3015)  │
                    ├────────────────────────────┤
                    │ • Anonymous Session Engine │
                    │ • Consent & Privacy Gate   │
                    │ • 7-Dimension Rating Model │
                    │ • Structured Report Intake │
                    │ • Public Knowledge Gate    │
                    │ • Governed Public AI Engine│
                    │ • Offline Sync Queue       │
                    └─────────────┬──────────────┘
                                  │
         ┌────────────────────────┼────────────────────────┐
         ▼                        ▼                        ▼
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│ PostgreSQL 15   │      │ Redis Event Bus │      │ Enterprise Core │
│ (statcitizen)   │      │(statgate:events)│      │  (Port 8096)    │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

---

## 4. Key Architectural Subsystems

### 4.1. Citizen Identity & Privacy Engine
- **Anonymous Sessions:** Instant ephemeral session (`cs_<timestamp>`) requiring zero personally identifiable information (PII).
- **Data Minimization:** No raw email or phone numbers stored in persistent logs; SHA-256 HMAC cryptographic hashing protects identities.
- **Explicit Consent Tokens:** Every submission references a `consent_id` defining permitted downstream processing, visibility, retention, and sensitivity.

### 4.2. Structured Reporting & Evidence Intake
- Configurable categorization (Service Delivery, Infrastructure, Facilities, Programmes, Community, Emergency).
- Attachment processing with virus scanning status tracking.
- Geolocation tracking strictly guarded by explicit location consent.

### 4.3. 7-Dimension Service Rating
Multi-dimensional evaluation across:
1. Overall Satisfaction
2. Accessibility
3. Waiting Time
4. Service Availability
5. Staff Professionalism
6. Service Quality
7. Intended Outcome Attainment

### 4.4. Closed-Loop Tracking Engine
- Provides instant lookup via Correlation ID (`sc_<timestamp>`).
- Live 5-step progress visualization: `Submitted` → `Received` → `Investigating` → `Action Taken` → `Resolved`.
- Real-time institutional response delivery to the citizen.

### 4.5. Governed Public Knowledge & AI Engine
- Strict separation between public publications and internal institutional data.
- Natural language query answering grounded strictly in verified publications and active consultations.
- Verifiable citations and fallback behavior when evidence is insufficient.

---

## 5. Directory & Package Structure

```text
StatCitizen/
├── backend/
│   ├── ai.go                 # Governed Citizen AI assistant & citations
│   ├── citizen_identity.go   # Identity, consent & core intake handlers
│   ├── config.go             # Configuration loader & validation
│   ├── consultations.go      # Consultations, surveys & rating handlers
│   ├── events.go             # Enterprise Redis event bus integration
│   ├── go.mod                # Go module definition
│   ├── integrations.go       # Fabric registration & HelpDesk bridge
│   ├── main.go               # HTTP engine & lifecycle orchestration
│   ├── middleware.go         # Security, rate limiting & headers
│   ├── models.go             # Authoritative domain entities
│   ├── persistence.go        # PostgreSQL migrations & connection pool
│   ├── routes.go             # Route registrations (/api/statcitizen/v1)
│   ├── statcitizen_test.go   # Automated unit & integration tests
│   └── workers.go            # Offline draft & dead letter sync workers
├── frontend/
│   ├── index.html            # Semantic, accessible HTML5 application
│   ├── styles.css            # Sovereign design system & tokens
│   └── app.js                # Frontend client logic & state machine
└── Dockerfile                # Multi-stage production container
```
