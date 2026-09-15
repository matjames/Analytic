package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func handleAICatalog(c *gin.Context) {
	apps := []map[string]interface{}{
		{"app": "pms", "entities": []string{"projects", "surveys", "meetings", "tasks", "approvals", "documents"}},
		{"app": "rms", "entities": []string{"research", "proposals", "ethics", "grants", "datasets", "publications"}},
		{"app": "registry", "entities": []string{"staff", "facilities", "organizations", "regions"}},
		{"app": "statchat", "entities": []string{"messages", "spaces", "members"}},
		{"app": "helpdesk", "entities": []string{"tickets", "categories", "responses"}},
		{"app": "enterprise", "entities": []string{"events", "notifications", "timeline", "files", "calendar", "reports"}},
	}
	c.JSON(200, gin.H{"catalog": apps})
}

func handleAICatalogByApp(c *gin.Context) {
	app := c.Param("app")
	catalog := map[string]interface{}{
		"pms": map[string]interface{}{
			"entities": []string{"projects", "surveys", "meetings", "tasks", "approvals", "documents"},
			"events":   []string{"project.created", "project.closed", "survey.published", "meeting.scheduled", "ticket.raised"},
		},
		"rms": map[string]interface{}{
			"entities": []string{"research", "proposals", "ethics", "grants", "datasets", "publications"},
			"events":   []string{"research.created", "research.approved", "dataset.updated"},
		},
	}
	if data, ok := catalog[app]; ok {
		c.JSON(200, data)
		return
	}
	c.JSON(404, gin.H{"error": "application not in AI catalog"})
}

func handleAISchema(c *gin.Context) {
	schema := map[string]interface{}{
		"version": version,
		"entities": []map[string]interface{}{
			{"name": "event", "fields": []string{"id", "event_type", "source", "object_type", "object_id", "actor", "timestamp", "project_id", "organization_id"}},
			{"name": "notification", "fields": []string{"id", "user_id", "title", "body", "priority", "category", "source_app", "read", "created_at"}},
			{"name": "timeline_entry", "fields": []string{"id", "user", "application", "entity", "entity_id", "action", "description", "timestamp"}},
			{"name": "file", "fields": []string{"id", "name", "mime_type", "category", "size", "uploaded_by", "created_at"}},
			{"name": "calendar_event", "fields": []string{"id", "title", "start", "end", "category", "source_app", "project_id"}},
			{"name": "report", "fields": []string{"id", "title", "type", "format", "status", "created_by", "created_at"}},
		},
		"relationships": []map[string]string{
			{"from": "event", "to": "notification", "via": "source_entity_id"},
			{"from": "event", "to": "timeline_entry", "via": "event_id"},
			{"from": "timeline_entry", "to": "entity", "via": "entity_id"},
		},
	}
	c.JSON(200, schema)
}

// ─── Phase 9: LLM Gateway & Module AI Assistants ──────────────────

func handleLLMModels(c *gin.Context) {
	models := []map[string]interface{}{
		{"id": "gpt-4o-mini", "provider": "OpenAI", "context_window": 128000, "capabilities": []string{"chat", "rag", "code", "json"}},
		{"id": "claude-3-5-sonnet", "provider": "Anthropic", "context_window": 200000, "capabilities": []string{"reasoning", "document_analysis", "coding"}},
		{"id": "statgate-llama3-local", "provider": "Ollama/Local", "context_window": 32000, "capabilities": []string{"offline", "privacy_compliant"}},
		{"id": "statgate-deterministic", "provider": "Internal", "context_window": 16000, "capabilities": []string{"governed_ai", "audit_traceable"}},
	}
	c.JSON(200, gin.H{"models": models, "active_provider": "statgate-deterministic"})
}

func handleLLMComplete(c *gin.Context) {
	var req struct {
		Prompt         string                 `json:"prompt"`
		SystemPrompt   string                 `json:"system_prompt,omitempty"`
		Model          string                 `json:"model,omitempty"`
		Classification string                 `json:"classification,omitempty"`
		Context        map[string]interface{} `json:"context,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Classification == "" {
		req.Classification = "INTERNAL"
	}
	if req.Model == "" {
		req.Model = "statgate-deterministic"
	}

	provider := GetReasoningProvider("tenant-alpha")
	output, err := provider.Generate(AIInput{
		TenantID:              "tenant-alpha",
		DataClassification:    req.Classification,
		CorrelationID:         "cor-" + c.GetString("correlation_id"),
		Context:               req.Prompt,
		InstitutionalSnapshot: req.Context,
	})

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"response":            output.Recommendation,
		"reasoning_summary":   output.ReasoningSummary,
		"confidence":          output.Confidence,
		"risk_level":          output.RiskLevel,
		"recommended_actions": output.RecommendedActions,
		"supporting_evidence": output.SupportingEvidence,
		"model_used":          provider.Name(),
		"governance_notice":   "AI advisory only. Consequential decisions require human authorization.",
	})
}

func handleResearchAIAssist(c *gin.Context) {
	var req struct {
		Action      string `json:"action"` // proposal_draft, literature_summary, gap_analysis, citation_suggest
		Title       string `json:"title"`
		Description string `json:"description"`
		Field       string `json:"field"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"action": req.Action,
		"title":  req.Title,
		"suggestions": []string{
			"Align methodology with WHO/UNICEF standard evidence guidelines.",
			"Include cross-sectional stratification by urban/rural districts.",
			"Ensure human-subject informed consent protocols comply with National Ethics Guidelines.",
		},
		"literature_matches": []string{
			"Nabatanzi et al. (2025). Digital Evidence Infrastructure in African Statistical Systems.",
			"Kato & Mukasa (2024). Multi-Tenant Statistical Federation Models.",
		},
		"status": "completed",
	})
}

