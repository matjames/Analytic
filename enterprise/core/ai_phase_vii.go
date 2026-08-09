package main

// Phase VII keeps AI orchestration in Enterprise Core.  Source systems remain
// authoritative; this package only consumes permission-filtered enterprise data.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type AIContextRequest struct {
	OrganizationID string `json:"organization_id,omitempty"`
	ProjectID      string `json:"project_id,omitempty"`
	Region         string `json:"region,omitempty"`
	District       string `json:"district,omitempty"`
	Application    string `json:"application,omitempty"`
	ObjectType     string `json:"object_type,omitempty"`
	ObjectID       string `json:"object_id,omitempty"`
	Limit          int    `json:"limit,omitempty"`
}

type AISource struct {
	ID        string `json:"id"`
	App       string `json:"app"`
	Entity    string `json:"entity"`
	RecordID  string `json:"record_id"`
	TenantID  string `json:"tenant_id"`
	Title     string `json:"title"`
	Timestamp string `json:"timestamp"`
	DeepLink  string `json:"deep_link,omitempty"`
}

type AIContext struct {
	ID       string             `json:"id"`
	UserID   string             `json:"user_id"`
	TenantID string             `json:"tenant_id"`
	Sources  []AISource         `json:"sources"`
	Records  []EnterpriseRecord `json:"records"`
	Created  string             `json:"created_at"`
}

type AIRequest struct {
	Question string `json:"question"`
	AIContextRequest
}

type AIFinding struct {
	Kind       string   `json:"kind"` // fact | inference | recommendation | uncertainty
	Statement  string   `json:"statement"`
	SourceIDs  []string `json:"source_ids,omitempty"`
	Confidence string   `json:"confidence"`
}

type AIResponse struct {
	ID          string      `json:"id"`
	Status      string      `json:"status"`
	ContextID   string      `json:"context_id"`
	Findings    []AIFinding `json:"findings"`
	Sources     []AISource  `json:"sources"`
	GeneratedAt string      `json:"generated_at"`
}

// AIModelAdapter is the provider boundary. Production adapters may call an
// approved provider, while the built-in adapter deliberately produces only
// evidence-derived statements and never fabricates an LLM answer.
type AIModelAdapter interface {
	ID() string
	Generate(AIContext, string) ([]AIFinding, error)
}

type groundedEvidenceAdapter struct{}

func (groundedEvidenceAdapter) ID() string { return "grounded-evidence" }
func (groundedEvidenceAdapter) Generate(ctx AIContext, question string) ([]AIFinding, error) {
	return groundedResponse(ctx, question).Findings, nil
}

type AIFeedback struct {
	ID         string `json:"id"`
	ResponseID string `json:"response_id"`
	Rating     string `json:"rating"`
	Comment    string `json:"comment,omitempty"`
	UserID     string `json:"user_id"`
	CreatedAt  string `json:"created_at"`
}

var aiStore = struct {
	sync.RWMutex
	contexts  map[string]AIContext
	responses map[string]AIResponse
	feedback  map[string]AIFeedback
}{contexts: map[string]AIContext{}, responses: map[string]AIResponse{}, feedback: map[string]AIFeedback{}}

func aiActor(c *gin.Context) string {
	if user := c.GetHeader("X-User-ID"); user != "" {
		return user
	}
	return c.Query("user_id")
}

func aiTenant(c *gin.Context) string {
	if tenant := c.GetHeader("X-Tenant-ID"); tenant != "" {
		return tenant
	}
	if tenant := getEnvValue("STATGATE_TENANT_ID"); tenant != "" {
		return tenant
	}
	return "statgate"
}

func aiAuthorize(c *gin.Context, action string) (string, bool) {
	user := aiActor(c)
	if user == "" {
		c.JSON(401, gin.H{"error": "X-User-ID is required"})
		return "", false
	}
	if !evaluatePermission(user, "analytics", action) && !evaluatePermission(user, "ai", action) {
		c.JSON(403, gin.H{"error": "not authorized for AI intelligence access"})
		return "", false
	}
	return user, true
}

