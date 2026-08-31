package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		return origin == "" || allowedOrigin(origin, r)
	},
}

// clientSubprotocols returns the Sec-WebSocket-Protocol values a client offered
// so the server can echo one back. Browsers drop a WebSocket whose request
// included a subprotocol unless the server negotiates (echoes) it.
func clientSubprotocols(r *http.Request) []string {
	var out []string
	for _, p := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// authRequired is permanently true. StatChat operates in AUTHENTICATION
// REQUIRED mode in every environment (SG-SEC-2026-08). The demo identity
// fallback has been removed.
func authRequired() bool {
	return true
}

func requestCurrentUser(r *http.Request) (model.User, error) {
	userID := requestUserID(r)
	if strings.TrimSpace(userID) == "" {
		// Authentication is mandatory; no anonymous/demo identities exist.
		return model.User{}, fmt.Errorf("missing user identity")
	}
	user, err := store.GetUserByID(userID)
	if err != nil {
		if identity, ok := r.Context().Value(requestIdentityKey).(model.User); ok {
			if syncErr := store.UpsertTrustedUser(identity); syncErr != nil {
				return model.User{}, syncErr
			}
			return identity, nil
		}
		return model.User{}, err
	}
	return user, nil
}

type sendMessageRequest struct {
	TenantID        string   `json:"tenantId,omitempty"`
	ConversationID  string   `json:"conversationId"`
	ChannelID       string   `json:"channelId,omitempty"`
	ParentMessageID string   `json:"parentMessageId,omitempty"`
	ThreadRootID    string   `json:"threadRootId,omitempty"`
	Sender          string   `json:"sender"`
	Text            string   `json:"text"`
	MentionUserIDs  []string `json:"mentionUserIds,omitempty"`
	MentionAll      bool     `json:"mentionAll,omitempty"`
}

type editMessageRequest struct {
	Text string `json:"text"`
}

type updateProfileRequest struct {
	Name      string `json:"name"`
	About     string `json:"about"`
	AvatarURL string `json:"avatarUrl"`
}

type contextKey string

const requestUserIDKey contextKey = "requestUserID"
const requestIdentityKey contextKey = "requestIdentity"
const requestWorkspaceIDKey contextKey = "requestWorkspaceID"

func sharedJWTSecret() string {
	if secret := strings.TrimSpace(os.Getenv("STATGATE_REGISTRY_JWT_SECRET")); secret != "" {
		return secret
	}
	return strings.TrimSpace(os.Getenv("STATCHAT_JWT_SECRET"))
}

func claimString(claims jwt.MapClaims, names ...string) string {
	for _, name := range names {
		if value, ok := claims[name]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func sharedIdentityFromClaims(claims jwt.MapClaims, userID string) model.User {
	name := claimString(claims, "name", "preferred_username", "username")
	if name == "" {
		name = "StatGate member " + userID
	}
	role := claimString(claims, "role", "userRole")
	roles := []string{}
	if role != "" {
		roles = append(roles, role)
	}
	return model.User{
		ID:             userID,
		Name:           name,
		Email:          claimString(claims, "email"),
		OrganizationID: claimString(claims, "tenantId", "tenant_id", "organizationId", "organization_id"),
		Roles:          roles,
		AvatarURL:      claimString(claims, "avatarUrl", "avatar_url"),
		Presence:       "online",
	}
}

func requestUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(requestUserIDKey).(string); ok && strings.TrimSpace(userID) != "" {
		return userID
	}
	return ""
}

func requestUserName(r *http.Request) string {
	if user, err := requestCurrentUser(r); err == nil && strings.TrimSpace(user.Name) != "" {
		return user.Name
	}
	return "StatChat User"
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func requestTenantID(r *http.Request) string {
	if identity, ok := r.Context().Value(requestIdentityKey).(model.User); ok {
		return resolveTenantID(identity.OrganizationID)
	}
	return "default"
}

func requestWorkspaceID(r *http.Request) string {
	if workspaceID, ok := r.Context().Value(requestWorkspaceIDKey).(string); ok {
		return strings.TrimSpace(workspaceID)
	}
	return ""
}

func requireConversationAccess(w http.ResponseWriter, r *http.Request, conversationID string) bool {
	allowed, err := store.CanAccessConversation(conversationID, requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify conversation access")
		return false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "you do not have access to this conversation")
		return false
	}
	return true
}

func requireMessageAccess(w http.ResponseWriter, r *http.Request, messageID string) (model.Message, bool) {
	message, err := store.GetMessageByID(messageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "message not found")
		return model.Message{}, false
	}
	if !requireConversationAccess(w, r, message.ConversationID) {
		return model.Message{}, false
	}
	return message, true
}

func RegisterRoutes(router *mux.Router) {
	router.Use(corsMiddleware)
	router.Use(authMiddleware)
	router.Use(rateLimitMiddleware)
	router.HandleFunc("/health", healthHandler).Methods(http.MethodGet)
	router.HandleFunc("/readyz", readinessHandler).Methods(http.MethodGet)
	router.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	router.HandleFunc("/users", allUsersHandler).Methods(http.MethodGet)
	router.HandleFunc("/users/me", currentUserHandler).Methods(http.MethodGet)
	router.HandleFunc("/users/me/profile", updateProfileHandler).Methods(http.MethodPut)
	router.HandleFunc("/users/me/settings", getUserSettingsHandler).Methods(http.MethodGet)
	router.HandleFunc("/users/me/settings", updateUserSettingsHandler).Methods(http.MethodPut)
	router.HandleFunc("/conversations/dm", createDMHandler).Methods(http.MethodPost)
	router.HandleFunc("/conversations/group", createGroupHandler).Methods(http.MethodPost)
	router.HandleFunc("/groups/templates", groupTemplatesHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/posts", postsHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/posts", createPostHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/post-media", uploadPostMediaHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/posts/{id}/like", togglePostLikeHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/posts/{id}/share", sharePostHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/posts/{id}/comments", postCommentsHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/posts/{id}/comments", addPostCommentHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/connections", connectionsHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/connections", createConnectionHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/connections", removeConnectionHandler).Methods(http.MethodDelete)
	router.HandleFunc("/collaboration/connection-requests", connectionRequestsHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/connection-requests/{id}/{action}", respondConnectionRequestHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/opportunities", opportunitiesHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/jobs", jobsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/communities", communitiesHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/communities/{id}/join", joinCommunityHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/communities/{id}/leave", leaveCommunityHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/communities/{id}/members", communityMembersHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/communities/{id}/members/{userId}", removeCommunityMemberHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/communities/{id}/owner", transferCommunityOwnerHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/communities/{id}/topics", communityTopicsHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/communities/{communityId}/topics/{topicId}", deleteCommunityTopicHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/communities/{communityId}/topics/{topicId}/replies", communityRepliesHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/communities/{communityId}/topics/{topicId}/replies/{replyId}", deleteCommunityReplyHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/documents", collaborationDocumentsHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/documents/{id}", collaborationDocumentHandler).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
	router.HandleFunc("/v1/chat/documents/{id}/members", collaborationDocumentMembersHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/documents/{id}/members/{userId}", removeCollaborationDocumentMemberHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/documents/{id}/revisions", collaborationDocumentRevisionsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/whiteboards", collaborationWhiteboardsHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/whiteboards/{id}", collaborationWhiteboardHandler).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
	router.HandleFunc("/v1/chat/whiteboards/{id}/members", collaborationWhiteboardMembersHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/whiteboards/{id}/members/{userId}", removeCollaborationWhiteboardMemberHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/whiteboards/{id}/revisions", collaborationWhiteboardRevisionsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/translation/languages", translationLanguagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/translation/translate", translateTextHandler).Methods(http.MethodPost)
	router.HandleFunc("/meetings", meetingsHandler).Methods(http.MethodGet)
	router.HandleFunc("/meetings", createMeetingHandler).Methods(http.MethodPost)
	router.HandleFunc("/meetings/rooms", meetingRoomsHandler).Methods(http.MethodGet)
	router.HandleFunc("/meetings/recordings", meetingRecordingsHandler).Methods(http.MethodGet)
	router.HandleFunc("/wellness/posts", wellnessPostsHandler).Methods(http.MethodGet)
	router.HandleFunc("/wellness/posts", createWellnessPostHandler).Methods(http.MethodPost)
	router.HandleFunc("/wellness/posts/{id}/comments", wellnessPostCommentsHandler).Methods(http.MethodGet)
	router.HandleFunc("/wellness/posts/{id}/comments", addWellnessCommentHandler).Methods(http.MethodPost)
	router.HandleFunc("/wellness/posts/{id}/like", toggleWellnessLikeHandler).Methods(http.MethodPost)
	router.HandleFunc("/wellness/posts/{id}/bookmark", toggleWellnessBookmarkHandler).Methods(http.MethodPost)
	router.HandleFunc("/wellness/posts/{id}/share", shareWellnessPostHandler).Methods(http.MethodPost)
	router.HandleFunc("/knowledge/experts", knowledgeExpertsHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/articles", knowledgeArticlesHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/ideas", knowledgeIdeasHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/posts", knowledgePostsHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/posts", createKnowledgePostHandler).Methods(http.MethodPost)
	router.HandleFunc("/knowledge/ideas/{id}/upvote", upvoteKnowledgeIdeaHandler).Methods(http.MethodPost)
	router.HandleFunc("/knowledge/experts/{id}/follow", followKnowledgeExpertHandler).Methods(http.MethodPost)
	router.HandleFunc("/channels", channelsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/channels", channelsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/channels", createChannelHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/channels/{id}", channelDetailHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/channels/{id}/join", joinChannelHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/channels/{id}/leave", leaveChannelHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/channels/{id}/archive", archiveChannelHandler).Methods(http.MethodPost)
	router.HandleFunc("/conversations", conversationsHandler).Methods(http.MethodGet)
	router.HandleFunc("/messages", messagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations", conversationsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations/object", objectConversationHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/messages", createMessageHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/attachments", uploadAttachmentHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/conversations/{id}/messages", conversationMessagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations/{id}/export", exportConversationHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations/{id}/members", conversationMembersHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/conversations/{id}/members/{userId}", removeConversationMemberHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/messages/{id}", editMessageHandler).Methods(http.MethodPut)
	router.HandleFunc("/v1/chat/messages/{id}", deleteMessageHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/messages/{id}/forward", forwardMessageHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/messages/{id}/saved", saveMessageHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/messages/{id}/saved", unsaveMessageHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/saved", savedMessagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/scheduled", scheduledMessagesHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/scheduled/{id}", cancelScheduledMessageHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/compliance/retention", retentionPolicyHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/compliance/retention", updateRetentionPolicyHandler).Methods(http.MethodPut)
	router.HandleFunc("/v1/compliance/retention/enforce", enforceRetentionHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/compliance/legal-holds", legalHoldsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/compliance/legal-holds", createLegalHoldHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/compliance/legal-holds/{id}/release", releaseLegalHoldHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/compliance/audit", complianceAuditHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/media", mediaCatalogHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/media/send", sendCatalogMediaHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/location", sendLocationHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/messages/{id}/reactions", addReactionHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/messages/{id}/reactions", removeReactionHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/messages/{id}/read", markMessageReadHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/conversations/{id}/pinned", pinnedMessagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations/{id}/pinned", pinMessageHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/conversations/{id}/pinned", unpinMessageHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/tasks", tasksHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/tasks", createTaskHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/tasks/{id}/status", updateTaskStatusHandler).Methods(http.MethodPut)
	router.HandleFunc("/v1/tasks/{id}", updateTaskHandler).Methods(http.MethodPut)
	router.HandleFunc("/v1/tasks/{id}", deleteTaskHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/tasks", tasksHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/tasks", createTaskHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/tasks/{id}", updateTaskHandler).Methods(http.MethodPut)
	router.HandleFunc("/v1/chat/tasks/{id}", deleteTaskHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/calendar", calendarEventsHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/calendar/{id}", calendarEventHandler).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
	router.HandleFunc("/v1/chat/polls", pollsHandler).Methods(http.MethodGet, http.MethodPost)
	router.HandleFunc("/v1/chat/polls/{id}/vote", votePollHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/assistant", assistantHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/notifications", notificationsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/notifications/{id}/read", markNotificationReadHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/notifications/read-all", markAllNotificationsReadHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/presence", presenceHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/presence", updatePresenceHandler).Methods(http.MethodPut)
	router.HandleFunc("/search", searchHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/search", searchHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations/{id}/favourite", toggleFavouriteHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/conversations/{id}/favourite", favouriteStatusHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/favourites", favouriteIdsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations/{id}/mute", muteConversationHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/chat/conversations/{id}/mute", unmuteConversationHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/chat/muted", mutedIdsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/chat/conversations/{id}/messages", clearConversationHandler).Methods(http.MethodDelete)
	router.HandleFunc("/v1/calls", createCallSessionHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls", listCallSessionsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/calls/{id}", getCallSessionHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/calls/{id}/join", joinCallSessionHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls/{id}/leave", leaveCallSessionHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls/{id}/end", endCallSessionHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls/{id}/participants", getCallParticipantsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/calls/{id}/participants/{userId}/remove", removeCallParticipantHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls/{id}/participants/{userId}/role", updateCallParticipantRoleHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls/{id}/quality", createCallQualitySampleHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls/{id}/quality", callQualitySamplesHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/calls/{id}/recordings", uploadCallRecordingHandler).Methods(http.MethodPost)
	router.HandleFunc("/v1/calls/{id}/recordings", listCallRecordingsHandler).Methods(http.MethodGet)
	router.HandleFunc("/v1/meetings/{id}/session", getMeetingSessionHandler).Methods(http.MethodGet)
	router.HandleFunc("/ws", wsHandler)
	router.HandleFunc("/ws/chat", wsHandler)
	router.HandleFunc("/uploads/{name}", uploadedFileHandler).Methods(http.MethodGet, http.MethodHead)
	router.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(optionsHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":       "ok",
		"service":      "statchat",
		"ready":        store.IsReady(),
		"authRequired": authRequired(),
	})
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	if !store.IsReady() {
		writeError(w, http.StatusServiceUnavailable, "database is not ready")
		return
	}
	writeJSON(w, map[string]interface{}{"status": "ok", "service": "statchat", "ready": true})
}

func currentUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := requestUserID(r)
	user, err := store.GetUserByID(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load current user")
		return
	}
	writeJSON(w, user)
}

func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	userID := requestUserID(r)
	if err := store.UpdateUserProfile(userID, req.Name, req.About, req.AvatarURL); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	user, err := store.GetUserByID(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load updated user")
		return
	}
	writeJSON(w, user)
}

func getUserSettingsHandler(w http.ResponseWriter, r *http.Request) {
	userID := requestUserID(r)
	settings, err := store.GetUserSettings(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}
	if settings.UserID == "" {
		settings.UserID = userID
		settings.Theme = "light"
		settings.AccentColor = "#0b5fff"
		settings.FontSize = "medium"
		settings.EnterToSend = true
		settings.Language = "English"
		settings.LastSeen = "everyone"
		settings.ProfilePhoto = "contacts"
		settings.ReadReceipts = true
		settings.TypingIndicator = true
		settings.VoiceNotes = true
		settings.ReadByDefault = false
		settings.AutoDownload = "never"
		settings.NotifMessages = true
		settings.NotifGroups = true
		settings.NotifMentions = true
		settings.NotifMeetings = true
		settings.NotifSound = true
		settings.NotifPreview = false
		settings.DownloadImages = "wifi"
		settings.DownloadVideos = "wifi"
		settings.DownloadDocuments = "wifi"
	}
	writeJSON(w, settings)
}

func updateUserSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var s model.UserSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	s.UserID = requestUserID(r)
	if err := store.UpsertUserSettings(s); err != nil {
		log.Printf("failed to save settings for %s: %v", s.UserID, err)
		writeError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}
	writeJSON(w, s)
}

func allUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := registryDirectory(r.Context(), strings.TrimSpace(r.Header.Get("Authorization")), requestTenantID(r))
	if err != nil {
		// A short Registry outage must not make active conversations unusable.
		// Return the collaboration cache while recording the degraded directory.
		log.Printf("registry directory unavailable: %v", err)
		users, err = store.GetUsersByOrganization(requestTenantID(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load users")
			return
		}
	}
	writeJSON(w, users)
}

type createGroupRequest struct {
	GroupID   string   `json:"groupId"`
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
}

type createDMRequest struct {
	TargetUserID string `json:"targetUserId"`
	TargetName   string `json:"targetName"`
}

func createGroupHandler(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.GroupID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "groupId and name are required")
		return
	}
	if len(req.MemberIDs) == 0 {
		req.MemberIDs = []string{requestUserID(r)}
	}
	requesterID := requestUserID(r)
	if !containsString(req.MemberIDs, requesterID) {
		req.MemberIDs = append(req.MemberIDs, requesterID)
	}
	conv, err := store.CreateGroupConversation(requestTenantID(r), req.GroupID, req.Name, req.MemberIDs, requesterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create group")
		return
	}
	writeJSON(w, conv)
}

