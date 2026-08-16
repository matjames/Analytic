package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"knowledgeportal/pkg/model"
	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
)

// ─── Public Portal: Content ────────────────────────────────────────────────

func ListPublicContentHandler(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	q := r.URL.Query().Get("q")
	items, err := store.ListContentItems(r.Context(), publicTenant(r), kind, "published", q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func GetPublicContentHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := store.GetContentItem(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	if item.Status != "published" {
		writeError(w, http.StatusNotFound, "content not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// ─── Public Portal: Datasets ───────────────────────────────────────────────

func ListPublicDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	items, err := store.ListDatasets(r.Context(), publicTenant(r), "published", q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func GetPublicDatasetHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	d, err := store.GetDataset(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "dataset not found")
		return
	}
	if d.Status != "published" {
		writeError(w, http.StatusNotFound, "dataset not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// ─── Public Portal: Repository / Library ───────────────────────────────────

func ListPublicRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	items, err := store.ListRepositoryItems(r.Context(), publicTenant(r), "published", q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func GetPublicRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	item, err := store.GetRepositoryItem(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository item not found")
		return
	}
	if item.Status != "published" {
		writeError(w, http.StatusNotFound, "repository item not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// ─── Public Portal: Cross-catalog search ───────────────────────────────────

func PublicSearchHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "q query parameter is required")
		return
	}
	results, err := store.SearchPublic(r.Context(), publicTenant(r), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// ─── Public Portal: Subscriptions & Feedback ───────────────────────────────

func CreateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var s model.Subscription
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if s.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	s.TenantID = publicTenant(r)
	if err := store.CreateSubscription(r.Context(), &s); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "subscription.created", "subscription", s.ID, map[string]interface{}{"email": s.Email, "topics": s.Topics})
	writeJSON(w, http.StatusCreated, s)
}

func CreateFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	var f model.Feedback
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if f.Subject == "" || f.Body == "" {
		writeError(w, http.StatusBadRequest, "subject and body are required")
		return
	}
	if f.Kind == "" {
		f.Kind = "feedback"
	}
	f.TenantID = publicTenant(r)
	if err := store.CreateFeedback(r.Context(), &f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "feedback.received", "feedback", f.ID, map[string]interface{}{"kind": f.Kind, "subject": f.Subject})
	writeJSON(w, http.StatusCreated, f)
}
