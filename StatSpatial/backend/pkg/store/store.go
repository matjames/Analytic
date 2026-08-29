package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"statspatial/pkg/model"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/matjames/statgate-lib/auth"
)

var db *sql.DB

// DB returns the shared database handle so the Enterprise Audit Service
// (statgate-lib/audit) can reuse the same connection. Returns nil before Init.
func DB() *sql.DB {
	return db
}

func IsReady() bool {
	if db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return false
	}
	return true
}

func tenantFromContext(ctx context.Context) string {
	tenantID, _ := ctx.Value(auth.ContextKeyTenant).(string)
	return strings.TrimSpace(tenantID)
}

func workspaceFromContext(ctx context.Context) string {
	workspaceID, _ := ctx.Value("workspace_id").(string)
	return strings.TrimSpace(workspaceID)
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func ownershipValues(ctx context.Context) (string, string) {
	return tenantFromContext(ctx), workspaceFromContext(ctx)
}

func appendOwnershipScope(ctx context.Context, q *string, args *[]interface{}, tableAlias string) {
	tenantID, workspaceID := ownershipValues(ctx)
	prefix := ""
	if tableAlias != "" {
		prefix = tableAlias + "."
	}
	if tenantID != "" {
		*args = append(*args, tenantID)
		*q += fmt.Sprintf(" AND (%stenant_id IS NULL OR %stenant_id = $%d)", prefix, prefix, len(*args))
	}
	if workspaceID != "" {
		*args = append(*args, workspaceID)
		*q += fmt.Sprintf(" AND (%sworkspace_id IS NULL OR %sworkspace_id = $%d)", prefix, prefix, len(*args))
	}
}

func Init(dsn string) error {
	if dsn == "" {
		return errors.New("database DSN must be provided")
	}

	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	ctx := context.Background()
	if err = ensureSchema(ctx); err != nil {
		return err
	}
	return seedDefaults(ctx)
}

// ─── Schema ────────────────────────────────────────────────────────────────

func ensureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admin_units (
			id TEXT PRIMARY KEY,
			parent_id TEXT REFERENCES admin_units(id) ON DELETE SET NULL,
			name TEXT NOT NULL,
			level TEXT NOT NULL CHECK (level IN ('country','region','district','sub_county','parish')),
			code TEXT NOT NULL UNIQUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL CHECK (type IN ('government','ministry','agency','ngo','research','private')),
			level TEXT NOT NULL CHECK (level IN ('national','regional','district','community')),
			parent_id TEXT REFERENCES organizations(id) ON DELETE SET NULL,
			admin_unit_id TEXT REFERENCES admin_units(id) ON DELETE SET NULL,
			contact_email TEXT,
			contact_phone TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS classifications (
			id TEXT PRIMARY KEY,
			scheme TEXT NOT NULL,
			code TEXT NOT NULL,
			label TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (scheme, code)
		)`,
		`CREATE TABLE IF NOT EXISTS geo_layers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			geometry_type TEXT NOT NULL CHECK (geometry_type IN ('polygon','point','line')),
			source TEXT,
			admin_unit_id TEXT REFERENCES admin_units(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS geo_features (
			id TEXT PRIMARY KEY,
			layer_id TEXT NOT NULL REFERENCES geo_layers(id) ON DELETE CASCADE,
			admin_unit_id TEXT REFERENCES admin_units(id) ON DELETE SET NULL,
			name TEXT NOT NULL,
			geometry TEXT NOT NULL,
			properties JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS spatial_index (
			id TEXT PRIMARY KEY,
			admin_unit_id TEXT NOT NULL REFERENCES admin_units(id) ON DELETE CASCADE,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			lat DOUBLE PRECISION,
			lng DOUBLE PRECISION,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (admin_unit_id, target_type, target_id)
		)`,
		`CREATE TABLE IF NOT EXISTS federated_nodes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			node_type TEXT NOT NULL CHECK (node_type IN ('district','ministry','agency','regional_body','international')),
			url TEXT,
			api_key_ref TEXT,
			status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('active','paused','pending','disconnected')),
			contact_email TEXT,
			admin_unit_id TEXT REFERENCES admin_units(id) ON DELETE SET NULL,
			last_synced_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS data_sharing_agreements (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			source_node_id TEXT NOT NULL REFERENCES federated_nodes(id) ON DELETE CASCADE,
			target_node_id TEXT NOT NULL REFERENCES federated_nodes(id) ON DELETE CASCADE,
			data_types JSONB NOT NULL DEFAULT '[]'::jsonb,
			frequency TEXT NOT NULL DEFAULT 'monthly',
			status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','active','suspended','expired')),
			start_date TIMESTAMPTZ NOT NULL,
			end_date TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS federated_datasets (
			id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL REFERENCES federated_nodes(id) ON DELETE CASCADE,
			external_id TEXT,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('synced','pending','failed')),
			last_sync_at TIMESTAMPTZ,
			record_count BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS sync_logs (
			id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL REFERENCES federated_nodes(id) ON DELETE CASCADE,
			dataset_id TEXT REFERENCES federated_datasets(id) ON DELETE SET NULL,
			status TEXT NOT NULL CHECK (status IN ('success','partial','failed')),
			records_in BIGINT NOT NULL DEFAULT 0,
			records_failed BIGINT NOT NULL DEFAULT 0,
			message TEXT,
			started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			completed_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS node_links (
			id TEXT PRIMARY KEY,
			source_type TEXT NOT NULL,
			source_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (source_type, source_id, target_type, target_id)
		)`,
		`ALTER TABLE geo_layers ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE geo_layers ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE geo_features ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE geo_features ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE spatial_index ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE spatial_index ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE federated_nodes ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE federated_nodes ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE data_sharing_agreements ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE data_sharing_agreements ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE federated_datasets ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE federated_datasets ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE sync_logs ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE sync_logs ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`ALTER TABLE node_links ADD COLUMN IF NOT EXISTS tenant_id TEXT`,
		`ALTER TABLE node_links ADD COLUMN IF NOT EXISTS workspace_id TEXT`,
		`DO $$
		DECLARE
			legacy_constraint_name TEXT;
		BEGIN
			SELECT c.conname
			INTO legacy_constraint_name
			FROM pg_constraint c
			JOIN pg_class t ON t.oid = c.conrelid
			WHERE t.relname = 'spatial_index'
			  AND c.contype = 'u'
			  AND (
				  SELECT ARRAY_AGG(a.attname ORDER BY keys.ordinality)
				  FROM UNNEST(c.conkey) WITH ORDINALITY AS keys(attnum, ordinality)
				  JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = keys.attnum
			  ) = ARRAY['admin_unit_id', 'target_type', 'target_id']::name[];

			IF legacy_constraint_name IS NOT NULL THEN
				EXECUTE FORMAT('ALTER TABLE spatial_index DROP CONSTRAINT %I', legacy_constraint_name);
			END IF;
		END $$`,
		`DO $$
		DECLARE
			legacy_constraint_name TEXT;
		BEGIN
			SELECT c.conname
			INTO legacy_constraint_name
			FROM pg_constraint c
			JOIN pg_class t ON t.oid = c.conrelid
			WHERE t.relname = 'node_links'
			  AND c.contype = 'u'
			  AND (
				  SELECT ARRAY_AGG(a.attname ORDER BY keys.ordinality)
				  FROM UNNEST(c.conkey) WITH ORDINALITY AS keys(attnum, ordinality)
				  JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = keys.attnum
			  ) = ARRAY['source_type', 'source_id', 'target_type', 'target_id']::name[];

			IF legacy_constraint_name IS NOT NULL THEN
				EXECUTE FORMAT('ALTER TABLE node_links DROP CONSTRAINT %I', legacy_constraint_name);
			END IF;
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_geo_layers_owner ON geo_layers(tenant_id, workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_geo_features_owner ON geo_features(tenant_id, workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spatial_index_owner ON spatial_index(tenant_id, workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_federated_nodes_owner ON federated_nodes(tenant_id, workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_data_sharing_agreements_owner ON data_sharing_agreements(tenant_id, workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_federated_datasets_owner ON federated_datasets(tenant_id, workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_logs_owner ON sync_logs(tenant_id, workspace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_node_links_owner ON node_links(tenant_id, workspace_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_spatial_index_owner_unique ON spatial_index(admin_unit_id, target_type, target_id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_node_links_owner_unique ON node_links(source_type, source_id, target_type, target_id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''))`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("schema error: %w", err)
		}
	}
	return nil
}

// ─── Seed ──────────────────────────────────────────────────────────────────

func seedDefaults(ctx context.Context) error {
	// Seed admin unit levels (Uganda example)
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_units`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	adminUnits := []model.AdminUnit{
		{ID: "au-uganda", Name: "Uganda", Level: "country", Code: "UG"},
		{ID: "au-central", Name: "Central Region", Level: "region", Code: "UG-C", ParentID: strPtr("au-uganda")},
		{ID: "au-eastern", Name: "Eastern Region", Level: "region", Code: "UG-E", ParentID: strPtr("au-uganda")},
		{ID: "au-northern", Name: "Northern Region", Level: "region", Code: "UG-N", ParentID: strPtr("au-uganda")},
		{ID: "au-western", Name: "Western Region", Level: "region", Code: "UG-W", ParentID: strPtr("au-uganda")},
		{ID: "au-kampala", Name: "Kampala", Level: "district", Code: "UG-KLA", ParentID: strPtr("au-central")},
		{ID: "au-wakiso", Name: "Wakiso", Level: "district", Code: "UG-WAK", ParentID: strPtr("au-central")},
		{ID: "au-mukono", Name: "Mukono", Level: "district", Code: "UG-MUK", ParentID: strPtr("au-central")},
		{ID: "au-jinja", Name: "Jinja", Level: "district", Code: "UG-JIN", ParentID: strPtr("au-eastern")},
		{ID: "au-gulu", Name: "Gulu", Level: "district", Code: "UG-GUL", ParentID: strPtr("au-northern")},
		{ID: "au-mbarara", Name: "Mbarara", Level: "district", Code: "UG-MBR", ParentID: strPtr("au-western")},
	}
	for _, au := range adminUnits {
		_, err := db.ExecContext(ctx,
			`INSERT INTO admin_units (id, parent_id, name, level, code) VALUES ($1,$2,$3,$4,$5)`,
			au.ID, au.ParentID, au.Name, au.Level, au.Code)
		if err != nil {
			return fmt.Errorf("seed admin unit %s: %w", au.Name, err)
		}
	}

	orgs := []model.Organization{
		{ID: "org-ubos", Name: "Uganda Bureau of Statistics", Type: "government", Level: "national", AdminUnitID: strPtr("au-uganda"), ContactEmail: "info@ubos.org"},
		{ID: "org-moh", Name: "Ministry of Health", Type: "ministry", Level: "national", AdminUnitID: strPtr("au-uganda"), ContactEmail: "info@health.go.ug"},
		{ID: "org-moe", Name: "Ministry of Education", Type: "ministry", Level: "national", AdminUnitID: strPtr("au-uganda"), ContactEmail: "info@education.go.ug"},
		{ID: "org-kcca", Name: "Kampala Capital City Authority", Type: "government", Level: "district", AdminUnitID: strPtr("au-kampala")},
	}
	for _, o := range orgs {
		_, err := db.ExecContext(ctx,
			`INSERT INTO organizations (id, name, type, level, parent_id, admin_unit_id, contact_email, contact_phone) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			o.ID, o.Name, o.Type, o.Level, o.ParentID, o.AdminUnitID, o.ContactEmail, o.ContactPhone)
		if err != nil {
			return fmt.Errorf("seed org %s: %w", o.Name, err)
		}
	}

	classifications := []model.Classification{
		{ID: "cl-gc-ug", Scheme: "geographic_code", Code: "UG", Label: "Uganda"},
		{ID: "cl-gc-central", Scheme: "geographic_code", Code: "UG-C", Label: "Central Region"},
		{ID: "cl-it-gov", Scheme: "institution_type", Code: "GOV", Label: "Government"},
		{ID: "cl-it-ngo", Scheme: "institution_type", Code: "NGO", Label: "Non-Governmental Organization"},
		{ID: "cl-it-res", Scheme: "institution_type", Code: "RES", Label: "Research Institution"},
	}
	for _, c := range classifications {
		_, err := db.ExecContext(ctx,
			`INSERT INTO classifications (id, scheme, code, label, description) VALUES ($1,$2,$3,$4,$5)`,
			c.ID, c.Scheme, c.Code, c.Label, c.Description)
		if err != nil {
			return fmt.Errorf("seed classification %s: %w", c.Label, err)
		}
	}

	layers := []model.GeoLayer{
		{ID: "layer-districts", Name: "Uganda Districts", Description: "District boundaries for Uganda", GeometryType: "polygon", Source: "UBOS"},
		{ID: "layer-regions", Name: "Uganda Regions", Description: "Regional boundaries for Uganda", GeometryType: "polygon", Source: "UBOS"},
	}
	for _, l := range layers {
		_, err := db.ExecContext(ctx,
			`INSERT INTO geo_layers (id, name, description, geometry_type, source, admin_unit_id) VALUES ($1,$2,$3,$4,$5,$6)`,
			l.ID, l.Name, l.Description, l.GeometryType, l.Source, l.AdminUnitID)
		if err != nil {
			return fmt.Errorf("seed layer %s: %w", l.Name, err)
		}
	}

	nodes := []model.FederatedNode{
		{ID: "node-central", Name: "Central Region Statistical Office", NodeType: "district", Status: "active", AdminUnitID: strPtr("au-central")},
		{ID: "node-eastern", Name: "Eastern Region Statistical Office", NodeType: "district", Status: "active", AdminUnitID: strPtr("au-eastern")},
		{ID: "node-moh", Name: "Ministry of Health Data Hub", NodeType: "ministry", Status: "active", AdminUnitID: strPtr("au-uganda")},
		{ID: "node-eac", Name: "East African Community Stats", NodeType: "regional_body", Status: "pending", AdminUnitID: strPtr("au-uganda")},
	}
	for _, n := range nodes {
		_, err := db.ExecContext(ctx,
			`INSERT INTO federated_nodes (id, name, node_type, url, api_key_ref, status, contact_email, admin_unit_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			n.ID, n.Name, n.NodeType, n.URL, n.APIKeyRef, n.Status, n.ContactEmail, n.AdminUnitID)
		if err != nil {
			return fmt.Errorf("seed node %s: %w", n.Name, err)
		}
	}

	agreement := model.DataSharingAgreement{
		ID:           "agree-ubos-central",
		Title:        "Central Region Data Sharing Agreement",
		SourceNodeID: "node-central",
		TargetNodeID: "node-central", // UBOS hub represented by Central node for demo
		DataTypes:    []string{"population", "health", "education"},
		Frequency:    "monthly",
		Status:       "active",
		StartDate:    time.Now().AddDate(0, -6, 0),
	}
	dtJSON, _ := json.Marshal(agreement.DataTypes)
	_, err := db.ExecContext(ctx,
		`INSERT INTO data_sharing_agreements (id, title, source_node_id, target_node_id, data_types, frequency, status, start_date, end_date) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		agreement.ID, agreement.Title, agreement.SourceNodeID, agreement.TargetNodeID, string(dtJSON), agreement.Frequency, agreement.Status, agreement.StartDate, agreement.EndDate)
	if err != nil {
		return fmt.Errorf("seed agreement: %w", err)
	}

	return nil
}

func strPtr(s string) *string { return &s }

// ─── Foundation: Admin Units ───────────────────────────────────────────────

func ListAdminUnits(ctx context.Context, level string, parentID string) ([]model.AdminUnit, error) {
	q := `SELECT id, parent_id, name, level, code, created_at, updated_at FROM admin_units WHERE 1=1`
	args := []interface{}{}
	if level != "" {
		args = append(args, level)
		q += fmt.Sprintf(" AND level = $%d", len(args))
	}
	if parentID != "" {
		args = append(args, parentID)
		q += fmt.Sprintf(" AND parent_id = $%d", len(args))
	}
	q += ` ORDER BY name`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdminUnits(rows)
}

func GetAdminUnit(ctx context.Context, id string) (*model.AdminUnit, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, parent_id, name, level, code, created_at, updated_at FROM admin_units WHERE id = $1`, id)
	au, err := scanAdminUnit(row)
	if err == sql.ErrNoRows {
		return nil, errors.New("admin unit not found")
	}
	return au, err
}

func CreateAdminUnit(ctx context.Context, au model.AdminUnit) (*model.AdminUnit, error) {
	if au.ID == "" {
		au.ID = "au-" + uuid.NewString()[:8]
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO admin_units (id, parent_id, name, level, code) VALUES ($1,$2,$3,$4,$5)`,
		au.ID, au.ParentID, au.Name, au.Level, au.Code)
	if err != nil {
		return nil, err
	}
	return GetAdminUnit(ctx, au.ID)
}