func createDMHandler(w http.ResponseWriter, r *http.Request) {
	var req createDMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.TargetUserID == "" {
		writeError(w, http.StatusBadRequest, "targetUserId is required")
		return
	}
	name := req.TargetName
	if name == "" {
		name = req.TargetUserID
	}
	conv, err := store.CreateDirectConversation(requestTenantID(r), requestUserID(r), req.TargetUserID, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}
	writeJSON(w, conv)
}

func groupTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, store.GetGroupTemplates())
}

func postsHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := store.GetPosts(requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load posts")
		return
	}
	writeJSON(w, posts)
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	var req model.Post
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	req.Type = strings.ToLower(strings.TrimSpace(req.Type))
	if req.Type == "" {
		req.Type = "text"
	}
	if req.Type != "text" && req.Type != "photo" && req.Type != "video" && req.Type != "article" {
		writeError(w, http.StatusBadRequest, "type must be text, photo, video, or article")
		return
	}
	if req.Type == "article" && strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "article title is required")
		return
	}
	if (req.Type == "photo" || req.Type == "video") && strings.TrimSpace(req.MediaURL) == "" {
		writeError(w, http.StatusBadRequest, "mediaUrl is required for photo and video posts")
		return
	}
	if req.MediaURL != "" {
		parsedMediaURL, parseErr := url.Parse(strings.TrimSpace(req.MediaURL))
		if parseErr != nil || (parsedMediaURL.Scheme != "" && parsedMediaURL.Scheme != "https" && parsedMediaURL.Scheme != "http") || (parsedMediaURL.Scheme == "" && !strings.HasPrefix(parsedMediaURL.Path, "/uploads/")) {
			writeError(w, http.StatusBadRequest, "mediaUrl must be an http(s) URL or an uploaded media path")
			return
		}
		req.MediaURL = strings.TrimSpace(req.MediaURL)
	}
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	req.TenantID = requestTenantID(r)
	req.AuthorID = currentUser.ID
	req.Author = currentUser.Name
	req.Org = currentUser.OrganizationID
	if len(currentUser.Roles) > 0 {
		req.Role = currentUser.Roles[0]
	}
	req.Likes, req.Comments, req.Shares = 0, 0, 0
	post, err := store.CreatePost(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create post")
		return
	}
	writeJSON(w, post)
}

