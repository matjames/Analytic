# StatGate — Geospatial & Remote Sensing (App 9)

Go backend for **Geospatial & Remote Sensing (P44)**, extending the P10
StatSpatial arc with a full-stack GIS, spatial analytics, remote sensing, drone
integration, map tiles and geocoding.

| Sub-system | Implementation |
|---|---|
| **GIS Service / Vector Engine** | `geo_layers`, `geo_features` (GeoJSON geometry + centroid for search) |
| **Spatial Analytics Engine** | buffer, point-in-polygon, haversine distance, radius search, layer summary (dependency-free pure Go) |
| **Raster Processing Engine** | `rasters` with deterministic band statistics |
| **Remote Sensing Service** | `scenes` (Sentinel-2 / Landsat / drone), `spectral_indices` (NDVI/NDWI processing) |
| **Drone Integration Service** | `drones`, `flight_plans`, `flights`, `telemetry` + mission summary |
| **Map Tile Service** | `map_tiles` XYZ registry |
| **Geocoding Service** | forward/reverse geocode + cache |

A **first-class StatGate microservice**: Registry-JWT identity
(`statgate-lib/auth`), Redis event bus (`statgate:events`), immutable audit,
dedicated schema (`gis` in `gis_intelligence`), `/health` `/ready` `/metrics`.

## Mandatory system hooks

1. **Emits** `spatial.layer.created`, `spatial.feature.updated`,
   `scene.ingested`, `spatial.analysis.completed`, `drone.flight.completed`,
   `tile.registered`, `object.link.created` to `statgate:events`.
2. **`object_links`** — features auto-link to layers; generic `/api/v1/links`.
3. `statgate-lib/auth` (fail-closed, `X-Tenant-ID` isolation), `/health`
   `/ready` `/metrics`.

## API surface (all `/api/v1`, Registry JWT required)

- **GIS**: `GET|POST /layers`, `GET|DELETE /layers/{id}`,
  `GET|POST /layers/{id}/features`, `GET|DELETE /features/{id}`
- **Analytics**: `POST /analysis/buffer|contains|distance`, `GET /analysis/search`,
  `GET /analysis/summary/{layerId}`
- **Remote sensing**: `GET|POST /rasters`, `GET|POST /scenes`,
  `GET /scenes/{id}`, `POST /scenes/{id}/process`, `GET /indices?scene_id=`
- **Drone**: `GET|POST /drones`, `GET /drones/{id}`, `PUT /drones/{id}/status`,
  `GET|POST /flight-plans`, `GET|POST /flights`,
  `POST|GET /flights/{id}/telemetry`, `POST /flights/{id}/end`
- **Tiles & geocoding**: `GET|POST /tiles`, `POST /geocode`,
  `GET /geocode/reverse?lat=&lng=`, `GET /geocode/cache`
- **Cross-app**: `POST /links`, `GET /links`, `GET /summary`

## Run

```powershell
cd backend
go run ./cmd/server      # :8103 (GIS_PORT)
curl http://localhost:8103/health
```

DB provisioned by `docker/postgres-init/19-create-gis-db.sql`; registered in
`docker-compose.yml` as `geointel` (host `8103`, `statgate-network`).

## Verify

```powershell
go build ./...
go test  ./...
go vet   ./...
```