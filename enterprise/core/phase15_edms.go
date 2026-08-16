package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════
// PHASE 15 — EXTENDED EDMS: Version Control, Folders, Knowledge,
//             Archives, Search, Comments, Metadata, Classification
// ═══════════════════════════════════════════════════════════════════

// ─── Additional Models ────────────────────────────────────────────

type DocumentVersion struct {
	ID             string    `json:"id"`
	DocumentID     string    `json:"document_id"`
	VersionNumber  string    `json:"version_number"`
	ChangeSummary  string    `json:"change_summary"`
	AuthorName     string    `json:"author_name"`
	SHA256Hash     string    `json:"sha256_hash"`
	FileSizeKB     int       `json:"file_size_kb"`
	IsMajorVersion bool      `json:"is_major_version"`
	CreatedAt      time.Time `json:"created_at"`
}

type DocumentFolder struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	ParentPath  string    `json:"parent_path,omitempty"`
	Description string    `json:"description,omitempty"`
	AccessLevel string    `json:"access_level"`
	DocCount    int       `json:"document_count"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type DocumentRepository struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	RepositoryType string    `json:"repository_type"`
	AccessLevel    string    `json:"access_level"`
	Department     string    `json:"owner_department,omitempty"`
	DocumentCount  int       `json:"document_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type DocumentComment struct {
	ID          string    `json:"id"`
	DocumentID  string    `json:"document_id"`
	AuthorName  string    `json:"author_name"`
	AuthorRole  string    `json:"author_role,omitempty"`
	Comment     string    `json:"comment"`
	CommentType string    `json:"comment_type"`
	Resolved    bool      `json:"resolved"`
	ParentID    string    `json:"parent_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type DocumentMetadataField struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	DataType   string    `json:"data_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type EDMSKnowledgeArticle struct {
	ID                 string    `json:"id"`
	ArticleNumber      string    `json:"article_number"`
	Title              string    `json:"title"`
	ArticleType        string    `json:"article_type"`
	Content            string    `json:"content"`
	Summary            string    `json:"summary,omitempty"`
	AuthorName         string    `json:"author_name"`
	Department         string    `json:"department,omitempty"`
	Status             string    `json:"status"`
	Tags               []string  `json:"tags,omitempty"`
	RelatedDocumentIDs []string  `json:"related_document_ids,omitempty"`
	ViewCount          int       `json:"view_count"`
	HelpfulVotes       int       `json:"helpful_votes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type WikiPage struct {
	ID         string    `json:"id"`
	PageNumber string    `json:"page_number"`
	Title      string    `json:"title"`
	Slug       string    `json:"slug"`
	Content    string    `json:"content"`
	ParentSlug string    `json:"parent_slug,omitempty"`
	AuthorName string    `json:"author_name"`
	Status     string    `json:"status"`
	Tags       []string  `json:"tags,omitempty"`
	ViewCount  int       `json:"view_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ArchiveRecord struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	ArchiveType         string    `json:"archive_type"`
	LocationDescription string    `json:"location_description,omitempty"`
	RetentionPolicyID   string    `json:"retention_policy_id,omitempty"`
	DocumentCount       int       `json:"document_count"`
	SizeMB              float64   `json:"size_mb"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
}

type DispositionWorkflow struct {
	ID                       string    `json:"id"`
	DocumentID               string    `json:"document_id"`
	DocumentTitle            string    `json:"document_title,omitempty"`
	DocumentNumber           string    `json:"document_number,omitempty"`
	RetentionPolicyID        string    `json:"retention_policy_id,omitempty"`
	ScheduledDispositionDate string    `json:"scheduled_disposition_date"`
	DispositionAction        string    `json:"disposition_action"`
	Status                   string    `json:"status"`
	ReviewedBy               string    `json:"reviewed_by,omitempty"`
	ApprovedBy               string    `json:"approved_by,omitempty"`
	Notes                    string    `json:"notes,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
}

