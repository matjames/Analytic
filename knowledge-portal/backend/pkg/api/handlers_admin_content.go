package api

import (
	"encoding/json"
	"net/http"
	"time"

	"knowledgeportal/pkg/model"
	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
)

// ─── Content management (P17 CMS) ──────────────────────────────────────────

func ListContentHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListContentItems(r.Context(), actorTenant(r), r.URL.Query().Get("kind"), r.URL.Query().Get("status"), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateContentHandler(w http.ResponseWriter, r *http.Request) {
	var c model.ContentItem
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if c.Title == "" || c.Slug == "" || c.Kind == "" {
		writeError(w, http.StatusBadRequest, "title, slug and kind are required")
		return
	}
	c.TenantID = actorTenant(r)
	c.UpdatedBy = actorID(r)
	if c.Status == "" {
		c.Status = "draft"
	}
	if err := store.CreateContentItem(r.Context(), &c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "content.created", "content", c.ID, map[string]interface{}{"title": c.Title, "kind": c.Kind})
	writeJSON(w, http.StatusCreated, c)
}

func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	item, err := store.GetContentItem(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func UpdateContentHandler(w http.ResponseWriter, r *http.Request) {
	c, err := store.GetContentItem(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	var body model.ContentItem
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	c.Kind, c.Title, c.Slug = body.Kind, body.Title, body.Slug
	c.Summary, c.Body, c.Status = body.Summary, body.Body, body.Status
	c.AuthorID, c.Tags = body.AuthorID, body.Tags
	c.UpdatedBy = actorID(r)
	if err := store.UpdateContentItem(r.Context(), c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "content.updated", "content", c.ID, map[string]interface{}{"status": c.Status})
	writeJSON(w, http.StatusOK, c)
}

func DeleteContentHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := store.DeleteContentItem(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "content.deleted", "content", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func PublishContentHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	c, err := store.GetContentItem(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	now := time.Now().UTC()
	c.PublishedAt = &now
	c.Status = "published"
	c.UpdatedBy = actorID(r)
	if err := store.UpdateContentItem(r.Context(), c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "content.published", "content", c.ID, map[string]interface{}{"title": c.Title, "kind": c.Kind})
	recordAudit(r, "content.published", "content", c.ID, map[string]interface{}{"title": c.Title})
	writeJSON(w, http.StatusOK, c)
}
