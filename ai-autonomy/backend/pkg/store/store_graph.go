package store

import (
	"context"
	"database/sql"
	"time"

	"aiengines/pkg/model"
)

// ─── Knowledge Graph: Nodes (P39) ────────────────────────────────────────────

func CreateGraphNode(ctx context.Context, n *model.GraphNode) error {
	n.ID = newID()
	n.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, tenant_id, type, label, properties, ref_object_type, ref_object_id, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		n.ID, n.TenantID, n.Type, n.Label, jsonB(n.Properties), n.RefObjectType, n.RefObjectID, n.CreatedBy, n.CreatedAt)
	return err
}

func scanNode(row row) (*model.GraphNode, error) {
	var n model.GraphNode
	var props []byte
	if err := row.Scan(&n.ID, &n.TenantID, &n.Type, &n.Label, &props, &n.RefObjectType, &n.RefObjectID, &n.CreatedBy, &n.CreatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(props, &n.Properties)
	return &n, nil
}

const nodeCols = `id, tenant_id, type, label, properties, ref_object_type, ref_object_id, created_by, created_at`

func ListGraphNodes(ctx context.Context, tenantID, typ string) ([]model.GraphNode, error) {
	query := `SELECT ` + nodeCols + ` FROM graph_nodes WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if typ != "" {
		query += ` AND type=$2`
		args = append(args, typ)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.GraphNode
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *n)
	}
	return out, rows.Err()
}

func GetGraphNode(ctx context.Context, id string) (*model.GraphNode, error) {
	return scanNode(db.QueryRowContext(ctx, `SELECT `+nodeCols+` FROM graph_nodes WHERE id=$1`, id))
}

func DeleteGraphNode(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM graph_nodes WHERE id=$1`, id)
	return err
}

// ─── Knowledge Graph: Edges (P39) ────────────────────────────────────────────

func CreateGraphEdge(ctx context.Context, e *model.GraphEdge) error {
	e.ID = newID()
	e.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO graph_edges (id, tenant_id, source_node, target_node, predicate, properties, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		e.ID, e.TenantID, e.SourceNode, e.TargetNode, e.Predicate, jsonB(e.Properties), e.CreatedBy, e.CreatedAt)
	return err
}

func scanEdge(row row) (*model.GraphEdge, error) {
	var e model.GraphEdge
	var props []byte
	if err := row.Scan(&e.ID, &e.TenantID, &e.SourceNode, &e.TargetNode, &e.Predicate, &props, &e.CreatedBy, &e.CreatedAt); err != nil {
		return nil, err
	}
	_ = jsonUnmarshal(props, &e.Properties)
	return &e, nil
}

const edgeCols = `id, tenant_id, source_node, target_node, predicate, properties, created_by, created_at`

func ListGraphEdges(ctx context.Context, tenantID, source, predicate string) ([]model.GraphEdge, error) {
	query := `SELECT ` + edgeCols + ` FROM graph_edges WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argc := 1
	if source != "" {
		argc++
		args = append(args, source)
		query += ` AND source_node=$` + itoa(argc)
	}
	if predicate != "" {
		argc++
		args = append(args, predicate)
		query += ` AND predicate=$` + itoa(argc)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.GraphEdge
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func DeleteGraphEdge(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM graph_edges WHERE id=$1`, id)
	return err
}

// ─── Knowledge Graph: Semantic Triplestore (P39) ─────────────────────────────

const tripleCols = `id, tenant_id, subject, predicate, object, confidence, provenance, created_by, created_at`

func CreateTriple(ctx context.Context, t *model.Triple) error {
	t.ID = newID()
	t.CreatedAt = nowT()
	if t.Confidence <= 0 {
		t.Confidence = 1
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO graph_statements (id, tenant_id, subject, predicate, object, confidence, provenance, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		t.ID, t.TenantID, t.Subject, t.Predicate, t.Object, t.Confidence, t.Provenance, t.CreatedBy, t.CreatedAt)
	return err
}

func scanTriple(row row) (*model.Triple, error) {
	var t model.Triple
	if err := row.Scan(&t.ID, &t.TenantID, &t.Subject, &t.Predicate, &t.Object, &t.Confidence, &t.Provenance, &t.CreatedBy, &t.CreatedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

// QueryTriples runs a triple-pattern query (any of subject/predicate/object may be empty).
func QueryTriples(ctx context.Context, tenantID, subject, predicate, object string) ([]model.Triple, error) {
	query := `SELECT ` + tripleCols + ` FROM graph_statements WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argc := 1
	cond := func(val string) string {
		argc++
		args = append(args, val)
		return ` AND ` + []string{"subject", "predicate", "object"}[argc-2] + `=$` + itoa(argc)
	}
	if subject != "" {
		query += cond(subject)
	}
	if predicate != "" {
		query += cond(predicate)
	}
	if object != "" {
		query += cond(object)
	}
	query += ` ORDER BY confidence DESC, created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Triple
	for rows.Next() {
		t, err := scanTriple(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

var _ = sql.ErrNoRows

func nowNow() time.Time { return time.Now().UTC() }

func minc(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Contextual Intelligence Indexer (P39) ───────────────────────────────────

func UpsertContext(ctx context.Context, e *model.ContextEntry) error {
	e.ID = newID()
	e.CreatedAt = nowT()
	e.Tokens = tokenize(e.Content)
	if e.Tokens == nil {
		e.Tokens = []string{}
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO context_index (id, tenant_id, entity_type, entity_id, content, tokens, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT DO NOTHING`,
		e.ID, e.TenantID, e.EntityType, e.EntityID, e.Content, jsonB(e.Tokens), e.CreatedAt)
	return err
}

// SearchContext returns context entries ranked by token overlap with the query.
// This is a deterministic, dependency-free approximation of the contextual
// intelligence indexer; it can be swapped for an embedding service later.
func SearchContext(ctx context.Context, tenantID, query string) ([]model.ContextHit, error) {
	q := tokenize(query)
	if len(q) == 0 {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx,
		`SELECT id, entity_type, entity_id, content FROM context_index WHERE tenant_id=$1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []model.ContextHit
	for rows.Next() {
		var id, entityType, entityID, content string
		if err := rows.Scan(&id, &entityType, &entityID, &content); err != nil {
			return nil, err
		}
		doc := tokenize(content)
		score := 0.0
		for _, qt := range q {
			for _, dt := range doc {
				if qt == dt {
					score++
					break
				}
			}
		}
		if score > 0 {
			hits = append(hits, model.ContextHit{
				EntityType: entityType,
				EntityID:   entityID,
				Score:      score / float64(len(q)),
				Content:    content,
			})
		}
	}
	return hits, rows.Err()
}

// ─── Object Linkage (cross-app contract) ─────────────────────────────────────

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