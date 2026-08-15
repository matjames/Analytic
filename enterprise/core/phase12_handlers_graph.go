package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — KNOWLEDGE GRAPH API HANDLERS
// Named queries only. Graph mutation is privileged (admin / institutional_lead).
// ═══════════════════════════════════════════════════════════════════════════════

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleListGraphObjects(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	objectType := c.Query("object_type")
	objects := phase12.ListObjects(tenantID, objectType)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(objects), "objects": objects})
}

// handleProjectGraphObject projects a UOI-canonical object into the
// institutional registry. Privileged.
func handleProjectGraphObject(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	var obj InstitutionalObject
	if err := c.ShouldBindJSON(&obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_object", "message": err.Error()})
		return
	}
	obj.TenantID = tenantID
	if err := phase12.ProjectObject(&obj); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_rejected", "message": err.Error()})
		return
	}
	recordAuditFromContext(c, "graph.object.create", "institutional_objects", obj.CanonicalID, map[string]interface{}{
		"object_type":   obj.ObjectType,
		"source_system": obj.SourceSystem,
	})
	c.JSON(http.StatusCreated, gin.H{"id": obj.ID, "canonical_id": obj.CanonicalID, "status": "projected"})
}

func handleListGraphEdges(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	edges := phase12.ListEdges(tenantID)
	c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID, "count": len(edges), "edges": edges})
}

// handleCreateGraphEdge creates a knowledge graph edge. Privileged:
// admin / institutional_lead (directive §8). Every mutation is audited.
func handleCreateGraphEdge(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	var edge GraphEdge
	if err := c.ShouldBindJSON(&edge); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_edge", "message": err.Error()})
		return
	}
	edge.TenantID = tenantID
	edge.CreatedBy = getContextUserID(c)
	if err := phase12.CreateEdge(&edge); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "edge_rejected", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": edge.ID, "status": "edge_created"})
}

// handleDeleteGraphEdge retires an edge (append/correction oriented).
// Privileged + audited (graph.edge.delete).
func handleDeleteGraphEdge(c *gin.Context) {
	if !requirePhase12Privilege(c) {
		return
	}
	tenantID := getContextTenantID(c)
	edgeID := c.Param("id")
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	if err := phase12.DeleteEdge(tenantID, edgeID, getContextUserID(c)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "edge_not_found", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": edgeID, "status": "edge_retired"})
}

// handleNamedQueryObjectivesAtRisk is the static-form named query.
func handleNamedQueryObjectivesAtRisk(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	result, err := phase12.RunNamedGraphQuery(tenantID, string(QueryObjectivesAtRisk), "")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query_failed", "message": err.Error()})
		return
	}
	recordAuditFromContext(c, "graph.query.objectives-at-risk", "institutional_graph_edges", "", nil)
	c.JSON(http.StatusOK, result)
}

// handleNamedQueryWithParam executes an approved named query with a single
// parameter. Query name is validated against the approved set — unknown
// names fail closed (directive §27).
func handleNamedQueryWithParam(c *gin.Context) {
	tenantID := getContextTenantID(c)
	if tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_context_required"})
		return
	}
	name := c.Param("name")
	id := c.Param("id")
	if !validNamedGraphQuery(name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "query_not_approved",
			"message": "Only named, parameterized graph queries are permitted; arbitrary graph expressions are rejected.",
		})
		return
	}
	result, err := phase12.RunNamedGraphQuery(tenantID, name, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query_failed", "message": err.Error()})
		return
	}
	recordAuditFromContext(c, "graph.query."+name, "institutional_graph_edges", id, nil)
	c.JSON(http.StatusOK, result)
}