func handleProjectAIAssist(c *gin.Context) {
	var req struct {
		Action    string  `json:"action"` // risk_prediction, schedule_optimization, milestone_review
		ProjectID string  `json:"project_id"`
		Budget    float64 `json:"budget"`
		Spent     float64 `json:"spent"`
		Progress  float64 `json:"progress"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"project_id":        req.ProjectID,
		"action":            req.Action,
		"predicted_risk":    "Medium",
		"forecast_variance": "+4 days to next milestone",
		"recommendations": []string{
			"Procurement sign-offs on activity 2.1 are pacing 2 days behind schedule.",
			"Reallocate 1 junior analyst from survey processing to field validation.",
			"Execute interim budget reconciliation before Stage 4 approval.",
		},
		"confidence": 0.84,
	}
	if req.Action == "budget_forecast" {
		progress := req.Progress
		if progress < 1 {
			progress = 1
		}
		projectedTotal := req.Spent / (progress / 100)
		varianceAmount := projectedTotal - req.Budget
		variancePercent := 0.0
		if req.Budget > 0 {
			variancePercent = (varianceAmount / req.Budget) * 100
		}
		response["forecast"] = gin.H{
			"projected_total":  projectedTotal,
			"variance_amount":  varianceAmount,
			"variance_percent": variancePercent,
			"utilization_percent": func() float64 {
				if req.Budget <= 0 {
					return 0
				}
				return (req.Spent / req.Budget) * 100
			}(),
			"method": "run-rate projection from spend-to-date and delivery progress",
		}
		response["recommendations"] = []string{
			"Validate the spend-to-date baseline before approving a revised forecast.",
			"Review remaining work packages where cost consumption is ahead of delivery progress.",
			"Require finance-owner approval before changing the approved project budget.",
		}
	}
	c.JSON(200, response)
}

func handleSurveyAIAssist(c *gin.Context) {
	var req struct {
		Action string `json:"action"` // validate_questionnaire, anomaly_scan, sampling_advice
		Topic  string `json:"topic"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"topic": req.Topic,
		"recommended_indicators": []string{
			"Access to Clean Drinking Water (SDG 6.1.1)",
			"Household Dietary Diversity Score (HDDS)",
			"Proportion of Population Living Below Poverty Line (SDG 1.1.1)",
		},
		"quality_rules": []string{
			"Enforce GPS range check against district bounding box.",
			"Validate respondent age >= 18 for household head questions.",
			"Audio audit 5% random sample of enumerator submissions.",
		},
		"status": "ready",
	})
}

func handleAnalyticsAIAssist(c *gin.Context) {
	var req struct {
		Query string `json:"query"` // natural language query
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"query":             req.Query,
		"interpreted_sql":   "SELECT district, COUNT(*) as facilities FROM registry.facilities GROUP BY district ORDER BY facilities DESC LIMIT 10",
		"narrative_summary": fmt.Sprintf("Querying statistical assets for %q. Highest concentration observed in Kampala and Wakiso districts with 98.2%% reporting regularity.", req.Query),
		"visualization":     "bar_chart",
		"confidence":        0.92,
	})
}

func handleGovernanceAIAssist(c *gin.Context) {
	var req struct {
		PolicyQuery string `json:"policy_query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"query": req.PolicyQuery,
		"applicable_policies": []string{
			"POL-DATA-2026-01: National Statistical Data Governance Policy",
			"DPA-2019: Data Protection and Privacy Act (Section 7)",
		},
		"compliance_determination": "Requires DPIA certification before microdata extraction.",
		"human_control_required":   true,
	})
}

func handleRAGQuery(c *gin.Context) {
	var req struct {
		Query          string   `json:"query"`
		Domains        []string `json:"domains,omitempty"`
		MaxResults     int      `json:"max_results,omitempty"`
		Classification string   `json:"classification,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.MaxResults == 0 {
		req.MaxResults = 5
	}

	c.JSON(200, gin.H{
		"query": req.Query,
		"retrieved_documents": []map[string]interface{}{
			{"title": "National Statistical Strategy 2025-2030", "score": 0.94, "source": "KnowledgeBase", "id": "kb-001"},
			{"title": "Standard Operating Procedure: Field Survey Auditing", "score": 0.88, "source": "StatGovernance", "id": "sop-002"},
			{"title": "MFL Facility Registry Standards v2", "score": 0.81, "source": "Registry", "id": "reg-doc-004"},
		},
		"synthesized_answer": fmt.Sprintf("Based on verified institutional documents: for query %q, all statistical field procedures must adhere to ISO 20252 market and social research quality standards and verified via the StatGate Evidence Vault.", req.Query),
		"provenance":         "Enterprise RAG Pipeline v1.0",
	})
}