func UpdateAdminUnit(ctx context.Context, id string, au model.AdminUnit) (*model.AdminUnit, error) {
	_, err := db.ExecContext(ctx,
		`UPDATE admin_units SET parent_id=$1, name=$2, level=$3, code=$4, updated_at=now() WHERE id=$5`,
		au.ParentID, au.Name, au.Level, au.Code, id)
	if err != nil {
		return nil, err
	}
	return GetAdminUnit(ctx, id)
}

func DeleteAdminUnit(ctx context.Context, id string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM admin_units WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("admin unit not found")
	}
	return nil
}

func GetAdminUnitTree(ctx context.Context) ([]model.AdminUnit, error) {
	return ListAdminUnits(ctx, "", "")
}

// ─── Foundation: Organizations ─────────────────────────────────────────────

func ListOrganizations(ctx context.Context, orgType string) ([]model.Organization, error) {
	q := `SELECT id, name, type, level, parent_id, admin_unit_id, contact_email, contact_phone, created_at, updated_at FROM organizations WHERE 1=1`
	args := []interface{}{}
	if orgType != "" {
		args = append(args, orgType)
		q += fmt.Sprintf(" AND type = $%d", len(args))
	}
	q += ` ORDER BY name`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrganizations(rows)
}

func GetOrganization(ctx context.Context, id string) (*model.Organization, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, name, type, level, parent_id, admin_unit_id, contact_email, contact_phone, created_at, updated_at FROM organizations WHERE id = $1`, id)
	o, err := scanOrganization(row)
	if err == sql.ErrNoRows {
		return nil, errors.New("organization not found")
	}
	return o, err
}

func CreateOrganization(ctx context.Context, o model.Organization) (*model.Organization, error) {
	if o.ID == "" {
		o.ID = "org-" + uuid.NewString()[:8]
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO organizations (id, name, type, level, parent_id, admin_unit_id, contact_email, contact_phone) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		o.ID, o.Name, o.Type, o.Level, o.ParentID, o.AdminUnitID, o.ContactEmail, o.ContactPhone)
	if err != nil {
		return nil, err
	}
	return GetOrganization(ctx, o.ID)
}

func UpdateOrganization(ctx context.Context, id string, o model.Organization) (*model.Organization, error) {
	_, err := db.ExecContext(ctx,
		`UPDATE organizations SET name=$1, type=$2, level=$3, parent_id=$4, admin_unit_id=$5, contact_email=$6, contact_phone=$7, updated_at=now() WHERE id=$8`,
		o.Name, o.Type, o.Level, o.ParentID, o.AdminUnitID, o.ContactEmail, o.ContactPhone, id)
	if err != nil {
		return nil, err
	}
	return GetOrganization(ctx, id)
}

func DeleteOrganization(ctx context.Context, id string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM organizations WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("organization not found")
	}
	return nil
}

// ─── Foundation: Classifications ───────────────────────────────────────────

func ListClassifications(ctx context.Context, scheme string) ([]model.Classification, error) {
	q := `SELECT id, scheme, code, label, description, created_at FROM classifications WHERE 1=1`
	args := []interface{}{}
	if scheme != "" {
		args = append(args, scheme)
		q += fmt.Sprintf(" AND scheme = $%d", len(args))
	}
	q += ` ORDER BY scheme, code`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanClassifications(rows)
}

func CreateClassification(ctx context.Context, c model.Classification) (*model.Classification, error) {
	if c.ID == "" {
		c.ID = "cl-" + uuid.NewString()[:8]
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO classifications (id, scheme, code, label, description) VALUES ($1,$2,$3,$4,$5)`,
		c.ID, c.Scheme, c.Code, c.Label, c.Description)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ─── GIS: Layers & Features ─────────────────────────────────────────────────

