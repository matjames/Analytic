package api

import (
	"encoding/json"
	"net/http"

	"aiengines/pkg/model"
	"aiengines/pkg/store"
)

// ─── Knowledge Graph: Nodes (P39) ────────────────────────────────────────────

func ListGraphNodesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListGraphNodes(r.Context(), actorTenant(r), r.URL.Query().Get("type"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateGraphNodeHandler creates a graph node and registers the entity in
// object_links when it references a source platform object (agent hook).
func CreateGraphNodeHandler(w http.ResponseWriter, r *http.Request) {
	var n model.GraphNode
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if n.Type == "" || n.Label == "" {
		writeError(w, http.StatusBadRequest, "type and label are required")
		return
	}
	n.TenantID = actorTenant(r)
	n.CreatedBy = actorID(r)
	if err := store.CreateGraphNode(r.Context(), &n); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n.RefObjectType != "" && n.RefObjectID != "" {
		_ = store.CreateObjectLink(r.Context(), &model.ObjectLink{
			TenantID:     n.TenantID,
			SourceType:   "graph_node",
			SourceID:     n.ID,
			TargetType:   n.RefObjectType,
			TargetID:     n.RefObjectID,
			Relationship: "represents",
		})
	}
	emitEvent(r.Context(), "graph.entity.registered", "graph_node", n.ID,
		map[string]interface{}{"type": n.Type, "label": n.Label})
	recordAudit(r, "graph.node.created", "graph_node", n.ID, map[string]interface{}{"type": n.Type})
	writeJSON(w, http.StatusCreated, n)
}

func GetGraphNodeHandler(w http.ResponseWriter, r *http.Request) {
	n, err := store.GetGraphNode(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func DeleteGraphNodeHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteGraphNode(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "graph.node.deleted", "graph_node", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Knowledge Graph: Neighbors & Edges (P39) ────────────────────────────────

// GraphNeighborsHandler returns the edges incident to a node.
func GraphNeighborsHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetGraphNode(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	tenant := actorTenant(r)
	outEdges, err := store.ListGraphEdges(r.Context(), tenant, id, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Reverse edges (target -> source) are resolved via neighbour lookup below.
	var neighbors []map[string]interface{}
	for _, e := range outEdges {
		neighbors = append(neighbors, map[string]interface{}{
			"node_id": e.TargetNode, "predicate": e.Predicate, "direction": "out",
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"node_id": id, "edges": outEdges, "neighbors": neighbors})
}

func ListGraphEdgesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := store.ListGraphEdges(r.Context(), actorTenant(r), q.Get("source"), q.Get("predicate"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateGraphEdgeHandler(w http.ResponseWriter, r *http.Request) {
	var e model.GraphEdge
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if e.SourceNode == "" || e.TargetNode == "" || e.Predicate == "" {
		writeError(w, http.StatusBadRequest, "source_node, target_node and predicate are required")
		return
	}
	e.TenantID = actorTenant(r)
	e.CreatedBy = actorID(r)
	if err := store.CreateGraphEdge(r.Context(), &e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "graph.entity.registered", "graph_edge", e.ID,
		map[string]interface{}{"source": e.SourceNode, "target": e.TargetNode, "predicate": e.Predicate})
	writeJSON(w, http.StatusCreated, e)
}

func DeleteGraphEdgeHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteGraphEdge(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Semantic Triplestore (P39) ──────────────────────────────────────────────

func CreateTripleHandler(w http.ResponseWriter, r *http.Request) {
	var t model.Triple
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if t.Subject == "" || t.Predicate == "" || t.Object == "" {
		writeError(w, http.StatusBadRequest, "subject, predicate and object are required")
		return
	}
	t.TenantID = actorTenant(r)
	t.CreatedBy = actorID(r)
	if err := store.CreateTriple(r.Context(), &t); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "graph.entity.registered", "graph_statement", t.ID,
		map[string]interface{}{"subject": t.Subject, "predicate": t.Predicate, "object": t.Object})
	writeJSON(w, http.StatusCreated, t)
}

// ListTriplesHandler runs a triple-pattern query (SPARQL-style).
func ListTriplesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := store.QueryTriples(r.Context(), actorTenant(r), q.Get("subject"), q.Get("predicate"), q.Get("object"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Contextual Intelligence Indexer (P39) ───────────────────────────────────

func EnrichIndexHandler(w http.ResponseWriter, r *http.Request) {
	var e model.ContextEntry
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if e.EntityType == "" || e.EntityID == "" || e.Content == "" {
		writeError(w, http.StatusBadRequest, "entity_type, entity_id and content are required")
		return
	}
	e.TenantID = actorTenant(r)
	if err := store.UpsertContext(r.Context(), &e); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func SearchIndexHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q query parameter is required")
		return
	}
	hits, err := store.SearchContext(r.Context(), actorTenant(r), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hits)
}

// ─── Cross-application object linkage ────────────────────────────────────────

func CreateLinkHandler(w http.ResponseWriter, r *http.Request) {
	var l model.ObjectLink
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if l.SourceType == "" || l.SourceID == "" || l.TargetType == "" || l.TargetID == "" {
		writeError(w, http.StatusBadRequest, "source_type/source_id and target_type/target_id are required")
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