type OCRResult struct {
	ID               string    `json:"id"`
	DocumentID       string    `json:"document_id"`
	ExtractedText    string    `json:"extracted_text,omitempty"`
	ConfidenceScore  float64   `json:"confidence_score"`
	LanguageDetected string    `json:"language_detected"`
	PageCount        int       `json:"page_count"`
	ProcessingStatus string    `json:"processing_status"`
	OCREngine        string    `json:"ocr_engine"`
	ProcessingTimeMs int       `json:"processing_time_ms"`
	CreatedAt        time.Time `json:"created_at"`
}

// ─── Seed Data ────────────────────────────────────────────────────

var defaultFolders = []DocumentFolder{
	{ID: "fld-001", Name: "Policies", Path: "/Policies", AccessLevel: "INTERNAL", DocCount: 12, CreatedBy: "Director General", CreatedAt: time.Now().Add(-180 * 24 * time.Hour)},
	{ID: "fld-002", Name: "Data Governance", Path: "/Policies/Data_Governance", ParentPath: "/Policies", AccessLevel: "INTERNAL", DocCount: 4, CreatedAt: time.Now().Add(-90 * 24 * time.Hour)},
	{ID: "fld-003", Name: "Statistical Bulletins", Path: "/Publications/Statistical_Bulletins", ParentPath: "/Publications", AccessLevel: "PUBLIC", DocCount: 38, CreatedAt: time.Now().Add(-120 * 24 * time.Hour)},
	{ID: "fld-004", Name: "Clinical Protocols", Path: "/Research/Clinical_Protocols", ParentPath: "/Research", AccessLevel: "CONFIDENTIAL", DocCount: 6, CreatedAt: time.Now().Add(-60 * 24 * time.Hour)},
	{ID: "fld-005", Name: "Grant Agreements", Path: "/Finance/Grant_Agreements", ParentPath: "/Finance", AccessLevel: "RESTRICTED", DocCount: 9, CreatedAt: time.Now().Add(-30 * 24 * time.Hour)},
	{ID: "fld-006", Name: "Standard Operating Procedures", Path: "/SOPs", AccessLevel: "INTERNAL", DocCount: 21, CreatedAt: time.Now().Add(-200 * 24 * time.Hour)},
	{ID: "fld-007", Name: "Meeting Minutes", Path: "/Governance/Meeting_Minutes", ParentPath: "/Governance", AccessLevel: "INTERNAL", DocCount: 14, CreatedAt: time.Now().Add(-45 * 24 * time.Hour)},
	{ID: "fld-008", Name: "Legal & Compliance", Path: "/Legal", AccessLevel: "RESTRICTED", DocCount: 7, CreatedAt: time.Now().Add(-150 * 24 * time.Hour)},
}

var defaultRepositories = []DocumentRepository{
	{ID: "repo-001", Name: "National Policy Repository", Description: "Official institutional policy documents, SOPs, and governance frameworks", RepositoryType: "Policies", AccessLevel: "INTERNAL", Department: "Governance & Standards", DocumentCount: 47, CreatedAt: time.Now().Add(-365 * 24 * time.Hour)},
	{ID: "repo-002", Name: "Statistical Publications Archive", Description: "National statistical bulletins, census reports, and indicator releases", RepositoryType: "Publications", AccessLevel: "PUBLIC", Department: "Dissemination Directorate", DocumentCount: 312, CreatedAt: time.Now().Add(-365 * 24 * time.Hour)},
	{ID: "repo-003", Name: "Research & Clinical Protocols", Description: "IRB-approved research protocols, clinical trial documents, and ethics clearances", RepositoryType: "Research", AccessLevel: "CONFIDENTIAL", Department: "Epidemiology & Research (RMS)", DocumentCount: 28, CreatedAt: time.Now().Add(-200 * 24 * time.Hour)},
	{ID: "repo-004", Name: "Legal & Contracts Repository", Description: "Grant agreements, procurement contracts, MoUs, and legal correspondence", RepositoryType: "Contracts", AccessLevel: "RESTRICTED", Department: "Legal & Procurement Directorate", DocumentCount: 64, CreatedAt: time.Now().Add(-300 * 24 * time.Hour)},
}

