package api

import (
	"encoding/json"
	"net/http"

	"geointel/pkg/model"
	"geointel/pkg/store"
)

// ─── Layers (P44 GIS service) ────────────────────────────────────────────────

func ListLayersHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListLayers(r.Context(), actorTenant(r), r.URL.Query().Get("kind"), actorWorkspace(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateLayerHandler(w http.ResponseWriter, r *http.Request) {
	var l model.GeoLayer
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if l.Name == "" || l.Kind == "" {
		writeError(w, http.StatusBadRequest, "name and kind are required")
		return
	}
	l.TenantID = actorTenant(r)
	l.WorkspaceID = actorWorkspace(r)
	l.CreatedBy = actorID(r)
	if err := store.CreateLayer(r.Context(), &l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "spatial.layer.created", "geo_layer", l.ID, map[string]interface{}{"name": l.Name, "kind": l.Kind})
	writeJSON(w, http.StatusCreated, l)
}

func GetLayerHandler(w http.ResponseWriter, r *http.Request) {
	l, err := store.GetLayer(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	if !workspaceAllowsRead(l.WorkspaceID, actorWorkspace(r)) {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func DeleteLayerHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	l, err := store.GetLayer(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	if !workspaceAllowsRead(l.WorkspaceID, actorWorkspace(r)) {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	if err := store.DeleteLayer(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "spatial.layer.deleted", "geo_layer", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Features (P44 vector engine) ────────────────────────────────────────────

func ListLayerFeaturesHandler(w http.ResponseWriter, r *http.Request) {
	layerID := varsOf(r)["id"]
	if _, err := store.GetLayer(r.Context(), layerID); err != nil {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	items, err := store.ListFeatures(r.Context(), layerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateFeatureHandler ingests a vector feature (required centroid for spatial
// search), emits spatial.feature.updated and links it to the source object.
func CreateFeatureHandler(w http.ResponseWriter, r *http.Request) {
	layerID := varsOf(r)["id"]
	layer, err := store.GetLayer(r.Context(), layerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	var f model.GeoFeature
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if f.Name == "" || f.Geometry == "" {
		writeError(w, http.StatusBadRequest, "name and geometry are required")
		return
	}
	f.TenantID = actorTenant(r)
	f.LayerID = layerID
	f.CreatedBy = actorID(r)
	if f.Centroid == "" {
		f.Centroid = store.DeriveCentroid(f.Geometry)
	}
	if err := store.CreateFeature(r.Context(), &f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "spatial.feature.updated", "geo_feature", f.ID, map[string]interface{}{"layer_id": layerID, "name": f.Name})
	_ = store.CreateObjectLink(r.Context(), &model.ObjectLink{
		TenantID: f.TenantID, SourceType: "geo_feature", SourceID: f.ID, TargetType: "geo_layer", TargetID: layerID, Relationship: "belongs_to",
	})
	recordAudit(r, "spatial.feature.created", "geo_feature", f.ID, map[string]interface{}{"layer_id": layerID, "kind": layer.Kind})
	writeJSON(w, http.StatusCreated, f)
}

func GetFeatureHandler(w http.ResponseWriter, r *http.Request) {
	f, err := store.GetFeature(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func DeleteFeatureHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteFeature(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}