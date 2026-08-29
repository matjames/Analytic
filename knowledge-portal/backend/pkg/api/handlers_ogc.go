package api

import (
	"net/http"
	"time"

	"knowledgeportal/pkg/model"
	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
)

// ─── OGC API - Features - Part 1: Core ─────────────────────────────────────
//
// These endpoints expose the Open Data Portal catalog through the OGC API
// - Features standard so that GIS clients (QGIS, ArcGIS, GeoServer consumers)
// can discover and consume published statistical datasets.
//
//   - GET /api/ogc/collections          → collection catalogue
//   - GET /api/ogc/collections/{id}     → single collection metadata
//   - GET /api/ogc/collections/{id}/items → GeoJSON feature set
//
// Each published dataset is surfaced as both a collection and a single
// GeoJSON feature (geometry null, dataset metadata in properties). This is the
// standard "catalog as feature collection" interpretation used by data portals
// that do not store server-side geometries.

// ogcLink is an OGC/JSON-schema link object.
type ogcLink struct {
	Href  string `json:"href"`
	Rel   string `json:"rel"`
	Type  string `json:"type,omitempty"`
	Title string `json:"title,omitempty"`
}

// ogcExtent describes the spatial/temporal bounds of a collection.
type ogcExtent struct {
	Spatial struct {
		BBox [][]float64 `json:"bbox"`
	} `json:"spatial"`
	Temporal struct {
		Interval [][]*time.Time `json:"interval"`
	} `json:"temporal"`
}

// ogcCollection is a single OGC collection (dataset).
type ogcCollection struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Extent      ogcExtent `json:"extent"`
	Links       []ogcLink `json:"links"`
}

// ogcCollectionsResponse is the root catalogue response.
type ogcCollectionsResponse struct {
	Links       []ogcLink       `json:"links"`
	Collections []ogcCollection `json:"collections"`
}

// ogcFeature is a GeoJSON feature wrapping dataset metadata.
type ogcFeature struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Geometry   interface{}            `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

// ogcFeatureCollection is the GeoJSON response of /items.
type ogcFeatureCollection struct {
	Type           string       `json:"type"`
	Features       []ogcFeature `json:"features"`
	NumberMatched  int          `json:"numberMatched"`
	NumberReturned int          `json:"numberReturned"`
	Links          []ogcLink    `json:"links"`
}

// CollectionFromDataset maps a published dataset into OGC collection metadata.
func CollectionFromDataset(d model.PublicDataset, baseURL string) ogcCollection {
	c := ogcCollection{
		ID:          d.ID,
		Title:       d.Title,
		Description: d.Description,
	}
	c.Extent.Spatial.BBox = [][]float64{{-180.0, -90.0, 180.0, 90.0}}
	if d.PublishedAt != nil {
		c.Extent.Temporal.Interval = [][]*time.Time{{d.PublishedAt, nil}}
	} else {
		now := time.Now()
		c.Extent.Temporal.Interval = [][]*time.Time{{&now, nil}}
	}
	c.Links = []ogcLink{
		{Href: baseURL + "/api/ogc/collections/" + d.ID, Rel: "self", Type: "application/json", Title: d.Title},
		{Href: baseURL + "/api/ogc/collections/" + d.ID + "/items", Rel: "items", Type: "application/geo+json", Title: d.Title + " (GeoJSON items)"},
	}
	return c
}

// FeatureFromDataset maps a published dataset into a GeoJSON feature (catalog
// interpretation: dataset metadata as properties, geometry left undefined).
func FeatureFromDataset(d model.PublicDataset) ogcFeature {
	props := map[string]interface{}{
		"title":        d.Title,
		"description":  d.Description,
		"license":      d.License,
		"format":       d.Format,
		"download_url": d.DownloadURL,
		"source_app":   d.SourceApp,
		"version":      d.Version,
		"tags":         d.Tags,
		"created_at":   d.CreatedAt.Format(time.RFC3339),
	}
	if d.PublishedAt != nil {
		props["published_at"] = d.PublishedAt.Format(time.RFC3339)
	}
	return ogcFeature{
		Type:       "Feature",
		ID:         d.ID,
		Geometry:   nil,
		Properties: props,
	}
}
// OGCCollectionsHandler returns the OGC API - Features collection catalogue.
func OGCCollectionsHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeError(w, http.StatusServiceUnavailable, "knowledge portal store is not initialised")
		return
	}
	datasets, err := store.ListDatasets(r.Context(), publicTenant(r), "published", "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	base := requestBaseURL(r)
	resp := ogcCollectionsResponse{
		Links: []ogcLink{
			{Href: base + "/api/ogc/collections", Rel: "self", Type: "application/json", Title: "OGC API - Features collections"},
		},
		Collections: make([]ogcCollection, 0, len(datasets)),
	}
	for _, d := range datasets {
		resp.Collections = append(resp.Collections, CollectionFromDataset(d, base))
	}
	writeJSON(w, http.StatusOK, resp)
}

// OGCCollectionHandler returns metadata for a single dataset collection.
func OGCCollectionHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeError(w, http.StatusServiceUnavailable, "knowledge portal store is not initialised")
		return
	}
	id := mux.Vars(r)["id"]
	d, err := store.GetDataset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "collection not found: "+id)
		return
	}
	if d.Status != "published" {
		writeError(w, http.StatusNotFound, "collection not found: "+id)
		return
	}
	writeJSON(w, http.StatusOK, CollectionFromDataset(*d, requestBaseURL(r)))
}

// OGCCollectionItemsHandler returns the published datasets (catalog as a
// feature collection) for a given collection.
func OGCCollectionItemsHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeError(w, http.StatusServiceUnavailable, "knowledge portal store is not initialised")
		return
	}
	id := mux.Vars(r)["id"]
	d, err := store.GetDataset(r.Context(), id)
	if err != nil || d.Status != "published" {
		writeError(w, http.StatusNotFound, "collection not found: "+id)
		return
	}

	base := requestBaseURL(r)
	// Catalog interpretation: one feature per item — here the item itself.
	features := []ogcFeature{FeatureFromDataset(*d)}
	resp := ogcFeatureCollection{
		Type:           "FeatureCollection",
		Features:       features,
		NumberMatched:  len(features),
		NumberReturned: len(features),
		Links: []ogcLink{
			{Href: base + "/api/ogc/collections/" + id + "/items", Rel: "self", Type: "application/geo+json", Title: d.Title + " items"},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

// requestBaseURL derives the public base URL of this service from the request,
// falling back to a scheme-relative default when headers are absent.
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	if host == "" {
		host = "localhost:8099"
	}
	return scheme + "://" + host
}