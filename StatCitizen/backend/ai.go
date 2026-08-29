package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── Governed Public AI Assistant ─────────────────────────────────────────────
//
// Rules:
// 1. Evidence-grounded: Only answers from approved Public Publications and Consultations.
// 2. Source-aware: Every claim cites an authoritative publication or consultation ID.
// 3. Privacy-aware: Zero access to internal institutional workflows, confidential tickets, or unredacted reports.
// 4. Insufficient-evidence behavior: Explicitly declines when no matching public evidence exists.
// 5. Rate-limited and audited.

type AIQueryRequest struct {
	Query     string `json:"query" binding:"required"`
	District  string `json:"district,omitempty"`
	Category  string `json:"category,omitempty"`
	Language  string `json:"language,omitempty"`
}

type AICitation struct {
	Title          string `json:"title"`
	Publisher      string `json:"publisher"`
	Category       string `json:"category"`
	PublishedAt    string `json:"published_at"`
	SourceObjectID string `json:"source_object_id,omitempty"`
	CanonicalID    string `json:"canonical_id"`
}

type AIQueryResponse struct {
	Query                string       `json:"query"`
	Answer               string       `json:"answer"`
	EvidenceGrounded     bool         `json:"evidence_grounded"`
	Confidence           float64      `json:"confidence"`
	Citations            []AICitation `json:"citations"`
	Disclaimer           string       `json:"disclaimer"`
	InsufficientEvidence bool         `json:"insufficient_evidence"`
	Timestamp            string       `json:"timestamp"`
}

