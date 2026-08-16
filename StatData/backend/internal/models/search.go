package models

import "time"

// SearchIndex represents an indexed corpus partition
type SearchIndex struct {
	ID            string    `json:"id"`
	IndexName     string    `json:"index_name"`
	DocumentCount int64     `json:"document_count"`
	Dimension     int       `json:"dimension"` // Vector embedding dimension, e.g. 384 or 768
	Status        string    `json:"status"` // ACTIVE, REINDEXING, ERROR
	TenantID      string    `json:"tenant_id"`
	LastIndexedAt time.Time `json:"last_indexed_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// IndexedDocument represents a universal searchable entity across all StatGate modules
type IndexedDocument struct {
	ID             string                 `json:"id"`
	IndexName      string                 `json:"index_name"`
	ResourceID     string                 `json:"resource_id"`
	ResourceType   string                 `json:"resource_type"` // DATASET, PIPELINE, MODEL, EXPERIMENT, REPORT, SURVEY, GIS_LAYER
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	Domain         string                 `json:"domain"`
	Classification string                 `json:"classification"` // PUBLIC, INTERNAL, CONFIDENTIAL, RESTRICTED
	Owner          string                 `json:"owner"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Vector         []float32              `json:"vector,omitempty"`
	TenantID       string                 `json:"tenant_id"`
	IndexedAt      time.Time              `json:"indexed_at"`
}

// HybridSearchRequest represents a multi-modal search query combining text, vector, and filters
type HybridSearchRequest struct {
	Query          string                 `json:"query"`
	Vector         []float32              `json:"vector,omitempty"`
	ResourceType   string                 `json:"resource_type,omitempty"`
	Domain         string                 `json:"domain,omitempty"`
	Classification string                 `json:"classification,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	TenantID       string                 `json:"tenant_id,omitempty"`
	Alpha          float64                `json:"alpha"` // 0.0 = full keyword/BM25, 1.0 = full semantic/vector, 0.5 = balanced hybrid
	Limit          int                    `json:"limit"`
	Offset         int                    `json:"offset"`
}

// SearchResultItem represents a ranked match in a search response
type SearchResultItem struct {
	DocumentID     string                 `json:"document_id"`
	ResourceID     string                 `json:"resource_id"`
	ResourceType   string                 `json:"resource_type"`
	Title          string                 `json:"title"`
	Snippet        string                 `json:"snippet"`
	Domain         string                 `json:"domain"`
	Classification string                 `json:"classification"`
	Score          float64                `json:"score"`
	BM25Score      float64                `json:"bm25_score"`
	VectorScore    float64                `json:"vector_score"`
	Highlights     []string               `json:"highlights,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	IndexedAt      time.Time              `json:"indexed_at"`
}

// SearchResponse encapsulates search results and facet distributions
type SearchResponse struct {
	Query         string             `json:"query"`
	TotalHits     int64              `json:"total_hits"`
	ExecutionMs   int64              `json:"execution_ms"`
	Results       []SearchResultItem `json:"results"`
	Facets        []FacetResult      `json:"facets,omitempty"`
	Suggestions   []string           `json:"suggestions,omitempty"`
}

// FacetResult groups hit distributions across dimensions
type FacetResult struct {
	Field  string            `json:"field"`
	Counts map[string]int64  `json:"counts"`
}

// SavedSearch records user search bookmarks and alerts
type SavedSearch struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	Filters   map[string]interface{} `json:"filters,omitempty"`
	TenantID  string    `json:"tenant_id"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}
