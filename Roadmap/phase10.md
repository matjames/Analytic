# PHASE 10 — GEOGRAPHIC INFORMATION SYSTEM (GIS), SPATIAL ANALYTICS & REMOTE SENSING

## Objective

Develop a comprehensive Geographic Information System (GIS) that integrates spatial intelligence into every StatGate module.

The GIS platform enables users to visualise, analyse, collect, monitor, and forecast geographically referenced data, making spatial analysis a first-class capability across statistics, research, projects, and governance.

---

## Vision

Every dataset in StatGate should have the ability to become spatially enabled.

Maps become the primary visualisation and analysis interface for field operations, surveys, censuses, projects, public health, agriculture, education, environment, and infrastructure.

---

## Service: StatSpatial

**Repository:** `StatSpatial/`  
**Backend:** Go + Gin → `StatSpatial/backend/` (:8094)  
**Frontend:** React 18 + Vite + TypeScript → (to be created) (:3013 or `:3014`)  
**Database:** `statspatial` (PostgreSQL 15 + PostGIS extension)  
**Auth:** Registry JWT middleware on all routes  
**Redis:** Redis event bus for spatial event publishing

---

## Current State

**Current implementation status:** StatSpatial has an authenticated Go API, PostgreSQL-backed spatial ownership records, workspace propagation, health/readiness endpoints, a React map surface, and project-location routes consumed by PMS. The project-location slice is live-certified; the wider GIS scope below remains in progress and must be completed incrementally.

---

## Backend Structure to Build

```
StatSpatial/backend/
├── main.go              ← Gin server, PostGIS DB init, Redis init, Prometheus, CORS, health
├── database.go          ← PostgreSQL + PostGIS connection + migrateDB()
├── routes.go            ← RegisterRoutes() grouped by resource
├── pkg/
│   ├── api/
│   │   ├── layers.go           ← spatial layer management
│   │   ├── features.go         ← GeoJSON feature CRUD
│   │   ├── boundaries.go       ← administrative boundary management
│   │   ├── analysis.go         ← spatial analytics (buffer, proximity, cluster)
│   │   ├── geocoding.go        ← geocoding + reverse geocoding
│   │   ├── tracking.go         ← GPS tracking
│   │   └── tiles.go            ← map tile serving
│   ├── model/
│   │   └── spatial_models.go   ← GeoJSON, feature, layer, boundary structs
│   └── store/
│       └── spatial_store.go    ← PostGIS query layer
├── middleware.go        ← registryAuthMiddleware(), getEnv()
└── events.go            ← Redis event publishing
```

---

## API Routes to Build

### Spatial Layer Management

```
GET    /api/layers                             ← list spatial layers
POST   /api/layers                             ← create spatial layer
GET    /api/layers/:id                         ← layer detail + metadata
PUT    /api/layers/:id                         ← update layer
DELETE /api/layers/:id                         ← delete layer
GET    /api/layers/:id/features                ← GeoJSON features in layer
POST   /api/layers/:id/features                ← add feature to layer
GET    /api/layers/:id/export                  ← export as GeoJSON / Shapefile / KML
POST   /api/layers/import                      ← import GeoJSON / Shapefile / KML / CSV
```

### Administrative Boundaries

```
GET    /api/boundaries                         ← list boundary sets (national, regional, district)
GET    /api/boundaries/:level                  ← boundaries at admin level
GET    /api/boundaries/:id                     ← specific boundary GeoJSON
POST   /api/boundaries                         ← upload / register boundary set
PUT    /api/boundaries/:id                     ← update boundary
```

### Feature Services

```
GET    /api/features/:id                       ← feature detail with geometry
PUT    /api/features/:id                       ← update feature
DELETE /api/features/:id                       ← delete feature
POST   /api/features/:id/attributes            ← add / update feature attributes
```

### Spatial Analytics

```
POST   /api/analytics/buffer                   ← buffer analysis (point/line/polygon + radius)
POST   /api/analytics/proximity                ← proximity query (nearest N features)
POST   /api/analytics/cluster                  ← cluster analysis (K-means or DBSCAN)
POST   /api/analytics/intersect                ← intersection of two layers
POST   /api/analytics/union                    ← union of two layers
POST   /api/analytics/difference               ← difference between layers
POST   /api/analytics/heatmap                  ← density heat map computation
GET    /api/analytics/statistics/:layerId      ← spatial statistics (area, count, density)
```

### Geocoding

```
GET    /api/geocode?address=                   ← geocode address → coordinates
GET    /api/reverse-geocode?lat=&lng=          ← reverse geocode coordinates → address
POST   /api/geocode/batch                      ← batch geocode a list of addresses
```

### GPS Tracking

```
POST   /api/tracking/location                  ← submit GPS position (from mobile)
GET    /api/tracking/:entityId/history         ← GPS track history for entity
GET    /api/tracking/:entityId/last            ← last known position
GET    /api/tracking/live                      ← SSE stream of live GPS positions
```

### Map Tiles

