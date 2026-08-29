# StatGate — Sovereign Analytical Hub Deployment & Architecture Guide
**Version:** 2.0.0-Enterprise  
**Target Environments:** National Statistical Offices (NSO), Line Ministries, International Analytical Hubs, Research Institutes, Multilateral Development Agencies  
**Compliance Standards:** SDMX-JSON 1.0, UN-FPOS, AU STATAFRIC, IATI 2.03, OGC GeoTIFF, ISO/IEC 27001, GDPR/Sovereignty Fail-Closed

---

## 1. Executive Architecture Overview

StatGate operates as a **Federated Sovereign Analytical Hub**. It allows sovereign nations, ministries, and global institutions to maintain strict data residency while securely participating in international evidence networks.

```
       ┌────────────────────────────────────────────────────────┐
       │              GLOBAL ANALYTICAL PEER NETWORK            │
       │   (UN STATAFRIC, EAC, WHO, AfCFTA, World Bank, AU)     │
       └───────────────────────────▲────────────────────────────┘
                                   │  SDMX-JSON / IATI Push Protocol (Port 8105 / 3016)
                                   │  Mutual TLS + Registry Key Vault
       ┌───────────────────────────▼────────────────────────────┐
       │             STATFEDERATION CYBER-DIPLOMACY             │
       │     (Bilateral DSAs, Quotas, Transboundary Audit)      │
       └───────────────────────────▲────────────────────────────┘
                                   │
 ┌─────────────────────────────────┴─────────────────────────────────┐
 │               STATGATE SOVEREIGN ANALYTICAL HUB                   │
 │                                                                   │
 │  ┌───────────────────────┐  ┌──────────────────────────────────┐  │
 │  │  OFFICIAL STATISTICS  │  │    INTELLIGENCE & LAKEHOUSE      │  │
 │  │  • Sampling Framework │  │    • Institutional Condition XII │  │
 │  │  • Crosstab & SDMX    │  │    • 3-Sigma Anomaly Lakehouse   │  │
 │  │  • Census/Surveys     │  │    • ABAC Security & Governance  │  │
 │  └───────────────────────┘  └──────────────────────────────────┘  │
 │  ┌───────────────────────┐  ┌──────────────────────────────────┐  │
 │  │ REAL-TIME TELEMETRY   │  │    OPEN DATA & KNOWLEDGE         │  │
 │  │  • StatIoT Bridge     │  │    • CKAN 3.0 Compatible API     │  │
 │  │  • GeoIntel (PostGIS) │  │    • IATI 2.03 Development XML   │  │
 │  │  • Spatial Indicators │  │    • Knowledge Portal (No-Auth)  │  │
 │  └───────────────────────┘  └──────────────────────────────────┘  │
 └───────────────────────────────────────────────────────────────────┘
```

---

## 2. Sector Deployment Profiles

StatGate is modularized into **Sector Profiles** so each institution deploys only the compute and storage resources required for their mandate:

| Profile Tag | Target Institution | Core Components Included | Primary Protocol |
|---|---|---|---|
| **`nso`** | National Statistical Offices & Bureaus | Analytics Core, StatCollect, Sampling Designer, Tabulation Engine, Knowledge Portal, StatFederation | SDMX-JSON 1.0, CSV, UN-FPOS |
| **`health`** | Ministries of Health, CDC, Hospitals | Analytics Core, StatIoT Bridge, GeoIntel Spatial, Anomaly Lakehouse, Executive Hub | FHIR, SDMX-Health, Sensor Streams |
| **`development`**| Finance & Planning, Donors, UN | PMS (Projects), RMS (Resources), IATI Exporter, Multilateral Report Generator | IATI 2.03 XML, OECD-DAC CRS |
| **`environment`**| Environment, Water, Forestry | GeoIntel (Sentinel-2, NDVI), StatIoT Water/Air Sensors, Spatial Disaggregation | OGC WMS/WFS, GeoTIFF, Sensor Feeds |
| **`research`** | Universities, Think Tanks | Knowledge Repository, Digital Library, JupyterHub, MLflow, Kaggle ML Connector | DOI/ORCID/ISBN, CKAN 3.0 API |
| **`full`** | National Sovereign Enterprise Gateway | All 15+ sub-systems and federation diplomacy network | All protocols |

---

## 3. Quickstart: 4-Hour Turnkey Deployment

### Step 1: System Prerequisites
- **OS:** Ubuntu 22.04 LTS+, RHEL 9+, Debian 12+, or Windows Server 2022
- **Hardware:** 8 vCPU, 32 GB RAM, 250 GB NVMe SSD
- **Software:** Docker Engine 24.0+ & Docker Compose v2.20+

### Step 2: Automated Bootstrap via Setup Script
Run the automated configuration script to generate cryptographically signed secrets and choose your profile:

```powershell
# Windows PowerShell
.\setup-hub.ps1 -Sector "nso" -HubName "National Bureau of Statistics" -Domain "statistics.gov.ug"
```

```bash
# Linux / macOS Shell
./setup-hub.sh --sector nso --hub-name "National Bureau of Statistics" --domain statistics.gov.ug
```

### Step 3: Launch Selected Profile
```bash
docker compose --profile nso up -d
```

### Step 4: Verify Deployment Readiness
```bash
# Verify health across foundational endpoints
curl http://localhost:8080/ready   # Analytics Core (Go)
curl http://localhost:5000/ready   # Analytics UI (Flask)
curl http://localhost:8105/health  # StatFederation Hub (Go)
curl http://localhost:8099/health  # Knowledge Open Data Portal
```

---

## 4. International Standards & Interoperability Endpoints

Every deployed hub automatically provides standard external interfaces:

| Standard | Endpoint | Description |
|---|---|---|
| **SDMX-REST 1.0** | `GET /api/sdmx/data/{indicator_code}` | Machine-readable time-series and aggregate indicators for UN, AU, World Bank |
| **CKAN 3.0 API** | `GET /api/3/action/package_list` | Open Data catalog package discovery for international metadata harvesters |
| **CKAN 3.0 Schema**| `GET /api/3/action/package_show?id={id}` | Complete DCAT/CKAN dataset schema and distribution links |
| **IATI 2.03 XML** | `GET /api/iati/activities.xml` | Official development aid and project disbursements formatted for OECD-DAC/IATI registry |
| **RSS / Atom Feed** | `GET /api/rss/datasets.xml` | Public subscription feed for new dataset releases and national gazettes |
| **Sovereign Push** | `POST /api/v1/federation/indicators/push`| Bilateral peer transmission protected by Data Sharing Agreements (DSAs) |
| **Spatial Disaggregation**| `POST /api/v1/statistics/indicators/compute/spatial` | PostGIS boundary-aggregated indicators by district/region |

---

## 5. Security & Sovereignty Guardrails

1. **Zero-Trust Token Validation:** All internal microservice requests mandate verified JWTs signed with `STATGATE_REGISTRY_JWT_SECRET` and internal service keys.
2. **Fail-Closed Attribute-Based Access Control (ABAC):** Multi-tenant scoping isolates individual line ministries. Without an explicitly validated security clearance (Levels 0–5), data access requests fail closed.
3. **Sovereign Microdata Protection:** Raw survey records and patient microdata **never leave the local sovereign node**. Only cryptographic indicator aggregates and anonymized statistical moments are transmitted across federation nodes.