func resolveAIContext(user, tenant string, req AIContextRequest) AIContext {
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	records := fetchEnterpriseRecords("", "", req.ProjectID, req.OrganizationID, req.Region, limit)
	filtered := make([]EnterpriseRecord, 0, len(records))
	sources := make([]AISource, 0, len(records))
	for _, record := range records {
		if record.TenantID != tenant {
			continue
		}
		if req.Application != "" && record.SourceApp != req.Application {
			continue
		}
		if req.ObjectType != "" && record.SourceEntity != req.ObjectType {
			continue
		}
		if req.ObjectID != "" && record.SourceID != req.ObjectID {
			continue
		}
		if req.District != "" && record.District != req.District {
			continue
		}
		filtered = append(filtered, record)
		sources = append(sources, AISource{ID: "src_" + record.SourceApp + "_" + record.SourceEntity + "_" + record.SourceID, App: record.SourceApp, Entity: record.SourceEntity, RecordID: record.SourceID, TenantID: record.TenantID, Title: record.EventType, Timestamp: record.Timestamp, DeepLink: aiSourceDeepLink(record)})
	}
	ctx := AIContext{ID: fmt.Sprintf("aictx_%d", time.Now().UnixNano()), UserID: user, TenantID: tenant, Sources: sources, Records: filtered, Created: nowUTC()}
	aiStore.Lock()
	aiStore.contexts[ctx.ID] = ctx
	aiStore.Unlock()
	recordAudit("ai.context.resolve", "enterprise", user, map[string]interface{}{"context_id": ctx.ID, "tenant_id": tenant, "source_count": len(sources)})
	return ctx
}

func aiSourceDeepLink(record EnterpriseRecord) string {
	base := map[string]string{
		"pms": getEnv("PMS_UI_URL", ""), "rms": getEnv("RMS_UI_URL", ""),
		"registry": getEnv("REGISTRY_UI_URL", ""), "statchat": getEnv("STATCHAT_UI_URL", ""),
		"helpdesk": getEnv("HELPDESK_UI_URL", ""), "analytics": getEnv("ANALYTICS_UI_URL", ""),
		"statcollect": getEnv("STATCOLLECT_UI_URL", ""),
	}[record.SourceApp]
	if base == "" {
		return ""
	}
	return fmt.Sprintf("%s?entity=%s&id=%s", base, record.SourceEntity, record.SourceID)
}

func persistAIResponse(resp AIResponse, user string) {
	aiStore.Lock()
	aiStore.responses[resp.ID] = resp
	aiStore.Unlock()
	if redisClient != nil {
		data, _ := jsonMarshal(resp)
		redisClient.Set(context.Background(), "statgate:ai:response:"+resp.ID, string(data), 7*24*time.Hour)
	}
	recordAudit("ai.response.create", "enterprise", user, map[string]interface{}{"response_id": resp.ID, "context_id": resp.ContextID, "status": resp.Status, "source_count": len(resp.Sources)})
}

// jsonMarshal is retained as a tiny boundary so the persistence path is easily
// replaced by the Enterprise PostgreSQL repository without changing handlers.
func jsonMarshal(v interface{}) ([]byte, error) { return json.Marshal(v) }

