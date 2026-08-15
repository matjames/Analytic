# StatGate Universal Object Identity & Data Fabric Catalog

**Document Version:** 1.0 (Phase y Convergence)  
**Status:** Authoritative Enterprise Data Fabric Specification  
**Directives:** Section 9 Universal Object Identity Standard, Phase IX Object Resolver Standard

---

## 1. Universal Object Identity Format

To ensure complete interoperability across disconnected databases, every institutional entity in StatGate must have a globally addressable Universal Identifier (URN):

```text
tenant:application:object_type:object_id
```

### Identifier Grammar:
- `tenant`: The multi-tenant partition key (e.g. `uganda-national`, `district-gulu`).
- `application`: Canonical source application code (`registry`, `pms`, `rms`, `statcollect`, `helpdesk`, `statchat`, `statgovernance`, `statspatial`, `core`).
- `object_type`: Entity model classification (`user`, `facility`, `project`, `milestone`, `research`, `survey`, `submission`, `ticket`, `discussion`, `risk`, `layer`).
- `object_id`: Natural unique primary key within the domain application database.

### Examples:
- `uganda-national:pms:project:proj-10492`
- `uganda-national:rms:research:res-8831`
- `uganda-national:statcollect:survey:surv-501`
- `uganda-national:statcollect:submission:sub-9901`
- `uganda-national:helpdesk:ticket:tkt-402`
- `uganda-national:statchat:discussion:disc-proj-10492`
- `uganda-national:statspatial:layer:layer-health-clinics`

---

## 2. Universal Object Metadata Schema

Every resolved enterprise object adheres to the standard fabric descriptor:

```json
{
  "urn": "uganda-national:pms:project:proj-10492",
  "tenant_id": "uganda-national",
  "source_app": "pms",
  "object_type": "project",
  "object_id": "proj-10492",
  "title": "Northern Corridor Cold Chain Infrastructure",
  "description": "Construction of vaccine refrigeration facilities across 8 districts.",
  "owner_id": "usr-project-director-1",
  "organization_id": "org-ministry-of-health",
  "created_by": "usr-project-director-1",
  "updated_by": "usr-lead-engineer-4",
  "created_at": "2026-01-15T09:30:00Z",
  "updated_at": "2026-08-15T16:45:00Z",
  "classification": "official_sensitive",
  "sensitivity_level": "medium",
  "workflow_state": "in_progress",
  "relationships": [
    {
      "relation": "has_discussion",
      "target_urn": "uganda-national:statchat:discussion:disc-proj-10492"
    },
    {
      "relation": "monitored_by_survey",
      "target_urn": "uganda-national:statcollect:survey:surv-501"
    },
    {
      "relation": "mapped_to_layer",
      "target_urn": "uganda-national:statspatial:layer:layer-health-clinics"
    }
  ],
  "links": {
    "deep_link": "http://localhost:3010/projects/proj-10492",
    "command_centre_link": "http://localhost:3000/fabric/explore?urn=uganda-national:pms:project:proj-10492"
  }
}
```

---

## 3. Interoperability & Knowledge Graph Integration

The Universal Object Resolver in Enterprise Core (`GET /api/v1/fabric/resolve?urn=...`) executes decentralized federated queries across application endpoints to assemble full entity graphs without monolithic table merges.