func connectionsHandler(w http.ResponseWriter, r *http.Request) {
	connections, err := store.GetConnections(requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load connections")
		return
	}
	writeJSON(w, connections)
}

func createConnectionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetUserID string `json:"targetUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.TargetUserID == "" {
		writeError(w, http.StatusBadRequest, "targetUserId is required")
		return
	}
	if req.TargetUserID == requestUserID(r) {
		writeError(w, http.StatusBadRequest, "cannot connect to yourself")
		return
	}
	conn, err := store.CreateConnection(requestUserID(r), req.TargetUserID, requestTenantID(r))
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "target user not found in this tenant")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create connection")
		return
	}
	writeJSON(w, conn)
}

func removeConnectionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TargetUserID string `json:"targetUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if err := store.RemoveConnection(requestUserID(r), req.TargetUserID, requestTenantID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove connection")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func opportunitiesHandler(w http.ResponseWriter, r *http.Request) {
	opportunities, err := store.GetOpportunities()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load opportunities")
		return
	}
	writeJSON(w, opportunities)
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs, err := store.GetJobs()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load jobs")
		return
	}
	writeJSON(w, jobs)
}

func meetingsHandler(w http.ResponseWriter, r *http.Request) {
	meetings, err := store.GetMeetings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load meetings")
		return
	}
	writeJSON(w, meetings)
}