func groundedResponse(ctx AIContext, question string) AIResponse {
	resp := AIResponse{ID: fmt.Sprintf("airesp_%d", time.Now().UnixNano()), ContextID: ctx.ID, Sources: ctx.Sources, GeneratedAt: nowUTC()}
	if len(ctx.Records) == 0 {
		resp.Status = "insufficient_evidence"
		resp.Findings = []AIFinding{{Kind: "uncertainty", Statement: "Insufficient evidence in the authorized enterprise context to answer this request.", Confidence: "insufficient_evidence"}}
		return resp
	}
	resp.Status = "grounded"
	bySource := map[string]int{}
	for _, r := range ctx.Records {
		bySource[r.SourceApp]++
	}
	apps := make([]string, 0, len(bySource))
	for app := range bySource {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	ids := make([]string, 0, len(ctx.Sources))
	for _, s := range ctx.Sources {
		ids = append(ids, s.ID)
	}
	resp.Findings = []AIFinding{{Kind: "fact", Statement: fmt.Sprintf("The authorized context contains %d records from %s.", len(ctx.Records), strings.Join(apps, ", ")), SourceIDs: ids, Confidence: "high"}}
	if question != "" {
		resp.Findings = append(resp.Findings, AIFinding{Kind: "uncertainty", Statement: "A configured model is required for a natural-language causal interpretation; no interpretation was generated from the evidence alone.", Confidence: "insufficient_evidence"})
	}
	return resp
}

func handleAIContextV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	var req AIContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid context request"})
		return
	}
	c.JSON(200, resolveAIContext(user, aiTenant(c), req))
}

func handleAIQueryV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	var req AIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid AI query"})
		return
	}
	ctx := resolveAIContext(user, aiTenant(c), req.AIContextRequest)
	resp := groundedResponse(ctx, req.Question)
	persistAIResponse(resp, user)
	c.JSON(200, resp)
}

func handleAIAnalyzeV1(c *gin.Context) { handleAIQueryV1(c) }

func handleAIInvestigateV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	var req AIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid investigation request"})
		return
	}
	ctx := resolveAIContext(user, aiTenant(c), req.AIContextRequest)
	resp := groundedResponse(ctx, req.Question)
	if resp.Status == "grounded" {
		resp.Findings = append(resp.Findings, AIFinding{Kind: "inference", Statement: "The listed sources are relevant to the investigation; causal attribution requires additional evidence or a configured approved model.", SourceIDs: sourceIDs(ctx.Sources), Confidence: "low"})
	}
	persistAIResponse(resp, user)
	c.JSON(200, resp)
}

func sourceIDs(sources []AISource) []string {
	out := make([]string, 0, len(sources))
	for _, s := range sources {
		out = append(out, s.ID)
	}
	return out
}

func handleAISourcesV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	aiStore.RLock()
	resp, found := aiStore.responses[c.Param("id")]
	aiStore.RUnlock()
	if !found {
		c.JSON(404, gin.H{"error": "AI response not found"})
		return
	}
	aiStore.RLock()
	ctx, contextFound := aiStore.contexts[resp.ContextID]
	aiStore.RUnlock()
	if !contextFound || ctx.TenantID != aiTenant(c) || (ctx.UserID != user && !evaluatePermission(user, "ai", "admin")) {
		c.JSON(404, gin.H{"error": "AI response not found"})
		return
	}
	c.JSON(200, gin.H{"response_id": resp.ID, "context_id": resp.ContextID, "sources": resp.Sources})
}

func handleAIRecommendV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "write")
	if !ok {
		return
	}
	var body struct{ ResponseID, Title, Description, SuggestedAction, Priority, ResponsibleRole, SuggestedDeadline string }
	if err := c.ShouldBindJSON(&body); err != nil || body.ResponseID == "" || body.Title == "" {
		c.JSON(400, gin.H{"error": "response_id and title are required"})
		return
	}
	aiStore.RLock()
	resp, found := aiStore.responses[body.ResponseID]
	aiStore.RUnlock()
	if !found {
		c.JSON(404, gin.H{"error": "AI response not found"})
		return
	}
	rec := AIRecommendation{ID: fmt.Sprintf("air_%d", time.Now().UnixNano()), Type: "action", Title: body.Title, Description: body.Description, SuggestedAction: body.SuggestedAction, Evidence: sourceIDs(resp.Sources), Confidence: 0, Status: "pending"}
	if len(resp.Sources) > 0 {
		rec.RelatedEntity = resp.Sources[0].Entity
		rec.RelatedEntityID = resp.Sources[0].RecordID
	}
	createAIRecommendation(rec)
	recordAudit("ai.recommendation.propose", "enterprise", user, map[string]interface{}{"response_id": body.ResponseID, "title": body.Title, "priority": body.Priority, "responsible_role": body.ResponsibleRole, "suggested_deadline": body.SuggestedDeadline})
	c.JSON(201, gin.H{"recommendation": rec, "requires_human_review": true})
}