// handleAIQuery processes natural language queries from citizens against governed public knowledge.
func handleAIQuery(c *gin.Context, cfg *Config) {
	var req AIQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid_request", Message: "Query string is required"})
		return
	}

	queryLower := strings.ToLower(strings.TrimSpace(req.Query))
	tenantID := getTenantID(c, cfg)
	corrID := getCorrelationID(c)

	if dbPool == nil {
		c.JSON(http.StatusOK, AIQueryResponse{
			Query:                req.Query,
			Answer:               "The public knowledge service is currently in maintenance mode. Please consult official institutional channels directly.",
			EvidenceGrounded:     false,
			Confidence:           0.0,
			Citations:            []AICitation{},
			Disclaimer:           "StatCitizen AI responses are grounded strictly in governed institutional publications.",
			InsufficientEvidence: true,
			Timestamp:            time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Extract query search tokens
	keywords := strings.Fields(queryLower)
	var matchedPublications []PublicPublication

	// Search approved public publications
	rows, err := dbPool.QueryContext(ctx,
		`SELECT id, canonical_id, category, title, summary, body, publisher, source_object_id, published_at
		 FROM public_publications
		 WHERE tenant_id=$1 AND status='published'
		 ORDER BY published_at DESC LIMIT 50`, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p PublicPublication
			var body, srcObjID, pubAt *string
			if err := rows.Scan(&p.ID, &p.CanonicalID, &p.Category, &p.Title, &p.Summary, &body, &p.Publisher, &srcObjID, &pubAt); err == nil {
				if body != nil {
					p.Body = *body
				}
				if srcObjID != nil {
					p.SourceObjectID = *srcObjID
				}
				if pubAt != nil {
					p.PublishedAt = *pubAt
				}

				// Check relevance score
				score := 0
				textToMatch := strings.ToLower(p.Title + " " + p.Summary + " " + p.Body + " " + p.Category)
				for _, kw := range keywords {
					if len(kw) > 2 && strings.Contains(textToMatch, kw) {
						score++
					}
				}
				if score > 0 {
					matchedPublications = append(matchedPublications, p)
				}
			}
		}
	}

	// Check active consultations
	cRows, cErr := dbPool.QueryContext(ctx,
		`SELECT id, canonical_id, title, description, category, published_at
		 FROM consultations
		 WHERE tenant_id=$1 AND status='published' AND close_date > NOW()
		 ORDER BY open_date DESC LIMIT 20`, tenantID)
	var matchedConsultations []Consultation
	if cErr == nil {
		defer cRows.Close()
		for cRows.Next() {
			var cons Consultation
			var cat, pubAt *string
			if err := cRows.Scan(&cons.ID, &cons.CanonicalID, &cons.Title, &cons.Description, &cat, &pubAt); err == nil {
				if cat != nil {
					cons.Category = *cat
				}
				if pubAt != nil {
					cons.PublishedAt = *pubAt
				}
				textToMatch := strings.ToLower(cons.Title + " " + cons.Description)
				for _, kw := range keywords {
					if len(kw) > 2 && strings.Contains(textToMatch, kw) {
						matchedConsultations = append(matchedConsultations, cons)
						break
					}
				}
			}
		}
	}

	var citations []AICitation
	var answerBuilder strings.Builder

	if len(matchedPublications) == 0 && len(matchedConsultations) == 0 {
		// Insufficient evidence behavior — never hallucinate or invent institutional information
		c.JSON(http.StatusOK, AIQueryResponse{
			Query:                req.Query,
			Answer:               "I could not find verified public records or official publications answering this specific inquiry in the public catalog. For authoritative verification, please check active public consultations or submit a structured inquiry through the platform.",
			EvidenceGrounded:     false,
			Confidence:           0.0,
			Citations:            []AICitation{},
			Disclaimer:           "StatCitizen AI operates under strict sovereign intelligence constraints: it only answers from verified public publications and never fabricates institutional data.",
			InsufficientEvidence: true,
			Timestamp:            time.Now().UTC().Format(time.RFC3339),
		})

		recordCitizenAudit(c, cfg, "citizen", "ai.query_no_evidence", "ai_assistant", "query", map[string]interface{}{
			"correlation_id": corrID,
			"keywords_count": len(keywords),
		})
		return
	}

	// Build grounded answer
	if len(matchedPublications) > 0 {
		topPub := matchedPublications[0]
		answerBuilder.WriteString(fmt.Sprintf("According to official publication '%s' published by %s: %s ", topPub.Title, topPub.Publisher, topPub.Summary))
		if topPub.Body != "" && len(topPub.Body) < 300 {
			answerBuilder.WriteString(topPub.Body)
		}
		for _, pub := range matchedPublications {
			citations = append(citations, AICitation{
				Title:          pub.Title,
				Publisher:      pub.Publisher,
				Category:       pub.Category,
				PublishedAt:    pub.PublishedAt,
				SourceObjectID: pub.SourceObjectID,
				CanonicalID:    pub.CanonicalID,
			})
			if len(citations) >= 3 {
				break
			}
		}
	}

	if len(matchedConsultations) > 0 {
		if answerBuilder.Len() > 0 {
			answerBuilder.WriteString("\n\nRelated Active Public Consultations:\n")
		}
		for _, cons := range matchedConsultations {
			answerBuilder.WriteString(fmt.Sprintf("• %s: %s\n", cons.Title, cons.Description))
			citations = append(citations, AICitation{
				Title:       cons.Title,
				Publisher:   "StatGate Public Consultation Gate",
				Category:    "consultation",
				PublishedAt: cons.PublishedAt,
				CanonicalID: cons.CanonicalID,
			})
			if len(citations) >= 5 {
				break
			}
		}
	}

	resp := AIQueryResponse{
		Query:                req.Query,
		Answer:               strings.TrimSpace(answerBuilder.String()),
		EvidenceGrounded:     true,
		Confidence:           0.92,
		Citations:            citations,
		Disclaimer:           "This response is grounded strictly in governed public records. For formal legal or policy interpretations, reference the cited source documents.",
		InsufficientEvidence: false,
		Timestamp:            time.Now().UTC().Format(time.RFC3339),
	}

	recordCitizenAudit(c, cfg, "citizen", "ai.query_answered", "ai_assistant", "query", map[string]interface{}{
		"correlation_id":  corrID,
		"citations_count": len(citations),
	})

	c.JSON(http.StatusOK, resp)
}
