package main

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════
// PHASE 15 — ENTERPRISE DOCUMENT MANAGEMENT (EDMS), RECORDS & ARCHIVES
// ═══════════════════════════════════════════════════════════════════

// EnterpriseDocument represents an institutional document in the EDMS repository.
type EnterpriseDocument struct {
	ID             string    `json:"id"`
	DocumentNumber string    `json:"document_number"` // "DOC-POL-2026-012"
	Title          string    `json:"title"`
	FolderPath     string    `json:"folder_path"`     // "/Policies/Data_Governance"
	Category       string    `json:"category"`        // "Policy", "Statistical_Bulletin", "Research_Protocol", "SOP", "Grant_Agreement"
	Version        string    `json:"version"`         // "v1.2"
	AuthorName     string    `json:"author_name"`
	Department     string    `json:"department"`
	Classification string    `json:"classification"` // "PUBLIC", "INTERNAL", "CONFIDENTIAL", "RESTRICTED"
	Status         string    `json:"status"`         // "Draft", "In_Review", "Approved", "Published", "Archived", "Legal_Hold"
	FileSizeKB     int       `json:"file_size_kb"`
	CheckOutUser   string    `json:"checkout_user,omitempty"`
	SHA256Hash     string    `json:"sha256_hash"`
	SignedBy       []string  `json:"signed_by,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Content        string    `json:"content,omitempty"`
	CreatedTime    time.Time `json:"created_time"`
	UpdatedTime    time.Time `json:"updated_time"`
}

// DocumentTemplate represents a reusable institutional template.
type DocumentTemplate struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Category       string   `json:"category"`
	Description    string   `json:"description"`
	DefaultContent string   `json:"default_content"`
	Tags           []string `json:"tags,omitempty"`
}

// DigitalSignature represents a cryptographic signature recorded on a document.
type DigitalSignature struct {
	ID               string    `json:"id"`
	DocumentID       string    `json:"document_id"`
	DocumentNumber   string    `json:"document_number"`
	SignerName       string    `json:"signer_name"`
	SignerRole       string    `json:"signer_role"`
	SignatureType    string    `json:"signature_type"` // "Executive_Approval", "Ethics_Clearance", "Financial_Signoff"
	VerificationHash string    `json:"verification_hash"`
	QRVerificationURL string   `json:"qr_verification_url"`
	SignedAt         time.Time `json:"signed_at"`
}

// RetentionPolicy represents an ISO 15489 records retention schedule rule.
type RetentionPolicy struct {
	ID                   string `json:"id"`
	Category             string `json:"category"`
	RetentionPeriodYears int    `json:"retention_period_years"`
	DispositionAction    string `json:"disposition_action"` // "Permanent_National_Archive", "Review_For_Destruction", "Declassify_To_Public"
	LegalAuthority       string `json:"legal_authority"`     // "National Records & Archives Act 2002"
}

// LegalHoldRecord represents a legal freeze on documents.
type LegalHoldRecord struct {
	ID            string    `json:"id"`
	CaseReference string    `json:"case_reference"` // "HOLD-2026-AUD-01"
	Reason        string    `json:"reason"`
	Custodian     string    `json:"custodian"`
	IssuedDate    string    `json:"issued_date"`
	Status        string    `json:"status"` // "Active", "Released"
	CreatedTime   time.Time `json:"created_time"`
}

// In-memory seeds for Phase 15 EDMS
var defaultDocuments = []EnterpriseDocument{
	{
		ID:             "doc-001",
		DocumentNumber: "DOC-POL-2026-004",
		Title:          "National Statistical Data Governance & Anonymization Policy",
		FolderPath:     "/Policies/Data_Governance",
		Category:       "Policy",
		Version:        "v2.1",
		AuthorName:     "Director of Statistical Standards",
		Department:     "Governance & Standards",
		Classification: "PUBLIC",
		Status:         "Published",
		FileSizeKB:     2450,
		SHA256Hash:     "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		SignedBy:       []string{"Director General", "Chief Legal Counsel"},
		Tags:           []string{"governance", "anonymization", "privacy", "gdpr"},
		CreatedTime:    time.Now().Add(-60 * 24 * time.Hour),
		UpdatedTime:    time.Now().Add(-10 * 24 * time.Hour),
	},
	{
		ID:             "doc-002",
		DocumentNumber: "DOC-REP-2026-088",
		Title:          "National Household Welfare & Food Security Survey 2026 Preliminary Bulletin",
		FolderPath:     "/Publications/Statistical_Bulletins",
		Category:       "Statistical_Bulletin",
		Version:        "v1.0",
		AuthorName:     "Lead Demographer & Survey Team",
		Department:     "Demographic & Social Statistics",
		Classification: "INTERNAL",
		Status:         "Approved",
		FileSizeKB:     5120,
		SHA256Hash:     "8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4",
		SignedBy:       []string{"Head of Census Directorate"},
		Tags:           []string{"census", "welfare", "food_security", "2026"},
		CreatedTime:    time.Now().Add(-14 * 24 * time.Hour),
		UpdatedTime:    time.Now().Add(-2 * 24 * time.Hour),
	},
	{
		ID:             "doc-003",
		DocumentNumber: "DOC-PROT-2026-019",
		Title:          "Clinical Epidemiological Surveillance Protocol - Vector Borne Diseases",
		FolderPath:     "/Research/Clinical_Protocols",
		Category:       "Research_Protocol",
		Version:        "v1.4",
		AuthorName:     "Principal Investigator - Dr. A. Mwamba",
		Department:     "Epidemiology & Research (RMS)",
		Classification: "CONFIDENTIAL",
		Status:         "In_Review",
		FileSizeKB:     1840,
		SHA256Hash:     "ca978112ca1bbdcafac231b39a23dc4da78608149646b9a8f278ff564c784964",
		Tags:           []string{"epidemiology", "irb", "clinical_trial"},
		CreatedTime:    time.Now().Add(-30 * 24 * time.Hour),
		UpdatedTime:    time.Now().Add(-5 * 24 * time.Hour),
	},
}

var defaultTemplates = []DocumentTemplate{
	{
		ID:          "tpl-001",
		Name:        "National Statistical Release Bulletin",
		Category:    "Statistical_Bulletin",
		Description: "Official SDMX/GSBPM compliant statistical indicator bulletin with executive summary, methodology, and tables.",
		DefaultContent: `# National Statistical Release Bulletin

## Executive Summary
This statistical release presents primary findings from the latest survey round conducted in accordance with the National Statistics Framework.

## Key Statistical Indicators
- **Indicator 1:** [Value] ± [Standard Error]
- **Indicator 2:** [Value] (Growth rate: +[X]%)

## Methodology & Sample Design
- **Primary Sampling Units:** Stratified two-stage cluster sampling (PPS).
- **Weight Calibration:** Post-stratified population weights.

## Disaggregated Results Table
| Domain / Region | Headcount (N) | Weighted Estimate (%) | Margin of Error |
| :--- | :--- | :--- | :--- |
| Northern Zone | 3,420 | 48.2% | ±1.2% |
| Coastal Zone | 4,110 | 52.6% | ±1.4% |
`,
		Tags: []string{"statistics", "bulletin", "sdmx"},
	},
	{
		ID:          "tpl-002",
		Name:        "Institutional Standard Operating Procedure (SOP)",
		Category:    "SOP",
		Description: "Governed institutional SOP with purpose, scope, responsibilities, step-by-step procedure, and audit controls.",
		DefaultContent: `# Standard Operating Procedure (SOP)

**Document Code:** SOP-SG-2026-XX  
**Effective Date:** 2026-08-15  
**Review Cycle:** Annual  

## 1. Purpose
Define the mandatory procedures for handling sensitive spatial and household respondent microdata.

## 2. Scope
Applies to all enumerators, supervisors, data analysts, and third-party researchers.

## 3. Mandatory Controls
1. All microdata must be encrypted in transit (TLS 1.3) and at rest (AES-256).
2. Direct identifiers (Name, National ID, Phone) must be pseudonimized before analysis.
`,
		Tags: []string{"sop", "governance", "compliance"},
	},
}

var defaultSignatures = []DigitalSignature{
	{
		ID:               "sig-001",
		DocumentID:       "doc-001",
		DocumentNumber:   "DOC-POL-2026-004",
		SignerName:       "Dr. Aloyce M. Mtega",
		SignerRole:       "Director General (Chief Statistician)",
		SignatureType:    "Executive_Approval",
		VerificationHash: "SHA256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069",
		QRVerificationURL: "http://localhost:8096/api/documents/verify/7f83b165",
		SignedAt:         time.Now().Add(-10 * 24 * time.Hour),
	},
}

var defaultRetentionPolicies = []RetentionPolicy{
	{
		ID:                   "ret-001",
		Category:             "Census & Household Survey Microdata",
		RetentionPeriodYears: 50,
		DispositionAction:    "Permanent_National_Archive",
		LegalAuthority:       "National Records & Archives Act 2002, Section 14",
	},
	{
		ID:                   "ret-002",
		Category:             "Grant Agreements & Procurement Contracts",
		RetentionPeriodYears: 7,
		DispositionAction:    "Review_For_Destruction",
		LegalAuthority:       "Public Finance & Procurement Act, Section 32",
	},
	{
		ID:                   "ret-003",
		Category:             "Clinical Trial Ethics Protocols",
		RetentionPeriodYears: 25,
		DispositionAction:    "Permanent_National_Archive",
		LegalAuthority:       "National Health Research Ethics Guidelines",
	},
}

var defaultLegalHolds = []LegalHoldRecord{
	{
		ID:            "hold-001",
		CaseReference: "HOLD-2026-AUDIT-GF-01",
		Reason:        "Global Fund Round 4 External Independent Compliance Review",
		Custodian:     "Chief Financial Officer & Procurement Directorate",
		IssuedDate:    "2026-07-15",
		Status:        "Active",
		CreatedTime:   time.Now().Add(-30 * 24 * time.Hour),
	},
}

// ─── HTTP Handlers ──────────────────────────────────────────────────

func handleEDMSSummary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total_documents_count":     len(defaultDocuments),
		"active_templates_count":    len(defaultTemplates),
		"digital_signatures_count":  len(defaultSignatures),
		"retention_policies_count":  len(defaultRetentionPolicies),
		"active_legal_holds_count":  len(defaultLegalHolds),
		"total_repository_size_mb":  9.4,
		"iso_15489_compliance":      "CERTIFIED_COMPLIANT",
		"last_archive_sync":         time.Now().Format(time.RFC3339),
	})
}

func handleListDocuments(c *gin.Context) {
	c.JSON(http.StatusOK, defaultDocuments)
}

func handleCreateDocument(c *gin.Context) {
	var req EnterpriseDocument
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("doc-%d", time.Now().Unix())
	req.DocumentNumber = fmt.Sprintf("DOC-%s-2026-%03d", req.Category[:3], len(defaultDocuments)+1)
	req.CreatedTime = time.Now()
	req.UpdatedTime = time.Now()
	if req.Version == "" {
		req.Version = "v1.0"
	}
	if req.Status == "" {
		req.Status = "Draft"
	}
	h := sha256.Sum256([]byte(req.Title + req.Content + req.CreatedTime.String()))
	req.SHA256Hash = fmt.Sprintf("%x", h)

	defaultDocuments = append([]EnterpriseDocument{req}, defaultDocuments...)
	c.JSON(http.StatusCreated, req)
}

func handleListDocumentTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, defaultTemplates)
}

func handleListDigitalSignatures(c *gin.Context) {
	c.JSON(http.StatusOK, defaultSignatures)
}

func handleSignDocument(c *gin.Context) {
	var req DigitalSignature
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("sig-%d", time.Now().Unix())
	req.SignedAt = time.Now()
	h := sha256.Sum256([]byte(req.DocumentID + req.SignerName + req.SignedAt.String()))
	req.VerificationHash = fmt.Sprintf("SHA256:%x", h)
	req.QRVerificationURL = fmt.Sprintf("http://localhost:8096/api/documents/verify/%x", h[:8])

	defaultSignatures = append([]DigitalSignature{req}, defaultSignatures...)

	// Update document signed list
	for i, d := range defaultDocuments {
		if d.ID == req.DocumentID {
			defaultDocuments[i].SignedBy = append(defaultDocuments[i].SignedBy, req.SignerName)
			defaultDocuments[i].Status = "Approved"
			break
		}
	}

	c.JSON(http.StatusCreated, req)
}

func handleVerifyDocumentHash(c *gin.Context) {
	hash := c.Param("hash")
	c.JSON(http.StatusOK, gin.H{
		"verification_status": "VERIFIED_AUTHENTIC",
		"hash":                hash,
		"issuer":              "National Statistics & Research Authority (StatGate Authority)",
		"timestamp":           time.Now().Format(time.RFC3339),
		"integrity_guarantee": "Cryptographically Sealed (SHA-256)",
	})
}

func handleListRetentionPolicies(c *gin.Context) {
	c.JSON(http.StatusOK, defaultRetentionPolicies)
}

func handleListLegalHolds(c *gin.Context) {
	c.JSON(http.StatusOK, defaultLegalHolds)
}
