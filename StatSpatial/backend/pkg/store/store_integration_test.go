package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"statspatial/pkg/model"

	"github.com/matjames/statgate-lib/auth"
)

func TestPostgresWorkspaceOwnershipCertification(t *testing.T) {
	dsn := os.Getenv("STATSPATIAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STATSPATIAL_TEST_DATABASE_URL is not set")
	}

	if err := Init(dsn); err != nil {
		t.Fatalf("init postgres store: %v", err)
	}

	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	tenantID := "tenant-cert-" + suffix
	workspaceOne := "workspace-one-" + suffix
	workspaceTwo := "workspace-two-" + suffix
	layerOneID := "layer-one-" + suffix
	layerTwoID := "layer-two-" + suffix
	featureOneID := "feature-one-" + suffix
	featureTwoID := "feature-two-" + suffix
	targetID := "dataset-shared-" + suffix

	t.Cleanup(func() {
		if DB() == nil {
			return
		}
		_, _ = DB().Exec(`DELETE FROM geo_features WHERE id IN ($1,$2)`, featureOneID, featureTwoID)
		_, _ = DB().Exec(`DELETE FROM geo_layers WHERE id IN ($1,$2)`, layerOneID, layerTwoID)
		_, _ = DB().Exec(`DELETE FROM spatial_index WHERE target_id = $1`, targetID)
	})

	ctxOne := workspaceContext(tenantID, workspaceOne)
	ctxTwo := workspaceContext(tenantID, workspaceTwo)

	layerOne, err := CreateGeoLayer(ctxOne, model.GeoLayer{
		ID:           layerOneID,
		Name:         "Certification Layer One",
		GeometryType: "polygon",
		Source:       "integration-test",
	})
	if err != nil {
		t.Fatalf("create workspace one layer: %v", err)
	}
	if layerOne.TenantID != tenantID || layerOne.WorkspaceID != workspaceOne {
		t.Fatalf("layer ownership mismatch: got tenant=%q workspace=%q", layerOne.TenantID, layerOne.WorkspaceID)
	}

	if _, err := CreateGeoLayer(ctxTwo, model.GeoLayer{
		ID:           layerTwoID,
		Name:         "Certification Layer Two",
		GeometryType: "polygon",
		Source:       "integration-test",
	}); err != nil {
		t.Fatalf("create workspace two layer: %v", err)
	}

	if _, err := CreateGeoFeature(ctxOne, model.GeoFeature{
		ID:       featureOneID,
		LayerID:  layerOneID,
		Name:     "Certification Feature One",
		Geometry: `{"type":"Point","coordinates":[32.58,0.35]}`,
	}); err != nil {
		t.Fatalf("create workspace one feature: %v", err)
	}
	if _, err := CreateGeoFeature(ctxTwo, model.GeoFeature{
		ID:       featureTwoID,
		LayerID:  layerTwoID,
		Name:     "Certification Feature Two",
		Geometry: `{"type":"Point","coordinates":[32.59,0.36]}`,
	}); err != nil {
		t.Fatalf("create workspace two feature: %v", err)
	}

	if err := CreateSpatialIndex(ctxOne, model.SpatialIndex{AdminUnitID: "au-kampala", TargetType: "dataset", TargetID: targetID, Lat: 0.35, Lng: 32.58}); err != nil {
		t.Fatalf("create workspace one spatial index: %v", err)
	}
	if err := CreateSpatialIndex(ctxTwo, model.SpatialIndex{AdminUnitID: "au-kampala", TargetType: "dataset", TargetID: targetID, Lat: 0.36, Lng: 32.59}); err != nil {
		t.Fatalf("create workspace two spatial index with same target: %v", err)
	}

	layersOne, err := ListGeoLayers(ctxOne)
	if err != nil {
		t.Fatalf("list workspace one layers: %v", err)
	}
	assertContainsLayer(t, layersOne, layerOneID)
	assertNotContainsLayer(t, layersOne, layerTwoID)

	featuresOne, err := ListGeoFeatures(ctxOne, "", "")
	if err != nil {
		t.Fatalf("list workspace one features: %v", err)
	}
	assertContainsFeature(t, featuresOne, featureOneID)
	assertNotContainsFeature(t, featuresOne, featureTwoID)

	indexOne, err := ListSpatialIndex(ctxOne, "au-kampala", "dataset")
	if err != nil {
		t.Fatalf("list workspace one spatial index: %v", err)
	}
	assertContainsSpatialTarget(t, indexOne, targetID, workspaceOne)
	assertNotContainsSpatialTarget(t, indexOne, targetID, workspaceTwo)
}

func workspaceContext(tenantID, workspaceID string) context.Context {
	ctx := context.WithValue(context.Background(), auth.ContextKeyTenant, tenantID)
	return context.WithValue(ctx, "workspace_id", workspaceID)
}

func assertContainsLayer(t *testing.T, layers []model.GeoLayer, id string) {
	t.Helper()
	for _, layer := range layers {
		if layer.ID == id {
			return
		}
	}
	t.Fatalf("expected layer %s in result", id)
}

func assertNotContainsLayer(t *testing.T, layers []model.GeoLayer, id string) {
	t.Helper()
	for _, layer := range layers {
		if layer.ID == id {
			t.Fatalf("did not expect layer %s in result", id)
		}
	}
}

func assertContainsFeature(t *testing.T, features []model.GeoFeature, id string) {
	t.Helper()
	for _, feature := range features {
		if feature.ID == id {
			return
		}
	}
	t.Fatalf("expected feature %s in result", id)
}

func assertNotContainsFeature(t *testing.T, features []model.GeoFeature, id string) {
	t.Helper()
	for _, feature := range features {
		if feature.ID == id {
			t.Fatalf("did not expect feature %s in result", id)
		}
	}
}

func assertContainsSpatialTarget(t *testing.T, items []model.SpatialIndex, targetID, workspaceID string) {
	t.Helper()
	for _, item := range items {
		if item.TargetID == targetID && item.WorkspaceID == workspaceID {
			return
		}
	}
	t.Fatalf("expected spatial target %s in workspace %s", targetID, workspaceID)
}

func assertNotContainsSpatialTarget(t *testing.T, items []model.SpatialIndex, targetID, workspaceID string) {
	t.Helper()
	for _, item := range items {
		if item.TargetID == targetID && item.WorkspaceID == workspaceID {
			t.Fatalf("did not expect spatial target %s in workspace %s", targetID, workspaceID)
		}
	}
}