func createMeetingHandler(w http.ResponseWriter, r *http.Request) {
	var req model.Meeting
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	meeting, err := store.CreateMeeting(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create meeting")
		return
	}
	writeJSON(w, meeting)
}

type meetingRoomResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Status   string `json:"status"`
	URL      string `json:"url"`
}

func meetingRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms, err := store.GetMeetingRooms()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load meeting rooms")
		return
	}
	responses := make([]meetingRoomResponse, 0, len(rooms))
	for _, room := range rooms {
		responses = append(responses, meetingRoomResponse{
			ID:       room.ID,
			Name:     room.Name,
			Capacity: room.Capacity,
			Status:   room.Status,
			URL:      room.URL,
		})
	}
	writeJSON(w, responses)
}

func meetingRecordingsHandler(w http.ResponseWriter, r *http.Request) {
	recordings, err := store.GetMeetingRecordings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load recordings")
		return
	}
	writeJSON(w, recordings)
}

func wellnessPostsHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := store.GetWellnessPosts(requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load wellness posts")
		return
	}
	writeJSON(w, posts)
}

func createWellnessPostHandler(w http.ResponseWriter, r *http.Request) {
	var req model.WellnessPost
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	req.TenantID = requestTenantID(r)
	req.AuthorID = currentUser.ID
	req.Author = currentUser.Name
	req.Handle = "@" + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(currentUser.Name), " ", ""))
	if req.Avatar == "" {
		nameRunes := []rune(strings.TrimSpace(currentUser.Name))
		if len(nameRunes) > 0 {
			req.Avatar = strings.ToUpper(string(nameRunes[0]))
		} else {
			req.Avatar = "U"
		}
	}
	if req.Category == "" || req.Category == "For You" {
		req.Category = "Mental Health"
	}
	post, err := store.CreateWellnessPost(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create wellness post")
		return
	}
	writeJSON(w, post)
}

func knowledgeExpertsHandler(w http.ResponseWriter, r *http.Request) {
	experts, err := store.GetKnowledgeExperts(requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge experts")
		return
	}
	writeJSON(w, experts)
}

func knowledgeArticlesHandler(w http.ResponseWriter, r *http.Request) {
	articles, err := store.GetKnowledgeArticles(requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge articles")
		return
	}
	writeJSON(w, articles)
}

func knowledgeIdeasHandler(w http.ResponseWriter, r *http.Request) {
	ideas, err := store.GetKnowledgeIdeas(requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge ideas")
		return
	}
	writeJSON(w, ideas)
}

func knowledgePostsHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := store.GetKnowledgePosts(requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge posts")
		return
	}
	writeJSON(w, posts)
}

func createKnowledgePostHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title    string `json:"title"`
		Category string `json:"category"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Category = strings.TrimSpace(req.Category)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, "title and content are required")
		return
	}
	if req.Category == "" {
		req.Category = "General"
	}
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	role := ""
	if len(currentUser.Roles) > 0 {
		role = currentUser.Roles[0]
	}
	post, err := store.CreateKnowledgePost(model.KnowledgePost{
		TenantID:  requestTenantID(r),
		Title:     req.Title,
		Author:    currentUser.Name,
		AuthorID:  currentUser.ID,
		Role:      role,
		Org:       currentUser.OrganizationID,
		Category:  req.Category,
		Content:   req.Content,
		CreatedBy: currentUser.ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create knowledge post")
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, post)
}

func channelsHandler(w http.ResponseWriter, r *http.Request) {
	channels, err := store.GetChannelsForUser(requestTenantID(r), requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load channels")
		return
	}
	writeJSON(w, channels)
}

func conversationsHandler(w http.ResponseWriter, r *http.Request) {
	conversations, err := store.GetConversations(requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load conversations")
		return
	}
	writeJSON(w, conversations)
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	conversationID := strings.TrimSpace(r.URL.Query().Get("conversationId"))
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "conversationId is required")
		return
	}
	if !requireConversationAccess(w, r, conversationID) {
		return
	}
	messages, err := store.GetMessagesForTenant(conversationID, requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load messages")
		return
	}
	writeJSON(w, messages)
}

func uploadAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	sniffBuffer := make([]byte, 512)
	n, err := io.ReadFull(file, sniffBuffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		writeError(w, http.StatusBadRequest, "failed to inspect upload")
		return
	}
	contentType := http.DetectContentType(sniffBuffer[:n])
	// WebM is a container and can be detected as video/webm even when
	// MediaRecorder created an audio-only voice note. Preserve a valid declared
	// audio type so it is rendered and served as playable audio.
	declaredType := strings.ToLower(strings.TrimSpace(strings.Split(header.Header.Get("Content-Type"), ";")[0]))
	if strings.HasPrefix(declaredType, "audio/") && isAllowedAttachmentType(declaredType) {
		contentType = declaredType
	}
	if !isAllowedAttachmentType(contentType) {
		writeError(w, http.StatusBadRequest, "file type is not allowed")
		return
	}

	conversationID := strings.TrimSpace(r.FormValue("conversationId"))
	if conversationID == "" {
		conversationID = "general"
	}
	if !requireConversationAccess(w, r, conversationID) {
		return
	}
	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		switch {
		case strings.HasPrefix(contentType, "audio/"):
			text = "Voice note"
		case strings.HasPrefix(contentType, "image/"):
			text = "Photo"
		case strings.HasPrefix(contentType, "video/"):
			text = "Video"
		default:
			text = "Shared media"
		}
	}
	sender := requestUserName(r)
	tenantID := requestTenantID(r)

	dir := ensureUploadDir()
	extension := strings.ToLower(filepath.Ext(header.Filename))
	safeName := fmt.Sprintf("%s%s", uuid.NewString(), extension)
	destinationPath := filepath.Join(dir, safeName)
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create upload")
		return
	}
	defer destinationFile.Close()

	if _, err := io.Copy(destinationFile, io.MultiReader(bytes.NewReader(sniffBuffer[:n]), file)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save upload")
		return
	}

	attachment := model.MessageAttachment{
		ID:        uuid.NewString(),
		FileName:  header.Filename,
		FileType:  contentType,
		URL:       fmt.Sprintf("/uploads/%s", safeName),
		MimeType:  contentType,
		CreatedAt: time.Now().UTC(),
	}
	message := model.Message{
		ID:             uuid.NewString(),
		TenantID:       tenantID,
		ConversationID: conversationID,
		SenderID:       requestUserID(r),
		Sender:         sender,
		Text:           text,
		CreatedAt:      time.Now().UTC(),
		Status:         "active",
		DeliveryStatus: "sent",
		Attachments:    []model.MessageAttachment{attachment},
	}
	attachment.MessageID = message.ID

	if err := store.StoreMessage(message); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}
	if err := store.StoreMessageAttachment(attachment); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save attachment")
		return
	}
	store.BroadcastMessage(message)
	_ = store.NotifyConversationMembers(message.ConversationID, requestUserID(r), sender, message.Text, fmt.Sprintf("/chat/%s", message.ConversationID), nil)
	writeJSON(w, message)
}

func uploadedFileHandler(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(mux.Vars(r)["name"])
	if name == "." || name == "" {
		http.NotFound(w, r)
		return
	}
	publicURL := "/uploads/" + name
	if attachment, err := store.GetMessageAttachmentByURL(publicURL); err == nil && attachment.MimeType != "" {
		w.Header().Set("Content-Type", attachment.MimeType)
	}
	http.ServeFile(w, r, filepath.Join(ensureUploadDir(), name))
}

func createMessageHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	var req sendMessageRequest
	var envelope struct {
		Event    string             `json:"event"`
		TenantID string             `json:"tenantId,omitempty"`
		Payload  sendMessageRequest `json:"payload"`
	}

	if err := json.Unmarshal(body, &envelope); err == nil && (envelope.Event != "" || envelope.Payload.ConversationID != "" || envelope.Payload.Sender != "" || envelope.Payload.Text != "") {
		req = envelope.Payload
		if req.TenantID == "" {
			req.TenantID = envelope.TenantID
		}
	} else {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}
	}

	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	if req.ConversationID == "" {
		req.ConversationID = "general"
	}
	if !requireConversationAccess(w, r, req.ConversationID) {
		return
	}
	identity, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing user identity")
		return
	}
	mentionUserIDs, err := resolveMessageMentions(req.ConversationID, requestUserID(r), requestTenantID(r), req.MentionUserIDs, req.MentionAll, identity)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sender := requestUserName(r)
	message := model.Message{
		ID:              uuid.NewString(),
		TenantID:        requestTenantID(r),
		ConversationID:  req.ConversationID,
		ChannelID:       req.ChannelID,
		ParentMessageID: req.ParentMessageID,
		ThreadRootID:    req.ThreadRootID,
		SenderID:        requestUserID(r),
		Sender:          sender,
		Text:            req.Text,
		CreatedAt:       time.Now().UTC(),
		Status:          "active",
		DeliveryStatus:  "sent",
		MentionUserIDs:  mentionUserIDs,
		MentionAll:      req.MentionAll,
	}
	if err := store.StoreMessageWithMentions(message); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}
	store.BroadcastMessage(message)
	store.NotifyConversationMembers(message.ConversationID, requestUserID(r), sender, message.Text, fmt.Sprintf("/chat/%s", message.ConversationID), message.MentionUserIDs)
	writeJSON(w, message)
}

func conversationMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	if !requireConversationAccess(w, r, conversationID) {
		return
	}
	messages, err := store.GetMessagesForTenant(conversationID, requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load conversation messages")
		return
	}
	writeJSON(w, messages)
}

func forwardMessageHandler(w http.ResponseWriter, r *http.Request) {
	source, ok := requireMessageAccess(w, r, mux.Vars(r)["id"])
	if !ok {
		return
	}
	if source.Status == "deleted" {
		writeError(w, http.StatusConflict, "deleted messages cannot be forwarded")
		return
	}
	var req struct {
		TargetConversationID string `json:"targetConversationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.TargetConversationID) == "" {
		writeError(w, http.StatusBadRequest, "targetConversationId is required")
		return
	}
	req.TargetConversationID = strings.TrimSpace(req.TargetConversationID)
	if !requireConversationAccess(w, r, req.TargetConversationID) {
		return
	}
	message, err := store.ForwardMessage(source, req.TargetConversationID, requestUserID(r), requestUserName(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to forward message")
		return
	}
	store.BroadcastMessage(message)
	store.NotifyConversationMembers(message.ConversationID, requestUserID(r), requestUserName(r), message.Text, fmt.Sprintf("/chat/%s", message.ConversationID), nil)
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, message)
}

func editMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]
	var req editMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	message, accessible := requireMessageAccess(w, r, messageID)
	if !accessible {
		return
	}
	allowed, err := store.CanModifyMessage(requestUserID(r), message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check message permissions")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "you do not have permission to edit this message")
		return
	}
	message.Text = req.Text
	if err := store.UpdateMessage(message); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update message")
		return
	}
	store.BroadcastEnvelope(message.TenantID, message.ConversationID, "message.update", map[string]interface{}{
		"id": message.ID, "conversationId": message.ConversationID, "text": message.Text, "updatedAt": message.UpdatedAt,
	})
	writeJSON(w, message)
}

func deleteMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]
	message, accessible := requireMessageAccess(w, r, messageID)
	if !accessible {
		return
	}
	allowed, err := store.CanModifyMessage(requestUserID(r), message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check message permissions")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "you do not have permission to delete this message")
		return
	}
	if err := store.SoftDeleteMessage(messageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete message")
		return
	}
	message.Status = "deleted"
	message.DeletedAt = time.Now().UTC()
	store.BroadcastEnvelope(message.TenantID, message.ConversationID, "message.delete", map[string]interface{}{
		"id": message.ID, "conversationId": message.ConversationID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		log.Printf("websocket auth failed: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// The frontend carries the JWT as "Bearer.<token>" in Sec-WebSocket-Protocol
	// (browsers cannot set WS headers). Browsers require the server to echo a
	// chosen subprotocol; if the response omits the header the connection is
	// rejected and the UI shows a reconnect loop. Negotiate by echoing any
	// subprotocol the client offered. Do this on a per-request copy so the
	// package-level upgrader stays shared.
	u := upgrader
	u.Subprotocols = clientSubprotocols(r)
	conn, err := u.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	client := store.NewClient(conn, "general", currentUser.ID)
	store.RegisterClient(client)
	defer func() {
		store.UnregisterClient(client)
		conn.Close()
	}()

	for {
		var payload map[string]interface{}
		if err := conn.ReadJSON(&payload); err != nil {
			log.Printf("read error: %v", err)
			return
		}

		action, _ := payload["action"].(string)
		trustedTenantID := resolveTenantID(currentUser.OrganizationID)
		switch action {
		case "join-conversation":
			conversationID, _ := payload["conversationId"].(string)
			if conversationID == "" {
				conversationID = "general"
			}
			allowed, accessErr := store.CanAccessConversation(conversationID, currentUser.ID, trustedTenantID)
			if accessErr != nil || !allowed {
				_ = conn.WriteJSON(map[string]string{"event": "error", "error": "conversation access denied"})
				continue
			}
			client.SetConversation(routeConversationKey(trustedTenantID, conversationID))
		case "typing":
			conversationID, _ := payload["conversationId"].(string)
			if conversationID == "" {
				continue
			}
			allowed, accessErr := store.CanAccessConversation(conversationID, currentUser.ID, trustedTenantID)
			if accessErr != nil || !allowed {
				continue
			}
			store.BroadcastEnvelope(trustedTenantID, conversationID, "typing", map[string]interface{}{
				"conversationId": conversationID,
				"userId":         currentUser.ID,
				"userName":       currentUser.Name,
			})
		case "stop-typing":
			conversationID, _ := payload["conversationId"].(string)
			if conversationID == "" {
				continue
			}
			allowed, accessErr := store.CanAccessConversation(conversationID, currentUser.ID, trustedTenantID)
			if accessErr != nil || !allowed {
				continue
			}
			store.BroadcastEnvelope(trustedTenantID, conversationID, "stop-typing", map[string]interface{}{
				"conversationId": conversationID,
				"userId":         currentUser.ID,
				"userName":       currentUser.Name,
			})
		case "presence-update":
			status, _ := payload["status"].(string)
			if status == "" {
				status = "online"
			}
			if err := store.UpdatePresence(currentUser.ID, status); err != nil {
				log.Printf("failed to update presence: %v", err)
			}
			members, err := store.GetConversationMembers("general")
			if err == nil {
				for _, memberID := range members {
					store.BroadcastEnvelope("", "general", "presence-update", map[string]interface{}{
						"userId":   currentUser.ID,
						"userName": currentUser.Name,
						"status":   status,
						"memberId": memberID,
					})
				}
			}
		case "send-message":
			message, err := buildMessageFromPayload(payload, currentUser)
			if err != nil {
				log.Printf("invalid websocket chat payload: %v", err)
				continue
			}
			allowed, accessErr := store.CanAccessConversation(message.ConversationID, currentUser.ID, trustedTenantID)
			if accessErr != nil || !allowed {
				_ = conn.WriteJSON(map[string]string{"event": "error", "error": "conversation access denied"})
				continue
			}
			message.TenantID = trustedTenantID
			mentionUserIDs, mentionErr := resolveMessageMentions(message.ConversationID, currentUser.ID, trustedTenantID, message.MentionUserIDs, message.MentionAll, currentUser)
			if mentionErr != nil {
				_ = conn.WriteJSON(map[string]string{"event": "error", "error": mentionErr.Error()})
				continue
			}
			message.MentionUserIDs = mentionUserIDs
			if err := store.StoreMessageWithMentions(message); err != nil {
				log.Printf("failed to store websocket message: %v", err)
				continue
			}
			store.BroadcastMessage(message)
			_ = store.NotifyConversationMembers(message.ConversationID, currentUser.ID, currentUser.Name, message.Text, fmt.Sprintf("/chat/%s", message.ConversationID), message.MentionUserIDs)
		case "join-call":
			sessionID, _ := payload["sessionId"].(string)
			if sessionID == "" {
				continue
			}
			session, sessionErr := store.GetCallSession(sessionID)
			allowed, accessErr := canAccessCallSession(session, currentUser.ID, trustedTenantID)
			if sessionErr != nil || accessErr != nil || !allowed || session.Status != model.CallStatusLive {
				_ = conn.WriteJSON(map[string]string{"event": "error", "error": "call access denied"})
				continue
			}
			store.SetConnCallIdentity(conn, sessionID, currentUser.ID)
			if _, err := store.JoinCallSession(sessionID, currentUser.ID, currentUser.Name); err != nil {
				_ = conn.WriteJSON(map[string]string{"event": "error", "error": "failed to join call"})
				continue
			}
			store.BroadcastCallState(sessionID, "call-participant-joined", map[string]string{
				"userId": currentUser.ID, "userName": currentUser.Name,
			})
		case "leave-call":
			sessionID, _ := payload["sessionId"].(string)
			if sessionID != "" {
				session, sessionErr := store.GetCallSession(sessionID)
				allowed, accessErr := canAccessCallSession(session, currentUser.ID, trustedTenantID)
				if sessionErr == nil && accessErr == nil && allowed {
					store.LeaveCallSession(sessionID, currentUser.ID)
					store.BroadcastCallState(sessionID, "call-participant-left", map[string]string{"userId": currentUser.ID})
				}
			}
			store.ClearConnCallIdentity(conn)
		case "signal":
			sessionID, _ := payload["sessionId"].(string)
			signalType, _ := payload["type"].(string)
			toUser, _ := payload["to"].(string)
			signalPayload, _ := payload["payload"].(string)
			if sessionID == "" || !isAllowedCallSignalType(signalType) {
				_ = conn.WriteJSON(map[string]string{"event": "error", "error": "invalid call signal"})
				continue
			}
			session, sessionErr := store.GetCallSession(sessionID)
			allowed, accessErr := canAccessCallSession(session, currentUser.ID, trustedTenantID)
			active, activeErr := store.IsActiveCallParticipant(sessionID, currentUser.ID)
			if sessionErr != nil || accessErr != nil || activeErr != nil || !allowed || !active || session.Status != model.CallStatusLive {
				_ = conn.WriteJSON(map[string]string{"event": "error", "error": "call signaling access denied"})
				continue
			}
			if toUser != "" {
				targetActive, targetErr := store.IsActiveCallParticipant(sessionID, toUser)
				if targetErr != nil || !targetActive {
					_ = conn.WriteJSON(map[string]string{"event": "error", "error": "call signal target is not active"})
					continue
				}
			}
			if signalType == "mute-requested" {
				role, roleErr := store.GetActiveCallParticipantRole(sessionID, currentUser.ID)
				if roleErr != nil || (role != "host" && role != "moderator") {
					_ = conn.WriteJSON(map[string]string{"event": "error", "error": "only hosts and moderators can request mute"})
					continue
				}
				if toUser == "" {
					_ = conn.WriteJSON(map[string]string{"event": "error", "error": "mute request requires a target"})
					continue
				}
			}
			store.SignalRelay(model.CallSignal{
				Type:      signalType,
				SessionID: sessionID,
				From:      currentUser.ID,
				FromName:  currentUser.Name,
				To:        toUser,
				Payload:   signalPayload,
			})
		}
	}
}

func currentUserFallbackName() string {
	if user, err := store.GetCurrentUser(); err == nil {
		return user.Name
	}
	return "StatChat User"
}

func buildMessageFromPayload(payload map[string]interface{}, currentUser model.User) (model.Message, error) {
	var text string
	var conversationID string
	var channelID string
	var tenantID string
	var topLevelTenant string
	var parentMessageID string
	var threadRootID string
	var mentionUserIDs []string
	var mentionAll bool

	if topTenant, ok := payload["tenantId"].(string); ok {
		topLevelTenant = topTenant
	}

	if nested, ok := payload["payload"].(map[string]interface{}); ok {
		if tenant, ok := nested["tenantId"].(string); ok && tenant != "" {
			tenantID = tenant
		} else {
			tenantID = topLevelTenant
		}
		payload = nested
	}

	text, _ = payload["text"].(string)
	conversationID, _ = payload["conversationId"].(string)
	channelID, _ = payload["channelId"].(string)
	parentMessageID, _ = payload["parentMessageId"].(string)
	threadRootID, _ = payload["threadRootId"].(string)
	mentionAll, _ = payload["mentionAll"].(bool)
	if rawMentions, ok := payload["mentionUserIds"].([]interface{}); ok {
		for _, rawMention := range rawMentions {
			if userID, ok := rawMention.(string); ok {
				mentionUserIDs = append(mentionUserIDs, userID)
			}
		}
	}
	if tenantID == "" {
		tenantID, _ = payload["tenantId"].(string)
	}

	if strings.TrimSpace(text) == "" {
		return model.Message{}, fmt.Errorf("text is required")
	}
	if strings.TrimSpace(conversationID) == "" {
		conversationID = "general"
	}
	return model.Message{
		ID:              uuid.NewString(),
		TenantID:        resolveTenantID(tenantID),
		ConversationID:  conversationID,
		ChannelID:       channelID,
		ParentMessageID: parentMessageID,
		ThreadRootID:    threadRootID,
		SenderID:        currentUser.ID,
		Sender:          currentUser.Name,
		Text:            text,
		CreatedAt:       time.Now().UTC(),
		Status:          "active",
		DeliveryStatus:  "sent",
		MentionUserIDs:  mentionUserIDs,
		MentionAll:      mentionAll,
	}, nil
}

func ensureUploadDir() string {
	dir := strings.TrimSpace(os.Getenv("STATCHAT_UPLOAD_DIR"))
	if dir == "" {
		dir = filepath.Join(".", "uploads")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("failed to ensure upload directory %s: %v", dir, err)
	}
	return dir
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/health" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/uploads") {
			next.ServeHTTP(w, r)
			return
		}
		if identity, workspaceID, ok := internalServiceIdentity(r); ok {
			ctx := context.WithValue(r.Context(), requestUserIDKey, identity.ID)
			ctx = context.WithValue(ctx, requestIdentityKey, identity)
			ctx = context.WithValue(ctx, requestWorkspaceIDKey, workspaceID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		secret := sharedJWTSecret()
		if secret == "" || secret == "statchat-dev-secret" {
			writeError(w, http.StatusInternalServerError, "jwt secret must be configured when auth is enabled")
			return
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
		if tokenString == "" && (r.URL.Path == "/ws" || r.URL.Path == "/ws/chat") {
			// WebSocket clients cannot set HTTP headers; the token is carried in
			// the Sec-WebSocket-Protocol field as "Bearer.<token>". Long-lived
			// JWTs are never placed in query strings (SG-SEC-2026-08).
			protocols := strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",")
			for _, p := range protocols {
				if strings.HasPrefix(strings.TrimSpace(p), "Bearer.") {
					tokenString = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p), "Bearer."))
					break
				}
			}
			if tokenString == "" {
				// A short-lived one-time handshake ticket (expires <= 5 min) may
				// be supplied as ?ticket= for browser clients. Query-string JWTs
				// with longer lifetimes are rejected below.
				ticket := strings.TrimSpace(r.URL.Query().Get("ticket"))
				if ticket != "" {
					tokenString = ticket
				}
			}
		}
		if tokenString == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(sharedJWTSecret()), nil
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		// Query-string tickets must be short-lived (<6 minutes) so a token can
		// never be harvested from logs/URL bars and replayed later.
		if r.URL.Path == "/ws" || r.URL.Path == "/ws/chat" {
			if ticket := strings.TrimSpace(r.URL.Query().Get("ticket")); ticket != "" {
				if exp, ok := claims["exp"].(float64); ok && int64(exp) > time.Now().Add(6*time.Minute).Unix() {
					writeError(w, http.StatusUnauthorized, "web socket ticket must be short-lived")
					return
				}
			}
		}

		userID := strings.TrimSpace(fmt.Sprint(claims["sub"]))
		if userID == "" {
			userID = strings.TrimSpace(fmt.Sprint(claims["user_id"]))
		}
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "missing user identity in token")
			return
		}
		workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
		if len(workspaceID) > 128 || strings.ContainsAny(workspaceID, " /\\\t\r\n") {
			writeError(w, http.StatusBadRequest, "invalid workspace context")
			return
		}
		membershipAuth := authHeader
		if membershipAuth == "" {
			membershipAuth = "Bearer " + tokenString
		}
		if workspaceID != "" && !workspaceMember(r, workspaceID, membershipAuth) {
			return
		}

		identity := sharedIdentityFromClaims(claims, userID)
		if store.IsReady() {
			if err := store.UpsertTrustedUser(identity); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to synchronize authenticated user")
				return
			}
		}
		ctx := context.WithValue(r.Context(), requestUserIDKey, userID)
		ctx = context.WithValue(ctx, requestIdentityKey, identity)
		ctx = context.WithValue(ctx, requestWorkspaceIDKey, workspaceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func internalObjectConversationIdentity(r *http.Request) (model.User, string, bool) {
	if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/conversations/object" {
		return model.User{}, "", false
	}
	expected := strings.TrimSpace(os.Getenv("STATGATE_INTERNAL_API_KEY"))
	supplied := strings.TrimSpace(r.Header.Get("X-Internal-API-Key"))
	userID := strings.TrimSpace(r.Header.Get("X-StatGate-User-ID"))
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
	if expected == "" || supplied == "" || !hmac.Equal([]byte(expected), []byte(supplied)) || userID == "" || tenantID == "" {
		return model.User{}, "", false
	}
	if len(workspaceID) > 128 || strings.ContainsAny(workspaceID, " /\\\t\r\n") {
		return model.User{}, "", false
	}
	return model.User{ID: userID, Name: userID, OrganizationID: tenantID, Presence: "online"}, workspaceID, true
}