func handleAIFeedbackV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	var feedback AIFeedback
	if err := c.ShouldBindJSON(&feedback); err != nil || feedback.ResponseID == "" || feedback.Rating == "" {
		c.JSON(400, gin.H{"error": "response_id and rating are required"})
		return
	}
	feedback.ID = fmt.Sprintf("aifb_%d", time.Now().UnixNano())
	feedback.UserID = user
	feedback.CreatedAt = nowUTC()
	aiStore.Lock()
	aiStore.feedback[feedback.ID] = feedback
	aiStore.Unlock()
	recordAudit("ai.feedback", "enterprise", user, map[string]interface{}{"feedback_id": feedback.ID, "response_id": feedback.ResponseID, "rating": feedback.Rating})
	c.JSON(201, feedback)
}

func handleAIModelsV1(c *gin.Context) {
	if _, ok := aiAuthorize(c, "read"); !ok {
		return
	}
	c.JSON(200, gin.H{"models": []gin.H{{"id": "grounded-evidence", "provider": "statgate", "status": "available", "purpose": "evidence-only analysis"}}, "provider_configuration_required": true})
}
func handleAIUsageV1(c *gin.Context) {
	if _, ok := aiAuthorize(c, "read"); !ok {
		return
	}
	aiStore.RLock()
	responses, feedback := len(aiStore.responses), len(aiStore.feedback)
	aiStore.RUnlock()
	c.JSON(200, gin.H{"responses": responses, "feedback": feedback, "external_model_calls": 0})
}
func handleAIEvaluationsV1(c *gin.Context) {
	if _, ok := aiAuthorize(c, "read"); !ok {
		return
	}
	c.JSON(200, gin.H{"evaluations": []interface{}{}, "status": "no evaluation run recorded"})
}
func handleAIBriefingsV1(c *gin.Context) {
	user, ok := aiAuthorize(c, "read")
	if !ok {
		return
	}
	tasks, approvals, alerts := userTasks(user), userApprovals(user), userAlerts(user)
	recommendations := listAIRecommendations("pending")
	findings := make([]AIFinding, 0, 3)
	if len(tasks) > 0 {
		findings = append(findings, AIFinding{Kind: "fact", Statement: fmt.Sprintf("You have %d assigned task(s).", len(tasks)), Confidence: "high"})
	}
	if len(approvals) > 0 {
		findings = append(findings, AIFinding{Kind: "fact", Statement: fmt.Sprintf("You have %d pending approval(s).", len(approvals)), Confidence: "high"})
	}
	if len(alerts) > 0 {
		findings = append(findings, AIFinding{Kind: "fact", Statement: fmt.Sprintf("%d active analytical alert(s) require attention in your scope.", len(alerts)), Confidence: "high"})
	}
	if len(findings) == 0 {
		findings = append(findings, AIFinding{Kind: "uncertainty", Statement: "No authorized work, approval, or alert evidence is currently available for your briefing.", Confidence: "insufficient_evidence"})
	}
	status := "insufficient_evidence"
	if len(tasks)+len(approvals)+len(alerts) > 0 {
		status = "verified"
	}
	c.JSON(200, gin.H{"status": status, "user_id": user, "findings": findings, "tasks": tasks, "approvals": approvals, "alerts": alerts, "recommendations": recommendations, "generated_at": nowUTC()})
}
