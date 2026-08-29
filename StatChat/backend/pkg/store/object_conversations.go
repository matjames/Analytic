package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"statchat/pkg/model"

	"github.com/google/uuid"
)

func GetObjectConversation(tenantID string, objectRef string) (model.Conversation, error) {
	tenantID = normalizedTenantID(tenantID)
	objectRef = strings.TrimSpace(objectRef)
	var conversation model.Conversation
	var conversationType string
	var channelID sql.NullString
	var memberIDsJSON []byte
	var metadataJSON []byte
	err := db.QueryRowContext(context.Background(), `
SELECT id, tenant_id, object_ref, name, type, channel_id, member_ids,
       COALESCE(category, ''), metadata
FROM conversations
WHERE tenant_id = $1 AND object_ref = $2`, tenantID, objectRef).Scan(
		&conversation.ID,
		&conversation.TenantID,
		&conversation.ObjectRef,
		&conversation.Name,
		&conversationType,
		&channelID,
		&memberIDsJSON,
		&conversation.Category,
		&metadataJSON,
	)
	if err != nil {
		return conversation, err
	}
	conversation.Type = model.ConversationType(conversationType)
	conversation.ChannelID = channelID.String
	if err := json.Unmarshal(memberIDsJSON, &conversation.MemberIDs); err != nil {
		return conversation, err
	}
	if err := json.Unmarshal(metadataJSON, &conversation.Metadata); err != nil {
		return conversation, err
	}
	return conversation, nil
}

func CreateOrGetObjectConversation(tenantID string, objectRef string, name string, creatorID string, memberIDs []string, metadata map[string]any) (model.Conversation, error) {
	tenantID = normalizedTenantID(tenantID)
	objectRef = strings.TrimSpace(objectRef)
	if existing, err := GetObjectConversation(tenantID, objectRef); err == nil {
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.Conversation{}, err
	}

	uniqueMembers := make([]string, 0, len(memberIDs)+1)
	seen := map[string]bool{}
	for _, memberID := range append(memberIDs, creatorID) {
		memberID = strings.TrimSpace(memberID)
		if memberID != "" && !seen[memberID] {
			seen[memberID] = true
			uniqueMembers = append(uniqueMembers, memberID)
		}
	}
	memberIDsJSON, err := json.Marshal(uniqueMembers)
	if err != nil {
		return model.Conversation{}, err
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return model.Conversation{}, err
	}

	conversationID := "object-" + uuid.NewString()
	_, err = db.ExecContext(context.Background(), `
INSERT INTO conversations (id, tenant_id, object_ref, name, type, member_ids, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tenant_id, object_ref) WHERE object_ref IS NOT NULL DO NOTHING`,
		conversationID, tenantID, objectRef, name, model.ConversationTypeGroup, memberIDsJSON, metadataJSON)
	if err != nil {
		return model.Conversation{}, err
	}
	if err := EnsureConversationOwner(conversationID, creatorID); err != nil {
		return model.Conversation{}, err
	}
	return GetObjectConversation(tenantID, objectRef)
}
