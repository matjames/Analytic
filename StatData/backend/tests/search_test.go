package tests

import (
	"context"
	"testing"

	"statdata-backend/internal/models"
	"statdata-backend/internal/search"
	"statdata-backend/internal/store"
)

func TestHybridSearchAndIndexing(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemStore()
	indexer := search.NewUniversalIndexer(memStore)
	engine := search.NewSearchEngine(memStore, indexer)

	// 1. Index additional documents
	err := indexer.IndexGeneric(
		ctx,
		"res-malaria-study",
		"RESEARCH_STUDY",
		"National Malaria Prevalence Survey 2026",
		"Comprehensive longitudinal survey of malaria incidence and seasonal transmission across 12 provinces.",
		"epidemiology",
		"PUBLIC",
		"default",
		[]string{"malaria", "epidemiology", "survey", "health"},
	)
	if err != nil {
		t.Fatalf("IndexGeneric failed: %v", err)
	}

	// 2. Hybrid Keyword & Semantic Search
	req := &models.HybridSearchRequest{
		Query: "malaria epidemiology survey",
		Alpha: 0.5,
		Limit: 10,
	}

	resp, err := engine.ExecuteHybridSearch(ctx, req)
	if err != nil {
		t.Fatalf("ExecuteHybridSearch failed: %v", err)
	}

	if resp.TotalHits == 0 || len(resp.Results) == 0 {
		t.Fatalf("Expected search results for malaria query, got 0")
	}

	topResult := resp.Results[0]
	if topResult.ResourceID != "res-malaria-study" {
		t.Errorf("Expected top result 'res-malaria-study', got '%s'", topResult.ResourceID)
	}
	if topResult.Score <= 0.0 {
		t.Errorf("Expected positive hybrid score, got %f", topResult.Score)
	}

	// 3. Autocomplete Suggestions
	suggestions, err := engine.AutoComplete(ctx, "National", "default", 5)
	if err != nil {
		t.Fatalf("AutoComplete failed: %v", err)
	}
	if len(suggestions) == 0 {
		t.Errorf("Expected autocomplete suggestions for 'National', got 0")
	}
}