```
GET    /api/tiles/:z/:x/:y.mvt                 ← vector tile (Mapbox Vector Tile format)
GET    /api/tiles/:layerId/:z/:x/:y.mvt        ← per-layer vector tile
```

### Project / Survey Integration

```
GET    /api/spatial/project/:id/map            ← project locations on map
GET    /api/spatial/survey/:id/map             ← survey coverage map
GET    /api/spatial/submissions/map            ← field submission heat map
POST   /api/spatial/link                       ← link geometry to any platform object
```

### Dashboard, Search, Activity

```
GET    /api/dashboard                          ← spatial dashboard summary
GET    /api/search?q=                          ← search layers, features, boundaries
GET    /api/activity                           ← enterprise activity timeline
```

---

## Database Schema to Build

```sql
-- Requires PostGIS extension
CREATE EXTENSION IF NOT EXISTS postgis;

-- Spatial layers
spatial_layers (id, name, type, geometry_type, srid, description, source, created_by, created_at)

-- Features (geometry stored as PostGIS geometry type)
spatial_features (id, layer_id, geometry GEOMETRY, properties JSONB, created_at, updated_at)

-- Administrative boundaries
admin_boundaries (id, name, level, parent_id, geometry GEOMETRY, country_code, created_at)

-- GPS tracking
gps_tracks (id, entity_type, entity_id, user_id, latitude, longitude, altitude, accuracy, timestamp)

-- Geocoding cache
geocode_cache (id, address_hash, address, latitude, longitude, country, created_at)

-- Object spatial links
spatial_object_links (id, object_type, object_id, geometry GEOMETRY, label, created_at)

-- Analytics results
spatial_analytics_jobs (id, type, input_params JSONB, result JSONB, status, created_at, completed_at)
```

---

## Frontend to Build

```
StatSpatial/frontend/           ← React 18 + Vite + TypeScript (:3014)
├── src/
│   ├── pages/
│   │   ├── MapExplorer.tsx         ← main interactive GIS map
│   │   ├── LayerManager.tsx        ← layer library and upload
│   │   ├── SpatialAnalysis.tsx     ← run spatial analytics
│   │   ├── BoundaryManager.tsx     ← admin boundary management
│   │   ├── GPSTracker.tsx          ← live GPS tracking dashboard
│   │   └── SpatialDashboard.tsx    ← summary spatial dashboard
│   └── components/
│       ├── MapCanvas.tsx           ← Leaflet/MapLibre map component
│       ├── LayerPanel.tsx          ← layer controls
│       ├── LegendPanel.tsx         ← map legend
│       ├── DrawingTools.tsx        ← draw shapes on map
│       ├── MeasurementTools.tsx    ← measure distance/area
│       └── FeaturePopup.tsx        ← feature attribute popup
```

**Map Library:** Leaflet (already referenced in Flask templates) or MapLibre GL JS for vector tiles

---

## Supported Data Formats

| Format | Import | Export |
|---|---|---|
| GeoJSON | ✓ | ✓ |
| Shapefile (.shp + .dbf + .shx) | ✓ | ✓ |
| KML / KMZ | ✓ | ✓ |
| GeoPackage (.gpkg) | ✓ | ✓ |
| CSV with lat/lng columns | ✓ | ✓ |
| GPX | ✓ | — |
| XYZ / Vector Tiles | — (serve only) | — |
| GeoTIFF (raster) | ✓ (display only) | — |

---

## Integration Points

| Module | Integration |
|---|---|
| StatCollect | GPS coordinates from field submissions mapped on spatial layer |
| PMS | Project activity locations plotted on map |
| RMS | Research field visit sites on map |
| Registry | Facilities and org-units mapped against admin boundaries |
| Enterprise Core | Object links connect any platform object to a geometry |
| Phase XI (Phase 12) | Enumerator GPS tracking and supervisor field maps |

---

## Acceptance Criteria

- [ ] StatSpatial Go backend running at :8094 with health/ready/metrics endpoints
- [ ] PostGIS extension enabled and spatial tables created
- [ ] Administrative boundary import and query operational
- [ ] GeoJSON layer CRUD operational
- [ ] Feature CRUD with PostGIS geometry storage operational
- [ ] Buffer analysis operational
- [ ] Proximity query operational
- [ ] Heat map computation operational
- [ ] Cluster analysis operational
- [ ] Geocoding (forward and reverse) operational
- [ ] GPS tracking (submit position, history, live stream) operational
- [ ] Vector tile serving operational (Mapbox Vector Tile format)
- [ ] Map Explorer React UI operational (Leaflet/MapLibre)
- [ ] Layer Manager UI operational
- [x] PMS project locations on map operational
- [ ] StatCollect submission heat map operational
- [ ] Registry facility mapping on admin boundaries operational

---

## Ports & Services

| Component | Port |
|---|---|
| StatSpatial API (Go) | :8094 |
| StatSpatial UI (React/Vite) | :3014 |

---

## Estimated Duration

12 weeks

## Milestone

Enterprise GIS Platform complete. Every dataset, project, survey, and field submission is spatially enabled and explorable on an interactive map.
