package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestFabricTenantResourceAccess(t *testing.T) {
	if !canAccessTenantResource("tenant-alpha", "tenant-alpha", "viewer") {
		t.Fatal("expected matching tenant access")
	}
	if canAccessTenantResource("tenant-alpha", "tenant-beta", "viewer") {
		t.Fatal("expected cross-tenant access denial")
	}
	if canAccessTenantResource("", "tenant-alpha", "viewer") {
		t.Fatal("expected tenantless resource denial for tenant user")
	}
	if !canAccessTenantResource("tenant-alpha", "tenant-beta", "platform_admin") {
		t.Fatal("expected platform administrator access")
	}
}

func TestFabricCreateRelationshipDerivesAuthenticatedTenant(t *testing.T) {
	ts := setupTestRouter()

	body := `{"from_type":"project","from_id":"P-ALPHA","to_type":"dataset","to_id":"D-ALPHA","relation_type":"evidences"}`
	w := doFabricAs(ts, "POST", "/api/fabric/relationships", body, "tenant-alpha", "analyst")
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var rel FabricRelationship
	if err := json.Unmarshal(w.Body.Bytes(), &rel); err != nil {
		t.Fatalf("failed to parse relationship: %v", err)
	}
	if rel.TenantID != "tenant-alpha" {
		t.Fatalf("expected authenticated tenant to be applied, got %q", rel.TenantID)
	}
}

func TestFabricCreateRelationshipRejectsCrossTenantForTenantUser(t *testing.T) {
	ts := setupTestRouter()

	body := `{"from_type":"project","from_id":"P-BETA","to_type":"dataset","to_id":"D-BETA","relation_type":"evidences","tenant_id":"tenant-gamma"}`
	w := doFabricAs(ts, "POST", "/api/fabric/relationships", body, "tenant-beta", "analyst")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFabricObjectsAndGraphAreTenantScoped(t *testing.T) {
	ts := setupTestRouter()

	bootstrapFabric()
	addFabricTestObject(CanonicalObject{
		CanonicalID:       "tenant-alpha:pms:project:P-ALPHA-GRAPH",
		ObjectID:          "P-ALPHA-GRAPH",
		ObjectType:        "project",
		SourceApplication: "pms",
		TenantID:          "tenant-alpha",
		Title:             "Alpha Graph Project",
		CreatedBy:         "tester",
		CreatedAt:         nowUTC(),
		UpdatedAt:         nowUTC(),
		Status:            "active",
		Classification:    "verified",
		Sensitivity:       "official",
		CanonicalURL:      "/workspace/project/P-ALPHA-GRAPH",
	})
	createFabricRelationship(FabricRelationship{
		ID:           "rel_alpha_graph_scope",
		FromType:     "project",
		FromID:       "P-ALPHA-GRAPH",
		ToType:       "dataset",
		ToID:         "D-ALPHA-GRAPH",
		RelationType: "evidences",
		TenantID:     "tenant-alpha",
		CreatedBy:    "tester",
	})
	createFabricRelationship(FabricRelationship{
		ID:           "rel_beta_graph_scope",
		FromType:     "project",
		FromID:       "P-ALPHA-GRAPH",
		ToType:       "dataset",
		ToID:         "D-BETA-GRAPH",
		RelationType: "evidences",
		TenantID:     "tenant-beta",
		CreatedBy:    "tester",
	})

	listW := doFabricAs(ts, "GET", "/api/fabric/objects?type=project&q=Alpha+Graph", "", "tenant-alpha", "analyst")
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listW.Code, listW.Body.String())
	}
	var objectsResp struct {
		Count   int               `json:"count"`
		Objects []CanonicalObject `json:"objects"`
	}
	if err := json.Unmarshal(listW.Body.Bytes(), &objectsResp); err != nil {
		t.Fatalf("failed to parse objects: %v", err)
	}
	if objectsResp.Count != 1 || objectsResp.Objects[0].TenantID != "tenant-alpha" {
		t.Fatalf("expected only tenant-alpha object, got %+v", objectsResp)
	}

	graphW := doFabricAs(ts, "GET", "/api/fabric/relationships/graph?type=project&id=P-ALPHA-GRAPH", "", "tenant-alpha", "analyst")
	if graphW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", graphW.Code, graphW.Body.String())
	}
	var graphResp struct {
		EdgesCount int                  `json:"edges_count"`
		Edges      []FabricRelationship `json:"edges"`
	}
	if err := json.Unmarshal(graphW.Body.Bytes(), &graphResp); err != nil {
		t.Fatalf("failed to parse graph: %v", err)
	}
	if graphResp.EdgesCount != 1 || graphResp.Edges[0].TenantID != "tenant-alpha" {
		t.Fatalf("expected only tenant-alpha edge, got %+v", graphResp)
	}

	platformW := doFabricAs(ts, "GET", "/api/fabric/relationships/graph?type=project&id=P-ALPHA-GRAPH", "", "tenant-gamma", "platform_admin")
	if platformW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", platformW.Code, platformW.Body.String())
	}
	if err := json.Unmarshal(platformW.Body.Bytes(), &graphResp); err != nil {
		t.Fatalf("failed to parse platform graph: %v", err)
	}
	if graphResp.EdgesCount < 2 {
		t.Fatalf("expected platform graph to include cross-tenant edges, got %+v", graphResp)
	}
}

func addFabricTestObject(obj CanonicalObject) {
	fabricStore.Lock()
	defer fabricStore.Unlock()
	fabricStore.objects[obj.CanonicalID] = obj
}

func doFabricAs(ts *ginTestRouter, method, path, body, tenantID, role string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+fabricTestToken(tenantID, role))
	w := httptest.NewRecorder()
	ts.engine.ServeHTTP(w, req)
	return w
}

func fabricTestToken(tenantID, role string) string {
	now := time.Now()
	claims := map[string]interface{}{
		"sub":       "fabric-user",
		"tenant_id": tenantID,
		"org_id":    "org-national",
		"role":      role,
		"email":     "fabric-user@statgate.local",
		"iat":       now.Unix(),
		"nbf":       now.Add(-30 * time.Second).Unix(),
		"exp":       now.Add(24 * time.Hour).Unix(),
		"iss":       "statgate-registry",
		"aud":       "statgate",
	}
	return signTestJWT(testJWTSecret, "HS256", claims)
}
