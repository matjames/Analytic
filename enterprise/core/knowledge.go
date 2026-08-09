package main

// Phase VI is implemented in the enterprise core so it reuses the existing
// audit, permission, workflow, task, decision, calendar and event services.

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type KnowledgeRelationship struct {
	ID        string `json:"id"`
	FromType  string `json:"from_type"`
	FromID    string `json:"from_id"`
	ToType    string `json:"to_type"`
	ToID      string `json:"to_id"`
	Relation  string `json:"relation"`
	Source    string `json:"source"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type KnowledgeArticle struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Body           string   `json:"body"`
	Category       string   `json:"category"`
	Tags           []string `json:"tags,omitempty"`
	Owner          string   `json:"owner"`
	Source         string   `json:"source"`
	Status         string   `json:"status"` // draft | review | published | archived
	Version        int      `json:"version"`
	ReviewDate     string   `json:"review_date,omitempty"`
	Visibility     string   `json:"visibility"` // restricted | organization
	ProjectID      string   `json:"project_id,omitempty"`
	Situation      string   `json:"situation,omitempty"`
	Problem        string   `json:"problem,omitempty"`
	Cause          string   `json:"cause,omitempty"`
	Intervention   string   `json:"intervention,omitempty"`
	Outcome        string   `json:"outcome,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

type DataDictionaryEntry struct {
	ID            string   `json:"id"`
	DatasetID     string   `json:"dataset_id"`
	Variable      string   `json:"variable"`
	Definition    string   `json:"definition"`
	DataType      string   `json:"data_type"`
	Source        string   `json:"source"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	Calculation   string   `json:"calculation,omitempty"`
	Owner         string   `json:"owner"`
	Version       int      `json:"version"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type KnowledgeSearchResult struct {
	Type     string                  `json:"type"`
	ID       string                  `json:"id"`
	Title    string                  `json:"title"`
	Summary  string                  `json:"summary,omitempty"`
	Source   string                  `json:"source"`
	DeepLink string                  `json:"deep_link"`
	Related  []KnowledgeRelationship `json:"related,omitempty"`
}

var knowledgeStore = struct {
	sync.RWMutex
	relationships map[string]KnowledgeRelationship
	articles      map[string]KnowledgeArticle
	dictionary    map[string]DataDictionaryEntry
}{relationships: make(map[string]KnowledgeRelationship), articles: make(map[string]KnowledgeArticle), dictionary: make(map[string]DataDictionaryEntry)}

func knowledgeActor(c *gin.Context) string {
	if actor := c.GetHeader("X-User-ID"); actor != "" {
		return actor
	}
	return c.Query("user_id")
}

func knowledgeReadable(user, resource string) bool {
	return user != "" && evaluatePermission(user, resource, "read")
}

func bootstrapKnowledge() {
	knowledgeStore.Lock()
	defer knowledgeStore.Unlock()
	// Keep bootstrapping idempotent: records received from applications remain
	// authoritative and no demo knowledge is injected into a tenant.
	if knowledgeStore.relationships == nil {
		knowledgeStore.relationships = make(map[string]KnowledgeRelationship)
	}
	if knowledgeStore.articles == nil {
		knowledgeStore.articles = make(map[string]KnowledgeArticle)
	}
	if knowledgeStore.dictionary == nil {
		knowledgeStore.dictionary = make(map[string]DataDictionaryEntry)
	}
}

func createKnowledgeRelationship(link KnowledgeRelationship) KnowledgeRelationship {
	// Relationship writes can arrive through both a source event and a replayed
	// event-bus delivery. Treat the natural graph edge as idempotent.
	knowledgeStore.RLock()
	for _, existing := range knowledgeStore.relationships {
		if existing.FromType == link.FromType && existing.FromID == link.FromID && existing.ToType == link.ToType && existing.ToID == link.ToID && existing.Relation == link.Relation {
			knowledgeStore.RUnlock()
			return existing
		}
	}
	knowledgeStore.RUnlock()
	if link.ID == "" {
		link.ID = fmt.Sprintf("krel_%d", time.Now().UnixNano())
	}
	if link.CreatedAt == "" {
		link.CreatedAt = nowUTC()
	}
	if link.Relation == "" {
		link.Relation = "related_to"
	}
	if link.Source == "" {
		link.Source = "enterprise"
	}
	knowledgeStore.Lock()
	knowledgeStore.relationships[link.ID] = link
	knowledgeStore.Unlock()
	recordAudit("knowledge.relationship.created", "enterprise", link.CreatedBy, map[string]interface{}{"relationship_id": link.ID, "from": link.FromType + ":" + link.FromID, "to": link.ToType + ":" + link.ToID, "relation": link.Relation})
	recordTimelineEntry(TimelineEntry{ID: fmt.Sprintf("tl_%d", time.Now().UnixNano()), User: link.CreatedBy, Application: "enterprise", Entity: "knowledge_relationship", EntityID: link.ID, Action: "knowledge.relationship.created", Description: fmt.Sprintf("Linked %s to %s", link.FromType, link.ToType), Timestamp: nowUTC()})
	return link
}

func relatedKnowledge(entityType, entityID string) []KnowledgeRelationship {
	knowledgeStore.RLock()
	defer knowledgeStore.RUnlock()
	links := make([]KnowledgeRelationship, 0)
	for _, link := range knowledgeStore.relationships {
		if (link.FromType == entityType && link.FromID == entityID) || (link.ToType == entityType && link.ToID == entityID) {
			links = append(links, link)
		}
	}
	sort.Slice(links, func(i, j int) bool { return links[i].CreatedAt > links[j].CreatedAt })
	return links
}

func deepLink(entityType, id string) string { return "/workspace/" + entityType + "/" + id }

func createKnowledgeArticle(article KnowledgeArticle) KnowledgeArticle {
	if article.ID == "" {
		article.ID = fmt.Sprintf("ka_%d", time.Now().UnixNano())
	}
	if article.Status == "" {
		article.Status = "draft"
	}
	if article.Visibility == "" {
		article.Visibility = "restricted"
	}
	if article.Version == 0 {
		article.Version = 1
	}
	article.CreatedAt, article.UpdatedAt = nowUTC(), nowUTC()
	knowledgeStore.Lock()
	knowledgeStore.articles[article.ID] = article
	knowledgeStore.Unlock()
	recordAudit("knowledge.article.created", "enterprise", article.Owner, map[string]interface{}{"article_id": article.ID, "version": article.Version, "status": article.Status, "source": article.Source})
	recordTimelineEntry(TimelineEntry{ID: fmt.Sprintf("tl_%d", time.Now().UnixNano()), User: article.Owner, Application: "enterprise", Entity: "knowledge_article", EntityID: article.ID, Action: "knowledge.article.created", Description: article.Title, ProjectID: article.ProjectID, Timestamp: nowUTC()})
	if article.ProjectID != "" {
		createKnowledgeRelationship(KnowledgeRelationship{FromType: "knowledge_article", FromID: article.ID, ToType: "project", ToID: article.ProjectID, Relation: "documents_learning_for", Source: article.Source, CreatedBy: article.Owner})
	}
	return article
}

func updateKnowledgeArticle(id string, updates KnowledgeArticle, actor string) (KnowledgeArticle, bool) {
	knowledgeStore.Lock()
	article, ok := knowledgeStore.articles[id]
	if !ok {
		knowledgeStore.Unlock()
		return article, false
	}
	if updates.Title != "" {
		article.Title = updates.Title
	}
	if updates.Summary != "" {
		article.Summary = updates.Summary
	}
	if updates.Body != "" {
		article.Body = updates.Body
	}
	if updates.Category != "" {
		article.Category = updates.Category
	}
	if updates.Tags != nil {
		article.Tags = updates.Tags
	}
	if updates.Status != "" {
		article.Status = updates.Status
	}
	if updates.ReviewDate != "" {
		article.ReviewDate = updates.ReviewDate
	}
	if updates.Visibility != "" {
		article.Visibility = updates.Visibility
	}
	article.Version++
	article.UpdatedAt = nowUTC()
	knowledgeStore.articles[id] = article
	knowledgeStore.Unlock()
	recordAudit("knowledge.article.updated", "enterprise", actor, map[string]interface{}{"article_id": id, "version": article.Version, "status": article.Status})
	recordTimelineEntry(TimelineEntry{ID: fmt.Sprintf("tl_%d", time.Now().UnixNano()), User: actor, Application: "enterprise", Entity: "knowledge_article", EntityID: id, Action: "knowledge.article.updated", Description: article.Title, ProjectID: article.ProjectID, Timestamp: nowUTC()})
	return article, true
}

func contextSummary(entityType, entityID string) map[string]interface{} {
	links := relatedKnowledge(entityType, entityID)
	groups := map[string][]KnowledgeRelationship{}
	for _, l := range links {
		otherType := l.ToType
		if l.ToType == entityType && l.ToID == entityID {
			otherType = l.FromType
		}
		groups[otherType] = append(groups[otherType], l)
	}
	return map[string]interface{}{"entity": map[string]string{"type": entityType, "id": entityID, "deep_link": deepLink(entityType, entityID)}, "relationships": links, "related_by_type": groups, "activity": timelineForKnowledge(entityType, entityID), "records": contextualRecords(entityType, entityID), "actions": gin.H{"discuss_in_statchat": getEnv("STATCHAT_API_URL", "http://localhost:4000") + "/v1/objects/" + entityType + "/" + entityID + "/discussion", "knowledge_search": "/api/knowledge/search?q=" + entityID}}
}

// contextualRecords assembles views from authoritative Phase IV/V services;
// it intentionally references source records rather than copying them.
func contextualRecords(entityType, entityID string) map[string]interface{} {
	projectID := entityID
	if entityType != "project" {
		projectID = ""
	}
	tasks := make([]EnterpriseTask, 0)
	for _, task := range listEnterpriseTasks("", "", "", 500) {
		if task.ProjectID == projectID || (task.SourceEntity == entityType && task.SourceEntityID == entityID) {
			tasks = append(tasks, task)
		}
	}
	decisions := make([]DecisionRecord, 0)
	for _, decision := range listDecisionRecords("", "", 500) {
		if decision.RelatedProject == projectID || decision.ID == entityID {
			decisions = append(decisions, decision)
		}
	}
	datasets := make([]Dataset, 0)
	for _, dataset := range listDatasets() {
		if dataset.ProjectID == projectID || (entityType == "dataset" && dataset.ID == entityID) {
			datasets = append(datasets, dataset)
		}
	}
	meetings := make([]CalendarEvent, 0)
	for _, meeting := range fetchCalendarEvents(500) {
		if meeting.ProjectID == projectID || (entityType == "meeting" && meeting.ID == entityID) {
			meetings = append(meetings, meeting)
		}
	}
	reports := make([]ReportRequest, 0)
	reportStore.RLock()
	for _, report := range reportStore.reports {
		if report.ID == entityID || (projectID != "" && fmt.Sprint(report.Filters["project_id"]) == projectID) {
			reports = append(reports, report)
		}
	}
	reportStore.RUnlock()
	return map[string]interface{}{"tasks": tasks, "decisions": decisions, "datasets": datasets, "meetings": meetings, "reports": reports}
}

// processKnowledgeForEvent maintains real graph links as source applications
// emit the existing enterprise domain events.
func processKnowledgeForEvent(ev DomainEvent) {
	if ev.ObjectType == "" || ev.ObjectID == "" {
		return
	}
	projectID := ev.ProjectID
	// Some established publishers place project_id in the event payload. Accept
	// both contracts while preserving the canonical top-level field.
	if projectID == "" {
		projectID, _ = ev.Payload["project_id"].(string)
	}
	if projectID != "" && !(ev.ObjectType == "project" && ev.ObjectID == projectID) {
		createKnowledgeRelationship(KnowledgeRelationship{FromType: ev.ObjectType, FromID: ev.ObjectID, ToType: "project", ToID: projectID, Relation: "belongs_to", Source: ev.Source, CreatedBy: ev.Actor})
	}
	for key, entityType := range map[string]string{"survey_id": "survey", "dataset_id": "dataset", "research_id": "research", "report_id": "report", "ticket_id": "ticket", "meeting_id": "meeting", "decision_id": "decision", "task_id": "task", "alert_id": "alert", "conversation_id": "conversation"} {
		if relatedID, ok := ev.Payload[key].(string); ok && relatedID != "" && !(entityType == ev.ObjectType && relatedID == ev.ObjectID) {
			createKnowledgeRelationship(KnowledgeRelationship{FromType: ev.ObjectType, FromID: ev.ObjectID, ToType: entityType, ToID: relatedID, Relation: "related_to", Source: ev.Source, CreatedBy: ev.Actor})
		}
	}
}

func timelineForKnowledge(entityType, entityID string) []TimelineEntry {
	all := fetchTimeline(100)
	out := make([]TimelineEntry, 0)
	for _, entry := range all {
		if entry.Entity == entityType && entry.EntityID == entityID {
			out = append(out, entry)
		}
	}
	return out
}

func knowledgeSearch(user, query string) []KnowledgeSearchResult {
	if !knowledgeReadable(user, "knowledge") {
		return []KnowledgeSearchResult{}
	}
	needle := strings.ToLower(strings.TrimSpace(query))
	results := make([]KnowledgeSearchResult, 0)
	knowledgeStore.RLock()
	for _, article := range knowledgeStore.articles {
		if article.Status == "archived" || (article.Visibility == "restricted" && article.Owner != user && !evaluatePermission(user, "knowledge", "write")) {
			continue
		}
		if needle == "" || strings.Contains(strings.ToLower(article.Title+" "+article.Summary+" "+article.Body+" "+strings.Join(article.Tags, " ")), needle) {
			results = append(results, KnowledgeSearchResult{Type: "knowledge_article", ID: article.ID, Title: article.Title, Summary: article.Summary, Source: article.Source, DeepLink: deepLink("knowledge", article.ID), Related: relatedKnowledge("knowledge_article", article.ID)})
		}
	}
	for _, entry := range knowledgeStore.dictionary {
		if needle == "" || strings.Contains(strings.ToLower(entry.Variable+" "+entry.Definition+" "+entry.DatasetID), needle) {
			results = append(results, KnowledgeSearchResult{Type: "data_dictionary", ID: entry.ID, Title: entry.Variable, Summary: entry.Definition, Source: entry.Source, DeepLink: deepLink("data-dictionary", entry.ID)})
		}
	}
	knowledgeStore.RUnlock()
	for _, decision := range listDecisionRecords("", "", 500) {
		if needle == "" || strings.Contains(strings.ToLower(decision.Decision+" "+decision.Context), needle) {
			results = append(results, KnowledgeSearchResult{Type: "decision", ID: decision.ID, Title: decision.Decision, Summary: decision.Context, Source: "enterprise", DeepLink: deepLink("decision", decision.ID), Related: relatedKnowledge("decision", decision.ID)})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Title < results[j].Title })
	return results
}

func handleCreateKnowledgeRelationship(c *gin.Context) {
	var link KnowledgeRelationship
	if c.ShouldBindJSON(&link) != nil || link.FromType == "" || link.FromID == "" || link.ToType == "" || link.ToID == "" {
		c.JSON(400, gin.H{"error": "from_type, from_id, to_type and to_id are required"})
		return
	}
	link.CreatedBy = knowledgeActor(c)
	c.JSON(201, createKnowledgeRelationship(link))
}
func handleEntityContext(c *gin.Context) {
	user := knowledgeActor(c)
	if !knowledgeReadable(user, "knowledge") {
		c.JSON(403, gin.H{"error": "knowledge access denied"})
		return
	}
	c.JSON(200, contextSummary(c.Param("type"), c.Param("id")))
}
func handleKnowledgeSearch(c *gin.Context) {
	user := knowledgeActor(c)
	if !knowledgeReadable(user, "knowledge") {
		c.JSON(403, gin.H{"error": "knowledge access denied"})
		return
	}
	results := knowledgeSearch(user, c.Query("q"))
	c.JSON(200, gin.H{"count": len(results), "results": results})
}
func handleKnowledgeHome(c *gin.Context) {
	user := knowledgeActor(c)
	if !knowledgeReadable(user, "knowledge") {
		c.JSON(403, gin.H{"error": "knowledge access denied"})
		return
	}
	results := knowledgeSearch(user, "")
	mine := make([]KnowledgeSearchResult, 0)
	for _, result := range results {
		if result.Type == "knowledge_article" {
			knowledgeStore.RLock()
			article := knowledgeStore.articles[result.ID]
			knowledgeStore.RUnlock()
			if article.Owner == user {
				mine = append(mine, result)
			}
		}
	}
	c.JSON(200, gin.H{"recent": results, "my_contributions": mine, "categories": []string{"articles", "guides", "sops", "methodologies", "lessons_learned", "data_dictionary", "reports", "research", "templates"}})
}
func handleCreateKnowledgeArticle(c *gin.Context) {
	var article KnowledgeArticle
	if c.ShouldBindJSON(&article) != nil || article.Title == "" {
		c.JSON(400, gin.H{"error": "title is required"})
		return
	}
	if article.Owner == "" {
		article.Owner = knowledgeActor(c)
	}
	if article.Owner == "" {
		c.JSON(400, gin.H{"error": "authenticated owner is required"})
		return
	}
	c.JSON(201, createKnowledgeArticle(article))
}
func handleListKnowledgeArticles(c *gin.Context) {
	user := knowledgeActor(c)
	if !knowledgeReadable(user, "knowledge") {
		c.JSON(403, gin.H{"error": "knowledge access denied"})
		return
	}
	c.JSON(200, gin.H{"count": len(knowledgeSearch(user, c.Query("q"))), "articles": knowledgeSearch(user, c.Query("q"))})
}
func handleUpdateKnowledgeArticle(c *gin.Context) {
	var update KnowledgeArticle
	if c.ShouldBindJSON(&update) != nil {
		c.JSON(400, gin.H{"error": "invalid article"})
		return
	}
	updated, ok := updateKnowledgeArticle(c.Param("id"), update, knowledgeActor(c))
	if !ok {
		c.JSON(404, gin.H{"error": "article not found"})
		return
	}
	c.JSON(200, updated)
}
func handleCreateDictionaryEntry(c *gin.Context) {
	var entry DataDictionaryEntry
	if c.ShouldBindJSON(&entry) != nil || entry.DatasetID == "" || entry.Variable == "" || entry.Definition == "" {
		c.JSON(400, gin.H{"error": "dataset_id, variable and definition are required"})
		return
	}
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("dict_%d", time.Now().UnixNano())
	}
	if entry.Version == 0 {
		entry.Version = 1
	}
	if entry.Status == "" {
		entry.Status = "published"
	}
	if entry.Owner == "" {
		entry.Owner = knowledgeActor(c)
	}
	entry.CreatedAt, entry.UpdatedAt = nowUTC(), nowUTC()
	knowledgeStore.Lock()
	knowledgeStore.dictionary[entry.ID] = entry
	knowledgeStore.Unlock()
	recordAudit("knowledge.dictionary.created", "enterprise", entry.Owner, map[string]interface{}{"entry_id": entry.ID, "dataset_id": entry.DatasetID, "variable": entry.Variable})
	c.JSON(201, entry)
}
func handleListDictionaryEntries(c *gin.Context) {
	user := knowledgeActor(c)
	if !knowledgeReadable(user, "knowledge") {
		c.JSON(403, gin.H{"error": "knowledge access denied"})
		return
	}
	dataset := c.Query("dataset_id")
	knowledgeStore.RLock()
	out := make([]DataDictionaryEntry, 0)
	for _, entry := range knowledgeStore.dictionary {
		if dataset == "" || entry.DatasetID == dataset {
			out = append(out, entry)
		}
	}
	knowledgeStore.RUnlock()
	c.JSON(200, gin.H{"count": len(out), "entries": out})
}
func handleKnowledgeAIContext(c *gin.Context) {
	user := knowledgeActor(c)
	if !knowledgeReadable(user, "knowledge") {
		c.JSON(403, gin.H{"error": "knowledge access denied"})
		return
	}
	entityType, entityID := c.Query("entity_type"), c.Query("entity_id")
	if entityType == "" || entityID == "" {
		c.JSON(400, gin.H{"error": "entity_type and entity_id are required"})
		return
	}
	c.JSON(200, gin.H{"entity_context": contextSummary(entityType, entityID), "sources": []map[string]string{{"type": entityType, "id": entityID, "deep_link": deepLink(entityType, entityID)}}, "instruction": "Use only these enterprise records as factual sources; cite their IDs in generated output."})
}

func handleKnowledgeQuality(c *gin.Context) {
	user := knowledgeActor(c)
	if !knowledgeReadable(user, "knowledge") {
		c.JSON(403, gin.H{"error": "knowledge access denied"})
		return
	}
	now := time.Now().UTC()
	issues := make([]map[string]interface{}, 0)
	knowledgeStore.RLock()
	defer knowledgeStore.RUnlock()
	seenTitles := map[string]string{}
	for _, article := range knowledgeStore.articles {
		if article.Owner == "" {
			issues = append(issues, map[string]interface{}{"type": "missing_owner", "article_id": article.ID})
		}
		if article.Status != "published" && article.Status != "archived" {
			issues = append(issues, map[string]interface{}{"type": "unapproved_content", "article_id": article.ID, "status": article.Status})
		}
		if article.ReviewDate != "" {
			if date, err := time.Parse(time.RFC3339, article.ReviewDate); err == nil && date.Before(now) {
				issues = append(issues, map[string]interface{}{"type": "review_due", "article_id": article.ID, "review_date": article.ReviewDate})
			}
		}
		key := strings.ToLower(strings.TrimSpace(article.Title))
		if prior, exists := seenTitles[key]; key != "" && exists {
			issues = append(issues, map[string]interface{}{"type": "duplicate_article", "article_id": article.ID, "duplicate_of": prior})
		} else if key != "" {
			seenTitles[key] = article.ID
		}
	}
	c.JSON(200, gin.H{"count": len(issues), "issues": issues})
}
