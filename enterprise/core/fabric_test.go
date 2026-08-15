package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPhaseIXFabricBootstrap(t *testing.T) {
	bootstrapFabric()

	fabricStore.RLock()
	appCount := len(fabricStore.applications)
	catCount := len(fabricStore.catalogue)
	dictCount := len(fabricStore.dictionary)
	knowCount := len(fabricStore.knowledge)
	semCount := len(fabricStore.mappings)
	relCount := len(fabricStore.relations)
	fabricStore.RUnlock()

	if appCount < 5 {
		t.Fatalf("Expected at least 5 registered applications, got %d", appCount)
	}
	if catCount < 3 {
		t.Fatalf("Expected at least 3 catalogue datasets, got %d", catCount)
	}
	if dictCount < 3 {
		t.Fatalf("Expected at least 3 dictionary variables, got %d", dictCount)
	}
	if knowCount < 4 {
		t.Fatalf("Expected at least 4 governed knowledge items, got %d", knowCount)
	}
	if semCount < 4 {
		t.Fatalf("Expected at least 4 semantic mappings, got %d", semCount)
	}
	if relCount < 5 {
		t.Fatalf("Expected at least 5 fabric relationships, got %d", relCount)
	}
}

func TestCanonicalObjectResolution(t *testing.T) {
	bootstrapFabric()

	// Test resolving existing canonical object
	obj := resolveCanonicalObject("project", "PROJ-001")
	if obj.ObjectID != "PROJ-001" || obj.SourceApplication != "pms" {
		t.Fatalf("Failed to resolve PROJ-001 correctly: %+v", obj)
	}
	if obj.Classification != "verified" {
		t.Fatalf("Expected verified classification, got %s", obj.Classification)
	}

	// Test dynamic fallback resolution
	dyn := resolveCanonicalObject("custom_entity", "CUST-999")
	if dyn.ObjectID != "CUST-999" || dyn.ObjectType != "custom_entity" {
		t.Fatalf("Failed dynamic canonical object generation: %+v", dyn)
	}
	if !strings.Contains(dyn.CanonicalID, "tenant_uganda_inst") {
		t.Fatalf("Dynamic canonical ID missing tenant: %s", dyn.CanonicalID)
	}
}

func TestFabricRelationshipGraph(t *testing.T) {
	bootstrapFabric()

	graph := buildFabricGraph("project", "PROJ-001", 2)
	nodesCount, ok := graph["nodes_count"].(int)
	if !ok || nodesCount < 2 {
		t.Fatalf("Expected at least 2 connected nodes in graph, got %v", graph["nodes_count"])
	}

	edgesCount, ok := graph["edges_count"].(int)
	if !ok || edgesCount < 1 {
		t.Fatalf("Expected at least 1 edge in graph, got %v", graph["edges_count"])
	}
}

func TestGovernedKnowledgeHierarchy(t *testing.T) {
	bootstrapFabric()

	ts := setupTestRouter()

	// 1. Verify filtering by classification
	w := ts.do("GET", "/api/fabric/knowledge?classification=verified", "")
	if w.Code != 200 {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var res struct {
		Count     int                     `json:"count"`
		Knowledge []GovernedKnowledgeItem `json:"knowledge"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	for _, k := range res.Knowledge {
		if k.Classification != "verified" {
			t.Fatalf("Expected only verified knowledge, found %s", k.Classification)
		}
	}
}

func TestDataCatalogueAndDictionary(t *testing.T) {
	bootstrapFabric()

	ts := setupTestRouter()

	// 1. Test Catalogue endpoint
	w := ts.do("GET", "/api/fabric/catalogue", "")
	if w.Code != 200 {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var catRes struct {
		Count    int                `json:"count"`
		Datasets []CatalogueDataset `json:"datasets"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &catRes); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	if catRes.Count < 3 {
		t.Fatalf("Expected at least 3 catalogue datasets, got %d", catRes.Count)
	}

	// 2. Test Semantic Term Resolution
	wSem := ts.do("GET", "/api/fabric/semantic/resolve?term=Health+Facility", "")
	if wSem.Code != 200 {
		t.Fatalf("Expected 200 OK for semantic resolve, got %d", wSem.Code)
	}

	var semRes struct {
		CanonicalConcept string `json:"canonical_concept"`
	}
	if err := json.Unmarshal(wSem.Body.Bytes(), &semRes); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	if semRes.CanonicalConcept != "facility" {
		t.Fatalf("Expected canonical concept 'facility', got '%s'", semRes.CanonicalConcept)
	}
}

func TestEventDrivenFabricUpdates(t *testing.T) {
	bootstrapFabric()

	ev := DomainEvent{
		ID:         "ev_test_survey_99",
		EventType:  "survey.created",
		Source:     "statcollect",
		ObjectType: "survey",
		ObjectID:   "SUR-999",
		Actor:      "field_director",
		TenantID:   "tenant_uganda_inst",
		Payload: map[string]interface{}{
			"name": "2026 Maternal Health Survey",
		},
		Timestamp: nowUTC(),
	}

	processFabricForEvent(ev)

	// Verify canonical object was created in fabric store
	obj := resolveCanonicalObject("survey", "SUR-999")
	if obj.ObjectID != "SUR-999" || obj.Title != "2026 Maternal Health Survey" {
		t.Fatalf("Event failed to update canonical object in fabric: %+v", obj)
	}
}
