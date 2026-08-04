package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type sendMessageRequest struct {
	TenantID        string `json:"tenantId,omitempty"`
	ConversationID  string `json:"conversationId"`
	ChannelID       string `json:"channelId,omitempty"`
	ParentMessageID string `json:"parentMessageId,omitempty"`
	ThreadRootID    string `json:"threadRootId,omitempty"`
	Sender          string `json:"sender"`
	Text            string `json:"text"`
}

type editMessageRequest struct {
	Text string `json:"text"`
}

type updateProfileRequest struct {
	Name      string `json:"name"`
	About     string `json:"about"`
	AvatarURL string `json:"avatarUrl"`
}

func RegisterRoutes(router *mux.Router) {
	router.Use(corsMiddleware)
	router.Use(authMiddleware)
	router.HandleFunc("/health", healthHandler).Methods(http.MethodGet)
	router.HandleFunc("/readyz", healthHandler).Methods(http.MethodGet)
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
	router.HandleFunc("/collaboration/posts/{id}/like", togglePostLikeHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/posts/{id}/comments", postCommentsHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/posts/{id}/comments", addPostCommentHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/connections", connectionsHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/connections", createConnectionHandler).Methods(http.MethodPost)
	router.HandleFunc("/collaboration/connections", removeConnectionHandler).Methods(http.MethodDelete)
	router.HandleFunc("/collaboration/opportunities", opportunitiesHandler).Methods(http.MethodGet)
	router.HandleFunc("/collaboration/jobs", jobsHandler).Methods(http.MethodGet)
	router.HandleFunc("/meetings", meetingsHandler).Methods(http.MethodGet)
	router.HandleFunc("/meetings", createMeetingHandler).Methods(http.MethodPost)
	router.HandleFunc("/meetings/rooms", meetingRoomsHandler).Methods(http.MethodGet)
	router.HandleFunc("/meetings/recordings", meetingRecordingsHandler).Methods(http.MethodGet)
	router.HandleFunc("/wellness/posts", wellnessPostsHandler).Methods(http.MethodGet)
	router.HandleFunc("/wellness/posts", createWellnessPostHandler).Methods(http.MethodPost)
	router.HandleFunc("/knowledge/experts", knowledgeExpertsHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/articles", knowledgeArticlesHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/ideas", knowledgeIdeasHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/posts", knowledgePostsHandler).Methods(http.MethodGet)
	router.HandleFunc("/knowledge/posts", createKnowledgePostHandler).Methods(http.MethodPost)
	router.HandleFunc("/channels", channelsHandler).Methods(http.MethodGet)
	router.HandleFunc("/conversations", conversationsHandler).Methods(http.MethodGet)
	router.HandleFunc("/messages", messagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/chat/conversations", conversationsHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/chat/messages", createMessageHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/chat/attachments", uploadAttachmentHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/chat/conversations/{id}/messages", conversationMessagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/chat/messages/{id}", editMessageHandler).Methods(http.MethodPut)
	router.HandleFunc("/api/v1/chat/messages/{id}", deleteMessageHandler).Methods(http.MethodDelete)
	router.HandleFunc("/api/v1/chat/messages/{id}/reactions", addReactionHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/chat/messages/{id}/reactions", removeReactionHandler).Methods(http.MethodDelete)
	router.HandleFunc("/api/v1/chat/messages/{id}/read", markMessageReadHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/chat/conversations/{id}/pinned", pinnedMessagesHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/chat/conversations/{id}/pinned", pinMessageHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/chat/conversations/{id}/pinned", unpinMessageHandler).Methods(http.MethodDelete)
	router.HandleFunc("/api/v1/tasks", tasksHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/tasks", createTaskHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/tasks/{id}/status", updateTaskStatusHandler).Methods(http.MethodPut)
	router.HandleFunc("/api/v1/notifications", notificationsHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/notifications/{id}/read", markNotificationReadHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/notifications/read-all", markAllNotificationsReadHandler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/presence", presenceHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/presence", updatePresenceHandler).Methods(http.MethodPut)
	router.HandleFunc("/ws", wsHandler)
	router.HandleFunc("/ws/chat", wsHandler)
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(ensureUploadDir()))))
	router.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(optionsHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":       "ok",
		"service":      "statchat",
		"authRequired": strings.EqualFold(os.Getenv("STATCHAT_AUTH_REQUIRED"), "true"),
	})
}

func currentUserHandler(w http.ResponseWriter, r *http.Request) {
	user, err := store.GetCurrentUser()
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
	if err := store.UpdateUserProfile("user-001", req.Name, req.About, req.AvatarURL); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	user, err := store.GetCurrentUser()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load updated user")
		return
	}
	writeJSON(w, user)
}

func getUserSettingsHandler(w http.ResponseWriter, r *http.Request) {
	settings, err := store.GetUserSettings("user-001")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}
	if settings.UserID == "" {
		settings.UserID = "user-001"
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
	s.UserID = "user-001"
	if err := store.UpsertUserSettings(s); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save settings")
		return
	}
	writeJSON(w, s)
}

func allUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := store.GetAllUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
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
		req.MemberIDs = []string{"user-001"}
	}
	conv, err := store.CreateGroupConversation(req.GroupID, req.Name, req.MemberIDs)
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
	conv, err := store.CreateDirectConversation("user-001", req.TargetUserID, name)
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
	posts, err := store.GetPosts()
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
	if req.Author == "" || req.Text == "" {
		writeError(w, http.StatusBadRequest, "author and text are required")
		return
	}
	post, err := store.CreatePost(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create post")
		return
	}
	writeJSON(w, post)
}

func connectionsHandler(w http.ResponseWriter, r *http.Request) {
	connections, err := store.GetConnections("user-001")
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
	conn, err := store.CreateConnection("user-001", req.TargetUserID)
	if err != nil {
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
	if err := store.RemoveConnection("user-001", req.TargetUserID); err != nil {
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

func meetingRoomsHandler(w http.ResponseWriter, r *http.Request) {
	rooms, err := store.GetMeetingRooms()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load meeting rooms")
		return
	}
	writeJSON(w, rooms)
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
	posts, err := store.GetWellnessPosts()
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
	if req.Author == "" || req.Text == "" {
		writeError(w, http.StatusBadRequest, "author and text are required")
		return
	}
	post, err := store.CreateWellnessPost(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create wellness post")
		return
	}
	writeJSON(w, post)
}

func knowledgeExpertsHandler(w http.ResponseWriter, r *http.Request) {
	experts, err := store.GetKnowledgeExperts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge experts")
		return
	}
	writeJSON(w, experts)
}

func knowledgeArticlesHandler(w http.ResponseWriter, r *http.Request) {
	articles, err := store.GetKnowledgeArticles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge articles")
		return
	}
	writeJSON(w, articles)
}

func knowledgeIdeasHandler(w http.ResponseWriter, r *http.Request) {
	ideas, err := store.GetKnowledgeIdeas()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge ideas")
		return
	}
	writeJSON(w, ideas)
}

func knowledgePostsHandler(w http.ResponseWriter, r *http.Request) {
	posts, err := store.GetKnowledgePosts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load knowledge posts")
		return
	}
	writeJSON(w, posts)
}

func createKnowledgePostHandler(w http.ResponseWriter, r *http.Request) {
	var req model.KnowledgePost
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Title == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, "title and content are required")
		return
	}
	post, err := store.CreateKnowledgePost(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create knowledge post")
		return
	}
	writeJSON(w, post)
}

func channelsHandler(w http.ResponseWriter, r *http.Request) {
	channels, err := store.GetChannels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load channels")
		return
	}
	writeJSON(w, channels)
}

