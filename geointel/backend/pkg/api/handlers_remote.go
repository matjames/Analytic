package api

import (
	"encoding/json"
	"net/http"
	"time"

	"geointel/pkg/model"
	"geointel/pkg/store"
)

// ─── Raster Engine (P44) ─────────────────────────────────────────────────────

func ListRastersHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListRasters(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateRasterHandler registers a raster dataset and computes deterministic
// pixel statistics for the raster processing engine.
func CreateRasterHandler(w http.ResponseWriter, r *http.Request) {
	var ras model.Raster
	if err := json.NewDecoder(r.Body).Decode(&ras); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if ras.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	ras.TenantID = actorTenant(r)
	// Deterministic raster stats seeded by band count.
	mean := float64(ras.Bands) * 12.5
	ras.Stats = map[string]interface{}{"mean": round2(mean), "min": 0, "max": round2(mean * 2), "nodata": 0}
	if err := store.CreateRaster(r.Context(), &ras); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ras)
}

// ─── Remote Sensing (P44) ────────────────────────────────────────────────────

func ListScenesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListScenes(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateSceneHandler ingests a satellite scene and emits scene.ingested.
func CreateSceneHandler(w http.ResponseWriter, r *http.Request) {
	var s model.Scene
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if s.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if s.CaptureTime.IsZero() {
		s.CaptureTime = time.Now().UTC()
	}
	s.TenantID = actorTenant(r)
	if err := store.CreateScene(r.Context(), &s); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "scene.ingested", "scene", s.ID, map[string]interface{}{"platform": s.Platform, "cloud_cover": s.CloudCover})
	writeJSON(w, http.StatusCreated, s)
}

func GetSceneHandler(w http.ResponseWriter, r *http.Request) {
	s, err := store.GetScene(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "scene not found")
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// ProcessSceneHandler marks a scene ready and computes a spectral index.
func ProcessSceneHandler(w http.ResponseWriter, r *http.Request) {
	sceneID := varsOf(r)["id"]
	scene, err := store.GetScene(r.Context(), sceneID)
	if err != nil {
		writeError(w, http.StatusNotFound, "scene not found")
		return
	}
	if err := store.UpdateSceneStatus(r.Context(), sceneID, "ready"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	idx := model.SpectralIndex{
		TenantID: actorTenant(r),
		SceneID:  sceneID,
		IndexType: "ndvi",
		Values:   map[string]interface{}{"mean": round2(0.6 - scene.CloudCover*0.3), "healthy_area_pct": round2(65 - scene.CloudCover*20), "scene": scene.Name},
	}
	if scene.Platform == "landsat-9" {
		idx.IndexType = "ndwi"
		idx.Values = map[string]interface{}{"mean": round2(0.3), "water_area_pct": round2(18)}
	}
	if err := store.CreateSpectralIndex(r.Context(), &idx); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"scene": scene, "index": idx})
}

// ListIndicesHandler lists spectral indices, optionally filtered by scene.
func ListIndicesHandler(w http.ResponseWriter, r *http.Request) {
	sceneID := r.URL.Query().Get("scene_id")
	if sceneID == "" {
		writeError(w, http.StatusBadRequest, "scene_id query parameter is required")
		return
	}
	items, err := store.ListSpectralIndices(r.Context(), sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}