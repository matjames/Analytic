package store

import (
	"context"
	"database/sql"
	"fmt"

	"geointel/pkg/model"
)

// ─── GIS Layers (P44) ────────────────────────────────────────────────────────

func CreateLayer(ctx context.Context, l *model.GeoLayer) error {
	l.ID = newID()
	l.CreatedAt = nowT()
	l.UpdatedAt = l.CreatedAt
	if l.Status == "" {
		l.Status = "active"
	}
	if l.GeometryType == "" {
		l.GeometryType = "polygon"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO geo_layers (id, tenant_id, workspace_id, name, kind, geometry_type, source, tile_layer, metadata, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		l.ID, l.TenantID, l.WorkspaceID, l.Name, l.Kind, l.GeometryType, l.Source, l.TileLayer, jsonB(l.Metadata), l.Status, l.CreatedBy, l.CreatedAt, l.UpdatedAt)
	return err
}

func scanLayer(row row) (*model.GeoLayer, error) {
	var l model.GeoLayer
	var meta []byte
	if err := row.Scan(&l.ID, &l.TenantID, &l.WorkspaceID, &l.Name, &l.Kind, &l.GeometryType, &l.Source, &l.TileLayer, &meta, &l.Status, &l.CreatedBy, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return nil, err
	}
	jsonUnmarshal(meta, &l.Metadata)
	return &l, nil
}

const layerCols = `id, tenant_id, name, kind, geometry_type, source, tile_layer, metadata, status, created_by, COALESCE(workspace_id, ''), created_at, updated_at`

func ListLayers(ctx context.Context, tenantID, kind, workspaceID string) ([]model.GeoLayer, error) {
	query := `SELECT ` + layerCols + ` FROM geo_layers WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if workspaceID != "" {
		query += ` AND (workspace_id=$2 OR workspace_id='')`
		args = append(args, workspaceID)
	}
	if kind != "" {
		query += fmt.Sprintf(` AND kind=$%d`, len(args)+1)
		args = append(args, kind)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.GeoLayer
	for rows.Next() {
		l, err := scanLayer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func GetLayer(ctx context.Context, id string) (*model.GeoLayer, error) {
	return scanLayer(db.QueryRowContext(ctx, `SELECT `+layerCols+` FROM geo_layers WHERE id=$1`, id))
}

func DeleteLayer(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM geo_layers WHERE id=$1`, id)
	return err
}

// ─── Features (P44 vector engine) ────────────────────────────────────────────

func CreateFeature(ctx context.Context, f *model.GeoFeature) error {
	f.ID = newID()
	f.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO geo_features (id, tenant_id, layer_id, name, geometry, properties, centroid, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		f.ID, f.TenantID, f.LayerID, f.Name, f.Geometry, jsonB(f.Properties), f.Centroid, f.CreatedBy, f.CreatedAt)
	return err
}

func scanFeature(row row) (*model.GeoFeature, error) {
	var f model.GeoFeature
	var props []byte
	if err := row.Scan(&f.ID, &f.TenantID, &f.LayerID, &f.Name, &f.Geometry, &props, &f.Centroid, &f.CreatedBy, &f.CreatedAt); err != nil {
		return nil, err
	}
	jsonUnmarshal(props, &f.Properties)
	return &f, nil
}

const featureCols = `id, tenant_id, layer_id, name, geometry, properties, centroid, created_by, created_at`

func ListFeatures(ctx context.Context, layerID string) ([]model.GeoFeature, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+featureCols+` FROM geo_features WHERE layer_id=$1 ORDER BY created_at DESC`, layerID)
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
		out = append(out, *f)
	}
	return out, rows.Err()
}

func GetFeature(ctx context.Context, id string) (*model.GeoFeature, error) {
	return scanFeature(db.QueryRowContext(ctx, `SELECT `+featureCols+` FROM geo_features WHERE id=$1`, id))
}

func DeleteFeature(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM geo_features WHERE id=$1`, id)
	return err
}

var _ = sql.ErrNoRows