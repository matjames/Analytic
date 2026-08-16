package api

import (
	"encoding/json"
	"net/http"

	"knowledgeportal/pkg/model"
	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
)

// ─── Repository / Library (P40) ────────────────────────────────────────────

func ListRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListRepositoryItems(r.Context(), actorTenant(r), r.URL.Query().Get("status"), r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	var it model.RepositoryItem
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if it.Title == "" || it.Kind == "" {
		writeError(w, http.StatusBadRequest, "title and kind are required")
		return
	}
	it.TenantID = actorTenant(r)
	it.UpdatedBy = actorID(r)
	if it.Status == "" {
		it.Status = "draft"
	}
	if it.Access == "" {
		it.Access = "open"
	}
	if err := store.CreateRepositoryItem(r.Context(), &it); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if it.DOI != "" {
		_ = store.CreateIdentifier(r.Context(), &model.PersistentIdentifier{TenantID: it.TenantID, ItemType: "repository_item", ItemID: it.ID, IDType: "doi", IDValue: it.DOI})
	}
	recordAudit(r, "repository.created", "repository_item", it.ID, map[string]interface{}{"title": it.Title, "kind": it.Kind})
	writeJSON(w, http.StatusCreated, it)
}

func GetRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	it, err := store.GetRepositoryItem(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "repository item not found")
		return
	}
	writeJSON(w, http.StatusOK, it)
}

func UpdateRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	it, err := store.GetRepositoryItem(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "repository item not found")
		return
	}
	var body model.RepositoryItem
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	it.Kind, it.Title = body.Kind, body.Title
	it.Authors, it.DOI, it.ISBN, it.ISSN = body.Authors, body.DOI, body.ISBN, body.ISSN
	it.Abstract, it.Status, it.Access, it.Rights = body.Abstract, body.Status, body.Access, body.Rights
	it.UpdatedBy = actorID(r)
	if err := store.UpdateRepositoryItem(r.Context(), it); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "repository.updated", "repository_item", it.ID, map[string]interface{}{"status": it.Status})
	writeJSON(w, http.StatusOK, it)
}

func DeleteRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := store.DeleteRepositoryItem(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "repository.item.archived", "repository_item", id, nil)
	recordAudit(r, "repository.deleted", "repository_item", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Persistent identifiers (P40) ──────────────────────────────────────────

func CreateIdentifierHandler(w http.ResponseWriter, r *http.Request) {
	var pi model.PersistentIdentifier
	if err := json.NewDecoder(r.Body).Decode(&pi); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if pi.ItemType == "" || pi.ItemID == "" || pi.IDType == "" || pi.IDValue == "" {
		writeError(w, http.StatusBadRequest, "item_type, item_id, id_type and id_value are required")
		return
	}
	pi.TenantID = actorTenant(r)
	if err := store.CreateIdentifier(r.Context(), &pi); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "identifier.registered", "identifier", pi.ID, map[string]interface{}{"id_type": pi.IDType, "id_value": pi.IDValue})
	writeJSON(w, http.StatusCreated, pi)
}
