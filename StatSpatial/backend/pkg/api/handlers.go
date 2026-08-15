package api

import (
	"encoding/json"
	"net/http"

	"statspatial/pkg/model"
	"statspatial/pkg/store"

	"github.com/gorilla/mux"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// HealthHandler returns service readiness.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":  "unhealthy",
			"service": "statspatial",
			"db":      "disconnected",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "statspatial",
		"db":      "connected",
	})
}

// ReadyHandler returns service readiness probe.
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// Admin Units Handlers
func ListAdminUnitsHandler(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	parentID := r.URL.Query().Get("parent_id")
	units, err := store.ListAdminUnits(r.Context(), level, parentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, units)
}

func GetAdminUnitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	unit, err := store.GetAdminUnit(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if unit == nil {
		writeError(w, http.StatusNotFound, "admin unit not found")
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

func CreateAdminUnitHandler(w http.ResponseWriter, r *http.Request) {
	var au model.AdminUnit
	if err := json.NewDecoder(r.Body).Decode(&au); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	created, err := store.CreateAdminUnit(r.Context(), au)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func UpdateAdminUnitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var au model.AdminUnit
	if err := json.NewDecoder(r.Body).Decode(&au); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := store.UpdateAdminUnit(r.Context(), id, au)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func DeleteAdminUnitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := store.DeleteAdminUnit(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func GetAdminUnitTreeHandler(w http.ResponseWriter, r *http.Request) {
	tree, err := store.GetAdminUnitTree(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tree)
}

// Organization Handlers
func ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) {
	orgType := r.URL.Query().Get("type")
	orgs, err := store.ListOrganizations(r.Context(), orgType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orgs)
}

func GetOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	org, err := store.GetOrganization(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if org == nil {
		writeError(w, http.StatusNotFound, "organization not found")
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func CreateOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	var org model.Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateOrganization(r.Context(), org)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func UpdateOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var org model.Organization
	if err := json.NewDecoder(r.Body).Decode(&org); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	updated, err := store.UpdateOrganization(r.Context(), id, org)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := store.DeleteOrganization(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// Classification Handlers
func ListClassificationsHandler(w http.ResponseWriter, r *http.Request) {
	scheme := r.URL.Query().Get("scheme")
	items, err := store.ListClassifications(r.Context(), scheme)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateClassificationHandler(w http.ResponseWriter, r *http.Request) {
	var c model.Classification
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateClassification(r.Context(), c)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// GeoLayer & Feature Handlers
func ListGeoLayersHandler(w http.ResponseWriter, r *http.Request) {
	layers, err := store.ListGeoLayers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, layers)
}

func CreateGeoLayerHandler(w http.ResponseWriter, r *http.Request) {
	var layer model.GeoLayer
	if err := json.NewDecoder(r.Body).Decode(&layer); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateGeoLayer(r.Context(), layer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func ListGeoFeaturesHandler(w http.ResponseWriter, r *http.Request) {
	layerID := r.URL.Query().Get("layer_id")
	adminUnitID := r.URL.Query().Get("admin_unit_id")
	features, err := store.ListGeoFeatures(r.Context(), layerID, adminUnitID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, features)
}

func CreateGeoFeatureHandler(w http.ResponseWriter, r *http.Request) {
	var feat model.GeoFeature
	if err := json.NewDecoder(r.Body).Decode(&feat); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateGeoFeature(r.Context(), feat)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Spatial Index Handlers
func ListSpatialIndexHandler(w http.ResponseWriter, r *http.Request) {
	adminUnitID := r.URL.Query().Get("admin_unit_id")
	targetType := r.URL.Query().Get("target_type")
	items, err := store.ListSpatialIndex(r.Context(), adminUnitID, targetType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateSpatialIndexHandler(w http.ResponseWriter, r *http.Request) {
	var si model.SpatialIndex
	if err := json.NewDecoder(r.Body).Decode(&si); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := store.CreateSpatialIndex(r.Context(), si); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "indexed", "target_id": si.TargetID})
}

// Federated Node Handlers
func ListFederatedNodesHandler(w http.ResponseWriter, r *http.Request) {
	nodeType := r.URL.Query().Get("node_type")
	nodes, err := store.ListFederatedNodes(r.Context(), nodeType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func GetFederatedNodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	node, err := store.GetFederatedNode(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if node == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func CreateFederatedNodeHandler(w http.ResponseWriter, r *http.Request) {
	var n model.FederatedNode
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateFederatedNode(r.Context(), n)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func UpdateFederatedNodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var n model.FederatedNode
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	updated, err := store.UpdateFederatedNode(r.Context(), id, n)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func DeleteFederatedNodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if err := store.DeleteFederatedNode(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// Data Sharing Agreements
func ListAgreementsHandler(w http.ResponseWriter, r *http.Request) {
	agreements, err := store.ListAgreements(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agreements)
}

func CreateAgreementHandler(w http.ResponseWriter, r *http.Request) {
	var a model.DataSharingAgreement
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateAgreement(r.Context(), a)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Datasets & Sync
func ListFederatedDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	datasets, err := store.ListFederatedDatasets(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, datasets)
}

func CreateFederatedDatasetHandler(w http.ResponseWriter, r *http.Request) {
	var d model.FederatedDataset
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateFederatedDataset(r.Context(), d)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func ListSyncLogsHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	logs, err := store.ListSyncLogs(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, logs)
}

func CreateSyncLogHandler(w http.ResponseWriter, r *http.Request) {
	var l model.SyncLog
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	created, err := store.CreateSyncLog(r.Context(), l)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Node Links
func ListNodeLinksHandler(w http.ResponseWriter, r *http.Request) {
	sourceType := r.URL.Query().Get("source_type")
	sourceID := r.URL.Query().Get("source_id")
	links, err := store.ListNodeLinks(r.Context(), sourceType, sourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, links)
}

func CreateNodeLinkHandler(w http.ResponseWriter, r *http.Request) {
	var link model.NodeLink
	if err := json.NewDecoder(r.Body).Decode(&link); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := store.CreateNodeLink(r.Context(), link); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

// Summary & Spatial Intelligence
func GetSummaryHandler(w http.ResponseWriter, r *http.Request) {
	summary, err := store.GetSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