var defaultVersions = []DocumentVersion{
	{ID: "ver-001", DocumentID: "doc-001", VersionNumber: "v1.0", ChangeSummary: "Initial draft submitted for review", AuthorName: "Director of Statistical Standards", SHA256Hash: "da39a3ee5e6b4b0d3255bfef95601890afd80709", FileSizeKB: 1820, IsMajorVersion: true, CreatedAt: time.Now().Add(-70 * 24 * time.Hour)},
	{ID: "ver-002", DocumentID: "doc-001", VersionNumber: "v1.5", ChangeSummary: "Incorporated Legal Counsel feedback on GDPR compliance clauses", AuthorName: "Chief Legal Counsel", SHA256Hash: "aab3238922bcc25a6f606eb525ffdc56", FileSizeKB: 2110, IsMajorVersion: false, CreatedAt: time.Now().Add(-30 * 24 * time.Hour)},
	{ID: "ver-003", DocumentID: "doc-001", VersionNumber: "v2.1", ChangeSummary: "Final approved version — anonymization thresholds revised to k=5", AuthorName: "Director General", SHA256Hash: "e3b0c44298fc1c149afbf4c8996fb924", FileSizeKB: 2450, IsMajorVersion: true, CreatedAt: time.Now().Add(-10 * 24 * time.Hour)},
	{ID: "ver-004", DocumentID: "doc-002", VersionNumber: "v1.0", ChangeSummary: "Preliminary bulletin issued for internal review", AuthorName: "Lead Demographer & Survey Team", SHA256Hash: "8f434346648f6b96df89dda901c5176", FileSizeKB: 5120, IsMajorVersion: true, CreatedAt: time.Now().Add(-14 * 24 * time.Hour)},
}

var defaultComments = []DocumentComment{
	{ID: "cmt-001", DocumentID: "doc-001", AuthorName: "Chief Legal Counsel", AuthorRole: "Legal", Comment: "Section 4.2 requires explicit reference to the Personal Data Protection Act 2024. Please update before final approval.", CommentType: "Review", Resolved: true, CreatedAt: time.Now().Add(-25 * 24 * time.Hour)},
	{ID: "cmt-002", DocumentID: "doc-001", AuthorName: "Director of Statistical Standards", AuthorRole: "Technical", Comment: "Anonymization threshold updated to k=5 per international best practice (ESS Handbook 2023). Addressed.", CommentType: "Annotation", Resolved: true, ParentID: "cmt-001", CreatedAt: time.Now().Add(-20 * 24 * time.Hour)},
	{ID: "cmt-003", DocumentID: "doc-003", AuthorName: "IRB Chairperson", AuthorRole: "Ethics", Comment: "Ethics clearance requires enrollment of 1,200 minimum participants. Protocol currently states 800. Please revise sample size justification.", CommentType: "Review", Resolved: false, CreatedAt: time.Now().Add(-3 * 24 * time.Hour)},
}

