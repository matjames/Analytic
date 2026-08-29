# StatCitizen — API Reference Manual

## Version: `v1.0.0`
## Base URI: `/api/statcitizen/v1`

---

## 1. System & Observability

### `GET /health`
Returns service operational status, uptime, and version.
```json
{
  "status": "healthy",
  "app": "statcitizen",
  "version": "1.0.0",
  "uptime": "1h23m4s",
  "timestamp": "2026-08-29T14:10:00Z"
}
```

### `GET /ready`
Returns readiness checks for PostgreSQL database and Redis event bus.

### `GET /metrics`
Exposes Prometheus-formatted metrics (`statcitizen_uptime_seconds`, `statcitizen_db_connected`).

---

## 2. Citizen Identity & Privacy

### `POST /api/statcitizen/v1/session`
Creates an anonymous citizen session. No PII required.
**Response (201 Created):**
```json
{
  "session_id": "cs_1724938210984",
  "token": "tok_32byteSecureRandomToken...",
  "tenant_id": "tenant_default",
  "expires_at": "2026-08-30T14:10:00Z"
}
```

### `POST /api/statcitizen/v1/consent`
Records explicit consent for a data collection interaction.
```json
{
  "session_id": "cs_1724938210984",
  "purpose": "service_problem_report",
  "data_categories": ["report_text", "location", "photos"],
  "visibility": "institution",
  "retention": "1_year",
  "sensitivity": "internal",
  "granted": true
}
```

---

## 3. Reports & Evidence Intake

### `GET /api/statcitizen/v1/reports/categories`
Returns active, configurable reporting categories.

### `POST /api/statcitizen/v1/reports`
Submits a structured problem report.
```json
{
  "category_id": "service_problem",
  "district": "Kampala",
  "title": "Medicine stockout at Katwe HC III",
  "description": "Essential antibiotics and antimalarials out of stock since Monday.",
  "facility_id": "Katwe HC III",
  "priority": "high",
  "anonymous": true,
  "location_consent": true,
  "latitude": 0.3136,
  "longitude": 32.5811,
  "consent_granted": true
}
```
**Response (201 Created):**
```json
{
  "id": "rpt_1724938299",
  "canonical_id": "tenant_default:statcitizen:report:rpt_1724938299",
  "status": "submitted",
  "correlation_id": "sc_1724938299102",
  "message": "Your report has been received. You can track its status using your correlation ID."
}
```

### `POST /api/statcitizen/v1/reports/:id/attachments`
Uploads evidence photos or documents (multipart/form-data, max 10MB).

---

## 4. Multi-Dimensional Service Ratings

### `GET /api/statcitizen/v1/ratings/dimensions`
Returns the 7 active configurable evaluation dimensions.

### `POST /api/statcitizen/v1/ratings`
Submits a multi-dimension service rating.
```json
{
  "service_id": "srv_health",
  "service_name": "Public Health Care",
  "district": "Kampala",
  "ratings": {
    "satisfaction": 5,
    "accessibility": 4,
    "wait_time": 3,
    "availability": 5,
    "staff": 5,
    "quality": 4,
    "outcome": 5
  },
  "comment": "Doctor was attentive and prompt once in consultation.",
  "anonymous": true,
  "consent_granted": true
}
```

---

## 5. Public Consultations & Surveys

### `GET /api/statcitizen/v1/consultations`
Lists published, active public consultations.

### `GET /api/statcitizen/v1/consultations/:id`
Fetches a specific consultation including structured questions.

### `POST /api/statcitizen/v1/consultations/:id/responses`
Submits citizen feedback/answers to an active consultation.

---

## 6. Closed-Loop Case Tracking

### `GET /api/statcitizen/v1/cases/track/:correlation_id`
Retrieves live institutional progress, step completions, and official responses using the tracking correlation ID.
```json
{
  "id": "rpt_1724938299",
  "tracking_type": "report",
  "title": "Medicine stockout at Katwe HC III",
  "status": "investigating",
  "correlation_id": "sc_1724938299102",
  "lifecycle_steps": [
    { "step": "submitted", "label": "Report Submitted", "completed": "true" },
    { "step": "received", "label": "Received by Institution", "completed": "true" },
    { "step": "investigating", "label": "Under Investigation", "completed": "true" },
    { "step": "resolved", "label": "Resolved & Outcome Provided", "completed": "false" }
  ]
}
```

---

## 7. Governed Public Knowledge & Citizen AI

### `GET /api/statcitizen/v1/public/publications`
Lists governed public publications filtered by category (`statistics`, `report`, `dataset`, `policy`).

### `POST /api/statcitizen/v1/ai/query`
Processes natural language queries against approved public evidence.
```json
{
  "query": "What did the recent public health survey find in Wakiso?"
}
```
**Response (200 OK):**
```json
{
  "query": "What did the recent public health survey find in Wakiso?",
  "answer": "According to official publication 'National Health Service Delivery Indicators — Q2 2026' published by Ministry of Health Statistics Division: Aggregated clinical attendance, immunization rates, and pharmaceutical availability across 146 districts.",
  "evidence_grounded": true,
  "confidence": 0.92,
  "citations": [
    {
      "title": "National Health Service Delivery Indicators — Q2 2026",
      "publisher": "Ministry of Health Statistics Division",
      "category": "statistics",
      "published_at": "2026-08-15",
      "canonical_id": "tenant_default:statcitizen:publication:pub_101"
    }
  ],
  "disclaimer": "This response is grounded strictly in governed public records.",
  "insufficient_evidence": false,
  "timestamp": "2026-08-29T14:15:00Z"
}
```

---

## 8. Universal Object Context

### `GET /api/statcitizen/v1/universal/context/:canonical_id`
Generates the standard enterprise 8-facet universal context:
`Overview`, `Activity`, `Relationships`, `Documents`, `Workflow`, `Decisions`, `AI Intelligence`, `Audit`.
