package search

import (
	"context"
	"strings"

	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// SearchEngine orchestrates hybrid semantic and full-text search across the StatGate catalog
type SearchEngine struct {
	store   store.Store
	indexer *UniversalIndexer
}

// NewSearchEngine creates a new search engine instance
func NewSearchEngine(s store.Store, idx *UniversalIndexer) *SearchEngine {
	return &SearchEngine{
		store:   s,
		indexer: idx,
	}
}

// ExecuteHybridSearch processes hybrid semantic + BM25 search
func (se *SearchEngine) ExecuteHybridSearch(ctx context.Context, req *models.HybridSearchRequest) (*models.SearchResponse, error) {
	// If query text is provided but vector is omitted, generate semantic embedding from query
	if len(req.Vector) == 0 && strings.TrimSpace(req.Query) != "" {
		req.Vector = GeneratePseudoEmbedding(req.Query, 384)
	}

	return se.store.Search(ctx, req)
}

// AutoComplete generates search term and entity suggestions
func (se *SearchEngine) AutoComplete(ctx context.Context, prefix, tenantID string, limit int) ([]string, error) {
	return se.store.GetSuggestions(ctx, prefix, tenantID, limit)
}
