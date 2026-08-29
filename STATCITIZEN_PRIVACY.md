# StatCitizen — Privacy Model & Data Minimization

## 1. Privacy-by-Design Framework

StatCitizen implements the principle of **Data Minimization by Design**:
- Citizens are never forced to register or provide PII for public reporting, feedback, or consultation participation.
- Anonymity is the default option across all submission interfaces.
- Geolocation tracking is strictly opt-in with explicit checkbox consent.

---

## 2. Explicit Consent Architecture

Every citizen interaction creates or references an authoritative `CitizenConsent` entity:
```sql
CREATE TABLE citizen_consents (
    id              TEXT PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    session_id      TEXT,
    citizen_id      TEXT,
    purpose         TEXT NOT NULL,
    data_categories TEXT[] NOT NULL,
    visibility      TEXT NOT NULL,
    retention       TEXT NOT NULL,
    sensitivity     TEXT NOT NULL,
    granted         BOOLEAN NOT NULL,
    granted_at      TIMESTAMPTZ,
    downstream_use  TEXT[]
);
```

---

## 3. Public vs. Internal Data Boundary

```text
               STATGATE REPOSITORY DATA
                          │
            ┌─────────────┴─────────────┐
            │                           │
     INTERNAL DATA                 PUBLIC DATA
  (Investigations, Tasks,     (Governed Publications,
   Confidential Tickets)       Consultations, Stats)
            │                           │
            └─────────────┬─────────────┘
                          │
              GOVERNANCE PUBLICATION GATE
             (StatCitizen Security Boundary)
```

Sensitive citizen reports and internal investigations are strictly isolated from the public query catalog and the Governed Public AI Assistant.
