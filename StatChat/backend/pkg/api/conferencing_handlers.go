package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"statchat/pkg/model"
	"statchat/pkg/store"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ── Call Session Lifecycle ──

type createCallRequest struct {
	Kind           model.CallKind `json:"kind"`
	RoomName       string         `json:"roomName"`
	ConversationID string         `json:"conversationId,omitempty"`
	HostID         string         `json:"hostId,omitempty"`
	HostName       string         `json:"hostName,omitempty"`
}

func canAccessCallSession(session model.CallSession, userID, tenantID string) (bool, error) {
	if strings.TrimSpace(session.TenantID) != resolveTenantID(tenantID) {
		return false, nil
	}
	if strings.TrimSpace(session.Conversation) == "" {
		return true, nil
	}
	return store.CanAccessConversation(session.Conversation, userID, tenantID)
}

func requireCallSessionAccess(w http.ResponseWriter, r *http.Request, sessionID string) (model.CallSession, bool) {
	session, err := store.GetCallSession(sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "call session not found")
		return model.CallSession{}, false
	}
	allowed, err := canAccessCallSession(session, requestUserID(r), requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify call access")
		return model.CallSession{}, false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "you do not have access to this call")
		return model.CallSession{}, false
	}
	return session, true
}

func isAllowedCallSignalType(signalType string) bool {
	switch signalType {
	case "offer", "answer", "ice-candidate", "screen-share-started", "screen-share-stopped", "mute-requested", "mute-accepted", "mute-declined":
		return true
	default:
		return false
	}
}

func createCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load current user")
		return
	}

	var req createCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Kind == "" {
		req.Kind = model.CallKindVideo
	}
	if req.Kind != model.CallKindVoice && req.Kind != model.CallKindVideo {
		writeError(w, http.StatusBadRequest, "kind must be voice or video")
		return
	}
	if req.ConversationID != "" && !requireConversationAccess(w, r, req.ConversationID) {
		return
	}
	if req.RoomName == "" {
		req.RoomName = fmt.Sprintf("%s's %s call", currentUser.Name, req.Kind)
	}
	if req.HostID == "" || req.HostID != currentUser.ID {
		req.HostID = currentUser.ID
	}
	if req.HostName == "" || req.HostName != currentUser.Name {
		req.HostName = currentUser.Name
	}

	session, err := store.CreateCallSession(model.CallSession{
		TenantID:     requestTenantID(r),
		RoomName:     req.RoomName,
		Kind:         req.Kind,
		HostID:       req.HostID,
		HostName:     req.HostName,
		Status:       model.CallStatusLive,
		Conversation: req.ConversationID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create call session")
		return
	}

	// Auto-join host as first participant.
	if _, err := store.JoinCallSession(session.ID, req.HostID, req.HostName); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to join host to session")
		return
	}

	writeJSON(w, session)
}

func listCallSessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := store.ListActiveCallSessions(requestTenantID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list call sessions")
		return
	}
	accessible := make([]model.CallSession, 0, len(sessions))
	for _, session := range sessions {
		allowed, accessErr := canAccessCallSession(session, requestUserID(r), requestTenantID(r))
		if accessErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to verify call access")
			return
		}
		if allowed {
			accessible = append(accessible, session)
		}
	}
	writeJSON(w, accessible)
}

func getCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session, allowed := requireCallSessionAccess(w, r, vars["id"])
	if !allowed {
		return
	}
	participants, err := store.GetActiveParticipants(session.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load participants")
		return
	}
	writeJSON(w, map[string]interface{}{
		"session":      session,
		"participants": participants,
	})
}

type joinCallRequest struct {
	UserID   string `json:"userId,omitempty"`
	UserName string `json:"userName,omitempty"`
}

func joinCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, allowed := requireCallSessionAccess(w, r, sessionID)
	if !allowed {
		return
	}
	if session.Status != model.CallStatusLive {
		writeError(w, http.StatusConflict, "call session is not live")
		return
	}

	var req joinCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	currentUser, err := requestCurrentUser(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authenticated user is required")
		return
	}
	req.UserID = currentUser.ID
	req.UserName = currentUser.Name

	participant, err := store.JoinCallSession(sessionID, req.UserID, req.UserName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to join call session")
		return
	}

	// Broadcast participant joined to the room.
	store.BroadcastCallState(sessionID, "call-participant-joined", participant)
	writeJSON(w, participant)
}

func leaveCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if _, allowed := requireCallSessionAccess(w, r, sessionID); !allowed {
		return
	}
	if err := store.LeaveCallSession(sessionID, requestUserID(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to leave call session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func removeCallParticipantHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session, allowed := requireCallSessionAccess(w, r, vars["id"])
	if !allowed {
		return
	}
	requesterRole, roleErr := store.GetActiveCallParticipantRole(session.ID, requestUserID(r))
	if roleErr != nil || (requesterRole != "host" && requesterRole != "moderator") {
		writeError(w, http.StatusForbidden, "only the call host or a moderator can remove participants")
		return
	}
	targetID := strings.TrimSpace(vars["userId"])
	if targetID == "" || targetID == session.HostID {
		writeError(w, http.StatusConflict, "the call host cannot be removed")
		return
	}
	targetRole, targetRoleErr := store.GetActiveCallParticipantRole(session.ID, targetID)
	if targetRoleErr != nil {
		writeError(w, http.StatusNotFound, "active call participant not found")
		return
	}
	if requesterRole == "moderator" && targetRole != "participant" {
		writeError(w, http.StatusForbidden, "moderators can only remove ordinary participants")
		return
	}
	removed, err := store.RemoveCallParticipant(session.ID, targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove call participant")
		return
	}
	if !removed {
		writeError(w, http.StatusNotFound, "active call participant not found")
		return
	}
	store.BroadcastCallState(session.ID, "call-participant-left", map[string]string{"userId": targetID, "reason": "removed-by-moderator"})
	writeJSON(w, map[string]any{"removed": true, "userId": targetID})
}

func updateCallParticipantRoleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	session, allowed := requireCallSessionAccess(w, r, vars["id"])
	if !allowed {
		return
	}
	if !canEndCall(session, requestUserID(r)) {
		writeError(w, http.StatusForbidden, "only the call host can manage roles")
		return
	}
	var request struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil || (request.Role != "moderator" && request.Role != "participant") {
		writeError(w, http.StatusBadRequest, "role must be moderator or participant")
		return
	}
	targetID := strings.TrimSpace(vars["userId"])
	if targetID == "" || targetID == session.HostID {
		writeError(w, http.StatusConflict, "the call host role cannot be changed")
		return
	}
	updated, err := store.UpdateCallParticipantRole(session.ID, targetID, request.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update call participant role")
		return
	}
	if !updated {
		writeError(w, http.StatusNotFound, "active call participant not found")
		return
	}
	store.BroadcastCallState(session.ID, "call-participant-role-changed", map[string]string{"userId": targetID, "role": request.Role})
	writeJSON(w, map[string]string{"userId": targetID, "role": request.Role})
}

func endCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	session, allowed := requireCallSessionAccess(w, r, sessionID)
	if !allowed {
		return
	}
	if !canEndCall(session, requestUserID(r)) {
		writeError(w, http.StatusForbidden, "only the call host can end the call")
		return
	}

	session, err := store.EndCallSession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to end call session")
		return
	}
	store.BroadcastCallState(sessionID, "call-ended", map[string]string{"sessionId": sessionID})
	writeJSON(w, session)
}

func canEndCall(session model.CallSession, userID string) bool {
	return strings.TrimSpace(userID) != "" && strings.TrimSpace(userID) == strings.TrimSpace(session.HostID)
}

func getCallParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, allowed := requireCallSessionAccess(w, r, vars["id"]); !allowed {
		return
	}
	participants, err := store.GetActiveParticipants(vars["id"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load participants")
		return
	}
	writeJSON(w, participants)
}

type callQualityRequest struct {
	RTTMs         float64 `json:"rttMs"`
	JitterMs      float64 `json:"jitterMs"`
	PacketLossPct float64 `json:"packetLossPct"`
	BitrateKbps   float64 `json:"bitrateKbps"`
}

func validCallQualityMetrics(request callQualityRequest) bool {
	values := []float64{request.RTTMs, request.JitterMs, request.PacketLossPct, request.BitrateKbps}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return false
		}
	}
	return request.RTTMs <= 60000 && request.JitterMs <= 10000 && request.PacketLossPct <= 100 && request.BitrateKbps <= 1000000
}

func classifyCallQuality(request callQualityRequest) string {
	if request.PacketLossPct < 1 && request.RTTMs < 150 && request.JitterMs < 30 {
		return "excellent"
	}
	if request.PacketLossPct < 3 && request.RTTMs < 300 && request.JitterMs < 50 {
		return "good"
	}
	if request.PacketLossPct < 8 && request.RTTMs < 600 && request.JitterMs < 100 {
		return "fair"
	}
	return "poor"
}

func createCallQualitySampleHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]
	if _, allowed := requireCallSessionAccess(w, r, sessionID); !allowed {
		return
	}
	active, err := store.IsActiveCallParticipant(sessionID, requestUserID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify active call participant")
		return
	}
	if !active {
		writeError(w, http.StatusForbidden, "only active call participants can report quality")
		return
	}
	var request callQualityRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&request); err != nil || !validCallQualityMetrics(request) {
		writeError(w, http.StatusBadRequest, "invalid call quality metrics")
		return
	}
	sample, err := store.SaveCallQualitySample(model.CallQualitySample{SessionID: sessionID, UserID: requestUserID(r), RTTMs: request.RTTMs, JitterMs: request.JitterMs, PacketLossPct: request.PacketLossPct, BitrateKbps: request.BitrateKbps, Quality: classifyCallQuality(request)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store call quality sample")
		return
	}
	writeJSON(w, sample)
}

func callQualitySamplesHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["id"]
	if _, allowed := requireCallSessionAccess(w, r, sessionID); !allowed {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	samples, err := store.GetCallQualitySamples(sessionID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load call quality samples")
		return
	}
	writeJSON(w, samples)
}

// ── Call Recordings ──

type callRecordingUploadRequest struct {
	Title string
}

func uploadCallRecordingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, allowed := requireCallSessionAccess(w, r, sessionID)
	if !allowed {
		return
	}
	if !canEndCall(session, requestUserID(r)) {
		writeError(w, http.StatusForbidden, "only the call host can upload recordings")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 256<<20)
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
	if !isAllowedCallRecording(header.Filename, header.Header.Get("Content-Type")) {
		writeError(w, http.StatusBadRequest, "recording must be a WebM file")
		return
	}
	if !hasWebMSignature(file) {
		writeError(w, http.StatusBadRequest, "recording content is not valid WebM")
		return
	}

	// Save recordings into the uploads directory so the existing
	// /uploads/ file server can serve them (matches the stored URL).
	dir := strings.TrimSpace(os.Getenv("STATCHAT_UPLOAD_DIR"))
	if dir == "" {
		dir = filepath.Join(".", "uploads")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create recording dir")
		return
	}

	extension := strings.ToLower(filepath.Ext(header.Filename))
	safeName := fmt.Sprintf("%s%s", uuid.NewString(), extension)
	destinationPath := filepath.Join(dir, safeName)
	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create recording file")
		return
	}
	defer destinationFile.Close()

	size, err := io.Copy(destinationFile, file)
	if err != nil {
		_ = os.Remove(destinationPath)
		writeError(w, http.StatusInternalServerError, "failed to save recording")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = "Meeting recording " + time.Now().UTC().Format("2006-01-02 15:04")
	}

	recording := model.CallRecording{
		SessionID: sessionID,
		Title:     title,
		FileName:  header.Filename,
		URL:       fmt.Sprintf("/uploads/%s", safeName),
		Size:      size,
		Duration:  strings.TrimSpace(r.FormValue("duration")),
		CreatedAt: time.Now().UTC(),
	}
	saved, err := store.SaveCallRecording(recording)
	if err != nil {
		_ = os.Remove(destinationPath)
		writeError(w, http.StatusInternalServerError, "failed to save recording metadata")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, saved)
}

func isAllowedCallRecording(filename, contentType string) bool {
	if !strings.EqualFold(filepath.Ext(strings.TrimSpace(filename)), ".webm") {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return false
	}
	return mediaType == "video/webm" || mediaType == "audio/webm"
}

func hasWebMSignature(file io.ReadSeeker) bool {
	var signature [4]byte
	if _, err := io.ReadFull(file, signature[:]); err != nil {
		return false
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return false
	}
	return signature == [4]byte{0x1a, 0x45, 0xdf, 0xa3}
}

func listCallRecordingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if _, allowed := requireCallSessionAccess(w, r, vars["id"]); !allowed {
		return
	}
	recordings, err := store.GetCallRecordings(vars["id"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load recordings")
		return
	}
	writeJSON(w, recordings)
}

// ── Meeting → Call session linking ──

func getMeetingSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	meetingID := vars["id"]

	// Find an active/live call session associated with this meeting room.
	// The meeting room url encodes a room token; try to find by room name matching title.
	var meetings, err = store.GetMeetings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load meetings")
		return
	}
	for _, m := range meetings {
		if m.ID != meetingID {
			continue
		}
		session, err := store.FindCallSessionByRoom(m.Room, requestTenantID(r))
		if err == nil && session.ID != "" {
			writeJSON(w, session)
			return
		}
		// No live session yet — create one and link to the meeting.
		currentUser, err := requestCurrentUser(r)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load current user")
			return
		}
		session, err = store.CreateCallSession(model.CallSession{
			TenantID: requestTenantID(r),
			RoomName: m.Title,
			RoomID:   m.Room,
			Kind:     model.CallKindVideo,
			HostID:   currentUser.ID,
			HostName: currentUser.Name,
			Status:   model.CallStatusLive,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create meeting session")
			return
		}
		if _, err := store.JoinCallSession(session.ID, currentUser.ID, currentUser.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to join meeting host to session")
			return
		}
		writeJSON(w, session)
		return
	}

	writeError(w, http.StatusNotFound, "meeting not found")
}
