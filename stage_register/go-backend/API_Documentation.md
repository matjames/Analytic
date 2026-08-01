# Organization Units API Documentation

## Overview

Simple and straightforward API for querying organizational units and facilities with full hierarchy information. Facility levels, authorities, and ownership endpoints are public (no authentication). Organization units endpoints require **Basic Authentication**.

## Base URL

```
http://localhost:9090/api
```

## Authentication

- **Organization units** (`/api/orgunits`, `/api/orgunits/*`): require **Basic Authentication**.
- **Facility levels, authorities, ownership** (`/api/level`, `/api/authority`, `/api/ownership`): **no authentication** required (public).

```bash
# Public (no auth)
curl "http://localhost:9090/api/level"
curl "http://localhost:9090/api/authority"
curl "http://localhost:9090/api/ownership"

# Org units (auth required)
curl -u "username:password" "http://localhost:9090/api/orgunits"
```

---

## Facility Levels

**GET** `/api/level`

List all facility levels (e.g. HC II, HC III, HC IV, Hospital). No authentication required. Responses do not include the `code` field.

**Authentication:** None (public)

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "mfl_uid": "oCvQlbteWkJ",
    "name": "HC III",
    "description": "Health Centre III",
    "createdAt": "2026-01-22T10:00:00Z",
    "updatedAt": "2026-01-22T10:00:00Z"
  }
]
```

**Example:**
```bash
curl "http://localhost:9090/api/level"
```

---

## Authorities

**GET** `/api/authority`

List all authority types (e.g. Ministry of Health, Private Authority). No authentication required. Responses do not include the `code` field.

**Authentication:** None (public)

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "mfl_uid": "LzNS5OKEdXq",
    "name": "Ministry of Health",
    "description": "",
    "createdAt": "2026-01-22T10:00:00Z",
    "updatedAt": "2026-01-22T10:00:00Z"
  }
]
```

**Example:**
```bash
curl "http://localhost:9090/api/authority"
```

---

## Ownership

**GET** `/api/ownership`

List all ownership types (e.g. Government, PFP, PNFP). No authentication required. Responses do not include the `code` field.