// internalServiceIdentity allows narrowly scoped service-to-service writes for
// canonical object conversations and their messages. The caller must provide
// the shared key, an explicit service identity, and a tenant scope.
func internalServiceIdentity(r *http.Request) (model.User, string, bool) {
	if r.URL.Path == "/v1/chat/conversations/object" {
		return internalObjectConversationIdentity(r)
	}
	if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/messages" {
		return model.User{}, "", false
	}
	expected := strings.TrimSpace(os.Getenv("STATGATE_INTERNAL_API_KEY"))
	supplied := strings.TrimSpace(r.Header.Get("X-Internal-API-Key"))
	userID := strings.TrimSpace(r.Header.Get("X-StatGate-User-ID"))
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	workspaceID := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
	if expected == "" || supplied == "" || !hmac.Equal([]byte(expected), []byte(supplied)) || userID == "" || tenantID == "" {
		return model.User{}, "", false
	}
	if len(workspaceID) > 128 || strings.ContainsAny(workspaceID, " /\\\t\r\n") {
		return model.User{}, "", false
	}
	return model.User{ID: userID, Name: userID, OrganizationID: tenantID, Presence: "online"}, workspaceID, true
}

func workspaceMember(r *http.Request, workspaceID, authHeader string) bool {
	base := strings.TrimRight(os.Getenv("STATGATE_ENTERPRISE_API_URL"), "/")
	if base == "" {
		base = "http://localhost:8096/api"
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, base+"/workspaces/"+url.PathEscape(workspaceID), nil)
	if err != nil {
		return false
	}
	request.Header.Set("Authorization", authHeader)
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusForbidden {
		return false
	}
	return response.StatusCode >= 200 && response.StatusCode < 300
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode failed: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	writeJSON(w, map[string]string{"error": message})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORSHeaders(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

var allowedAttachmentTypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"text/plain":      true,
	"audio/mpeg":      true,
	"audio/mp4":       true,
	"audio/x-m4a":     true,
	"audio/aac":       true,
	"audio/flac":      true,
	"audio/wav":       true,
	"audio/ogg":       true,
	"audio/webm":      true,
	"video/mp4":       true,
	"video/quicktime": true,
	"video/x-msvideo": true,
	"video/webm":      true,
}

func isAllowedAttachmentType(contentType string) bool {
	if contentType == "" {
		return false
	}
	if allowedAttachmentTypes[contentType] {
		return true
	}
	if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "audio/") || strings.HasPrefix(contentType, "video/") {
		return allowedAttachmentTypes[contentType]
	}
	return false
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return
	}
	if allowedOrigin(origin, r) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