var defaultKnowledgeArticles = []EDMSKnowledgeArticle{
	{
		ID: "art-001", ArticleNumber: "KA-GOV-2026-001",
		Title: "Statistical Data Governance Framework — Key Principles", ArticleType: "Knowledge_Article",
		Content: "## Overview\nThis article summarises the key principles of the National Statistical Data Governance Framework.\n\n## Core Principles\n1. **Data Sovereignty** — All national statistical data is an institutional asset.\n2. **Anonymization First** — No microdata shall be released without k-anonymity (k>=5).\n3. **Audit Trails** — Every data access event must be logged.",
		Summary: "Key principles of the national data governance framework.", AuthorName: "Director of Statistical Standards", Department: "Governance & Standards",
		Status: "Published", Tags: []string{"governance", "data-quality", "privacy"}, ViewCount: 142, HelpfulVotes: 38,
		CreatedAt: time.Now().Add(-45 * 24 * time.Hour), UpdatedAt: time.Now().Add(-5 * 24 * time.Hour),
	},
	{
		ID: "art-002", ArticleNumber: "KA-RES-2026-004",
		Title: "IRB Ethics Submission Process — Step-by-Step Guide", ArticleType: "Procedure",
		Content: "## Process Overview\nAll research involving human subjects must receive IRB clearance before data collection begins.\n\n## Steps\n1. Complete the IRB Application Form (FORM-IRB-001)\n2. Attach study protocol and informed consent forms\n3. Submit to ethics@statgate.gov.ke\n4. Await review (minimum 21 working days)",
		Summary: "Step-by-step guide for IRB ethics submissions.", AuthorName: "IRB Secretariat", Department: "Epidemiology & Research (RMS)",
		Status: "Published", Tags: []string{"irb", "ethics", "research"}, ViewCount: 89, HelpfulVotes: 25,
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour), UpdatedAt: time.Now().Add(-10 * 24 * time.Hour),
	},
	{
		ID: "art-003", ArticleNumber: "KA-STA-2026-007",
		Title: "Survey Methodology Standards — Household Survey Design", ArticleType: "SOP",
		Content: "## Scope\nApplies to all national household surveys.\n\n## Sample Design\n- Stratified two-stage cluster sampling (PPS) is mandatory.\n- Minimum effective sample size: 4,000 households per domain.\n- Design effect (DEFF) must be accounted for.",
		Summary: "Official survey methodology standards for household survey design.", AuthorName: "Chief Methodologist", Department: "Statistical Methods & Standards",
		Status: "Published", Tags: []string{"sampling", "methodology", "household-survey"}, ViewCount: 203, HelpfulVotes: 67,
		CreatedAt: time.Now().Add(-90 * 24 * time.Hour), UpdatedAt: time.Now().Add(-2 * 24 * time.Hour),
	},
}

var defaultWikiPages = []WikiPage{
	{ID: "wiki-001", PageNumber: "WIKI-2026-001", Title: "StatGate Knowledge Base", Slug: "home", Content: "# StatGate Institutional Knowledge Base\n\nWelcome to the central institutional knowledge repository.\n\n## Quick Navigation\n- [Data Governance](governance)\n- [Research Procedures](research)\n- [Survey Standards](survey-standards)", AuthorName: "Knowledge Management Unit", Status: "Published", ViewCount: 891, CreatedAt: time.Now().Add(-365 * 24 * time.Hour), UpdatedAt: time.Now().Add(-2 * 24 * time.Hour)},
	{ID: "wiki-002", PageNumber: "WIKI-2026-002", Title: "Data Governance Hub", Slug: "governance", ParentSlug: "home", Content: "# Data Governance Hub\n\n## Policies\n- National Data Governance Policy (DOC-POL-2026-004)\n- Statistical Secrecy Guidelines\n- Data Sharing Framework", AuthorName: "Director of Statistical Standards", Status: "Published", ViewCount: 344, CreatedAt: time.Now().Add(-200 * 24 * time.Hour), UpdatedAt: time.Now().Add(-7 * 24 * time.Hour)},
}

var defaultArchives = []ArchiveRecord{
	{ID: "arc-001", Name: "National Statistical Long-Term Archive 2000-2020", ArchiveType: "Long_Term", LocationDescription: "Secured physical archive — Basement B2, NSO Headquarters", DocumentCount: 1847, SizeMB: 28450.5, IsActive: true, CreatedAt: time.Now().Add(-365 * 24 * time.Hour)},
	{ID: "arc-002", Name: "Operational Records Archive 2021-Present", ArchiveType: "Operational", LocationDescription: "On-premises NAS — Data Centre A, Rack 12", DocumentCount: 412, SizeMB: 9410.2, IsActive: true, CreatedAt: time.Now().Add(-180 * 24 * time.Hour)},
	{ID: "arc-003", Name: "Disaster Recovery Cloud Archive", ArchiveType: "Disaster_Recovery", LocationDescription: "AWS S3 Glacier — af-south-1 (Cape Town)", DocumentCount: 2259, SizeMB: 37860.0, IsActive: true, CreatedAt: time.Now().Add(-90 * 24 * time.Hour)},
}

