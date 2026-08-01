# StatGate Field Operations API

## Overview

The StatGate Field Operations API provides a registry of field survey stations, operational territories, station tiers, operating models, managing organisations, and approved public resources.

**Base URL:** `http://localhost:9090/api`

## Public registry endpoints

### Station tiers

**GET** `/api/level`

Returns the field-station tiers: Community Enumeration Point, Subcounty Field Station, District Coordination Hub, and Regional Operations Hub.

### Operating models

**GET** `/api/ownership`

Returns StatGate operating models: Public Sector, Non-profit Partner, and Private Partner.

### Managing organisations

**GET** `/api/authority`

Returns organisations responsible for field operations.

### Field-station filters

**GET** `/api/facilities/filters`

Returns the available region, district, municipality/DLG, and subcounty hierarchy used by the registry filters.

### Field-station catalogue

**GET** `/api/facilities/public?page=1&pageSize=50`

Lists field survey stations. Optional filters include `q`, `region`, `district`, `subcounty`, `level`, `ownership`, and `authority`.

### Registry summaries

**GET** `/api/facilities/summary/ownership-totals`

**GET** `/api/facilities/summary/ownership-by-level`

Both endpoints accept optional `region`, `district`, and `subcounty` parameters.

## Documents and resources

### List published resources

**GET** `/api/documents`

Returns published SOPs, manuals, training materials, and other approved resources. Use `category` to narrow results.

### Download a resource

**GET** `/api/documents/{id}/download`

Downloads a published document by ID.

## Authentication

Public registry, filter, summary, tier, operating-model, managing-organisation, and document-list endpoints do not require authentication. Administrative requests require a bearer token issued through the StatGate Registry sign-in workflow.

## Example

```bash
curl "http://localhost:9090/api/facilities/filters"
curl "http://localhost:9090/api/facilities/summary/ownership-totals?region=Acholi"
```
