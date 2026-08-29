package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	statgatechat "github.com/matjames/statgate-lib/statchat"
)

func projectDiscussionClient() *statgatechat.Client {
	return statgatechat.NewClient(os.Getenv("STATCHAT_API_URL"), os.Getenv("STATGATE_INTERNAL_API_KEY"))
}

func projectDiscussionRef(projectID string) string {
	return "obj:pms:project:" + strings.TrimSpace(projectID)
}

func ensureProjectDiscussion(c *gin.Context, projectID, projectName string) (statgatechat.Conversation, error) {
	return projectDiscussionClient().EnsureObjectConversation(
		c.Request.Context(),
		c.GetHeader("Authorization"),
		workspaceIDContext(c),
		statgatechat.ObjectConversationRequest{
			ObjectRef: projectDiscussionRef(projectID),
			Name:      projectName + " discussion",
			Metadata: map[string]any{
				"module": "pms", "entity": "project", "entityId": projectID, "workspaceId": workspaceIDContext(c),
			},
		},
	)
}

func loadProjectDiscussion(c *gin.Context, project Project) ([]ChatMessage, error) {
	conversation, err := ensureProjectDiscussion(c, project.ID, project.Name)
	if err != nil {
		return nil, err
	}
	messages, err := projectDiscussionClient().Messages(c.Request.Context(), c.GetHeader("Authorization"), workspaceIDContext(c), conversation.ID)
	if err != nil {
		return nil, err
	}
	result := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		channel := strings.TrimSpace(message.ChannelID)
		if channel == "" {
			channel = "general"
		}
		result = append(result, ChatMessage{
			ID: message.ID, ProjectID: project.ID, Channel: channel, Sender: message.Sender,
			Role: "Member", Message: message.Text, Timestamp: message.CreatedAt,
		})
	}
	return result, nil
}

func sendProjectDiscussionMessage(c *gin.Context, input ChatMessage) (ChatMessage, error) {
	var project Project
	err := DB.QueryRow(`SELECT id, name FROM pms.projects WHERE id=$1 AND (workspace_id = NULLIF($2, '') OR NULLIF($2, '') IS NULL)`, input.ProjectID, workspaceIDContext(c)).Scan(&project.ID, &project.Name)
	if err != nil {
		return ChatMessage{}, err
	}
	conversation, err := ensureProjectDiscussion(c, project.ID, project.Name)
	if err != nil {
		return ChatMessage{}, err
	}
	channel := strings.TrimSpace(input.Channel)
	if channel == "" {
		channel = "general"
	}
	message, err := projectDiscussionClient().SendMessage(c.Request.Context(), c.GetHeader("Authorization"), workspaceIDContext(c), conversation.ID, channel, strings.TrimSpace(input.Message))
	if err != nil {
		return ChatMessage{}, err
	}
	return ChatMessage{ID: message.ID, ProjectID: project.ID, Channel: channel, Sender: message.Sender, Role: "Member", Message: message.Text, Timestamp: message.CreatedAt}, nil
}

func writeProjectDiscussionError(c *gin.Context, err error) {
	status := http.StatusBadGateway
	if upstream, ok := err.(*statgatechat.StatusError); ok && upstream.StatusCode >= 400 && upstream.StatusCode < 500 {
		status = upstream.StatusCode
	}
	c.JSON(status, gin.H{"error": fmt.Sprintf("project discussion unavailable: %v", err)})
}
