# StatGate Authoritative Database Catalog

See full catalog specification and convergence roadmap in [docs/DATABASE_CATALOG.md](file:///c:/Users/PC/Desktop/Analytic/docs/DATABASE_CATALOG.md).

## Classification Summary
- **ENTERPRISE-OWNED:** `enterprise_events`, `enterprise_audit_log`, `workflows`, `workflow_instances`, `workflow_tasks`, `decisions`, `action_items`, `evidence_records`, `timeline_entries`, `knowledge_nodes`, `knowledge_edges`, `analytics_datasets`, `analytics_kpis`, `analytics_alerts`, `ai_investigations`.
- **DOMAIN-OWNED:** Registry (`users`, `tenants`, `organizations`, `facilities`), PMS (`projects`, `milestones`, `budgets`), RMS (`research_projects`, `ethics_reviews`, `publications`), StatCollect (`surveys`, `submissions`), HelpDesk (`tickets`, `ticket_comments`), StatChat (`channels`, `discussions`, `messages`), StatSpatial (`admin_units`, `geo_layers`, `geo_features`, `federated_nodes`).
- **MIGRATION REQUIRED (File storage & embedded workflows):** Local disk attachments -> MinIO S3; local workflow state machines -> Enterprise Core Workflows.
