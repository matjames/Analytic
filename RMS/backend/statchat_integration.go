package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	statgatechat "github.com/matjames/statgate-lib/statchat"
)

func researchDiscussionClient() *statgatechat.Client {
	return statgatechat.NewClient(os.Getenv("STATCHAT_API_URL"), os.Getenv("STATGATE_INTERNAL_API_KEY"))
}

func researchDiscussionRef(researchID string) string {
	return "obj:rms:research:" + strings.TrimSpace(researchID)
}

func ensureResearchDiscussion(c *gin.Context, researchID, researchName string) (statgatechat.Conversation, error) {
	return researchDiscussionClient().EnsureObjectConversation(
		c.Request.Context(), c.GetHeader("Authorization"), workspaceIDContext(c),
		statgatechat.ObjectConversationRequest{
			ObjectRef: researchDiscussionRef(researchID),
			Name:      researchName + " discussion",
			Metadata: map[string]any{
				"module": "rms", "entity": "research", "entityId": researchID, "workspaceId": workspaceIDContext(c),
			},
		},
	)
}

func loadResearchDiscussion(c *gin.Context, research ResearchProject) ([]ChatMessage, error) {
	conversation, err := ensureResearchDiscussion(c, research.ID, research.Name)
	if err != nil {
		return nil, err
	}
	messages, err := researchDiscussionClient().Messages(c.Request.Context(), c.GetHeader("Authorization"), workspaceIDContext(c), conversation.ID)
	if err != nil {
		return nil, err
	}
	result := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		channel := strings.TrimSpace(message.ChannelID)
		if channel == "" {
			channel = "general"
		}
		result = append(result, ChatMessage{ID: message.ID, ResearchID: research.ID, Sender: message.Sender, Channel: channel, Message: message.Text, CreatedTime: message.CreatedAt})
	}
	return result, nil
}

func sendResearchDiscussionMessage(c *gin.Context, researchID, channel, text string) (ChatMessage, error) {
	var research ResearchProject
	err := DB.QueryRow(`SELECT id, name FROM rms.research_projects WHERE id=$1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)`, researchID, workspaceIDContext(c)).Scan(&research.ID, &research.Name)
	if err != nil {
		return ChatMessage{}, err
	}
	conversation, err := ensureResearchDiscussion(c, research.ID, research.Name)
	if err != nil {
		return ChatMessage{}, err
	}
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = "general"
	}
	message, err := researchDiscussionClient().SendMessage(c.Request.Context(), c.GetHeader("Authorization"), workspaceIDContext(c), conversation.ID, channel, strings.TrimSpace(text))
	if err != nil {
		return ChatMessage{}, err
	}
	return ChatMessage{ID: message.ID, ResearchID: research.ID, Sender: message.Sender, Channel: channel, Message: message.Text, CreatedTime: message.CreatedAt}, nil
}

func writeResearchDiscussionError(c *gin.Context, err error) {
	status := http.StatusBadGateway
	if upstream, ok := err.(*statgatechat.StatusError); ok && upstream.StatusCode >= 400 && upstream.StatusCode < 500 {
		status = upstream.StatusCode
	}
	c.JSON(status, gin.H{"error": fmt.Sprintf("research discussion unavailable: %v", err)})
}