func allowedOrigin(origin string, r *http.Request) bool {
	allowedOrigins := getAllowedOrigins()
	if len(allowedOrigins) > 0 {
		for _, candidate := range allowedOrigins {
			if strings.EqualFold(candidate, origin) {
				return true
			}
		}
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Host == r.Host {
		return true
	}
	if strings.HasPrefix(parsed.Host, "localhost") || strings.HasPrefix(parsed.Host, "127.0.0.1") || strings.HasPrefix(parsed.Host, "[::1]") {
		return true
	}
	return false
}

func getAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("STATCHAT_CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	allowed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			allowed = append(allowed, part)
		}
	}
	return allowed
}

const (
	defaultRateLimitRequests = 120
	defaultRateLimitWindow   = time.Minute
)

type rateLimitEntry struct {
	count     int
	lastReset time.Time
}

var rateLimitStore = struct {
	sync.Mutex
	entries map[string]*rateLimitEntry
}{
	entries: map[string]*rateLimitEntry{},
}

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		ip := clientIP(r)
		if ip == "" {
			ip = "unknown"
		}
		if !allowRate(ip) {
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowRate(ip string) bool {
	rateLimitStore.Lock()
	defer rateLimitStore.Unlock()

	entry, ok := rateLimitStore.entries[ip]
	if !ok {
		rateLimitStore.entries[ip] = &rateLimitEntry{count: 1, lastReset: time.Now()}
		return true
	}

	if time.Since(entry.lastReset) > defaultRateLimitWindow {
		entry.count = 1
		entry.lastReset = time.Now()
		return true
	}

	if entry.count >= defaultRateLimitRequests {
		return false
	}

	entry.count++
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