**Authentication:** None (public)

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "mfl_uid": "h9qxIk88JV",
    "name": "Government",
    "description": "",
    "createdAt": "2026-01-22T10:00:00Z",
    "updatedAt": "2026-01-22T10:00:00Z"
  }
]
```

**Example:**
```bash
curl "http://localhost:9090/api/ownership"
```

---

## Organization Units Endpoints

### 1. List Organization Units

```
GET /api/orgunits
```

Query all organizational units with various filters.

#### Query Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `level` | int | Hierarchy level (1=National, 2=Region, 3=District, 6=Facility, etc.) | `level=6` |
| `minLevel` | int | Minimum hierarchy level | `minLevel=2` |
| `maxLevel` | int | Maximum hierarchy level | `maxLevel=5` |
| `parentId` | int64 | Filter by parent ID | `parentId=100` |
| `regionId` | int64 | Filter by region ID | `regionId=5` |
| `regionName` | string | Filter by region name (partial match) | `regionName=Central` |
| `districtId` | string | District ID (numeric) or **mfl_uid** | `districtId=REJuxCmTwXG` |
| `districtName` | string | Filter by district name (partial match) | `districtName=Kampala` |
| `district` | string | Alias for `districtName` | `district=Agago District` |
| `subcountyId` | int64 | Filter by subcounty ID | `subcountyId=200` |
| `subcountyName` | string | Filter by subcounty name (partial match) | `subcountyName=Nakawa` |
| `ownership` | string | Filter by ownership: **code**, **name**, or **mfl_uid** (e.g. GOV, Government) | `ownership=GOV` |
| `authority` | string | Filter by authority: **code**, **name**, or **mfl_uid** | `authority=MOH` |
| `facility_level` | string | Filter by facility level type: **mfl_uid** or **name** (e.g. HC III) | `facility_level=HC III` |
| `status` | string | Filter by status | `status=Active` |
| `reporting` | boolean | Filter by reporting status | `reporting=true` |
| `licensed` | boolean | Filter by licensed status | `licensed=true` |
| `search` | string | Search in name field | `search=Health` |
| `name` | string | Exact name match | `name=Kampala District` |
| `mflUid` | string | Filter by MFL UID | `mflUid=akV6429SUqu` |
| `mfluid` | string | Alias for `mflUid` | `mfluid=akV6429SUqu` |
| `updatedSince` | string | Only units updated on or after date (YYYY-MM-DD or ISO) | `updatedSince=2025-12-19` |
| `page` | int | Page number | `page=1` |
| `pageSize` | int | Items per page (max 500) | `pageSize=50` |
| `paging` | boolean | Enable pagination | `paging=true` |
| `includeChildren` | boolean | Include immediate children | `includeChildren=true` |

#### Examples

**Get all regions:**
```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=2"
```

**Get all districts:**
```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=3&paging=false"
```

**Get facilities in a specific district:**
```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&districtId=50"
```

**Get government facilities in Central region:**
```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&regionName=Central&ownership=GOV"
```

**Search for health centers:**
```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?search=Health+Center&level=6"
```

**Get active reporting facilities:**
```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&status=Active&reporting=true"
```

#### Response Format

```json
{
  "pager": {
    "page": 1,
    "pageSize": 50,
    "pageCount": 3,
    "total": 135
  },
  "orgunits": [
    {
      "id": 100,
      "mfl_uid": "district_uid",
      "name": "Kampala District",
      "level": 3,
      "parent": {
        "id": 5,
        "mfl_uid": "region_uid",
        "name": "Central Region"
      },
      "path": "/national_uid/region_uid/district_uid",
      "created": "2026-01-22T13:27:13.589Z",
      "lastUpdated": "2026-01-22T13:27:20.283Z"
    }
  ],
  "total": 135,
  "page": 1,
  "pageSize": 50
}
```

**Facility Response** (level=6):
```json
{
  "orgunits": [
    {
      "id": 2545,
      "mfl_uid": "facility_uid",
      "name": "Mulago Hospital",
      "level": 6,
      "parent": {
        "id": 200,
        "mfl_uid": "subcounty_uid",
        "name": "Kawempe Division"
      },
      "path": "/national_uid/region_uid/district_uid/dlg_uid/subcounty_uid/facility_uid",
      "created": "2026-01-28T14:00:22.935032+03:00",
      "lastUpdated": "2026-01-28T14:00:22.935032+03:00",
      // Hierarchy object (only for level 6)
      "hierarchy": {
        "region": {
          "id": 5,
          "mfl_uid": "region_uid",
          "name": "Central Region"
        },
        "district": {
          "id": 50,
          "mfl_uid": "district_uid",
          "name": "Kampala District"
        },
        "municipality": {
          "id": 45,
          "mfl_uid": "kcca_uid",
          "name": "Kampala Capital City Authority"
        },
        "division": {
          "id": 200,
          "mfl_uid": "subcounty_uid",
          "name": "Kawempe Division"
        }
      },
      // Facility-specific fields
      "historical_id": "8008022870916",
      "status": "Active",
      "reporting": true,
      "licensed": true,
      "ownership": {
        "mfl_uid": "h9qxIk88JV",
        "name": "Government"
      },
      "authority": {
        "mfl_uid": "LzNS5OKEdXq",
        "name": "Ministry of Health"
      },
      "facility_level": {
        "mfl_uid": "HCIV",
        "name": "Health Centre IV"
      },
      "longitude": 32.5825,
      "latitude": 0.3476,
      "opening_date": "1962-06-01",
      "bed_capacity": 1500,
      "address": "Mulago Hill, Kampala",
      "contact_personname": "Dr. John Doe",
      "contact_personmobile": "+256700123456",
      "contact_personemail": "john.doe@mulago.go.ug"
    }
  ],
  "total": 1
}
```

**Note:** The `hierarchy` object adapts based on the facility's location:
- Urban facilities may have `municipality` and `division` 
- Rural facilities may have `dlg` and `subcounty`
```

---

### 2. Get Single Organization Unit

```
GET /api/orgunits/:id
```

Get a specific organizational unit by MFL UID.

#### Parameters

- `:id` - MFL UID of the organization unit

#### Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `includeChildren` | boolean | Include immediate children |

#### Example

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits/akV6429SUqu?includeChildren=true"
```

---

### 3. Get Units by Level

```
GET /api/orgunits/level/:level
```

Get all organizational units at a specific level.

#### Parameters

- `:level` - Level number (1=National, 2=Region, 3=District, etc.)

#### Query Parameters

- `page` - Page number
- `pageSize` - Items per page

#### Example

```bash
# Get all districts (paginated)
curl -u "dev:password" "http://localhost:9090/api/orgunits/level/3?page=1&pageSize=20"
```

---

### 4. Get District Facilities

```
GET /api/orgunits/district/:id/facilities
```

Get all facilities under a specific district.

#### Parameters

- `:id` - District ID

#### Query Parameters

- `page` - Page number
- `pageSize` - Items per page
- `paging` - Enable/disable pagination

#### Example

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits/district/50/facilities?page=1&pageSize=50"
```

---

### 5. Get Subcounty Facilities

```
GET /api/orgunits/subcounty/:id/facilities
```

Get all facilities under a specific subcounty.

#### Parameters

- `:id` - Subcounty ID

#### Query Parameters

- `page` - Page number
- `pageSize` - Items per page
- `paging` - Enable/disable pagination

#### Example

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits/subcounty/200/facilities?paging=false"
```

---

### 6. Get Children

```
GET /api/orgunits/:id/children
```

Get immediate children of an organizational unit.

#### Parameters

- `:id` - MFL UID of the parent organization unit

#### Example

```bash
# Get all districts under a region
curl -u "dev:password" "http://localhost:9090/api/orgunits/region_uid/children"
```

---

### 7. Get Organization Tree

```
GET /api/orgunits/tree
```

Get the full organizational hierarchy tree.

#### Query Parameters

- `rootId` - Optional root MFL UID to start from

#### Example

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits/tree"
```

