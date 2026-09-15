package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// UniversalIndexer transforms heterogeneous domain objects into unified indexed search documents
type UniversalIndexer struct {
	store store.Store
}

// NewUniversalIndexer creates an indexer
func NewUniversalIndexer(s store.Store) *UniversalIndexer {
	return &UniversalIndexer{store: s}
}

// IndexDataset indexes a catalog dataset
func (ui *UniversalIndexer) IndexDataset(ctx context.Context, ds *models.Dataset) error {
	docText := fmt.Sprintf("%s %s %s %s", ds.Name, ds.Description, ds.Domain, strings.Join(ds.Tags, " "))
	vec := GeneratePseudoEmbedding(docText, 384)

	doc := &models.IndexedDocument{
		ID:             "doc-ds-" + ds.ID,
		IndexName:      "statgate_global",
		ResourceID:     ds.ID,
		ResourceType:   "DATASET",
		Title:          ds.Name,
		Content:        ds.Description,
		Domain:         ds.Domain,
		Classification: string(ds.Classification),
		Owner:          ds.OwnerTeam,
		Tags:           ds.Tags,
		Vector:         vec,
		TenantID:       ds.TenantID,
		WorkspaceID:    store.WorkspaceID(ctx),
		IndexedAt:      time.Now().UTC(),
		Metadata: map[string]interface{}{
			"format":        ds.Format,
			"quality_score": ds.QualityScore,
			"row_count":     ds.RowCount,
		},
	}
	return ui.store.IndexDocument(ctx, doc)
}

// IndexModel indexes a registered machine learning / statistical model
func (ui *UniversalIndexer) IndexModel(ctx context.Context, model *models.RegisteredModel) error {
	docText := fmt.Sprintf("%s %s %s %s", model.Name, model.Description, model.Domain, model.Framework)
	vec := GeneratePseudoEmbedding(docText, 384)

	doc := &models.IndexedDocument{
		ID:             "doc-model-" + model.ID,
		IndexName:      "statgate_global",
		ResourceID:     model.ID,
		ResourceType:   "MODEL",
		Title:          model.Name,
		Content:        model.Description,
		Domain:         model.Domain,
		Classification: "INTERNAL",
		Owner:          model.CreatedBy,
		Tags:           []string{model.Domain, model.Framework, string(model.LatestStage)},
		Vector:         vec,
		TenantID:       model.TenantID,
		WorkspaceID:    store.WorkspaceID(ctx),
		IndexedAt:      time.Now().UTC(),
		Metadata: map[string]interface{}{
			"framework":    model.Framework,
			"latest_stage": model.LatestStage,
		},
	}
	return ui.store.IndexDocument(ctx, doc)
}

// IndexGeneric indexes arbitrary cross-system entity
func (ui *UniversalIndexer) IndexGeneric(ctx context.Context, resourceID, resourceType, title, content, domain, classification, tenantID string, tags []string) error {
	docText := fmt.Sprintf("%s %s %s %s", title, content, domain, strings.Join(tags, " "))
	vec := GeneratePseudoEmbedding(docText, 384)

	doc := &models.IndexedDocument{
		ID:             fmt.Sprintf("doc-%s-%s", strings.ToLower(resourceType), uuid.New().String()[:8]),
		IndexName:      "statgate_global",
		ResourceID:     resourceID,
		ResourceType:   resourceType,
		Title:          title,
		Content:        content,
		Domain:         domain,
		Classification: classification,
		Owner:          "system",
		Tags:           tags,
		Vector:         vec,
		TenantID:       tenantID,
		WorkspaceID:    store.WorkspaceID(ctx),
		IndexedAt:      time.Now().UTC(),
	}
	return ui.store.IndexDocument(ctx, doc)
}
