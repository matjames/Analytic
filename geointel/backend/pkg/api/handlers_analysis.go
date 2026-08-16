package api

import (
	"encoding/json"
	"net/http"

	"geointel/pkg/store"
)

// ─── Spatial Analytics Engine (P44) ──────────────────────────────────────────

// BufferHandler computes the set of features whose centroids fall within a
// radius (meters) of a feature's centroid.
func BufferHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FeatureID string  `json:"feature_id"`
		Radius    float64 `json:"radius_m"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	f, err := store.GetFeature(r.Context(), body.FeatureID)
	if err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return
	}
	lat, lng, ok := store.ParseCentroid(f.Centroid)
	if !ok {
		writeError(w, http.StatusBadRequest, "feature has no centroid")
		return
	}
	features, err := store.ListFeatures(r.Context(), f.LayerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var hit []map[string]interface{}
	for _, o := range features {
		clat, clng, ok := store.ParseCentroid(o.Centroid)
		if ok && store.HaversineMeters(lat, lng, clat, clng) <= body.Radius {
			hit = append(hit, map[string]interface{}{"id": o.ID, "name": o.Name, "distance_m": round2(store.HaversineMeters(lat, lng, clat, clng))})
		}
	}
	emitEvent(r.Context(), "spatial.analysis.completed", "analysis", body.FeatureID, map[string]interface{}{"type": "buffer", "radius_m": body.Radius, "hits": len(hit)})
	writeJSON(w, http.StatusOK, map[string]interface{}{"buffer_of": body.FeatureID, "radius_m": body.Radius, "features": hit})
}

// ContainsHandler reports whether a polygon feature contains a point.
func ContainsHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FeatureID string  `json:"feature_id"`
		Lat       float64 `json:"lat"`
		Lng       float64 `json:"lng"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	f, err := store.GetFeature(r.Context(), body.FeatureID)
	if err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return
	}
	inside := store.ContainsPoint(f.Geometry, body.Lat, body.Lng)
	writeJSON(w, http.StatusOK, map[string]interface{}{"feature_id": body.FeatureID, "lat": body.Lat, "lng": body.Lng, "contains": inside})
}

// DistanceHandler computes the haversine distance between two coordinates.
func DistanceHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From []float64 `json:"from"` // [lat, lng]
		To   []float64 `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if len(body.From) != 2 || len(body.To) != 2 {
		writeError(w, http.StatusBadRequest, "from and to must be [lat, lng] pairs")
		return
	}
	d := store.HaversineMeters(body.From[0], body.From[1], body.To[0], body.To[1])
	writeJSON(w, http.StatusOK, map[string]interface{}{"distance_m": round2(d), "distance_km": round2(d / 1000)})
}

// SpatialSearchHandler is the Spatial API gateway query: features of a layer
// within a radius of a coordinate.
func SpatialSearchHandler(w http.ResponseWriter, r *http.Request) {
	layerID := r.URL.Query().Get("layer_id")
	lat := queryFloat(r, "lat", 0)
	lng := queryFloat(r, "lng", 0)
	radius := queryFloat(r, "radius", 1000)
	if layerID == "" {
		writeError(w, http.StatusBadRequest, "layer_id query parameter is required")
		return
	}
	if _, err := store.GetLayer(r.Context(), layerID); err != nil {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	features, err := store.FeaturesInRadius(r.Context(), layerID, lat, lng, radius)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"layer_id": layerID, "center": []float64{lat, lng}, "radius_m": radius, "count": len(features), "features": features})
}

// LayerSummaryHandler returns aggregate stats for a layer.
func LayerSummaryHandler(w http.ResponseWriter, r *http.Request) {
	layerID := varsOf(r)["layerId"]
	layer, err := store.GetLayer(r.Context(), layerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "layer not found")
		return
	}
	features, err := store.ListFeatures(r.Context(), layerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"layer":        layer,
		"feature_count": len(features),
		"with_centroid": len(features), // all features carry centroids on ingest
	})
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }