package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type researchAssistantRequest struct {
	Task   string `json:"task"`
	Prompt string `json:"prompt"`
}

var researchAssistantTasks = map[string]string{
	"proposal":     "proposal planning",
	"literature":   "literature synthesis",
	"gap-analysis": "research gap analysis",
}

func dbPostResearchAssistant(c *gin.Context) {
	var request researchAssistantRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assistant request"})
		return
	}
	task := strings.ToLower(strings.TrimSpace(request.Task))
	if task == "" {
		task = "proposal"
	}
	if _, ok := researchAssistantTasks[task]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task must be proposal, literature, or gap-analysis"})
		return
	}
	prompt := strings.TrimSpace(request.Prompt)
	if len(prompt) > 4000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt must be 4000 characters or fewer"})
		return
	}
	if prompt == "" {
		prompt = fmt.Sprintf("Help me with %s for this study. Identify practical next steps and clearly call out missing evidence.", researchAssistantTasks[task])
	}

	evidence, err := loadResearchQualityEvidence(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "research study not found"})
		return
	}
	groundedContext := fmt.Sprintf(
		"Study name: %s\nDescription: %s\nPrincipal investigator: %s\nStage: %s\nProgress: %.0f%%\nProposal records: %d (approved: %d, incomplete: %d)\nEthics records: %d (approved: %d, pending: %d)\nDatasets: %d (incomplete metadata: %d)\nPublications: %d (published: %d, missing DOI: %d)\nJournal submissions: %d (accepted: %d)\nArchives: %d (verified: %d)\nOverdue tasks: %d",
		evidence.Name, trimAssistantContext(evidence.Description, 2000), evidence.PrincipalInvestigator, evidence.Stage, evidence.Progress,
		evidence.ProposalCount, evidence.ApprovedProposalCount, evidence.IncompleteProposalCount,
		evidence.EthicsCount, evidence.ApprovedEthicsCount, evidence.PendingEthicsCount,
		evidence.DatasetCount, evidence.IncompleteDatasetCount,
		evidence.PublicationCount, evidence.PublishedPublicationCount, evidence.PublicationMissingDOI,
		evidence.SubmissionCount, evidence.AcceptedSubmissionCount,
		evidence.ArchiveCount, evidence.VerifiedArchiveCount, evidence.OverdueTaskCount,
	)
	output, model, providerErr := generateResearchAssistant(c.Request.Context(), task, prompt, groundedContext)
	if providerErr != nil {
		c.JSON(researchAssistantProviderErrorCode(providerErr), gin.H{"error": providerErr.Error(), "task": task})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mode":           "provider-backed-research-assistant",
		"provider":       "chat-completions",
		"model":          model,
		"task":           task,
		"researchId":     c.Param("id"),
		"output":         output,
		"groundedAt":     time.Now().UTC(),
		"groundedFields": []string{"study metadata", "proposal evidence", "ethics evidence", "dataset evidence", "publication evidence", "submission evidence", "preservation evidence", "overdue task evidence"},
		"disclaimer":     "Provider-generated research assistance is advisory. Verify sources, methods, ethics, and conclusions with qualified human reviewers before use.",
	})
}

func trimAssistantContext(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
