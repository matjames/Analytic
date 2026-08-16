package store

import (
	"context"

	"geointel/pkg/model"
)

// ─── Map Tiles (P44 tile service) ────────────────────────────────────────────

func CreateMapTile(ctx context.Context, t *model.MapTile) error {
	t.ID = newID()
	t.CreatedAt = nowT()
	if t.Format == "" {
		t.Format = "png"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO map_tiles (id, tenant_id, layer_id, name, z, x, y, format, path, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		t.ID, t.TenantID, t.LayerID, t.Name, t.Z, t.X, t.Y, t.Format, t.Path, t.CreatedAt)
	return err
}

func ListMapTiles(ctx context.Context, tenantID, layerID string) ([]model.MapTile, error) {
	query := `SELECT id, tenant_id, layer_id, name, z, x, y, format, path, created_at FROM map_tiles WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if layerID != "" {
		query += ` AND layer_id=$2`
		args = append(args, layerID)
	}
	query += ` ORDER BY z, x, y LIMIT 500`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.MapTile
	for rows.Next() {
		var t model.MapTile
		if err := rows.Scan(&t.ID, &t.TenantID, &t.LayerID, &t.Name, &t.Z, &t.X, &t.Y, &t.Format, &t.Path, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ─── Geocoding (P44 geocoding service) ───────────────────────────────────────

func SaveGeocode(ctx context.Context, g *model.GeocodeResult) error {
	g.ID = newID()
	g.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO geocode_cache (id, tenant_id, query, address, lat, lng, confidence, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		g.ID, g.TenantID, g.Query, g.Address, g.Lat, g.Lng, g.Confidence, g.CreatedAt)
	return err
}

func ListGeocodeCache(ctx context.Context, tenantID string) ([]model.GeocodeResult, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, query, address, lat, lng, confidence, created_at
		FROM geocode_cache WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 200`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.GeocodeResult
	for rows.Next() {
		var g model.GeocodeResult
		if err := rows.Scan(&g.ID, &g.TenantID, &g.Query, &g.Address, &g.Lat, &g.Lng, &g.Confidence, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// ─── Spatial search (Spatial API gateway) ────────────────────────────────────

// FeaturesInRadius returns features of a layer whose centroid falls within
// `radius` meters of (lat, lng). Geometry containment uses haversine on the
// centroid, so features must carry a centroid (computed on ingest).
func FeaturesInRadius(ctx context.Context, layerID string, lat, lng, radius float64) ([]model.GeoFeature, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, layer_id, name, geometry, properties, centroid, created_by, created_at
		FROM geo_features WHERE layer_id=$1 AND centroid IS NOT NULL AND centroid <> ''`, layerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.GeoFeature
	for rows.Next() {
		f, err := scanFeature(rows)
		if err != nil {
			return nil, err
		}
		clat, clng, ok := ParseCentroid(f.Centroid)
		if ok && HaversineMeters(lat, lng, clat, clng) <= radius {
			out = append(out, *f)
		}
	}
	return out, rows.Err()
}

// ─── Object linkage (cross-app contract) ─────────────────────────────────────

func CreateObjectLink(ctx context.Context, l *model.ObjectLink) error {
	l.ID = newID()
	l.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO object_links (id, tenant_id, source_type, source_id, target_type, target_id, relationship, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		l.ID, l.TenantID, l.SourceType, l.SourceID, l.TargetType, l.TargetID, l.Relationship, l.CreatedAt)
	return err
}

func ListObjectLinks(ctx context.Context, objectType, objectID string) ([]model.ObjectLink, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, source_type, source_id, target_type, target_id, relationship, created_at
		FROM object_links
		WHERE (source_type=$1 AND source_id=$2) OR (target_type=$1 AND target_id=$2)
		ORDER BY created_at DESC`, objectType, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ObjectLink
	for rows.Next() {
		var l model.ObjectLink
		if err := rows.Scan(&l.ID, &l.TenantID, &l.SourceType, &l.SourceID, &l.TargetType, &l.TargetID, &l.Relationship, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}