---

## Organizational Levels

| Level | Description |
|-------|-------------|
| 1 | National |
| 2 | Region |
| 3 | District / City |
| 4 | DLG / Municipality |
| 5 | Subcounty / Division |
| 6 | Health Facility |
| 7 | Parish |
| 8 | Village |

---

## Understanding Level Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `level` | int | Organizational level number | `6` |
| `facility_level` | object | Facility type (only for level 6) | `{"mfl_uid": "HCIII", "name": "Health Centre III"}` |

**Example for a Health Center:**
```json
{
  "level": 6,
  "facility_level": {
    "mfl_uid": "HCIII",
    "name": "Health Centre III"
  }
}
```

**Facility Types** (in `facility_level` object):
- HC II (Health Centre II)
- HC III (Health Centre III)
- HC IV (Health Centre IV)
- Hospital (General Hospital, Regional Referral Hospital, etc.)
- Clinic
- Medical Center
- etc.

---

## Hierarchy Object (Level 6 Only)

For facilities (level 6), a `hierarchy` object is included that provides easy access to the facility's administrative location:

```json
"hierarchy": {
  "region": {
    "id": 5,
    "mfl_uid": "region_uid",
    "name": "Central Region"
  },
  "district": {
    "id": 50,
    "mfl_uid": "district_uid",
    "name": "Kampala District"
  },
  "municipality": {
    "id": 45,
    "mfl_uid": "kcca_uid",
    "name": "Kampala Capital City Authority"
  },
  "division": {
    "id": 200,
    "mfl_uid": "subcounty_uid",
    "name": "Kawempe Division"
  }
}
```

**Hierarchy Structure:**
- `region` - Always present for facilities
- `district` - Always present for facilities
- `dlg` OR `municipality` - Depending on the area type (rural vs urban)
- `subcounty` OR `division` - Depending on the area type (rural vs urban)

**Examples:**

**Urban Facility (Kampala):**
```json
"hierarchy": {
  "region": { ... },
  "district": { ... },
  "municipality": { "name": "Kampala Capital City Authority" },
  "division": { "name": "Kawempe Division" }
}
```

**Rural Facility:**
```json
"hierarchy": {
  "region": { ... },
  "district": { ... },
  "dlg": { "name": "Mbale District Local Government" },
  "subcounty": { "name": "Budadiri Subcounty" }
}
```

---

## Common Use Cases

### Get All Regions

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=2&paging=false"
```

### Get All Districts in Central Region

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=3&regionName=Central"
```

### Get All Government Facilities

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&ownership=GOV"
```

### Get Active Facilities in Kampala District

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&districtName=Kampala&status=Active"
```

### Get Licensed Reporting Facilities

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&licensed=true&reporting=true"
```

### Search for Specific Health Center

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?search=Mulago&level=6"
```

### Get Facilities by Subcounty

```bash
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&subcountyName=Nakawa"
```

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Invalid parameter value"
}
```

### 401 Unauthorized

```json
{
  "error": "Unauthorized"
}
```

### 404 Not Found

```json
{
  "error": "Organization unit not found"
}
```

### 500 Internal Server Error

```json
{
  "error": "Internal server error message"
}
```

---

## Notes

1. **Facility levels, authorities, ownership** (`/api/level`, `/api/authority`, `/api/ownership`) are public and do not require authentication. Their responses omit the `code` field.
2. `level` (query param) is the **hierarchy** level number (1=National, 2=Region, 3=District, 6=Facility, etc.). Use `facility_level` to filter by facility type (e.g. HC III).
3. `ownership` and `authority` filters accept code, name, or mfl_uid (partial name match is case-insensitive).
4. `facility_level` object (only for level 6) shows the facility type like HC II, HC III, HC IV, Hospital, etc.
5. All facilities (level 6) include a `hierarchy` object for easy access to region, district, dlg/municipality, and subcounty/division.
6. Facilities include ownership, authority, and facility_level information (as objects with mfl_uid and name).
7. The `id` field is the unique admin_unit_id for the organizational unit.
8. The `parent` object includes full details (id, mfl_uid, name) of the immediate parent.
9. Pagination is enabled by default with a page size of 50; maximum page size is 500.
10. All text searches are case-insensitive; multiple filters can be combined in a single request.
11. The hierarchy object intelligently uses `dlg`/`subcounty` for rural areas and `municipality`/`division` for urban areas.
12. Use `district` as an alias for `districtName`; use `mfluid` as an alias for `mflUid`.

---

## Testing

### Run Backend

```bash
cd go-backend
go run main.go
```

### Test with cURL

```bash
# Get all regions
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=2"

# Get facilities in a district
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&districtId=50"

# Search for facilities
curl -u "dev:password" "http://localhost:9090/api/orgunits?level=6&search=Health"
```

---

## Version

- **Version**: 1.1
- **Updated**: February 19, 2026
- **Database View**: `orgunits` (flattened hierarchy view)
- **Public reference APIs**: `/api/level`, `/api/authority`, `/api/ownership` (no auth)
