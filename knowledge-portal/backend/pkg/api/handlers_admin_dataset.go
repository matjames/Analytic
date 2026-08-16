package api

import (
	"encoding/json"
	"net/http"
	"time"

	"knowledgeportal/pkg/model"
	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
)

// ─── Open data set management (P41) ────────────────────────────────────────

func ListDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListDatasets(r.Context(), actorTenant(r), r.URL.Query().Get("status"), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateDatasetHandler(w http.ResponseWriter, r *http.Request) {
	var d model.PublicDataset
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if d.Title == "" || d.Slug == "" || d.License == "" || d.Format == "" {
		writeError(w, http.StatusBadRequest, "title, slug, license and format are required")
		return
	}
	d.TenantID = actorTenant(r)
	d.UpdatedBy = actorID(r)
	if d.Status == "" {
		d.Status = "draft"
	}
	if d.Version == "" {
		d.Version = "1.0"
	}
	if err := store.CreateDataset(r.Context(), &d); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "dataset.created", "dataset", d.ID, map[string]interface{}{"title": d.Title, "format": d.Format})
	writeJSON(w, http.StatusCreated, d)
}

func GetDatasetHandler(w http.ResponseWriter, r *http.Request) {
	d, err := store.GetDataset(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "dataset not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func UpdateDatasetHandler(w http.ResponseWriter, r *http.Request) {
	d, err := store.GetDataset(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "dataset not found")
		return
	}
	var body model.PublicDataset
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	d.Title, d.Slug = body.Title, body.Slug
	d.Description, d.License, d.Format = body.Description, body.License, body.Format
	d.SizeBytes, d.DownloadURL = body.SizeBytes, body.DownloadURL
	d.Status, d.Version, d.Tags = body.Status, body.Version, body.Tags
	d.UpdatedBy = actorID(r)
	if err := store.UpdateDataset(r.Context(), d); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "dataset.updated", "dataset", d.ID, map[string]interface{}{"status": d.Status})
	writeJSON(w, http.StatusOK, d)
}

func DeleteDatasetHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := store.DeleteDataset(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "dataset.deleted", "dataset", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func PublishDatasetHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	d, err := store.GetDataset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "dataset not found")
		return
	}
	now := time.Now().UTC()
	d.PublishedAt = &now
	d.Status = "published"
	d.UpdatedBy = actorID(r)
	if err := store.UpdateDataset(r.Context(), d); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "dataset.published", "dataset", d.ID, map[string]interface{}{"title": d.Title, "version": d.Version})
	recordAudit(r, "dataset.published", "dataset", d.ID, map[string]interface{}{"title": d.Title})
	writeJSON(w, http.StatusOK, d)
}