var defaultDispositions = []DispositionWorkflow{
	{ID: "disp-001", DocumentID: "doc-002", DocumentTitle: "NHW Food Security Survey 2019 Microdata", DocumentNumber: "DOC-MIC-2019-044", ScheduledDispositionDate: "2069-12-31", DispositionAction: "Permanent_National_Archive", Status: "Scheduled", Notes: "50-year retention per National Records & Archives Act 2002, Section 14.", CreatedAt: time.Now().Add(-30 * 24 * time.Hour)},
	{ID: "disp-002", DocumentID: "doc-003", DocumentTitle: "Global Fund Procurement Records — Round 3", DocumentNumber: "DOC-FIN-2019-128", ScheduledDispositionDate: "2026-12-31", DispositionAction: "Review_For_Destruction", Status: "Under_Review", ReviewedBy: "Chief Financial Officer", Notes: "7-year procurement retention cycle expires Q4 2026.", CreatedAt: time.Now().Add(-14 * 24 * time.Hour)},
}

// ─── Version Control Handlers ─────────────────────────────────────

func handleListDocumentVersions(c *gin.Context) {
	docID := c.Param("id")
	var versions []DocumentVersion
	for _, v := range defaultVersions {
		if v.DocumentID == docID {
			versions = append(versions, v)
		}
	}
	if versions == nil {
		versions = []DocumentVersion{}
	}
	c.JSON(http.StatusOK, versions)
}

