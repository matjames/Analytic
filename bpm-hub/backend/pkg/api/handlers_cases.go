package api

import (
	"encoding/json"
	"net/http"

	"bpmhub/pkg/model"
	"bpmhub/pkg/store"
)

// ─── Case Management (P48) ───────────────────────────────────────────────────

func ListCasesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListCases(r.Context(), actorTenant(r), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateCaseHandler(w http.ResponseWriter, r *http.Request) {
	var c model.CaseInstance
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if c.Name == "" || c.Key == "" {
		writeError(w, http.StatusBadRequest, "name and key are required")
		return
	}
	c.TenantID = actorTenant(r)
	if err := store.CreateCase(r.Context(), &c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "case.created", "case_instance", c.ID, map[string]interface{}{"key": c.Key, "name": c.Name})
	writeJSON(w, http.StatusCreated, c)
}

func GetCaseHandler(w http.ResponseWriter, r *http.Request) {
	c, err := store.GetCase(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func UpdateCaseHandler(w http.ResponseWriter, r *http.Request) {
	c, err := store.GetCase(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	var body model.CaseInstance
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.Status != "" {
		if err := store.UpdateCaseStatus(r.Context(), c.ID, body.Status); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	c.Status = body.Status
	writeJSON(w, http.StatusOK, c)
}

func DeleteCaseHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteCase(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func ListCaseItemsHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetCase(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	items, err := store.ListCaseItems(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func AddCaseItemHandler(w http.ResponseWriter, r *http.Request) {
	ci := model.CaseItem{TenantID: actorTenant(r), CaseID: varsOf(r)["id"]}
	if err := json.NewDecoder(r.Body).Decode(&ci); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if _, err := store.GetCase(r.Context(), ci.CaseID); err != nil {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	if err := store.AddCaseItem(r.Context(), &ci); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, ci)
}

// LinkCaseHandler registers a case against a platform object in object_links.
func LinkCaseHandler(w http.ResponseWriter, r *http.Request) {
	caseID := varsOf(r)["id"]
	c, err := store.GetCase(r.Context(), caseID)
	if err != nil {
		writeError(w, http.StatusNotFound, "case not found")
		return
	}
	var body struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.TargetType == "" || body.TargetID == "" {
		writeError(w, http.StatusBadRequest, "target_type and target_id are required")
		return
	}
	link := model.ObjectLink{
		TenantID: c.TenantID, SourceType: "case", SourceID: c.ID,
		TargetType: body.TargetType, TargetID: body.TargetID, Relationship: "tracks",
	}
	if err := store.CreateObjectLink(r.Context(), &link); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "object.link.created", "object_link", link.ID,
		map[string]interface{}{"source": "case:" + c.ID, "target": body.TargetType + ":" + body.TargetID})
	writeJSON(w, http.StatusCreated, link)
}