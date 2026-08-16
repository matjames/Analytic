package store

import (
	"context"

	"geointel/pkg/model"
)

// ─── Rasters (P44 raster engine) ─────────────────────────────────────────────

func CreateRaster(ctx context.Context, r *model.Raster) error {
	r.ID = newID()
	r.CreatedAt = nowT()
	if r.Bands == 0 {
		r.Bands = 1
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO rasters (id, tenant_id, name, source, bands, width, height, bbox, stats, file_path, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		r.ID, r.TenantID, r.Name, r.Source, r.Bands, r.Width, r.Height, r.BBox, jsonB(r.Stats), r.FilePath, r.CreatedAt)
	return err
}

const rasterCols = `id, tenant_id, name, source, bands, width, height, bbox, stats, file_path, created_at`

func ListRasters(ctx context.Context, tenantID string) ([]model.Raster, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+rasterCols+` FROM rasters WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Raster
	for rows.Next() {
		var r model.Raster
		var stats []byte
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Source, &r.Bands, &r.Width, &r.Height, &r.BBox, &stats, &r.FilePath, &r.CreatedAt); err != nil {
			return nil, err
		}
		jsonUnmarshal(stats, &r.Stats)
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRaster returns a raster by id (used by the processing endpoint).
func GetRaster(ctx context.Context, id string) (*model.Raster, error) {
	var r model.Raster
	var stats []byte
	err := db.QueryRowContext(ctx, `SELECT `+rasterCols+` FROM rasters WHERE id=$1`, id).
		Scan(&r.ID, &r.TenantID, &r.Name, &r.Source, &r.Bands, &r.Width, &r.Height, &r.BBox, &stats, &r.FilePath, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	jsonUnmarshal(stats, &r.Stats)
	return &r, nil
}

// ─── Scenes & Spectral Indexes (P44 remote sensing service) ──────────────────

func CreateScene(ctx context.Context, s *model.Scene) error {
	s.ID = newID()
	s.CreatedAt = nowT()
	if s.Status == "" {
		s.Status = "ingested"
	}
	if s.Platform == "" {
		s.Platform = "sentinel-2"
	}
	if s.Resolution == 0 {
		s.Resolution = 10
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO scenes (id, tenant_id, name, platform, capture_time, bbox, cloud_cover, resolution, status, thumb_url, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		s.ID, s.TenantID, s.Name, s.Platform, s.CaptureTime, s.BBox, s.CloudCover, s.Resolution, s.Status, s.ThumbURL, s.CreatedAt)
	return err
}

func scanScene(row row) (*model.Scene, error) {
	var s model.Scene
	if err := row.Scan(&s.ID, &s.TenantID, &s.Name, &s.Platform, &s.CaptureTime, &s.BBox, &s.CloudCover, &s.Resolution, &s.Status, &s.ThumbURL, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

const sceneCols = `id, tenant_id, name, platform, capture_time, bbox, cloud_cover, resolution, status, thumb_url, created_at`

func ListScenes(ctx context.Context, tenantID string) ([]model.Scene, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+sceneCols+` FROM scenes WHERE tenant_id=$1 ORDER BY capture_time DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Scene
	for rows.Next() {
		s, err := scanScene(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func GetScene(ctx context.Context, id string) (*model.Scene, error) {
	return scanScene(db.QueryRowContext(ctx, `SELECT `+sceneCols+` FROM scenes WHERE id=$1`, id))
}

func UpdateSceneStatus(ctx context.Context, id, status string) error {
	_, err := db.ExecContext(ctx, `UPDATE scenes SET status=$2 WHERE id=$1`, id, status)
	return err
}

func CreateSpectralIndex(ctx context.Context, idx *model.SpectralIndex) error {
	idx.ID = newID()
	idx.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO spectral_indices (id, tenant_id, scene_id, index_type, values, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		idx.ID, idx.TenantID, idx.SceneID, idx.IndexType, jsonB(idx.Values), idx.CreatedAt)
	return err
}

func ListSpectralIndices(ctx context.Context, sceneID string) ([]model.SpectralIndex, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, scene_id, index_type, values, created_at FROM spectral_indices
		WHERE scene_id=$1 ORDER BY created_at DESC`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.SpectralIndex
	for rows.Next() {
		var idx model.SpectralIndex
		var vals []byte
		if err := rows.Scan(&idx.ID, &idx.TenantID, &idx.SceneID, &idx.IndexType, &vals, &idx.CreatedAt); err != nil {
			return nil, err
		}
		jsonUnmarshal(vals, &idx.Values)
		out = append(out, idx)
	}
	return out, rows.Err()
}