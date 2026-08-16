package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"knowledgeportal/pkg/model"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB

// DB returns the shared database handle for the Enterprise Audit Service
// (statgate-lib/audit) to reuse. Returns nil before Init.
func DB() *sql.DB { return db }

// IsReady reports database connectivity, used by the health probes.
func IsReady() bool {
	if db == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return db.PingContext(ctx) == nil
}

// Init opens the connection pool and applies the schema (auto-migration)
// following the standardised StatGate microservice pattern.
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
	return ensureSchema(context.Background())
}

func ensureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS content_items (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			org_id TEXT,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			slug TEXT NOT NULL,
			summary TEXT,
			body TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			author_id TEXT,
			tags JSONB NOT NULL DEFAULT '[]'::jsonb,
			published_at TIMESTAMPTZ,
			updated_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_content_tenant ON content_items(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_content_status ON content_items(status)`,
		`CREATE INDEX IF NOT EXISTS idx_content_slug ON content_items(slug)`,

		`CREATE TABLE IF NOT EXISTS public_datasets (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			title TEXT NOT NULL,
			slug TEXT NOT NULL,
			description TEXT,
			license TEXT NOT NULL,
			format TEXT NOT NULL,
			size_bytes BIGINT NOT NULL DEFAULT 0,
			download_url TEXT,
			source_app TEXT,
			source_object_type TEXT,
			source_object_id TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			version TEXT NOT NULL DEFAULT '1.0',
			tags JSONB NOT NULL DEFAULT '[]'::jsonb,
			updated_by TEXT,
			published_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_dataset_tenant ON public_datasets(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_dataset_status ON public_datasets(status)`,

		`CREATE TABLE IF NOT EXISTS repository_items (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			authors JSONB NOT NULL DEFAULT '[]'::jsonb,
			doi TEXT,
			isbn TEXT,
			issn TEXT,
			abstract TEXT,
			status TEXT NOT NULL DEFAULT 'draft',
			access TEXT NOT NULL DEFAULT 'open',
			rights TEXT,
			published_at TIMESTAMPTZ,
			updated_by TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_repo_tenant ON repository_items(tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_repo_status ON repository_items(status)`,

		`CREATE TABLE IF NOT EXISTS persistent_identifiers (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			item_type TEXT NOT NULL,
			item_id TEXT NOT NULL,
			id_type TEXT NOT NULL,
			id_value TEXT NOT NULL,
			provider TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (id_type, id_value)
		)`,

		`CREATE TABLE IF NOT EXISTS subscriptions (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			email TEXT NOT NULL,
			topics JSONB NOT NULL DEFAULT '[]'::jsonb,
			frequency TEXT NOT NULL DEFAULT 'weekly',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			UNIQUE (tenant_id, email)
		)`,

		`CREATE TABLE IF NOT EXISTS feedback (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			contact TEXT,
			subject TEXT NOT NULL,
			body TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'open',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		// Cross-application object linkage (mirrors docker/02-create-object-links.sql).
		`CREATE TABLE IF NOT EXISTS object_links (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			relationship TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_links_source ON object_links(source_type, source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_links_target ON object_links(target_type, target_id)`,

		`CREATE TABLE IF NOT EXISTS portal_settings (
			tenant_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value JSONB NOT NULL DEFAULT '{}'::jsonb,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (tenant_id, key)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

func newID() string  { return uuid.New().String() }
func now() time.Time { return time.Now().UTC() }

// ─── Public Datasets (P41) ─────────────────────────────────────────────────

func CreateDataset(ctx context.Context, d *model.PublicDataset) error {
	d.ID = newID()
	d.CreatedAt = now()
	d.UpdatedAt = d.CreatedAt
	tags, _ := json.Marshal(d.Tags)
	_, err := db.ExecContext(ctx, `
		INSERT INTO public_datasets (id, tenant_id, title, slug, description, license, format, size_bytes, download_url, source_app, source_object_type, source_object_id, status, version, tags, updated_by, published_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		d.ID, d.TenantID, d.Title, d.Slug, d.Description, d.License, d.Format, d.SizeBytes, d.DownloadURL, d.SourceApp, d.SourceObjType, d.SourceObjID, d.Status, d.Version, tags, d.UpdatedBy, d.PublishedAt, d.CreatedAt, d.UpdatedAt)
	return err
}

func scanDataset(row interface{ Scan(...interface{}) error }) (*model.PublicDataset, error) {
	var d model.PublicDataset
	var tags []byte
	var pub sql.NullTime
	if err := row.Scan(&d.ID, &d.TenantID, &d.Title, &d.Slug, &d.Description, &d.License, &d.Format, &d.SizeBytes, &d.DownloadURL, &d.SourceApp, &d.SourceObjType, &d.SourceObjID, &d.Status, &d.Version, &tags, &d.UpdatedBy, &pub, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(tags, &d.Tags)
	if pub.Valid {
		d.PublishedAt = &pub.Time
	}
	return &d, nil
}

func ListDatasets(ctx context.Context, tenantID, status, q string) ([]model.PublicDataset, error) {
	where := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	argc := 1
	if status != "" {
		argc++
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", argc))
	}
	if q != "" {
		argc++
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf("(lower(title) LIKE $%d OR lower(description) LIKE $%d)", argc, argc))
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, title, slug, description, license, format, size_bytes, download_url, source_app, source_object_type, source_object_id, status, version, tags, updated_by, published_at, created_at, updated_at
		FROM public_datasets WHERE `+strings.Join(where, " AND ")+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.PublicDataset
	for rows.Next() {
		d, err := scanDataset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func GetDataset(ctx context.Context, id string) (*model.PublicDataset, error) {
	return scanDataset(db.QueryRowContext(ctx, `
		SELECT id, tenant_id, title, slug, description, license, format, size_bytes, download_url, source_app, source_object_type, source_object_id, status, version, tags, updated_by, published_at, created_at, updated_at
		FROM public_datasets WHERE id=$1`, id))
}

func UpdateDataset(ctx context.Context, d *model.PublicDataset) error {
	d.UpdatedAt = now()
	tags, _ := json.Marshal(d.Tags)
	_, err := db.ExecContext(ctx, `
		UPDATE public_datasets SET title=$2, slug=$3, description=$4, license=$5, format=$6, size_bytes=$7, download_url=$8, status=$9, version=$10, tags=$11, published_at=$12, updated_by=$13, updated_at=$14
		WHERE id=$1`,
		d.ID, d.Title, d.Slug, d.Description, d.License, d.Format, d.SizeBytes, d.DownloadURL, d.Status, d.Version, tags, d.PublishedAt, d.UpdatedBy, d.UpdatedAt)
	return err
}

func DeleteDataset(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM public_datasets WHERE id=$1`, id)
	return err
}

// ─── Repository / Library (P40) ────────────────────────────────────────────

func CreateRepositoryItem(ctx context.Context, it *model.RepositoryItem) error {
	it.ID = newID()
	it.CreatedAt = now()
	it.UpdatedAt = it.CreatedAt
	authors, _ := json.Marshal(it.Authors)
	_, err := db.ExecContext(ctx, `
		INSERT INTO repository_items (id, tenant_id, kind, title, authors, doi, isbn, issn, abstract, status, access, rights, published_at, updated_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		it.ID, it.TenantID, it.Kind, it.Title, authors, it.DOI, it.ISBN, it.ISSN, it.Abstract, it.Status, it.Access, it.Rights, it.PublishedAt, it.UpdatedBy, it.CreatedAt, it.UpdatedAt)
	return err
}

func scanRepo(row interface{ Scan(...interface{}) error }) (*model.RepositoryItem, error) {
	var it model.RepositoryItem
	var authors []byte
	var pub sql.NullTime
	if err := row.Scan(&it.ID, &it.TenantID, &it.Kind, &it.Title, &authors, &it.DOI, &it.ISBN, &it.ISSN, &it.Abstract, &it.Status, &it.Access, &it.Rights, &pub, &it.UpdatedBy, &it.CreatedAt, &it.UpdatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(authors, &it.Authors)
	if pub.Valid {
		it.PublishedAt = &pub.Time
	}
	return &it, nil
}

func ListRepositoryItems(ctx context.Context, tenantID, status, q string) ([]model.RepositoryItem, error) {
	where := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	argc := 1
	if status != "" {
		argc++
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", argc))
	}
	if q != "" {
		argc++
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf("(lower(title) LIKE $%d OR lower(abstract) LIKE $%d)", argc, argc))
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, kind, title, authors, doi, isbn, issn, abstract, status, access, rights, published_at, updated_by, created_at, updated_at
		FROM repository_items WHERE `+strings.Join(where, " AND ")+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.RepositoryItem
	for rows.Next() {
		it, err := scanRepo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

func GetRepositoryItem(ctx context.Context, id string) (*model.RepositoryItem, error) {
	return scanRepo(db.QueryRowContext(ctx, `
		SELECT id, tenant_id, kind, title, authors, doi, isbn, issn, abstract, status, access, rights, published_at, updated_by, created_at, updated_at
		FROM repository_items WHERE id=$1`, id))
}

func UpdateRepositoryItem(ctx context.Context, it *model.RepositoryItem) error {
	it.UpdatedAt = now()
	authors, _ := json.Marshal(it.Authors)
	_, err := db.ExecContext(ctx, `
		UPDATE repository_items SET kind=$2, title=$3, authors=$4, doi=$5, isbn=$6, issn=$7, abstract=$8, status=$9, access=$10, rights=$11, published_at=$12, updated_by=$13, updated_at=$14
		WHERE id=$1`,
		it.ID, it.Kind, it.Title, authors, it.DOI, it.ISBN, it.ISSN, it.Abstract, it.Status, it.Access, it.Rights, it.PublishedAt, it.UpdatedBy, it.UpdatedAt)
	return err
}

func DeleteRepositoryItem(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM repository_items WHERE id=$1`, id)
	return err
}

// ─── Persistent Identifiers (P40) ─────────────────────────────────────────

func CreateIdentifier(ctx context.Context, pi *model.PersistentIdentifier) error {
	pi.ID = newID()
	pi.CreatedAt = now()
	_, err := db.ExecContext(ctx, `
		INSERT INTO persistent_identifiers (id, tenant_id, item_type, item_id, id_type, id_value, provider, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		pi.ID, pi.TenantID, pi.ItemType, pi.ItemID, pi.IDType, pi.IDValue, pi.Provider, pi.CreatedAt)
	return err
}

// ─── Subscriptions (P17) ──────────────────────────────────────────────────

func CreateSubscription(ctx context.Context, s *model.Subscription) error {
	s.ID = newID()
	s.CreatedAt = now()
	if s.Frequency == "" {
		s.Frequency = "weekly"
	}
	if s.Status == "" {
		s.Status = "active"
	}
	topics, _ := json.Marshal(s.Topics)
	_, err := db.ExecContext(ctx, `
		INSERT INTO subscriptions (id, tenant_id, email, topics, frequency, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (tenant_id, email) DO UPDATE SET status='active'`,
		s.ID, s.TenantID, s.Email, topics, s.Frequency, s.Status, s.CreatedAt)
	return err
}

func ListSubscriptions(ctx context.Context, tenantID string) ([]model.Subscription, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, email, topics, frequency, status, created_at
		FROM subscriptions WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Subscription
	for rows.Next() {
		var s model.Subscription
		var topics []byte
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Email, &topics, &s.Frequency, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(topics, &s.Topics)
		out = append(out, s)
	}
	return out, rows.Err()
}

// ─── Feedback (P17 / P41) ─────────────────────────────────────────────────

func CreateFeedback(ctx context.Context, f *model.Feedback) error {
	f.ID = newID()
	f.CreatedAt = now()
	if f.Status == "" {
		f.Status = "open"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO feedback (id, tenant_id, kind, contact, subject, body, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		f.ID, f.TenantID, f.Kind, f.Contact, f.Subject, f.Body, f.Status, f.CreatedAt)
	return err
}

func ListFeedback(ctx context.Context, tenantID string) ([]model.Feedback, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, kind, contact, subject, body, status, created_at
		FROM feedback WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Feedback
	for rows.Next() {
		var f model.Feedback
		if err := rows.Scan(&f.ID, &f.TenantID, &f.Kind, &f.Contact, &f.Subject, &f.Body, &f.Status, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ─── On-board a platform object link (cross-app communication) ─────────────

func CreateObjectLink(ctx context.Context, l *model.ObjectLink) error {
	l.ID = newID()
	l.CreatedAt = now()
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

// ─── Public cross-catalog search (P17 / P41 / P47) ─────────────────────────

func SearchPublic(ctx context.Context, tenantID, q string) ([]model.SearchResult, error) {
	var out []model.SearchResult
	if q == "" {
		return out, nil
	}
	like := "%" + strings.ToLower(q) + "%"
	rows, err := db.QueryContext(ctx, `
		SELECT 'content', id, title, summary, status, published_at FROM content_items
		WHERE tenant_id=$1 AND status='published' AND (lower(title) LIKE $2 OR lower(summary) LIKE $2 OR lower(body) LIKE $2)
		UNION ALL
		SELECT 'dataset', id, title, description, status, published_at FROM public_datasets
		WHERE tenant_id=$1 AND status='published' AND (lower(title) LIKE $2 OR lower(description) LIKE $2)
		UNION ALL
		SELECT 'repository', id, title, abstract, status, published_at FROM repository_items
		WHERE tenant_id=$1 AND status='published' AND (lower(title) LIKE $2 OR lower(abstract) LIKE $2)
		ORDER BY published_at DESC NULLS LAST`, tenantID, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s model.SearchResult
		var pub sql.NullTime
		if err := rows.Scan(&s.Type, &s.ID, &s.Title, &s.Summary, &s.Status, &pub); err != nil {
			return nil, err
		}
		if pub.Valid {
			s.PublishedAt = &pub.Time
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// PortalSummary returns counts for the admin overview.
func PortalSummary(ctx context.Context, tenantID string) (*model.PortalSummary, error) {
	s := &model.PortalSummary{}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM content_items WHERE tenant_id=$1`, tenantID).Scan(&s.ContentCount); err != nil {
		return nil, err
	}
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM public_datasets WHERE tenant_id=$1`, tenantID).Scan(&s.DatasetCount)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM repository_items WHERE tenant_id=$1`, tenantID).Scan(&s.RepositoryCount)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM content_items WHERE tenant_id=$1 AND status='published'`, tenantID).Scan(&s.PublishedCount)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM subscriptions WHERE tenant_id=$1 AND status='active'`, tenantID).Scan(&s.SubscriptionCount)
	_ = db.QueryRowContext(ctx, `SELECT count(*) FROM feedback WHERE tenant_id=$1`, tenantID).Scan(&s.FeedbackCount)
	return s, nil
}





// ─── Content (P17 CMS) ─────────────────────────────────────────────────────

func CreateContentItem(ctx context.Context, c *model.ContentItem) error {
	c.ID = newID()
	c.CreatedAt = now()
	c.UpdatedAt = c.CreatedAt
	tags, _ := json.Marshal(c.Tags)
	_, err := db.ExecContext(ctx, `
		INSERT INTO content_items (id, tenant_id, org_id, kind, title, slug, summary, body, status, author_id, tags, published_at, updated_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		c.ID, c.TenantID, c.OrgID, c.Kind, c.Title, c.Slug, c.Summary, c.Body, c.Status, c.AuthorID, tags, c.PublishedAt, c.UpdatedBy, c.CreatedAt, c.UpdatedAt)
	return err
}

func ListContentItems(ctx context.Context, tenantID, kind, status, q string) ([]model.ContentItem, error) {
	where := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	argc := 1
	if kind != "" {
		argc++
		args = append(args, kind)
		where = append(where, fmt.Sprintf("kind=$%d", argc))
	}
	if status != "" {
		argc++
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", argc))
	}
	if q != "" {
		argc++
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, fmt.Sprintf("(lower(title) LIKE $%d OR lower(summary) LIKE $%d OR lower(body) LIKE $%d)", argc, argc, argc))
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, org_id, kind, title, slug, summary, body, status, author_id, tags, published_at, updated_by, created_at, updated_at
		FROM content_items WHERE `+strings.Join(where, " AND ")+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ContentItem
	for rows.Next() {
		var c model.ContentItem
		var tags []byte
		var pub sql.NullTime
		if err := rows.Scan(&c.ID, &c.TenantID, &c.OrgID, &c.Kind, &c.Title, &c.Slug, &c.Summary, &c.Body, &c.Status, &c.AuthorID, &tags, &pub, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tags, &c.Tags)
		if pub.Valid {
			c.PublishedAt = &pub.Time
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func GetContentItem(ctx context.Context, id string) (*model.ContentItem, error) {
	var c model.ContentItem
	var tags []byte
	var pub sql.NullTime
	err := db.QueryRowContext(ctx, `
		SELECT id, tenant_id, org_id, kind, title, slug, summary, body, status, author_id, tags, published_at, updated_by, created_at, updated_at
		FROM content_items WHERE id=$1`, id).
		Scan(&c.ID, &c.TenantID, &c.OrgID, &c.Kind, &c.Title, &c.Slug, &c.Summary, &c.Body, &c.Status, &c.AuthorID, &tags, &pub, &c.UpdatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(tags, &c.Tags)
	if pub.Valid {
		c.PublishedAt = &pub.Time
	}
	return &c, nil
}

func UpdateContentItem(ctx context.Context, c *model.ContentItem) error {
	c.UpdatedAt = now()
	tags, _ := json.Marshal(c.Tags)
	_, err := db.ExecContext(ctx, `
		UPDATE content_items SET kind=$2, title=$3, slug=$4, summary=$5, body=$6, status=$7, author_id=$8, tags=$9, published_at=$10, updated_by=$11, updated_at=$12
		WHERE id=$1`,
		c.ID, c.Kind, c.Title, c.Slug, c.Summary, c.Body, c.Status, c.AuthorID, tags, c.PublishedAt, c.UpdatedBy, c.UpdatedAt)
	return err
}

func DeleteContentItem(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM content_items WHERE id=$1`, id)
	return err
}