func conversationsHandler(w http.ResponseWriter, r *http.Request) {
	conversations, err := store.GetConversations()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load conversations")
		return
	}
	writeJSON(w, conversations)
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	conversationID := r.URL.Query().Get("conversationId")
	messages, err := store.GetMessages(conversationID)
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

	conversationID := strings.TrimSpace(r.FormValue("conversationId"))
	if conversationID == "" {
		conversationID = "general"
	}
	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		text = "Shared media"
	}
	sender := strings.TrimSpace(r.FormValue("sender"))
	if sender == "" {
		sender = "StatChat User"
	}

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
	if _, err := io.Copy(destinationFile, file); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save upload")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
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
		ConversationID: conversationID,
		Sender:         sender,
		Text:           text,
		CreatedAt:      time.Now().UTC(),
		Status:         "active",
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
	writeJSON(w, message)
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

	if req.Text == "" || req.Sender == "" {
		writeError(w, http.StatusBadRequest, "sender and text are required")
		return
	}
	if req.ConversationID == "" {
		req.ConversationID = "general"
	}
	message := model.Message{
		ID:              uuid.NewString(),
		TenantID:        resolveTenantID(req.TenantID),
		ConversationID:  req.ConversationID,
		ChannelID:       req.ChannelID,
		ParentMessageID: req.ParentMessageID,
		ThreadRootID:    req.ThreadRootID,
		Sender:          req.Sender,
		Text:            req.Text,
		CreatedAt:       time.Now().UTC(),
		Status:          "active",
		DeliveryStatus:  "sent",
	}
	if err := store.StoreMessage(message); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save message")
		return
	}
	store.BroadcastMessage(message)
	writeJSON(w, message)
}

func conversationMessagesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	conversationID := vars["id"]
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenantId"))
	messages, err := store.GetMessagesForTenant(conversationID, tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load conversation messages")
		return
	}
	writeJSON(w, messages)
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
	message, err := store.GetMessageByID(messageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "message not found")
		return
	}
	message.Text = req.Text
	if err := store.UpdateMessage(message); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update message")
		return
	}
	writeJSON(w, message)
}

func deleteMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	messageID := vars["id"]
	if err := store.SoftDeleteMessage(messageID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete message")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	client := store.NewClient(conn, "general")
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
		switch action {
		case "join-conversation":
			conversationID, _ := payload["conversationId"].(string)
			tenantID, _ := payload["tenantId"].(string)
			if conversationID == "" {
				conversationID = "general"
			}
			client.SetConversation(routeConversationKey(tenantID, conversationID))
		case "send-message":
			message, err := buildMessageFromPayload(payload)
			if err != nil {
				log.Printf("invalid websocket chat payload: %v", err)
				continue
			}
			if err := store.StoreMessage(message); err != nil {
				log.Printf("failed to store websocket message: %v", err)
				continue
			}
			store.BroadcastMessage(message)
		}
	}
}

func buildMessageFromPayload(payload map[string]interface{}) (model.Message, error) {
	var sender string
	var text string
	var conversationID string
	var channelID string
	var tenantID string
	var topLevelTenant string

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

	sender, _ = payload["sender"].(string)
	text, _ = payload["text"].(string)
	conversationID, _ = payload["conversationId"].(string)
	channelID, _ = payload["channelId"].(string)
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
		ID:             uuid.NewString(),
		TenantID:       resolveTenantID(tenantID),
		ConversationID: conversationID,
		ChannelID:      channelID,
		Sender:         sender,
		Text:           text,
		CreatedAt:      time.Now().UTC(),
		Status:         "active",
		DeliveryStatus: "sent",
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
		if r.URL.Path == "/health" || r.URL.Path == "/readyz" || strings.HasPrefix(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/uploads") {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.EqualFold(os.Getenv("STATCHAT_AUTH_REQUIRED"), "true") {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			secret := os.Getenv("STATCHAT_JWT_SECRET")
			if secret == "" {
				secret = "statchat-dev-secret"
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
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