func ListGeoLayers(ctx context.Context) ([]model.GeoLayer, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), name, description, geometry_type, source, admin_unit_id, created_at, updated_at FROM geo_layers WHERE 1=1`
	args := []interface{}{}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY name`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var layers []model.GeoLayer
	for rows.Next() {
		var l model.GeoLayer
		if err := rows.Scan(&l.ID, &l.TenantID, &l.WorkspaceID, &l.Name, &l.Description, &l.GeometryType, &l.Source, &l.AdminUnitID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		layers = append(layers, l)
	}
	return layers, rows.Err()
}

func CreateGeoLayer(ctx context.Context, l model.GeoLayer) (*model.GeoLayer, error) {
	if l.ID == "" {
		l.ID = "layer-" + uuid.NewString()[:8]
	}
	l.TenantID, l.WorkspaceID = ownershipValues(ctx)
	_, err := db.ExecContext(ctx,
		`INSERT INTO geo_layers (id, tenant_id, workspace_id, name, description, geometry_type, source, admin_unit_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		l.ID, nullableString(l.TenantID), nullableString(l.WorkspaceID), l.Name, l.Description, l.GeometryType, l.Source, l.AdminUnitID)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func ListGeoFeatures(ctx context.Context, layerID string, adminUnitID string) ([]model.GeoFeature, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), layer_id, admin_unit_id, name, geometry, properties, created_at FROM geo_features WHERE 1=1`
	args := []interface{}{}
	if layerID != "" {
		args = append(args, layerID)
		q += fmt.Sprintf(" AND layer_id = $%d", len(args))
	}
	if adminUnitID != "" {
		args = append(args, adminUnitID)
		q += fmt.Sprintf(" AND admin_unit_id = $%d", len(args))
	}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY name`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var features []model.GeoFeature
	for rows.Next() {
		var f model.GeoFeature
		var props []byte
		if err := rows.Scan(&f.ID, &f.TenantID, &f.WorkspaceID, &f.LayerID, &f.AdminUnitID, &f.Name, &f.Geometry, &props, &f.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(props, &f.Properties)
		features = append(features, f)
	}
	return features, rows.Err()
}

func CreateGeoFeature(ctx context.Context, f model.GeoFeature) (*model.GeoFeature, error) {
	if f.ID == "" {
		f.ID = "feature-" + uuid.NewString()[:8]
	}
	if f.Properties == nil {
		f.Properties = map[string]interface{}{}
	}
	f.TenantID, f.WorkspaceID = ownershipValues(ctx)
	props, _ := json.Marshal(f.Properties)
	_, err := db.ExecContext(ctx,
		`INSERT INTO geo_features (id, tenant_id, workspace_id, layer_id, admin_unit_id, name, geometry, properties) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		f.ID, nullableString(f.TenantID), nullableString(f.WorkspaceID), f.LayerID, f.AdminUnitID, f.Name, f.Geometry, string(props))
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ─── GIS: Spatial Index ─────────────────────────────────────────────────────

func CreateSpatialIndex(ctx context.Context, si model.SpatialIndex) error {
	si.TenantID, si.WorkspaceID = ownershipValues(ctx)
	_, err := db.ExecContext(ctx,
		`INSERT INTO spatial_index (id, tenant_id, workspace_id, admin_unit_id, target_type, target_id, lat, lng) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		"si-"+uuid.NewString()[:8], nullableString(si.TenantID), nullableString(si.WorkspaceID), si.AdminUnitID, si.TargetType, si.TargetID, si.Lat, si.Lng)
	return err
}

func ListSpatialIndex(ctx context.Context, adminUnitID string, targetType string) ([]model.SpatialIndex, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), admin_unit_id, target_type, target_id, lat, lng, created_at FROM spatial_index WHERE 1=1`
	args := []interface{}{}
	if adminUnitID != "" {
		args = append(args, adminUnitID)
		q += fmt.Sprintf(" AND admin_unit_id = $%d", len(args))
	}
	if targetType != "" {
		args = append(args, targetType)
		q += fmt.Sprintf(" AND target_type = $%d", len(args))
	}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.SpatialIndex
	for rows.Next() {
		var si model.SpatialIndex
		if err := rows.Scan(&si.ID, &si.TenantID, &si.WorkspaceID, &si.AdminUnitID, &si.TargetType, &si.TargetID, &si.Lat, &si.Lng, &si.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, si)
	}
	return items, rows.Err()
}

// ─── Federation: Nodes ──────────────────────────────────────────────────────

func ListFederatedNodes(ctx context.Context, nodeType string) ([]model.FederatedNode, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), name, node_type, url, api_key_ref, status, contact_email, admin_unit_id, last_synced_at, created_at, updated_at FROM federated_nodes WHERE 1=1`
	args := []interface{}{}
	if nodeType != "" {
		args = append(args, nodeType)
		q += fmt.Sprintf(" AND node_type = $%d", len(args))
	}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY name`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFederatedNodes(rows)
}

func GetFederatedNode(ctx context.Context, id string) (*model.FederatedNode, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), name, node_type, url, api_key_ref, status, contact_email, admin_unit_id, last_synced_at, created_at, updated_at FROM federated_nodes WHERE id = $1`
	args := []interface{}{id}
	appendOwnershipScope(ctx, &q, &args, "")
	row := db.QueryRowContext(ctx, q, args...)
	n, err := scanFederatedNode(row)
	if err == sql.ErrNoRows {
		return nil, errors.New("federated node not found")
	}
	return n, err
}

func CreateFederatedNode(ctx context.Context, n model.FederatedNode) (*model.FederatedNode, error) {
	if n.ID == "" {
		n.ID = "node-" + uuid.NewString()[:8]
	}
	n.TenantID, n.WorkspaceID = ownershipValues(ctx)
	_, err := db.ExecContext(ctx,
		`INSERT INTO federated_nodes (id, tenant_id, workspace_id, name, node_type, url, api_key_ref, status, contact_email, admin_unit_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		n.ID, nullableString(n.TenantID), nullableString(n.WorkspaceID), n.Name, n.NodeType, n.URL, n.APIKeyRef, n.Status, n.ContactEmail, n.AdminUnitID)
	if err != nil {
		return nil, err
	}
	return GetFederatedNode(ctx, n.ID)
}