func handleCheckoutDocument(c *gin.Context) {
	docID := c.Param("id")
	var req struct {
		UserName string `json:"user_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for i, d := range defaultDocuments {
		if d.ID == docID {
			if d.CheckOutUser != "" {
				c.JSON(http.StatusConflict, gin.H{
					"error":         "document_locked",
					"message":       fmt.Sprintf("Document is currently checked out by %s", d.CheckOutUser),
					"checkout_user": d.CheckOutUser,
				})
				return
			}
			defaultDocuments[i].CheckOutUser = req.UserName
			defaultDocuments[i].UpdatedTime = time.Now()
			c.JSON(http.StatusOK, gin.H{
				"status":         "checked_out",
				"document_id":    docID,
				"checkout_user":  req.UserName,
				"checked_out_at": time.Now(),
				"message":        "Document locked for editing. Remember to check in when done.",
			})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
}

func handleCheckinDocument(c *gin.Context) {
	docID := c.Param("id")
	var req struct {
		UserName      string `json:"user_name"`
		ChangeSummary string `json:"change_summary"`
		Content       string `json:"content,omitempty"`
		IsMajor       bool   `json:"is_major_version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for i, d := range defaultDocuments {
		if d.ID == docID {
			newVer := bumpVersion(d.Version, req.IsMajor)
			h := sha256.Sum256([]byte(d.Content + req.ChangeSummary + time.Now().String()))
			newVersion := DocumentVersion{
				ID: fmt.Sprintf("ver-%d", time.Now().Unix()), DocumentID: docID,
				VersionNumber: newVer, ChangeSummary: req.ChangeSummary, AuthorName: req.UserName,
				SHA256Hash: fmt.Sprintf("%x", h), FileSizeKB: d.FileSizeKB, IsMajorVersion: req.IsMajor, CreatedAt: time.Now(),
			}
			defaultVersions = append(defaultVersions, newVersion)
			defaultDocuments[i].CheckOutUser = ""
			defaultDocuments[i].Version = newVer
			defaultDocuments[i].UpdatedTime = time.Now()
			if req.Content != "" {
				defaultDocuments[i].Content = req.Content
				h2 := sha256.Sum256([]byte(req.Content))
				defaultDocuments[i].SHA256Hash = fmt.Sprintf("%x", h2)
			}
			c.JSON(http.StatusOK, gin.H{"status": "checked_in", "new_version": newVer, "version_record": newVersion, "document": defaultDocuments[i]})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
}

func bumpVersion(current string, major bool) string {
	var maj, min int
	fmt.Sscanf(current, "v%d.%d", &maj, &min)
	if major {
		return fmt.Sprintf("v%d.0", maj+1)
	}
	return fmt.Sprintf("v%d.%d", maj, min+1)
}

// ─── Folder & Repository Handlers ────────────────────────────────

func handleListFolders(c *gin.Context) {
	c.JSON(http.StatusOK, defaultFolders)
}

func handleCreateFolder(c *gin.Context) {
	var req DocumentFolder
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("fld-%d", time.Now().Unix())
	req.DocCount = 0
	req.CreatedAt = time.Now()
	defaultFolders = append(defaultFolders, req)
	c.JSON(http.StatusCreated, req)
}

func handleListRepositories(c *gin.Context) {
	c.JSON(http.StatusOK, defaultRepositories)
}

func handleCreateRepository(c *gin.Context) {
	var req DocumentRepository
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("repo-%d", time.Now().Unix())
	req.DocumentCount = 0
	req.CreatedAt = time.Now()
	defaultRepositories = append(defaultRepositories, req)
	c.JSON(http.StatusCreated, req)
}

// ─── Document Search ──────────────────────────────────────────────

func handleDocumentSearch(c *gin.Context) {
	query := strings.ToLower(c.Query("q"))
	category := c.Query("category")
	classification := c.Query("classification")
	status := c.Query("status")

	var results []EnterpriseDocument
	for _, d := range defaultDocuments {
		textMatch := query == "" ||
			strings.Contains(strings.ToLower(d.Title), query) ||
			strings.Contains(strings.ToLower(d.DocumentNumber), query) ||
			strings.Contains(strings.ToLower(d.Content), query) ||
			strings.Contains(strings.ToLower(d.Department), query) ||
			strings.Contains(strings.ToLower(strings.Join(d.Tags, " ")), query)
		catMatch := category == "" || strings.EqualFold(d.Category, category)
		classMatch := classification == "" || strings.EqualFold(d.Classification, classification)
		statusMatch := status == "" || strings.EqualFold(d.Status, status)
		if textMatch && catMatch && classMatch && statusMatch {
			results = append(results, d)
		}
	}
	if results == nil {
		results = []EnterpriseDocument{}
	}
	c.JSON(http.StatusOK, gin.H{"query": query, "total_hits": len(results), "results": results, "search_type": "keyword", "indexed_at": time.Now().Format(time.RFC3339)})
}

// ─── Document Comments ────────────────────────────────────────────

func handleListDocumentComments(c *gin.Context) {
	docID := c.Param("id")
	var comments []DocumentComment
	for _, cmt := range defaultComments {
		if cmt.DocumentID == docID {
			comments = append(comments, cmt)
		}
	}
	if comments == nil {
		comments = []DocumentComment{}
	}
	c.JSON(http.StatusOK, comments)
}

func handleCreateDocumentComment(c *gin.Context) {
	docID := c.Param("id")
	var req DocumentComment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("cmt-%d", time.Now().Unix())
	req.DocumentID = docID
	req.CreatedAt = time.Now()
	if req.CommentType == "" {
		req.CommentType = "General"
	}
	defaultComments = append(defaultComments, req)
	c.JSON(http.StatusCreated, req)
}

func handleResolveDocumentComment(c *gin.Context) {
	cmtID := c.Param("commentId")
	for i, cmt := range defaultComments {
		if cmt.ID == cmtID {
			defaultComments[i].Resolved = true
			c.JSON(http.StatusOK, defaultComments[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
}

// ─── Knowledge Articles ───────────────────────────────────────────

func handleListEDMSKnowledgeArticles(c *gin.Context) {
	articleType := c.Query("type")
	var results []EDMSKnowledgeArticle
	for _, a := range defaultKnowledgeArticles {
		if articleType == "" || strings.EqualFold(a.ArticleType, articleType) {
			results = append(results, a)
		}
	}
	if results == nil {
		results = []EDMSKnowledgeArticle{}
	}
	c.JSON(http.StatusOK, results)
}

func handleGetEDMSKnowledgeArticle(c *gin.Context) {
	id := c.Param("id")
	for i, a := range defaultKnowledgeArticles {
		if a.ID == id || a.ArticleNumber == id {
			defaultKnowledgeArticles[i].ViewCount++
			c.JSON(http.StatusOK, defaultKnowledgeArticles[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
}

func handleCreateEDMSKnowledgeArticle(c *gin.Context) {
	var req EDMSKnowledgeArticle
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("art-%d", time.Now().Unix())
	typeCode := "GEN"
	if len(req.ArticleType) >= 3 {
		typeCode = strings.ToUpper(req.ArticleType[:3])
	}
	req.ArticleNumber = fmt.Sprintf("KA-%s-2026-%03d", typeCode, len(defaultKnowledgeArticles)+1)
	req.Status = "Draft"
	req.ViewCount = 0
	req.HelpfulVotes = 0
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	defaultKnowledgeArticles = append(defaultKnowledgeArticles, req)
	c.JSON(http.StatusCreated, req)
}

// ─── Wiki Pages ───────────────────────────────────────────────────

func handleListWikiPages(c *gin.Context) {
	c.JSON(http.StatusOK, defaultWikiPages)
}

func handleGetWikiPage(c *gin.Context) {
	slug := c.Param("slug")
	for i, p := range defaultWikiPages {
		if p.Slug == slug || p.ID == slug {
			defaultWikiPages[i].ViewCount++
			c.JSON(http.StatusOK, defaultWikiPages[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "wiki page not found"})
}

// ─── Archive Management ───────────────────────────────────────────

func handleListArchives(c *gin.Context) {
	c.JSON(http.StatusOK, defaultArchives)
}

func handleGetArchiveSummary(c *gin.Context) {
	totalDocs := 0
	totalMB := 0.0
	for _, a := range defaultArchives {
		totalDocs += a.DocumentCount
		totalMB += a.SizeMB
	}
	c.JSON(http.StatusOK, gin.H{
		"total_archives": len(defaultArchives), "total_archived_docs": totalDocs,
		"total_size_gb": totalMB / 1024, "archives": defaultArchives,
		"last_sync": time.Now().Format(time.RFC3339),
	})
}

func handleArchiveDocument(c *gin.Context) {
	docID := c.Param("id")
	var req struct {
		ArchiveID  string `json:"archive_id"`
		Reason     string `json:"reason"`
		ArchivedBy string `json:"archived_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for i, d := range defaultDocuments {
		if d.ID == docID {
			defaultDocuments[i].Status = "Archived"
			defaultDocuments[i].UpdatedTime = time.Now()
			c.JSON(http.StatusOK, gin.H{"status": "archived", "document_id": docID, "archive_id": req.ArchiveID, "archived_by": req.ArchivedBy, "archived_at": time.Now()})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
}

// ─── Disposition Workflows ────────────────────────────────────────

func handleListDispositions(c *gin.Context) {
	c.JSON(http.StatusOK, defaultDispositions)
}

func handleCreateDisposition(c *gin.Context) {
	var req DispositionWorkflow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("disp-%d", time.Now().Unix())
	req.Status = "Scheduled"
	req.CreatedAt = time.Now()
	defaultDispositions = append(defaultDispositions, req)
	c.JSON(http.StatusCreated, req)
}

// ─── Legal Hold Write ─────────────────────────────────────────────

func handleCreateLegalHold(c *gin.Context) {
	var req LegalHoldRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("hold-%d", time.Now().Unix())
	req.Status = "Active"
	if req.IssuedDate == "" {
		req.IssuedDate = time.Now().Format("2006-01-02")
	}
	req.CreatedTime = time.Now()
	defaultLegalHolds = append(defaultLegalHolds, req)
	c.JSON(http.StatusCreated, req)
}

func handleReleaseLegalHold(c *gin.Context) {
	id := c.Param("id")
	for i, h := range defaultLegalHolds {
		if h.ID == id {
			defaultLegalHolds[i].Status = "Released"
			c.JSON(http.StatusOK, gin.H{"status": "released", "hold_id": id, "released_at": time.Now(), "record": defaultLegalHolds[i]})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "legal hold not found"})
}

// ─── Retention Policy Write ───────────────────────────────────────

func handleCreateRetentionPolicy(c *gin.Context) {
	var req RetentionPolicy
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("ret-%d", time.Now().Unix())
	defaultRetentionPolicies = append(defaultRetentionPolicies, req)
	c.JSON(http.StatusCreated, req)
}

// ─── OCR Service Stub ─────────────────────────────────────────────

func handleSubmitOCR(c *gin.Context) {
	docID := c.Param("id")
	result := OCRResult{
		ID: fmt.Sprintf("ocr-%d", time.Now().Unix()), DocumentID: docID,
		ExtractedText: "[OCR_PENDING] Document queued for optical character recognition. Estimated: 2-5 minutes.",
		ProcessingStatus: "Queued", OCREngine: "Tesseract-5.0 + Google Vision API", CreatedAt: time.Now(),
	}
	c.JSON(http.StatusAccepted, result)
}

func handleGetOCRResult(c *gin.Context) {
	docID := c.Param("id")
	c.JSON(http.StatusOK, OCRResult{
		ID: fmt.Sprintf("ocr-demo-%s", docID), DocumentID: docID,
		ExtractedText: "National Statistical Data Governance Policy\n\n1. Purpose\nThis policy establishes mandatory controls for anonymization of national census microdata...\n\n2. Scope\nApplies to all data custodians, researchers, and third-party data users...",
		ConfidenceScore: 94.7, LanguageDetected: "en", PageCount: 12,
		ProcessingStatus: "Completed", OCREngine: "Tesseract-5.0 + Google Vision API",
		ProcessingTimeMs: 3420, CreatedAt: time.Now().Add(-10 * time.Minute),
	})
}

// ─── Document Metadata ─────────────────────────────────────────────

func handleGetDocumentMetadata(c *gin.Context) {
	docID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"document_id": docID,
		"metadata": []DocumentMetadataField{
			{ID: "meta-001", DocumentID: docID, Key: "sdmx_dsd_id", Value: "STATGATE:NATPOLICY(1.0)", DataType: "string", CreatedAt: time.Now()},
			{ID: "meta-002", DocumentID: docID, Key: "language", Value: "en", DataType: "string", CreatedAt: time.Now()},
			{ID: "meta-003", DocumentID: docID, Key: "review_cycle", Value: "annual", DataType: "string", CreatedAt: time.Now()},
			{ID: "meta-004", DocumentID: docID, Key: "subject_classification", Value: "Data Governance; Statistical Methods", DataType: "string", CreatedAt: time.Now()},
			{ID: "meta-005", DocumentID: docID, Key: "confidentiality_end_date", Value: "2031-12-31", DataType: "date", CreatedAt: time.Now()},
		},
	})
}

// ─── EDMS Full Dashboard ──────────────────────────────────────────

func handleEDMSDashboard(c *gin.Context) {
	totalDocs := len(defaultDocuments)
	archived, inReview, approved, published, legalHold := 0, 0, 0, 0, 0
	for _, d := range defaultDocuments {
		switch d.Status {
		case "Archived":
			archived++
		case "In_Review":
			inReview++
		case "Approved":
			approved++
		case "Published":
			published++
		case "Legal_Hold":
			legalHold++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"total_documents": totalDocs,
		"by_status": gin.H{
			"draft": totalDocs - archived - inReview - approved - published - legalHold,
			"in_review": inReview, "approved": approved, "published": published,
			"archived": archived, "legal_hold": legalHold,
		},
		"total_folders": len(defaultFolders), "total_repositories": len(defaultRepositories),
		"total_templates": len(defaultTemplates), "total_signatures": len(defaultSignatures),
		"active_legal_holds": len(defaultLegalHolds), "retention_policies": len(defaultRetentionPolicies),
		"knowledge_articles": len(defaultKnowledgeArticles), "wiki_pages": len(defaultWikiPages),
		"total_archives": len(defaultArchives), "pending_dispositions": len(defaultDispositions),
		"total_repository_size_mb": 9410.2, "iso_15489_compliance": "CERTIFIED_COMPLIANT",
		"last_updated": time.Now().Format(time.RFC3339),
	})
}
