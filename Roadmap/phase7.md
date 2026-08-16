# PHASE 7 — DATA MANAGEMENT, DATA GOVERNANCE & KNOWLEDGE INFRASTRUCTURE

## Objective

Develop the complete enterprise data ecosystem for StatGate.

This phase establishes the platform as a trusted source for managing the entire lifecycle of organisational data, metadata, documents, digital assets, institutional knowledge, and evidence.

Every dataset, report, document, image, publication, AI model, statistical output, research artifact, and organisational record is governed through this framework.

---

## Vision

StatGate becomes the Single Source of Truth (SSOT) for organisational knowledge and evidence. The platform supports secure storage, versioning, lineage, governance, sharing, preservation, discovery, and reuse of information.

---

## Services Involved

This phase is implemented across the Enterprise Integration Layer:

| Service | Technology | Port |
|---|---|---|
| Enterprise Core | Go + Gin | :8096 |
| Enterprise Search | Go + Gin | :8095 |

---

## Service: Enterprise Core (Data & Knowledge Layer)

**Repository:** `enterprise/core/`  
**Backend:** Go + Gin (:8096)  
**Database:** `statgate` (PostgreSQL 15, `enterprise` schema)  
**Auth:** Registry JWT middleware (`middleware_security.go`)

### Backend File Structure (relevant to Phase 7)

```
enterprise/core/
├── knowledge.go              ← knowledge articles, categories, tags
├── files.go                  ← universal file service (upload, versioning, download)
├── analytics_data_layer.go   ← enterprise data catalog and dataset federation
├── analytics_lineage.go      ← data lineage tracking
├── analytics_quality.go      ← data quality reporting
├── backups.go                ← backup management
├── resilience_engine.go      ← disaster recovery and resilience
├── recovery_drills.go        ← DR drill execution
├── persistence.go            ← shared persistence utilities
└── phase12_graph*.go         ← institutional knowledge graph (Phase XII)
```

---

## What Exists ✅

### Knowledge Base (`knowledge.go`)

```
GET    /api/knowledge                       ← list knowledge articles
POST   /api/knowledge                       ← create knowledge article
GET    /api/knowledge/:id                   ← article detail
PUT    /api/knowledge/:id                   ← update article
DELETE /api/knowledge/:id                   ← delete article
GET    /api/knowledge/categories            ← categories list
POST   /api/knowledge/categories            ← create category
GET    /api/knowledge/tags                  ← tags list
GET    /api/knowledge/search?q=             ← search knowledge base
GET    /api/knowledge/:id/related           ← related articles
```

### Universal File Service (`files.go`)

```
GET    /api/files                           ← list files (with filters)
GET    /api/files/:id                       ← file metadata
POST   /api/files                           ← upload file
GET    /api/files/:id/download              ← download file
PUT    /api/files/:id                       ← update file metadata
DELETE /api/files/:id                       ← delete file
GET    /api/files/:id/versions              ← file version history
GET    /api/files/project/:id               ← files by project
```

### Data Catalog & Analytics Data Layer (`analytics_data_layer.go`)

```
GET    /api/analytics/records                          ← enterprise data records
GET    /api/analytics/records/:source/:entity/:id      ← specific record by source
GET    /api/analytics/datasets                         ← dataset catalog
GET    /api/analytics/datasets/:id                     ← dataset detail
GET    /api/analytics/freshness                        ← data freshness report
GET    /api/analytics/events                           ← analytics events
```

### Data Lineage (`analytics_lineage.go`)

```
GET    /api/analytics/lineage/:id                      ← lineage graph for entity
```
- Tracks source → transformation → destination for datasets and records

### Data Quality (`analytics_quality.go`)

```
GET    /api/analytics/quality                          ← data quality dashboard report
GET    /api/analytics/quality/reports                  ← quality report list
GET    /api/analytics/quality/issues                   ← active quality issues
```

### Backup Management (`backups.go`)

```
GET    /api/backups                        ← list backups
POST   /api/backups                        ← trigger backup
GET    /api/backups/:id                    ← backup detail
DELETE /api/backups/:id                    ← delete backup
GET    /api/backups/status                 ← backup system status
```

### Knowledge Graph (`phase12_graph*.go`)

PostgreSQL adjacency-list graph with depth-limited BFS:

```
GET    /api/graph/objects                  ← list institutional objects (projections)
POST   /api/graph/objects                  ← register institutional object
GET    /api/graph/edges                    ← list graph edges
POST   /api/graph/edges                    ← create graph relationship
DELETE /api/graph/edges/:id                ← delete graph edge (append/correction only)
GET    /api/graph/queries/neighbours/:id   ← BFS neighbours (max 5 hops)
GET    /api/graph/queries/path/:a/:b       ← shortest path between objects
GET    /api/graph/queries/related/:id      ← related objects by type
```

13 approved relationship types. Tenant isolation enforced on every query. Named parameterised queries only.

### Resilience & Recovery (`resilience_engine.go`, `recovery_drills.go`)

```
GET    /api/resilience/status              ← platform resilience status
GET    /api/resilience/runbooks            ← DR runbooks list
GET    /api/resilience/drills              ← recovery drill history
POST   /api/resilience/drills              ← run a recovery drill
GET    /api/resilience/incidents           ← incident timeline
```

---

## Service: Enterprise Search

**Repository:** `enterprise/search/`  
**Backend:** Go (:8095)  
**Role:** Cross-application full-text search across all StatGate modules

### What Exists ✅
- Service is built and exposes a search API
- Indexes records from Enterprise Core entities

### What is Missing ❌
- **Semantic / vector search** — search is keyword-based; no embedding-based semantic search
- **Per-module index coverage** — not all modules publish records to enterprise search
- **AI-powered result ranking** — not implemented

---

## What is Missing ❌

### Data Management
- **Data Warehouse / Data Lake** — `lakehouse.go` is an in-memory structure; no persistent DWH or data lake
- **Master Data Management (MDM)** — no golden record management across modules
- **Reference Data Management** — no central reference data registry
- **Data Marketplace** — no data sharing marketplace
- **Open Data Portal** — not implemented
- **Microdata Repository** — not implemented

### Knowledge & Discovery
- **Semantic / vector search** — enterprise search is keyword-based
- **Taxonomy Management** — not implemented
- **Ontology Management** — not implemented
- **Glossary Management** — not implemented
- **Document OCR** — not implemented
- **Automatic metadata generation** — not implemented
- **Duplicate detection** — not implemented
- **Knowledge recommendations** — not implemented

### Governance & Retention
- **Data Classification** — no classification levels (public, internal, confidential, restricted)
- **Retention Policies** — no automated retention policy enforcement
- **Data Provenance** — lineage exists but provenance chain not formalised
- **Legal Hold** — not implemented

---

## Database Tables (Enterprise Schema)

```sql
-- Knowledge Base
knowledge_articles, knowledge_categories, knowledge_tags, knowledge_article_tags

-- File Management
files, file_versions, file_permissions

-- Data Catalog
enterprise_datasets, enterprise_records, data_lineage, data_quality_reports, data_quality_issues

-- Knowledge Graph (Phase XII)
institutional_objects, institutional_graph_edges

-- Backup & Recovery
backup_jobs, backup_artifacts, recovery_drills, resilience_incidents

-- Shared
object_links, audit_logs, timeline_entries
```

---

## Integration Points

| Module | Integration |
|---|---|
| All modules | File upload/download via Enterprise Core file service |
| All modules | Activity timeline published to Enterprise Core |
| All modules | Object links registered in `object_links` table |
| Enterprise Search | Indexes records from Enterprise Core; cross-app search |
| Phase XII (Phase 13) | Knowledge graph used for institutional intelligence |

---

## Acceptance Criteria

- [x] Knowledge base (articles, categories, tags) operational
- [x] Universal file service (upload, download, versioning) operational
- [x] Data catalog and enterprise dataset registry operational
- [x] Data lineage tracking operational
- [x] Data quality reporting operational
- [x] Backup management operational
- [x] Resilience engine and recovery drills operational
- [x] Knowledge graph (PostgreSQL adjacency-list, BFS queries) operational
- [x] Enterprise search service operational (keyword)
- [ ] Semantic / vector search operational
- [ ] Data classification (sensitivity levels) operational
- [ ] Retention policy enforcement operational
- [ ] Glossary / taxonomy / ontology management operational
- [ ] Open data portal operational
- [ ] Document OCR operational
- [ ] Automatic metadata generation operational

---

## Ports & Services

| Component | Port |
|---|---|
| Enterprise Core (Go) | :8096 |
| Enterprise Search (Go) | :8095 |

---

## Estimated Duration

10 weeks

## Milestone

Enterprise Data & Knowledge Platform complete. Every dataset, document, and knowledge article is catalogued, versioned, governed, searchable, and linked through the knowledge graph.