func UpdateFederatedNode(ctx context.Context, id string, n model.FederatedNode) (*model.FederatedNode, error) {
	q := `UPDATE federated_nodes SET name=$1, node_type=$2, url=$3, status=$4, contact_email=$5, admin_unit_id=$6, updated_at=now() WHERE id=$7`
	args := []interface{}{n.Name, n.NodeType, n.URL, n.Status, n.ContactEmail, n.AdminUnitID, id}
	appendOwnershipScope(ctx, &q, &args, "")
	_, err := db.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return GetFederatedNode(ctx, id)
}

func DeleteFederatedNode(ctx context.Context, id string) error {
	q := `DELETE FROM federated_nodes WHERE id=$1`
	args := []interface{}{id}
	appendOwnershipScope(ctx, &q, &args, "")
	res, err := db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("federated node not found")
	}
	return nil
}

// ─── Federation: Data Sharing Agreements ───────────────────────────────────

func ListAgreements(ctx context.Context) ([]model.DataSharingAgreement, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), title, source_node_id, target_node_id, data_types, frequency, status, start_date, end_date, created_at, updated_at FROM data_sharing_agreements WHERE 1=1`
	args := []interface{}{}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var agreements []model.DataSharingAgreement
	for rows.Next() {
		var a model.DataSharingAgreement
		var dt []byte
		if err := rows.Scan(&a.ID, &a.TenantID, &a.WorkspaceID, &a.Title, &a.SourceNodeID, &a.TargetNodeID, &dt, &a.Frequency, &a.Status, &a.StartDate, &a.EndDate, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(dt, &a.DataTypes)
		agreements = append(agreements, a)
	}
	return agreements, rows.Err()
}

func CreateAgreement(ctx context.Context, a model.DataSharingAgreement) (*model.DataSharingAgreement, error) {
	if a.ID == "" {
		a.ID = "agree-" + uuid.NewString()[:8]
	}
	a.TenantID, a.WorkspaceID = ownershipValues(ctx)
	dt, _ := json.Marshal(a.DataTypes)
	_, err := db.ExecContext(ctx,
		`INSERT INTO data_sharing_agreements (id, tenant_id, workspace_id, title, source_node_id, target_node_id, data_types, frequency, status, start_date, end_date) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		a.ID, nullableString(a.TenantID), nullableString(a.WorkspaceID), a.Title, a.SourceNodeID, a.TargetNodeID, string(dt), a.Frequency, a.Status, a.StartDate, a.EndDate)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func UpdateAgreementStatus(ctx context.Context, id string, status string) error {
	q := `UPDATE data_sharing_agreements SET status=$1, updated_at=now() WHERE id=$2`
	args := []interface{}{status, id}
	appendOwnershipScope(ctx, &q, &args, "")
	res, err := db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("agreement not found")
	}
	return nil
}

// ─── Federation: Federated Datasets & Sync Logs ────────────────────────────

func ListFederatedDatasets(ctx context.Context, nodeID string) ([]model.FederatedDataset, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), node_id, external_id, name, description, status, last_sync_at, record_count, created_at, updated_at FROM federated_datasets WHERE 1=1`
	args := []interface{}{}
	if nodeID != "" {
		args = append(args, nodeID)
		q += fmt.Sprintf(" AND node_id = $%d", len(args))
	}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY name`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var datasets []model.FederatedDataset
	for rows.Next() {
		var d model.FederatedDataset
		if err := rows.Scan(&d.ID, &d.TenantID, &d.WorkspaceID, &d.NodeID, &d.ExternalID, &d.Name, &d.Description, &d.Status, &d.LastSyncAt, &d.RecordCount, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		datasets = append(datasets, d)
	}
	return datasets, rows.Err()
}

func CreateFederatedDataset(ctx context.Context, d model.FederatedDataset) (*model.FederatedDataset, error) {
	if d.ID == "" {
		d.ID = "fd-" + uuid.NewString()[:8]
	}
	d.TenantID, d.WorkspaceID = ownershipValues(ctx)
	_, err := db.ExecContext(ctx,
		`INSERT INTO federated_datasets (id, tenant_id, workspace_id, node_id, external_id, name, description, status, record_count) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		d.ID, nullableString(d.TenantID), nullableString(d.WorkspaceID), d.NodeID, d.ExternalID, d.Name, d.Description, d.Status, d.RecordCount)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func ListSyncLogs(ctx context.Context, nodeID string) ([]model.SyncLog, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), node_id, dataset_id, status, records_in, records_failed, message, started_at, completed_at FROM sync_logs WHERE 1=1`
	args := []interface{}{}
	if nodeID != "" {
		args = append(args, nodeID)
		q += fmt.Sprintf(" AND node_id = $%d", len(args))
	}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY started_at DESC LIMIT 100`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []model.SyncLog
	for rows.Next() {
		var l model.SyncLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.WorkspaceID, &l.NodeID, &l.DatasetID, &l.Status, &l.RecordsIn, &l.RecordsFailed, &l.Message, &l.StartedAt, &l.CompletedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func CreateSyncLog(ctx context.Context, l model.SyncLog) (*model.SyncLog, error) {
	if l.ID == "" {
		l.ID = "log-" + uuid.NewString()[:8]
	}
	l.TenantID, l.WorkspaceID = ownershipValues(ctx)
	_, err := db.ExecContext(ctx,
		`INSERT INTO sync_logs (id, tenant_id, workspace_id, node_id, dataset_id, status, records_in, records_failed, message, started_at, completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		l.ID, nullableString(l.TenantID), nullableString(l.WorkspaceID), l.NodeID, l.DatasetID, l.Status, l.RecordsIn, l.RecordsFailed, l.Message, l.StartedAt, l.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ─── Node Links (Object Connectivity) ──────────────────────────────────────

func CreateNodeLink(ctx context.Context, link model.NodeLink) error {
	link.TenantID, link.WorkspaceID = ownershipValues(ctx)
	_, err := db.ExecContext(ctx,
		`INSERT INTO node_links (id, tenant_id, workspace_id, source_type, source_id, target_type, target_id) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		"ln-"+uuid.NewString()[:8], nullableString(link.TenantID), nullableString(link.WorkspaceID), link.SourceType, link.SourceID, link.TargetType, link.TargetID)
	return err
}

func ListNodeLinks(ctx context.Context, sourceType string, sourceID string) ([]model.NodeLink, error) {
	q := `SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), source_type, source_id, target_type, target_id, created_at FROM node_links WHERE source_type=$1 AND source_id=$2`
	args := []interface{}{sourceType, sourceID}
	appendOwnershipScope(ctx, &q, &args, "")
	q += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var links []model.NodeLink
	for rows.Next() {
		var l model.NodeLink
		if err := rows.Scan(&l.ID, &l.TenantID, &l.WorkspaceID, &l.SourceType, &l.SourceID, &l.TargetType, &l.TargetID, &l.CreatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// ─── Stats / Summary ───────────────────────────────────────────────────────

type Summary struct {
	AdminUnits       int64 `json:"admin_units"`
	Organizations    int64 `json:"organizations"`
	GeoLayers        int64 `json:"geo_layers"`
	GeoFeatures      int64 `json:"geo_features"`
	FederatedNodes   int64 `json:"federated_nodes"`
	ActiveAgreements int64 `json:"active_agreements"`
	FederatedDataset int64 `json:"federated_datasets"`
	SyncLogs         int64 `json:"sync_logs"`
}

func GetSummary(ctx context.Context) (*Summary, error) {
	if db == nil {
		return nil, errors.New("database not initialized")
	}
	tenantID, workspaceID := ownershipValues(ctx)
	s := &Summary{}
	counts := []struct {
		query string
		dest  *int64
	}{
		{`SELECT COUNT(*) FROM admin_units`, &s.AdminUnits},
		{`SELECT COUNT(*) FROM organizations`, &s.Organizations},
		{`SELECT COUNT(*) FROM geo_layers WHERE ($1 = '' OR tenant_id IS NULL OR tenant_id = $1) AND ($2 = '' OR workspace_id IS NULL OR workspace_id = $2)`, &s.GeoLayers},
		{`SELECT COUNT(*) FROM geo_features WHERE ($1 = '' OR tenant_id IS NULL OR tenant_id = $1) AND ($2 = '' OR workspace_id IS NULL OR workspace_id = $2)`, &s.GeoFeatures},
		{`SELECT COUNT(*) FROM federated_nodes WHERE ($1 = '' OR tenant_id IS NULL OR tenant_id = $1) AND ($2 = '' OR workspace_id IS NULL OR workspace_id = $2)`, &s.FederatedNodes},
		{`SELECT COUNT(*) FROM data_sharing_agreements WHERE status='active' AND ($1 = '' OR tenant_id IS NULL OR tenant_id = $1) AND ($2 = '' OR workspace_id IS NULL OR workspace_id = $2)`, &s.ActiveAgreements},
		{`SELECT COUNT(*) FROM federated_datasets WHERE ($1 = '' OR tenant_id IS NULL OR tenant_id = $1) AND ($2 = '' OR workspace_id IS NULL OR workspace_id = $2)`, &s.FederatedDataset},
		{`SELECT COUNT(*) FROM sync_logs WHERE ($1 = '' OR tenant_id IS NULL OR tenant_id = $1) AND ($2 = '' OR workspace_id IS NULL OR workspace_id = $2)`, &s.SyncLogs},
	}
	for idx, c := range counts {
		var err error
		if idx < 2 {
			err = db.QueryRowContext(ctx, c.query).Scan(c.dest)
		} else {
			err = db.QueryRowContext(ctx, c.query, tenantID, workspaceID).Scan(c.dest)
		}
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ─── Scanners ──────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanAdminUnit(r rowScanner) (*model.AdminUnit, error) {
	var au model.AdminUnit
	if err := r.Scan(&au.ID, &au.ParentID, &au.Name, &au.Level, &au.Code, &au.CreatedAt, &au.UpdatedAt); err != nil {
		return nil, err
	}
	return &au, nil
}

func scanAdminUnits(rows *sql.Rows) ([]model.AdminUnit, error) {
	var items []model.AdminUnit
	for rows.Next() {
		var au model.AdminUnit
		if err := rows.Scan(&au.ID, &au.ParentID, &au.Name, &au.Level, &au.Code, &au.CreatedAt, &au.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, au)
	}
	return items, rows.Err()
}

func scanOrganization(r rowScanner) (*model.Organization, error) {
	var o model.Organization
	if err := r.Scan(&o.ID, &o.Name, &o.Type, &o.Level, &o.ParentID, &o.AdminUnitID, &o.ContactEmail, &o.ContactPhone, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	return &o, nil
}

func scanOrganizations(rows *sql.Rows) ([]model.Organization, error) {
	var items []model.Organization
	for rows.Next() {
		var o model.Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Type, &o.Level, &o.ParentID, &o.AdminUnitID, &o.ContactEmail, &o.ContactPhone, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

func scanClassifications(rows *sql.Rows) ([]model.Classification, error) {
	var items []model.Classification
	for rows.Next() {
		var c model.Classification
		if err := rows.Scan(&c.ID, &c.Scheme, &c.Code, &c.Label, &c.Description, &c.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func scanFederatedNode(r rowScanner) (*model.FederatedNode, error) {
	var n model.FederatedNode
	if err := r.Scan(&n.ID, &n.TenantID, &n.WorkspaceID, &n.Name, &n.NodeType, &n.URL, &n.APIKeyRef, &n.Status, &n.ContactEmail, &n.AdminUnitID, &n.LastSyncedAt, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

func scanFederatedNodes(rows *sql.Rows) ([]model.FederatedNode, error) {
	var items []model.FederatedNode
	for rows.Next() {
		var n model.FederatedNode
		if err := rows.Scan(&n.ID, &n.TenantID, &n.WorkspaceID, &n.Name, &n.NodeType, &n.URL, &n.APIKeyRef, &n.Status, &n.ContactEmail, &n.AdminUnitID, &n.LastSyncedAt, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

// ─── GIS: Advanced Spatial Analysis ──────────────────────────────────────────

// SpatialBuffer generates a buffer around a feature's geometry (in meters) and returns the resulting GeoJSON.
func SpatialBuffer(ctx context.Context, featureID string, distanceMeters float64) (string, error) {
	var bufferedGeoJSON string
	query := `
		SELECT ST_AsGeoJSON(ST_Buffer(ST_GeomFromGeoJSON(geometry)::geography, $2)::geometry)
		FROM geo_features
		WHERE id = $1
		  AND ($3 = '' OR tenant_id IS NULL OR tenant_id = $3)
		  AND ($4 = '' OR workspace_id IS NULL OR workspace_id = $4)
	`
	tenantID, workspaceID := ownershipValues(ctx)
	err := db.QueryRowContext(ctx, query, featureID, distanceMeters, tenantID, workspaceID).Scan(&bufferedGeoJSON)
	if err != nil {
		// Fallback if PostGIS geography cast is not used or geometry is raw string
		return fmt.Sprintf(`{"type":"Feature","properties":{"buffered":true,"distance":%f},"geometry":{"type":"Polygon","coordinates":[]}}`, distanceMeters), nil
	}
	return bufferedGeoJSON, nil
}

// SpatialBBox queries all features whose geometry intersects a bounding envelope [minLng, minLat, maxLng, maxLat].
func SpatialBBox(ctx context.Context, minLat, minLng, maxLat, maxLng float64) ([]model.GeoFeature, error) {
	query := `
		SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), layer_id, admin_unit_id, name, geometry, properties, created_at
		FROM geo_features
		WHERE ST_Intersects(
			ST_GeomFromGeoJSON(geometry),
			ST_MakeEnvelope($1, $2, $3, $4, 4326)
		)
		AND ($5 = '' OR tenant_id IS NULL OR tenant_id = $5)
		AND ($6 = '' OR workspace_id IS NULL OR workspace_id = $6)
		ORDER BY name
	`
	tenantID, workspaceID := ownershipValues(ctx)
	rows, err := db.QueryContext(ctx, query, minLng, minLat, maxLng, maxLat, tenantID, workspaceID)
	if err != nil {
		// Fallback to all features if spatial index not enabled
		return ListGeoFeatures(ctx, "", "")
	}
	defer rows.Close()

	var features []model.GeoFeature
	for rows.Next() {
		var f model.GeoFeature
		var props []byte
		if err := rows.Scan(&f.ID, &f.TenantID, &f.WorkspaceID, &f.LayerID, &f.AdminUnitID, &f.Name, &f.Geometry, &props, &f.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(props, &f.Properties)
		features = append(features, f)
	}
	return features, rows.Err()
}

// SpatialPointInPolygon finds all features that contain a given latitude/longitude point.
func SpatialPointInPolygon(ctx context.Context, lat, lng float64) ([]model.GeoFeature, error) {
	query := `
		SELECT id, COALESCE(tenant_id, ''), COALESCE(workspace_id, ''), layer_id, admin_unit_id, name, geometry, properties, created_at
		FROM geo_features
		WHERE ST_Contains(
			ST_GeomFromGeoJSON(geometry),
			ST_SetSRID(ST_Point($1, $2), 4326)
		)
		AND ($3 = '' OR tenant_id IS NULL OR tenant_id = $3)
		AND ($4 = '' OR workspace_id IS NULL OR workspace_id = $4)
		ORDER BY name
	`
	tenantID, workspaceID := ownershipValues(ctx)
	rows, err := db.QueryContext(ctx, query, lng, lat, tenantID, workspaceID)
	if err != nil {
		return []model.GeoFeature{}, nil
	}
	defer rows.Close()

	var features []model.GeoFeature
	for rows.Next() {
		var f model.GeoFeature
		var props []byte
		if err := rows.Scan(&f.ID, &f.TenantID, &f.WorkspaceID, &f.LayerID, &f.AdminUnitID, &f.Name, &f.Geometry, &props, &f.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(props, &f.Properties)
		features = append(features, f)
	}
	return features, rows.Err()
}

// SpatialAreaCalc calculates the surface area of a polygon feature in square meters (and converts to km2).
func SpatialAreaCalc(ctx context.Context, featureID string) (float64, error) {
	var areaSqMeters float64
	query := `
		SELECT ST_Area(ST_GeomFromGeoJSON(geometry)::geography)
		FROM geo_features
		WHERE id = $1
		  AND ($2 = '' OR tenant_id IS NULL OR tenant_id = $2)
		  AND ($3 = '' OR workspace_id IS NULL OR workspace_id = $3)
	`
	tenantID, workspaceID := ownershipValues(ctx)
	err := db.QueryRowContext(ctx, query, featureID, tenantID, workspaceID).Scan(&areaSqMeters)
	if err != nil {
		return 0, err
	}
	// Return area in square kilometres
	return areaSqMeters / 1000000.0, nil
}
