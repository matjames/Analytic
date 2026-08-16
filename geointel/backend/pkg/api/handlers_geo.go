package api

import (
	"encoding/json"
	"net/http"

	"geointel/pkg/model"
	"geointel/pkg/store"
)

// ─── Map Tile Service (P44) ──────────────────────────────────────────────────

func ListTilesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListMapTiles(r.Context(), actorTenant(r), r.URL.Query().Get("layer_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// RegisterTileHandler registers a generated XYZ tile and emits tile.registered.
func RegisterTileHandler(w http.ResponseWriter, r *http.Request) {
	var t model.MapTile
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if t.LayerID == "" || t.Name == "" {
		writeError(w, http.StatusBadRequest, "layer_id and name are required")
		return
	}
	if _, err := store.GetLayer(r.Context(), t.LayerID); err != nil {
		writeError(w, http.StatusBadRequest, "layer does not exist")
		return
	}
	t.TenantID = actorTenant(r)
	if err := store.CreateMapTile(r.Context(), &t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "tile.registered", "map_tile", t.ID, map[string]interface{}{"layer_id": t.LayerID, "z": t.Z, "x": t.X, "y": t.Y})
	writeJSON(w, http.StatusCreated, t)
}

// ─── Geocoding Service (P44) ─────────────────────────────────────────────────

// GeocodeHandler resolves a free-text query to coordinates deterministically
// (attribute-based approximation) and caches the result.
func GeocodeHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	lat, lng, conf := deterministicGeocode(body.Query)
	g := model.GeocodeResult{
		TenantID:   actorTenant(r),
		Query:      body.Query,
		Address:    body.Query,
		Lat:        lat,
		Lng:        lng,
		Confidence: conf,
	}
	_ = store.SaveGeocode(r.Context(), &g)
	writeJSON(w, http.StatusOK, g)
}

func ReverseGeocodeHandler(w http.ResponseWriter, r *http.Request) {
	lat := queryFloat(r, "lat", 0)
	lng := queryFloat(r, "lng", 0)
	result := model.GeocodeResult{
		TenantID:   actorTenant(r),
		Address:    "Reverse: " + f2s(lat) + ", " + f2s(lng),
		Lat:        lat,
		Lng:        lng,
		Confidence: 0.5,
	}
	_ = store.SaveGeocode(r.Context(), &result)
	writeJSON(w, http.StatusOK, result)
}

func GeocodeCacheHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListGeocodeCache(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Cross-app object linkage + overview ─────────────────────────────────────

func CreateLinkHandler(w http.ResponseWriter, r *http.Request) {
	var l model.ObjectLink
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if l.SourceType == "" || l.SourceID == "" || l.TargetType == "" || l.TargetID == "" {
		writeError(w, http.StatusBadRequest, "source and target are required")
		return
	}
	l.TenantID = actorTenant(r)
	if err := store.CreateObjectLink(r.Context(), &l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "object.link.created", "object_link", l.ID,
		map[string]interface{}{"source": l.SourceType + ":" + l.SourceID, "target": l.TargetType + ":" + l.TargetID})
	writeJSON(w, http.StatusCreated, l)
}

func ListLinksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	objectType, objectID := q.Get("type"), q.Get("id")
	if objectType == "" || objectID == "" {
		writeError(w, http.StatusBadRequest, "type and id query parameters are required")
		return
	}
	links, err := store.ListObjectLinks(r.Context(), objectType, objectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, links)
}

func SummaryHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service":   SourceApplication,
		"tenant_id": actorTenant(r),
		"endpoints": []string{"/layers", "/features", "/analysis/*", "/scenes", "/rasters", "/drones", "/flights", "/tiles", "/geocode"},
	})
}